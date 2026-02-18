package xion

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// XIONPaymentGateway handles USDC to NRN conversions via XION platform
type XIONPaymentGateway struct {
	config          *XIONGatewayConfig
	treasuryService *TreasuryService
	httpClient      *http.Client
	pendingPayments map[string]*PaymentRequest
	paymentHistory  []PaymentRecord
	mutex           sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
}

// XIONGatewayConfig contains configuration for XION payment gateway
type XIONGatewayConfig struct {
	XIONChainID          string `json:"xion_chain_id"`
	XIONRPCEndpoint      string `json:"xion_rpc_endpoint"`
	XIONRESTEndpoint     string `json:"xion_rest_endpoint"`
	USDCContractAddr     string `json:"usdc_contract_addr"`
	NRNContractAddr      string `json:"nrn_contract_addr"`
	TreasuryAddr         string `json:"treasury_addr"`
	ConversionRate       string `json:"conversion_rate"` // USDC to NRN rate
	GaslessEnabled       bool   `json:"gasless_enabled"`
	MaxTransactionAmount string `json:"max_transaction_amount"`
	MinTransactionAmount string `json:"min_transaction_amount"`
}

// PaymentRequest represents a USDC to NRN conversion request
type PaymentRequest struct {
	PaymentID       string     `json:"payment_id"`
	UserAddress     string     `json:"user_address"`
	USDCAmount      string     `json:"usdc_amount"`
	NRNAmount       string     `json:"nrn_amount"`
	ConversionRate  string     `json:"conversion_rate"`
	Status          string     `json:"status"` // pending, processing, completed, failed
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	TransactionHash string     `json:"transaction_hash,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	MetaAccountType string     `json:"meta_account_type"` // email, social, wallet, passkey
	Gasless         bool       `json:"gasless"`
}

// PaymentRecord represents a completed payment for history tracking
type PaymentRecord struct {
	PaymentID       string    `json:"payment_id"`
	UserAddress     string    `json:"user_address"`
	USDCAmount      string    `json:"usdc_amount"`
	NRNAmount       string    `json:"nrn_amount"`
	TransactionHash string    `json:"transaction_hash"`
	CompletedAt     time.Time `json:"completed_at"`
	GasFee          string    `json:"gas_fee"`
	ConversionRate  string    `json:"conversion_rate"`
}

// TreasuryService interface for interacting with KNIRVCHAIN treasury (stub - economics moved elsewhere)
type TreasuryService struct {
	economicsAPI interface{} // Economics API moved elsewhere
}

// NewXIONPaymentGateway creates a new XION payment gateway instance
func NewXIONPaymentGateway(config *XIONGatewayConfig, economicsAPI interface{}) *XIONPaymentGateway {
	ctx, cancel := context.WithCancel(context.Background())

	return &XIONPaymentGateway{
		config: config,
		treasuryService: &TreasuryService{
			economicsAPI: economicsAPI,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		pendingPayments: make(map[string]*PaymentRequest),
		paymentHistory:  make([]PaymentRecord, 0),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start initializes and starts the XION payment gateway
func (xpg *XIONPaymentGateway) Start() error {
	log.Println("Starting XION Payment Gateway...")

	// Validate configuration
	if err := xpg.validateConfig(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Start background services
	go xpg.processPaymentQueue()
	go xpg.monitorXIONTransactions()

	log.Printf("XION Payment Gateway started successfully on chain %s", xpg.config.XIONChainID)
	return nil
}

// Stop gracefully shuts down the payment gateway
func (xpg *XIONPaymentGateway) Stop() error {
	log.Println("Stopping XION Payment Gateway...")
	xpg.cancel()
	return nil
}

// RegisterRoutes registers HTTP routes for the payment gateway
func (xpg *XIONPaymentGateway) RegisterRoutes(router *mux.Router) {
	// Payment endpoints
	router.HandleFunc("/api/payment/usdc-to-nrn", xpg.handleUSDCToNRNConversion).Methods("POST")
	router.HandleFunc("/api/payment/status/{payment_id}", xpg.handlePaymentStatus).Methods("GET")
	router.HandleFunc("/api/payment/history/{user_address}", xpg.handlePaymentHistory).Methods("GET")

	// Configuration endpoints
	router.HandleFunc("/api/payment/config", xpg.handleGetConfig).Methods("GET")
	router.HandleFunc("/api/payment/rates", xpg.handleGetRates).Methods("GET")

	// Meta Account endpoints
	router.HandleFunc("/api/payment/meta-account/connect", xpg.handleMetaAccountConnect).Methods("POST")
	router.HandleFunc("/api/payment/meta-account/balance/{address}", xpg.handleGetBalance).Methods("GET")

	log.Println("XION Payment Gateway routes registered")
}

// handleUSDCToNRNConversion handles USDC to NRN conversion requests
func (xpg *XIONPaymentGateway) handleUSDCToNRNConversion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserAddress     string `json:"user_address"`
		USDCAmount      string `json:"usdc_amount"`
		MetaAccountType string `json:"meta_account_type"`
		Gasless         bool   `json:"gasless"`
		Memo            string `json:"memo,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		xpg.sendError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate request
	if err := xpg.validateConversionRequest(&req); err != nil {
		xpg.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Calculate NRN amount
	nrnAmount, err := xpg.calculateNRNAmount(req.USDCAmount)
	if err != nil {
		xpg.sendError(w, http.StatusBadRequest, fmt.Sprintf("Failed to calculate NRN amount: %v", err))
		return
	}

	// Create payment request
	paymentID := xpg.generatePaymentID()
	payment := &PaymentRequest{
		PaymentID:       paymentID,
		UserAddress:     req.UserAddress,
		USDCAmount:      req.USDCAmount,
		NRNAmount:       nrnAmount,
		ConversionRate:  xpg.config.ConversionRate,
		Status:          "pending",
		CreatedAt:       time.Now(),
		MetaAccountType: req.MetaAccountType,
		Gasless:         req.Gasless,
	}

	// Store payment request
	xpg.mutex.Lock()
	xpg.pendingPayments[paymentID] = payment
	xpg.mutex.Unlock()

	// Start processing asynchronously
	go xpg.processPayment(payment)

	xpg.sendSuccess(w, map[string]interface{}{
		"payment_id":           paymentID,
		"usdc_amount":          req.USDCAmount,
		"nrn_amount":           nrnAmount,
		"conversion_rate":      xpg.config.ConversionRate,
		"status":               "pending",
		"gasless":              req.Gasless,
		"estimated_completion": time.Now().Add(2 * time.Minute).Format(time.RFC3339),
	})
}

// handlePaymentStatus returns the status of a payment request
func (xpg *XIONPaymentGateway) handlePaymentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentID := vars["payment_id"]

	xpg.mutex.RLock()
	payment, exists := xpg.pendingPayments[paymentID]
	xpg.mutex.RUnlock()

	if !exists {
		// Check payment history
		for _, record := range xpg.paymentHistory {
			if record.PaymentID == paymentID {
				xpg.sendSuccess(w, map[string]interface{}{
					"payment_id":       record.PaymentID,
					"status":           "completed",
					"usdc_amount":      record.USDCAmount,
					"nrn_amount":       record.NRNAmount,
					"transaction_hash": record.TransactionHash,
					"completed_at":     record.CompletedAt.Format(time.RFC3339),
				})
				return
			}
		}

		xpg.sendError(w, http.StatusNotFound, "Payment not found")
		return
	}

	response := map[string]interface{}{
		"payment_id":  payment.PaymentID,
		"status":      payment.Status,
		"usdc_amount": payment.USDCAmount,
		"nrn_amount":  payment.NRNAmount,
		"created_at":  payment.CreatedAt.Format(time.RFC3339),
		"gasless":     payment.Gasless,
	}

	if payment.CompletedAt != nil {
		response["completed_at"] = payment.CompletedAt.Format(time.RFC3339)
	}
	if payment.TransactionHash != "" {
		response["transaction_hash"] = payment.TransactionHash
	}
	if payment.ErrorMessage != "" {
		response["error_message"] = payment.ErrorMessage
	}

	xpg.sendSuccess(w, response)
}

