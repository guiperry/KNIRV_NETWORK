package data_engine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"backend_server/internal/messages"
)

func TestNewDataEngine(t *testing.T) {
	config := DataEngineConfig{
		EnableKafka:     false,
		EnableChromaDB:  false,
		EnableWebSocket: false,
		EnableRESTAPI:   false,
		WindowSize:      time.Minute,
		MetricsInterval: time.Second * 30,
	}

	engine := NewDataEngine(config)

	assert.NotNil(t, engine)
	assert.False(t, engine.IsRunning())
	assert.NotNil(t, engine.alertChan)
	assert.NotNil(t, engine.metricsChan)
	assert.Equal(t, config, engine.config)
}

func TestDataEngine_GetAlertChannel(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})

	channel := engine.GetAlertChannel()
	assert.NotNil(t, channel)
}

func TestDataEngine_GetMetricsChannel(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})

	channel := engine.GetMetricsChannel()
	assert.NotNil(t, channel)
}

func TestDataEngine_Start_Stop(t *testing.T) {
	config := DataEngineConfig{
		EnableKafka:     false,
		EnableChromaDB:  false,
		EnableWebSocket: false,
		EnableRESTAPI:   false,
		WindowSize:      time.Minute,
		MetricsInterval: time.Second * 30,
	}

	engine := NewDataEngine(config)

	// Test Start
	err := engine.Start()
	assert.NoError(t, err)
	assert.True(t, engine.IsRunning())

	// Test Stop
	err = engine.Stop()
	assert.NoError(t, err)
	assert.False(t, engine.IsRunning())
}

func TestDataEngine_Start_AlreadyRunning(t *testing.T) {
	config := DataEngineConfig{
		EnableKafka:     false,
		EnableChromaDB:  false,
		EnableWebSocket: false,
		EnableRESTAPI:   false,
		WindowSize:      time.Minute,
		MetricsInterval: time.Second * 30,
	}

	engine := NewDataEngine(config)

	// Start first time
	err := engine.Start()
	assert.NoError(t, err)
	assert.True(t, engine.IsRunning())

	// Try to start again
	err = engine.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Cleanup
	engine.Stop()
}

func TestDataEngine_ProcessEvent_NotRunning(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})

	event := Event{
		Type:      "test",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"key": "value"},
	}

	err := engine.ProcessEvent(event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestDataEngine_ProcessBlockchainEvent(t *testing.T) {
	config := DataEngineConfig{
		EnableKafka:     false,
		EnableChromaDB:  false,
		EnableWebSocket: false,
		EnableRESTAPI:   false,
		WindowSize:      time.Minute,
		MetricsInterval: time.Second * 30,
	}

	engine := NewDataEngine(config)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	msg := messages.BlockchainEventMsg{
		Type:      "block_created",
		Source:    "blockchain",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"transactions": 5},
	}

	err = engine.ProcessBlockchainEvent(msg)
	assert.NoError(t, err)
}

func TestDataEngine_ProcessNetworkUpdate(t *testing.T) {
	config := DataEngineConfig{
		EnableKafka:     false,
		EnableChromaDB:  false,
		EnableWebSocket: false,
		EnableRESTAPI:   false,
		WindowSize:      time.Minute,
		MetricsInterval: time.Second * 30,
	}

	engine := NewDataEngine(config)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	msg := messages.NetworkUpdateMsg{
		Type:      "peer_connected",
		Source:    "network",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"latency": 25},
		PeerCount: 5,
		Latency:   time.Millisecond * 25,
	}

	err = engine.ProcessNetworkUpdate(msg)
	assert.NoError(t, err)
}

func TestDataEngine_ProcessLogMsg(t *testing.T) {
	config := DataEngineConfig{
		EnableKafka:     false,
		EnableChromaDB:  false,
		EnableWebSocket: false,
		EnableRESTAPI:   false,
		WindowSize:      time.Minute,
		MetricsInterval: time.Second * 30,
	}

	engine := NewDataEngine(config)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	msg := messages.LogMsg{
		Level:     "info",
		Message:   "Test log message",
		Source:    "test-service",
		Component: "test-component",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"component": "test"},
	}

	err = engine.ProcessLogMsg(msg)
	assert.NoError(t, err)
}

func TestDataEngine_GetActiveAlerts(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})

	alerts := engine.GetActiveAlerts()
	// Should be nil when alerting system is not initialized
	assert.Nil(t, alerts)
}

func TestDataEngine_ResolveAlert(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})

	// Test resolving non-existent alert
	resolved := engine.ResolveAlert("non-existent")
	assert.False(t, resolved)
}

func TestDataEngine_IsRunning(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})

	// Initially not running
	assert.False(t, engine.IsRunning())

	// Start engine
	config := DataEngineConfig{
		EnableKafka:     false,
		EnableChromaDB:  false,
		EnableWebSocket: false,
		EnableRESTAPI:   false,
		WindowSize:      time.Minute,
		MetricsInterval: time.Second * 30,
	}

	runningEngine := NewDataEngine(config)
	err := runningEngine.Start()
	require.NoError(t, err)
	defer runningEngine.Stop()

	assert.True(t, runningEngine.IsRunning())
}

func TestDataEngine_GetProducer(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})

	producer := engine.GetProducer()
	assert.Nil(t, producer) // Should be nil when not started with Kafka
}

func TestDataEngine_GetChromaDB(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})

	chromaDB := engine.GetChromaDB()
	assert.Nil(t, chromaDB) // Should be nil when not started with ChromaDB
}

