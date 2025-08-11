package app

import (
	"blockchain-app/internal/economics"
	"blockchain-app/internal/graphchain"
	"blockchain-app/internal/network"
	"blockchain-app/internal/nrv"
	"blockchain-app/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// TestnetConfig holds testnet-specific configuration
type TestnetConfig struct {
	Enabled     bool   `json:"enabled"`
	InMemory    bool   `json:"in_memory"`
	PrePopulate bool   `json:"pre_populate"`
	MaxNodes    int    `json:"max_nodes"`
	ChainID     string `json:"chain_id"`
	Port        int    `json:"port"`
	LocalMode   bool   `json:"local_mode"`
}

// Config holds the application configuration
type Config struct {
	Testnet TestnetConfig `json:"testnet"`
}

type App struct {
	graphchain      *graphchain.GraphChain
	nrvSystem       *nrv.NRVSystem
	nrnIntegration  *economics.NRNIntegration
	proofOfSolution *economics.ProofOfSolution
	rpc             *network.RPCServer
	storage         storage.GraphStorage
	logger          *zap.Logger
	config          *Config
}

func NewApp(homeDir string, rpcPort int) (*App, error) {
	logger, _ := zap.NewProduction()

	// Initialize BluntDB storage
	storageInstance, err := storage.NewBluntDBStorage(fmt.Sprintf("%s/data", homeDir))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize BluntDB storage: %w", err)
	}

	// Initialize GraphChain
	gc := graphchain.NewGraphChain(storageInstance)

	// Initialize NRV system
	nrvSystem := nrv.NewNRVSystem("local-peer", nil)

	// Get KNIRVROOT URL from environment or use default
	knirvRootURL := os.Getenv("KNIRVROOT_URL")
	if knirvRootURL == "" {
		knirvRootURL = "http://localhost:1317" // Default KNIRVROOT URL
	}

	// Initialize NRN integration
	nrnIntegration := economics.NewNRNIntegration(knirvRootURL, nrvSystem)

	// Initialize Proof-of-Solution
	proofOfSolution := economics.NewProofOfSolution(nrnIntegration, nrvSystem)

	// Initialize RPC server with NRV system and economics
	rpc := network.NewRPCServerWithEconomics(gc, nrvSystem, nrnIntegration, proofOfSolution, logger, rpcPort)

	return &App{
		graphchain:      gc,
		nrvSystem:       nrvSystem,
		nrnIntegration:  nrnIntegration,
		proofOfSolution: proofOfSolution,
		rpc:             rpc,
		storage:         storageInstance,
		logger:          logger,
	}, nil
}

// NewAppWithConfig creates a new App instance with optional configuration
func NewAppWithConfig(homeDir string, rpcPort int, config *Config) (*App, error) {
	logger, _ := zap.NewProduction()

	var storageInstance storage.GraphStorage
	var err error

	// Use in-memory storage for testnet if configured
	if config != nil && config.Testnet.Enabled && config.Testnet.InMemory {
		logger.Info("Using in-memory storage for testnet")
		storageInstance, err = storage.NewMemoryStorage()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize memory storage: %w", err)
		}
	} else {
		// Initialize BluntDB storage
		storageInstance, err = storage.NewBluntDBStorage(fmt.Sprintf("%s/data", homeDir))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize BluntDB storage: %w", err)
		}
	}

	// Initialize GraphChain
	gc := graphchain.NewGraphChain(storageInstance)

	// Initialize NRV system
	nrvSystem := nrv.NewNRVSystem("local-peer", nil)

	// Get KNIRVROOT URL from environment or use default
	knirvRootURL := os.Getenv("KNIRVROOT_URL")
	if knirvRootURL == "" {
		knirvRootURL = "http://localhost:1317" // Default KNIRVROOT URL
	}

	// Initialize NRN integration
	nrnIntegration := economics.NewNRNIntegration(knirvRootURL, nrvSystem)

	// Initialize Proof-of-Solution
	proofOfSolution := economics.NewProofOfSolution(nrnIntegration, nrvSystem)

	// Initialize RPC server with NRV system and economics
	rpc := network.NewRPCServerWithEconomics(gc, nrvSystem, nrnIntegration, proofOfSolution, logger, rpcPort)

	app := &App{
		graphchain:      gc,
		nrvSystem:       nrvSystem,
		nrnIntegration:  nrnIntegration,
		proofOfSolution: proofOfSolution,
		rpc:             rpc,
		storage:         storageInstance,
		logger:          logger,
		config:          config,
	}

	// Pre-populate test data if testnet mode is enabled
	if config != nil && config.Testnet.Enabled && config.Testnet.PrePopulate {
		if err := app.prePopulateTestData(); err != nil {
			logger.Warn("Failed to pre-populate test data", zap.Error(err))
		}
	}

	return app, nil
}

