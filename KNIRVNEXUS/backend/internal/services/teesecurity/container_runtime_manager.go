// File: backend/internal/services/teesecurity/container_runtime_manager.go
package teesecurity

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
)

// ContainerRuntimeManager manages runtime selection and fallback strategy
type ContainerRuntimeManager struct {
	kaliProfile         *KaliLinuxProfile
	nativeRuntime       *NativeContainerRuntime
	podmanFallback      *PodmanRuntime
	preferredRuntime    string // "native-go" or "podman"
}

// PodmanRuntime wraps Podman container operations (fallback)
type PodmanRuntime struct {
	userID  int
	groupID int
}

// NewContainerRuntimeManager creates a runtime manager with appropriate fallback
func NewContainerRuntimeManager(kaliProfile *KaliLinuxProfile) (*ContainerRuntimeManager, error) {
	manager := &ContainerRuntimeManager{
		kaliProfile:      kaliProfile,
		preferredRuntime: kaliProfile.PreferredRuntime,
	}

	// Try primary runtime first
	if kaliProfile.IsKaliLinux && kaliProfile.PreferredRuntime == "native-go" {
		nativeRuntime, err := NewNativeContainerRuntime(kaliProfile)
		if err != nil {
			log.Printf("Native runtime failed: %v. Falling back to Podman...", err)
			manager.preferredRuntime = "podman"
		} else {
			manager.nativeRuntime = nativeRuntime
			return manager, nil
		}
	}

	// Initialize Podman fallback for all non-Kali systems or on native failure
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %v", err)
	}

	userID, _ := strconv.Atoi(currentUser.Uid)
	groupID, _ := strconv.Atoi(currentUser.Gid)

	podmanRuntime := &PodmanRuntime{
		userID:  userID,
		groupID: groupID,
	}

	if err := podmanRuntime.validate(context.Background()); err != nil {
		log.Printf("Warning: Podman validation failed: %v", err)
		log.Println("Container runtime features will be disabled (suitable for development/testing)")
		manager.preferredRuntime = "disabled"
		return manager, nil
	}

	manager.podmanFallback = podmanRuntime
	manager.preferredRuntime = "podman"

	return manager, nil
}

// RunContainer executes a container using the appropriate runtime
func (crm *ContainerRuntimeManager) RunContainer(ctx context.Context, opts ContainerOptions) (*ContainerResult, error) {
	if crm.nativeRuntime != nil && crm.preferredRuntime == "native-go" {
		return crm.nativeRuntime.RunContainer(ctx, opts)
	}

	if crm.podmanFallback != nil {
		return crm.podmanFallback.RunContainer(ctx, opts)
	}

	if crm.preferredRuntime == "disabled" {
		return nil, fmt.Errorf("container runtime is disabled (cannot create nested containers in containerized environment)")
	}

	return nil, fmt.Errorf("no container runtime available")
}

// GetActiveRuntime returns the currently active runtime name
func (crm *ContainerRuntimeManager) GetActiveRuntime() string {
	return crm.preferredRuntime
}

// PodmanRuntime methods

// validate checks if Podman is available and functional
func (pr *PodmanRuntime) validate(ctx context.Context) error {
	_, err := exec.LookPath("podman")
	if err != nil {
		return fmt.Errorf("podman not found: %v", err)
	}

	cmd := exec.CommandContext(ctx, "podman", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("podman test failed: %v", err)
	}

	log.Println("Podman fallback runtime validated successfully")
	return nil
}

// RunContainer executes a container using Podman
func (pr *PodmanRuntime) RunContainer(ctx context.Context, opts ContainerOptions) (*ContainerResult, error) {
	result := &ContainerResult{
		ContainerID: fmt.Sprintf("podman-%d", os.Getpid()),
	}

	log.Printf("Running container with Podman: %s", opts.Name)

	cmd := []string{"podman", "run", "--rm"}

	// Add security options
	if opts.SecurityOpts != nil {
		for _, opt := range opts.SecurityOpts {
			cmd = append(cmd, "--security-opt", opt)
		}
	}

	// Add environment variables
	if opts.Env != nil {
		for _, env := range opts.Env {
			cmd = append(cmd, "-e", env)
		}
	}

	// Add volumes
	if opts.Volumes != nil {
		for _, vol := range opts.Volumes {
			cmd = append(cmd, "-v", vol)
		}
	}

	// Add container name
	if opts.Name != "" {
		cmd = append(cmd, "--name", opts.Name)
	}

	// Add image
	cmd = append(cmd, opts.Image)

	// Add arguments
	if opts.Args != nil {
		cmd = append(cmd, opts.Args...)
	}

	execCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)

	output, err := execCmd.CombinedOutput()
	if err != nil {
		result.ExitCode = 1
		result.Stderr = string(output)
	} else {
		result.ExitCode = 0
		result.Stdout = string(output)
	}

	return result, nil
}
