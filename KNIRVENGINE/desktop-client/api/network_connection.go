package api

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// NetworkConnection implements TargetSystemConnection for network monitoring
type NetworkConnection struct {
	target    *TargetSystem
	connected bool
	startTime time.Time
	mutex     sync.RWMutex
}

// NewNetworkConnection creates a new network connection
func NewNetworkConnection(target *TargetSystem) (TargetSystemConnection, error) {
	return &NetworkConnection{
		target: target,
	}, nil
}

// Connect establishes the network connection
func (c *NetworkConnection) Connect(ctx context.Context) error {
	_ = ctx // Context not used in current implementation
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected {
		return nil
	}

	// Test basic network connectivity
	if _, err := net.LookupHost("google.com"); err != nil {
		return fmt.Errorf("network connectivity test failed: %v", err)
	}

	c.connected = true
	c.startTime = time.Now()
	return nil
}

// Disconnect closes the network connection
func (c *NetworkConnection) Disconnect(ctx context.Context) error {
	_ = ctx // Context not used in current implementation
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.connected = false
	return nil
}

// IsConnected returns the connection status
func (c *NetworkConnection) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// GetCapabilities returns available network capabilities
func (c *NetworkConnection) GetCapabilities() []string {
	return []string{
		"ping",
		"port_scan",
		"dns_lookup",
		"trace_route",
		"network_info",
		"bandwidth_test",
	}
}

// Execute executes a network operation
func (c *NetworkConnection) Execute(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("network not connected")
	}

	switch operation {
	case "ping":
		return c.ping(ctx, params)
	case "dns_lookup":
		return c.dnsLookup(ctx, params)
	case "network_info":
		return c.getNetworkInfo(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// GetStatus returns detailed network status
func (c *NetworkConnection) GetStatus() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"connected": c.connected,
		"type":      "network",
		"uptime":    time.Since(c.startTime).String(),
	}
}

// GetType returns the target system type
func (c *NetworkConnection) GetType() TargetSystemType {
	return TargetTypeNetwork
}

// Network operations (simplified implementations)

func (c *NetworkConnection) ping(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	host := getStringParam(params, "host", "")
	if host == "" {
		return nil, fmt.Errorf("host parameter is required")
	}

	start := time.Now()
	_, err := net.LookupHost(host)
	duration := time.Since(start)

	return map[string]interface{}{
		"success":  err == nil,
		"host":     host,
		"duration": duration.String(),
		"error":    fmt.Sprintf("%v", err),
	}, nil
}

func (c *NetworkConnection) dnsLookup(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	host := getStringParam(params, "host", "")
	if host == "" {
		return nil, fmt.Errorf("host parameter is required")
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed: %v", err)
	}

	var ipStrings []string
	for _, ip := range ips {
		ipStrings = append(ipStrings, ip.String())
	}

	return map[string]interface{}{
		"success": true,
		"host":    host,
		"ips":     ipStrings,
		"count":   len(ipStrings),
	}, nil
}

func (c *NetworkConnection) getNetworkInfo(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx    // Context not used in current implementation
	_ = params // Parameters not used in current implementation
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %v", err)
	}

	var interfaceInfo []map[string]interface{}
	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()
		var addrStrings []string
		for _, addr := range addrs {
			addrStrings = append(addrStrings, addr.String())
		}

		interfaceInfo = append(interfaceInfo, map[string]interface{}{
			"name":      iface.Name,
			"addresses": addrStrings,
			"flags":     iface.Flags.String(),
		})
	}

	return map[string]interface{}{
		"success":    true,
		"interfaces": interfaceInfo,
		"count":      len(interfaceInfo),
	}, nil
}
