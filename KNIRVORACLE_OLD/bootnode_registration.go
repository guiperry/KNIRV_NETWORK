package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RegistrationPayload struct {
	ChainID string `json:"chainID"`
	IP      string `json:"ip,omitempty"` // omitempty if you want it to be optional
	Port    int    `json:"port"`
	Type    string `json:"type"`   // "bootnode" or "client" etc.
	PeerID  string `json:"nodeID"` // Add PeerID
}

func RegisterWithRegistry(registryURL string, nodeChainID string, nodeP2PPort int, nodeIP string, nodePeerID string) error {
	// Ensure registry URL has proper scheme
	if !strings.HasPrefix(registryURL, "http://") && !strings.HasPrefix(registryURL, "https://") {
		registryURL = "http://" + registryURL
	}

	// Prepare the registration payload
	payload := RegistrationPayload{
		ChainID: nodeChainID, // This is the logical ChainID (e.g., KNIRVORACLE-5000)
		IP:      nodeIP,
		Port:    nodeP2PPort, // This is the P2P listening port
		Type:    "bootnode",
		PeerID:  nodePeerID, // The actual libp2p Peer ID
	}

	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal registration payload: %w", err)
	}

	// Set up HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare registry endpoints
	baseURL, err := url.Parse(registryURL)
	if err != nil {
		return fmt.Errorf("invalid registry URL: %w", err)
	}
	registerURL := baseURL.JoinPath("/register").String()
	statusURL := baseURL.JoinPath("/status").String()

	// Create the request
	req, err := http.NewRequest("POST", registerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		// Open status page on error
		fmt.Println("Failed to create HTTP request. Opening registry status page...")
		if openErr := openBrowser(statusURL); openErr != nil {
			fmt.Printf("Note: Could not open browser: %v\n", openErr)
			fmt.Printf("Please check the registry status manually at: %s\n", statusURL)
		}
		return fmt.Errorf("failed to create registration request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KNIRVORACLE-Node/1.0")

	// Send the request
	log.Printf("Registering node %s (PeerID: %s) at %s:%d with KNIRVORACLE Node Registry", nodeChainID, nodePeerID, nodeIP, nodeP2PPort)
	resp, err := client.Do(req)
	if err != nil {
		// Open status page on error
		fmt.Println("Failed to connect to registry. Opening registry status page...")
		if openErr := openBrowser(statusURL); openErr != nil {
			fmt.Printf("Note: Could not open browser: %v\n", openErr)
			fmt.Printf("Please check the registry status manually at: %s\n", statusURL)
		}
		return fmt.Errorf("failed to send registration request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Open status page on error
		fmt.Println("Failed to read response. Opening registry status page...")
		if openErr := openBrowser(statusURL); openErr != nil {
			fmt.Printf("Note: Could not open browser: %v\n", openErr)
			fmt.Printf("Please check the registry status manually at: %s\n", statusURL)
		}
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check the status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Open status page on non-OK status
		fmt.Println("Registry returned error. Opening registry status page...")
		if openErr := openBrowser(statusURL); openErr != nil {
			fmt.Printf("Note: Could not open browser: %v\n", openErr)
			fmt.Printf("Please check the registry status manually at: %s\n", statusURL)
		}
		return fmt.Errorf("registry returned non-successful status: %d - %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		// Open status page on error
		fmt.Println("Failed to parse response. Opening registry status page...")
		if openErr := openBrowser(statusURL); openErr != nil {
			fmt.Printf("Note: Could not open browser: %v\n", openErr)
			fmt.Printf("Please check the registry status manually at: %s\n", statusURL)
		}
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Log successful registration details
	if message, ok := responseData["message"].(string); ok {
		log.Printf("Registry response: %s", message)
	}
	if registeredIP, ok := responseData["registeredIp"].(string); ok {
		log.Printf("Registered with IP: %s", registeredIP)
	}

	return nil
}

// ... in your bootnode startup logic ...
// err := registerWithRegistry("http://registry.knirv.com:3003", "KNIRVORACLE-[chainID]]", Port, "detectedPublicIP")
// if err != nil {
//     log.Printf("[Bootnode][%s] WARNING: Failed to register with registry: %v", "KNIRVORACLE-5000", err)
// }
