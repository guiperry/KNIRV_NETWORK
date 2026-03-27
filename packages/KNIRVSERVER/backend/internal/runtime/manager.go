// Copyright 2026 KNIRV-SERVER
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

// CognitiveEngineInterface defines the interface for Cognitive Engine integration.
// It is implemented by cognitiveengine.CognitiveEngine (outer, server-wide engine)
// and called from within the UCM to report per-DVE agent metrics.
type CognitiveEngineInterface interface {
	OnAgentTaskComplexity(dveID string, complexity int) error
	OnAgentResourceUsage(dveID string, cpuPercent float64, memoryMB int64) error
}

// ContainerRegisterFunc is the callback invoked when a new container must be
// announced to the P2P routing layer (e.g. via the DVE announcement topic).
type ContainerRegisterFunc func(ctx context.Context, containerID, cryptoHash, objectType string) error

// ContainerUnregisterFunc is the callback invoked when a container is torn down
// and must be removed from the P2P routing layer.
type ContainerUnregisterFunc func(ctx context.Context, containerID string) error

// UnifiedContainerManager manages all container types
type UnifiedContainerManager struct {
	teeService      *teesecurity.TEESecurityService
	ebpfManager     ebpf.ManagerInterface
	vcManager       *ebpf.VirtualContainerManager
	runtimeSelector *RuntimeSelector
	assetRegistry   *AssetRegistry
	cognitiveEngine CognitiveEngineInterface
	// router callbacks wired by SetRouterFuncs from main.go
	registerFn   ContainerRegisterFunc
	unregisterFn ContainerUnregisterFunc
	containers   map[string]*UnifiedContainer
	agentPolicies map[string]*ebpf.SecurityPolicy
	mu            sync.RWMutex
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
		agentPolicies:   make(map[string]*ebpf.SecurityPolicy),
	}
}

// SetCognitiveEngine wires the outer (server-wide) CognitiveEngine as the
// receiver for per-DVE agent task-complexity and resource-usage callbacks.
// Must be called before containers are created.
func (ucm *UnifiedContainerManager) SetCognitiveEngine(engine CognitiveEngineInterface) {
	ucm.mu.Lock()
	defer ucm.mu.Unlock()
	ucm.cognitiveEngine = engine
}

