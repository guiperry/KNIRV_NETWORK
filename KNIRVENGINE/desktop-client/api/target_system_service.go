package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TargetSystemType represents the type of target system
type TargetSystemType string

const (
	TargetTypeBrowser     TargetSystemType = "browser"
	TargetTypeFilesystem  TargetSystemType = "filesystem"
	TargetTypeApplication TargetSystemType = "application"
	TargetTypeSystem      TargetSystemType = "system"
	TargetTypeNetwork     TargetSystemType = "network"
	TargetTypeDatabase    TargetSystemType = "database"
	TargetTypeMobile      TargetSystemType = "mobile"
)

// TargetSystemStatus represents the connection status
type TargetSystemStatus string

const (
	StatusDisconnected TargetSystemStatus = "disconnected"
	StatusConnected    TargetSystemStatus = "connected"
	StatusConnecting   TargetSystemStatus = "connecting"
	StatusError        TargetSystemStatus = "error"
	StatusMonitoring   TargetSystemStatus = "monitoring"
	StatusLimited      TargetSystemStatus = "limited"
)

// TargetSystem represents a target system configuration
type TargetSystem struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Type             TargetSystemType       `json:"type"`
	Description      string                 `json:"description"`
	Status           TargetSystemStatus     `json:"status"`
	Capabilities     []string               `json:"capabilities"`
	Permissions      []string               `json:"permissions"`
	Security         string                 `json:"security"`
	ConnectionMethod string                 `json:"connectionMethod"`
	DataAccess       []string               `json:"dataAccess"`
	LastActivity     time.Time              `json:"lastActivity"`
	ActiveAgents     int                    `json:"activeAgents"`
	OwnerID          int64                  `json:"ownerId"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
	Config           map[string]interface{} `json:"config"`
	ConnectionInfo   map[string]interface{} `json:"connectionInfo"`
}

// TargetSystemConnection represents an active connection to a target system
type TargetSystemConnection interface {
	// Connect establishes the connection
	Connect(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect(ctx context.Context) error

	// IsConnected returns the connection status
	IsConnected() bool

	// GetCapabilities returns available capabilities
	GetCapabilities() []string

	// Execute executes a command/operation on the target system
	Execute(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error)

	// GetStatus returns detailed status information
	GetStatus() map[string]interface{}

	// GetType returns the target system type
	GetType() TargetSystemType
}

// TargetSystemService manages target system connections
type TargetSystemService struct {
	connections     map[string]TargetSystemConnection
	targets         map[string]*TargetSystem
	securityManager *ConnectionSecurityManager
	mutex           sync.RWMutex
}

// NewTargetSystemService creates a new target system service
func NewTargetSystemService() *TargetSystemService {
	return &TargetSystemService{
		connections:     make(map[string]TargetSystemConnection),
		targets:         make(map[string]*TargetSystem),
		securityManager: NewConnectionSecurityManager(),
	}
}

// RegisterHandlers registers the target system API handlers
func (s *TargetSystemService) RegisterHandlers(router *mux.Router) {
	router.HandleFunc("/api/v1/targets", s.handleListTargets).Methods("GET")
	router.HandleFunc("/api/v1/targets", s.handleCreateTarget).Methods("POST")
	router.HandleFunc("/api/v1/targets/{id}", s.handleGetTarget).Methods("GET")
	router.HandleFunc("/api/v1/targets/{id}", s.handleUpdateTarget).Methods("PUT")
	router.HandleFunc("/api/v1/targets/{id}", s.handleDeleteTarget).Methods("DELETE")
	router.HandleFunc("/api/v1/targets/{id}/connect", s.handleConnect).Methods("POST")
	router.HandleFunc("/api/v1/targets/{id}/disconnect", s.handleDisconnect).Methods("POST")
	router.HandleFunc("/api/v1/targets/{id}/test", s.handleTestConnection).Methods("POST")
	router.HandleFunc("/api/v1/targets/{id}/execute", s.handleExecute).Methods("POST")
	router.HandleFunc("/api/v1/targets/{id}/status", s.handleGetStatus).Methods("GET")
	router.HandleFunc("/api/v1/targets/{id}/security", s.handleCreateSecurityContext).Methods("POST")
	router.HandleFunc("/api/v1/targets/{id}/security/{sessionId}", s.handleRevokeSecurityContext).Methods("DELETE")
}

// CreateTarget creates a new target system
func (s *TargetSystemService) CreateTarget(target *TargetSystem) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Generate ID if not provided
	if target.ID == "" {
		target.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	target.CreatedAt = now
	target.UpdatedAt = now
	target.LastActivity = now

	// Set default status
	if target.Status == "" {
		target.Status = StatusDisconnected
	}

	// Initialize config if nil
	if target.Config == nil {
		target.Config = make(map[string]interface{})
	}
	if target.ConnectionInfo == nil {
		target.ConnectionInfo = make(map[string]interface{})
	}

	// Store the target
	s.targets[target.ID] = target

	return nil
}

// GetTarget retrieves a target by ID
func (s *TargetSystemService) GetTarget(id string) (*TargetSystem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	target, ok := s.targets[id]
	if !ok {
		return nil, fmt.Errorf("target not found: %s", id)
	}

	return target, nil
}

// ListTargets retrieves all targets for a user
func (s *TargetSystemService) ListTargets(userID int64) ([]*TargetSystem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var targets []*TargetSystem
	for _, target := range s.targets {
		if target.OwnerID == userID {
			targets = append(targets, target)
		}
	}

	return targets, nil
}

// UpdateTarget updates an existing target
func (s *TargetSystemService) UpdateTarget(target *TargetSystem) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	existing, ok := s.targets[target.ID]
	if !ok {
		return fmt.Errorf("target not found: %s", target.ID)
	}

	// Preserve creation time and update timestamp
	target.CreatedAt = existing.CreatedAt
	target.UpdatedAt = time.Now()

	// Store the updated target
	s.targets[target.ID] = target

	return nil
}

// DeleteTarget deletes a target
func (s *TargetSystemService) DeleteTarget(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Disconnect if connected
	if conn, ok := s.connections[id]; ok {
		conn.Disconnect(context.Background())
		delete(s.connections, id)
	}

	// Delete the target
	delete(s.targets, id)

	return nil
}

// ConnectTarget establishes a connection to a target system
func (s *TargetSystemService) ConnectTarget(ctx context.Context, id string) error {
	return s.ConnectTargetWithSecurity(ctx, id, "", SecurityLevelBasic)
}

// ConnectTargetWithSecurity establishes a secure connection to a target system
func (s *TargetSystemService) ConnectTargetWithSecurity(ctx context.Context, id string, sessionID string, securityLevel SecurityLevel) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	target, ok := s.targets[id]
	if !ok {
		return fmt.Errorf("target not found: %s", id)
	}

	// Check if already connected
	if conn, ok := s.connections[id]; ok && conn.IsConnected() {
		return nil // Already connected
	}

	// Create connection based on target type
	conn, err := s.createConnection(target)
	if err != nil {
		target.Status = StatusError
		return fmt.Errorf("failed to create connection: %v", err)
	}

	// Update status to connecting
	target.Status = StatusConnecting
	target.LastActivity = time.Now()

	// Attempt to connect
	if err := conn.Connect(ctx); err != nil {
		target.Status = StatusError
		return fmt.Errorf("failed to connect: %v", err)
	}

	// Wrap connection with security if session ID is provided
	var finalConn TargetSystemConnection = conn
	if sessionID != "" {
		finalConn = NewSecureTargetSystemConnection(conn, s.securityManager, sessionID, securityLevel)
	}

	// Store the connection and update status
	s.connections[id] = finalConn
	target.Status = StatusConnected
	target.LastActivity = time.Now()

	return nil
}

// DisconnectTarget closes a connection to a target system
func (s *TargetSystemService) DisconnectTarget(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	target, ok := s.targets[id]
	if !ok {
		return fmt.Errorf("target not found: %s", id)
	}

	// Check if connected
	conn, ok := s.connections[id]
	if !ok {
		target.Status = StatusDisconnected
		return nil // Already disconnected
	}

	// Disconnect
	if err := conn.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect: %v", err)
	}

	// Remove connection and update status
	delete(s.connections, id)
	target.Status = StatusDisconnected
	target.LastActivity = time.Now()

	return nil
}

// ExecuteOperation executes an operation on a target system
func (s *TargetSystemService) ExecuteOperation(ctx context.Context, id string, operation string, params map[string]interface{}) (interface{}, error) {
	s.mutex.RLock()
	conn, ok := s.connections[id]
	target := s.targets[id]
	s.mutex.RUnlock()

	if !ok || !conn.IsConnected() {
		return nil, fmt.Errorf("target not connected: %s", id)
	}

	// Update last activity
	if target != nil {
		s.mutex.Lock()
		target.LastActivity = time.Now()
		s.mutex.Unlock()
	}

	return conn.Execute(ctx, operation, params)
}

// GetTargetStatus returns the status of a target system
func (s *TargetSystemService) GetTargetStatus(id string) (map[string]interface{}, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	target, ok := s.targets[id]
	if !ok {
		return nil, fmt.Errorf("target not found: %s", id)
	}

	status := map[string]interface{}{
		"id":           target.ID,
		"name":         target.Name,
		"type":         target.Type,
		"status":       target.Status,
		"lastActivity": target.LastActivity,
		"activeAgents": target.ActiveAgents,
	}

	// Add connection-specific status if connected
	if conn, ok := s.connections[id]; ok && conn.IsConnected() {
		connStatus := conn.GetStatus()
		for k, v := range connStatus {
			status[k] = v
		}
	}

	return status, nil
}

// createConnection creates a connection based on target type
func (s *TargetSystemService) createConnection(target *TargetSystem) (TargetSystemConnection, error) {
	switch target.Type {
	case TargetTypeBrowser:
		return NewBrowserConnection(target)
	case TargetTypeFilesystem:
		return NewFilesystemConnection(target)
	case TargetTypeDatabase:
		return NewDatabaseConnection(target)
	case TargetTypeNetwork:
		return NewNetworkConnection(target)
	case TargetTypeApplication:
		return NewApplicationConnection(target)
	case TargetTypeSystem:
		return NewSystemConnection(target)
	case TargetTypeMobile:
		return NewMobileConnection(target)
	default:
		return nil, fmt.Errorf("unsupported target type: %s", target.Type)
	}
}

// HTTP Handlers

// handleListTargets handles GET /api/v1/targets
func (s *TargetSystemService) handleListTargets(w http.ResponseWriter, r *http.Request) {
	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	targets, err := s.ListTargets(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list targets: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"targets": targets,
	})
}

// handleCreateTarget handles POST /api/v1/targets
func (s *TargetSystemService) handleCreateTarget(w http.ResponseWriter, r *http.Request) {
	var target TargetSystem
	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// For simplicity, we'll use a fixed user ID
	userID := int64(1)
	target.OwnerID = userID

	if err := s.CreateTarget(&target); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create target: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(target)
}

// handleGetTarget handles GET /api/v1/targets/{id}
func (s *TargetSystemService) handleGetTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	target, err := s.GetTarget(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Target not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(target)
}

// handleUpdateTarget handles PUT /api/v1/targets/{id}
func (s *TargetSystemService) handleUpdateTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var target TargetSystem
	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	target.ID = id
	if err := s.UpdateTarget(&target); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update target: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(target)
}

// handleDeleteTarget handles DELETE /api/v1/targets/{id}
func (s *TargetSystemService) handleDeleteTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.DeleteTarget(id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete target: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleConnect handles POST /api/v1/targets/{id}/connect
func (s *TargetSystemService) handleConnect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.ConnectTarget(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "connected",
	})
}

// handleDisconnect handles POST /api/v1/targets/{id}/disconnect
func (s *TargetSystemService) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.DisconnectTarget(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to disconnect: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "disconnected",
	})
}

// handleTestConnection handles POST /api/v1/targets/{id}/test
func (s *TargetSystemService) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	target, err := s.GetTarget(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Target not found: %v", err), http.StatusNotFound)
		return
	}

	// Create a temporary connection to test
	conn, err := s.createConnection(target)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create connection: %v", err), http.StatusInternalServerError)
		return
	}

	// Test the connection
	if err := conn.Connect(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Disconnect the test connection
	conn.Disconnect(r.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleExecute handles POST /api/v1/targets/{id}/execute
func (s *TargetSystemService) handleExecute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var request struct {
		Operation string                 `json:"operation"`
		Params    map[string]interface{} `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	result, err := s.ExecuteOperation(r.Context(), id, request.Operation, request.Params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute operation: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"result": result,
	})
}

