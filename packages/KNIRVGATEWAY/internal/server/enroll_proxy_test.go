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

// TestEnrollRouteForwardsPostToSocketUnchanged runs a real HTTP server on a Unix socket
// as the enrollment upstream and checks that the gateway forwards POST /enroll/{step}
// there with the path and body unchanged, and does not forward other methods.
func TestEnrollRouteForwardsPostToSocketUnchanged(t *testing.T) {
	dir, err := os.MkdirTemp("", "enrp")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "e.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	var calls int
	var gotMethod, gotPath, gotBody string
	upstream := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})}
	go func() { _ = upstream.Serve(ln) }()
	t.Cleanup(func() { _ = upstream.Close() })

	router := mux.NewRouter()
	registerEnrollRoute(router, sock)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/enroll/challenge", strings.NewReader(`{"x":1}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || rec.Body.String() != "ok" {
		t.Fatalf("POST /enroll/challenge = %d %q, want 201 ok", rec.Code, rec.Body.String())
	}
	if calls != 1 || gotMethod != http.MethodPost || gotPath != "/enroll/challenge" || gotBody != `{"x":1}` {
		t.Fatalf("upstream saw calls=%d method=%s path=%s body=%s", calls, gotMethod, gotPath, gotBody)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/enroll/challenge", nil))
	if calls != 1 {
		t.Fatalf("GET must not reach the enrollment upstream, calls=%d code=%d", calls, rec.Code)
	}
}

// TestEnrollRouteAbsentWithoutSocketPath checks that no enrollment route exists when
// the socket is not configured.
func TestEnrollRouteAbsentWithoutSocketPath(t *testing.T) {
	router := mux.NewRouter()
	registerEnrollRoute(router, "")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/enroll/challenge", strings.NewReader("{}")))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unconfigured enrollment must be 404, got %d", rec.Code)
	}
}

// TestEnrollRouteBadGatewayWhenSocketDown checks that a configured route fails closed
// with 502 while the enrollment socket is not listening.
func TestEnrollRouteBadGatewayWhenSocketDown(t *testing.T) {
	dir, err := os.MkdirTemp("", "enrd")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	router := mux.NewRouter()
	registerEnrollRoute(router, filepath.Join(dir, "absent.sock"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/enroll/challenge", strings.NewReader("{}")))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("enrollment with socket down = %d, want 502", rec.Code)
	}
}
