package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
)

// stage2Upstream is a Unix-socket upstream that records each request it receives. The
// gateway's real router reaches it through setupRoutes, so these tests exercise the same
// route table the public listener serves.
type stage2Upstream struct {
	mu    sync.Mutex
	paths []string
}

func (u *stage2Upstream) record(path string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.paths = append(u.paths, path)
}

func (u *stage2Upstream) count() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.paths)
}

func (u *stage2Upstream) start(t *testing.T) string {
	t.Helper()
	return startUnixUpstream(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u.record(r.URL.RequestURI())
		w.Header().Set("X-From", "upstream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream"))
	}))
}

// stage2Router builds the gateway's real route table (setupRoutes) for cfg. Optional
// model-call handler is installed before setupRoutes, as New does.
func stage2Router(t *testing.T, cfg *config.Config, model http.Handler) *Server {
	t.Helper()
	// A real WebGUI index.html, so an unguarded path would be answered by the SPA
	// catch-all with 200 instead of the empty 404 an empty directory gives.
	webgui := t.TempDir()
	if err := os.WriteFile(filepath.Join(webgui, "index.html"), []byte("<!doctype html><title>webgui</title>"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &Server{
		config:          cfg,
		logger:          zap.NewNop(),
		router:          mux.NewRouter(),
		modelCallProxy:  model,
		webguiStaticDir: webgui,
	}
	if err := s.setupRoutes(); err != nil {
		t.Fatalf("setupRoutes: %v", err)
	}
	return s
}

func serve(s *Server, method, target, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	return rec
}

// TestStage2AppPathsForwardToAppSocket checks that the launcher-owned prefixes added in
// stage 2 (/app, /network, /turn, /gateway except model-proxy) reach the app socket.
func TestStage2AppPathsForwardToAppSocket(t *testing.T) {
	var app stage2Upstream
	cfg := &config.Config{Port: 8080, KnirvServerAppSocketPath: app.start(t)}
	s := stage2Router(t, cfg, nil)

	cases := []struct{ method, target string }{
		{http.MethodGet, "/app"},
		{http.MethodGet, "/app/"},
		{http.MethodGet, "/app/_next/static/chunks/main.js"},
		{http.MethodGet, "/app/menu/index.txt"},
		{http.MethodGet, "/network"},
		{http.MethodGet, "/network/status?x=1"},
		{http.MethodPost, "/turn"},
		{http.MethodGet, "/turn/health"},
		{http.MethodGet, "/gateway"},
		{http.MethodGet, "/gateway/status"},
	}
	for _, c := range cases {
		before := app.count()
		rec := serve(s, c.method, c.target, "", nil)
		if rec.Code != http.StatusOK || rec.Header().Get("X-From") != "upstream" {
			t.Fatalf("%s %s = %d (X-From %q), want forwarded 200", c.method, c.target, rec.Code, rec.Header().Get("X-From"))
		}
		if app.count() != before+1 {
			t.Fatalf("%s %s did not reach the app socket", c.method, c.target)
		}
	}
	if got := app.paths[len(app.paths)-1]; got != "/gateway/status" {
		t.Fatalf("last forwarded path = %q, want /gateway/status (path must pass unchanged)", got)
	}
}

// TestStage2GatewayModelProxyNotForwarded checks that /gateway/model-proxy/ still reaches
// the gateway's model-call proxy and never the app socket.
func TestStage2GatewayModelProxyNotForwarded(t *testing.T) {
	var app stage2Upstream
	var modelCalls int
	model := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		modelCalls++
		w.WriteHeader(http.StatusTeapot)
	})
	cfg := &config.Config{Port: 8080, KnirvServerAppSocketPath: app.start(t)}
	s := stage2Router(t, cfg, model)

	for _, target := range []string{
		"/gateway/model-proxy/cid-1/anthropic/v1/messages",
		"/gateway/model-proxy/cid-1/openai/v1/chat/completions",
	} {
		rec := serve(s, http.MethodPost, target, "{}", nil)
		if rec.Code != http.StatusTeapot {
			t.Fatalf("POST %s = %d, want model proxy 418", target, rec.Code)
		}
	}
	if modelCalls != 2 {
		t.Fatalf("model proxy calls = %d, want 2", modelCalls)
	}
	if app.count() != 0 {
		t.Fatalf("model-proxy request reached the app socket (%d calls)", app.count())
	}
}

