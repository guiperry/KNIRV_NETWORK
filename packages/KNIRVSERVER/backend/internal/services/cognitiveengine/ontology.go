package cognitiveengine

import (
	"fmt"
	"sync"
	"time"

	"backend_server/internal/services/icme"
	"go.uber.org/zap"
)

// OntologyEntityType classifies knowledge-graph entities in the DVE ontology.
type OntologyEntityType string

const (
	EntityTypeNode       OntologyEntityType = "dve_node"
	EntityTypeTask       OntologyEntityType = "validation_task"
	EntityTypeResult     OntologyEntityType = "validation_result"
	EntityTypeAdaptation OntologyEntityType = "adaptation_event"
	EntityTypePattern    OntologyEntityType = "failure_pattern"
	EntityTypePolicy     OntologyEntityType = "guardrail_policy"
	EntityTypeViolation  OntologyEntityType = "policy_violation"
)

// OntologyEntity is a typed node in the DVE knowledge graph.
type OntologyEntity struct {
	ID         string
	Type       OntologyEntityType
	Label      string
	Properties map[string]interface{}
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// OntologyRelation is a directed, typed edge in the knowledge graph.
type OntologyRelation struct {
	SourceID   string
	TargetID   string
	RelType    string // "ran_on", "caused", "triggered", "resolves", "violates"
	Properties map[string]interface{}
	CreatedAt  time.Time
}

// DVEOntologyManager maintains an in-memory DVE ontology and routes updates
// into the KNIRVGRAPH TemporalHypergraph for graph-based reasoning.
//
// The hypergraph is optional: pass nil to run in standalone mode.
type DVEOntologyManager struct {
	entities   map[string]*OntologyEntity
	relations  []OntologyRelation
	hypergraph *icme.TemporalHypergraph
	logger     *zap.Logger
	mu         sync.RWMutex
}

// NewDVEOntologyManager creates an ontology manager.
func NewDVEOntologyManager(hg *icme.TemporalHypergraph, logger *zap.Logger) *DVEOntologyManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DVEOntologyManager{
		entities:   make(map[string]*OntologyEntity),
		relations:  make([]OntologyRelation, 0, 128),
		hypergraph: hg,
		logger:     logger,
	}
}

// UpsertEntity inserts or updates an entity and routes a corresponding
// IntentionalSignal into the KNIRVGRAPH hypergraph.
func (o *DVEOntologyManager) UpsertEntity(entity *OntologyEntity) {
	o.mu.Lock()
	defer o.mu.Unlock()

	entity.UpdatedAt = time.Now()
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = entity.UpdatedAt
	}
	o.entities[entity.ID] = entity

	// Route to KNIRVGRAPH hypergraph as an IntentionalSignal
	if o.hypergraph != nil {
		sig := &icme.IntentionalSignal{
			ID:            fmt.Sprintf("ont_%s_%d", entity.ID, time.Now().UnixNano()),
			Source:        icme.SourceValidation,
			ObjectiveName: string(entity.Type),
			Content:       entity.Label,
			Timestamp:     time.Now(),
			Scope:         icme.ScopeDVE,
			// Add the entity itself as an extracted entity so the hypergraph
			// builds a concrete node for it.
			Entities: []icme.ExtractedEntity{
				{
					ID:    entity.ID,
					Text:  entity.Label,
					Label: string(entity.Type),
					Score: 1.0,
				},
			},
		}
		o.hypergraph.InsertSignal(sig)
	}

	o.logger.Debug("ontology entity upserted",
		zap.String("id", entity.ID),
		zap.String("type", string(entity.Type)),
	)
}

// AddRelation records a directed relationship between two entities.
// The relation list is capped at 10 000 entries (rolling window).
func (o *DVEOntologyManager) AddRelation(rel OntologyRelation) {
	o.mu.Lock()
	defer o.mu.Unlock()

	rel.CreatedAt = time.Now()
	o.relations = append(o.relations, rel)

	if len(o.relations) > 10_000 {
		o.relations = o.relations[len(o.relations)-10_000:]
	}

	// If both entities exist, insert a hypergraph signal with the relation
	if o.hypergraph != nil {
		src, srcOK := o.entities[rel.SourceID]
		dst, dstOK := o.entities[rel.TargetID]
		if srcOK && dstOK {
			sig := &icme.IntentionalSignal{
				ID:            fmt.Sprintf("rel_%s_%s_%d", rel.SourceID, rel.TargetID, time.Now().UnixNano()),
				Source:        icme.SourceValidation,
				ObjectiveName: rel.RelType,
				Content:       fmt.Sprintf("%s %s %s", src.Label, rel.RelType, dst.Label),
				Timestamp:     time.Now(),
				Scope:         icme.ScopeDVE,
				Entities: []icme.ExtractedEntity{
					{ID: rel.SourceID, Text: src.Label, Label: string(src.Type), Score: 1.0},
					{ID: rel.TargetID, Text: dst.Label, Label: string(dst.Type), Score: 1.0},
				},
				Relations: []icme.ExtractedRelation{
					{
						FromEntityID: rel.SourceID,
						ToEntityID:   rel.TargetID,
						RelationType: rel.RelType,
						Confidence:   1.0,
					},
				},
			}
			o.hypergraph.InsertSignal(sig)
		}
	}
}

