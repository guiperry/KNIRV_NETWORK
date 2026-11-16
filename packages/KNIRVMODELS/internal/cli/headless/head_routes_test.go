package headless

import (
	"bytes"
	"context"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"knirvmodels/internal/cli/controller"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// bareHeartDecisionServer builds a bare Server and runs the real registerBaseRoutes
// wiring, which is where registerRoutes mounts the decision routes. It does not call
// NewServer or registerRoutes: both run registerModels, which needs the SDK frame
// core and panics in this environment (see the U2 report).
//
// The app data dir sits under a regular file, so it cannot be created. A train POST
// then starts a background cycle that fails before writing anything. Without this,
// the goroutine extracts the trainer into the temp dir and races t.TempDir cleanup.
//
// Every KNIRV_HEAD_* and KNIRV_HEART_DECISION_* variable is cleared for the test, so
// the caller's shell cannot change the cycle's config. t.Setenv restores each value
// at cleanup.
func bareHeartDecisionServer(t *testing.T) *Server {
	t.Helper()
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "KNIRV_HEAD_") || strings.HasPrefix(name, "KNIRV_HEART_DECISION_") {
			t.Setenv(name, "")
			os.Unsetenv(name)
		}
	}
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KNIRV_APP_DATA_DIR", filepath.Join(blocker, "appdata"))
	s := &Server{controller: controller.NewController(false), mux: http.NewServeMux()}
	s.registerBaseRoutes()
	return s
}

func serveRoute(s *Server, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestHeartDecisionRoutesServeNewPaths(t *testing.T) {
	s := bareHeartDecisionServer(t)
	if rec := serveRoute(s, http.MethodPost, heartDecisionTrainPath); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "HEAD cycle started") {
		t.Fatalf("POST %s: got %d %q", heartDecisionTrainPath, rec.Code, rec.Body.String())
	}
	if rec := serveRoute(s, http.MethodGet, heartDecisionStatusPath); rec.Code != http.StatusOK {
		t.Fatalf("GET %s: got %d", heartDecisionStatusPath, rec.Code)
	}
	if rec := serveRoute(s, http.MethodGet, heartDecisionTrainPath); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET %s: got %d, want 405", heartDecisionTrainPath, rec.Code)
	}
	if rec := serveRoute(s, http.MethodPost, heartDecisionStatusPath); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST %s: got %d, want 405", heartDecisionStatusPath, rec.Code)
	}
}

func TestLegacyHeadAliasesMatchNewRoutes(t *testing.T) {
	s := bareHeartDecisionServer(t)

	newStatus := serveRoute(s, http.MethodGet, heartDecisionStatusPath)
	oldStatus := serveRoute(s, http.MethodGet, legacyHeadStatusPath)
	if oldStatus.Code != newStatus.Code || oldStatus.Body.String() != newStatus.Body.String() {
		t.Fatalf("status alias differs: old %d %q, new %d %q", oldStatus.Code, oldStatus.Body.String(), newStatus.Code, newStatus.Body.String())
	}

	newTrain := serveRoute(s, http.MethodPost, heartDecisionTrainPath)
	oldTrain := serveRoute(s, http.MethodPost, legacyHeadTrainPath)
	if oldTrain.Code != newTrain.Code || oldTrain.Body.String() != newTrain.Body.String() {
		t.Fatalf("train alias differs: old %d %q, new %d %q", oldTrain.Code, oldTrain.Body.String(), newTrain.Code, newTrain.Body.String())
	}

	if rec := serveRoute(s, http.MethodGet, legacyHeadTrainPath); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET %s: got %d, want 405", legacyHeadTrainPath, rec.Code)
	}
	if rec := serveRoute(s, http.MethodPost, legacyHeadStatusPath); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST %s: got %d, want 405", legacyHeadStatusPath, rec.Code)
	}
}

