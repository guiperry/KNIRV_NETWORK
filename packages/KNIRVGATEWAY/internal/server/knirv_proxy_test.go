package server

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// startUnixUpstream serves handler on a temp Unix socket and returns its path.
// Temp dirs only; nothing here touches a real secret or key.
func startUnixUpstream(t *testing.T, handler http.Handler) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "knirvsock")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: handler}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return sock
}

// TestKnirvRoutesForwardToProofSocketUnchanged checks that /api/v1/knirv/* reaches the
// proof socket with method, path and body unchanged.
func TestKnirvRoutesForwardToProofSocketUnchanged(t *testing.T) {
	var calls int
	var gotMethod, gotPath, gotBody string
	sock := startUnixUpstream(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	router := mux.NewRouter()
	registerKnirvSocketRoutes(router, sock)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/knirv/projects/p1/proofs", strings.NewReader(`{"x":1}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || rec.Body.String() != "ok" {
		t.Fatalf("POST knirv = %d %q, want 201 ok", rec.Code, rec.Body.String())
	}
	if calls != 1 || gotMethod != http.MethodPost || gotPath != "/api/v1/knirv/projects/p1/proofs" || gotBody != `{"x":1}` {
		t.Fatalf("proof upstream saw calls=%d method=%s path=%s body=%s", calls, gotMethod, gotPath, gotBody)
	}
}

// TestGitRoutesAreNotPublic checks that /git/* is internal only: the gateway answers 404
// on its public router, never proxies to the git socket and never serves the SPA.
func TestGitRoutesAreNotPublic(t *testing.T) {
	var upstreamCalls, backendCalls int
	sock := startUnixUpstream(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		w.WriteHeader(http.StatusOK)
	}))

	router := mux.NewRouter()
	registerKnirvSocketRoutes(router, sock)
	registerGitPublicNotFound(router)
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendCalls++
		w.WriteHeader(http.StatusOK)
	})

	for _, req := range []struct{ method, path string }{
		{http.MethodGet, "/git/proj.git/info/refs?service=git-receive-pack"},
		{http.MethodPost, "/git/proj.git/git-receive-pack"},
		{http.MethodPost, "/git/proj.git/create"},
		{http.MethodGet, "/git/"},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(req.method, req.path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s %s on public gateway = %d, want 404", req.method, req.path, rec.Code)
		}
	}
	if upstreamCalls != 0 || backendCalls != 0 {
		t.Fatalf("git socket calls=%d catch-all calls=%d, want 0 and 0", upstreamCalls, backendCalls)
	}
}

// TestKnirvAndGitPrefixesNeverFallThroughWhenUnset checks that with no sockets
// configured, /api/v1/knirv/* and /git/* return 404 and never reach a generic
// /api/v1/ backend catch-all.
func TestKnirvAndGitPrefixesNeverFallThroughWhenUnset(t *testing.T) {
	// Production order: the knirv/git prefixes are registered before the generic
	// /api/v1/ backend catch-all (see server.go), so the catch-all must not win.
	var backendCalls int
	router := mux.NewRouter()
	registerKnirvSocketRoutes(router, "")
	registerGitPublicNotFound(router)
	router.PathPrefix("/api/v1/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendCalls++
		w.WriteHeader(http.StatusOK)
	})

	for _, path := range []string{"/api/v1/knirv/validator-key", "/api/v1/knirv/projects/p/proofs", "/git/p.git/info/refs"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("GET %s with no socket = %d, want 404", path, rec.Code)
		}
	}
	if backendCalls != 0 {
		t.Fatalf("backend catch-all reached %d times, want 0", backendCalls)
	}
}

// TestKnirvRouteBadGatewayWhenSocketDown checks that a configured route fails closed.
func TestKnirvRouteBadGatewayWhenSocketDown(t *testing.T) {
	dir, err := os.MkdirTemp("", "knirvdown")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	router := mux.NewRouter()
	registerKnirvSocketRoutes(router, filepath.Join(dir, "absent.sock"))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/knirv/validator-key", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("knirv with socket down = %d, want 502", rec.Code)
	}
}
