package websocket

import (
	"time"
)

// BroadcastPolicyViolation broadcasts a policy violation event to all subscribed clients
func (ws *WebSocketService) BroadcastPolicyViolation(violation PolicyViolationUpdate) {
	violation.Timestamp = time.Now().Format(time.RFC3339)
	ws.Broadcast("policy-violation", violation)
}

// BroadcastWorkflowExecution broadcasts a workflow execution update to all subscribed clients
func (ws *WebSocketService) BroadcastWorkflowExecution(update WorkflowExecutionUpdate) {
	update.Timestamp = time.Now().Format(time.RFC3339)
	ws.Broadcast("workflow-execution", update)
}

// BroadcastAgentStatus broadcasts an agent status update to all subscribed clients
func (ws *WebSocketService) BroadcastAgentStatus(update AgentStatusUpdate) {
	update.Timestamp = time.Now().Format(time.RFC3339)
	ws.Broadcast("agent-status", update)
}

// BroadcastNetworkTopology broadcasts a network topology update to all subscribed clients
func (ws *WebSocketService) BroadcastNetworkTopology(update NetworkTopologyUpdate) {
	update.Timestamp = time.Now().Format(time.RFC3339)
	ws.Broadcast("network-topology", update)
}

// BroadcastResourceMetrics broadcasts resource metrics to all subscribed clients
func (ws *WebSocketService) BroadcastResourceMetrics(update ResourceMetricsUpdate) {
	update.Timestamp = time.Now().Format(time.RFC3339)
	ws.Broadcast("resource-metrics", update)
}
