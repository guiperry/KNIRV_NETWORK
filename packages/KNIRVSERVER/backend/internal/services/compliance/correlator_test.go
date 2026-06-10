package compliance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCorrelator(t *testing.T) {
	c := NewCorrelator()
	assert.NotNil(t, c)
	assert.True(t, c.VerifyChain())
	assert.Empty(t, c.TipHash())
}

func TestRecordEvent(t *testing.T) {
	c := NewCorrelator()
	event := c.RecordEvent("node-1", "agent-1", "policy_violation", "high", nil)

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, "node-1", event.NodeID)
	assert.Equal(t, "agent-1", event.AgentID)
	assert.Equal(t, "policy_violation", event.EventType)
	assert.Equal(t, "high", event.Severity)
	assert.NotEmpty(t, event.ChainHash)
	assert.False(t, event.Timestamp.IsZero())
}

func TestRecordEventWithDetails(t *testing.T) {
	c := NewCorrelator()
	details := map[string]interface{}{
		"action": "delete",
		"resource": "users",
	}
	event := c.RecordEvent("node-1", "agent-1", "audit", "medium", details)

	assert.Equal(t, "delete", event.Details["action"])
	assert.Equal(t, "users", event.Details["resource"])
}

func TestGetEventsByNode(t *testing.T) {
	c := NewCorrelator()
	c.RecordEvent("node-1", "agent-1", "violation", "high", nil)
	c.RecordEvent("node-2", "agent-2", "violation", "low", nil)
	c.RecordEvent("node-1", "agent-1", "compliance", "info", nil)

	events := c.GetEvents("node-1", 0)
	assert.Len(t, events, 2)

	all := c.GetEvents("", 0)
	assert.Len(t, all, 3)
}

func TestGetEventsLimit(t *testing.T) {
	c := NewCorrelator()
	for i := 0; i < 10; i++ {
		c.RecordEvent("node-1", "agent-1", "audit", "info", nil)
	}

	events := c.GetEvents("node-1", 3)
	assert.Len(t, events, 3)
}

func TestVerifyChainValid(t *testing.T) {
	c := NewCorrelator()
	c.RecordEvent("node-1", "agent-1", "violation", "high", nil)
	c.RecordEvent("node-2", "agent-2", "info", "low", nil)
	c.RecordEvent("node-1", "agent-1", "audit", "medium", nil)

	assert.True(t, c.VerifyChain())
}

func TestVerifyChainTampered(t *testing.T) {
	c := NewCorrelator()
	c.RecordEvent("node-1", "agent-1", "first", "high", nil)
	c.RecordEvent("node-2", "agent-2", "second", "low", nil)

	eventsBefore := c.GetEvents("", 0)
	eventsBefore[0].EventType = "tampered"

	assert.False(t, c.VerifyChain())
}

func TestTipHash(t *testing.T) {
	c := NewCorrelator()
	assert.Empty(t, c.TipHash())

	e1 := c.RecordEvent("node-1", "agent-1", "first", "low", nil)
	assert.Equal(t, e1.ChainHash, c.TipHash())

	e2 := c.RecordEvent("node-2", "agent-2", "second", "high", nil)
	assert.Equal(t, e2.ChainHash, c.TipHash())
}

func TestGetEventsEmpty(t *testing.T) {
	c := NewCorrelator()
	events := c.GetEvents("", 0)
	assert.Empty(t, events)

	events = c.GetEvents("node-1", 0)
	assert.Empty(t, events)
}
