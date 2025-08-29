package main

import (
	"context"
	"crypto-wallet-backend/internal/api"
	"crypto-wallet-backend/internal/config"
	"crypto-wallet-backend/internal/database"
	"crypto-wallet-backend/internal/services"
	"crypto-wallet-backend/pkg/logger"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Initialize logger
	logger := logger.NewLogger(logger.Config{Level: cfg.LogLevel, Output: "stdout"})

	// Initialize database
	db, err := database.NewDatabase(database.Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "password",
		DBName:   "knirvwallet",
		SSLMode:  "disable",
	})
	if err != nil {
		logger.Fatal("Failed to initialize database: %v", err)
	}

	// Initialize services
	serviceContainer := services.NewContainer(db.GetDB(), cfg, logger)

	// Initialize API router
	router := api.NewRouter(serviceContainer)

	// Configure Gin
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting server on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}
