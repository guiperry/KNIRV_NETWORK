package memory

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Handlers holds all HTTP handlers for the unified memory system
type Handlers struct {
	system *UnifiedMemorySystem
	logger *zap.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(system *UnifiedMemorySystem, logger *zap.Logger) *Handlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handlers{
		system: system,
		logger: logger,
	}
}

// RegisterRoutes registers all memory API routes under /api/v1/memory/
func (h *Handlers) RegisterRoutes(router *mux.Router) {
	// Storage endpoints
	router.HandleFunc("/api/v1/memory/store", h.HandleStoreInteraction).Methods("POST")
	router.HandleFunc("/api/v1/memory/execute/{id}", h.HandleExecuteSolution).Methods("POST")

	// Knowledge base endpoints
	router.HandleFunc("/api/v1/memory/knowledge-bases", h.HandleListKnowledgeBases).Methods("GET")
	router.HandleFunc("/api/v1/memory/knowledge-bases/{id}", h.HandleGetKnowledgeBase).Methods("GET")
	router.HandleFunc("/api/v1/memory/knowledge-bases/{id}/query", h.HandleQueryKnowledgeBase).Methods("POST")

	// Error node endpoints
	router.HandleFunc("/api/v1/memory/error-nodes", h.HandleCreateErrorNode).Methods("POST")
	router.HandleFunc("/api/v1/memory/error-nodes", h.HandleListErrorNodes).Methods("GET")

	// Event streaming endpoint
	router.HandleFunc("/api/v1/memory/events", h.HandleGetEvents).Methods("GET")

	// Ontology endpoints
	router.HandleFunc("/api/v1/memory/ontology/entities", h.HandleListEntities).Methods("GET")
	router.HandleFunc("/api/v1/memory/ontology/relations", h.HandleListRelations).Methods("GET")

	// Cross-backend query
	router.HandleFunc("/api/v1/memory/query", h.HandleQuery).Methods("POST")
}

// HandleStoreInteraction stores a new agent interaction
func (h *Handlers) HandleStoreInteraction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID     string `json:"agent_id"`
		ErrorDesc   string `json:"error_desc"`
		SolutionCode string `json:"solution_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.AgentID == "" || req.ErrorDesc == "" {
		http.Error(w, "agent_id and error_desc are required", http.StatusBadRequest)
		return
	}

	err := h.system.StoreInteraction(r.Context(), req.AgentID, req.ErrorDesc, req.SolutionCode)
	if err != nil {
		h.logger.Error("failed to store interaction", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleExecuteSolution executes a solution by ID
func (h *Handlers) HandleExecuteSolution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var params map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		params = make(map[string]interface{})
	}

	if h.system.GetVaultService() == nil {
		http.Error(w, "vault service not available", http.StatusServiceUnavailable)
		return
	}

	result, err := h.system.GetVaultService().ExecuteSolution(id, params)
	if err != nil {
		h.logger.Error("failed to execute solution", zap.String("id", id), zap.Error(err))
		http.Error(w, "solution not found or execution failed", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": result})
}

// HandleListKnowledgeBases lists all knowledge bases
func (h *Handlers) HandleListKnowledgeBases(w http.ResponseWriter, r *http.Request) {
	// Placeholder - would query the knowledge base service
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"knowledge_bases": []string{},
		"total":           0,
	})
}

// HandleGetKnowledgeBase gets a specific knowledge base
func (h *Handlers) HandleGetKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    id,
		"status": "not_implemented",
	})
}

// HandleQueryKnowledgeBase queries a knowledge base
func (h *Handlers) HandleQueryKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if h.system.GetGraphRAGClient() == nil {
		http.Error(w, "graphrag not available", http.StatusServiceUnavailable)
		return
	}

	result, err := h.system.GetGraphRAGClient().Query(r.Context(), id, &GraphRAGQuery{
		Query: req.Query,
		Mode:  "hybrid",
		Limit: req.Limit,
	})
	if err != nil {
		h.logger.Error("query failed", zap.String("kb_id", id), zap.Error(err))
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleCreateErrorNode creates a new error node
func (h *Handlers) HandleCreateErrorNode(w http.ResponseWriter, r *http.Request) {
	var node ErrorNode
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if node.ID == "" {
		node.ID = uuid.New().String()
	}

	if h.system.GetVaultService() == nil {
		http.Error(w, "vault service not available", http.StatusServiceUnavailable)
		return
	}

	if err := h.system.GetVaultService().RegisterError(&node); err != nil {
		h.logger.Error("failed to register error", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(node)
}

// HandleListErrorNodes lists all error nodes
func (h *Handlers) HandleListErrorNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error_nodes": []string{},
		"total":       0,
	})
}

// HandleGetEvents returns recent events (placeholder)
func (h *Handlers) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": []string{},
		"total":  0,
	})
}

// HandleListEntities lists ontology entities
func (h *Handlers) HandleListEntities(w http.ResponseWriter, r *http.Request) {
	if h.system.GetOntologyManager() == nil {
		http.Error(w, "ontology not available", http.StatusServiceUnavailable)
		return
	}

	entities := h.system.GetOntologyManager().QueryByType(EntityTypePattern)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entities": entities,
		"total":    len(entities),
	})
}

// HandleListRelations lists ontology relations
func (h *Handlers) HandleListRelations(w http.ResponseWriter, r *http.Request) {
	if h.system.GetOntologyManager() == nil {
		http.Error(w, "ontology not available", http.StatusServiceUnavailable)
		return
	}

	// Return empty for now
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"relations": []string{},
		"total":     0,
	})
}

// HandleQuery performs a cross-backend query
func (h *Handlers) HandleQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.system.Query(r.Context(), &req)
	if err != nil {
		h.logger.Error("query failed", zap.Error(err))
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
