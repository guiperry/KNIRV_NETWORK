package resilience

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

type EventBus struct {
	subscribers map[string][]EventHandler
	mu          sync.RWMutex
	logger      *logrus.Logger

	subscriberCount int64
	eventCount      int64
}

func NewEventBus(logger *logrus.Logger) *EventBus {
	if logger == nil {
		logger = logrus.New()
	}
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
		logger:      logger,
	}
}

func (eb *EventBus) Subscribe(eventTypes []string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, eventType := range eventTypes {
		eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
		atomic.AddInt64(&eb.subscriberCount, 1)
		eb.logger.Debugf("Subscribed handler to event type: %s", eventType)
	}
}

func (eb *EventBus) Unsubscribe(eventTypes []string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, eventType := range eventTypes {
		handlers := eb.subscribers[eventType]
		for i, h := range handlers {
			if handlerFunc, ok := h.(EventHandlerFunc); ok {
				if targetHandlerFunc, ok := handler.(EventHandlerFunc); ok {
					if sameHandler(handlerFunc, targetHandlerFunc) {
						eb.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
						atomic.AddInt64(&eb.subscriberCount, -1)
						eb.logger.Debugf("Unsubscribed handler from event type: %s", eventType)
						break
					}
				}
			}
		}
	}
}

func sameHandler(a, b EventHandlerFunc) bool {
	return fmt.Sprintf("%p", a) == fmt.Sprintf("%p", b)
}

func (eb *EventBus) Publish(event *Event) {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	atomic.AddInt64(&eb.eventCount, 1)
	eb.logger.Debugf("Publishing event: %s from %s (ID: %s)", event.Type, event.Source, event.EventID)

	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h.HandleEvent(event); err != nil {
				eb.logger.Errorf("Error handling event %s: %v", event.Type, err)
			}
		}(handler)
	}
}

func (eb *EventBus) PublishSync(event *Event) {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	atomic.AddInt64(&eb.eventCount, 1)
	eb.logger.Debugf("Publishing event sync: %s from %s (ID: %s)", event.Type, event.Source, event.EventID)

	var wg sync.WaitGroup
	for _, handler := range handlers {
		wg.Add(1)
		go func(h EventHandler) {
			defer wg.Done()
			if err := h.HandleEvent(event); err != nil {
				eb.logger.Errorf("Error handling event %s: %v", event.Type, err)
			}
		}(handler)
	}
	wg.Wait()
}

func (eb *EventBus) GetSubscriberCount(eventType string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	return len(eb.subscribers[eventType])
}

func (eb *EventBus) GetAllEventTypes() []string {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	eventTypes := make([]string, 0, len(eb.subscribers))
	for eventType := range eb.subscribers {
		eventTypes = append(eventTypes, eventType)
	}
	return eventTypes
}

func (eb *EventBus) GetTotalStats() (subscribers int64, events int64) {
	return atomic.LoadInt64(&eb.subscriberCount), atomic.LoadInt64(&eb.eventCount)
}

func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers = make(map[string][]EventHandler)
	atomic.StoreInt64(&eb.subscriberCount, 0)
	eb.logger.Info("EventBus cleared")
}

type EventHandlerFunc func(event *Event) error

func (f EventHandlerFunc) HandleEvent(event *Event) error {
	return f(event)
}
