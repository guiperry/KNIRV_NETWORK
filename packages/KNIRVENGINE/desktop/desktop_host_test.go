// desktop_host_test.go
package desktop

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockHRMEngine implements HRMEngine interface for testing
type MockHRMEngine struct {
	initialized bool
	running     bool
	mu          sync.RWMutex
}

func NewMockHRMEngine() *MockHRMEngine {
	return &MockHRMEngine{}
}

func (m *MockHRMEngine) LoadHRMModule(wasmPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initialized = true
	return nil
}

func (m *MockHRMEngine) ProcessCognitive(input *HRMInput) (*HRMOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return nil, fmt.Errorf("HRM engine not initialized")
	}

	return &HRMOutput{
		ReasoningResult:    fmt.Sprintf("Processed: %s", input.Context),
		Confidence:         0.85,
		ProcessingTime:     0.1,
		LModuleActivations: []float32{0.1, 0.2, 0.3},
		HModuleActivations: []float32{0.4, 0.5, 0.6},
	}, nil
}

func (m *MockHRMEngine) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

func (m *MockHRMEngine) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
	return nil
}

// Close implements the HRMEngineInterface Close method for tests
func (m *MockHRMEngine) Close() error {
	return m.Shutdown()
}

// MockQRLinkageService implements QRLinkageService interface for testing
type MockQRLinkageService struct {
	qrCodes map[string]*QRLinkageInfo
	mu      sync.RWMutex
}

func NewMockQRLinkageService() *MockQRLinkageService {
	return &MockQRLinkageService{
		qrCodes: make(map[string]*QRLinkageInfo),
	}
}

// GetModelInfo returns mock model info for HRM engine
func (m *MockHRMEngine) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{"model": "mock-hrm", "wasm_loaded": false}
}

func (m *MockHRMEngine) InitializeModules(lCount, hCount uint32) error {
	// Mock initialize
	return nil
}

func (m *MockHRMEngine) LoadWeights(weightsPath string) error {
	// Mock loading weights
	return nil
}

func (m *MockHRMEngine) ProcessCognitiveInput(input *HRMInput) (*HRMOutput, error) {
	return m.ProcessCognitive(input)
}

func (m *MockQRLinkageService) GenerateQRCode(deviceID string) (*QRLinkageInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	qrInfo := &QRLinkageInfo{
		QRCode:    fmt.Sprintf("qr_code_%s", deviceID),
		DeviceID:  deviceID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Used:      false,
	}

	m.qrCodes[deviceID] = qrInfo
	return qrInfo, nil
}

