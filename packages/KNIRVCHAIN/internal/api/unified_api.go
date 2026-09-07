package api

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/internal/auth"
	ulorastore "KNIRVCHAIN/internal/ulora"
)

// UnifiedAPI provides a single interface for all Oracle operations
type UnifiedAPI struct {
	config       *config.Config
	server       *http.Server
	router       *mux.Router
	isRunning    bool
	tokenManager *auth.TokenManager
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
func NewUnifiedAPI(cfg *config.Config) *UnifiedAPI {
	api := &UnifiedAPI{
		config:       cfg,
		router:       mux.NewRouter(),
		isRunning:    false,
		tokenManager: auth.NewTokenManager("knirvchain-secret-key"),
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

	// Node.js services (removed - moved elsewhere)
	// Binary services (removed - moved elsewhere)

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

	// Economics integration (moved elsewhere)

	// Health check
	api.router.HandleFunc("/api/health", api.handleHealth).Methods("GET")

	// Auth endpoints (public)
	api.router.HandleFunc("/api/auth/token", api.handleGenerateToken).Methods("POST")

	// Protected mining endpoints
	api.router.HandleFunc("/api/mining/propose", api.authMiddleware(api.handleMiningProposal)).Methods("POST")
	api.router.HandleFunc("/api/mining/validate", api.authMiddleware(api.handleMiningValidation)).Methods("POST")

	// Content-addressed universal adapter bundles. Bundle hashes are validated
	// before touching the filesystem, so this route cannot escape the store.
	api.router.HandleFunc("/api/ulora/{hash}", api.handleULoRABundle).Methods("GET")

	// Static file serving for Web GUI
	api.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./webGUI/build/")))
}

func (api *UnifiedAPI) uloraStore() (*ulorastore.Store, error) {
	if dir := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); dir != "" {
		return ulorastore.NewStore(dir)
	}
	if api.config != nil && strings.TrimSpace(api.config.BlockchainDatabasePath) != "" {
		return ulorastore.NewStore(filepath.Dir(api.config.BlockchainDatabasePath))
	}
	return ulorastore.NewStore(filepath.Join(os.TempDir(), "knirvchain"))
}

func (api *UnifiedAPI) handleULoRABundle(w http.ResponseWriter, r *http.Request) {
	store, err := api.uloraStore()
	if err != nil {
		http.Error(w, "bundle store unavailable", http.StatusInternalServerError)
		return
	}
	f, err := store.Open(mux.Vars(r)["hash"])
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "invalid bundle hash", http.StatusBadRequest)
		}
		return
	}
	defer f.Close()
	if r.URL.Query().Get("manifest") == "1" {
		manifest, err := manifestFromBundle(f)
		if err != nil {
			http.Error(w, "bundle manifest unavailable", http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(manifest)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.ulora+gzip")
	_, _ = io.Copy(w, f)
}

func manifestFromBundle(f *os.File) ([]byte, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	var reader io.Reader = f
	if gz, err := gzip.NewReader(f); err == nil {
		defer gz.Close()
		reader = gz
	} else if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	tr := tar.NewReader(reader)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("manifest.json not found")
		}
		if err != nil {
			return nil, err
		}
		if h.Name == "manifest.json" {
			return io.ReadAll(io.LimitReader(tr, 4<<20))
		}
	}
}

// authMiddleware wraps handlers with JWT authentication
func (api *UnifiedAPI) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		claims, err := api.tokenManager.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "claims", claims)
		next(w, r.WithContext(ctx))
	}
}

// handleGenerateToken generates a JWT token
func (api *UnifiedAPI) handleGenerateToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID      string   `json:"user_id"`
		WalletAddr  string   `json:"wallet_addr"`
		Permissions []string `json:"permissions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	perms := make([]auth.Permission, len(req.Permissions))
	for i, p := range req.Permissions {
		perms[i] = auth.Permission(p)
	}

	token, err := api.tokenManager.GenerateToken(req.UserID, req.WalletAddr, perms)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// handleMiningProposal handles mining proposal submissions (placeholder)
func (api *UnifiedAPI) handleMiningProposal(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleMiningValidation handles mining validation (placeholder)
func (api *UnifiedAPI) handleMiningValidation(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

	// Node.js and binary services have been moved elsewhere
	// Return empty services list

	runningCount := 0

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

	// Node.js and binary services have been moved elsewhere
	// Return empty services list

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"services": services,
		"total":    len(services),
		"message":  "Services have been moved to separate components",
	})
}

// handleStartService starts a specific service
func (api *UnifiedAPI) handleStartService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["name"]

	// Node.js and binary services have been moved elsewhere
	http.Error(w, fmt.Sprintf("Service %s management has been moved to separate components", serviceName), http.StatusGone)
}

// handleStopService stops a specific service
func (api *UnifiedAPI) handleStopService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["name"]

	// Node.js and binary services have been moved elsewhere
	http.Error(w, fmt.Sprintf("Service %s management has been moved to separate components", serviceName), http.StatusGone)
}

// handleRestartService restarts a specific service
func (api *UnifiedAPI) handleRestartService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["name"]

	// Node.js and binary services have been moved elsewhere
	http.Error(w, fmt.Sprintf("Service %s management has been moved to separate components", serviceName), http.StatusGone)
}
