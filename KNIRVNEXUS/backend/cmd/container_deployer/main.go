package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

//go:embed all:scripts/*
//go:embed container.yaml
//go:embed all:ansible/*
//go:embed all:golang-app-source/*
//go:embed pod.yaml

var embeddedFiles embed.FS

// Global logger for deployment
var deploymentLogger *log.Logger
var deploymentLogFile *os.File

const (
	appName              = "knirvnexus"
	ansibleCloudDir      = "ansible/cloud-deploy"
	ansibleLocalDir      = "ansible/local-deploy"
	golangAppSourceDir   = "golang-app-source"
	outputKataGuestDir   = "output-kata-guest"
	kataConfigDir        = "/etc/kata-containers"
	customKataKernelName = "kali-clean-tee"
	containerImageName   = "knirvnexus-go-app"
	artifactsDirName     = "artifacts"
)

// initDeploymentLog initializes the deployment logger with a timestamped log file
// in a log subfolder of the current working directory
func initDeploymentLog() error {
	// Get the current working directory
	cwdLogDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}

	// Create log subfolder in CWD
	cwdLogsDir := filepath.Join(cwdLogDir, "log")
	if err := os.MkdirAll(cwdLogsDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Create timestamped log file
	timestamp := time.Now().Format("20060102_150405")
	logFileName := fmt.Sprintf("nexus-deployment_%s.log", timestamp)
	logFilePath := filepath.Join(cwdLogsDir, logFileName)

	// Create log file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file at %s: %v", logFilePath, err)
	}

	deploymentLogFile = logFile

	// Create logger that writes to stdout and log file
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	deploymentLogger = log.New(multiWriter, "", log.LstdFlags)

	// Also configure the standard logger to write to file
	log.SetOutput(multiWriter)

	deploymentLogger.Printf("========================================")
	deploymentLogger.Printf("KNIRV-NEXUS Deployment Log")
	deploymentLogger.Printf("Started: %s", time.Now().Format("2006-01-02 15:04:05"))
	deploymentLogger.Printf("Log File: %s", logFilePath)
	deploymentLogger.Printf("========================================")

	return nil
}

// closeDeploymentLog closes the deployment log file
func closeDeploymentLog() {
	if deploymentLogFile != nil {
		deploymentLogger.Printf("========================================")
		deploymentLogger.Printf("Deployment completed at: %s", time.Now().Format("2006-01-02 15:04:05"))
		deploymentLogger.Printf("========================================")
		deploymentLogFile.Close()
	}
}

// getAppDataDirectory returns the Linux XDG Base Directory for app data
// Following Linux standards: ~/.local/share/knirvnexus/container_deployer
func getAppDataDirectory() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}

	// Use ~/.local/share/knirvnexus/container_deployer (XDG Base Directory Specification)
	appDataDir := filepath.Join(usr.HomeDir, ".local", "share", appName, "container_deployer")
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create app data directory %s: %v", appDataDir, err)
	}

	return appDataDir, nil
}

// getOsBuilderArtifactDirectory returns the os_builder's artifact directory
// where Kata container artifacts are stored
func getOsBuilderArtifactDirectory() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}

	// Path to os_builder's artifact directory
	osBuilderArtifactDir := filepath.Join(usr.HomeDir, ".local", "share", appName, "os_builder", artifactsDirName)
	return osBuilderArtifactDir, nil
}

// getArtifactDirectory returns the directory where built artifacts are stored
func getArtifactDirectory(appDataDir string) string {
	return filepath.Join(appDataDir, artifactsDirName)
}

// getEmbeddedResourcesDirectory returns the directory where embedded files are extracted
func getEmbeddedResourcesDirectory(appDataDir string) string {
	return filepath.Join(appDataDir, "resources")
}

