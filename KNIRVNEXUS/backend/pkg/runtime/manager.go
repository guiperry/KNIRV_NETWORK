// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/ebpf"
	"backend_server/internal/services/teesecurity"

	"lukechampine.com/blake3"
)

// UnifiedContainerManager manages all container types
type UnifiedContainerManager struct {
	teeService      *teesecurity.TEESecurityService
	ebpfManager     ebpf.ManagerInterface
	vcManager       *ebpf.VirtualContainerManager
	runtimeSelector *RuntimeSelector
	assetRegistry   *AssetRegistry
	containers      map[string]*UnifiedContainer
	mu              sync.RWMutex
}

// NewUnifiedContainerManager creates a new unified container manager
func NewUnifiedContainerManager(
	teeService *teesecurity.TEESecurityService,
	ebpfManager ebpf.ManagerInterface,
	vcManager *ebpf.VirtualContainerManager,
) *UnifiedContainerManager {
	return &UnifiedContainerManager{
		teeService:      teeService,
		ebpfManager:     ebpfManager,
		vcManager:       vcManager,
		runtimeSelector: NewRuntimeSelector(),
		assetRegistry:   NewAssetRegistry(),
		containers:      make(map[string]*UnifiedContainer),
	}
}

// CreateContainer creates a new unified container with the specified mode
func (ucm *UnifiedContainerManager) CreateContainer(
	ctx context.Context,
	spec *ContainerSpec,
	mode RuntimeMode,
	objectType ObjectType,
	securityLevel SecurityLevel,
) (*UnifiedContainer, error) {
	// Validate spec
	if spec == nil {
		return nil, fmt.Errorf("container spec is required")
	}
	if spec.Image == "" {
		return nil, fmt.Errorf("container image is required")
	}

	// Select appropriate runtime based on mode and host capabilities
	runtime, err := ucm.runtimeSelector.SelectRuntime(mode, securityLevel)
	if err != nil {
		return nil, fmt.Errorf("runtime selection failed: %w", err)
	}

	container := &UnifiedContainer{
		ID:            generateContainerID(),
		Mode:          mode,
		ObjectType:    objectType,
		SecurityLevel: securityLevel,
		Spec:          spec,
		Runtime:       runtime,
		CreatedAt:     time.Now(),
		Status:        ContainerStatusCreating,
	}

	// Initialize eBPF security if available
	if ucm.ebpfManager != nil {
		// TODO: Fix eBPF initialization - NewSyscallMonitor returns (monitor, error)
		// monitor, err := ebpf.NewSyscallMonitor()
		// if err == nil {
		// 	container.EBPFMonitor = monitor
		// }
		// SandboxLSM not available in current implementation
	}

	// Initialize TEE attestation if requested
	if securityLevel == SecurityLevelExtreme && ucm.teeService != nil {
		// TODO: Fix TEE attester initialization
		// attester := teesecurity.NewTEEAttester(ucm.teeService)
		// attestation, err := attester.GenerateAttestation(ctx)
		// if err != nil {
		// 	log.Printf("Warning: TEE attestation failed: %v", err)
		// } else {
		// 	container.TEEAttester = attester
		// 	container.Attestation = attestation
		// }
	}

	// Initialize 3D renderer if object is 3D type
	if objectType == ObjectType3D {
		glbRenderer := NewGLBRendererImpl()
		container.GLBRenderer = glbRenderer
	}

	// Create container via runtime
	if err := runtime.Create(spec); err != nil {
		return nil, fmt.Errorf("container creation failed: %w", err)
	}

	// Start the container
	if err := runtime.Start(); err != nil {
		runtime.Delete() // Cleanup on failure
		return nil, fmt.Errorf("container start failed: %w", err)
	}

	container.Status = ContainerStatusRunning

	// Generate cryptographic hash for routing
	container.CryptoHash = ucm.generateCryptoHash(container)

	// Register with KNIRVROUTER DHT
	if err := ucm.registerWithRouter(ctx, container); err != nil {
		log.Printf("Warning: Router registration failed: %v", err)
	}

	ucm.mu.Lock()
	ucm.containers[container.ID] = container
	ucm.mu.Unlock()

	return container, nil
}

