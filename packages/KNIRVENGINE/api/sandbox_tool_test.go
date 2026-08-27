package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestTreeSitterParsesRealSyntax(t *testing.T) {
	tree, err := parseWithTreeSitter("package sample\nfunc Answer() int { return 42 }\n", "go")
	if err != nil {
		t.Fatalf("parseWithTreeSitter: %v", err)
	}
	if tree.Type != "source_file" {
		t.Fatalf("root type = %q, want source_file", tree.Type)
	}
	if len(tree.Children) == 0 || tree.Children[0].Type != "package_clause" {
		t.Fatalf("expected parser-produced package clause, got %#v", tree.Children)
	}
}

func TestFindSAMLInFlowsDecodesCapturedForm(t *testing.T) {
	xml := `<Response ID="response-id"><Issuer>issuer</Issuer></Response>`
	session := &SandboxSession{proxyFlows: []SandboxProxyFlow{{ID: 7, RequestBody: "SAMLResponse=" + base64.StdEncoding.EncodeToString([]byte(xml))}}}
	if got := findSAMLInFlows(session, 7); got != xml {
		t.Fatalf("SAML flow decode = %q, want %q", got, xml)
	}
	if got := findSAMLInFlows(session, 8); got != "" {
		t.Fatalf("unexpected SAML from another flow: %q", got)
	}
}

func TestParseCutterOutput(t *testing.T) {
	result, err := parseCutterOutput([]byte(`[{"name":"main","addr":4198400,"size":32}][{"offset":4198400,"opcode":"ret"}]`))
	if err != nil {
		t.Fatalf("parseCutterOutput: %v", err)
	}
	if len(result.Functions) != 1 || result.Functions[0].Address != "0x401000" {
		t.Fatalf("functions = %#v", result.Functions)
	}
	if !strings.Contains(result.Listing, "opcode") {
		t.Fatalf("listing did not contain formatted disassembly: %q", result.Listing)
	}
}

func TestParseCutterOutputWithoutDisassemblyGraph(t *testing.T) {
	result, err := parseCutterOutput([]byte(`[{"name":"main","addr":4198400,"size":32}]`))
	if err != nil {
		t.Fatalf("parseCutterOutput without graph: %v", err)
	}
	if len(result.Functions) != 1 || result.Functions[0].Name != "main" {
		t.Fatalf("functions = %#v", result.Functions)
	}
	if result.Listing != "" {
		t.Fatalf("listing = %q, want empty when Rizin does not emit a graph", result.Listing)
	}
}

func TestValidateCutterBinaryPath(t *testing.T) {
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "target")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	session := &SandboxSession{TargetCommand: binaryPath, binds: []SandboxBind{{Mode: "ro-bind", Src: tempDir, Dst: "/target"}}}

	got, err := validateCutterBinaryPath(session, "")
	if err == nil || got != "" {
		t.Fatalf("empty binary path = (%q, %v), want validation error", got, err)
	}
	got, err = validateCutterBinaryPath(session, binaryPath)
	if err != nil || got != binaryPath {
		t.Fatalf("target binary = (%q, %v), want (%q, nil)", got, err, binaryPath)
	}
	if _, err := validateCutterBinaryPath(session, tempDir); err == nil {
		t.Fatal("directory was accepted as a Cutter binary")
	}

	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := validateCutterBinaryPath(session, outside); err == nil {
		t.Fatal("unmounted binary was accepted")
	}
}

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

func TestMountedTargetDirSkipsSandboxBootstrapBinds(t *testing.T) {
	session := &SandboxSession{binds: []SandboxBind{
		{Mode: "ro-bind", Src: "/usr", Dst: "/usr"},
		{Mode: "ro-bind", Src: "/lib", Dst: "/lib"},
		{Mode: "bind", Src: "/workspace/target", Dst: "/workspace/target"},
	}}
	if got := mountedTargetDir(session); got != "/workspace/target" {
		t.Fatalf("mountedTargetDir() = %q, want project bind", got)
	}
}

func TestAllPlannedToolsAreRegistered(t *testing.T) {
	for _, tool := range []string{"semgrep", "jadx", "ilspy", "jwt-tool"} {
		if !isLane1Tool(tool) {
			t.Errorf("Lane 1 tool %q is not registered", tool)
		}
	}
	for _, tool := range []string{"bpftrace", "tshark", "zeek", "afl-fuzz"} {
		if !isLane2Tool(tool) {
			t.Errorf("Lane 2 tool %q is not registered", tool)
		}
	}
	if !isLane3Tool("frida") || !isLane5Tool("cutter") || !isLane6Tool("tree-sitter") || !isLane6Tool("saml-raider") {
		t.Error("one or more Lane 3/5/6 tools are not registered")
	}
}

