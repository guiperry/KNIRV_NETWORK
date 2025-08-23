package services

import (
	"crypto-wallet-backend/internal/config"
	"crypto-wallet-backend/pkg/logger"

	"gorm.io/gorm"
)

type Container struct {
	DB     *gorm.DB
	Config *config.Config
	Logger *logger.Logger

	// Services
	MultichainWalletService *MultichainWalletService
	WalletSyncService       *WalletSyncService
	AIAgentService          *AIAgentService
}

func NewContainer(db *gorm.DB, cfg *config.Config, logger *logger.Logger) *Container {
	container := &Container{
		DB:     db,
		Config: cfg,
		Logger: logger,
	}

	// Initialize services
	// container.SecurityService = NewSecurityService(cfg, logger)
	// container.UserService = NewUserService(db, container.SecurityService, logger)
	// container.WalletService = NewWalletService(db, container.SecurityService, logger)
	// container.BlockchainService = NewBlockchainService(cfg, logger)
	// container.MarketDataService = NewMarketDataService(cfg, logger)
	// container.NotificationService = NewNotificationService(cfg, logger)

	// Initialize multichain wallet service
	container.MultichainWalletService = NewMultichainWalletService(container)

	// Initialize wallet sync service
	container.WalletSyncService = NewWalletSyncService(container)

	// Initialize AI agent service
	container.AIAgentService = NewAIAgentService(db, cfg, logger)

	return container
}
