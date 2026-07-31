package governance

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSocketServer mirrors the GovernanceAgent wire protocol so the client can
// be tested without importing the backend_server module.
type testSocketServer struct {
	ln       net.Listener
	mu       sync.Mutex
	requests []socketRequest
	handler  func(method string, args json.RawMessage) (interface{}, error)
}

func startTestSocketServer(t *testing.T, handler func(method string, args json.RawMessage) (interface{}, error)) string {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "governance.sock")
	ln, err := net.Listen("unix", socketPath)
	require.NoError(t, err)

	srv := &testSocketServer{ln: ln, handler: handler}
	go srv.acceptLoop()

	t.Cleanup(func() { ln.Close() })
	return socketPath
}

func (s *testSocketServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *testSocketServer) handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		payload, err := readTestFrame(conn)
		if err != nil {
			return
		}

		var req socketRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			s.writeResponse(conn, false, nil, "invalid request")
			return
		}

		s.mu.Lock()
		s.requests = append(s.requests, req)
		s.mu.Unlock()

		result, err := s.handler(req.Method, req.Args)
		if err != nil {
			s.writeResponse(conn, false, nil, err.Error())
			continue
		}
		rawResult, _ := json.Marshal(result)
		s.writeResponse(conn, true, rawResult, "")
	}
}

func (s *testSocketServer) writeResponse(conn net.Conn, ok bool, result json.RawMessage, errMsg string) {
	resp := socketResponse{OK: ok, Result: result, Error: errMsg}
	payload, _ := json.Marshal(resp)
	_ = writeTestFrame(conn, payload)
}

func (s *testSocketServer) recordedMethods() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var methods []string
	for _, r := range s.requests {
		methods = append(methods, r.Method)
	}
	return methods
}

func readTestFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header)
	if length == 0 {
		return []byte{}, nil
	}
	payload := make([]byte, length)
	_, err := io.ReadFull(conn, payload)
	return payload, err
}

func writeTestFrame(conn net.Conn, payload []byte) error {
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(payload)))
	if _, err := conn.Write(header); err != nil {
		return err
	}
	_, err := conn.Write(payload)
	return err
}

func TestSocketClientCheckIdentity(t *testing.T) {
	socketPath := startTestSocketServer(t, func(method string, args json.RawMessage) (interface{}, error) {
		assert.Equal(t, socketMethodCheckIdentity, method)
		return TrustEnvelope{
			IdentityID: "id-1",
			NodeID:     "node-1",
			AgentID:    "agent-1",
			TrustLevel: "high",
			TrustScore: 0.85,
			ExpiresAt:  time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		}, nil
	})

	client, err := NewClient(socketPath)
	require.NoError(t, err)
	defer client.Close()

	env, err := client.CheckIdentity("node-1", "agent-1")
	require.NoError(t, err)
	assert.Equal(t, "node-1", env.NodeID)
	assert.Equal(t, "agent-1", env.AgentID)
	assert.Equal(t, "high", env.TrustLevel)
	assert.Equal(t, 0.85, env.TrustScore)
}

func TestSocketClientEvaluatePolicy(t *testing.T) {
	socketPath := startTestSocketServer(t, func(method string, args json.RawMessage) (interface{}, error) {
		assert.Equal(t, socketMethodEvaluatePolicy, method)
		var input PolicyInput
		json.Unmarshal(args, &input)
		assert.Equal(t, "invoke_tool", input.Action)
		return PolicyDecision{
			Decision:     "allow",
			Reason:       "ok",
			MatchedRules: []string{"allow_default"},
		}, nil
	})

	client, err := NewClient(socketPath)
	require.NoError(t, err)
	defer client.Close()

	decision, err := client.EvaluatePolicy(PolicyInput{
		NodeID:     "node-1",
		AgentID:    "agent-1",
		Action:     "invoke_tool",
		ActionType: "tool_call",
	})
	require.NoError(t, err)
	assert.Equal(t, "allow", decision.Decision)
	assert.Equal(t, []string{"allow_default"}, decision.MatchedRules)
}

func TestSocketClientRecordToolCall(t *testing.T) {
	socketPath := startTestSocketServer(t, func(method string, args json.RawMessage) (interface{}, error) {
		assert.Equal(t, socketMethodRecordToolCall, method)
		return map[string]interface{}{"status": "allowed"}, nil
	})

	client, err := NewClient(socketPath)
	require.NoError(t, err)
	defer client.Close()

	status, err := client.RecordToolCall("agent-1", "node-1", "search_web", map[string]interface{}{"q": "x"})
	require.NoError(t, err)
	assert.Equal(t, ToolCallAllowed, status)
}

func TestSocketClientErrorResponse(t *testing.T) {
	socketPath := startTestSocketServer(t, func(method string, args json.RawMessage) (interface{}, error) {
		return nil, assert.AnError
	})

	client, err := NewClient(socketPath)
	require.NoError(t, err)
	defer client.Close()

	_, err = client.CheckIdentity("node-1", "agent-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestSocketClientSequentialRequests(t *testing.T) {
	socketPath := startTestSocketServer(t, func(method string, args json.RawMessage) (interface{}, error) {
		return TrustEnvelope{NodeID: "node-1", TrustLevel: "high", TrustScore: 0.9}, nil
	})

	client, err := NewClient(socketPath)
	require.NoError(t, err)
	defer client.Close()

	for i := 0; i < 3; i++ {
		env, err := client.CheckIdentity("node-1", "agent-1")
		require.NoError(t, err)
		assert.Equal(t, "node-1", env.NodeID)
	}
}

func TestSocketClientDefaultPath(t *testing.T) {
	assert.Equal(t, "/var/run/knirvserver/governance.sock", DefaultSocketPath)
}

func TestSocketClientNoServer(t *testing.T) {
	_, err := NewClient(filepath.Join(t.TempDir(), "missing.sock"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dial")
}

func TestSocketClientClose(t *testing.T) {
	socketPath := startTestSocketServer(t, func(method string, args json.RawMessage) (interface{}, error) {
		return TrustEnvelope{NodeID: "node-1", TrustLevel: "high", TrustScore: 0.9}, nil
	})

	client, err := NewClient(socketPath)
	require.NoError(t, err)

	require.NoError(t, client.Close())
	require.NoError(t, client.Close()) // idempotent

	_, err = client.CheckIdentity("node-1", "agent-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestSocketFrameRoundTrip(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		payload := []byte(`{"method":"check_identity"}`)
		_ = writeFrame(server, payload)
	}()

	payload, err := readFrame(client)
	require.NoError(t, err)
	assert.JSONEq(t, `{"method":"check_identity"}`, string(payload))
}
