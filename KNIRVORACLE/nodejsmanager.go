package main

import (
	"KNIRVORACLE/utils"
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"KNIRVORACLE/config"
)

// NodeJSProcess represents a running Node.js process
type NodeJSProcess struct {
	Name       string
	Cmd        *exec.Cmd
	StdoutPipe *bufio.Scanner
	StderrPipe *bufio.Scanner
	StopChan   chan struct{}
}

// NodeJSManager manages Node.js processes
type NodeJSManager struct {
	Config         *config.NodeJSServicesConfig
	Processes      map[string]*NodeJSProcess
	PeerID         string
	mutex          sync.Mutex
	DiscoveryMgr   DiscoveryService
	Blockchain     *BlockchainStruct
}

// NewNodeJSManager creates a new Node.js process manager, accepting DiscoveryService and Blockchain
func NewNodeJSManager(cfg *config.NodeJSServicesConfig, nodeID string, discoveryMgr DiscoveryService, bc *BlockchainStruct) *NodeJSManager {
	return &NodeJSManager{
		Config:       cfg,
		Processes:    make(map[string]*NodeJSProcess),
		PeerID:       nodeID,
		DiscoveryMgr: discoveryMgr,
		Blockchain:   bc,
	}
}

// StartTunnelRegistry starts the tunnel registry service
func (m *NodeJSManager) StartTunnelRegistry() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.Config.Enabled || !m.Config.TunnelRegistry.Enabled {
		return fmt.Errorf("tunnel registry service is not enabled in configuration")
	}

	// Check if already running
	if _, exists := m.Processes["tunnel-registry"]; exists {
		return fmt.Errorf("tunnel registry service is already running")
	}

	scriptPath := m.Config.TunnelRegistry.ScriptPath
	if scriptPath == "" {
		return fmt.Errorf("no script path provided for tunnel registry service")
	}

	// Check if the script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found for tunnel registry service: %s", scriptPath)
	}

	// --- Find available ports ---
	actualHTTPPort := utils.FindAvailablePort(uint64(m.Config.TunnelRegistry.HTTPPort))
	if actualHTTPPort != uint64(m.Config.TunnelRegistry.HTTPPort) {
		log.Printf("[TunnelRegistry] HTTP Port %d in use, using %d instead.", m.Config.TunnelRegistry.HTTPPort, actualHTTPPort)
	}
	actualControlPort := utils.FindAvailablePort(uint64(m.Config.TunnelRegistry.ControlPort))
	if actualControlPort != uint64(m.Config.TunnelRegistry.ControlPort) {
		log.Printf("[TunnelRegistry] Control Port %d in use, using %d instead.", m.Config.TunnelRegistry.ControlPort, actualControlPort)
	}
	actualPublicRelayPort := utils.FindAvailablePort(uint64(m.Config.TunnelRegistry.PublicRelayPort))
	if actualPublicRelayPort != uint64(m.Config.TunnelRegistry.PublicRelayPort) {
		log.Printf("[TunnelRegistry] Public Relay Port %d in use, using %d instead.", m.Config.TunnelRegistry.PublicRelayPort, actualPublicRelayPort)
	}
	actualSTUNPort := utils.FindAvailablePort(uint64(m.Config.TunnelRegistry.STUNPort))
	if actualSTUNPort != uint64(m.Config.TunnelRegistry.STUNPort) {
		log.Printf("[TunnelRegistry] STUN Port %d in use, using %d instead.", m.Config.TunnelRegistry.STUNPort, actualSTUNPort)
	}

	// Find available port for Go internal API
	goInternalAPIPort := utils.FindAvailablePort(8080) // Default to 8080 if not specified
	
	// Prepare environment variables
	env := []string{
		fmt.Sprintf("HTTP_API_PORT=%d", actualHTTPPort),
		fmt.Sprintf("CONTROL_PORT=%d", actualControlPort),
		fmt.Sprintf("PUBLIC_RELAY_PORT=%d", actualPublicRelayPort),
		fmt.Sprintf("STUN_PORT=%d", actualSTUNPort),
		fmt.Sprintf("PUBLIC_HOST=%s", m.Config.TunnelRegistry.ServerPublicHost),
		fmt.Sprintf("RELAY_SERVER_PEER_ID=%s", m.PeerID),
		fmt.Sprintf("GO_INTERNAL_API_PORT=%d", goInternalAPIPort), // Pass the Go internal API port
	}

	// Prepare the command
	cmd := exec.Command("node", scriptPath)
	cmd.Env = append(os.Environ(), env...)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stdout for tunnel registry service: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stderr for tunnel registry service: %v", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start tunnel registry service: %v", err)
	}

	log.Printf("Started tunnel registry service (PID: %d)", cmd.Process.Pid)

	// Create a process object
	process := &NodeJSProcess{
		Name:       "tunnel-registry",
		Cmd:        cmd,
		StdoutPipe: bufio.NewScanner(stdout),
		StderrPipe: bufio.NewScanner(stderr),
		StopChan:   make(chan struct{}),
	}

	// Handle stdout in a goroutine
	go func() {
		for process.StdoutPipe.Scan() {
			log.Printf("[TunnelRegistry] %s", process.StdoutPipe.Text())
		}
	}()

	// Handle stderr in a goroutine
	go func() {
		for process.StderrPipe.Scan() {
			log.Printf("[TunnelRegistry ERROR] %s", process.StderrPipe.Text())
		}
	}()

	// Store the process
	m.Processes["tunnel-registry"] = process

	return nil
}

