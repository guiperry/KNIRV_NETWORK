package wallet

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// NRNWallet manages NRN tokens on the XION network
type NRNWallet struct {
	address    string
	privateKey *PrivateKey
	client     *XIONClient
	history    *TransactionHistory
	mu         sync.RWMutex

	// Cached balance with TTL
	cachedBalance uint64
	cacheExpiry   time.Time
	cacheTTL      time.Duration

	// Account info
	accountNumber uint64
	sequence      uint64
}

// NewNRNWallet creates a new NRN wallet
// key parameter can be:
// - Path to private key file
// - "mock" for testing with mock client
// - Empty string to generate new key
func NewNRNWallet(key string) (*NRNWallet, error) {
	var privKey *PrivateKey
	var address string

	if key == "mock" {
		// Mock wallet for testing
		privKey = NewPrivateKey([]byte("mock_private_key_for_testing"))
		address = "xion1mockaddress"
	} else if key == "" {
		return nil, fmt.Errorf("private key required (provide file path or 'mock' for testing)")
	} else {
		// Load private key from file
		keyBytes, err := os.ReadFile(key)
		if err != nil {
			// If file doesn't exist, treat as raw key
			keyBytes = []byte(key)
		}
		privKey = NewPrivateKey(keyBytes)

		// Derive address from public key
		pubKey := privKey.GetPublicKey()
		address = deriveAddress(pubKey)
	}

	client := NewDefaultXIONClient()

	wallet := &NRNWallet{
		address:    address,
		privateKey: privKey,
		client:     client,
		history:    &TransactionHistory{Transactions: make([]HistoricalTransaction, 0)},
		cacheTTL:   30 * time.Second,
	}

	return wallet, nil
}

// NewNRNWalletWithClient creates a wallet with a custom XION client (for testing)
func NewNRNWalletWithClient(address string, privKey *PrivateKey, client *XIONClient) *NRNWallet {
	return &NRNWallet{
		address:    address,
		privateKey: privKey,
		client:     client,
		history:    &TransactionHistory{Transactions: make([]HistoricalTransaction, 0)},
		cacheTTL:   30 * time.Second,
	}
}

// Address returns the wallet address
func (w *NRNWallet) Address() string {
	return w.address
}

// Balance queries the current NRN balance from the XION network
func (w *NRNWallet) Balance(ctx context.Context) (uint64, error) {
	w.mu.RLock()
	// Check if cached balance is still valid
	if time.Now().Before(w.cacheExpiry) {
		balance := w.cachedBalance
		w.mu.RUnlock()
		return balance, nil
	}
	w.mu.RUnlock()

	// Query fresh balance from network
	balance, err := w.client.QueryBalance(ctx, w.address, "unrn")
	if err != nil {
		return 0, fmt.Errorf("failed to query balance: %w", err)
	}

	// Convert from micro-NRN to NRN (1 NRN = 1,000,000 unrn)
	nrnBalance := balance / 1000000

	// Update cache
	w.mu.Lock()
	w.cachedBalance = nrnBalance
	w.cacheExpiry = time.Now().Add(w.cacheTTL)
	w.mu.Unlock()

	return nrnBalance, nil
}

// HasBalance checks if the wallet has sufficient balance
func (w *NRNWallet) HasBalance(ctx context.Context, required uint64) (bool, error) {
	balance, err := w.Balance(ctx)
	if err != nil {
		return false, err
	}
	return balance >= required, nil
}

// Spend creates and broadcasts a transaction to spend NRN
func (w *NRNWallet) Spend(ctx context.Context, amount uint64, memo string) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check balance first
	balance, err := w.client.QueryBalance(ctx, w.address, "unrn")
	if err != nil {
		return "", fmt.Errorf("failed to query balance: %w", err)
	}

	nrnBalance := balance / 1000000
	if nrnBalance < amount {
		return "", fmt.Errorf("insufficient balance: have %d NRN, need %d NRN", nrnBalance, amount)
	}

	// Get account info (needed for transaction)
	// In a real implementation, this would query the network
	// For now, use cached values and increment sequence
	accountNumber := w.accountNumber
	sequence := w.sequence

	// Build transaction
	// Spending to self for now (actual recipient would be treasury or burn address)
	tx := NewTransactionBuilder(w.client.chainID, accountNumber, sequence).
		WithMemo(memo).
		WithFee(1000, "unrn", 200000). // 0.001 NRN fee
		WithSend(w.address, w.address, amount*1000000, "unrn"). // Convert NRN to unrn
		Build()

	// Sign transaction
	signedTx, err := SignTransaction(tx, w.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Serialize transaction
	txBytes, err := signedTx.Serialize()
	if err != nil {
		return "", fmt.Errorf("failed to serialize transaction: %w", err)
	}

	// Broadcast to network
	txHash, err := w.client.BroadcastTx(ctx, txBytes)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Update sequence for next transaction
	w.sequence++

	// Invalidate balance cache
	w.cacheExpiry = time.Time{}

	// Add to history
	w.history.AddTransaction(HistoricalTransaction{
		TxHash:    txHash,
		Timestamp: time.Now(),
		Amount:    amount,
		Memo:      memo,
		Status:    "broadcasted",
	})

	return txHash, nil
}

// GetTransactionHistory returns recent transaction history
func (w *NRNWallet) GetTransactionHistory(limit int) []HistoricalTransaction {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.history.GetRecent(limit)
}

// WaitForTransaction waits for a transaction to be confirmed
func (w *NRNWallet) WaitForTransaction(ctx context.Context, txHash string, timeout time.Duration) error {
	_, err := w.client.WaitForTransaction(ctx, txHash, timeout)
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	// Update transaction status in history
	w.mu.Lock()
	defer w.mu.Unlock()

	for i := range w.history.Transactions {
		if w.history.Transactions[i].TxHash == txHash {
			w.history.Transactions[i].Status = "confirmed"
			break
		}
	}

	// Invalidate balance cache
	w.cacheExpiry = time.Time{}

	return nil
}

// RefreshBalance forces a balance refresh from the network
func (w *NRNWallet) RefreshBalance(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	balance, err := w.client.QueryBalance(ctx, w.address, "unrn")
	if err != nil {
		return fmt.Errorf("failed to refresh balance: %w", err)
	}

	nrnBalance := balance / 1000000

	w.cachedBalance = nrnBalance
	w.cacheExpiry = time.Now().Add(w.cacheTTL)

	return nil
}

// GetClient returns the XION client (for testing)
func (w *NRNWallet) GetClient() *XIONClient {
	return w.client
}

// deriveAddress derives a bech32 address from a public key
// Simplified implementation - in production use proper bech32 encoding
func deriveAddress(pubKey *PublicKey) string {
	// Simple derivation: xion1 + first 40 chars of pubkey
	addr := "xion1"
	if len(pubKey.Key) > 40 {
		addr += pubKey.Key[:40]
	} else {
		addr += pubKey.Key
	}
	return addr
}
