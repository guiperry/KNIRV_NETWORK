package embed

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/runtime"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/server"
	"go.uber.org/zap"
)

// GatewayConfig represents the configuration for the embedded gateway
type GatewayConfig struct {
	Port              int    `json:"port"`
	ChainID           string `json:"chainID"`
	Mode              string `json:"mode"`
	OracleOwnerKey    string `json:"oracleOwnerKey"`
	AutoOpenBrowser   bool   `json:"autoOpenBrowser"`
	EnableOracle      bool   `json:"enableOracle"`
	EnableModelServer bool   `json:"enableModelServer"`
}

// DefaultConfig returns a default configuration for the embedded gateway
func DefaultConfig() *GatewayConfig {
	return &GatewayConfig{
		Port:              8080,
		ChainID:           "testnet",
		Mode:              "gateway_nest",
		OracleOwnerKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		AutoOpenBrowser:   false,
		EnableOracle:      true,
		EnableModelServer: true,
	}
}

// Gateway represents an embedded KNIRVGATEWAY instance
type Gateway struct {
	cfg     *GatewayConfig
	logger  *zap.Logger
	runtime *runtime.Runtime
	server  *server.Server
	oracle  interface{} // Can be *oracle.Oracle or *os.Process
	stopCh  chan struct{}
}

// NewGateway creates a new embedded gateway instance
func NewGateway(cfg *GatewayConfig, logger *zap.Logger) (*Gateway, error) {
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

	rt, err := runtime.NewRuntime(logger)
	if err != nil {
		return nil, err
	}

	return &Gateway{
		cfg:     cfg,
		logger:  logger,
		runtime: rt,
		stopCh:  make(chan struct{}),
	}, nil
}

// Start initializes and starts the embedded gateway
func (g *Gateway) Start() error {
	g.logger.Info("Starting embedded KNIRVGATEWAY",
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

	g.logger.Info("Embedded KNIRVGATEWAY started successfully")

	// Start signal handler
	go g.handleSignals()

	return nil
}

func (g *Gateway) startOracle(cfg *oracle.OracleConfig) error {
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

func (g *Gateway) handleSignals() {
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

// Stop gracefully stops the embedded gateway
func (g *Gateway) Stop() error {
	g.logger.Info("Stopping embedded KNIRVGATEWAY")

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

	g.logger.Info("Embedded KNIRVGATEWAY stopped")
	return nil
}

// GetRuntime returns the runtime instance
func (g *Gateway) GetRuntime() *runtime.Runtime {
	return g.runtime
}

// GetServer returns the server instance
func (g *Gateway) GetServer() *server.Server {
	return g.server
}

// GetPort returns the configured port
func (g *Gateway) GetPort() int {
	return g.cfg.Port
}

// GetChainID returns the configured chain ID
func (g *Gateway) GetChainID() string {
	return g.cfg.ChainID
}

// IsRunning returns true if the gateway is running
func (g *Gateway) IsRunning() bool {
	select {
	case <-g.stopCh:
		return false
	default:
		return true
	}
}
