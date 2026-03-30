package cognitiveengine

import (
	"fmt"
	"log"
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

type NLPEntityExtractor struct {
	mu       sync.RWMutex
	patterns map[string]*ExtractionPattern
	eventBus *EventBus
}

type ExtractionPattern struct {
	Pattern    string
	EntityType OntologyEntityType
	Properties map[string]string
}

type ExtractedEntities struct {
	Entities   []OntologyEntity
	Relations  []OntologyRelation
	Confidence float64
}

func NewNLPEntityExtractor(eventBus *EventBus) *NLPEntityExtractor {
	extractor := &NLPEntityExtractor{
		patterns: make(map[string]*ExtractionPattern),
		eventBus: eventBus,
	}
	extractor.registerDefaultPatterns()
	return extractor
}

func (nlp *NLPEntityExtractor) registerDefaultPatterns() {
	nlp.patterns["node_id"] = &ExtractionPattern{
		Pattern:    `node[_\-]?(\w+)`,
		EntityType: EntityTypeNode,
		Properties: map[string]string{"id": "node_$1"},
	}
	nlp.patterns["task_id"] = &ExtractionPattern{
		Pattern:    `task[_\-]?(\w+)`,
		EntityType: EntityTypeTask,
		Properties: map[string]string{"id": "task_$1"},
	}
	nlp.patterns["policy_id"] = &ExtractionPattern{
		Pattern:    `policy[_\-]?(\w+)`,
		EntityType: EntityTypePolicy,
		Properties: map[string]string{"id": "policy_$1"},
	}
}

func (nlp *NLPEntityExtractor) ExtractFromText(text string) ExtractedEntities {
	nlp.mu.RLock()
	defer nlp.mu.RUnlock()

	var entities []OntologyEntity
	var relations []OntologyRelation

	for name, pattern := range nlp.patterns {
		_ = name
		entity := nlp.extractEntity(text, pattern)
		if entity != nil {
			entities = append(entities, *entity)
		}
	}

	rels := nlp.extractRelations(text)
	relations = append(relations, rels...)

	return ExtractedEntities{
		Entities:   entities,
		Relations:  relations,
		Confidence: 0.8,
	}
}

func (nlp *NLPEntityExtractor) extractEntity(text string, pattern *ExtractionPattern) *OntologyEntity {
	return &OntologyEntity{
		ID:         pattern.Properties["id"],
		Type:       pattern.EntityType,
		Label:      text,
		Properties: make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}
}

func (nlp *NLPEntityExtractor) extractRelations(text string) []OntologyRelation {
	var relations []OntologyRelation

	keywords := map[string][]string{
		"ran_on":    {"executed", "ran", "processed"},
		"caused":    {"caused", "triggered", "led to"},
		"triggered": {"triggered", "initiated"},
		"resolves":  {"resolves", "fixes", "addresses"},
		"violates":  {"violates", "breaches", "breaks"},
	}

	for relType, words := range keywords {
		for _, word := range words {
			if textContains(text, word) {
				relations = append(relations, OntologyRelation{
					RelType:   relType,
					CreatedAt: time.Now(),
				})
			}
		}
	}

	return relations
}

func textContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (nlp *NLPEntityExtractor) RegisterPattern(name string, pattern *ExtractionPattern) {
	nlp.mu.Lock()
	defer nlp.mu.Unlock()
	nlp.patterns[name] = pattern
}

func (nlp *NLPEntityExtractor) UnregisterPattern(name string) {
	nlp.mu.Lock()
	defer nlp.mu.Unlock()
	delete(nlp.patterns, name)
}

func (nlp *NLPEntityExtractor) GetPatterns() []*ExtractionPattern {
	nlp.mu.RLock()
	defer nlp.mu.RUnlock()

	patterns := make([]*ExtractionPattern, 0, len(nlp.patterns))
	for _, p := range nlp.patterns {
		patterns = append(patterns, p)
	}
	return patterns
}

type GraphQueryEngine struct {
	mu          sync.RWMutex
	ontology    *DVEOntologyManager
	queryCache  map[string]*CachedQuery
	maxCacheAge time.Duration
}

type CachedQuery struct {
	Result []*OntologyEntity
	Expiry time.Time
}

type QueryResult struct {
	Entities  []*OntologyEntity
	Relations []OntologyRelation
	Path      []string
	Depth     int
}

func NewGraphQueryEngine(ontology *DVEOntologyManager) *GraphQueryEngine {
	return &GraphQueryEngine{
		ontology:    ontology,
		queryCache:  make(map[string]*CachedQuery),
		maxCacheAge: 5 * time.Minute,
	}
}

func (gqe *GraphQueryEngine) QueryByProperty(entityType OntologyEntityType, key string, value interface{}) []*OntologyEntity {
	gqe.mu.RLock()
	cacheKey := fmt.Sprintf("%s:%s:%v", entityType, key, value)
	if cached, ok := gqe.queryCache[cacheKey]; ok && time.Now().Before(cached.Expiry) {
		defer gqe.mu.RUnlock()
		return cached.Result
	}
	gqe.mu.RUnlock()

	gqe.mu.Lock()
	defer gqe.mu.Unlock()

	entities := gqe.ontology.QueryByType(entityType)
	var results []*OntologyEntity
	for _, e := range entities {
		if v, ok := e.Properties[key]; ok && fmt.Sprintf("%v", v) == fmt.Sprintf("%v", value) {
			results = append(results, e)
		}
	}

	gqe.queryCache[cacheKey] = &CachedQuery{
		Result: results,
		Expiry: time.Now().Add(gqe.maxCacheAge),
	}

	return results
}

func (gqe *GraphQueryEngine) FindPath(sourceID, targetID string, maxDepth int) *QueryResult {
	gqe.mu.RLock()
	defer gqe.mu.RUnlock()

	result := &QueryResult{
		Entities:  make([]*OntologyEntity, 0),
		Relations: make([]OntologyRelation, 0),
		Path:      make([]string, 0),
		Depth:     0,
	}

	visited := make(map[string]bool)
	queue := []struct {
		ID   string
		Path []string
	}{{sourceID, []string{sourceID}}}

	for len(queue) > 0 && result.Depth < maxDepth {
		current := queue[0]
		queue = queue[1:]

		if current.ID == targetID {
			result.Path = current.Path
			result.Depth = len(current.Path) - 1
			return result
		}

		if visited[current.ID] {
			continue
		}
		visited[current.ID] = true

		if entity, ok := gqe.ontology.GetEntity(current.ID); ok {
			result.Entities = append(result.Entities, entity)
		}

		relations := gqe.ontology.FindRelations(current.ID)
		for _, rel := range relations {
			if !visited[rel.TargetID] {
				newPath := make([]string, len(current.Path))
				copy(newPath, current.Path)
				newPath = append(newPath, rel.TargetID)
				queue = append(queue, struct {
					ID   string
					Path []string
				}{rel.TargetID, newPath})
				result.Relations = append(result.Relations, rel)
			}
		}
	}

	return result
}

func (gqe *GraphQueryEngine) FindRelated(entityID string, relationType string, depth int) []*OntologyEntity {
	gqe.mu.RLock()
	defer gqe.mu.RUnlock()

	var results []*OntologyEntity
	visited := make(map[string]bool)

	gqe.findRelatedRecursive(entityID, relationType, depth, visited, &results)

	return results
}

func (gqe *GraphQueryEngine) findRelatedRecursive(entityID string, relationType string, depth int, visited map[string]bool, results *[]*OntologyEntity) {
	if depth < 0 || visited[entityID] {
		return
	}

	visited[entityID] = true

	if entity, ok := gqe.ontology.GetEntity(entityID); ok {
		*results = append(*results, entity)
	}

	relations := gqe.ontology.FindRelations(entityID)
	for _, rel := range relations {
		if relationType == "" || rel.RelType == relationType {
			gqe.findRelatedRecursive(rel.TargetID, relationType, depth-1, visited, results)
		}
	}
}

func (gqe *GraphQueryEngine) ClearCache() {
	gqe.mu.Lock()
	defer gqe.mu.Unlock()
	gqe.queryCache = make(map[string]*CachedQuery)
}

func (gqe *GraphQueryEngine) SetCacheAge(d time.Duration) {
	gqe.mu.Lock()
	defer gqe.mu.Unlock()
	gqe.maxCacheAge = d
}

type StateChangeMonitor struct {
	mu        sync.RWMutex
	ontology  *DVEOntologyManager
	snapshots map[string]map[string]interface{}
	listeners map[string]chan StateChange
	eventBus  *EventBus
	stopCh    chan struct{}
}

type StateChange struct {
	EntityID  string
	Property  string
	OldValue  interface{}
	NewValue  interface{}
	Timestamp time.Time
}

func NewStateChangeMonitor(ontology *DVEOntologyManager, eventBus *EventBus) *StateChangeMonitor {
	return &StateChangeMonitor{
		ontology:  ontology,
		snapshots: make(map[string]map[string]interface{}),
		listeners: make(map[string]chan StateChange),
		eventBus:  eventBus,
		stopCh:    make(chan struct{}),
	}
}

func (scm *StateChangeMonitor) Start() {
	go scm.monitorLoop()
	log.Println("StateChangeMonitor: started")
}

func (scm *StateChangeMonitor) Stop() {
	close(scm.stopCh)
	log.Println("StateChangeMonitor: stopped")
}

func (scm *StateChangeMonitor) monitorLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-scm.stopCh:
			return
		case <-ticker.C:
			scm.checkForChanges()
		}
	}
}

