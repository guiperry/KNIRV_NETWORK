package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/gorilla/mux"
)

// ToolHeadlessResult is the response shape for Lane 5 (headless-with-native-UI)
// tools. Structured data is rendered in KNIRVENGINE's own UI.
type ToolHeadlessResult struct {
	Tool       string         `json:"tool"`
	Functions  []FunctionInfo `json:"functions,omitempty"`
	Decompiled string         `json:"decompiled,omitempty"`
	Listing    string         `json:"listing,omitempty"`
	StartedAt  string         `json:"startedAt"`
	DurationMs int64          `json:"durationMs"`
}

// FunctionInfo represents a single function found in a binary.
type FunctionInfo struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Size    int    `json:"size,omitempty"`
}

// lane5Adapter configures a Lane 5 tool.
type lane5Adapter struct {
	binary      string
	analysisCmd func(session *SandboxSession, binaryPath string, args json.RawMessage) ([]string, error)
	parseOutput func(stdout []byte) (*ToolHeadlessResult, error)
}

// lane5Adapters maps tool names to their Lane 5 adapters.
var lane5Adapters = map[string]lane5Adapter{}

// registerLane5Tool registers a Lane 5 (headless-with-native-UI) tool adapter.
func registerLane5Tool(name string, adapter lane5Adapter) {
	lane5Adapters[name] = adapter
}

// handleToolHeadless handles POST /api/v1/sandboxes/{id}/tools/{tool}/analyze.
// Runs a headless analysis command and returns structured data for KNIRVENGINE's UI.
func (m *SandboxManager) handleToolHeadless(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	tool := vars["tool"]

	session, err := m.GetSession(sessionID)
	if err != nil {
		RespondWithNotFound(w, "Sandbox session")
		return
	}

	adapter, ok := lane5Adapters[tool]
	if !ok {
		RespondWithNotFound(w, "Tool")
		return
	}

	var req struct {
		BinaryPath string          `json:"binaryPath"`
		Args       json.RawMessage `json:"args,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithValidationError(w, fmt.Sprintf("invalid request: %v", err))
		return
	}
	if req.BinaryPath == "" {
		req.BinaryPath = mountedTargetDir(session)
	}

	argv, err := adapter.analysisCmd(session, req.BinaryPath, req.Args)
	if err != nil {
		RespondWithValidationError(w, fmt.Sprintf("invalid arguments: %v", err))
		return
	}

	binary := resolveSandboxTool(adapter.binary)
	if binary == adapter.binary {
		if _, err := exec.LookPath(binary); err != nil {
			RespondWithValidationError(w, fmt.Sprintf("tool binary %q not found", binary))
			return
		}
	}

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
		if stdout.Len() == 0 {
			RespondWithInternalError(w, fmt.Sprintf("%s failed: %v\nstderr: %s", tool, err, stderr.String()))
			return
		}
	}

	if adapter.parseOutput != nil {
		result, err := adapter.parseOutput(stdout.Bytes())
		if err != nil {
			RespondWithInternalError(w, fmt.Sprintf("failed to parse output: %v", err))
			return
		}
		RespondWithSuccess(w, result, fmt.Sprintf("%s analysis complete", tool))
		return
	}

	RespondWithSuccess(w, ToolHeadlessResult{
		Tool:      tool,
		Listing:   stdout.String(),
		StartedAt: "", // filled by caller
	}, fmt.Sprintf("%s analysis complete", tool))
}

// isLane5Tool reports whether a tool is registered as a Lane 5 adapter.
func isLane5Tool(name string) bool {
	_, ok := lane5Adapters[name]
	return ok
}
