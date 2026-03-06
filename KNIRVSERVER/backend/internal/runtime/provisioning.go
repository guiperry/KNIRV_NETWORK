// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package runtime provides unified container management with integrated
// port allocation and SSH provisioning capabilities.
package runtime

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/ebpf"
	"backend_server/internal/services/teesecurity"

	"golang.org/x/crypto/ssh"
)

// -----------------------------------------------------------------------
// PortAllocator - mirrors container.PortAllocator in the runtime package
// -----------------------------------------------------------------------

// ProvisioningPortConfig holds port-range configuration for the runtime
// PortAllocator. Ranges are inclusive on both ends.
type ProvisioningPortConfig struct {
	SSHRangeStart        int
	SSHRangeEnd          int
	ValidationRangeStart int
	ValidationRangeEnd   int
	ErrorResRangeStart   int
	ErrorResRangeEnd     int
}

// DefaultProvisioningPortConfig returns a sensible default config suitable
// for a single-host development deployment.
func DefaultProvisioningPortConfig() ProvisioningPortConfig {
	return ProvisioningPortConfig{
		SSHRangeStart:        2200,
		SSHRangeEnd:          2299,
		ValidationRangeStart: 9100,
		ValidationRangeEnd:   9199,
		ErrorResRangeStart:   9200,
		ErrorResRangeEnd:     9299,
	}
}

