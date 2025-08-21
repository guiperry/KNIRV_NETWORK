package agentserver

import (
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterRoutes registers all agent server routes with the provided router
func (as *AgentServer) RegisterRoutes(r *mux.Router) {
	// Create a subrouter for agent server endpoints
	agentRouter := r.PathPrefix("/agents").Subrouter()

	// Apply CORS middleware if enabled
	if as.enableCORS {
		agentRouter.Use(as.corsMiddleware)
	}

	// Basic agent management endpoints
	agentRouter.HandleFunc("/info", as.HandleServerInfo).Methods("GET")
	agentRouter.HandleFunc("/list", as.HandleListAgents).Methods("GET")
	agentRouter.HandleFunc("/upload", as.HandleUploadAgent).Methods("POST")
	
	// Agent file endpoints
	agentRouter.HandleFunc("/{name}", as.HandleDownloadAgent).Methods("GET")
	agentRouter.HandleFunc("/delete/{name}", as.HandleDeleteAgent).Methods("DELETE")

	// Runtime management endpoints (if runtime manager is enabled)
	if as.enableRuntime && as.runtimeManager != nil {
		runtimeRouter := agentRouter.PathPrefix("/runtime").Subrouter()
		
		runtimeRouter.HandleFunc("/start", as.HandleStartAgent).Methods("POST")
		runtimeRouter.HandleFunc("/stop/{id}", as.HandleStopAgent).Methods("POST")
		runtimeRouter.HandleFunc("/agents", as.HandleListRunningAgents).Methods("GET")
		runtimeRouter.HandleFunc("/agent/{id}", as.HandleGetAgent).Methods("GET")
		runtimeRouter.HandleFunc("/status", as.HandleRuntimeStatus).Methods("GET")
	}

	// Handle OPTIONS requests for CORS
	if as.enableCORS {
		agentRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
}

// corsMiddleware provides CORS support
func (as *AgentServer) corsMiddleware(next http.Handler) http.Handler {
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
