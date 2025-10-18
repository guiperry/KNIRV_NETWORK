package dns

import (
	"net/http"

	"github.com/gorilla/mux"

	"backend-server/internal/web/middleware"
)

// RegisterRoutes registers all DNS service routes with the provided router
func (ds *DynamicDNSService) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for DNS endpoints
	dnsRouter := r.PathPrefix("/api/dns").Subrouter()

	// For development, make all routes public (remove auth requirement)
	// TODO: Re-enable authentication for production

	// DNS record management routes (public for development)
	recordRouter := dnsRouter.PathPrefix("/records").Subrouter()
	recordRouter.HandleFunc("", ds.HandleListDNSRecords).Methods("GET")
	recordRouter.HandleFunc("", ds.HandleCreateDNSRecord).Methods("POST")
	recordRouter.HandleFunc("/{id}", ds.HandleGetDNSRecord).Methods("GET")
	recordRouter.HandleFunc("/{id}", ds.HandleUpdateDNSRecord).Methods("PUT")
	recordRouter.HandleFunc("/{id}", ds.HandleDeleteDNSRecord).Methods("DELETE")

	// DNS zone management routes (public for development)
	zoneRouter := dnsRouter.PathPrefix("/zones").Subrouter()
	zoneRouter.HandleFunc("", ds.HandleListDNSZones).Methods("GET")

	// System status routes (public)
	statusRouter := dnsRouter.PathPrefix("/status").Subrouter()
	statusRouter.HandleFunc("", ds.HandleGetDNSStatus).Methods("GET")

	// Handle OPTIONS requests for CORS
	dnsRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
