package dvecreation

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/objects"
	"backend_server/internal/services/blockchain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockChainClient struct {
	registeredNodes map[string]string
	sessions        map[string]*objects.ChainSession
	currentHeight   uint64
	chainID         string
}

func NewMockChainClient() *MockChainClient {
	return &MockChainClient{
		registeredNodes: make(map[string]string),
		sessions:        make(map[string]*objects.ChainSession),
		currentHeight:   100,
		chainID:         "knirv-test-1",
	}
}

func (m *MockChainClient) VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error) {
	return &objects.NRNPayment{
		ID:            txHash,
		Amount:        expectedAmount,
		TxHash:        txHash,
		Status:        "confirmed",
		BlockHeight:   100,
		Confirmations: 6,
		CreatedAt:     time.Now(),
		ConfirmedAt:   &time.Time{},
	}, nil
}

func (m *MockChainClient) GetTransactionPool() ([]*blockchain.Transaction, error) {
	return nil, nil
}

func (m *MockChainClient) SubmitTransaction(tx *blockchain.Transaction) (string, error) {
	return "mock-tx-hash", nil
}

func (m *MockChainClient) GetAccountBalance(address string) (int64, error) {
	return 1000000, nil
}

func (m *MockChainClient) GetBlockHeight() (uint64, error) {
	return m.currentHeight, nil
}

func (m *MockChainClient) GetChainID() (string, error) {
	return m.chainID, nil
}

func (m *MockChainClient) RegisterDVENode(nodeID, ownerAddress string, stakeAmount int64) (string, error) {
	txHash := "reg-" + nodeID
	m.registeredNodes[nodeID] = txHash
	return txHash, nil
}

func (m *MockChainClient) CreateChainSession(dveNodeID, ownerAddress string) (*objects.ChainSession, error) {
	session := &objects.ChainSession{
		SessionID:     "session-" + dveNodeID,
		DVENodeID:     dveNodeID,
		OwnerAddress:  ownerAddress,
		SessionKey:    []byte("test-session-key"),
		SessionToken:  "test-token",
		BlockHeight:   m.currentHeight,
		ChainID:       m.chainID,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(30 * time.Minute),
		LastValidated: time.Now(),
		Status:        "active",
		PQCSignature:  []byte("test-sig"),
	}
	m.sessions[session.SessionID] = session
	return session, nil
}

func (m *MockChainClient) ValidateSession(sessionID string) (*objects.ChainSession, error) {
	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, nil
	}
	return session, nil
}

func (m *MockChainClient) Close() error {
	return nil
}

func setupTestDVECreationService(t *testing.T) (*DVECreationService, *database.BuntDBManager) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_dve_creation.db")

	db, err := database.NewBuntDB(dbPath)
	require.NoError(t, err)

	service, err := NewDVECreationService(db)
	require.NoError(t, err)

	t.Cleanup(func() {
		if service.running {
			service.Stop()
		}
		db.Close()
		os.Remove(dbPath)
	})

	return service, db
}

func TestNewDVECreationService(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
	assert.False(t, service.running)
	assert.NotNil(t, service.activeCreations)
	assert.NotNil(t, service.activeSessions)
	assert.Equal(t, int64(1000), service.minStakeAmount)
}

func TestDVECreationService_SetChainClient(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	service.mu.RLock()
	defer service.mu.RUnlock()
	assert.Equal(t, mockClient, service.chainClient)
}

func TestDVECreationService_CreateDVENode(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:        "user-123",
		OwnerAddress:   "knirv1owner123",
		Name:           "My DVE Node",
		TEEType:        "sgx",
		TEEAttestation: "test-attestation-quote",
		StakeAmount:    5000,
		Capabilities:   []string{"reasoning", "validation"},
		ResourceLimits: objects.ResourceLimits{
			MaxCPU:    4.0,
			MaxMemory: 8 * 1024 * 1024 * 1024,
		},
		Persistent: true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.DVECreation)
	assert.NotNil(t, resp.Session)
	assert.NotEmpty(t, resp.DVECreation.DVENodeID)
	assert.Equal(t, "user-123", resp.DVECreation.OwnerID)
	assert.Equal(t, "active", resp.DVECreation.Status)
}

func TestDVECreationService_CreateDVENode_InsufficientStake(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "My DVE Node",
		StakeAmount:  500, // Less than minStakeAmount
		Persistent:   true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "minimum stake amount")
}

func TestDVECreationService_CreateDVENode_MissingOwner(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "",
		OwnerAddress: "knirv1owner123",
		Name:         "My DVE Node",
		StakeAmount:  5000,
		Persistent:   true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "owner ID and address are required")
}

func TestDVECreationService_GetDVECreation(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "Test DVE",
		StakeAmount:  5000,
		Persistent:   true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)

	creation, err := service.GetDVECreation(resp.DVECreation.ID)
	require.NoError(t, err)
	assert.Equal(t, resp.DVECreation.ID, creation.ID)
	assert.Equal(t, "user-123", creation.OwnerID)
}

