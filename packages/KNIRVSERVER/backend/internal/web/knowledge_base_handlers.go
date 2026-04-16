// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"backend_server/internal/services/knowledge_base"
	"backend_server/internal/web/middleware"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// KnowledgeBaseHandlers provides HTTP handlers for Knowledge Base management
// Implements GraphRAG-RS FFI harness for embedded graph neural reasoning
type KnowledgeBaseHandlers struct {
	kbService *knowledge_base.KnowledgeBaseService
}

// NewKnowledgeBaseHandlers creates knowledge base HTTP handlers
func NewKnowledgeBaseHandlers(kbService *knowledge_base.KnowledgeBaseService) *KnowledgeBaseHandlers {
	return &KnowledgeBaseHandlers{kbService: kbService}
}

type kbResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

func writeKBJSON(w http.ResponseWriter, status int, resp kbResponse) {
	resp.Timestamp = getCurrentTimestamp()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// GetKnowledgeBases handles GET /api/v1/knowledge-base/objects
func (h *KnowledgeBaseHandlers) GetKnowledgeBases(w http.ResponseWriter, r *http.Request) {
	// Optional filtering parameters
	status := r.URL.Query().Get("status")
	author := r.URL.Query().Get("author")
	search := r.URL.Query().Get("search")

	kbList, err := h.kbService.ListKnowledgeBases(r.Context(), status, author, search)
	if err != nil {
		writeKBJSON(w, http.StatusInternalServerError, kbResponse{Error: err.Error()})
		return
	}

	if kbList == nil {
		kbList = []*knowledge_base.KnowledgeBase{}
	}

	writeKBJSON(w, http.StatusOK, kbResponse{Success: true, Data: kbList})
}

// PostKnowledgeBase handles POST /api/v1/knowledge-base/objects
func (h *KnowledgeBaseHandlers) PostKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string                 `json:"name"`
		Description    string                 `json:"description"`
		Version        string                 `json:"version"`
		Author         string                 `json:"author"`
		Type           string                 `json:"type"` // GraphRAG, Vector, Semantic, etc.
		EmbeddingModel string                 `json:"embedding_model"`
		Configuration  map[string]interface{} `json:"configuration"`
		Tags           []string               `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeKBJSON(w, http.StatusBadRequest, kbResponse{Error: "invalid request body: " + err.Error()})
		return
	}

	if req.Name == "" {
		writeKBJSON(w, http.StatusBadRequest, kbResponse{Error: "name is required"})
		return
	}

	kb := &knowledge_base.KnowledgeBase{
		ID:             uuid.New().String(),
		Name:           req.Name,
		Description:    req.Description,
		Version:        req.Version,
		Author:         req.Author,
		Type:           req.Type,
		Status:         "uploaded",
		EmbeddingModel: req.EmbeddingModel,
		Configuration:  req.Configuration,
		Tags:           req.Tags,
		UploadedAt:     time.Now(),
		LastModified:   time.Now(),
		UploadedBy:     "system", // TODO: Get from auth context
	}

	created, err := h.kbService.CreateKnowledgeBase(r.Context(), kb)
	if err != nil {
		writeKBJSON(w, http.StatusInternalServerError, kbResponse{Error: err.Error()})
		return
	}

	writeKBJSON(w, http.StatusCreated, kbResponse{Success: true, Data: created})
}

// GetKnowledgeBase handles GET /api/v1/knowledge-base/objects/{id}
func (h *KnowledgeBaseHandlers) GetKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	kbID := mux.Vars(r)["id"]
	kb, err := h.kbService.GetKnowledgeBase(r.Context(), kbID)
	if err != nil {
		writeKBJSON(w, http.StatusNotFound, kbResponse{Error: "knowledge base not found"})
		return
	}
	writeKBJSON(w, http.StatusOK, kbResponse{Success: true, Data: kb})
}

// PutKnowledgeBase handles PUT /api/v1/knowledge-base/objects/{id}
func (h *KnowledgeBaseHandlers) PutKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	kbID := mux.Vars(r)["id"]
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeKBJSON(w, http.StatusBadRequest, kbResponse{Error: "invalid request body: " + err.Error()})
		return
	}

	updated, err := h.kbService.UpdateKnowledgeBase(r.Context(), kbID, updates)
	if err != nil {
		writeKBJSON(w, http.StatusInternalServerError, kbResponse{Error: err.Error()})
		return
	}

	writeKBJSON(w, http.StatusOK, kbResponse{Success: true, Data: updated})
}

// DeleteKnowledgeBase handles DELETE /api/v1/knowledge-base/objects/{id}
func (h *KnowledgeBaseHandlers) DeleteKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	kbID := mux.Vars(r)["id"]
	if err := h.kbService.DeleteKnowledgeBase(r.Context(), kbID); err != nil {
		writeKBJSON(w, http.StatusInternalServerError, kbResponse{Error: err.Error()})
		return
	}
	writeKBJSON(w, http.StatusOK, kbResponse{Success: true, Data: map[string]string{"message": "deleted"}})
}

// GetKnowledgeBaseSummary handles GET /api/v1/knowledge-base/summary
func (h *KnowledgeBaseHandlers) GetKnowledgeBaseSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.kbService.GetSummary(r.Context())
	if err != nil {
		writeKBJSON(w, http.StatusInternalServerError, kbResponse{Error: err.Error()})
		return
	}
	writeKBJSON(w, http.StatusOK, kbResponse{Success: true, Data: summary})
}

// DeployKnowledgeBase handles POST /api/v1/knowledge-base/objects/{id}/deploy
func (h *KnowledgeBaseHandlers) DeployKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	kbID := mux.Vars(r)["id"]

	deployed, err := h.kbService.DeployKnowledgeBase(r.Context(), kbID)
	if err != nil {
		writeKBJSON(w, http.StatusInternalServerError, kbResponse{Error: err.Error()})
		return
	}

	writeKBJSON(w, http.StatusOK, kbResponse{Success: true, Data: deployed})
}

// QueryKnowledgeBase handles POST /api/v1/knowledge-base/objects/{id}/query
// Executes GraphRAG queries against the knowledge base
func (h *KnowledgeBaseHandlers) QueryKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	kbID := mux.Vars(r)["id"]

	var queryReq struct {
		Query  string                 `json:"query"`
		Mode   string                 `json:"mode"` // local, global, hybrid
		Limit  int                    `json:"limit"`
		Params map[string]interface{} `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&queryReq); err != nil {
		writeKBJSON(w, http.StatusBadRequest, kbResponse{Error: "invalid query body: " + err.Error()})
		return
	}

	if queryReq.Query == "" {
		writeKBJSON(w, http.StatusBadRequest, kbResponse{Error: "query is required"})
		return
	}

	result, err := h.kbService.QueryGraphRAG(r.Context(), kbID, &knowledge_base.GraphRAGQuery{
		Query:  queryReq.Query,
		Mode:   queryReq.Mode,
		Limit:  queryReq.Limit,
		Params: queryReq.Params,
	})
	if err != nil {
		writeKBJSON(w, http.StatusInternalServerError, kbResponse{Error: fmt.Sprintf("query failed: %v", err)})
		return
	}

	writeKBJSON(w, http.StatusOK, kbResponse{Success: true, Data: result})
}