func main() {
	// Initialize deployment logging
	if err := initDeploymentLog(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize deployment logging: %v\n", err)
	}
	defer closeDeploymentLog()

	fmt.Println("KNIRV-NEXUS Deployment Orchestrator")
	fmt.Println("----------------------------------")

	if runtime.GOOS != "linux" {
		log.Fatalf("This orchestrator is designed for Linux (Ubuntu 22.04) hosts. Detected: %s", runtime.GOOS)
	}

	// Get persistent app data directory
	appDataDir, err := getAppDataDirectory()
	if err != nil {
		log.Fatalf("Failed to initialize app data directory: %v", err)
	}
	log.Printf("App data directory: %s", appDataDir)

	// Create artifact directory for storing built images
	artifactDir := getArtifactDirectory(appDataDir)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		log.Fatalf("Failed to create artifact directory: %v", err)
	}

	// Create resources directory for embedded files
	resourcesDir := getEmbeddedResourcesDirectory(appDataDir)
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		log.Fatalf("Failed to create resources directory: %v", err)
	}

	log.Printf("Resources directory: %s", resourcesDir)
	log.Printf("Artifacts directory: %s", artifactDir)

	err = extractEmbeddedFiles(resourcesDir)
	if err != nil {
		log.Fatalf("Failed to extract embedded files: %v", err)
	}

	fmt.Println("\nChecking system prerequisites...")
	if err := checkPrerequisites(); err != nil {
		log.Fatalf("Prerequisite check failed: %v", err)
	}
	fmt.Println("All prerequisites met.")

	// Check for Kata container artifacts built by os_builder
	kataArtifactsAvailable := checkKataArtifacts()

	// Parse command-line flags for non-interactive mode
	action := flag.String("action", "", "Action to perform: 1=Deploy container, 2=Deploy native, 3=Install Go app only, 4=Exit")
	deployType := flag.String("deploy-type", "", "Deployment type: local or cloud")
	deployMode := flag.String("deploy-mode", "", "Deployment mode: container, kata, or native")
	envFlag := flag.String("env", "", "Environment: development, testnet, or production")
	showBuild := flag.Bool("show-build", false, "Show verbose Docker build output")
	flag.Parse()

	// Determine deployment type and mode
	var selectedDeployType string
	var selectedDeployMode string
	if *deployType != "" {
		selectedDeployType = *deployType
		if selectedDeployType != "local" && selectedDeployType != "cloud" {
			log.Fatalf("Invalid deployment type: %s (must be 'local' or 'cloud')", selectedDeployType)
		}
	} else {
		// Ask user interactively
		fmt.Println("\nSelect deployment type:")
		fmt.Println("1. Local deployment")
		fmt.Println("2. Cloud deployment")
		fmt.Print("Enter your choice (1 or 2): ")
		var choice string
		fmt.Scanln(&choice)
		switch choice {
		case "1":
			selectedDeployType = "local"
		case "2":
			selectedDeployType = "cloud"
		default:
			log.Fatalf("Invalid choice: %s", choice)
		}
	}

	// Determine deployment mode (container, kata, native)
	if *deployMode != "" {
		selectedDeployMode = *deployMode
		if selectedDeployMode != "container" && selectedDeployMode != "kata" && selectedDeployMode != "native" {
			log.Fatalf("Invalid deployment mode: %s (must be 'container', 'kata', or 'native')", selectedDeployMode)
		}
	} else {
		// Ask user interactively for deployment mode
		fmt.Println("\nSelect deployment mode:")
		fmt.Println("1. Container-based (Docker/Kata)")
		fmt.Println("2. Native binary (direct on OS)")
		fmt.Print("Enter your choice (1 or 2): ")
		var modeChoice string
		fmt.Scanln(&modeChoice)
		switch modeChoice {
		case "1":
			selectedDeployMode = "container"
		case "2":
			selectedDeployMode = "native"
		default:
			log.Fatalf("Invalid choice: %s", modeChoice)
		}
	}

	fmt.Printf("\n✓ Using %s deployment configuration with %s mode\n", selectedDeployType, selectedDeployMode)

	// Select deployment environment (dev/testnet/production)
	var environment string
	if *envFlag != "" {
		// Use flag value if provided
		environment = *envFlag
		if environment != "development" && environment != "testnet" && environment != "production" {
			log.Fatalf("Invalid environment: %s (must be 'development', 'testnet', or 'production')", environment)
		}
	} else {
		// Ask user interactively
		fmt.Println("\nSelect deployment environment:")
		fmt.Println("1. Development (.env.development)")
		fmt.Println("2. Testnet (.env.testnet)")
		fmt.Println("3. Production (will prompt for values or URL)")
		fmt.Print("Enter your choice (1, 2, or 3): ")
		var envChoice string
		fmt.Scanln(&envChoice)
		switch envChoice {
		case "1":
			environment = "development"
		case "2":
			environment = "testnet"
		case "3":
			environment = "production"
		default:
			log.Fatalf("Invalid choice: %s", envChoice)
		}
	}

	fmt.Printf("✓ Using %s environment\n", environment)
	fmt.Printf("  (The unified binary will extract embedded .env at runtime via KNIRV_ENV variable)\n")

	// If action flag provided, execute it non-interactively
	if *action != "" {
		executeActionWithDeployType(*action, resourcesDir, artifactDir, selectedDeployType, selectedDeployMode, environment, *showBuild)
		return
	}

	// Display Kata artifacts status before interactive menu
	if kataArtifactsAvailable {
		fmt.Println("\n✓ Kata container artifacts detected from os_builder.")
		fmt.Println("These artifacts will be used automatically during deployment.")
	} else {
		fmt.Println("\n⚠️  No Kata container artifacts found.")
		fmt.Println("Please run os_builder first to build the Kata container before deploying.")
		fmt.Print("\nDo you want to continue anyway? (y/N): ")
		var continueAnyway string
		fmt.Scanln(&continueAnyway)
		if continueAnyway != "y" && continueAnyway != "Y" {
			fmt.Println("Exiting. Please run os_builder first.")
			return
		}
	}

	// Otherwise, run interactive mode
	for {
		fmt.Println("\nSelect an action:")
		fmt.Println("1. Deploy KNIRV-NEXUS (container or native)")
		fmt.Println("2. Only install the KNIRV-NEXUS binary on existing setup")
		fmt.Println("3. Exit")
		fmt.Print("Enter your choice: ")

		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			fmt.Println("\nStarting KNIRV-NEXUS deployment...")
			runDeployNewContainer(resourcesDir, artifactDir, selectedDeployType, selectedDeployMode, environment, true, *showBuild)
		case "2":
			fmt.Println("\nStarting Go app install on existing setup...")
			runInstallGoAppOnly(resourcesDir, artifactDir, selectedDeployType, selectedDeployMode)
		case "3":
			fmt.Println("Exiting.")
			return
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}

