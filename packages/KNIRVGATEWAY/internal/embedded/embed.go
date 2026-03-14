package embedded

import (
	"embed"
	"os"

	"go.uber.org/zap"

	"context"

	"os/signal"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/runtime"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/server"
)

// Embed all service directories (excluding node_modules, build artifacts)
//
//go:embed all:webgui
var WebGUIFS embed.FS

//go:embed all:network-website/*
var NetworkWebsiteFS embed.FS

//go:embed knirv-oracle
var OracleBinary []byte

// OracleConfig represents the configuration for the embedded oracle
type OracleConfig struct {
	Port              int    `json:"port"`
	ChainID           string `json:"chainID"`
	Mode              string `json:"mode"`
	OracleOwnerKey    string `json:"oracleOwnerKey"`
	AutoOpenBrowser   bool   `json:"autoOpenBrowser"`
	EnableOracle      bool   `json:"enableOracle"`
	EnableModelServer bool   `json:"enableModelServer"`
}

// DefaultConfig returns a default configuration for the embedded oracle
func DefaultConfig() *OracleConfig {
	return &OracleConfig{
		Port:              8080,
		ChainID:           "testnet",
		Mode:              "oracle_nest",
		OracleOwnerKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		AutoOpenBrowser:   false,
		EnableOracle:      true,
		EnableModelServer: true,
	}
}

// Oracle represents an embedded KNIRVORACLE instance
type Oracle struct {
	cfg     *OracleConfig
	logger  *zap.Logger
	runtime *runtime.Runtime
	server  *server.Server
	oracle  interface{} // Can be *oracle.Oracle or *os.Process
	stopCh  chan struct{}
}

// NewOracle creates a new embedded oracle instance
func NewOracle(cfg *OracleConfig, logger *zap.Logger) (*Oracle, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if logger == nil {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	}

	rt, err := runtime.NewRuntime(logger, WebGUIFS, NetworkWebsiteFS, OracleBinary)
	if err != nil {
		return nil, err
	}

	return &Oracle{
		cfg:     cfg,
		logger:  logger,
		runtime: rt,
		stopCh:  make(chan struct{}),
	}, nil
}

// Start initializes and starts the embedded oracle
func (g *Oracle) Start() error {
	g.logger.Info("Starting embedded KNIRVORACLE",
		zap.String("mode", g.cfg.Mode),
		zap.Int("port", g.cfg.Port),
		zap.String("chainID", g.cfg.ChainID),
	)

	// Setup runtime
	if err := g.runtime.Setup(); err != nil {
		return err
	}

	// Set environment variables
	os.Setenv("KNIRV_MODE", g.cfg.Mode)
	os.Setenv("PORT", string(rune(g.cfg.Port)))
	os.Setenv("CHAIN_ID", g.cfg.ChainID)
	os.Setenv("ORACLE_OWNER_KEY", g.cfg.OracleOwnerKey)
	os.Setenv("ENABLE_ORACLE", "true")
	os.Setenv("ENABLE_MODEL_SERVER", "true")

	// Initialize oracle configuration
	oracleCfg, err := oracle.LoadConfigFromEnv()
	if err != nil {
		return err
	}

	// Start oracle
	if g.cfg.EnableOracle {
		if err := g.startOracle(oracleCfg); err != nil {
			return err
		}
	}

	// Initialize HTTP server
	coreCfg, err := config.Load()
	if err != nil {
		return err
	}

	g.server, err = server.New(coreCfg, g.runtime.GetWebGUIStaticPath(), g.runtime.GetNetworkWebsitePath(), g.logger)
	if err != nil {
		return err
	}

	// Start server
	go func() {
		g.logger.Info("Starting HTTP server", zap.Int("port", g.cfg.Port))
		if err := g.server.Start(); err != nil {
			g.logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	g.logger.Info("Embedded KNIRVORACLE started successfully")

	// Start signal handler
	go g.handleSignals()

	return nil
}

func (g *Oracle) startOracle(cfg *oracle.OracleConfig) error {
	// Check if oracle binary exists and owner key is set
	oracleBinaryPath := g.runtime.GetOracleBinaryPath()
	if _, err := os.Stat(oracleBinaryPath); err == nil && cfg.OwnerPrivateKey != "" {
		// Binary exists and owner key is set, start as separate process
		g.logger.Info("Starting knirv-oracle as separate process...", zap.String("path", oracleBinaryPath))
		// TODO: Implement process management
	} else {
		// Binary doesn't exist or owner key not set, initialize directly
		g.logger.Warn("knirv-oracle binary not found or owner key not set, initializing directly...")

		// Validate oracle configuration
		if err := oracle.ValidateConfig(cfg); err != nil {
			return err
		}

		g.logger.Info("Oracle configuration summary", zap.String("config", oracle.ConfigSummary(cfg)))

		// Create oracle instance
		oracleNode, err := oracle.NewOracle(cfg, g.logger)
		if err != nil {
			return err
		}

		g.oracle = oracleNode

		// Start oracle services
		go func() {
			g.logger.Info("Starting knirv-oracle services...")
			if err := oracleNode.Start(); err != nil {
				g.logger.Fatal("Oracle node failed to start", zap.Error(err))
			}
		}()
	}

	return nil
}

func (g *Oracle) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		g.logger.Info("Received shutdown signal")
		g.Stop()
	case <-g.stopCh:
		g.logger.Info("Received stop signal")
	}
}

// Stop gracefully stops the embedded oracle
func (g *Oracle) Stop() error {
	g.logger.Info("Stopping embedded KNIRVORACLE")

	// Close stop channel
	close(g.stopCh)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if g.server != nil {
		if err := g.server.Stop(ctx); err != nil {
			g.logger.Error("Error stopping server", zap.Error(err))
		}
	}

	// Stop oracle
	if g.oracle != nil {
		switch oracleInstance := g.oracle.(type) {
		case *oracle.Oracle:
			if err := oracleInstance.Stop(); err != nil {
				g.logger.Error("Error stopping oracle node", zap.Error(err))
			}
		case *os.Process:
			// TODO: Implement process termination
		}
	}

	// Cleanup runtime
	if err := g.runtime.Cleanup(); err != nil {
		g.logger.Error("Failed to cleanup runtime", zap.Error(err))
	}

	g.logger.Info("Embedded KNIRVORACLE stopped")
	return nil
}

// GetRuntime returns the runtime instance
func (g *Oracle) GetRuntime() *runtime.Runtime {
	return g.runtime
}

// GetServer returns the server instance
func (g *Oracle) GetServer() *server.Server {
	return g.server
}

// GetPort returns the configured port
func (g *Oracle) GetPort() int {
	return g.cfg.Port
}

// GetChainID returns the configured chain ID
func (g *Oracle) GetChainID() string {
	return g.cfg.ChainID
}

// IsRunning returns true if the oracle is running
func (g *Oracle) IsRunning() bool {
	select {
	case <-g.stopCh:
		return false
	default:
		return true
	}
}
