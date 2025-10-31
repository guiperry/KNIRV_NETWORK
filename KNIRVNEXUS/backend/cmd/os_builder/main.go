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
//go:embed all:ansible-deploy/*
//go:embed inventory.ini
//go:embed golang-app-source/knirv-nexus
//go:embed all:terraform-deploy/*

var embeddedFiles embed.FS

// Global logger for deployment
var deploymentLogger *log.Logger
var deploymentLogFile *os.File

const (
	appName              = "knirvnexus"
	packerBaseKaliDir    = "packer-base-kali"
	outputBaseKaliDir    = "output-kali-base-box"
	baseKaliOVAName      = "kali-base-box.ova"
	packerKataGuestDir   = "packer-kata-guest"
	terraformDeployDir   = "terraform-deploy"
	ansibleDeployDir     = "ansible-deploy"
	inventoryFile        = "inventory.ini"
	golangAppSourceDir   = "golang-app-source"
	outputKataGuestDir   = "output-kata-guest" // Where Packer drops kernel/rootfs
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
// Following Linux standards: ~/.local/share/knirvnexus/os_builder/os_builder
func getAppDataDirectory() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}

	// Use ~/.local/share/knirvnexus/os_builder/os_builder (XDG Base Directory Specification)
	appDataDir := filepath.Join(usr.HomeDir, ".local", "share", appName)
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

	// Check if base Kali image exists in artifact directory
	baseKaliOutputDir := filepath.Join(artifactDir, outputBaseKaliDir)
	baseKaliPath := filepath.Join(baseKaliOutputDir, baseKaliOVAName)
	hasBaseKali := isBaseKaliImageAvailable(baseKaliPath)

	if !hasBaseKali {
		fmt.Println("\n⚠️  Base Kali image (kali-base-box.ova) not found.")
		fmt.Println("This is required as the foundation for the Kata container build.")
		fmt.Println("Building base Kali image now...")
		runBuildBaseKali(resourcesDir, artifactDir)
		fmt.Println("\n✓ Base Kali image built successfully!")
	} else {
		fmt.Printf("\n✓ Base Kali image found at: %s\n", baseKaliPath)
	}

	// Parse command-line flags for non-interactive mode
	action := flag.String("action", "", "Action to perform: 0=Build base Kali, 1=Full deploy, 1b=Full deploy (Terraform fallback), 2=Deploy new container, 3=Install Go app only, 4=Exit")
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
		fmt.Println("1. Build and deploy a new KNIRV-NEXUS Kata Container (full fresh install with Packer)")
		fmt.Println("1b. Build and deploy a new KNIRV-NEXUS Kata Container (full fresh install with Terraform fallback)")
		fmt.Println("2. Deploy a new KNIRV-NEXUS Kata Container (assuming Kata is configured)")
		fmt.Println("3. Only install the KNIRV-NEXUS Go App on an existing, configured Kata setup")
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
			fmt.Println("\nStarting full fresh build and deploy (using Packer)...")
			runFullDeploy(resourcesDir, artifactDir)
		case "1b":
			fmt.Println("\nStarting full fresh build and deploy (using Terraform fallback)...")
			runFullDeployTerraform(resourcesDir, artifactDir)
		case "2":
			fmt.Println("\nStarting new container deploy (assuming Kata configured)...")
			runDeployNewContainer(resourcesDir, artifactDir)
		case "3":
			fmt.Println("\nStarting Go app install on existing Kata setup...")
			runInstallGoAppOnly(resourcesDir, artifactDir)
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
func ensureGoAppSourceInArtifacts(resourcesDir, artifactDir string) error {
	// Source and destination paths
	srcAppSourcePath := filepath.Join(resourcesDir, golangAppSourceDir)
	dstAppSourcePath := filepath.Join(artifactDir, golangAppSourceDir)

	// Check if source exists
	if _, err := os.Stat(srcAppSourcePath); os.IsNotExist(err) {
		return fmt.Errorf("golang app source not found at %s", srcAppSourcePath)
	}

	// Check if destination already exists (don't overwrite)
	if _, err := os.Stat(dstAppSourcePath); err == nil {
		log.Printf("Go app source already exists in artifacts at %s", dstAppSourcePath)
		return nil
	}

	// Copy entire directory from resources to artifacts
	log.Printf("Copying Go app source from resources to artifacts...")
	copyCmd := exec.Command("cp", "-r", srcAppSourcePath, dstAppSourcePath)
	if err := copyCmd.Run(); err != nil {
		return fmt.Errorf("failed to copy Go app source to artifacts: %v", err)
	}

	log.Printf("✓ Go app source copied to artifacts: %s", dstAppSourcePath)
	return nil
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
func runFullDeploy(resourcesDir, artifactDir string) {
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

	fmt.Println("--- Step 1: Building Custom Kata Kernel and Rootfs (via Packer) ---")
	packerWorkDir := filepath.Join(resourcesDir, packerKataGuestDir)
	outputKataGuestPath := filepath.Join(artifactDir, outputKataGuestDir) // Store in artifact dir
	if err := os.MkdirAll(outputKataGuestPath, 0755); err != nil {
		log.Fatalf("Failed to create output directory for Kata artifacts: %v", err)
	}

	// Copy the base Kali OVA to the packer work directory
	// so that kata-kali-guest.pkr.hcl can reference it locally
	baseKaliSource := filepath.Join(baseKaliOutputDir, baseKaliOVAName)
	baseKaliDest := filepath.Join(packerWorkDir, "packer-base-kali", "output-kali-base-box", baseKaliOVAName)
	baseKaliDestDir := filepath.Dir(baseKaliDest)
	if err := os.MkdirAll(baseKaliDestDir, 0755); err != nil {
		log.Fatalf("Failed to create base Kali reference directory: %v", err)
	}
	copyCmd := exec.Command("cp", baseKaliSource, baseKaliDest)
	if err := copyCmd.Run(); err != nil {
		log.Fatalf("Failed to copy base Kali OVA to packer work directory: %v", err)
	}

	err := runCmd("packer", []string{"build", "kata-kali-guest.pkr.hcl"}, packerWorkDir)
	if err != nil {
		log.Fatalf("Packer build failed: %v", err)
	}
	fmt.Println("--- Packer build completed. ---")

	fmt.Println("--- Step 2: Deploying Kata Containers and Go App (via Ansible) ---")
	ansibleWorkDir := filepath.Join(resourcesDir, ansibleDeployDir)
	inventoryPath := filepath.Join(resourcesDir, inventoryFile)
	goAppSourcePath := filepath.Join(artifactDir, golangAppSourceDir) // Use from artifacts for persistence

	// Verify Go app source exists in artifacts
	if _, err := os.Stat(goAppSourcePath); os.IsNotExist(err) {
		log.Fatalf("Go app source not found in artifacts at %s", goAppSourcePath)
	}

	// We need to pass the paths to the custom kernel/rootfs to the Ansible playbook
	// This can be done via Ansible extra vars (-e).
	extraVars := []string{
		fmt.Sprintf("custom_kata_kernel_path_local=%s", filepath.Join(outputKataGuestPath, "vmlinuz-"+customKataKernelName)),
		fmt.Sprintf("custom_kata_rootfs_path_local=%s", filepath.Join(outputKataGuestPath, "kata-rootfs-"+customKataKernelName+".img")),
		fmt.Sprintf("go_app_source_path=%s", goAppSourcePath),
		fmt.Sprintf("container_image_name=%s", containerImageName),
		fmt.Sprintf("artifact_directory=%s", artifactDir), // Pass artifact directory to Ansible
	}

	ansibleArgs := []string{"-i", inventoryPath, "deploy-kata-app.yml", "-e", strings.Join(extraVars, " ")}
	err = runCmd("ansible-playbook", ansibleArgs, ansibleWorkDir)
	if err != nil {
		log.Fatalf("Ansible deployment failed: %v", err)
	}
	fmt.Println("--- Full deployment completed successfully! ---")
}

func runDeployNewContainer(resourcesDir, artifactDir string) {
	fmt.Println("--- Deploying new KNIRV-NEXUS Kata Container (assuming Kata configured) ---")
	ansibleWorkDir := filepath.Join(resourcesDir, ansibleDeployDir)
	inventoryPath := filepath.Join(resourcesDir, inventoryFile)
	goAppSourcePath := filepath.Join(artifactDir, golangAppSourceDir) // Use from artifacts for persistence

	// Verify Go app source exists in artifacts
	if _, err := os.Stat(goAppSourcePath); os.IsNotExist(err) {
		log.Fatalf("Go app source not found in artifacts at %s", goAppSourcePath)
	}

	// Check if custom Kata artifacts exist in artifact directory
	customKernelPath := filepath.Join(artifactDir, outputKataGuestDir, "vmlinuz-"+customKataKernelName)
	customRootfsPath := filepath.Join(artifactDir, outputKataGuestDir, "kata-rootfs-"+customKataKernelName+".img")

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

	extraVars := []string{
		fmt.Sprintf("custom_kata_kernel_path_local=%s", kernelPath),
		fmt.Sprintf("custom_kata_rootfs_path_local=%s", rootfsPath),
		fmt.Sprintf("go_app_source_path=%s", goAppSourcePath),
		fmt.Sprintf("container_image_name=%s", containerImageName),
		fmt.Sprintf("artifact_directory=%s", artifactDir), // Pass artifact directory to Ansible
	}

	// We only want to run tasks related to building the container image and running it.
	// This requires tagging the Ansible playbook properly for partial execution.
	// For simplicity, let's just re-run the whole deploy playbook, but a more robust
	// solution would use --tags or --start-at-task.
	ansibleArgs := []string{"-i", inventoryPath, "deploy-kata-app.yml", "-e", strings.Join(extraVars, " ")}
	err := runCmd("ansible-playbook", ansibleArgs, ansibleWorkDir)
	if err != nil {
		log.Fatalf("Ansible container deployment failed: %v", err)
	}
	fmt.Println("--- New container deployed successfully! ---")
}

func runInstallGoAppOnly(resourcesDir, artifactDir string) {
	fmt.Println("--- Only installing KNIRV-NEXUS Go App on existing Kata setup ---")
	ansibleWorkDir := filepath.Join(resourcesDir, ansibleDeployDir)
	inventoryPath := filepath.Join(resourcesDir, inventoryFile)
	goAppSourcePath := filepath.Join(artifactDir, golangAppSourceDir) // Use from artifacts for persistence

	// Verify that the artifact directory exists and contains expected structure
	if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
		log.Fatalf("Artifact directory %s does not exist", artifactDir)
	}
	log.Printf("Using artifact directory: %s", artifactDir)

	// Verify Go app source exists in artifacts
	if _, err := os.Stat(goAppSourcePath); os.IsNotExist(err) {
		log.Fatalf("Go app source not found in artifacts at %s", goAppSourcePath)
	}
	log.Printf("Using Go app source from artifacts: %s", goAppSourcePath)

	// Check for any existing artifacts that might be relevant
	kataArtifactsDir := filepath.Join(artifactDir, outputKataGuestDir)
	if _, err := os.Stat(kataArtifactsDir); err == nil {
		log.Printf("Found Kata artifacts directory: %s", kataArtifactsDir)
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
	err := runCmd("ansible-playbook", ansibleArgs, ansibleWorkDir)
	if err != nil {
		log.Fatalf("Ansible Go app installation failed: %v", err)
	}
	fmt.Println("--- KNIRV-NEXUS Go App installed successfully! ---")
}

func runFullDeployTerraform(resourcesDir, artifactDir string) {
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

	fmt.Println("--- Step 2: Deploying Kata Containers and Go App (via Ansible) ---")
	ansibleWorkDir := filepath.Join(resourcesDir, ansibleDeployDir)
	inventoryPath := filepath.Join(resourcesDir, inventoryFile)
	goAppSourcePath := filepath.Join(artifactDir, golangAppSourceDir) // Use from artifacts for persistence

	// Verify Go app source exists in artifacts
	if _, err := os.Stat(goAppSourcePath); os.IsNotExist(err) {
		log.Fatalf("Go app source not found in artifacts at %s", goAppSourcePath)
	}

	// We need to pass the paths to the custom kernel/rootfs to the Ansible playbook
	// This can be done via Ansible extra vars (-e).
	extraVars := []string{
		fmt.Sprintf("custom_kata_kernel_path_local=%s", kernelPath),
		fmt.Sprintf("custom_kata_rootfs_path_local=%s", rootfsPath),
		fmt.Sprintf("go_app_source_path=%s", goAppSourcePath),
		fmt.Sprintf("container_image_name=%s", containerImageName),
		fmt.Sprintf("artifact_directory=%s", artifactDir), // Pass artifact directory to Ansible
	}

	ansibleArgs := []string{"-i", inventoryPath, "deploy-kata-app.yml", "-e", strings.Join(extraVars, " ")}
	err = runCmd("ansible-playbook", ansibleArgs, ansibleWorkDir)
	if err != nil {
		log.Fatalf("Ansible deployment failed: %v", err)
	}
	fmt.Println("--- Full deployment completed successfully! ---")
}

// executeAction performs the specified action non-interactively
func executeAction(action string, resourcesDir, artifactDir string) {
	switch action {
	case "0":
		fmt.Println("\nRebuilding base Kali image...")
		runBuildBaseKali(resourcesDir, artifactDir)
		fmt.Println("\n✓ Base Kali image rebuilt successfully!")
	case "1":
		fmt.Println("\nStarting full fresh build and deploy (using Packer)...")
		runFullDeploy(resourcesDir, artifactDir)
	case "1b":
		fmt.Println("\nStarting full fresh build and deploy (using Terraform fallback)...")
		runFullDeployTerraform(resourcesDir, artifactDir)
	case "2":
		fmt.Println("\nStarting new container deploy (assuming Kata configured)...")
		runDeployNewContainer(resourcesDir, artifactDir)
	case "3":
		fmt.Println("\nStarting Go app install on existing Kata setup...")
		runInstallGoAppOnly(resourcesDir, artifactDir)
	case "4":
		fmt.Println("Exiting.")
	default:
		fmt.Printf("Invalid action: %s\n", action)
		fmt.Println("Valid actions are: 0, 1, 1b, 2, 3, or 4")
		os.Exit(1)
	}
}
