// Package embedded_knirvchain provides KNIRV-ORACLE client integration
package embedded_knirvchain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// HTTPOracleClient implements OracleClient using HTTP requests to KNIRV-ORACLE
type HTTPOracleClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	apiKey     string
}

// NewHTTPOracleClient creates a new HTTP-based KNIRV-ORACLE client
func NewHTTPOracleClient(baseURL string, apiKey string) *HTTPOracleClient {
	return &HTTPOracleClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		timeout: 10 * time.Second,
	}
}

// SignalNRNBurn signals the KNIRV-ORACLE to burn NRN tokens via IBC
func (c *HTTPOracleClient) SignalNRNBurn(ctx context.Context, tokenHash string, agentID string, amount int64) error {
	// Create burn signal payload
	burnPayload := map[string]interface{}{
		"burn_request": map[string]interface{}{
			"token_hash":     tokenHash,
			"agent_id":       agentID,
			"amount":         amount,
			"burn_reason":    "skill_invocation_payment",
			"source_chain":   "knirvrouter",
			"target_chain":   "knirv-oracle",
			"timestamp":      time.Now().Unix(),
		},
		"ibc_metadata": map[string]interface{}{
			"channel_id":     "channel-0",
			"port_id":        "transfer",
			"timeout_height": map[string]interface{}{
				"revision_number": 1,
				"revision_height": 1000000,
			},
			"timeout_timestamp": time.Now().Add(1 * time.Hour).UnixNano(),
		},
		"signature_metadata": map[string]interface{}{
			"signer":         "knirvrouter-embedded-chain",
			"signature_type": "secp256k1",
			"timestamp":      time.Now().Unix(),
		},
	}

	// Serialize payload
	jsonData, err := json.Marshal(burnPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal burn payload: %v", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/nrn/burn", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KNIRVROUTER-EmbeddedChain/1.0")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("NRN burn signal failed with status %d", resp.StatusCode)
	}

	// Parse response
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			TransactionHash string `json:"transaction_hash"`
			IBCPacketHash   string `json:"ibc_packet_hash"`
			Status          string `json:"status"`
		} `json:"data"`
		Error string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("NRN burn signal failed: %s", response.Error)
	}

	log.Printf("Successfully signaled NRN burn to oracle - TxHash: %s, IBCHash: %s, Status: %s", 
		response.Data.TransactionHash, response.Data.IBCPacketHash, response.Data.Status)
	
	return nil
}

// MockOracleClient provides a mock implementation for testing
type MockOracleClient struct {
	burnHistory []BurnRecord
}

// BurnRecord represents a record of NRN burn transaction
type BurnRecord struct {
	TokenHash       string    `json:"token_hash"`
	AgentID         string    `json:"agent_id"`
	Amount          int64     `json:"amount"`
	TransactionHash string    `json:"transaction_hash"`
	IBCPacketHash   string    `json:"ibc_packet_hash"`
	Status          string    `json:"status"`
	Timestamp       time.Time `json:"timestamp"`
}

// NewMockOracleClient creates a new mock KNIRV-ORACLE client
func NewMockOracleClient() *MockOracleClient {
	return &MockOracleClient{
		burnHistory: make([]BurnRecord, 0),
	}
}

// SignalNRNBurn simulates NRN burn signaling (mock implementation)
func (m *MockOracleClient) SignalNRNBurn(ctx context.Context, tokenHash string, agentID string, amount int64) error {
	// Simulate network delay
	time.Sleep(100 * time.Millisecond)

	// Generate mock transaction hashes
	txHash := fmt.Sprintf("tx_%s_%d", tokenHash[:8], time.Now().Unix())
	ibcHash := fmt.Sprintf("ibc_%s_%d", agentID[:8], time.Now().Unix())

	// Record the burn
	burnRecord := BurnRecord{
		TokenHash:       tokenHash,
		AgentID:         agentID,
		Amount:          amount,
		TransactionHash: txHash,
		IBCPacketHash:   ibcHash,
		Status:          "pending",
		Timestamp:       time.Now(),
	}

	m.burnHistory = append(m.burnHistory, burnRecord)

	log.Printf("Mock Oracle: Signaled NRN burn - Agent: %s, Amount: %d, TxHash: %s", 
		agentID, amount, txHash)
	
	return nil
}

