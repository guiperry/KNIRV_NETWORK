package p2p

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MessageHandler handles incoming gossip messages
type MessageHandler func([]byte) error

// GossipMessage represents a gossip message
type GossipMessage struct {
	Topic     string    `json:"topic"`
	Data      []byte    `json:"data"`
	From      PeerID    `json:"from"`
	Timestamp time.Time `json:"timestamp"`
}

// GossipManager manages GossipSub protocol
// In production, this would use github.com/libp2p/go-libp2p-pubsub
type GossipManager struct {
	topics   map[string][]MessageHandler
	messages []GossipMessage
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewGossipManager creates a new gossip manager
func NewGossipManager(logger *zap.Logger) *GossipManager {
	return &GossipManager{
		topics:   make(map[string][]MessageHandler),
		messages: make([]GossipMessage, 0),
		logger:   logger,
	}
}

// Start starts the gossip manager
func (gm *GossipManager) Start() error {
	gm.logger.Info("Starting GossipSub manager")
	return nil
}

// Stop stops the gossip manager
func (gm *GossipManager) Stop() error {
	gm.logger.Info("Stopping GossipSub manager")
	return nil
}

// Subscribe subscribes to a topic
func (gm *GossipManager) Subscribe(topic string, handler MessageHandler) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if _, exists := gm.topics[topic]; !exists {
		gm.topics[topic] = make([]MessageHandler, 0)
	}

	gm.topics[topic] = append(gm.topics[topic], handler)

	gm.logger.Info("Subscribed to topic",
		zap.String("topic", topic),
	)

	return nil
}

// Unsubscribe unsubscribes from a topic
func (gm *GossipManager) Unsubscribe(topic string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	delete(gm.topics, topic)

	gm.logger.Info("Unsubscribed from topic",
		zap.String("topic", topic),
	)

	return nil
}

// Publish publishes a message to a topic
func (gm *GossipManager) Publish(topic string, data []byte) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	message := GossipMessage{
		Topic:     topic,
		Data:      data,
		From:      PeerID("local"),
		Timestamp: time.Now(),
	}

	// Store message
	gm.messages = append(gm.messages, message)

	// Call handlers
	handlers, exists := gm.topics[topic]
	if !exists {
		gm.logger.Debug("No subscribers for topic",
			zap.String("topic", topic),
		)
		return nil
	}

	gm.logger.Debug("Publishing message",
		zap.String("topic", topic),
		zap.Int("size", len(data)),
		zap.Int("handlers", len(handlers)),
	)

	// Call handlers asynchronously
	for _, handler := range handlers {
		go func(h MessageHandler, d []byte) {
			if err := h(d); err != nil {
				gm.logger.Error("Handler error",
					zap.String("topic", topic),
					zap.Error(err),
				)
			}
		}(handler, data)
	}

	return nil
}

// GetTopics returns all subscribed topics
func (gm *GossipManager) GetTopics() []string {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	topics := make([]string, 0, len(gm.topics))
	for topic := range gm.topics {
		topics = append(topics, topic)
	}

	return topics
}

// GetMessageCount returns the number of messages received
func (gm *GossipManager) GetMessageCount() int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	return len(gm.messages)
}

// GetRecentMessages returns recent messages for a topic
func (gm *GossipManager) GetRecentMessages(topic string, limit int) []GossipMessage {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	messages := make([]GossipMessage, 0)
	for i := len(gm.messages) - 1; i >= 0 && len(messages) < limit; i-- {
		if gm.messages[i].Topic == topic {
			messages = append(messages, gm.messages[i])
		}
	}

	return messages
}

// GetStats returns gossip statistics
func (gm *GossipManager) GetStats() map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	handlerCounts := make(map[string]int)
	for topic, handlers := range gm.topics {
		handlerCounts[topic] = len(handlers)
	}

	return map[string]interface{}{
		"topics":         len(gm.topics),
		"total_messages": len(gm.messages),
		"handler_counts": handlerCounts,
	}
}

// ValidateMessage validates a gossip message
func (gm *GossipManager) ValidateMessage(msg *GossipMessage) error {
	if msg.Topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	if len(msg.Data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	return nil
}
