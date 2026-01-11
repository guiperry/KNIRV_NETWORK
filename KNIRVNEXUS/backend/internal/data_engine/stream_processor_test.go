package data_engine

import (
	"testing"
	"time"
)

func TestNewStreamProcessor(t *testing.T) {
	config := StreamProcessorConfig{
		KafkaBrokers:   []string{"localhost:9092"},
		ConsumerGroup:  "test-group",
		Topics:         []string{"test-topic"},
		BatchSize:      50,
		CommitInterval: 500 * time.Millisecond,
	}

	processor := NewStreamProcessor(config)

	if processor == nil {
		t.Fatal("Stream processor is nil")
	}

	if processor.config.KafkaBrokers[0] != "localhost:9092" {
		t.Errorf("Expected Kafka broker 'localhost:9092', got '%s'", processor.config.KafkaBrokers[0])
	}

	if processor.config.ConsumerGroup != "test-group" {
		t.Errorf("Expected consumer group 'test-group', got '%s'", processor.config.ConsumerGroup)
	}

	if processor.config.Topics[0] != "test-topic" {
		t.Errorf("Expected topic 'test-topic', got '%s'", processor.config.Topics[0])
	}

	if processor.handlers == nil {
		t.Error("Handlers map is nil")
	}

	if processor.metrics == nil {
		t.Error("Metrics aggregator is nil")
	}
}

func TestNewStreamProcessor_Defaults(t *testing.T) {
	config := StreamProcessorConfig{}

	processor := NewStreamProcessor(config)

	if processor.config.KafkaBrokers[0] != "localhost:9092" {
		t.Errorf("Expected default Kafka broker 'localhost:9092', got '%s'", processor.config.KafkaBrokers[0])
	}

	if processor.config.ConsumerGroup != "KNIRVORACLE-terminal" {
		t.Errorf("Expected default consumer group 'KNIRVORACLE-terminal', got '%s'", processor.config.ConsumerGroup)
	}

	if processor.config.Topics[0] != "KNIRVORACLE-events" {
		t.Errorf("Expected default topic 'KNIRVORACLE-events', got '%s'", processor.config.Topics[0])
	}

	if processor.config.BatchSize != 100 {
		t.Errorf("Expected default batch size 100, got %d", processor.config.BatchSize)
	}

	if processor.config.CommitInterval != 1*time.Second {
		t.Errorf("Expected default commit interval 1s, got %v", processor.config.CommitInterval)
	}
}

func TestStreamProcessor_IsRunning(t *testing.T) {
	config := StreamProcessorConfig{}
	processor := NewStreamProcessor(config)

	// Initially not running
	if processor.IsRunning() {
		t.Error("Expected processor to not be running initially")
	}

	// After calling Start (even if it fails), it should be marked as running
	// But since we don't have Kafka running, we'll just test the method exists
	if processor.IsRunning() != false {
		// This is expected to be false
	}
}

func TestStreamProcessor_Stop(t *testing.T) {
	config := StreamProcessorConfig{}
	processor := NewStreamProcessor(config)

	// Stop when not started should not error
	err := processor.Stop()
	if err != nil {
		t.Errorf("Stop on unstarted processor failed: %v", err)
	}

	// Should still not be running
	if processor.IsRunning() {
		t.Error("Expected processor to not be running after stop")
	}
}

func TestStreamProcessor_RegisterHandler(t *testing.T) {
	config := StreamProcessorConfig{}
	processor := NewStreamProcessor(config)

	handler := func(event Event) error {
		return nil
	}

	// Register handler
	processor.RegisterHandler(SystemEventType, handler)

	// Check that handler was registered
	processor.handlerMutex.RLock()
	handlers, exists := processor.handlers[SystemEventType]
	processor.handlerMutex.RUnlock()

	if !exists {
		t.Error("Handler was not registered")
	}

	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(handlers))
	}
}

func TestStreamProcessor_GetMetrics(t *testing.T) {
	config := StreamProcessorConfig{}
	processor := NewStreamProcessor(config)

	metrics := processor.GetMetrics()

	if metrics == nil {
		t.Error("Metrics snapshot is nil")
		return
	}

	if metrics.EventCounts == nil {
		t.Error("Event counts map is nil")
	}

	if metrics.EventCountsByMin == nil {
		t.Error("Event counts by minute map is nil")
	}

	if metrics.TotalEvents != 0 {
		t.Errorf("Expected total events 0, got %d", metrics.TotalEvents)
	}

	// Test that we can safely access the metrics without nil pointer dereference
	_ = len(metrics.EventCounts)
	_ = len(metrics.EventCountsByMin)
}