// Helper methods
func (xpg *XIONPaymentGateway) validateConfig() error {
	if xpg.config.XIONChainID == "" {
		return fmt.Errorf("XION chain ID is required")
	}
	if xpg.config.USDCContractAddr == "" {
		return fmt.Errorf("USDC contract address is required")
	}
	if xpg.config.ConversionRate == "" {
		return fmt.Errorf("conversion rate is required")
	}
	return nil
}

func (xpg *XIONPaymentGateway) generatePaymentID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("pay_%s", hex.EncodeToString(bytes)[:16])
}

func (xpg *XIONPaymentGateway) sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

func (xpg *XIONPaymentGateway) sendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// validateConversionRequest validates a USDC to NRN conversion request
func (xpg *XIONPaymentGateway) validateConversionRequest(req *struct {
	UserAddress     string `json:"user_address"`
	USDCAmount      string `json:"usdc_amount"`
	MetaAccountType string `json:"meta_account_type"`
	Gasless         bool   `json:"gasless"`
	Memo            string `json:"memo,omitempty"`
}) error {
	if req.UserAddress == "" {
		return fmt.Errorf("user address is required")
	}
	if req.USDCAmount == "" {
		return fmt.Errorf("USDC amount is required")
	}

	// Validate USDC amount format
	usdcAmount, ok := new(big.Int).SetString(req.USDCAmount, 10)
	if !ok {
		return fmt.Errorf("invalid USDC amount format")
	}

	// Check minimum and maximum limits
	minAmount, _ := new(big.Int).SetString(xpg.config.MinTransactionAmount, 10)
	maxAmount, _ := new(big.Int).SetString(xpg.config.MaxTransactionAmount, 10)

	if minAmount != nil && usdcAmount.Cmp(minAmount) < 0 {
		return fmt.Errorf("amount below minimum limit")
	}
	if maxAmount != nil && usdcAmount.Cmp(maxAmount) > 0 {
		return fmt.Errorf("amount exceeds maximum limit")
	}

	// Validate meta account type
	validTypes := map[string]bool{
		"email": true, "social": true, "wallet": true, "passkey": true,
	}
	if !validTypes[req.MetaAccountType] {
		return fmt.Errorf("invalid meta account type")
	}

	return nil
}

