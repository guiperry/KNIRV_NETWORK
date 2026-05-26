package dve_workspace

import (
	"net/http"

	"github.com/gorilla/mux"

	"backend_server/internal/web/middleware"
)

// RegisterRoutes registers all DVE service routes with the provided router
func (s *DVEService) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for DVE endpoints
	dveRouter := r.PathPrefix("/dve").Subrouter()

	// Apply authentication middleware to all DVE routes
	dveRouter.Use(authMiddleware.RequireAuth)

	// Environment routes
	envRouter := dveRouter.PathPrefix("/environments").Subrouter()
	envRouter.HandleFunc("", s.HandleListEnvironments).Methods("GET")
	envRouter.HandleFunc("", s.HandleCreateEnvironment).Methods("POST")
	envRouter.HandleFunc("/{id}", s.HandleGetEnvironment).Methods("GET")
	envRouter.HandleFunc("/{id}", s.HandleDeleteEnvironment).Methods("DELETE")
	envRouter.HandleFunc("/{id}/start", s.HandleStartEnvironment).Methods("POST")
	envRouter.HandleFunc("/{id}/stop", s.HandleStopEnvironment).Methods("POST")

	// Session routes
	sessionRouter := dveRouter.PathPrefix("/sessions").Subrouter()
	sessionRouter.HandleFunc("", s.HandleListSessions).Methods("GET")
	sessionRouter.HandleFunc("", s.HandleCreateSession).Methods("POST")

	// Project routes
	projectRouter := dveRouter.PathPrefix("/projects").Subrouter()
	projectRouter.HandleFunc("", s.HandleListProjects).Methods("GET")
	projectRouter.HandleFunc("", s.HandleCreateProject).Methods("POST")

	// DVE Workspace config routes (runtime-configurable settings)
	workspaceRouter := r.PathPrefix("/api/dve-workspace").Subrouter()
	workspaceRouter.Use(authMiddleware.RequireAuth)
	workspaceRouter.HandleFunc("/config", s.HandleGetWorkspaceConfig).Methods("GET")
	workspaceRouter.HandleFunc("/config", s.HandleUpdateWorkspaceConfig).Methods("PUT")
	workspaceRouter.HandleFunc("/rootfs-status", s.HandleGetRootfsStatus).Methods("GET")
	workspaceRouter.HandleFunc("/rootfs-bootstrap", s.HandleBootstrapRootfs).Methods("POST")
	workspaceRouter.HandleFunc("/stats", s.HandleGetWorkspaceStats).Methods("GET")

	// Handle OPTIONS requests for CORS
	dveRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
