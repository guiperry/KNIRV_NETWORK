package messages

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBlockchainEventMsg(t *testing.T) {
	now := time.Now()
	msg := BlockchainEventMsg{
		Type:      "block_mined",
		Source:    "miner_1",
		Timestamp: now,
		Data: map[string]interface{}{
			"block_hash": "abc123",
			"height":     100,
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal BlockchainEventMsg: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled BlockchainEventMsg
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal BlockchainEventMsg: %v", err)
	}

	if unmarshaled.Type != msg.Type {
		t.Errorf("Expected type %s, got %s", msg.Type, unmarshaled.Type)
	}
	if unmarshaled.Source != msg.Source {
		t.Errorf("Expected source %s, got %s", msg.Source, unmarshaled.Source)
	}
	if unmarshaled.Data["block_hash"] != msg.Data["block_hash"] {
		t.Errorf("Expected block_hash %s, got %s", msg.Data["block_hash"], unmarshaled.Data["block_hash"])
	}
}

func TestNetworkUpdateMsg(t *testing.T) {
	now := time.Now()
	msg := NetworkUpdateMsg{
		Type:          "network_status",
		Source:        "node_1",
		Timestamp:     now,
		Data:          map[string]interface{}{"status": "connected"},
		PeerCount:     10,
		Latency:       time.Millisecond * 50,
		UploadSpeed:   100.5,
		DownloadSpeed: 200.7,
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal NetworkUpdateMsg: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled NetworkUpdateMsg
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal NetworkUpdateMsg: %v", err)
	}

	if unmarshaled.PeerCount != msg.PeerCount {
		t.Errorf("Expected peer count %d, got %d", msg.PeerCount, unmarshaled.PeerCount)
	}
	if unmarshaled.Latency != msg.Latency {
		t.Errorf("Expected latency %v, got %v", msg.Latency, unmarshaled.Latency)
	}
	if unmarshaled.UploadSpeed != msg.UploadSpeed {
		t.Errorf("Expected upload speed %f, got %f", msg.UploadSpeed, unmarshaled.UploadSpeed)
	}
	if unmarshaled.DownloadSpeed != msg.DownloadSpeed {
		t.Errorf("Expected download speed %f, got %f", msg.DownloadSpeed, unmarshaled.DownloadSpeed)
	}
}

func TestLogMsg(t *testing.T) {
	now := time.Now()
	msg := LogMsg{
		Level:     "info",
		Message:   "Test message",
		Source:    "test_component",
		Component: "logger",
		Fields: map[string]interface{}{
			"user_id": 123,
			"action":  "login",
		},
		Timestamp: now,
		Data: map[string]interface{}{
			"extra": "data",
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal LogMsg: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled LogMsg
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal LogMsg: %v", err)
	}

	if unmarshaled.Level != msg.Level {
		t.Errorf("Expected level %s, got %s", msg.Level, unmarshaled.Level)
	}
	if unmarshaled.Message != msg.Message {
		t.Errorf("Expected message %s, got %s", msg.Message, unmarshaled.Message)
	}
	if unmarshaled.Source != msg.Source {
		t.Errorf("Expected source %s, got %s", msg.Source, unmarshaled.Source)
	}
	if unmarshaled.Component != msg.Component {
		t.Errorf("Expected component %s, got %s", msg.Component, unmarshaled.Component)
	}
	if unmarshaled.Fields["user_id"] != float64(msg.Fields["user_id"].(int)) {
		t.Errorf("Expected user_id %v, got %v", msg.Fields["user_id"], unmarshaled.Fields["user_id"])
	}
	if unmarshaled.Data["extra"] != msg.Data["extra"] {
		t.Errorf("Expected extra data %v, got %v", msg.Data["extra"], unmarshaled.Data["extra"])
	}
}