func TestAFLUsesDocumentedCorePatternFallback(t *testing.T) {
	adapter, ok := lane2Adapters["afl-fuzz"]
	if !ok {
		t.Fatal("afl-fuzz adapter is not registered")
	}
	if !containsString(adapter.env, "AFL_I_DONT_CARE_ABOUT_MISSING_CRASHES=1") {
		t.Fatalf("afl-fuzz environment = %v, want AFL core-pattern fallback", adapter.env)
	}
	if !containsString(adapter.env, "AFL_SKIP_CPUFREQ=1") {
		t.Fatalf("afl-fuzz environment = %v, want AFL CPU-frequency fallback", adapter.env)
	}
}

func TestAFLWritesToSessionWorkspaceRatherThanProject(t *testing.T) {
	adapter := lane2Adapters["afl-fuzz"]
	args, err := adapter.buildArgs(&SandboxSession{}, nil)
	if err != nil {
		t.Fatalf("build AFL arguments: %v", err)
	}
	if !containsString(args, filepath.Join(aflWorkspaceMount, "out")) {
		t.Fatalf("AFL arguments = %v, want output under %s", args, aflWorkspaceMount)
	}
	if containsString(args, "out") {
		t.Fatalf("AFL arguments = %v, must not use a relative project output directory", args)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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

func TestEnsureToolDependencyUsesManagedVenv(t *testing.T) {
	withFakeManagedTools(t)
	st := EnsureToolDependency("semgrep", func(string, ...string) error {
		t.Fatal("installer should not run when the managed venv already has semgrep")
		return nil
	})
	if !st.Present {
		t.Fatal("expected managed semgrep executable to be detected")
	}
}

func TestPlannedAcquisitionStrategies(t *testing.T) {
	assertStrategy := func(binary string, expected interface{}) {
		t.Helper()
		actual := acquireStrategyFor(binary)
		switch expected.(type) {
		case pipStrategy:
			if _, ok := actual.(pipStrategy); !ok {
				t.Errorf("%s should use pip strategy, got %T", binary, actual)
			}
		case dotnetToolStrategy:
			if _, ok := actual.(dotnetToolStrategy); !ok {
				t.Errorf("%s should use dotnet tool strategy, got %T", binary, actual)
			}
		case githubReleaseStrategy:
			if _, ok := actual.(githubReleaseStrategy); !ok {
				t.Errorf("%s should use GitHub release strategy, got %T", binary, actual)
			}
		case packageManagerStrategy:
			if _, ok := actual.(packageManagerStrategy); !ok {
				t.Errorf("%s should use package manager strategy, got %T", binary, actual)
			}
		}
	}
	for _, binary := range []string{"semgrep", "jwt_tool.py", "frida"} {
		assertStrategy(binary, pipStrategy{})
	}
	for _, binary := range []string{"jadx", "frida-server"} {
		assertStrategy(binary, githubReleaseStrategy{})
	}
	strategy, ok := toolAcquireStrategies["ilspycmd"].(dotnetToolStrategy)
	if !ok {
		t.Fatalf("ilspycmd should use dotnetToolStrategy, got %T", toolAcquireStrategies["ilspycmd"])
	}
	if strategy.version != "9.1.0.7988" {
		t.Fatalf("ilspycmd version = %q, want .NET 8-compatible 9.1.0.7988", strategy.version)
	}
	for _, binary := range []string{"java", "dotnet", "rizin", "zeek"} {
		assertStrategy(binary, packageManagerStrategy{})
	}
}

func TestHandleToolStreamStart_MissingSession(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	req := httptest.NewRequest("POST", "/api/v1/sandboxes/nonexistent/tools/bpftrace/start", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing session, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolStreamStart_UnknownTool(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "stream-test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/sandboxes/"+session.ID+"/tools/nonexistent/start", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown tool, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolStreamStop_NoStream(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "stream-test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/sandboxes/"+session.ID+"/tools/bpftrace/stop", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when no stream exists, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolStreamWS_NoStream(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "stream-test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/sandboxes/"+session.ID+"/tools/bpftrace/ws", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when no stream exists, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolAttach_MissingSession(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	body := `{"pid": 1234}`
	req := httptest.NewRequest("POST", "/api/v1/sandboxes/nonexistent/tools/frida/attach", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing session, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolAttach_UnknownTool(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "attach-test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	body := `{"pid": 1234}`
	req := httptest.NewRequest("POST", "/api/v1/sandboxes/"+session.ID+"/tools/nonexistent/attach", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown tool, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolDetach_NoAttach(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "attach-test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/sandboxes/"+session.ID+"/tools/frida/detach", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when no attach exists, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolNative_MissingSession(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	req := httptest.NewRequest("POST", "/api/v1/sandboxes/nonexistent/tools/tree-sitter/native", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing session, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolNative_UnknownTool(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "native-test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/sandboxes/"+session.ID+"/tools/nonexistent/native", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown tool, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolNative_RegisteredTool(t *testing.T) {
	registerLane6Tool("echo-native", func(s *SandboxSession, args json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(map[string]string{"echo": string(args)})
	})
	defer func() { delete(lane6Registry.tools, "echo-native") }()

	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "native-test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	body := `{"message": "hello"}`
	req := httptest.NewRequest("POST", "/api/v1/sandboxes/"+session.ID+"/tools/echo-native/native", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for registered native tool, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool           `json:"success"`
		Data    ToolScanResult `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected successful response")
	}
	if resp.Data.Tool != "echo-native" {
		t.Errorf("expected tool echo-native, got %s", resp.Data.Tool)
	}
	if resp.Data.StartedAt.IsZero() {
		t.Error("expected non-zero StartedAt")
	}
}

func TestHandleToolHeadless_MissingSession(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	body := `{"binaryPath": "/tmp/test.bin"}`
	req := httptest.NewRequest("POST", "/api/v1/sandboxes/nonexistent/tools/cutter/analyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing session, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolHeadless_UnknownTool(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "headless-test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	body := `{"binaryPath": "/tmp/test.bin"}`
	req := httptest.NewRequest("POST", "/api/v1/sandboxes/"+session.ID+"/tools/nonexistent/analyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown tool, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolHeadless_StartedAtIsTime(t *testing.T) {
	registerLane5Tool("echo-headless", lane5Adapter{
		binary: "echo",
		analysisCmd: func(session *SandboxSession, binaryPath string, args json.RawMessage) ([]string, error) {
			return []string{"hello"}, nil
		},
		parseOutput: func(stdout []byte) (*ToolHeadlessResult, error) {
			return &ToolHeadlessResult{
				Tool:       "echo-headless",
				RawOutput:  string(stdout),
				Functions:  nil,
				Decompiled: "",
				Listing:    string(stdout),
				StartedAt:  time.Now(),
				DurationMs: 0,
			}, nil
		},
	})
	defer func() { delete(lane5Adapters, "echo-headless") }()

	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "headless-test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	// Restored sessions may not have the process-owned context populated. The
	// handler must fall back to the request context rather than panic and make
	// the GUI reverse proxy report an EOF/502.
	session.ctx = nil

	body := `{"binaryPath": "/tmp/test.bin"}`
	req := httptest.NewRequest("POST", "/api/v1/sandboxes/"+session.ID+"/tools/echo-headless/analyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool               `json:"success"`
		Data    ToolHeadlessResult `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected successful response")
	}
	if resp.Data.StartedAt.IsZero() {
		t.Error("expected non-zero StartedAt time")
	}
}

func TestResolveInnerPid_NoOuterPid(t *testing.T) {
	session := &SandboxSession{Pid: 0}
	if err := session.resolveInnerPid(); err == nil {
		t.Error("expected error when outer PID is 0")
	}
}

func TestResolveTargetPid_NoInnerPid(t *testing.T) {
	session := &SandboxSession{Pid: 0, InnerPid: 0}
	if _, err := session.resolveTargetPid(); err == nil {
		t.Error("expected error when inner PID is 0")
	}
}

func TestHandleToolProxychains_UnknownSession(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	body := `{"chainType": "dynamic", "proxyList": []}`
	req := httptest.NewRequest("POST", "/api/v1/sandboxes/nonexistent/tools/proxychains/configure", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing session, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolProxychains_ValidConfig(t *testing.T) {
	m := NewSandboxManager()
	router := mux.NewRouter()
	m.registerToolRoutes(router)

	session, err := m.CreateSession(1, SandboxLaunchConfig{
		TargetLabel:   "proxy-test",
		TargetCommand: "sleep 60",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	body := `{
		"chainType": "dynamic",
		"proxyList": [
			{"type": "socks5", "host": "127.0.0.1", "port": 9050},
			{"type": "http", "host": "10.0.0.1", "port": 8080}
		],
		"quietMode": true,
		"tcpReadTimeout": 3000
	}`
	req := httptest.NewRequest("POST", "/api/v1/sandboxes/"+session.ID+"/tools/proxychains/configure", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 200 or 500, got %d: %s", w.Code, w.Body.String())
	}

	if w.Code == http.StatusOK {
		var resp struct {
			Success bool              `json:"success"`
			Data    map[string]string `json:"data"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if !resp.Success {
			t.Fatal("expected successful response")
		}
		if resp.Data["status"] != "configured" {
			t.Errorf("expected status configured, got %s", resp.Data["status"])
		}
		if resp.Data["configPath"] == "" {
			t.Error("expected non-empty configPath")
		}
	}
}
