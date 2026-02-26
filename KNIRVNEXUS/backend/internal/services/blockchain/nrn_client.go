package blockchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"backend_server/internal/objects"
)

// NRNClient handles communication with the KNIRVORACLE blockchain for NRN token operations
type NRNClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewNRNClient creates a new NRN blockchain client
func NewNRNClient(baseURL string) *NRNClient {
	return &NRNClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// VerifyPaymentTransaction verifies an NRN payment transaction on the blockchain
func (nc *NRNClient) VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error) {
	// Query the blockchain for the transaction
	url := fmt.Sprintf("%s/chain", nc.baseURL)
	resp, err := nc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query blockchain: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("blockchain query failed with status: %d", resp.StatusCode)
	}

	var chainData struct {
		Blocks []*Block `json:"blocks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chainData); err != nil {
		return nil, fmt.Errorf("failed to decode blockchain response: %w", err)
	}

	// Search for the transaction in the blockchain
	for _, block := range chainData.Blocks {
		for _, tx := range block.Transactions {
			if tx.TransactionHash == txHash {
				// Verify transaction details
				if tx.Value != expectedAmount {
					return nil, fmt.Errorf("transaction amount mismatch: expected %d, got %d", expectedAmount, tx.Value)
				}
				if tx.To != expectedRecipient {
					return nil, fmt.Errorf("transaction recipient mismatch: expected %s, got %s", expectedRecipient, tx.To)
				}

				// Check confirmations (simplified - in real implementation, check against current block height)
				confirmations := 1 // Simplified for this implementation

				payment := &objects.NRNPayment{
					ID:            tx.TransactionHash,
					Amount:        tx.Value,
					TxHash:        tx.TransactionHash,
					Status:        "confirmed",
					BlockHeight:   int64(block.BlockNumber),
					Confirmations: confirmations,
					CreatedAt:     time.Unix(tx.Timestamp, 0),
					ConfirmedAt:   &time.Time{}, // Set to current time for confirmed tx
				}
				*payment.ConfirmedAt = time.Now()

				return payment, nil
			}
		}
	}

	return nil, fmt.Errorf("transaction %s not found in blockchain", txHash)
}

// GetTransactionPool checks if a transaction is in the pending pool
func (nc *NRNClient) GetTransactionPool() ([]*Transaction, error) {
	url := fmt.Sprintf("%s/txn_pool", nc.baseURL)
	resp, err := nc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction pool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transaction pool query failed with status: %d", resp.StatusCode)
	}

	var pool []*Transaction
	if err := json.NewDecoder(resp.Body).Decode(&pool); err != nil {
		return nil, fmt.Errorf("failed to decode transaction pool response: %w", err)
	}

	return pool, nil
}

// SubmitTransaction submits a signed transaction to the blockchain
func (nc *NRNClient) SubmitTransaction(tx *Transaction) (string, error) {
	url := fmt.Sprintf("%s/transaction", nc.baseURL)

	txJSON, err := json.Marshal(tx)
	if err != nil {
		return "", fmt.Errorf("failed to marshal transaction: %w", err)
	}

	resp, err := nc.httpClient.Post(url, "application/json", bytes.NewBuffer(txJSON))
	if err != nil {
		return "", fmt.Errorf("failed to submit transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("transaction submission failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		TxHash string `json:"tx_hash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode submission response: %w", err)
	}

	return result.TxHash, nil
}

// GetAccountBalance retrieves the NRN balance for an account
func (nc *NRNClient) GetAccountBalance(address string) (int64, error) {
	// This would typically query the blockchain state
	// For now, return a mock balance for testing
	if address == "" {
		return 0, fmt.Errorf("invalid address")
	}
	// Mock implementation - in real system, query blockchain state
	return 1000000, nil // 1000 NRN in smallest units
}

// Block represents a blockchain block (simplified)
type Block struct {
	BlockNumber  int            `json:"block_number"`
	Transactions []*Transaction `json:"transactions"`
	Timestamp    int64          `json:"timestamp"`
	Hash         string         `json:"hash"`
	PrevHash     string         `json:"prev_hash"`
}

// Transaction represents a blockchain transaction (simplified)
type Transaction struct {
	TransactionHash string `json:"transaction_hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           int64  `json:"value"`
	Data            []byte `json:"data"`
	Timestamp       int64  `json:"timestamp"`
	Signature       []byte `json:"signature"`
	PublicKey       string `json:"public_key"`
	Type            string `json:"type"`
	Fee             uint64 `json:"fee"`
	Status          string `json:"status"`
	ChainID         string `json:"chain_id"`
	BlockHeight     uint64 `json:"block_height"`
	PQCSignature    []byte `json:"pqc_signature"`
}
