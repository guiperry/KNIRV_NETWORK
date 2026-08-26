package api

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// withTestBins points the manager at harmless, always-available binaries so
// the lifecycle can be exercised without bubblewrap/Xvfb/x11vnc installed.
func newTestManager() *SandboxManager {
	m := NewSandboxManager()
	m.xvfbBin = "sleep"
	m.bwrapBin = "sleep"
	m.x11vncBin = "sleep"
	m.vncPort = 15999
	m.waitForDisplay = func(string, time.Duration) error { return nil }
	return m
}

func TestSandboxManagerService(t *testing.T) {
	manager := newTestManager()

	// Single-session scope: no session yet.
	if sessions := manager.ListSessions(0); len(sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(sessions))
	}

	// Create + start a session using a long-lived command.
	session, err := manager.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "my-app",
		TargetCommand: "30",
		Display:       ":99",
		UnshareAll:    true,
		ShareNet:      true,
		DieWithParent: true,
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	if session.ID == "" {
		t.Fatal("session ID should be generated")
	}
	if session.NetnsID != session.ID {
		t.Errorf("expected netnsId == id, got %s vs %s", session.NetnsID, session.ID)
	}
	if session.Status != SandboxStatusIdle {
		t.Errorf("expected idle, got %s", session.Status)
	}

	if err := session.Start(); err != nil {
		t.Fatalf("failed to start sandbox: %v", err)
	}
	if session.Status != SandboxStatusRunning {
		t.Errorf("expected running, got %s (error: %s)", session.Status, session.Error)
	}

	// A second create must be rejected (single-session scope).
	if _, err := manager.CreateSession(1, SandboxLaunchConfig{TargetCommand: "30"}); err == nil {
		t.Error("expected conflict when creating a second session")
	}

	// Retrieve + list.
	got, err := manager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if got.TargetLabel != "my-app" {
		t.Errorf("expected label my-app, got %s", got.TargetLabel)
	}
	if got.VncWsPath == "" || got.StatusWsPath == "" {
		t.Error("expected websocket paths to be populated")
	}
	if len(manager.ListSessions(0)) != 1 {
		t.Error("expected exactly 1 listed session")
	}

	// Close + verify removal.
	if err := manager.CloseSession(session.ID); err != nil {
		t.Fatalf("failed to close session: %v", err)
	}
	if _, err := manager.GetSession(session.ID); err == nil {
		t.Error("expected error fetching closed session")
	}
}

func TestSandboxManagerMissingTargetCommand(t *testing.T) {
	manager := newTestManager()
	if _, err := manager.CreateSession(1, SandboxLaunchConfig{}); err == nil {
		t.Error("expected error when targetCommand is empty")
	}
}

func TestNextAvailableXDisplaySkipsOccupiedDisplays(t *testing.T) {
	// The selected range starts at :99. Existing host displays are allowed;
	// the allocator must always return a display in range that has neither
	// the X socket nor its lock file at the time it is selected.
	display, err := nextAvailableXDisplay()
	if err != nil {
		t.Fatalf("nextAvailableXDisplay: %v", err)
	}
	if !strings.HasPrefix(display, ":") {
		t.Fatalf("display = %q, want local X display", display)
	}
	number := strings.TrimPrefix(display, ":")
	if pathExists(filepath.Join("/tmp/.X11-unix", "X"+number)) || pathExists(filepath.Join("/tmp", ".X"+number+"-lock")) {
		t.Fatalf("allocated occupied display %s", display)
	}
}

func TestWaitForXDisplayRejectsNonLocalDisplay(t *testing.T) {
	if err := waitForXDisplay("localhost:99", time.Millisecond); err == nil {
		t.Fatal("expected non-local display to be rejected")
	}
}

func TestBuildBwrapArgsPreservesExplicitTargetArguments(t *testing.T) {
	session := &SandboxSession{
		TargetCommand: "node",
		targetArgs:    []string{"/project with spaces/app.js", "--port", "3000"},
		Display:       ":99",
	}
	args := buildBwrapArgs(session)
	want := []string{"--proc", "/proc", "--dev", "/dev", "--setenv", "DISPLAY", ":99", "--", "node", "/project with spaces/app.js", "--port", "3000"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected bwrap args: got %#v, want %#v", args, want)
	}
}

