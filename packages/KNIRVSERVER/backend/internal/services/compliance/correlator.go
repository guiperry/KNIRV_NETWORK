package compliance

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type ComplianceEvent struct {
	ID          string                 `json:"id"`
	NodeID      string                 `json:"node_id"`
	AgentID     string                 `json:"agent_id"`
	EventType   string                 `json:"event_type"`
	Severity    string                 `json:"severity"`
	Details     map[string]interface{} `json:"details"`
	ChainHash   string                 `json:"chain_hash"`
	Timestamp   time.Time              `json:"timestamp"`
}

type Correlator struct {
	mu      sync.RWMutex
	events  []*ComplianceEvent
	index   map[string][]int
	tip     string
}

func NewCorrelator() *Correlator {
	return &Correlator{
		events: make([]*ComplianceEvent, 0),
		index:  make(map[string][]int),
	}
}

func (c *Correlator) RecordEvent(nodeID, agentID, eventType, severity string, details map[string]interface{}) *ComplianceEvent {
	c.mu.Lock()
	defer c.mu.Unlock()

	ts := time.Now().UTC()
	chainInput := c.tip + nodeID + agentID + eventType + severity + ts.String()
	hash := sha256.Sum256([]byte(chainInput))

	event := &ComplianceEvent{
		ID:        hex.EncodeToString(hash[:])[:16],
		NodeID:    nodeID,
		AgentID:   agentID,
		EventType: eventType,
		Severity:  severity,
		Details:   details,
		ChainHash: hex.EncodeToString(hash[:]),
		Timestamp: ts,
	}

	c.tip = event.ChainHash
	idx := len(c.events)
	c.events = append(c.events, event)
	c.index[nodeID] = append(c.index[nodeID], idx)

	return event
}

func (c *Correlator) GetEvents(nodeID string, limit int) []*ComplianceEvent {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*ComplianceEvent
	if nodeID != "" {
		indices := c.index[nodeID]
		for _, idx := range indices {
			result = append(result, c.events[idx])
		}
	} else {
		result = make([]*ComplianceEvent, len(c.events))
		copy(result, c.events)
	}

	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result
}

func (c *Correlator) VerifyChain() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var prevHash string
	for _, event := range c.events {
		chainInput := prevHash + event.NodeID + event.AgentID + event.EventType + event.Severity + event.Timestamp.String()
		hash := sha256.Sum256([]byte(chainInput))
		computedHash := hex.EncodeToString(hash[:])
		if computedHash != event.ChainHash {
			return false
		}
		prevHash = event.ChainHash
	}
	return true
}

func (c *Correlator) TipHash() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tip
}