// --- File Extraction ---
func extractEmbeddedFiles(dest string) error {
	return fs.WalkDir(embeddedFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dest, path), 0755)
		}

		content, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, path), content, 0644)
	})
}

// --- Prerequisite Checks ---
func checkCommand(cmd string) error {
	path, err := exec.LookPath(cmd)
	if err != nil {
		return fmt.Errorf("'%s' not found. Please install it. (%v)", cmd, err)
	}
	log.Printf("Found %s at: %s", cmd, path)
	return nil
}

func installAnsible() error {
	fmt.Println("Installing Ansible...")

	installCmd := exec.Command("bash", "-c", `
		sudo apt-get update && sudo apt-get install -y ansible
	`)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install ansible: %v", err)
	}

	fmt.Println("Ansible installed successfully")
	return nil
}

func installContainerd() error {
	fmt.Println("Installing containerd...")

	installCmd := exec.Command("bash", "-c", `
		sudo apt-get update && sudo apt-get install -y containerd
	`)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install containerd: %v", err)
	}

	fmt.Println("containerd installed successfully")
	return nil
}

func installNerdctl() error {
	fmt.Println("Installing nerdctl...")

	installCmd := exec.Command("bash", "-c", `
		wget https://github.com/containerd/nerdctl/releases/download/v1.7.1/nerdctl-1.7.1-linux-amd64.tar.gz
		sudo tar Cxzvvf /usr/local/bin nerdctl-1.7.1-linux-amd64.tar.gz
		rm nerdctl-1.7.1-linux-amd64.tar.gz
	`)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install nerdctl: %v", err)
	}

	fmt.Println("nerdctl installed successfully")
	return nil
}

