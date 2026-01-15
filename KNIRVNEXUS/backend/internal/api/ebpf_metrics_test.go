package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend_server/internal/ebpf"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEBPFManager is a mock implementation of the eBPF ManagerInterface
type MockEBPFManager struct {
	mock.Mock
}

// Ensure MockEBPFManager implements ebpf.ManagerInterface
var _ ebpf.ManagerInterface = (*MockEBPFManager)(nil)

func (m *MockEBPFManager) GetMetrics() *ebpf.Metrics {
	args := m.Called()
	return args.Get(0).(*ebpf.Metrics)
}

func (m *MockEBPFManager) Initialize(ctx context.Context, config *ebpf.Config) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockEBPFManager) Shutdown() error {
	args := m.Called()
	return args.Error(0)
}

// ebpfManagerWrapper wraps MockEBPFManager to implement the ebpf.Manager interface
type ebpfManagerWrapper struct {
	mock *MockEBPFManager
}

func (w *ebpfManagerWrapper) GetMetrics() *ebpf.Metrics {
	return w.mock.GetMetrics()
}

func (w *ebpfManagerWrapper) Initialize(ctx context.Context, config *ebpf.Config) error {
	return w.mock.Initialize(ctx, config)
}

func (w *ebpfManagerWrapper) Shutdown() error {
	return w.mock.Shutdown()
}

// minimalEBPFMock is a simple mock that implements just the methods needed for testing
type minimalEBPFMock struct {
	metrics *ebpf.Metrics
}

func (m *minimalEBPFMock) GetMetrics() *ebpf.Metrics {
	return m.metrics
}

func (m *minimalEBPFMock) Initialize(ctx context.Context, config *ebpf.Config) error {
	return nil
}

func (m *minimalEBPFMock) Shutdown() error {
	return nil
}

func TestRegisterEBPFMetrics(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router
	router := gin.New()

	// Create mock eBPF manager
	mockManager := &MockEBPFManager{}

	// Mock the GetMetrics call
	expectedMetrics := &ebpf.Metrics{
		Initialized:      true,
		ProgramsAttached: 5,
	}
	mockManager.On("GetMetrics").Return(expectedMetrics)

	// Register the metrics endpoint
	RegisterEBPFMetrics(router, mockManager)

	// Create a test request
	req, err := http.NewRequest("GET", "/api/v1/ebpf/metrics", nil)
	assert.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Assert the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse the response body
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Assert the response content
	assert.Equal(t, float64(5), response["programs_loaded"])
	assert.Equal(t, true, response["initialized"])
}

func TestRegisterEBPFMetrics_WithUninitializedManager(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router
	router := gin.New()

	// Create mock eBPF manager
	mockManager := &MockEBPFManager{}

	// Mock the GetMetrics call with uninitialized state
	expectedMetrics := &ebpf.Metrics{
		Initialized:      false,
		ProgramsAttached: 0,
	}
	mockManager.On("GetMetrics").Return(expectedMetrics)

	// Register the metrics endpoint
	RegisterEBPFMetrics(router, mockManager)

	// Create a test request
	req, err := http.NewRequest("GET", "/api/v1/ebpf/metrics", nil)
	assert.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Assert the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse the response body
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Assert the response content
	assert.Equal(t, float64(0), response["programs_loaded"])
	assert.Equal(t, false, response["initialized"])

	// Verify that the mock was called
	mockManager.AssertExpectations(t)
}
