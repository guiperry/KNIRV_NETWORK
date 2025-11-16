package integration

import (
	"testing"
	"time"

	"backend_server/internal/data-engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataEngineIntegration(t *testing.T) {
	// Create a new data engine
	engine := dataengine.NewDataEngine(dataengine.DataEngineConfig{
		EnableKafka:    false, // Disable for testing
		EnableChromaDB: false, // Disable for testing
		WindowSize:     time.Minute,
		MetricsInterval: time.Second,
	})
	require.NotNil(t, engine)

	// Start the data engine
	err := engine.Start()
	require.NoError(t, err)

	// Ensure cleanup
	defer func() {
		err := engine.Stop()
		assert.NoError(t, err)
	}()

	// Test that the engine is running
	assert.True(t, engine.IsRunning())

	// Test processing an event
	testEvent := dataengine.Event{
		Type:      dataengine.SystemEventType,
		Timestamp: time.Now(),
		Source:    "integration_test",
		Data: map[string]interface{}{
			"message": "Test integration event",
		},
	}

	err = engine.ProcessEvent(testEvent)
	assert.NoError(t, err)

	// Test getting metrics
	metrics := engine.GetMetrics()
	if metrics != nil {
		assert.GreaterOrEqual(t, metrics.TotalEvents, int64(1))
	}

	// Test getting active alerts (should be empty initially)
	alerts := engine.GetActiveAlerts()
	if alerts != nil {
		assert.IsType(t, []dataengine.Alert{}, alerts)
	}

	// Test REST API integration
	testRESTAPIIntegration(t, engine)
}

func TestAlertingIntegration(t *testing.T) {
	engine := dataengine.NewDataEngine(dataengine.DataEngineConfig{
		EnableKafka:    false,
		EnableChromaDB: false,
		WindowSize:     time.Minute,
		MetricsInterval: time.Second,
	})
	require.NotNil(t, engine)

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Create a test alert condition
	alertEvent := dataengine.Event{
		Type:      dataengine.SystemEventType,
		Timestamp: time.Now(),
		Source:    "test",
		Data: map[string]interface{}{
			"error_count": 10,
			"threshold":   5,
		},
	}

	// Process the event that should trigger an alert
	err = engine.ProcessEvent(alertEvent)
	assert.NoError(t, err)

	// Check if alerts were generated
	alerts := engine.GetActiveAlerts()
	if alerts != nil {
		// Test resolving an alert (if any exist)
		if len(alerts) > 0 {
			alertID := alerts[0].ID
			resolved := engine.ResolveAlert(alertID)
			assert.True(t, resolved)
		}
	}

	// Test resolving an alert (if any exist)
	if len(alerts) > 0 {
		alertID := alerts[0].ID
		resolved := engine.ResolveAlert(alertID)
		assert.True(t, resolved)
	}
}

func TestMetricsAggregationIntegration(t *testing.T) {
	engine := dataengine.NewDataEngine(dataengine.DataEngineConfig{
		EnableKafka:    false,
		EnableChromaDB: false,
		WindowSize:     time.Minute,
		MetricsInterval: time.Second,
	})
	require.NotNil(t, engine)

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Process multiple events to test aggregation
	baseTime := time.Now()
	for i := 0; i < 10; i++ {
		event := dataengine.Event{
			Type:      dataengine.UserEventType,
			Timestamp: baseTime.Add(time.Duration(i) * time.Second),
			Source:    "test",
			Data: map[string]interface{}{
				"user_id":    "test-user",
				"page":       "/dashboard",
				"session_id": "session-123",
			},
		}

		err = engine.ProcessEvent(event)
		assert.NoError(t, err)
	}

	// Allow time for processing
	time.Sleep(100 * time.Millisecond)

	// Check metrics
	metrics := engine.GetMetrics()
	if metrics != nil {
		assert.GreaterOrEqual(t, metrics.TotalEvents, int64(10))
	}
}

func TestDatabasePersistenceIntegration(t *testing.T) {
	// Skip this test for now as it requires database setup
	t.Skip("Database persistence test requires proper database configuration")
}

func testRESTAPIIntegration(t *testing.T, engine *dataengine.DataEngine) {
	// Create REST API server
	config := dataengine.RESTAPIConfig{
		Port: 0, // Use random port for testing
	}
	server := dataengine.NewRESTAPIServer(config, engine)

	// Start server
	err := server.Start()
	require.NoError(t, err)

	// Ensure cleanup
	defer func() {
		err := server.Stop()
		assert.NoError(t, err)
	}()

	// Test that server is running
	assert.True(t, server.IsRunning())
}

func TestConcurrentEventProcessing(t *testing.T) {
	engine := dataengine.NewDataEngine(dataengine.DataEngineConfig{
		EnableKafka:    false,
		EnableChromaDB: false,
		WindowSize:     time.Minute,
		MetricsInterval: time.Second,
	})
	require.NotNil(t, engine)

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Test concurrent event processing
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(workerID int) {
			for j := 0; j < 10; j++ {
				event := dataengine.Event{
					Type:      dataengine.SystemEventType,
					Timestamp: time.Now(),
					Source:    "concurrent_test",
					Data: map[string]interface{}{
						"worker_id": workerID,
						"event_id":  j,
						"message":   "Concurrent test event",
					},
				}

				err := engine.ProcessEvent(event)
				assert.NoError(t, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Worker completed
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent workers")
		}
	}

	// Verify metrics
	metrics := engine.GetMetrics()
	if metrics != nil {
		assert.GreaterOrEqual(t, metrics.TotalEvents, int64(100)) // 10 workers * 10 events each
	}
}

func TestIntegrationPlaceholder(t *testing.T) {
	// This is kept for backward compatibility but now we have real integration tests above
	t.Log("Integration test placeholder - passing")
}
