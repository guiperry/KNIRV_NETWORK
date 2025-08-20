package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// ServerConfig represents the API gateway configuration
type ServerConfig struct {
	// Server settings
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`

	// Service ports
	DVEManagerPort     int `mapstructure:"dve_manager_port"`
	ValidationCorePort int `mapstructure:"validation_core_port"`

	// Logging
	LogLevel string `mapstructure:"log_level"`
}

// APIGateway represents the main API gateway server
type APIGateway struct {
	config *ServerConfig
	router *gin.Engine
	server *http.Server

	// Service proxies for domain services (managed by backend orchestrator)
	dveManagerProxy     *httputil.ReverseProxy
	validationCoreProxy *httputil.ReverseProxy
}

// NewAPIGateway creates a new API gateway instance
func NewAPIGateway(config *ServerConfig) (*APIGateway, error) {
	// Set Gin mode
	if config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	gateway := &APIGateway{
		config: config,
		router: gin.New(),
	}

	// Setup middleware
	gateway.router.Use(gin.Logger())
	gateway.router.Use(gin.Recovery())

	// CORS middleware
	gateway.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Setup service proxies
	if err := gateway.setupProxies(); err != nil {
		return nil, fmt.Errorf("failed to setup proxies: %w", err)
	}

	// Setup routes
	gateway.setupRoutes()

	return gateway, nil
}

// setupProxies creates reverse proxies for the domain services
func (g *APIGateway) setupProxies() error {
	// DVE Manager proxy
	dveURL, err := url.Parse(fmt.Sprintf("http://localhost:%d", g.config.DVEManagerPort))
	if err != nil {
		return fmt.Errorf("invalid DVE manager URL: %w", err)
	}
	g.dveManagerProxy = httputil.NewSingleHostReverseProxy(dveURL)

	// Validation Core proxy
	validationURL, err := url.Parse(fmt.Sprintf("http://localhost:%d", g.config.ValidationCorePort))
	if err != nil {
		return fmt.Errorf("invalid validation core URL: %w", err)
	}
	g.validationCoreProxy = httputil.NewSingleHostReverseProxy(validationURL)

	return nil
}

// setupRoutes configures the API routes
func (g *APIGateway) setupRoutes() {
	// Health check
	g.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":     "healthy",
			"version":    Version,
			"build_time": BuildTime,
			"git_commit": GitCommit,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Version endpoint
	g.router.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version":    Version,
			"build_time": BuildTime,
			"git_commit": GitCommit,
		})
	})

	// API routes
	api := g.router.Group("/api/v1")
	{
		// DVE Manager routes
		api.Any("/dve/*path", func(c *gin.Context) {
			g.dveManagerProxy.ServeHTTP(c.Writer, c.Request)
		})

		// Validation Core routes
		api.Any("/validation/*path", func(c *gin.Context) {
			g.validationCoreProxy.ServeHTTP(c.Writer, c.Request)
		})

		// Gateway-specific routes
		api.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"gateway": "running",
				"services": gin.H{
					"dve_manager":     g.isServiceRunning(g.config.DVEManagerPort),
					"validation_core": g.isServiceRunning(g.config.ValidationCorePort),
				},
			})
		})
	}
}

// isServiceRunning checks if a service is running on the given port
func (g *APIGateway) isServiceRunning(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// waitForServices waits for the domain services to be available
func (g *APIGateway) waitForServices() error {
	log.Println("Waiting for domain services to be available...")

	// Wait for DVE Manager
	for i := 0; i < 30; i++ {
		if g.isServiceRunning(g.config.DVEManagerPort) {
			log.Printf("DVE Manager is available on port %d", g.config.DVEManagerPort)
			break
		}
		time.Sleep(1 * time.Second)
	}

	// Wait for Validation Core
	for i := 0; i < 30; i++ {
		if g.isServiceRunning(g.config.ValidationCorePort) {
			log.Printf("Validation Core is available on port %d", g.config.ValidationCorePort)
			break
		}
		time.Sleep(1 * time.Second)
	}

	log.Println("Domain services are ready")
	return nil
}

// Start starts the API gateway server
func (g *APIGateway) Start() error {
	// Wait for domain services to be available (managed by backend orchestrator)
	if err := g.waitForServices(); err != nil {
		return err
	}

	// Create HTTP server
	g.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", g.config.Host, g.config.Port),
		Handler:      g.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting API Gateway on %s:%d", g.config.Host, g.config.Port)
		if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	return nil
}

// Stop stops the API gateway server
func (g *APIGateway) Stop() error {
	// Stop HTTP server
	if g.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := g.server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}

	// Note: Domain services are managed by backend orchestrator, not by this gateway

	return nil
}

// loadConfig loads configuration from file and environment
func loadConfig() (*ServerConfig, error) {
	// Set default values
	viper.SetDefault("host", "0.0.0.0")
	viper.SetDefault("port", 8081)
	viper.SetDefault("dve_manager_port", 8082)
	viper.SetDefault("validation_core_port", 8083)
	viper.SetDefault("log_level", "info")

	// Set config file name and paths
	viper.SetConfigName("nexus-server")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvPrefix("NEXUS")

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		log.Println("No config file found, using defaults and environment variables")
	}

	var config ServerConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

func main() {
	// Parse command line flags
	var configFile = flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Print version information
	fmt.Printf("KNIRV-NEXUS API Gateway v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

	// Set config file if provided
	if *configFile != "" {
		viper.SetConfigFile(*configFile)
	}

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create API gateway
	gateway, err := NewAPIGateway(config)
	if err != nil {
		log.Fatalf("Failed to create API gateway: %v", err)
	}

	// Start the gateway
	if err := gateway.Start(); err != nil {
		log.Fatalf("Failed to start API gateway: %v", err)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API gateway...")
	if err := gateway.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	log.Println("API gateway stopped")
}
