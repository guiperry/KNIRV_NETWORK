package dataengine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"backend-server/internal/messages"
)

// EventType represents different types of events that can be produced
type EventType string

const (
	// Blockchain events
	BlockchainEventType EventType = "blockchain"
	BlockCreatedEvent   EventType = "block_created"
	TxSubmittedEvent    EventType = "tx_submitted"
	TxConfirmedEvent    EventType = "tx_confirmed"
	TxRejectedEvent     EventType = "tx_rejected"

	// Network events
	NetworkEventType      EventType = "network"
	PeerConnectedEvent    EventType = "dev_connected"
	PeerDisconnectedEvent EventType = "dev_disconnected"
	NodeConnectedEvent    EventType = "node_connected"
	NodeDisconnectedEvent EventType = "node_disconnected"

	// User interaction events
	UserEventType        EventType = "user"
	CommandExecutedEvent EventType = "command_executed"
	PageViewEvent        EventType = "page_view"
	UserLoginEvent       EventType = "user_login"
	UserLogoutEvent      EventType = "user_logout"
	UserActionEvent      EventType = "user_action"

	// System events
	SystemEventType    EventType = "system"
	ErrorEvent         EventType = "error"
	WarningEvent       EventType = "warning"
	InfoEvent          EventType = "info"
	SystemStartedEvent EventType = "system_started"
	SystemStoppedEvent EventType = "system_stopped"
	SystemErrorEvent   EventType = "system_error"
)

// Event represents a generic event to be sent to Kafka
type Event struct {
	Type       EventType              `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	Source     string                 `json:"source"`
	Data       map[string]interface{} `json:"data"`
	UserID     string                 `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	DeviceInfo map[string]string      `json:"device_info,omitempty"`
}

// EventProducer handles producing events to Kafka
type EventProducer struct {
	writer      *kafka.Writer
	isConnected bool
	config      EventProducerConfig
}

// EventProducerConfig contains configuration for the event producer
type EventProducerConfig struct {
	KafkaBrokers []string
	ClientID     string
	BatchSize    int
	BatchTimeout time.Duration
	Async        bool
}

// NewEventProducer creates a new event producer
func NewEventProducer(config EventProducerConfig) *EventProducer {
	if len(config.KafkaBrokers) == 0 {
		config.KafkaBrokers = []string{"localhost:9092"}
	}

	if config.ClientID == "" {
		config.ClientID = "KNIRVORACLE-terminal"
	}

	if config.BatchSize == 0 {
		config.BatchSize = 100
	}

	if config.BatchTimeout == 0 {
		config.BatchTimeout = 1 * time.Second
	}

	return &EventProducer{
		config: config,
	}
}

// Connect establishes a connection to Kafka
func (p *EventProducer) Connect(ctx context.Context) error {
	// Create a new Kafka writer
	p.writer = &kafka.Writer{
		Addr:         kafka.TCP(p.config.KafkaBrokers...),
		Topic:        "KNIRVORACLE-events",
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    p.config.BatchSize,
		BatchTimeout: p.config.BatchTimeout,
		Async:        p.config.Async,
		RequiredAcks: kafka.RequireOne,
	}

	// Test connection by sending a ping message
	pingEvent := Event{
		Type:      SystemEventType,
		Timestamp: time.Now(),
		Source:    "event_producer",
		Data: map[string]interface{}{
			"message": "ping",
		},
	}

	err := p.ProduceEvent(ctx, pingEvent)
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}

	p.isConnected = true
	return nil
}

// ProduceEvent sends an event to Kafka
func (p *EventProducer) ProduceEvent(ctx context.Context, event Event) error {
	if p.writer == nil {
		return fmt.Errorf("event producer not connected")
	}

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Convert event to JSON
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create a Kafka message
	message := kafka.Message{
		Key:   []byte(string(event.Type)),
		Value: value,
		Time:  event.Timestamp,
		Headers: []kafka.Header{
			{Key: "source", Value: []byte(event.Source)},
			{Key: "type", Value: []byte(event.Type)},
		},
	}

	// Send the message
	err = p.writer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	return nil
}

// ProduceBlockchainEvent produces a blockchain event
func (p *EventProducer) ProduceBlockchainEvent(ctx context.Context, eventType EventType, data map[string]interface{}) error {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Source:    "blockchain",
		Data:      data,
	}

	return p.ProduceEvent(ctx, event)
}

// ProduceNetworkEvent produces a network event
func (p *EventProducer) ProduceNetworkEvent(ctx context.Context, eventType EventType, data map[string]interface{}) error {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Source:    "network",
		Data:      data,
	}

	return p.ProduceEvent(ctx, event)
}

// ProduceUserEvent produces a user interaction event
func (p *EventProducer) ProduceUserEvent(ctx context.Context, eventType EventType, data map[string]interface{}, userID, sessionID string) error {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Source:    "user",
		Data:      data,
		UserID:    userID,
		SessionID: sessionID,
	}

	return p.ProduceEvent(ctx, event)
}

// ProduceSystemEvent produces a system event
func (p *EventProducer) ProduceSystemEvent(ctx context.Context, eventType EventType, data map[string]interface{}) error {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Source:    "system",
		Data:      data,
	}

	return p.ProduceEvent(ctx, event)
}

// Close closes the Kafka connection
func (p *EventProducer) Close() error {
	if p.writer != nil {
		err := p.writer.Close()
		p.writer = nil
		p.isConnected = false
		return err
	}

	return nil
}

// IsConnected returns whether the producer is connected to Kafka
func (p *EventProducer) IsConnected() bool {
	return p.isConnected
}

// ConvertBlockchainEventMsg converts a blockchain event message to a Kafka event
func ConvertBlockchainEventMsg(msg messages.BlockchainEventMsg) Event {
	// msg.Data is already map[string]interface{}, so we can use it directly
	dataMap := msg.Data
	if dataMap == nil {
		dataMap = make(map[string]interface{})
	}

	return Event{
		Type:      EventType(msg.Type),
		Timestamp: msg.Timestamp,
		Source:    "blockchain",
		Data:      dataMap,
	}
}

// ConvertNetworkUpdateMsg converts a network update message to a Kafka event
func ConvertNetworkUpdateMsg(msg messages.NetworkUpdateMsg) Event {
	return Event{
		Type:      NetworkEventType,
		Timestamp: time.Now(),
		Source:    "network",
		Data: map[string]interface{}{
			"dev_count":      msg.PeerCount,
			"latency_ms":     msg.Latency.Milliseconds(),
			"upload_speed":   msg.UploadSpeed,
			"download_speed": msg.DownloadSpeed,
		},
	}
}

// ConvertLogMsg converts a log message to a Kafka event
func ConvertLogMsg(msg messages.LogMsg) Event {
	var eventType EventType

	switch msg.Level {
	case "error":
		eventType = ErrorEvent
	case "warning", "warn":
		eventType = WarningEvent
	default:
		eventType = InfoEvent
	}

	return Event{
		Type:      eventType,
		Timestamp: msg.Timestamp,
		Source:    msg.Component,
		Data: map[string]interface{}{
			"message": msg.Message,
			"fields":  msg.Fields,
			"level":   msg.Level,
		},
	}
}