func TestNewMetricsAggregator(t *testing.T) {
	aggregator := NewMetricsAggregator()

	if aggregator == nil {
		t.Fatal("Metrics aggregator is nil")
	}

	if aggregator.eventCounts == nil {
		t.Error("Event counts map is nil")
	}

	if aggregator.eventCountsByMin == nil {
		t.Error("Event counts by minute map is nil")
	}

	if aggregator.totalEvents != 0 {
		t.Errorf("Expected total events 0, got %d", aggregator.totalEvents)
	}
}

func TestMetricsAggregator_RecordEvent(t *testing.T) {
	aggregator := NewMetricsAggregator()

	event := Event{
		Type:      SystemEventType,
		Timestamp: time.Now(),
		Source:    "test",
		Data:      map[string]interface{}{"test": "data"},
	}

	// Record event
	aggregator.RecordEvent(event)

	// Check metrics
	if aggregator.totalEvents != 1 {
		t.Errorf("Expected total events 1, got %d", aggregator.totalEvents)
	}

	if count, exists := aggregator.eventCounts[SystemEventType]; !exists || count != 1 {
		t.Errorf("Expected event count 1 for SystemEventType, got %d", count)
	}

	// Check by minute
	minute := event.Timestamp.Format("2006-01-02 15:04")
	if minCounts, exists := aggregator.eventCountsByMin[minute]; !exists {
		t.Error("Minute counts not found")
	} else if count, exists := minCounts[SystemEventType]; !exists || count != 1 {
		t.Errorf("Expected minute count 1 for SystemEventType, got %d", count)
	}
}

func TestMetricsAggregator_GetSnapshot(t *testing.T) {
	aggregator := NewMetricsAggregator()

	event := Event{
		Type:      ErrorEvent,
		Timestamp: time.Now(),
		Source:    "test",
		Data:      map[string]interface{}{"error": "test error"},
	}

	aggregator.RecordEvent(event)

	snapshot := aggregator.GetSnapshot()

	if snapshot.TotalEvents != 1 {
		t.Errorf("Expected total events 1, got %d", snapshot.TotalEvents)
	}

	if count, exists := snapshot.EventCounts[ErrorEvent]; !exists || count != 1 {
		t.Errorf("Expected event count 1 for ErrorEvent, got %d", count)
	}

	// Allow for very small uptime (test execution time)
	if snapshot.UptimeSeconds < 0 {
		t.Errorf("Expected non-negative uptime, got %d", snapshot.UptimeSeconds)
	}
}

func TestMetricsSnapshot_GetEventRate(t *testing.T) {
	snapshot := &MetricsSnapshot{
		TotalEvents:   100,
		UptimeSeconds: 50,
	}

	rate := snapshot.GetEventRate()

	expected := 2.0 // 100 events / 50 seconds
	if rate != expected {
		t.Errorf("Expected event rate %.2f, got %.2f", expected, rate)
	}
}

func TestMetricsSnapshot_GetEventRate_ZeroUptime(t *testing.T) {
	snapshot := &MetricsSnapshot{
		TotalEvents:   100,
		UptimeSeconds: 0,
	}

	rate := snapshot.GetEventRate()

	if rate != 0 {
		t.Errorf("Expected event rate 0 for zero uptime, got %.2f", rate)
	}
}

func TestMetricsSnapshot_GetEventRateForType(t *testing.T) {
	snapshot := &MetricsSnapshot{
		EventCounts: map[EventType]int64{
			ErrorEvent: 25,
			InfoEvent:  75,
		},
		UptimeSeconds: 100,
	}

	rate := snapshot.GetEventRateForType(ErrorEvent)

	expected := 0.25 // 25 events / 100 seconds
	if rate != expected {
		t.Errorf("Expected event rate %.2f for ErrorEvent, got %.2f", expected, rate)
	}
}

func TestMetricsSnapshot_GetEventRateForType_ZeroUptime(t *testing.T) {
	snapshot := &MetricsSnapshot{
		EventCounts: map[EventType]int64{
			ErrorEvent: 25,
		},
		UptimeSeconds: 0,
	}

	rate := snapshot.GetEventRateForType(ErrorEvent)

	if rate != 0 {
		t.Errorf("Expected event rate 0 for zero uptime, got %.2f", rate)
	}
}

