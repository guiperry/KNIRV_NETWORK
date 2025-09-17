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

	"KNIRVORACLE/economics"
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
		return &EconomicMetrics{}, nil
	}

	// TODO: Convert EconomicsResponse to EconomicMetrics
	_, err := ei.makeRequest("GET", "/economics/metrics", nil)
	if err != nil {
		return nil, err
	}
	return &EconomicMetrics{}, nil
}

// GetServiceMetrics retrieves KNIRVORACLE-specific metrics from the economics service
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

// Integration helper functions for existing KNIRVORACLE code

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
	// TODO: Implement proper initialization
	return nil
}

func (ei *EconomicsIntegrationImpl) Connect() error {
	// TODO: Implement connection logic
	return nil
}

func (ei *EconomicsIntegrationImpl) Disconnect() error {
	// TODO: Implement disconnection logic
	return nil
}

func (ei *EconomicsIntegrationImpl) IsConnected() bool {
	// TODO: Implement connection status check
	return ei.enabled
}

func (ei *EconomicsIntegrationImpl) GetTokenBalance(address string, tokenType string) (*big.Int, error) {
	// TODO: Implement token balance retrieval
	return big.NewInt(0), nil
}

func (ei *EconomicsIntegrationImpl) TransferTokens(from, to string, amount *big.Int, tokenType string) (*TransferResult, error) {
	// TODO: Implement token transfer
	return &TransferResult{}, nil
}

func (ei *EconomicsIntegrationImpl) MintTokens(to string, amount *big.Int, tokenType string) (*MintResult, error) {
	// TODO: Implement token minting
	return &MintResult{}, nil
}

func (ei *EconomicsIntegrationImpl) BurnTokens(from string, amount *big.Int, tokenType string) (*BurnResult, error) {
	// TODO: Implement token burning
	return &BurnResult{}, nil
}

func (ei *EconomicsIntegrationImpl) GetTokenMetrics(tokenType string) (*TokenMetrics, error) {
	// TODO: Implement token metrics retrieval
	return &TokenMetrics{}, nil
}

func (ei *EconomicsIntegrationImpl) GetMarketData() (*MarketData, error) {
	// TODO: Implement market data retrieval
	return &MarketData{}, nil
}

func (ei *EconomicsIntegrationImpl) Start(ctx context.Context) error {
	// TODO: Implement service start
	return nil
}

func (ei *EconomicsIntegrationImpl) Stop() error {
	// TODO: Implement service stop
	return nil
}
