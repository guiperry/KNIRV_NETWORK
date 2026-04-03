package main

import (
	"KNIRVGRAPH/internal/app"
	"context"
	"flag"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

func getAppDataDir() string {
	switch runtime.GOOS {
	case "windows":
		if dir := os.Getenv("APPDATA"); dir != "" {
			return filepath.Join(dir, "knirvgraph")
		}
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "Application Support", "knirvgraph")
		}
	case "linux":
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "knirvgraph")
		}
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".local", "share", "knirvgraph")
		}
	}
	if usr, err := user.Current(); err == nil {
		return filepath.Join(usr.HomeDir, ".local", "share", "knirvgraph")
	}
	return "data"
}

func main() {
	var (
		homeDir     = flag.String("home", "", "Home directory for GraphChain data (empty = auto-detect)")
		rpcPort     = flag.Int("rpc-port", 8080, "RPC server port")
		port        = flag.Int("port", 8081, "Alternative port specification")
		socketPath  = flag.String("socket", "", "Unix socket path for HTTP server (overrides port)")
		testnetMode = flag.Bool("testnet", false, "Run in testnet mode")
		inMemory    = flag.Bool("memory", false, "Use in-memory storage")
		prePopulate = flag.Bool("populate", false, "Pre-populate test data")
		maxNodes    = flag.Int("max-nodes", 1000, "Maximum nodes in testnet mode")
		headless    = flag.Bool("headless", true, "Run in headless mode (default: true)")
	)
	flag.Parse()

	if *homeDir == "" {
		*homeDir = getAppDataDir()
	}

	if err := os.MkdirAll(*homeDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory %s: %v", *homeDir, err)
	}

	if *rpcPort == 8080 && *port != 8081 {
		*rpcPort = *port
	}

	ctx := context.Background()

	if *headless {
		log.Println("Starting KNIRVGRAPH in headless mode (no frontend)")
	}

	var config *app.Config
	if *testnetMode {
		config = &app.Config{
			Testnet: app.TestnetConfig{
				Enabled:     true,
				InMemory:    *inMemory,
				PrePopulate: *prePopulate,
				MaxNodes:    *maxNodes,
				ChainID:     "knirvgraph-testnet-1",
				Port:        *rpcPort,
				LocalMode:   true,
			},
		}
		log.Println("Starting KNIRVGRAPH in testnet mode")
	}

	if *socketPath != "" {
		log.Printf("Using Unix socket: %s", *socketPath)
	}

	app, err := app.NewAppWithConfig(*homeDir, *rpcPort, config, true, *socketPath)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
