package app

import (
	"KNIRVGRAPH/internal/economics"
	"KNIRVGRAPH/internal/graphchain"
	"KNIRVGRAPH/internal/network"
	"KNIRVGRAPH/internal/nrv"
	"KNIRVGRAPH/internal/dht"
	"KNIRVGRAPH/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func getOracleSocketPath() string {
	if envPath := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); envPath != "" {
		return filepath.Join(envPath, "sockets", "oracle.sock")
	}
	return "/var/lib/knirvserver/sockets/oracle.sock"
}

func resolveKNIRVOracleURL(logger *zap.Logger) string {
	type candidate struct {
		baseURL     string
		healthPaths []string
		label       string
	}

	// ── priority 1: explicit env vars ──────────────────────────────────────
	var candidates []candidate
	if envURL := os.Getenv("KNIRV_ORACLED_RPC_URL"); envURL != "" {
		candidates = append(candidates, candidate{baseURL: envURL, healthPaths: []string{"/oracle/v3/health", "/health"}, label: "env var KNIRV_ORACLED_RPC_URL"})
	}
	if envURL := os.Getenv("KNIRVORACLE_URL"); envURL != "" {
		candidates = append(candidates, candidate{baseURL: envURL, healthPaths: []string{"/oracle/v3/health", "/health"}, label: "env var KNIRVORACLE_URL"})
	}

	// ── priority 2: public DNS (oracle.knirv.network) ─────────────────────
	// Tried before local-only fallbacks so the DNS record created by the
	// KNIRVSERVER DNS service is selected first when reachable.
	candidates = append(candidates, candidate{
		baseURL: "https://oracle.knirv.network", healthPaths: []string{"/oracle/v3/health", "/health"}, label: "cloudflare public DNS",
	})

	// ── priority 3: internal Unix socket ──────────────────────────────────
	// When the KNIRVORACLE subprocess runs on the same host, it binds a Unix
	// socket that is faster and more secure than TCP loopback.
	socketPath := getOracleSocketPath()
	if _, err := os.Stat(socketPath); err == nil {
		candidates = append(candidates, candidate{
			baseURL: "http://localhost", healthPaths: []string{"/oracle/v3/health", "/health"}, label: "unix socket: " + socketPath,
		})
	}

	// ── priority 4: local TCP fallbacks ───────────────────────────────────
	// Fall back to loopback ports for legacy/sidecar configurations.
	candidates = append(candidates,
		candidate{baseURL: "http://127.0.0.1:8084", healthPaths: []string{"/oracle/v3/health", "/health"}, label: "local KNIRVSERVER oracle proxy"},
		candidate{baseURL: "http://127.0.0.1:1317", healthPaths: []string{"/health"}, label: "local legacy KNIRVSERVER oracle"},
	)

	// ── probe each candidate ──────────────────────────────────────────────
	client := &http.Client{Timeout: 5 * time.Second}
	for _, c := range candidates {
		if c.baseURL == "" {
			continue
		}
		for _, healthPath := range c.healthPaths {
			var resp *http.Response
			var err error

			if strings.HasPrefix(c.label, "unix socket:") {
				// Unix socket requires a custom transport — the default
				// http.Client cannot dial unix:// URIs.
				socketClient := &http.Client{
					Timeout: 5 * time.Second,
					Transport: &http.Transport{
						DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
							return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
						},
					},
				}
				resp, err = socketClient.Get(c.baseURL + healthPath)
			} else {
				resp, err = client.Get(c.baseURL + healthPath)
			}
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				if strings.HasPrefix(c.label, "unix socket:") {
					logger.Info("KNIRVORACLE: using discovered endpoint", zap.String("url", "unix://"+socketPath), zap.String("source", c.label))
					return "unix://" + socketPath
				}
				logger.Info("KNIRVORACLE: using discovered endpoint", zap.String("url", c.baseURL), zap.String("source", c.label))
				return c.baseURL
			}
		}
	}

	logger.Info("KNIRVORACLE: no compatible endpoint discovered, economics integration disabled")
	return ""
}

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
	Testnet    TestnetConfig    `json:"testnet"`
	Node       NodeConfig       `json:"node"`
	Network    NetworkConfig    `json:"network"`
	Storage    StorageConfig    `json:"storage"`
	DHT        DhtConfig        `json:"dht"`
	DRQ        DrqConfig        `json:"drq"`
	Clustering ClusteringConfig `json:"clustering"`
	Topology   TopologyConfig   `json:"topology"`
	Validation ValidationConfig `json:"validation"`
}

