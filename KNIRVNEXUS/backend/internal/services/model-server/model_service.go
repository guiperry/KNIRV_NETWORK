package objectserver

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"backend_server/internal/config"
	"backend_server/internal/database"
)

// ModelServer represents the model server service
type ModelServer struct {
	config *config.Config
	db     *database.BuntDBManager

	// Configuration
	modelDir      string
	maxModels     int
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
	ModelDir  string    `json:"model_dir"`
	StartTime time.Time `json:"start_time"`
	Version   string    `json:"version"`
}

// ModelInfo represents information about a plugin model
type ModelInfo struct {
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
	Models []ModelInfo `json:"objects"`
	Count  int         `json:"count"`
}

// NewModelServer creates a new model server instance
func NewModelServer(config *config.Config, db *database.BuntDBManager) (*ModelServer, error) {
	// Set default values from config with fallbacks
	modelDir := config.ModelServer.StoragePath
	if modelDir == "" {
		modelDir = "./models" // fallback
	}
	maxModels := config.ModelServer.MaxModels
	if maxModels <= 0 {
		maxModels = 10 // fallback
	}
	enableRuntime := true
	enableCORS := config.ModelServer.EnableCORS
	if config.ModelServer.StoragePath == "" {
		// If config is empty (test case), use defaults
		enableCORS = true
	}

	// Create model directory if it doesn't exist
	if err := ensureModelDirectory(modelDir); err != nil {
		return nil, fmt.Errorf("failed to create model directory: %w", err)
	}

	serverInfo := &ServerInfo{
		Name:      "KNIRV-NEXUS Plugin Model Server",
		Port:      0, // Will be set by the main server
		ModelDir:  modelDir,
		StartTime: time.Now(),
		Version:   "2.0.0",
	}

	service := &ModelServer{
		config:        config,
		db:            db,
		modelDir:      modelDir,
		maxModels:     maxModels,
		enableRuntime: enableRuntime,
		enableCORS:    enableCORS,
		serverInfo:    serverInfo,
		startTime:     time.Now(),
		running:       false,
	}

	return service, nil
}

// Start starts the model server service
func (as *ModelServer) Start() error {
	log.Println("Starting Model Server service...")

	// Initialize runtime manager if enabled
	if as.enableRuntime {
		ctx := context.Background()
		var err error
		as.runtimeManager, err = NewRuntimeManager(ctx, as.modelDir, as.maxModels)
		if err != nil {
			return fmt.Errorf("failed to create runtime manager: %w", err)
		}

		if err := as.runtimeManager.Start(); err != nil {
			return fmt.Errorf("failed to start runtime manager: %w", err)
		}

		log.Printf("Runtime manager started with max %d objects", as.maxModels)
	}

	as.running = true
	log.Println("Model Server service started successfully")
	return nil
}

// Stop stops the model server service
func (as *ModelServer) Stop() error {
	log.Println("Stopping Model Server service...")

	as.running = false

	// Stop runtime manager if it exists
	if as.runtimeManager != nil {
		if err := as.runtimeManager.Stop(); err != nil {
			log.Printf("Error stopping runtime manager: %v", err)
		}
	}

	log.Println("Model Server service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (as *ModelServer) IsRunning() bool {
	return as.running
}

// GetServerInfo returns server information
func (as *ModelServer) GetServerInfo() *ServerInfo {
	return as.serverInfo
}

// GetRuntimeManager returns the runtime manager (if enabled)
func (as *ModelServer) GetRuntimeManager() *RuntimeManager {
	return as.runtimeManager
}

// ensureModelDirectory creates the model directory if it doesn't exist
func ensureModelDirectory(modelDir string) error {
	return os.MkdirAll(modelDir, 0755)
}
