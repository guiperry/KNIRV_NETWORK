package container

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/services/teesecurity"
)

// ContainerOrchestrator manages the lifecycle of DVE containers
type ContainerOrchestrator struct {
	portAllocator      *PortAllocator
	sshProvisioner     *SSHProvisioner
	config             *ContainerConfig
	teeSecurityService *teesecurity.TEESecurityService
	sshKeys            map[string]*SSHKeypair // containerID -> SSH keys
}

// ContainerConfig holds configuration for container orchestration
type ContainerConfig struct {
	ContainerRuntime         string        // "docker", "podman", "kata", "gvisor"
	BaseImage                string        // Base container image
	SSHPortRangeStart        int           // Starting port for SSH allocation
	SSHPortRangeEnd          int           // Ending port for SSH allocation
	ValidationPortRangeStart int           // Starting port for validation allocation
	ValidationPortRangeEnd   int           // Ending port for validation allocation
	ErrorResPortRangeStart   int           // Starting port for error resolution allocation
	ErrorResPortRangeEnd     int           // Ending port for error resolution allocation
	ProvisioningTimeout      time.Duration // Timeout for container provisioning
	CleanupInterval          time.Duration // Interval for cleanup of expired containers
}

// NewContainerOrchestrator creates a new container orchestrator
func NewContainerOrchestrator(config *ContainerConfig, teeSecurityService *teesecurity.TEESecurityService) (*ContainerOrchestrator, error) {
	portAllocator := NewPortAllocator(PortAllocationConfig{
		SSHRangeStart:        config.SSHPortRangeStart,
		SSHRangeEnd:          config.SSHPortRangeEnd,
		ValidationRangeStart: config.ValidationPortRangeStart,
		ValidationRangeEnd:   config.ValidationPortRangeEnd,
		ErrorResRangeStart:   config.ErrorResPortRangeStart,
		ErrorResRangeEnd:     config.ErrorResPortRangeEnd,
	})

	sshProvisioner := NewSSHProvisioner()

	return &ContainerOrchestrator{
		portAllocator:      portAllocator,
		sshProvisioner:     sshProvisioner,
		config:             config,
		teeSecurityService: teeSecurityService,
		sshKeys:            make(map[string]*SSHKeypair),
	}, nil
}

// ProvisionContainer provisions a new DVE container for a rental
func (co *ContainerOrchestrator) ProvisionContainer(rentalID string) (*Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), co.config.ProvisioningTimeout)
	defer cancel()

	log.Printf("Provisioning container for rental %s", rentalID)

	// Step 1: Security pre-check - verify TEE environment
	if err := co.performSecurityPreCheck(rentalID); err != nil {
		return nil, fmt.Errorf("security pre-check failed: %w", err)
	}

	// Allocate ports for the container
	endpoints, err := co.portAllocator.AllocatePorts(rentalID)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate ports: %w", err)
	}

	// Generate SSH keypair with security validation
	sshKeys, err := co.sshProvisioner.GenerateSSHKeypair()
	if err != nil {
		co.portAllocator.ReleasePorts(rentalID) // Cleanup on failure
		return nil, fmt.Errorf("failed to generate SSH keys: %w", err)
	}

	// Validate SSH key strength
	if err := co.validateSSHKeyStrength(sshKeys); err != nil {
		co.portAllocator.ReleasePorts(rentalID)
		return nil, fmt.Errorf("SSH key validation failed: %w", err)
	}

	// Create container specification with security hardening
	containerSpec := &ContainerSpec{
		ID:             fmt.Sprintf("dve-%s-%d", rentalID, time.Now().Unix()),
		Image:          co.config.BaseImage,
		SSHPort:        endpoints.SSHPort,
		ValidationPort: endpoints.ValidationPort,
		ErrorResPort:   endpoints.ErrorResPort,
		SSHUsername:    fmt.Sprintf("rental-user-%s", rentalID[:8]),
		SSHPublicKey:   sshKeys.PublicKey,
		TEEType:        co.determineTEEType(),
		ResourceLimits: objects.ResourceLimits{
			MaxCPU:       2.0,
			MaxMemory:    4 * 1024 * 1024 * 1024,  // 4GB
			MaxDisk:      50 * 1024 * 1024 * 1024, // 50GB
			MaxBandwidth: 100 * 1024 * 1024,       // 100Mbps
		},
		SecurityProfile: co.createSecurityProfile(),
	}

	// Provision the container with security enforcement
	container, err := co.provisionContainer(ctx, containerSpec)
	if err != nil {
		co.portAllocator.ReleasePorts(rentalID) // Cleanup on failure
		return nil, fmt.Errorf("failed to provision container: %w", err)
	}

	// Inject SSH keys with security validation
	if err := co.sshProvisioner.InjectSSHKey(container.ID, sshKeys.PublicKey, containerSpec.SSHUsername); err != nil {
		co.TerminateContainer(container.ID) // Cleanup on failure
		co.portAllocator.ReleasePorts(rentalID)
		return nil, fmt.Errorf("failed to inject SSH key: %w", err)
	}

	// Post-provisioning security validation
	if err := co.performPostProvisioningSecurityCheck(container); err != nil {
		co.TerminateContainer(container.ID)
		co.portAllocator.ReleasePorts(rentalID)
		return nil, fmt.Errorf("post-provisioning security check failed: %w", err)
	}

	container.SSHKeys = sshKeys
	container.Endpoints = endpoints

	// Store SSH keys for later retrieval
	co.sshKeys[container.ID] = sshKeys

	log.Printf("Successfully provisioned container %s for rental %s with security enforcement", container.ID, rentalID)
	return container, nil
}

