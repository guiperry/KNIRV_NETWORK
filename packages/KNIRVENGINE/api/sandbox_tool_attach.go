package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// ToolAttachState tracks a Lane 3 (RPC attach) tool session.
type ToolAttachState struct {
	Tool       string   `json:"tool"`
	Attached   bool     `json:"attached"`
	Pid        int      `json:"pid,omitempty"`
	Log        []string `json:"log"`
	mutex      sync.RWMutex
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	clients    map[*websocket.Conn]bool
	cancel     context.CancelFunc
	sessionID  string
	bridgePath string
}

// lane3Registry tracks active attach sessions per session+tool.
var lane3Registry = struct {
	sync.RWMutex
	sessions map[string]*ToolAttachState // key: sessionID+"/"+tool
}{
	sessions: make(map[string]*ToolAttachState),
}

// lane3Adapter configures a Lane 3 tool.
type lane3Adapter struct {
	binary        string
	prerequisites []string
	buildArgs     func(session *SandboxSession, pid int, args json.RawMessage) ([]string, error)
	needsJoin     bool
}

// lane3Adapters maps tool names to their Lane 3 adapters.
var lane3Adapters = map[string]lane3Adapter{}

// registerLane3Tool registers a Lane 3 (RPC attach) tool adapter.
func registerLane3Tool(name string, adapter lane3Adapter) {
	lane3Adapters[name] = adapter
}