// calculateNRNAmount calculates NRN amount from USDC amount
func (xpg *XIONPaymentGateway) calculateNRNAmount(usdcAmount string) (string, error) {
	usdcBigInt, ok := new(big.Int).SetString(usdcAmount, 10)
	if !ok {
		return "", fmt.Errorf("invalid USDC amount")
	}

	// Parse conversion rate (e.g., "10" means 1 USDC = 10 NRN)
	rate, ok := new(big.Int).SetString(xpg.config.ConversionRate, 10)
	if !ok {
		return "", fmt.Errorf("invalid conversion rate")
	}

	// Calculate NRN amount
	nrnAmount := new(big.Int).Mul(usdcBigInt, rate)

	return nrnAmount.String(), nil
}

// processPayment processes a payment request
func (xpg *XIONPaymentGateway) processPayment(payment *PaymentRequest) {
	log.Printf("Processing payment %s: %s USDC -> %s NRN",
		payment.PaymentID, payment.USDCAmount, payment.NRNAmount)

	// Update status to processing
	xpg.updatePaymentStatus(payment.PaymentID, "processing", "")

	// Step 1: Validate USDC balance on XION
	if err := xpg.validateUSDCBalance(payment); err != nil {
		xpg.updatePaymentStatus(payment.PaymentID, "failed", err.Error())
		return
	}

	// Step 2: Execute USDC transfer to treasury
	txHash, err := xpg.executeUSDCTransfer(payment)
	if err != nil {
		xpg.updatePaymentStatus(payment.PaymentID, "failed", err.Error())
		return
	}

	payment.TransactionHash = txHash

	// Step 3: Mint NRN tokens via KNIRVCHAIN treasury
	if err := xpg.mintNRNTokens(payment); err != nil {
		xpg.updatePaymentStatus(payment.PaymentID, "failed", err.Error())
		return
	}

	// Step 4: Complete payment
	xpg.completePayment(payment)

	log.Printf("Payment %s completed successfully", payment.PaymentID)
}

