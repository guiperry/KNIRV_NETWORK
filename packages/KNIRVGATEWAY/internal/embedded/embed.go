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

// Embed only build artifacts — not entire source directories.
// webgui: Next.js static export (out/)
//
//go:embed all:webgui/out
var WebGUIFS embed.FS

// GatewayConfig represents the configuration for the embedded gateway runner.
// The oracle itself has moved to KNIRVSERVER and is no longer managed here.
type GatewayConfig struct {
	Port            int    `json:"port"`
	ChainID         string `json:"chainID"`
	Mode            string `json:"mode"`
	AutoOpenBrowser bool   `json:"autoOpenBrowser"`
}

// DefaultConfig returns a default configuration for the embedded gateway runner.
func DefaultConfig() *GatewayConfig {
	return &GatewayConfig{
		Port:            8080,
		ChainID:         "testnet",
		Mode:            "oracle_nest",
		AutoOpenBrowser: false,
	}
}

type Gateway struct {
	cfg     *GatewayConfig
	logger  *zap.Logger
	runtime *runtime.Runtime
	server  *server.Server
	stopCh  chan struct{}
}

// NewGateway creates a new gateway runner.
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

	rt, err := runtime.NewRuntime(logger, WebGUIFS, nil)
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

// Start initialises and starts the  gateway.
func (g *Gateway) Start() error {
	g.logger.Info("Starting KNIRVGATEWAY",
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

	g.server, err = server.New(coreCfg, g.runtime.GetWebGUIStaticPath(), g.logger)
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

// Stop gracefully stops the embedded gateway.
func (g *Gateway) Stop() error {
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
func (g *Gateway) GetRuntime() *runtime.Runtime {
	return g.runtime
}

// GetServer returns the server instance.
func (g *Gateway) GetServer() *server.Server {
	return g.server
}

// GetPort returns the configured port.
func (g *Gateway) GetPort() int {
	return g.cfg.Port
}

// GetChainID returns the configured chain ID.
func (g *Gateway) GetChainID() string {
	return g.cfg.ChainID
}

// IsRunning returns true if the gateway is running.
func (g *Gateway) IsRunning() bool {
	select {
	case <-g.stopCh:
		return false
	default:
		return true
	}
}