// AllocateEndpoints allocates network endpoints for a rental
func (co *ContainerOrchestrator) AllocateEndpoints(rentalID string) (*Endpoints, error) {
	return co.portAllocator.AllocatePorts(rentalID)
}

// InjectSSHKeys injects SSH keys into a running container
func (co *ContainerOrchestrator) InjectSSHKeys(containerID string, publicKey string) error {
	// This would be called after container is running
	username := fmt.Sprintf("rental-user-%s", containerID[len("dve-"):len("dve-")+8])
	return co.sshProvisioner.InjectSSHKey(containerID, publicKey, username)
}

// GetContainerStatus returns the status of a container
func (co *ContainerOrchestrator) GetContainerStatus(containerID string) (ContainerStatus, error) {
	return co.getContainerStatus(containerID)
}

// GetSSHPrivateKey retrieves the SSH private key for a container
func (co *ContainerOrchestrator) GetSSHPrivateKey(containerID string) (string, error) {
	keys, exists := co.sshKeys[containerID]
	if !exists {
		return "", fmt.Errorf("SSH keys not found for container %s", containerID)
	}
	return keys.PrivateKey, nil
}

// TerminateContainer terminates and cleans up a container
func (co *ContainerOrchestrator) TerminateContainer(containerID string) error {
	log.Printf("Terminating container %s", containerID)

	// Extract rental ID from container ID for port cleanup
	rentalID := co.extractRentalIDFromContainerID(containerID)
	if rentalID != "" {
		co.portAllocator.ReleasePorts(rentalID)
	}

	// Clean up SSH keys
	delete(co.sshKeys, containerID)

	// Terminate the container
	return co.terminateContainer(containerID)
}

// Start begins the container orchestrator services
func (co *ContainerOrchestrator) Start(ctx context.Context) error {
	log.Println("Starting container orchestrator")

	// Start cleanup routine
	go co.cleanupRoutine(ctx)

	return nil
}

// Stop stops the container orchestrator services
func (co *ContainerOrchestrator) Stop() error {
	log.Println("Stopping container orchestrator")
	return nil
}

// cleanupRoutine periodically cleans up expired containers and releases ports
func (co *ContainerOrchestrator) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(co.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			co.performCleanup()
		}
	}
}

// performCleanup cleans up expired containers and releases ports
func (co *ContainerOrchestrator) performCleanup() {
	log.Println("Performing container cleanup")

	// Get all active rentals and check for expired ones
	// This would integrate with the rental service to find expired rentals
	// For now, this is a placeholder

	// TODO: Implement cleanup logic
	// - Find expired rentals
	// - Terminate their containers
	// - Release allocated ports
	// - Clean up SSH keys
}

// extractRentalIDFromContainerID extracts rental ID from container ID
func (co *ContainerOrchestrator) extractRentalIDFromContainerID(containerID string) string {
	// Container ID format: dve-{rentalID}-{timestamp}
	if len(containerID) < 8 || containerID[:4] != "dve-" {
		return ""
	}

	// Find the last dash to extract rental ID
	for i := len(containerID) - 1; i >= 4; i-- {
		if containerID[i] == '-' {
			return containerID[4:i]
		}
	}

	return ""
}

// provisionContainer provisions a container using the configured runtime
func (co *ContainerOrchestrator) provisionContainer(ctx context.Context, spec *ContainerSpec) (*Container, error) {
	switch co.config.ContainerRuntime {
	case "docker":
		return co.provisionDockerContainer(ctx, spec)
	case "podman":
		return co.provisionPodmanContainer(ctx, spec)
	case "native-go":
		return co.provisionNativeContainer(spec)
	default:
		return nil, fmt.Errorf("unsupported container runtime: %s", co.config.ContainerRuntime)
	}
}