// GetBurnHistory returns the history of burn transactions (for testing)
func (m *MockOracleClient) GetBurnHistory() []BurnRecord {
	return m.burnHistory
}

// GetBurnByTokenHash finds a burn record by token hash (for testing)
func (m *MockOracleClient) GetBurnByTokenHash(tokenHash string) *BurnRecord {
	for _, record := range m.burnHistory {
		if record.TokenHash == tokenHash {
			return &record
		}
	}
	return nil
}

// UpdateBurnStatus updates the status of a burn transaction (for testing)
func (m *MockOracleClient) UpdateBurnStatus(tokenHash string, status string) bool {
	for i, record := range m.burnHistory {
		if record.TokenHash == tokenHash {
			m.burnHistory[i].Status = status
			log.Printf("Mock Oracle: Updated burn status for %s to %s", tokenHash, status)
			return true
		}
	}
	return false
}

// GetTotalBurnedAmount returns the total amount burned by an agent (for testing)
func (m *MockOracleClient) GetTotalBurnedAmount(agentID string) int64 {
	total := int64(0)
	for _, record := range m.burnHistory {
		if record.AgentID == agentID && record.Status == "completed" {
			total += record.Amount
		}
	}
	return total
}

// IBCOracleClient provides IBC-specific oracle client functionality
type IBCOracleClient struct {
	*HTTPOracleClient
	channelID string
	portID    string
}

// NewIBCOracleClient creates a new IBC-enabled KNIRV-ORACLE client
func NewIBCOracleClient(baseURL string, apiKey string, channelID string, portID string) *IBCOracleClient {
	return &IBCOracleClient{
		HTTPOracleClient: NewHTTPOracleClient(baseURL, apiKey),
		channelID:        channelID,
		portID:           portID,
	}
}

// SignalNRNBurn signals NRN burn with IBC-specific parameters
func (c *IBCOracleClient) SignalNRNBurn(ctx context.Context, tokenHash string, agentID string, amount int64) error {
	// Create IBC-specific burn signal payload
	burnPayload := map[string]interface{}{
		"burn_request": map[string]interface{}{
			"token_hash":     tokenHash,
			"agent_id":       agentID,
			"amount":         amount,
			"burn_reason":    "skill_invocation_payment",
			"source_chain":   "knirvrouter",
			"target_chain":   "knirv-oracle",
			"timestamp":      time.Now().Unix(),
		},
		"ibc_metadata": map[string]interface{}{
			"channel_id":     c.channelID,
			"port_id":        c.portID,
			"timeout_height": map[string]interface{}{
				"revision_number": 1,
				"revision_height": 1000000,
			},
			"timeout_timestamp": time.Now().Add(1 * time.Hour).UnixNano(),
		},
		"routing_metadata": map[string]interface{}{
			"source_router":  "knirvrouter-embedded-chain",
			"target_oracle":  "knirv-oracle-main",
			"priority":       "normal",
			"retry_policy":   "exponential_backoff",
		},
	}

	// Serialize payload
	jsonData, err := json.Marshal(burnPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal IBC burn payload: %v", err)
	}

	// Create HTTP request to IBC-specific endpoint
	url := fmt.Sprintf("%s/api/v1/ibc/nrn/burn", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create IBC HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KNIRVROUTER-EmbeddedChain-IBC/1.0")
	req.Header.Set("X-IBC-Channel-ID", c.channelID)
	req.Header.Set("X-IBC-Port-ID", c.portID)
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("IBC HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("IBC NRN burn signal failed with status %d", resp.StatusCode)
	}

	// Parse response
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			TransactionHash string `json:"transaction_hash"`
			IBCPacketHash   string `json:"ibc_packet_hash"`
			ChannelID       string `json:"channel_id"`
			PortID          string `json:"port_id"`
			Status          string `json:"status"`
		} `json:"data"`
		Error string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode IBC response: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("IBC NRN burn signal failed: %s", response.Error)
	}

	log.Printf("Successfully signaled IBC NRN burn - Channel: %s, Port: %s, TxHash: %s", 
		response.Data.ChannelID, response.Data.PortID, response.Data.TransactionHash)
	
	return nil
}
