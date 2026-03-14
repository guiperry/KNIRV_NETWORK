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
		preferredRuntime: kaliProfile.PreferredRuntime, // Initialize with profile's preferred runtime
	}

	// Determine runtime strategy based on host environment
	switch kaliProfile.HostEnvironment {
	case "docker":
		// If running in Docker, always prefer Podman for nested container orchestration
		log.Println("Running in Docker environment, forcing Podman as preferred runtime.")
		manager.preferredRuntime = "podman"
		// Initialize Podman directly
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
			log.Printf("Warning: Podman validation failed in Docker environment: %v", err)
			log.Println("Container runtime features will be disabled.")
			manager.preferredRuntime = "disabled"
		} else {
			manager.podmanFallback = podmanRuntime
		}
		return manager, nil

	case "bare-metal", "kata-container":
		// On bare-metal or Kata (VM-level isolation), prioritize native Go runtime
		if kaliProfile.PreferredRuntime == "native-go" {
			nativeRuntime, err := NewNativeContainerRuntime(kaliProfile)
			if err != nil {
				log.Printf("Native runtime failed to initialize on %s: %v. Falling back to Podman...",
					kaliProfile.HostEnvironment, err)
				manager.preferredRuntime = "podman" // Fallback to Podman
			} else {
				manager.nativeRuntime = nativeRuntime
				return manager, nil // Native runtime successfully initialized
			}
		} else {
			log.Printf("Profile prefers Podman (%s) on %s environment, initializing Podman.",
				kaliProfile.PreferredRuntime, kaliProfile.HostEnvironment)
			manager.preferredRuntime = "podman"
		}

		// If native runtime failed or not preferred, initialize Podman fallback
		if manager.preferredRuntime == "podman" {
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
				log.Printf("Warning: Podman validation failed on %s environment: %v",
					kaliProfile.HostEnvironment, err)
				log.Println("Container runtime features will be disabled.")
				manager.preferredRuntime = "disabled"
			} else {
				manager.podmanFallback = podmanRuntime
			}
		}

	default:
		// Fallback for unknown host environments (should not happen with proper detection)
		log.Printf("Unknown host environment: %s. Defaulting to Podman.", kaliProfile.HostEnvironment)
		manager.preferredRuntime = "podman"
		// Initialize Podman directly as a fallback
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
			log.Printf("Warning: Podman validation failed in unknown environment: %v", err)
			log.Println("Container runtime features will be disabled.")
			manager.preferredRuntime = "disabled"
		} else {
			manager.podmanFallback = podmanRuntime
		}
	}

	log.Printf("ContainerRuntimeManager initialized. Active Runtime: %s, Host Environment: %s",
		manager.preferredRuntime, kaliProfile.HostEnvironment)
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