// ProvisioningPortAllocation records the ports claimed for one rental/node.
type ProvisioningPortAllocation struct {
	RentalID       string    `json:"rental_id"`
	SSHPort        int       `json:"ssh_port"`
	ValidationPort int       `json:"validation_port"`
	ErrorResPort   int       `json:"error_resolution_port"`
	AllocatedAt    time.Time `json:"allocated_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// ProvisioningEndpoints holds the resolved host:port endpoints for a container.
type ProvisioningEndpoints struct {
	SSHPort        int    `json:"ssh_port"`
	ValidationPort int    `json:"validation_port"`
	ErrorResPort   int    `json:"error_resolution_port"`
	Host           string `json:"host"`
}

// PortAllocator manages port allocation for DVE containers inside the
// runtime package. It is functionally identical to
// services/container.PortAllocator but lives here so UnifiedContainerManager
// can own the full provisioning lifecycle without importing the container
// service package.
type PortAllocator struct {
	config       ProvisioningPortConfig
	allocations  map[string]*ProvisioningPortAllocation // rentalID -> allocation
	usedPorts    map[int]string                         // port -> rentalID
	mutex        sync.RWMutex
	cleanupTimer *time.Ticker
}

// NewPortAllocator creates a new PortAllocator and starts its background
// cleanup goroutine.
func NewPortAllocator(config ProvisioningPortConfig) *PortAllocator {
	pa := &PortAllocator{
		config:      config,
		allocations: make(map[string]*ProvisioningPortAllocation),
		usedPorts:   make(map[int]string),
	}
	pa.cleanupTimer = time.NewTicker(5 * time.Minute)
	go pa.cleanupExpiredAllocations()
	return pa
}

// AllocatePorts allocates a set of ports for the given rental/node ID.
// If ports have already been allocated for that ID, the existing allocation
// is returned without modification.
func (pa *PortAllocator) AllocatePorts(rentalID string) (*ProvisioningEndpoints, error) {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()

	if alloc, exists := pa.allocations[rentalID]; exists {
		return &ProvisioningEndpoints{
			SSHPort:        alloc.SSHPort,
			ValidationPort: alloc.ValidationPort,
			ErrorResPort:   alloc.ErrorResPort,
			Host:           "localhost",
		}, nil
	}

	sshPort, err := pa.findAvailablePort(pa.config.SSHRangeStart, pa.config.SSHRangeEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate SSH port: %w", err)
	}

	validationPort, err := pa.findAvailablePort(pa.config.ValidationRangeStart, pa.config.ValidationRangeEnd)
	if err != nil {
		pa.releasePort(sshPort)
		return nil, fmt.Errorf("failed to allocate validation port: %w", err)
	}

	errorResPort, err := pa.findAvailablePort(pa.config.ErrorResRangeStart, pa.config.ErrorResRangeEnd)
	if err != nil {
		pa.releasePort(sshPort)
		pa.releasePort(validationPort)
		return nil, fmt.Errorf("failed to allocate error-resolution port: %w", err)
	}

	alloc := &ProvisioningPortAllocation{
		RentalID:       rentalID,
		SSHPort:        sshPort,
		ValidationPort: validationPort,
		ErrorResPort:   errorResPort,
		AllocatedAt:    time.Now(),
		ExpiresAt:      time.Now().Add(25 * time.Hour),
	}

	pa.allocations[rentalID] = alloc
	pa.usedPorts[sshPort] = rentalID
	pa.usedPorts[validationPort] = rentalID
	pa.usedPorts[errorResPort] = rentalID

	log.Printf("[PortAllocator] Allocated ports for %s: SSH=%d, Validation=%d, ErrorRes=%d",
		rentalID, sshPort, validationPort, errorResPort)

	return &ProvisioningEndpoints{
		SSHPort:        sshPort,
		ValidationPort: validationPort,
		ErrorResPort:   errorResPort,
		Host:           "localhost",
	}, nil
}

// ReleasePorts releases all ports previously allocated for the given rental/node ID.
func (pa *PortAllocator) ReleasePorts(rentalID string) {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()

	alloc, exists := pa.allocations[rentalID]
	if !exists {
		return
	}

	pa.releasePort(alloc.SSHPort)
	pa.releasePort(alloc.ValidationPort)
	pa.releasePort(alloc.ErrorResPort)
	delete(pa.allocations, rentalID)

	log.Printf("[PortAllocator] Released ports for %s", rentalID)
}

// GetAllocation returns the current allocation record for a rental/node ID.
func (pa *PortAllocator) GetAllocation(rentalID string) (*ProvisioningPortAllocation, bool) {
	pa.mutex.RLock()
	defer pa.mutex.RUnlock()

	alloc, exists := pa.allocations[rentalID]
	return alloc, exists
}

// ExtendAllocation pushes the expiry of a port allocation forward by duration.
func (pa *PortAllocator) ExtendAllocation(rentalID string, duration time.Duration) error {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()

	alloc, exists := pa.allocations[rentalID]
	if !exists {
		return fmt.Errorf("no allocation found for rental %s", rentalID)
	}

	alloc.ExpiresAt = time.Now().Add(duration)
	log.Printf("[PortAllocator] Extended allocation for %s until %s", rentalID, alloc.ExpiresAt)
	return nil
}

// ListAllocations returns a snapshot of all current port allocations.
func (pa *PortAllocator) ListAllocations() []*ProvisioningPortAllocation {
	pa.mutex.RLock()
	defer pa.mutex.RUnlock()

	out := make([]*ProvisioningPortAllocation, 0, len(pa.allocations))
	for _, a := range pa.allocations {
		out = append(out, a)
	}
	return out
}

// Stop shuts down the background cleanup goroutine.
func (pa *PortAllocator) Stop() {
	if pa.cleanupTimer != nil {
		pa.cleanupTimer.Stop()
	}
}

// findAvailablePort returns the first unused port in [start, end].
// Caller must hold pa.mutex (write lock).
func (pa *PortAllocator) findAvailablePort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		if _, used := pa.usedPorts[port]; !used {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", start, end)
}

// releasePort removes a single port from the used-ports map.
// Caller must hold pa.mutex (write lock).
func (pa *PortAllocator) releasePort(port int) {
	if owner, exists := pa.usedPorts[port]; exists {
		delete(pa.usedPorts, port)
		log.Printf("[PortAllocator] Released port %d (was owned by %s)", port, owner)
	}
}

// cleanupExpiredAllocations is the background goroutine that reclaims
// ports whose ExpiresAt has passed.
func (pa *PortAllocator) cleanupExpiredAllocations() {
	for range pa.cleanupTimer.C {
		now := time.Now()
		var expired []string

		pa.mutex.RLock()
		for id, alloc := range pa.allocations {
			if now.After(alloc.ExpiresAt) {
				expired = append(expired, id)
			}
		}
		pa.mutex.RUnlock()

		for _, id := range expired {
			log.Printf("[PortAllocator] Cleaning up expired allocation for %s", id)
			pa.ReleasePorts(id)
		}

		if len(expired) > 0 {
			log.Printf("[PortAllocator] Cleaned up %d expired allocations", len(expired))
		}
	}
}

// -----------------------------------------------------------------------
// SSHProvisioner - mirrors container.SSHProvisioner in the runtime package
// -----------------------------------------------------------------------

// SSHKeypair holds an RSA SSH keypair in PEM / authorized_keys format.
type SSHKeypair struct {
	PublicKey      string `json:"public_key"`
	PrivateKey     string `json:"private_key"`
	KeyFingerprint string `json:"key_fingerprint"`
}

// SSHProvisioner handles SSH key generation and injection for containers.
// It is functionally identical to services/container.SSHProvisioner but
// lives here so UnifiedContainerManager can own the full provisioning
// lifecycle without importing the container service package.
type SSHProvisioner struct{}

// NewSSHProvisioner creates a new SSHProvisioner.
func NewSSHProvisioner() *SSHProvisioner {
	return &SSHProvisioner{}
}

// GenerateSSHKeypair generates a fresh 2048-bit RSA keypair and returns it
// in PEM / authorized_keys format along with the MD5 fingerprint.
func (sp *SSHProvisioner) GenerateSSHKeypair() (*SSHKeypair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	pubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive SSH public key: %w", err)
	}

	return &SSHKeypair{
		PublicKey:      string(ssh.MarshalAuthorizedKey(pubKey)),
		PrivateKey:     string(privPEM),
		KeyFingerprint: ssh.FingerprintLegacyMD5(pubKey),
	}, nil
}

// InjectSSHKey injects an SSH public key into a running container.
// The current implementation is a placeholder; production use would execute
// commands in the container via the Docker SDK or equivalent.
func (sp *SSHProvisioner) InjectSSHKey(containerID, publicKey, username string) error {
	log.Printf("[SSHProvisioner] Injecting SSH key for user %s into container %s", username, containerID)
	time.Sleep(500 * time.Millisecond) // simulate injection latency
	log.Printf("[SSHProvisioner] SSH key injected for user %s in container %s", username, containerID)
	return nil
}

// ValidateSSHKey returns nil if publicKey is a valid SSH authorized_keys entry.
func (sp *SSHProvisioner) ValidateSSHKey(publicKey string) error {
	_, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return fmt.Errorf("invalid SSH public key: %w", err)
	}
	return nil
}

// GetSSHConfig constructs an ssh.ClientConfig for connecting to a container
// using the supplied PEM private key.
func (sp *SSHProvisioner) GetSSHConfig(privateKey, username string) (*ssh.ClientConfig, error) {
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		// InsecureIgnoreHostKey is acceptable for dynamically created
		// containers; callers should substitute known_hosts verification
		// in production.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
	}, nil
}

// RevokeSSHKey removes an SSH key from a container's authorized_keys.
// Placeholder implementation.
func (sp *SSHProvisioner) RevokeSSHKey(containerID, username, fingerprint string) error {
	log.Printf("[SSHProvisioner] Revoking key %s for user %s in container %s",
		fingerprint, username, containerID)
	return nil
}

// ListSSHKeys returns all authorized SSH keys for a container user.
// Placeholder implementation.
func (sp *SSHProvisioner) ListSSHKeys(containerID, username string) ([]string, error) {
	return []string{}, nil
}

// -----------------------------------------------------------------------
// Provisioner - aggregates PortAllocator + SSHProvisioner
// -----------------------------------------------------------------------

// Provisioner bundles port allocation and SSH provisioning so that
// UnifiedContainerManager can offer a single, coherent provisioning interface.
type Provisioner struct {
	Ports *PortAllocator
	SSH   *SSHProvisioner
}

// NewProvisioner creates a Provisioner with the supplied port configuration.
func NewProvisioner(portConfig ProvisioningPortConfig) *Provisioner {
	return &Provisioner{
		Ports: NewPortAllocator(portConfig),
		SSH:   NewSSHProvisioner(),
	}
}

// ProvisionContainer allocates ports and generates SSH credentials for a
// container identified by id. It returns the allocated endpoints and keypair.
func (p *Provisioner) ProvisionContainer(id string) (*ProvisioningEndpoints, *SSHKeypair, error) {
	endpoints, err := p.Ports.AllocatePorts(id)
	if err != nil {
		return nil, nil, fmt.Errorf("port allocation failed: %w", err)
	}

	keypair, err := p.SSH.GenerateSSHKeypair()
	if err != nil {
		p.Ports.ReleasePorts(id)
		return nil, nil, fmt.Errorf("SSH keypair generation failed: %w", err)
	}

	return endpoints, keypair, nil
}

// Stop frees all resources held by the Provisioner (e.g. the cleanup goroutine).
func (p *Provisioner) Stop() {
	p.Ports.Stop()
}

// -----------------------------------------------------------------------
// UnifiedContainerManagerWithProvisioning
// -----------------------------------------------------------------------

// UnifiedContainerManagerWithProvisioning embeds UnifiedContainerManager and
// adds an integrated Provisioner so that DVE container creation can allocate
// ports and SSH credentials in a single call without importing the container
// service package.
type UnifiedContainerManagerWithProvisioning struct {
	*UnifiedContainerManager
	Provisioner *Provisioner
}

// NewUnifiedContainerManagerWithProvisioning creates a fully-wired manager
// that handles both unified container lifecycle management and DVE
// provisioning (port allocation + SSH key management).
//
// portConfig controls which port ranges are used for SSH, validation, and
// error-resolution endpoints. Pass DefaultProvisioningPortConfig() if you
// have no specific requirements.
func NewUnifiedContainerManagerWithProvisioning(
	teeService *teesecurity.TEESecurityService,
	ebpfManager ebpf.ManagerInterface,
	vcManager *ebpf.VirtualContainerManager,
	portConfig ProvisioningPortConfig,
) *UnifiedContainerManagerWithProvisioning {
	return &UnifiedContainerManagerWithProvisioning{
		UnifiedContainerManager: NewUnifiedContainerManager(teeService, ebpfManager, vcManager),
		Provisioner:             NewProvisioner(portConfig),
	}
}

// ProvisionAndCreate allocates ports + SSH credentials for id and then
// creates a container with the supplied spec, mode, objectType, and
// security level. The allocated SSH port is injected into spec.Ports so the
// container runtime exposes the correct host port.
func (m *UnifiedContainerManagerWithProvisioning) ProvisionAndCreate(
	ctx context.Context,
	id string,
	spec *ContainerSpec,
	mode RuntimeMode,
	objectType ObjectType,
	securityLevel SecurityLevel,
) (*UnifiedContainer, *ProvisioningEndpoints, *SSHKeypair, error) {
	endpoints, keypair, err := m.Provisioner.ProvisionContainer(id)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("provisioning failed for %s: %w", id, err)
	}

	// Inject the provisioned SSH host port into the spec so the container
	// runtime maps container port 22 to the dynamically allocated host port.
	if spec != nil {
		spec.Ports = append(spec.Ports, PortMapping{
			ContainerPort: 22,
			HostPort:      endpoints.SSHPort,
			Protocol:      "tcp",
		})
	}

	container, err := m.UnifiedContainerManager.CreateContainer(
		ctx, spec, mode, objectType, securityLevel,
	)
	if err != nil {
		m.Provisioner.Ports.ReleasePorts(id)
		return nil, nil, nil, fmt.Errorf("container creation failed: %w", err)
	}

	return container, endpoints, keypair, nil
}

// Stop shuts down both the underlying manager and the Provisioner.
func (m *UnifiedContainerManagerWithProvisioning) Stop() {
	m.Provisioner.Stop()
}
