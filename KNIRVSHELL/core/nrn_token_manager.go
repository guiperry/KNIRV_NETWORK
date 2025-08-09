package core

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/guiperry/KNIRVCHAIN-CLI/config"
	"github.com/sirupsen/logrus"
)

// NRNTokenManager manages NRN token operations
type NRNTokenManager struct {
	config         *config.WalletConfig
	logger         *logrus.Logger
	balance        *big.Int
	transactions   []*NRNTransaction
	faucetClient   *FaucetClient
	knirvRootClient *KNIRVRootClient
}

// NRNTransaction represents an NRN token transaction
type NRNTransaction struct {
	ID          string    `json:"id"`
	Hash        string    `json:"hash"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Amount      *big.Int  `json:"amount"`
	Type        string    `json:"type"` // transfer, faucet, burn, stake
	Status      string    `json:"status"`
	BlockHeight int64     `json:"block_height"`
	Timestamp   time.Time `json:"timestamp"`
	GasFee      *big.Int  `json:"gas_fee"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// FaucetClient handles NRN faucet operations
type FaucetClient struct {
	*APIClient
	faucetURL string
	logger    *logrus.Logger
}

// FaucetRequest represents a faucet request
type FaucetRequest struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
}

// FaucetResponse represents a faucet response
type FaucetResponse struct {
	TransactionHash string `json:"transaction_hash"`
	Amount          string `json:"amount"`
	Status          string `json:"status"`
	Message         string `json:"message"`
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

	// Get balance from KNIRVROOT
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

	// TODO: Implement actual NRN transfer via KNIRVROOT
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

// RequestFromFaucet requests NRN tokens from the faucet
func (ntm *NRNTokenManager) RequestFromFaucet(ctx context.Context, address string, amount string) (*NRNTransaction, error) {
	if !ntm.config.NRN.Enabled {
		return nil, fmt.Errorf("NRN is disabled")
	}

	ntm.logger.Infof("Requesting %s NRN from faucet for address: %s", amount, address)

	// Use KNIRVROOT client for faucet request
	err := ntm.knirvRootClient.RequestNRNFromFaucet(ctx, address, amount)
	if err != nil {
		return nil, fmt.Errorf("faucet request failed: %w", err)
	}

	// Parse amount
	amountBig, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount format: %s", amount)
	}

	// Create transaction record
	tx := &NRNTransaction{
		ID:        generateTransactionID(),
		From:      "faucet",
		To:        address,
		Amount:    amountBig,
		Type:      "faucet",
		Status:    "confirmed",
		Timestamp: time.Now(),
		GasFee:    big.NewInt(0),
		Hash:      fmt.Sprintf("0x%x", generateTransactionHash(nil)),
		Metadata:  map[string]interface{}{"source": "faucet"},
	}

	// Update local balance
	ntm.balance.Add(ntm.balance, amountBig)

	// Add to transaction history
	ntm.transactions = append(ntm.transactions, tx)

	ntm.logger.Infof("Faucet request completed: %s NRN received", amount)
	return tx, nil
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
		"current_balance":    ntm.balance.String(),
		"total_transactions": len(ntm.transactions),
		"total_transferred":  totalTransferred.String(),
		"total_received":     totalReceived.String(),
		"total_burned":       totalBurned.String(),
		"auto_refill_enabled": ntm.config.NRN.AutoRefill,
		"min_balance":        ntm.config.NRN.MinBalance,
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
