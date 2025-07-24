package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// WebConnection represents a connection to a third-party service
type WebConnection struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Provider      string                 `json:"provider"`
	Status        string                 `json:"status"`
	AuthType      string                 `json:"auth_type"` // "oauth", "api_key", etc.
	Credentials   map[string]interface{} `json:"credentials,omitempty"`
	Scopes        []string               `json:"scopes,omitempty"`
	LastConnected time.Time              `json:"last_connected"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	OwnerID       int64                  `json:"owner_id"`
}

// WebConnectionsService manages connections to third-party services
type WebConnectionsService struct {
	connections map[string]*WebConnection
	mutex       sync.RWMutex
}

// NewWebConnectionsService creates a new web connections service
func NewWebConnectionsService() *WebConnectionsService {
	return &WebConnectionsService{
		connections: make(map[string]*WebConnection),
	}
}

// GetConnection retrieves a connection by ID
func (s *WebConnectionsService) GetConnection(id string) (*WebConnection, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	connection, exists := s.connections[id]
	if !exists {
		return nil, fmt.Errorf("connection not found: %s", id)
	}

	return connection, nil
}

// ListConnections retrieves all connections for a user
func (s *WebConnectionsService) ListConnections(userID int64) ([]*WebConnection, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var connections []*WebConnection
	for _, connection := range s.connections {
		if connection.OwnerID == userID {
			connections = append(connections, connection)
		}
	}

	return connections, nil
}

// CreateConnection creates a new connection
func (s *WebConnectionsService) CreateConnection(connection *WebConnection) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Set timestamps
	now := time.Now()
	connection.CreatedAt = now
	connection.UpdatedAt = now

	// Store connection
	s.connections[connection.ID] = connection

	return nil
}

// UpdateConnection updates an existing connection
func (s *WebConnectionsService) UpdateConnection(connection *WebConnection) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if connection exists
	_, exists := s.connections[connection.ID]
	if !exists {
		return fmt.Errorf("connection not found: %s", connection.ID)
	}

	// Update timestamp
	connection.UpdatedAt = time.Now()

	// Update connection
	s.connections[connection.ID] = connection

	return nil
}

// DeleteConnection deletes a connection
func (s *WebConnectionsService) DeleteConnection(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if connection exists
	_, exists := s.connections[id]
	if !exists {
		return fmt.Errorf("connection not found: %s", id)
	}

	// Delete connection
	delete(s.connections, id)

	return nil
}

// TestConnection tests a connection to verify it's working
func (s *WebConnectionsService) TestConnection(id string) (bool, error) {
	s.mutex.RLock()
	connection, exists := s.connections[id]
	s.mutex.RUnlock()

	if !exists {
		return false, fmt.Errorf("connection not found: %s", id)
	}

	// In a real implementation, this would test the connection by making an API call
	// For now, we'll simulate a successful test
	log.Printf("Testing connection %s to %s", id, connection.Provider)

	// Update last connected timestamp
	s.mutex.Lock()
	connection.LastConnected = time.Now()
	connection.Status = "connected"
	s.mutex.Unlock()

	return true, nil
}

// RegisterHandlers registers the web connections API handlers
func (s *WebConnectionsService) RegisterHandlers(router *mux.Router) {
	router.HandleFunc("/api/v1/web-connections", s.handleListConnections).Methods("GET")
	router.HandleFunc("/api/v1/web-connections", s.handleCreateConnection).Methods("POST")
	router.HandleFunc("/api/v1/web-connections/{id}", s.handleGetConnection).Methods("GET")
	router.HandleFunc("/api/v1/web-connections/{id}", s.handleUpdateConnection).Methods("PUT")
	router.HandleFunc("/api/v1/web-connections/{id}", s.handleDeleteConnection).Methods("DELETE")
	router.HandleFunc("/api/v1/web-connections/{id}/test", s.handleTestConnection).Methods("POST")
	router.HandleFunc("/api/v1/web-connections/oauth/callback", s.handleOAuthCallback).Methods("GET")
	// AuthKit token endpoint
	router.HandleFunc("/api/v1/authkit-token", s.handleAuthKitToken).Methods("POST")
}

// handleListConnections handles GET /api/v1/web-connections
func (s *WebConnectionsService) handleListConnections(w http.ResponseWriter, r *http.Request) {
	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	connections, err := s.ListConnections(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list connections: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connections": connections,
	})
}

// handleGetConnection handles GET /api/v1/web-connections/{id}
func (s *WebConnectionsService) handleGetConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	connection, err := s.GetConnection(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get connection: %v", err), http.StatusNotFound)
		return
	}

	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	// Check ownership
	if connection.OwnerID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connection": connection,
	})
}

// handleCreateConnection handles POST /api/v1/web-connections
func (s *WebConnectionsService) handleCreateConnection(w http.ResponseWriter, r *http.Request) {
	var connection WebConnection
	if err := json.NewDecoder(r.Body).Decode(&connection); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	// Set owner ID
	connection.OwnerID = userID

	// Create connection
	if err := s.CreateConnection(&connection); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create connection: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connection": connection,
	})
}

// handleUpdateConnection handles PUT /api/v1/web-connections/{id}
func (s *WebConnectionsService) handleUpdateConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get existing connection
	existingConnection, err := s.GetConnection(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get connection: %v", err), http.StatusNotFound)
		return
	}

	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	// Check ownership
	if existingConnection.OwnerID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse request body
	var connection WebConnection
	if err := json.NewDecoder(r.Body).Decode(&connection); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Ensure ID matches
	connection.ID = id

	// Preserve owner ID and creation time
	connection.OwnerID = existingConnection.OwnerID
	connection.CreatedAt = existingConnection.CreatedAt

	// Update connection
	if err := s.UpdateConnection(&connection); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update connection: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connection": connection,
	})
}

// handleDeleteConnection handles DELETE /api/v1/web-connections/{id}
func (s *WebConnectionsService) handleDeleteConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get existing connection
	existingConnection, err := s.GetConnection(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get connection: %v", err), http.StatusNotFound)
		return
	}

	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	// Check ownership
	if existingConnection.OwnerID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Delete connection
	if err := s.DeleteConnection(id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete connection: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Connection deleted successfully",
	})
}

// handleTestConnection handles POST /api/v1/web-connections/{id}/test
func (s *WebConnectionsService) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get existing connection
	existingConnection, err := s.GetConnection(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get connection: %v", err), http.StatusNotFound)
		return
	}

	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	// Check ownership
	if existingConnection.OwnerID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Test connection
	success, err := s.TestConnection(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to test connection: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
		"message": "Connection test successful",
	})
}

// handleOAuthCallback handles GET /api/v1/web-connections/oauth/callback
func (s *WebConnectionsService) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	error := r.URL.Query().Get("error")

	if error != "" {
		// Handle OAuth error
		http.Error(w, fmt.Sprintf("OAuth error: %s", error), http.StatusBadRequest)
		return
	}

	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// In a real implementation, you would:
	// 1. Validate the state parameter to prevent CSRF
	// 2. Exchange the code for an access token
	// 3. Store the access token securely
	// 4. Redirect the user back to the application

	// For now, we'll just acknowledge the callback
	log.Printf("Received OAuth callback with code: %s and state: %s", code, state)

	// Render a simple HTML page that closes the popup window
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Authorization Successful</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					text-align: center;
					margin-top: 50px;
					background-color: #f5f5f5;
				}
				.container {
					background-color: white;
					border-radius: 8px;
					box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
					padding: 30px;
					max-width: 500px;
					margin: 0 auto;
				}
				h1 {
					color: #4CAF50;
				}
				p {
					margin: 20px 0;
					color: #666;
				}
				button {
					background-color: #4CAF50;
					color: white;
					border: none;
					padding: 10px 20px;
					border-radius: 4px;
					cursor: pointer;
					font-size: 16px;
				}
				button:hover {
					background-color: #45a049;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<h1>Authorization Successful</h1>
				<p>You have successfully authorized the application.</p>
				<p>You can close this window now.</p>
				<button onclick="window.close()">Close Window</button>
			</div>
			<script>
				// Send message to parent window
				window.opener.postMessage({ type: 'oauth_success', code: '` + code + `', state: '` + state + `' }, '*');
				// Auto-close after 3 seconds
				setTimeout(function() {
					window.close();
				}, 3000);
			</script>
		</body>
		</html>
	`))
}

// handleAuthKitToken handles POST /api/v1/authkit-token
// This endpoint generates tokens for PicaOS AuthKit integration
func (s *WebConnectionsService) handleAuthKitToken(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse request body
	var req struct {
		Identity     string `json:"identity"`
		IdentityType string `json:"identityType"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Identity == "" {
		req.Identity = "user_default" // Default identity
	}
	if req.IdentityType == "" {
		req.IdentityType = "user" // Default identity type
	}

	// For now, we'll generate a simple token
	// In a production environment, you would:
	// 1. Validate the user's authentication
	// 2. Generate a secure token using your PICA_SANDBOX_API_KEY
	// 3. Store the token mapping for later use

	token := map[string]interface{}{
		"token":        fmt.Sprintf("authkit_%d_%s", time.Now().Unix(), req.Identity),
		"identity":     req.Identity,
		"identityType": req.IdentityType,
		"expiresAt":    time.Now().Add(24 * time.Hour).Unix(),
		"createdAt":    time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(token); err != nil {
		log.Printf("Error encoding AuthKit token response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Generated AuthKit token for identity: %s (type: %s)", req.Identity, req.IdentityType)
}
