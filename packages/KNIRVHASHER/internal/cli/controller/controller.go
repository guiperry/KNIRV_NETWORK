package controller

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"knirvhasher/internal/analyzer"
	"knirvhasher/internal/cli/embedded"
	"knirvhasher/internal/client"
	"knirvhasher/internal/config"
	"knirvhasher/pkg/hashing/methods/asic"
	"knirvhasher/pkg/hashing/validation"
)

type Controller struct {
	model     *uiModel
	deployer  *analyzer.Deployer
	apiClient *client.APIClient

	activeOps map[string]OperationStatus
	opsMu     sync.RWMutex

	pipelineLogChan chan PipelineLogMsg
	logChan         chan string

	serverCmd       *exec.Cmd
	knirvbaseCmd    *exec.Cmd
	knirvbaseCancel context.CancelFunc
	serverReady     bool
	serverStarting  bool
	goatMode        bool

	pipelineState struct {
		Cmd           *exec.Cmd
		Running       bool
		StopRequested bool
		Mu            sync.Mutex
	}
}

type uiModel struct {
	ServerCmd       *exec.Cmd
	ServerReady     bool
	ServerStarting  bool
	DeviceIP        string
	DeviceType      string
	CryptoEnabled   bool
	PipelineRunning bool
	PipelineStage   string
	PipelineType    string
	PipelineStages  []PipelineStage
}

type OperationStatus struct {
	ID        string
	Type      string
	Status    string
	Progress  float64
	Message   string
	StartTime time.Time
}

type PipelineStage struct {
	Name    string
	BinName string
	Args    []string
	Desc    string
}

type PipelineLogMsg struct {
	Log        string
	StageIndex int
	Stage      string
	Error      bool
	Complete   bool
}

type DiscoveryResult struct {
	Devices      []analyzer.DeviceInfo
	SelectedIP   string
	SelectedType string
	Output       string
	Duration     float64
}

func NewController(goatMode bool) *Controller {
	config := analyzer.DefaultDeployerConfig()
	deployer, _ := analyzer.NewDeployer(config)

	return &Controller{
		model:           &uiModel{},
		deployer:        deployer,
		activeOps:       make(map[string]OperationStatus),
		pipelineLogChan: make(chan PipelineLogMsg, 100),
		logChan:         make(chan string, 100),
		goatMode:        goatMode,
	}
}

func (c *Controller) GetProgressChan() <-chan PipelineLogMsg {
	return c.pipelineLogChan
}

func (c *Controller) IsServerReady() bool {
	return c.model.ServerReady
}

func (c *Controller) IsServerStarting() bool {
	return c.model.ServerStarting
}

func (c *Controller) GetAPIClient() *client.APIClient {
	return c.apiClient
}

func (c *Controller) GetActiveDevice() *analyzer.DeviceInfo {
	if c.deployer == nil {
		return nil
	}
	return c.deployer.GetActiveDevice()
}

func (c *Controller) GetLogChan() <-chan string {
	return c.logChan
}

func (c *Controller) GetPipelineLogChan() <-chan PipelineLogMsg {
	return c.pipelineLogChan
}

