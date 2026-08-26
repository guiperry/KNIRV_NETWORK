package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// ToolScanResult is the unified response shape for Lane 1 (batch scan) and
// Lane 6 (native Go) tool executions. The frontend never needs to distinguish
// between them.
type ToolScanResult struct {
	Tool       string          `json:"tool"`
	RawOutput  string          `json:"rawOutput,omitempty"`
	Structured json.RawMessage `json:"structured,omitempty"`
	StartedAt  time.Time       `json:"startedAt"`
	DurationMs int64           `json:"durationMs"`
}

// toolScanAdapter builds the argv and parses the output for a specific Lane 1
// tool. Each tool gets a small adapter; all subprocess lifecycle is handled by
// the generic handler.
type toolScanAdapter struct {
	// binary is the command to run (resolved via resolveSandboxTool).
	binary string
	// buildArgs constructs the argv from the tool's input.
	buildArgs func(session *SandboxSession, args json.RawMessage) ([]string, error)
	// parseOutput transforms the raw stdout into a structured result.
	// If nil, only RawOutput is returned.
	parseOutput func(stdout []byte) (json.RawMessage, error)
}

// lane1Adapters maps tool names to their Lane 1 adapters.
var lane1Adapters = map[string]toolScanAdapter{}

// lane1Running tracks in-flight scans per session+tool to prevent overlap.
var lane1Running struct {
	sync.Mutex
	sessions map[string]bool // key: sessionID+"/"+tool
}

func init() {
	lane1Running.sessions = make(map[string]bool)
}

// registerLane1Tool registers a Lane 1 (batch scan) tool adapter.
func registerLane1Tool(name string, adapter toolScanAdapter) {
	lane1Adapters[name] = adapter
}

// handleToolScan is the generic Lane 1 + Lane 6 HTTP handler for
// POST /api/v1/sandboxes/{id}/tools/{tool}/run.
// For Lane 1 tools, it spawns a subprocess and captures output.
// For Lane 6 tools, the adapter's buildArgs returns empty args and the
// handler calls a registered native function instead.
func (m *SandboxManager) handleToolScan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	tool := vars["tool"]

	session, err := m.GetSession(sessionID)
	if err != nil {
		RespondWithNotFound(w, "Sandbox session")
		return
	}

	adapter, ok := lane1Adapters[tool]
	if !ok {
		RespondWithNotFound(w, "Tool")
		return
	}

	// Prevent overlapping scans for the same tool+session.
	lane1Running.Lock()
	key := sessionID + "/" + tool
	if lane1Running.sessions[key] {
		lane1Running.Unlock()
		RespondWithError(w, http.StatusConflict, "scan already running for "+tool, ErrorCodeConflict)
		return
	}
	lane1Running.sessions[key] = true
	lane1Running.Unlock()
	defer func() {
		lane1Running.Lock()
		delete(lane1Running.sessions, key)
		lane1Running.Unlock()
	}()

	var args json.RawMessage
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&args)
	}

	argv, err := adapter.buildArgs(session, args)
	if err != nil {
		RespondWithValidationError(w, fmt.Sprintf("invalid arguments: %v", err))
		return
	}

	// Resolve the binary, preferring bundled copy.
	if err := ensureToolAvailable(adapter.binary); err != nil {
		RespondWithValidationError(w, err.Error())
		return
	}
	binary := resolveSandboxTool(adapter.binary)

	startedAt := time.Now()
	cmd := exec.CommandContext(session.ctx, binary, argv...)
	cmd.Env = session.toolEnv(binary)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		RespondWithInternalError(w, fmt.Sprintf("failed to start %s: %v", tool, err))
		return
	}
	if err := cmd.Wait(); err != nil {
		// Many tools exit non-zero on findings; still capture output.
		if stdout.Len() == 0 {
			RespondWithInternalError(w, fmt.Sprintf("%s failed: %v\nstderr: %s", tool, err, stderr.String()))
			return
		}
	}

	result := ToolScanResult{
		Tool:       tool,
		RawOutput:  stdout.String(),
		StartedAt:  startedAt,
		DurationMs: time.Since(startedAt).Milliseconds(),
	}

	if adapter.parseOutput != nil {
		if structured, err := adapter.parseOutput(stdout.Bytes()); err == nil {
			result.Structured = structured
		}
	}

	RespondWithSuccess(w, result, fmt.Sprintf("%s scan complete", tool))
}

// registerToolRoutes registers all tool-related routes on the given router.
// Called from SandboxManager.RegisterHandlers.
func (m *SandboxManager) registerToolRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/run", m.handleToolScan).Methods("POST")
	router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/native", m.handleToolNative).Methods("POST")
	router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/start", m.handleToolStreamStart).Methods("POST")
	router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/stop", m.handleToolStreamStop).Methods("POST")
	router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/ws", m.handleToolStreamWS)
	router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/attach", m.handleToolAttach).Methods("POST")
	router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/detach", m.handleToolDetach).Methods("POST")
	router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/attach/ws", m.handleToolAttachWS)
	router.HandleFunc("/api/v1/sandboxes/{id}/tools/proxychains/configure", m.handleToolProxychains).Methods("POST")
	router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/analyze", m.handleToolHeadless).Methods("POST")
}

// mountedTargetDir returns the first bind-mount destination that looks like
// a project root (the mounted target the operator wants tools to analyze).
// Falls back to /tmp if no binds are configured.
func mountedTargetDir(s *SandboxSession) string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	// The sandbox bootstrapping binds (/usr, /lib, /proc, …) precede the
	// project bind.  Never let a file-oriented tool accidentally scan one of
	// those system trees just because it is first in the bwrap argv.
	systemMounts := map[string]bool{
		"/usr": true, "/lib": true, "/lib64": true, "/bin": true,
		"/sbin": true, "/proc": true, "/dev": true, "/tmp": true,
	}
	for _, b := range s.binds {
		if b.Dst != "" && b.Mode != "tmpfs" && !systemMounts[b.Dst] {
			return b.Dst
		}
	}
	return "/tmp"
}

// toolBinaryResolved resolves a tool binary name to an absolute path if
// bundled, or returns the bare name for PATH lookup.
func toolBinaryResolved(name string) string {
	return resolveSandboxTool(name)
}

// isLane1Tool reports whether a tool name is registered as a Lane 1 adapter.
func isLane1Tool(name string) bool {
	_, ok := lane1Adapters[name]
	return ok
}

// sanitizeToolName ensures a tool name contains only alphanumerics, hyphens,
// underscores, and dots — preventing path traversal in route parameters.
func sanitizeToolName(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			return r
		default:
			return -1
		}
	}, name)
}