// StartPaymentGateway starts the payment gateway service
func (m *NodeJSManager) StartPaymentGateway() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.Config.Enabled || !m.Config.PaymentGateway.Enabled {
		return fmt.Errorf("payment gateway service is not enabled in configuration")
	}

	// Check if already running
	if _, exists := m.Processes["payment-gateway"]; exists {
		return fmt.Errorf("payment gateway service is already running")
	}

	scriptPath := m.Config.PaymentGateway.ScriptPath
	if scriptPath == "" {
		return fmt.Errorf("no script path provided for payment gateway service")
	}

	// Check if the script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found for payment gateway service: %s", scriptPath)
	}

	// --- Find available port ---
	actualHTTPPort := utils.FindAvailablePort(uint64(m.Config.PaymentGateway.HTTPPort))
	if actualHTTPPort != uint64(m.Config.PaymentGateway.HTTPPort) {
		log.Printf("[PaymentGateway] HTTP Port %d in use, using %d instead.", m.Config.PaymentGateway.HTTPPort, actualHTTPPort)
	}

	// Prepare environment variables
	env := []string{
		fmt.Sprintf("HTTP_API_PORT=%d", actualHTTPPort),
		fmt.Sprintf("API_KEY=%s", m.Config.PaymentGateway.APIKey),
		fmt.Sprintf("NODE_ENV=%s", "production"), // Set to production by default
	}

	// Prepare the command
	cmd := exec.Command("node", scriptPath)
	cmd.Env = append(os.Environ(), env...)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stdout for payment gateway service: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stderr for payment gateway service: %v", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start payment gateway service: %v", err)
	}

	log.Printf("Started payment gateway service (PID: %d)", cmd.Process.Pid)

	// Create a process object
	process := &NodeJSProcess{
		Name:       "payment-gateway",
		Cmd:        cmd,
		StdoutPipe: bufio.NewScanner(stdout),
		StderrPipe: bufio.NewScanner(stderr),
		StopChan:   make(chan struct{}),
	}

	// Handle stdout in a goroutine
	go func() {
		for process.StdoutPipe.Scan() {
			log.Printf("[PaymentGateway] %s", process.StdoutPipe.Text())
		}
	}()

	// Handle stderr in a goroutine
	go func() {
		for process.StderrPipe.Scan() {
			log.Printf("[PaymentGateway ERROR] %s", process.StderrPipe.Text())
		}
	}()

	// Store the process
	m.Processes["payment-gateway"] = process

	return nil
}

