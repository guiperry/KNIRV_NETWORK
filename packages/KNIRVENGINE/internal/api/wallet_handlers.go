package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"KNIRVENGINE/desktop-client/internal/services"

	"github.com/gorilla/mux"
)

// WalletAccount represents a wallet account
type WalletAccount struct {
	ID          string    `json:"id"`
	Address     string    `json:"address"`
	Name        string    `json:"name"`
	Balance     string    `json:"balance"`
	NRNBalance  string    `json:"nrnBalance"`
	IsActive    bool      `json:"isActive"`
	KeyringType string    `json:"keyringType"`
	CreatedAt   time.Time `json:"createdAt"`
}

// WalletBalance represents the overall wallet balance
type WalletBalance struct {
	NRNBalance    float64       `json:"nrnBalance"`
	USDValue      float64       `json:"usdValue"`
	Change24h     float64       `json:"change24h"`
	WalletAddress string        `json:"walletAddress"`
	Assets        []CryptoAsset `json:"assets"`
}

// CryptoAsset represents a cryptocurrency asset
type CryptoAsset struct {
	Symbol string  `json:"symbol"`
	Name   string  `json:"name"`
	Price  string  `json:"price"`
	Change float64 `json:"change"`
	Amount string  `json:"amount"`
	Value  string  `json:"value"`
	Icon   string  `json:"icon"`
}

// WalletTransaction represents a wallet transaction
type WalletTransaction struct {
	ID        string `json:"id"`
	Hash      string `json:"hash,omitempty"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    string `json:"amount"`
	Token     string `json:"token,omitempty"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	GasUsed   string `json:"gasUsed,omitempty"`
	Fee       string `json:"fee,omitempty"`
	SkillID   string `json:"skillId,omitempty"`
	NRNAmount string `json:"nrnAmount,omitempty"`
}

// TransactionRequest represents a transaction request
type TransactionRequest struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    string `json:"amount"`
	Token     string `json:"token,omitempty"`
	Memo      string `json:"memo,omitempty"`
	GasLimit  string `json:"gasLimit,omitempty"`
	ChainID   string `json:"chainId,omitempty"`
	SkillID   string `json:"skillId,omitempty"`
	NRNAmount string `json:"nrnAmount,omitempty"`
}

// NRNBalance represents NRN token balance details
type NRNBalance struct {
	Available   string `json:"available"`
	Staked      string `json:"staked"`
	Earned      string `json:"earned"`
	TotalSpent  string `json:"totalSpent"`
	LastUpdated int64  `json:"lastUpdated"`
}

// ControllerConnectionStatus represents the connection status with KNIRVCONTROLLER
type ControllerConnectionStatus struct {
	Connected          bool   `json:"connected"`
	ControllerEndpoint string `json:"endpoint,omitempty"`
	WalletLinked       bool   `json:"walletLinked"`
	LastSync           string `json:"lastSync,omitempty"`
	Error              string `json:"error,omitempty"`
}

// WalletHandlers contains all wallet-related HTTP handlers
type WalletHandlers struct {
	// Mock data for demonstration
	mockWallets        []WalletAccount
	mockTransactions   []WalletTransaction
	mockBalance        WalletBalance
	controllerStatus   ControllerConnectionStatus
	knirvoracleService *services.KNIRVOracleService
}

