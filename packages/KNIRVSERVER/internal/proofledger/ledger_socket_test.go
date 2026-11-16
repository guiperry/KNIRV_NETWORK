package proofledger

import (
	"context"
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"knirv-server/internal/dveevidence"
	"knirv-server/internal/knirvproof"
)

const testToken = "test-token-not-a-secret"

// TestGitPushLandsThroughGitSocketHandler runs a real `git push` against the proof
// ledger's /git/* handler. The handler is served on a Unix socket, the same as
// production. Only a test-local loopback bridge in front of that socket is used,
// because the git CLI cannot dial a Unix socket directly. Everything lives in temp
// dirs with a fake token.
func TestGitPushLandsThroughGitSocketHandler(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	// Short temp dirs: Unix socket paths are limited to ~108 bytes.
	base, err := os.MkdirTemp("", "plg")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	root := filepath.Join(base, "ledger")
	home := filepath.Join(base, "home")
	work := filepath.Join(base, "work")
	for _, d := range []string{home, work} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	authorize := knirvproof.AuthorizerFunc(func(_ context.Context, token, _, _ string) (*knirvproof.Principal, error) {
		if token != testToken {
			return nil, knirvproof.ErrUnauthorized
		}
		return &knirvproof.Principal{ID: "test-principal"}, nil
	})
	ledger, err := New(root, &dveevidence.IngestService{}, authorize)
	if err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	RegisterRoutes(engine, ledger)

	// Serve the handler on a Unix socket, as KNIRVSERVER does for /git/*.
	sock := filepath.Join(base, "git.sock")
	unixLn, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	upstream := &http.Server{Handler: engine}
	go func() { _ = upstream.Serve(unixLn) }()
	t.Cleanup(func() { _ = upstream.Close() })

	// Test-only loopback bridge to the socket so the git CLI can speak smart HTTP.
	target, _ := url.Parse("http://git")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}
	bridge := httptest.NewServer(proxy)
	t.Cleanup(bridge.Close)

	gitEnv := append(os.Environ(), "HOME="+home, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+filepath.Join(home, "gitconfig"), "GIT_TERMINAL_PROMPT=0")
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = gitEnv
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}

	run(work, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(work, "f.txt"), []byte("hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(work, "add", "f.txt")
	run(work, "-c", "user.email=test@example.invalid", "-c", "user.name=test", "commit", "-q", "-m", "init")
	head := run(work, "rev-parse", "HEAD")

	const projectID = "proj-socket-test"
	remote := bridge.URL + "/git/" + projectID + ".git"
	auth := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("tester:"+testToken))

	// Without credentials the handler must refuse, so the push must fail.
	unauth := exec.Command("git", "-C", work, "push", "-q", remote, "HEAD:refs/heads/main")
	unauth.Env = gitEnv
	if out, err := unauth.CombinedOutput(); err == nil {
		t.Fatalf("unauthenticated push succeeded; ledger accepted no credentials:\n%s", out)
	}

	// Authenticated push must land.
	run(work, "-c", "http.extraHeader="+auth, "push", "-q", remote, "HEAD:refs/heads/main")

	bare := filepath.Join(root, projectID+".git")
	got := runIn(t, gitEnv, bare, "rev-parse", "refs/heads/main")
	if got != head {
		t.Fatalf("ledger refs/heads/main = %s, want pushed HEAD %s", got, head)
	}
}

func runIn(t *testing.T, env []string, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}