func (c *Controller) StartDriver(ctx context.Context) error {
	// Loaded before the reuse check below so that check can confirm a
	// candidate already-running hasher-host is actually in the requested
	// mode, not just healthy - see findRunningHasherHost's doc comment.
	deviceConfig, _ := config.LoadDeviceConfig()
	wantDirect := deviceConfig != nil && deviceConfig.DirectMode

	if port := findRunningHasherHost(wantDirect); port > 0 {
		c.model.ServerReady = true
		c.model.ServerStarting = false
		c.apiClient = client.NewAPIClient(port)
		return nil
	}

	hostPath, err := embedded.GetHasherHostPathForce()
	if err != nil {
		return fmt.Errorf("failed to extract hasher-host: %w", err)
	}

	binDir, err := embedded.GetBinDir()
	if err != nil {
		return fmt.Errorf("failed to get binary directory: %w", err)
	}

	var args []string
	if deviceConfig != nil && deviceConfig.IP != "" {
		args = append(args, "--device="+deviceConfig.IP)
		args = append(args, "--discover=false")
		args = append(args, "--force-redeploy=true")
		if deviceConfig.Password != "" {
			args = append(args, "--server-ssh-password="+deviceConfig.Password)
		}
	} else {
		args = append(args, "--discover=true")
		args = append(args, "--auto-deploy=true")
	}

	if deviceConfig != nil && deviceConfig.FramesDir != "" {
		args = append(args, "--frames-dir="+deviceConfig.FramesDir)
	}

	if deviceConfig != nil && deviceConfig.CGMinerHost != "" {
		args = append(args, "--cgminer-host="+deviceConfig.CGMinerHost)
	}

	if deviceConfig != nil && deviceConfig.DirectMode {
		args = append(args, "--direct")
	}

	cmd := exec.Command(hostPath, args...)
	cmd.Dir = binDir
	cmd.Env = os.Environ()
	if deviceConfig != nil {
		if deviceConfig.IP != "" {
			cmd.Env = append(cmd.Env, "DEVICE_IP="+deviceConfig.IP)
		}
		if deviceConfig.Password != "" {
			cmd.Env = append(cmd.Env, "DEVICE_PASSWORD="+deviceConfig.Password)
		}
		if deviceConfig.Username != "" {
			cmd.Env = append(cmd.Env, "DEVICE_USERNAME="+deviceConfig.Username)
		}
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start hasher-host: %w", err)
	}

	c.serverCmd = cmd
	c.model.ServerCmd = cmd
	c.model.ServerStarting = true

	go c.streamOutput(stdoutPipe, stderrPipe)
	go c.waitForServerReady()

	return nil
}

func (c *Controller) streamOutput(stdout, stderr io.ReadCloser) {
	go func() {
		defer stdout.Close()
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				select {
				case c.logChan <- string(buf[:n]):
				default:
				}
			}
		}
	}()

	go func() {
		defer stderr.Close()
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				select {
				case c.logChan <- string(buf[:n]):
				default:
				}
			}
		}
	}()
}

