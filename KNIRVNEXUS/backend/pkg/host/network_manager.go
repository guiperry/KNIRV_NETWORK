package host

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NetworkManager manages and monitors network configuration
type NetworkManager struct {
	ctx    context.Context
	config *HostConfig
	mu     sync.RWMutex

	interfaces    []NetworkInterface
	firewallRules []FirewallRule
	p2pConfig     *P2PNetworkConfig
	lastUpdate    time.Time
	running       bool
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name        string   `json:"name"`
	IPAddresses []string `json:"ip_addresses"`
	MACAddress  string   `json:"mac_address"`
	Status      string   `json:"status"`
	Speed       string   `json:"speed"`
	MTU         int      `json:"mtu"`
	Type        string   `json:"type"` // ethernet, wifi, loopback, etc.

	// Statistics
	RxBytes   uint64 `json:"rx_bytes"`
	TxBytes   uint64 `json:"tx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	TxPackets uint64 `json:"tx_packets"`
	RxErrors  uint64 `json:"rx_errors"`
	TxErrors  uint64 `json:"tx_errors"`

	// KNIRV-specific
	IsKNIRVInterface bool   `json:"is_knirv_interface"`
	P2PEnabled       bool   `json:"p2p_enabled"`
	TunnelType       string `json:"tunnel_type,omitempty"`
}

// FirewallRule represents a firewall rule
type FirewallRule struct {
	ID          string `json:"id"`
	Chain       string `json:"chain"`       // INPUT, OUTPUT, FORWARD
	Target      string `json:"target"`      // ACCEPT, DROP, REJECT
	Protocol    string `json:"protocol"`    // tcp, udp, icmp, all
	Source      string `json:"source"`      // IP/CIDR
	Destination string `json:"destination"` // IP/CIDR
	Port        string `json:"port"`        // port or port range
	Interface   string `json:"interface"`   // network interface
	State       string `json:"state"`       // NEW, ESTABLISHED, RELATED
	Comment     string `json:"comment"`

	// KNIRV-specific
	IsKNIRVRule bool   `json:"is_knirv_rule"`
	ServiceType string `json:"service_type,omitempty"`
}

// NetworkInfo contains comprehensive network information
type NetworkInfo struct {
	Interfaces    []NetworkInterface `json:"interfaces"`
	FirewallRules []FirewallRule     `json:"firewall_rules"`
	P2PConfig     *P2PNetworkConfig  `json:"p2p_config"`
	LastUpdate    time.Time          `json:"last_update"`
}

// P2PNetworkConfig contains P2P network configuration
type P2PNetworkConfig struct {
	ListenAddresses    []string       `json:"listen_addresses"`
	BootstrapPeers     []string       `json:"bootstrap_peers"`
	NetworkID          string         `json:"network_id"`
	ProtocolVersion    string         `json:"protocol_version"`
	MaxPeers           int            `json:"max_peers"`
	EnableNATTraversal bool           `json:"enable_nat_traversal"`
	EnableRelay        bool           `json:"enable_relay"`
	CustomPorts        map[string]int `json:"custom_ports"`

	// Security settings
	EnableEncryption bool     `json:"enable_encryption"`
	AllowedPeers     []string `json:"allowed_peers"`
	BlockedPeers     []string `json:"blocked_peers"`
}

// NewNetworkManager creates a new network manager
func NewNetworkManager(ctx context.Context, config *HostConfig) (*NetworkManager, error) {
	nm := &NetworkManager{
		ctx:    ctx,
		config: config,
		p2pConfig: &P2PNetworkConfig{
			ListenAddresses:    []string{"/ip4/0.0.0.0/tcp/4001", "/ip6/::/tcp/4001"},
			NetworkID:          "knirv-nexus",
			ProtocolVersion:    "1.0.0",
			MaxPeers:           50,
			EnableNATTraversal: true,
			EnableRelay:        true,
			EnableEncryption:   true,
			CustomPorts: map[string]int{
				"dve-manager":     8080,
				"validation-core": 8081,
				"model-server":    8082,
				"data-engine":     8083,
				"inference":       8084,
			},
		},
	}

	return nm, nil
}

// Start begins network monitoring
func (nm *NetworkManager) Start() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if nm.running {
		return fmt.Errorf("network manager is already running")
	}

	nm.running = true

	// Initial network scan
	if err := nm.scanNetworkInterfaces(); err != nil {
		return fmt.Errorf("initial network scan failed: %w", err)
	}

	if err := nm.scanFirewallRules(); err != nil {
		return fmt.Errorf("initial firewall scan failed: %w", err)
	}

	// Setup KNIRV-specific firewall rules
	if err := nm.setupKNIRVFirewallRules(); err != nil {
		return fmt.Errorf("failed to setup KNIRV firewall rules: %w", err)
	}

	// Start monitoring loop
	go nm.monitorLoop()

	return nil
}

// Stop stops network monitoring
func (nm *NetworkManager) Stop() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.running = false
	return nil
}

// GetNetworkInfo returns current network information
func (nm *NetworkManager) GetNetworkInfo() (*NetworkInfo, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	// Create a comprehensive network info structure
	networkInfo := &NetworkInfo{
		Interfaces:    make([]NetworkInterface, len(nm.interfaces)),
		FirewallRules: make([]FirewallRule, len(nm.firewallRules)),
		P2PConfig:     nm.p2pConfig,
		LastUpdate:    nm.lastUpdate,
	}

	// Copy interfaces
	copy(networkInfo.Interfaces, nm.interfaces)

	// Copy firewall rules
	copy(networkInfo.FirewallRules, nm.firewallRules)

	return networkInfo, nil
}

// HealthCheck verifies the network manager is working properly
func (nm *NetworkManager) HealthCheck() error {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if !nm.running {
		return fmt.Errorf("network manager is not running")
	}

	// Check if data is stale
	if time.Since(nm.lastUpdate) > nm.config.NetworkInterval*2 {
		return fmt.Errorf("network data is stale (last update: %v)", nm.lastUpdate)
	}

	// Check if we have at least one active interface
	hasActiveInterface := false
	for _, iface := range nm.interfaces {
		if iface.Status == "up" && iface.Name != "lo" {
			hasActiveInterface = true
			break
		}
	}

	if !hasActiveInterface {
		return fmt.Errorf("no active network interfaces found")
	}

	return nil
}

// monitorLoop runs the periodic monitoring loop
func (nm *NetworkManager) monitorLoop() {
	ticker := time.NewTicker(nm.config.NetworkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			nm.mu.RLock()
			running := nm.running
			nm.mu.RUnlock()

			if !running {
				return
			}

			if err := nm.scanNetworkInterfaces(); err != nil {
				fmt.Printf("Error scanning network interfaces: %v\n", err)
			}

			if err := nm.scanFirewallRules(); err != nil {
				fmt.Printf("Error scanning firewall rules: %v\n", err)
			}
		}
	}
}

// scanNetworkInterfaces scans and updates network interface information
func (nm *NetworkManager) scanNetworkInterfaces() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var networkInterfaces []NetworkInterface

	for _, iface := range interfaces {
		netIface := NetworkInterface{
			Name:       iface.Name,
			MACAddress: iface.HardwareAddr.String(),
			MTU:        iface.MTU,
		}

		// Get interface status
		if iface.Flags&net.FlagUp != 0 {
			netIface.Status = "up"
		} else {
			netIface.Status = "down"
		}

		// Determine interface type
		netIface.Type = nm.determineInterfaceType(iface.Name)

		// Get IP addresses
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				netIface.IPAddresses = append(netIface.IPAddresses, addr.String())
			}
		}

		// Get interface statistics
		nm.getInterfaceStatistics(&netIface)

		// Check if this is a KNIRV-related interface
		nm.identifyKNIRVInterface(&netIface)

		networkInterfaces = append(networkInterfaces, netIface)
	}

	nm.mu.Lock()
	nm.interfaces = networkInterfaces
	nm.lastUpdate = time.Now()
	nm.mu.Unlock()

	return nil
}

// determineInterfaceType determines the type of network interface
func (nm *NetworkManager) determineInterfaceType(name string) string {
	if name == "lo" {
		return "loopback"
	} else if strings.HasPrefix(name, "eth") {
		return "ethernet"
	} else if strings.HasPrefix(name, "wlan") || strings.HasPrefix(name, "wifi") {
		return "wifi"
	} else if strings.HasPrefix(name, "docker") {
		return "docker"
	} else if strings.HasPrefix(name, "br-") {
		return "bridge"
	} else if strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "tap") {
		return "tunnel"
	} else if strings.HasPrefix(name, "veth") {
		return "virtual"
	}
	return "unknown"
}

// getInterfaceStatistics gets network interface statistics
func (nm *NetworkManager) getInterfaceStatistics(iface *NetworkInterface) {
	// Read from /proc/net/dev
	cmd := exec.Command("cat", "/proc/net/dev")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, iface.Name+":") {
			fields := strings.Fields(line)
			if len(fields) >= 17 {
				// Parse statistics (simplified)
				if rxBytes, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					iface.RxBytes = rxBytes
				}
				if rxPackets, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
					iface.RxPackets = rxPackets
				}
				if rxErrors, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
					iface.RxErrors = rxErrors
				}
				if txBytes, err := strconv.ParseUint(fields[9], 10, 64); err == nil {
					iface.TxBytes = txBytes
				}
				if txPackets, err := strconv.ParseUint(fields[10], 10, 64); err == nil {
					iface.TxPackets = txPackets
				}
				if txErrors, err := strconv.ParseUint(fields[11], 10, 64); err == nil {
					iface.TxErrors = txErrors
				}
			}
			break
		}
	}
}

// identifyKNIRVInterface identifies if an interface is KNIRV-related
func (nm *NetworkManager) identifyKNIRVInterface(iface *NetworkInterface) {
	// Check for KNIRV-specific interface patterns
	knirvPatterns := []string{"knirv", "nexus", "p2p", "dve"}

	for _, pattern := range knirvPatterns {
		if strings.Contains(strings.ToLower(iface.Name), pattern) {
			iface.IsKNIRVInterface = true
			iface.P2PEnabled = true
			break
		}
	}

	// Check for tunnel interfaces that might be used by KNIRV
	if iface.Type == "tunnel" {
		iface.TunnelType = "unknown"
		if strings.Contains(iface.Name, "wireguard") {
			iface.TunnelType = "wireguard"
		} else if strings.Contains(iface.Name, "openvpn") {
			iface.TunnelType = "openvpn"
		}
	}
}

// scanFirewallRules scans current firewall rules
func (nm *NetworkManager) scanFirewallRules() error {
	// Use iptables to get current rules
	cmd := exec.Command("iptables", "-L", "-n", "--line-numbers")
	output, err := cmd.Output()
	if err != nil {
		// Firewall might not be available or accessible
		return nil
	}

	var firewallRules []FirewallRule

	lines := strings.Split(string(output), "\n")
	currentChain := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for chain headers
		if strings.HasPrefix(line, "Chain ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentChain = parts[1]
			}
			continue
		}

		// Skip table headers
		if strings.HasPrefix(line, "num ") {
			continue
		}

		// Parse rule line
		rule := nm.parseFirewallRule(line, currentChain)
		if rule != nil {
			firewallRules = append(firewallRules, *rule)
		}
	}

	nm.mu.Lock()
	nm.firewallRules = firewallRules
	nm.mu.Unlock()

	return nil
}

// parseFirewallRule parses a firewall rule line
func (nm *NetworkManager) parseFirewallRule(line, chain string) *FirewallRule {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil
	}

	rule := &FirewallRule{
		ID:     fmt.Sprintf("%s-%s", chain, fields[0]),
		Chain:  chain,
		Target: fields[1],
	}

	if len(fields) > 2 {
		rule.Protocol = fields[2]
	}

	// Check if this is a KNIRV-related rule
	nm.identifyKNIRVFirewallRule(rule, line)

	return rule
}

// identifyKNIRVFirewallRule identifies if a firewall rule is KNIRV-related
func (nm *NetworkManager) identifyKNIRVFirewallRule(rule *FirewallRule, line string) {
	knirvPorts := []string{"8080", "8081", "8082", "8083", "8084", "4001"}
	knirvKeywords := []string{"knirv", "nexus", "dve", "p2p"}

	lineLower := strings.ToLower(line)

	// Check for KNIRV ports
	for _, port := range knirvPorts {
		if strings.Contains(line, port) {
			rule.IsKNIRVRule = true
			rule.Port = port
			break
		}
	}

	// Check for KNIRV keywords
	for _, keyword := range knirvKeywords {
		if strings.Contains(lineLower, keyword) {
			rule.IsKNIRVRule = true
			rule.Comment = keyword
			break
		}
	}
}

// setupKNIRVFirewallRules sets up necessary firewall rules for KNIRV services
func (nm *NetworkManager) setupKNIRVFirewallRules() error {
	// This would implement firewall rule setup
	// For now, just log that we would set up rules
	fmt.Println("Setting up KNIRV firewall rules...")

	// In a real implementation, this would:
	// 1. Check if required rules exist
	// 2. Add missing rules for KNIRV services
	// 3. Configure P2P networking rules
	// 4. Set up security policies

	return nil
}