func TestLegacyHeadAliasLogsDeprecationOnce(t *testing.T) {
	s := bareHeartDecisionServer(t)
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	// A wrong-method request gets 405 and must not log or use up the once-guard.
	serveRoute(s, http.MethodGet, legacyHeadTrainPath)
	if got := buf.String(); strings.Contains(got, "DEPRECATED") {
		t.Fatalf("wrong-method request logged a deprecation:\n%s", got)
	}

	for i := 0; i < 3; i++ {
		serveRoute(s, http.MethodGet, legacyHeadStatusPath)
	}
	serveRoute(s, http.MethodGet, heartDecisionStatusPath)
	serveRoute(s, http.MethodPost, legacyHeadTrainPath)
	serveRoute(s, http.MethodPost, legacyHeadTrainPath)

	got := buf.String()
	if n := strings.Count(got, "DEPRECATED: "+legacyHeadStatusPath+" "); n != 1 {
		t.Fatalf("status alias logged %d deprecation lines, want 1:\n%s", n, got)
	}
	if n := strings.Count(got, "DEPRECATED: "+legacyHeadTrainPath+" "); n != 1 {
		t.Fatalf("train alias logged %d deprecation lines, want 1:\n%s", n, got)
	}
	if n := strings.Count(got, "DEPRECATED:"); n != 2 {
		t.Fatalf("logged %d deprecation lines in total, want 2:\n%s", n, got)
	}
}

// The backend reaches every model over the daemon's socket, so the daemon's mux
// must carry the whole /models/* namespace after the migration: hasher (validator),
// heart (generator), head (vision), and the HEART-hosted decision routes.
func TestDaemonServesModelsNamespace(t *testing.T) {
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	s := NewServer(t.TempDir()+"/h.sock", 0o660)
	for _, p := range []string{
		sdk.RegistryPath, sdk.HealthPath,
		sdk.HeadPrefix + "/models", sdk.HeadPrefix + "/capabilities", sdk.HeadPrefix + "/status",
		sdk.HeartPrefix + "/health",
		sdk.HeartDecisionPrefix + "/models", sdk.HeartDecisionPrefix + "/matrix", sdk.HeartDecisionPrefix + "/reason",
		sdk.HasherPrefix + "/status",
	} {
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code == http.StatusNotFound && rec.Body.String() == "404 page not found\n" {
			t.Fatalf("%s is not routed", p)
		}
	}
	// HEAD is the vision model now: its manifest route is routed and returns the
	// VLM manifest (200) whether or not a backend is promoted.
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, sdk.HeadPrefix+"/models", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: got %d, want 200 (routed VLM manifest)", sdk.HeadPrefix+"/models", rec.Code)
	}
}

// End to end over a real unix socket with the SDK client the backend uses: the
// registry lists all four models (hasher validator, heart generator, head vision,
// hands server_tasks) and asking about HEART does not build it.
func TestRegistryOverUnixSocketWithSDKClient(t *testing.T) {
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	t.Setenv("KNIRV_AUDIT_DIR", t.TempDir())
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	sock := t.TempDir() + "/m.sock"
	s := NewServer(sock, 0o660)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: s.mux}
	go srv.Serve(ln)
	defer srv.Close()

	c := sdk.NewClient("http://models", sdk.NewUnixHTTPClient(sock, 5*time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := c.Registry(ctx)
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{}
	for _, r := range rows {
		roles[r.Name] = r.Role
	}
	want := map[string]string{
		"hasher": sdk.RoleValidator,
		"heart":  sdk.RoleGenerator,
		"head":   sdk.RoleVision,
		"hands":  sdk.RoleHands,
	}
	if len(roles) != len(want) {
		t.Fatalf("registry has %d models, want %d: %+v", len(roles), len(want), rows)
	}
	for name, role := range want {
		if roles[name] != role {
			t.Fatalf("registry role %s = %q, want %q (%+v)", name, roles[name], role, rows)
		}
	}
	h, err := c.HeartHealth(ctx)
	if err != nil || h.Loaded {
		t.Fatalf("heart health %+v %v", h, err)
	}
	rows, _ = c.Registry(ctx)
	for _, r := range rows {
		if r.Name == "heart" && r.Loaded {
			t.Fatal("asking about HEART built it")
		}
	}
	if hs, err := c.HasherStatus(ctx); err != nil || hs.Loaded {
		t.Fatalf("hasher status %+v %v", hs, err)
	}
	res, err := c.HasherValidate(ctx, sdk.HasherValidateRequest{Prediction: 1, Domain: "no-such-domain"})
	if err != nil || res.Domain != "no-such-domain" {
		t.Fatalf("validate %+v %v", res, err)
	}
	// HEAD is the vision model now: its /models/head/models manifest route is routed
	// and returns the VLM manifest (200) whether or not a backend is promoted.
	if _, err := c.Manifest(ctx); err != nil {
		t.Fatalf("manifest: %v", err)
	}
	// HANDS reaches /models/hands/status and reports as server_tasks in the registry.
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, sdk.HandsPrefix+"/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: got %d, want 200 (HANDS reachable)", sdk.HandsPrefix+"/status", rec.Code)
	}
}