// TestStage2GitPublicRouteIs404 is the behavioural guard for /git/*: through the real
// router, no git path is answered by the SPA, the app socket or the backend.
func TestStage2GitPublicRouteIs404(t *testing.T) {
	var app, backend stage2Upstream
	cfg := &config.Config{
		Port:                     8080,
		KnirvServerAppSocketPath: app.start(t),
		BackendSocketPath:        backend.start(t),
	}
	s := stage2Router(t, cfg, nil)

	for _, c := range []struct{ method, target string }{
		{http.MethodGet, "/git/proj1.git/info/refs?service=git-receive-pack"},
		{http.MethodPost, "/git/proj1.git/git-receive-pack"},
		{http.MethodPost, "/git/proj1.git/create"},
		{http.MethodGet, "/git/"},
	} {
		rec := serve(s, c.method, c.target, "", nil)
		if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), "webgui") {
			t.Fatalf("%s %s = %d (%q), want 404 and not the SPA shell", c.method, c.target, rec.Code, rec.Body.String())
		}
	}
	if app.count() != 0 || backend.count() != 0 {
		t.Fatalf("/git/* reached an upstream: app=%d backend=%d", app.count(), backend.count())
	}
}

// TestStage2KnirvAgentCallersReachBackend checks the KNIRVAGENT calls that depend on
// KNIRV_SERVER_BASE_URL now that it is the public origin: each one reaches the backend
// through the gateway's /dve/ and /api/v1/ routes.
func TestStage2KnirvAgentCallersReachBackend(t *testing.T) {
	var app, backend stage2Upstream
	cfg := &config.Config{
		Port:                     8080,
		KnirvServerAppSocketPath: app.start(t),
		BackendSocketPath:        backend.start(t),
	}
	s := stage2Router(t, cfg, nil)

	cases := []struct{ method, target, wantPath string }{
		{http.MethodPost, "/dve/sessions", "/dve/sessions"},
		{http.MethodGet, "/dve/controller-session?environment_id=env1", "/dve/controller-session?environment_id=env1"},
		{http.MethodPost, "/dve/sessions/sess-1/chat-socket", "/dve/sessions/sess-1/chat-socket"},
		{http.MethodPost, "/api/v1/dve/node-1/agent/response", "/api/v1/dve/node-1/agent/response"},
	}
	for _, c := range cases {
		before := backend.count()
		rec := serve(s, c.method, c.target, "{}", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s = %d, want backend 200", c.method, c.target, rec.Code)
		}
		if backend.count() != before+1 || backend.paths[len(backend.paths)-1] != c.wantPath {
			t.Fatalf("%s %s did not reach backend as %s", c.method, c.target, c.wantPath)
		}
	}
	if app.count() != 0 {
		t.Fatalf("agent callers reached the app socket (%d)", app.count())
	}
}

// TestStage2MonitorRoutesRequireAdmin checks the admin gate on monitor routes: 401 with no
// token, 403 for a non-admin role, 200 for an admin, and the monitor socket is reached
// only on the admin path.
func TestStage2MonitorRoutesRequireAdmin(t *testing.T) {
	const secret = "stage2-test-secret"
	t.Setenv("KNIRV_JWT_SECRET", secret)

	var monitor stage2Upstream
	cfg := &config.Config{Port: 8080, MonitorSocketPath: monitor.start(t)}
	s := stage2Router(t, cfg, nil)

	sign := func(role string) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &adminClaims{
			Role:             role,
			RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		})
		out, err := tok.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}
		return out
	}
	auth := func(tok string) map[string]string { return map[string]string{"Authorization": "Bearer " + tok} }

	paths := []string{
		"/api/v1/monitor/metrics",
		"/api/v1/knirvbase/collections",
		"/api/v1/network/stats",
		"/api/v1/onboarding/applications",
	}
	for _, p := range paths {
		if rec := serve(s, http.MethodGet, p, "", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("GET %s without token = %d, want 401", p, rec.Code)
		}
		if rec := serve(s, http.MethodGet, p, "", auth(sign("user"))); rec.Code != http.StatusForbidden {
			t.Fatalf("GET %s as user = %d, want 403", p, rec.Code)
		}
	}
	if monitor.count() != 0 {
		t.Fatalf("monitor socket reached without admin role (%d calls)", monitor.count())
	}
	for _, p := range paths {
		if rec := serve(s, http.MethodGet, p, "", auth(sign("admin"))); rec.Code != http.StatusOK {
			t.Fatalf("GET %s as admin = %d, want 200", p, rec.Code)
		}
	}
	if monitor.count() != len(paths) {
		t.Fatalf("admin requests reached monitor %d times, want %d", monitor.count(), len(paths))
	}
}

