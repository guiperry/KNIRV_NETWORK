package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type EventType string

const (
	EventTypeModuleLog        EventType = "module:log"
	EventTypeNodeStatus       EventType = "node:status"
	EventTypeNodeMetrics      EventType = "node:metrics"
	EventTypeNodeTask         EventType = "node:task"
	EventTypeTaskProgress     EventType = "task:progress"
	EventTypeTaskComplete     EventType = "task:complete"
	EventTypeTaskFailed       EventType = "task:failed"
	EventTypePolicyViolation  EventType = "policy:violation"
	EventTypePolicyUpdate     EventType = "policy:update"
	EventTypeChainSession     EventType = "chain:session"
	EventTypeSecretRetrieved  EventType = "secret:retrieved"
	EventTypeEvidencePack     EventType = "evidence:pack"
	EventTypeGuardrailTrigger EventType = "guardrail:trigger"
	EventTypeSystemHealth     EventType = "system:health"
	EventTypeP2PDiscovery     EventType = "p2p:discovery"
	EventTypeWorkflowUpdate   EventType = "workflow:update"
	EventTypeOntologyIngest   EventType = "ontology:ingest"
	EventTypeEmbeddingUpdate  EventType = "embedding:update"
	EventTypeAlignmentDrift   EventType = "alignment:drift"
)

