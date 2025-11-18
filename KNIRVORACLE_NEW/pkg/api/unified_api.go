package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"KNIRVORACLE/config"
	"KNIRVORACLE/pkg/services/binary"
	"KNIRVORACLE/pkg/services/nodejs"
)

// UnifiedAPI provides a single interface for all Oracle operations
type UnifiedAPI struct {
	config        *config.Config
	server        *http.Server
	nodeJSManager *nodejs.EmbeddedNodeJSManager
	binaryManager *binary.EmbeddedBinaryManager
	router        *mux.Router
	isRunning     bool
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Running      bool      `json:"running"`
	Port         uint64    `json:"port,omitempty"`
	PID          int       `json:"pid,omitempty"`
	StartTime    time.Time `json:"start_time,omitempty"`
	ResponseTime int64     `json:"response_time,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// OracleStatus represents the overall Oracle status
type OracleStatus struct {
	ChainID         string          `json:"chain_id"`
	Role            string          `json:"role"`
	IsRoot          bool            `json:"is_root"`
	IsBootnode      bool            `json:"is_bootnode"`
	TestnetEnabled  bool            `json:"testnet_enabled"`
	Uptime          string          `json:"uptime"`
	Services        []ServiceStatus `json:"services"`
	TotalServices   int             `json:"total_services"`
	RunningServices int             `json:"running_services"`
	LastUpdated     time.Time       `json:"last_updated"`
}

// NewUnifiedAPI creates a new unified API instance
func NewUnifiedAPI(cfg *config.Config, nodeJSMgr *nodejs.EmbeddedNodeJSManager, binaryMgr *binary.EmbeddedBinaryManager) *UnifiedAPI {
	api := &UnifiedAPI{
		config:        cfg,
		nodeJSManager: nodeJSMgr,
		binaryManager: binaryMgr,
		router:        mux.NewRouter(),
		isRunning:     false,
	}

	api.setupRoutes()
	return api
}

// setupRoutes configures all API routes
func (api *UnifiedAPI) setupRoutes() {
	// Oracle status and management
	api.router.HandleFunc("/api/oracle/status", api.handleOracleStatus).Methods("GET")
	api.router.HandleFunc("/api/oracle/info", api.handleOracleInfo).Methods("GET")

	// Service management
	api.router.HandleFunc("/api/services", api.handleListServices).Methods("GET")
	api.router.HandleFunc("/api/services/{name}/start", api.handleStartService).Methods("POST")
	api.router.HandleFunc("/api/services/{name}/stop", api.handleStopService).Methods("POST")
	api.router.HandleFunc("/api/services/{name}/restart", api.handleRestartService).Methods("POST")
	api.router.HandleFunc("/api/services/{name}/status", api.handleServiceStatus).Methods("GET")

	// Node.js services
	api.router.HandleFunc("/api/nodejs/services", api.handleNodeJSServices).Methods("GET")
	api.router.HandleFunc("/api/nodejs/status", api.handleNodeJSStatus).Methods("GET")

	// Binary services
	api.router.HandleFunc("/api/binary/services", api.handleBinaryServices).Methods("GET")
	api.router.HandleFunc("/api/binary/status", api.handleBinaryStatus).Methods("GET")

	// Tunnel management
	api.router.HandleFunc("/api/tunnel/status", api.handleTunnelStatus).Methods("GET")
	api.router.HandleFunc("/api/tunnel/connections", api.handleTunnelConnections).Methods("GET")

	// Wallet management
	api.router.HandleFunc("/api/wallet/status", api.handleWalletStatus).Methods("GET")
	api.router.HandleFunc("/api/wallet/balance", api.handleWalletBalance).Methods("GET")

	// Payment management
	api.router.HandleFunc("/api/payments/status", api.handlePaymentStatus).Methods("GET")
	api.router.HandleFunc("/api/payments/history", api.handlePaymentHistory).Methods("GET")

	// Plugin management
	api.router.HandleFunc("/api/plugins", api.handlePlugins).Methods("GET")
	api.router.HandleFunc("/api/plugins/{name}/enable", api.handleEnablePlugin).Methods("POST")
	api.router.HandleFunc("/api/plugins/{name}/disable", api.handleDisablePlugin).Methods("POST")

	// Network monitoring
	api.router.HandleFunc("/api/network/status", api.handleNetworkStatus).Methods("GET")
	api.router.HandleFunc("/api/network/peers", api.handleNetworkPeers).Methods("GET")

	// Economics integration
	api.router.HandleFunc("/api/economics/status", api.handleEconomicsStatus).Methods("GET")
	api.router.HandleFunc("/api/economics/metrics", api.handleEconomicsMetrics).Methods("GET")

	// Health check
	api.router.HandleFunc("/api/health", api.handleHealth).Methods("GET")

	// Static file serving for Web GUI
	api.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./webGUI/build/")))
}

// Start starts the unified API server
func (api *UnifiedAPI) Start(port int) error {
	if api.isRunning {
		return fmt.Errorf("unified API server is already running")
	}

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(api.router)

	api.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}

	log.Printf("Starting unified API server on port %d", port)

	go func() {
		if err := api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Unified API server error: %v", err)
		}
	}()

	api.isRunning = true
	log.Printf("Unified API server started successfully")
	return nil
}

// Stop stops the unified API server
func (api *UnifiedAPI) Stop() error {
	if !api.isRunning {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := api.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown unified API server: %w", err)
	}

	api.isRunning = false
	log.Printf("Unified API server stopped")
	return nil
}

// handleOracleStatus returns the overall Oracle status
func (api *UnifiedAPI) handleOracleStatus(w http.ResponseWriter, r *http.Request) {
	var services []ServiceStatus

	// Get Node.js services status
	if api.nodeJSManager != nil {
		nodeJSStatuses := api.nodeJSManager.GetServiceStatuses()
		for _, status := range nodeJSStatuses {
			services = append(services, ServiceStatus{
				Name:      status.Name,
				Type:      "nodejs",
				Running:   status.Running,
				Port:      status.Port,
				PID:       status.PID,
				StartTime: status.StartTime,
				Error:     status.Error,
			})
		}
	}

	// Get binary services status
	if api.binaryManager != nil {
		binaryStatuses := api.binaryManager.GetServiceStatuses()
		for _, status := range binaryStatuses {
			services = append(services, ServiceStatus{
				Name:      status.Name,
				Type:      "binary",
				Running:   status.Running,
				Port:      status.Port,
				PID:       status.PID,
				StartTime: status.StartTime,
				Error:     status.Error,
			})
		}
	}

	runningCount := 0
	for _, service := range services {
		if service.Running {
			runningCount++
		}
	}

	role := "client"
	if api.config.IsRoot {
		role = "root"
	} else if api.config.IsBootnode {
		role = "bootnode"
	}

	status := OracleStatus{
		ChainID:         api.config.ChainID,
		Role:            role,
		IsRoot:          api.config.IsRoot,
		IsBootnode:      api.config.IsBootnode,
		TestnetEnabled:  api.config.Testnet.Enabled,
		Services:        services,
		TotalServices:   len(services),
		RunningServices: runningCount,
		LastUpdated:     time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleOracleInfo returns basic Oracle information
func (api *UnifiedAPI) handleOracleInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"chain_id":        api.config.ChainID,
		"version":         "2.0.0",
		"role":            config.DetermineRoleFromConfig(api.config).String(),
		"is_root":         api.config.IsRoot,
		"is_bootnode":     api.config.IsBootnode,
		"testnet_enabled": api.config.Testnet.Enabled,
		"api_version":     "v1",
		"timestamp":       time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleHealth returns health status
func (api *UnifiedAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(time.Now()).String(), // This would be actual uptime in real implementation
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleListServices returns all services
func (api *UnifiedAPI) handleListServices(w http.ResponseWriter, r *http.Request) {
	var services []ServiceStatus

	// Get Node.js services
	if api.nodeJSManager != nil {
		nodeJSStatuses := api.nodeJSManager.GetServiceStatuses()
		for _, status := range nodeJSStatuses {
			services = append(services, ServiceStatus{
				Name:      status.Name,
				Type:      "nodejs",
				Running:   status.Running,
				Port:      status.Port,
				PID:       status.PID,
				StartTime: status.StartTime,
				Error:     status.Error,
			})
		}
	}

	// Get binary services
	if api.binaryManager != nil {
		binaryStatuses := api.binaryManager.GetServiceStatuses()
		for _, status := range binaryStatuses {
			services = append(services, ServiceStatus{
				Name:      status.Name,
				Type:      "binary",
				Running:   status.Running,
				Port:      status.Port,
				PID:       status.PID,
				StartTime: status.StartTime,
				Error:     status.Error,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"services": services,
		"total":    len(services),
	})
}

// handleStartService starts a specific service
func (api *UnifiedAPI) handleStartService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["name"]

	var err error

	// Try Node.js services first
	if api.nodeJSManager != nil {
		err = api.nodeJSManager.StartService(serviceName)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "success",
				"message": fmt.Sprintf("Service %s started successfully", serviceName),
			})
			return
		}
	}

	// Try binary services
	if api.binaryManager != nil {
		err = api.binaryManager.StartService(serviceName)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "success",
				"message": fmt.Sprintf("Service %s started successfully", serviceName),
			})
			return
		}
	}

	http.Error(w, fmt.Sprintf("Failed to start service %s: %v", serviceName, err), http.StatusInternalServerError)
}

// handleStopService stops a specific service
func (api *UnifiedAPI) handleStopService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["name"]

	var err error

	// Try Node.js services first
	if api.nodeJSManager != nil {
		err = api.nodeJSManager.StopService(serviceName)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "success",
				"message": fmt.Sprintf("Service %s stopped successfully", serviceName),
			})
			return
		}
	}

	// Try binary services
	if api.binaryManager != nil {
		err = api.binaryManager.StopService(serviceName)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "success",
				"message": fmt.Sprintf("Service %s stopped successfully", serviceName),
			})
			return
		}
	}

	http.Error(w, fmt.Sprintf("Failed to stop service %s: %v", serviceName, err), http.StatusInternalServerError)
}

// handleRestartService restarts a specific service
func (api *UnifiedAPI) handleRestartService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["name"]

	var err error

	// Try binary services first (they have restart capability)
	if api.binaryManager != nil {
		err = api.binaryManager.RestartService(serviceName)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "success",
				"message": fmt.Sprintf("Service %s restarted successfully", serviceName),
			})
			return
		}
	}

	// For Node.js services, do stop then start
	if api.nodeJSManager != nil {
		if stopErr := api.nodeJSManager.StopService(serviceName); stopErr == nil {
			time.Sleep(2 * time.Second) // Wait before restart
			if startErr := api.nodeJSManager.StartService(serviceName); startErr == nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"status":  "success",
					"message": fmt.Sprintf("Service %s restarted successfully", serviceName),
				})
				return
			}
		}
	}

	http.Error(w, fmt.Sprintf("Failed to restart service %s: %v", serviceName, err), http.StatusInternalServerError)
}