// handleToolAttach handles POST /api/v1/sandboxes/{id}/tools/{tool}/attach.
// Starts the RPC bridge subprocess and attaches to the target PID.
func (m *SandboxManager) handleToolAttach(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	tool := vars["tool"]

	session, err := m.GetSession(sessionID)
	if err != nil {
		RespondWithNotFound(w, "Sandbox session")
		return
	}

	adapter, ok := lane3Adapters[tool]
	if !ok {
		RespondWithNotFound(w, "Tool")
		return
	}

	var req struct {
		Pid  int             `json:"pid"`
		Args json.RawMessage `json:"args,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithValidationError(w, fmt.Sprintf("invalid request: %v", err))
		return
	}
	if req.Pid == 0 {
		var targetErr error
		req.Pid, targetErr = session.resolveTargetPid()
		if targetErr != nil {
			RespondWithValidationError(w, "no target PID available — is the sandbox running?")
			return
		}
	}

	lane3Registry.Lock()
	key := sessionID + "/" + tool
	if existing, ok := lane3Registry.sessions[key]; ok {
		lane3Registry.Unlock()
		stopToolAttachment(existing)
		lane3Registry.Lock()
		delete(lane3Registry.sessions, key)
	}

	argv, err := adapter.buildArgs(session, req.Pid, req.Args)
	if err != nil {
		lane3Registry.Unlock()
		RespondWithValidationError(w, fmt.Sprintf("invalid arguments: %v", err))
		return
	}

	if err := ensureToolAvailable(adapter.binary); err != nil {
		lane3Registry.Unlock()
		RespondWithValidationError(w, err.Error())
		return
	}
	for _, prerequisite := range adapter.prerequisites {
		if err := ensureToolAvailable(prerequisite); err != nil {
			lane3Registry.Unlock()
			RespondWithValidationError(w, err.Error())
			return
		}
	}
	if tool == "frida" {
		if err := session.ensureFridaServer(); err != nil {
			lane3Registry.Unlock()
			RespondWithValidationError(w, err.Error())
			return
		}
	}
	binary := resolveSandboxTool(adapter.binary)

	cancel := context.CancelFunc(func() {})

	cmd, stdin, stdout, stderr, err := session.startToolProcess(binary, adapter.needsJoin, argv...)
	if err != nil {
		cancel()
		lane3Registry.Unlock()
		RespondWithInternalError(w, fmt.Sprintf("failed to start %s bridge: %v", tool, err))
		return
	}

	_ = stderr

	state := &ToolAttachState{
		Tool:      tool,
		Attached:  true,
		Pid:       req.Pid,
		cmd:       cmd,
		stdin:     stdin,
		clients:   make(map[*websocket.Conn]bool),
		cancel:    cancel,
		sessionID: sessionID,
	}
	lane3Registry.sessions[key] = state
	lane3Registry.Unlock()

	// Read bridge output.
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			state.mutex.Lock()
			state.Log = append(state.Log, line)
			clients := make([]*websocket.Conn, 0, len(state.clients))
			for c := range state.clients {
				clients = append(clients, c)
			}
			state.mutex.Unlock()
			for _, c := range clients {
				_ = c.WriteJSON(map[string]interface{}{
					"type": "attach_log",
					"line": line,
				})
			}
		}
		// Bridge exited.
		state.mutex.Lock()
		state.Attached = false
		state.mutex.Unlock()
		for c := range state.clients {
			_ = c.WriteJSON(map[string]interface{}{
				"type": "attach_detached",
			})
		}
	}()
	go func() { _, _ = io.Copy(io.Discard, stderr); _ = cmd.Wait() }()

	RespondWithSuccess(w, map[string]interface{}{
		"attached": true,
		"pid":      req.Pid,
		"tool":     tool,
	}, fmt.Sprintf("%s attached to PID %d", tool, req.Pid))
}

// handleToolAttachWS handles WebSocket communication for an attached tool.
func (m *SandboxManager) handleToolAttachWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	tool := vars["tool"]

	lane3Registry.RLock()
	key := sessionID + "/" + tool
	state, ok := lane3Registry.sessions[key]
	lane3Registry.RUnlock()
	if !ok {
		RespondWithNotFound(w, "Tool attach session")
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	state.mutex.Lock()
	state.clients[conn] = true
	// Replay log.
	for _, line := range state.Log {
		_ = conn.WriteJSON(map[string]interface{}{
			"type": "attach_log",
			"line": line,
		})
	}
	state.mutex.Unlock()

	// Read commands from the frontend and forward to the bridge.
	go func() {
		defer func() {
			state.mutex.Lock()
			delete(state.clients, conn)
			state.mutex.Unlock()
			conn.Close()
		}()
		for {
			var msg struct {
				Command string          `json:"command"`
				Args    json.RawMessage `json:"args,omitempty"`
			}
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			state.mutex.RLock()
			stdin := state.stdin
			state.mutex.RUnlock()
			if stdin != nil {
				cmdJSON, _ := json.Marshal(msg)
				_, _ = stdin.Write(append(cmdJSON, '\n'))
			}
		}
	}()
}

// handleToolDetach handles detaching from a tool.
func (m *SandboxManager) handleToolDetach(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	tool := vars["tool"]

	lane3Registry.Lock()
	key := sessionID + "/" + tool
	state, ok := lane3Registry.sessions[key]
	if !ok {
		lane3Registry.Unlock()
		RespondWithNotFound(w, "Tool attach session")
		return
	}
	delete(lane3Registry.sessions, key)
	lane3Registry.Unlock()

	stopToolAttachment(state)

	RespondWithSuccess(w, map[string]string{"status": "detached", "tool": tool}, fmt.Sprintf("%s detached", tool))
}

func stopToolAttachment(state *ToolAttachState) {
	if state == nil {
		return
	}
	state.cancel()
	state.mutex.Lock()
	if state.stdin != nil {
		_ = state.stdin.Close()
		state.stdin = nil
	}
	for c := range state.clients {
		_ = c.Close()
	}
	state.clients = make(map[*websocket.Conn]bool)
	state.Attached = false
	state.mutex.Unlock()
	if state.cmd != nil && state.cmd.Process != nil {
		_ = state.cmd.Process.Kill()
	}
}

func stopToolAttachmentsForSandbox(sessionID string) {
	lane3Registry.Lock()
	attachments := make([]*ToolAttachState, 0)
	for key, state := range lane3Registry.sessions {
		if strings.HasPrefix(key, sessionID+"/") {
			delete(lane3Registry.sessions, key)
			attachments = append(attachments, state)
		}
	}
	lane3Registry.Unlock()
	for _, state := range attachments {
		stopToolAttachment(state)
	}
}

// sendAttachCommand sends a command to an attached tool's bridge process.
func sendAttachCommand(sessionID, tool string, command string, args json.RawMessage) error {
	lane3Registry.RLock()
	key := sessionID + "/" + tool
	state, ok := lane3Registry.sessions[key]
	lane3Registry.RUnlock()
	if !ok {
		return fmt.Errorf("no attach session for %s", tool)
	}
	state.mutex.RLock()
	stdin := state.stdin
	state.mutex.RUnlock()
	if stdin == nil {
		return fmt.Errorf("attach session has no stdin")
	}
	cmdJSON, _ := json.Marshal(map[string]interface{}{
		"command": command,
		"args":    args,
	})
	_, err := stdin.Write(append(cmdJSON, '\n'))
	return err
}

// getAttachState returns the current attach state for a tool.
func getAttachState(sessionID, tool string) (*ToolAttachState, error) {
	lane3Registry.RLock()
	defer lane3Registry.RUnlock()
	key := sessionID + "/" + tool
	state, ok := lane3Registry.sessions[key]
	if !ok {
		return nil, fmt.Errorf("no attach session for %s", tool)
	}
	return state, nil
}

// isLane3Tool reports whether a tool is registered as a Lane 3 adapter.
func isLane3Tool(name string) bool {
	_, ok := lane3Adapters[name]
	return ok
}
