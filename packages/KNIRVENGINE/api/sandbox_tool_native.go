package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// nativeToolFunc is the signature for Lane 6 (native Go) tool implementations.
// These run in-process — no subprocess spawning. They receive the session and
// the raw JSON arguments from the request body.
type nativeToolFunc func(s *SandboxSession, args json.RawMessage) (json.RawMessage, error)

// lane6Registry maps tool names to their native Go implementations.
var lane6Registry = struct {
	sync.RWMutex
	tools map[string]nativeToolFunc
}{
	tools: make(map[string]nativeToolFunc),
}

// registerLane6Tool registers a Lane 6 (native Go) tool implementation.
func registerLane6Tool(name string, fn nativeToolFunc) {
	lane6Registry.Lock()
	defer lane6Registry.Unlock()
	lane6Registry.tools[name] = fn
}

// isLane6Tool reports whether a tool is registered as a native Go implementation.
func isLane6Tool(name string) bool {
	lane6Registry.RLock()
	defer lane6Registry.RUnlock()
	_, ok := lane6Registry.tools[name]
	return ok
}

// handleToolNative is the generic Lane 6 HTTP handler for
// POST /api/v1/sandboxes/{id}/tools/{tool}/native.
// It calls the registered native function in-process.
func (m *SandboxManager) handleToolNative(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	tool := vars["tool"]

	session, err := m.GetSession(sessionID)
	if err != nil {
		RespondWithNotFound(w, "Sandbox session")
		return
	}

	lane6Registry.RLock()
	fn, ok := lane6Registry.tools[tool]
	lane6Registry.RUnlock()
	if !ok {
		RespondWithNotFound(w, "Tool")
		return
	}

	var args json.RawMessage
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&args)
	}

	startedAt := time.Now()
	structured, err := fn(session, args)
	if err != nil {
		RespondWithInternalError(w, fmt.Sprintf("tool %s failed: %v", tool, err))
		return
	}

	result := ToolScanResult{
		Tool:       tool,
		Structured: structured,
		StartedAt:  startedAt,
		DurationMs: time.Since(startedAt).Milliseconds(),
	}

	RespondWithSuccess(w, result, fmt.Sprintf("%s complete", tool))
}
