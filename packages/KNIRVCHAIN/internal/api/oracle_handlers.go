package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// handleServiceStatus returns status for a specific service (stub - services moved elsewhere)
func (api *UnifiedAPI) handleServiceStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["name"]

	// Services moved elsewhere - return inactive status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":    serviceName,
		"type":    "unknown",
		"running": false,
		"status":  "inactive",
		"error":   "Service moved to separate components",
	})
}

// handleTunnelStatus returns tunnel service status (stub - services moved elsewhere)
func (api *UnifiedAPI) handleTunnelStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_running": false,
		"status":          "inactive",
		"message":         "Tunnel services moved to separate components",
	})
}

// handleTunnelConnections returns active tunnel connections (stub - services moved elsewhere)
func (api *UnifiedAPI) handleTunnelConnections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connections":    []map[string]interface{}{},
		"total":          0,
		"active_tunnels": 0,
		"message":        "Tunnel services moved to separate components",
	})
}

// handleWalletStatus returns wallet service status (stub - services moved elsewhere)
func (api *UnifiedAPI) handleWalletStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "inactive",
		"connected": false,
		"network":   api.config.ChainID,
		"message":   "Wallet services moved to separate components",
		"last_sync": time.Now(),
	})
}

// handleWalletBalance returns wallet balance (stub - services moved elsewhere)
func (api *UnifiedAPI) handleWalletBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"balance": map[string]interface{}{
			"message": "Wallet services moved to separate components",
		},
		"last_updated": time.Now(),
	})
}

// handlePaymentStatus returns payment gateway status (stub - services moved elsewhere)
func (api *UnifiedAPI) handlePaymentStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_running": false,
		"status":          "inactive",
		"message":         "Payment services moved to separate components",
	})
}

// handlePaymentHistory returns payment history (stub - services moved elsewhere)
func (api *UnifiedAPI) handlePaymentHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payments": []map[string]interface{}{},
		"total":    0,
		"message":  "Payment services moved to separate components",
	})
}
