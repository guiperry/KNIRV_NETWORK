package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"nexus-backend/internal/services/cde"
	dataengine "nexus-backend/internal/services/data-engine"
	"nexus-backend/internal/services/inference"
	"nexus-backend/pkg/host"
)

// APIServer represents the REST API server
type APIServer struct {
	// Core services
	hostController   *host.HostController
	dataEngine       *dataengine.BuntDBDataEngine
	agentServer      *agentserver.AgentServer
	inferenceService *inference.AdaptiveHostService
	cdeService       *cde.CDEService

	// Server configuration
	port   int
	server *http.Server
	router *mux.Router

	// Middleware
	authMiddleware *AuthMiddleware

	// State
	isRunning bool
}

// APIConfig contains configuration for the API server
type APIConfig struct {
	Port           int           `yaml:"port"`
	Host           string        `yaml:"host"`
	EnableCORS     bool          `yaml:"enable_cors"`
	CORSOrigins    []string      `yaml:"cors_origins"`
	JWTSecret      string        `yaml:"jwt_secret"`
	RateLimitRPS   int           `yaml:"rate_limit_rps"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Offset   int `json:"offset"`
	Limit    int `json:"limit"`
}

// NewAPIServer creates a new API server
func NewAPIServer(
	hostController *host.HostController,
	dataEngine *dataengine.BuntDBDataEngine,
	agentServer *agentserver.AgentServer,
	inferenceService *inference.AdaptiveHostService,
	cdeService *cde.CDEService,
	config APIConfig,
) (*APIServer, error) {

	// Create auth middleware
	authMiddleware, err := NewAuthMiddleware(dataEngine, config.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth middleware: %w", err)
	}

	server := &APIServer{
		hostController:   hostController,
		dataEngine:       dataEngine,
		agentServer:      agentServer,
		inferenceService: inferenceService,
		cdeService:       cdeService,
		port:             config.Port,
		authMiddleware:   authMiddleware,
	}

	// Setup router
	server.setupRouter(config)

	// Create HTTP server
	server.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      server.router,
		ReadTimeout:  config.RequestTimeout,
		WriteTimeout: config.RequestTimeout,
		IdleTimeout:  60 * time.Second,
	}

	return server, nil
}

// setupRouter sets up the API routes
func (s *APIServer) setupRouter(config APIConfig) {
	s.router = mux.NewRouter()

	// Setup CORS if enabled
	if config.EnableCORS {
		c := cors.New(cors.Options{
			AllowedOrigins:   config.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		})
		s.router.Use(c.Handler)
	}

	// Global middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.requestIDMiddleware)
	s.router.Use(s.recoveryMiddleware)

	// API version prefix
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Public routes (no authentication required)
	s.setupPublicRoutes(api)

	// Protected routes (authentication required)
	protected := api.PathPrefix("").Subrouter()
	protected.Use(s.authMiddleware.Authenticate)
	s.setupProtectedRoutes(protected)

	// Admin routes (admin authentication required)
	admin := api.PathPrefix("/admin").Subrouter()
	admin.Use(s.authMiddleware.Authenticate)
	admin.Use(s.authMiddleware.RequireRole("admin"))
	s.setupAdminRoutes(admin)
}

// setupPublicRoutes sets up public API routes
func (s *APIServer) setupPublicRoutes(router *mux.Router) {
	// Health check
	router.HandleFunc("/health", s.handleHealth).Methods("GET")
	router.HandleFunc("/status", s.handleStatus).Methods("GET")

	// Authentication
	router.HandleFunc("/auth/login", s.handleLogin).Methods("POST")
	router.HandleFunc("/auth/register", s.handleRegister).Methods("POST")
	router.HandleFunc("/auth/refresh", s.handleRefreshToken).Methods("POST")

	// Public system information
	router.HandleFunc("/system/info", s.handleSystemInfo).Methods("GET")
}

// setupProtectedRoutes sets up protected API routes
func (s *APIServer) setupProtectedRoutes(router *mux.Router) {
	// User management
	router.HandleFunc("/users/me", s.handleGetCurrentUser).Methods("GET")
	router.HandleFunc("/users/me", s.handleUpdateCurrentUser).Methods("PUT")
	router.HandleFunc("/users/me/sessions", s.handleGetUserSessions).Methods("GET")

	// Agent management
	router.HandleFunc("/agents", s.handleListAgents).Methods("GET")
	router.HandleFunc("/agents", s.handleCreateAgent).Methods("POST")
	router.HandleFunc("/agents/{id}", s.handleGetAgent).Methods("GET")
	router.HandleFunc("/agents/{id}", s.handleUpdateAgent).Methods("PUT")
	router.HandleFunc("/agents/{id}", s.handleDeleteAgent).Methods("DELETE")
	router.HandleFunc("/agents/{id}/start", s.handleStartAgent).Methods("POST")
	router.HandleFunc("/agents/{id}/stop", s.handleStopAgent).Methods("POST")
	router.HandleFunc("/agents/{id}/status", s.handleGetAgentStatus).Methods("GET")

	// DVE Nodes
	router.HandleFunc("/dve-nodes", s.handleListDVENodes).Methods("GET")
	router.HandleFunc("/dve-nodes/{id}", s.handleGetDVENode).Methods("GET")
	router.HandleFunc("/dve-nodes/{id}/status", s.handleGetDVENodeStatus).Methods("GET")

	// Validation tasks
	router.HandleFunc("/validation/tasks", s.handleListValidationTasks).Methods("GET")
	router.HandleFunc("/validation/tasks", s.handleCreateValidationTask).Methods("POST")
	router.HandleFunc("/validation/tasks/{id}", s.handleGetValidationTask).Methods("GET")
	router.HandleFunc("/validation/tasks/{id}", s.handleUpdateValidationTask).Methods("PUT")
	router.HandleFunc("/validation/tasks/{id}/results", s.handleGetValidationResults).Methods("GET")

	// CDE (Cloud Development Environment)
	router.HandleFunc("/cde/environments", s.handleListCDEEnvironments).Methods("GET")
	router.HandleFunc("/cde/environments", s.handleCreateCDEEnvironment).Methods("POST")
	router.HandleFunc("/cde/environments/{id}", s.handleGetCDEEnvironment).Methods("GET")
	router.HandleFunc("/cde/environments/{id}", s.handleDeleteCDEEnvironment).Methods("DELETE")
	router.HandleFunc("/cde/environments/{id}/start", s.handleStartCDEEnvironment).Methods("POST")
	router.HandleFunc("/cde/environments/{id}/stop", s.handleStopCDEEnvironment).Methods("POST")
	router.HandleFunc("/cde/sessions", s.handleListCDESessions).Methods("GET")
	router.HandleFunc("/cde/sessions", s.handleCreateCDESession).Methods("POST")
	router.HandleFunc("/cde/projects", s.handleListCDEProjects).Methods("GET")
	router.HandleFunc("/cde/projects", s.handleCreateCDEProject).Methods("POST")

	// Inference service
	router.HandleFunc("/inference/models", s.handleListModels).Methods("GET")
	router.HandleFunc("/inference/models/{id}/finetune", s.handleStartFineTuning).Methods("POST")
	router.HandleFunc("/inference/models/{id}/status", s.handleGetModelStatus).Methods("GET")

	// Metrics and monitoring
	router.HandleFunc("/metrics", s.handleGetMetrics).Methods("GET")
	router.HandleFunc("/alerts", s.handleGetAlerts).Methods("GET")
	router.HandleFunc("/reports", s.handleGetReports).Methods("GET")
}

// setupAdminRoutes sets up admin-only API routes
func (s *APIServer) setupAdminRoutes(router *mux.Router) {
	// User management (admin)
	router.HandleFunc("/users", s.handleListAllUsers).Methods("GET")
	router.HandleFunc("/users/{id}", s.handleGetUser).Methods("GET")
	router.HandleFunc("/users/{id}", s.handleUpdateUser).Methods("PUT")
	router.HandleFunc("/users/{id}", s.handleDeleteUser).Methods("DELETE")
	router.HandleFunc("/users/{id}/sessions", s.handleGetUserSessionsAdmin).Methods("GET")

	// System management
	router.HandleFunc("/system/status", s.handleSystemStatus).Methods("GET")
	router.HandleFunc("/system/config", s.handleGetSystemConfig).Methods("GET")
	router.HandleFunc("/system/config", s.handleUpdateSystemConfig).Methods("PUT")

	// DVE Node management (admin)
	router.HandleFunc("/dve-nodes", s.handleCreateDVENode).Methods("POST")
	router.HandleFunc("/dve-nodes/{id}", s.handleUpdateDVENode).Methods("PUT")
	router.HandleFunc("/dve-nodes/{id}", s.handleDeleteDVENode).Methods("DELETE")

	// System reports
	router.HandleFunc("/reports/system", s.handleGetSystemReports).Methods("GET")
	router.HandleFunc("/reports/export", s.handleExportReports).Methods("GET")

	// Database management
	router.HandleFunc("/database/stats", s.handleDatabaseStats).Methods("GET")
	router.HandleFunc("/database/backup", s.handleDatabaseBackup).Methods("POST")
}

// Start starts the API server
func (s *APIServer) Start() error {
	if s.isRunning {
		return fmt.Errorf("API server is already running")
	}

	log.Printf("Starting API server on port %d", s.port)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	s.isRunning = true
	log.Printf("API server started successfully on port %d", s.port)

	return nil
}

// Stop stops the API server
func (s *APIServer) Stop() error {
	if !s.isRunning {
		return nil
	}

	log.Println("Stopping API server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown API server: %w", err)
	}

	s.isRunning = false
	log.Println("API server stopped successfully")

	return nil
}

// IsRunning returns whether the API server is running
func (s *APIServer) IsRunning() bool {
	return s.isRunning
}

// Helper methods

// writeJSON writes a JSON response
func (s *APIServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := APIResponse{
		Success:   status < 400,
		Data:      data,
		Timestamp: time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// writeError writes an error response
func (s *APIServer) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := APIResponse{
		Success:   false,
		Error:     message,
		Timestamp: time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON error response: %v", err)
	}
}

// parsePagination parses pagination parameters from request
func (s *APIServer) parsePagination(r *http.Request) PaginationParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   offset,
		Limit:    pageSize,
	}
}
