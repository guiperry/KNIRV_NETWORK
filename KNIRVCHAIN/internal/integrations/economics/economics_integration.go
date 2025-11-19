package economics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"KNIRVCHAIN/economics"
)

// Transaction represents a basic transaction structure
type Transaction struct {
	Value           uint64 // Transaction value
	From            string // Sender address
	To              string // Recipient address
	TransactionHash string // Transaction hash
	Timestamp       int64  // Transaction timestamp
	Type            string // Transaction type
	Data            []byte // Transaction data
}

// EconomicsIntegrationImpl handles integration with the economics service
type EconomicsIntegrationImpl struct {
	economicsURL     string
	httpClient       *http.Client
	enabled          bool
	economicsService *economics.EconomicsService
	localMode        bool
}

// EconomicsRequest represents a request to the economics service
type EconomicsRequest struct {
	UserID string                 `json:"user_id"`
	Amount string                 `json:"amount"`
	Type   string                 `json:"type"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// EconomicsResponse represents a response from the economics service
type EconomicsResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// TransactionEvent represents a blockchain transaction event for economics
type TransactionEvent struct {
	TxID      string                 `json:"tx_id"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Amount    *big.Int               `json:"amount"`
	Type      string                 `json:"type"`
	Success   bool                   `json:"success"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// WalletActivityEvent represents wallet activity for economics tracking
type WalletActivityEvent struct {
	WalletID  string                 `json:"wallet_id"`
	Activity  string                 `json:"activity"`
	Amount    *big.Int               `json:"amount,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewEconomicsIntegration creates a new economics integration instance
func NewEconomicsIntegration() EconomicsIntegration {
	economicsURL := os.Getenv("ECONOMICS_SERVICE_URL")
	localMode := os.Getenv("ECONOMICS_LOCAL_MODE") == "true"

	if economicsURL == "" {
		economicsURL = "http://localhost:8090"
	}

	ei := &EconomicsIntegrationImpl{
		economicsURL: economicsURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		enabled:   true,
		localMode: localMode,
	}

	// If local mode is enabled, initialize the economics service directly
	if localMode {
		if err := ei.initializeLocalEconomicsService(); err != nil {
			log.Printf("Failed to initialize local economics service: %v", err)
			ei.localMode = false // Fall back to remote mode
		}
	}

	return ei
}

// initializeLocalEconomicsService initializes the local economics service
func (ei *EconomicsIntegrationImpl) initializeLocalEconomicsService() error {
	config := &economics.ServiceConfig{
		Port:        "8090",
		NRNContract: "nrn_contract_address_placeholder",
		XionRPC:     "https://rpc.xion-testnet-1.burnt.com:443",
		ComponentConfig: economics.ComponentConfig{
			KNIRVChainURL: "https://chain.knirv.com",
			KNIRVNexusURL: "https://nexus.knirv.com",
			KNIRVRootURL:  "https://root.knirv.com",
			KNIRVGraphURL: "https://graph.knirv.com",
		},
		DatabasePath: "./economics.db",
		EnableCORS:   true,
		LogLevel:     "info",
	}

	economicsService, err := economics.NewEconomicsService(config)
	if err != nil {
		return fmt.Errorf("failed to create economics service: %w", err)
	}

	ei.economicsService = economicsService

	// Start the service in a goroutine
	go func() {
		if err := ei.economicsService.Start(); err != nil {
			log.Printf("Economics service error: %v", err)
		}
	}()

	log.Println("Local economics service initialized and started")
	return nil
}

// ProcessPayment processes a payment through the economics service
func (ei *EconomicsIntegrationImpl) ProcessPayment(userID, amount, paymentType string, metadata map[string]interface{}) (*EconomicsResponse, error) {
	if !ei.enabled {
		return &EconomicsResponse{Success: true, Data: map[string]interface{}{"status": "disabled"}}, nil
	}

	request := EconomicsRequest{
		UserID: userID,
		Amount: amount,
		Type:   paymentType,
		Data:   metadata,
	}

	var endpoint string
	switch paymentType {
	case "skill_invocation":
		endpoint = "/economics/skill/invoke"
		request.Data["skill_id"] = metadata["skill_id"]
	case "llm_registration":
		endpoint = "/economics/llm/register"
		request.Data["llm_id"] = metadata["llm_id"]
	default:
		return nil, fmt.Errorf("unsupported payment type: %s", paymentType)
	}

	return ei.makeRequest("POST", endpoint, request)
}

// RecordTransaction records a blockchain transaction with the economics service
func (ei *EconomicsIntegrationImpl) RecordTransaction(event TransactionEvent) error {
	if !ei.enabled {
		return nil
	}

	// For now, just log the transaction
	// In a real implementation, this would send to the economics service
	log.Printf("Recording transaction event: %+v", event)
	return nil
}

// RecordWalletActivity records wallet activity with the economics service
func (ei *EconomicsIntegrationImpl) RecordWalletActivity(event WalletActivityEvent) error {
	if !ei.enabled {
		return nil
	}

	// For now, just log the activity
	// In a real implementation, this would send to the economics service
	log.Printf("Recording wallet activity: %+v", event)
	return nil
}

// GetEconomicMetrics retrieves economic metrics from the economics service
func (ei *EconomicsIntegrationImpl) GetEconomicMetrics() (*EconomicMetrics, error) {
	if !ei.enabled {
		return &EconomicMetrics{
			LastUpdated: time.Now(),
		}, nil
	}

	if ei.localMode && ei.economicsService != nil {
		// For local mode, return simulated economic metrics
		return &EconomicMetrics{
			TotalMarketCap:     big.NewInt(5000000000000), // $5T
			TotalVolume24h:     big.NewInt(100000000000),  // $100B
			ActiveTokens:       50,
			TotalTransactions:  1000000,
			AverageGasPrice:    big.NewInt(20000000000), // 20 gwei
			NetworkUtilization: 0.75,                    // 75%
			TokenMetrics: map[string]*TokenMetrics{
				"NRN": {
					TokenID:         "NRN",
					Price:           big.NewInt(1000000),
					MarketCap:       big.NewInt(1000000000000),
					Volume24h:       big.NewInt(10000000000),
					PriceChange24h:  2.5,
					VolumeChange24h: -5.2,
					Holders:         10000,
					Transactions24h: 5000,
					Liquidity:       big.NewInt(50000000000),
					LastUpdated:     time.Now(),
				},
			},
			LastUpdated: time.Now(),
		}, nil
	}

	// For remote mode, make HTTP request
	response, err := ei.makeRequest("GET", "/economics/metrics", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get economic metrics: %w", err)
	}

	// Convert EconomicsResponse to EconomicMetrics
	if response.Success && response.Data != nil {
		// response.Data is already map[string]interface{}
		metricsData := response.Data
		metrics := &EconomicMetrics{
			LastUpdated: time.Now(),
		}

		// Parse available fields
		if totalMarketCap, ok := metricsData["total_market_cap"].(string); ok {
			if cap := new(big.Int); cap != nil {
				if _, success := cap.SetString(totalMarketCap, 10); success {
					metrics.TotalMarketCap = cap
				}
			}
		}

		if totalVolume, ok := metricsData["total_volume_24h"].(string); ok {
			if vol := new(big.Int); vol != nil {
				if _, success := vol.SetString(totalVolume, 10); success {
					metrics.TotalVolume24h = vol
				}
			}
		}

		if activeTokens, ok := metricsData["active_tokens"].(float64); ok {
			metrics.ActiveTokens = int(activeTokens)
		}

		if totalTx, ok := metricsData["total_transactions"].(float64); ok {
			metrics.TotalTransactions = uint64(totalTx)
		}

		if networkUtil, ok := metricsData["network_utilization"].(float64); ok {
			metrics.NetworkUtilization = networkUtil
		}

		return metrics, nil
	}

	// Fallback to default metrics
	return &EconomicMetrics{
		LastUpdated: time.Now(),
	}, nil
}

// GetServiceMetrics retrieves KNIRVCHAIN-specific metrics from the economics service
func (ei *EconomicsIntegrationImpl) GetServiceMetrics() (*EconomicsResponse, error) {
	if !ei.enabled {
		return &EconomicsResponse{Success: true, Data: map[string]interface{}{"status": "disabled"}}, nil
	}

	return ei.makeRequest("GET", "/economics/service/knirvoracle/metrics", nil)
}

// HealthCheck checks if the economics service is healthy
func (ei *EconomicsIntegrationImpl) HealthCheck() bool {
	if !ei.enabled {
		return true
	}

	resp, err := ei.makeRequest("GET", "/economics/health", nil)
	return err == nil && resp.Success
}

// makeRequest makes an HTTP request to the economics service
func (ei *EconomicsIntegrationImpl) makeRequest(method, endpoint string, data interface{}) (*EconomicsResponse, error) {
	url := ei.economicsURL + endpoint

	var reqBody []byte
	var err error
	if data != nil {
		reqBody, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	var req *http.Request
	if reqBody != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ei.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var economicsResp EconomicsResponse
	if err := json.NewDecoder(resp.Body).Decode(&economicsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &economicsResp, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, economicsResp.Error)
	}

	return &economicsResp, nil
}

// Enable enables the economics integration
func (ei *EconomicsIntegrationImpl) Enable() {
	ei.enabled = true
	log.Println("Economics integration enabled")
}

// Disable disables the economics integration
func (ei *EconomicsIntegrationImpl) Disable() {
	ei.enabled = false
	log.Println("Economics integration disabled")
}

// IsEnabled returns whether the economics integration is enabled
func (ei *EconomicsIntegrationImpl) IsEnabled() bool {
	return ei.enabled
}

// IsLocalMode returns whether the economics service is running locally
func (ei *EconomicsIntegrationImpl) IsLocalMode() bool {
	return ei.localMode
}

// GetLocalEconomicsService returns the local economics service instance
func (ei *EconomicsIntegrationImpl) GetLocalEconomicsService() *economics.EconomicsService {
	if ei.localMode {
		return ei.economicsService
	}
	return nil
}

// StopLocalEconomicsService stops the local economics service
func (ei *EconomicsIntegrationImpl) StopLocalEconomicsService() error {
	if !ei.localMode || ei.economicsService == nil {
		return fmt.Errorf("local economics service is not running")
	}

	ei.economicsService.Stop()
	log.Println("Local economics service stopped")
	return nil
}

// StartBackgroundSync starts a background goroutine to sync with the economics service
func (ei *EconomicsIntegrationImpl) StartBackgroundSync(ctx context.Context) {
	if !ei.enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(2 * time.Minute) // Sync every 2 minutes
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ei.performSync()
			}
		}
	}()
}