// StartBootnodeRegistry starts the bootnode registry service
func (m *NodeJSManager) StartBootnodeRegistry() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.Config.Enabled || !m.Config.BootnodeRegistry.Enabled {
		return fmt.Errorf("bootnode registry service is not enabled in configuration")
	}

	// Check if already running
	if _, exists := m.Processes["bootnode-registry"]; exists {
		return fmt.Errorf("bootnode registry service is already running")
	}

	scriptPath := m.Config.BootnodeRegistry.ScriptPath
	if scriptPath == "" {
		return fmt.Errorf("no script path provided for bootnode registry service")
	}

	// Check if the script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found for bootnode registry service: %s", scriptPath)
	}

	// --- Find available port ---
	actualHTTPPort := utils.FindAvailablePort(uint64(m.Config.BootnodeRegistry.HTTPPort))
	if actualHTTPPort != uint64(m.Config.BootnodeRegistry.HTTPPort) {
		log.Printf("[BootnodeRegistry] HTTP Port %d in use, using %d instead.", m.Config.BootnodeRegistry.HTTPPort, actualHTTPPort)
	}

	// Prepare environment variables
	env := []string{
		fmt.Sprintf("HTTP_API_PORT=%d", actualHTTPPort),
		fmt.Sprintf("NODE_ENV=%s", "production"),
	}

	// Prepare the command
	cmd := exec.Command("node", scriptPath)
	cmd.Env = append(os.Environ(), env...)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stdout for bootnode registry service: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stderr for bootnode registry service: %v", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start bootnode registry service: %v", err)
	}

	log.Printf("Started bootnode registry service (PID: %d)", cmd.Process.Pid)

	// Create a process object
	process := &NodeJSProcess{
		Name:       "bootnode-registry",
		Cmd:        cmd,
		StdoutPipe: bufio.NewScanner(stdout),
		StderrPipe: bufio.NewScanner(stderr),
		StopChan:   make(chan struct{}),
	}

	// Handle stdout in a goroutine
	go func() {
		for process.StdoutPipe.Scan() {
			log.Printf("[BootnodeRegistry] %s", process.StdoutPipe.Text())
		}
	}()

	// Handle stderr in a goroutine
	go func() {
		for process.StderrPipe.Scan() {
			log.Printf("[BootnodeRegistry ERROR] %s", process.StderrPipe.Text())
		}
	}()

	// Store the process
	m.Processes["bootnode-registry"] = process

	return nil
}

// StartNotarySystem starts the notary system service
func (m *NodeJSManager) StartNotarySystem() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.Config.Enabled || !m.Config.NotarySystem.Enabled {
		return fmt.Errorf("notary system service is not enabled in configuration")
	}

	// Check if already running
	if _, exists := m.Processes["notary-system"]; exists {
		return fmt.Errorf("notary system service is already running")
	}

	scriptPath := m.Config.NotarySystem.ScriptPath
	if scriptPath == "" {
		return fmt.Errorf("no script path provided for notary system service")
	}

	// Check if the script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found for notary system service: %s", scriptPath)
	}

	// --- Find available port ---
	actualHTTPPort := utils.FindAvailablePort(uint64(m.Config.NotarySystem.HTTPPort))
	if actualHTTPPort != uint64(m.Config.NotarySystem.HTTPPort) {
		log.Printf("[NotarySystem] HTTP Port %d in use, using %d instead.", m.Config.NotarySystem.HTTPPort, actualHTTPPort)
	}

	// Prepare environment variables
	env := []string{
		fmt.Sprintf("HTTP_API_PORT=%d", actualHTTPPort),
		fmt.Sprintf("NODE_ENV=%s", "production"),
	}

	// Prepare the command
	cmd := exec.Command("node", scriptPath)
	cmd.Env = append(os.Environ(), env...)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stdout for notary system service: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stderr for notary system service: %v", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start notary system service: %v", err)
	}

	log.Printf("Started notary system service (PID: %d)", cmd.Process.Pid)

	// Create a process object
	process := &NodeJSProcess{
		Name:       "notary-system",
		Cmd:        cmd,
		StdoutPipe: bufio.NewScanner(stdout),
		StderrPipe: bufio.NewScanner(stderr),
		StopChan:   make(chan struct{}),
	}

	// Handle stdout in a goroutine
	go func() {
		for process.StdoutPipe.Scan() {
			log.Printf("[NotarySystem] %s", process.StdoutPipe.Text())
		}
	}()

	// Handle stderr in a goroutine
	go func() {
		for process.StderrPipe.Scan() {
			log.Printf("[NotarySystem ERROR] %s", process.StderrPipe.Text())
		}
	}()

	// Store the process
	m.Processes["notary-system"] = process

	return nil
}

