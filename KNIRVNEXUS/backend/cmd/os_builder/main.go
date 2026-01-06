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

//go:embed all:packer-base-kali/*
//go:embed all:packer-kata-guest/*
//go:embed all:packer-aws-kali/*
// Added for Kali Docker image build
//go:embed all:packer-kali-docker/*
//go:embed inventory.ini
//go:embed all:terraform-deploy/*

var embeddedFiles embed.FS

// Global logger for deployment
var deploymentLogger *log.Logger
var deploymentLogFile *os.File

const (
	appName              = "knirvnexus"
	packerBaseKaliDir    = "packer-base-kali"
	packerKaliDockerDir  = "packer-kali-docker" // New constant
	outputBaseKaliDir    = "output-kali-base-box"
	baseKaliOVAName      = "kali-base-box.ova"
	packerAWSKaliDir     = "packer-aws-kali"
	terraformDeployDir   = "terraform-deploy"
	outputKataGuestDir   = "output-kata-guest"
	customKataKernelName = "kali-clean-tee"
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
// Following Linux standards: ~/.local/share/knirvnexus/os_builder
func getAppDataDirectory() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}

	// Use ~/.local/share/knirvnexus/os_builder (XDG Base Directory Specification)
	appDataDir := filepath.Join(usr.HomeDir, ".local", "share", appName, "os_builder")
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create app data directory %s: %v", appDataDir, err)
	}

	return appDataDir, nil
}

// getArtifactDirectory returns the directory where built artifacts are stored
func getArtifactDirectory(appDataDir string) string {
	return filepath.Join(appDataDir, artifactsDirName)
}

// getEmbeddedResourcesDirectory returns the directory where embedded files are extracted
func getEmbeddedResourcesDirectory(appDataDir string) string {
	return filepath.Join(appDataDir, "resources")
}