// NetworkConfig holds network-specific configuration
type NetworkConfig struct {
	P2PPort    int    `json:"p2p_port"`
	RPCPort    int    `json:"rpc_port"`
	APIPort    int    `json:"api_port"`
	SocketPath string `json:"socket_path"`
}

// StorageConfig holds storage-specific configuration
type StorageConfig struct {
	DBType      string `json:"db_type"`
	Path        string `json:"path"`
	CacheSizeGB int    `json:"cache_size_gb"`
}

// DhtConfig holds DHT-specific configuration
type DhtConfig struct {
	Enabled        bool     `json:"enabled"`
	BootstrapPeers []string `json:"bootstrap_peers"`
}

// DrqConfig holds DRQ-specific configuration
type DrqConfig struct {
	Enabled        bool    `json:"enabled"`
	HistoryLength  int     `json:"history_length"`
	LearningRate   float64 `json:"learning_rate"`
	DiscountFactor float64 `json:"discount_factor"`
	SyncInterval   string  `json:"sync_interval"` // e.g., "10s"
}

// ClusteringConfig holds clustering-specific configuration
type ClusteringConfig struct {
	EmbeddingModel      string  `json:"embedding_model"` // e.g., "bert_base_768"
	SimilarityThreshold float64 `json:"similarity_threshold"`
	MaxClusterSize      int     `json:"max_cluster_size"`
	SpatialIndex        string  `json:"spatial_index"` // e.g., "kdtree"
}

// TopologyConfig holds topology-specific configuration
type TopologyConfig struct {
	MinDegree          int     `json:"min_degree"`
	ScalingExponent    float64 `json:"scaling_exponent"`
	RewireProbability  float64 `json:"rewire_probability"`
	PageRankIterations int     `json:"pagerank_iterations"`
}

// ValidationConfig holds validation-specific configuration
type ValidationConfig struct {
	DVEClientEnabled bool   `json:"dve_client_enabled"`
	MinAttestations  int    `json:"min_attestations"`
	Timeout          string `json:"timeout"` // e.g., "300s"
}

// App represents the main GraphChain application
type App struct {
	graphchain      *graphchain.GraphChain
	nrvSystem       *nrv.NRVSystem
	nrnIntegration  *economics.NRNIntegration
	proofOfSolution *economics.ProofOfSolution
	rpc             *network.RPCServer
	storage         storage.GraphStorage
	dhtManager      *dht.DHTClientAdapter
	logger          *zap.Logger
	config          *Config
}