// performSync performs a sync with the economics service
func (ei *EconomicsIntegrationImpl) performSync() {
	// Health check
	if !ei.HealthCheck() {
		log.Println("Economics service health check failed")
		return
	}

	// Get metrics
	metrics, err := ei.GetEconomicMetrics()
	if err != nil {
		log.Printf("Failed to get economic metrics: %v", err)
		return
	}

	log.Printf("Economics sync successful: %+v", metrics)
}

// Integration helper functions for existing KNIRVCHAIN code

// IntegrateTransactionProcessing integrates transaction processing with economics
func (ei *EconomicsIntegrationImpl) IntegrateTransactionProcessing(tx Transaction) error {
	// Convert uint64 value to *big.Int for Amount field
	amount := big.NewInt(0).SetUint64(tx.Value)

	// Convert int64 timestamp to time.Time
	timestamp := time.Unix(tx.Timestamp, 0)

	event := TransactionEvent{
		TxID:      tx.TransactionHash,
		From:      tx.From,
		To:        tx.To,
		Amount:    amount,
		Type:      tx.Type,
		Success:   true, // Assume success if we're recording it
		Timestamp: timestamp,
		Metadata: map[string]interface{}{
			"data": tx.Data,
		},
	}

	return ei.RecordTransaction(event)
}