func installGo() error {
	fmt.Println("Installing Go...")

	installCmd := exec.Command("bash", "-c", `
		wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
		sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
		rm go1.21.0.linux-amd64.tar.gz
		echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
		export PATH=$PATH:/usr/local/go/bin
	`)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install go: %v", err)
	}

	fmt.Println("Go installed successfully")
	return nil
}

func installSshpass() error {
	fmt.Println("Installing sshpass...")

	installCmd := exec.Command("bash", "-c", `
		sudo apt-get update && sudo apt-get install -y sshpass
	`)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install sshpass: %v", err)
	}

	fmt.Println("sshpass installed successfully")
	return nil
}

func checkPrerequisites() error {
	cmds := []string{"ansible-playbook", "containerd", "nerdctl", "go", "sshpass"}

	for _, cmd := range cmds {
		if err := checkCommand(cmd); err != nil {
			fmt.Printf("Missing prerequisite: %s\n", cmd)
			fmt.Printf("Attempting to install %s...\n", cmd)

			var installErr error
			switch cmd {
			case "ansible-playbook":
				installErr = installAnsible()
			case "containerd":
				installErr = installContainerd()
			case "nerdctl":
				installErr = installNerdctl()
			case "go":
				installErr = installGo()
			case "sshpass":
				installErr = installSshpass()
			}

			if installErr != nil {
				return fmt.Errorf("failed to install %s: %v", cmd, installErr)
			}

			// Verify installation after attempting to install
			if err := checkCommand(cmd); err != nil {
				return fmt.Errorf("%s installation completed but command not found: %v", cmd, err)
			}
		}
	}

	// Check for TEE specific device nodes/modules on host
	// This is highly dependent on your TEE (SGX, SEV-SNP, TDX)
	// Example for SGX:
	if _, err := os.Stat("/dev/sgx/enclave"); os.IsNotExist(err) {
		log.Printf("Warning: /dev/sgx/enclave not found. SGX TEE passthrough may not work.")
		log.Printf("Ensure Intel SGX is enabled in BIOS, 'isgx' kernel module is loaded, and sgx-utils are installed on your host system.")
	} else if err != nil {
		return fmt.Errorf("error checking /dev/sgx/enclave: %v", err)
	} else {
		log.Println("/dev/sgx/enclave found. SGX TEE support seems present.")
	}

	return nil
}

// --- Artifact Management ---
// checkKataArtifacts checks if Kata container artifacts built by os_builder are available
func checkKataArtifacts() bool {
	// Get os_builder artifact directory
	osBuilderArtifactDir, err := getOsBuilderArtifactDirectory()
	if err != nil {
		log.Printf("Failed to get os_builder artifact directory: %v", err)
		return false
	}

	kataArtifactsDir := filepath.Join(osBuilderArtifactDir, outputKataGuestDir)
	kernelPath := filepath.Join(kataArtifactsDir, "vmlinuz-"+customKataKernelName)
	rootfsPath := filepath.Join(kataArtifactsDir, "kata-rootfs-"+customKataKernelName+".img")

	// Check if both kernel and rootfs exist
	kernelExists := false
	rootfsExists := false

	if _, err := os.Stat(kernelPath); err == nil {
		kernelExists = true
		log.Printf("Found Kata kernel: %s", kernelPath)
	}

	if _, err := os.Stat(rootfsPath); err == nil {
		rootfsExists = true
		log.Printf("Found Kata rootfs: %s", rootfsPath)
	}

	return kernelExists && rootfsExists
}


// --- Command Execution Helper with Timeout & Signal Handling ---
func runCmd(name string, args []string, workingDir string) error {
	// No timeout - long-running processes like kernel compilation should not be interrupted
	return runCmdWithTimeout(name, args, workingDir, 24*time.Hour)
}