func (c *Controller) waitForServerReady() {
	startTime := time.Now()
	timeout := 5 * time.Minute
	portFile := "/tmp/hasher-host.port"

	for time.Since(startTime) < timeout {
		portBytes, err := os.ReadFile(portFile)
		if err == nil {
			port, err := strconv.Atoi(strings.TrimSpace(string(portBytes)))
			if err == nil && isHasherHostRunning(port) {
				c.model.ServerReady = true
				c.model.ServerStarting = false
				c.apiClient = client.NewAPIClient(port)
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// waitForASICReadyOrFail blocks until the configured DEVICE_IP's ASIC is
// confirmed operational via a real health-check probe - the same one
// asic.NewASICClient/Connect performs internally (GetDeviceInfo plus a
// genuine ComputeHash round-trip, not just "is the port open") - or a
// generous timeout elapses. If no ASIC device is configured, it returns
// immediately: a software/CUDA pipeline has nothing to wait on.
func (c *Controller) waitForASICReadyOrFail(ctx context.Context) error {
	deviceConfig, _ := config.LoadDeviceConfig()
	if deviceConfig == nil || deviceConfig.IP == "" {
		return nil
	}

	address := net.JoinHostPort(deviceConfig.IP, "8888")

	const (
		pollInterval = 2 * time.Second
		statusEvery  = 5 * time.Second
		maxWait      = 90 * time.Second
	)

	fmt.Printf("[pipeline] Waiting for ASIC at %s to come online before starting pipeline (up to %s)...\n", address, maxWait)

	start := time.Now()
	lastStatus := start
	for {
		asicClient, err := asic.NewASICClient(address)
		if err == nil {
			ready := !asicClient.IsUsingFallback()
			asicClient.Close()
			if ready {
				fmt.Printf("[pipeline] ASIC at %s confirmed online after %s\n", address, time.Since(start).Round(time.Second))
				return nil
			}
		}

		if time.Since(start) >= maxWait {
			return fmt.Errorf("ASIC at %s did not come online within %s", address, maxWait)
		}

		if time.Since(lastStatus) >= statusEvery {
			fmt.Printf("[pipeline] Still waiting for ASIC at %s (%s elapsed)...\n", address, time.Since(start).Round(time.Second))
			lastStatus = time.Now()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func (c *Controller) StopDriver(ctx context.Context) error {
	if c.serverCmd == nil || c.serverCmd.Process == nil {
		return nil
	}

	fmt.Println("Shutting down hasher-host...")

	if c.apiClient != nil {
		shutdownURL := c.apiClient.BaseURL + "/api/v1/shutdown"
		http.Post(shutdownURL, "application/json", nil)
	}

	done := make(chan error, 1)
	go func() {
		done <- c.serverCmd.Wait()
	}()

	select {
	case <-done:
		fmt.Println("hasher-host shut down successfully.")
	case <-time.After(15 * time.Second):
		fmt.Println("hasher-host shutdown timeout, force killing...")
		c.serverCmd.Process.Kill()
	}

	c.serverCmd = nil
	c.model.ServerCmd = nil
	c.model.ServerReady = false
	c.model.ServerStarting = false
	c.apiClient = nil

	return nil
}

func (c *Controller) GetDriverStatus() map[string]interface{} {
	status := map[string]interface{}{
		"running":  c.model.ServerReady,
		"starting": c.model.ServerStarting,
	}

	if c.model.ServerReady && c.apiClient != nil {
		if health, err := c.apiClient.GetHealth(); err == nil {
			status["health"] = health
		}
	}

	return status
}

func (c *Controller) DiscoverASIC(ctx context.Context) (*DiscoveryResult, error) {
	if c.deployer == nil {
		return nil, fmt.Errorf("deployer not initialized")
	}

	var logBuffer bytes.Buffer
	c.deployer.SetLogWriter(&logBuffer)

	result, err := c.deployer.RunDiscovery()
	if err != nil {
		return nil, err
	}

	devices := c.deployer.GetDevices()
	selectedIP := ""
	selectedType := ""

	if len(devices) > 0 {
		c.deployer.SelectDevice(0)
		selectedIP = devices[0].IPAddress
		selectedType = devices[0].DeviceType
	}

	return &DiscoveryResult{
		Devices:      devices,
		SelectedIP:   selectedIP,
		SelectedType: selectedType,
		Output:       result.Output,
		Duration:     result.Duration,
	}, nil
}

func (c *Controller) ProbeASIC(ctx context.Context) error {
	if c.deployer == nil {
		return fmt.Errorf("deployer not initialized")
	}

	device := c.deployer.GetActiveDevice()
	if device == nil {
		return fmt.Errorf("no device selected - run Discovery first")
	}

	_, err := c.deployer.RunProbe()
	return err
}

func (c *Controller) DetectProtocol(ctx context.Context) error {
	if c.deployer == nil {
		return fmt.Errorf("deployer not initialized")
	}

	device := c.deployer.GetActiveDevice()
	if device == nil {
		return fmt.Errorf("no device selected - run Discovery first")
	}

	_, err := c.deployer.RunProtocol()
	return err
}

func (c *Controller) ProvisionASIC(ctx context.Context, deviceIP string) error {
	if c.deployer == nil {
		return fmt.Errorf("deployer not initialized")
	}

	if deviceIP != "" {
		if err := c.deployer.SelectDeviceByIP(deviceIP); err != nil {
			return err
		}
	}

	_, err := c.deployer.RunProvision()
	return err
}

func (c *Controller) TroubleshootASIC(ctx context.Context) error {
	if c.deployer == nil {
		return fmt.Errorf("deployer not initialized")
	}

	device := c.deployer.GetActiveDevice()
	if device == nil {
		return fmt.Errorf("no device selected - run Discovery first")
	}

	_, err := c.deployer.RunTroubleshoot()
	return err
}

func (c *Controller) ConfigureASIC(ctx context.Context) (string, error) {
	if c.deployer == nil {
		return "", fmt.Errorf("deployer not initialized")
	}

	device := c.deployer.GetActiveDevice()
	if device == nil {
		return "", fmt.Errorf("no device selected")
	}

	var output strings.Builder
	output.WriteString("Current Configuration:\n")
	output.WriteString(strings.Repeat("-", 40) + "\n\n")
	output.WriteString(fmt.Sprintf("  Active Device: %s\n", device.IPAddress))
	output.WriteString(fmt.Sprintf("    Device Type:   %s\n", device.DeviceType))
	output.WriteString(fmt.Sprintf("    Protocol:      %s\n", device.Protocol.String()))
	output.WriteString(fmt.Sprintf("    Accessible:    %v\n", device.Accessible))

	return output.String(), nil
}

func (c *Controller) ManageRules(ctx context.Context, action string, args ...string) (string, error) {
	validator, err := validation.NewLogicalValidator()
	if err != nil {
		return "", fmt.Errorf("error creating validator: %w", err)
	}

	switch action {
	case "list":
		var output strings.Builder
		output.WriteString("Logical Validation Rules\n")
		output.WriteString("==========================\n\n")

		if len(args) > 0 {
			domain := args[0]
			rules, err := validator.KnowledgeBase.GetRules(domain)
			if err != nil {
				return "", err
			}
			output.WriteString(fmt.Sprintf("Domain: %s (%d rules)\n\n", domain, len(rules)))
			for i, rule := range rules {
				output.WriteString(fmt.Sprintf("[%d] %s\n", i, rule.String()))
			}
		} else {
			for domain, rules := range validator.KnowledgeBase.Domains {
				output.WriteString(fmt.Sprintf("Domain: %s (%d rules)\n", domain, len(rules)))
				for i, rule := range rules {
					output.WriteString(fmt.Sprintf("  [%d] %s\n", i, rule.String()))
				}
			}
		}
		return output.String(), nil

	case "add":
		if len(args) < 3 {
			return "", fmt.Errorf("usage: add <domain> <type> <conclusion>")
		}
		domain := args[0]
		ruleType := args[1]
		conclusion := strings.Join(args[2:], " ")

		rule, err := validation.NewLogicalRule(ruleType, []string{}, conclusion, "Added via API")
		if err != nil {
			return "", err
		}

		if err := validator.KnowledgeBase.AddRule(domain, rule); err != nil {
			return "", err
		}

		return fmt.Sprintf("Rule added to domain '%s'", domain), nil

	case "delete":
		if len(args) < 2 {
			return "", fmt.Errorf("usage: delete <domain> <index>")
		}
		domain := args[0]
		var index int
		fmt.Sscanf(args[1], "%d", &index)

		if err := validator.KnowledgeBase.RemoveRule(domain, index); err != nil {
			return "", err
		}

		return fmt.Sprintf("Rule %d deleted from domain '%s'", index, domain), nil

	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (c *Controller) TestASIC(ctx context.Context) error {
	if c.deployer == nil {
		return fmt.Errorf("deployer not initialized")
	}

	device := c.deployer.GetActiveDevice()
	if device == nil {
		return fmt.Errorf("no device selected - run Discovery first")
	}

	_, err := c.deployer.RunTest()
	return err
}

func (c *Controller) RunPipeline(ctx context.Context, pipelineType string) error {
	if err := c.startKNIRVBase(ctx); err != nil {
		return err
	}

	// --- TEMPORARY DEBUG INSTRUMENTATION ---
	// Block here until the configured ASIC is confirmed ready (or
	// definitively fails/times out) before launching any pipeline stage.
	// The stages below produce a lot of their own log output, and letting
	// them run concurrently with hasher-host/hasher-server's own startup
	// (CGMiner + real stratum pool handshake, which can legitimately take
	// up to ~40s) buried the one thing needed to diagnose a stuck ASIC:
	// hasher-host's own account of what happened. Remove/relax once ASIC
	// startup is reliable and fast enough not to need this.
	if err := c.waitForASICReadyOrFail(ctx); err != nil {
		return err
	}
	// --- END TEMPORARY DEBUG INSTRUMENTATION ---

	binDir, err := embedded.GetBinDir()
	if err != nil {
		return fmt.Errorf("failed to get binary directory: %w", err)
	}

	// When the CLI was started with -goat, enforce goat mode regardless of
	// the pipeline type sent over HTTP, so the flag always reaches data-mapper.
	if c.goatMode && pipelineType != "arxiv" && pipelineType != "demo" {
		pipelineType = "goat"
	}

	stages := buildPipelineStages(pipelineType)

	c.model.PipelineRunning = true
	c.model.PipelineStage = "initializing"
	c.model.PipelineStages = stages
	c.model.PipelineType = pipelineType

	c.pipelineState.Mu.Lock()
	c.pipelineState.StopRequested = false
	c.pipelineState.Mu.Unlock()

	// Each stage processes one batch (~100 records) and exits. Once every
	// stage has completed a batch, restart from the first stage so the next
	// batch flows through the whole pipeline again. This loops indefinitely
	// until a stage errors out or a stop is requested via Cleanup/StopPipeline.
	batchNum := 0
batchLoop:
	for {
		c.pipelineState.Mu.Lock()
		stopRequested := c.pipelineState.StopRequested
		c.pipelineState.Mu.Unlock()
		if stopRequested {
			break
		}

		batchNum++
		fmt.Printf("[pipeline] Starting batch %d (%s)\n", batchNum, pipelineType)

		for i, stage := range stages {
			c.model.PipelineStage = stage.Name

			binaryPath := filepath.Join(binDir, stage.BinName)

			if _, err := os.Stat(binaryPath); err != nil {
				binaryPath, err = embedded.GetBinaryPath(stage.BinName)
				if err != nil {
					c.model.PipelineRunning = false
					return fmt.Errorf("binary not found: %s (%w)", stage.BinName, err)
				}
			}

			var cmd *exec.Cmd
			if len(stage.Args) > 0 {
				cmd = exec.Command(binaryPath, stage.Args...)
			} else {
				cmd = exec.Command(binaryPath)
			}
			cmd.Dir = binDir

			if stage.BinName == "data-mapper" || stage.BinName == "data-seeder" {
				// Ensure spaCy shared lib and Python packages are findable.
				// Filter out any inherited PYTHONPATH/LD_LIBRARY_PATH first so our
				// values are not shadowed by duplicates when glibc does the lookup.
				existingPyPath := os.Getenv("PYTHONPATH")
				pyPath := pythonSitePackages()
				if existingPyPath != "" {
					pyPath = pyPath + ":" + existingPyPath
				}
				existingLd := os.Getenv("LD_LIBRARY_PATH")
				ldPath := binDir
				if existingLd != "" {
					ldPath += ":" + existingLd
				}
				baseEnv := filterEnv(os.Environ(), "PYTHONPATH", "LD_LIBRARY_PATH")
				cmd.Env = append(baseEnv,
					"LD_LIBRARY_PATH="+ldPath,
					"PYTHONPATH="+pyPath,
				)
			}

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return fmt.Errorf("failed to create stdout pipe: %w", err)
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				return fmt.Errorf("failed to create stderr pipe: %w", err)
			}

			if err := cmd.Start(); err != nil {
				c.model.PipelineRunning = false
				return fmt.Errorf("failed to start %s: %w", stage.Name, err)
			}

			c.pipelineState.Mu.Lock()
			c.pipelineState.Cmd = cmd
			c.pipelineState.Running = true
			c.pipelineState.Mu.Unlock()

			go c.streamPipelineOutput(cmd, stdout, stderr, i, stage)

			err = cmd.Wait()
			c.pipelineState.Mu.Lock()
			c.pipelineState.Running = false
			stopRequested = c.pipelineState.StopRequested
			c.pipelineState.Mu.Unlock()

			if err != nil {
				c.model.PipelineRunning = false
				c.model.PipelineStage = "failed: " + stage.Name
				select {
				case c.pipelineLogChan <- PipelineLogMsg{
					Log:        fmt.Sprintf("%s failed: %v", stage.Name, err),
					StageIndex: i,
					Stage:      stage.Name,
					Error:      true,
					Complete:   true,
				}:
				default:
				}
				return fmt.Errorf("%s failed: %w", stage.Name, err)
			}

			if stopRequested {
				c.model.PipelineRunning = false
				c.model.PipelineStage = "stopped"
				return nil
			}

			// The Hugging Face GOAT API can hand back a batch that the encoder's
			// checkpoint has already fully deduplicated (or a genuine repeat, since
			// we don't control the ordering it serves us). When that happens the
			// encoder writes an empty training_frames.json, and data-seeder would
			// otherwise fatal out on it. Treat that as "nothing to train yet" and
			// restart the pipeline for the next batch instead of failing the whole run.
			if stage.BinName == "data-encoder" {
				empty, checkErr := encoderOutputIsEmpty()
				if checkErr != nil {
					fmt.Printf("[pipeline] Warning: could not inspect encoder output, continuing to data-seeder: %v\n", checkErr)
				} else if empty {
					fmt.Printf("[pipeline] Batch %d: encoder produced 0 training frames (likely a duplicate Hugging Face batch) - skipping data-seeder and starting over\n", batchNum)
					continue batchLoop
				}
			}
		}

		fmt.Printf("[pipeline] Batch %d complete, restarting pipeline for next batch\n", batchNum)
	}

	c.model.PipelineRunning = false
	c.model.PipelineStage = "stopped"
	return nil
}

func (c *Controller) streamPipelineOutput(cmd *exec.Cmd, stdout, stderr io.ReadCloser, stageIndex int, stage PipelineStage) {
	go func() {
		defer stdout.Close()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				if len(line) > 150 {
					line = line[:147] + "..."
				}
				prefixed := fmt.Sprintf("[%s] [%s] %s", time.Now().Format("15:04:05"), stage.Name, line)
				// Emit to the parent process stdout so backend_server's SubprocessWriter captures it
				fmt.Fprintln(os.Stdout, prefixed)
				logMsg := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), line)
				// Non-blocking send: the primary log path is os.Stdout above.
				// Blocking here would stall the pipe reader, fill the OS pipe buffer,
				// and deadlock the subprocess — causing data-encoder and data-seeder
				// logs to disappear entirely.
				select {
				case c.pipelineLogChan <- PipelineLogMsg{
					Log:        logMsg,
					StageIndex: stageIndex,
					Stage:      stage.Name,
				}:
				default:
				}
			}
		}
	}()

	go func() {
		defer stderr.Close()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				if len(line) > 150 {
					line = line[:147] + "..."
				}
				prefixed := fmt.Sprintf("[%s] [%s] [stderr] %s", time.Now().Format("15:04:05"), stage.Name, line)
				// Emit to the parent process stderr so backend_server's SubprocessWriter captures it
				fmt.Fprintln(os.Stderr, prefixed)
				logMsg := fmt.Sprintf("[%s] [stderr] %s", time.Now().Format("15:04:05"), line)
				select {
				case c.pipelineLogChan <- PipelineLogMsg{
					Log:        logMsg,
					StageIndex: stageIndex,
					Stage:      stage.Name,
				}:
				default:
				}
			}
		}
	}()
}

func (c *Controller) RunMathVerifier(ctx context.Context) error {
	hostPath, err := embedded.GetHasherHostPathForce()
	if err != nil {
		return fmt.Errorf("failed to extract hasher-host: %w", err)
	}

	binDir, err := embedded.GetBinDir()
	if err != nil {
		return fmt.Errorf("failed to get binary directory: %w", err)
	}

	schemaPath := filepath.Join(binDir, "math_schema.yaml")
	if _, err := os.Stat(schemaPath); err != nil {
		cwd, _ := os.Getwd()
		schemaPath = filepath.Join(cwd, "pipeline", "2_DATA_ENCODER", "config", "math_schema.yaml")
		if _, err := os.Stat(schemaPath); err != nil {
			return fmt.Errorf("math_schema.yaml not found")
		}
	}

	args := []string{"--schema=" + schemaPath}

	deviceConfig, _ := config.LoadDeviceConfig()
	if deviceConfig != nil && deviceConfig.IP != "" {
		args = append(args, "--device="+deviceConfig.IP)
		args = append(args, "--discover=false")
		args = append(args, "--force-redeploy=true")
		if deviceConfig.Password != "" {
			args = append(args, "--server-ssh-password="+deviceConfig.Password)
		}
	} else {
		args = append(args, "--discover=true")
		args = append(args, "--auto-deploy=true")
	}

	if deviceConfig != nil && deviceConfig.FramesDir != "" {
		args = append(args, "--frames-dir="+deviceConfig.FramesDir)
	}

	cmd := exec.Command(hostPath, args...)
	cmd.Dir = binDir
	cmd.Env = os.Environ()

	if deviceConfig != nil {
		if deviceConfig.IP != "" {
			cmd.Env = append(cmd.Env, "DEVICE_IP="+deviceConfig.IP)
		}
		if deviceConfig.Password != "" {
			cmd.Env = append(cmd.Env, "DEVICE_PASSWORD="+deviceConfig.Password)
		}
		if deviceConfig.Username != "" {
			cmd.Env = append(cmd.Env, "DEVICE_USERNAME="+deviceConfig.Username)
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MATHASHER server: %w", err)
	}

	c.pipelineState.Mu.Lock()
	c.pipelineState.Cmd = cmd
	c.pipelineState.Running = true
	c.pipelineState.Mu.Unlock()

	go c.streamPipelineOutput(cmd, stdout, stderr, 0, PipelineStage{
		Name:    "MATHASHER",
		BinName: "hasher-host",
		Desc:    "Math verification server",
	})

	return nil
}

func (c *Controller) SendChat(ctx context.Context, message string) (string, error) {
	if !c.model.ServerReady || c.apiClient == nil {
		return "", fmt.Errorf("server not ready. Please start hasher-host first")
	}

	resp, err := c.apiClient.CallCryptoTransformer(message, nil)
	if err != nil {
		return "", err
	}

	return resp.Response, nil
}

func (c *Controller) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"driver_running": c.model.ServerReady,
		"server_ready":   c.model.ServerReady,
		"device_ip":      c.model.DeviceIP,
		"device_type":    c.model.DeviceType,
		"uptime":         time.Since(time.Now()).String(),
	}

	if c.deployer != nil {
		devices := c.deployer.GetDevices()
		status["devices_discovered"] = len(devices)
	}

	return status
}

func (c *Controller) Cleanup() {
	c.stopKNIRVBase()
	if c.model.ServerCmd != nil {
		c.StopDriver(context.Background())
	}

	if c.deployer != nil {
		c.deployer.Cleanup()
	}

	c.pipelineState.Mu.Lock()
	c.pipelineState.StopRequested = true
	if c.pipelineState.Cmd != nil {
		c.pipelineState.Cmd.Process.Kill()
		c.pipelineState.Running = false
	}
	c.pipelineState.Mu.Unlock()

	fmt.Println("Controller cleanup complete")
}

func (c *Controller) startKNIRVBase(ctx context.Context) error {
	addr := os.Getenv("KNIRVBASE_ADDR")
	if addr == "" {
		addr = "127.0.0.1:50052"
		os.Setenv("KNIRVBASE_ADDR", addr)
	}
	if c.knirvbaseCmd != nil {
		return nil
	}
	if health, err := http.Get("http://" + addr + "/health"); err == nil {
		health.Body.Close()
		if health.StatusCode < 500 {
			return nil
		}
	}
	path, err := embedded.GetBinaryPath("knirvbase")
	if err != nil {
		if os.Getenv("KNIRVBASE_REQUIRED") == "1" {
			return fmt.Errorf("knirvbase binary unavailable: %w", err)
		}
		select {
		case c.logChan <- "WARN: KNIRVBASE binary unavailable; pipeline submission disabled\n":
		default:
		}
		return nil
	}
	args := []string{"--addr=" + addr, "--network-id=" + envOrDefault("KNIRVBASE_NETWORK_ID", "knirvbase-default")}
	// KNIRVBASE is a controller-owned service, not a child of the HTTP request
	// that started the pipeline. In particular, net/http cancels r.Context()
	// after the asynchronous start response is written. Tie this process to an
	// explicit controller lifecycle instead and stop it from Cleanup.
	serviceCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(serviceCtx, path, args...)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start knirvbase: %w", err)
	}
	c.knirvbaseCmd = cmd
	c.knirvbaseCancel = cancel
	startupCtx, startupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer startupCancel()
	for i := 0; i < 100; i++ {
		select {
		case <-startupCtx.Done():
			c.stopKNIRVBase()
			return fmt.Errorf("knirvbase startup: %w", startupCtx.Err())
		case <-time.After(100 * time.Millisecond):
		}
		resp, err := http.Get("http://" + addr + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
	}
	c.stopKNIRVBase()
	return fmt.Errorf("knirvbase did not become healthy")
}

func (c *Controller) stopKNIRVBase() {
	if c.knirvbaseCancel != nil {
		c.knirvbaseCancel()
		c.knirvbaseCancel = nil
	}
	if c.knirvbaseCmd != nil && c.knirvbaseCmd.Process != nil {
		_ = c.knirvbaseCmd.Process.Kill()
		_ = c.knirvbaseCmd.Wait()
		c.knirvbaseCmd = nil
	}
}
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func buildPipelineStages(pipelineType string) []PipelineStage {
	trainerStage := PipelineStage{
		Name:    "data-seeder",
		BinName: "data-seeder",
		Args:    []string{"-verbose", "-epochs", "5", "-sequential", "-hash-method", "auto"},
		Desc:    "Data Trainer - Neural network training",
	}
	dataConnectorStage := PipelineStage{
		Name:    "data-connector",
		BinName: "data-connector",
		Args:    []string{},
		Desc:    "Data Connector - Priority source ingest and canonical RawRecord staging",
	}

	switch pipelineType {
	case "arxiv":
		return []PipelineStage{
			dataConnectorStage,
			{Name: "data-mapper", BinName: "data-mapper",
				Args: []string{"-single-batch"}, Desc: "Data Mapper - staged source records"},
			{Name: "data-encoder", BinName: "data-encoder",
				Args: []string{"-workers", "2"}, Desc: "Data Encoder - Tokenization and embeddings"},
			trainerStage,
		}
	case "demo":
		return []PipelineStage{
			dataConnectorStage,
			{Name: "data-mapper", BinName: "data-mapper",
				Args: []string{"-demo"}, Desc: "Data Mapper - staged source records"},
			trainerStage,
		}
	default: // all profiles begin with the source connector
		return []PipelineStage{
			dataConnectorStage,
			{Name: "data-mapper", BinName: "data-mapper",
				// -single-batch makes data-mapper process one batch (~100 records)
				// and exit, instead of looping forever inside its own process.
				// This lets the pipeline hand off to data-encoder / data-seeder
				// after every batch, and lets RunPipeline restart the whole
				// pipeline for the next batch (see RunPipeline below).
				Args: []string{"-single-batch"}, Desc: "Data Mapper - staged source records"},
			{Name: "data-encoder", BinName: "data-encoder",
				Args: []string{"-workers", "2"}, Desc: "Data Encoder - Tokenization and embeddings"},
			trainerStage,
		}
	}
}

// encoderOutputIsEmpty reports whether data-encoder's training_frames.json
// output (shared with data-seeder via the default frames directory) contains
// zero records. Used to decide whether to skip data-seeder for a batch
// rather than let it fatal out on an empty input file.
func encoderOutputIsEmpty() (bool, error) {
	cfg, err := config.LoadDeviceConfig()
	if err != nil {
		return false, fmt.Errorf("failed to resolve frames directory: %w", err)
	}

	path := filepath.Join(cfg.FramesDir, "training_frames.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read encoder output %s: %w", path, err)
	}

	var frames []json.RawMessage
	if err := json.Unmarshal(data, &frames); err != nil {
		return false, fmt.Errorf("failed to parse encoder output %s: %w", path, err)
	}

	return len(frames) == 0, nil
}

