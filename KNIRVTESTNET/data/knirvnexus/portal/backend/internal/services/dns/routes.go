package dns

import (
	"net/http"

	"github.com/gorilla/mux"

	"nexus-backend/internal/web/middleware"
)

// RegisterRoutes registers all DNS service routes with the provided router
func (ds *DynamicDNSService) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for DNS endpoints
	dnsRouter := r.PathPrefix("/dns").Subrouter()

	// Apply authentication middleware to protected routes
	protectedRouter := dnsRouter.PathPrefix("").Subrouter()
	protectedRouter.Use(authMiddleware.RequireAuth)

	// DNS record management routes (protected)
	recordRouter := protectedRouter.PathPrefix("/records").Subrouter()
	recordRouter.HandleFunc("", ds.HandleListDNSRecords).Methods("GET")
	recordRouter.HandleFunc("", ds.HandleCreateDNSRecord).Methods("POST")
	recordRouter.HandleFunc("/{id}", ds.HandleGetDNSRecord).Methods("GET")
	recordRouter.HandleFunc("/{id}", ds.HandleUpdateDNSRecord).Methods("PUT")
	recordRouter.HandleFunc("/{id}", ds.HandleDeleteDNSRecord).Methods("DELETE")

	// DNS zone management routes (protected)
	zoneRouter := protectedRouter.PathPrefix("/zones").Subrouter()
	zoneRouter.HandleFunc("", ds.HandleListDNSZones).Methods("GET")

	// System status routes (some may be public for monitoring)
	statusRouter := dnsRouter.PathPrefix("/status").Subrouter()
	statusRouter.HandleFunc("", ds.HandleGetDNSStatus).Methods("GET")

	// Handle OPTIONS requests for CORS
	dnsRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