// prePopulateTestData adds sample nodes and edges for testing
func (app *App) prePopulateTestData() error {
	app.logger.Info("Pre-populating test data for testnet")

	// Create sample ErrorNodes
	errorNodes := []struct {
		ID          string
		Description string
		ErrorType   string
	}{
		{"error_001", "Network timeout error", "network"},
		{"error_002", "Database connection failed", "database"},
		{"error_003", "Authentication failed", "auth"},
		{"error_004", "File not found", "filesystem"},
		{"error_005", "Memory allocation error", "memory"},
		{"error_006", "Invalid input format", "validation"},
		{"error_007", "Service unavailable", "service"},
		{"error_008", "Rate limit exceeded", "rate_limit"},
		{"error_009", "Configuration error", "config"},
		{"error_010", "SSL certificate expired", "security"},
	}

	// Create sample SkillNodes
	skillNodes := []struct {
		ID          string
		Name        string
		Description string
		Category    string
	}{
		{"skill_001", "Error Handler", "Handles network timeout errors", "error_handling"},
		{"skill_002", "DB Reconnect", "Reconnects to database on failure", "database"},
		{"skill_003", "Auth Retry", "Retries authentication with backoff", "authentication"},
		{"skill_004", "File Recovery", "Recovers missing files from backup", "filesystem"},
		{"skill_005", "Memory Cleanup", "Cleans up memory leaks", "memory_management"},
	}

	// Add ErrorNodes to storage
	for _, node := range errorNodes {
		nodeData := map[string]interface{}{
			"id":          node.ID,
			"type":        "ErrorNode",
			"description": node.Description,
			"error_type":  node.ErrorType,
			"created_at":  "2025-08-06T00:00:00Z",
		}

		data, err := json.Marshal(nodeData)
		if err != nil {
			return fmt.Errorf("failed to marshal error node %s: %w", node.ID, err)
		}

		if err := app.storage.PutNode(node.ID, data); err != nil {
			return fmt.Errorf("failed to store error node %s: %w", node.ID, err)
		}
	}

	// Add SkillNodes to storage
	for _, node := range skillNodes {
		nodeData := map[string]interface{}{
			"id":          node.ID,
			"type":        "SkillNode",
			"name":        node.Name,
			"description": node.Description,
			"category":    node.Category,
			"created_at":  "2025-08-06T00:00:00Z",
		}

		data, err := json.Marshal(nodeData)
		if err != nil {
			return fmt.Errorf("failed to marshal skill node %s: %w", node.ID, err)
		}

		if err := app.storage.PutNode(node.ID, data); err != nil {
			return fmt.Errorf("failed to store skill node %s: %w", node.ID, err)
		}
	}

	// Create relationships between ErrorNodes and SkillNodes
	relationships := []struct {
		ErrorID string
		SkillID string
	}{
		{"error_001", "skill_001"}, // Network timeout -> Error Handler
		{"error_002", "skill_002"}, // DB connection -> DB Reconnect
		{"error_003", "skill_003"}, // Auth failed -> Auth Retry
		{"error_004", "skill_004"}, // File not found -> File Recovery
		{"error_005", "skill_005"}, // Memory error -> Memory Cleanup
	}

	// Add edges for relationships
	for i, rel := range relationships {
		edgeID := fmt.Sprintf("edge_%03d", i+1)
		edgeData := map[string]interface{}{
			"id":     edgeID,
			"from":   rel.ErrorID,
			"to":     rel.SkillID,
			"type":   "handles",
			"weight": 1.0,
		}

		data, err := json.Marshal(edgeData)
		if err != nil {
			return fmt.Errorf("failed to marshal edge %s: %w", edgeID, err)
		}

		if err := app.storage.PutEdge(edgeID, data); err != nil {
			return fmt.Errorf("failed to store edge %s: %w", edgeID, err)
		}
	}

	app.logger.Info("Successfully pre-populated test data",
		zap.Int("error_nodes", len(errorNodes)),
		zap.Int("skill_nodes", len(skillNodes)),
		zap.Int("relationships", len(relationships)))

	return nil
}

func (app *App) Start(ctx context.Context) error {
	app.logger.Info("Starting GraphChain application with NRV system and economics")

	// Start NRV system
	if err := app.nrvSystem.Start(); err != nil {
		return fmt.Errorf("failed to start NRV system: %w", err)
	}

	// Start NRN integration
	if app.nrnIntegration != nil {
		if err := app.nrnIntegration.Start(ctx); err != nil {
			app.logger.Warn("Failed to start NRN integration", zap.Error(err))
		}
	}

	// Start RPC server
	if err := app.rpc.Start(ctx); err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	select {
	case <-c:
		app.logger.Info("Received interrupt signal, shutting down...")
		return app.Stop(ctx)
	case <-ctx.Done():
		return app.Stop(ctx)
	}
}

func (app *App) Stop(ctx context.Context) error {
	app.logger.Info("Stopping GraphChain application")

	// Stop RPC server
	if err := app.rpc.Stop(ctx); err != nil {
		app.logger.Error("Failed to stop RPC server", zap.Error(err))
	}

	// Stop NRV system
	if err := app.nrvSystem.Stop(); err != nil {
		app.logger.Error("Failed to stop NRV system", zap.Error(err))
	}

	// Close storage
	if err := app.storage.Close(); err != nil {
		app.logger.Error("Failed to close storage", zap.Error(err))
	}

	return nil
}
