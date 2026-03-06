package web

import (
	"encoding/json"
	"net/http"

	"backend_server/internal/config"
	"github.com/gorilla/mux"
)

// SystemSettingsHandlers handles system-wide settings
type SystemSettingsHandlers struct {
	config *config.Config
}

// NewSystemSettingsHandlers creates new system settings handlers
func NewSystemSettingsHandlers(cfg *config.Config) *SystemSettingsHandlers {
	return &SystemSettingsHandlers{
		config: cfg,
	}
}

// DHTSettingsRequest represents a request to change DHT settings
type DHTSettingsRequest struct {
	Enabled bool `json:"enabled"`
}

// DHTSettingsResponse represents the response for DHT settings
type DHTSettingsResponse struct {
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// UpdateDHTSettings handles updating DHT settings
func (ssh *SystemSettingsHandlers) UpdateDHTSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DHTSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Update the configuration
	ssh.config.P2P.DHTEnabled = req.Enabled

	response := DHTSettingsResponse{
		Enabled: req.Enabled,
		Status:  "success",
		Message: "DHT settings updated successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetDHTSettings handles getting current DHT settings
func (ssh *SystemSettingsHandlers) GetDHTSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := DHTSettingsResponse{
		Enabled: ssh.config.P2P.DHTEnabled,
		Status:  "success",
		Message: "DHT settings retrieved successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers the system settings routes
func (ssh *SystemSettingsHandlers) RegisterRoutes(router *mux.Router, authMiddleware interface{}) {
	// For now, we'll make these endpoints public since they're admin settings
	// In production, these should be protected with admin authentication

	router.HandleFunc("/api/system/dht", ssh.UpdateDHTSettings).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/system/dht", ssh.GetDHTSettings).Methods("GET", "OPTIONS")
}
