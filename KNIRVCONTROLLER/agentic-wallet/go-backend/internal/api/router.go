package api

import (
	"crypto-wallet-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func NewRouter(container *services.Container) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	// TODO: Add CORS and security headers middleware

	// Initialize handlers
	// authHandler := handlers.NewAuthHandler(container.UserService, container.SecurityService)
	// walletHandler := handlers.NewWalletHandler(container.WalletService)
	// agentHandler := handlers.NewAIAgentHandler(container.AIAgentService)
	// marketHandler := handlers.NewMarketDataHandler(container.MarketDataService)
	multichainWalletHandler := NewMultichainWalletHandler(container.MultichainWalletService)
	walletSyncHandler := NewWalletSyncHandler(container.WalletSyncService)

	// Public routes
	public := router.Group("/api/v1")
	{
		// Multichain wallet public routes
		public.GET("/multichain/chains", multichainWalletHandler.GetSupportedChains)
		public.POST("/multichain/mnemonic/generate", multichainWalletHandler.GenerateMnemonic)
		public.POST("/multichain/wallet/generate/:chain", multichainWalletHandler.GenerateWalletForChain)
	}

	// Protected routes (commented out auth middleware for now)
	protected := router.Group("/api/v1")
	// protected.Use(middleware.AuthRequired(container.SecurityService))
	{
		// Multichain wallet routes
		protected.POST("/multichain/wallet/create", multichainWalletHandler.CreateMultichainWallet)
		protected.POST("/multichain/wallet/import", multichainWalletHandler.ImportWallet)
		protected.GET("/multichain/balance/:chain/:address", multichainWalletHandler.GetWalletBalance)

		// Wallet synchronization routes
		protected.POST("/sync/session/create", walletSyncHandler.CreateSyncSession)
		protected.POST("/sync/browser/connect", walletSyncHandler.ConnectBrowserWallet)
		protected.POST("/sync/wallets", walletSyncHandler.SyncWallets)
		protected.POST("/sync/transactions", walletSyncHandler.SyncTransactions)
		protected.GET("/sync/session/:session_id", walletSyncHandler.GetSyncSession)
		protected.GET("/sync/ws", walletSyncHandler.HandleWebSocket)
		protected.POST("/sync/cleanup", walletSyncHandler.CleanupExpiredSessions)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return router
}
