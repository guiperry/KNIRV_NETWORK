package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// XIONClient handles communication with the XION network
type XIONClient struct {
	rpcEndpoint string
	apiEndpoint string
	httpClient  *http.Client
	chainID     string
}

// NewXIONClient creates a new XION network client
func NewXIONClient(rpcEndpoint, apiEndpoint, chainID string) *XIONClient {
	return &XIONClient{
		rpcEndpoint: rpcEndpoint,
		apiEndpoint: apiEndpoint,
		chainID:     chainID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewDefaultXIONClient creates a client with default XION network endpoints
func NewDefaultXIONClient() *XIONClient {
	return NewXIONClient(
		"https://rpc.xion.network",
		"https://api.xion.network",
		"xion-mainnet-1",
	)
}

// BalanceResponse represents the balance query response
type BalanceResponse struct {
	Balances []Balance `json:"balances"`
}

// Balance represents a token balance
type Balance struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

// QueryBalance queries the NRN balance for an address
func (c *XIONClient) QueryBalance(ctx context.Context, address, denom string) (uint64, error) {
	if denom == "" {
		denom = "unrn" // micro-NRN (1 NRN = 1,000,000 unrn)
	}

	url := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", c.apiEndpoint, address)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to query balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("balance query failed (status %d): %s", resp.StatusCode, string(body))
	}

	var balanceResp BalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&balanceResp); err != nil {
		return 0, fmt.Errorf("failed to decode balance response: %w", err)
	}

	// Find the requested denomination
	for _, bal := range balanceResp.Balances {
		if bal.Denom == denom {
			var amount uint64
			_, err := fmt.Sscanf(bal.Amount, "%d", &amount)
			if err != nil {
				return 0, fmt.Errorf("failed to parse amount: %w", err)
			}
			return amount, nil
		}
	}

	// Denomination not found, balance is 0
	return 0, nil
}

// BroadcastTxRequest represents a transaction broadcast request
type BroadcastTxRequest struct {
	TxBytes string `json:"tx_bytes"`
	Mode    string `json:"mode"`
}

// BroadcastTxResponse represents a transaction broadcast response
type BroadcastTxResponse struct {
	TxResponse TxResponse `json:"tx_response"`
}

// TxResponse contains transaction result information
type TxResponse struct {
	Height    string `json:"height"`
	TxHash    string `json:"txhash"`
	Code      uint32 `json:"code"`
	RawLog    string `json:"raw_log"`
	Codespace string `json:"codespace"`
}

// BroadcastTx broadcasts a signed transaction to the XION network
func (c *XIONClient) BroadcastTx(ctx context.Context, txBytes []byte) (string, error) {
	url := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs", c.apiEndpoint)

	// Encode transaction bytes to base64
	txBytesB64 := encodeBase64(txBytes)

	reqBody := BroadcastTxRequest{
		TxBytes: txBytesB64,
		Mode:    "BROADCAST_MODE_SYNC",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("broadcast failed (status %d): %s", resp.StatusCode, string(body))
	}

	var txResp BroadcastTxResponse
	if err := json.Unmarshal(body, &txResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if transaction was accepted
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction rejected (code %d): %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return txResp.TxResponse.TxHash, nil
}

// GetTxRequest represents a transaction query request
type GetTxResponse struct {
	TxResponse TxResponse `json:"tx_response"`
}

// GetTransaction queries transaction status by hash
func (c *XIONClient) GetTransaction(ctx context.Context, txHash string) (*TxResponse, error) {
	url := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", c.apiEndpoint, txHash)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("transaction query failed (status %d): %s", resp.StatusCode, string(body))
	}

	var txResp GetTxResponse
	if err := json.NewDecoder(resp.Body).Decode(&txResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &txResp.TxResponse, nil
}

// WaitForTransaction waits for a transaction to be confirmed
func (c *XIONClient) WaitForTransaction(ctx context.Context, txHash string, timeout time.Duration) (*TxResponse, error) {
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for transaction %s", txHash)
		}

		txResp, err := c.GetTransaction(ctx, txHash)
		if err == nil {
			return txResp, nil
		}

		// Wait before retrying
		time.Sleep(1 * time.Second)
	}
}

// NodeInfo represents XION node information
type NodeInfoResponse struct {
	NodeInfo NodeInfo `json:"node_info"`
}

type NodeInfo struct {
	Network string `json:"network"`
	Version string `json:"version"`
}

// GetNodeInfo queries node information to verify connectivity
func (c *XIONClient) GetNodeInfo(ctx context.Context) (*NodeInfo, error) {
	url := fmt.Sprintf("%s/cosmos/base/tendermint/v1beta1/node_info", c.apiEndpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query node info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node info query failed (status %d): %s", resp.StatusCode, string(body))
	}

	var nodeInfoResp NodeInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&nodeInfoResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &nodeInfoResp.NodeInfo, nil
}

// Helper function to encode bytes to base64
func encodeBase64(data []byte) string {
	// Simple base64 encoding
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	var result []byte
	padding := (3 - len(data)%3) % 3

	for i := 0; i < len(data); i += 3 {
		var block uint32
		for j := 0; j < 3 && i+j < len(data); j++ {
			block = block<<8 | uint32(data[i+j])
		}

		// Shift to align properly
		if i+3 > len(data) {
			block <<= 8 * uint(3-(len(data)-i))
		}

		for j := 0; j < 4; j++ {
			if i+j*3/4 < len(data) || j < 4-padding {
				result = append(result, base64Chars[(block>>(6*(3-j)))&0x3F])
			} else {
				result = append(result, '=')
			}
		}
	}

	return string(result)
}