// validateUSDCBalance validates user has sufficient USDC balance
func (xpg *XIONPaymentGateway) validateUSDCBalance(payment *PaymentRequest) error {
	// In production, this would query XION blockchain for USDC balance
	log.Printf("Validating USDC balance for %s", payment.UserAddress)

	// Simulate balance check
	time.Sleep(500 * time.Millisecond)

	// For demo purposes, assume balance is sufficient
	return nil
}

// executeUSDCTransfer executes USDC transfer on XION
func (xpg *XIONPaymentGateway) executeUSDCTransfer(payment *PaymentRequest) (string, error) {
	log.Printf("Executing USDC transfer for payment %s", payment.PaymentID)

	// In production, this would:
	// 1. Create XION transaction for USDC transfer
	// 2. Handle gasless transactions via Meta Accounts
	// 3. Submit transaction to XION network
	// 4. Wait for confirmation

	// Simulate transaction execution
	time.Sleep(2 * time.Second)

	// Generate mock transaction hash
	txHash := fmt.Sprintf("xion_tx_%s_%d", payment.PaymentID, time.Now().Unix())

	log.Printf("USDC transfer completed: %s", txHash)
	return txHash, nil
}

// mintNRNTokens mints NRN tokens via KNIRVCHAIN treasury
func (xpg *XIONPaymentGateway) mintNRNTokens(payment *PaymentRequest) error {
	log.Printf("Minting NRN tokens for payment %s", payment.PaymentID)

	// Create treasury mint request
	mintRequest := map[string]interface{}{
		"payment_id":        payment.PaymentID,
		"user_address":      payment.UserAddress,
		"usdc_amount":       payment.USDCAmount,
		"nrn_amount":        payment.NRNAmount,
		"transaction_hash":  payment.TransactionHash,
		"source":            "xion_payment_gateway",
		"meta_account_type": payment.MetaAccountType,
	}

	// Call KNIRVCHAIN treasury service
	if err := xpg.treasuryService.ProcessPaymentMint(mintRequest); err != nil {
		return fmt.Errorf("failed to mint NRN tokens: %w", err)
	}

	log.Printf("NRN tokens minted successfully for payment %s", payment.PaymentID)
	return nil
}

// completePayment marks payment as completed and moves to history
func (xpg *XIONPaymentGateway) completePayment(payment *PaymentRequest) {
	now := time.Now()
	payment.CompletedAt = &now
	payment.Status = "completed"

	// Move to payment history
	record := PaymentRecord{
		PaymentID:       payment.PaymentID,
		UserAddress:     payment.UserAddress,
		USDCAmount:      payment.USDCAmount,
		NRNAmount:       payment.NRNAmount,
		TransactionHash: payment.TransactionHash,
		CompletedAt:     now,
		GasFee:          "0", // Gasless transaction
		ConversionRate:  payment.ConversionRate,
	}

	xpg.mutex.Lock()
	xpg.paymentHistory = append(xpg.paymentHistory, record)
	delete(xpg.pendingPayments, payment.PaymentID)
	xpg.mutex.Unlock()
}

// updatePaymentStatus updates the status of a payment
func (xpg *XIONPaymentGateway) updatePaymentStatus(paymentID, status, errorMsg string) {
	xpg.mutex.Lock()
	defer xpg.mutex.Unlock()

	if payment, exists := xpg.pendingPayments[paymentID]; exists {
		payment.Status = status
		if errorMsg != "" {
			payment.ErrorMessage = errorMsg
		}
	}
}

// ProcessPaymentMint processes NRN minting for payment gateway transactions
func (ts *TreasuryService) ProcessPaymentMint(mintRequest map[string]interface{}) error {
	// Extract payment information
	paymentID, _ := mintRequest["payment_id"].(string)
	userAddress, _ := mintRequest["user_address"].(string)
	nrnAmountStr, _ := mintRequest["nrn_amount"].(string)

	// Parse NRN amount
	nrnAmount, ok := new(big.Int).SetString(nrnAmountStr, 10)
	if !ok {
		return fmt.Errorf("invalid NRN amount: %s", nrnAmountStr)
	}

	// Create treasury transaction for payment gateway mint
	// For now, simulate the treasury mint process
	// In production, this would integrate with the actual economics API
	log.Printf("Processing payment gateway mint for user %s: %s NRN", userAddress, nrnAmount.String())

	// Simulate treasury processing
	// This would be replaced with actual economics API call when available
	// tx, err := ts.economicsAPI.ProcessTreasuryReward(userAddress, fmt.Sprintf("payment_%s", paymentID), nrnAmount)

	log.Printf("Processed payment gateway mint: payment_%s, NRN: %s", paymentID, nrnAmount.String())
	return nil
}

