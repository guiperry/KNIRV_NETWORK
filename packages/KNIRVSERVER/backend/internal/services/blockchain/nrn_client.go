package blockchain

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

// NRNClient handles communication with the KNIRVCHAIN blockchain.
// Supports both HTTP mode (when address starts with http:// or https://) and gRPC mode.
type NRNClient struct {
	// HTTP mode fields
	httpBaseURL string
	httpClient  *http.Client

	// gRPC mode fields
	conn   *grpc.ClientConn
	client pb.BlockchainServiceClient
}

// NewNRNClient creates a new NRN blockchain client.
// If address starts with http:// or https://, HTTP mode is used.
// Otherwise, gRPC mode is used (stripping any http/https prefix for compatibility).
func NewNRNClient(address string, useTLS bool, certFile string) (*NRNClient, error) {
	// HTTP mode: when address starts with http:// or https://
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return &NRNClient{
			httpBaseURL: address,
			httpClient:  &http.Client{Timeout: 30 * time.Second},
		}, nil
	}

	// gRPC mode: strip any accidental http/https prefix
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

	// Add timeout for connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, grpcAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to blockchain: %w", err)
	}

	return &NRNClient{
		conn:   conn,
		client: pb.NewBlockchainServiceClient(conn),
	}, nil
}

// Close closes the connection to the blockchain
func (nc *NRNClient) Close() error {
	if nc.conn != nil {
		return nc.conn.Close()
	}
	return nil
}

