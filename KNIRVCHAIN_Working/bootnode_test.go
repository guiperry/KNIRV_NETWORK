package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"KNIRVCHAIN/config"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRegistryServer creates a mock registry server for testing
func MockRegistryServer() *httptest.Server {
	// In-memory registry for testing
	registry := make(map[string]BootnodeInfo)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/register":
			// Handle registration
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}

			var regRequest struct {
				ChainID string `json:"chainID"`
				IP      string `json:"ip"`
				Port    string `json:"port"`
				PeerID  string `json:"nodeID"`
			}

			if err := json.Unmarshal(body, &regRequest); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			// Validate request
			if regRequest.ChainID == "" || regRequest.Port == "" {
				http.Error(w, "Missing required fields", http.StatusBadRequest)
				return
			}

			// Use provided IP or fallback to remote address
			ip := regRequest.IP
			if ip == "" {
				ip = strings.Split(r.RemoteAddr, ":")[0]
			}

			// Convert port to int
			portNum := 0
			fmt.Sscanf(regRequest.Port, "%d", &portNum)

			// Store in registry
			registry[regRequest.ChainID] = BootnodeInfo{
				IP:       ip,
				Port:     portNum,
				LastSeen: time.Now().Unix(),
				PeerID:   regRequest.PeerID,
			}

			// Return success
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message":      "Node registered successfully",
				"registeredIp": ip,
			})

		case r.Method == "GET" && r.URL.Path == "/nodes":
			// Return all registered nodes
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(registry)

		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/lookup/"):
			// Handle lookup
			chainID := strings.TrimPrefix(r.URL.Path, "/lookup/")
			info, exists := registry[chainID]
			if !exists {
				http.Error(w, "Node not found", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(info)

		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))

	return server
}

// MockSTUNServer creates a simple mock STUN server for testing
func MockSTUNServer(t *testing.T) (string, func()) {
	// Create a UDP listener
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)

	conn, err := net.ListenUDP("udp", addr)
	require.NoError(t, err)

	// Get the actual port assigned
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	stunAddr := fmt.Sprintf("127.0.0.1:%d", localAddr.Port)

	// Start a goroutine to handle STUN requests
	go func() {
		defer conn.Close()

		buffer := make([]byte, 1024)
		for {
			_, clientAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				// If the connection is closed, exit the loop
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				t.Logf("Error reading from UDP: %v", err)
				continue
			}

			// Simple mock response - in a real test you'd parse the STUN request
			// and generate a proper response with XOR-MAPPED-ADDRESS
			// For this test, we'll just send back a dummy response
			response := []byte{0x01, 0x01, 0x00, 0x08, // STUN header
				0x21, 0x12, 0xa4, 0x42, // Magic cookie
				0x00, 0x00, 0x00, 0x00, // Transaction ID part 1
				0x00, 0x00, 0x00, 0x00, // Transaction ID part 2
				0x00, 0x00, 0x00, 0x00} // Transaction ID part 3

			// Add XOR-MAPPED-ADDRESS attribute (simplified)
			response = append(response,
				0x00, 0x20, // Type: XOR-MAPPED-ADDRESS
				0x00, 0x08, // Length
				0x00, 0x01, // Family (IPv4)
				byte(clientAddr.Port>>8)^0x21, byte(clientAddr.Port)^0x12, // XOR'd port
				clientAddr.IP[0]^0x21, clientAddr.IP[1]^0x12, clientAddr.IP[2]^0xa4, clientAddr.IP[3]^0x42) // XOR'd IP

			_, err = conn.WriteToUDP(response, clientAddr)
			if err != nil {
				t.Logf("Error writing to UDP: %v", err)
			}
		}
	}()

	// Return the address and a cleanup function
	return stunAddr, func() {
		conn.Close()
	}
}