// CreateNestedObject creates a new NOC container (universal method)
func (ucm *UnifiedContainerManager) CreateNestedObject(
	ctx context.Context,
	config *NestedObjectConfig,
) (*UnifiedContainer, error) {
	if config == nil {
		return nil, fmt.Errorf("nested object config is required")
	}

	// Build container spec based on object type
	spec := ucm.buildSpecForObjectType(config)

	container, err := ucm.CreateContainer(
		ctx,
		spec,
		RuntimeModeObject,
		config.ObjectType,
		SecurityLevelStrong,
	)
	if err != nil {
		return nil, err
	}

	container.ObjectConfig = config

	// Initialize viewport proxy if enabled
	if config.EnableViewport {
		viewportProxy := NewViewportProxyImpl(container, config.ViewportRenderers)
		if err := viewportProxy.Start(); err != nil {
			log.Printf("Warning: Viewport proxy failed: %v", err)
		} else {
			container.ViewportProxy = viewportProxy
		}
	}

	return container, nil
}

// GetContainer retrieves a container by ID
func (ucm *UnifiedContainerManager) GetContainer(id string) (*UnifiedContainer, error) {
	ucm.mu.RLock()
	defer ucm.mu.RUnlock()

	container, ok := ucm.containers[id]
	if !ok {
		return nil, fmt.Errorf("container not found: %s", id)
	}

	return container, nil
}

// ListContainers lists all containers
func (ucm *UnifiedContainerManager) ListContainers() []*UnifiedContainer {
	ucm.mu.RLock()
	defer ucm.mu.RUnlock()

	containers := make([]*UnifiedContainer, 0, len(ucm.containers))
	for _, container := range ucm.containers {
		containers = append(containers, container)
	}

	return containers
}

// DestroyContainer stops and removes a container
func (ucm *UnifiedContainerManager) DestroyContainer(ctx context.Context, id string) error {
	ucm.mu.Lock()
	container, ok := ucm.containers[id]
	if !ok {
		ucm.mu.Unlock()
		return fmt.Errorf("container not found: %s", id)
	}
	delete(ucm.containers, id)
	ucm.mu.Unlock()

	// Stop viewport proxy if running
	if container.ViewportProxy != nil {
		if err := container.ViewportProxy.Stop(); err != nil {
			log.Printf("Error stopping viewport proxy: %v", err)
		}
	}

	// Close GLB renderer if active
	if container.GLBRenderer != nil {
		if err := container.GLBRenderer.Close(); err != nil {
			log.Printf("Error closing GLB renderer: %v", err)
		}
	}

	// Close model server if active
	if container.ModelServer != nil {
		if err := container.ModelServer.Close(); err != nil {
			log.Printf("Error closing model server: %v", err)
		}
	}

	// Unregister from router
	if err := ucm.unregisterFromRouter(ctx, container); err != nil {
		log.Printf("Warning: Router unregistration failed: %v", err)
	}

	// Stop and delete container
	if err := container.Runtime.Stop(); err != nil {
		log.Printf("Error stopping container: %v", err)
	}

	if err := container.Runtime.Delete(); err != nil {
		return fmt.Errorf("container deletion failed: %w", err)
	}

	container.Status = ContainerStatusStopped

	return nil
}

