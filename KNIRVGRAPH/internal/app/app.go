package app

import (
	"blockchain-app/internal/graphchain"
	"blockchain-app/internal/network"
	"blockchain-app/internal/nrv"
	"blockchain-app/internal/storage"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

type App struct {
	graphchain *graphchain.GraphChain
	nrvSystem  *nrv.NRVSystem
	rpc        *network.RPCServer
	storage    storage.Storage
	logger     *zap.Logger
}

func NewApp(homeDir string, rpcPort int) (*App, error) {
	logger, _ := zap.NewProduction()

	// Initialize BluntDB storage
	storage, err := storage.NewBluntDBStorage(fmt.Sprintf("%s/data", homeDir))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize BluntDB storage: %w", err)
	}

	// Initialize GraphChain
	gc := graphchain.NewGraphChain(storage)

	// Initialize NRV system
	nrvSystem := nrv.NewNRVSystem("local-peer", nil)

	// Initialize RPC server with NRV system
	rpc := network.NewRPCServerWithNRV(gc, nrvSystem, logger, rpcPort)

	return &App{
		graphchain: gc,
		nrvSystem:  nrvSystem,
		rpc:        rpc,
		storage:    storage,
		logger:     logger,
	}, nil
}

func (app *App) Start(ctx context.Context) error {
	app.logger.Info("Starting GraphChain application with NRV system")

	// Start NRV system
	if err := app.nrvSystem.Start(); err != nil {
		return fmt.Errorf("failed to start NRV system: %w", err)
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