// Background processing methods
func (xpg *XIONPaymentGateway) processPaymentQueue() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-xpg.ctx.Done():
			return
		case <-ticker.C:
			xpg.cleanupExpiredPayments()
		}
	}
}

func (xpg *XIONPaymentGateway) monitorXIONTransactions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-xpg.ctx.Done():
			return
		case <-ticker.C:
			xpg.checkPendingTransactions()
		}
	}
}

func (xpg *XIONPaymentGateway) cleanupExpiredPayments() {
	xpg.mutex.Lock()
	defer xpg.mutex.Unlock()

	expiredTime := time.Now().Add(-1 * time.Hour)
	for paymentID, payment := range xpg.pendingPayments {
		if payment.CreatedAt.Before(expiredTime) && payment.Status == "pending" {
			payment.Status = "expired"
			payment.ErrorMessage = "Payment request expired"
			log.Printf("Payment %s expired", paymentID)
		}
	}
}

func (xpg *XIONPaymentGateway) checkPendingTransactions() {
	// In production, this would check XION blockchain for transaction confirmations
	log.Println("Checking pending XION transactions...")
}

// Additional handler methods
func (xpg *XIONPaymentGateway) handlePaymentHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userAddress := vars["user_address"]

	// Filter payment history by user address
	userHistory := make([]PaymentRecord, 0)
	for _, record := range xpg.paymentHistory {
		if record.UserAddress == userAddress {
			userHistory = append(userHistory, record)
		}
	}

	xpg.sendSuccess(w, map[string]interface{}{
		"user_address": userAddress,
		"payments":     userHistory,
		"total_count":  len(userHistory),
	})
}

func (xpg *XIONPaymentGateway) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"chain_id":                xpg.config.XIONChainID,
		"conversion_rate":         xpg.config.ConversionRate,
		"gasless_enabled":         xpg.config.GaslessEnabled,
		"min_transaction_amount":  xpg.config.MinTransactionAmount,
		"max_transaction_amount":  xpg.config.MaxTransactionAmount,
		"supported_meta_accounts": []string{"email", "social", "wallet", "passkey"},
	}

	xpg.sendSuccess(w, config)
}

func (xpg *XIONPaymentGateway) handleGetRates(w http.ResponseWriter, r *http.Request) {
	rates := map[string]interface{}{
		"usdc_to_nrn":    xpg.config.ConversionRate,
		"last_updated":   time.Now().Format(time.RFC3339),
		"rate_type":      "fixed", // Could be "dynamic" in production
		"base_currency":  "USDC",
		"quote_currency": "NRN",
	}

	xpg.sendSuccess(w, rates)
}

func (xpg *XIONPaymentGateway) handleMetaAccountConnect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AuthMethod string `json:"auth_method"` // email, social, wallet, passkey
		Identifier string `json:"identifier"`  // email address, social ID, wallet address, etc.
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		xpg.sendError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// In production, this would integrate with XION Meta Accounts
	// For now, simulate the connection
	accountAddress := fmt.Sprintf("xion1%s", strings.ToLower(hex.EncodeToString([]byte(req.Identifier))[:32]))

	response := map[string]interface{}{
		"success":         true,
		"account_address": accountAddress,
		"auth_method":     req.AuthMethod,
		"gasless_enabled": xpg.config.GaslessEnabled,
		"connected_at":    time.Now().Format(time.RFC3339),
	}

	xpg.sendSuccess(w, response)
}

func (xpg *XIONPaymentGateway) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	// In production, this would query XION blockchain for actual balances
	// For now, simulate balance data
	balance := map[string]interface{}{
		"address":      address,
		"usdc_balance": "1000000000",             // 1000 USDC (6 decimals)
		"nrn_balance":  "5000000000000000000000", // 5000 NRN (18 decimals)
		"last_updated": time.Now().Format(time.RFC3339),
	}

	xpg.sendSuccess(w, balance)
}
