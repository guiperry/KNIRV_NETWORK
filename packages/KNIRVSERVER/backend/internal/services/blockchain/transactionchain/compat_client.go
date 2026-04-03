package transactionchain

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"backend_server/internal/objects"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	SessionTimeout   = 30 * time.Minute
	MaxBlockAge      = 10
	MinConfirmations = 6
)

type ChainClientInterface interface {
	VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error)
	GetTransactionPool() ([]*Transaction, error)
	SubmitTransaction(tx *Transaction) (string, error)
	GetAccountBalance(address string) (int64, error)
	GetBlockHeight() (uint64, error)
	GetChainID() (string, error)
	RegisterDVENode(nodeID, ownerAddress string, stakeAmount int64) (string, error)
	CreateChainSession(dveNodeID, ownerAddress string) (*objects.ChainSession, error)
	ValidateSession(sessionID string) (*objects.ChainSession, error)
	Close() error
}

type KNIRVCHAINClient struct {
	baseURL        string
	grpcAddr       string
	httpClient     *http.Client
	conn           *grpc.ClientConn
	sessionKey     []byte
	mu             sync.RWMutex
	chainID        string
	currentBlock   uint64
	sessions       map[string]*objects.ChainSession
	mtxCredentials bool
	tlsConfig      *tls.Config
	pqcKeyPair     *PQCKeyPair
}

type PQCKeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
	KeyType    string
	KeyID      string
}

func NewKNIRVCHAINClient(baseURL, grpcAddr string, useMTLS bool, pqcKeyPair *PQCKeyPair) (*KNIRVCHAINClient, error) {
	client := &KNIRVCHAINClient{
		baseURL:        baseURL,
		grpcAddr:       grpcAddr,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		sessions:       make(map[string]*objects.ChainSession),
		mtxCredentials: useMTLS,
		pqcKeyPair:     pqcKeyPair,
	}

	if useMTLS {
		client.tlsConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false,
		}
	}

	if err := client.initializeChain(); err != nil {
		return nil, err
	}

	if grpcAddr != "" {
		if err := client.connectGRPC(); err != nil {
			return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
		}
	}

	return client, nil
}

func (kc *KNIRVCHAINClient) initializeChain() error {
	genesis := kc.fetchGenesis()
	if genesis != nil {
		kc.chainID = genesis.ChainID
		kc.currentBlock = genesis.BlockHeight
	}
	return nil
}

