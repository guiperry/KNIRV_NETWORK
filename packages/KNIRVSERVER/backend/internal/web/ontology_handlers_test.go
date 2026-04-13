package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_server/internal/services/cognitiveengine"

	"github.com/gorilla/mux"
)

type mockOntologyService struct {
	entities    map[string]*cognitiveengine.OntologyEntity
	relations   []cognitiveengine.OntologyRelation
	entityCount int
	relCount    int
}

func (m *mockOntologyService) GetEntity(id string) (*cognitiveengine.OntologyEntity, bool) {
	e, ok := m.entities[id]
	return e, ok
}

func (m *mockOntologyService) QueryByType(t cognitiveengine.OntologyEntityType) []*cognitiveengine.OntologyEntity {
	var result []*cognitiveengine.OntologyEntity
	for _, e := range m.entities {
		if t == "" || e.Type == t {
			result = append(result, e)
		}
	}
	return result
}

func (m *mockOntologyService) FindRelations(sourceID string) []cognitiveengine.OntologyRelation {
	var result []cognitiveengine.OntologyRelation
	for _, r := range m.relations {
		if r.SourceID == sourceID {
			result = append(result, r)
		}
	}
	return result
}

func (m *mockOntologyService) Stats() (int, int) {
	return m.entityCount, m.relCount
}

func TestOntologyHandlers_GetStats(t *testing.T) {
	service := &mockOntologyService{
		entityCount: 25,
		relCount:    100,
		entities: map[string]*cognitiveengine.OntologyEntity{
			"node_1": {
				ID:    "node_1",
				Type:  cognitiveengine.EntityTypeNode,
				Label: "DVE Node 1",
			},
		},
	}

	handler := NewOntologyHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/ontology/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["entity_count"].(float64) != 25 {
		t.Errorf("expected entity_count 25, got %v", response["entity_count"])
	}

	if response["relation_count"].(float64) != 100 {
		t.Errorf("expected relation_count 100, got %v", response["relation_count"])
	}
}

func TestOntologyHandlers_ListEntities(t *testing.T) {
	service := &mockOntologyService{
		entities: map[string]*cognitiveengine.OntologyEntity{
			"node_1": {
				ID:        "node_1",
				Type:      cognitiveengine.EntityTypeNode,
				Label:     "Test Node",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			"node_2": {
				ID:        "node_2",
				Type:      cognitiveengine.EntityTypeNode,
				Label:     "Test Node 2",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	handler := NewOntologyHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/ontology/entities?type=dve_node&limit=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	entities := response["entities"].([]interface{})
	if len(entities) != 2 {
		t.Errorf("expected 2 entities, got %d", len(entities))
	}
}

func TestOntologyHandlers_GetEntity(t *testing.T) {
	service := &mockOntologyService{
		entities: map[string]*cognitiveengine.OntologyEntity{
			"node_1": {
				ID:        "node_1",
				Type:      cognitiveengine.EntityTypeNode,
				Label:     "Test Node",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	handler := NewOntologyHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/ontology/entities/node_1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var entity cognitiveengine.OntologyEntity
	if err := json.Unmarshal(w.Body.Bytes(), &entity); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if entity.ID != "node_1" {
		t.Errorf("expected entity ID node_1, got %s", entity.ID)
	}
}

func TestOntologyHandlers_GetEntityNotFound(t *testing.T) {
	service := &mockOntologyService{
		entities: map[string]*cognitiveengine.OntologyEntity{},
	}

	handler := NewOntologyHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/ontology/entities/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestOntologyHandlers_ListRelations(t *testing.T) {
	service := &mockOntologyService{
		relations: []cognitiveengine.OntologyRelation{
			{
				SourceID:  "node_1",
				TargetID:  "task_1",
				RelType:   "ran_on",
				CreatedAt: time.Now(),
			},
			{
				SourceID:  "node_1",
				TargetID:  "task_2",
				RelType:   "ran_on",
				CreatedAt: time.Now(),
			},
		},
	}

	handler := NewOntologyHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/ontology/relations?source_id=node_1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["count"].(float64) != 2 {
		t.Errorf("expected 2 relations, got %v", response["count"])
	}
}

func TestOntologyHandlers_SearchEntities(t *testing.T) {
	service := &mockOntologyService{
		entities: map[string]*cognitiveengine.OntologyEntity{
			"node_1": {
				ID:        "node_1",
				Type:      cognitiveengine.EntityTypeNode,
				Label:     "Production Node Alpha",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			"node_2": {
				ID:        "node_2",
				Type:      cognitiveengine.EntityTypeNode,
				Label:     "Development Node Beta",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	handler := NewOntologyHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/ontology/search?q=production&type=dve_node", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	results := response["results"].([]interface{})
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestOntologyHandlers_SearchMissingQuery(t *testing.T) {
	service := &mockOntologyService{}

	handler := NewOntologyHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/ontology/search", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
