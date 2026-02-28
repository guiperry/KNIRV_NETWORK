package blockchain

import (
	"context"
	"fmt"
	"time"

	"backend_server/internal/objects"
	pb "backend_server/internal/proto/blockchain"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// NRNClient handles communication with the KNIRVCHAIN blockchain using gRPC/mTLS
type NRNClient struct {
	conn   *grpc.ClientConn
	client pb.BlockchainServiceClient
}

// NewNRNClient creates a new NRN blockchain client
func NewNRNClient(address string, useTLS bool, certFile string) (*NRNClient, error) {
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

	conn, err := grpc.DialContext(ctx, address, opts...)
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

// GetTransactionPool is deprecated in production, use SubmitTransaction directly
func (nc *NRNClient) GetTransactionPool() ([]*Transaction, error) {
	return nil, fmt.Errorf("GetTransactionPool is deprecated in production gRPC client")
}

// SubmitTransaction submits a signed transaction to the blockchain
func (nc *NRNClient) SubmitTransaction(tx *Transaction) (string, error) {
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

// GetAccountBalance retrieves the NRN balance for an account
func (nc *NRNClient) GetAccountBalance(address string) (int64, error) {
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

// GetBlockHeight returns the current block height
func (nc *NRNClient) GetBlockHeight() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := nc.client.GetBlockHeight(ctx, &pb.GetBlockHeightRequest{})
	if err != nil {
		return 0, fmt.Errorf("failed to get block height: %w", err)
	}

	return resp.Height, nil
}

// RegisterDVENode registers a DVE node on-chain
func (nc *NRNClient) RegisterDVENode(nodeID, ownerAddress string, stakeAmount int64) (string, error) {
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
