package cognitiveengine

import (
	"sync"
	"time"
)

// EventType classifies internal Cognitive Engine events.
type EventType string

const (
	EventValidationResult   EventType = "validation_result"
	EventHighFailureRate    EventType = "high_failure_rate"
	EventNodeOverload       EventType = "node_overload"
	EventGuardrailViolation EventType = "guardrail_violation"
	EventEBPFSecurityAlert  EventType = "ebpf_security_alert"
	EventResourcePressure   EventType = "resource_pressure"
	EventPatternDetected    EventType = "pattern_detected"
	EventAdaptationRequired EventType = "adaptation_required"
)

// EngineEvent carries information about an internal engine occurrence.
type EngineEvent struct {
	Type      EventType
	Source    string
	Payload   interface{}
	Timestamp time.Time
}

// EventBus provides lightweight publish-subscribe within the Cognitive Engine.
// It is goroutine-safe.  Publish never blocks; a full subscriber channel causes
// the event to be silently dropped rather than stalling the publisher.
type EventBus struct {
	subscribers map[EventType][]chan EngineEvent
	mu          sync.RWMutex
	bufferSize  int
}

// NewEventBus creates an EventBus where each subscriber channel has capacity
// bufferSize.
func NewEventBus(bufferSize int) *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]chan EngineEvent),
		bufferSize:  bufferSize,
	}
}

// Subscribe returns a receive-only channel that receives events of type t.
// The caller must drain the channel; events are dropped when the channel is full.
func (eb *EventBus) Subscribe(t EventType) <-chan EngineEvent {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan EngineEvent, eb.bufferSize)
	eb.subscribers[t] = append(eb.subscribers[t], ch)
	return ch
}

// Publish sends an event to all registered subscribers for its type.
// If a subscriber's channel is full the event is dropped for that subscriber.
func (eb *EventBus) Publish(event EngineEvent) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	for _, ch := range eb.subscribers[event.Type] {
		select {
		case ch <- event:
		default:
			// drop – subscriber is not keeping up
		}
	}
}

// Close drains and closes all subscriber channels.
// After Close, further Publish calls are no-ops.
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, chans := range eb.subscribers {
		for _, ch := range chans {
			close(ch)
		}
	}
	eb.subscribers = make(map[EventType][]chan EngineEvent)
}
