// Package embedded_knirvchain provides integration with standalone KNIRVCHAIN
package embedded_knirvchain

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

// ChainIntegration integrates embedded KNIRVCHAIN with standalone KNIRVCHAIN
type ChainIntegration struct {
	embeddedChain   *EmbeddedKNIRVChain
	endpointHandler *EndpointHandler
	httpServer      *http.Server
	isRunning       bool
}

// NewChainIntegration creates a new chain integration
func NewChainIntegration(config *EmbeddedChainConfig) *ChainIntegration {
	embeddedChain := NewEmbeddedKNIRVChain(config)
	endpointHandler := NewEndpointHandler(embeddedChain)

	return &ChainIntegration{
		embeddedChain:   embeddedChain,
		endpointHandler: endpointHandler,
		isRunning:       false,
	}
}

// Initialize initializes the embedded KNIRVCHAIN integration
func (ci *ChainIntegration) Initialize() error {
	log.Println("Initializing Embedded KNIRVCHAIN Integration...")

	// Initialize the embedded chain
	if err := ci.embeddedChain.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize embedded chain: %w", err)
	}

	// Set up KNIRVGRAPH client
	ci.setupKNIRVGraphClient()

	// Set up KNIRV-ORACLE client
	ci.setupOracleClient()

	log.Println("Embedded KNIRVCHAIN Integration initialized successfully")
	return nil
}

// setupKNIRVGraphClient sets up the KNIRVGRAPH client for error cluster queries
func (ci *ChainIntegration) setupKNIRVGraphClient() {
	// Check for KNIRVGRAPH endpoint in environment
	knirvgraphEndpoint := os.Getenv("KNIRVGRAPH_ENDPOINT")
	if knirvgraphEndpoint == "" {
		knirvgraphEndpoint = os.Getenv("KNIRVGRAPH_URL")
	}

	if knirvgraphEndpoint != "" {
		log.Printf("Setting up HTTP KNIRVGRAPH client: %s", knirvgraphEndpoint)
		httpClient := NewHTTPKNIRVGraphClient(knirvgraphEndpoint)
		ci.embeddedChain.SetKNIRVGraphClient(httpClient)
	} else {
		log.Println("No KNIRVGRAPH endpoint configured, using mock client for testing")
		mockClient := NewMockKNIRVGraphClient()
		ci.embeddedChain.SetKNIRVGraphClient(mockClient)
	}
}

// setupOracleClient sets up the KNIRV-ORACLE client for NRN burn signaling
func (ci *ChainIntegration) setupOracleClient() {
	// Check for KNIRV-ORACLE endpoint in environment
	oracleEndpoint := os.Getenv("KNIRVORACLE_ENDPOINT")
	if oracleEndpoint == "" {
		oracleEndpoint = os.Getenv("ORACLE_ENDPOINT")
	}

	oracleAPIKey := os.Getenv("ORACLE_API_KEY")

	if oracleEndpoint != "" {
		// Check if IBC configuration is available
		channelID := os.Getenv("IBC_CHANNEL_ID")
		portID := os.Getenv("IBC_PORT_ID")

		if channelID != "" && portID != "" {
			log.Printf("Setting up IBC KNIRV-ORACLE client: %s (Channel: %s, Port: %s)",
				oracleEndpoint, channelID, portID)
			ibcClient := NewIBCOracleClient(oracleEndpoint, oracleAPIKey, channelID, portID)
			ci.embeddedChain.SetOracleClient(ibcClient)
		} else {
			log.Printf("Setting up HTTP KNIRV-ORACLE client: %s", oracleEndpoint)
			httpClient := NewHTTPOracleClient(oracleEndpoint, oracleAPIKey)
			ci.embeddedChain.SetOracleClient(httpClient)
		}
	} else {
		log.Println("No KNIRV-ORACLE endpoint configured, using mock client for testing")
		mockClient := NewMockOracleClient()
		ci.embeddedChain.SetOracleClient(mockClient)
	}
}

// StartHTTPServer starts the HTTP server for embedded KNIRVCHAIN endpoints
func (ci *ChainIntegration) StartHTTPServer(port string) error {
	if ci.isRunning {
		return fmt.Errorf("HTTP server is already running")
	}

	router := mux.NewRouter()

	// Create a subrouter for embedded KNIRVCHAIN endpoints
	chainRouter := router.PathPrefix("/embedded-chain").Subrouter()
	ci.endpointHandler.RegisterRoutes(chainRouter)

	// Add CORS middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Add logging middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
		})
	})

	ci.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Embedded KNIRVCHAIN HTTP server on port %s", port)
		if err := ci.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	ci.isRunning = true
	return nil
}