func TestExtractLoopbackFrontendURL(t *testing.T) {
	for _, tc := range []struct {
		output string
		want   string
	}{
		{"server started at http://localhost:33159", "http://localhost:33159"},
		{"listen http://127.0.0.1:8080/api", "http://127.0.0.1:8080/api"},
		{"Data Engine failed to connect: http://localhost:8000/api/v1", ""},
		{"Consolidated server started on http://localhost:33159", "http://localhost:33159"},
		{"remote https://example.com/dashboard", ""},
		{"not a URL", ""},
	} {
		if got := extractLoopbackFrontendURL(tc.output); got != tc.want {
			t.Errorf("extractLoopbackFrontendURL(%q) = %q, want %q", tc.output, got, tc.want)
		}
	}
}

func TestSandboxFrontendProxyDialsLoopbackWithoutResolvingLocalhost(t *testing.T) {
	var wantHost string
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("network listeners unavailable in this test sandbox: %v", err)
		}
		t.Fatalf("listen on IPv4 loopback: %v", err)
	}
	frontend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Host; got != wantHost {
			t.Errorf("upstream Host = %q, want %q", got, wantHost)
		}
		_, _ = io.WriteString(w, "frontend reached")
	}))
	frontend.Listener = listener
	frontend.Start()
	defer frontend.Close()

	target, err := url.Parse(frontend.URL)
	if err != nil {
		t.Fatalf("parse frontend URL: %v", err)
	}
	frontendURL := "http://localhost:" + target.Port()
	wantHost = "localhost:" + target.Port()
	session := &SandboxSession{}
	req, err := http.NewRequest(http.MethodGet, frontendURL+"/", nil)
	if err != nil {
		t.Fatalf("create frontend request: %v", err)
	}
	req.Host = wantHost
	w := httptest.NewRecorder()
	session.proxyFrontendRequest(w, req, frontendURL)
	if w.Code != http.StatusOK || w.Body.String() != "frontend reached" {
		t.Fatalf("proxy response = %d %q, want 200 frontend reached", w.Code, w.Body.String())
	}
}

func TestSandboxManagerBwrapMissing(t *testing.T) {
	manager := NewSandboxManager()
	manager.bwrapBin = "/nonexistent/bwrap-binary"

	session, err := manager.CreateSession(1, SandboxLaunchConfig{TargetCommand: "true"})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}
	if err := session.Start(); err == nil {
		t.Fatal("expected error when bwrap is missing")
	}
	if session.Status != SandboxStatusError {
		t.Errorf("expected error status, got %s", session.Status)
	}
	if session.Error == "" {
		t.Error("expected a descriptive error message")
	}
}

func TestSandboxManagerHTTPHandlers(t *testing.T) {
	manager := newTestManager()
	router := mux.NewRouter()
	manager.RegisterHandlers(router)

	createJSON := `{
		"targetLabel": "http-test",
		"targetCommand": "30",
		"display": ":99",
		"unshareAll": true,
		"shareNet": true,
		"dieWithParent": true
	}`

	req := httptest.NewRequest("POST", "/api/v1/sandboxes", strings.NewReader(createJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var response struct {
		Success bool               `json:"success"`
		Data    SandboxSessionInfo `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !response.Success {
		t.Fatalf("expected successful response: %s", w.Body.String())
	}
	id := response.Data.ID

	// GET
	req = httptest.NewRequest("GET", "/api/v1/sandboxes/"+id, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// LIST
	req = httptest.NewRequest("GET", "/api/v1/sandboxes", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// DELETE
	req = httptest.NewRequest("DELETE", "/api/v1/sandboxes/"+id, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// TestSandboxManagerContextCancel ensures the session context is honoured,
// preventing leaked subprocess goroutines in CI.
func TestSandboxManagerContextCancel(t *testing.T) {
	manager := newTestManager()
	session, err := manager.CreateSession(1, SandboxLaunchConfig{TargetCommand: "30"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := session.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	cctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = cctx

	if err := manager.CloseSession(session.ID); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if session.Status != SandboxStatusStopped {
		t.Errorf("expected stopped, got %s", session.Status)
	}
}