// IndexKnowledgeBase handles POST /api/v1/knowledge-base/objects/{id}/index
// Rebuilds the GraphRAG index for the knowledge base
func (h *KnowledgeBaseHandlers) IndexKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	kbID := mux.Vars(r)["id"]

	var indexReq struct {
		Strategy string `json:"strategy"` // incremental, full
		Async    bool   `json:"async"`
	}

	json.NewDecoder(r.Body).Decode(&indexReq)

	indexStatus, err := h.kbService.IndexKnowledgeBase(r.Context(), kbID, indexReq.Strategy, indexReq.Async)
	if err != nil {
		writeKBJSON(w, http.StatusInternalServerError, kbResponse{Error: err.Error()})
		return
	}

	writeKBJSON(w, http.StatusOK, kbResponse{Success: true, Data: indexStatus})
}

// GetIndexStatus handles GET /api/v1/knowledge-base/objects/{id}/index-status
func (h *KnowledgeBaseHandlers) GetIndexStatus(w http.ResponseWriter, r *http.Request) {
	kbID := mux.Vars(r)["id"]

	status, err := h.kbService.GetIndexStatus(r.Context(), kbID)
	if err != nil {
		writeKBJSON(w, http.StatusInternalServerError, kbResponse{Error: err.Error()})
		return
	}

	writeKBJSON(w, http.StatusOK, kbResponse{Success: true, Data: status})
}

// RegisterRoutes registers all knowledge base routes with the router
func (h *KnowledgeBaseHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	kbRouter := r.PathPrefix("/api/v1/knowledge-base").Subrouter()
	if authMiddleware != nil {
		kbRouter.Use(authMiddleware.OptionalAuth)
	}

	// CRUD operations
	kbRouter.HandleFunc("/objects", h.GetKnowledgeBases).Methods("GET", "OPTIONS")
	kbRouter.HandleFunc("/objects", h.PostKnowledgeBase).Methods("POST", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}", h.GetKnowledgeBase).Methods("GET", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}", h.PutKnowledgeBase).Methods("PUT", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}", h.DeleteKnowledgeBase).Methods("DELETE", "OPTIONS")

	// GraphRAG operations
	kbRouter.HandleFunc("/objects/{id}/query", h.QueryKnowledgeBase).Methods("POST", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}/index", h.IndexKnowledgeBase).Methods("POST", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}/index-status", h.GetIndexStatus).Methods("GET", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}/deploy", h.DeployKnowledgeBase).Methods("POST", "OPTIONS")

	// Summary
	kbRouter.HandleFunc("/summary", h.GetKnowledgeBaseSummary).Methods("GET", "OPTIONS")
}
