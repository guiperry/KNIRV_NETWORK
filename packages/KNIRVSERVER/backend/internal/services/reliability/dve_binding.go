package reliability

import "time"

type DVEBinding struct {
	DVEID         string `json:"dve_id"`
	NodeID        string `json:"node_id"`
	AgentID       string `json:"agent_id"`
	BreakerID     string `json:"breaker_id"`
	BreachEventID string `json:"breach_event_id,omitempty"`
	BoundAt       int64  `json:"bound_at"`
}

type DVEBindingManager struct {
	bindings map[string]*DVEBinding
}

func NewDVEBindingManager() *DVEBindingManager {
	return &DVEBindingManager{
		bindings: make(map[string]*DVEBinding),
	}
}

func (dm *DVEBindingManager) Bind(dveID, nodeID, agentID, breakerID string) *DVEBinding {
	binding := &DVEBinding{
		DVEID:     dveID,
		NodeID:    nodeID,
		AgentID:   agentID,
		BreakerID: breakerID,
		BoundAt:   time.Now().UnixNano(),
	}
	dm.bindings[breakerID] = binding
	return binding
}

func (dm *DVEBindingManager) Unbind(breakerID string) {
	delete(dm.bindings, breakerID)
}

func (dm *DVEBindingManager) GetBinding(breakerID string) (*DVEBinding, bool) {
	b, ok := dm.bindings[breakerID]
	return b, ok
}

func (dm *DVEBindingManager) ListBindings() []*DVEBinding {
	result := make([]*DVEBinding, 0, len(dm.bindings))
	for _, b := range dm.bindings {
		result = append(result, b)
	}
	return result
}
