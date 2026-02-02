package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"hasher/internal/analyzer"
	"hasher/internal/cli/embedded"
	"hasher/internal/cli/ui"
	"hasher/internal/client"
)

const (
	portFile = "/tmp/hasher-host.port"
)

// CLI configuration flags
var (
	deviceIP      = flag.String("device", "", "ASIC device IP for hasher-host (empty = auto-discover)")
	discoveryFlag = flag.Bool("discover", true, "enable network discovery in hasher-host")
	monitorLogs   = flag.Bool("monitor-logs", true, "enable server log monitoring")
)

func main() {
	flag.Parse()
	var hasherHostCmd *exec.Cmd
	hasherHostStarted := false
	hasherHostPort := 8080 // Default port, will be updated if hasher-host finds different port

	// Initialize embedded binaries
	fmt.Println("Hasher CLI starting...")
	initEmbeddedBinaries()

	// Set up signal handler for clean shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create log channel for server output
	logChan := make(chan string, 100)

	// Create UI model and pass log channel
	model := ui.NewModel()

	// Handle server shutdown with ASIC cleanup
	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal.")
		cleanupASICDevice(model.Deployer)
		shutdownHasherHost(hasherHostCmd, hasherHostStarted, hasherHostPort)
		os.Exit(0)
	}()

	// Try to start hasher-host orchestrator
	hasherHostCmd, hasherHostStarted, hasherHostPort = startHasherHost(logChan)

	model.ServerCmd = hasherHostCmd
	model.ServerReady = hasherHostStarted

	// Update the API client to use the correct port
	model.APIClient = client.NewAPIClient(hasherHostPort)

	// Start the Bubble Tea UI with alternate screen and mouse support
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseAllMotion())

	// Start log listener and send log messages to program
	go func() {
		for log := range logChan {
			p.Send(ui.AppendLogMsg{Log: log})
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		cleanupASICDevice(model.Deployer)
		os.Exit(1)
	}

	// Ensure cleanup when exiting normally
	cleanupASICDevice(model.Deployer)
}

// initEmbeddedBinaries extracts embedded binaries to app data directory
func initEmbeddedBinaries() {
	binDir, err := embedded.GetBinDir()
	if err != nil {
		fmt.Printf("Warning: Could not determine binary directory: %v\n", err)
		return
	}

	// Check for embedded binaries
	binaries := embedded.CheckEmbeddedBinaries()
	embeddedCount := 0
	for _, b := range binaries {
		if b.Embedded {
			embeddedCount++
		}
	}

	if embeddedCount > 0 {
		fmt.Printf("Extracting %d embedded binaries to %s...\n", embeddedCount, binDir)
		if err := embedded.EnsureExtracted(); err != nil {
			fmt.Printf("Warning: Could not extract binaries: %v\n", err)
		} else {
			fmt.Println("Binaries extracted successfully.")
		}
	} else {
		fmt.Printf("No embedded binaries found. Binary directory: %s\n", binDir)
		fmt.Println("Build binaries with: make build-all")
	}
}

// startHasherHost attempts to start the hasher-host orchestrator
func startHasherHost(logChan chan string) (*exec.Cmd, bool, int) {
	// Try to find hasher-host binary
	hostPath, err := embedded.GetHasherHostPath()
	if err != nil {
		// Try looking in the working directory
		hostPath = findHasherHostExecutable()
		if hostPath == "" {
			logChan <- "hasher-host not found. Run Discovery to provision ASIC devices."
			return nil, false, 8080
		}
	}

	// Check if hasher-host is already running on common ports
	if port := findRunningHasherHost(); port > 0 {
		logChan <- fmt.Sprintf("Found existing hasher-host on port %d.", port)
		return nil, true, port
	}

	logChan <- fmt.Sprintf("Starting hasher-host from %s...", hostPath)

	// Build hasher-host arguments based on CLI flags
	args := []string{"--mode=api", "--port=0"}

	// Add device-specific flags if provided
	if *deviceIP != "" {
		args = append(args, "--device", *deviceIP, "--discover=false")
		logChan <- fmt.Sprintf("Using device %s (discovery disabled)", *deviceIP)
	} else {
		if !*discoveryFlag {
			args = append(args, "--discover=false")
		}
		if !*monitorLogs {
			args = append(args, "--monitor-server-logs=false")
		}
	}

	// Start hasher-host with configured arguments
	cmd := exec.Command(hostPath, args...)

	// Create pipes to capture output and forward to the UI
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		logChan <- fmt.Sprintf("Error starting hasher-host: %v", err)
		return nil, false, 8080
	}

	logChan <- fmt.Sprintf("hasher-host started with PID %d", cmd.Process.Pid)

	// Goroutine to forward stdout to the log channel
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdoutPipe.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				logChan <- string(buf[:n])
			}
		}
	}()

	// Goroutine to forward stderr to the log channel
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderrPipe.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				logChan <- string(buf[:n])
			}
		}
	}()

	// --- NEW LOGIC: Wait for port file and health check ---
	fmt.Println("Waiting for hasher-host to become healthy...")
	startTime := time.Now()
	timeout := 30 * time.Second // Increased timeout
	var actualPort int

	for time.Since(startTime) < timeout {
		portBytes, err := os.ReadFile(portFile)
		if err == nil {
			port, err := strconv.Atoi(strings.TrimSpace(string(portBytes)))
			if err == nil {
				actualPort = port
				// Now that we have a port, check the health endpoint
				if isHasherHostRunning(actualPort) {
					fmt.Printf("hasher-host is ready on port %d!\n", actualPort)
					logChan <- fmt.Sprintf("Hasher-host is healthy on port %d.", actualPort)
					return cmd, true, actualPort
				}
			}
		}
		// Wait a bit before retrying
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("hasher-host startup timed out, continuing without orchestrator.")
	logChan <- "Error: hasher-host startup timed out."

	// Attempt to kill the process we started, since it's not healthy
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
	return cmd, false, 8080 // default port
}