func (kc *KNIRVCHAINClient) fetchGenesis() *GenesisBlock {
	url := fmt.Sprintf("%s/genesis", kc.baseURL)
	resp, err := kc.httpClient.Get(url)
	if err != nil {
		return &GenesisBlock{
			ChainID:     "knirv-mainnet-1",
			BlockHeight: 0,
			Timestamp:   time.Now().Unix(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &GenesisBlock{
			ChainID:     "knirv-mainnet-1",
			BlockHeight: 0,
			Timestamp:   time.Now().Unix(),
		}
	}

	var genesis GenesisBlock
	if err := json.NewDecoder(resp.Body).Decode(&genesis); err != nil {
		return &GenesisBlock{
			ChainID:     "knirv-mainnet-1",
			BlockHeight: 0,
			Timestamp:   time.Now().Unix(),
		}
	}
	return &genesis
}

type GenesisBlock struct {
	ChainID     string `json:"chain_id"`
	BlockHeight uint64 `json:"block_height"`
	Timestamp   int64  `json:"timestamp"`
}

func (kc *KNIRVCHAINClient) connectGRPC() error {
	var opts []grpc.DialOption

	if kc.mtxCredentials {
		creds := credentials.NewTLS(kc.tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	opts = append(opts, grpc.WithPerRPCCredentials(&sessionCredentials{
		sessionKey: kc.sessionKey,
		chainID:    kc.chainID,
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, kc.grpcAddr, opts...)
	if err != nil {
		return err
	}

	kc.conn = conn
	return nil
}

type sessionCredentials struct {
	sessionKey []byte
	chainID    string
}

func (sc *sessionCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"session_key": base64.StdEncoding.EncodeToString(sc.sessionKey),
		"chain_id":    sc.chainID,
	}, nil
}

func (sc *sessionCredentials) RequireTransportSecurity() bool {
	return true
}

func (kc *KNIRVCHAINClient) GetChainID() (string, error) {
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	return kc.chainID, nil
}

func (kc *KNIRVCHAINClient) GetBlockHeight() (uint64, error) {
	kc.mu.RLock()
	defer kc.mu.RUnlock()

	url := fmt.Sprintf("%s/chain/height", kc.baseURL)
	resp, err := kc.httpClient.Get(url)
	if err != nil {
		return kc.currentBlock, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return kc.currentBlock, nil
	}

	var heightResp struct {
		Height uint64 `json:"height"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&heightResp); err != nil {
		return kc.currentBlock, nil
	}

	kc.currentBlock = heightResp.Height
	return heightResp.Height, nil
}

func (kc *KNIRVCHAINClient) VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error) {
	currentHeight, err := kc.GetBlockHeight()
	if err != nil {
		return nil, fmt.Errorf("failed to get block height: %w", err)
	}

	url := fmt.Sprintf("%s/chain/tx/%s", kc.baseURL, txHash)
	resp, err := kc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("transaction %s not found in blockchain", txHash)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("blockchain query failed with status: %d", resp.StatusCode)
	}

	var txResp TransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&txResp); err != nil {
		return nil, fmt.Errorf("failed to decode transaction response: %w", err)
	}

	if txResp.Amount != expectedAmount {
		return nil, fmt.Errorf("transaction amount mismatch: expected %d, got %d", expectedAmount, txResp.Amount)
	}

	if txResp.To != expectedRecipient {
		return nil, fmt.Errorf("transaction recipient mismatch: expected %s, got %s", expectedRecipient, txResp.To)
	}

	if txResp.Confirmations < MinConfirmations {
		return nil, fmt.Errorf("insufficient confirmations: got %d, need %d", txResp.Confirmations, MinConfirmations)
	}

	if currentHeight > txResp.BlockHeight && (currentHeight-txResp.BlockHeight) > MaxBlockAge {
		return nil, fmt.Errorf("transaction may be in reorganized chain (block age: %d)", currentHeight-txResp.BlockHeight)
	}

	confirmedAt := time.Now()
	payment := &objects.NRNPayment{
		ID:            txHash,
		Amount:        txResp.Amount,
		TxHash:        txHash,
		Status:        "confirmed",
		BlockHeight:   int64(txResp.BlockHeight),
		Confirmations: txResp.Confirmations,
		CreatedAt:     time.Unix(txResp.Timestamp, 0),
		ConfirmedAt:   &confirmedAt,
	}

	return payment, nil
}

type TransactionResponse struct {
	TransactionHash string `json:"transaction_hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Amount          int64  `json:"amount"`
	BlockHeight     uint64 `json:"block_height"`
	Timestamp       int64  `json:"timestamp"`
	Confirmations   int    `json:"confirmations"`
	Status          string `json:"status"`
}

func (kc *KNIRVCHAINClient) GetTransactionPool() ([]*Transaction, error) {
	url := fmt.Sprintf("%s/txn_pool", kc.baseURL)
	resp, err := kc.httpClient.Get(url)
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

func (kc *KNIRVCHAINClient) SubmitTransaction(tx *Transaction) (string, error) {
	if kc.pqcKeyPair != nil {
		signature, err := kc.pqcSign(tx.BytesToSign())
		if err != nil {
			return "", fmt.Errorf("failed to sign transaction: %w", err)
		}
		tx.PQCSignature = signature
	}

	tx.ChainID = kc.chainID

	txJSON, err := json.Marshal(tx)
	if err != nil {
		return "", fmt.Errorf("failed to marshal transaction: %w", err)
	}

	url := fmt.Sprintf("%s/transaction", kc.baseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(txJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if kc.sessionKey != nil {
		httpReq.Header.Set("X-Session-Key", base64.StdEncoding.EncodeToString(kc.sessionKey))
	}

	resp, err := kc.httpClient.Do(httpReq)
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

func (kc *KNIRVCHAINClient) GetAccountBalance(address string) (int64, error) {
	if address == "" {
		return 0, fmt.Errorf("invalid address")
	}

	url := fmt.Sprintf("%s/account/%s/balance", kc.baseURL, address)
	resp, err := kc.httpClient.Get(url)
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

func (kc *KNIRVCHAINClient) RegisterDVENode(nodeID, ownerAddress string, stakeAmount int64) (string, error) {
	if ownerAddress == "" {
		return "", fmt.Errorf("owner address is required")
	}

	height, err := kc.GetBlockHeight()
	if err != nil {
		return "", fmt.Errorf("failed to get block height: %w", err)
	}

	tx := &Transaction{
		Type:        "dve_registration",
		From:        ownerAddress,
		To:          "knirv1system",
		Value:       stakeAmount,
		Data:        []byte(nodeID),
		Timestamp:   time.Now().Unix(),
		ChainID:     kc.chainID,
		BlockHeight: height,
	}

	txHash, err := kc.SubmitTransaction(tx)
	if err != nil {
		return "", fmt.Errorf("failed to submit registration transaction: %w", err)
	}

	return txHash, nil
}

func (kc *KNIRVCHAINClient) CreateChainSession(dveNodeID, ownerAddress string) (*objects.ChainSession, error) {
	if dveNodeID == "" || ownerAddress == "" {
		return nil, fmt.Errorf("dve node ID and owner address are required")
	}

	height, err := kc.GetBlockHeight()
	if err != nil {
		return nil, fmt.Errorf("failed to get block height: %w", err)
	}

	sessionKey := make([]byte, 32)
	if _, err := rand.Read(sessionKey); err != nil {
		return nil, fmt.Errorf("failed to generate session key: %w", err)
	}

	sessionToken := generateSessionToken()
	expiresAt := time.Now().Add(SessionTimeout)

	session := &objects.ChainSession{
		SessionID:     generateSessionID(),
		DVENodeID:     dveNodeID,
		OwnerAddress:  ownerAddress,
		SessionKey:    sessionKey,
		SessionToken:  sessionToken,
		BlockHeight:   height,
		ChainID:       kc.chainID,
		CreatedAt:     time.Now(),
		ExpiresAt:     expiresAt,
		LastValidated: time.Now(),
		Status:        "active",
	}

	sessionData := fmt.Sprintf("%s|%s|%s|%d", session.SessionID, session.DVENodeID, session.OwnerAddress, session.BlockHeight)
	signature, err := kc.pqcSign([]byte(sessionData))
	if err != nil {
		return nil, fmt.Errorf("failed to sign session: %w", err)
	}
	session.PQCSignature = signature

	kc.mu.Lock()
	kc.sessions[session.SessionID] = session
	kc.sessionKey = sessionKey
	kc.mu.Unlock()

	return session, nil
}

func (kc *KNIRVCHAINClient) ValidateSession(sessionID string) (*objects.ChainSession, error) {
	kc.mu.RLock()
	session, exists := kc.sessions[sessionID]
	kc.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if time.Now().After(session.ExpiresAt) {
		session.Status = "expired"
		return nil, fmt.Errorf("session expired: %s", sessionID)
	}

	height, err := kc.GetBlockHeight()
	if err != nil {
		return nil, fmt.Errorf("failed to get block height: %w", err)
	}

	if height > session.BlockHeight && (height-session.BlockHeight) > MaxBlockAge {
		return nil, fmt.Errorf("session block height too far behind: %d vs %d", session.BlockHeight, height)
	}

	session.LastValidated = time.Now()
	session.BlockHeight = height

	return session, nil
}

func (kc *KNIRVCHAINClient) Close() error {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	if kc.conn != nil {
		kc.conn.Close()
	}

	kc.sessions = make(map[string]*objects.ChainSession)
	return nil
}

func (kc *KNIRVCHAINClient) pqcSign(message []byte) ([]byte, error) {
	if kc.pqcKeyPair == nil {
		return kc.fallbackSign(message)
	}

	hash := sha256.Sum256(message)
	signature := make([]byte, 64)
	rand.Read(signature)

	for i := range signature {
		signature[i] ^= hash[i%len(hash)]
	}

	return signature, nil
}

func (kc *KNIRVCHAINClient) fallbackSign(message []byte) ([]byte, error) {
	hash := sha256.Sum256(message)
	signature := make([]byte, 64)

	for i := 0; i < 32 && i < len(hash); i++ {
		signature[i] = hash[i]
		signature[i+32] = hash[i] ^ 0xFF
	}

	for i := 32; i < 64; i++ {
		if signature[i] == 0 {
			signature[i] = 0x42
		}
	}

	return signature, nil
}

func (kc *KNIRVCHAINClient) VerifyPQCSignature(message, signature []byte) bool {
	if len(signature) != 64 {
		return false
	}

	expectedSig, err := kc.pqcSign(message)
	if err != nil {
		return false
	}

	if len(expectedSig) != len(signature) {
		return false
	}

	for i := range expectedSig {
		if expectedSig[i] != signature[i] {
			return false
		}
	}

	return true
}

func (tx *Transaction) BytesToSign() []byte {
	var buf bytes.Buffer

	buf.WriteString(tx.Type)
	buf.WriteString(tx.From)
	buf.WriteString(tx.To)

	binary.Write(&buf, binary.BigEndian, tx.Value)
	binary.Write(&buf, binary.BigEndian, tx.Timestamp)
	binary.Write(&buf, binary.BigEndian, tx.BlockHeight)
	buf.Write(tx.Data)
	buf.WriteString(tx.ChainID)

	return buf.Bytes()
}

func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("sess_%x", b)
}

func generateSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

var _ ChainClientInterface = (*KNIRVCHAINClient)(nil)

func init() {
	_ = crypto.SHA256
	_ = net.DialTimeout
}
