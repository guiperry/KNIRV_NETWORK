package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend_server/internal/services/cognitiveengine"

	"github.com/gorilla/mux"
)

type OntologyService interface {
	GetEntity(id string) (*cognitiveengine.OntologyEntity, bool)
	QueryByType(t cognitiveengine.OntologyEntityType) []*cognitiveengine.OntologyEntity
	FindRelations(sourceID string) []cognitiveengine.OntologyRelation
	Stats() (entityCount int, relationCount int)
}

type OntologyHandlers struct {
	ontology OntologyService
}

func NewOntologyHandlers(ontology OntologyService) *OntologyHandlers {
	return &OntologyHandlers{
		ontology: ontology,
	}
}

func (h *OntologyHandlers) RegisterRoutes(r *mux.Router) {
	ontologyRouter := r.PathPrefix("/api/ontology").Subrouter()

	ontologyRouter.HandleFunc("/stats", h.GetStats).Methods("GET", "OPTIONS")
	ontologyRouter.HandleFunc("/entities", h.ListEntities).Methods("GET", "OPTIONS")
	ontologyRouter.HandleFunc("/entities/{id}", h.GetEntity).Methods("GET", "OPTIONS")
	ontologyRouter.HandleFunc("/relations", h.ListRelations).Methods("GET", "OPTIONS")
	ontologyRouter.HandleFunc("/search", h.SearchEntities).Methods("GET", "OPTIONS")
}

func (h *OntologyHandlers) GetStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	entityCount, relationCount := h.ontology.Stats()

	entityTypes := map[string]int{
		"dve_node":          0,
		"validation_task":   0,
		"validation_result": 0,
		"adaptation_event":  0,
		"failure_pattern":   0,
		"guardrail_policy":  0,
		"policy_violation":  0,
	}

	entities := h.ontology.QueryByType("")
	for _, entity := range entities {
		entityTypes[string(entity.Type)]++
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entity_count":   entityCount,
		"relation_count": relationCount,
		"entity_types":   entityTypes,
	})
}

func (h *OntologyHandlers) ListEntities(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	entityType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	var entities []*cognitiveengine.OntologyEntity
	if entityType != "" {
		entities = h.ontology.QueryByType(cognitiveengine.OntologyEntityType(entityType))
	} else {
		entities = h.ontology.QueryByType("")
	}

	if len(entities) > limit {
		entities = entities[:limit]
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entities": entities,
		"count":    len(entities),
		"limit":    limit,
	})
}

func (h *OntologyHandlers) GetEntity(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	entityID := vars["id"]

	entity, ok := h.ontology.GetEntity(entityID)
	if !ok {
		http.Error(w, `{"error":"entity not found"}`, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(entity)
}

func (h *OntologyHandlers) ListRelations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sourceID := r.URL.Query().Get("source_id")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	var relations []cognitiveengine.OntologyRelation
	if sourceID != "" {
		relations = h.ontology.FindRelations(sourceID)
	}

	if len(relations) > limit {
		relations = relations[:limit]
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"relations": relations,
		"count":     len(relations),
		"limit":     limit,
	})
}

func (h *OntologyHandlers) SearchEntities(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query().Get("q")
	entityType := r.URL.Query().Get("type")

	if query == "" {
		http.Error(w, `{"error":"query parameter 'q' required"}`, http.StatusBadRequest)
		return
	}

	allEntities := h.ontology.QueryByType(cognitiveengine.OntologyEntityType(entityType))

	var results []*cognitiveengine.OntologyEntity
	queryLower := toLower(query)
	for _, entity := range allEntities {
		if containsLower(entity.Label, queryLower) {
			results = append(results, entity)
		}
		if len(results) >= 50 {
			break
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"count":   len(results),
		"query":   query,
	})
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func containsLower(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
