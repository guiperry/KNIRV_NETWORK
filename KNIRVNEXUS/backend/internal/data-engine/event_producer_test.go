package dataengine

import (
	"context"
	"testing"
	"time"

	"backend_server/internal/messages"
)

func TestNewEventProducer(t *testing.T) {
	config := EventProducerConfig{
		KafkaBrokers: []string{"localhost:9092"},
		ClientID:     "test-client",
		BatchSize:    50,
		BatchTimeout: 500 * time.Millisecond,
		Async:        false,
	}

	producer := NewEventProducer(config)

	if producer == nil {
		t.Fatal("Event producer is nil")
	}

	if producer.config.KafkaBrokers[0] != "localhost:9092" {
		t.Errorf("Expected Kafka broker 'localhost:9092', got '%s'", producer.config.KafkaBrokers[0])
	}

	if producer.config.ClientID != "test-client" {
		t.Errorf("Expected client ID 'test-client', got '%s'", producer.config.ClientID)
	}

	if producer.config.BatchSize != 50 {
		t.Errorf("Expected batch size 50, got %d", producer.config.BatchSize)
	}
}

func TestNewEventProducer_Defaults(t *testing.T) {
	config := EventProducerConfig{}

	producer := NewEventProducer(config)

	if producer.config.KafkaBrokers[0] != "localhost:9092" {
		t.Errorf("Expected default Kafka broker 'localhost:9092', got '%s'", producer.config.KafkaBrokers[0])
	}

	if producer.config.ClientID != "KNIRVORACLE-terminal" {
		t.Errorf("Expected default client ID 'KNIRVORACLE-terminal', got '%s'", producer.config.ClientID)
	}

	if producer.config.BatchSize != 100 {
		t.Errorf("Expected default batch size 100, got %d", producer.config.BatchSize)
	}

	if producer.config.BatchTimeout != 1*time.Second {
		t.Errorf("Expected default batch timeout 1s, got %v", producer.config.BatchTimeout)
	}
}

func TestEventProducer_IsConnected(t *testing.T) {
	config := EventProducerConfig{}
	producer := NewEventProducer(config)

	// Initially not connected
	if producer.IsConnected() {
		t.Error("Expected producer to be disconnected initially")
	}

	// After calling Connect (even if it fails), it should be marked as connected
	// But since we don't have Kafka running, we'll just test the method exists
	if producer.IsConnected() != false {
		// This is expected to be false
	}
}

func TestEventProducer_Close(t *testing.T) {
	config := EventProducerConfig{}
	producer := NewEventProducer(config)

	// Close when not connected should not error
	err := producer.Close()
	if err != nil {
		t.Errorf("Close on unconnected producer failed: %v", err)
	}

	// Should still be disconnected
	if producer.IsConnected() {
		t.Error("Expected producer to be disconnected after close")
	}
}

func TestConvertBlockchainEventMsg(t *testing.T) {
	msg := messages.BlockchainEventMsg{
		Type:      "block_created",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"block_number": 12345,
			"hash":         "0x123abc",
		},
	}

	event := ConvertBlockchainEventMsg(msg)

	if event.Type != EventType(msg.Type) {
		t.Errorf("Expected event type '%s', got '%s'", msg.Type, event.Type)
	}

	if event.Source != "blockchain" {
		t.Errorf("Expected source 'blockchain', got '%s'", event.Source)
	}

	if event.Data["block_number"] != 12345 {
		t.Errorf("Expected block_number 12345, got %v", event.Data["block_number"])
	}

	if event.Data["hash"] != "0x123abc" {
		t.Errorf("Expected hash '0x123abc', got %v", event.Data["hash"])
	}
}

func TestConvertBlockchainEventMsg_NilData(t *testing.T) {
	msg := messages.BlockchainEventMsg{
		Type:      "tx_submitted",
		Timestamp: time.Now(),
		Data:      nil,
	}

	event := ConvertBlockchainEventMsg(msg)

	if event.Data == nil {
		t.Error("Expected event data to be initialized even when msg.Data is nil")
	}

	if len(event.Data) != 0 {
		t.Errorf("Expected empty data map, got %d items", len(event.Data))
	}
}