// findRunningHasherHost checks if hasher-host is already running on any port and returns the port
func findRunningHasherHost() int {
	// Common ports to check
	ports := []int{8080, 8081, 8082, 8083, 8084, 8085, 8008, 9000}
	client := &http.Client{Timeout: 2 * time.Second}

	for _, port := range ports {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/api/v1/health", port))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return port
			}
		}
	}
	return 0
}

// isHasherHostRunning checks if hasher-host API is responding on a specific port
func isHasherHostRunning(port int) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/api/v1/health", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// findHasherHostExecutable searches for hasher-host in common locations
func findHasherHostExecutable() string {
	// Check app data directory
	binDir, err := embedded.GetBinDir()
	if err == nil {
		paths := []string{
			binDir + "/hasher-host",
			binDir + "/hasher-host-linux-amd64",
			binDir + "/hasher-host-darwin-amd64",
			binDir + "/hasher-host-darwin-arm64",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	// Check current directory and common paths
	paths := []string{
		"./hasher-host",
		"./bin/hasher-host",
		"../hasher-host",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// cleanupASICDevice removes deployed binaries from the ASIC device
func cleanupASICDevice(deployer *analyzer.Deployer) {
	if deployer == nil {
		return
	}

	device := deployer.GetActiveDevice()
	if device == nil {
		return
	}

	fmt.Printf("Cleaning up binaries from ASIC device %s...\n", device.IPAddress)

	// Try to connect and cleanup
	if err := deployer.Connect(); err != nil {
		fmt.Printf("Could not connect to device for cleanup: %v\n", err)
		return
	}
	defer deployer.Disconnect()

	// Cleanup removes hasher-server and any temporary files
	if err := deployer.Cleanup(); err != nil {
		fmt.Printf("Cleanup warning: %v\n", err)
	} else {
		fmt.Println("ASIC device cleanup complete.")
	}
}

// shutdownHasherHost gracefully shuts down the hasher-host process
func shutdownHasherHost(cmd *exec.Cmd, started bool, port int) {
	if !started {
		return
	}

	fmt.Println("Shutting down hasher-host...")

	// If we started the process, we can wait for it to exit.
	if cmd != nil && cmd.Process != nil {
		// Attempt graceful shutdown via API. Fire and forget is okay.
		shutdownURL := fmt.Sprintf("http://localhost:%d/api/v1/shutdown", port)
		http.Post(shutdownURL, "application/json", nil)

		// Wait for process to terminate with timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			fmt.Println("hasher-host shut down successfully.")
		case <-time.After(15 * time.Second): // Increased timeout
			fmt.Println("hasher-host shutdown timeout, force killing...")
			cmd.Process.Kill()
		}
	} else if port > 0 { // It was running, but we didn't start it. We can still ask it to shut down.
		shutdownURL := fmt.Sprintf("http://localhost:%d/api/v1/shutdown", port)
		resp, err := http.Post(shutdownURL, "application/json", nil)
		if err == nil {
			resp.Body.Close()
			fmt.Println("Shutdown request sent to pre-existing hasher-host.")
		}
	}
}
