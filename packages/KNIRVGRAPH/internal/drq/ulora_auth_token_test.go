package drq

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// isolateULoRAAuthEnv clears every variable the token resolver reads, so a test
// never picks up a developer's real token or app-data directory.
//
// The launcher's system app data dir is pointed at an unusable path (a regular
// file blocks it), and HOME and XDG_DATA_HOME point into a temp dir, so the
// fallback chain never reads or creates a real /var/lib or home directory. It
// returns the XDG_DATA_HOME used, the last-resort location for those tests.
func isolateULoRAAuthEnv(t *testing.T) (xdg string) {
	t.Helper()
	t.Setenv(uloraAuthTokenEnv, "")
	t.Setenv(uloraAuthTokenFileEnv, "")
	t.Setenv(uloraDataDirEnv, "")
	t.Setenv(knirvAppDataDirEnv, "")
	root := t.TempDir()
	blocker := filepath.Join(root, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	origSystem := defaultSystemAppDataDir
	defaultSystemAppDataDir = filepath.Join(blocker, "knirvserver")
	t.Cleanup(func() { defaultSystemAppDataDir = origSystem })
	t.Setenv("HOME", filepath.Join(root, "home"))
	xdg = filepath.Join(root, "xdg")
	t.Setenv("XDG_DATA_HOME", xdg)
	return xdg
}

// writeTokenFile writes a token file with the given mode in dir, creating dir.
func writeTokenFile(t *testing.T, path, token string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(token+"\n"), mode); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod token file: %v", err)
	}
}

func TestResolveULoRAAuthTokenExplicitEnvWins(t *testing.T) {
	isolateULoRAAuthEnv(t)
	dir := t.TempDir()
	writeTokenFile(t, filepath.Join(dir, "handoff"), "file-token-test-only", 0o600)
	writeTokenFile(t, filepath.Join(dir, "ulora", "auth_token"), "other-token-test-only", 0o600)
	t.Setenv(uloraAuthTokenEnv, fakeULoRABearer)
	t.Setenv(uloraAuthTokenFileEnv, filepath.Join(dir, "handoff"))
	t.Setenv(knirvAppDataDirEnv, dir)

	got, err := resolveULoRAAuthToken()
	if err != nil || got != fakeULoRABearer {
		t.Fatalf("explicit env must win: got %q, err %v", got, err)
	}
}

func TestResolveULoRAAuthTokenFromHandoffFileAndRotation(t *testing.T) {
	isolateULoRAAuthEnv(t)
	path := filepath.Join(t.TempDir(), "handoff")
	writeTokenFile(t, path, fakeULoRABearer, 0o600)
	t.Setenv(uloraAuthTokenFileEnv, path)

	got, err := resolveULoRAAuthToken()
	if err != nil || got != fakeULoRABearer {
		t.Fatalf("handoff file: got %q, err %v", got, err)
	}

	// Rotation: the file is re-read, so a rewritten token is picked up with no
	// restart.
	writeTokenFile(t, path, "rotated-token-test-only", 0o600)
	got, err = resolveULoRAAuthToken()
	if err != nil || got != "rotated-token-test-only" {
		t.Fatalf("rotated handoff file: got %q, err %v", got, err)
	}
}

func TestResolveULoRAAuthTokenFromULoRADataDir(t *testing.T) {
	isolateULoRAAuthEnv(t)
	dir := t.TempDir()
	writeTokenFile(t, filepath.Join(dir, "auth_token"), fakeULoRABearer, 0o600)
	t.Setenv(uloraDataDirEnv, dir)

	got, err := resolveULoRAAuthToken()
	if err != nil || got != fakeULoRABearer {
		t.Fatalf("ULORA_DATA_DIR token file: got %q, err %v", got, err)
	}
}