// initializeKNIRVOracleService initializes the KNIRVORACLE service
func initializeKNIRVOracleService() *services.KNIRVOracleService {
	config := services.KNIRVOracleConfig{
		BaseURL: getEnvOrDefault("KNIRVORACLE_URL", "http://localhost:8080"),
		APIKey:  getEnvOrDefault("KNIRVORACLE_API_KEY", ""),
		Timeout: 30,
	}
	return services.NewKNIRVOracleService(config)
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// NewWalletHandlers creates a new wallet handlers instance
func NewWalletHandlers() *WalletHandlers {
	return &WalletHandlers{
		mockWallets: []WalletAccount{
			{
				ID:          "wallet-1",
				Address:     "0x742d35Cc6aa34567...8B9fA2e1C4D",
				Name:        "Main Wallet",
				Balance:     "312.75",
				NRNBalance:  "1247",
				IsActive:    true,
				KeyringType: "hd",
				CreatedAt:   time.Now().Add(-24 * time.Hour),
			},
		},
		mockTransactions: []WalletTransaction{
			{
				ID:        "tx-1",
				Hash:      "0x1234567890abcdef",
				From:      "0x742d35Cc6aa34567...8B9fA2e1C4D",
				To:        "0x987654321fedcba0",
				Amount:    "125",
				Token:     "NRN",
				Status:    "completed",
				Timestamp: time.Now().Add(-2 * time.Hour).Unix(),
				Fee:       "0.001",
			},
			{
				ID:        "tx-2",
				Hash:      "0xabcdef1234567890",
				From:      "0x987654321fedcba0",
				To:        "0x742d35Cc6aa34567...8B9fA2e1C4D",
				Amount:    "0.5",
				Token:     "ETH",
				Status:    "completed",
				Timestamp: time.Now().Add(-24 * time.Hour).Unix(),
				Fee:       "0.002",
			},
		},
		mockBalance: WalletBalance{
			NRNBalance:    1247,
			USDValue:      312.75,
			Change24h:     5.2,
			WalletAddress: "0x742d35Cc6aa34567...8B9fA2e1C4D",
			Assets: []CryptoAsset{
				{
					Symbol: "NRN",
					Name:   "Neural Reasoning Network",
					Price:  "$0.251",
					Change: 5.2,
					Amount: "1247",
					Value:  "$312.75",
					Icon:   "/assets/nrn-icon.png",
				},
				{
					Symbol: "BTC",
					Name:   "Bitcoin",
					Price:  "$47,842.50",
					Change: 3.24,
					Amount: "0.2845",
					Value:  "$13,613.25",
					Icon:   "/assets/btc-icon.png",
				},
				{
					Symbol: "ETH",
					Name:   "Ethereum",
					Price:  "$2,845.32",
					Change: -1.86,
					Amount: "4.2567",
					Value:  "$12,115.89",
					Icon:   "/assets/eth-icon.png",
				},
			},
		},
		controllerStatus: ControllerConnectionStatus{
			Connected:    false,
			WalletLinked: false,
		},
		knirvoracleService: initializeKNIRVOracleService(),
	}
}

// GetWallets returns all wallet accounts
func (wh *WalletHandlers) GetWallets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    wh.mockWallets,
	})
}

// GetWallet returns a specific wallet account
func (wh *WalletHandlers) GetWallet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	walletID := vars["id"]

	for _, wallet := range wh.mockWallets {
		if wallet.ID == walletID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    wallet,
			})
			return
		}
	}

	http.Error(w, "Wallet not found", http.StatusNotFound)
}

// GetBalance returns the wallet balance
func (wh *WalletHandlers) GetBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to get balance from KNIRVORACLE first
	if len(wh.mockWallets) > 0 {
		activeWallet := wh.mockWallets[0] // Get first wallet as active

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		knirvoracleBalance, err := wh.knirvoracleService.GetWalletBalance(ctx, activeWallet.Address)
		if err == nil && knirvoracleBalance.Success {
			// Convert KNIRVORACLE response to our format
			balance := WalletBalance{
				NRNBalance:    parseFloat(knirvoracleBalance.NRNBalance),
				USDValue:      parseFloat(knirvoracleBalance.USDValue),
				Change24h:     wh.mockBalance.Change24h, // Keep mock change for now
				WalletAddress: knirvoracleBalance.Address,
				Assets:        wh.mockBalance.Assets, // Keep mock assets for now
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    balance,
			})
			return
		}

		log.Printf("Failed to get balance from KNIRVORACLE: %v", err)
	}

	// Fallback to mock data
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    wh.mockBalance,
	})
}

// parseFloat safely parses a string to float64
func parseFloat(s string) float64 {
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val
	}
	return 0.0
}

// GetNRNBalance returns the NRN token balance details
func (wh *WalletHandlers) GetNRNBalance(w http.ResponseWriter, r *http.Request) {
	nrnBalance := NRNBalance{
		Available:   "1247",
		Staked:      "500",
		Earned:      "47",
		TotalSpent:  "253",
		LastUpdated: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    nrnBalance,
	})
}

// GetAssets returns the wallet assets
func (wh *WalletHandlers) GetAssets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    wh.mockBalance.Assets,
	})
}

