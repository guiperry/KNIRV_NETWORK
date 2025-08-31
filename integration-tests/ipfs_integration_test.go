package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// IPFSTestSuite contains all IPFS integration tests for KNIRV production network
type IPFSTestSuite struct {
	BaseURL    string
	GatewayURL string
	Client     *http.Client
}

// IPFSVersionResponse represents the IPFS version API response
type IPFSVersionResponse struct {
	Version string `json:"Version"`
	Commit  string `json:"Commit"`
	Repo    string `json:"Repo"`
	System  string `json:"System"`
	Golang  string `json:"Golang"`
}

// IPFSAddResponse represents the IPFS add API response
type IPFSAddResponse struct {
	Name string `json:"Name"`
	Hash string `json:"Hash"`
	Size string `json:"Size"`
}

// IPFSIDResponse represents the IPFS ID API response
type IPFSIDResponse struct {
	ID              string   `json:"ID"`
	PublicKey       string   `json:"PublicKey"`
	Addresses       []string `json:"Addresses"`
	AgentVersion    string   `json:"AgentVersion"`
	ProtocolVersion string   `json:"ProtocolVersion"`
}

// NewIPFSTestSuite creates a new IPFS test suite
func NewIPFSTestSuite() *IPFSTestSuite {
	return &IPFSTestSuite{
		BaseURL:    "http://localhost:5001",
		GatewayURL: "http://localhost:8080",
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TestIPFSConnectivity tests basic IPFS connectivity
func (suite *IPFSTestSuite) TestIPFSConnectivity(t *testing.T) {
	t.Log("Testing IPFS API connectivity...")
	
	resp, err := suite.Client.Get(suite.BaseURL + "/api/v0/version")
	if err != nil {
		t.Fatalf("Failed to connect to IPFS API: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("IPFS API returned status %d", resp.StatusCode)
	}
	
	var version IPFSVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		t.Fatalf("Failed to decode version response: %v", err)
	}
	
	t.Logf("IPFS Version: %s", version.Version)
	t.Logf("IPFS Commit: %s", version.Commit)
	
	// Verify KNIRV-specific agent version
	if !strings.Contains(version.Version, "kubo") {
		t.Errorf("Expected kubo in version string, got: %s", version.Version)
	}
}

// TestIPFSNodeID tests IPFS node identification
func (suite *IPFSTestSuite) TestIPFSNodeID(t *testing.T) {
	t.Log("Testing IPFS node ID...")
	
	resp, err := suite.Client.Get(suite.BaseURL + "/api/v0/id")
	if err != nil {
		t.Fatalf("Failed to get IPFS node ID: %v", err)
	}
	defer resp.Body.Close()
	
	var nodeID IPFSIDResponse
	if err := json.NewDecoder(resp.Body).Decode(&nodeID); err != nil {
		t.Fatalf("Failed to decode node ID response: %v", err)
	}
	
	t.Logf("IPFS Node ID: %s", nodeID.ID)
	t.Logf("Agent Version: %s", nodeID.AgentVersion)
	t.Logf("Protocol Version: %s", nodeID.ProtocolVersion)
	t.Logf("Addresses: %v", nodeID.Addresses)
	
	// Verify KNIRV-specific configuration
	if !strings.Contains(nodeID.AgentVersion, "knirv-production") {
		t.Errorf("Expected knirv-production in agent version, got: %s", nodeID.AgentVersion)
	}
	
	if len(nodeID.Addresses) == 0 {
		t.Error("No addresses found for IPFS node")
	}
}

// TestIPFSContentOperations tests adding and retrieving content
func (suite *IPFSTestSuite) TestIPFSContentOperations(t *testing.T) {
	t.Log("Testing IPFS content operations...")
	
	// Test content to add
	testContent := fmt.Sprintf("KNIRV Network Test Content - %d", time.Now().Unix())
	
	// Create multipart form for file upload
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	
	if _, err := part.Write([]byte(testContent)); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	
	writer.Close()
	
	// Add content to IPFS
	resp, err := suite.Client.Post(
		suite.BaseURL+"/api/v0/add",
		writer.FormDataContentType(),
		&buf,
	)
	if err != nil {
		t.Fatalf("Failed to add content to IPFS: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("IPFS add returned status %d", resp.StatusCode)
	}
	
	var addResp IPFSAddResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("Failed to decode add response: %v", err)
	}
	
	t.Logf("Content added with hash: %s", addResp.Hash)
	t.Logf("Content size: %s bytes", addResp.Size)
	
	// Retrieve content via API
	catResp, err := suite.Client.Get(suite.BaseURL + "/api/v0/cat?arg=" + addResp.Hash)
	if err != nil {
		t.Fatalf("Failed to retrieve content from IPFS: %v", err)
	}
	defer catResp.Body.Close()
	
	retrievedContent, err := io.ReadAll(catResp.Body)
	if err != nil {
		t.Fatalf("Failed to read retrieved content: %v", err)
	}
	
	if string(retrievedContent) != testContent {
		t.Errorf("Retrieved content doesn't match original. Expected: %s, Got: %s", 
			testContent, string(retrievedContent))
	}
	
	// Test gateway access
	gatewayResp, err := suite.Client.Get(suite.GatewayURL + "/ipfs/" + addResp.Hash)
	if err != nil {
		t.Fatalf("Failed to access content via gateway: %v", err)
	}
	defer gatewayResp.Body.Close()
	
	gatewayContent, err := io.ReadAll(gatewayResp.Body)
	if err != nil {
		t.Fatalf("Failed to read gateway content: %v", err)
	}
	
	if string(gatewayContent) != testContent {
		t.Errorf("Gateway content doesn't match original. Expected: %s, Got: %s", 
			testContent, string(gatewayContent))
	}
	
	t.Log("Content operations test passed")
}

// TestIPFSPinning tests content pinning functionality
func (suite *IPFSTestSuite) TestIPFSPinning(t *testing.T) {
	t.Log("Testing IPFS pinning functionality...")
	
	// Use a known hash for testing (empty directory)
	testHash := "QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn"
	
	// Pin the content
	pinResp, err := suite.Client.Post(
		suite.BaseURL+"/api/v0/pin/add?arg="+testHash,
		"application/json",
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to pin content: %v", err)
	}
	defer pinResp.Body.Close()
	
	if pinResp.StatusCode != http.StatusOK {
		t.Fatalf("Pin operation returned status %d", pinResp.StatusCode)
	}
	
	// List pinned content
	listResp, err := suite.Client.Get(suite.BaseURL + "/api/v0/pin/ls")
	if err != nil {
		t.Fatalf("Failed to list pinned content: %v", err)
	}
	defer listResp.Body.Close()
	
	listBody, err := io.ReadAll(listResp.Body)
	if err != nil {
		t.Fatalf("Failed to read pin list response: %v", err)
	}
	
	if !strings.Contains(string(listBody), testHash) {
		t.Errorf("Pinned hash %s not found in pin list", testHash)
	}
	
	t.Log("Pinning test passed")
}

// TestIPFSSwarmConnectivity tests IPFS swarm connectivity
func (suite *IPFSTestSuite) TestIPFSSwarmConnectivity(t *testing.T) {
	t.Log("Testing IPFS swarm connectivity...")
	
	resp, err := suite.Client.Get(suite.BaseURL + "/api/v0/swarm/peers")
	if err != nil {
		t.Fatalf("Failed to get swarm peers: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read swarm peers response: %v", err)
	}
	
	t.Logf("Swarm peers response: %s", string(body))
	
	// Note: In a local test environment, there might be no peers
	// This test mainly verifies the API is accessible
	t.Log("Swarm connectivity test completed")
}

// RunIPFSIntegrationTests runs all IPFS integration tests
func TestIPFSIntegration(t *testing.T) {
	suite := NewIPFSTestSuite()
	
	// Wait for IPFS to be ready
	t.Log("Waiting for IPFS to be ready...")
	for i := 0; i < 30; i++ {
		resp, err := suite.Client.Get(suite.BaseURL + "/api/v0/version")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
		if i == 29 {
			t.Fatal("IPFS not ready after 30 seconds")
		}
	}
	
	t.Run("Connectivity", suite.TestIPFSConnectivity)
	t.Run("NodeID", suite.TestIPFSNodeID)
	t.Run("ContentOperations", suite.TestIPFSContentOperations)
	t.Run("Pinning", suite.TestIPFSPinning)
	t.Run("SwarmConnectivity", suite.TestIPFSSwarmConnectivity)
}
