package pluginserver

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

// PluginServer represents the plugin server service
type PluginServer struct {
	config *config.Config
	db     *database.BuntDBManager

	// Configuration
	pluginDir     string
	maxPlugins    int
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
	PluginDir string    `json:"plugin_dir"`
	StartTime time.Time `json:"start_time"`
	Version   string    `json:"version"`
}

// PluginInfo represents information about a plugin unit
type PluginInfo struct {
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
	Plugins []PluginInfo `json:"objects"`
	Count   int          `json:"count"`
}

// NewPluginServer creates a new plugin server instance
func NewPluginServer(config *config.Config, db *database.BuntDBManager) (*PluginServer, error) {
	// Set default values from config with fallbacks
	pluginDir := viper.GetString("model_server.storage_path")
	if pluginDir == "" {
		pluginDir = "./models" // fallback (keeping backend config keys for now)
	}
	maxPlugins := config.ModelServer.MaxModels
	if maxPlugins <= 0 {
		maxPlugins = 10 // fallback
	}
	enableRuntime := true
	enableCORS := config.ModelServer.EnableCORS
	if config.ModelServer.StoragePath == "" {
		// If config is empty (test case), use defaults
		enableCORS = true
	}

	// Create plugin directory if it doesn't exist
	if err := ensurePluginDirectory(pluginDir); err != nil {
		return nil, fmt.Errorf("failed to create plugin directory: %w", err)
	}

	serverInfo := &ServerInfo{
		Name:      "KNIRV-SERVER Agentic Memory Plugin Server",
		Port:      0, // Will be set by the main server
		PluginDir: pluginDir,
		StartTime: time.Now(),
		Version:   "2.0.0",
	}

	service := &PluginServer{
		config:        config,
		db:            db,
		pluginDir:     pluginDir,
		maxPlugins:    maxPlugins,
		enableRuntime: enableRuntime,
		enableCORS:    enableCORS,
		serverInfo:    serverInfo,
		startTime:     time.Now(),
		running:       false,
	}

	return service, nil
}

// Start starts the plugin server service
func (as *PluginServer) Start() error {
	log.Println("Starting Plugin Server service...")

	// Initialize runtime manager if enabled
	if as.enableRuntime {
		ctx := context.Background()
		var err error
		as.runtimeManager, err = NewRuntimeManager(ctx, as.pluginDir, as.maxPlugins)
		if err != nil {
			return fmt.Errorf("failed to create runtime manager: %w", err)
		}

		if err := as.runtimeManager.Start(); err != nil {
			return fmt.Errorf("failed to start runtime manager: %w", err)
		}

		log.Printf("Runtime manager started with max %d plugins", as.maxPlugins)
	}

	as.running = true
	log.Println("Plugin Server service started successfully")
	return nil
}

// Stop stops the plugin server service
func (as *PluginServer) Stop() error {
	log.Println("Stopping Plugin Server service...")

	as.running = false

	// Stop runtime manager if it exists
	if as.runtimeManager != nil {
		if err := as.runtimeManager.Stop(); err != nil {
			log.Printf("Error stopping runtime manager: %v", err)
		}
	}

	log.Println("Plugin Server service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (as *PluginServer) IsRunning() bool {
	return as.running
}

// GetServerInfo returns server information
func (as *PluginServer) GetServerInfo() *ServerInfo {
	return as.serverInfo
}

// GetRuntimeManager returns the runtime manager (if enabled)
func (as *PluginServer) GetRuntimeManager() *RuntimeManager {
	return as.runtimeManager
}

// ensurePluginDirectory creates the plugin directory if it doesn't exist
func ensurePluginDirectory(pluginDir string) error {
	return os.MkdirAll(pluginDir, 0755)
}