// SetRouterFuncs wires the P2P routing callbacks so that newly created
// containers are announced to the DVE P2P overlay and removed when they are
// torn down.  Must be called before containers are created.
func (ucm *UnifiedContainerManager) SetRouterFuncs(register ContainerRegisterFunc, unregister ContainerUnregisterFunc) {
	ucm.mu.Lock()
	defer ucm.mu.Unlock()
	ucm.registerFn = register
	ucm.unregisterFn = unregister
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

	// Initialize eBPF security for all containers
	if ucm.ebpfManager != nil {
		if err := ucm.initializeEBPFSecurity(container, objectType); err != nil {
			log.Printf("Warning: eBPF security initialization failed for %s: %v", container.ID, err)
		}
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

	// Initialize viewport proxy for agent type (provides terminal interface)
	if objectType == ObjectTypeAgent && spec != nil {
		viewportProxy := NewViewportProxyImpl(container, []string{"http", "websocket"})
		if err := viewportProxy.Start(); err != nil {
			log.Printf("Warning: Agent viewport proxy failed: %v", err)
		} else {
			container.ViewportProxy = viewportProxy
		}

		// Initialize agent security policy for oh-my-pi
		if err := ucm.initializeOhMyPiAgentPolicy(container); err != nil {
			log.Printf("Warning: oh-my-pi agent policy initialization failed: %v", err)
		}
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

	case ObjectTypeAgent:
		// oh-my-pi agentic runtime container
		spec.Image = "knirv-agent-oh-my-pi:latest"
		spec.Command = []string{"/usr/local/bin/oh-my-pi", "--serve", "--port", "8080"}
		spec.Environment = map[string]string{
			"OH_MY_PI_MODE":       "server",
			"OH_MY_PI_WORKSPACE":  "/workspace/active-memory",
			"OH_MY_PI_TOOLS":      "git,python,curl,browser,lsp",
			"MARKDOWN_OUTPUT_DIR": "/workspace/active-memory/agent-output",
			"VIEWPORT_PORT":       "8080",
			"JUPYTER_PORT":        "8888",
			"LSP_ENABLED":         "true",
			"HEADLESS_BROWSER":    "true",
		}
		spec.Ports = []PortMapping{
			{ContainerPort: 8080, HostPort: 0, Protocol: "tcp"}, // Agent viewport
			{ContainerPort: 8888, HostPort: 0, Protocol: "tcp"}, // Jupyter kernel
			{ContainerPort: 9090, HostPort: 0, Protocol: "tcp"}, // LSP server
		}
		spec.Volumes = []VolumeMount{
			{Source: "/var/knirv/active-memory", Target: "/workspace/active-memory", ReadOnly: false},
			{Source: "/var/knirv/workspace", Target: "/workspace", ReadOnly: false},
		}
		spec.Resources = ResourceLimits{
			CPUCores: 4,
			MemoryMB: 8192,
			DiskMB:   40960,
		}
	}

	// Apply metadata overrides for all object types
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

// registerWithRouter announces the container to the P2P routing layer via the
// callback registered with SetRouterFuncs.  When no callback is registered
// (e.g. in standalone / unit-test mode) the call is silently skipped.
func (ucm *UnifiedContainerManager) registerWithRouter(ctx context.Context, c *UnifiedContainer) error {
	ucm.mu.RLock()
	fn := ucm.registerFn
	ucm.mu.RUnlock()
	if fn == nil {
		return nil
	}
	if err := fn(ctx, c.ID, c.CryptoHash, string(c.ObjectType)); err != nil {
		return fmt.Errorf("router registration for container %s failed: %w", c.ID, err)
	}
	log.Printf("Container %s registered with P2P router (hash: %s, type: %s)",
		c.ID, c.CryptoHash, c.ObjectType)
	return nil
}

// unregisterFromRouter removes the container from the P2P routing layer via
// the callback registered with SetRouterFuncs.
func (ucm *UnifiedContainerManager) unregisterFromRouter(ctx context.Context, c *UnifiedContainer) error {
	ucm.mu.RLock()
	fn := ucm.unregisterFn
	ucm.mu.RUnlock()
	if fn == nil {
		return nil
	}
	if err := fn(ctx, c.ID); err != nil {
		return fmt.Errorf("router deregistration for container %s failed: %w", c.ID, err)
	}
	log.Printf("Container %s unregistered from P2P router", c.ID)
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

// initializeEBPFSecurity initializes eBPF security for a container
func (ucm *UnifiedContainerManager) initializeEBPFSecurity(container *UnifiedContainer, objectType ObjectType) error {
	if ucm.ebpfManager == nil {
		return fmt.Errorf("eBPF manager not available")
	}

	// Create syscall monitor with appropriate policy based on object type
	var policy *ebpf.AgentPolicyConfig
	switch objectType {
	case ObjectTypeAgent:
		policy = NewOhMyPiAgentPolicyConfig()
	default:
		// Default minimal policy for other object types
		policy = &ebpf.AgentPolicyConfig{
			AllowNetwork:    false,
			AllowFilesystem: false,
			RequireTEE:      false,
			MaxMemoryMB:     4096,
			MaxCPUPercent:   50,
			AllowedSyscalls: getMinimalSyscalls(),
			AllowedPaths:    []string{"/tmp"},
			AllowedNetworks: []string{},
		}
	}

	// Create security policy
	securityPolicyManager := ebpf.NewSecurityProfileManager()
	sp, err := securityPolicyManager.CreateAgentPolicy(0, policy)
	if err != nil {
		return fmt.Errorf("failed to create security policy: %w", err)
	}

	// Store the policy for this container
	ucm.mu.Lock()
	ucm.agentPolicies[container.ID] = sp
	ucm.mu.Unlock()

	log.Printf("eBPF security initialized for container %s (type: %s)", container.ID, objectType)
	return nil
}

// initializeOhMyPiAgentPolicy initializes the specific security policy for oh-my-pi agents
func (ucm *UnifiedContainerManager) initializeOhMyPiAgentPolicy(container *UnifiedContainer) error {
	if container.Spec == nil {
		return fmt.Errorf("container spec is required")
	}

	// Get or create security policy
	ucm.mu.RLock()
	policy, exists := ucm.agentPolicies[container.ID]
	ucm.mu.RUnlock()

	if !exists {
		// Create policy if not already created
		agentPolicy := NewOhMyPiAgentPolicyConfig()
		securityPolicyManager := ebpf.NewSecurityProfileManager()
		var err error
		policy, err = securityPolicyManager.CreateAgentPolicy(0, agentPolicy)
		if err != nil {
			return fmt.Errorf("failed to create oh-my-pi agent policy: %w", err)
		}

		ucm.mu.Lock()
		ucm.agentPolicies[container.ID] = policy
		ucm.mu.Unlock()
	}

	// Report to Cognitive Engine if available
	ucm.mu.RLock()
	cognitiveEngine := ucm.cognitiveEngine
	ucm.mu.RUnlock()

	if cognitiveEngine != nil {
		// Estimate complexity based on resource limits
		complexity := estimateTaskComplexity(container.Spec.Resources)
		if err := cognitiveEngine.OnAgentTaskComplexity(container.ID, complexity); err != nil {
			log.Printf("Warning: failed to report task complexity to cognitive engine: %v", err)
		}
	}

	log.Printf("oh-my-pi agent policy initialized for container %s", container.ID)
	return nil
}

// getMinimalSyscalls returns the minimal set of syscalls for basic containers
func getMinimalSyscalls() []uint32 {
	return []uint32{
		1,   // exit
		9,   // mmap
		10,  // munmap
		11,  // brk
		21,  // access
		35,  // nanosleep
		60,  // exit_group
		231, // clock_nanosleep
		257, // openat
		259, // read
		262, // newfstatat
		280, // utimensat
		281, // write
	}
}

// estimateTaskComplexity estimates task complexity based on resource limits
func estimateTaskComplexity(resources ResourceLimits) int {
	complexity := 1

	// Factor in CPU cores
	if resources.CPUCores >= 8 {
		complexity += 3
	} else if resources.CPUCores >= 4 {
		complexity += 2
	} else if resources.CPUCores >= 2 {
		complexity += 1
	}

	// Factor in memory
	if resources.MemoryMB >= 16384 {
		complexity += 3
	} else if resources.MemoryMB >= 8192 {
		complexity += 2
	} else if resources.MemoryMB >= 4096 {
		complexity += 1
	}

	// Factor in disk
	if resources.DiskMB >= 20480 {
		complexity += 2
	} else if resources.DiskMB >= 10240 {
		complexity += 1
	}

	return complexity
}

// GetAgentPolicy returns the security policy for an agent container
func (ucm *UnifiedContainerManager) GetAgentPolicy(containerID string) *ebpf.SecurityPolicy {
	ucm.mu.RLock()
	defer ucm.mu.RUnlock()
	return ucm.agentPolicies[containerID]
}

// UpdateAgentResourceUsage reports resource usage to the Cognitive Engine
func (ucm *UnifiedContainerManager) UpdateAgentResourceUsage(dveID string, cpuPercent float64, memoryMB int64) error {
	ucm.mu.RLock()
	cognitiveEngine := ucm.cognitiveEngine
	ucm.mu.RUnlock()

	if cognitiveEngine != nil {
		return cognitiveEngine.OnAgentResourceUsage(dveID, cpuPercent, memoryMB)
	}
	return nil
}

// CreateAgentRuntimeCapability creates the capability advertisement for P2P discovery
func (ucm *UnifiedContainerManager) CreateAgentRuntimeCapability(nodeID string) *AgentRuntimeCapability {
	return &AgentRuntimeCapability{
		NodeID:            nodeID,
		AgentEngineVer:    "oh-my-pi-1.0",
		SupportedTools:    []string{"git", "python", "curl", "browser", "lsp", "shell", "docker"},
		ActiveMemoryMount: "/workspace/active-memory",
		MaxConcurrent:     4,
		ViewportEnabled:   true,
	}
}