// TestStage2ServerBaseURLLoopGuard checks the loop guard for the legacy KNIRVSERVER
// fallback: a base URL equal to the gateway's public origin is never proxied to.
func TestStage2ServerBaseURLLoopGuard(t *testing.T) {
	cases := []struct {
		base, public string
		want         bool
	}{
		{"https://gateway.example.net", "https://gateway.example.net", true},
		{"https://GATEWAY.example.net/", "https://gateway.example.net", true},
		{"http://127.0.0.1:8090", "https://gateway.example.net", false},
		{"https://gateway.example.net:8443", "https://gateway.example.net", false},
		{"", "https://gateway.example.net", false},
		{"http://127.0.0.1:8090", "", false},
	}
	for _, c := range cases {
		if got := serverBaseURLLoopsToGateway(c.base, c.public); got != c.want {
			t.Errorf("serverBaseURLLoopsToGateway(%q, %q) = %v, want %v", c.base, c.public, got, c.want)
		}
	}
}

// stage2Token signs a bearer token with the given role using the gateway's admin secret.
func stage2Token(t *testing.T, secret, role string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &adminClaims{
		Role:             role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	})
	out, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return out
}

// TestStage2BackendMonitorRoutesRequireAdmin checks that the monitor-owned paths served by the
// backend socket (the whole /api/v1/monitor/ prefix and exact /api/v1/actuarial/metrics) are
// admin-only through the real router: 401 without a token, 403 for a user, 200 for an admin, and
// the backend is reached only on the admin path. An unrelated /api/v1/ path stays unguarded.
func TestStage2BackendMonitorRoutesRequireAdmin(t *testing.T) {
	const secret = "stage2-backend-monitor-secret"
	t.Setenv("KNIRV_JWT_SECRET", secret)

	var backend stage2Upstream
	cfg := &config.Config{Port: 8080, BackendSocketPath: backend.start(t)}
	s := stage2Router(t, cfg, nil)

	gated := []string{
		"/api/v1/monitor/root-failover",
		"/api/v1/monitor/grafana",
		"/api/v1/monitor/bao-audit",
		"/api/v1/monitor/unlisted",
		"/api/v1/actuarial/metrics",
	}
	for _, p := range gated {
		if rec := serve(s, http.MethodGet, p, "", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("GET %s without token = %d, want 401", p, rec.Code)
		}
		auth := map[string]string{"Authorization": "Bearer " + stage2Token(t, secret, "user")}
		if rec := serve(s, http.MethodGet, p, "", auth); rec.Code != http.StatusForbidden {
			t.Fatalf("GET %s as user = %d, want 403", p, rec.Code)
		}
	}
	if backend.count() != 0 {
		t.Fatalf("backend reached without admin role (%d calls)", backend.count())
	}
	for _, p := range gated {
		auth := map[string]string{"Authorization": "Bearer " + stage2Token(t, secret, "admin")}
		if rec := serve(s, http.MethodGet, p, "", auth); rec.Code != http.StatusOK {
			t.Fatalf("GET %s as admin = %d, want 200", p, rec.Code)
		}
	}
	if backend.count() != len(gated) {
		t.Fatalf("admin requests reached backend %d times, want %d", backend.count(), len(gated))
	}

	// Control: the gate is exact. Other backend API paths still need no token.
	if rec := serve(s, http.MethodGet, "/api/v1/actuarial/other", "", nil); rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/actuarial/other without token = %d, want 200 (gate must be exact)", rec.Code)
	}
}