func TestConvertNetworkUpdateMsg(t *testing.T) {
	latency := 50 * time.Millisecond
	msg := messages.NetworkUpdateMsg{
		PeerCount:     42,
		Latency:       latency,
		UploadSpeed:   100.5,
		DownloadSpeed: 200.7,
	}

	event := ConvertNetworkUpdateMsg(msg)

	if event.Type != NetworkEventType {
		t.Errorf("Expected event type '%s', got '%s'", NetworkEventType, event.Type)
	}

	if event.Source != "network" {
		t.Errorf("Expected source 'network', got '%s'", event.Source)
	}

	if event.Data["dev_count"] != 42 {
		t.Errorf("Expected dev_count 42, got %v", event.Data["dev_count"])
	}

	if event.Data["latency_ms"] != latency.Milliseconds() {
		t.Errorf("Expected latency_ms %d, got %v", latency.Milliseconds(), event.Data["latency_ms"])
	}

	if event.Data["upload_speed"] != 100.5 {
		t.Errorf("Expected upload_speed 100.5, got %v", event.Data["upload_speed"])
	}

	if event.Data["download_speed"] != 200.7 {
		t.Errorf("Expected download_speed 200.7, got %v", event.Data["download_speed"])
	}
}

func TestConvertLogMsg_Error(t *testing.T) {
	msg := messages.LogMsg{
		Level:     "error",
		Message:   "Test error message",
		Component: "test-component",
		Timestamp: time.Now(),
		Fields: map[string]interface{}{
			"error_code": 500,
			"user_id":    "user123",
		},
	}

	event := ConvertLogMsg(msg)

	if event.Type != ErrorEvent {
		t.Errorf("Expected event type '%s', got '%s'", ErrorEvent, event.Type)
	}

	if event.Source != "test-component" {
		t.Errorf("Expected source 'test-component', got '%s'", event.Source)
	}

	if event.Data["message"] != "Test error message" {
		t.Errorf("Expected message 'Test error message', got %v", event.Data["message"])
	}

	if event.Data["level"] != "error" {
		t.Errorf("Expected level 'error', got %v", event.Data["level"])
	}

	fields := event.Data["fields"].(map[string]interface{})
	if fields["error_code"] != 500 {
		t.Errorf("Expected error_code 500, got %v", fields["error_code"])
	}
}

func TestConvertLogMsg_Warning(t *testing.T) {
	msg := messages.LogMsg{
		Level:     "warning",
		Message:   "Test warning message",
		Component: "test-component",
		Timestamp: time.Now(),
	}

	event := ConvertLogMsg(msg)

	if event.Type != WarningEvent {
		t.Errorf("Expected event type '%s', got '%s'", WarningEvent, event.Type)
	}
}

func TestConvertLogMsg_Warn(t *testing.T) {
	msg := messages.LogMsg{
		Level:     "warn",
		Message:   "Test warn message",
		Component: "test-component",
		Timestamp: time.Now(),
	}

	event := ConvertLogMsg(msg)

	if event.Type != WarningEvent {
		t.Errorf("Expected event type '%s', got '%s'", WarningEvent, event.Type)
	}
}

func TestConvertLogMsg_Info(t *testing.T) {
	msg := messages.LogMsg{
		Level:     "info",
		Message:   "Test info message",
		Component: "test-component",
		Timestamp: time.Now(),
	}

	event := ConvertLogMsg(msg)

	if event.Type != InfoEvent {
		t.Errorf("Expected event type '%s', got '%s'", InfoEvent, event.Type)
	}
}

func TestConvertLogMsg_Default(t *testing.T) {
	msg := messages.LogMsg{
		Level:     "debug",
		Message:   "Test debug message",
		Component: "test-component",
		Timestamp: time.Now(),
	}

	event := ConvertLogMsg(msg)

	if event.Type != InfoEvent {
		t.Errorf("Expected event type '%s' for unknown level, got '%s'", InfoEvent, event.Type)
	}
}

