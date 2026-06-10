package reliability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEscalationManager(t *testing.T) {
	em := NewEscalationManager()
	assert.NotNil(t, em)
	assert.Empty(t, em.GetHistory("", 0))
}

func TestAddAndGetRoutes(t *testing.T) {
	em := NewEscalationManager()
	route := EscalationRoute{
		FromLevel: "warning",
		ToLevel:   "critical",
		Action:    "notify-admin",
		Notifies:  []string{"admin@example.com"},
	}
	em.AddRoute(route)

	routes := em.GetRoutes("warning")
	assert.Len(t, routes, 1)
	assert.Equal(t, EscalationLevel("critical"), routes[0].ToLevel)
	assert.Equal(t, "notify-admin", routes[0].Action)
	assert.Contains(t, routes[0].Notifies, "admin@example.com")
}

func TestGetNoRoutes(t *testing.T) {
	em := NewEscalationManager()
	routes := em.GetRoutes("nonexistent")
	assert.Empty(t, routes)
}

func TestRecordEvent(t *testing.T) {
	em := NewEscalationManager()
	event := &EscalationEvent{
		ID:        "evt-1",
		NodeID:    "node-1",
		FromLevel: "info",
		ToLevel:   "warning",
		Reason:    "CPU over 90%",
		Action:    "scale-up",
		Timestamp: time.Now(),
	}
	em.RecordEvent(event)

	history := em.GetHistory("node-1", 0)
	assert.Len(t, history, 1)
	assert.Equal(t, "evt-1", history[0].ID)
	assert.Equal(t, "scale-up", history[0].Action)
}

func TestGetHistoryFilterByNode(t *testing.T) {
	em := NewEscalationManager()
	em.RecordEvent(&EscalationEvent{ID: "evt-1", NodeID: "node-1", Timestamp: time.Now()})
	em.RecordEvent(&EscalationEvent{ID: "evt-2", NodeID: "node-2", Timestamp: time.Now()})
	em.RecordEvent(&EscalationEvent{ID: "evt-3", NodeID: "node-1", Timestamp: time.Now()})

	history := em.GetHistory("node-1", 0)
	assert.Len(t, history, 2)
}

func TestGetHistoryAll(t *testing.T) {
	em := NewEscalationManager()
	em.RecordEvent(&EscalationEvent{ID: "evt-1", NodeID: "node-1", Timestamp: time.Now()})
	em.RecordEvent(&EscalationEvent{ID: "evt-2", NodeID: "node-2", Timestamp: time.Now()})

	history := em.GetHistory("", 0)
	assert.Len(t, history, 2)
}

func TestGetHistoryLimit(t *testing.T) {
	em := NewEscalationManager()
	for i := 0; i < 10; i++ {
		em.RecordEvent(&EscalationEvent{
			ID:        string(rune('0' + i)),
			NodeID:    "node-1",
			Timestamp: time.Now(),
		})
	}

	history := em.GetHistory("node-1", 3)
	assert.Len(t, history, 3)
}

func TestGetMultipleRoutes(t *testing.T) {
	em := NewEscalationManager()
	em.AddRoute(EscalationRoute{FromLevel: "warning", ToLevel: "critical", Action: "alert"})
	em.AddRoute(EscalationRoute{FromLevel: "warning", ToLevel: "severe", Action: "page"})

	routes := em.GetRoutes("warning")
	assert.Len(t, routes, 2)
}
