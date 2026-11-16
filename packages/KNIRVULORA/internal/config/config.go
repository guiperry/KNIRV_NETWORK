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
	defaultTimeout    = 300 * time.Second
)

type Config struct {
	SocketPath     string
	AuthToken      string
	DataDir        string
	PythonBin      string
	VenvDir        string
	ConnectorDir   string
	RequestTimeout time.Duration
}

func Load() (*Config, error) {
	socketPath := strings.TrimSpace(os.Getenv("ULORA_SOCKET_PATH"))
	if socketPath == "" {
		socketPath = defaultSocketPath
	}

	// Data dir resolution is the same in KNIRVGRAPH (drq.uloraDataDirFromEnv): an
	// explicit ULORA_DATA_DIR, then <KNIRV_APP_DATA_DIR>/ulora, then
	// <launcher app data dir>/ulora (see resolveAppDataDir).
	dataDir := strings.TrimSpace(os.Getenv("ULORA_DATA_DIR"))
	if dataDir == "" {
		appDataDir, err := resolveAppDataDir()
		if err != nil {
			return nil, fmt.Errorf("resolve ulorad data dir: %w", err)
		}
		dataDir = filepath.Join(appDataDir, "ulora")
	}

	// ULORA_AUTH_TOKEN is optional. When it is unset, ulorad uses the token in its
	// own data dir (reused across restarts) or generates one on first start.
	authToken := strings.TrimSpace(os.Getenv("ULORA_AUTH_TOKEN"))
	if authToken == "" {
		generated, err := LoadOrCreateAuthToken(dataDir)
		if err != nil {
			return nil, fmt.Errorf("resolve ulorad auth token: %w", err)
		}
		authToken = generated
	}

	// Connectors are cached per model rather than carried in bundles, so the
	// directory is separate from bundles/ and survives them: rebuilding every
	// skill should not re-derive the same projections.
	connectorDir := strings.TrimSpace(os.Getenv("ULORA_CONNECTOR_DIR"))
	if connectorDir == "" {
		connectorDir = filepath.Join(dataDir, "connectors")
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
		SocketPath:     socketPath,
		AuthToken:      authToken,
		DataDir:        dataDir,
		PythonBin:      pythonBin,
		VenvDir:        venvDir,
		ConnectorDir:   connectorDir,
		RequestTimeout: timeout,
	}, nil
}
