package launcher

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	knirvgraph "KNIRVGRAPH"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// Every test here runs a fake shell script in place of knirvgraph. The script's
// name is unique so the manager's `pkill -x <name>` cannot match a real process,
// and every listener port is 0 so no fixed TCP port is claimed by the test.

const n17Token = "n17-test-token-not-real"

// writeFakeKNIRVGraph writes a stand-in for knirvgraph. It records its arguments
// and environment under $KNIRV_APP_DATA_DIR, then writes its pid last and sleeps
// (exec, so the pid is the sleep's). The pid file is the readiness signal.
func writeFakeKNIRVGraph(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fakegraph-n17-test")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$KNIRV_APP_DATA_DIR/fake-args.txt"
env > "$KNIRV_APP_DATA_DIR/fake-env.txt"
echo $$ > "$KNIRV_APP_DATA_DIR/fake-pid.txt"
exec sleep 300
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// n17Isolate points the launcher's app data dir at a temp dir and makes the
// manager use the fake binary, even if the embedded one would be found first.
func n17Isolate(t *testing.T, fake string) (appDir string) {
	t.Helper()
	appDir = filepath.Join(t.TempDir(), "knirvserver")
	t.Setenv("KNIRV_APP_DATA_DIR", appDir)
	t.Setenv("KNIRV_GRAPH_BINARY_PATH", fake)
	t.Setenv("KNIRV_GRAPH_BINARY_DIR", "")
	return appDir
}

func n17Readiness(t *testing.T, appDir string) (pid int, args, env string) {
	t.Helper()
	pidFile := filepath.Join(appDir, "fake-pid.txt")
	deadline := time.Now().Add(5 * time.Second)
	for {
		if b, err := os.ReadFile(pidFile); err == nil && len(strings.TrimSpace(string(b))) > 0 {
			p, err := strconv.Atoi(strings.TrimSpace(string(b)))
			if err != nil {
				t.Fatalf("pid file: %v", err)
			}
			a, _ := os.ReadFile(filepath.Join(appDir, "fake-args.txt"))
			e, _ := os.ReadFile(filepath.Join(appDir, "fake-env.txt"))
			return p, string(a), string(e)
		}
		if time.Now().After(deadline) {
			t.Fatal("fake knirvgraph never reported its pid")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// n17LastEnv returns the value a child sees for key. On duplicates the last entry
// wins, which is how exec interprets the environment.
func n17LastEnv(envText, key string) (string, bool) {
	val, found := "", false
	for _, line := range strings.Split(envText, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if ok && k == key {
			val, found = v, true
		}
	}
	return val, found
}

func n17ProcessGone(pid int) bool {
	return errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
}

// The child gets the launcher's ulorad directory, socket and token, and the
// app data dir; the token is only ever passed in the environment.
func TestKNIRVGraphChildEnvMatchesLauncherUloraContract(t *testing.T) {
	fake := writeFakeKNIRVGraph(t)
	appDir := n17Isolate(t, fake)

	m, err := startKNIRVGraphWith(appDir, n17Token, fake, "", knirvGraphPorts{}, zap.NewNop())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = m.Stop() })
	_, _, env := n17Readiness(t, appDir)

	want := map[string]string{
		"KNIRV_APP_DATA_DIR": appDir,
		"ULORA_DATA_DIR":     uloraDataDir(appDir),
		"ULORA_SOCKET_PATH":  filepath.Join(appDir, "sockets", "ulora.sock"),
		"ULORA_AUTH_TOKEN":   n17Token,
	}
	for k, v := range want {
		if got, ok := n17LastEnv(env, k); !ok || got != v {
			t.Fatalf("child %s = %q (present %v), want %q", k, got, ok, v)
		}
	}
}

// The token reaches the child environment but never a log line.
func TestKNIRVGraphLogsNeverContainToken(t *testing.T) {
	fake := writeFakeKNIRVGraph(t)
	appDir := n17Isolate(t, fake)

	core, logs := observer.New(zap.DebugLevel)
	m, err := startKNIRVGraphWith(appDir, n17Token, fake, "", knirvGraphPorts{}, zap.New(core))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = m.Stop() })
	n17Readiness(t, appDir)

	if logs.Len() == 0 {
		t.Fatal("expected manager log lines")
	}
	for _, entry := range logs.All() {
		if strings.Contains(fmt.Sprintf("%+v", entry), n17Token) {
			t.Fatalf("log entry leaks the auth token: %+v", entry)
		}
	}
}

