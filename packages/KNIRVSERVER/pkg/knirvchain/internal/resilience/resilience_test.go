package resilience

import (
	"sync"
	"testing"
	"time"
)

func TestEventBus_Subscribe(t *testing.T) {
	eb := NewEventBus(nil)

	handler := EventHandlerFunc(func(e *Event) error {
		return nil
	})

	eb.Subscribe([]string{EventTypeBlockAdded}, handler)

	count := eb.GetSubscriberCount(EventTypeBlockAdded)
	if count != 1 {
		t.Errorf("Expected 1 subscriber, got %d", count)
	}
}

func TestEventBus_Publish(t *testing.T) {
	eb := NewEventBus(nil)

	var received bool
	var mu sync.Mutex

	handler := EventHandlerFunc(func(e *Event) error {
		mu.Lock()
		received = true
		mu.Unlock()
		return nil
	})

	eb.Subscribe([]string{EventTypeBlockAdded}, handler)

	event := NewEvent(EventTypeBlockAdded, "test", nil)
	eb.Publish(event)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if !received {
		t.Error("Event was not received by handler")
	}
	mu.Unlock()
}

func TestEventBus_Unsubscribe(t *testing.T) {
	eb := NewEventBus(nil)

	handler := EventHandlerFunc(func(e *Event) error {
		return nil
	})

	eb.Subscribe([]string{EventTypeBlockAdded}, handler)
	eb.Unsubscribe([]string{EventTypeBlockAdded}, handler)

	count := eb.GetSubscriberCount(EventTypeBlockAdded)
	if count != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", count)
	}
}

func TestEventBus_GetAllEventTypes(t *testing.T) {
	eb := NewEventBus(nil)

	handler := EventHandlerFunc(func(e *Event) error {
		return nil
	})

	eb.Subscribe([]string{EventTypeBlockAdded, EventTypeSkillMined}, handler)

	types := eb.GetAllEventTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 event types, got %d", len(types))
	}
}

func TestEventBus_Clear(t *testing.T) {
	eb := NewEventBus(nil)

	handler := EventHandlerFunc(func(e *Event) error {
		return nil
	})

	eb.Subscribe([]string{EventTypeBlockAdded, EventTypeSkillMined}, handler)
	eb.Clear()

	count := eb.GetSubscriberCount(EventTypeBlockAdded)
	if count != 0 {
		t.Errorf("Expected 0 subscribers after clear, got %d", count)
	}
}

func TestAPIClient_CircuitBreaker(t *testing.T) {
	client := NewAPIClient("http://localhost:8080",
		WithTimeout(1*time.Second),
		WithRetries(0),
	)

	isOpen, failures := client.GetCircuitBreakerStatus("/test")
	if isOpen {
		t.Error("Circuit breaker should be closed initially")
	}
	if failures != 0 {
		t.Errorf("Expected 0 failures initially, got %d", failures)
	}
}
