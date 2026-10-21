// llama provisions llama.cpp when needed and exposes local chat endpoints.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/guiperry/knirv-sdk-go/models/frame"
	"github.com/guiperry/knirv/llama/internal/adapters"
	"github.com/guiperry/knirv/llama/internal/config"
	"github.com/guiperry/knirv/llama/internal/httpapi"
	"github.com/guiperry/knirv/llama/internal/install"
	"github.com/guiperry/knirv/llama/internal/runtime"
)

func main() {
	var dataDir, listen, llamaAddress, serverPath, modelPath, modelURL, modelName string
	var noInstall bool
	var unixSocket string
	var parallel, ctxSize, threads int
	var apiKey string
	flag.StringVar(&dataDir, "data-dir", "", "directory for llama.cpp and models")
	flag.StringVar(&listen, "listen", "127.0.0.1:8080", "address for chat API")
	flag.StringVar(&unixSocket, "unix-socket", "", "path for chat API Unix socket (overrides -listen)")
	flag.StringVar(&llamaAddress, "llama-address", "127.0.0.1:8000", "llama-server address")
	flag.StringVar(&serverPath, "server-path", "", "path to llama-server")
	flag.StringVar(&modelPath, "model-path", "", "path to a GGUF model")
	flag.StringVar(&modelURL, "model-url", "", "URL for the model downloaded on first run")
	flag.StringVar(&modelName, "model-name", "", "file name used when caching a model URL")
	flag.IntVar(&parallel, "parallel", 1, "number of parallel llama-server slots (1 keeps the embedded CPU model uncontended)")
	flag.IntVar(&ctxSize, "ctx-size", 0, "context window in tokens (0 = llama-server default)")
	flag.IntVar(&threads, "threads", 0, "CPU threads for llama-server (0 = llama-server default)")
	flag.StringVar(&apiKey, "api-key", "", "shared-secret token required from API clients (defense-in-depth; bind address already restricts access)")
	flag.BoolVar(&noInstall, "no-install", false, "fail instead of installing missing dependencies")
	flag.Parse()
	// Environment overrides make a model upgrade operational rather than a
	// binary rebuild. CLI flags still win for one-off operator overrides.
	if modelPath == "" {
		modelPath = os.Getenv("KNIRV_LLAMA_MODEL_PATH")
	}
	if modelURL == "" {
		modelURL = os.Getenv("KNIRV_LLAMA_MODEL_URL")
	}
	if modelName == "" {
		modelName = os.Getenv("KNIRV_LLAMA_MODEL_NAME")
	}

	dataDir, configPath, err := config.Paths(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	if saved, err := config.Load(configPath); err == nil {
		if serverPath == "" {
			serverPath = saved.ServerPath
		}
		// A requested URL is an explicit upgrade/downgrade request; never let a
		// previously cached model silently win over it.
		if modelPath == "" && modelURL == "" {
			modelPath = saved.ModelPath
		}
	}
	if modelName == "" && modelURL != "" {
		if u, err := url.Parse(modelURL); err == nil {
			modelName = filepath.Base(u.Path)
		}
	}
	if modelName == "" && modelPath != "" {
		modelName = filepath.Base(modelPath)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	installer := install.New()
	installer.Prog = install.NewProgress(os.Stderr)
	result, err := installer.Ensure(ctx, install.Options{
		DataDir:    dataDir,
		ServerPath: serverPath,
		ServerURL:  os.Getenv("KNIRV_LLAMA_SERVER_URL"),
		ModelPath:  modelPath,
		ModelURL:   modelURL,
		ModelName:  modelName,
		NoInstall:  noInstall,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := config.Save(configPath, config.Config{ServerPath: result.ServerPath, ModelPath: result.ModelPath, ModelName: result.ModelName}); err != nil {
		log.Fatal(err)
	}

	// llama-server is started by the supervisor so its adapter set can change: it
	// loads adapters only at startup, so adding one is a restart (internal/adapters).
	store, err := adapters.OpenStore(filepath.Join(dataDir, "adapters"))
	if err != nil {
		log.Fatal(err)
	}
	gate := adapters.NewGate()
	base := adapters.Base{Model: result.ModelName, Path: result.ModelPath, SHA256: memoSHA(dataDir, result.ModelPath)}
	var restart adapters.Restarter
	var sup *runtime.Supervisor
	external := runtime.Healthy(context.Background(), llamaAddress)
	if !external {
		// Surface what would otherwise be a silent timeout: an unreadable or truncated
		// model, a taken port, memory too small to hold the model.
		report := runtime.Preflight(result.ServerPath, result.ModelPath, llamaAddress)
		for _, w := range report.Warnings {
			log.Printf("preflight warning: %s", w)
		}
		if !report.OK() {
			for _, e := range report.Errors {
				log.Printf("preflight error: %s", e)
			}
			log.Fatalf("llama-server cannot start: %d preflight error(s) above", len(report.Errors))
		}
		sup = &runtime.Supervisor{
			Path: result.ServerPath, Model: result.ModelPath, Address: llamaAddress,
			Base: runtime.Options{Parallel: parallel, CtxSize: ctxSize, Threads: threads, APIKey: apiKey,
				Mlock: os.Getenv("KNIRV_LLAMA_MLOCK") == "1"},
			Logf: func(format string, args ...any) { log.Printf("llama-server: "+format, args...) },
		}
		defer sup.Stop()
		restart = func(ctx context.Context, set []adapters.Entry) error {
			list := make([]runtime.LoRAAdapter, len(set))
			for i, e := range set {
				list[i] = runtime.LoRAAdapter{Path: e.Path}
			}
			return sup.Start(ctx, list)
		}
	} else {
		log.Printf("llama-server already running at %s; adapters cannot be managed", llamaAddress)
	}
	mgr := adapters.NewManager(store, base, restart, gate)
	if err := mgr.Restore(context.Background()); err != nil {
		log.Fatalf("start llama-server: %v", err)
	}
	handler, err := httpapi.NewServer(llamaAddress, result.ModelName, apiKey, mgr, httpapi.Options{
		Frames: frame.Default(), Mode: httpapi.ParseFrameMode(os.Getenv("KNIRV_FRAME_MODE")),
		AdmitSlots: max(parallel, 1), // one admission slot per llama-server slot
	})
	if err != nil {
		log.Fatal(err)
	}

	var listener net.Listener
	if unixSocket != "" {
		if err := os.RemoveAll(unixSocket); err != nil {
			log.Fatalf("failed to remove stale unix socket: %v", err)
		}
		listener, err = net.Listen("unix", unixSocket)
		if err != nil {
			log.Fatalf("failed to create unix socket listener: %v", err)
		}
	} else {
		listener, err = net.Listen("tcp", listen)
		if err != nil {
			log.Fatalf("failed to listen on %s: %v", listen, err)
		}
	}

	httpServer := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() { <-signals; _ = httpServer.Shutdown(context.Background()) }()

	if unixSocket != "" {
		fmt.Printf("llama chat API listening on unix socket %s (POST /v1/chat/completions)\n", unixSocket)
	} else {
		fmt.Printf("llama chat API listening on http://%s (POST /v1/chat/completions)\n", listen)
	}
	if err := httpServer.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// memoSHA returns a function that hashes the base model once per (path, size,
// mtime), caching the result in the data directory: a multi-GB model takes seconds
// to hash and is asked about whenever an adapter is registered.
func memoSHA(dataDir, modelPath string) func() (string, error) {
	return func() (string, error) {
		st, err := os.Stat(modelPath)
		if err != nil {
			return "", err
		}
		cachePath := filepath.Join(dataDir, "adapters", "base-sha256.json")
		key := fmt.Sprintf("%s|%d|%d", modelPath, st.Size(), st.ModTime().UnixNano())
		var cache map[string]string
		if raw, err := os.ReadFile(cachePath); err == nil {
			_ = json.Unmarshal(raw, &cache)
		}
		if v := cache[key]; v != "" {
			return v, nil
		}
		f, err := os.Open(modelPath)
		if err != nil {
			return "", err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
		sum := hex.EncodeToString(h.Sum(nil))
		_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
		if raw, err := json.Marshal(map[string]string{key: sum}); err == nil {
			_ = os.WriteFile(cachePath, raw, 0o644)
		}
		return sum, nil
	}
}
