package host

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ContainerManager manages container orchestration
type ContainerManager struct {
	ctx    context.Context
	config *HostConfig
	mu     sync.RWMutex

	runtime    string // docker or podman
	containers map[string]*Container
	networks   []ContainerNetwork

	lastUpdate time.Time
	running    bool
}

// Container represents a container instance
type Container struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Image     string    `json:"image"`
	Status    string    `json:"status"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	StartedAt time.Time `json:"started_at"`

	// Resource usage
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryUsage uint64  `json:"memory_usage"`
	MemoryLimit uint64  `json:"memory_limit"`
	NetworkRx   uint64  `json:"network_rx"`
	NetworkTx   uint64  `json:"network_tx"`

	// Configuration
	Ports       []PortMapping     `json:"ports"`
	Volumes     []VolumeMapping   `json:"volumes"`
	Environment map[string]string `json:"environment"`
	Networks    []string          `json:"networks"`

	// KNIRV-specific
	IsKNIRVContainer bool   `json:"is_knirv_container"`
	ServiceType      string `json:"service_type,omitempty"`
	P2PEnabled       bool   `json:"p2p_enabled"`

	// Security
	Privileged     bool     `json:"privileged"`
	SecurityOpts   []string `json:"security_opts"`
	ReadOnlyRootfs bool     `json:"read_only_rootfs"`
}

// ContainerNetwork represents a container network
type ContainerNetwork struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Driver   string            `json:"driver"`
	Scope    string            `json:"scope"`
	Internal bool              `json:"internal"`
	Options  map[string]string `json:"options"`

	// KNIRV-specific
	IsKNIRVNetwork bool `json:"is_knirv_network"`
	P2PEnabled     bool `json:"p2p_enabled"`
	Encrypted      bool `json:"encrypted"`
}

// PortMapping represents a port mapping
type PortMapping struct {
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
	HostIP        string `json:"host_ip"`
}

// VolumeMapping represents a volume mapping
type VolumeMapping struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Mode          string `json:"mode"` // ro, rw
	Type          string `json:"type"` // bind, volume, tmpfs
}

// NewContainerManager creates a new container manager
func NewContainerManager(ctx context.Context, config *HostConfig) (*ContainerManager, error) {
	cm := &ContainerManager{
		ctx:        ctx,
		config:     config,
		runtime:    config.ContainerRuntime,
		containers: make(map[string]*Container),
	}

	// Verify container runtime is available
	if err := cm.verifyRuntime(); err != nil {
		return nil, fmt.Errorf("container runtime verification failed: %w", err)
	}

	return cm, nil
}

// Start begins container monitoring
func (cm *ContainerManager) Start() error {
	cm.mu.Lock()
	if cm.running {
		cm.mu.Unlock()
		return fmt.Errorf("container manager is already running")
	}
	cm.running = true
	cm.mu.Unlock()

	// Initial container scan
	if err := cm.scanContainers(); err != nil {
		return fmt.Errorf("initial container scan failed: %w", err)
	}

	if err := cm.scanNetworks(); err != nil {
		return fmt.Errorf("initial network scan failed: %w", err)
	}

	// Setup KNIRV networks if needed
	if err := cm.setupKNIRVNetworks(); err != nil {
		return fmt.Errorf("failed to setup KNIRV networks: %w", err)
	}

	// Start monitoring loop
	go cm.monitorLoop()

	return nil
}

// Stop stops container monitoring
func (cm *ContainerManager) Stop() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.running = false
	return nil
}

// GetContainerList returns current container list
func (cm *ContainerManager) GetContainerList() ([]*Container, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var containers []*Container
	for _, container := range cm.containers {
		// Return a copy to prevent modification
		containerCopy := *container
		containers = append(containers, &containerCopy)
	}

	return containers, nil
}

// GetKNIRVContainers returns only KNIRV-related containers
func (cm *ContainerManager) GetKNIRVContainers() ([]*Container, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var knirvContainers []*Container
	for _, container := range cm.containers {
		if container.IsKNIRVContainer {
			containerCopy := *container
			knirvContainers = append(knirvContainers, &containerCopy)
		}
	}

	return knirvContainers, nil
}

// HealthCheck verifies the container manager is working properly
func (cm *ContainerManager) HealthCheck() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.running {
		return fmt.Errorf("container manager is not running")
	}

	// Check if runtime is still available
	if err := cm.verifyRuntime(); err != nil {
		return fmt.Errorf("container runtime not available: %w", err)
	}

	// Check if data is stale
	if time.Since(cm.lastUpdate) > 60*time.Second {
		return fmt.Errorf("container data is stale (last update: %v)", cm.lastUpdate)
	}

	return nil
}

// verifyRuntime verifies the container runtime is available
func (cm *ContainerManager) verifyRuntime() error {
	cmd := exec.Command(cm.runtime, "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s runtime not available: %w", cm.runtime, err)
	}
	return nil
}

// monitorLoop runs the periodic monitoring loop
func (cm *ContainerManager) monitorLoop() {
	ticker := time.NewTicker(30 * time.Second) // Container monitoring every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.mu.RLock()
			running := cm.running
			cm.mu.RUnlock()

			if !running {
				return
			}

			if err := cm.scanContainers(); err != nil {
				fmt.Printf("Error scanning containers: %v\n", err)
			}

			if err := cm.scanNetworks(); err != nil {
				fmt.Printf("Error scanning networks: %v\n", err)
			}
		}
	}
}

// scanContainers scans and updates container information
func (cm *ContainerManager) scanContainers() error {
	// Get container list
	cmd := exec.Command(cm.runtime, "ps", "-a", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Parse container information
	containers := make(map[string]*Container)

	// Split output by lines for JSON parsing
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		container, err := cm.parseContainerJSON(line)
		if err != nil {
			continue // Skip invalid entries
		}

		// Get detailed container information
		if err := cm.enrichContainerInfo(container); err != nil {
			fmt.Printf("Warning: failed to enrich container info for %s: %v\n", container.ID, err)
		}

		containers[container.ID] = container
	}

	cm.mu.Lock()
	cm.containers = containers
	cm.lastUpdate = time.Now()
	cm.mu.Unlock()

	return nil
}

// parseContainerJSON parses container JSON output
func (cm *ContainerManager) parseContainerJSON(jsonLine string) (*Container, error) {
	var rawContainer map[string]interface{}
	if err := json.Unmarshal([]byte(jsonLine), &rawContainer); err != nil {
		return nil, fmt.Errorf("failed to parse container JSON: %w", err)
	}

	container := &Container{
		ID:     getString(rawContainer, "ID"),
		Name:   getString(rawContainer, "Names"),
		Image:  getString(rawContainer, "Image"),
		Status: getString(rawContainer, "Status"),
		State:  getString(rawContainer, "State"),
	}

	// Parse creation time
	if createdStr := getString(rawContainer, "CreatedAt"); createdStr != "" {
		if created, err := time.Parse(time.RFC3339, createdStr); err == nil {
			container.CreatedAt = created
		}
	}

	// Identify KNIRV containers
	cm.identifyKNIRVContainer(container)

	return container, nil
}

// getString safely gets a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// enrichContainerInfo gets detailed container information
func (cm *ContainerManager) enrichContainerInfo(container *Container) error {
	// Get detailed container inspect information
	cmd := exec.Command(cm.runtime, "inspect", container.ID)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	var inspectData []map[string]interface{}
	if err := json.Unmarshal(output, &inspectData); err != nil {
		return fmt.Errorf("failed to parse inspect data: %w", err)
	}

	if len(inspectData) == 0 {
		return fmt.Errorf("no inspect data returned")
	}

	data := inspectData[0]

	// Extract additional information
	if config, ok := data["Config"].(map[string]interface{}); ok {
		// Environment variables
		if env, ok := config["Env"].([]interface{}); ok {
			container.Environment = make(map[string]string)
			for _, envVar := range env {
				if envStr, ok := envVar.(string); ok {
					parts := strings.SplitN(envStr, "=", 2)
					if len(parts) == 2 {
						container.Environment[parts[0]] = parts[1]
					}
				}
			}
		}
	}

	// Network settings
	if networkSettings, ok := data["NetworkSettings"].(map[string]interface{}); ok {
		if networks, ok := networkSettings["Networks"].(map[string]interface{}); ok {
			for networkName := range networks {
				container.Networks = append(container.Networks, networkName)
			}
		}
	}

	return nil
}

// identifyKNIRVContainer identifies if a container is KNIRV-related
func (cm *ContainerManager) identifyKNIRVContainer(container *Container) {
	knirvKeywords := []string{
		"knirv", "nexus", "dve-manager", "validation-core",
		"model-server", "data_engine", "inference",
	}

	nameLower := strings.ToLower(container.Name)
	imageLower := strings.ToLower(container.Image)

	for _, keyword := range knirvKeywords {
		if strings.Contains(nameLower, keyword) || strings.Contains(imageLower, keyword) {
			container.IsKNIRVContainer = true

			// Determine service type
			if strings.Contains(nameLower, "dve-manager") || strings.Contains(imageLower, "dve-manager") {
				container.ServiceType = "dve-manager"
			} else if strings.Contains(nameLower, "validation-core") || strings.Contains(imageLower, "validation-core") {
				container.ServiceType = "validation-core"
			} else if strings.Contains(nameLower, "model-server") || strings.Contains(imageLower, "model-server") {
				container.ServiceType = "model-server"
			} else if strings.Contains(nameLower, "data_engine") || strings.Contains(imageLower, "data_engine") {
				container.ServiceType = "data_engine"
			} else if strings.Contains(nameLower, "inference") || strings.Contains(imageLower, "inference") {
				container.ServiceType = "inference"
			} else {
				container.ServiceType = "knirv-other"
			}

			// Check for P2P enablement
			container.P2PEnabled = strings.Contains(nameLower, "p2p") ||
				strings.Contains(imageLower, "p2p") ||
				container.ServiceType == "dve-manager"

			break
		}
	}
}

// scanNetworks scans container networks
func (cm *ContainerManager) scanNetworks() error {
	cmd := exec.Command(cm.runtime, "network", "ls", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	var networks []ContainerNetwork

	// Parse network information
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		network, err := cm.parseNetworkJSON(line)
		if err != nil {
			continue // Skip invalid entries
		}

		networks = append(networks, *network)
	}

	cm.mu.Lock()
	cm.networks = networks
	cm.mu.Unlock()

	return nil
}

// parseNetworkJSON parses network JSON output
func (cm *ContainerManager) parseNetworkJSON(jsonLine string) (*ContainerNetwork, error) {
	var rawNetwork map[string]interface{}
	if err := json.Unmarshal([]byte(jsonLine), &rawNetwork); err != nil {
		return nil, fmt.Errorf("failed to parse network JSON: %w", err)
	}

	network := &ContainerNetwork{
		ID:     getString(rawNetwork, "ID"),
		Name:   getString(rawNetwork, "Name"),
		Driver: getString(rawNetwork, "Driver"),
		Scope:  getString(rawNetwork, "Scope"),
	}

	// Identify KNIRV networks
	cm.identifyKNIRVNetwork(network)

	return network, nil
}

// identifyKNIRVNetwork identifies if a network is KNIRV-related
func (cm *ContainerManager) identifyKNIRVNetwork(network *ContainerNetwork) {
	knirvPatterns := []string{"knirv", "nexus", "p2p", "dve"}

	nameLower := strings.ToLower(network.Name)

	for _, pattern := range knirvPatterns {
		if strings.Contains(nameLower, pattern) {
			network.IsKNIRVNetwork = true
			network.P2PEnabled = strings.Contains(nameLower, "p2p")
			network.Encrypted = true // Default to encrypted for KNIRV networks
			break
		}
	}
}

// setupKNIRVNetworks sets up necessary networks for KNIRV services
func (cm *ContainerManager) setupKNIRVNetworks() error {
	// Check if KNIRV network exists
	knirvNetworkExists := false
	for _, network := range cm.networks {
		if network.Name == "knirv-server" {
			knirvNetworkExists = true
			break
		}
	}

	// Create KNIRV network if it doesn't exist
	if !knirvNetworkExists {
		cmd := exec.Command(cm.runtime, "network", "create",
			"--driver", "bridge",
			"--subnet", "172.20.0.0/16",
			"--opt", "encrypted=true",
			"knirv-server")

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create KNIRV network: %w", err)
		}

		fmt.Println("Created KNIRV network: knirv-server")
	}

	return nil
}
