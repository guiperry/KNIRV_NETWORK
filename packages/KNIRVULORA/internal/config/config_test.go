package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// isolateAppDataDir points every input of the default data-dir chain at temp
// locations, so a test never reads or creates a real /var/lib or home directory.
// The system dir is made unusable (a regular file blocks it), so resolution falls
// through to XDG_DATA_HOME.
func isolateAppDataDir(t *testing.T) (xdg string) {
	t.Helper()
	root := t.TempDir()
	blocker := filepath.Join(root, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	origSystem := defaultSystemAppDataDir
	defaultSystemAppDataDir = filepath.Join(blocker, "knirvserver")
	t.Cleanup(func() { defaultSystemAppDataDir = origSystem })
	t.Setenv("ULORA_DATA_DIR", "")
	t.Setenv("KNIRV_APP_DATA_DIR", "")
	t.Setenv("HOME", filepath.Join(root, "home"))
	xdg = filepath.Join(root, "xdg")
	t.Setenv("XDG_DATA_HOME", xdg)
	return xdg
}

func TestLoadDefaults(t *testing.T) {
	xdg := isolateAppDataDir(t)
	t.Setenv("ULORA_SOCKET_PATH", "")
	t.Setenv("ULORA_AUTH_TOKEN", "test-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.SocketPath != defaultSocketPath {
		t.Fatalf("expected default socket path %s, got %s", defaultSocketPath, cfg.SocketPath)
	}
	if cfg.AuthToken != "test-token" {
		t.Fatalf("expected auth token, got %s", cfg.AuthToken)
	}
	// Neither ULORA_DATA_DIR nor KNIRV_APP_DATA_DIR is set, so the launcher's
	// fallback chain decides: XDG here, because the system dir is unusable.
	if want := filepath.Join(xdg, "knirvserver", "ulora"); cfg.DataDir != want {
		t.Fatalf("expected default data dir %s, got %s", want, cfg.DataDir)
	}
	if cfg.PythonBin != "python3" {
		t.Fatalf("expected default python bin python3, got %s", cfg.PythonBin)
	}
	if cfg.RequestTimeout != defaultTimeout {
		t.Fatalf("expected default timeout %v, got %v", defaultTimeout, cfg.RequestTimeout)
	}
}

// With no env, a writable system app data dir wins over XDG, as in the launcher.
func TestLoadDataDirPrefersWritableSystemAppDataDir(t *testing.T) {
	xdg := isolateAppDataDir(t)
	system := filepath.Join(t.TempDir(), "knirvserver")
	defaultSystemAppDataDir = system
	t.Setenv("ULORA_AUTH_TOKEN", "test-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if want := filepath.Join(system, "ulora"); cfg.DataDir != want {
		t.Fatalf("DataDir = %q, want writable system dir %q (XDG was %q)", cfg.DataDir, want, xdg)
	}
}

// With no XDG_DATA_HOME either, the last launcher fallback is ~/.local/share.
func TestLoadDataDirFallsBackToHomeLocalShare(t *testing.T) {
	isolateAppDataDir(t)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("ULORA_AUTH_TOKEN", "test-token")
	home := os.Getenv("HOME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if want := filepath.Join(home, ".local", "share", "knirvserver", "ulora"); cfg.DataDir != want {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, want)
	}
}

func TestLoadCustomValues(t *testing.T) {
	tmpDir := t.TempDir()
	customSocket := tmpDir + "/custom.sock"
	customAuth := "my-secret-token"
	customData := tmpDir + "/data"
	customPython := "/usr/bin/python3"
	customVenv := tmpDir + "/venv"
	customTimeout := "60s"

	origSocket := os.Getenv("ULORA_SOCKET_PATH")
	origAuth := os.Getenv("ULORA_AUTH_TOKEN")
	origData := os.Getenv("ULORA_DATA_DIR")
	origPython := os.Getenv("ULORA_PYTHON_BIN")
	origVenv := os.Getenv("ULORA_VENV_DIR")
	origTimeout := os.Getenv("ULORA_REQUEST_TIMEOUT")
	defer func() {
		os.Setenv("ULORA_SOCKET_PATH", origSocket)
		os.Setenv("ULORA_AUTH_TOKEN", origAuth)
		os.Setenv("ULORA_DATA_DIR", origData)
		os.Setenv("ULORA_PYTHON_BIN", origPython)
		os.Setenv("ULORA_VENV_DIR", origVenv)
		os.Setenv("ULORA_REQUEST_TIMEOUT", origTimeout)
	}()

	os.Setenv("ULORA_SOCKET_PATH", customSocket)
	os.Setenv("ULORA_AUTH_TOKEN", customAuth)
	os.Setenv("ULORA_DATA_DIR", customData)
	os.Setenv("ULORA_PYTHON_BIN", customPython)
	os.Setenv("ULORA_VENV_DIR", customVenv)
	os.Setenv("ULORA_REQUEST_TIMEOUT", customTimeout)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.SocketPath != customSocket {
		t.Fatalf("expected socket path %s, got %s", customSocket, cfg.SocketPath)
	}
	if cfg.AuthToken != customAuth {
		t.Fatalf("expected auth token %s, got %s", customAuth, cfg.AuthToken)
	}
	if cfg.DataDir != customData {
		t.Fatalf("expected data dir %s, got %s", customData, cfg.DataDir)
	}
	if cfg.PythonBin != customPython {
		t.Fatalf("expected python bin %s, got %s", customPython, cfg.PythonBin)
	}
	if cfg.VenvDir != customVenv {
		t.Fatalf("expected venv dir %s, got %s", customVenv, cfg.VenvDir)
	}
	expectedTimeout, _ := time.ParseDuration(customTimeout)
	if cfg.RequestTimeout != expectedTimeout {
		t.Fatalf("expected timeout %v, got %v", expectedTimeout, cfg.RequestTimeout)
	}
}

// With ULORA_AUTH_TOKEN unset, Load generates a token into the data dir and a
// second start reuses it.
func TestLoadGeneratesThenReusesAuthTokenWhenEnvUnset(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("ULORA_AUTH_TOKEN", "")
	t.Setenv("ULORA_DATA_DIR", dataDir)
	t.Setenv("KNIRV_APP_DATA_DIR", "")

	first, err := Load()
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if len(first.AuthToken) != 2*generatedTokenBytes {
		t.Fatalf("generated token length = %d, want %d", len(first.AuthToken), 2*generatedTokenBytes)
	}
	onDisk, err := os.ReadFile(AuthTokenPath(dataDir))
	if err != nil {
		t.Fatalf("token file not written: %v", err)
	}
	if strings.TrimSpace(string(onDisk)) != first.AuthToken {
		t.Fatal("token file does not hold the generated token")
	}

	second, err := Load()
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if second.AuthToken != first.AuthToken {
		t.Fatal("restart did not reuse the persisted token")
	}
}

func TestLoadDataDirFromAppDataDir(t *testing.T) {
	app := t.TempDir()
	t.Setenv("ULORA_DATA_DIR", "")
	t.Setenv("KNIRV_APP_DATA_DIR", app)
	t.Setenv("ULORA_AUTH_TOKEN", "test-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if want := filepath.Join(app, "ulora"); cfg.DataDir != want {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, want)
	}
}

func TestLoadExplicitDataDirBeatsAppDataDir(t *testing.T) {
	app := t.TempDir()
	explicit := t.TempDir()
	t.Setenv("ULORA_DATA_DIR", explicit)
	t.Setenv("KNIRV_APP_DATA_DIR", app)
	t.Setenv("ULORA_AUTH_TOKEN", "test-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.DataDir != explicit {
		t.Fatalf("DataDir = %q, want explicit %q", cfg.DataDir, explicit)
	}
}

func TestLoadInvalidTimeout(t *testing.T) {
	origTimeout := os.Getenv("ULORA_REQUEST_TIMEOUT")
	defer os.Setenv("ULORA_REQUEST_TIMEOUT", origTimeout)
	os.Setenv("ULORA_AUTH_TOKEN", "test-token")
	os.Setenv("ULORA_REQUEST_TIMEOUT", "invalid")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config should fall back to default timeout: %v", err)
	}
	if cfg.RequestTimeout != defaultTimeout {
		t.Fatalf("expected default timeout %v, got %v", defaultTimeout, cfg.RequestTimeout)
	}
}

func TestLoadVenvDirDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	customData := tmpDir + "/data"

	origData := os.Getenv("ULORA_DATA_DIR")
	origAuth := os.Getenv("ULORA_AUTH_TOKEN")
	origVenv := os.Getenv("ULORA_VENV_DIR")
	defer func() {
		os.Setenv("ULORA_DATA_DIR", origData)
		os.Setenv("ULORA_AUTH_TOKEN", origAuth)
		os.Setenv("ULORA_VENV_DIR", origVenv)
	}()

	os.Setenv("ULORA_AUTH_TOKEN", "test-token")
	os.Setenv("ULORA_DATA_DIR", customData)
	os.Unsetenv("ULORA_VENV_DIR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	expectedVenv := customData + "/venv"
	if cfg.VenvDir != expectedVenv {
		t.Fatalf("expected venv dir %s, got %s", expectedVenv, cfg.VenvDir)
	}
}

func TestConfigFields(t *testing.T) {
	c := &Config{
		SocketPath:     "/tmp/test.sock",
		AuthToken:      "token",
		DataDir:        "/tmp/data",
		PythonBin:      "python3",
		VenvDir:        "/tmp/venv",
		RequestTimeout: 60 * time.Second,
	}

	if c.SocketPath != "/tmp/test.sock" {
		t.Fatal("socket path mismatch")
	}
	if c.AuthToken != "token" {
		t.Fatal("auth token mismatch")
	}
	if c.RequestTimeout != 60*time.Second {
		t.Fatal("request timeout mismatch")
	}
}