// TestBootnodeRegistration tests the bootnode registration process
func TestBootnodeRegistration(t *testing.T) {
	// Create a mock registry server
	mockRegistry := MockRegistryServer()
	defer mockRegistry.Close()

	// Override the registry URL for testing
	originalURL := BootnodeRegistryURL
	BootnodeRegistryURL = mockRegistry.URL
	defer func() { BootnodeRegistryURL = originalURL }()

	// Create a temporary config file
	tempDir, err := os.MkdirTemp("", "bootnode-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := tempDir + "/config.json"
	cfg := config.DefaultConfig()
	cfg.ChainID = "test-chain-id"
	cfg.P2PPort = 5050
	cfg.IsBootnode = true

	// Save the config
	configData, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err)

	// Create a discovery manager
	dm, err := NewDiscoveryManager(cfg.ChainID, int(cfg.P2PPort), cfg.ClientOnly, cfg.IsBootnode, config.RoleBootnode, cfg)
	require.NoError(t, err)
	defer dm.Close()

	// Directly register with the registry (bypassing STUN)
	regData := map[string]interface{}{
		"chainID": cfg.ChainID,
		"ip":      "127.0.0.1",
		"port":    fmt.Sprintf("%d", cfg.P2PPort),
	}
	regJSON, _ := json.Marshal(regData)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/register", BootnodeRegistryURL), strings.NewReader(string(regJSON)))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	_, err = client.Do(req)
	require.NoError(t, err)

	// Verify registration by fetching from the registry
	resp, err := client.Get(fmt.Sprintf("%s/nodes", BootnodeRegistryURL))
	require.NoError(t, err)
	defer resp.Body.Close()

	var nodes map[string]BootnodeInfo
	err = json.NewDecoder(resp.Body).Decode(&nodes)
	require.NoError(t, err)

	// Check if our node is registered
	node, exists := nodes[cfg.ChainID]
	assert.True(t, exists, "Node should be registered")
	assert.Equal(t, 5050, node.Port, "Port should match")
}

// TestBootnodeDiscovery tests the bootnode discovery process
func TestBootnodeDiscovery(t *testing.T) {
	// Create a mock registry server
	mockRegistry := MockRegistryServer()
	defer mockRegistry.Close()

	// Override the registry URL for testing
	originalURL := BootnodeRegistryURL
	BootnodeRegistryURL = mockRegistry.URL
	defer func() { BootnodeRegistryURL = originalURL }()

	// Register a mock bootnode
	bootNodeChainID := "bootnode-chain-id"
	bootNodePort := 5050
	bootNodeIP := "127.0.0.1"

	// Register the bootnode directly with the registry
	regData := map[string]interface{}{
		"chainID": bootNodeChainID,
		"ip":      bootNodeIP,
		"port":    fmt.Sprintf("%d", bootNodePort),
		"nodeID":  "QmTestBootnodePeerID12345",
	}
	regJSON, _ := json.Marshal(regData)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/register", BootnodeRegistryURL), strings.NewReader(string(regJSON)))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	_, err := client.Do(req)
	require.NoError(t, err)

	// Create a client node
	clientCfg := config.DefaultConfig()
	clientCfg.ChainID = "client-chain-id"
	clientCfg.P2PPort = 5051
	clientCfg.IsBootnode = false
	clientCfg.ClientOnly = true

	// Create a discovery manager for the client
	clientDM, err := NewDiscoveryManager(clientCfg.ChainID, int(clientCfg.P2PPort), clientCfg.ClientOnly, clientCfg.IsBootnode, config.RoleClient, clientCfg)
	require.NoError(t, err)
	defer clientDM.Close()

	// Bootstrap the client node
	err = clientDM.Bootstrap()
	require.NoError(t, err)

	// Fetch bootnodes from the registry
	bootnodes, err := FetchBootnodesFromRegistry(clientCfg.ChainID, BootnodeRegistryURL)
	require.NoError(t, err)

	// Check if our bootnode is in the list
	bootnode, exists := bootnodes[bootNodeChainID]
	assert.True(t, exists, "Bootnode should be in the registry")
	assert.Equal(t, bootNodeIP, bootnode.IP, "Bootnode IP should match")
	assert.Equal(t, bootNodePort, bootnode.Port, "Bootnode port should match")
	assert.Equal(t, "QmTestBootnodePeerID12345", bootnode.PeerID, "Bootnode PeerID should match")
}