// VerifyPaymentTransaction verifies an NRN payment transaction on the blockchain
func (nc *NRNClient) VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error) {
	if nc.httpBaseURL != "" {
		return nc.httpVerifyPaymentTransaction(txHash, expectedAmount, expectedRecipient)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := nc.client.VerifyPayment(ctx, &pb.VerifyPaymentRequest{
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
		CreatedAt:   time.Now(), // Default
	}

	if resp.ConfirmedAt != nil {
		payment.ConfirmedAt = new(time.Time)
		*payment.ConfirmedAt = resp.ConfirmedAt.AsTime()
		payment.CreatedAt = resp.ConfirmedAt.AsTime()
	}

	return payment, nil
}

func (nc *NRNClient) httpVerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error) {
	resp, err := nc.httpClient.Get(nc.httpBaseURL + "/chain")
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

// GetTransactionPool retrieves the pending transaction pool.
// In gRPC mode this is deprecated; in HTTP mode it queries /txn_pool.
func (nc *NRNClient) GetTransactionPool() ([]*Transaction, error) {
	if nc.httpBaseURL != "" {
		return nc.httpGetTransactionPool()
	}
	return nil, fmt.Errorf("GetTransactionPool is deprecated in production gRPC client")
}

func (nc *NRNClient) httpGetTransactionPool() ([]*Transaction, error) {
	resp, err := nc.httpClient.Get(nc.httpBaseURL + "/txn_pool")
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

// SubmitTransaction submits a signed transaction to the blockchain
func (nc *NRNClient) SubmitTransaction(tx *Transaction) (string, error) {
	if nc.httpBaseURL != "" {
		return nc.httpSubmitTransaction(tx)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := nc.client.SubmitTransaction(ctx, &pb.SubmitTransactionRequest{
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

func (nc *NRNClient) httpSubmitTransaction(tx *Transaction) (string, error) {
	txJSON, err := json.Marshal(tx)
	if err != nil {
		return "", fmt.Errorf("failed to marshal transaction: %w", err)
	}

	resp, err := nc.httpClient.Post(nc.httpBaseURL+"/transaction", "application/json", bytes.NewBuffer(txJSON))
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

// GetAccountBalance retrieves the NRN balance for an account
func (nc *NRNClient) GetAccountBalance(address string) (int64, error) {
	if address == "" {
		return 0, fmt.Errorf("address cannot be empty")
	}

	if nc.httpBaseURL != "" {
		return nc.httpGetAccountBalance(address)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := nc.client.GetBalance(ctx, &pb.GetBalanceRequest{
		Address: address,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to get balance via gRPC: %w", err)
	}

	return resp.Balance, nil
}

func (nc *NRNClient) httpGetAccountBalance(address string) (int64, error) {
	resp, err := nc.httpClient.Get(fmt.Sprintf("%s/account/%s/balance", nc.httpBaseURL, address))
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

// GetBlockHeight returns the current block height
func (nc *NRNClient) GetBlockHeight() (uint64, error) {
	if nc.httpBaseURL != "" {
		return nc.httpGetBlockHeight()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := nc.client.GetBlockHeight(ctx, &pb.GetBlockHeightRequest{})
	if err != nil {
		return 0, fmt.Errorf("failed to get block height: %w", err)
	}

	return resp.Height, nil
}

func (nc *NRNClient) httpGetBlockHeight() (uint64, error) {
	resp, err := nc.httpClient.Get(nc.httpBaseURL + "/chain/height")
	if err != nil {
		return 0, fmt.Errorf("failed to query block height: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("block height query failed with status: %d", resp.StatusCode)
	}

	var heightResp struct {
		Height uint64 `json:"height"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&heightResp); err != nil {
		return 0, fmt.Errorf("failed to decode height response: %w", err)
	}

	return heightResp.Height, nil
}

// RegisterDVENode registers a DVE node on-chain
func (nc *NRNClient) RegisterDVENode(nodeID, ownerAddress string, stakeAmount int64) (string, error) {
	if nc.httpBaseURL != "" {
		return "", fmt.Errorf("RegisterDVENode requires gRPC mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := nc.client.RegisterDVENode(ctx, &pb.RegisterDVENodeRequest{
		NodeId:       nodeID,
		OwnerAddress: ownerAddress,
		StakeAmount:  stakeAmount,
	})

	if err != nil {
		return "", fmt.Errorf("failed to register DVE node: %w", err)
	}

	if !resp.Success {
		return "", fmt.Errorf("node registration rejected by blockchain")
	}

	return resp.TxHash, nil
}

// CreateChainSession creates an exclusive session with the blockchain
func (nc *NRNClient) CreateChainSession(dveNodeID, ownerAddress string) (*objects.ChainSession, error) {
	if nc.httpBaseURL != "" {
		return nil, fmt.Errorf("CreateChainSession requires gRPC mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := nc.client.CreateChainSession(ctx, &pb.CreateChainSessionRequest{
		DveNodeId:    dveNodeID,
		OwnerAddress: ownerAddress,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create chain session: %w", err)
	}

	return &objects.ChainSession{
		SessionID:    resp.SessionId,
		DVENodeID:    dveNodeID,
		ExpiresAt:    resp.ExpiresAt.AsTime(),
		PQCSignature: resp.PqcSignature,
	}, nil
}

// ValidateSession validates an existing chain session
func (nc *NRNClient) ValidateSession(sessionID string) (*objects.ChainSession, error) {
	if nc.httpBaseURL != "" {
		return nil, fmt.Errorf("ValidateSession requires gRPC mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := nc.client.ValidateSession(ctx, &pb.ValidateSessionRequest{
		SessionId: sessionID,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}

	if !resp.Valid {
		return nil, fmt.Errorf("invalid session")
	}

	return &objects.ChainSession{
		SessionID: sessionID,
		ExpiresAt: resp.ExpiresAt.AsTime(),
	}, nil
}

// GetSecret retrieves a secret from the blockchain central authority
func (nc *NRNClient) GetSecret(sessionID, secretKey string) (string, error) {
	if nc.httpBaseURL != "" {
		return "", fmt.Errorf("GetSecret requires gRPC mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := nc.client.GetSecret(ctx, &pb.GetSecretRequest{
		SessionId: sessionID,
		SecretKey: secretKey,
	})

	if err != nil {
		return "", fmt.Errorf("failed to get secret via gRPC: %w", err)
	}

	if !resp.Success {
		return "", fmt.Errorf("secret retrieval rejected by blockchain")
	}

	return resp.SecretValue, nil
}

// GetChainID returns the blockchain ID
func (nc *NRNClient) GetChainID() (string, error) {
	// Simple static implementation for now
	return "knirv-chain-1", nil
}

// Transaction represents a blockchain transaction (kept for compatibility)
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

// Block represents a blockchain block (kept for compatibility)
type Block struct {
	BlockNumber  uint64         `json:"block_number"`
	Transactions []*Transaction `json:"transactions"`
	Timestamp    int64          `json:"timestamp"`
	Hash         string         `json:"hash"`
	PrevHash     string         `json:"prev_hash"`
}
