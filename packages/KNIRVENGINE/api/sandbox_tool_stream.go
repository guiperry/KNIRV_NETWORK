package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// ToolEvent is a single event emitted by a Lane 2 streaming tool.
type ToolEvent struct {
	Tool      string          `json:"tool"`
	Timestamp time.Time       `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	RawLine   string          `json:"rawLine,omitempty"`
}

// toolStreamState tracks a running Lane 2 stream.
type toolStreamState struct {
	tool      string
	cmd       *exec.Cmd
	clients   map[*websocket.Conn]bool
	mutex     sync.RWMutex
	events    []ToolEvent
	cancel    context.CancelFunc
	sessionID string
}

// lane2Registry tracks active streams per session+tool.
var lane2Registry = struct {
	sync.RWMutex
	streams map[string]*toolStreamState // key: sessionID+"/"+tool
}{
	streams: make(map[string]*toolStreamState),
}

// lane2LineParser is an optional per-tool parser that transforms a raw output
// line into a typed ToolEvent. If nil, a raw-line event is emitted.
type lane2LineParser func(tool string, line string) (*ToolEvent, error)

// lane2Parsers maps tool names to their line parsers.
var lane2Parsers = map[string]lane2LineParser{}

// registerLane2Parser registers a line parser for a Lane 2 tool.
func registerLane2Parser(tool string, parser lane2LineParser) {
	lane2Parsers[tool] = parser
}

// lane2Adapter builds argv for a Lane 2 tool at start time.
type lane2Adapter struct {
	binary    string
	buildArgs func(session *SandboxSession, args json.RawMessage) ([]string, error)
	needsJoin bool // whether to use spawnJoined (namespace join)
}

// lane2Adapters maps tool names to their Lane 2 adapters.
var lane2Adapters = map[string]lane2Adapter{}

// registerLane2Tool registers a Lane 2 (streaming) tool adapter.
func registerLane2Tool(name string, adapter lane2Adapter) {
	lane2Adapters[name] = adapter
}

// handleToolStreamStart handles POST /api/v1/sandboxes/{id}/tools/{tool}/start.
func (m *SandboxManager) handleToolStreamStart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	tool := vars["tool"]

	session, err := m.GetSession(sessionID)
	if err != nil {
		RespondWithNotFound(w, "Sandbox session")
		return
	}

	adapter, ok := lane2Adapters[tool]
	if !ok {
		RespondWithNotFound(w, "Tool")
		return
	}

	// Check for existing stream.
	lane2Registry.Lock()
	key := sessionID + "/" + tool
	if existing, ok := lane2Registry.streams[key]; ok {
		lane2Registry.Unlock()
		// Stop existing stream first.
		existing.cancel()
		delete(lane2Registry.streams, key)
	}

	var args json.RawMessage
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&args)
	}

	argv, err := adapter.buildArgs(session, args)
	if err != nil {
		lane2Registry.Unlock()
		RespondWithValidationError(w, fmt.Sprintf("invalid arguments: %v", err))
		return
	}

	binary := resolveSandboxTool(adapter.binary)
	if binary == adapter.binary {
		if _, err := exec.LookPath(binary); err != nil {
			lane2Registry.Unlock()
			RespondWithValidationError(w, fmt.Sprintf("tool binary %q not found", binary))
			return
		}
	}

	cancel := context.CancelFunc(func() {})

	var cmd *exec.Cmd
	if adapter.needsJoin {
		cmd, err = session.spawnJoined(binary, argv...)
	} else {
		cmd, err = session.spawn(binary, argv...)
	}
	if err != nil {
		cancel()
		lane2Registry.Unlock()
		RespondWithInternalError(w, fmt.Sprintf("failed to start %s: %v", tool, err))
		return
	}

	state := &toolStreamState{
		tool:      tool,
		cmd:       cmd,
		clients:   make(map[*websocket.Conn]bool),
		cancel:    cancel,
		sessionID: sessionID,
	}
	lane2Registry.streams[key] = state
	lane2Registry.Unlock()

	// Stream stdout/stderr to clients.
	go func() {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("tool %s: StdoutPipe failed: %v", tool, err)
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("tool %s: StderrPipe failed: %v", tool, err)
			return
		}
		go streamReader(state, stdout, false)
		go streamReader(state, stderr, true)
	}()

	RespondWithSuccess(w, map[string]string{"status": "started", "tool": tool}, fmt.Sprintf("%s streaming started", tool))
}

// streamReader reads lines from a tool's output pipe and fans them out.
func streamReader(state *toolStreamState, reader io.Reader, isStderr bool) {
	buf := make([]byte, 4096)
	var lineBuf strings.Builder
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				ch := buf[i]
				if ch == '\n' {
					line := lineBuf.String()
					lineBuf.Reset()
					if line != "" {
						emitEvent(state, line, isStderr)
					}
				} else {
					lineBuf.WriteByte(ch)
				}
			}
		}
		if err != nil {
			if lineBuf.Len() > 0 {
				emitEvent(state, lineBuf.String(), isStderr)
			}
			return
		}
	}
}

// emitEvent parses and broadcasts a single output line.
func emitEvent(state *toolStreamState, line string, isStderr bool) {
	event := ToolEvent{
		Tool:      state.tool,
		Timestamp: time.Now(),
		Type:      "output",
		RawLine:   line,
	}
	if isStderr {
		event.Type = "stderr"
	}

	// Apply tool-specific parser if registered.
	if parser, ok := lane2Parsers[state.tool]; ok {
		parsed, err := parser(state.tool, line)
		if err == nil && parsed != nil {
			event = *parsed
		}
	}

	state.mutex.Lock()
	state.events = append(state.events, event)
	clients := make([]*websocket.Conn, 0, len(state.clients))
	for c := range state.clients {
		clients = append(clients, c)
	}
	state.mutex.Unlock()

	for _, c := range clients {
		if err := c.WriteJSON(map[string]interface{}{
			"type":  "tool_event",
			"event": event,
		}); err != nil {
			state.mutex.Lock()
			delete(state.clients, c)
			state.mutex.Unlock()
			c.Close()
		}
	}
}

// handleToolStreamStop handles POST /api/v1/sandboxes/{id}/tools/{tool}/stop.
func (m *SandboxManager) handleToolStreamStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	tool := vars["tool"]

	lane2Registry.Lock()
	key := sessionID + "/" + tool
	state, ok := lane2Registry.streams[key]
	if !ok {
		lane2Registry.Unlock()
		RespondWithNotFound(w, "Tool stream")
		return
	}
	delete(lane2Registry.streams, key)
	lane2Registry.Unlock()

	state.cancel()
	if state.cmd != nil && state.cmd.Process != nil {
		_ = state.cmd.Process.Kill()
	}

	// Close all WS clients.
	state.mutex.Lock()
	for c := range state.clients {
		c.Close()
	}
	state.mutex.Unlock()

	RespondWithSuccess(w, map[string]string{"status": "stopped", "tool": tool}, fmt.Sprintf("%s streaming stopped", tool))
}

// handleToolStreamWS handles GET /api/v1/sandboxes/{id}/tools/{tool}/ws.
func (m *SandboxManager) handleToolStreamWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	tool := vars["tool"]

	lane2Registry.RLock()
	key := sessionID + "/" + tool
	state, ok := lane2Registry.streams[key]
	lane2Registry.RUnlock()
	if !ok {
		RespondWithNotFound(w, "Tool stream")
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		RespondWithInternalError(w, fmt.Sprintf("WebSocket upgrade failed: %v", err))
		return
	}

	state.mutex.Lock()
	state.clients[conn] = true
	// Replay historical events to the new client.
	for _, event := range state.events {
		_ = conn.WriteJSON(map[string]interface{}{
			"type":  "tool_event",
			"event": event,
		})
	}
	state.mutex.Unlock()

	// Keep the connection alive and detect disconnects.
	go func() {
		defer func() {
			state.mutex.Lock()
			delete(state.clients, conn)
			state.mutex.Unlock()
			conn.Close()
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()
}