// IntegrateWalletOperation integrates wallet operations with economics
func (ei *EconomicsIntegrationImpl) IntegrateWalletOperation(walletID, operation string, amount *big.Int, metadata map[string]interface{}) error {
	event := WalletActivityEvent{
		WalletID:  walletID,
		Activity:  operation,
		Amount:    amount,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	return ei.RecordWalletActivity(event)
}

// IntegratePaymentProcessing integrates payment processing with economics
func (ei *EconomicsIntegrationImpl) IntegratePaymentProcessing(userID, amount, paymentType string, metadata map[string]interface{}) (*EconomicsResponse, error) {
	return ei.ProcessPayment(userID, amount, paymentType, metadata)
}

// AddEconomicsEndpoints adds economics-related endpoints to the HTTP server
func (ei *EconomicsIntegrationImpl) AddEconomicsEndpoints() {
	// Add endpoint to get economics metrics
	http.HandleFunc("/api/economics/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metrics, err := ei.GetEconomicMetrics()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	})

	// Add endpoint to get service-specific metrics
	http.HandleFunc("/api/economics/service-metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metrics, err := ei.GetServiceMetrics()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	})

	// Add endpoint to check economics integration status
	http.HandleFunc("/api/economics/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		status := map[string]interface{}{
			"enabled":     ei.IsEnabled(),
			"healthy":     ei.HealthCheck(),
			"service_url": ei.economicsURL,
			"timestamp":   time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    status,
		})
	})
}