// runCmdWithTimeout executes a command with proper timeout and signal handling
func runCmdWithTimeout(name string, args []string, workingDir string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workingDir
	cmd.Stdin = os.Stdin // Allow interactive input for Packer/Ansible
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM, // Kill child if parent dies
	}

	// For Terraform, capture output to file to preserve full logs
	if name == "terraform" {
		timestamp := time.Now().Format("20060102_150405")
		outputLogPath := filepath.Join(workingDir, fmt.Sprintf("terraform_output_%s.log", timestamp))
		outputLogFile, err := os.OpenFile(outputLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("Warning: Failed to create terraform output log: %v", err)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		} else {
			defer outputLogFile.Close()
			// Write terraform output to both stdout and file
			cmd.Stdout = io.MultiWriter(os.Stdout, outputLogFile)
			cmd.Stderr = io.MultiWriter(os.Stderr, outputLogFile)
			log.Printf("Terraform output will be logged to: %s", outputLogPath)
		}
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	log.Printf("Executing: %s %s (in %s) with timeout %v", name, strings.Join(args, " "), workingDir, timeout)

	// Setup signal handling to gracefully pass through signals to the subprocess
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan) // CRITICAL: Stop signal handler when function exits

	// Forward signals to the child process
	go func() {
		for sig := range sigChan {
			log.Printf("Received signal %v, forwarding to subprocess...", sig)
			if cmd.Process != nil {
				cmd.Process.Signal(sig) // Send signal to child, don't kill immediately
			}
		}
	}()

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Command exceeded timeout of %v, sending SIGTERM to subprocess...", timeout)
			if cmd.Process != nil {
				cmd.Process.Signal(syscall.SIGTERM)
				time.Sleep(2 * time.Second) // Give process time to shut down gracefully
				if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
					cmd.Process.Kill() // Force kill if still running
				}
			}
			return fmt.Errorf("command '%s %s' exceeded timeout of %v", name, strings.Join(args, " "), timeout)
		}
		return fmt.Errorf("command '%s %s' failed: %v", name, strings.Join(args, " "), err)
	}
	return nil
}

// --- Workflow Functions ---
// getAnsibleDirectory returns the appropriate ansible directory based on deployment type
func getAnsibleDirectory(resourcesDir, deployType string) string {
	if deployType == "cloud" {
		return filepath.Join(resourcesDir, ansibleCloudDir)
	}
	return filepath.Join(resourcesDir, ansibleLocalDir)
}

