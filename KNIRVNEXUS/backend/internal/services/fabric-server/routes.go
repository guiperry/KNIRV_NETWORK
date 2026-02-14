package fabricserver

import (
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterRoutes registers all fabric server routes with the provided router
func (as *FabricServer) RegisterRoutes(r *mux.Router) {
	// Create a subrouter for fabric server endpoints
	fabricRouter := r.PathPrefix("/objects").Subrouter()

	// Apply CORS middleware if enabled
	if as.enableCORS {
		fabricRouter.Use(as.corsMiddleware)
	}

	// Basic fabric management endpoints
	fabricRouter.HandleFunc("/info", as.HandleServerInfo).Methods("GET")
	fabricRouter.HandleFunc("/list", as.HandleListFabrics).Methods("GET")
	fabricRouter.HandleFunc("/upload", as.HandleUploadFabric).Methods("POST")

	// Fabric file endpoints
	fabricRouter.HandleFunc("/{name}", as.HandleDownloadFabric).Methods("GET")
	fabricRouter.HandleFunc("/delete/{name}", as.HandleDeleteFabric).Methods("DELETE")

	// Runtime management endpoints (if runtime manager is enabled)
	if as.enableRuntime && as.runtimeManager != nil {
		runtimeRouter := fabricRouter.PathPrefix("/runtime").Subrouter()

		runtimeRouter.HandleFunc("/start", as.HandleStartFabric).Methods("POST")
		runtimeRouter.HandleFunc("/stop/{id}", as.HandleStopFabric).Methods("POST")
		runtimeRouter.HandleFunc("/objects", as.HandleListRunningFabrics).Methods("GET")
		runtimeRouter.HandleFunc("/fabric/{id}", as.HandleGetFabric).Methods("GET")
		runtimeRouter.HandleFunc("/status", as.HandleRuntimeStatus).Methods("GET")
	}

}

// corsMiddleware provides CORS support
func (as *FabricServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if as.enableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