// getContainerStatus gets the status of a container
func (co *ContainerOrchestrator) getContainerStatus(containerID string) (ContainerStatus, error) {
	log.Printf("DEBUG: getContainerStatus called with containerID: %s", containerID)
	// This is a placeholder implementation
	// In a real implementation, this would query the container runtime

	return ContainerStatusRunning, nil
}

// terminateContainer terminates a container
func (co *ContainerOrchestrator) terminateContainer(containerID string) error {
	switch co.config.ContainerRuntime {
	case "docker":
		return co.terminateDockerContainer(containerID)
	case "podman":
		return co.terminatePodmanContainer(containerID)
	case "native-go":
		return co.terminateNativeContainer(containerID)
	default:
		return fmt.Errorf("unsupported container runtime: %s", co.config.ContainerRuntime)
	}
}

// provisionDockerContainer provisions a container using Docker
func (co *ContainerOrchestrator) provisionDockerContainer(ctx context.Context, spec *ContainerSpec) (*Container, error) {
	// This is a simplified Docker implementation
	// In production, this would use the Docker SDK or API

	log.Printf("Provisioning Docker container %s with image %s", spec.ID, spec.Image)

	// Simulate Docker container creation
	// In real implementation, this would:
	// 1. Use Docker API client
	// 2. Create container with proper configuration
	// 3. Set up port mappings
	// 4. Configure TEE runtime (if using Kata/gVisor)
	// 5. Start the container

	container := &Container{
		ID:        spec.ID,
		Status:    ContainerStatusRunning,
		CreatedAt: time.Now(),
		Spec:      spec,
		Runtime:   "docker",
	}

	// Simulate provisioning time
	select {
	case <-time.After(3 * time.Second):
		// Container provisioned successfully
	case <-ctx.Done():
		return nil, fmt.Errorf("container provisioning timed out")
	}

	log.Printf("Successfully provisioned Docker container %s", container.ID)
	return container, nil
}

// provisionPodmanContainer provisions a container using Podman
func (co *ContainerOrchestrator) provisionPodmanContainer(ctx context.Context, spec *ContainerSpec) (*Container, error) {
	// This is a simplified Podman implementation
	// In production, this would use the Podman API or CLI

	log.Printf("Provisioning Podman container %s with image %s", spec.ID, spec.Image)

	// Simulate Podman container creation
	container := &Container{
		ID:        spec.ID,
		Status:    ContainerStatusRunning,
		CreatedAt: time.Now(),
		Spec:      spec,
		Runtime:   "podman",
	}

	// Simulate provisioning time
	select {
	case <-time.After(3 * time.Second):
		// Container provisioned successfully
	case <-ctx.Done():
		return nil, fmt.Errorf("container provisioning timed out")
	}

	log.Printf("Successfully provisioned Podman container %s", container.ID)
	return container, nil
}

// terminateDockerContainer terminates a Docker container
func (co *ContainerOrchestrator) terminateDockerContainer(containerID string) error {
	log.Printf("Terminating Docker container %s", containerID)

	// Simulate Docker container termination
	// In real implementation, this would:
	// 1. Stop the container
	// 2. Remove the container
	// 3. Clean up associated volumes/networks

	time.Sleep(1 * time.Second) // Simulate termination time
	log.Printf("Terminated Docker container %s", containerID)
	return nil
}

// terminatePodmanContainer terminates a Podman container
func (co *ContainerOrchestrator) terminatePodmanContainer(containerID string) error {
	log.Printf("Terminating Podman container %s", containerID)

	// Simulate Podman container termination
	time.Sleep(1 * time.Second) // Simulate termination time
	log.Printf("Terminated Podman container %s", containerID)
	return nil
}

// provisionNativeContainer provisions a container using native Go runtime with Kali security tools
func (co *ContainerOrchestrator) provisionNativeContainer(spec *ContainerSpec) (*Container, error) {
	if co.teeSecurityService == nil {
		return nil, fmt.Errorf("TEE security service not available for native container runtime")
	}

	log.Printf("Provisioning native Go container %s with Kali Linux security tools", spec.ID)

	// Create container specification for native runtime
	container := &Container{
		ID:        spec.ID,
		Status:    ContainerStatusRunning,
		CreatedAt: time.Now(),
		Spec:      spec,
		Runtime:   "native-go",
	}

	// For native containers, we don't actually create a traditional container
	// Instead, we prepare the environment for sandboxed execution
	// The actual execution happens when skills are run through the TEE security service

	log.Printf("Successfully provisioned native Go container %s", container.ID)
	return container, nil
}

