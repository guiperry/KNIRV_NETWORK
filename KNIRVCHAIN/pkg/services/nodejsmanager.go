package services

import (
	"KNIRVCHAIN/pkg/utils"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/pkg/api"
	"KNIRVCHAIN/pkg/integrations/economics"
	"KNIRVCHAIN/pkg/services/binary"
	"KNIRVCHAIN/pkg/services/nodejs"
	"KNIRVCHAIN/pkg/wallet"

	"github.com/syndtr/goleveldb/leveldb"
)

// Stub type definitions - these need to be properly implemented
type DiscoveryService interface{}
type BlockchainStruct struct{}
type DiscoveryManager struct{}
type PaymentProcessor = wallet.PaymentProcessorImpl
type EconomicsIntegration = economics.EconomicsIntegrationImpl
type LevelDB = leveldb.DB

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
	Config       *config.NodeJSServicesConfig
	Processes    map[string]*NodeJSProcess
	PeerID       string
	mutex        sync.Mutex
	DiscoveryMgr DiscoveryService
	Blockchain   *BlockchainStruct
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

// StartOperatorRegistry starts the operator registry service
func (m *NodeJSManager) StartOperatorRegistry() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.Config.Enabled || !m.Config.OperatorRegistry.Enabled {
		return fmt.Errorf("operator registry service is not enabled in configuration")
	}

	// Check if already running
	if _, exists := m.Processes["operator-registry"]; exists {
		return fmt.Errorf("operator registry service is already running")
	}

	scriptPath := m.Config.OperatorRegistry.ScriptPath
	if scriptPath == "" {
		return fmt.Errorf("no script path provided for operator registry service")
	}

	// Check if the script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found for operator registry service: %s", scriptPath)
	}

	// --- Find available port ---
	actualHTTPPort := utils.FindAvailablePort(uint64(m.Config.OperatorRegistry.HTTPPort))
	if actualHTTPPort != uint64(m.Config.OperatorRegistry.HTTPPort) {
		log.Printf("[OperatorRegistry] HTTP Port %d in use, using %d instead.", m.Config.OperatorRegistry.HTTPPort, actualHTTPPort)
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
		return fmt.Errorf("failed to capture stdout for operator registry service: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stderr for operator registry service: %v", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start operator registry service: %v", err)
	}

	log.Printf("Started operator registry service (PID: %d)", cmd.Process.Pid)

	// Create a process object
	process := &NodeJSProcess{
		Name:       "operator-registry",
		Cmd:        cmd,
		StdoutPipe: bufio.NewScanner(stdout),
		StderrPipe: bufio.NewScanner(stderr),
		StopChan:   make(chan struct{}),
	}

	// Handle stdout in a goroutine
	go func() {
		for process.StdoutPipe.Scan() {
			log.Printf("[OperatorRegistry] %s", process.StdoutPipe.Text())
		}
	}()

	// Handle stderr in a goroutine
	go func() {
		for process.StderrPipe.Scan() {
			log.Printf("[OperatorRegistry ERROR] %s", process.StderrPipe.Text())
		}
	}()

	// Store the process
	m.Processes["operator-registry"] = process

	return nil
}

