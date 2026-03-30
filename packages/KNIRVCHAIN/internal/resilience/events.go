package resilience

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type EventHandler interface {
	HandleEvent(event *Event) error
}

type Event struct {
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Timestamp string                 `json:"timestamp"`
	EventID   string                 `json:"event_id"`
}

func NewEvent(eventType, source string, data map[string]interface{}) *Event {
	return &Event{
		Type:      eventType,
		Source:    source,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		EventID:   generateEventID(eventType, source),
	}
}

func generateEventID(eventType, source string) string {
	data := eventType + source + time.Now().Format(time.RFC3339Nano)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

const (
	EventTypeBlockAdded           = "block.added"
	EventTypeBlockValidated       = "block.validated"
	EventTypeTransactionSubmitted = "transaction.submitted"
	EventTypeTransactionConfirmed = "transaction.confirmed"
	EventTypeSkillMined           = "skill.mined"
	EventTypeCapabilityMinted     = "capability.minted"
	EventTypePropertyMade         = "property.made"
	EventTypeErrorOccurred        = "error.occurred"
	EventTypeHealthCheck          = "health.check"
	EventTypeNodeDiscovered       = "node.discovered"
	EventTypeNodeLost             = "node.lost"
	EventTypeCircuitBreakerOpen   = "circuit.breaker.open"
	EventTypeCircuitBreakerClosed = "circuit.breaker.closed"
	EventTypeServiceConnected     = "service.connected"
	EventTypeServiceDisconnected  = "service.disconnected"
)
