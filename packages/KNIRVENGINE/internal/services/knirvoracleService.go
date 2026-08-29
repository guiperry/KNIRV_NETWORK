package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// KNIRVOracleService provides integration with KNIRVORACLE for agent minting,
// capability registration, wallet operations, and network interactions
type KNIRVOracleService struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// Configuration for KNIRVORACLE service
type KNIRVOracleConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Timeout int    `json:"timeout_seconds"`
}

// Agent minting request structure
type AgentMintRequest struct {
	AgentID     string                 `json:"agent_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Owner       string                 `json:"owner"`
	Metadata    map[string]interface{} `json:"metadata"`
	ImageURL    string                 `json:"image_url,omitempty"`
}

// Agent minting response structure
type AgentMintResponse struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id"`
	AgentNFTID    string `json:"agent_nft_id"`
	Message       string `json:"message"`
}

// Capability registration request structure
type CapabilityRegistrationRequest struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Description   string                 `json:"description"`
	Schema        map[string]interface{} `json:"schema"`
	Owner         string                 `json:"owner"`
	GasFeeNRN     uint64                 `json:"gas_fee_nrn"`
	LocationHints []string               `json:"location_hints,omitempty"`
}

// Capability registration response structure
type CapabilityRegistrationResponse struct {
	Success      bool   `json:"success"`
	CapabilityID string `json:"capability_id"`
	TxHash       string `json:"tx_hash"`
	Message      string `json:"message"`
}

// Faucet request structure
type FaucetRequest struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
	Reason  string `json:"reason,omitempty"`
}

// Faucet response structure
type FaucetResponse struct {
	Success   bool   `json:"success"`
	RequestID string `json:"request_id"`
	TxHash    string `json:"tx_hash"`
	Amount    string `json:"amount"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// Wallet balance response structure
type WalletBalanceResponse struct {
	Success    bool   `json:"success"`
	Address    string `json:"address"`
	Balance    string `json:"balance"`
	NRNBalance string `json:"nrn_balance"`
	USDValue   string `json:"usd_value"`
}

// Transaction request structure
type TransactionRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount string `json:"amount"`
	Token  string `json:"token"`
	Memo   string `json:"memo,omitempty"`
	GasFee string `json:"gas_fee,omitempty"`
}

// Transaction response structure
type TransactionResponse struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id"`
	TxHash        string `json:"tx_hash"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// NewKNIRVOracleService creates a new KNIRVORACLE service instance
func NewKNIRVOracleService(config KNIRVOracleConfig) *KNIRVOracleService {
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &KNIRVOracleService{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// makeRequest performs HTTP requests to KNIRVORACLE with proper error handling
func (k *KNIRVOracleService) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := k.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KNIRVENGINE-Desktop-Client/1.0")
	if k.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+k.apiKey)
	}

	resp, err := k.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// MintAgent mints a new agent as an NFT on KNIRVORACLE
func (k *KNIRVOracleService) MintAgent(ctx context.Context, req AgentMintRequest) (*AgentMintResponse, error) {
	resp, err := k.makeRequest(ctx, "POST", "/agent/mint", req)
	if err != nil {
		return nil, fmt.Errorf("failed to mint agent: %w", err)
	}
	defer resp.Body.Close()

	var result AgentMintResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("agent minting failed: %s", result.Message)
	}

	return &result, nil
}

// RegisterCapability registers a capability with KNIRVORACLE
func (k *KNIRVOracleService) RegisterCapability(ctx context.Context, req CapabilityRegistrationRequest) (*CapabilityRegistrationResponse, error) {
	resp, err := k.makeRequest(ctx, "POST", "/wallet/mcp/create_register_capability", req)
	if err != nil {
		return nil, fmt.Errorf("failed to register capability: %w", err)
	}
	defer resp.Body.Close()

	var result CapabilityRegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("capability registration failed: %s", result.Message)
	}

	return &result, nil
}

// RequestFaucet requests tokens from the KNIRVORACLE faucet
func (k *KNIRVOracleService) RequestFaucet(ctx context.Context, req FaucetRequest) (*FaucetResponse, error) {
	resp, err := k.makeRequest(ctx, "POST", "/api/mint/nrv", req)
	if err != nil {
		return nil, fmt.Errorf("failed to request faucet: %w", err)
	}
	defer resp.Body.Close()

	var result FaucetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("faucet request failed: %s", result.Message)
	}

	return &result, nil
}

// GetWalletBalance retrieves wallet balance from KNIRVORACLE
func (k *KNIRVOracleService) GetWalletBalance(ctx context.Context, address string) (*WalletBalanceResponse, error) {
	endpoint := fmt.Sprintf("/balance/%s", address)
	resp, err := k.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet balance: %w", err)
	}
	defer resp.Body.Close()

	var result WalletBalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("failed to get balance: status %d", resp.StatusCode)
	}

	return &result, nil
}

// SendTransaction sends a transaction through KNIRVORACLE
func (k *KNIRVOracleService) SendTransaction(ctx context.Context, req TransactionRequest) (*TransactionResponse, error) {
	resp, err := k.makeRequest(ctx, "POST", "/transactions", req)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}
	defer resp.Body.Close()

	var result TransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("transaction failed: %s", result.Message)
	}

	return &result, nil
}

// GetAgentCapabilities retrieves capabilities for a specific agent
func (k *KNIRVOracleService) GetAgentCapabilities(ctx context.Context, agentID string) ([]map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/agent/capabilities/%s", agentID)
	resp, err := k.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent capabilities: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success      bool                     `json:"success"`
		Capabilities []map[string]interface{} `json:"capabilities"`
		Message      string                   `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get capabilities: %s", result.Message)
	}

	return result.Capabilities, nil
}

// InvokeCapability invokes a capability through KNIRVORACLE
func (k *KNIRVOracleService) InvokeCapability(ctx context.Context, capabilityID string, inputData map[string]interface{}) (map[string]interface{}, error) {
	reqBody := map[string]interface{}{
		"capability_id":    capabilityID,
		"interaction_type": "invoke",
		"input_data":       inputData,
		"timestamp":        time.Now().Unix(),
	}

	resp, err := k.makeRequest(ctx, "POST", "/wallet/mcp/create_invoke_capability", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke capability: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("capability invocation failed")
	}

	return result, nil
}

// GetTreasuryStatus retrieves treasury status from KNIRVORACLE
func (k *KNIRVOracleService) GetTreasuryStatus(ctx context.Context) (map[string]interface{}, error) {
	resp, err := k.makeRequest(ctx, "GET", "/api/treasury/status", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get treasury status: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// HealthCheck performs a health check against KNIRVORACLE
func (k *KNIRVOracleService) HealthCheck(ctx context.Context) error {
	resp, err := k.makeRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KNIRVORACLE unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
