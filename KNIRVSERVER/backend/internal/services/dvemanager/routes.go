package dvemanager

import (
	"net/http"

	"github.com/gorilla/mux"

	"backend_server/internal/web/middleware"
)

// RegisterRoutes registers all DVE manager routes with the provided router
func (dm *DVEManager) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	dveRouter := r.PathPrefix("/api/dve-nodes").Subrouter()

	protectedRouter := dveRouter.PathPrefix("").Subrouter()
	protectedRouter.Use(authMiddleware.RequireAuth)

	nodeRouter := protectedRouter.PathPrefix("/nodes").Subrouter()
	nodeRouter.HandleFunc("", dm.HandleListNodes).Methods("GET")
	nodeRouter.HandleFunc("", dm.HandleRegisterNode).Methods("POST")
	nodeRouter.HandleFunc("/{id}", dm.HandleGetNode).Methods("GET")
	nodeRouter.HandleFunc("/{id}/status", dm.HandleUpdateNodeStatus).Methods("PUT")
	nodeRouter.HandleFunc("/{id}", dm.HandleDeleteNode).Methods("DELETE")

	nodeTasksRouter := protectedRouter.PathPrefix("/nodes/{nodeId}/tasks").Subrouter()
	nodeTasksRouter.HandleFunc("", dm.HandleGetNodeTasks).Methods("GET")
	nodeTasksRouter.HandleFunc("", dm.HandleCreateNodeTask).Methods("POST")
	nodeTasksRouter.HandleFunc("/{taskId}", dm.HandleGetNodeTask).Methods("GET")
	nodeTasksRouter.HandleFunc("/{taskId}", dm.HandleUpdateNodeTask).Methods("PUT")

	nodeMetricsRouter := protectedRouter.PathPrefix("/nodes/{nodeId}/metrics").Subrouter()
	nodeMetricsRouter.HandleFunc("", dm.HandleGetNodeMetrics).Methods("GET")
	nodeMetricsRouter.HandleFunc("/history", dm.HandleGetNodeMetricsHistory).Methods("GET")

	taskRouter := protectedRouter.PathPrefix("/tasks").Subrouter()
	taskRouter.HandleFunc("/allocate", dm.HandleAllocateTask).Methods("POST")
	taskRouter.HandleFunc("", dm.HandleListTasks).Methods("GET")
	taskRouter.HandleFunc("/{id}", dm.HandleGetTask).Methods("GET")

	healthRouter := dveRouter.PathPrefix("/health").Subrouter()
	healthRouter.HandleFunc("", dm.HandleGetSystemHealth).Methods("GET")
	healthRouter.HandleFunc("/status", dm.HandleGetSystemStatus).Methods("GET")

	metricsRouter := protectedRouter.PathPrefix("/metrics").Subrouter()
	metricsRouter.HandleFunc("", dm.HandleGetMetrics).Methods("GET")

	topologyRouter := dveRouter.PathPrefix("/topology").Subrouter()
	topologyRouter.HandleFunc("", dm.HandleGetNetworkTopology).Methods("GET")

	dveRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
