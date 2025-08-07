package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/knirv/nexus-backend/internal/config"
	"github.com/knirv/nexus-backend/internal/database"
	"github.com/knirv/nexus-backend/internal/services/gateway"
	"github.com/knirv/nexus-backend/pkg/sse"
	"github.com/spf13/viper"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewBuntDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize SSE manager
	sseManager := sse.NewSSEManager()

	// Initialize API Gateway
	apiGateway, err := gateway.NewAPIGateway(db.GetDB(), sseManager, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize API Gateway: %v", err)
	}

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Setup CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Setup routes
	apiGateway.SetupRoutes(router)

	// Create HTTP server
	server := &http.Server{
		Addr:    cfg.API.Address,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("API Gateway starting on %s", cfg.API.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down API Gateway...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	// Close SSE connections
	sseManager.Close()

	log.Println("API Gateway stopped")
}

func init() {
	// Set default configuration values
	viper.SetDefault("database.path", "/app/data/api-gateway.db")
	viper.SetDefault("api.address", ":8080")
	viper.SetDefault("environment", "development")
	viper.SetDefault("auth.jwt_secret", "your-secret-key")
	viper.SetDefault("auth.token_expiry", "24h")

	// Environment variable bindings
	viper.BindEnv("database.path", "KNIRV_DATABASE_PATH")
	viper.BindEnv("api.address", "KNIRV_API_ADDRESS")
	viper.BindEnv("environment", "KNIRV_ENVIRONMENT")
	viper.BindEnv("auth.jwt_secret", "KNIRV_JWT_SECRET")
	viper.BindEnv("auth.token_expiry", "KNIRV_TOKEN_EXPIRY")
}
