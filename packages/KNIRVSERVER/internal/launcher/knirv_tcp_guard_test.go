package launcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestGitTCPRouteAnswers404NotSPA checks that /git/* on the TCP router is a 404 and
// never falls through to the SPA NoRoute (which would serve HTML).
func TestGitTCPRouteAnswers404NotSPA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerGitTCPNotFound(r)
	r.NoRoute(func(c *gin.Context) { c.String(http.StatusOK, "<html>spa</html>") })

	for _, p := range []string{"/git/proj.git/info/refs", "/git/proj/info/refs", "/git/"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404", p, w.Code)
		}
		if strings.Contains(w.Body.String(), "spa") {
			t.Fatalf("GET %s served SPA body", p)
		}
	}
}

// TestEmptyBackendSocketCreatesNoRouteSockets checks that with no backend socket
// configured, the route socket paths are empty and no socket is created in the
// working directory.
func TestEmptyBackendSocketCreatesNoRouteSockets(t *testing.T) {
	app := &ServerApp{config: &Config{BackendSocket: ""}, proofRoutes: gin.New(), proofLedgerRoutes: gin.New()}
	if p := app.knirvProofSocketPath(); p != "" {
		t.Fatalf("proof socket path = %q, want empty", p)
	}
	if p := app.gitSocketPath(); p != "" {
		t.Fatalf("git socket path = %q, want empty", p)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.startKnirvRouteSockets(ctx)
	for _, name := range []string{"knirvproof.sock", "git.sock"} {
		if _, err := os.Lstat(name); err == nil {
			t.Fatalf("%s was created in the working directory", name)
		}
	}
}

// TestStaleRegularFileAtSocketPathIsNotRemoved checks that a non-socket file at the
// socket path is left alone (listen fails) rather than deleted.
func TestStaleRegularFileAtSocketPathIsNotRemoved(t *testing.T) {
	dir := t.TempDir()
	app := &ServerApp{config: &Config{BackendSocket: filepath.Join(dir, "backend.sock")}, proofRoutes: gin.New(), proofLedgerRoutes: gin.New()}
	victim := app.gitSocketPath()
	if err := os.WriteFile(victim, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.startKnirvRouteSockets(ctx)
	time.Sleep(200 * time.Millisecond)
	b, err := os.ReadFile(victim)
	if err != nil || string(b) != "keep me" {
		t.Fatalf("regular file at socket path was removed or changed: err=%v body=%q", err, b)
	}
}
