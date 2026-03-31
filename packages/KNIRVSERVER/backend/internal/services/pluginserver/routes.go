package pluginserver

import (
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterRoutes registers all plugin server routes with the provided router
func (as *PluginServer) RegisterRoutes(r *mux.Router) {
	// Create a subrouter for plugin server endpoints
	pluginRouter := r.PathPrefix("/objects").Subrouter()

	// Apply CORS middleware if enabled
	if as.enableCORS {
		pluginRouter.Use(as.corsMiddleware)
	}

	// Basic plugin management endpoints
	pluginRouter.HandleFunc("/info", as.HandleServerInfo).Methods("GET")
	pluginRouter.HandleFunc("/list", as.HandleListPlugins).Methods("GET")
	pluginRouter.HandleFunc("/upload", as.HandleUploadPlugin).Methods("POST")

	// Plugin file endpoints
	pluginRouter.HandleFunc("/{name}", as.HandleDownloadPlugin).Methods("GET")
	pluginRouter.HandleFunc("/delete/{name}", as.HandleDeletePlugin).Methods("DELETE")

	// Runtime management endpoints (if runtime manager is enabled)
	if as.enableRuntime && as.runtimeManager != nil {
		runtimeRouter := pluginRouter.PathPrefix("/runtime").Subrouter()

		runtimeRouter.HandleFunc("/start", as.HandleStartPlugin).Methods("POST")
		runtimeRouter.HandleFunc("/stop/{id}", as.HandleStopPlugin).Methods("POST")
		runtimeRouter.HandleFunc("/objects", as.HandleListRunningPlugins).Methods("GET")
		runtimeRouter.HandleFunc("/plugin/{id}", as.HandleGetPlugin).Methods("GET")
		runtimeRouter.HandleFunc("/status", as.HandleRuntimeStatus).Methods("GET")
	}

}

// corsMiddleware provides CORS support
func (as *PluginServer) corsMiddleware(next http.Handler) http.Handler {
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
