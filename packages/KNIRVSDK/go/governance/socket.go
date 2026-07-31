package governance

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// DefaultSocketPath is exported so the KNIRVSERVER GovernanceAgent and this
// client agree on the socket location without requiring config on both sides.
const DefaultSocketPath = "/var/run/knirvserver/governance.sock"

// Socket protocol methods.
const (
	socketMethodCheckIdentity  = "check_identity"
	socketMethodEvaluatePolicy = "evaluate_policy"
	socketMethodRecordToolCall = "record_tool_call"
)

// TrustEnvelope is declared locally so KNIRVSDK has no dependency on
// KNIRVSERVER internals.
type TrustEnvelope struct {
	IdentityID string    `json:"identity_id"`
	NodeID     string    `json:"node_id"`
	AgentID    string    `json:"agent_id,omitempty"`
	TrustLevel string    `json:"trust_level"`
	TrustScore float64   `json:"trust_score"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// PolicyInput carries the normalized request the evaluator inspects.
type PolicyInput struct {
	NodeID     string                 `json:"node_id"`
	AgentID    string                 `json:"agent_id,omitempty"`
	Action     string                 `json:"action"`
	ActionType string                 `json:"action_type"`
	Metrics    map[string]float64     `json:"metrics,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// PolicyDecision is the outcome of a policy evaluation.
type PolicyDecision struct {
	Decision     string   `json:"decision"`
	Reason       string   `json:"reason,omitempty"`
	MatchedRules []string `json:"matched_rules,omitempty"`
}

// ToolCallStatus enumerates the outcome of an audited tool call.
type ToolCallStatus string

const (
	ToolCallAllowed ToolCallStatus = "allowed"
	ToolCallDenied  ToolCallStatus = "denied"
	ToolCallFlagged ToolCallStatus = "flagged"
	ToolCallBlocked ToolCallStatus = "blocked"
)

// socketRequest is the wire request envelope. Args is method-specific JSON.
type socketRequest struct {
	Method string          `json:"method"`
	Args   json.RawMessage `json:"args,omitempty"`
}

// socketResponse is the wire response envelope.
type socketResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// Client is a Unix-domain-socket client for the KNIRVSERVER GovernanceAgent.
// It embeds governance checks directly so agent frameworks avoid HTTP round
// trips on the enforcement path.
type Client struct {
	socketPath string
	conn       net.Conn
	mu         sync.Mutex
}

// NewClient connects to the GovernanceAgent at socketPath. Pass an empty
// string to use DefaultSocketPath.
func NewClient(socketPath string) (*Client, error) {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("governance socket dial: %w", err)
	}
	return &Client{socketPath: socketPath, conn: conn}, nil
}

// CheckIdentity resolves a trust envelope for the node/agent pair.
func (c *Client) CheckIdentity(nodeID, agentID string) (*TrustEnvelope, error) {
	args, _ := json.Marshal(map[string]string{"node_id": nodeID, "agent_id": agentID})
	var result TrustEnvelope
	if err := c.roundTrip(socketMethodCheckIdentity, args, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// EvaluatePolicy runs a policy evaluation over the given input.
func (c *Client) EvaluatePolicy(input PolicyInput) (*PolicyDecision, error) {
	args, _ := json.Marshal(input)
	var result PolicyDecision
	if err := c.roundTrip(socketMethodEvaluatePolicy, args, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RecordToolCall audits a tool call through the MCP gateway.
func (c *Client) RecordToolCall(agentID, nodeID, toolName string, args map[string]interface{}) (ToolCallStatus, error) {
	reqArgs, _ := json.Marshal(map[string]interface{}{
		"agent_id":  agentID,
		"node_id":   nodeID,
		"tool_name": toolName,
		"args":      args,
	})
	var result struct {
		Status ToolCallStatus `json:"status"`
	}
	if err := c.roundTrip(socketMethodRecordToolCall, reqArgs, &result); err != nil {
		return ToolCallDenied, err
	}
	return result.Status, nil
}

// Close terminates the socket connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) roundTrip(method string, args json.RawMessage, result interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("governance socket client is closed")
	}

	req := socketRequest{Method: method, Args: args}
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	if err := writeFrame(c.conn, payload); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	respPayload, err := readFrame(c.conn)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var resp socketResponse
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	if !resp.OK {
		return fmt.Errorf("governance %s failed: %s", method, resp.Error)
	}
	if result != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}
	return nil
}

// writeFrame writes a 4-byte big-endian length followed by the payload.
func writeFrame(conn net.Conn, payload []byte) error {
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(payload)))
	if _, err := conn.Write(header); err != nil {
		return err
	}
	_, err := conn.Write(payload)
	return err
}

// readFrame reads a 4-byte big-endian length then the frame payload.
func readFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := readFull(conn, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header)
	if length == 0 {
		return []byte{}, nil
	}
	if length > 1<<24 {
		return nil, fmt.Errorf("frame too large: %d bytes", length)
	}
	payload := make([]byte, length)
	if _, err := readFull(conn, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func readFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
