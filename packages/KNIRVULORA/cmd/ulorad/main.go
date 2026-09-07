package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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

	go func() {
		if err := srv.Serve(); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	log.Println("ulorad daemon started")
	<-stop
	log.Println("ulorad daemon shutting down")
	_ = srv.Shutdown()
}
