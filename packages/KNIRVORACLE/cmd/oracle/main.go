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
	"github.com/knirvcorp/knirvoracle/internal/oracle/actuarial"
	"github.com/knirvcorp/knirvoracle/internal/oracle/chainverify"
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

	if *configFile != "" {
		log.Printf("Config file specified: %s (loading from file not yet implemented)", *configFile)
	}
	if *dataDir != "" {
		if err := os.Setenv("ORACLE_DATA_DIR", *dataDir); err != nil {
			log.Fatalf("Failed to set ORACLE_DATA_DIR: %v", err)
		}
	}

	resolvedSocketPath, err := resolveSocketPath(*socketPath)
	if err != nil {
		log.Fatalf("Failed to resolve oracle socket path: %v", err)
	}
	*socketPath = resolvedSocketPath

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	oracleInst, err := initOracleFromKeyFile(logger)
	if err != nil {
		log.Fatalf("Failed to initialize Oracle from root.key: %v", err)
	}
	if oracleInst == nil {
		log.Println("Oracle not activated: no usable root.key found")
		return
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

	// Actuarial syndicate settlement payouts: the only path allowed to
	// disburse NRN for a syndicate settlement, gated on a verified
	// SETTLEMENT_COMMIT proof from KNIRVCHAIN (see internal/oracle/actuarial).
	chainVerifyURL := os.Getenv("ORACLE_CHAIN_VERIFY_URL")
	actuarialHandler := actuarial.NewHandler(oracleInst, chainverify.NewClient(chainVerifyURL), logger)
	mux.HandleFunc("/oracle/v3/actuarial/settlements/payout", actuarialHandler.HandleSettlementPayout)

	server := &http.Server{Handler: mux}

	if err := oracleInst.Start(); err != nil {
		log.Fatalf("Failed to start Oracle: %v", err)
	}

	log.Println("Oracle services started — tunnel managed by KNIRVGATEWAY")

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

func resolveSocketPath(currentFlag string) (string, error) {
	if currentFlag != "" && currentFlag != "/var/run/knirv/oracle.sock" {
		return currentFlag, nil
	}

	if envPath := os.Getenv("ORACLE_SOCKET_PATH"); envPath != "" {
		return envPath, nil
	}

	appDataDir, err := getOSAppDataDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(appDataDir, "sockets", "oracle.sock"), nil
}

func handleHealth(o *oracle.Oracle) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","oracle":"active","version":"` + Version + `"}`))
	}
}
