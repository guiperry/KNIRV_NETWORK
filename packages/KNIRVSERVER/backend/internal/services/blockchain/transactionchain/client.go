package transactionchain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"backend_server/internal/objects"
	pb "backend_server/internal/proto/blockchain"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client handles communication with the transaction chain.
// Supports both HTTP mode (when address starts with http:// or https://) and gRPC mode.
type Client struct {
	httpBaseURL   string
	httpClient    *http.Client
	balanceReader interface {
		GetAccountBalance(address string) (int64, error)
	}

	conn   *grpc.ClientConn
	client pb.BlockchainServiceClient
}

// NewClient creates a new transaction-chain client.
func NewClient(address string, useTLS bool, certFile string) (*Client, error) {
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return &Client{
			httpBaseURL: address,
			httpClient:  &http.Client{Timeout: 30 * time.Second},
		}, nil
	}

	grpcAddr := strings.TrimPrefix(strings.TrimPrefix(address, "https://"), "http://")
	var opts []grpc.DialOption

	if useTLS && certFile != "" {
		creds, err := credentials.NewClientTLSFromFile(certFile, "")
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, grpcAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to blockchain: %w", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewBlockchainServiceClient(conn),
	}, nil
}

// NewNRNClient is a temporary compatibility constructor while imports migrate.
func NewNRNClient(address string, useTLS bool, certFile string) (*Client, error) {
	return NewClient(address, useTLS, certFile)
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) SetBalanceReader(reader interface {
	GetAccountBalance(address string) (int64, error)
}) {
	c.balanceReader = reader
}

func (c *Client) VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error) {
	if c.httpBaseURL != "" {
		return c.httpVerifyPaymentTransaction(txHash, expectedAmount, expectedRecipient)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.VerifyPayment(ctx, &pb.VerifyPaymentRequest{
		TxHash:            txHash,
		ExpectedAmount:    expectedAmount,
		ExpectedRecipient: expectedRecipient,
	})
	if err != nil {
		return nil, fmt.Errorf("blockchain gRPC error: %w", err)
	}
	if !resp.Verified {
		return nil, fmt.Errorf("payment not verified by blockchain")
	}

	payment := &objects.NRNPayment{
		ID:          resp.PaymentId,
		Amount:      resp.Amount,
		TxHash:      txHash,
		Status:      resp.Status,
		BlockHeight: resp.BlockHeight,
		CreatedAt:   time.Now(),
	}
	if resp.ConfirmedAt != nil {
		payment.ConfirmedAt = new(time.Time)
		*payment.ConfirmedAt = resp.ConfirmedAt.AsTime()
		payment.CreatedAt = resp.ConfirmedAt.AsTime()
	}

	return payment, nil
}

func (c *Client) httpVerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error) {
	resp, err := c.httpClient.Get(c.httpBaseURL + "/chain")
	if err != nil {
		return nil, fmt.Errorf("failed to query blockchain: %w", err)
	}
	defer resp.Body.Close()

	var chainResp struct {
		Blocks []*Block `json:"blocks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chainResp); err != nil {
		return nil, fmt.Errorf("failed to decode chain response: %w", err)
	}

	for _, block := range chainResp.Blocks {
		for _, tx := range block.Transactions {
			if tx.TransactionHash == txHash {
				if tx.Value != expectedAmount {
					return nil, fmt.Errorf("amount mismatch: expected %d, got %d", expectedAmount, tx.Value)
				}
				if tx.To != expectedRecipient {
					return nil, fmt.Errorf("recipient mismatch: expected %s, got %s", expectedRecipient, tx.To)
				}
				return &objects.NRNPayment{
					ID:     txHash,
					Amount: tx.Value,
					TxHash: txHash,
					Status: "confirmed",
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("transaction %s not found in blockchain", txHash)
}

func (c *Client) GetTransactionPool() ([]*Transaction, error) {
	if c.httpBaseURL != "" {
		return c.httpGetTransactionPool()
	}
	return nil, fmt.Errorf("GetTransactionPool is deprecated in production gRPC client")
}

func (c *Client) httpGetTransactionPool() ([]*Transaction, error) {
	resp, err := c.httpClient.Get(c.httpBaseURL + "/txn_pool")
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction pool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transaction pool query failed with status: %d", resp.StatusCode)
	}

	var pool []*Transaction
	if err := json.NewDecoder(resp.Body).Decode(&pool); err != nil {
		return nil, fmt.Errorf("failed to decode pool response: %w", err)
	}

	return pool, nil
}

func (c *Client) SubmitTransaction(tx *Transaction) (string, error) {
	if c.httpBaseURL != "" {
		return c.httpSubmitTransaction(tx)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.SubmitTransaction(ctx, &pb.SubmitTransactionRequest{
		From:         tx.From,
		To:           tx.To,
		Value:        tx.Value,
		Data:         tx.Data,
		Signature:    tx.Signature,
		PublicKey:    tx.PublicKey,
		PqcSignature: tx.PQCSignature,
	})
	if err != nil {
		return "", fmt.Errorf("failed to submit transaction via gRPC: %w", err)
	}

	return resp.TxHash, nil
}

func (c *Client) httpSubmitTransaction(tx *Transaction) (string, error) {
	txJSON, err := json.Marshal(tx)
	if err != nil {
		return "", fmt.Errorf("failed to marshal transaction: %w", err)
	}

	resp, err := c.httpClient.Post(c.httpBaseURL+"/transaction", "application/json", bytes.NewBuffer(txJSON))
	if err != nil {
		return "", fmt.Errorf("failed to submit transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("transaction submission failed with status: %d", resp.StatusCode)
	}

	var result struct {
		TxHash string `json:"tx_hash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode submission response: %w", err)
	}

	return result.TxHash, nil
}

