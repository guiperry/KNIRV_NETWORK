package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/guiperry/KNIRVCHAIN-CLI/config"
	"github.com/sirupsen/logrus"
)

// NRNTokenManager manages NRN token operations
type NRNTokenManager struct {
	config          *config.WalletConfig
	logger          *logrus.Logger
	balance         *big.Int
	transactions    []*NRNTransaction
	faucetClient    *FaucetClient
	knirvRootClient *KNIRVRootClient
}

// NRNTransaction represents an NRN token transaction
type NRNTransaction struct {
	ID          string                 `json:"id"`
	Hash        string                 `json:"hash"`
	From        string                 `json:"from"`
	To          string                 `json:"to"`
	Amount      *big.Int               `json:"amount"`
	Type        string                 `json:"type"` // transfer, faucet, burn, stake
	Status      string                 `json:"status"`
	BlockHeight int64                  `json:"block_height"`
	Timestamp   time.Time              `json:"timestamp"`
	GasFee      *big.Int               `json:"gas_fee"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// FaucetClient handles NRV faucet operations for testnet
type FaucetClient struct {
	*APIClient
	faucetURL string
	logger    *logrus.Logger
}

// FaucetRequest represents a testnet faucet request
type FaucetRequest struct {
	Address string `json:"address"`
	Amount  int    `json:"amount"`
	Reason  string `json:"reason,omitempty"`
}

// FaucetResponse represents a testnet faucet response
type FaucetResponse struct {
	Success          bool   `json:"success"`
	RequestID        string `json:"request_id,omitempty"`
	Address          string `json:"address,omitempty"`
	Amount           int    `json:"amount,omitempty"`
	TxHash           string `json:"tx_hash,omitempty"`
	Timestamp        string `json:"timestamp,omitempty"`
	EstimatedConfirm string `json:"estimated_confirmation,omitempty"`
	Error            string `json:"error,omitempty"`
	Code             string `json:"code,omitempty"`
	RetryAfter       int    `json:"retry_after,omitempty"`
	LimitType        string `json:"limit_type,omitempty"`
}

// FaucetStatus represents the current faucet status
type FaucetStatus struct {
	FaucetEnabled    bool                   `json:"faucet_enabled"`
	CurrentBalance   int                    `json:"current_balance"`
	DailyLimit       int                    `json:"daily_limit"`
	RemainingToday   int                    `json:"remaining_today"`
	CurrentQueueSize int                    `json:"current_queue_size"`
	SuccessRateToday float64                `json:"success_rate_today"`
	RateLimits       map[string]interface{} `json:"rate_limits"`
	SupportedAmounts map[string]interface{} `json:"supported_amounts"`
	LastFunding      string                 `json:"last_funding,omitempty"`
	NextFundingEst   string                 `json:"next_funding_estimate,omitempty"`
}

// FaucetHistoryEntry represents a single faucet request in history
type FaucetHistoryEntry struct {
	RequestID string `json:"request_id"`
	Amount    int    `json:"amount"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	TxHash    string `json:"tx_hash,omitempty"`
	Error     string `json:"error,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// FaucetHistory represents the request history for an address
type FaucetHistory struct {
	Address       string               `json:"address"`
	TotalRequests int                  `json:"total_requests"`
	TotalAmount   int                  `json:"total_amount"`
	History       []FaucetHistoryEntry `json:"history"`
}

// NewNRNTokenManager creates a new NRN token manager
func NewNRNTokenManager(cfg *config.WalletConfig, knirvRootClient *KNIRVRootClient, logger *logrus.Logger) *NRNTokenManager {
	faucetClient := &FaucetClient{
		APIClient: NewAPIClient(cfg.NRN.FaucetURL, WithLogger(logger)),
		faucetURL: cfg.NRN.FaucetURL,
		logger:    logger,
	}

	return &NRNTokenManager{
		config:          cfg,
		logger:          logger,
		balance:         big.NewInt(0),
		transactions:    make([]*NRNTransaction, 0),
		faucetClient:    faucetClient,
		knirvRootClient: knirvRootClient,
	}
}

// GetBalance returns the current NRN balance
func (ntm *NRNTokenManager) GetBalance() *big.Int {
	return new(big.Int).Set(ntm.balance)
}

// UpdateBalance updates the NRN balance from the network
func (ntm *NRNTokenManager) UpdateBalance(ctx context.Context, address string) error {
	ntm.logger.Debugf("Updating NRN balance for address: %s", address)

	// Get balance from KNIRVORACLE
	balanceStr, err := ntm.knirvRootClient.GetNRNBalance(ctx, address)
	if err != nil {
		return fmt.Errorf("failed to get NRN balance: %w", err)
	}

	// Parse balance
	balance, ok := new(big.Int).SetString(balanceStr, 10)
	if !ok {
		return fmt.Errorf("invalid balance format: %s", balanceStr)
	}

	ntm.balance = balance
	ntm.logger.Infof("NRN balance updated: %s", balance.String())

	// Check if auto-refill is needed
	if ntm.config.NRN.AutoRefill {
		if err := ntm.checkAndRefill(ctx, address); err != nil {
			ntm.logger.Errorf("Auto-refill failed: %v", err)
		}
	}

	return nil
}

// Transfer transfers NRN tokens to another address
func (ntm *NRNTokenManager) Transfer(ctx context.Context, from, to string, amount *big.Int) (*NRNTransaction, error) {
	ntm.logger.Infof("Transferring %s NRN from %s to %s", amount.String(), from, to)

	// Check balance
	if ntm.balance.Cmp(amount) < 0 {
		return nil, fmt.Errorf("insufficient balance: have %s, need %s", ntm.balance.String(), amount.String())
	}

	// Create transaction
	tx := &NRNTransaction{
		ID:        generateTransactionID(),
		From:      from,
		To:        to,
		Amount:    new(big.Int).Set(amount),
		Type:      "transfer",
		Status:    "pending",
		Timestamp: time.Now(),
		GasFee:    big.NewInt(0), // NRN transfers might be gasless
		Metadata:  make(map[string]interface{}),
	}

	// TODO: Implement actual NRN transfer via KNIRVORACLE
	// For now, we'll simulate the transfer
	tx.Hash = fmt.Sprintf("0x%x", generateTransactionHash(tx))
	tx.Status = "confirmed"
	tx.BlockHeight = 12345 // Mock block height

	// Update local balance
	ntm.balance.Sub(ntm.balance, amount)

	// Add to transaction history
	ntm.transactions = append(ntm.transactions, tx)

	ntm.logger.Infof("NRN transfer completed: %s", tx.Hash)
	return tx, nil
}

// RequestFromFaucet requests NRV tokens from the testnet faucet
func (ntm *NRNTokenManager) RequestFromFaucet(ctx context.Context, address string, amount string) (*NRNTransaction, error) {
	if !ntm.config.NRN.Enabled {
		return nil, fmt.Errorf("NRN is disabled")
	}

	ntm.logger.Infof("Requesting %s NRV from testnet faucet for address: %s", amount, address)

	// Validate address format
	if err := ntm.ValidateAddress(address); err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	// Use new testnet faucet with retry logic
	return ntm.RequestFromFaucetWithRetry(ctx, address, amount, "CLI request", 3)
}

// GetTransactionHistory returns the NRN transaction history
func (ntm *NRNTokenManager) GetTransactionHistory() []*NRNTransaction {
	// Return a copy of the transactions
	history := make([]*NRNTransaction, len(ntm.transactions))
	copy(history, ntm.transactions)
	return history
}

// GetTransaction returns a specific transaction by ID
func (ntm *NRNTokenManager) GetTransaction(id string) (*NRNTransaction, error) {
	for _, tx := range ntm.transactions {
		if tx.ID == id {
			return tx, nil
		}
	}
	return nil, fmt.Errorf("transaction not found: %s", id)
}

// EstimateFee estimates the fee for an NRN transaction
func (ntm *NRNTokenManager) EstimateFee(ctx context.Context, txType string, amount *big.Int) (*big.Int, error) {
	ntm.logger.Debugf("Estimating fee for %s transaction, amount: %s", txType, amount.String())

	// TODO: Implement actual fee estimation
	// For now, return zero fee (gasless transactions)
	fee := big.NewInt(0)

	ntm.logger.Debugf("Estimated fee: %s", fee.String())
	return fee, nil
}

// checkAndRefill checks if balance is below minimum and refills if needed
func (ntm *NRNTokenManager) checkAndRefill(ctx context.Context, address string) error {
	minBalance, ok := new(big.Int).SetString(ntm.config.NRN.MinBalance, 10)
	if !ok {
		return fmt.Errorf("invalid min balance format: %s", ntm.config.NRN.MinBalance)
	}

	if ntm.balance.Cmp(minBalance) < 0 {
		ntm.logger.Infof("Balance below minimum (%s), requesting refill", minBalance.String())

		// Calculate refill amount (2x minimum balance)
		refillAmount := new(big.Int).Mul(minBalance, big.NewInt(2))

		_, err := ntm.RequestFromFaucet(ctx, address, refillAmount.String())
		if err != nil {
			return fmt.Errorf("auto-refill failed: %w", err)
		}

		ntm.logger.Infof("Auto-refill completed: %s NRN", refillAmount.String())
	}

	return nil
}

// Burn burns NRN tokens (removes them from circulation)
func (ntm *NRNTokenManager) Burn(ctx context.Context, address string, amount *big.Int) (*NRNTransaction, error) {
	ntm.logger.Infof("Burning %s NRN from address: %s", amount.String(), address)

	// Check balance
	if ntm.balance.Cmp(amount) < 0 {
		return nil, fmt.Errorf("insufficient balance for burn: have %s, need %s", ntm.balance.String(), amount.String())
	}

	// Create burn transaction
	tx := &NRNTransaction{
		ID:        generateTransactionID(),
		From:      address,
		To:        "0x0000000000000000000000000000000000000000", // Burn address
		Amount:    new(big.Int).Set(amount),
		Type:      "burn",
		Status:    "confirmed",
		Timestamp: time.Now(),
		GasFee:    big.NewInt(0),
		Hash:      fmt.Sprintf("0x%x", generateTransactionHash(nil)),
		Metadata:  map[string]interface{}{"operation": "burn"},
	}

	// Update local balance
	ntm.balance.Sub(ntm.balance, amount)

	// Add to transaction history
	ntm.transactions = append(ntm.transactions, tx)

	ntm.logger.Infof("NRN burn completed: %s tokens burned", amount.String())
	return tx, nil
}

// GetNRNStats returns NRN token statistics
func (ntm *NRNTokenManager) GetNRNStats() map[string]interface{} {
	totalTransferred := big.NewInt(0)
	totalReceived := big.NewInt(0)
	totalBurned := big.NewInt(0)

	for _, tx := range ntm.transactions {
		switch tx.Type {
		case "transfer":
			totalTransferred.Add(totalTransferred, tx.Amount)
		case "faucet":
			totalReceived.Add(totalReceived, tx.Amount)
		case "burn":
			totalBurned.Add(totalBurned, tx.Amount)
		}
	}

	return map[string]interface{}{
		"current_balance":     ntm.balance.String(),
		"total_transactions":  len(ntm.transactions),
		"total_transferred":   totalTransferred.String(),
		"total_received":      totalReceived.String(),
		"total_burned":        totalBurned.String(),
		"auto_refill_enabled": ntm.config.NRN.AutoRefill,
		"min_balance":         ntm.config.NRN.MinBalance,
	}
}

// Helper functions

func generateTransactionID() string {
	return fmt.Sprintf("nrn_%d", time.Now().UnixNano())
}

func generateTransactionHash(tx *NRNTransaction) []byte {
	// Simple hash generation for demo purposes
	data := fmt.Sprintf("%d", time.Now().UnixNano())
	return []byte(data)
}

// Enhanced Testnet Faucet Client Methods

// RequestTokens requests NRV tokens from the testnet faucet
func (fc *FaucetClient) RequestTokens(ctx context.Context, address string, amount int, reason string) (*FaucetResponse, error) {
	fc.logger.Debugf("Requesting %d NRV tokens for address: %s", amount, address)

	request := FaucetRequest{
		Address: address,
		Amount:  amount,
		Reason:  reason,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fc.faucetURL+"/api/faucet/request", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KNIRV-CLI-Go/1.0.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var response FaucetResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	fc.logger.Debugf("Faucet response: success=%v, request_id=%s", response.Success, response.RequestID)
	return &response, nil
}

// GetStatus retrieves the current faucet status
func (fc *FaucetClient) GetStatus(ctx context.Context) (*FaucetStatus, error) {
	fc.logger.Debug("Getting faucet status")

	req, err := http.NewRequestWithContext(ctx, "GET", fc.faucetURL+"/api/faucet/status", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "KNIRV-CLI-Go/1.0.0")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var status FaucetStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	fc.logger.Debugf("Faucet status: enabled=%v, balance=%d", status.FaucetEnabled, status.CurrentBalance)
	return &status, nil
}

// GetHistory retrieves the request history for an address
func (fc *FaucetClient) GetHistory(ctx context.Context, address string, limit int) (*FaucetHistory, error) {
	fc.logger.Debugf("Getting faucet history for address: %s (limit: %d)", address, limit)

	url := fmt.Sprintf("%s/api/faucet/history?address=%s&limit=%d", fc.faucetURL, address, limit)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "KNIRV-CLI-Go/1.0.0")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var history FaucetHistory
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	fc.logger.Debugf("Retrieved %d history entries for address: %s", len(history.History), address)
	return &history, nil
}

// CheckHealth checks the faucet service health
func (fc *FaucetClient) CheckHealth(ctx context.Context) (map[string]interface{}, error) {
	fc.logger.Debug("Checking faucet health")

	req, err := http.NewRequestWithContext(ctx, "GET", fc.faucetURL+"/api/faucet/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "KNIRV-CLI-Go/1.0.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	fc.logger.Debugf("Faucet health status: %v", health["status"])
	return health, nil
}

// Enhanced NRNTokenManager Methods for Testnet Faucet Integration

// GetFaucetStatus returns the current testnet faucet status
func (ntm *NRNTokenManager) GetFaucetStatus(ctx context.Context) (*FaucetStatus, error) {
	return ntm.faucetClient.GetStatus(ctx)
}

// GetFaucetHistory returns the faucet request history for an address
func (ntm *NRNTokenManager) GetFaucetHistory(ctx context.Context, address string, limit int) (*FaucetHistory, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	return ntm.faucetClient.GetHistory(ctx, address, limit)
}

// CheckFaucetHealth checks the testnet faucet service health
func (ntm *NRNTokenManager) CheckFaucetHealth(ctx context.Context) (map[string]interface{}, error) {
	return ntm.faucetClient.CheckHealth(ctx)
}

// RequestFromFaucetWithRetry requests tokens with automatic retry logic
func (ntm *NRNTokenManager) RequestFromFaucetWithRetry(ctx context.Context, address string, amount string, reason string, maxRetries int) (*NRNTransaction, error) {
	amountInt, err := strconv.Atoi(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount format: %s", amount)
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		ntm.logger.Infof("Faucet request attempt %d/%d", attempt, maxRetries)

		response, err := ntm.faucetClient.RequestTokens(ctx, address, amountInt, reason)
		if err != nil {
			lastErr = err
			ntm.logger.Warnf("Attempt %d failed: %v", attempt, err)

			if attempt < maxRetries {
				// Wait before retry (exponential backoff)
				waitTime := time.Duration(attempt*attempt) * time.Second
				ntm.logger.Infof("Waiting %v before retry...", waitTime)
				time.Sleep(waitTime)
			}
			continue
		}

		if !response.Success {
			lastErr = fmt.Errorf("faucet request failed: %s", response.Error)

			// Check if it's a rate limit error
			if response.Code == "RATE_LIMITED" && response.RetryAfter > 0 {
				if attempt < maxRetries {
					waitTime := time.Duration(response.RetryAfter) * time.Second
					ntm.logger.Infof("Rate limited. Waiting %v before retry...", waitTime)
					time.Sleep(waitTime)
					continue
				}
			}

			ntm.logger.Warnf("Attempt %d failed: %s", attempt, response.Error)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt*2) * time.Second)
			}
			continue
		}

		// Success - create transaction record
		amountBig, _ := new(big.Int).SetString(amount, 10)
		tx := &NRNTransaction{
			ID:        response.RequestID,
			From:      "testnet-faucet",
			To:        address,
			Amount:    amountBig,
			Type:      "faucet",
			Status:    "confirmed",
			Timestamp: time.Now(),
			GasFee:    big.NewInt(0),
			Hash:      response.TxHash,
			Metadata: map[string]interface{}{
				"source":     "testnet-faucet",
				"request_id": response.RequestID,
				"faucet_url": ntm.faucetClient.faucetURL,
				"attempts":   attempt,
				"reason":     reason,
			},
		}

		// Update local balance
		ntm.balance.Add(ntm.balance, amountBig)
		ntm.transactions = append(ntm.transactions, tx)

		ntm.logger.Infof("Faucet request successful after %d attempts: %s NRV received (TX: %s)", attempt, amount, response.TxHash)
		return tx, nil
	}

	return nil, fmt.Errorf("faucet request failed after %d attempts: %w", maxRetries, lastErr)
}

// ValidateAddress validates a KNIRV address format
func (ntm *NRNTokenManager) ValidateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	if !strings.HasPrefix(address, "knirv1") {
		return fmt.Errorf("address must start with 'knirv1'")
	}

	if len(address) < 20 {
		return fmt.Errorf("address too short (minimum 20 characters)")
	}

	// Additional validation can be added here
	return nil
}
