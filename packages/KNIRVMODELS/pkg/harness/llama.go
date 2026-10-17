package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"knirvmodels/pkg/knirvllama"
)

// NewLlamaFallback builds the llama.cpp manager from the embedded binary. appDataDir
// is the server data directory (the one KNIRV_APP_DATA_DIR names) and socket is where
// the llama chat wrapper listens; the Router in front of it owns the public socket.
// The same environment variables the server has always used tune it:
// KNIRV_LLAMA_PARALLEL, KNIRV_LLAMA_CTX_SIZE, KNIRV_LLAMA_THREADS and the model
// selectors KNIRV_LLAMA_MODEL_URL, _NAME and _PATH (the wrapper reads those itself).
func NewLlamaFallback(appDataDir, socket string, logf func(string, ...any)) (*knirvllama.Manager, error) {
	binary, err := knirvllama.ExtractEmbeddedBinary(filepath.Join(appDataDir, "bin"))
	if err != nil {
		return nil, fmt.Errorf("extract KNIRVLLAMA binary: %w", err)
	}
	cfg := knirvllama.DefaultManagerConfig()
	cfg.BinaryPath = binary
	cfg.DataDir = filepath.Join(appDataDir, "knirvllama")
	cfg.SocketPath = socket
	cfg.Parallel = envInt("KNIRV_LLAMA_PARALLEL", cfg.Parallel)
	cfg.CtxSize = envInt("KNIRV_LLAMA_CTX_SIZE", cfg.CtxSize)
	cfg.Threads = envInt("KNIRV_LLAMA_THREADS", cfg.Threads)
	cfg.Logf = logf
	return knirvllama.NewManager(cfg, zap.NewNop()), nil
}

func envInt(name string, def int) int {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
