package launcher

import (
	"context"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gin-gonic/gin"
)

// newAppSocketTestApp builds the real router (setupRoutes) inside a temp dir. HOME, XDG
// and KNIRV_APP_DATA_DIR point there, the validator key is a fixed dummy supplied through
// the env var, and the embedded SPA is an in-memory FS. No real key, .env or secret is
// read or written.
func newAppSocketTestApp(t *testing.T) (*ServerApp, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "xdg"))
	t.Setenv("KNIRV_APP_DATA_DIR", filepath.Join(tmp, "appdata"))
	dummyKey := make([]byte, 32)
	for i := range dummyKey {
		dummyKey[i] = 1
	}
	t.Setenv("KNIRV_PROOF_VALIDATOR_X25519_PRIVATE_KEY", base64.StdEncoding.EncodeToString(dummyKey))

	prevFiles := embeddedFiles
	embeddedFiles = fstest.MapFS{"frontend/out/index.html": {Data: []byte("<html>spa</html>")}}
	t.Cleanup(func() { embeddedFiles = prevFiles })

	// Unused TCP port for the gateway: any proxy to the gateway fails fast with 502.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gatewayPort := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	app := &ServerApp{
		config: &Config{
			BackendSocket:    filepath.Join(tmp, "sockets", "backend.sock"),
			GatewaySocket:    filepath.Join(tmp, "sockets", "gateway.sock"),
			GatewayPort:      gatewayPort,
			ProofStoreDir:    filepath.Join(tmp, "proof-protocol"),
			ProofLedgerDir:   filepath.Join(tmp, "proof-ledger"),
			ProofChainSocket: filepath.Join(tmp, "chain.sock"),
			NetworkMode:      "development",
		},
		router:            gin.New(),
		internalAuthToken: "test-internal-token",
	}
	if err := app.setupRoutes(); err != nil {
		t.Fatalf("setupRoutes: %v", err)
	}
	return app, tmp
}

// TestPublicTCPListenerAnswersOnlyHealthAndVersion runs the real router behind the public
// TCP handler. /health and /version answer, and every moved path answers 404.
func TestPublicTCPListenerAnswersOnlyHealthAndVersion(t *testing.T) {
	app, _ := newAppSocketTestApp(t)
	tcp := publicTCPHandler(app.router)

	for _, path := range []string{"/health", "/version"} {
		w := httptest.NewRecorder()
		tcp.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("TCP GET %s = %d, want 200", path, w.Code)
		}
	}

	moved := []string{
		"/", "/login", "/menu", "/_next/static/x.js", "/api/v1/system/info", "/api/v1/system/update",
		"/api/dve/projects", "/api/v1/knirv/validator-key", "/arena/", "/tunnel", "/tunnel/status",
		"/ws", "/ws/dve/d1/inner", "/git/proj.git/info/refs", "/explorer", "/network",
	}
	for _, path := range moved {
		w := httptest.NewRecorder()
		tcp.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("TCP GET %s = %d, want 404", path, w.Code)
		}
		if strings.Contains(w.Body.String(), "spa") {
			t.Fatalf("TCP GET %s served the SPA", path)
		}
	}
	w := httptest.NewRecorder()
	tcp.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/health", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("TCP POST /health = %d, want 404", w.Code)
	}
}

// TestAppSocketServesMovedPaths checks that the internal app socket serves the full
// router (so KNIRVGATEWAY can forward to it) with mode 0600 and no TCP bind.
func TestAppSocketServesMovedPaths(t *testing.T) {
	app, tmp := newAppSocketTestApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	app.startKnirvRouteSockets(ctx)

	sock := app.knirvServerAppSocketPath()
	if sock != filepath.Join(tmp, "sockets", "knirvserver-app.sock") {
		t.Fatalf("app socket path = %q", sock)
	}
	waitForSocket(t, sock)

	fi, err := os.Stat(sock)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("app socket mode = %o, want 600", fi.Mode().Perm())
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", sock)
		}},
	}
	get := func(path string, headers map[string]string) (int, string) {
		req, _ := http.NewRequest(http.MethodGet, "http://knirvserver-app"+path, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("GET %s over app socket: %v", path, err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	if code, _ := get("/api/v1/system/info", nil); code != http.StatusOK {
		t.Fatalf("app socket /api/v1/system/info = %d, want 200", code)
	}
	if code, body := get("/", nil); code != http.StatusOK || !strings.Contains(body, "spa") {
		t.Fatalf("app socket / = %d %q, want the SPA", code, body)
	}
	if code, _ := get("/git/proj.git/info/refs", nil); code != http.StatusNotFound {
		t.Fatalf("app socket /git/* = %d, want 404 (git lives on git.sock)", code)
	}
}

// TestForwardedRequestNeverLoopsBackToGateway checks the loop guard. A gateway-forwarded
// request with a gateway Referer is answered locally by the embedded SPA, while the same
// request without the forward marker is proxied to the gateway (502 with it down).
func TestForwardedRequestNeverLoopsBackToGateway(t *testing.T) {
	app, _ := newAppSocketTestApp(t)
	// A real server: the proxy to the gateway needs CloseNotify, which a recorder lacks.
	srv := httptest.NewServer(app.router)
	t.Cleanup(srv.Close)

	get := func(marker bool) (int, string) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/_next/static/missing.js", nil)
		req.Header.Set("Referer", "http://gateway/gateway/page")
		if marker {
			req.Header.Set(knirvServerAppForwardHeader, "1")
		}
		resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	if code, body := get(true); code != http.StatusOK || !strings.Contains(body, "spa") {
		t.Fatalf("forwarded asset = %d %q, want the local embedded SPA", code, body)
	}
	if code, _ := get(false); code != http.StatusBadGateway {
		t.Fatalf("direct asset with gateway referer = %d, want proxy to gateway (502 with it down)", code)
	}
}

// TestAppSocketPathEmptyWithoutBackendSocket checks that no app socket path is derived from
// an empty backend socket, so nothing is created in the working directory.
func TestAppSocketPathEmptyWithoutBackendSocket(t *testing.T) {
	app := &ServerApp{config: &Config{}}
	if got := app.knirvServerAppSocketPath(); got != "" {
		t.Fatalf("knirvServerAppSocketPath() = %q, want empty", got)
	}
}

func waitForSocket(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if fi, err := os.Stat(path); err == nil && fi.Mode()&os.ModeSocket != 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("socket %s never appeared", path)
}