// Interface implementation stubs - these need to be properly implemented

func (ei *EconomicsIntegrationImpl) Initialize(config *EconomicsConfig) error {
	if config == nil {
		return fmt.Errorf("economics config cannot be nil")
	}

	// Update configuration
	if config.ServiceURL != "" {
		ei.economicsURL = config.ServiceURL
	}

	// Initialize HTTP client with timeout
	ei.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Test connection to economics service
	if err := ei.testConnection(); err != nil {
		log.Printf("Warning: Failed to connect to economics service at %s: %v", ei.economicsURL, err)
		// Don't fail initialization, just log the warning
	}

	ei.enabled = true
	log.Printf("Economics integration initialized with service URL: %s", ei.economicsURL)
	return nil
}

// testConnection tests the connection to the economics service
func (ei *EconomicsIntegrationImpl) testConnection() error {
	if ei.localMode && ei.economicsService != nil {
		// For local mode, just check if the service is initialized
		return nil
	}

	// For remote mode, make a health check request
	resp, err := ei.httpClient.Get(ei.economicsURL + "/health")
	if err != nil {
		return fmt.Errorf("failed to connect to economics service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("economics service returned status %d", resp.StatusCode)
	}

	return nil
}

func (ei *EconomicsIntegrationImpl) Connect() error {
	if !ei.enabled {
		return fmt.Errorf("economics integration is disabled")
	}

	return ei.testConnection()
}

func (ei *EconomicsIntegrationImpl) Disconnect() error {
	if ei.localMode && ei.economicsService != nil {
		// For local mode, stop the economics service
		if err := ei.economicsService.Stop(); err != nil {
			return fmt.Errorf("failed to stop local economics service: %w", err)
		}
	}

	ei.enabled = false
	log.Println("Economics integration disconnected")
	return nil
}

func (ei *EconomicsIntegrationImpl) IsConnected() bool {
	if !ei.enabled {
		return false
	}

	// Test the connection
	if err := ei.testConnection(); err != nil {
		return false
	}

	return true
}

func (ei *EconomicsIntegrationImpl) GetTokenBalance(address string, tokenType string) (*big.Int, error) {
	if address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	if tokenType == "" {
		tokenType = "NRN" // Default to NRN token
	}

	if ei.localMode && ei.economicsService != nil {
		// For local mode, we'll simulate a balance lookup
		// In a real implementation, this would query the local database
		// For now, return a default balance
		return big.NewInt(1000000), nil // 1 NRN default balance
	}

	// For remote mode, make HTTP request
	url := fmt.Sprintf("%s/api/tokens/%s/balance/%s", ei.economicsURL, tokenType, address)
	resp, err := ei.httpClient.Get(url)
	if err != nil {
		return big.NewInt(0), fmt.Errorf("failed to get token balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return big.NewInt(0), fmt.Errorf("economics service returned status %d", resp.StatusCode)
	}

	var response struct {
		Success bool   `json:"success"`
		Balance string `json:"balance"`
		Error   string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return big.NewInt(0), fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return big.NewInt(0), fmt.Errorf("economics service error: %s", response.Error)
	}

	balance := new(big.Int)
	if _, ok := balance.SetString(response.Balance, 10); !ok {
		return big.NewInt(0), fmt.Errorf("invalid balance format: %s", response.Balance)
	}

	return balance, nil
}

func (ei *EconomicsIntegrationImpl) TransferTokens(from, to string, amount *big.Int, tokenType string) (*TransferResult, error) {
	if from == "" || to == "" {
		return nil, fmt.Errorf("from and to addresses cannot be empty")
	}

	if amount == nil || amount.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	if tokenType == "" {
		tokenType = "NRN" // Default to NRN token
	}

	// Create transfer request
	transferReq := map[string]interface{}{
		"from":       from,
		"to":         to,
		"amount":     amount.String(),
		"token_type": tokenType,
		"timestamp":  time.Now().Unix(),
	}

	if ei.localMode && ei.economicsService != nil {
		// For local mode, simulate the transfer
		return &TransferResult{
			TxHash:    fmt.Sprintf("local_tx_%d", time.Now().UnixNano()),
			From:      from,
			To:        to,
			Amount:    amount,
			Fee:       big.NewInt(1000), // 0.001 NRN fee
			Status:    "confirmed",
			Timestamp: time.Now(),
		}, nil
	}

	// For remote mode, make HTTP request
	reqBody, err := json.Marshal(transferReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transfer request: %w", err)
	}

	url := fmt.Sprintf("%s/api/tokens/%s/transfer", ei.economicsURL, tokenType)
	resp, err := ei.httpClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to transfer tokens: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("economics service returned status %d", resp.StatusCode)
	}

	var response struct {
		Success bool   `json:"success"`
		TxHash  string `json:"tx_hash"`
		Error   string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("economics service error: %s", response.Error)
	}

	return &TransferResult{
		TxHash:    response.TxHash,
		From:      from,
		To:        to,
		Amount:    amount,
		Fee:       big.NewInt(1000), // Default fee
		Status:    "confirmed",
		Timestamp: time.Now(),
	}, nil
}

func (ei *EconomicsIntegrationImpl) MintTokens(to string, amount *big.Int, tokenType string) (*MintResult, error) {
	if to == "" {
		return nil, fmt.Errorf("to address cannot be empty")
	}

	if amount == nil || amount.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	if tokenType == "" {
		tokenType = "NRN" // Default to NRN token
	}

	// Create mint request
	mintReq := map[string]interface{}{
		"to":         to,
		"amount":     amount.String(),
		"token_type": tokenType,
		"timestamp":  time.Now().Unix(),
	}

	if ei.localMode && ei.economicsService != nil {
		// For local mode, simulate the minting
		return &MintResult{
			TxHash:    fmt.Sprintf("mint_tx_%d", time.Now().UnixNano()),
			To:        to,
			Amount:    amount,
			NewSupply: big.NewInt(1000000000), // Simulated new supply
			Status:    "confirmed",
			Timestamp: time.Now(),
		}, nil
	}

	// For remote mode, make HTTP request
	reqBody, err := json.Marshal(mintReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mint request: %w", err)
	}

	url := fmt.Sprintf("%s/api/tokens/%s/mint", ei.economicsURL, tokenType)
	resp, err := ei.httpClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to mint tokens: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("economics service returned status %d", resp.StatusCode)
	}

	var response struct {
		Success   bool   `json:"success"`
		TxHash    string `json:"tx_hash"`
		NewSupply string `json:"new_supply"`
		Error     string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("economics service error: %s", response.Error)
	}

	newSupply := new(big.Int)
	if response.NewSupply != "" {
		if _, ok := newSupply.SetString(response.NewSupply, 10); !ok {
			newSupply = big.NewInt(0)
		}
	}

	return &MintResult{
		TxHash:    response.TxHash,
		To:        to,
		Amount:    amount,
		NewSupply: newSupply,
		Status:    "confirmed",
		Timestamp: time.Now(),
	}, nil
}

