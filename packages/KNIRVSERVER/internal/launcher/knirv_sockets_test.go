package launcher

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestKnirvRouteSocketsServeOnlyOnUnixSockets starts the proof and git listeners in a
// temp dir and checks that each engine answers on its Unix socket with mode 0600. It
// also checks that the socket paths sit beside the backend socket, as the gateway
// expects. Nothing binds TCP and no real secret or key is read.
func TestKnirvRouteSocketsServeOnlyOnUnixSockets(t *testing.T) {
	dir, err := os.MkdirTemp("", "knsock")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	proof := gin.New()
	proof.GET("/api/v1/knirv/validator-key", func(c *gin.Context) { c.String(http.StatusOK, "proof") })
	git := gin.New()
	git.GET("/git/:project_id/info/refs", func(c *gin.Context) { c.String(http.StatusOK, "git") })

	app := &ServerApp{
		config:            &Config{BackendSocket: filepath.Join(dir, "backend.sock")},
		proofRoutes:       proof,
		proofLedgerRoutes: git,
	}
	if got, want := app.knirvProofSocketPath(), filepath.Join(dir, "knirvproof.sock"); got != want {
		t.Fatalf("proof socket = %s, want %s", got, want)
	}
	if got, want := app.gitSocketPath(), filepath.Join(dir, "git.sock"); got != want {
		t.Fatalf("git socket = %s, want %s", got, want)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go app.startKnirvRouteSockets(ctx)

	client := func(sock string) *http.Client {
		return &http.Client{Transport: &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
			return net.Dial("unix", sock)
		}}, Timeout: 5 * time.Second}
	}
	get := func(sock, path string) (int, string) {
		t.Helper()
		var lastErr error
		for i := 0; i < 50; i++ {
			resp, err := client(sock).Get("http://knirv" + path)
			if err == nil {
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				return resp.StatusCode, string(b)
			}
			lastErr = err
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("socket %s never answered: %v", sock, lastErr)
		return 0, ""
	}

	if code, body := get(app.knirvProofSocketPath(), "/api/v1/knirv/validator-key"); code != http.StatusOK || body != "proof" {
		t.Fatalf("proof socket = %d %q", code, body)
	}
	if code, body := get(app.gitSocketPath(), "/git/proj/info/refs"); code != http.StatusOK || body != "git" {
		t.Fatalf("git socket = %d %q", code, body)
	}
	for _, sock := range []string{app.knirvProofSocketPath(), app.gitSocketPath()} {
		info, err := os.Stat(sock)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("%s mode = %o, want 600", sock, perm)
		}
	}
}