type Event struct {
	Type      EventType   `json:"type"`
	ID        string      `json:"id"`
	NodeID    string      `json:"node_id,omitempty"`
	TaskID    string      `json:"task_id,omitempty"`
	Source    string      `json:"source"`
	Action    string      `json:"action"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type EventBroadcaster struct {
	ws               *WebSocketService
	handlers         map[EventType][]EventHandler
	handlerMu        sync.RWMutex
	subscribers      map[string]chan Event
	subscriberMu     sync.RWMutex
	eventHistory     []Event
	historyMu        sync.RWMutex
	historyMax       int
	broadcastChannel chan Event
	stopChan         chan struct{}
}

type EventHandler func(Event)

type EventSubscription struct {
	ID      string
	Types   []EventType
	Handler EventHandler
}

func NewEventBroadcaster(ws *WebSocketService) *EventBroadcaster {
	eb := &EventBroadcaster{
		ws:               ws,
		handlers:         make(map[EventType][]EventHandler),
		subscribers:      make(map[string]chan Event),
		historyMax:       1000,
		broadcastChannel: make(chan Event, 500),
		stopChan:         make(chan struct{}),
	}
	return eb
}

func (eb *EventBroadcaster) Start() {
	log.Println("Event broadcaster starting...")
	go eb.processEvents()
	go eb.cleanupRoutine()
	log.Println("Event broadcaster started")
}

func (eb *EventBroadcaster) Stop() {
	close(eb.stopChan)
}

func (eb *EventBroadcaster) processEvents() {
	for {
		select {
		case event := <-eb.broadcastChannel:
			eb.dispatchEvent(event)
			eb.storeEvent(event)
			eb.publishToWebSocket(event)
		case <-eb.stopChan:
			return
		}
	}
}

func (eb *EventBroadcaster) dispatchEvent(event Event) {
	eb.handlerMu.RLock()
	defer eb.handlerMu.RUnlock()

	if handlers, ok := eb.handlers[event.Type]; ok {
		for _, handler := range handlers {
			go handler(event)
		}
	}

	if handlers, ok := eb.handlers["*"]; ok {
		for _, handler := range handlers {
			go handler(event)
		}
	}
}

func (eb *EventBroadcaster) publishToWebSocket(event Event) {
	if eb.ws == nil {
		return
	}

	payload := map[string]interface{}{
		"event_id":  event.ID,
		"type":      event.Type,
		"source":    event.Source,
		"action":    event.Action,
		"data":      event.Data,
		"node_id":   event.NodeID,
		"task_id":   event.TaskID,
		"timestamp": event.Timestamp.Format(time.RFC3339),
	}

	eb.ws.Broadcast(string(event.Type), payload)
}

func (eb *EventBroadcaster) storeEvent(event Event) {
	eb.historyMu.Lock()
	defer eb.historyMu.Unlock()

	eb.eventHistory = append(eb.eventHistory, event)
	if len(eb.eventHistory) > eb.historyMax {
		eb.eventHistory = eb.eventHistory[len(eb.eventHistory)-eb.historyMax:]
	}
}

func (eb *EventBroadcaster) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			eb.cleanup()
		case <-eb.stopChan:
			return
		}
	}
}

func (eb *EventBroadcaster) cleanup() {
	eb.historyMu.Lock()
	defer eb.historyMu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)
	kept := make([]Event, 0, len(eb.eventHistory))
	for _, event := range eb.eventHistory {
		if event.Timestamp.After(cutoff) {
			kept = append(kept, event)
		}
	}
	eb.eventHistory = kept
}

func (eb *EventBroadcaster) Subscribe(id string, types []EventType, handler EventHandler) {
	eb.handlerMu.Lock()
	defer eb.handlerMu.Unlock()

	for _, t := range types {
		eb.handlers[t] = append(eb.handlers[t], handler)
	}
	log.Printf("Event subscriber %s subscribed to %d event types", id, len(types))
}

func (eb *EventBroadcaster) Unsubscribe(id string) {
	eb.handlerMu.Lock()
	defer eb.handlerMu.Unlock()
	log.Printf("Unsubscribing event handler: %s", id)
}

func (eb *EventBroadcaster) Emit(event Event) {
	event.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	event.Timestamp = time.Now()
	select {
	case eb.broadcastChannel <- event:
	default:
		log.Printf("Event broadcast channel full, dropping event: %s", event.Type)
	}
}

func (eb *EventBroadcaster) EmitNodeStatus(nodeID, status, source string, data map[string]interface{}) {
	eb.Emit(Event{
		Type:   EventTypeNodeStatus,
		NodeID: nodeID,
		Source: source,
		Action: "status_changed",
		Data:   data,
	})
}

func (eb *EventBroadcaster) EmitNodeMetrics(nodeID, source string, metrics map[string]interface{}) {
	eb.Emit(Event{
		Type:   EventTypeNodeMetrics,
		NodeID: nodeID,
		Source: source,
		Action: "metrics_updated",
		Data:   metrics,
	})
}

func (eb *EventBroadcaster) EmitTaskProgress(taskID, nodeID, source string, progress int) {
	eb.Emit(Event{
		Type:   EventTypeTaskProgress,
		TaskID: taskID,
		NodeID: nodeID,
		Source: source,
		Action: "progress_updated",
		Data: map[string]interface{}{
			"progress": progress,
		},
	})
}

func (eb *EventBroadcaster) EmitTaskComplete(taskID, nodeID, source string, result map[string]interface{}) {
	eb.Emit(Event{
		Type:   EventTypeTaskComplete,
		TaskID: taskID,
		NodeID: nodeID,
		Source: source,
		Action: "completed",
		Data:   result,
	})
}

func (eb *EventBroadcaster) EmitTaskFailed(taskID, nodeID, source, errorMsg string) {
	eb.Emit(Event{
		Type:   EventTypeTaskFailed,
		TaskID: taskID,
		NodeID: nodeID,
		Source: source,
		Action: "failed",
		Data: map[string]interface{}{
			"error": errorMsg,
		},
	})
}

func (eb *EventBroadcaster) EmitPolicyViolation(nodeID, source, violationType, details string) {
	eb.Emit(Event{
		Type:   EventTypePolicyViolation,
		NodeID: nodeID,
		Source: source,
		Action: "violation_detected",
		Data: map[string]interface{}{
			"violation_type": violationType,
			"details":        details,
		},
	})
}

func (eb *EventBroadcaster) EmitPolicyUpdate(policyID, source, policyType string, rules map[string]interface{}) {
	eb.Emit(Event{
		Type:   EventTypePolicyUpdate,
		Source: source,
		Action: "policy_changed",
		Data: map[string]interface{}{
			"policy_id":   policyID,
			"policy_type": policyType,
			"rules":       rules,
		},
	})
}

func (eb *EventBroadcaster) EmitChainSession(sessionID, nodeID, source string, sessionData map[string]interface{}) {
	eb.Emit(Event{
		Type:   EventTypeChainSession,
		NodeID: nodeID,
		Source: source,
		Action: "session_updated",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"data":       sessionData,
		},
	})
}

func (eb *EventBroadcaster) EmitEvidencePack(evidenceID, nodeID, source string, evidence map[string]interface{}) {
	eb.Emit(Event{
		Type:   EventTypeEvidencePack,
		NodeID: nodeID,
		Source: source,
		Action: "pack_created",
		Data: map[string]interface{}{
			"evidence_id": evidenceID,
			"evidence":    evidence,
		},
	})
}

func (eb *EventBroadcaster) EmitGuardrailTrigger(nodeID, source, guardrailType, trigger string, context map[string]interface{}) {
	eb.Emit(Event{
		Type:   EventTypeGuardrailTrigger,
		NodeID: nodeID,
		Source: source,
		Action: "triggered",
		Data: map[string]interface{}{
			"guardrail_type": guardrailType,
			"trigger":        trigger,
			"context":        context,
		},
	})
}

func (eb *EventBroadcaster) EmitSystemHealth(health map[string]interface{}) {
	eb.Emit(Event{
		Type:   EventTypeSystemHealth,
		Source: "system",
		Action: "health_updated",
		Data:   health,
	})
}

func (eb *EventBroadcaster) EmitP2PDiscovery(peerID, source, status string, peerInfo map[string]interface{}) {
	eb.Emit(Event{
		Type:   EventTypeP2PDiscovery,
		Source: source,
		Action: status,
		Data: map[string]interface{}{
			"peer_id":   peerID,
			"peer_info": peerInfo,
		},
	})
}

func (eb *EventBroadcaster) EmitWorkflowUpdate(workflowID, source string, status string, steps []map[string]interface{}) {
	eb.Emit(Event{
		Type:   EventTypeWorkflowUpdate,
		Source: source,
		Action: "workflow_" + status,
		Data: map[string]interface{}{
			"workflow_id": workflowID,
			"steps":       steps,
		},
	})
}

func (eb *EventBroadcaster) EmitOntologyIngest(ontologyID, source string, entities int) {
	eb.Emit(Event{
		Type:   EventTypeOntologyIngest,
		Source: source,
		Action: "ontology_ingested",
		Data: map[string]interface{}{
			"ontology_id": ontologyID,
			"entities":    entities,
		},
	})
}

func (eb *EventBroadcaster) EmitEmbeddingUpdate(embeddingID, source string, dimension int) {
	eb.Emit(Event{
		Type:   EventTypeEmbeddingUpdate,
		Source: source,
		Action: "embedding_updated",
		Data: map[string]interface{}{
			"embedding_id": embeddingID,
			"dimension":    dimension,
		},
	})
}

func (eb *EventBroadcaster) EmitAlignmentDrift(nodeID, source string, driftScore float64, threshold float64) {
	eb.Emit(Event{
		Type:   EventTypeAlignmentDrift,
		NodeID: nodeID,
		Source: source,
		Action: "drift_detected",
		Data: map[string]interface{}{
			"drift_score": driftScore,
			"threshold":   threshold,
		},
	})
}

func (eb *EventBroadcaster) GetEventHistory(eventType EventType, limit int) []Event {
	eb.historyMu.RLock()
	defer eb.historyMu.RUnlock()

	var filtered []Event
	if eventType != "" {
		for _, event := range eb.eventHistory {
			if event.Type == eventType {
				filtered = append(filtered, event)
			}
		}
	} else {
		filtered = eb.eventHistory
	}

	if limit > 0 && len(filtered) > limit {
		return filtered[len(filtered)-limit:]
	}
	return filtered
}

func (eb *EventBroadcaster) GetRecentEvents(limit int) []Event {
	eb.historyMu.RLock()
	defer eb.historyMu.RUnlock()

	count := limit
	if count > len(eb.eventHistory) {
		count = len(eb.eventHistory)
	}
	if count == 0 {
		return []Event{}
	}

	result := make([]Event, count)
	copy(result, eb.eventHistory[len(eb.eventHistory)-count:])
	return result
}

// Broadcast sends an event with the given type and payload
func (eb *EventBroadcaster) Broadcast(eventType string, payload interface{}) {
	eb.Emit(Event{
		Type:   EventType(eventType),
		Source: "payment",
		Action: "broadcast",
		Data:   payload,
	})
}

func (eb *EventBroadcaster) ExportEventsJSON(limit int) ([]byte, error) {
	events := eb.GetRecentEvents(limit)
	return json.MarshalIndent(events, "", "  ")
}
