package data_engine

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// DataEngineHandlers contains HTTP handlers for data_engine endpoints
type DataEngineHandlers struct {
	dataEngine *DataEngine
	upgrader   websocket.Upgrader
}

// NewDataEngineHandlers creates a new instance of DataEngineHandlers
func NewDataEngineHandlers(dataEngine *DataEngine) *DataEngineHandlers {
	return &DataEngineHandlers{
		dataEngine: dataEngine,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}
}

// HandleHealth handles health check requests
func (h *DataEngineHandlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"service":   "data_engine",
	}

	// Add data engine status
	if h.dataEngine != nil {
		health["data_engine"] = map[string]interface{}{
			"running": h.dataEngine.IsRunning(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// HandleMetrics handles metrics requests
func (h *DataEngineHandlers) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if h.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	// Get metrics
	metrics := h.dataEngine.GetMetrics()
	if metrics == nil {
		http.Error(w, "No metrics available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// HandleAlerts handles alerts requests
func (h *DataEngineHandlers) HandleAlerts(w http.ResponseWriter, r *http.Request) {
	if h.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	// Get alerts
	alerts := h.dataEngine.GetActiveAlerts()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// HandleEvents handles events requests
func (h *DataEngineHandlers) HandleEvents(w http.ResponseWriter, r *http.Request) {
	if h.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	limit := 100 // default
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Get recent events (this would need to be implemented in DataEngine)
	events := make([]interface{}, 0) // Placeholder

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"limit":  limit,
	})
}

// HandleWebSocket handles WebSocket connections
func (h *DataEngineHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if h.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	// Upgrade connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// Send initial data
	if h.dataEngine != nil {
		// Send metrics
		metrics := h.dataEngine.GetMetrics()
		if metrics != nil {
			err := conn.WriteJSON(map[string]interface{}{
				"type":    "metrics",
				"metrics": metrics,
			})
			if err != nil {
				return
			}
		}

		// Send alerts
		alerts := h.dataEngine.GetActiveAlerts()
		if len(alerts) > 0 {
			err := conn.WriteJSON(map[string]interface{}{
				"type":   "alerts",
				"alerts": alerts,
			})
			if err != nil {
				return
			}
		}
	}

	// Handle incoming messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		// Handle different message types
		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "resolve_alert":
			alertID, ok := msg["alert_id"].(string)
			if !ok {
				continue
			}

			// Resolve alert
			resolved := h.dataEngine.ResolveAlert(alertID)

			// Send response
			err := conn.WriteJSON(map[string]interface{}{
				"type":     "alert_resolved",
				"alert_id": alertID,
				"resolved": resolved,
			})
			if err != nil {
				return
			}

		case "get_metrics":
			metrics := h.dataEngine.GetMetrics()
			err := conn.WriteJSON(map[string]interface{}{
				"type":    "metrics",
				"metrics": metrics,
			})
			if err != nil {
				return
			}

		case "get_alerts":
			alerts := h.dataEngine.GetActiveAlerts()
			err := conn.WriteJSON(map[string]interface{}{
				"type":   "alerts",
				"alerts": alerts,
			})
			if err != nil {
				return
			}
		}
	}
}

// HandleResolveAlert handles alert resolution requests
func (h *DataEngineHandlers) HandleResolveAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	alertID := vars["id"]
	if alertID == "" {
		http.Error(w, "Alert ID required", http.StatusBadRequest)
		return
	}

	resolved := h.dataEngine.ResolveAlert(alertID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"alert_id": alertID,
		"resolved": resolved,
	})
}

// BuntDBDataEngineHandlers contains HTTP handlers for BuntDB data_engine endpoints
type BuntDBDataEngineHandlers struct {
	dataEngine *BuntDBDataEngine
	upgrader   websocket.Upgrader
}

// NewBuntDBDataEngineHandlers creates a new instance of BuntDBDataEngineHandlers
func NewBuntDBDataEngineHandlers(dataEngine *BuntDBDataEngine) *BuntDBDataEngineHandlers {
	return &BuntDBDataEngineHandlers{
		dataEngine: dataEngine,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}
}

// HandleHealth handles health check requests for BuntDB data engine
func (h *BuntDBDataEngineHandlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"service":   "buntdb-data_engine",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// HandleMetrics handles metrics requests for BuntDB data engine
func (h *BuntDBDataEngineHandlers) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if h.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	// Get metrics snapshot
	metrics := h.dataEngine.GetMetrics()
	if metrics == nil {
		http.Error(w, "No metrics available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// HandleAlerts handles alerts requests for BuntDB data engine
func (h *BuntDBDataEngineHandlers) HandleAlerts(w http.ResponseWriter, r *http.Request) {
	if h.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	// Get active alerts (placeholder - would need to be implemented)
	alerts := make([]interface{}, 0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// HandleEvents handles events requests for BuntDB data engine
func (h *BuntDBDataEngineHandlers) HandleEvents(w http.ResponseWriter, r *http.Request) {
	if h.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	limit := 100 // default
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Get recent events (placeholder - would need to be implemented)
	events := make([]interface{}, 0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"limit":  limit,
	})
}

// HandleWebSocket handles WebSocket connections for BuntDB data engine
func (h *BuntDBDataEngineHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if h.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	// Upgrade connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// Send initial data
	metrics := h.dataEngine.GetMetrics()
	if metrics != nil {
		err := conn.WriteJSON(map[string]interface{}{
			"type":    "metrics",
			"metrics": metrics,
		})
		if err != nil {
			return
		}
	}

	// Handle incoming messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		// Handle different message types
		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "get_metrics":
			metrics := h.dataEngine.GetMetrics()
			err := conn.WriteJSON(map[string]interface{}{
				"type":    "metrics",
				"metrics": metrics,
			})
			if err != nil {
				return
			}
		}
	}
}

// HandleResolveAlert handles alert resolution requests for BuntDB data engine
func (h *BuntDBDataEngineHandlers) HandleResolveAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	alertID := vars["id"]
	if alertID == "" {
		http.Error(w, "Alert ID required", http.StatusBadRequest)
		return
	}

	// Resolve alert (placeholder - would need to be implemented)
	resolved := true

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"alert_id": alertID,
		"resolved": resolved,
	})
}