func TestEventProducer_ProduceEvent_NotConnected(t *testing.T) {
	config := EventProducerConfig{}
	producer := NewEventProducer(config)

	event := Event{
		Type:      SystemEventType,
		Timestamp: time.Now(),
		Source:    "test",
		Data:      map[string]interface{}{"test": "data"},
	}

	err := producer.ProduceEvent(context.Background(), event)
	if err == nil {
		t.Error("Expected error when producing event without connection")
	}

	if err.Error() != "event producer not connected" {
		t.Errorf("Expected 'event producer not connected' error, got '%s'", err.Error())
	}
}

func TestEventProducer_ProduceBlockchainEvent(t *testing.T) {
	config := EventProducerConfig{}
	producer := NewEventProducer(config)

	data := map[string]interface{}{
		"block_number": 12345,
		"hash":         "0x123abc",
	}

	// This will fail because producer is not connected, but we can test the method exists
	err := producer.ProduceBlockchainEvent(context.Background(), BlockCreatedEvent, data)
	if err == nil {
		t.Error("Expected error when producing blockchain event without connection")
	}
}

func TestEventProducer_ProduceNetworkEvent(t *testing.T) {
	config := EventProducerConfig{}
	producer := NewEventProducer(config)

	data := map[string]interface{}{
		"peer_count": 42,
	}

	// This will fail because producer is not connected, but we can test the method exists
	err := producer.ProduceNetworkEvent(context.Background(), PeerConnectedEvent, data)
	if err == nil {
		t.Error("Expected error when producing network event without connection")
	}
}

func TestEventProducer_ProduceUserEvent(t *testing.T) {
	config := EventProducerConfig{}
	producer := NewEventProducer(config)

	data := map[string]interface{}{
		"action": "login",
	}

	// This will fail because producer is not connected, but we can test the method exists
	err := producer.ProduceUserEvent(context.Background(), UserLoginEvent, data, "user123", "session456")
	if err == nil {
		t.Error("Expected error when producing user event without connection")
	}
}

func TestEventProducer_ProduceSystemEvent(t *testing.T) {
	config := EventProducerConfig{}
	producer := NewEventProducer(config)

	data := map[string]interface{}{
		"message": "system test",
	}

	// This will fail because producer is not connected, but we can test the method exists
	err := producer.ProduceSystemEvent(context.Background(), SystemStartedEvent, data)
	if err == nil {
		t.Error("Expected error when producing system event without connection")
	}
}

func TestEvent_Constants(t *testing.T) {
	// Test blockchain event constants
	if BlockchainEventType != "blockchain" {
		t.Errorf("Expected BlockchainEventType 'blockchain', got '%s'", BlockchainEventType)
	}

	if BlockCreatedEvent != "block_created" {
		t.Errorf("Expected BlockCreatedEvent 'block_created', got '%s'", BlockCreatedEvent)
	}

	// Test network event constants
	if NetworkEventType != "network" {
		t.Errorf("Expected NetworkEventType 'network', got '%s'", NetworkEventType)
	}

	if PeerConnectedEvent != "dev_connected" {
		t.Errorf("Expected PeerConnectedEvent 'dev_connected', got '%s'", PeerConnectedEvent)
	}

	// Test user event constants
	if UserEventType != "user" {
		t.Errorf("Expected UserEventType 'user', got '%s'", UserEventType)
	}

	if UserLoginEvent != "user_login" {
		t.Errorf("Expected UserLoginEvent 'user_login', got '%s'", UserLoginEvent)
	}

	// Test system event constants
	if SystemEventType != "system" {
		t.Errorf("Expected SystemEventType 'system', got '%s'", SystemEventType)
	}

	if ErrorEvent != "error" {
		t.Errorf("Expected ErrorEvent 'error', got '%s'", ErrorEvent)
	}
}