// handleGetStatus handles GET /api/v1/targets/{id}/status
func (s *TargetSystemService) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	status, err := s.GetTargetStatus(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleCreateSecurityContext handles POST /api/v1/targets/{id}/security
func (s *TargetSystemService) handleCreateSecurityContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var request struct {
		UserID      int64                  `json:"userId"`
		Permissions []Permission           `json:"permissions"`
		Constraints map[string]interface{} `json:"constraints"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Check if target exists
	if _, err := s.GetTarget(id); err != nil {
		http.Error(w, fmt.Sprintf("Target not found: %v", err), http.StatusNotFound)
		return
	}

	// Create security context
	ctx, err := s.securityManager.CreateSecurityContext(request.UserID, request.Permissions, request.Constraints)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create security context: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionId":   ctx.SessionID,
		"expiresAt":   ctx.ExpiresAt,
		"permissions": ctx.Permissions,
	})
}

// handleRevokeSecurityContext handles DELETE /api/v1/targets/{id}/security/{sessionId}
func (s *TargetSystemService) handleRevokeSecurityContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	if err := s.securityManager.RevokeSession(sessionID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to revoke session: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ClearDemoData removes all demo/sample target systems from the service
func (s *TargetSystemService) ClearDemoData() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Disconnect all connections first
	for id, conn := range s.connections {
		if conn.IsConnected() {
			conn.Disconnect(context.Background())
		}
		delete(s.connections, id)
	}

	// Clear all targets
	s.targets = make(map[string]*TargetSystem)
	log.Println("All target system demo data cleared")
	return nil
}