// Implement interface methods expected by DesktopClient tests
func (m *MockQRLinkageService) GenerateTargetAssignmentQR(targetSystemID string, capabilities []string) (*QRCode, error) {
	// Return a mock QRCode
	qr := &QRCode{
		Data:      []byte("mock_target_qr"),
		Image:     []byte("img"),
		SessionID: "session-" + targetSystemID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	return qr, nil
}

func (m *MockQRLinkageService) GenerateTransactionSignQR(transactionData *TransactionData) (*QRCode, error) {
	qr := &QRCode{
		Data:      []byte("mock_tx_qr"),
		Image:     []byte("img"),
		SessionID: "session-tx",
		ExpiresAt: time.Now().Add(2 * time.Minute),
	}
	return qr, nil
}

func (m *MockQRLinkageService) GetSession(sessionID string) (*LinkageSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// For tests, return a dummy pending session
	session := &LinkageSession{
		SessionID:   sessionID,
		DesktopID:   "test-desktop",
		LinkageType: LinkageTypeTargetAssignment,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
		Status:      LinkageStatusPending,
	}
	return session, true
}

func (m *MockQRLinkageService) StartService() {
	// No-op for tests
}

func (m *MockQRLinkageService) UpdateSessionStatus(sessionID string, status LinkageStatus, mobileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.qrCodes[mobileID]; ok {
		s.Used = true
	}
	return nil
}

func (m *MockQRLinkageService) ValidateQRCode(qrCode string) (*QRLinkageInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, info := range m.qrCodes {
		if info.QRCode == qrCode && !info.Used && time.Now().Before(info.ExpiresAt) {
			return info, nil
		}
	}

	return nil, fmt.Errorf("invalid or expired QR code")
}

// Test helper functions
func createTestDesktopClient(t *testing.T) *DesktopClient {
	tempDir, err := os.MkdirTemp("", "test_desktop")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	client := &DesktopClient{
		hrmEngine:         NewMockHRMEngine(),
		qrLinkage:         NewMockQRLinkageService(),
		mobileConnections: make(map[string]*MobileConnection),
		agentSessions:     make(map[string]*AgentSession),
		desktopID:         "test-desktop-id",
		endpoint:          "http://localhost:8080",
		publicKey:         "test-public-key",
		running:           false,
	}

	return client
}

// TestDesktopClient_NewDesktopClient tests the constructor
func TestDesktopClient_NewDesktopClient(t *testing.T) {
	client := createTestDesktopClient(t)

	assert.NotNil(t, client)
	assert.NotEmpty(t, client.desktopID)
	assert.NotNil(t, client.mobileConnections)
	assert.NotNil(t, client.agentSessions)
	assert.False(t, client.running)
}

// TestDesktopClient_Initialize tests desktop client initialization
func TestDesktopClient_Initialize(t *testing.T) {
	client := createTestDesktopClient(t)

	err := client.Initialize()

	assert.NoError(t, err)
	assert.True(t, client.hrmEngine.IsInitialized())
}

// TestDesktopClient_Start tests starting the desktop client
func TestDesktopClient_Start(t *testing.T) {
	client := createTestDesktopClient(t)

	err := client.Initialize()
	require.NoError(t, err)

	// Start in a goroutine since it's blocking
	go func() {
		err := client.Start()
		assert.NoError(t, err)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	assert.True(t, client.IsRunning())

	// Stop the client
	err = client.Stop()
	assert.NoError(t, err)
}

// TestDesktopClient_Stop tests stopping the desktop client
func TestDesktopClient_Stop(t *testing.T) {
	client := createTestDesktopClient(t)

	err := client.Initialize()
	require.NoError(t, err)

	// Start and then stop
	go func() {
		client.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	err = client.Stop()
	assert.NoError(t, err)
	assert.False(t, client.IsRunning())
}

// TestDesktopClient_RegisterMobileDevice tests mobile device registration
func TestDesktopClient_RegisterMobileDevice(t *testing.T) {
	client := createTestDesktopClient(t)

	deviceID := "test-mobile-device"
	walletAddress := "0x1234567890abcdef"
	publicKey := "test-mobile-public-key"

	err := client.RegisterMobileDevice(deviceID, walletAddress, publicKey)

	assert.NoError(t, err)

	// Check that device was registered
	client.mutex.RLock()
	connection, exists := client.mobileConnections[deviceID]
	client.mutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, deviceID, connection.DeviceID)
	assert.Equal(t, walletAddress, connection.WalletAddress)
	assert.Equal(t, publicKey, connection.PublicKey)
}

// TestDesktopClient_UnregisterMobileDevice tests mobile device unregistration
func TestDesktopClient_UnregisterMobileDevice(t *testing.T) {
	client := createTestDesktopClient(t)

	deviceID := "test-mobile-device"

	// Register first
	err := client.RegisterMobileDevice(deviceID, "0x1234", "pubkey")
	require.NoError(t, err)

	// Then unregister
	err = client.UnregisterMobileDevice(deviceID)

	assert.NoError(t, err)

	// Check that device was unregistered
	client.mutex.RLock()
	_, exists := client.mobileConnections[deviceID]
	client.mutex.RUnlock()

	assert.False(t, exists)
}

// TestDesktopClient_GetMobileDevices tests getting mobile devices
func TestDesktopClient_GetMobileDevices(t *testing.T) {
	client := createTestDesktopClient(t)

	// Register multiple devices
	devices := []string{"device1", "device2", "device3"}
	for _, deviceID := range devices {
		err := client.RegisterMobileDevice(deviceID, "0x1234", "pubkey")
		require.NoError(t, err)
	}

	mobileDevices := client.GetMobileDevices()

	assert.Len(t, mobileDevices, 3)

	// Check that all devices are present
	deviceIDs := make([]string, len(mobileDevices))
	for i, device := range mobileDevices {
		deviceIDs[i] = device.DeviceID
	}

	for _, expectedID := range devices {
		assert.Contains(t, deviceIDs, expectedID)
	}
}

// TestDesktopClient_CreateAgentSession tests agent session creation
func TestDesktopClient_CreateAgentSession(t *testing.T) {
	client := createTestDesktopClient(t)

	agentID := "test-agent"
	deviceID := "test-device"

	sessionID, err := client.CreateAgentSession(agentID, deviceID)

	assert.NoError(t, err)
	assert.NotEmpty(t, sessionID)

	// Check that session was created
	client.mutex.RLock()
	session, exists := client.agentSessions[sessionID]
	client.mutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, agentID, session.AgentID)
	assert.Equal(t, deviceID, session.DeviceID)
}

// TestDesktopClient_CloseAgentSession tests agent session closure
func TestDesktopClient_CloseAgentSession(t *testing.T) {
	client := createTestDesktopClient(t)

	agentID := "test-agent"
	deviceID := "test-device"

	// Create session first
	sessionID, err := client.CreateAgentSession(agentID, deviceID)
	require.NoError(t, err)

	// Then close it
	err = client.CloseAgentSession(sessionID)

	assert.NoError(t, err)

	// Check that session was closed
	client.mutex.RLock()
	_, exists := client.agentSessions[sessionID]
	client.mutex.RUnlock()

	assert.False(t, exists)
}

// TestDesktopClient_ConcurrentAccess tests thread safety
func TestDesktopClient_ConcurrentAccess(t *testing.T) {
	client := createTestDesktopClient(t)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent device registration
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			deviceID := fmt.Sprintf("device-%d", id)
			err := client.RegisterMobileDevice(deviceID, "0x1234", "pubkey")
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Check that all devices were registered
	devices := client.GetMobileDevices()
	assert.Len(t, devices, numGoroutines)
}

// TestDesktopClient_HTTPHandlers tests HTTP endpoint handlers
func TestDesktopClient_HTTPHandlers(t *testing.T) {
	client := createTestDesktopClient(t)

	err := client.Initialize()
	require.NoError(t, err)

	// Test status endpoint
	req := httptest.NewRequest("GET", "/status", nil)
	recorder := httptest.NewRecorder()

	client.handleStatus(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "desktop_id")
}
