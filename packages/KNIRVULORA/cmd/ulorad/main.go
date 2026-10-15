package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"ulora/internal/config"
	"ulora/internal/daemon"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: ulorad [flags]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
	}

	enginesDir := flag.String("engines-dir", "", "path to Python engine scripts directory (defaults to embedded relative path)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	srv, err := daemon.NewServer(cfg)
	if err != nil {
		log.Fatalf("start server: %v", err)
	}

	if *enginesDir != "" {
		srv.SetEnginesDir(*enginesDir)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	var stopping atomic.Bool
	go func() {
		// Shutdown closes the listener, which makes Serve return an error; that is
		// the normal way out, not a failure.
		if err := srv.Serve(); err != nil && !stopping.Load() {
			log.Fatalf("serve: %v", err)
		}
	}()

	log.Println("ulorad daemon started")
	<-stop
	stopping.Store(true)
	log.Println("ulorad daemon shutting down")
	_ = srv.Shutdown()
}