// NewApp creates a new GraphChain application instance
func NewApp(homeDir string, rpcPort int, enableAutoRelay bool) (*App, error) {
	logger, _ := zap.NewProduction()

	// Initialize default config
	config := &Config{
		Testnet: TestnetConfig{
			Enabled:     false,
			InMemory:    false,
			PrePopulate: false,
			MaxNodes:    1000,
			ChainID:     "knirvgraph-1",
			Port:        rpcPort,
			LocalMode:   true,
		},
		Node: GetDefaultNodeConfig(NODE_FULL), // Use default full node config
		Network: NetworkConfig{
			P2PPort: 9001, // Default libp2p port
			RPCPort: rpcPort,
			APIPort: 1317,
		},
		Storage: StorageConfig{
			DBType:      "bluntdb",
			Path:        fmt.Sprintf("%s/data", homeDir),
			CacheSizeGB: 16,
		},
		DHT: DhtConfig{
			Enabled:        true,
			BootstrapPeers: []string{os.Getenv("KNIRV_BOOTSTRAP_PEER_1"), os.Getenv("KNIRV_BOOTSTRAP_PEER_2"), os.Getenv("KNIRV_BOOTSTRAP_PEER_3")},
		},
		DRQ: DrqConfig{
			Enabled:        true,
			HistoryLength:  3,
			LearningRate:   0.01,
			DiscountFactor: 0.95,
			SyncInterval:   "10s",
		},
		Clustering: ClusteringConfig{
			EmbeddingModel:      "bert_base_768",
			SimilarityThreshold: 0.85,
			MaxClusterSize:      100,
			SpatialIndex:        "kdtree",
		},
		Topology: TopologyConfig{
			MinDegree:          3,
			ScalingExponent:    2.7,
			RewireProbability:  0.01,
			PageRankIterations: 100,
		},
		Validation: ValidationConfig{
			DVEClientEnabled: true,
			MinAttestations:  5,
			Timeout:          "300s",
		},
	}

	// Initialize BluntDB storage
	storageInstance, err := storage.NewBluntDBStorage(config.Storage.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize BluntDB storage: %w", err)
	}

	// Initialize GraphChain
	gc := graphchain.NewGraphChain(storageInstance)

	// Initialize NRV system
	nrvSystem := nrv.NewNRVSystem("local-peer", nil)

	// Initialize NRN integration (local-only — no oracle dependency)
	nrnIntegration := economics.NewNRNIntegration()

	// Initialize Proof-of-Solution
	proofOfSolution := economics.NewProofOfSolution(nrnIntegration, nrvSystem)

	// Initialize RPC server with NRV system and economics (will set app reference later)
	var rpc *network.RPCServer

	// Initialize DHT client adapter (connects to KNIRVGATEWAY)
	dhtAdapter, err := dht.NewDHTClientAdapter(
		fmt.Sprintf("knirvgraph-%s", config.Testnet.ChainID),
		config.Testnet.ChainID,
		config.DHT.BootstrapPeers,
		enableAutoRelay,
	)
	if err != nil {
		logger.Warn("Failed to initialize DHT client", zap.Error(err))
		dhtAdapter = nil
	}

	app := &App{
		graphchain:      gc,
		nrvSystem:       nrvSystem,
		nrnIntegration:  nrnIntegration,
		proofOfSolution: proofOfSolution,
		storage:         storageInstance,
		dhtManager:      dhtAdapter,
		logger:          logger,
		config:          config,
	}

	// Initialize RPC server with app reference
	rpc = network.NewRPCServerWithEconomics(gc, nrvSystem, nrnIntegration, proofOfSolution, app, logger, config.Network.RPCPort, config.Network.SocketPath)
	app.rpc = rpc

	return app, nil
}

// GetConfig returns the application configuration
func (app *App) GetConfig() *Config {
	return app.config
}

