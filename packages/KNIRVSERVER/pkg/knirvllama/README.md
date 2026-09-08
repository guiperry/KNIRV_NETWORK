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
| `KNIRV_LLAMA_MODEL_URL` | Explicit GGUF URL to download/cache instead of the default TinyLlama model |
| `KNIRV_LLAMA_MODEL_NAME` | Cache filename for `KNIRV_LLAMA_MODEL_URL`; defaults to the URL filename |
| `KNIRV_LLAMA_MODEL_PATH` | Use an already-downloaded GGUF instead of downloading a model |
| `KNIRV_LLAMA_API_KEY` | Optional explicit shared-secret token forwarded to llama-server as `--api-key` (a fresh 32-byte token is auto-generated when unset). The Cognitive Engine's `LlamaProvider` reads this same value to set its `Authorization: Bearer …` header. |

## Deterministic llama-server tunables (Phase A)

`ManagerConfig` exposes four tunables that are forwarded to llama-server at
launch so deployments don't inherit the binary's unsafe defaults:

| Field | Default | Forwarded as | Effect |
|---|---|---|---|
| `Parallel` | `1` | `--parallel N` | Keep the embedded CPU model uncontended. Matches the single-worker Cognitive Engine dream-task queue. |
| `CtxSize`  | `0` (binary default) | `--ctx-size N` | Pin the context window to the chosen model's actual capacity. |
| `Threads`  | `0` (binary default) | `--threads N` | Pin CPU threads to the container's actual allocation. |
| `APIKey`   | auto-generated 32-byte hex | `--api-key KEY` | Defense-in-depth token required from API clients. |

`LlamaProvider.SetAPIKey(manager.APIKey())` (Phase C) wires the same token
through to dream-task requests.

## Optional: upgrade the embedded model (Phase H)

The default model remains TinyLlama-1.1B (zero-config self-install). To
opt into a larger instruct quant, set `llama.model_url`, `llama.model_name`,
or `llama.model_path` in the active server YAML configuration. The launcher
downloads and caches a configured URL on first run; an explicit URL takes
precedence over the previously cached default model.

### 7B reference configuration

For an operator-managed upgrade, the official Qwen GGUF repository publishes
the 7.61B Qwen2.5 Instruct model. The configured `Q3_K_M` artifact is a
single GGUF file, which the embedded downloader can install directly; the
official `Q4_K_M` artifact is split into two files and requires merge support.
It is a useful first candidate for this engine because it is an instruction
model with JSON output support:

```yaml
model_url: "https://huggingface.co/Qwen/Qwen2.5-7B-Instruct-GGUF/resolve/main/qwen2.5-7b-instruct-q3_k_m.gguf"
model_name: "qwen2.5-7b-instruct-q3_k_m.gguf"
```

That quant is about 3.81 GB on disk. Reserve additional RAM for the runtime
and KV cache (at least 8 GB free RAM is a practical floor; 12–16 GB is the
safer CPU-only target), and test it with a modest context size before
increasing `ctx_size`. Keep `parallel: 1` for the single cognitive worker.


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