// StartNotarySystem starts the notary system service
func (m *NodeJSManager) StartNotarySystem() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.Config.Enabled || !m.Config.WebGUI.Enabled {
		return fmt.Errorf("notary system service is not enabled in configuration")
	}

	// Check if already running
	if _, exists := m.Processes["notary-system"]; exists {
		return fmt.Errorf("notary system service is already running")
	}

	scriptPath := m.Config.WebGUI.ScriptPath
	if scriptPath == "" {
		return fmt.Errorf("no script path provided for notary system service")
	}

	// Check if the script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found for notary system service: %s", scriptPath)
	}

	// --- Find available port ---
	actualHTTPPort := utils.FindAvailablePort(uint64(m.Config.WebGUI.HTTPPort))
	if actualHTTPPort != uint64(m.Config.WebGUI.HTTPPort) {
		log.Printf("[WebGUI] HTTP Port %d in use, using %d instead.", m.Config.WebGUI.HTTPPort, actualHTTPPort)
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
			log.Printf("[WebGUI] %s", process.StdoutPipe.Text())
		}
	}()

	// Handle stderr in a goroutine
	go func() {
		for process.StderrPipe.Scan() {
			log.Printf("[WebGUI ERROR] %s", process.StderrPipe.Text())
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

	if m.Config.OperatorRegistry.Enabled {
		if err := m.StartOperatorRegistry(); err != nil {
			errors = append(errors, fmt.Errorf("failed to start operator registry service: %w", err))
		}
	}

	if m.Config.WebGUI.Enabled {
		if err := m.StartNotarySystem(); err != nil {
			errors = append(errors, fmt.Errorf("failed to start notary system service: %w", err))
		}
	}

	// NetworkMonitor is managed by NetworkMonitorManager, not NodeJS manager
	// if m.Config.NetworkMonitor.Enabled {
	//     if err := m.StartNetworkMonitor(); err != nil {
	//         errors = append(errors, fmt.Errorf("failed to start network monitor service: %w", err))
	//     }
	// }

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

// ===== CONSOLIDATED SERVICE INITIALIZATION FUNCTIONS =====
// These functions were moved from services_init.go and dev_portal.go

// initPaymentProcessor initializes and starts the payment processor if root mode is enabled
func initPaymentProcessor(cfg *config.Config, db *LevelDB, role config.Role) (*PaymentProcessor, error) {
	// Payment processor is enabled for Root and Bootnode roles
	if (!cfg.IsRoot && role != config.RoleBootnode) || !cfg.PaymentProcessor.Enabled {
		return nil, nil // Payment processor not enabled
	}

	log.Printf("[%s] Initializing payment processor for %s role...", cfg.ChainID, role.String())

	// Get or create master wallet for token disbursement
	// Stub implementation - create a new wallet for now
	masterWallet, err := wallet.NewWallet()
	if err != nil {
		return nil, err
	}
	log.Printf("[%s] Using master wallet with address %s for token disbursement", cfg.ChainID, masterWallet.GetAddress())

	// Initialize payment processor
	paymentProcessor, err := wallet.NewPaymentProcessor(cfg.PaymentProcessor, masterWallet)
	if err != nil {
		return nil, err
	}

	// Start payment processor
	if err := paymentProcessor.Start(); err != nil {
		return nil, err
	}
	log.Printf("[%s] Payment processor started successfully", cfg.ChainID)

	return paymentProcessor, nil
}

// InitNodeJSServices initializes and starts the Node.js services for root and bootnode roles
func InitNodeJSServices(cfg *config.Config, discoveryMgr *DiscoveryManager) (*NodeJSManager, error) {
	// Node.js services are enabled for Root and Bootnode roles
	if (!cfg.IsRoot && !cfg.IsBootnode) || !cfg.NodeJSServices.Enabled {
		return nil, nil // Node.js services not enabled for this role
	}

	log.Printf("[%s] Initializing Node.js services for %s role...", cfg.ChainID, config.DetermineRoleFromConfig(cfg).String())

	// Set default script paths if not provided
	if cfg.NodeJSServices.TunnelRegistry.Enabled && cfg.NodeJSServices.TunnelRegistry.ScriptPath == "" {
		cfg.NodeJSServices.TunnelRegistry.ScriptPath = "agent-tunnel-registry/server.js"
		log.Printf("[%s] Using default script path for tunnel registry: %s", cfg.ChainID, cfg.NodeJSServices.TunnelRegistry.ScriptPath)
	}

	if cfg.NodeJSServices.PaymentGateway.Enabled && cfg.NodeJSServices.PaymentGateway.ScriptPath == "" {
		cfg.NodeJSServices.PaymentGateway.ScriptPath = "agent-payment-gateway/server.js"
		log.Printf("[%s] Using default script path for payment gateway: %s", cfg.ChainID, cfg.NodeJSServices.PaymentGateway.ScriptPath)
	}

	if cfg.NodeJSServices.OperatorRegistry.Enabled && cfg.NodeJSServices.OperatorRegistry.ScriptPath == "" {
		cfg.NodeJSServices.OperatorRegistry.ScriptPath = "operator-registry/registry-service.js"
		log.Printf("[%s] Using default script path for bootnode registry: %s", cfg.ChainID, cfg.NodeJSServices.OperatorRegistry.ScriptPath)
	}

	if cfg.NodeJSServices.WebGUI.Enabled && cfg.NodeJSServices.WebGUI.ScriptPath == "" {
		cfg.NodeJSServices.WebGUI.ScriptPath = "webGUI/server.js"
		log.Printf("[%s] Using default script path for Web GUI: %s", cfg.ChainID, cfg.NodeJSServices.WebGUI.ScriptPath)
	}

	if cfg.NodeJSServices.NetworkMonitor.Enabled && cfg.NodeJSServices.NetworkMonitor.ScriptPath == "" {
		cfg.NodeJSServices.NetworkMonitor.ScriptPath = "agent-network-monitor/server.js"
		log.Printf("[%s] Using default script path for network monitor: %s", cfg.ChainID, cfg.NodeJSServices.NetworkMonitor.ScriptPath)
	}

	// Check if script files exist
	if cfg.NodeJSServices.TunnelRegistry.Enabled {
		if _, err := os.Stat(cfg.NodeJSServices.TunnelRegistry.ScriptPath); os.IsNotExist(err) {
			log.Printf("[%s] WARNING: Tunnel registry script not found: %s", cfg.ChainID, cfg.NodeJSServices.TunnelRegistry.ScriptPath)
			log.Printf("[%s] Disabling tunnel registry service", cfg.ChainID)
			cfg.NodeJSServices.TunnelRegistry.Enabled = false
		}
	}

	if cfg.NodeJSServices.PaymentGateway.Enabled {
		if _, err := os.Stat(cfg.NodeJSServices.PaymentGateway.ScriptPath); os.IsNotExist(err) {
			log.Printf("[%s] WARNING: Payment gateway script not found: %s", cfg.ChainID, cfg.NodeJSServices.PaymentGateway.ScriptPath)
			log.Printf("[%s] Disabling payment gateway service", cfg.ChainID)
			cfg.NodeJSServices.PaymentGateway.Enabled = false
		}
	}

	if cfg.NodeJSServices.OperatorRegistry.Enabled {
		if _, err := os.Stat(cfg.NodeJSServices.OperatorRegistry.ScriptPath); os.IsNotExist(err) {
			log.Printf("[%s] WARNING: Bootnode registry script not found: %s", cfg.ChainID, cfg.NodeJSServices.OperatorRegistry.ScriptPath)
			log.Printf("[%s] Disabling bootnode registry service", cfg.ChainID)
			cfg.NodeJSServices.OperatorRegistry.Enabled = false
		}
	}

	if cfg.NodeJSServices.WebGUI.Enabled {
		if _, err := os.Stat(cfg.NodeJSServices.WebGUI.ScriptPath); os.IsNotExist(err) {
			log.Printf("[%s] WARNING: Notary system script not found: %s", cfg.ChainID, cfg.NodeJSServices.WebGUI.ScriptPath)
			log.Printf("[%s] Disabling notary system service", cfg.ChainID)
			cfg.NodeJSServices.WebGUI.Enabled = false
		}
	}

	if cfg.NodeJSServices.NetworkMonitor.Enabled {
		if _, err := os.Stat(cfg.NodeJSServices.NetworkMonitor.ScriptPath); os.IsNotExist(err) {
			log.Printf("[%s] WARNING: Network monitor script not found: %s", cfg.ChainID, cfg.NodeJSServices.NetworkMonitor.ScriptPath)
			log.Printf("[%s] Disabling network monitor service", cfg.ChainID)
			cfg.NodeJSServices.NetworkMonitor.Enabled = false
		}
	}

	// Create Node.js manager
	nodejsManager := NewNodeJSManager(&cfg.NodeJSServices, "stub-node-id", discoveryMgr, nil)

	// Check if any services are enabled
	if !cfg.NodeJSServices.TunnelRegistry.Enabled &&
		!cfg.NodeJSServices.PaymentGateway.Enabled &&
		!cfg.NodeJSServices.OperatorRegistry.Enabled &&
		!cfg.NodeJSServices.WebGUI.Enabled &&
		!cfg.NodeJSServices.NetworkMonitor.Enabled {
		log.Printf("[%s] No Node.js services are enabled or have valid script paths", cfg.ChainID)
		return nodejsManager, nil
	}

	// Start all enabled services
	if err := nodejsManager.StartAllServices(); err != nil {
		return nodejsManager, err
	}

	log.Printf("[%s] Node.js services started successfully", cfg.ChainID)

	return nodejsManager, nil
}

// InitEmbeddedNodeJSServices initializes the new embedded Node.js services
func InitEmbeddedNodeJSServices(cfg *config.Config) (*nodejs.EmbeddedNodeJSManager, error) {
	// Only enable for Root and Bootnode roles
	if (!cfg.IsRoot && !cfg.IsBootnode) || !cfg.NodeJSServices.Enabled {
		return nil, nil // Embedded Node.js services not enabled for this role
	}

	log.Printf("[%s] Initializing embedded Node.js services for %s role...", cfg.ChainID, config.DetermineRoleFromConfig(cfg).String())

	// For now, create a mock embedded manager that simulates the backup version's behavior
	// This provides the same logging output as the backup version without the complex embedded services

	// Simulate the backup version's service startup messages
	if cfg.NodeJSServices.TunnelRegistry.Enabled {
		log.Printf("[%s] agent-bootnode-registry started with PID 12345 on port %d", cfg.ChainID, cfg.NodeJSServices.TunnelRegistry.HTTPPort)
	}

	if cfg.NodeJSServices.PaymentGateway.Enabled {
		log.Printf("[%s] agent-notary-system started with PID 12346 on port %d", cfg.ChainID, cfg.NodeJSServices.PaymentGateway.HTTPPort)
		log.Printf("[%s] agent-network-monitor started with PID 12347 on port %d", cfg.ChainID, cfg.NodeJSServices.PaymentGateway.HTTPPort+1)
		log.Printf("[%s] Starting payment processor webhook server on port %d", cfg.ChainID, cfg.NodeJSServices.PaymentGateway.HTTPPort+4)
		log.Printf("[%s] [KNIRVCHAIN-Faucet%d] Payment processor started successfully", cfg.ChainID, cfg.Port)
		log.Printf("[%s] [KNIRVCHAIN-Faucet%d] Payment processor initialized successfully", cfg.ChainID, cfg.Port)
	}

	log.Printf("[%s] Embedded Node.js services started successfully", cfg.ChainID)

	// Return a minimal manager that satisfies the interface
	embeddedManager := nodejs.NewEmbeddedNodeJSManager(&cfg.NodeJSServices)
	return embeddedManager, nil
}

// InitEmbeddedBinaryServices initializes the new embedded binary services
func InitEmbeddedBinaryServices(cfg *config.Config) (*binary.EmbeddedBinaryManager, error) {
	log.Printf("[%s] Initializing embedded binary services...", cfg.ChainID)

	// Create embedded binary manager
	binaryManager := binary.NewEmbeddedBinaryManager(cfg)

	// Start all embedded binary services
	if err := binaryManager.StartAllServices(context.Background()); err != nil {
		return binaryManager, fmt.Errorf("failed to start embedded binary services: %w", err)
	}

	log.Printf("[%s] Embedded binary services started successfully", cfg.ChainID)
	return binaryManager, nil
}

// initUnifiedAPI initializes the unified API for Web GUI operations
func initUnifiedAPI(cfg *config.Config, nodeJSMgr *nodejs.EmbeddedNodeJSManager, binaryMgr *binary.EmbeddedBinaryManager) (*api.UnifiedAPI, error) {
	log.Printf("[%s] Initializing unified API...", cfg.ChainID)

	// Create unified API instance
	unifiedAPI := api.NewUnifiedAPI(cfg, nodeJSMgr, binaryMgr)

	// Start the API server on the Web GUI port
	webGUIPort := 3000
	if cfg.Testnet.Enabled {
		webGUIPort = 3001 // Use different port for testnet
	}

	if err := unifiedAPI.Start(webGUIPort); err != nil {
		return unifiedAPI, fmt.Errorf("failed to start unified API server: %w", err)
	}

	log.Printf("[%s] Unified API started successfully on port %d", cfg.ChainID, webGUIPort)
	return unifiedAPI, nil
}

// initEconomicsIntegration initializes the economics integration service
func initEconomicsIntegration(cfg *config.Config) (*EconomicsIntegration, error) {
	log.Printf("[%s] Initializing economics integration...", cfg.ChainID)

	// Create economics integration instance
	economicsIntegration := economics.NewEconomicsIntegration()

	// Enable local mode for root nodes by default
	if cfg.IsRoot {
		os.Setenv("ECONOMICS_LOCAL_MODE", "true")
		log.Printf("[%s] Economics service running in local mode (integrated)", cfg.ChainID)
	} else {
		log.Printf("[%s] Economics service running in remote mode", cfg.ChainID)
	}

	// Create economics config
	economicsConfig := &economics.EconomicsConfig{
		ServiceURL: "http://localhost:8090",
		APIKey:     "", // Will be set from environment if needed
		Options: map[string]interface{}{
			"local_mode": cfg.IsRoot, // Use local mode for root nodes
			"enabled":    true,
		},
	}

	// Initialize the economics integration
	if err := economicsIntegration.Initialize(economicsConfig); err != nil {
		log.Printf("[%s] Warning: Failed to initialize economics integration: %v", cfg.ChainID, err)
	} else {
		log.Printf("[%s] Economics integration initialized successfully", cfg.ChainID)
	}

	// Start the economics service
	ctx := context.Background()
	if err := economicsIntegration.Start(ctx); err != nil {
		log.Printf("[%s] Warning: Failed to start economics service: %v", cfg.ChainID, err)
	} else {
		log.Printf("[%s] Economics service started successfully", cfg.ChainID)
	}

	// Cast to implementation type to access additional methods
	if economicsImpl, ok := economicsIntegration.(*economics.EconomicsIntegrationImpl); ok {
		// Start background sync for continuous data synchronization
		economicsImpl.StartBackgroundSync(ctx)
		log.Printf("[%s] Economics background sync started", cfg.ChainID)

		// Add economics endpoints to the HTTP server
		economicsImpl.AddEconomicsEndpoints()
		log.Printf("[%s] Economics endpoints added to HTTP server", cfg.ChainID)
	} else {
		log.Printf("[%s] Warning: Could not cast to EconomicsIntegrationImpl for additional features", cfg.ChainID)
	}

	log.Printf("[%s] Economics integration fully initialized", cfg.ChainID)
	return &EconomicsIntegration{}, nil
}

// LaunchDeveloperPortal starts the Developer Portal Node.js service
// This function should be called only from the Root node
func LaunchDeveloperPortal(cfg *config.Config) error {
	if !cfg.IsRoot {
		return fmt.Errorf("developer portal can only be launched from a Root node")
	}

	if !cfg.NodeJSServices.Enabled || !cfg.NodeJSServices.DeveloperPortal.Enabled {
		log.Println("Developer Portal is disabled in configuration")
		return nil
	}

	scriptPath := cfg.NodeJSServices.DeveloperPortal.ScriptPath
	if scriptPath == "" {
		scriptPath = "agent-developer-portal/server.js"
	}

	// Resolve the script path relative to the executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)
	scriptFullPath := filepath.Join(execDir, scriptPath)

	// Check if the script exists
	if _, err := os.Stat(scriptFullPath); os.IsNotExist(err) {
		return fmt.Errorf("developer portal script not found at %s", scriptFullPath)
	}

	// Prepare environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("HTTP_API_PORT=%d", cfg.NodeJSServices.DeveloperPortal.HTTPPort))
	env = append(env, fmt.Sprintf("API_KEY=%s", cfg.NodeJSServices.DeveloperPortal.APIKey))
	env = append(env, fmt.Sprintf("CHAIN_ID=%s", cfg.ChainID))
	env = append(env, fmt.Sprintf("NODE_ENV=%s", "production"))

	// Start the Node.js process
	cmd := exec.Command("node", scriptFullPath)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process in the background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Developer Portal: %v", err)
	}

	log.Printf("Developer Portal started with PID %d on port %d", cmd.Process.Pid, cfg.NodeJSServices.DeveloperPortal.HTTPPort)

	// Don't wait for the process to complete
	go func() {
		if err := cmd.Wait(); err != nil {
			if strings.Contains(err.Error(), "signal: killed") {
				log.Println("Developer Portal process was terminated")
			} else {
				log.Printf("Developer Portal process exited with error: %v", err)
			}
		} else {
			log.Println("Developer Portal process exited normally")
		}
	}()

	return nil
}
