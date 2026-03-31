package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BlockchainService handles blockchain payment processing for NRN tokens
type BlockchainService struct {
	chainAPIURL string
	httpClient  *http.Client
}

// BlockchainTransaction represents a blockchain transaction
type BlockchainTransaction struct {
	TxHash      string `json:"tx_hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      int64  `json:"amount"`
	Token       string `json:"token"`
	Status      string `json:"status"`
	BlockHeight int64  `json:"block_height"`
	Timestamp   int64  `json:"timestamp"`
	GasUsed     int64  `json:"gas_used"`
	GasPrice    int64  `json:"gas_price"`
}

// BlockchainWallet represents a blockchain wallet
type BlockchainWallet struct {
	Address string `json:"address"`
	Balance int64  `json:"balance"`
	Token   string `json:"token"`
	ChainID string `json:"chain_id"`
}

// BlockchainPaymentRequest represents a blockchain payment request
type BlockchainPaymentRequest struct {
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
	Amount      int64  `json:"amount"`
	Token       string `json:"token"`
	Memo        string `json:"memo,omitempty"`
}

// BlockchainPaymentResponse represents a blockchain payment response
type BlockchainPaymentResponse struct {
	TxHash    string `json:"tx_hash"`
	Status    string `json:"status"`
	GasUsed   int64  `json:"gas_used"`
	GasPrice  int64  `json:"gas_price"`
	Fee       int64  `json:"fee"`
	Timestamp int64  `json:"timestamp"`
}

// NewBlockchainService creates a new blockchain service
func NewBlockchainService(chainAPIURL string) *BlockchainService {
	return &BlockchainService{
		chainAPIURL: chainAPIURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetWalletBalance retrieves the NRN token balance for a wallet address
func (b *BlockchainService) GetWalletBalance(address string, token string) (*BlockchainWallet, error) {
	url := fmt.Sprintf("%s/api/v1/wallets/%s/balance?token=%s", b.chainAPIURL, address, token)

	resp, err := b.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("blockchain API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("blockchain API error (status %d): %s", resp.StatusCode, string(body))
	}

	var wallet BlockchainWallet
	if err := json.Unmarshal(body, &wallet); err != nil {
		return nil, fmt.Errorf("failed to parse wallet response: %w", err)
	}

	return &wallet, nil
}

// CreatePayment creates a blockchain payment transaction
func (b *BlockchainService) CreatePayment(req BlockchainPaymentRequest) (*BlockchainPaymentResponse, error) {
	url := fmt.Sprintf("%s/api/v1/transactions", b.chainAPIURL)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("blockchain API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("blockchain API error (status %d): %s", resp.StatusCode, string(body))
	}

	var paymentResp BlockchainPaymentResponse
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		return nil, fmt.Errorf("failed to parse payment response: %w", err)
	}

	return &paymentResp, nil
}

// GetTransaction retrieves a blockchain transaction by hash
func (b *BlockchainService) GetTransaction(txHash string) (*BlockchainTransaction, error) {
	url := fmt.Sprintf("%s/api/v1/transactions/%s", b.chainAPIURL, txHash)

	resp, err := b.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("blockchain API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("blockchain API error (status %d): %s", resp.StatusCode, string(body))
	}

	var tx BlockchainTransaction
	if err := json.Unmarshal(body, &tx); err != nil {
		return nil, fmt.Errorf("failed to parse transaction response: %w", err)
	}

	return &tx, nil
}

// VerifyTransaction verifies a blockchain transaction
func (b *BlockchainService) VerifyTransaction(txHash string, expectedAmount int64, expectedRecipient string) (bool, error) {
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		return false, fmt.Errorf("failed to get transaction: %w", err)
	}

	if tx.Status != "confirmed" {
		return false, fmt.Errorf("transaction not confirmed: %s", tx.Status)
	}

	if tx.Amount != expectedAmount {
		return false, fmt.Errorf("amount mismatch: expected %d, got %d", expectedAmount, tx.Amount)
	}

	if tx.To != expectedRecipient {
		return false, fmt.Errorf("recipient mismatch: expected %s, got %s", expectedRecipient, tx.To)
	}

	return true, nil
}

// EstimateGas estimates the gas cost for a transaction
func (b *BlockchainService) EstimateGas(req BlockchainPaymentRequest) (int64, int64, error) {
	url := fmt.Sprintf("%s/api/v1/transactions/estimate-gas", b.chainAPIURL)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return 0, 0, fmt.Errorf("blockchain API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("blockchain API error (status %d): %s", resp.StatusCode, string(body))
	}

	var estimate struct {
		GasUsed  int64 `json:"gas_used"`
		GasPrice int64 `json:"gas_price"`
	}

	if err := json.Unmarshal(body, &estimate); err != nil {
		return 0, 0, fmt.Errorf("failed to parse estimate response: %w", err)
	}

	return estimate.GasUsed, estimate.GasPrice, nil
}

// GetTransactionHistory retrieves transaction history for a wallet address
func (b *BlockchainService) GetTransactionHistory(address string, limit int, offset int) ([]BlockchainTransaction, error) {
	url := fmt.Sprintf("%s/api/v1/wallets/%s/transactions?limit=%d&offset=%d", b.chainAPIURL, address, limit, offset)

	resp, err := b.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("blockchain API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("blockchain API error (status %d): %s", resp.StatusCode, string(body))
	}

	var transactions []BlockchainTransaction
	if err := json.Unmarshal(body, &transactions); err != nil {
		return nil, fmt.Errorf("failed to parse transactions response: %w", err)
	}

	return transactions, nil
}
