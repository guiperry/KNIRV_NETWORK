package transformer

import (
	"reflect"
	"testing"
)

func tinyEvolveConfig() *UnifiedConfig {
	return &UnifiedConfig{
		VocabSize:    20,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   16,
		FFNHiddenDim: 16,
		Activation:   "tanh",
		Passes:       3,
	}
}

func syntheticRecords() []TrainingRecord {
	return []TrainingRecord{
		{Context: []int{1, 2, 3}, TargetTokenID: 5},
		{Context: []int{2, 3, 4}, TargetTokenID: 6},
		{Context: []int{1, 3, 5}, TargetTokenID: 7},
		{Context: []int{4, 2, 1}, TargetTokenID: 8},
	}
}

func TestEvolveSeeds_ImprovesOrHoldsFitness(t *testing.T) {
	cfg := tinyEvolveConfig()
	seeds := BuildDefaultSeedStore(cfg)
	records := syntheticRecords()

	evCfg := EvolveConfig{Generations: 25, MutationsPerGen: 4, NegativeSamples: 8, Seed: 42}
	evolved, history, err := EvolveSeeds(cfg, seeds, records, evCfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evolved == nil {
		t.Fatal("expected non-nil evolved seeds")
	}
	if len(history) != evCfg.Generations+1 {
		t.Fatalf("expected %d history entries, got %d", evCfg.Generations+1, len(history))
	}
	for i := 1; i < len(history); i++ {
		if history[i] < history[i-1] {
			t.Fatalf("fitness decreased at generation %d: %f -> %f", i, history[i-1], history[i])
		}
	}
	if history[len(history)-1] < history[0] {
		t.Fatalf("expected final fitness >= initial fitness, got %f < %f", history[len(history)-1], history[0])
	}
	if reflect.DeepEqual(evolved.Embeddings, seeds.Embeddings) && reflect.DeepEqual(evolved.OutputSeed, seeds.OutputSeed) {
		t.Fatal("expected evolved seeds to differ from the untrained baseline")
	}
}

func TestEvolveSeeds_NoRecordsErrors(t *testing.T) {
	cfg := tinyEvolveConfig()
	seeds := BuildDefaultSeedStore(cfg)
	_, _, err := EvolveSeeds(cfg, seeds, nil, DefaultEvolveConfig())
	if err == nil {
		t.Fatal("expected error for empty records")
	}
}

func TestHasherTrainer_TrainActuallyChangesSeedsAndIsNotANoOp(t *testing.T) {
	htCfg := &HasherTransformerConfig{
		VocabSize:    20,
		EmbedDim:     8,
		NumLayers:    1,
		NumHeads:     2,
		ContextLen:   16,
		FFNHiddenDim: 16,
		Activation:   "tanh",
	}
	model := NewHasherTransformer(htCfg, nil)
	baselineOutputSeed := model.OutputSeed
	baselineEmbeddings := make([][][32]byte, len(model.Embeddings))
	copy(baselineEmbeddings, model.Embeddings)

	data := []HasherDataSample{
		{InputTokens: []int{1, 2, 3}, OutputTokens: []int{5}},
		{InputTokens: []int{2, 3, 4}, OutputTokens: []int{6}},
		{InputTokens: []int{1, 3, 5}, OutputTokens: []int{7}},
	}
	trainer := NewHasherTrainer(model, &HasherTrainingConfig{Epochs: 1}, data)

	if err := trainer.Train(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	changed := model.OutputSeed != baselineOutputSeed
	for i := range model.Embeddings {
		if !reflect.DeepEqual(model.Embeddings[i], baselineEmbeddings[i]) {
			changed = true
		}
	}
	if !changed {
		t.Fatal("expected Train to modify at least one seed — it must no longer be a no-op")
	}
}

func TestHasherTrainer_TrainNoDataErrors(t *testing.T) {
	htCfg := &HasherTransformerConfig{
		VocabSize: 10, EmbedDim: 4, NumLayers: 1, NumHeads: 1, ContextLen: 8, FFNHiddenDim: 8, Activation: "tanh",
	}
	model := NewHasherTransformer(htCfg, nil)
	trainer := NewHasherTrainer(model, &HasherTrainingConfig{}, nil)
	if err := trainer.Train(); err == nil {
		t.Fatal("expected error for empty training data")
	}
}

func TestLoadTrainingRecordsFromLedger_MissingLedger(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadTrainingRecordsFromLedger(dir, 100, 10)
	if err == nil {
		t.Fatal("expected error for missing ledger")
	}
}

func TestLoadTrainingRecordsFromLedger_ParsesAndCaps(t *testing.T) {
	dir := t.TempDir()
	writeTestLedger(t, dir, []string{
		`{"timestamp":"2026-01-01T00:00:00Z","source_file":"x","target_token_id":3,"asic_slots":[1,2,3,4,5,6,7,8,9,10,11,12],"best_seed":"KSX5x2mzKf23YCPjhEVNacOzrEx0aC10lqwavlVn8FU=","seed_bytes":32}`,
		`{"timestamp":"2026-01-01T00:00:01Z","source_file":"x","target_token_id":7,"asic_slots":[1,2,3,4,5,6,7,8,9,10,11,13],"best_seed":"TSlfKOB4dyLw/cGW5CNjfu//wWMgHuN1CLRiKuZ1u/A=","seed_bytes":32}`,
		`{"timestamp":"2026-01-01T00:00:02Z","source_file":"x","target_token_id":9,"asic_slots":[1,2,3,4,5,6,7,8,9,10,11,14],"best_seed":"5u00znIhG5uEAkJ+gqNOWZ6vFjYzBKxdm9hyhqVnyw4=","seed_bytes":32}`,
	})
	records, err := LoadTrainingRecordsFromLedger(dir, 20, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected records capped at 2, got %d", len(records))
	}
	for _, r := range records {
		if len(r.Context) != 12 {
			t.Fatalf("expected 12-token context from asic_slots, got %d", len(r.Context))
		}
		if r.TargetTokenID < 0 || r.TargetTokenID >= 20 {
			t.Fatalf("target token out of vocab range: %d", r.TargetTokenID)
		}
	}
}