// NewAppWithConfig creates a new App instance with optional configuration
func NewAppWithConfig(homeDir string, rpcPort int, appConfig *Config, enableAutoRelay bool, socketPath string) (*App, error) {
	logger, _ := zap.NewProduction()

	// Use provided config or initialize default
	config := appConfig
	if config == nil {
		config = &Config{
			Testnet: TestnetConfig{
				Enabled:     false,
				InMemory:    false,
				PrePopulate: false,
				MaxNodes:    1000,
				ChainID:     "knirvgraph-1",
				Port:        rpcPort,
				LocalMode:   true,
			},
			Node: GetDefaultNodeConfig(NODE_FULL), // Use default full node config
			Network: NetworkConfig{
				P2PPort:    9001, // Default libp2p port
				RPCPort:    rpcPort,
				APIPort:    1317,
				SocketPath: socketPath,
			},
			Storage: StorageConfig{
				DBType:      "bluntdb",
				Path:        fmt.Sprintf("%s/data", homeDir),
				CacheSizeGB: 16,
			},
			DHT: DhtConfig{
				Enabled:        true,
				BootstrapPeers: []string{os.Getenv("KNIRV_BOOTSTRAP_PEER_1"), os.Getenv("KNIRV_BOOTSTRAP_PEER_2"), os.Getenv("KNIRV_BOOTSTRAP_PEER_3")},
			},
			DRQ: DrqConfig{
				Enabled:        true,
				HistoryLength:  3,
				LearningRate:   0.01,
				DiscountFactor: 0.95,
				SyncInterval:   "10s",
			},
			Clustering: ClusteringConfig{
				EmbeddingModel:      "bert_base_768",
				SimilarityThreshold: 0.85,
				MaxClusterSize:      100,
				SpatialIndex:        "kdtree",
			},
			Topology: TopologyConfig{
				MinDegree:          3,
				ScalingExponent:    2.7,
				RewireProbability:  0.01,
				PageRankIterations: 100,
			},
			Validation: ValidationConfig{
				DVEClientEnabled: true,
				MinAttestations:  5,
				Timeout:          "300s",
			},
		}
	} else {
		// Ensure critical fields are initialized even when config is provided
		if config.Storage.Path == "" {
			config.Storage.Path = fmt.Sprintf("%s/data", homeDir)
		}
		if config.Storage.DBType == "" {
			config.Storage.DBType = "bluntdb"
		}
		if config.Storage.CacheSizeGB == 0 {
			config.Storage.CacheSizeGB = 16
		}
		if config.Network.RPCPort == 0 {
			config.Network.RPCPort = rpcPort
		}
		if config.Network.SocketPath == "" && socketPath != "" {
			config.Network.SocketPath = socketPath
		}
	}

	var storageInstance storage.GraphStorage
	var err error

	// Use in-memory storage for testnet if configured
	if config.Testnet.Enabled && config.Testnet.InMemory {
		logger.Info("Using in-memory storage for testnet")
		storageInstance, err = storage.NewMemoryStorage()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize memory storage: %w", err)
		}
	} else {
		// Initialize BluntDB storage
		storageInstance, err = storage.NewBluntDBStorage(config.Storage.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize BluntDB storage: %w", err)
		}
	}

	// Initialize GraphChain
	gc := graphchain.NewGraphChain(storageInstance)

	// Initialize NRV system
	nrvSystem := nrv.NewNRVSystem("local-peer", nil)

	// Initialize NRN integration (local-only — no oracle dependency)
	nrnIntegration := economics.NewNRNIntegration()

	// Initialize Proof-of-Solution
	proofOfSolution := economics.NewProofOfSolution(nrnIntegration, nrvSystem)

	// Initialize RPC server with NRV system and economics (will set app reference later)
	var rpc *network.RPCServer

	// Initialize DHT client adapter (connects to KNIRVGATEWAY)
	dhtAdapter, err := dht.NewDHTClientAdapter(
		fmt.Sprintf("knirvgraph-%s", config.Testnet.ChainID),
		config.Testnet.ChainID,
		config.DHT.BootstrapPeers,
		enableAutoRelay,
	)
	if err != nil {
		logger.Warn("Failed to initialize DHT client", zap.Error(err))
		dhtAdapter = nil
	}

	app := &App{
		graphchain:      gc,
		nrvSystem:       nrvSystem,
		nrnIntegration:  nrnIntegration,
		proofOfSolution: proofOfSolution,
		storage:         storageInstance,
		dhtManager:      dhtAdapter,
		logger:          logger,
		config:          config,
	}

	// Initialize RPC server with app reference
	rpc = network.NewRPCServerWithEconomics(gc, nrvSystem, nrnIntegration, proofOfSolution, app, logger, config.Network.RPCPort, config.Network.SocketPath)
	app.rpc = rpc

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
		app.nrnIntegration.Start()
	}

	// Start DHT manager
	if app.dhtManager != nil {
		if err := app.dhtManager.Start(); err != nil {
			app.logger.Warn("Failed to start DHT manager", zap.Error(err))
		} else {
			app.logger.Info("DHT manager started successfully")
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

	// Stop DHT manager
	if app.dhtManager != nil {
		app.dhtManager.Stop()
		app.logger.Info("DHT manager stopped")
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

// AnnounceSkill announces a new skill minted on the Graph via DHT
func (app *App) AnnounceSkill(skillID, name, description, category string, metadata map[string]string) error {
	if app.dhtManager == nil {
		app.logger.Warn("DHT manager not available, cannot announce skill")
		return fmt.Errorf("DHT manager not available")
	}

	return app.dhtManager.AnnounceSkill(skillID, name, description, category, metadata)
}

// AnnounceCapability announces a new capability minted on the Graph via DHT
func (app *App) AnnounceCapability(capabilityID, name, description string, schema interface{}, metadata map[string]string) error {
	if app.dhtManager == nil {
		app.logger.Warn("DHT manager not available, cannot announce capability")
		return fmt.Errorf("DHT manager not available")
	}

	return app.dhtManager.AnnounceCapability(capabilityID, name, description, schema, metadata)
}

// AnnounceProperty announces a new property minted on the Graph via DHT
func (app *App) AnnounceProperty(propertyID, name, propertyType string, value interface{}, metadata map[string]string) error {
	if app.dhtManager == nil {
		app.logger.Warn("DHT manager not available, cannot announce property")
		return fmt.Errorf("DHT manager not available")
	}

	return app.dhtManager.AnnounceProperty(propertyID, name, propertyType, value, metadata)
}

// IsNetworkPaused returns whether the network is currently paused
func (app *App) IsNetworkPaused() bool {
	if app.dhtManager == nil {
		return false
	}

	return app.dhtManager.IsNetworkPaused()
}