func TestMetricsSnapshot_GetEventRateForType_NotFound(t *testing.T) {
	snapshot := &MetricsSnapshot{
		EventCounts: map[EventType]int64{
			ErrorEvent: 25,
		},
		UptimeSeconds: 100,
	}

	rate := snapshot.GetEventRateForType(WarningEvent)

	if rate != 0 {
		t.Errorf("Expected event rate 0 for non-existent event type, got %.2f", rate)
	}
}

func TestMetricsSnapshot_GetRecentEventRate(t *testing.T) {
	now := time.Now()
	minute := now.Format("2006-01-02 15:04")

	snapshot := &MetricsSnapshot{
		EventCountsByMin: map[string]map[EventType]int64{
			minute: {
				ErrorEvent: 30,
				InfoEvent:  30,
			},
		},
	}

	rate := snapshot.GetRecentEventRate()

	expected := 1.0 // 60 events / 60 seconds
	if rate != expected {
		t.Errorf("Expected recent event rate %.2f, got %.2f", expected, rate)
	}
}

func TestMetricsSnapshot_GetRecentEventRate_NoData(t *testing.T) {
	snapshot := &MetricsSnapshot{
		EventCountsByMin: map[string]map[EventType]int64{},
	}

	rate := snapshot.GetRecentEventRate()

	if rate != 0 {
		t.Errorf("Expected recent event rate 0 for no data, got %.2f", rate)
	}
}

func TestMetricsSnapshot_String(t *testing.T) {
	snapshot := &MetricsSnapshot{
		TotalEvents:   150,
		UptimeSeconds: 300,
	}

	str := snapshot.String()

	expected := "Total Events: 150, Uptime: 300s, Rate: 0.50 events/sec"
	if str != expected {
		t.Errorf("Expected string '%s', got '%s'", expected, str)
	}
}

func TestStreamProcessor_ProcessEvent(t *testing.T) {
	config := StreamProcessorConfig{}
	processor := NewStreamProcessor(config)

	handlerCalled := false
	handler := func(event Event) error {
		handlerCalled = true
		return nil
	}

	// Register handler for specific event type
	processor.RegisterHandler(ErrorEvent, handler)

	// Process event
	event := Event{
		Type:      ErrorEvent,
		Timestamp: time.Now(),
		Source:    "test",
		Data:      map[string]interface{}{"error": "test"},
	}

	processor.processEvent(event)

	if !handlerCalled {
		t.Error("Handler was not called")
	}

	// Test with nil event data to ensure no panic
	eventWithNilData := Event{
		Type:      ErrorEvent,
		Timestamp: time.Now(),
		Source:    "test",
		Data:      nil,
	}

	// This should not panic
	processor.processEvent(eventWithNilData)
}

func TestStreamProcessor_ProcessEvent_ParentType(t *testing.T) {
	config := StreamProcessorConfig{}
	processor := NewStreamProcessor(config)

	handlerCalled := false
	handler := func(event Event) error {
		handlerCalled = true
		return nil
	}

	// Register handler for parent event type
	processor.RegisterHandler(SystemEventType, handler)

	// Process event with child type
	event := Event{
		Type:      "system_error", // This should trigger the SystemEventType handler
		Timestamp: time.Now(),
		Source:    "test",
		Data:      map[string]interface{}{"error": "test"},
	}

	processor.processEvent(event)

	if !handlerCalled {
		t.Error("Parent type handler was not called")
	}

	// Test with nil event data to ensure no panic
	eventWithNilData := Event{
		Type:      "system_error",
		Timestamp: time.Now(),
		Source:    "test",
		Data:      nil,
	}

	// This should not panic
	processor.processEvent(eventWithNilData)
}

func TestStreamProcessor_ProcessEvent_NoHandler(t *testing.T) {
	config := StreamProcessorConfig{}
	processor := NewStreamProcessor(config)

	// Process event without any handlers registered
	event := Event{
		Type:      InfoEvent,
		Timestamp: time.Now(),
		Source:    "test",
		Data:      map[string]interface{}{"info": "test"},
	}

	// This should not panic
	processor.processEvent(event)

	// Test with nil event data to ensure no panic
	eventWithNilData := Event{
		Type:      InfoEvent,
		Timestamp: time.Now(),
		Source:    "test",
		Data:      nil,
	}

	// This should not panic
	processor.processEvent(eventWithNilData)
}
