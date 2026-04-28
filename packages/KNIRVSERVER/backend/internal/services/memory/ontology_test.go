package memory_test

import (
	"testing"
	"time"

	"backend_server/internal/services/memory"
)

func TestNewOntologyManager(t *testing.T) {
	om := memory.NewOntologyManager(nil)
	if om == nil {
		t.Fatal("expected non-nil OntologyManager")
	}
}

func TestUpsertEntity(t *testing.T) {
	om := memory.NewOntologyManager(nil)

	entity := &memory.OntologyEntity{
		ID:        "test-entity-1",
		Type:      memory.EntityTypePattern,
		Label:     "Test Entity",
		Properties: make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}

	om.UpsertEntity(entity)

	// Verify entity was added
	retrieved, found := om.GetEntity("test-entity-1")
	if !found {
		t.Fatal("expected to find entity")
	}

	if retrieved.Label != "Test Entity" {
		t.Errorf("expected label 'Test Entity', got '%s'", retrieved.Label)
	}
}

func TestAddRelation(t *testing.T) {
	om := memory.NewOntologyManager(nil)

	// Add two entities
	entity1 := &memory.OntologyEntity{
		ID:    "entity-1",
		Type:  memory.EntityTypeNode,
		Label: "Entity 1",
	}
	entity2 := &memory.OntologyEntity{
		ID:    "entity-2",
		Type:  memory.EntityTypeTask,
		Label: "Entity 2",
	}

	om.UpsertEntity(entity1)
	om.UpsertEntity(entity2)

	// Add relation
	om.AddRelation(memory.OntologyRelation{
		SourceID: "entity-1",
		TargetID: "entity-2",
		RelType:  "ran_on",
	})

	// Check relations
	rels := om.FindRelations("entity-1")
	if len(rels) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(rels))
	}

	if rels[0].TargetID != "entity-2" {
		t.Errorf("expected target 'entity-2', got '%s'", rels[0].TargetID)
	}
}

func TestQueryByType(t *testing.T) {
	om := memory.NewOntologyManager(nil)

	// Add entities of different types
	om.UpsertEntity(&memory.OntologyEntity{
		ID:   "node-1",
		Type: memory.EntityTypeNode,
	})
	om.UpsertEntity(&memory.OntologyEntity{
		ID:   "node-2",
		Type: memory.EntityTypeNode,
	})
	om.UpsertEntity(&memory.OntologyEntity{
		ID:   "task-1",
		Type: memory.EntityTypeTask,
	})

	// Query by type
	nodes := om.QueryByType(memory.EntityTypeNode)
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}

	tasks := om.QueryByType(memory.EntityTypeTask)
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}

func TestStats(t *testing.T) {
	om := memory.NewOntologyManager(nil)

	// Add some entities and relations
	om.UpsertEntity(&memory.OntologyEntity{
		ID:   "entity-1",
		Type: memory.EntityTypeNode,
	})
	om.UpsertEntity(&memory.OntologyEntity{
		ID:   "entity-2",
		Type: memory.EntityTypeTask,
	})
	om.AddRelation(memory.OntologyRelation{
		SourceID: "entity-1",
		TargetID: "entity-2",
		RelType:  "ran_on",
	})

	entityCount, relationCount := om.Stats()

	if entityCount != 2 {
		t.Errorf("expected 2 entities, got %d", entityCount)
	}

	if relationCount != 1 {
		t.Errorf("expected 1 relation, got %d", relationCount)
	}
}