func runDeployNewContainer(resourcesDir, artifactDir, deployType, deployMode, environment string, tailLogs bool, showBuild bool) {
	fmt.Printf("--- Deploying new KNIRV-NEXUS (%s deployment, %s mode, %s environment) ---\n", deployType, deployMode, environment)

	// Handle native deployment
	if deployMode == "native" {
		runNativeDeployment(resourcesDir, artifactDir, deployType, environment)
		return
	}

	// Handle container-based deployment
	ansibleWorkDir := getAnsibleDirectory(resourcesDir, deployType)
	inventoryPath := filepath.Join(ansibleWorkDir, "inventory.ini")

	// Use golang-app-source from resources directory
	goAppSourcePath := filepath.Join(resourcesDir, golangAppSourceDir)

	// Verify Go app source exists in resources
	if _, err := os.Stat(goAppSourcePath); os.IsNotExist(err) {
		log.Fatalf("Go app source not found in resources at %s", goAppSourcePath)
	}

	var ansiblePlaybook string
	var extraVars []string

	if deployType == "local" {
		// Local deployment uses Docker with Kali Linux tools
		ansiblePlaybook = "deploy-docker-kali.yml"
		extraVars = []string{
			fmt.Sprintf("go_app_source_path=%s", goAppSourcePath),
			fmt.Sprintf("container_image_name=%s", containerImageName),
			fmt.Sprintf("knirv_environment=%s", environment),
			"privileged_container=true",
			fmt.Sprintf("show_build_output=%t", showBuild),
		}
	} else {
		// Cloud deployment uses Kata containers
		ansiblePlaybook = "deploy-kata-app.yml"

		// Get os_builder artifact directory for Kata artifacts
		osBuilderArtifactDir, err := getOsBuilderArtifactDirectory()
		if err != nil {
			log.Fatalf("Failed to get os_builder artifact directory: %v", err)
		}

		// Check if custom Kata artifacts exist in os_builder artifact directory
		customKernelPath := filepath.Join(osBuilderArtifactDir, outputKataGuestDir, "vmlinuz-"+customKataKernelName)
		customRootfsPath := filepath.Join(osBuilderArtifactDir, outputKataGuestDir, "kata-rootfs-"+customKataKernelName+".img")

		// Use custom artifacts if they exist, otherwise fall back to system paths
		kernelPath := customKernelPath
		rootfsPath := customRootfsPath

		if _, err := os.Stat(customKernelPath); os.IsNotExist(err) {
			log.Printf("Custom Kata kernel not found at %s, using system path", customKernelPath)
			kernelPath = filepath.Join(kataConfigDir, "vmlinuz-"+customKataKernelName)
		}

		if _, err := os.Stat(customRootfsPath); os.IsNotExist(err) {
			log.Printf("Custom Kata rootfs not found at %s, using system path", customRootfsPath)
			rootfsPath = filepath.Join(kataConfigDir, "kata-rootfs-"+customKataKernelName+".img")
		}

		extraVars = []string{
			fmt.Sprintf("custom_kata_kernel_path_local=%s", kernelPath),
			fmt.Sprintf("custom_kata_rootfs_path_local=%s", rootfsPath),
			fmt.Sprintf("go_app_source_path=%s", goAppSourcePath),
			fmt.Sprintf("container_image_name=%s", containerImageName),
			fmt.Sprintf("artifact_directory=%s", artifactDir),
			fmt.Sprintf("knirv_environment=%s", environment),
		}
	}

	ansibleArgs := []string{"-i", inventoryPath, ansiblePlaybook, "-e", strings.Join(extraVars, " ")}
	err := runCmd("ansible-playbook", ansibleArgs, ansibleWorkDir)
	if err != nil {
		log.Fatalf("Ansible container deployment failed: %v", err)
	}
	fmt.Println("--- New container deployed successfully! ---")

	// Check container status and tail logs
	containerName := "knirvnexus-kali-local"

	// Wait a moment for container to fully start
	time.Sleep(2 * time.Second)

	// Check if container is running
	fmt.Printf("\n🔍 Checking container status...\n")
	checkCmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Status}}")
	statusOutput, err := checkCmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to check container status: %v", err)
	} else {
		status := strings.TrimSpace(string(statusOutput))
		if status != "" {
			fmt.Printf("✓ Container is running: %s\n", status)
		} else {
			fmt.Printf("⚠️  Container may not be running. Checking all containers...\n")
			allCmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Status}}")
			allOutput, _ := allCmd.Output()
			fmt.Printf("Status: %s\n", strings.TrimSpace(string(allOutput)))
		}
	}

	// Tail logs only if requested (interactive mode)
	if tailLogs {
		// Tail the container logs
		fmt.Printf("\n📜 Tailing container logs (Press Ctrl+C to exit)...\n")
		fmt.Println("-----------------------------------------------------------")

		logsCmd := exec.Command("docker", "logs", "-f", containerName)
		logsCmd.Stdout = os.Stdout
		logsCmd.Stderr = os.Stderr

		// Set up signal handling to allow graceful exit with Ctrl+C
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Start tailing logs in a goroutine
		logsDone := make(chan error, 1)
		go func() {
			logsDone <- logsCmd.Run()
		}()

		// Wait for either logs to finish or user interrupt
		select {
		case <-sigChan:
			fmt.Println("\n\n📋 Log tailing stopped by user")
			if logsCmd.Process != nil {
				logsCmd.Process.Kill()
			}
		case err := <-logsDone:
			if err != nil {
				log.Printf("\nLog tailing ended: %v", err)
			}
		}
	} else {
		fmt.Println("\n✓ Container deployed and running")
		fmt.Printf("  View logs: docker logs %s\n", containerName)
		fmt.Printf("  Follow logs: docker logs -f %s\n", containerName)
		fmt.Printf("  Access shell: docker exec -it %s /bin/bash\n", containerName)
	}

	fmt.Println("\n✓ Deployment session complete")
}

