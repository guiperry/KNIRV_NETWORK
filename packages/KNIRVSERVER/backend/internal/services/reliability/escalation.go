package reliability

import "time"

type EscalationRoute struct {
	FromLevel EscalationLevel `json:"from_level"`
	ToLevel   EscalationLevel `json:"to_level"`
	Action    string          `json:"action"`
	Notifies  []string        `json:"notifies,omitempty"`
}

type EscalationManager struct {
	routes   map[EscalationLevel][]EscalationRoute
	history  []*EscalationEvent
}

type EscalationEvent struct {
	ID        string          `json:"id"`
	NodeID    string          `json:"node_id"`
	FromLevel EscalationLevel `json:"from_level"`
	ToLevel   EscalationLevel `json:"to_level"`
	Reason    string          `json:"reason"`
	Action    string          `json:"action"`
	Timestamp time.Time       `json:"timestamp"`
}

func NewEscalationManager() *EscalationManager {
	return &EscalationManager{
		routes:  make(map[EscalationLevel][]EscalationRoute),
		history: make([]*EscalationEvent, 0),
	}
}

func (em *EscalationManager) AddRoute(route EscalationRoute) {
	em.routes[route.FromLevel] = append(em.routes[route.FromLevel], route)
}

func (em *EscalationManager) GetRoutes(level EscalationLevel) []EscalationRoute {
	return em.routes[level]
}

func (em *EscalationManager) RecordEvent(event *EscalationEvent) {
	em.history = append(em.history, event)
}

func (em *EscalationManager) GetHistory(nodeID string, limit int) []*EscalationEvent {
	var result []*EscalationEvent
	for _, e := range em.history {
		if nodeID == "" || e.NodeID == nodeID {
			result = append(result, e)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result
}
