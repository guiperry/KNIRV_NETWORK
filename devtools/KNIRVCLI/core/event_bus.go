package core

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// EventBus manages event distribution across the application
type EventBus struct {
	subscribers map[string][]EventHandler
	mu          sync.RWMutex
	logger      *logrus.Logger
}

// NewEventBus creates a new event bus
func NewEventBus(logger *logrus.Logger) *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
		logger:      logger,
	}
}

// Subscribe subscribes a handler to specific event types
func (eb *EventBus) Subscribe(eventTypes []string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, eventType := range eventTypes {
		eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
		eb.logger.Debugf("Subscribed handler to event type: %s", eventType)
	}
}

// Unsubscribe removes a handler from event types
func (eb *EventBus) Unsubscribe(eventTypes []string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, eventType := range eventTypes {
		handlers := eb.subscribers[eventType]
		for i, h := range handlers {
			if h == handler {
				// Remove handler from slice
				eb.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
				eb.logger.Debugf("Unsubscribed handler from event type: %s", eventType)
				break
			}
		}
	}
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(event *Event) {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	eb.logger.Debugf("Publishing event: %s from %s", event.Type, event.Source)

	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h.HandleEvent(event); err != nil {
				eb.logger.Errorf("Error handling event %s: %v", event.Type, err)
			}
		}(handler)
	}
}

// GetSubscriberCount returns the number of subscribers for an event type
func (eb *EventBus) GetSubscriberCount(eventType string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	return len(eb.subscribers[eventType])
}

// GetAllEventTypes returns all event types that have subscribers
func (eb *EventBus) GetAllEventTypes() []string {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	eventTypes := make([]string, 0, len(eb.subscribers))
	for eventType := range eb.subscribers {
		eventTypes = append(eventTypes, eventType)
	}
	return eventTypes
}