func pythonSitePackages() string {
	cmd := exec.Command("python3", "-c", "import site; print(site.getusersitepackages(), end='')")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// filterEnv returns env with any entries whose key matches one of the provided
// keys removed. Use this before appending new values for those keys so that
// duplicate entries don't shadow the intended values (glibc uses the first).
func filterEnv(env []string, keys ...string) []string {
	prefixes := make([]string, len(keys))
	for i, k := range keys {
		prefixes[i] = k + "="
	}
	out := make([]string, 0, len(env))
	for _, e := range env {
		skip := false
		for _, p := range prefixes {
			if strings.HasPrefix(e, p) {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, e)
		}
	}
	return out
}

// findRunningHasherHost looks for an already-running hasher-host that can be
// reused instead of starting a new one, but only if its mode actually
// matches wantDirect - reusing a stale instance regardless of mode used to
// mean a hasher-host started in CGMiner mode before -direct was ever passed
// (or vice versa) would be silently kept running forever, with the newly
// requested mode's args never even constructed. A mode mismatch is treated
// the same as "not found," so StartDriver falls through to launching a
// fresh, correctly-configured instance.
func findRunningHasherHost(wantDirect bool) int {
	ports := []int{8080, 8081, 8082, 8083, 8084, 8085, 8008, 9000}
	client := &http.Client{Timeout: 2 * time.Second}
	for _, port := range ports {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/api/v1/health", port))
		if err != nil {
			continue
		}
		var health struct {
			DirectMode bool `json:"direct_mode"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&health)
		statusCode := resp.StatusCode
		resp.Body.Close()
		if statusCode != 200 {
			continue
		}
		if decodeErr != nil {
			// Older hasher-host build without direct_mode in its health
			// response - only safe to reuse when direct mode isn't being
			// requested, since we can't confirm what mode it's actually in.
			if !wantDirect {
				return port
			}
			continue
		}
		if health.DirectMode == wantDirect {
			return port
		}
	}
	return 0
}

func isHasherHostRunning(port int) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/api/v1/health", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