// StopHTTPServer stops the HTTP server
func (ci *ChainIntegration) StopHTTPServer() error {
	if !ci.isRunning {
		return nil
	}

	log.Println("Stopping Embedded KNIRVCHAIN HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ci.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	ci.isRunning = false
	log.Println("Embedded KNIRVCHAIN HTTP server stopped")
	return nil
}

// GetEmbeddedChain returns the embedded chain instance
func (ci *ChainIntegration) GetEmbeddedChain() *EmbeddedKNIRVChain {
	return ci.embeddedChain
}

// GetEndpointHandler returns the endpoint handler
func (ci *ChainIntegration) GetEndpointHandler() *EndpointHandler {
	return ci.endpointHandler
}

// IsRunning returns whether the integration is running
func (ci *ChainIntegration) IsRunning() bool {
	return ci.isRunning
}

// Shutdown shuts down the entire integration
func (ci *ChainIntegration) Shutdown() error {
	log.Println("Shutting down Embedded KNIRVCHAIN Integration...")

	// Stop HTTP server
	if err := ci.StopHTTPServer(); err != nil {
		log.Printf("Error stopping HTTP server: %v", err)
	}

	// Shutdown embedded chain
	if err := ci.embeddedChain.Shutdown(); err != nil {
		log.Printf("Error shutting down embedded chain: %v", err)
	}

	log.Println("Embedded KNIRVCHAIN Integration shutdown complete")
	return nil
}

// RegisterWithExistingRouter registers embedded KNIRVCHAIN endpoints with an existing router
func (ci *ChainIntegration) RegisterWithExistingRouter(router *mux.Router) {
	// Create a subrouter for embedded KNIRVCHAIN endpoints
	chainRouter := router.PathPrefix("/embedded-chain").Subrouter()
	ci.endpointHandler.RegisterRoutes(chainRouter)

	log.Println("Embedded KNIRVCHAIN endpoints registered with existing router")
}

// HealthCheck performs a health check on the embedded chain
func (ci *ChainIntegration) HealthCheck() map[string]interface{} {
	skills, _ := ci.embeddedChain.GetSkills(nil)
	chains, _ := ci.embeddedChain.GetSkillChains()

	return map[string]interface{}{
		"status":       "healthy",
		"initialized":  ci.embeddedChain.isInitialized,
		"running":      ci.isRunning,
		"skill_count":  len(skills),
		"chain_count":  len(chains),
		"memory_usage": ci.embeddedChain.calculateMemoryUsage(),
		"timestamp":    time.Now().Unix(),
	}
}

// GetMetrics returns metrics about the embedded chain
func (ci *ChainIntegration) GetMetrics() map[string]interface{} {
	skills, _ := ci.embeddedChain.GetSkills(nil)
	chains, _ := ci.embeddedChain.GetSkillChains()

	totalUsage := int64(0)
	totalConsensusScore := 0.0
	for _, skill := range skills {
		totalUsage += int64(skill.UsageCount)
		totalConsensusScore += skill.ConsensusScore
	}

	avgConsensusScore := 0.0
	if len(skills) > 0 {
		avgConsensusScore = totalConsensusScore / float64(len(skills))
	}

	return map[string]interface{}{
		"total_skills":            len(skills),
		"total_chains":            len(chains),
		"total_skill_usage":       totalUsage,
		"average_consensus_score": avgConsensusScore,
		"memory_usage":            ci.embeddedChain.calculateMemoryUsage(),
		"active_invocations":      len(ci.embeddedChain.activeInvocations),
		"consensus_nodes":         len(ci.embeddedChain.consensusNodes),
		"timestamp":               time.Now().Unix(),
	}
}

// CreateDefaultSkills creates some default skills for testing
func (ci *ChainIntegration) CreateDefaultSkills() error {
	log.Println("Creating default skills for testing...")

	// Create a simple test skill
	testSkill := &LoRAAdapterSkill{
		SkillID:                "test-skill-001",
		SkillName:              "Test Code Generation",
		Description:            "A test skill for code generation",
		BaseModelCompatibility: "hrm",
		Version:                1,
		Rank:                   8,
		Alpha:                  16.0,
		WeightsA:               make([]float32, 8*1024), // 8x1024 matrix
		WeightsB:               make([]float32, 1024*8), // 1024x8 matrix
		AdditionalMetadata: map[string]string{
			"capabilities": `["code_generation", "debugging"]`,
			"author":       "KNIRV System",
		},
	}

	// Initialize weights with small random values
	for i := range testSkill.WeightsA {
		testSkill.WeightsA[i] = 0.01 * float32(i%100) / 100.0
	}
	for i := range testSkill.WeightsB {
		testSkill.WeightsB[i] = 0.01 * float32(i%100) / 100.0
	}

	if err := ci.embeddedChain.RegisterSkill(testSkill); err != nil {
		return fmt.Errorf("failed to register test skill: %w", err)
	}

	// Create another test skill
	testSkill2 := &LoRAAdapterSkill{
		SkillID:                "test-skill-002",
		SkillName:              "Test Data Analysis",
		Description:            "A test skill for data analysis",
		BaseModelCompatibility: "hrm",
		Version:                1,
		Rank:                   4,
		Alpha:                  8.0,
		WeightsA:               make([]float32, 4*1024), // 4x1024 matrix
		WeightsB:               make([]float32, 1024*4), // 1024x4 matrix
		AdditionalMetadata: map[string]string{
			"capabilities": `["data_analysis", "visualization"]`,
			"author":       "KNIRV System",
		},
	}

	// Initialize weights with small random values
	for i := range testSkill2.WeightsA {
		testSkill2.WeightsA[i] = 0.005 * float32(i%200) / 200.0
	}
	for i := range testSkill2.WeightsB {
		testSkill2.WeightsB[i] = 0.005 * float32(i%200) / 200.0
	}

	if err := ci.embeddedChain.RegisterSkill(testSkill2); err != nil {
		return fmt.Errorf("failed to register test skill 2: %w", err)
	}

	log.Println("Default skills created successfully")
	return nil
}

// GetDefaultConfig returns a default configuration for embedded KNIRVCHAIN
func GetDefaultConfig() *EmbeddedChainConfig {
	return &EmbeddedChainConfig{
		ModelKernel:           "hrm",
		MaxMemoryMB:           512,
		ConsensusThreshold:    0.75,
		LoRAAdapterCacheSize:  100,
		SkillChainDepth:       10,
		EnableRealTimeUpdates: true,
	}
}