// GetEntity retrieves an entity by ID.
func (o *DVEOntologyManager) GetEntity(id string) (*OntologyEntity, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	e, ok := o.entities[id]
	return e, ok
}

// QueryByType returns all entities of a given type.
func (o *DVEOntologyManager) QueryByType(t OntologyEntityType) []*OntologyEntity {
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

// FindRelations returns all relations whose source is sourceID.
func (o *DVEOntologyManager) FindRelations(sourceID string) []OntologyRelation {
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

// IndexLearningState converts the engine's LearningState into ontology entities
// and pushes them to the KNIRVGRAPH hypergraph.  Call this from the periodic
// ontology update loop.
func (o *DVEOntologyManager) IndexLearningState(state *LearningState) {
	// Node performance → EntityTypeNode
	for nodeID, nm := range state.NodePerformance {
		o.UpsertEntity(&OntologyEntity{
			ID:    "node_" + nodeID,
			Type:  EntityTypeNode,
			Label: fmt.Sprintf("DVE Node %s", nodeID),
			Properties: map[string]interface{}{
				"tasks_processed":     nm.TasksProcessed,
				"success_rate":        nm.SuccessRate,
				"reliability_score":   nm.ReliabilityScore,
				"avg_processing_time": nm.AvgProcessingTime,
				"last_active":         nm.LastActive,
			},
		})
	}

	// Task-type performance → EntityTypeTask
	for taskType, tm := range state.TaskTypePerformance {
		taskEntity := &OntologyEntity{
			ID:    "tasktype_" + taskType,
			Type:  EntityTypeTask,
			Label: fmt.Sprintf("Task Type: %s", taskType),
			Properties: map[string]interface{}{
				"tasks_processed":     tm.TasksProcessed,
				"success_rate":        tm.SuccessRate,
				"avg_processing_time": tm.AvgProcessingTime,
				"avg_score":           tm.AvgScore,
				"failure_patterns":    tm.FailurePatterns,
			},
		}
		o.UpsertEntity(taskEntity)

		// Record "ran_on" relations between task types and nodes that processed them
		for nodeID := range state.NodePerformance {
			o.AddRelation(OntologyRelation{
				SourceID: "tasktype_" + taskType,
				TargetID: "node_" + nodeID,
				RelType:  "ran_on",
			})
		}
	}

	// Adaptation history → EntityTypeAdaptation
	for _, event := range state.AdaptationHistory {
		o.UpsertEntity(&OntologyEntity{
			ID:    "adapt_" + event.ID,
			Type:  EntityTypeAdaptation,
			Label: fmt.Sprintf("Adaptation: %s", event.AdaptationType),
			Properties: map[string]interface{}{
				"trigger_reason":  event.TriggerReason,
				"adaptation_type": event.AdaptationType,
				"timestamp":       event.Timestamp,
				"expected_impact": event.ExpectedImpact,
			},
		})
	}
}

// IndexFailurePatterns converts the PatternAnalyzer's patterns into ontology
// entities and links them to the task types they affect.
func (o *DVEOntologyManager) IndexFailurePatterns(patterns map[string]*FailurePattern) {
	for key, fp := range patterns {
		patternEntity := &OntologyEntity{
			ID:    "pattern_" + key,
			Type:  EntityTypePattern,
			Label: fp.Description,
			Properties: map[string]interface{}{
				"frequency":        fp.Frequency,
				"avg_impact":       fp.AvgImpact,
				"suggested_action": fp.SuggestedAction,
				"last_seen":        fp.LastSeen,
			},
		}
		o.UpsertEntity(patternEntity)

		// Link pattern to affected task types
		for _, taskType := range fp.TaskTypes {
			o.AddRelation(OntologyRelation{
				SourceID: "pattern_" + key,
				TargetID: "tasktype_" + taskType,
				RelType:  "affects",
			})
		}
	}
}

// Stats returns the current entity and relation counts.
func (o *DVEOntologyManager) Stats() (entityCount int, relationCount int) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.entities), len(o.relations)
}
