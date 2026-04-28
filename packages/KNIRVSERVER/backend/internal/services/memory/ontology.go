package memory

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// OntologyManager maintains an in-memory knowledge graph and routes updates
type OntologyManager struct {
	entities   map[string]*OntologyEntity
	relations  []OntologyRelation
	mu         sync.RWMutex
	logger     *zap.Logger
}

// NewOntologyManager creates a new ontology manager
func NewOntologyManager(logger *zap.Logger) *OntologyManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &OntologyManager{
		entities:  make(map[string]*OntologyEntity),
		relations: make([]OntologyRelation, 0, 128),
		logger:    logger,
	}
}

// UpsertEntity inserts or updates an entity
func (o *OntologyManager) UpsertEntity(entity *OntologyEntity) {
	o.mu.Lock()
	defer o.mu.Unlock()

	entity.UpdatedAt = time.Now()
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = entity.UpdatedAt
	}
	o.entities[entity.ID] = entity

	o.logger.Debug("ontology entity upserted",
		zap.String("id", entity.ID),
		zap.String("type", string(entity.Type)),
	)
}

// AddRelation records a directed relationship between two entities
func (o *OntologyManager) AddRelation(rel OntologyRelation) {
	o.mu.Lock()
	defer o.mu.Unlock()

	rel.CreatedAt = time.Now()
	o.relations = append(o.relations, rel)

	if len(o.relations) > 10000 {
		o.relations = o.relations[len(o.relations)-10000:]
	}
}

// GetEntity retrieves an entity by ID
func (o *OntologyManager) GetEntity(id string) (*OntologyEntity, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	e, ok := o.entities[id]
	return e, ok
}

// QueryByType returns all entities of a given type
func (o *OntologyManager) QueryByType(t OntologyEntityType) []*OntologyEntity {
	o.mu.RLock()
	defer o.mu.RUnlock()
	var result []*OntologyEntity
	for _, e := range o.entities {
		if e.Type == t {
			result = append(result, e)
		}
	}
	return result
}

// FindRelations returns all relations whose source is sourceID
func (o *OntologyManager) FindRelations(sourceID string) []OntologyRelation {
	o.mu.RLock()
	defer o.mu.RUnlock()
	var result []OntologyRelation
	for _, r := range o.relations {
		if r.SourceID == sourceID {
			result = append(result, r)
		}
	}
	return result
}

// Stats returns the current entity and relation counts
func (o *OntologyManager) Stats() (entityCount int, relationCount int) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.entities), len(o.relations)
}