func runInstallGoAppOnly(resourcesDir, artifactDir, deployType, deployMode string) {
	fmt.Printf("--- Only installing KNIRV-NEXUS Go App on existing setup (%s deployment, %s mode) ---\n", deployType, deployMode)

	// Handle native deployment binary installation
	if deployMode == "native" {
		runNativeBinaryInstall(resourcesDir, artifactDir, deployType)
		return
	}

	ansibleWorkDir := getAnsibleDirectory(resourcesDir, deployType)
	inventoryPath := filepath.Join(ansibleWorkDir, "inventory.ini")

	// Use golang-app-source from resources directory
	goAppSourcePath := filepath.Join(resourcesDir, golangAppSourceDir)

	// Verify Go app source exists in resources
	if _, err := os.Stat(goAppSourcePath); os.IsNotExist(err) {
		log.Fatalf("Go app source not found in resources at %s", goAppSourcePath)
	}
	log.Printf("Using Go app source from resources: %s", goAppSourcePath)

	// Check for any existing Kata artifacts in os_builder directory
	osBuilderArtifactDir, err := getOsBuilderArtifactDirectory()
	if err != nil {
		log.Printf("Warning: Failed to get os_builder artifact directory: %v", err)
	} else {
		kataArtifactsDir := filepath.Join(osBuilderArtifactDir, outputKataGuestDir)
		if _, err := os.Stat(kataArtifactsDir); err == nil {
			log.Printf("Found Kata artifacts directory: %s", kataArtifactsDir)
		}
	}

	extraVars := []string{
		fmt.Sprintf("go_app_source_path=%s", goAppSourcePath),
		fmt.Sprintf("container_image_name=%s", containerImageName),
		fmt.Sprintf("artifact_directory=%s", artifactDir),
	}

	ansibleArgs := []string{"-i", inventoryPath, "deploy-kata-app.yml", "--tags", "go_app_build,go_app_run", "-e", strings.Join(extraVars, " ")}
	err = runCmd("ansible-playbook", ansibleArgs, ansibleWorkDir)
	if err != nil {
		log.Fatalf("Ansible Go app installation failed: %v", err)
	}
	fmt.Println("--- KNIRV-NEXUS Go App installed successfully! ---")
}

// --- Native Deployment Functions ---

// runNativeDeployment handles native deployment to EC2 or local system
func runNativeDeployment(resourcesDir, artifactDir, deployType, environment string) {
	fmt.Printf("--- Deploying knirv-nexus NATIVELY (%s deployment, %s environment) ---\n", deployType, environment)

	// Build the binary first
	fmt.Println("Building knirv-nexus binary...")
	binaryPath := buildKNIRVNexusBinary(artifactDir, environment)

	if deployType == "local" {
		// Local native deployment (for testing)
		runLocalNativeDeployment(binaryPath, environment)
	} else {
		// Cloud native deployment to AWS EC2
		runCloudNativeDeployment(resourcesDir, binaryPath, environment)
	}
}

// buildKNIRVNexusBinary builds the knirv-nexus binary
func buildKNIRVNexusBinary(artifactDir, environment string) string {
	fmt.Println("Compiling knirv-nexus with embedded environment...")

	// Navigate to KNIRVNEXUS directory
	knirvNexusDir := filepath.Join(getCurrentRepoRoot(), "KNIRVNEXUS")

	// Build output path
	outputPath := filepath.Join(artifactDir, "knirv-nexus")

	buildCmd := exec.Command("bash", "-c", fmt.Sprintf(`
		cd %s &&
		go mod tidy &&
		CGO_ENABLED=1 go build \
			-tags 'embed_%s' \
			-ldflags="-s -w -X main.Version=$(git describe --tags --always) -X main.BuildTime=$(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)" \
			-o %s \
			.
	`, knirvNexusDir, environment, outputPath))

	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		log.Fatalf("Failed to build knirv-nexus binary: %v", err)
	}

	fmt.Printf("✓ Binary built successfully: %s\n", outputPath)
	return outputPath
}