func TestResolveULoRAAuthTokenFromAppDataDirUlora(t *testing.T) {
	isolateULoRAAuthEnv(t)
	app := t.TempDir()
	writeTokenFile(t, filepath.Join(app, "ulora", "auth_token"), fakeULoRABearer, 0o600)
	t.Setenv(knirvAppDataDirEnv, app)

	got, err := resolveULoRAAuthToken()
	if err != nil || got != fakeULoRABearer {
		t.Fatalf("KNIRV_APP_DATA_DIR/ulora token file: got %q, err %v", got, err)
	}
}

func TestResolveULoRAAuthTokenDataDirBeatsAppDataDir(t *testing.T) {
	isolateULoRAAuthEnv(t)
	app := t.TempDir()
	data := t.TempDir()
	writeTokenFile(t, filepath.Join(app, "ulora", "auth_token"), "app-token-test-only", 0o600)
	writeTokenFile(t, filepath.Join(data, "auth_token"), fakeULoRABearer, 0o600)
	t.Setenv(knirvAppDataDirEnv, app)
	t.Setenv(uloraDataDirEnv, data)

	got, err := resolveULoRAAuthToken()
	if err != nil || got != fakeULoRABearer {
		t.Fatalf("ULORA_DATA_DIR must take precedence: got %q, err %v", got, err)
	}
}

// With no ULORA_DATA_DIR and no KNIRV_APP_DATA_DIR, the reader uses the launcher's
// fallback chain (here XDG, because the system dir is unusable), the same
// directory ulorad writes to.
func TestResolveULoRAAuthTokenFromLauncherFallbackDir(t *testing.T) {
	xdg := isolateULoRAAuthEnv(t)
	writeTokenFile(t, filepath.Join(xdg, "knirvserver", "ulora", "auth_token"), fakeULoRABearer, 0o600)

	got, err := resolveULoRAAuthToken()
	if err != nil || got != fakeULoRABearer {
		t.Fatalf("launcher fallback dir: got %q, err %v", got, err)
	}
}

func TestResolveULoRAAuthTokenUnavailableWithoutAnySource(t *testing.T) {
	isolateULoRAAuthEnv(t)
	if _, err := resolveULoRAAuthToken(); !errors.Is(err, ErrULoRAAuthTokenUnavailable) {
		t.Fatalf("no source: want ErrULoRAAuthTokenUnavailable, got %v", err)
	}

	// Directories configured, but ulorad has not written a token there yet.
	t.Setenv(uloraDataDirEnv, t.TempDir())
	if _, err := resolveULoRAAuthToken(); !errors.Is(err, ErrULoRAAuthTokenUnavailable) {
		t.Fatalf("dir without token file: want ErrULoRAAuthTokenUnavailable, got %v", err)
	}
}

// A source that is present but unusable must fail closed with a distinct error,
// never fall through to a weaker source, and never reveal the token.
func TestResolveULoRAAuthTokenRefusesUnsafeFiles(t *testing.T) {
	cases := map[string]struct {
		content string
		mode    os.FileMode
	}{
		"world readable": {fakeULoRABearer, 0o644},
		"group writable": {fakeULoRABearer, 0o620},
		"other readable": {fakeULoRABearer, 0o604},
		"world writable": {fakeULoRABearer, 0o666},
		"empty file":     {"   ", 0o600},
		"multi-line":     {"part-one\npart-two", 0o600},
		"embedded space": {"part one", 0o600},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			isolateULoRAAuthEnv(t)
			path := filepath.Join(t.TempDir(), "auth_token")
			writeTokenFile(t, path, tc.content, tc.mode)
			t.Setenv(uloraDataDirEnv, filepath.Dir(path))

			got, err := resolveULoRAAuthToken()
			if err == nil {
				t.Fatalf("unsafe token file must be refused, got token of length %d", len(got))
			}
			if errors.Is(err, ErrULoRAAuthTokenUnavailable) {
				t.Fatalf("a present-but-unusable file must not report 'absent': %v", err)
			}
			if strings.Contains(err.Error(), fakeULoRABearer) || strings.Contains(err.Error(), "part-one") {
				t.Fatalf("error must not contain the token value: %v", err)
			}
		})
	}
}

