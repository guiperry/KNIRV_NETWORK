package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"knirvmodels/pkg/harness"
)

// runInference starts every pre-trained model this binary serves for inference:
// Colibri as the primary harness, llama.cpp as its fallback, both behind one socket.
// It blocks until SIGINT or SIGTERM.
//
//	KNIRV_COLIBRI=0            run llama.cpp alone
//	KNIRV_LLAMA_FALLBACK=0     run Colibri alone
//	KNIRV_COLIBRI_MODEL=<id>   choose Colibri's model instead of its recommendation
//	KNIRV_LLAMA_KEEP=1         keep llama.cpp running while Colibri serves
//	KNIRV_COLIBRI_VERSION      Colibri release tag to install (default v2.0.0)
//	KNIRV_COLIBRI_SOURCE_URL   Colibri source archive URL (mirror / air-gapped install)
func runInference(socket string) {
	logf := func(format string, args ...any) { log.Printf("[MODELS] "+format, args...) }
	appData := inferenceAppDataDir()
	if err := os.MkdirAll(filepath.Dir(socket), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "inference socket directory: %v\n", err)
		os.Exit(1)
	}

	cfg := harness.Config{
		FrontSocket:  socket,
		KeepFallback: os.Getenv("KNIRV_LLAMA_KEEP") == "1",
		Logf:         logf,
	}
	if os.Getenv("KNIRV_COLIBRI") != "0" {
		cfg.Primary = harness.NewColibri(harness.ColibriConfig{
			Home: filepath.Join(appData, "colibri"),
			Logf: logf,
		})
	}
	if os.Getenv("KNIRV_LLAMA_FALLBACK") != "0" {
		llamaSocket := filepath.Join(filepath.Dir(socket), "llama-cpp.sock")
		m, err := harness.NewLlamaFallback(appData, llamaSocket, logf)
		if err != nil {
			logf("llama.cpp fallback unavailable: %v", err)
		} else {
			cfg.Fallback = m
		}
	}
	if cfg.Primary == nil && cfg.Fallback == nil {
		fmt.Fprintln(os.Stderr, "no model harness enabled")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	logf("starting model harness (primary: colibri=%t, fallback: llama.cpp=%t) on unix://%s",
		cfg.Primary != nil, cfg.Fallback != nil, socket)
	harness.New(cfg).Run(ctx)
	logf("model harness stopped")
}

func inferenceAppDataDir() string {
	if d := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); d != "" {
		return d
	}
	if err := os.MkdirAll("/var/lib/knirvserver", 0o755); err == nil {
		return "/var/lib/knirvserver"
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "knirvserver")
	}
	return filepath.Join(os.TempDir(), "knirvserver")
}
