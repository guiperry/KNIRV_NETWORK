package validation

import (
	"net/http"

	"github.com/gorilla/mux"

	"nexus-backend/internal/web/middleware"
)

// RegisterRoutes registers all validation service routes with the provided router
func (vc *ValidationCore) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for validation endpoints
	validationRouter := r.PathPrefix("/validation").Subrouter()

	// Apply authentication middleware to protected routes
	protectedRouter := validationRouter.PathPrefix("").Subrouter()
	protectedRouter.Use(authMiddleware.RequireAuth)

	// Task management routes (protected)
	taskRouter := protectedRouter.PathPrefix("/tasks").Subrouter()
	taskRouter.HandleFunc("", vc.HandleListTasks).Methods("GET")
	taskRouter.HandleFunc("", vc.HandleCreateTask).Methods("POST")
	taskRouter.HandleFunc("/{id}", vc.HandleGetTask).Methods("GET")
	taskRouter.HandleFunc("/{id}/execute", vc.HandleExecuteTask).Methods("POST")
	taskRouter.HandleFunc("/{id}/results", vc.HandleGetTaskResults).Methods("GET")

	// Queue management routes (protected)
	queueRouter := protectedRouter.PathPrefix("/queue").Subrouter()
	queueRouter.HandleFunc("/status", vc.HandleGetTaskQueue).Methods("GET")

	// Metrics routes (protected)
	metricsRouter := protectedRouter.PathPrefix("/metrics").Subrouter()
	metricsRouter.HandleFunc("", vc.HandleGetValidationMetrics).Methods("GET")

	// System status routes (some may be public for monitoring)
	statusRouter := validationRouter.PathPrefix("/status").Subrouter()
	statusRouter.HandleFunc("", vc.HandleGetValidationStatus).Methods("GET")

	// Handle OPTIONS requests for CORS
	validationRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
