package transactionchain

import (
	"backend_server/internal/objects"
	legacy "backend_server/internal/services/blockchain"
)

// Client is the transaction-chain boundary used by KNIRVSERVER services.
// For the initial migration it wraps the existing legacy blockchain client.
type Client struct {
	legacy *legacy.NRNClient
}

func NewClient(address string, useTLS bool, certFile string) (*Client, error) {
	client, err := legacy.NewNRNClient(address, useTLS, certFile)
	if err != nil {
		return nil, err
	}
	return &Client{legacy: client}, nil
}

func NewClientFromLegacy(client *legacy.NRNClient) *Client {
	if client == nil {
		return nil
	}
	return &Client{legacy: client}
}

func (c *Client) Legacy() *legacy.NRNClient {
	if c == nil {
		return nil
	}
	return c.legacy
}

func (c *Client) Close() error {
	if c == nil || c.legacy == nil {
		return nil
	}
	return c.legacy.Close()
}

func (c *Client) VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error) {
	return c.legacy.VerifyPaymentTransaction(txHash, expectedAmount, expectedRecipient)
}

func (c *Client) GetTransactionPool() ([]*legacy.Transaction, error) {
	return c.legacy.GetTransactionPool()
}

func (c *Client) SubmitTransaction(tx *legacy.Transaction) (string, error) {
	return c.legacy.SubmitTransaction(tx)
}

func (c *Client) GetAccountBalance(address string) (int64, error) {
	return c.legacy.GetAccountBalance(address)
}

func (c *Client) GetBlockHeight() (uint64, error) {
	return c.legacy.GetBlockHeight()
}

func (c *Client) GetChainID() (string, error) {
	return c.legacy.GetChainID()
}

func (c *Client) RegisterDVENode(nodeID, ownerAddress string, stakeAmount int64) (string, error) {
	return c.legacy.RegisterDVENode(nodeID, ownerAddress, stakeAmount)
}

func (c *Client) CreateChainSession(dveNodeID, ownerAddress string) (*objects.ChainSession, error) {
	return c.legacy.CreateChainSession(dveNodeID, ownerAddress)
}

func (c *Client) ValidateSession(sessionID string) (*objects.ChainSession, error) {
	return c.legacy.ValidateSession(sessionID)
}

func (c *Client) GetSecret(sessionID, secretKey string) (string, error) {
	return c.legacy.GetSecret(sessionID, secretKey)
}
