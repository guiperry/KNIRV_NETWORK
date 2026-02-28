package icme_test

import (
	"testing"
	"time"

	icme "backend_server/internal/services/icme"
)

func TestIntentRegistry_GlobalScope(t *testing.T) {
	t.Skip("Requires BuntDB instance")
}

func TestIntentRegistry_DVEScope(t *testing.T) {
	t.Skip("Requires BuntDB instance")
}

func TestTemporalHypergraph_InsertSignal(t *testing.T) {
	hg := icme.NewTemporalHypergraph(10*time.Minute, 1000, nil)

	signal := &icme.IntentionalSignal{
		ID:            "test-signal-1",
		AgentID:       "agent-1",
		DVEID:         "dve-1",
		Content:       "The API returned error 500",
		Timestamp:     time.Now(),
		ObjectiveName: "api-reliability",
		Entities: []icme.ExtractedEntity{
			{ID: "ent_1", Text: "API", Label: "ERROR", Score: 0.9, Start: 4, End: 7},
			{ID: "ent_2", Text: "500", Label: "ERROR", Score: 0.9, Start: 19, End: 22},
		},
		Relations: []icme.ExtractedRelation{
			{FromEntityID: "ent_1", ToEntityID: "ent_2", RelationType: "CAUSED_BY", Confidence: 0.75},
		},
	}

	hg.InsertSignal(signal)

	nodes, edges := hg.Snapshot()
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
	if len(edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(edges))
	}
}

func TestTemporalHypergraph_Neighbors(t *testing.T) {
	hg := icme.NewTemporalHypergraph(10*time.Minute, 1000, nil)

	signal := &icme.IntentionalSignal{
		ID:        "test-signal-1",
		AgentID:   "agent-1",
		Timestamp: time.Now(),
		Entities: []icme.ExtractedEntity{
			{ID: "ent_1", Text: "API", Label: "ERROR", Score: 0.9, Start: 0, End: 3},
			{ID: "ent_2", Text: "Database", Label: "ERROR", Score: 0.9, Start: 0, End: 8},
		},
		Relations: []icme.ExtractedRelation{
			{FromEntityID: "ent_1", ToEntityID: "ent_2", RelationType: "DEPENDS_ON", Confidence: 0.8},
		},
	}

	hg.InsertSignal(signal)

	neighbors := hg.Neighbors("API:ERROR", 2, "")
	if len(neighbors) != 2 {
		t.Errorf("expected 2 neighbors, got %d", len(neighbors))
	}
}

func TestDelegationFramework_HardBoundary(t *testing.T) {
	t.Skip("Requires IntentRegistry")
}

func TestDelegationFramework_Authorization(t *testing.T) {
	t.Skip("Requires IntentRegistry")
}

func TestHybridSearchEngine_Search(t *testing.T) {
	t.Skip("Requires full ICME setup")
}

func TestFactualityAdapter_ComputeAlignmentScore(t *testing.T) {
	t.Skip("Requires IntentRegistry and validation endpoint")
}