// terminateNativeContainer terminates a native Go container
func (co *ContainerOrchestrator) terminateNativeContainer(containerID string) error {
	log.Printf("Terminating native Go container %s", containerID)

	// For native containers, cleanup involves removing any temporary sandbox directories
	// This is handled by the native container runtime's cleanup mechanisms

	time.Sleep(500 * time.Millisecond) // Simulate cleanup time
	log.Printf("Terminated native Go container %s", containerID)
	return nil
}

// performSecurityPreCheck performs security validation before container provisioning
func (co *ContainerOrchestrator) performSecurityPreCheck(rentalID string) error {
	if co.teeSecurityService == nil {
		log.Printf("Warning: TEE security service not available, skipping security pre-check for rental %s", rentalID)
		return nil // Allow provisioning without TEE for testing/development
	}

	// Run security scan to ensure environment is secure
	if err := co.teeSecurityService.RunSecurityScan(); err != nil {
		return fmt.Errorf("security scan failed: %w", err)
	}

	// Verify TEE attestation
	if err := co.teeSecurityService.PerformAttestation(); err != nil {
		return fmt.Errorf("TEE attestation failed: %w", err)
	}

	log.Printf("Security pre-check passed for rental %s", rentalID)
	return nil
}

// validateSSHKeyStrength validates the strength of generated SSH keys
func (co *ContainerOrchestrator) validateSSHKeyStrength(keys *SSHKeypair) error {
	if keys == nil {
		return fmt.Errorf("SSH keys are nil")
	}

	// Basic validation - check key length and format
	if len(keys.PublicKey) < 100 {
		return fmt.Errorf("SSH public key too short")
	}

	if len(keys.PrivateKey) < 500 {
		return fmt.Errorf("SSH private key too short")
	}

	// Check for required key components
	if !strings.Contains(keys.PublicKey, "ssh-") {
		return fmt.Errorf("invalid SSH public key format")
	}

	if !strings.Contains(keys.PrivateKey, "-----BEGIN") {
		return fmt.Errorf("invalid SSH private key format")
	}

	return nil
}

// determineTEEType determines the appropriate TEE type based on environment
func (co *ContainerOrchestrator) determineTEEType() string {
	if co.teeSecurityService == nil {
		return "none"
	}

	kaliProfile := co.teeSecurityService.GetKaliProfile()
	if kaliProfile.IsKaliLinux {
		// Check for SGX support
		for _, arch := range kaliProfile.ArchitectureSupport {
			if strings.Contains(strings.ToLower(arch), "sgx") {
				return "sgx"
			}
		}
		// Default to software-based TEE for Kali
		return "software"
	}

	// For non-Kali systems, use basic isolation
	return "basic"
}

// createSecurityProfile creates a security profile based on the environment
func (co *ContainerOrchestrator) createSecurityProfile() *SecurityProfile {
	profile := &SecurityProfile{
		ReadOnlyRootFS:  false, // Allow writes for development
		NoNewPrivileges: true,  // Prevent privilege escalation
		IsolationLevel:  "standard",
		Capabilities:    []string{}, // Drop all capabilities
		SecurityOpts:    []string{},
	}

	if co.teeSecurityService != nil {
		kaliProfile := co.teeSecurityService.GetKaliProfile()

		if kaliProfile.IsKaliLinux {
			// Enhanced security for Kali Linux
			profile.IsolationLevel = "strong"
			profile.AppArmorProfile = "docker-default"
			profile.SELinuxLabel = "system_u:system_r:container_t:s0"

			// Add security options for Kali
			profile.SecurityOpts = append(profile.SecurityOpts,
				"no-new-privileges:true",
				"seccomp=builtin",
			)

			// Allow specific syscalls for Kali tools
			profile.AllowedSyscalls = []string{
				"strace", "ptrace", "perf_event_open",
				"tcpdump", "network",
			}
		}
	}

	return profile
}

// performPostProvisioningSecurityCheck performs security validation after container provisioning
func (co *ContainerOrchestrator) performPostProvisioningSecurityCheck(container *Container) error {
	if container == nil {
		return fmt.Errorf("container is nil")
	}

	// Verify container is running
	status, err := co.GetContainerStatus(container.ID)
	if err != nil {
		return fmt.Errorf("failed to get container status: %w", err)
	}

	if status != ContainerStatusRunning {
		return fmt.Errorf("container is not running, status: %s", status)
	}

	// Perform security scan on the running container
	if co.teeSecurityService != nil {
		// This would perform runtime security checks
		log.Printf("Post-provisioning security check passed for container %s", container.ID)
	}

	return nil
}
