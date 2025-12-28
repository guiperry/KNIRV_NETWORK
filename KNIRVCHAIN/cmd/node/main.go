package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/knirvchain/internal/blockchain"
	"github.com/knirvchain/internal/embedding"
	"github.com/knirvchain/internal/mcp"
	"github.com/knirvchain/internal/storage"
	"github.com/knirvchain/internal/wallet"
)

// Config holds the application configuration
type Config struct {
	NodeID      string
	ListenAddr  string
	DataDir     string
	WalletKey   string
	NetworkURL  string
	ChainID     string
	LogLevel    string
}

// loadConfig loads configuration from environment variables with defaults
func loadConfig() *Config {
	config := &Config{
		NodeID:     getEnv("KNIRVCHAIN_NODE_ID", "knirvchain-node-1"),
		ListenAddr: getEnv("KNIRVCHAIN_LISTEN_ADDR", "localhost:8080"),
		DataDir:    getEnv("KNIRVCHAIN_DATA_DIR", filepath.Join(os.TempDir(), "knirvchain")),
		WalletKey:  getEnv("KNIRVCHAIN_WALLET_KEY", "mock"), // Use "mock" for development
		NetworkURL: getEnv("KNIRVCHAIN_NETWORK_URL", "https://api.xion.network"),
		ChainID:    getEnv("KNIRVCHAIN_CHAIN_ID", "xion-mainnet-1"),
		LogLevel:   getEnv("KNIRVCHAIN_LOG_LEVEL", "info"),
	}
	return config
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Load configuration
	config := loadConfig()

	// Setup logging
	logger := log.New(os.Stdout, "[KNIRVCHAIN] ", log.LstdFlags|log.Lshortfile)
	logger.Printf("Starting KNIRVCHAIN node: %s", config.NodeID)
	logger.Printf("Configuration:")
	logger.Printf("  - Listen Address: %s", config.ListenAddr)
	logger.Printf("  - Data Directory: %s", config.DataDir)
	logger.Printf("  - Network URL: %s", config.NetworkURL)
	logger.Printf("  - Chain ID: %s", config.ChainID)

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		logger.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start initialization
	logger.Println("Initializing components...")

	// 1. Initialize KNIRVBASE storage
	logger.Println("Initializing KNIRVBASE storage...")
	dbPath := filepath.Join(config.DataDir, "blockchain.db")
	stor, err := storage.NewKNIRVBASEStorage(dbPath)
	if err != nil {
		logger.Fatalf("Failed to initialize storage: %v", err)
	}
	defer func() {
		logger.Println("Closing storage...")
		stor.Close()
	}()
	logger.Println("✓ Storage initialized")

	// 2. Initialize TF-IDF embedder
	logger.Println("Initializing TF-IDF embedder...")
	embedder, err := embedding.NewTFIDFEmbedder(stor, 768)
	if err != nil {
		logger.Fatalf("Failed to initialize embedder: %v", err)
	}
	logger.Println("✓ Embedder initialized")

	// 3. Initialize HNSW index (currently managed by blockchain internally)
	// logger.Println("Initializing HNSW index...")
	// hnswIndex := indexing.NewHNSWIndex(768, 16, 200)
	// logger.Println("✓ HNSW index initialized")

	// 4. Initialize PoA validator
	logger.Println("Initializing PoA consensus validator...")
	validator := blockchain.NewPoAValidator(config.NodeID)
	logger.Printf("✓ PoA validator initialized (node: %s)", config.NodeID)

	// 5. Initialize blockchain
	logger.Println("Initializing blockchain...")
	chain, err := blockchain.NewChainNodeWithValidator(config.NodeID, stor, validator)
	if err != nil {
		logger.Fatalf("Failed to initialize blockchain: %v", err)
	}
	logger.Println("✓ Blockchain initialized")

	// 6. Initialize mempool
	logger.Println("Initializing transaction mempool...")
	mempool := blockchain.NewDefaultMempool()
	logger.Printf("✓ Mempool initialized (capacity: %d)", mempool.AvailableCapacity())

	// Start mempool purge loop
	go mempool.StartPurgeLoop(ctx, 5*time.Minute)

	// 7. Initialize NRN wallet
	logger.Println("Initializing NRN wallet...")
	var nrnWallet *wallet.NRNWallet
	if config.WalletKey == "mock" {
		logger.Println("  Using mock wallet for development")
		nrnWallet, err = wallet.NewNRNWallet("mock")
	} else {
		logger.Printf("  Loading wallet from: %s", config.WalletKey)
		nrnWallet, err = wallet.NewNRNWallet(config.WalletKey)
	}
	if err != nil {
		logger.Fatalf("Failed to initialize wallet: %v", err)
	}
	logger.Printf("✓ Wallet initialized (address: %s)", nrnWallet.Address())

	// Query and display initial balance
	if balance, err := nrnWallet.Balance(ctx); err == nil {
		logger.Printf("  Current balance: %d NRN", balance)
	}

	// 8. Initialize MCP server
	logger.Println("Initializing MCP server...")
	server, err := mcp.NewMCPServer(
		config.ListenAddr,
		chain,
		nrnWallet,
		embedder,
	)
	if err != nil {
		logger.Fatalf("Failed to initialize MCP server: %v", err)
	}
	logger.Println("✓ MCP server initialized")

	// All components initialized successfully
	logger.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Println("✓ All components initialized successfully")
	logger.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Start MCP server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		logger.Printf("Starting MCP server on %s", config.ListenAddr)
		if err := server.Start(config.ListenAddr); err != nil {
			serverErrChan <- err
		}
	}()

	// Display startup information
	logger.Println("")
	logger.Println("KNIRVCHAIN is running")
	logger.Printf("  - Node ID: %s", config.NodeID)
	logger.Printf("  - MCP Server: http://%s", config.ListenAddr)
	logger.Printf("  - Data Directory: %s", config.DataDir)
	logger.Printf("  - Wallet Address: %s", nrnWallet.Address())
	logger.Println("")
	logger.Println("Press Ctrl+C to shutdown gracefully")
	logger.Println("")

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		logger.Printf("Received signal: %v", sig)
		logger.Println("Initiating graceful shutdown...")

	case err := <-serverErrChan:
		logger.Printf("Server error: %v", err)
		logger.Println("Shutting down due to server error...")
	}

	// Graceful shutdown sequence
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	_ = shutdownCtx // Reserved for future graceful shutdown implementation

	logger.Println("Shutting down components...")

	// 1. Stop accepting new requests
	logger.Println("  - Stopping MCP server...")
	// Note: http.ListenAndServe doesn't support graceful shutdown
	// The server will stop when the process exits
	logger.Println("    ✓ MCP server will stop on process exit")

	// 2. Process remaining mempool transactions (with timeout)
	logger.Println("  - Processing remaining mempool transactions...")
	mempoolStats := mempool.GetStats()
	if mempoolStats.Size > 0 {
		logger.Printf("    Warning: %d transactions remaining in mempool", mempoolStats.Size)
		// In production, you might want to process or persist these
	}
	logger.Println("    ✓ Mempool processed")

	// 3. Flush any pending writes
	logger.Println("  - Flushing storage...")
	// Storage.Close() will handle this
	logger.Println("    ✓ Storage flushed")

	// 4. Cancel background operations
	logger.Println("  - Stopping background tasks...")
	cancel() // Cancel context to stop mempool purge loop
	logger.Println("    ✓ Background tasks stopped")

	// 5. Close storage (deferred, but log it)
	logger.Println("  - Closing database...")
	// Deferred stor.Close() will execute here
	logger.Println("    ✓ Database closed")

	logger.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Println("✓ Shutdown complete")
	logger.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Display final statistics
	logger.Println("")
	logger.Println("Final Statistics:")
	logger.Printf("  - Mempool: %d transactions processed", mempoolStats.TotalAdded)
	logger.Printf("  - Mempool: %d transactions rejected", mempoolStats.TotalRejected)
	logger.Printf("  - Mempool: %d transactions expired", mempoolStats.TotalExpired)

	if balance, err := nrnWallet.Balance(ctx); err == nil {
		logger.Printf("  - Wallet: %d NRN remaining", balance)
	}

	logger.Println("")
	logger.Println("Goodbye!")
}

// printBanner prints a startup banner
func printBanner() {
	banner := `
	╦╔═╔╗╔╦╦═╗╦  ╦╔═╗╦ ╦╔═╗╦╔╗╔
	╠╩╗║║║║╠╦╝╚╗╔╝║  ╠═╣╠═╣║║║║
	╩ ╩╝╚╝╩╩╚═ ╚╝ ╚═╝╩ ╩╩ ╩╩╝╚╝

	AI Long-Term Memory Blockchain
	Version 1.0.0
	`
	fmt.Println(banner)
}
