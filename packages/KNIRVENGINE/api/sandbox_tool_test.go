package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestHandleToolScan_MissingSession(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	req := httptest.NewRequest("POST", "/api/v1/sandboxes/nonexistent/tools/semgrep/run", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing session, got %d", w.Code)
	}
}

func TestHandleToolScan_UnknownTool(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	// Create a session (don't start it — we're just testing routing).
	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	body := `{"ruleset": "p/secrets"}`
	req := httptest.NewRequest("POST",
		"/api/v1/sandboxes/"+session.ID+"/tools/nonexistent/run",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown tool, got %d", w.Code)
	}
}

func TestHandleToolScan_SemgrepParsing(t *testing.T) {
	// Test the Semgrep JSON parsing directly.
	jsonOutput := `{
		"results": [
			{
				"check_id": "go.lang.security.audit.crypto.use-of-md5",
				"path": "internal/cache/keys.go",
				"start": {"line": 42},
				"extra": {
					"message": "MD5 is a weak hash",
					"severity": "ERROR",
					"metadata": {"fixable": true}
				}
			}
		]
	}`

	adapter := lane1Adapters["semgrep"]
	result, err := adapter.parseOutput([]byte(jsonOutput))
	if err != nil {
		t.Fatalf("parseOutput failed: %v", err)
	}

	var findings []SemgrepFinding
	if err := json.Unmarshal(result, &findings); err != nil {
		t.Fatalf("failed to unmarshal findings: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "go.lang.security.audit.crypto.use-of-md5" {
		t.Errorf("wrong rule ID: %s", f.RuleID)
	}
	if f.File != "internal/cache/keys.go" {
		t.Errorf("wrong file: %s", f.File)
	}
	if f.Line != 42 {
		t.Errorf("wrong line: %d", f.Line)
	}
	if f.Severity != "ERROR" {
		t.Errorf("wrong severity: %s", f.Severity)
	}
	if !f.HasFix {
		t.Errorf("expected hasFix to be true")
	}
}

func TestLane1RunningLock(t *testing.T) {
	// Reset the running tracker.
	lane1Running.Lock()
	lane1Running.sessions = make(map[string]bool)
	lane1Running.Unlock()

	key := "test-session/semgrep"

	// Acquire.
	lane1Running.Lock()
	lane1Running.sessions[key] = true
	lane1Running.Unlock()

	// Verify it's locked.
	lane1Running.Lock()
	if !lane1Running.sessions[key] {
		t.Error("expected session to be locked")
	}
	lane1Running.Unlock()

	// Release.
	lane1Running.Lock()
	delete(lane1Running.sessions, key)
	lane1Running.Unlock()

	// Verify unlocked.
	lane1Running.Lock()
	if lane1Running.sessions[key] {
		t.Error("expected session to be unlocked")
	}
	lane1Running.Unlock()
}

func TestSpawnJoined_RequiresInnerPid(t *testing.T) {
	m := NewSandboxManager()
	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Without a started sandbox, spawnJoined should fail because there's no
	// outer PID to resolve an inner from.
	session.Pid = 0
	_, err = session.spawnJoined("ls")
	if err == nil {
		t.Error("expected error when InnerPid and Pid are both 0")
	}
	if !strings.Contains(err.Error(), "no outer PID") {
		t.Errorf("expected 'no outer PID' error, got: %v", err)
	}
}

func TestProxychainsConfGeneration(t *testing.T) {
	cfg := proxychainsConfig{
		ChainType: "dynamic",
		ProxyList: []proxychainsProxy{
			{Type: "socks5", Host: "127.0.0.1", Port: 9050},
			{Type: "http", Host: "10.0.0.1", Port: 8080},
		},
		QuietMode:      true,
		TCPReadTimeout: 3000,
	}

	conf := buildProxychainsConf(cfg)

	if !strings.Contains(conf, "dynamic_chain") {
		t.Errorf("expected dynamic_chain in config, got:\n%s", conf)
	}
	if !strings.Contains(conf, "socks5 127.0.0.1 9050") {
		t.Errorf("expected socks5 proxy in config, got:\n%s", conf)
	}
	if !strings.Contains(conf, "http 10.0.0.1 8080") {
		t.Errorf("expected http proxy in config, got:\n%s", conf)
	}
	if !strings.Contains(conf, "quiet_mode") {
		t.Errorf("expected quiet_mode in config, got:\n%s", conf)
	}
	if !strings.Contains(conf, "tcp_read_time_out 3000") {
		t.Errorf("expected tcp_read_time_out in config, got:\n%s", conf)
	}
}

func TestIsRoot(t *testing.T) {
	// On a typical dev machine we're not root. On CI we might be.
	// Just verify the function doesn't panic.
	_ = isRoot()
}

func TestEnsureToolDependency_NonLinux(t *testing.T) {
	// This test is platform-dependent; on Linux it should check the strategy.
	// We just verify it doesn't panic.
	st := EnsureToolDependency("semgrep", realCommandRunner)
	// We can't assert presence without knowing the environment.
	_ = st.Binary
}