// The client, end to end: no explicit token and no environment token. The token
// comes from ulorad's file, the fake ulorad enforces it, and a rotated file is
// used on the next compile.
func TestKNIRVULORAClientResolvesTokenFromFileAndRotates(t *testing.T) {
	isolateULoRAAuthEnv(t)
	dataDir := t.TempDir()
	tokenPath := filepath.Join(dataDir, "auth_token")
	writeTokenFile(t, tokenPath, fakeULoRABearer, 0o600)
	t.Setenv(uloraDataDirEnv, dataDir)

	var accepted atomic.Int32
	socketPath := newUnixServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if uloraAuthRejects(w, r, fakeULoRABearer) {
			return
		}
		accepted.Add(1)
		writeULoRACompileOK(w)
	}))
	client := NewKNIRVULORAClient(socketPath)

	if _, err := compileOnceForTest(client); err != nil {
		t.Fatalf("compile with file-resolved token: %v", err)
	}

	// Rotate: the old token is now wrong, so ulorad must reject a stale copy.
	writeTokenFile(t, tokenPath, "rotated-token-test-only", 0o600)
	if _, err := compileOnceForTest(client); err == nil {
		t.Fatal("the rotated token is not the one the fake ulorad accepts; the compile must fail")
	}
	if accepted.Load() != 1 {
		t.Fatalf("accepted requests = %d, want 1", accepted.Load())
	}
}

// Unsafe token file: the client refuses before dialling, so nothing reaches ulorad.
func TestKNIRVULORAClientRefusesUnsafeTokenFileBeforeDialling(t *testing.T) {
	isolateULoRAAuthEnv(t)
	dataDir := t.TempDir()
	writeTokenFile(t, filepath.Join(dataDir, "auth_token"), fakeULoRABearer, 0o644)
	t.Setenv(uloraDataDirEnv, dataDir)

	var hits atomic.Int32
	socketPath := newUnixServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		writeULoRACompileOK(w)
	}))
	_, err := compileOnceForTest(NewKNIRVULORAClient(socketPath))
	if err == nil {
		t.Fatal("a world-readable token file must not be used")
	}
	if hits.Load() != 0 {
		t.Fatalf("an unsafe token file must not lead to a request, reached %d times", hits.Load())
	}
}

// A variable that is set but blank is unusable, so it must fail closed rather than
// fall through to a weaker source (here, a valid token file).
func TestResolveULoRAAuthTokenBlankEnvFailsClosed(t *testing.T) {
	cases := map[string]string{
		"blank token env":      uloraAuthTokenEnv,
		"blank token file env": uloraAuthTokenFileEnv,
	}
	for name, envVar := range cases {
		t.Run(name, func(t *testing.T) {
			isolateULoRAAuthEnv(t)
			dir := t.TempDir()
			writeTokenFile(t, filepath.Join(dir, "ulora", "auth_token"), "file-token-test-only", 0o600)
			t.Setenv(knirvAppDataDirEnv, dir)
			t.Setenv(envVar, "   ")

			got, err := resolveULoRAAuthToken()
			if err == nil {
				t.Fatalf("a blank %s must fail closed, resolved a token of length %d", envVar, len(got))
			}
			if got != "" {
				t.Fatal("no token may be returned on failure")
			}
		})
	}
}

// 0640 is an accepted handoff mode (owner rw, group r); the token resolves.
func TestResolveULoRAAuthTokenAcceptsGroupReadableHandoff(t *testing.T) {
	isolateULoRAAuthEnv(t)
	path := filepath.Join(t.TempDir(), "handoff")
	writeTokenFile(t, path, fakeULoRABearer, 0o640)
	t.Setenv(uloraAuthTokenFileEnv, path)

	got, err := resolveULoRAAuthToken()
	if err != nil {
		t.Fatalf("0640 handoff must be accepted: %v", err)
	}
	if got != fakeULoRABearer {
		t.Fatal("0640 handoff returned the wrong token")
	}
}
