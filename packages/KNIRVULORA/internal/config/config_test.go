package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	origSocket := os.Getenv("ULORA_SOCKET_PATH")
	origAuth := os.Getenv("ULORA_AUTH_TOKEN")
	origData := os.Getenv("ULORA_DATA_DIR")
	defer func() {
		os.Setenv("ULORA_SOCKET_PATH", origSocket)
		os.Setenv("ULORA_AUTH_TOKEN", origAuth)
		os.Setenv("ULORA_DATA_DIR", origData)
	}()

	os.Unsetenv("ULORA_SOCKET_PATH")
	os.Setenv("ULORA_AUTH_TOKEN", "test-token")
	os.Unsetenv("ULORA_DATA_DIR")

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
	if cfg.DataDir != defaultDataDir {
		t.Fatalf("expected default data dir %s, got %s", defaultDataDir, cfg.DataDir)
	}
	if cfg.PythonBin != "python3" {
		t.Fatalf("expected default python bin python3, got %s", cfg.PythonBin)
	}
	if cfg.RequestTimeout != defaultTimeout {
		t.Fatalf("expected default timeout %v, got %v", defaultTimeout, cfg.RequestTimeout)
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

func TestLoadMissingAuthToken(t *testing.T) {
	origAuth := os.Getenv("ULORA_AUTH_TOKEN")
	defer os.Setenv("ULORA_AUTH_TOKEN", origAuth)
	os.Unsetenv("ULORA_AUTH_TOKEN")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when ULORA_AUTH_TOKEN is not set")
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
		SocketPath:    "/tmp/test.sock",
		AuthToken:     "token",
		DataDir:       "/tmp/data",
		PythonBin:     "python3",
		VenvDir:       "/tmp/venv",
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