func TestDataEngine_GetAlerting(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})

	alerting := engine.GetAlerting()
	assert.Nil(t, alerting) // Should be nil when not started
}

func TestDataEngine_GetAggregator(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})

	aggregator := engine.GetAggregator()
	assert.Nil(t, aggregator) // Should be nil when not started
}

func TestDataEngine_GetMetrics(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})

	metrics := engine.GetMetrics()
	assert.Nil(t, metrics) // Should be nil when not started
}

// Handler tests
func TestNewDataEngineHandlers(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})
	handlers := NewDataEngineHandlers(engine)

	assert.NotNil(t, handlers)
	assert.Equal(t, engine, handlers.dataEngine)
	assert.NotNil(t, handlers.upgrader)
}

func TestDataEngineHandlers_HandleHealth(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})
	handlers := NewDataEngineHandlers(engine)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handlers.HandleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, "data_engine", response["service"])
}

func TestDataEngineHandlers_HandleMetrics_NoEngine(t *testing.T) {
	handlers := NewDataEngineHandlers(nil)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handlers.HandleMetrics(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDataEngineHandlers_HandleMetrics_NoMetrics(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})
	handlers := NewDataEngineHandlers(engine)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handlers.HandleMetrics(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDataEngineHandlers_HandleAlerts_NoEngine(t *testing.T) {
	handlers := NewDataEngineHandlers(nil)

	req := httptest.NewRequest("GET", "/alerts", nil)
	w := httptest.NewRecorder()

	handlers.HandleAlerts(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDataEngineHandlers_HandleEvents_NoEngine(t *testing.T) {
	handlers := NewDataEngineHandlers(nil)

	req := httptest.NewRequest("GET", "/events", nil)
	w := httptest.NewRecorder()

	handlers.HandleEvents(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDataEngineHandlers_HandleEvents_WithLimit(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})
	handlers := NewDataEngineHandlers(engine)

	req := httptest.NewRequest("GET", "/events?limit=50", nil)
	w := httptest.NewRecorder()

	handlers.HandleEvents(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, float64(50), response["limit"])
}

func TestDataEngineHandlers_HandleResolveAlert_InvalidMethod(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})
	handlers := NewDataEngineHandlers(engine)

	req := httptest.NewRequest("GET", "/alerts/123/resolve", nil)
	w := httptest.NewRecorder()

	handlers.HandleResolveAlert(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestDataEngineHandlers_HandleResolveAlert_NoEngine(t *testing.T) {
	handlers := NewDataEngineHandlers(nil)

	req := httptest.NewRequest("POST", "/alerts/123/resolve", nil)
	w := httptest.NewRecorder()

	handlers.HandleResolveAlert(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDataEngineHandlers_HandleResolveAlert_NoID(t *testing.T) {
	engine := NewDataEngine(DataEngineConfig{})
	handlers := NewDataEngineHandlers(engine)

	req := httptest.NewRequest("POST", "/alerts//resolve", nil)
	w := httptest.NewRecorder()

	handlers.HandleResolveAlert(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// BuntDB Handler tests
func TestNewBuntDBDataEngineHandlers(t *testing.T) {
	// Note: This would need a BuntDBDataEngine instance in a real test
	handlers := NewBuntDBDataEngineHandlers(nil)

	assert.NotNil(t, handlers)
	assert.Nil(t, handlers.dataEngine)
	assert.NotNil(t, handlers.upgrader)
}

func TestBuntDBDataEngineHandlers_HandleHealth(t *testing.T) {
	handlers := NewBuntDBDataEngineHandlers(nil)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handlers.HandleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, "buntdb-data_engine", response["service"])
}

func TestBuntDBDataEngineHandlers_HandleMetrics_NoEngine(t *testing.T) {
	handlers := NewBuntDBDataEngineHandlers(nil)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handlers.HandleMetrics(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestBuntDBDataEngineHandlers_HandleAlerts_NoEngine(t *testing.T) {
	handlers := NewBuntDBDataEngineHandlers(nil)

	req := httptest.NewRequest("GET", "/alerts", nil)
	w := httptest.NewRecorder()

	handlers.HandleAlerts(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestBuntDBDataEngineHandlers_HandleEvents_NoEngine(t *testing.T) {
	handlers := NewBuntDBDataEngineHandlers(nil)

	req := httptest.NewRequest("GET", "/events", nil)
	w := httptest.NewRecorder()

	handlers.HandleEvents(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestBuntDBDataEngineHandlers_HandleResolveAlert_InvalidMethod(t *testing.T) {
	handlers := NewBuntDBDataEngineHandlers(nil)

	req := httptest.NewRequest("GET", "/alerts/123/resolve", nil)
	w := httptest.NewRecorder()

	handlers.HandleResolveAlert(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestBuntDBDataEngineHandlers_HandleResolveAlert_NoEngine(t *testing.T) {
	handlers := NewBuntDBDataEngineHandlers(nil)

	req := httptest.NewRequest("POST", "/alerts/123/resolve", nil)
	w := httptest.NewRecorder()

	handlers.HandleResolveAlert(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestBuntDBDataEngineHandlers_HandleResolveAlert_NoID(t *testing.T) {
	handlers := NewBuntDBDataEngineHandlers(nil)

	req := httptest.NewRequest("POST", "/alerts//resolve", nil)
	w := httptest.NewRecorder()

	handlers.HandleResolveAlert(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code) // No engine available
}
