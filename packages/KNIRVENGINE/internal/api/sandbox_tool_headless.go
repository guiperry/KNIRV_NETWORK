package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ToolHeadlessResult is the response shape for Lane 5 (headless-with-native-UI)
// tools. Structured data is rendered in KNIRVENGINE's own UI.
type ToolHeadlessResult struct {
	Tool       string         `json:"tool"`
	RawOutput  string         `json:"rawOutput,omitempty"`
	Functions  []FunctionInfo `json:"functions,omitempty"`
	Decompiled string         `json:"decompiled,omitempty"`
	Listing    string         `json:"listing,omitempty"`
	StartedAt  time.Time      `json:"startedAt"`
	DurationMs int64          `json:"durationMs"`
}

const toolAnalysisTimeout = 5 * time.Minute

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
	clearToolResponseDeadline(w)

	// A panic in a net/http handler closes the connection without a response;
	// the static GUI's reverse proxy can only surface that as an opaque EOF/502.
	// Keep this recovery local to tool execution so the operator receives a JSON
	// error and the engine log retains the complete stack for diagnosis.
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("panic while handling headless tool analysis: %v\n%s", recovered, debug.Stack())
			RespondWithInternalError(w, fmt.Sprintf("%s analysis aborted unexpectedly; inspect the engine log for details", mux.Vars(r)["tool"]))
		}
	}()

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
		// A Cutter analysis needs an executable file, not the project directory.
		// In the usual binary launch flow TargetCommand is exactly that file.
		req.BinaryPath = session.TargetCommand
	}
	if tool == "cutter" {
		var err error
		req.BinaryPath, err = validateCutterBinaryPath(session, req.BinaryPath)
		if err != nil {
			RespondWithValidationError(w, err.Error())
			return
		}
	}

	argv, err := adapter.analysisCmd(session, req.BinaryPath, req.Args)
	if err != nil {
		RespondWithValidationError(w, fmt.Sprintf("invalid arguments: %v", err))
		return
	}

	if err := ensureToolAvailable(adapter.binary); err != nil {
		RespondWithValidationError(w, err.Error())
		return
	}
	binary := resolveSandboxTool(adapter.binary)

	startedAt := time.Now()
	commandContext := session.ctx
	if commandContext == nil {
		// Sessions created through the HTTP API always have a manager context,
		// but using the request context is safe for restored/test sessions and
		// prevents exec.CommandContext from panicking on a nil context.
		commandContext = r.Context()
	}
	commandContext, cancel := context.WithTimeout(commandContext, toolAnalysisTimeout)
	defer cancel()
	cmd := exec.CommandContext(commandContext, binary, argv...)
	cmd.Env = session.toolEnv(binary)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		RespondWithInternalError(w, fmt.Sprintf("failed to start %s: %v", tool, err))
		return
	}
	if err := cmd.Wait(); err != nil {
		if commandContext.Err() == context.DeadlineExceeded {
			RespondWithError(w, http.StatusGatewayTimeout, fmt.Sprintf("%s analysis exceeded the %s limit", tool, toolAnalysisTimeout), ErrorCodeTimeout)
			return
		}
		if stdout.Len() == 0 {
			RespondWithInternalError(w, fmt.Sprintf("%s failed: %v\nstderr: %s", tool, err, stderr.String()))
			return
		}
	}

	duration := time.Since(startedAt)
	if adapter.parseOutput != nil {
		result, err := adapter.parseOutput(stdout.Bytes())
		if err != nil {
			RespondWithInternalError(w, fmt.Sprintf("failed to parse output: %v", err))
			return
		}
		result.StartedAt = startedAt
		result.DurationMs = duration.Milliseconds()
		RespondWithSuccess(w, result, fmt.Sprintf("%s analysis complete", tool))
		return
	}

	RespondWithSuccess(w, ToolHeadlessResult{
		Tool:       tool,
		RawOutput:  stdout.String(),
		Listing:    stdout.String(),
		StartedAt:  startedAt,
		DurationMs: duration.Milliseconds(),
	}, fmt.Sprintf("%s analysis complete", tool))
}

// validateCutterBinaryPath restricts Cutter to the launched executable or a
// regular file beneath an explicitly mounted host directory. Headless tools
// run in the engine process, so accepting arbitrary host paths here would both
// be misleading (the path is not necessarily visible in the sandbox) and let
// a sandbox session inspect unrelated files.
func validateCutterBinaryPath(session *SandboxSession, binaryPath string) (string, error) {
	if strings.TrimSpace(binaryPath) == "" {
		return "", fmt.Errorf("binary path is required: launch a native executable or enter a binary inside a mounted project directory")
	}
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return "", fmt.Errorf("resolve binary path: %w", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("binary path is not accessible: %w", err)
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("inspect binary path: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("binary path must name a regular file, not %s", fileTypeName(info))
	}

	session.mutex.RLock()
	targetCommand := session.TargetCommand
	binds := append([]SandboxBind(nil), session.binds...)
	session.mutex.RUnlock()
	if pathsReferToSameFile(resolvedPath, targetCommand) {
		return resolvedPath, nil
	}
	for _, bind := range binds {
		if bind.Mode == "tmpfs" || bind.Src == "" {
			continue
		}
		root, err := filepath.EvalSymlinks(bind.Src)
		if err == nil && pathIsWithin(resolvedPath, root) {
			return resolvedPath, nil
		}
	}
	return "", fmt.Errorf("binary path must be the launched target or inside an explicitly mounted directory")
}

func pathsReferToSameFile(path, candidate string) bool {
	if candidate == "" || !filepath.IsAbs(candidate) {
		return false
	}
	resolvedCandidate, err := filepath.EvalSymlinks(candidate)
	return err == nil && path == resolvedCandidate
}

func pathIsWithin(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func fileTypeName(info os.FileInfo) string {
	if info.IsDir() {
		return "a directory"
	}
	return "a non-regular file"
}

// isLane5Tool reports whether a tool is registered as a Lane 5 adapter.
func isLane5Tool(name string) bool {
	_, ok := lane5Adapters[name]
	return ok
}
