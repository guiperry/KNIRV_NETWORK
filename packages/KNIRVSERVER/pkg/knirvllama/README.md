# KNIRVLLAMA - Embedded Local Llama.cpp Inference Provider

## Overview

KNIRVLLAMA embeds a self-contained `llama` binary that provisions and serves
llama.cpp models behind an OpenAI-compatible HTTP API. It follows the same
embedded-binary pattern as `pkg/knirvhasher`, `pkg/knirvagent`, and other
`pkg/knirv*` packages.

## Usage

### Extracting the binary

```go
import "knirv-server/pkg/knirvllama"

binaryPath, err := knirvllama.ExtractEmbeddedBinary("")
if err != nil {
    log.Fatal(err)
}
```

### Managing the subprocess

```go
import (
    "context"
    "go.uber.org/zap"
    "knirv-server/pkg/knirvllama"
)

cfg := knirvllama.DefaultManagerConfig()
cfg.BinaryPath = binaryPath
manager := knirvllama.NewManager(cfg, zap.NewNop())

if err := manager.Start(context.Background()); err != nil {
    log.Fatal(err)
}
defer manager.Stop(context.Background())

healthy, err := manager.GetHealth()
listenAddr := manager.GetListenAddr()
```

## Environment Variables

| Variable | Purpose |
|---|---|
| `KNIRV_LLAMA_BINARY_DIR` | Override extraction directory for the vendored binary |
| `KNIRV_LLAMA_BINARY_PATH` | Use an existing binary instead of the embedded one |
| `KNIRV_APP_DATA_DIR` | Root data directory (defaults to `/var/lib/knirvserver`) |
| `KNIRV_LLAMA_SERVER_URL` | Override the prebuilt Linux `llama-server` artifact URL; defaults to `https://releases.knirv.com/knirv/llama/linux-amd64/llama-server.gz` |

On first run, the embedded launcher downloads and caches the Linux `llama-server`
release. If that artifact cannot be downloaded or used, it falls back to a native
llama.cpp build and installs only missing `git`, `cmake`, and build-tool packages
through `apt-get`. Existing cached binaries and installed packages are not replaced.

## Configuration

```go
type ManagerConfig struct {
	BinaryPath   string        // Path to the knirvllama binary
	ListenAddr   string        // Loopback HTTP address for the chat API
	SocketPath   string        // Optional Unix socket for HTTP API
	LlamaAddress string        // llama-server child address
    DataDir      string        // Data directory for llama.cpp state
    StartTimeout time.Duration // Max time to wait for health check
    StopTimeout  time.Duration // Max time to wait for graceful shutdown
    EnvOverrides map[string]string
}
```