// StartNetworkMonitor starts the network monitor service
func (m *NodeJSManager) StartNetworkMonitor() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.Config.Enabled || !m.Config.NetworkMonitor.Enabled {
		return fmt.Errorf("network monitor service is not enabled in configuration")
	}

	// Check if already running
	if _, exists := m.Processes["network-monitor"]; exists {
		return fmt.Errorf("network monitor service is already running")
	}

	scriptPath := m.Config.NetworkMonitor.ScriptPath
	if scriptPath == "" {
		return fmt.Errorf("no script path provided for network monitor service")
	}

	// Check if the script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found for network monitor service: %s", scriptPath)
	}

	// --- Find available port ---
	actualHTTPPort := utils.FindAvailablePort(uint64(m.Config.NetworkMonitor.HTTPPort))
	if actualHTTPPort != uint64(m.Config.NetworkMonitor.HTTPPort) {
		log.Printf("[NetworkMonitor] HTTP Port %d in use, using %d instead.", m.Config.NetworkMonitor.HTTPPort, actualHTTPPort)
	}

	// Prepare environment variables
	env := []string{
		fmt.Sprintf("HTTP_API_PORT=%d", actualHTTPPort),
		fmt.Sprintf("NODE_ENV=%s", "production"),
	}

	// Prepare the command
	cmd := exec.Command("node", scriptPath)
	cmd.Env = append(os.Environ(), env...)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stdout for network monitor service: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stderr for network monitor service: %v", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start network monitor service: %v", err)
	}

	log.Printf("Started network monitor service (PID: %d)", cmd.Process.Pid)

	// Create a process object
	process := &NodeJSProcess{
		Name:       "network-monitor",
		Cmd:        cmd,
		StdoutPipe: bufio.NewScanner(stdout),
		StderrPipe: bufio.NewScanner(stderr),
		StopChan:   make(chan struct{}),
	}

	// Handle stdout in a goroutine
	go func() {
		for process.StdoutPipe.Scan() {
			log.Printf("[NetworkMonitor] %s", process.StdoutPipe.Text())
		}
	}()

	// Handle stderr in a goroutine
	go func() {
		for process.StderrPipe.Scan() {
			log.Printf("[NetworkMonitor ERROR] %s", process.StderrPipe.Text())
		}
	}()

	// Store the process
	m.Processes["network-monitor"] = process

	return nil
}

// StartAllServices starts all enabled Node.js services
func (m *NodeJSManager) StartAllServices() error {
	if !m.Config.Enabled {
		return fmt.Errorf("node.js services are not enabled in configuration")
	}

	var errors []error

	if m.Config.TunnelRegistry.Enabled {
		if err := m.StartTunnelRegistry(); err != nil {
			errors = append(errors, fmt.Errorf("failed to start tunnel registry service: %w", err))
		}
	}

	if m.Config.PaymentGateway.Enabled {
		if err := m.StartPaymentGateway(); err != nil {
			errors = append(errors, fmt.Errorf("failed to start payment gateway service: %w", err))
		}
	}

	if m.Config.BootnodeRegistry.Enabled {
		if err := m.StartBootnodeRegistry(); err != nil {
			errors = append(errors, fmt.Errorf("failed to start bootnode registry service: %w", err))
		}
	}

	if m.Config.NotarySystem.Enabled {
		if err := m.StartNotarySystem(); err != nil {
			errors = append(errors, fmt.Errorf("failed to start notary system service: %w", err))
		}
	}

	if m.Config.NetworkMonitor.Enabled {
		if err := m.StartNetworkMonitor(); err != nil {
			errors = append(errors, fmt.Errorf("failed to start network monitor service: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to start some Node.js services: %v", errors)
	}

	return nil
}

// StopService stops a specific Node.js service
func (m *NodeJSManager) StopService(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	process, exists := m.Processes[name]
	if !exists {
		return fmt.Errorf("service %s is not running", name)
	}

	log.Printf("Stopping %s service...", name)

	// Signal to stop the goroutines
	close(process.StopChan)

	// Send SIGTERM to the process
	if err := process.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("Failed to send SIGTERM to %s service: %v", name, err)
	}

	// Wait for the process to exit gracefully
	done := make(chan error, 1)
	go func() {
		done <- process.Cmd.Wait()
	}()

	// Wait for the process to exit or timeout
	select {
	case err := <-done:
		if err != nil {
			log.Printf("%s service exited with error: %v", name, err)
		} else {
			log.Printf("%s service exited gracefully", name)
		}
	case <-time.After(5 * time.Second):
		// Force kill if it doesn't exit within 5 seconds
		log.Printf("%s service did not exit gracefully, forcing kill...", name)
		if err := process.Cmd.Process.Kill(); err != nil {
			log.Printf("Failed to kill %s service: %v", name, err)
		}
	}

	// Remove from the processes map
	delete(m.Processes, name)

	return nil
}

// StopAllServices stops all running Node.js services
func (m *NodeJSManager) StopAllServices() {
	m.mutex.Lock()
	services := make([]string, 0, len(m.Processes))
	for name := range m.Processes {
		services = append(services, name)
	}
	m.mutex.Unlock()

	for _, name := range services {
		if err := m.StopService(name); err != nil {
			log.Printf("Error stopping %s service: %v", name, err)
		}
	}
}

// IsServiceRunning checks if a specific service is running
func (m *NodeJSManager) IsServiceRunning(name string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.Processes[name]
	return exists
}

// GetRunningServices returns a list of running services
func (m *NodeJSManager) GetRunningServices() []string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	services := make([]string, 0, len(m.Processes))
	for name := range m.Processes {
		services = append(services, name)
	}

	return services
}