// Stop ends the child process: it is not running and its pid is gone.
func TestKNIRVGraphStopsOnShutdown(t *testing.T) {
	fake := writeFakeKNIRVGraph(t)
	appDir := n17Isolate(t, fake)

	m, err := startKNIRVGraphWith(appDir, n17Token, fake, "", knirvGraphPorts{}, zap.NewNop())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	pid, _, _ := n17Readiness(t, appDir)
	if !m.IsRunning() {
		t.Fatal("manager should report running after start")
	}

	if err := m.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if m.IsRunning() {
		t.Fatal("manager still reports running after stop")
	}
	deadline := time.Now().Add(5 * time.Second)
	for !n17ProcessGone(pid) {
		if time.Now().After(deadline) {
			t.Fatalf("pid %d still alive after stop", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// The child is given the graph socket and its health checks go over that socket
// only. The launcher config sets no public address.
func TestKNIRVGraphUsesUnixSocketOnly(t *testing.T) {
	fake := writeFakeKNIRVGraph(t)
	appDir := n17Isolate(t, fake)

	m, err := startKNIRVGraphWith(appDir, n17Token, fake, "", knirvGraphPorts{}, zap.NewNop())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = m.Stop() })
	_, args, _ := n17Readiness(t, appDir)

	if !strings.Contains(args, "--socket\n"+filepath.Join(appDir, "sockets", "graph.sock")) {
		t.Fatalf("child args lack the graph socket: %q", args)
	}
	if !strings.HasPrefix(m.GetBaseURL(), "http://unix/") {
		t.Fatalf("health base URL %q is not a unix socket URL", m.GetBaseURL())
	}
}

// A missing or unusable binary is logged and the launcher carries on: the
// optional start returns, nothing is recorded as running, and the backend is
// not told to skip its own graph.
func TestKNIRVGraphMissingBinaryDegrades(t *testing.T) {
	appDir := filepath.Join(t.TempDir(), "knirvserver")
	t.Setenv("KNIRV_APP_DATA_DIR", appDir)
	orig := extractKNIRVGraphBinary
	t.Cleanup(func() { extractKNIRVGraphBinary = orig })
	extractKNIRVGraphBinary = func(string) (string, error) {
		return "", errors.New("embedded knirvgraph unavailable")
	}

	app := &ServerApp{internalAuthToken: n17Token}
	app.startKNIRVGraphOptional() // must not panic or exit

	if app.knirvGraphManager != nil {
		t.Fatal("manager recorded after a failed extraction")
	}
	if env := app.graphOwnershipEnv(); len(env) != 0 {
		t.Fatalf("backend told to skip its graph with no launcher graph: %v", env)
	}
	app.stopKNIRVGraph() // no-op, must not panic
}

// Wiring: the optional start extracts the binary, starts it with the launcher's
// env, records the manager, and stopKNIRVGraph ends it.
func TestKNIRVGraphStartAndStopThroughApp(t *testing.T) {
	fake := writeFakeKNIRVGraph(t)
	appDir := n17Isolate(t, fake)
	orig := extractKNIRVGraphBinary
	t.Cleanup(func() { extractKNIRVGraphBinary = orig })
	extractKNIRVGraphBinary = func(string) (string, error) { return fake, nil }

	app := &ServerApp{internalAuthToken: n17Token}
	app.startKNIRVGraphOptional()
	if app.knirvGraphManager == nil || !app.knirvGraphManager.IsRunning() {
		t.Fatal("launcher did not start KNIRVGRAPH")
	}
	pid, _, env := n17Readiness(t, appDir)
	if got, _ := n17LastEnv(env, "ULORA_DATA_DIR"); got != uloraDataDir(appDir) {
		t.Fatalf("child ULORA_DATA_DIR = %q, want %q", got, uloraDataDir(appDir))
	}
	if env := app.graphOwnershipEnv(); len(env) != 1 || env[0] != "KNIRV_GRAPH_ENABLED=false" {
		t.Fatalf("backend ownership env = %v, want KNIRV_GRAPH_ENABLED=false", env)
	}

	app.stopKNIRVGraph()
	if app.knirvGraphManager != nil {
		t.Fatal("manager still recorded after stop")
	}
	deadline := time.Now().Add(5 * time.Second)
	for !n17ProcessGone(pid) {
		if time.Now().After(deadline) {
			t.Fatalf("pid %d still alive after stopKNIRVGraph", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// The ownership env only exists while the launcher's graph is running.
func TestGraphOwnershipEnvOnlyWhenLauncherOwnsGraph(t *testing.T) {
	app := &ServerApp{}
	if env := app.graphOwnershipEnv(); env != nil {
		t.Fatalf("env = %v with no launcher graph, want nil", env)
	}
	app.knirvGraphManager = knirvgraph.NewManager(&knirvgraph.ManagerConfig{}, zap.NewNop())
	env := app.graphOwnershipEnv()
	if len(env) != 1 || env[0] != "KNIRV_GRAPH_ENABLED=false" {
		t.Fatalf("env = %v, want [KNIRV_GRAPH_ENABLED=false]", env)
	}
}

// Wiring in source: the optional start sits after ulorad and before backend
// start, the backend ownership env is added, and the graph stop comes after
// backend stop. A removed call site fails here even though the unit tests above
// call the helpers directly.
func TestKNIRVGraphLauncherCallSitesWired(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "launcher.go", nil, 0)
	if err != nil {
		t.Fatalf("parse launcher.go: %v", err)
	}
	pos := map[string][]token.Pos{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "app" {
			pos[sel.Sel.Name] = append(pos[sel.Sel.Name], call.Pos())
		}
		return true
	})
	one := func(name string) token.Pos {
		t.Helper()
		ps := pos[name]
		if len(ps) != 1 {
			t.Fatalf("app.%s called %d times in launcher.go, want 1", name, len(ps))
		}
		return ps[0]
	}
	ulora := one("startULoRA")
	graph := one("startKNIRVGraphOptional")
	backend := one("startBackend")
	if !(ulora < graph && graph < backend) {
		t.Fatalf("KNIRVGRAPH start must sit between startULoRA and startBackend")
	}
	if len(pos["graphOwnershipEnv"]) != 1 {
		t.Fatalf("app.graphOwnershipEnv must be called exactly once in launcher.go")
	}
	if !(one("stopBackend") < one("stopKNIRVGraph")) {
		t.Fatalf("KNIRVGRAPH must stop after the backend")
	}
}

// The graph child gets the same KNIRV_ORACLED_RPC_URL the backend gives its own
// graph: http://127.0.0.1:<backend API port>, through the real optional start.
func TestKNIRVGraphChildGetsOracledRPCURL(t *testing.T) {
	fake := writeFakeKNIRVGraph(t)
	appDir := n17Isolate(t, fake)
	orig := extractKNIRVGraphBinary
	t.Cleanup(func() { extractKNIRVGraphBinary = orig })
	extractKNIRVGraphBinary = func(string) (string, error) { return fake, nil }

	app := &ServerApp{internalAuthToken: n17Token, config: &Config{BackendPort: 18082}}
	app.startKNIRVGraphOptional()
	if app.knirvGraphManager == nil {
		t.Fatal("launcher did not start KNIRVGRAPH")
	}
	t.Cleanup(app.stopKNIRVGraph)
	_, _, env := n17Readiness(t, appDir)
	if got, ok := n17LastEnv(env, "KNIRV_ORACLED_RPC_URL"); !ok || got != "http://127.0.0.1:18082" {
		t.Fatalf("child KNIRV_ORACLED_RPC_URL = %q (present %v), want http://127.0.0.1:18082", got, ok)
	}
}

// With no configured backend port the variable is left unset rather than pointing
// at port 0.
func TestKNIRVGraphOracledRPCURLOmittedWithoutBackendPort(t *testing.T) {
	if got := (&ServerApp{}).knirvGraphOracledRPCURL(); got != "" {
		t.Fatalf("nil config URL = %q, want empty", got)
	}
	if got := (&ServerApp{config: &Config{}}).knirvGraphOracledRPCURL(); got != "" {
		t.Fatalf("zero port URL = %q, want empty", got)
	}
	cfg := knirvGraphConfig("/d", "/b", "t", "", knirvGraphPorts{})
	if _, ok := cfg.EnvOverrides["KNIRV_ORACLED_RPC_URL"]; ok {
		t.Fatal("KNIRV_ORACLED_RPC_URL set with an empty URL")
	}
}
