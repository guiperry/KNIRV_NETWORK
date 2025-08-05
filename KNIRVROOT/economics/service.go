package economics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// EconomicsService is the main service that orchestrates the token economics system
type EconomicsService struct {
	tokenEconomics *TokenEconomics
	api            *EconomicsAPI
	integration    *EconomicsIntegration
	server         *http.Server
	config         *ServiceConfig
}

type ServiceConfig struct {
	Port              string          `json:"port"`
	NRNContract       string          `json:"nrn_contract"`
	XionRPC           string          `json:"xion_rpc"`
	ComponentConfig   ComponentConfig `json:"component_config"`
	DatabasePath      string          `json:"database_path"`
	EnableCORS        bool            `json:"enable_cors"`
	LogLevel          string          `json:"log_level"`
}

// MockLevelDB is a simple in-memory implementation for testing
type MockLevelDB struct {
	data map[string][]byte
}

func NewMockLevelDB() *MockLevelDB {
	return &MockLevelDB{
		data: make(map[string][]byte),
	}
}

func (db *MockLevelDB) Put(key, value []byte) error {
	db.data[string(key)] = value
	return nil
}

func (db *MockLevelDB) Get(key []byte) ([]byte, error) {
	value, exists := db.data[string(key)]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}
	return value, nil
}

func (db *MockLevelDB) Delete(key []byte) error {
	delete(db.data, string(key))
	return nil
}

func (db *MockLevelDB) Close() error {
	return nil
}

func NewEconomicsService(config *ServiceConfig) (*EconomicsService, error) {
	// Initialize database (using mock for now)
	db := NewMockLevelDB()

	// Initialize token economics
	tokenEconomics, err := NewTokenEconomics(config.NRNContract, config.XionRPC, db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token economics: %w", err)
	}

	// Initialize API
	api := NewEconomicsAPI(tokenEconomics)

	// Initialize integration
	integration := NewEconomicsIntegration(tokenEconomics, config.ComponentConfig)

	return &EconomicsService{
		tokenEconomics: tokenEconomics,
		api:            api,
		integration:    integration,
		config:         config,
	}, nil
}

func (es *EconomicsService) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start token economics system
	if err := es.tokenEconomics.Start(ctx); err != nil {
		return fmt.Errorf("failed to start token economics: %w", err)
	}

	// Start integration service
	if err := es.integration.Start(ctx); err != nil {
		return fmt.Errorf("failed to start integration service: %w", err)
	}

	// Setup HTTP server
	router := mux.NewRouter()
	
	// Register API routes
	es.api.RegisterRoutes(router)
	
	// Add integration status endpoint
	router.HandleFunc("/economics/integration/status", es.handleIntegrationStatus).Methods("GET")
	
	// Add service info endpoint
	router.HandleFunc("/economics/info", es.handleServiceInfo).Methods("GET")

	// Setup CORS if enabled
	var handler http.Handler = router
	if es.config.EnableCORS {
		c := cors.New(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"*"},
		})
		handler = c.Handler(router)
	}

	// Create HTTP server
	es.server = &http.Server{
		Addr:         ":" + es.config.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting Economics Service on port %s", es.config.Port)
		if err := es.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Economics Service...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := es.server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	cancel() // Cancel the main context to stop background processes

	log.Println("Economics Service stopped")
	return nil
}

func (es *EconomicsService) Stop() error {
	if es.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return es.server.Shutdown(ctx)
	}
	return nil
}

func (es *EconomicsService) handleIntegrationStatus(w http.ResponseWriter, r *http.Request) {
	status := es.integration.GetIntegrationStatus()
	es.api.sendSuccess(w, status)
}

func (es *EconomicsService) handleServiceInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service":     "KNIRV Economics Service",
		"version":     "1.0.0",
		"port":        es.config.Port,
		"nrn_contract": es.config.NRNContract,
		"xion_rpc":    es.config.XionRPC,
		"components": map[string]string{
			"knirvchain": es.config.ComponentConfig.KNIRVChainURL,
			"knirvnexus": es.config.ComponentConfig.KNIRVNexusURL,
			"knirvroot":  es.config.ComponentConfig.KNIRVRootURL,
			"knirvgraph": es.config.ComponentConfig.KNIRVGraphURL,
		},
		"started_at": time.Now(),
		"status":     "running",
	}
	es.api.sendSuccess(w, info)
}

// GetDefaultConfig returns a default configuration for the economics service
func GetDefaultConfig() *ServiceConfig {
	return &ServiceConfig{
		Port:        "8090",
		NRNContract: "nrn_contract_address_placeholder",
		XionRPC:     "https://rpc.xion-testnet-1.burnt.com:443",
		ComponentConfig: ComponentConfig{
			KNIRVChainURL: "http://localhost:8080",
			KNIRVNexusURL: "http://localhost:8081",
			KNIRVRootURL:  "http://localhost:8082",
			KNIRVGraphURL: "http://localhost:8083",
		},
		DatabasePath: "./economics.db",
		EnableCORS:   true,
		LogLevel:     "info",
	}
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() *ServiceConfig {
	config := GetDefaultConfig()

	if port := os.Getenv("ECONOMICS_PORT"); port != "" {
		config.Port = port
	}
	if contract := os.Getenv("NRN_CONTRACT"); contract != "" {
		config.NRNContract = contract
	}
	if rpc := os.Getenv("XION_RPC"); rpc != "" {
		config.XionRPC = rpc
	}
	if chainURL := os.Getenv("KNIRVCHAIN_URL"); chainURL != "" {
		config.ComponentConfig.KNIRVChainURL = chainURL
	}
	if nexusURL := os.Getenv("KNIRVNEXUS_URL"); nexusURL != "" {
		config.ComponentConfig.KNIRVNexusURL = nexusURL
	}
	if rootURL := os.Getenv("KNIRVROOT_URL"); rootURL != "" {
		config.ComponentConfig.KNIRVRootURL = rootURL
	}
	if graphURL := os.Getenv("KNIRVGRAPH_URL"); graphURL != "" {
		config.ComponentConfig.KNIRVGraphURL = graphURL
	}
	if dbPath := os.Getenv("ECONOMICS_DB_PATH"); dbPath != "" {
		config.DatabasePath = dbPath
	}

	return config
}

// Example usage function
func RunEconomicsService() error {
	config := LoadConfigFromEnv()
	
	service, err := NewEconomicsService(config)
	if err != nil {
		return fmt.Errorf("failed to create economics service: %w", err)
	}

	return service.Start()
}