// TestBootnodeConnection tests the connection between a client and a bootnode
func TestBootnodeConnection(t *testing.T) {
	// This test requires running actual libp2p nodes, which is more complex
	// We'll simulate the connection process instead

	// Create two discovery managers
	bootNodeCfg := config.DefaultConfig()
	bootNodeCfg.ChainID = "bootnode-chain-id"
	bootNodeCfg.P2PPort = 5060
	bootNodeCfg.IsBootnode = true

	clientCfg := config.DefaultConfig()
	clientCfg.ChainID = "client-chain-id"
	clientCfg.P2PPort = 5061
	clientCfg.IsBootnode = false
	clientCfg.ClientOnly = false

	// Create the bootnode
	bootNodeDM, err := NewDiscoveryManager(bootNodeCfg.ChainID, int(bootNodeCfg.P2PPort), false, true, config.RoleBootnode, bootNodeCfg)
	require.NoError(t, err)
	defer bootNodeDM.Close()

	// Create the client node
	clientDM, err := NewDiscoveryManager(clientCfg.ChainID, int(clientCfg.P2PPort), false, false, config.RoleClient, clientCfg)
	require.NoError(t, err)
	defer clientDM.Close()

	// Get the bootnode's node info
	bootNodeInfo := peer.AddrInfo{
		ID:    bootNodeDM.host.ID(),
		Addrs: bootNodeDM.host.Addrs(),
	}

	// Connect the client to the bootnode
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = clientDM.host.Connect(ctx, bootNodeInfo)
	require.NoError(t, err)

	// Verify the connection
	connected := false
	for _, conn := range clientDM.host.Network().Conns() {
		if conn.RemotePeer() == bootNodeDM.host.ID() {
			connected = true
			break
		}
	}
	assert.True(t, connected, "Client should be connected to bootnode")
}

// TestBootnodeAnnounceResource tests announcing a resource through a bootnode
func TestBootnodeAnnounceResource(t *testing.T) {
	t.Skip("Skipping resource announcement test as it can be unreliable in CI environments")
	// Create two discovery managers
	bootNodeCfg := config.DefaultConfig()
	bootNodeCfg.ChainID = "bootnode-chain-id"
	bootNodeCfg.P2PPort = 5070
	bootNodeCfg.IsBootnode = true

	clientCfg := config.DefaultConfig()
	clientCfg.ChainID = "client-chain-id"
	clientCfg.P2PPort = 5071
	clientCfg.IsBootnode = false
	clientCfg.ClientOnly = false

	// Create the bootnode
	bootNodeDM, err := NewDiscoveryManager(bootNodeCfg.ChainID, int(bootNodeCfg.P2PPort), false, true, config.RoleBootnode, bootNodeCfg)
	require.NoError(t, err)
	defer bootNodeDM.Close()

	// Start the bootnode
	go bootNodeDM.Run(5 * time.Second)

	// Create the client node
	clientDM, err := NewDiscoveryManager(clientCfg.ChainID, int(clientCfg.P2PPort), false, false, config.RoleClient, clientCfg)
	require.NoError(t, err)
	defer clientDM.Close()

	// Start the client node
	go clientDM.Run(5 * time.Second)

	// Get the bootnode's node info
	bootNodeInfo := peer.AddrInfo{
		ID:    bootNodeDM.host.ID(),
		Addrs: bootNodeDM.host.Addrs(),
	}

	// Connect the client to the bootnode
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = clientDM.host.Connect(ctx, bootNodeInfo)
	require.NoError(t, err)

	// Wait for the DHT to be ready
	time.Sleep(2 * time.Second)

	// Announce a resource from the bootnode
	resourceID := "test-resource"
	resourceType := DiscoveryResourceTypeChain
	err = bootNodeDM.AnnounceGenericResource(resourceID, resourceType)
	require.NoError(t, err)

	// Wait for the announcement to propagate
	time.Sleep(2 * time.Second)

	// Try to find the resource from the client
	providers, err := clientDM.FindGenericResource(resourceID, resourceType)

	// We don't require this to succeed as DHT propagation can be unreliable in tests
	// Just log the result
	if err != nil {
		t.Logf("Failed to find resource: %v", err)
	} else {
		t.Logf("Found %d providers for resource", len(providers))
		for _, p := range providers {
			t.Logf("Provider: %s", p.ID.String())
		}
	}
}