// getContainerDeployerBinaryPath returns the path to the knirv-nexus binary in container_deployer's golang-app-source
func getContainerDeployerBinaryPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}
	// Path to knirv-nexus binary in container_deployer's golang-app-source
	binaryPath := filepath.Join(usr.HomeDir, ".local", "share", appName, "container_deployer", "resources", "golang-app-source", "knirv-nexus")
	
	// Verify the binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return "", fmt.Errorf("knirv-nexus binary not found at expected location: %s. Please ensure 'make binary' has been run.", binaryPath)
	} else if err != nil {
		return "", fmt.Errorf("error checking for knirv-nexus binary: %v", err)
	}

	return binaryPath, nil
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

	// Check if base Kali image exists in artifact directory
	baseKaliOutputDir := filepath.Join(artifactDir, outputBaseKaliDir)
	baseKaliPath := filepath.Join(baseKaliOutputDir, baseKaliOVAName)
	hasBaseKali := isBaseKaliImageAvailable(baseKaliPath)

	// Display status of base Kali image
	if !hasBaseKali {
		fmt.Println("\n⚠️  Base Kali image (kali-base-box.ova) not found.")
		fmt.Println("This is required as the foundation for the Kata container build.")
		fmt.Println("You can build it using option 0 from the menu below.")
	} else {
		fmt.Printf("\n✓ Base Kali image found at: %s\n", baseKaliPath)
	}

	// Parse command-line flags for non-interactive mode
	action := flag.String("action", "", "Action to perform: 0=Build base Kali, 1=Build Kali Docker Image, 2=Build Kata Container, 3=Build AWS AMI, 4=Exit")
	flag.Parse()

	// If action flag provided, execute it non-interactively
	if *action != "" {
		executeAction(*action, resourcesDir, artifactDir)
		return
	}

	// Otherwise, run interactive mode
	for {
		fmt.Println("\nSelect an action:")
		fmt.Println("0. Build base Kali image (packer-base-kali) - only if rebuilding")
		fmt.Println("1. Build Kali Docker Image")
		fmt.Println("2. Build KNIRV-NEXUS Kata Container (using Terraform)")
		fmt.Println("3. Build AWS AMI (Kali Linux for native deployment)")
		fmt.Println("4. Exit")
		fmt.Print("Enter your choice: ")

		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "0":
			fmt.Println("\nRebuilding base Kali image...")
			runBuildBaseKali(resourcesDir, artifactDir)
			fmt.Println("\n✓ Base Kali image rebuilt successfully!")
		case "1":
			fmt.Println("\nBuilding Kali Docker Image...")
			runBuildKaliDocker(resourcesDir, artifactDir)
		case "2":
			fmt.Println("\nStarting Kata Container build (using Terraform)...")
			runBuildKataContainer(resourcesDir, artifactDir)
		case "3":
			fmt.Println("\nBuilding AWS AMI (Kali Linux for native deployment)...")
			runBuildAWSAMI(resourcesDir, artifactDir)
		case "4":
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

func installPacker() error {
	fmt.Println("Installing Packer...")

	// Download and install Packer
	installCmd := exec.Command("bash", "-c", `
		curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
		echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
		sudo apt-get update && sudo apt-get install -y packer
	`)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install packer: %v", err)
	}

	// Install required Packer plugins
	fmt.Println("Installing Packer plugins...")
	pluginCmd := exec.Command("bash", "-c", `
		packer init
	`)
	pluginCmd.Stdout = os.Stdout
	pluginCmd.Stderr = os.Stderr
	if err := pluginCmd.Run(); err != nil {
		return fmt.Errorf("failed to install packer plugins: %v", err)
	}

	fmt.Println("Packer and plugins installed successfully")
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

func installVirtualBox() error {
	fmt.Println("Installing VirtualBox...")

	installCmd := exec.Command("bash", "-c", `
		sudo apt-get update && sudo apt-get install -y virtualbox
	`)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install virtualbox: %v", err)
	}

	fmt.Println("VirtualBox installed successfully")
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

func installTerraform() error {
	fmt.Println("Installing Terraform...")

	installCmd := exec.Command("bash", "-c", `
		curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
		echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
		sudo apt-get update && sudo apt-get install -y terraform
	`)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install terraform: %v", err)
	}

	fmt.Println("Terraform installed successfully")
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
	cmds := []string{"packer", "ansible-playbook", "VBoxManage", "containerd", "nerdctl", "go", "terraform", "sshpass"}

	for _, cmd := range cmds {
		if err := checkCommand(cmd); err != nil {
			fmt.Printf("Missing prerequisite: %s\n", cmd)
			fmt.Printf("Attempting to install %s...\n", cmd)

			var installErr error
			switch cmd {
			case "packer":
				installErr = installPacker()
			case "ansible-playbook":
				installErr = installAnsible()
			case "VBoxManage":
				installErr = installVirtualBox()
			case "containerd":
				installErr = installContainerd()
			case "nerdctl":
				installErr = installNerdctl()
			case "go":
				installErr = installGo()
			case "terraform":
				installErr = installTerraform()
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

// --- AWS AMI Artifact Management ---

// isAWSAMIBuildAvailable checks if the AWS AMI build directory exists
func isAWSAMIBuildAvailable(resourcesDir string) bool {
	awsKaliWorkDir := filepath.Join(resourcesDir, packerAWSKaliDir)
	if _, err := os.Stat(awsKaliWorkDir); err == nil {
		return true
	}
	return false
}

// runBuildAWSAMI builds a Kali Linux AMI for AWS native deployment
func runBuildAWSAMI(resourcesDir, _ string) {
	fmt.Println("--- Building AWS AMI (Kali Linux for Native Deployment) ---")

	// Check AWS prerequisites
	if !isAWSConfigured() {
		fmt.Println("⚠️  AWS credentials not configured.")
		fmt.Println("Please set the following environment variables:")
		fmt.Println("  export AWS_ACCESS_KEY_ID=your_access_key")
		fmt.Println("  export AWS_SECRET_ACCESS_KEY=your_secret_key")
		fmt.Println("  export AWS_DEFAULT_REGION=us-east-1")
		log.Fatalf("AWS credentials not configured. Cannot build AMI.")
	}

	awsKaliWorkDir := filepath.Join(resourcesDir, packerAWSKaliDir)

	// Initialize Packer with Amazon plugin
	fmt.Println("Initializing Packer with Amazon plugin...")
	err := runCmd("packer", []string{"init", "."}, awsKaliWorkDir)
	if err != nil {
		log.Fatalf("Failed to initialize Packer: %v", err)
	}

	// Get AWS region from environment or use default
	region := getAWSRegion()
	fmt.Printf("Using AWS region: %s\n", region)

	// Build the AMI
	fmt.Println("Building AWS AMI with Packer...")
	buildArgs := []string{
		"build",
		"-force",
		"-var", fmt.Sprintf("aws_region=%s", region),
		"-var", fmt.Sprintf("aws_ami_name=knirvnexus-kali-%s", time.Now().Format("2006-01-02")),
		"-var", "aws_instance_type=t3.medium",
		"-var", "aws_ami_description=KNIRVNEXUS Kali Linux - Native deployment ready",
		"kali-aws-ami.pkr.hcl",
	}

	err = runCmd("packer", buildArgs, awsKaliWorkDir)
	if err != nil {
		log.Fatalf("Failed to build AWS AMI: %v", err)
	}

	fmt.Println("\n✓ AWS AMI build completed successfully!")
	fmt.Println("The AMI is now available in your AWS account and can be used for native deployment.")
	fmt.Println("Use the container_deployer with --deploy-mode native to deploy KNIRV-NEXUS to EC2.")
}

// isAWSConfigured checks if AWS credentials are configured
func isAWSConfigured() bool {
	// Check for environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return false
	}
	return true
}

// getAWSRegion returns the AWS region from environment or default
func getAWSRegion() string {
	region := os.Getenv("AWS_DEFAULT_REGION")
	if region == "" {
		return "us-east-1"
	}
	return region
}

// --- Base Kali Image Management ---
func isBaseKaliImageAvailable(ovaPath string) bool {
	// Check if the OVA file exists
	if _, err := os.Stat(ovaPath); err == nil {
		return true
	}
	return false
}

func runBuildBaseKali(resourcesDir, artifactDir string) {
	fmt.Println("--- Building Base Kali Image (packer-base-kali) ---")
	packerBaseKaliWorkDir := filepath.Join(resourcesDir, packerBaseKaliDir)

	// Ensure artifact output directory exists
	baseKaliOutputDir := filepath.Join(artifactDir, outputBaseKaliDir)
	if err := os.MkdirAll(baseKaliOutputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory for base Kali image: %v", err)
	}

	fmt.Printf("Packer will output to: %s\n", baseKaliOutputDir)
	fmt.Println("Executing Packer build for base Kali image...")
	err := runCmd("packer", []string{"build", "-force", "-var", fmt.Sprintf("output_directory=%s", baseKaliOutputDir), "."}, packerBaseKaliWorkDir)
	if err != nil {
		log.Fatalf("Base Kali Packer build failed: %v", err)
	}

	// Verify the OVA file was created in the artifact output directory
	ovaPathInArtifact := filepath.Join(baseKaliOutputDir, baseKaliOVAName)
	if _, err := os.Stat(ovaPathInArtifact); err != nil {
		log.Fatalf("OVA file not found at expected location %s: %v", ovaPathInArtifact, err)
	}

	fmt.Printf("✓ Base Kali image successfully built and saved to: %s\n", ovaPathInArtifact)
}

func runBuildKaliDocker(resourcesDir, artifactDir string) {
	fmt.Println("--- Building Kali Docker Image (packer-kali-docker) ---")
	packerKaliDockerWorkDir := filepath.Join(resourcesDir, packerKaliDockerDir)

	// Get the path to the knirv-nexus binary
	knirvNexusBinaryPath, err := getContainerDeployerBinaryPath()
	if err != nil {
		log.Fatalf("Failed to get knirv-nexus binary path: %v", err)
	}

	// Initialize Packer plugins for the Docker builder
	fmt.Println("Initializing Packer plugins for Kali Docker image build...")
	err = runCmd("packer", []string{"init", "."}, packerKaliDockerWorkDir)
	if err != nil {
		log.Fatalf("Failed to initialize Packer plugins for Kali Docker image: %v", err)
	}

	fmt.Println("Executing Packer build for Kali Docker image...")
	// Packer Docker builder automatically tags the image to the local Docker daemon
	buildArgs := []string{
		"build",
		"-force",
		"-var", fmt.Sprintf("knirv_nexus_binary_path=%s", knirvNexusBinaryPath), // Pass binary path as a variable
		".",
	}
	err = runCmd("packer", buildArgs, packerKaliDockerWorkDir)
	if err != nil {
		log.Fatalf("Kali Docker Packer build failed: %v", err)
	}

	fmt.Printf("✓ Kali Docker image successfully built and tagged as 'knirvnexus-kali-base:latest'\n")
	fmt.Println("You can verify with 'docker images knirvnexus-kali-base'")

	// Export the Docker image to a tar archive and save it to the artifacts directory
	fmt.Println("Exporting Kali Docker image to artifacts directory...")
	imageTarPath := filepath.Join(artifactDir, "knirvnexus-kali-base.tar")
	
	// Ensure the artifacts directory exists
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		log.Fatalf("Failed to create artifact directory %s: %v", artifactDir, err)
	}

	saveCmdArgs := []string{"save", "-o", imageTarPath, "knirvnexus-kali-base:latest"}
	err = runCmd("docker", saveCmdArgs, "") // Run in current directory
	if err != nil {
		log.Fatalf("Failed to save Docker image to archive: %v", err)
	}
	fmt.Printf("✓ Kali Docker image saved to %s\n", imageTarPath)
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
func runBuildKataContainer(resourcesDir, artifactDir string) {
	// Ensure base Kali image exists before proceeding with Kata build
	baseKaliOutputDir := filepath.Join(artifactDir, outputBaseKaliDir)
	baseKaliPath := filepath.Join(baseKaliOutputDir, baseKaliOVAName)
	if !isBaseKaliImageAvailable(baseKaliPath) {
		fmt.Println("--- Step 0: Base Kali image not found. Building it first... ---")
		runBuildBaseKali(resourcesDir, artifactDir)
		fmt.Println()
	} else {
		fmt.Printf("✓ Using existing base Kali image: %s\n\n", baseKaliPath)
	}

	fmt.Println("--- Step 1: Building Custom Kata Kernel and Rootfs (via Terraform) ---")
	terraformWorkDir := filepath.Join(resourcesDir, terraformDeployDir)
	outputKataGuestPath := filepath.Join(artifactDir, outputKataGuestDir) // Store in artifact dir
	if err := os.MkdirAll(outputKataGuestPath, 0755); err != nil {
		log.Fatalf("Failed to create output directory for Kata artifacts: %v", err)
	}

	// Create terraform.tfvars file with variables
	tfvarsPath := filepath.Join(terraformWorkDir, "terraform.tfvars")
	tfvarsContent := fmt.Sprintf(`ova_source_path          = "%s"
output_directory         = "%s"
artifact_directory       = "%s"
vm_name                  = "KataKaliGuestBuilder"
ssh_username             = "kaliadmin"
ssh_password             = "kaliadmin"
ssh_host                 = "127.0.0.1"
ssh_port                 = 22
`, baseKaliPath, outputKataGuestPath, artifactDir)

	if err := os.WriteFile(tfvarsPath, []byte(tfvarsContent), 0644); err != nil {
		log.Fatalf("Failed to write terraform.tfvars: %v", err)
	}
	log.Printf("Created terraform.tfvars at %s", tfvarsPath)

	// Initialize Terraform
	fmt.Println("Initializing Terraform...")
	err := runCmd("terraform", []string{"init"}, terraformWorkDir)
	if err != nil {
		log.Fatalf("Terraform init failed: %v", err)
	}

	// Plan Terraform
	fmt.Println("Planning Terraform configuration...")
	err = runCmd("terraform", []string{"plan", "-out=tfplan"}, terraformWorkDir)
	if err != nil {
		log.Fatalf("Terraform plan failed: %v", err)
	}

	// Apply Terraform
	fmt.Println("Applying Terraform configuration (this will take 45+ minutes for kernel compilation)...")
	err = runCmd("terraform", []string{"apply", "-auto-approve", "tfplan"}, terraformWorkDir)
	if err != nil {
		log.Fatalf("Terraform apply failed: %v", err)
	}

	fmt.Println("--- Terraform build completed. ---")

	// Verify the artifacts were created
	kernelPath := filepath.Join(outputKataGuestPath, "vmlinuz-"+customKataKernelName)
	rootfsPath := filepath.Join(outputKataGuestPath, "kata-rootfs-"+customKataKernelName+".img")

	if _, err := os.Stat(kernelPath); os.IsNotExist(err) {
		log.Fatalf("Kernel artifact not found at %s", kernelPath)
	}
	if _, err := os.Stat(rootfsPath); os.IsNotExist(err) {
		log.Fatalf("Rootfs artifact not found at %s", rootfsPath)
	}

	fmt.Printf("✓ Kernel artifact: %s\n", kernelPath)
	fmt.Printf("✓ Rootfs artifact: %s\n", rootfsPath)

	fmt.Println("\n--- Kata Container build completed successfully! ---")
	fmt.Println("Build artifacts are ready for deployment.")
}

// executeAction performs the specified action non-interactively
func executeAction(action string, resourcesDir, artifactDir string) {
	switch action {
	case "0":
		fmt.Println("\nRebuilding base Kali image...")
		runBuildBaseKali(resourcesDir, artifactDir)
		fmt.Println("\n✓ Base Kali image rebuilt successfully!")
	case "1":
		fmt.Println("\nBuilding Kali Docker Image...")
		runBuildKaliDocker(resourcesDir, artifactDir)
	case "2":
		fmt.Println("\nStarting Kata Container build (using Terraform)...")
		runBuildKataContainer(resourcesDir, artifactDir)
	case "3":
		fmt.Println("\nBuilding AWS AMI (Kali Linux for native deployment)...")
		runBuildAWSAMI(resourcesDir, artifactDir)
		fmt.Println("\n✓ AWS AMI built successfully!")
	case "4":
		fmt.Println("Exiting.")
	default:
		fmt.Printf("Invalid action: %s\n", action)
		fmt.Println("Valid actions are: 0, 1, 2, 3, or 4")
		os.Exit(1)
	}
}
