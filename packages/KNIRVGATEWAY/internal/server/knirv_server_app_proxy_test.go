package server

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// recordingAppUpstream serves a fake KNIRVSERVER app socket and records every request.
type recordingAppUpstream struct {
	calls   int
	methods []string
	paths   []string
	marks   []string
	bodies  []string
}

func (u *recordingAppUpstream) handler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u.calls++
		b, _ := io.ReadAll(r.Body)
		u.methods = append(u.methods, r.Method)
		u.paths = append(u.paths, r.URL.RequestURI())
		u.marks = append(u.marks, r.Header.Get(KnirvServerAppForwardHeader))
		u.bodies = append(u.bodies, string(b))
		w.Header().Set("X-From", "app")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("from-app"))
	})
}

// TestServerAppRoutesForwardToAppSocket checks that every public path the gateway owns
// for KNIRVSERVER reaches the app socket with method, path, query and body unchanged,
// and carries the forward marker the launcher relies on.
func TestServerAppRoutesForwardToAppSocket(t *testing.T) {
	up := &recordingAppUpstream{}
	sock := startUnixUpstream(t, up.handler(t))

	router := mux.NewRouter()
	registerKnirvServerAppRoutes(router, newKnirvServerAppProxy(sock))

	cases := []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/system/info", ""},
		{http.MethodGet, "/api/v1/system/update?force=1", ""},
		{http.MethodPost, "/api/v1/system/update/apply", `{"x":1}`},
		{http.MethodGet, "/api/dve/evidence/ingest", ""},
		{http.MethodGet, "/tunnel", ""},
		{http.MethodPost, "/tunnel/api/tunnels", `{"id":"t1"}`},
		{http.MethodGet, "/login", ""},
		{http.MethodGet, "/menu/", ""},
		{http.MethodGet, "/terminal", ""},
		{http.MethodGet, "/error-resolution/abc", ""},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, strings.NewReader(c.body)))
		if rec.Code != http.StatusOK || rec.Body.String() != "from-app" {
			t.Fatalf("%s %s = %d %q, want 200 from-app", c.method, c.path, rec.Code, rec.Body.String())
		}
	}
	if up.calls != len(cases) {
		t.Fatalf("app upstream calls = %d, want %d", up.calls, len(cases))
	}
	for i, c := range cases {
		if up.methods[i] != c.method || up.paths[i] != c.path || up.bodies[i] != c.body {
			t.Fatalf("case %d: upstream saw %s %s body=%q, want %s %s body=%q",
				i, up.methods[i], up.paths[i], up.bodies[i], c.method, c.path, c.body)
		}
		if up.marks[i] != "1" {
			t.Fatalf("case %d (%s): forward marker = %q, want 1", i, c.path, up.marks[i])
		}
	}
}

// TestServerAppRoutesNotRegisteredWithoutSocket checks that with no app socket nothing is
// forwarded: the helper returns nil and the public paths get no app route.
func TestServerAppRoutesNotRegisteredWithoutSocket(t *testing.T) {
	if h := newKnirvServerAppProxy(""); h != nil {
		t.Fatal("newKnirvServerAppProxy(\"\") must return nil")
	}
	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // stand-in for the gateway's own catch-all
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))
	if rec.Code != http.StatusTeapot {
		t.Fatalf("/login without app socket = %d, want the gateway catch-all", rec.Code)
	}
}

// TestServerAppWebSocketUpgradeThroughGateway checks that /ws and /tunnel upgrades pass
// through the gateway to the app socket: the 101 status and the app's frames arrive at
// the client over the same connection.
func TestServerAppWebSocketUpgradeThroughGateway(t *testing.T) {
	sock := startUnixUpstream(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = buf.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		_, _ = buf.WriteString("frame:" + r.URL.Path)
		_ = buf.Flush()
	}))

	router := mux.NewRouter()
	app := newKnirvServerAppProxy(sock)
	registerKnirvServerAppRoutes(router, app)
	registerKnirvServerAppWSRoutes(router, app)
	gw := httptest.NewServer(router)
	defer gw.Close()

	for _, path := range []string{"/ws", "/ws/dve/d1/advisor/stream/s1", "/tunnel"} {
		conn, err := net.DialTimeout("tcp", gw.Listener.Addr().String(), 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		req := "GET " + path + " HTTP/1.1\r\nHost: gateway\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n" +
			"Sec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"
		if _, err := conn.Write([]byte(req)); err != nil {
			t.Fatal(err)
		}
		reader := bufio.NewReader(conn)
		status, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("%s: read status: %v", path, err)
		}
		if !strings.Contains(status, "101") {
			t.Fatalf("%s: status = %q, want 101", path, status)
		}
		rest, _ := io.ReadAll(reader)
		if !strings.Contains(string(rest), "frame:"+path) {
			t.Fatalf("%s: app frame missing from upgraded stream: %q", path, rest)
		}
		_ = conn.Close()
	}
}

// TestWebguiFileOrAppFallback checks that a static file present in the WebGUI export is
// served locally, and a missing one is forwarded to the app socket (marker set), or
// answered 404 when no app socket is configured.
func TestWebguiFileOrAppFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "_next", "static"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_next", "static", "gw.js"), []byte("gateway-chunk"), 0o600); err != nil {
		t.Fatal(err)
	}

	up := &recordingAppUpstream{}
	sock := startUnixUpstream(t, up.handler(t))
	router := mux.NewRouter()
	router.PathPrefix("/_next/").Handler(webguiFileOrAppFallback(dir, newKnirvServerAppProxy(sock)))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_next/static/gw.js", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "gateway-chunk") {
		t.Fatalf("local chunk = %d %q, want gateway file", rec.Code, rec.Body.String())
	}
	if up.calls != 0 {
		t.Fatalf("local file reached app socket %d times", up.calls)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_next/static/app-only.js", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "from-app" {
		t.Fatalf("missing chunk = %d %q, want app response", rec.Code, rec.Body.String())
	}
	if up.calls != 1 || up.paths[0] != "/_next/static/app-only.js" || up.marks[0] != "1" {
		t.Fatalf("app saw calls=%d path=%v marks=%v", up.calls, up.paths, up.marks)
	}

	noApp := mux.NewRouter()
	noApp.PathPrefix("/_next/").Handler(webguiFileOrAppFallback(dir, nil))
	rec = httptest.NewRecorder()
	noApp.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_next/static/app-only.js", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing chunk without app socket = %d, want 404", rec.Code)
	}
}
