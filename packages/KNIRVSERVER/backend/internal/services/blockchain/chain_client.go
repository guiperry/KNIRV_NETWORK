package blockchain

import (
	"context"
	"fmt"

	"backend_server/internal/objects"
	"github.com/sirupsen/logrus"
)

// APIClient matches the structure of KNIRVCLI APIClient
type APIClient interface {
	Get(ctx context.Context, path string, result interface{}) error
	Post(ctx context.Context, path string, body interface{}, result interface{}) error
}

// ChainClient wraps the KNIRVCLI APIClient for backend use
type ChainClient struct {
	client APIClient
	logger *logrus.Logger
}

// NewChainClient creates a new chain client
func NewChainClient(apiClient APIClient) *ChainClient {
	return &ChainClient{
		client: apiClient,
		logger: logrus.New(),
	}
}

// RegisterDVENode registers a DVE node on-chain via the APIClient
func (cc *ChainClient) RegisterDVENode(ctx context.Context, nodeID, ownerAddress string, stakeAmount int64) (string, error) {
	request := map[string]interface{}{
		"node_id":       nodeID,
		"owner_address": ownerAddress,
		"stake_amount":  stakeAmount,
	}

	var response struct {
		Success bool   `json:"success"`
		TxHash  string `json:"tx_hash"`
		Error   string `json:"error"`
	}

	err := cc.client.Post(ctx, "/api/v1/nodes/register", request, &response)
	if err != nil {
		return "", fmt.Errorf("failed to register node via chain API: %w", err)
	}

	if !response.Success {
		return "", fmt.Errorf("node registration failed: %s", response.Error)
	}

	return response.TxHash, nil
}

// VerifyPayment verifies a payment on-chain
func (cc *ChainClient) VerifyPayment(ctx context.Context, txHash string) (*objects.NRNPayment, error) {
	var response struct {
		Success bool               `json:"success"`
		Payment *objects.NRNPayment `json:"payment"`
		Error   string             `json:"error"`
	}

	err := cc.client.Get(ctx, fmt.Sprintf("/api/v1/payments/verify/%s", txHash), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to verify payment via chain API: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("payment verification failed: %s", response.Error)
	}

	return response.Payment, nil
}

// GetBalance retrieves NRN balance for an address
func (cc *ChainClient) GetBalance(ctx context.Context, address string) (int64, error) {
	var response struct {
		Success bool  `json:"success"`
		Balance int64 `json:"balance"`
		Error   string `json:"error"`
	}

	err := cc.client.Get(ctx, fmt.Sprintf("/api/v1/accounts/balance/%s", address), &response)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance via chain API: %w", err)
	}

	if !response.Success {
		return 0, fmt.Errorf("failed to get balance: %s", response.Error)
	}

	return response.Balance, nil
}

// CreateChainSession initializes a session with the chain
func (cc *ChainClient) CreateChainSession(ctx context.Context, dveNodeID, ownerAddress string) (*objects.ChainSession, error) {
	request := map[string]interface{}{
		"node_id":       dveNodeID,
		"owner_address": ownerAddress,
	}

	var response struct {
		Success bool                  `json:"success"`
		Session *objects.ChainSession `json:"session"`
		Error   string                `json:"error"`
	}

	err := cc.client.Post(ctx, "/api/v1/sessions/create", request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain session via API: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("chain session creation failed: %s", response.Error)
	}

	return response.Session, nil
}