func TestDVECreationService_GetUserDVECreations(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req1 := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "DVE 1",
		StakeAmount:  5000,
		Persistent:   true,
	}

	req2 := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "DVE 2",
		StakeAmount:  5000,
		Persistent:   true,
	}

	req3 := &objects.DVECreationRequest{
		OwnerID:      "user-456",
		OwnerAddress: "knirv1owner456",
		Name:         "DVE 3",
		StakeAmount:  5000,
		Persistent:   true,
	}

	_, err = service.CreateDVENode(req1)
	require.NoError(t, err)
	_, err = service.CreateDVENode(req2)
	require.NoError(t, err)
	_, err = service.CreateDVENode(req3)
	require.NoError(t, err)

	creations, err := service.GetUserDVECreations("user-123")
	require.NoError(t, err)
	assert.Len(t, creations, 2)
}

func TestDVECreationService_GetDVESession(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "Test DVE",
		StakeAmount:  5000,
		Persistent:   true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)

	session, err := service.GetDVESession(resp.Session.ID)
	require.NoError(t, err)
	assert.Equal(t, resp.Session.ID, session.ID)
}

func TestDVECreationService_RefreshSession(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "Test DVE",
		StakeAmount:  5000,
		Persistent:   true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)

	oldSessionID := resp.Session.ID

	newSession, err := service.RefreshSession(resp.DVECreation.ID)
	require.NoError(t, err)
	assert.NotEqual(t, oldSessionID, newSession.ID)
	assert.Equal(t, "active", newSession.Status)
}

func TestDVECreationService_RevokeSession(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "Test DVE",
		StakeAmount:  5000,
		Persistent:   true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)

	err = service.RevokeSession(resp.Session.ID)
	require.NoError(t, err)

	session, err := service.GetDVESession(resp.Session.ID)
	require.NoError(t, err)
	assert.Equal(t, "revoked", session.Status)
}

func TestDVECreationService_DecommissionDVENode(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "Test DVE",
		StakeAmount:  5000,
		Persistent:   true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)

	err = service.DecommissionDVENode(resp.DVECreation.ID, "user-123")
	require.NoError(t, err)

	creation, err := service.GetDVECreation(resp.DVECreation.ID)
	require.NoError(t, err)
	assert.Equal(t, "decommissioned", creation.Status)
}

func TestDVECreationService_DecommissionDVENode_Unauthorized(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "Test DVE",
		StakeAmount:  5000,
		Persistent:   true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)

	err = service.DecommissionDVENode(resp.DVECreation.ID, "user-456")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestDVECreationService_Heartbeat(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "Test DVE",
		StakeAmount:  5000,
		Persistent:   true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)

	originalHeartbeat := resp.DVECreation.LastHeartbeat

	err = service.Heartbeat(resp.DVECreation.ID)
	require.NoError(t, err)

	updatedCreation, err := service.GetDVECreation(resp.DVECreation.ID)
	require.NoError(t, err)
	assert.True(t, updatedCreation.LastHeartbeat.After(originalHeartbeat) || updatedCreation.LastHeartbeat.Equal(originalHeartbeat))
}

func TestDVECreationService_GetStats(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "Test DVE",
		StakeAmount:  5000,
		Persistent:   true,
	}

	_, err = service.CreateDVENode(req)
	require.NoError(t, err)

	stats, err := service.GetStats()
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.TotalCreations)
	assert.GreaterOrEqual(t, stats.ActiveCreations, int64(0))
}

func TestDVECreationService_Start(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	err := service.Start()
	require.NoError(t, err)
	assert.True(t, service.running)

	err = service.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	service.Stop()
}

func TestDVECreationService_Stop(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	err := service.Stop()
	assert.NoError(t, err)

	err = service.Start()
	require.NoError(t, err)

	err = service.Stop()
	assert.NoError(t, err)
	assert.False(t, service.running)
}

func TestDVECreationService_IsRunning(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	assert.False(t, service.IsRunning())

	err := service.Start()
	require.NoError(t, err)

	assert.True(t, service.IsRunning())

	service.Stop()
	assert.False(t, service.IsRunning())
}

func TestDVECreationService_GetDVECreationByNodeID(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "Test DVE",
		StakeAmount:  5000,
		Persistent:   true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)

	creation, err := service.GetDVECreationByNodeID(resp.DVECreation.DVENodeID)
	require.NoError(t, err)
	assert.Equal(t, resp.DVECreation.ID, creation.ID)
}

func TestDVECreationService_GetActiveSessionByCreationID(t *testing.T) {
	service, _ := setupTestDVECreationService(t)

	mockClient := NewMockChainClient()
	service.SetChainClient(mockClient)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	req := &objects.DVECreationRequest{
		OwnerID:      "user-123",
		OwnerAddress: "knirv1owner123",
		Name:         "Test DVE",
		StakeAmount:  5000,
		Persistent:   true,
	}

	resp, err := service.CreateDVENode(req)
	require.NoError(t, err)

	session, err := service.GetActiveSessionByCreationID(resp.DVECreation.ID)
	require.NoError(t, err)
	assert.Equal(t, resp.Session.ID, session.ID)
	assert.Equal(t, "active", session.Status)
}
