package fabricserver

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"backend_server/internal/config"
	"backend_server/internal/database"
	"github.com/spf13/viper"
)

// FabricServer represents the fabric server service
type FabricServer struct {
	config *config.Config
	db     *database.BuntDBManager

	// Configuration
	fabricDir     string
	maxFabrics    int
	enableRuntime bool
	enableCORS    bool

	// Components
	runtimeManager *RuntimeManager

	// Server info
	serverInfo *ServerInfo
	startTime  time.Time
	running    bool
}

// ServerInfo represents information about this server instance
type ServerInfo struct {
	Name      string    `json:"name"`
	Port      int       `json:"port"`
	FabricDir string    `json:"fabric_dir"`
	StartTime time.Time `json:"start_time"`
	Version   string    `json:"version"`
}

// FabricInfo represents information about a fabric unit
type FabricInfo struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	Hash         string    `json:"hash,omitempty"`
}

// UploadResponse represents the response from an upload operation
type UploadResponse struct {
	Success  bool   `json:"success"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Message  string `json:"message,omitempty"`
}

// ListResponse represents the response from a list operation
type ListResponse struct {
	Fabrics []FabricInfo `json:"objects"`
	Count   int          `json:"count"`
}

// NewFabricServer creates a new fabric server instance
func NewFabricServer(config *config.Config, db *database.BuntDBManager) (*FabricServer, error) {
	// Set default values from config with fallbacks
	fabricDir := viper.GetString("model_server.storage_path")
	if fabricDir == "" {
		fabricDir = "./models" // fallback (keeping backend config keys for now)
	}
	maxFabrics := config.ModelServer.MaxModels
	if maxFabrics <= 0 {
		maxFabrics = 10 // fallback
	}
	enableRuntime := true
	enableCORS := config.ModelServer.EnableCORS
	if config.ModelServer.StoragePath == "" {
		// If config is empty (test case), use defaults
		enableCORS = true
	}

	// Create fabric directory if it doesn't exist
	if err := ensureFabricDirectory(fabricDir); err != nil {
		return nil, fmt.Errorf("failed to create fabric directory: %w", err)
	}

	serverInfo := &ServerInfo{
		Name:      "KNIRV-SERVER Agentic Memory Fabric Server",
		Port:      0, // Will be set by the main server
		FabricDir: fabricDir,
		StartTime: time.Now(),
		Version:   "2.0.0",
	}

	service := &FabricServer{
		config:        config,
		db:            db,
		fabricDir:     fabricDir,
		maxFabrics:    maxFabrics,
		enableRuntime: enableRuntime,
		enableCORS:    enableCORS,
		serverInfo:    serverInfo,
		startTime:     time.Now(),
		running:       false,
	}

	return service, nil
}

// Start starts the fabric server service
func (as *FabricServer) Start() error {
	log.Println("Starting Fabric Server service...")

	// Initialize runtime manager if enabled
	if as.enableRuntime {
		ctx := context.Background()
		var err error
		as.runtimeManager, err = NewRuntimeManager(ctx, as.fabricDir, as.maxFabrics)
		if err != nil {
			return fmt.Errorf("failed to create runtime manager: %w", err)
		}

		if err := as.runtimeManager.Start(); err != nil {
			return fmt.Errorf("failed to start runtime manager: %w", err)
		}

		log.Printf("Runtime manager started with max %d fabrics", as.maxFabrics)
	}

	as.running = true
	log.Println("Fabric Server service started successfully")
	return nil
}

// Stop stops the fabric server service
func (as *FabricServer) Stop() error {
	log.Println("Stopping Fabric Server service...")

	as.running = false

	// Stop runtime manager if it exists
	if as.runtimeManager != nil {
		if err := as.runtimeManager.Stop(); err != nil {
			log.Printf("Error stopping runtime manager: %v", err)
		}
	}

	log.Println("Fabric Server service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (as *FabricServer) IsRunning() bool {
	return as.running
}

// GetServerInfo returns server information
func (as *FabricServer) GetServerInfo() *ServerInfo {
	return as.serverInfo
}

// GetRuntimeManager returns the runtime manager (if enabled)
func (as *FabricServer) GetRuntimeManager() *RuntimeManager {
	return as.runtimeManager
}

// ensureFabricDirectory creates the fabric directory if it doesn't exist
func ensureFabricDirectory(fabricDir string) error {
	return os.MkdirAll(fabricDir, 0755)
}