func (scm *StateChangeMonitor) checkForChanges() {
	scm.mu.Lock()
	defer scm.mu.Unlock()

	entities := scm.ontology.QueryByType(EntityTypeNode)
	for _, entity := range entities {
		currentProps := make(map[string]interface{})
		for k, v := range entity.Properties {
			currentProps[k] = v
		}

		previousProps := scm.snapshots[entity.ID]

		for k, newVal := range currentProps {
			oldVal, existed := previousProps[k]
			if !existed || !equals(oldVal, newVal) {
				change := StateChange{
					EntityID:  entity.ID,
					Property:  k,
					OldValue:  oldVal,
					NewValue:  newVal,
					Timestamp: time.Now(),
				}
				scm.notifyListeners(change)

				if scm.eventBus != nil {
					scm.eventBus.Publish(EngineEvent{
						Type:      EventPatternDetected,
						Source:    "state_change_monitor",
						Payload:   change,
						Timestamp: time.Now(),
					})
				}
			}
		}

		scm.snapshots[entity.ID] = currentProps
	}
}

func equals(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func (scm *StateChangeMonitor) Subscribe(entityID string) <-chan StateChange {
	scm.mu.Lock()
	defer scm.mu.Unlock()

	ch := make(chan StateChange, 100)
	scm.listeners[entityID] = ch
	return ch
}

func (scm *StateChangeMonitor) Unsubscribe(entityID string) {
	scm.mu.Lock()
	defer scm.mu.Unlock()
	if ch, ok := scm.listeners[entityID]; ok {
		close(ch)
		delete(scm.listeners, entityID)
	}
}

func (scm *StateChangeMonitor) notifyListeners(change StateChange) {
	for entityID, ch := range scm.listeners {
		if entityID == change.EntityID || entityID == "*" {
			select {
			case ch <- change:
			default:
			}
		}
	}
}

func (scm *StateChangeMonitor) GetSnapshot(entityID string) map[string]interface{} {
	scm.mu.RLock()
	defer scm.mu.RUnlock()

	if props, ok := scm.snapshots[entityID]; ok {
		snapshot := make(map[string]interface{})
		for k, v := range props {
			snapshot[k] = v
		}
		return snapshot
	}
	return nil
}

func (scm *StateChangeMonitor) GetHistory(entityID string) []StateChange {
	scm.mu.RLock()
	defer scm.mu.RUnlock()
	return nil
}