// GetTransactions returns wallet transactions
func (wh *WalletHandlers) GetTransactions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 10 // default
	offset := 0 // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	// Apply pagination
	transactions := wh.mockTransactions
	if offset >= len(transactions) {
		transactions = []WalletTransaction{}
	} else {
		end := offset + limit
		if end > len(transactions) {
			end = len(transactions)
		}
		transactions = transactions[offset:end]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    transactions,
		"total":   len(wh.mockTransactions),
		"limit":   limit,
		"offset":  offset,
	})
}

// SendTransaction processes a transaction request
func (wh *WalletHandlers) SendTransaction(w http.ResponseWriter, r *http.Request) {
	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a new transaction
	newTx := WalletTransaction{
		ID:        fmt.Sprintf("tx-%d", time.Now().Unix()),
		Hash:      fmt.Sprintf("0x%x", time.Now().Unix()),
		From:      req.From,
		To:        req.To,
		Amount:    req.Amount,
		Token:     req.Token,
		Status:    "pending",
		Timestamp: time.Now().Unix(),
		Fee:       "0.001",
		SkillID:   req.SkillID,
		NRNAmount: req.NRNAmount,
	}

	// Add to mock transactions
	wh.mockTransactions = append([]WalletTransaction{newTx}, wh.mockTransactions...)

	// Simulate processing delay
	go func() {
		time.Sleep(5 * time.Second)
		// Update status to completed
		for i, tx := range wh.mockTransactions {
			if tx.ID == newTx.ID {
				wh.mockTransactions[i].Status = "completed"
				break
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    newTx,
	})
}

// CreateWallet creates a new wallet
func (wh *WalletHandlers) CreateWallet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	newWallet := WalletAccount{
		ID:          fmt.Sprintf("wallet-%d", time.Now().Unix()),
		Address:     fmt.Sprintf("0x%x", time.Now().Unix()),
		Name:        req.Name,
		Balance:     "0.00",
		NRNBalance:  "0",
		IsActive:    true,
		KeyringType: "hd",
		CreatedAt:   time.Now(),
	}

	wh.mockWallets = append(wh.mockWallets, newWallet)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    newWallet,
	})
}

// GetControllerStatus returns the current KNIRVCONTROLLER connection status
func (wh *WalletHandlers) GetControllerStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    wh.controllerStatus,
	})
}

// LinkController establishes a connection with KNIRVCONTROLLER
func (wh *WalletHandlers) LinkController(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Endpoint string `json:"endpoint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Endpoint == "" {
		http.Error(w, "Controller endpoint is required", http.StatusBadRequest)
		return
	}

	// Mock controller connection - in production this would make actual HTTP calls
	// to verify the controller is available and establish the link
	wh.controllerStatus = ControllerConnectionStatus{
		Connected:          true,
		ControllerEndpoint: req.Endpoint,
		WalletLinked:       true,
		LastSync:           time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    wh.controllerStatus,
	})
}

// RegisterWalletRoutes registers all wallet-related routes
func (s *SimpleAPIServer) RegisterWalletRoutes() {
	walletHandlers := NewWalletHandlers()

	// Wallet management routes
	s.router.HandleFunc("/api/v1/wallet/accounts", walletHandlers.GetWallets).Methods("GET")
	s.router.HandleFunc("/api/v1/wallet/accounts/{id}", walletHandlers.GetWallet).Methods("GET")
	s.router.HandleFunc("/api/v1/wallet/create", walletHandlers.CreateWallet).Methods("POST")

	// Balance and assets routes
	s.router.HandleFunc("/api/v1/wallet/balance", walletHandlers.GetBalance).Methods("GET")
	s.router.HandleFunc("/api/v1/wallet/nrn-balance", walletHandlers.GetNRNBalance).Methods("GET")
	s.router.HandleFunc("/api/v1/wallet/assets", walletHandlers.GetAssets).Methods("GET")

	// Transaction routes
	s.router.HandleFunc("/api/v1/wallet/transactions", walletHandlers.GetTransactions).Methods("GET")
	s.router.HandleFunc("/api/v1/wallet/send", walletHandlers.SendTransaction).Methods("POST")

	// Controller connection routes
	s.router.HandleFunc("/api/v1/controller/status", walletHandlers.GetControllerStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/controller/link", walletHandlers.LinkController).Methods("POST")

	log.Println("✅ Wallet API routes registered")
}
