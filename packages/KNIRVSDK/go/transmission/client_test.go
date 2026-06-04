package main

import (
	//"context"
	"fmt"
	"testing"

	"strings"
	//"time"

	//"github.com/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP/sdk/go/parser"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

type mockProviderFinder struct {
	findFunc func(id, resourceType string) ([]peer.AddrInfo, error)
}

func (m *mockProviderFinder) FindResourceProviders(id, resourceType string) ([]peer.AddrInfo, error) {
	return m.findFunc(id, resourceType)
}

// setupTestServer creates a test server that responds to resource fetch requests
func setupTestServer(t *testing.T) (host.Host, string) {
	// Create a libp2p host for testing
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Set up a handler for the resource fetch protocol
	h.SetStreamHandler(protocol.ID("/knirv/resource-fetch/1.0.0"), func(s network.Stream) {
		defer s.Close()

		// Read the request
		buf := make([]byte, 2048) // Increased buffer size for potentially longer requests
		n, err := s.Read(buf)
		if err != nil {
			t.Logf("Error reading request: %v", err)
			// Optionally write an error to the stream if the client expects it
			// _, _ = s.Write([]byte("ERROR: Could not read request"))
			return
		}

		// Client sends requests like "GET /path?query\n"
		// We trim space to remove the trailing newline for easier matching.
		request := strings.TrimSpace(string(buf[:n]))
		var response string
		sendError := false
		errorMessage := "ERROR: Resource not found by test server" // Default error

		switch request {
		case "GET /block?number=123":
			response = `{"number": 123, "hash": "0x123abc", "data": "test block data"}`
		case "GET /":
			response = `{"status": "ok", "message": "Root resource"}`
		case "GET /nonexistent":
			sendError = true
			errorMessage = "ERROR: Resource not found" // Specific error for this case, matching client expectation
		default:
			sendError = true
			errorMessage = fmt.Sprintf("ERROR: Unknown request to test server: %s", request)
		}

		if sendError {
			_, err = s.Write([]byte(errorMessage))
		} else {
			_, err = s.Write([]byte(response))
		}
		if err != nil {
			t.Logf("Error writing response to stream: %v", err)
		}
	})

	// Get the server's multiaddr
	var serverAddr string
	for _, addr := range h.Addrs() {
		serverAddr = fmt.Sprintf("%s/p2p/%s", addr.String(), h.ID().String())
		break
	}

	return h, serverAddr
}

func TestClient_FetchResource(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Set up a test server
	server, serverAddr := setupTestServer(t)
	defer server.Close()

	// Create a client with the test server as bootstrap peer
	config := ClientConfig{
		BootstrapPeers: []string{serverAddr},
		P2PPort:        0, // Use random port
		LogEnabled:     true,
	}

	client, err := New(config, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Bootstrap the client
	err = client.Bootstrap()
	if err != nil {
		t.Fatalf("Failed to bootstrap client: %v", err)
	}

	// Create a mock ProviderFinder
	mockFinder := &mockProviderFinder{
		findFunc: func(id, resourceType string) ([]peer.AddrInfo, error) {
			ma, _ := multiaddr.NewMultiaddr(serverAddr)
			ai, _ := peer.AddrInfoFromP2pAddr(ma)
			return []peer.AddrInfo{*ai}, nil
		},
	}
	client.providerFinder = mockFinder

	// Test fetching a resource
	t.Run("Fetch block resource", func(t *testing.T) {
		resource, err := client.FetchResource("knirv://testchain.chain/block?number=123")
		if err != nil {
			t.Fatalf("Failed to fetch resource: %v", err)
		}

		expected := `{"number": 123, "hash": "0x123abc", "data": "test block data"}`
		if resource.String() != expected {
			t.Errorf("Expected %q, got %q", expected, resource.String())
		}
	})

	t.Run("Fetch root resource", func(t *testing.T) {
		resource, err := client.FetchResource("knirv://testchain.chain/")
		if err != nil {
			t.Fatalf("Failed to fetch resource: %v", err)
		}

		expected := `{"status": "ok", "message": "Root resource"}`
		if resource.String() != expected {
			t.Errorf("Expected %q, got %q", expected, resource.String())
		}
	})

	t.Run("Fetch nonexistent resource", func(t *testing.T) {
		_, err := client.FetchResource("knirv://testchain.chain/nonexistent")
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
		// FetchResource returns ErrResourceNotFound when all providers fail,
		// regardless of the specific error from the provider (which is logged).
		expectedErrorString := ErrResourceNotFound.Error()
		if err.Error() != expectedErrorString {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestClient_Close(t *testing.T) {
	config := DefaultConfig()
	client, err := New(config, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Close the client
	err = client.Close()
	if err != nil {
		t.Fatalf("Failed to close client: %v", err)
	}

	// Verify that operations fail after closing
	_, err = client.FetchResource("knirv://testchain.chain/")
	if err != ErrClientClosed {
		t.Errorf("Expected ErrClientClosed, got %v", err)
	}

	// Verify that closing again doesn't cause issues
	err = client.Close()
	if err != nil {
		t.Errorf("Second close failed: %v", err)
	}
}

func TestClient_GetPeerID(t *testing.T) {
	config := DefaultConfig()
	client, err := New(config, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	peerID := client.GetPeerID()
	if peerID == "" {
		t.Error("Expected non-empty peer ID")
	}
}

func TestClient_GetMultiaddrs(t *testing.T) {
	config := DefaultConfig()
	client, err := New(config, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	addrs := client.GetMultiaddrs()
	if len(addrs) == 0 {
		t.Error("Expected at least one multiaddr")
	}

	for _, addr := range addrs {
		if addr == "" {
			t.Error("Expected non-empty multiaddr")
		}
	}
}
