package data_engine

import (
	"github.com/gorilla/mux"
)

// RegisterRoutes registers data_engine routes with the provided router
func (de *DataEngine) RegisterRoutes(router *mux.Router) {
	// Create handlers
	handlers := NewDataEngineHandlers(de)

	// Create data_engine subrouter
	dataEngineRouter := router.PathPrefix("/api/data_engine").Subrouter()

	// Health endpoint
	dataEngineRouter.HandleFunc("/health", handlers.HandleHealth).Methods("GET")

	// Metrics endpoints
	dataEngineRouter.HandleFunc("/metrics", handlers.HandleMetrics).Methods("GET")

	// Alerts endpoints
	dataEngineRouter.HandleFunc("/alerts", handlers.HandleAlerts).Methods("GET")
	dataEngineRouter.HandleFunc("/alerts/{id}/resolve", handlers.HandleResolveAlert).Methods("POST")

	// Events endpoints
	dataEngineRouter.HandleFunc("/events", handlers.HandleEvents).Methods("GET")

	// WebSocket endpoint
	dataEngineRouter.HandleFunc("/ws", handlers.HandleWebSocket)
}

// RegisterBuntDBRoutes registers BuntDB data_engine routes with the provided router
func (bde *BuntDBDataEngine) RegisterRoutes(router *mux.Router) {
	// Create handlers for BuntDB data engine
	handlers := NewBuntDBDataEngineHandlers(bde)

	// Create data_engine subrouter
	dataEngineRouter := router.PathPrefix("/api/data_engine").Subrouter()

	// Health endpoint
	dataEngineRouter.HandleFunc("/health", handlers.HandleHealth).Methods("GET")

	// Metrics endpoints
	dataEngineRouter.HandleFunc("/metrics", handlers.HandleMetrics).Methods("GET")

	// Alerts endpoints
	dataEngineRouter.HandleFunc("/alerts", handlers.HandleAlerts).Methods("GET")
	dataEngineRouter.HandleFunc("/alerts/{id}/resolve", handlers.HandleResolveAlert).Methods("POST")

	// Events endpoints
	dataEngineRouter.HandleFunc("/events", handlers.HandleEvents).Methods("GET")

	// WebSocket endpoint
	dataEngineRouter.HandleFunc("/ws", handlers.HandleWebSocket)
}
