package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/routes"
	"go.uber.org/zap"
)

var (
	socketPath = flag.String("socket", "/var/run/knirv/oracle.sock", "Unix socket path")
	dataDir    = flag.String("data-dir", "", "Oracle data directory")
	configFile = flag.String("config", "", "Config file path")
)

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	flag.Parse()

	fmt.Printf("KNIRVORACLE v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

	cfg, err := oracle.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if *configFile != "" {
		log.Printf("Config file specified: %s (loading from file not yet implemented)", *configFile)
	}
	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	if err := oracle.ValidateConfig(cfg); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	log.Printf("Oracle config: %s", oracle.ConfigSummary(cfg))

	oracleInst, err := oracle.NewOracle(cfg, logger)
	if err != nil {
		log.Fatalf("Failed to create Oracle: %v", err)
	}

	socketDir := filepath.Dir(*socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		log.Fatalf("Failed to create socket directory: %v", err)
	}

	if err := os.Remove(*socketPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to remove existing socket: %v", err)
	}

	listener, err := net.Listen("unix", *socketPath)
	if err != nil {
		log.Fatalf("Failed to create socket listener: %v", err)
	}
	if err := os.Chmod(*socketPath, 0600); err != nil {
		log.Printf("Warning: failed to set socket permissions: %v", err)
	}

	log.Printf("Oracle listening on %s", *socketPath)

	mux := http.NewServeMux()
	oracleRoutes := routes.NewOracleRoutes(oracleInst, logger)
	oracleRoutes.RegisterRoutes(mux)
	mux.HandleFunc("/health", handleHealth(oracleInst))

	server := &http.Server{Handler: mux}

	if err := oracleInst.Start(); err != nil {
		log.Fatalf("Failed to start Oracle: %v", err)
	}

	log.Println("Oracle services started")

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down Oracle...")

	if err := oracleInst.Stop(); err != nil {
		log.Printf("Error stopping Oracle: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	listener.Close()
}

func handleHealth(o *oracle.Oracle) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","oracle":"active","version":"` + Version + `"}`))
	}
}
