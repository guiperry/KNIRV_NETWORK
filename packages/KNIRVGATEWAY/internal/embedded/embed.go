package embedded

import (
	"context"
	"embed"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/runtime"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/server"
	"go.uber.org/zap"
)

// Embed all service directories (excluding node_modules, build artifacts)
//
//go:embed all:webgui
var WebGUIFS embed.FS

//go:embed all:network-website/*
var NetworkWebsiteFS embed.FS

//go:embed knirv-oracle
var OracleBinary []byte

// OracleConfig represents the configuration for the embedded gateway runner.
// The oracle itself has moved to KNIRVSERVER and is no longer managed here.
type OracleConfig struct {
	Port            int    `json:"port"`
	ChainID         string `json:"chainID"`
	Mode            string `json:"mode"`
	AutoOpenBrowser bool   `json:"autoOpenBrowser"`
}

// DefaultConfig returns a default configuration for the embedded gateway runner.
func DefaultConfig() *OracleConfig {
	return &OracleConfig{
		Port:            8080,
		ChainID:         "testnet",
		Mode:            "oracle_nest",
		AutoOpenBrowser: false,
	}
}

// Oracle is the embedded KNIRVGATEWAY runner (name retained for API compatibility).
// It no longer manages an oracle process; the oracle runs inside KNIRVSERVER.
type Oracle struct {
	cfg     *OracleConfig
	logger  *zap.Logger
	runtime *runtime.Runtime
	server  *server.Server
	stopCh  chan struct{}
}

// NewOracle creates a new embedded gateway runner.
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

// Start initialises and starts the embedded gateway.
func (g *Oracle) Start() error {
	g.logger.Info("Starting embedded KNIRVGATEWAY",
		zap.String("mode", g.cfg.Mode),
		zap.Int("port", g.cfg.Port),
		zap.String("chainID", g.cfg.ChainID),
	)

	if err := g.runtime.Setup(); err != nil {
		return err
	}

	os.Setenv("KNIRV_MODE", g.cfg.Mode)
	os.Setenv("CHAIN_ID", g.cfg.ChainID)

	coreCfg, err := config.Load()
	if err != nil {
		return err
	}

	g.server, err = server.New(coreCfg, g.runtime.GetWebGUIStaticPath(), g.runtime.GetNetworkWebsitePath(), g.logger)
	if err != nil {
		return err
	}

	go func() {
		g.logger.Info("Starting HTTP server", zap.Int("port", g.cfg.Port))
		if err := g.server.Start(); err != nil {
			g.logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	time.Sleep(2 * time.Second)

	g.logger.Info("Embedded KNIRVGATEWAY started successfully")

	go g.handleSignals()

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

// Stop gracefully stops the embedded gateway.
func (g *Oracle) Stop() error {
	g.logger.Info("Stopping embedded KNIRVGATEWAY")

	close(g.stopCh)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if g.server != nil {
		if err := g.server.Stop(ctx); err != nil {
			g.logger.Error("Error stopping server", zap.Error(err))
		}
	}

	if err := g.runtime.Cleanup(); err != nil {
		g.logger.Error("Failed to cleanup runtime", zap.Error(err))
	}

	g.logger.Info("Embedded KNIRVGATEWAY stopped")
	return nil
}

// GetRuntime returns the runtime instance.
func (g *Oracle) GetRuntime() *runtime.Runtime {
	return g.runtime
}

// GetServer returns the server instance.
func (g *Oracle) GetServer() *server.Server {
	return g.server
}

// GetPort returns the configured port.
func (g *Oracle) GetPort() int {
	return g.cfg.Port
}

// GetChainID returns the configured chain ID.
func (g *Oracle) GetChainID() string {
	return g.cfg.ChainID
}

// IsRunning returns true if the gateway is running.
func (g *Oracle) IsRunning() bool {
	select {
	case <-g.stopCh:
		return false
	default:
		return true
	}
}
