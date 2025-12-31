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
//go:embed golang-app-source/knirv-nexus
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
	outputKataGuestDir   = "output-kata-guest" // Where built kernel/rootfs artifacts are stored
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

	// Copy golang-app-source to artifacts directory for persistence
	err = ensureGoAppSourceInArtifacts(resourcesDir, artifactDir)
	if err != nil {
		log.Fatalf("Failed to ensure Go app source in artifacts: %v", err)
	}

	fmt.Println("\nChecking system prerequisites...")
	if err := checkPrerequisites(); err != nil {
		log.Fatalf("Prerequisite check failed: %v", err)
	}
	fmt.Println("All prerequisites met.")

	// Check for Kata container artifacts built by os_builder
	kataArtifactsAvailable := checkKataArtifacts()

	// Parse command-line flags for non-interactive mode
	action := flag.String("action", "", "Action to perform: 1=Deploy new container, 2=Install Go app only, 3=Exit")
	deployType := flag.String("deploy-type", "", "Deployment type: local or cloud")
	flag.Parse()

	// Determine deployment type
	var selectedDeployType string
	if *deployType != "" {
		// Use flag value if provided
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

	fmt.Printf("\n✓ Using %s deployment configuration\n", selectedDeployType)

	// If action flag provided, execute it non-interactively
	if *action != "" {
		executeActionWithDeployType(*action, resourcesDir, artifactDir, selectedDeployType)
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
		fmt.Println("1. Deploy a new KNIRV-NEXUS Kata Container (assuming Kata is configured)")
		fmt.Println("2. Only install the KNIRV-NEXUS Go App on an existing, configured Kata setup")
		fmt.Println("3. Exit")
		fmt.Print("Enter your choice: ")

		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			fmt.Println("\nStarting new container deploy (assuming Kata configured)...")
			runDeployNewContainer(resourcesDir, artifactDir, selectedDeployType)
		case "2":
			fmt.Println("\nStarting Go app install on existing Kata setup...")
			runInstallGoAppOnly(resourcesDir, artifactDir, selectedDeployType)
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

func ensureGoAppSourceInArtifacts(resourcesDir, artifactDir string) error {
	// Source and destination paths
	srcAppSourcePath := filepath.Join(resourcesDir, golangAppSourceDir)
	dstAppSourcePath := filepath.Join(artifactDir, golangAppSourceDir)

	// Check if source exists
	if _, err := os.Stat(srcAppSourcePath); os.IsNotExist(err) {
		return fmt.Errorf("golang app source not found at %s", srcAppSourcePath)
	}

	// Check if destination already exists (don't overwrite, but ensure permissions)
	if _, err := os.Stat(dstAppSourcePath); err == nil {
		log.Printf("Go app source already exists in artifacts at %s", dstAppSourcePath)
		// Ensure the binary has execute permissions even if it already exists
		binaryPath := filepath.Join(dstAppSourcePath, "knirv-nexus")
		chmodCmd := exec.Command("chmod", "+x", binaryPath)
		if err := chmodCmd.Run(); err != nil {
			return fmt.Errorf("failed to set execute permissions on existing binary: %v", err)
		}
		return nil
	}

	// Copy entire directory from resources to artifacts
	log.Printf("Copying Go app source from resources to artifacts...")
	copyCmd := exec.Command("cp", "-r", srcAppSourcePath, dstAppSourcePath)
	if err := copyCmd.Run(); err != nil {
		return fmt.Errorf("failed to copy Go app source to artifacts: %v", err)
	}

	// Ensure the binary has execute permissions
	binaryPath := filepath.Join(dstAppSourcePath, "knirv-nexus")
	chmodCmd := exec.Command("chmod", "+x", binaryPath)
	if err := chmodCmd.Run(); err != nil {
		return fmt.Errorf("failed to set execute permissions on binary: %v", err)
	}

	log.Printf("✓ Go app source copied to artifacts: %s", dstAppSourcePath)
	return nil
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

func runDeployNewContainer(resourcesDir, artifactDir, deployType string) {
	fmt.Printf("--- Deploying new KNIRV-NEXUS Container (%s deployment) ---\n", deployType)
	ansibleWorkDir := getAnsibleDirectory(resourcesDir, deployType)
	inventoryPath := filepath.Join(ansibleWorkDir, "inventory.ini")

	// Use golang-app-source from container_deployer's own artifact directory
	// This is placed here during the container_deployer build process via Makefile
	goAppSourcePath := filepath.Join(artifactDir, golangAppSourceDir)

	// Verify Go app source exists in container_deployer's artifacts
	if _, err := os.Stat(goAppSourcePath); os.IsNotExist(err) {
		log.Fatalf("Go app source not found in container_deployer artifacts at %s", goAppSourcePath)
	}

	var ansiblePlaybook string
	var extraVars []string

	if deployType == "local" {
		// Local deployment uses Docker
		ansiblePlaybook = "deploy-docker-app.yml"
		extraVars = []string{
			fmt.Sprintf("go_app_source_path=%s", goAppSourcePath),
			fmt.Sprintf("container_image_name=%s", containerImageName),
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
			fmt.Sprintf("artifact_directory=%s", artifactDir), // Pass artifact directory to Ansible
		}
	}

	ansibleArgs := []string{"-i", inventoryPath, ansiblePlaybook, "-e", strings.Join(extraVars, " ")}
	err := runCmd("ansible-playbook", ansibleArgs, ansibleWorkDir)
	if err != nil {
		log.Fatalf("Ansible container deployment failed: %v", err)
	}
	fmt.Println("--- New container deployed successfully! ---")
}

func runInstallGoAppOnly(resourcesDir, artifactDir, deployType string) {
	fmt.Printf("--- Only installing KNIRV-NEXUS Go App on existing Kata setup (%s deployment) ---\n", deployType)
	ansibleWorkDir := getAnsibleDirectory(resourcesDir, deployType)
	inventoryPath := filepath.Join(ansibleWorkDir, "inventory.ini")

	// Use golang-app-source from container_deployer's own artifact directory
	// This is placed here during the container_deployer build process via Makefile
	goAppSourcePath := filepath.Join(artifactDir, golangAppSourceDir)

	// Verify that the artifact directory exists and contains expected structure
	if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
		log.Fatalf("Artifact directory %s does not exist", artifactDir)
	}
	log.Printf("Using container_deployer artifact directory: %s", artifactDir)

	// Verify Go app source exists in container_deployer's artifacts
	if _, err := os.Stat(goAppSourcePath); os.IsNotExist(err) {
		log.Fatalf("Go app source not found in container_deployer artifacts at %s", goAppSourcePath)
	}
	log.Printf("Using Go app source from container_deployer artifacts: %s", goAppSourcePath)

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
		fmt.Sprintf("artifact_directory=%s", artifactDir), // Pass artifact directory to Ansible
	}

	// This mode needs specific tags for Ansible.
	// We need to modify the deploy-kata-app.yml to have tags for different sections,
	// e.g., 'kata_config', 'go_app_build', 'go_app_run'.
	// For this example, let's assume tags 'go_app_build' and 'go_app_run' exist.
	ansibleArgs := []string{"-i", inventoryPath, "deploy-kata-app.yml", "--tags", "go_app_build,go_app_run", "-e", strings.Join(extraVars, " ")}
	err = runCmd("ansible-playbook", ansibleArgs, ansibleWorkDir)
	if err != nil {
		log.Fatalf("Ansible Go app installation failed: %v", err)
	}
	fmt.Println("--- KNIRV-NEXUS Go App installed successfully! ---")
}

// executeActionWithDeployType performs the specified action non-interactively with deployment type
func executeActionWithDeployType(action string, resourcesDir, artifactDir, deployType string) {
	switch action {
	case "1":
		fmt.Println("\nStarting new container deploy (assuming Kata configured)...")
		runDeployNewContainer(resourcesDir, artifactDir, deployType)
	case "2":
		fmt.Println("\nStarting Go app install on existing Kata setup...")
		runInstallGoAppOnly(resourcesDir, artifactDir, deployType)
	case "3":
		fmt.Println("Exiting.")
	default:
		fmt.Printf("Invalid action: %s\n", action)
		fmt.Println("Valid actions are: 1, 2, or 3")
		os.Exit(1)
	}
}
