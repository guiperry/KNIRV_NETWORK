package cde

import (
	"net/http"

	"github.com/gorilla/mux"

	"nexus-backend/internal/web/middleware"
)

// RegisterRoutes registers all CDE service routes with the provided router
func (s *CDEService) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for CDE endpoints
	cdeRouter := r.PathPrefix("/cde").Subrouter()

	// Apply authentication middleware to all CDE routes
	cdeRouter.Use(authMiddleware.RequireAuth)

	// Environment routes
	envRouter := cdeRouter.PathPrefix("/environments").Subrouter()
	envRouter.HandleFunc("", s.HandleListEnvironments).Methods("GET")
	envRouter.HandleFunc("", s.HandleCreateEnvironment).Methods("POST")
	envRouter.HandleFunc("/{id}", s.HandleGetEnvironment).Methods("GET")
	envRouter.HandleFunc("/{id}", s.HandleDeleteEnvironment).Methods("DELETE")
	envRouter.HandleFunc("/{id}/start", s.HandleStartEnvironment).Methods("POST")
	envRouter.HandleFunc("/{id}/stop", s.HandleStopEnvironment).Methods("POST")

	// Session routes
	sessionRouter := cdeRouter.PathPrefix("/sessions").Subrouter()
	sessionRouter.HandleFunc("", s.HandleListSessions).Methods("GET")
	sessionRouter.HandleFunc("", s.HandleCreateSession).Methods("POST")

	// Project routes
	projectRouter := cdeRouter.PathPrefix("/projects").Subrouter()
	projectRouter.HandleFunc("", s.HandleListProjects).Methods("GET")
	projectRouter.HandleFunc("", s.HandleCreateProject).Methods("POST")

	// Handle OPTIONS requests for CORS
	cdeRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