// buildSpecForObjectType builds container spec based on object type
func (ucm *UnifiedContainerManager) buildSpecForObjectType(config *NestedObjectConfig) *ContainerSpec {
	spec := &ContainerSpec{
		ID:          generateContainerID(),
		NetworkMode: "bridge",
		Resources: ResourceLimits{
			CPUCores: 4,
			MemoryMB: 8192,
			DiskMB:   20480,
		},
		Environment: make(map[string]string),
	}

	// Configure based on object type
	switch config.ObjectType {
	case ObjectType3D:
		spec.Image = "glb-renderer:latest"
		spec.Command = []string{"/usr/local/bin/renderer", "--serve"}
		spec.Environment = map[string]string{
			"RENDERER_MODE":  "webgl",
			"ENABLE_SHADOWS": "true",
			"ENABLE_PBR":     "true",
		}
		spec.Ports = []PortMapping{
			{ContainerPort: 8080, HostPort: 0, Protocol: "tcp"},
		}

	case ObjectTypeAPI, ObjectTypeWebApp:
		// Generic web service container
		spec.Image = "alpine:latest"
		spec.Ports = []PortMapping{
			{ContainerPort: 8080, HostPort: 0, Protocol: "tcp"},
		}

	case ObjectTypeP2P:
		// P2P router/node
		spec.Image = "knirvrouter:latest"
		spec.Ports = []PortMapping{
			{ContainerPort: 5001, HostPort: 0, Protocol: "tcp"}, // API
			{ContainerPort: 4001, HostPort: 0, Protocol: "tcp"}, // P2P
			{ContainerPort: 3478, HostPort: 0, Protocol: "udp"}, // STUN
			{ContainerPort: 3479, HostPort: 0, Protocol: "udp"}, // TURN
		}
		spec.CapabilitySet = []string{"NET_ADMIN", "NET_BIND_SERVICE"}

	case ObjectTypeBlockchain, ObjectTypeOracle:
		// Blockchain/oracle node
		spec.Image = "blockchain-node:latest"
		spec.Resources.CPUCores = 8
		spec.Resources.MemoryMB = 16384

	case ObjectTypeModel:
		// Model server
		spec.Image = "model-server:latest"
		spec.Resources.CPUCores = 8
		spec.Resources.MemoryMB = 32768
		spec.Ports = []PortMapping{
			{ContainerPort: 11434, HostPort: 0, Protocol: "tcp"},
		}

	default:
		// Custom object type - use provided metadata
		if config.Metadata != nil {
			if image, ok := config.Metadata["image"].(string); ok {
				spec.Image = image
			}
			if cmd, ok := config.Metadata["command"].([]interface{}); ok {
				spec.Command = make([]string, len(cmd))
				for i, c := range cmd {
					if str, ok := c.(string); ok {
						spec.Command[i] = str
					}
				}
			}
			if env, ok := config.Metadata["environment"].(map[string]interface{}); ok {
				for k, v := range env {
					if str, ok := v.(string); ok {
						spec.Environment[k] = str
					}
				}
			}
			if vols, ok := config.Metadata["volumes"].(map[string]interface{}); ok {
				spec.Volumes = make([]VolumeMount, 0, len(vols))
				for source, target := range vols {
					if targetStr, ok := target.(string); ok {
						spec.Volumes = append(spec.Volumes, VolumeMount{
							Source:   source,
							Target:   targetStr,
							ReadOnly: false,
						})
					}
				}
			}
			if caps, ok := config.Metadata["capabilities"].([]interface{}); ok {
				spec.CapabilitySet = make([]string, 0, len(caps))
				for _, cap := range caps {
					if capStr, ok := cap.(string); ok {
						spec.CapabilitySet = append(spec.CapabilitySet, capStr)
					}
				}
			}
		}
	}

	// Apply custom service ports from config
	if len(config.ServicePorts) > 0 {
		spec.Ports = make([]PortMapping, 0, len(config.ServicePorts))
		for _, port := range config.ServicePorts {
			spec.Ports = append(spec.Ports, PortMapping{
				ContainerPort: port,
				HostPort:      0, // Auto-assign
				Protocol:      "tcp",
			})
		}
	}

	return spec
}

// generateCryptoHash generates a cryptographic hash for container routing
func (ucm *UnifiedContainerManager) generateCryptoHash(c *UnifiedContainer) string {
	// Use BLAKE3 for cryptographic hash derivation
	// Concatenate all hash components:
	// 1. Container ID
	// 2. Node public key (from P2P identity)
	// 3. Genesis timestamp
	// 4. Container creation time
	// 5. Object type (NEW: allows type-based routing)

	data := fmt.Sprintf("%s:%s:%s:%s:%s",
		c.ID,
		ucm.runtimeSelector.NodePublicKey,
		ucm.runtimeSelector.GenesisTime.Format(time.RFC3339),
		c.CreatedAt.Format(time.RFC3339),
		c.ObjectType,
	)

	// Compute BLAKE3 hash
	hash := blake3.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// registerWithRouter registers the container with KNIRVROUTER DHT
func (ucm *UnifiedContainerManager) registerWithRouter(
	_ context.Context,
	c *UnifiedContainer,
) error {
	// TODO: Implement actual KNIRVROUTER client integration
	// For now, this is a placeholder that will be implemented in Phase 2
	log.Printf("Registering container %s with router (hash: %s, type: %s)",
		c.ID, c.CryptoHash, c.ObjectType)
	return nil
}

// unregisterFromRouter unregisters the container from KNIRVROUTER DHT
func (ucm *UnifiedContainerManager) unregisterFromRouter(
	_ context.Context,
	c *UnifiedContainer,
) error {
	// TODO: Implement actual KNIRVROUTER client integration
	log.Printf("Unregistering container %s from router (hash: %s)",
		c.ID, c.CryptoHash)
	return nil
}

// generateContainerID generates a unique container ID
func generateContainerID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("container-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("container-%x", b)
}