// getCurrentRepoRoot returns the current repository root directory
func getCurrentRepoRoot() string {
	// Try to find the repo root by looking for KNIRVNEXUS directory
	cwd, err := os.Getwd()
	if err != nil {
		return "/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK"
	}
	return cwd
}

// runCloudNativeDeployment deploys to AWS EC2 using Ansible
func runCloudNativeDeployment(resourcesDir, binaryPath, environment string) {
	ansibleWorkDir := filepath.Join(resourcesDir, ansibleCloudDir, "playbooks")
	playbookPath := filepath.Join(ansibleWorkDir, "deploy-native-kali.yml")

	// Get dynamic inventory for AWS EC2
	inventoryPath := filepath.Join(resourcesDir, ansibleCloudDir, "inventory", "aws_ec2.yml")

	extraVars := []string{
		fmt.Sprintf("binary_source_path=%s", binaryPath),
		fmt.Sprintf("env=%s", environment),
	}

	ansibleCmd := []string{
		"ansible-playbook",
		"-i", inventoryPath,
		playbookPath,
	}

	for _, v := range extraVars {
		ansibleCmd = append(ansibleCmd, "-e", v)
	}

	fmt.Printf("Running Ansible playbook: %s\n", playbookPath)
	fmt.Printf("Extra vars: %v\n", extraVars)

	err := runCmd(ansibleCmd[0], ansibleCmd[1:], ansibleWorkDir)
	if err != nil {
		log.Fatalf("Ansible native deployment failed: %v", err)
	}

	fmt.Println("✓ Native deployment to AWS EC2 completed successfully!")
}

// runLocalNativeDeployment deploys locally for testing
func runLocalNativeDeployment(binaryPath, environment string) {
	fmt.Println("--- Local Native Deployment ---")
	fmt.Printf("Binary: %s\n", binaryPath)
	fmt.Printf("Environment: %s\n", environment)
	fmt.Println("")
	fmt.Println("To run locally:")
	fmt.Printf("  sudo %s --config /etc/knirv-nexus/production.yaml\n", binaryPath)
	fmt.Println("")
	fmt.Println("Note: This is for testing only. For production, use cloud deployment.")
}

// runNativeBinaryInstall installs the binary on an already provisioned system
func runNativeBinaryInstall(resourcesDir, artifactDir, deployType string) {
	fmt.Println("--- Installing KNIRV-NEXUS Binary on Existing System ---")

	// Build the binary
	binaryPath := buildKNIRVNexusBinary(artifactDir, "production")

	// For local deployment, just show instructions
	if deployType == "local" {
		fmt.Printf("\nBinary built at: %s\n", binaryPath)
		fmt.Println("Copy to target system and run:")
		fmt.Printf("  sudo cp %s /usr/local/bin/knirv-nexus\n", binaryPath)
		fmt.Printf("  sudo chmod +x /usr/local/bin/knirv-nexus\n")
		fmt.Println("")
	}

	fmt.Println("✓ Binary installation prepared!")
}

// executeActionWithDeployType performs the specified action non-interactively with deployment type
func executeActionWithDeployType(action string, resourcesDir, artifactDir, deployType, deployMode, environment string, showBuild bool) {
	switch action {
	case "1":
		fmt.Println("\nStarting KNIRV-NEXUS deployment...")
		runDeployNewContainer(resourcesDir, artifactDir, deployType, deployMode, environment, false, showBuild)
	case "2":
		fmt.Println("\nStarting Go app install on existing setup...")
		runInstallGoAppOnly(resourcesDir, artifactDir, deployType, deployMode)
	case "3":
		fmt.Println("Exiting.")
	default:
		fmt.Printf("Invalid action: %s\n", action)
		fmt.Println("Valid actions are: 1, 2, or 3")
		os.Exit(1)
	}
}