func (c *Client) GetAccountBalance(address string) (int64, error) {
	if address == "" {
		return 0, fmt.Errorf("address cannot be empty")
	}
	if c.balanceReader != nil {
		return c.balanceReader.GetAccountBalance(address)
	}
	if c.httpBaseURL != "" {
		return c.httpGetAccountBalance(address)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.GetBalance(ctx, &pb.GetBalanceRequest{Address: address})
	if err != nil {
		return 0, fmt.Errorf("failed to get balance via gRPC: %w", err)
	}
	return resp.Balance, nil
}

func (c *Client) httpGetAccountBalance(address string) (int64, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/account/%s/balance", c.httpBaseURL, address))
	if err != nil {
		return 0, fmt.Errorf("failed to query account balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("account balance query failed with status: %d", resp.StatusCode)
	}

	var balanceResp struct {
		Balance int64 `json:"balance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&balanceResp); err != nil {
		return 0, fmt.Errorf("failed to decode balance response: %w", err)
	}

	return balanceResp.Balance, nil
}

func (c *Client) GetBlockHeight() (uint64, error) {
	if c.httpBaseURL != "" {
		return c.httpGetBlockHeight()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.GetBlockHeight(ctx, &pb.GetBlockHeightRequest{})
	if err != nil {
		return 0, fmt.Errorf("failed to get block height via gRPC: %w", err)
	}

	return resp.Height, nil
}

func (c *Client) httpGetBlockHeight() (uint64, error) {
	resp, err := c.httpClient.Get(c.httpBaseURL + "/block_height")
	if err != nil {
		return 0, fmt.Errorf("failed to query blockchain: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("block height query failed with status: %d", resp.StatusCode)
	}

	var result struct {
		Height uint64 `json:"height"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode block height response: %w", err)
	}

	return result.Height, nil
}

// TransferNRN submits an NRN token transfer transaction
func (c *Client) TransferNRN(senderID, recipientID string, amount int64) (*objects.NRNPayment, error) {
	if senderID == "" || recipientID == "" {
		return nil, fmt.Errorf("sender and recipient IDs are required")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	if c.httpBaseURL != "" {
		return c.httpTransferNRN(senderID, recipientID, amount)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.SubmitTransaction(ctx, &pb.SubmitTransactionRequest{
		From:  senderID,
		To:    recipientID,
		Value: amount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to submit NRN transfer via gRPC: %w", err)
	}

	return &objects.NRNPayment{
		ID:        fmt.Sprintf("nrm_%s_%d", senderID, time.Now().Unix()),
		TxHash:    resp.TxHash,
		Amount:    amount,
		Status:    resp.Status,
		CreatedAt: time.Now(),
	}, nil
}

func (c *Client) httpTransferNRN(senderID, recipientID string, amount int64) (*objects.NRNPayment, error) {
	tx := &Transaction{
		From:  senderID,
		To:    recipientID,
		Value: amount,
	}

	txJSON, err := json.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	resp, err := c.httpClient.Post(c.httpBaseURL+"/transfer", "application/json", bytes.NewBuffer(txJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to submit NRN transfer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("NRN transfer failed with status: %d", resp.StatusCode)
	}

	var result struct {
		TxHash string `json:"tx_hash"`
		Status  string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode transfer response: %w", err)
	}

	return &objects.NRNPayment{
		ID:        fmt.Sprintf("nrm_%s_%d", senderID, time.Now().Unix()),
		TxHash:    result.TxHash,
		Amount:    amount,
		Status:    result.Status,
		CreatedAt: time.Now(),
	}, nil
}

func (c *Client) RegisterDVENode(nodeID, ownerAddress string, stakeAmount int64) (string, error) {
	if c.httpBaseURL != "" {
		return "", fmt.Errorf("RegisterDVENode requires gRPC mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.RegisterDVENode(ctx, &pb.RegisterDVENodeRequest{
		NodeId:       nodeID,
		OwnerAddress: ownerAddress,
		StakeAmount:  stakeAmount,
	})
	if err != nil {
		return "", fmt.Errorf("failed to register DVE node: %w", err)
	}

	return resp.TxHash, nil
}

func (c *Client) CreateChainSession(dveNodeID, ownerAddress string) (*objects.ChainSession, error) {
	if c.httpBaseURL != "" {
		return nil, fmt.Errorf("CreateChainSession requires gRPC mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.CreateChainSession(ctx, &pb.CreateChainSessionRequest{
		DveNodeId:    dveNodeID,
		OwnerAddress: ownerAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chain session: %w", err)
	}

	createdAt := time.Now()
	expiresAt := createdAt.Add(30 * time.Minute)
	if resp.ExpiresAt != nil {
		expiresAt = resp.ExpiresAt.AsTime()
	}

	return &objects.ChainSession{
		SessionID:     resp.SessionId,
		DVENodeID:     dveNodeID,
		OwnerAddress:  ownerAddress,
		BlockHeight:   0,
		ChainID:       "knirv-chain-1",
		CreatedAt:     createdAt,
		ExpiresAt:     expiresAt,
		LastValidated: createdAt,
		Status:        "active",
		PQCSignature:  resp.PqcSignature,
	}, nil
}

func (c *Client) ValidateSession(sessionID string) (*objects.ChainSession, error) {
	if c.httpBaseURL != "" {
		return nil, fmt.Errorf("ValidateSession requires gRPC mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.ValidateSession(ctx, &pb.ValidateSessionRequest{SessionId: sessionID})
	if err != nil {
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}
	if !resp.Valid {
		return nil, fmt.Errorf("session is not valid: %s", sessionID)
	}

	validatedAt := time.Now()
	expiresAt := validatedAt.Add(30 * time.Minute)
	if resp.ExpiresAt != nil {
		expiresAt = resp.ExpiresAt.AsTime()
	}

	return &objects.ChainSession{
		SessionID:     sessionID,
		BlockHeight:   0,
		ChainID:       "knirv-chain-1",
		CreatedAt:     validatedAt,
		ExpiresAt:     expiresAt,
		LastValidated: validatedAt,
		Status:        "active",
	}, nil
}

func (c *Client) GetSecret(sessionID, secretKey string) (string, error) {
	if c.httpBaseURL != "" {
		return "", fmt.Errorf("GetSecret requires gRPC mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.GetSecret(ctx, &pb.GetSecretRequest{
		SessionId: sessionID,
		SecretKey: secretKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get secret: %w", err)
	}

	return resp.SecretValue, nil
}

func (c *Client) GetChainID() (string, error) {
	return "knirv-chain-1", nil
}

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

type Block struct {
	BlockNumber  uint64         `json:"block_number"`
	Transactions []*Transaction `json:"transactions"`
	Timestamp    int64          `json:"timestamp"`
	Hash         string         `json:"hash"`
	PrevHash     string         `json:"prev_hash"`
}
