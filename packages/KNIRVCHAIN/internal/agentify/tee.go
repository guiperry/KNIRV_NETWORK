// tee.go
package agentify

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// TEE (Trusted Execution Environment) defines the interface for executing code in a secure environment
type TEE interface {
	// Start starts the TEE
	Start() error

	// Stop stops the TEE
	Stop() error

	// Execute executes a command in the TEE and returns the stdout, stderr, exit code, and error
	Execute(command string, args []string) (stdout string, stderr string, exitCode int, err error)

	// GetInfo returns information about the TEE
	GetInfo() map[string]interface{}
}

// TEEConfig defines the configuration for a TEE
type TEEConfig struct {
	// The working directory for the TEE
	WorkingDir string

	// Environment variables for the TEE
	Env map[string]string

	// For container and VM TEEs
	Image string
	Tag   string

	// For VM TEEs
	Memory int
	CPU    int
}

// ProcessTEE implements the TEE interface using a process
type ProcessTEE struct {
	config TEEConfig
	mutex  sync.Mutex
}

// NewProcessTEE creates a new ProcessTEE
func NewProcessTEE(config TEEConfig) *ProcessTEE {
	return &ProcessTEE{
		config: config,
	}
}

// Start starts the ProcessTEE
func (t *ProcessTEE) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Create the working directory if it doesn't exist
	if t.config.WorkingDir != "" {
		if err := os.MkdirAll(t.config.WorkingDir, 0755); err != nil {
			return fmt.Errorf("failed to create working directory: %v", err)
		}
	}

	return nil
}

// Stop stops the ProcessTEE
func (t *ProcessTEE) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Nothing to do for ProcessTEE
	return nil
}

// Execute executes a command in the ProcessTEE
func (t *ProcessTEE) Execute(command string, args []string) (string, string, int, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Create the command
	cmd := exec.Command(command, args...)

	// Set the working directory
	if t.config.WorkingDir != "" {
		cmd.Dir = t.config.WorkingDir
	}

	// Set environment variables
	if len(t.config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range t.config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()

	// Get the exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return stdout.String(), stderr.String(), exitCode, err
}

// GetInfo returns information about the ProcessTEE
func (t *ProcessTEE) GetInfo() map[string]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return map[string]interface{}{
		"type":       "process",
		"workingDir": t.config.WorkingDir,
		"env":        t.config.Env,
	}
}

// ContainerTEE implements the TEE interface using a container
type ContainerTEE struct {
	config      TEEConfig
	containerID string
	mutex       sync.Mutex
}

// NewContainerTEE creates a new ContainerTEE
func NewContainerTEE(config TEEConfig) *ContainerTEE {
	return &ContainerTEE{
		config: config,
	}
}

// Start starts the ContainerTEE
func (t *ContainerTEE) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Create the working directory if it doesn't exist
	if t.config.WorkingDir != "" {
		if err := os.MkdirAll(t.config.WorkingDir, 0755); err != nil {
			return fmt.Errorf("failed to create working directory: %v", err)
		}
	}

	// Pull the container image
	pullCmd := exec.Command("docker", "pull", fmt.Sprintf("%s:%s", t.config.Image, t.config.Tag))
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull container image: %v", err)
	}

	// Create the container
	createArgs := []string{
		"create",
		"--rm",
		"-v", fmt.Sprintf("%s:/workspace", t.config.WorkingDir),
	}

	// Add environment variables
	for k, v := range t.config.Env {
		createArgs = append(createArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add the image
	createArgs = append(createArgs, fmt.Sprintf("%s:%s", t.config.Image, t.config.Tag))

	// Add a command to keep the container running
	createArgs = append(createArgs, "tail", "-f", "/dev/null")

	// Create the container
	createCmd := exec.Command("docker", createArgs...)
	var stdout bytes.Buffer
	createCmd.Stdout = &stdout
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	// Get the container ID
	t.containerID = stdout.String()

	// Start the container
	startCmd := exec.Command("docker", "start", t.containerID)
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	return nil
}

// Stop stops the ContainerTEE
func (t *ContainerTEE) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.containerID == "" {
		return nil
	}

	// Stop the container
	stopCmd := exec.Command("docker", "stop", t.containerID)
	if err := stopCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container: %v", err)
	}

	t.containerID = ""
	return nil
}

// Execute executes a command in the ContainerTEE
func (t *ContainerTEE) Execute(command string, args []string) (string, string, int, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.containerID == "" {
		return "", "", 1, fmt.Errorf("container not started")
	}

	// Create the command
	execArgs := []string{
		"exec",
		"-w", "/workspace",
		t.containerID,
		command,
	}
	execArgs = append(execArgs, args...)

	// Execute the command
	cmd := exec.Command("docker", execArgs...)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()

	// Get the exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return stdout.String(), stderr.String(), exitCode, err
}

// GetInfo returns information about the ContainerTEE
func (t *ContainerTEE) GetInfo() map[string]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return map[string]interface{}{
		"type":        "container",
		"image":       t.config.Image,
		"tag":         t.config.Tag,
		"containerID": t.containerID,
		"workingDir":  t.config.WorkingDir,
		"env":         t.config.Env,
	}
}

// VMTEE implements the TEE interface using a VM
type VMTEE struct {
	config TEEConfig
	vmID   string
	mutex  sync.Mutex
}

// NewVMTEE creates a new VMTEE
func NewVMTEE(config TEEConfig) *VMTEE {
	return &VMTEE{
		config: config,
	}
}

// Start starts the VMTEE
func (t *VMTEE) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Create the working directory if it doesn't exist
	if t.config.WorkingDir != "" {
		if err := os.MkdirAll(t.config.WorkingDir, 0755); err != nil {
			return fmt.Errorf("failed to create working directory: %v", err)
		}
	}

	// In a real implementation, we would start a VM here
	// For now, we'll just simulate it
	t.vmID = "vm-" + filepath.Base(t.config.WorkingDir)

	return nil
}

// Stop stops the VMTEE
func (t *VMTEE) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vmID == "" {
		return nil
	}

	// In a real implementation, we would stop the VM here
	// For now, we'll just simulate it
	t.vmID = ""

	return nil
}

// Execute executes a command in the VMTEE
func (t *VMTEE) Execute(command string, args []string) (string, string, int, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vmID == "" {
		return "", "", 1, fmt.Errorf("VM not started")
	}

	// In a real implementation, we would execute the command in the VM
	// For now, we'll just simulate it by executing it locally

	// Create the command
	cmd := exec.Command(command, args...)

	// Set the working directory
	if t.config.WorkingDir != "" {
		cmd.Dir = t.config.WorkingDir
	}

	// Set environment variables
	if len(t.config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range t.config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()

	// Get the exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return stdout.String(), stderr.String(), exitCode, err
}

// GetInfo returns information about the VMTEE
func (t *VMTEE) GetInfo() map[string]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return map[string]interface{}{
		"type":       "vm",
		"image":      t.config.Image,
		"memory":     t.config.Memory,
		"cpu":        t.config.CPU,
		"vmID":       t.vmID,
		"workingDir": t.config.WorkingDir,
		"env":        t.config.Env,
	}
}
