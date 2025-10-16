package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed all:packer-kata-guest/*
//go:embed all:ansible-deploy/*
//go:embed inventory.ini
//go:embed golang-app-source/knirv-nexus

var embeddedFiles embed.FS

const (
	tempDirPrefix        = "nexus-deployer_"
	packerKataGuestDir   = "packer-kata-guest"
	ansibleDeployDir     = "ansible-deploy"
	inventoryFile        = "inventory.ini"
	golangAppSourceDir   = "golang-app-source"
	outputKataGuestDir   = "output-kata-guest" // Where Packer drops kernel/rootfs
	kataConfigDir        = "/etc/kata-containers"
	customKataKernelName = "kali-clean-tee"
	containerImageName   = "knirvnexus-go-app"
)

func main() {
	fmt.Println("KNIRV-NEXUS Deployment Orchestrator")
	fmt.Println("----------------------------------")

	if runtime.GOOS != "linux" {
		log.Fatalf("This orchestrator is designed for Linux (Ubuntu 22.04) hosts. Detected: %s", runtime.GOOS)
	}

	tempWorkDir, err := os.MkdirTemp("", tempDirPrefix)
	if err != nil {
		log.Fatalf("Failed to create temporary working directory: %v", err)
	}
	defer os.RemoveAll(tempWorkDir) // Clean up on exit

	log.Printf("Working in temporary directory: %s", tempWorkDir)

	err = extractEmbeddedFiles(tempWorkDir)
	if err != nil {
		log.Fatalf("Failed to extract embedded files: %v", err)
	}

	fmt.Println("\nChecking system prerequisites...")
	if err := checkPrerequisites(); err != nil {
		log.Fatalf("Prerequisite check failed: %v", err)
	}
	fmt.Println("All prerequisites met.")

	for {
		fmt.Println("\nSelect an action:")
		fmt.Println("1. Build and deploy a new KNIRV-NEXUS Kata Container (full fresh install)")
		fmt.Println("2. Deploy a new KNIRV-NEXUS Kata Container (assuming Kata is configured)")
		fmt.Println("3. Only install the KNIRV-NEXUS Go App on an existing, configured Kata setup")
		fmt.Println("4. Exit")
		fmt.Print("Enter your choice: ")

		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			fmt.Println("\nStarting full fresh build and deploy...")
			runFullDeploy(tempWorkDir)
		case "2":
			fmt.Println("\nStarting new container deploy (assuming Kata configured)...")
			runDeployNewContainer(tempWorkDir)
		case "3":
			fmt.Println("\nStarting Go app install on existing Kata setup...")
			runInstallGoAppOnly(tempWorkDir)
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

func checkPrerequisites() error {
	cmds := []string{"packer", "ansible-playbook", "VBoxManage", "containerd", "nerdctl", "go"}
	
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

// --- Command Execution Helper ---
func runCmd(name string, args []string, workingDir string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // Allow interactive input for sudo passwords

	log.Printf("Executing: %s %s (in %s)", name, strings.Join(args, " "), workingDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command '%s %s' failed: %v", name, strings.Join(args, " "), err)
	}
	return nil
}

// --- Workflow Functions ---
func runFullDeploy(workDir string) {
	fmt.Println("--- Step 1: Building Custom Kata Kernel and Rootfs (via Packer) ---")
	packerWorkDir := filepath.Join(workDir, packerKataGuestDir)
	outputKataGuestPath := filepath.Join(workDir, outputKataGuestDir) // Packer outputs here
	if err := os.MkdirAll(outputKataGuestPath, 0755); err != nil {
		log.Fatalf("Failed to create output directory for Kata artifacts: %v", err)
	}

	// Adjust paths for fetch in ansible.
	// We need to modify the embedded ansible playbook to fetch to the correct outputKataGuestPath
	// This is a bit tricky with embedded files. A simpler approach is to use absolute paths
	// in the playbook directly, or have the orchestrator rewrite the playbook.
	// For now, assume playbook_dir/../output-kata-guest correctly resolves from the packerWorkDir
	// when packer is run.

	// Placeholder for setting correct output path for fetch in kata-guest-provisioner.yml
	// In a real scenario, you might parse and modify the playbook or set an environment variable
	// that the playbook reads. For simplicity, we assume default paths.

	err := runCmd("packer", []string{"build", "kata-kali-guest.pkr.hcl"}, packerWorkDir)
	if err != nil {
		log.Fatalf("Packer build failed: %v", err)
	}
	fmt.Println("--- Packer build completed. ---")

	fmt.Println("--- Step 2: Deploying Kata Containers and Go App (via Ansible) ---")
	ansibleWorkDir := filepath.Join(workDir, ansibleDeployDir)
	inventoryPath := filepath.Join(workDir, inventoryFile)
	goAppSourcePath := filepath.Join(workDir, golangAppSourceDir)

	// We need to pass the paths to the custom kernel/rootfs to the Ansible playbook
	// This can be done via Ansible extra vars (-e).
	extraVars := []string{
		fmt.Sprintf("custom_kata_kernel_path_local=%s", filepath.Join(outputKataGuestPath, "vmlinuz-"+customKataKernelName)),
		fmt.Sprintf("custom_kata_rootfs_path_local=%s", filepath.Join(outputKataGuestPath, "kata-rootfs-"+customKataKernelName+".img")),
		fmt.Sprintf("go_app_source_path=%s", goAppSourcePath),
		fmt.Sprintf("container_image_name=%s", containerImageName),
	}

	ansibleArgs := []string{"-i", inventoryPath, "deploy-kata-app.yml", "-e", strings.Join(extraVars, " ")}
	err = runCmd("ansible-playbook", ansibleArgs, ansibleWorkDir)
	if err != nil {
		log.Fatalf("Ansible deployment failed: %v", err)
	}
	fmt.Println("--- Full deployment completed successfully! ---")
}

func runDeployNewContainer(workDir string) {
	fmt.Println("--- Deploying new KNIRV-NEXUS Kata Container (assuming Kata configured) ---")
	ansibleWorkDir := filepath.Join(workDir, ansibleDeployDir)
	inventoryPath := filepath.Join(workDir, inventoryFile)
	goAppSourcePath := filepath.Join(workDir, golangAppSourceDir)

	// This path implies the kernel/rootfs are already in /usr/share/kata-containers
	// or Kata is configured to fetch them. If not, this step might fail.
	extraVars := []string{
		// These paths are only used by the "Copy custom Kata kernel/rootfs" tasks.
		// If Kata is already configured and these files are in place, these
		// vars are technically not needed for this mode, but good to include for safety.
		fmt.Sprintf("custom_kata_kernel_path_local=%s", filepath.Join(kataConfigDir, "vmlinuz-"+customKataKernelName)),
		fmt.Sprintf("custom_kata_rootfs_path_local=%s", filepath.Join(kataConfigDir, "kata-rootfs-"+customKataKernelName+".img")),
		fmt.Sprintf("go_app_source_path=%s", goAppSourcePath),
		fmt.Sprintf("container_image_name=%s", containerImageName),
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

func runInstallGoAppOnly(workDir string) {
	fmt.Println("--- Only installing KNIRV-NEXUS Go App on existing Kata setup ---")
	ansibleWorkDir := filepath.Join(workDir, ansibleDeployDir)
	inventoryPath := filepath.Join(workDir, inventoryFile)
	goAppSourcePath := filepath.Join(workDir, golangAppSourceDir)

	extraVars := []string{
		fmt.Sprintf("go_app_source_path=%s", goAppSourcePath),
		fmt.Sprintf("container_image_name=%s", containerImageName),
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
