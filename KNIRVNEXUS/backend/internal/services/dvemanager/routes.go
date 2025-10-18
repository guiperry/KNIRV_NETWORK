package dvemanager

import (
	"net/http"

	"github.com/gorilla/mux"

	"backend-server/internal/web/middleware"
)

// RegisterRoutes registers all DVE manager routes with the provided router
func (dm *DVEManager) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for DVE manager endpoints
	dveRouter := r.PathPrefix("/dve").Subrouter()

	// Apply authentication middleware to protected routes
	protectedRouter := dveRouter.PathPrefix("").Subrouter()
	protectedRouter.Use(authMiddleware.RequireAuth)

	// Node management routes
	nodeRouter := protectedRouter.PathPrefix("/nodes").Subrouter()
	nodeRouter.HandleFunc("", dm.HandleListNodes).Methods("GET")
	nodeRouter.HandleFunc("", dm.HandleRegisterNode).Methods("POST")
	nodeRouter.HandleFunc("/{id}", dm.HandleGetNode).Methods("GET")
	nodeRouter.HandleFunc("/{id}/status", dm.HandleUpdateNodeStatus).Methods("PUT")

	// Task allocation routes
	taskRouter := protectedRouter.PathPrefix("/tasks").Subrouter()
	taskRouter.HandleFunc("/allocate", dm.HandleAllocateTask).Methods("POST")

	// System health and status routes (some may be public for monitoring)
	healthRouter := dveRouter.PathPrefix("/health").Subrouter()
	healthRouter.HandleFunc("", dm.HandleGetSystemHealth).Methods("GET")
	healthRouter.HandleFunc("/status", dm.HandleGetSystemStatus).Methods("GET")

	// Metrics routes (protected)
	metricsRouter := protectedRouter.PathPrefix("/metrics").Subrouter()
	metricsRouter.HandleFunc("", dm.HandleGetMetrics).Methods("GET")

	// Handle OPTIONS requests for CORS
	dveRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
