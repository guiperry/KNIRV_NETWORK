package modelserver

import (
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterRoutes registers all model server routes with the provided router
func (as *ModelServer) RegisterRoutes(r *mux.Router) {
	// Create a subrouter for model server endpoints
	modelRouter := r.PathPrefix("/models").Subrouter()

	// Apply CORS middleware if enabled
	if as.enableCORS {
		modelRouter.Use(as.corsMiddleware)
	}

	// Basic model management endpoints
	modelRouter.HandleFunc("/info", as.HandleServerInfo).Methods("GET")
	modelRouter.HandleFunc("/list", as.HandleListModels).Methods("GET")
	modelRouter.HandleFunc("/upload", as.HandleUploadModel).Methods("POST")

	// Model file endpoints
	modelRouter.HandleFunc("/{name}", as.HandleDownloadModel).Methods("GET")
	modelRouter.HandleFunc("/delete/{name}", as.HandleDeleteModel).Methods("DELETE")

	// Runtime management endpoints (if runtime manager is enabled)
	if as.enableRuntime && as.runtimeManager != nil {
		runtimeRouter := modelRouter.PathPrefix("/runtime").Subrouter()

		runtimeRouter.HandleFunc("/start", as.HandleStartModel).Methods("POST")
		runtimeRouter.HandleFunc("/stop/{id}", as.HandleStopModel).Methods("POST")
		runtimeRouter.HandleFunc("/models", as.HandleListRunningModels).Methods("GET")
		runtimeRouter.HandleFunc("/model/{id}", as.HandleGetModel).Methods("GET")
		runtimeRouter.HandleFunc("/status", as.HandleRuntimeStatus).Methods("GET")
	}

	// Handle OPTIONS requests for CORS
	if as.enableCORS {
		modelRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
}

// corsMiddleware provides CORS support
func (as *ModelServer) corsMiddleware(next http.Handler) http.Handler {
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
