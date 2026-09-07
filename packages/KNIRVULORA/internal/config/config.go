package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultSocketPath = "./ulora.sock"
	defaultDataDir    = "./ulora_data"
	defaultTimeout    = 300 * time.Second
)

type Config struct {
	SocketPath   string
	AuthToken    string
	DataDir      string
	PythonBin    string
	VenvDir      string
	RequestTimeout time.Duration
}

func Load() (*Config, error) {
	socketPath := strings.TrimSpace(os.Getenv("ULORA_SOCKET_PATH"))
	if socketPath == "" {
		socketPath = defaultSocketPath
	}

	authToken := strings.TrimSpace(os.Getenv("ULORA_AUTH_TOKEN"))
	if authToken == "" {
		return nil, fmt.Errorf("ULORA_AUTH_TOKEN is required")
	}

	dataDir := strings.TrimSpace(os.Getenv("ULORA_DATA_DIR"))
	if dataDir == "" {
		dataDir = defaultDataDir
	}

	pythonBin := strings.TrimSpace(os.Getenv("ULORA_PYTHON_BIN"))
	if pythonBin == "" {
		pythonBin = "python3"
	}

	venvDir := strings.TrimSpace(os.Getenv("ULORA_VENV_DIR"))
	if venvDir == "" {
		venvDir = filepath.Join(dataDir, "venv")
	}

	timeoutStr := strings.TrimSpace(os.Getenv("ULORA_REQUEST_TIMEOUT"))
	timeout := defaultTimeout
	if timeoutStr != "" {
		if parsed, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsed
		}
	}

	return &Config{
		SocketPath:    socketPath,
		AuthToken:     authToken,
		DataDir:       dataDir,
		PythonBin:     pythonBin,
		VenvDir:       venvDir,
		RequestTimeout: timeout,
	}, nil
}
