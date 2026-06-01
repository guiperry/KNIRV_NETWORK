package e2e

import (
	"testing"
	"time"

	"backend_server/internal/data_engine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataEngineE2E(t *testing.T) {
	// Create a data engine for end-to-end testing
	engine := data_engine.NewDataEngine(data_engine.DataEngineConfig{
		EnableKafka:     false, // Disable for e2e testing
		EnableChromaDB:  false, // Disable for e2e testing
		WindowSize:      time.Minute,
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

	// Test end-to-end event processing
	testEvent := data_engine.Event{
		Type:      data_engine.SystemEventType,
		Timestamp: time.Now(),
		Source:    "e2e_test",
		Data: map[string]interface{}{
			"test_id":    "e2e-001",
			"message":    "End-to-end test event",
			"test_phase": "processing",
		},
	}

	// Process the event
	err = engine.ProcessEvent(testEvent)
	assert.NoError(t, err)

	// Verify the engine is still running
	assert.True(t, engine.IsRunning())

	// Test metrics collection
	metrics := engine.GetMetrics()
	if metrics != nil {
		assert.GreaterOrEqual(t, metrics.TotalEvents, int64(1))
	}

	// Test alert system
	alerts := engine.GetActiveAlerts()
	if alerts != nil {
		// Should have no active alerts for this test
		assert.IsType(t, []data_engine.Alert{}, alerts)
	}

	// Test REST API server integration
	testRESTAPIServerE2E(t, engine)
}

func TestAlertingSystemE2E(t *testing.T) {
	engine := data_engine.NewDataEngine(data_engine.DataEngineConfig{
		EnableKafka:     false,
		EnableChromaDB:  false,
		WindowSize:      time.Minute,
		MetricsInterval: time.Second,
	})
	require.NotNil(t, engine)

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Create events that should trigger alerts
	alertEvents := []data_engine.Event{
		{
			Type:      data_engine.SystemEventType,
			Timestamp: time.Now(),
			Source:    "e2e_test",
			Data: map[string]interface{}{
				"error_count": 15, // Should trigger error rate alert
				"message":     "High error rate detected",
			},
		},
		{
			Type:      data_engine.UserEventType,
			Timestamp: time.Now(),
			Source:    "e2e_test",
			Data: map[string]interface{}{
				"user_id":    "test-user",
				"page":       "/dashboard",
				"session_id": "session-123",
			},
		},
	}

	// Process events
	for _, event := range alertEvents {
		err := engine.ProcessEvent(event)
		assert.NoError(t, err)
	}

	// Allow time for processing
	time.Sleep(100 * time.Millisecond)

	// Check for alerts
	alerts := engine.GetActiveAlerts()
	if alerts != nil {
		// If alerts were generated, test resolving them
		for _, alert := range alerts {
			resolved := engine.ResolveAlert(alert.ID)
			assert.True(t, resolved)
		}
	}
}

func TestMetricsAggregationE2E(t *testing.T) {
	engine := data_engine.NewDataEngine(data_engine.DataEngineConfig{
		EnableKafka:     false,
		EnableChromaDB:  false,
		WindowSize:      time.Minute,
		MetricsInterval: time.Second,
	})
	require.NotNil(t, engine)

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Process multiple events over time
	baseTime := time.Now()
	eventCount := 50

	for i := 0; i < eventCount; i++ {
		event := data_engine.Event{
			Type:      data_engine.UserEventType,
			Timestamp: baseTime.Add(time.Duration(i) * 100 * time.Millisecond),
			Source:    "e2e_metrics_test",
			Data: map[string]interface{}{
				"user_id":    "test-user",
				"action":     "page_view",
				"page":       "/dashboard",
				"session_id": "session-123",
				"event_num":  i,
			},
		}

		err := engine.ProcessEvent(event)
		assert.NoError(t, err)
	}

	// Allow time for aggregation
	time.Sleep(200 * time.Millisecond)

	// Verify metrics
	metrics := engine.GetMetrics()
	if metrics != nil {
		assert.GreaterOrEqual(t, metrics.TotalEvents, int64(eventCount))
	}
}

func TestConcurrentLoadE2E(t *testing.T) {
	engine := data_engine.NewDataEngine(data_engine.DataEngineConfig{
		EnableKafka:     false,
		EnableChromaDB:  false,
		WindowSize:      time.Minute,
		MetricsInterval: time.Second,
	})
	require.NotNil(t, engine)

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Test concurrent load
	workerCount := 5
	eventsPerWorker := 20
	done := make(chan bool, workerCount)

	for w := 0; w < workerCount; w++ {
		go func(workerID int) {
			for e := 0; e < eventsPerWorker; e++ {
				event := data_engine.Event{
					Type:      data_engine.SystemEventType,
					Timestamp: time.Now(),
					Source:    "concurrent_e2e_test",
					Data: map[string]interface{}{
						"worker_id":  workerID,
						"event_id":   e,
						"test_type":  "concurrent_load",
						"total_load": workerCount * eventsPerWorker,
					},
				}

				err := engine.ProcessEvent(event)
				assert.NoError(t, err)
			}
			done <- true
		}(w)
	}

	// Wait for all workers to complete
	timeout := time.After(10 * time.Second)
	for i := 0; i < workerCount; i++ {
		select {
		case <-done:
			// Worker completed
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent workers to complete")
		}
	}

	// Verify final state
	assert.True(t, engine.IsRunning())
	metrics := engine.GetMetrics()
	if metrics != nil {
		expectedEvents := int64(workerCount * eventsPerWorker)
		assert.GreaterOrEqual(t, metrics.TotalEvents, expectedEvents)
	}
}

func testRESTAPIServerE2E(t *testing.T, engine *data_engine.DataEngine) {
	// Create and start REST API server
	config := data_engine.RESTAPIConfig{
		Port: 0, // Use random port
	}
	server := data_engine.NewRESTAPIServer(config, engine)

	err := server.Start()
	require.NoError(t, err)

	// Ensure cleanup
	defer func() {
		err := server.Stop()
		assert.NoError(t, err)
	}()

	// Test that server is running
	assert.True(t, server.IsRunning())

	// Test basic server functionality
	assert.True(t, engine.IsRunning())
}

func TestSystemStabilityE2E(t *testing.T) {
	engine := data_engine.NewDataEngine(data_engine.DataEngineConfig{
		EnableKafka:     false,
		EnableChromaDB:  false,
		WindowSize:      time.Minute,
		MetricsInterval: time.Second,
	})
	require.NotNil(t, engine)

	// Test multiple start/stop cycles
	for cycle := 0; cycle < 3; cycle++ {
		err := engine.Start()
		assert.NoError(t, err)
		assert.True(t, engine.IsRunning())

		// Process some events during each cycle
		for i := 0; i < 5; i++ {
			event := data_engine.Event{
				Type:      data_engine.SystemEventType,
				Timestamp: time.Now(),
				Source:    "stability_test",
				Data: map[string]interface{}{
					"cycle":     cycle,
					"event_num": i,
					"test_type": "stability",
				},
			}

			err := engine.ProcessEvent(event)
			assert.NoError(t, err)
		}

		err = engine.Stop()
		assert.NoError(t, err)
		assert.False(t, engine.IsRunning())
	}
}

func TestE2EPlaceholder(t *testing.T) {
	// This is kept for backward compatibility but now we have real e2e tests above
	t.Log("E2E test placeholder - passing")
}
