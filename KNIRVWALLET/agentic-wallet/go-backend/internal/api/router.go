package api

import (
	"crypto-wallet-backend/internal/api/handlers"
	"crypto-wallet-backend/internal/api/middleware"
	"crypto-wallet-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func NewRouter(container *services.Container) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(container.UserService, container.SecurityService)
	walletHandler := handlers.NewWalletHandler(container.WalletService)
	agentHandler := handlers.NewAIAgentHandler(container.AIAgentService)
	marketHandler := handlers.NewMarketDataHandler(container.MarketDataService)

	// Public routes
	public := router.Group("/api/v1")
	{
		public.POST("/auth/register", authHandler.Register)
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/refresh", authHandler.RefreshToken)
		public.GET("/market/prices", marketHandler.GetPrices)
		public.GET("/agents/marketplace", agentHandler.GetMarketplaceAgents)
	}

	// Protected routes
	protected := router.Group("/api/v1")
	protected.Use(middleware.AuthRequired(container.SecurityService))
	{
		// User routes
		protected.GET("/user/profile", authHandler.GetProfile)
		protected.PUT("/user/profile", authHandler.UpdateProfile)
		protected.POST("/auth/logout", authHandler.Logout)

		// Wallet routes
		protected.GET("/wallets", walletHandler.GetWallets)
		protected.POST("/wallets", walletHandler.CreateWallet)
		protected.GET("/wallets/:id", walletHandler.GetWallet)
		protected.GET("/wallets/:id/transactions", walletHandler.GetTransactions)
		protected.POST("/wallets/:id/send", walletHandler.SendTransaction)

		// AI Agent routes
		protected.GET("/agents", agentHandler.GetUserAgents)
		protected.POST("/agents", agentHandler.CreateAgent)
		protected.GET("/agents/:id", agentHandler.GetAgent)
		protected.POST("/agents/:id/execute", agentHandler.ExecuteAgent)
		protected.PUT("/agents/:id/activate", agentHandler.ActivateAgent)
		protected.PUT("/agents/:id/deactivate", agentHandler.DeactivateAgent)
		protected.GET("/agents/:id/executions", agentHandler.GetExecutions)
		protected.GET("/agents/:id/trades", agentHandler.GetTrades)

		// Market data routes
		protected.GET("/market/portfolio", marketHandler.GetPortfolioValue)
		protected.GET("/market/charts/:symbol", marketHandler.GetChartData)
	}

	// WebSocket endpoint
	router.GET("/ws", handlers.HandleWebSocket(container))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return router
}