func (ei *EconomicsIntegrationImpl) BurnTokens(from string, amount *big.Int, tokenType string) (*BurnResult, error) {
	if from == "" {
		return nil, fmt.Errorf("from address cannot be empty")
	}

	if amount == nil || amount.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	if tokenType == "" {
		tokenType = "NRN" // Default to NRN token
	}

	// Create burn request
	burnReq := map[string]interface{}{
		"from":       from,
		"amount":     amount.String(),
		"token_type": tokenType,
		"timestamp":  time.Now().Unix(),
	}

	if ei.localMode && ei.economicsService != nil {
		// For local mode, simulate the burning
		return &BurnResult{
			TxHash:    fmt.Sprintf("burn_tx_%d", time.Now().UnixNano()),
			From:      from,
			Amount:    amount,
			NewSupply: big.NewInt(999000000), // Simulated new supply after burn
			Status:    "confirmed",
			Timestamp: time.Now(),
		}, nil
	}

	// For remote mode, make HTTP request
	reqBody, err := json.Marshal(burnReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal burn request: %w", err)
	}

	url := fmt.Sprintf("%s/api/tokens/%s/burn", ei.economicsURL, tokenType)
	resp, err := ei.httpClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to burn tokens: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("economics service returned status %d", resp.StatusCode)
	}

	var response struct {
		Success   bool   `json:"success"`
		TxHash    string `json:"tx_hash"`
		NewSupply string `json:"new_supply"`
		Error     string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("economics service error: %s", response.Error)
	}

	newSupply := new(big.Int)
	if response.NewSupply != "" {
		if _, ok := newSupply.SetString(response.NewSupply, 10); !ok {
			newSupply = big.NewInt(0)
		}
	}

	return &BurnResult{
		TxHash:    response.TxHash,
		From:      from,
		Amount:    amount,
		NewSupply: newSupply,
		Status:    "confirmed",
		Timestamp: time.Now(),
	}, nil
}

func (ei *EconomicsIntegrationImpl) GetTokenMetrics(tokenType string) (*TokenMetrics, error) {
	if tokenType == "" {
		tokenType = "NRN" // Default to NRN token
	}

	if ei.localMode && ei.economicsService != nil {
		// For local mode, return simulated metrics
		return &TokenMetrics{
			TokenID:         tokenType,
			Price:           big.NewInt(1000000),       // $1.00 in micro-units
			MarketCap:       big.NewInt(1000000000000), // $1B market cap
			Volume24h:       big.NewInt(10000000000),   // $10M volume
			PriceChange24h:  2.5,                       // 2.5% increase
			VolumeChange24h: -5.2,                      // 5.2% decrease
			Holders:         10000,
			Transactions24h: 5000,
			Liquidity:       big.NewInt(50000000000), // $50M liquidity
			LastUpdated:     time.Now(),
		}, nil
	}

	// For remote mode, make HTTP request
	url := fmt.Sprintf("%s/api/tokens/%s/metrics", ei.economicsURL, tokenType)
	resp, err := ei.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get token metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("economics service returned status %d", resp.StatusCode)
	}

	var response struct {
		Success bool          `json:"success"`
		Data    *TokenMetrics `json:"data"`
		Error   string        `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("economics service error: %s", response.Error)
	}

	if response.Data == nil {
		return &TokenMetrics{
			TokenID:     tokenType,
			LastUpdated: time.Now(),
		}, nil
	}

	return response.Data, nil
}

func (ei *EconomicsIntegrationImpl) GetMarketData() (*MarketData, error) {
	if ei.localMode && ei.economicsService != nil {
		// For local mode, return simulated market data
		return &MarketData{
			Timestamp:       time.Now(),
			TotalMarketCap:  big.NewInt(5000000000000), // $5T total market cap
			TotalVolume:     big.NewInt(100000000000),  // $100B total volume
			DominanceIndex:  0.45,                      // 45% dominance
			VolatilityIndex: 0.25,                      // 25% volatility
			TokenPrices: map[string]*big.Int{
				"NRN": big.NewInt(1000000),     // $1.00
				"BTC": big.NewInt(50000000000), // $50,000
				"ETH": big.NewInt(3000000000),  // $3,000
			},
			ExchangeRates: map[string]float64{
				"USD/EUR": 0.85,
				"USD/GBP": 0.75,
				"USD/JPY": 110.0,
			},
			TrendIndicators: map[string]*TrendIndicator{
				"NRN": {
					Name:      "NRN",
					Value:     1.025, // 2.5% increase
					Direction: "up",
					Strength:  0.7,
					Timestamp: time.Now(),
				},
			},
		}, nil
	}

	// For remote mode, make HTTP request
	url := fmt.Sprintf("%s/api/market/data", ei.economicsURL)
	resp, err := ei.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get market data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("economics service returned status %d", resp.StatusCode)
	}

	var response struct {
		Success bool        `json:"success"`
		Data    *MarketData `json:"data"`
		Error   string      `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("economics service error: %s", response.Error)
	}

	if response.Data == nil {
		return &MarketData{
			Timestamp: time.Now(),
		}, nil
	}

	return response.Data, nil
}

func (ei *EconomicsIntegrationImpl) Start(ctx context.Context) error {
	if !ei.enabled {
		return fmt.Errorf("economics integration is disabled")
	}

	// Test connection
	if err := ei.testConnection(); err != nil {
		log.Printf("Warning: Economics service connection test failed: %v", err)
		// Don't fail startup, just log the warning
	}

	// Start local economics service if in local mode
	if ei.localMode && ei.economicsService != nil {
		go func() {
			if err := ei.economicsService.Start(); err != nil {
				log.Printf("Error starting local economics service: %v", err)
			}
		}()
	}

	log.Println("Economics integration started")
	return nil
}

func (ei *EconomicsIntegrationImpl) Stop() error {
	if ei.localMode && ei.economicsService != nil {
		if err := ei.economicsService.Stop(); err != nil {
			return fmt.Errorf("failed to stop local economics service: %w", err)
		}
	}

	ei.enabled = false
	log.Println("Economics integration stopped")
	return nil
}
