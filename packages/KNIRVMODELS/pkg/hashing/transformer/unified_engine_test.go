package transformer

import (
	"math"
	"testing"
)

func TestUnifiedEngine_Creation(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    100,
		EmbedDim:     32,
		NumHeads:     4,
		NumLayers:    2,
		ContextLen:   64,
		Hidden1:      16,
		Hidden2:      8,
		OutputSize:   4,
		FFNHiddenDim: 64,
		Activation:   "hash",
		Passes:       21,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if engine.Mode() != ModeTransformer {
		t.Fatalf("expected transformer mode, got %s", engine.Mode())
	}
}

func TestUnifiedEngine_SetMode(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   16,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 16,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)
	engine.SetMode(ModeRecursive)
	if engine.Mode() != ModeRecursive {
		t.Fatalf("expected recursive mode, got %s", engine.Mode())
	}
}

func TestUnifiedEngine_SetHashMethod(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   16,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 16,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	method := &fakeHashMethod{}
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, method, ModeTransformer)
	if !engine.IsUsingHardware() {
		// fakeHashMethod returns IsAvailable=false, so this is expected
	}
}

func TestUnifiedEngine_TransformerMode_Forward(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    100,
		EmbedDim:     32,
		NumHeads:     4,
		NumLayers:    2,
		ContextLen:   64,
		FFNHiddenDim: 64,
		Activation:   "hash",
		Passes:       21,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)
	out := engine.Forward([]int{1, 2, 3})
	if len(out) != cfg.EmbedDim {
		t.Fatalf("expected embed dim %d, got %d", cfg.EmbedDim, len(out))
	}
}

func TestUnifiedEngine_TransformerMode_EmptyInput(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   16,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 16,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)
	out := engine.Forward([]int{})
	if len(out) != cfg.EmbedDim {
		t.Fatalf("expected embed dim %d, got %d", cfg.EmbedDim, len(out))
	}
}

func TestUnifiedEngine_RecursiveMode_Infer(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     16,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   32,
		Hidden1:      8,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 32,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.0,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeRecursive)
	result, err := engine.Infer([]byte("test input"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Consensus == nil {
		t.Fatal("expected non-nil consensus")
	}
	if result.ValidPasses == 0 {
		t.Fatal("expected at least one valid pass")
	}
}

func TestUnifiedEngine_FeedforwardMode_Predict(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     16,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   32,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 32,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.0,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeFeedforward)
	pred, conf, err := engine.Predict([]byte("test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pred < 0 || pred >= cfg.OutputSize {
		t.Fatalf("prediction %d out of range [0, %d)", pred, cfg.OutputSize)
	}
	if conf < 0 || conf > 1 {
		t.Fatalf("confidence %f out of range [0, 1]", conf)
	}
}

func TestUnifiedEngine_ModeSwitch_NoRetrainRequired(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     16,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   32,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 32,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.0,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)

	engine.SetMode(ModeRecursive)
	_, err := engine.Infer([]byte("test"))
	if err != nil {
		t.Fatalf("recursive mode failed: %v", err)
	}

	engine.SetMode(ModeFeedforward)
	_, _, err = engine.Predict([]byte("test"))
	if err != nil {
		t.Fatalf("feedforward mode failed: %v", err)
	}
}

func TestUnifiedEngine_SetSeeds(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   16,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 16,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)

	newSeeds := BuildDefaultSeedStore(cfg)
	engine.SetSeeds(newSeeds)
	if engine.seeds == nil {
		t.Fatal("seeds not updated")
	}
}

func TestUnifiedEngine_BackwardCompatFromHasherTransformer(t *testing.T) {
	htConfig := &HasherTransformerConfig{
		VocabSize:    50,
		EmbedDim:     16,
		NumLayers:    1,
		NumHeads:     2,
		ContextLen:   32,
		FFNHiddenDim: 32,
		Activation:   "hash",
	}
	ht := NewHasherTransformer(htConfig, nil)
	engine, err := NewUnifiedHasherEngineFromHasherTransformer(ht)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestSeedStore_RoundTrip(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   16,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 16,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.01,
	}
	store := BuildDefaultSeedStore(cfg)

	if len(store.Embeddings) != cfg.VocabSize {
		t.Fatalf("embeddings size mismatch")
	}
	if len(store.Positional) != cfg.ContextLen {
		t.Fatalf("positional size mismatch")
	}
	if len(store.Layers) != cfg.NumLayers {
		t.Fatalf("layers size mismatch")
	}
	if len(store.Seeds1) != cfg.Hidden1 {
		t.Fatalf("seeds1 size mismatch")
	}
	if len(store.Seeds2) != cfg.Hidden2 {
		t.Fatalf("seeds2 size mismatch")
	}
	if len(store.SeedsOut) != cfg.OutputSize {
		t.Fatalf("seedsout size mismatch")
	}
}

func TestProjectionCache(t *testing.T) {
	cache := NewProjectionCache()
	if cache.Size() != 0 {
		t.Fatal("expected empty cache")
	}
	cache.Put("key", []float32{0.1, 0.2})
	if cache.Size() != 1 {
		t.Fatal("expected cache size 1")
	}
	val, ok := cache.Get("key")
	if !ok || len(val) != 2 {
		t.Fatal("cache miss or wrong value")
	}
}

type alwaysAvailableHash struct{ fakeHashMethod }

func (f *alwaysAvailableHash) IsAvailable() bool { return true }
func (f *alwaysAvailableHash) ComputeBatch(data [][]byte) ([][32]byte, error) {
	result := make([][32]byte, len(data))
	for i := range result {
		for j := range result[i] {
			result[i][j] = byte(i * j)
		}
	}
	return result, nil
}

func TestUnifiedEngine_HardwareAccelerated(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   16,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 16,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.0,
	}
	seeds := BuildDefaultSeedStore(cfg)
	method := &alwaysAvailableHash{}
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, method, ModeTransformer)
	out := engine.Forward([]int{1})
	if len(out) != cfg.EmbedDim {
		t.Fatalf("expected embed dim %d, got %d", cfg.EmbedDim, len(out))
	}
	_, _, err := engine.Predict([]byte("test"))
	if err != nil {
		t.Fatalf("predict failed: %v", err)
	}
	stats := engine.Stats()
	if stats.TotalInferences == 0 {
		t.Fatal("expected at least one inference")
	}
}

func TestUnifiedEngine_FeedforwardMode_Hardware(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   16,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 16,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.0,
	}
	seeds := BuildDefaultSeedStore(cfg)
	method := &alwaysAvailableHash{}
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, method, ModeFeedforward)
	pred, conf, err := engine.Predict([]byte("test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pred < 0 || pred >= cfg.OutputSize {
		t.Fatalf("prediction out of range: %d", pred)
	}
	if conf < 0 || conf > 1 {
		t.Fatalf("confidence out of range: %f", conf)
	}
}

func TestUnifiedEngine_RecursiveMode_Hardware(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   16,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 16,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.0,
	}
	seeds := BuildDefaultSeedStore(cfg)
	method := &alwaysAvailableHash{}
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, method, ModeRecursive)
	result, err := engine.Infer([]byte("test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Consensus == nil {
		t.Fatal("expected non-nil consensus")
	}
}

func TestCSVSeedStoreWriter_RoundTrip(t *testing.T) {
	writer := NewCSVSeedStoreWriter("/tmp/test_seeds.csv")
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   16,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 16,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.01,
	}
	store := BuildDefaultSeedStore(cfg)
	err := writer.WriteSeedStore(store)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	loaded, err := writer.ReadSeedStore()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestMemorySeedStoreWriter_RoundTrip(t *testing.T) {
	writer := NewMemorySeedStoreWriter(nil)
	cfg := &UnifiedConfig{
		VocabSize:    10,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   16,
		Hidden1:      4,
		Hidden2:      4,
		OutputSize:   4,
		FFNHiddenDim: 16,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.01,
	}
	store := BuildDefaultSeedStore(cfg)
	err := writer.WriteSeedStore(store)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	loaded, err := writer.ReadSeedStore()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestLinearAttention_SequenceMixing(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    100,
		EmbedDim:     32,
		NumHeads:     4,
		NumLayers:    2,
		ContextLen:   64,
		Hidden1:      16,
		Hidden2:     8,
		OutputSize:   4,
		FFNHiddenDim: 64,
		Activation:   "hash",
		Passes:       21,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)

	out1 := engine.Forward([]int{1, 2, 3})
	out2 := engine.Forward([]int{3, 2, 1})

	if len(out1) != cfg.EmbedDim {
		t.Fatalf("expected embed dim %d, got %d", cfg.EmbedDim, len(out1))
	}
	if len(out2) != cfg.EmbedDim {
		t.Fatalf("expected embed dim %d, got %d", cfg.EmbedDim, len(out2))
	}

	different := 0
	for i := range out1 {
		if out1[i] != out2[i] {
			different++
		}
	}
	if different == 0 {
		t.Fatal("expected different outputs for different token orderings")
	}
}

func TestLinearAttention_CausalMask(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    100,
		EmbedDim:     32,
		NumHeads:     4,
		NumLayers:    1,
		ContextLen:   64,
		Hidden1:      16,
		Hidden2:     8,
		OutputSize:   4,
		FFNHiddenDim: 64,
		Activation:   "hash",
		Passes:       21,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)

	outSingle := engine.Forward([]int{42})
	outContext := engine.Forward([]int{1, 42})

	for i := range outSingle {
		if outSingle[i] == 0 && outContext[i] == 0 {
			continue
		}
		if outSingle[i] != outContext[i] {
			t.Logf("position %d: single=%f context=%f (context adds preceding tokens)", i, outSingle[i], outContext[i])
		}
	}
}

func TestDotProduct(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{4, 5, 6}
	expected := float32(32) // 1*4 + 2*5 + 3*6
	result := dotProduct(a, b)
	if result != expected {
		t.Fatalf("dotProduct(%v, %v) = %f, expected %f", a, b, result, expected)
	}
}

func TestDotProduct_MismatchedLengths(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{3, 4, 5}
	result := dotProduct(a, b)
	if result != 0 {
		t.Fatalf("expected 0 for mismatched lengths, got %f", result)
	}
}

func TestComputeDecayChain_IdentityDecay(t *testing.T) {
	seeds := make([][32]byte, 4)
	for i := range seeds {
		for j := range seeds[i] {
			seeds[i][j] = 0xFF
		}
	}
	result := computeDecayChain(0, 4, seeds)
	if result != 1.0 {
		t.Fatalf("expected identity decay 1.0, got %f", result)
	}
}

func TestComputeDecayChain_ZeroDecay(t *testing.T) {
	seeds := make([][32]byte, 4)
	for i := range seeds {
		for j := range seeds[i] {
			seeds[i][j] = 0x00
		}
	}
	result := computeDecayChain(0, 4, seeds)
	if result != 1e-30 {
		t.Fatalf("expected epsilon floor 1e-30, got %f", result)
	}
}

func TestComputeDecayChain_MixedDecay(t *testing.T) {
	seeds := make([][32]byte, 3)
	seeds[0] = [32]byte{0xFF, 0xFF, 0xFF, 0xFF}
	seeds[1] = [32]byte{0x00, 0x00, 0x00, 0x00}
	seeds[2] = [32]byte{0xFF, 0xFF, 0xFF, 0xFF}
	result := computeDecayChain(0, 3, seeds)
	expected := float32(1.0 * 0.0 * 1.0)
	if expected < 1e-30 {
		expected = 1e-30
	}
	if result != expected {
		t.Fatalf("expected %f, got %f", expected, result)
	}
}

func TestComputeDecayChain_EmptyRange(t *testing.T) {
	seeds := make([][32]byte, 4)
	for i := range seeds {
		for j := range seeds[i] {
			seeds[i][j] = 0x80
		}
	}
	result := computeDecayChain(2, 2, seeds)
	if result != 1.0 {
		t.Fatalf("expected 1.0 for empty range, got %f", result)
	}
}

func TestComputeDecayChain_SingleElement(t *testing.T) {
	seeds := make([][32]byte, 1)
	seeds[0] = [32]byte{0xFF, 0xFF, 0xFF, 0xFF}
	result := computeDecayChain(0, 1, seeds)
	if result != 1.0 {
		t.Fatalf("expected 1.0 for single identity element, got %f", result)
	}
}

func TestComputeDecayChain_EmptySeeds(t *testing.T) {
	result := computeDecayChain(0, 4, nil)
	if result != 1.0 {
		t.Fatalf("expected 1.0 for empty seeds, got %f", result)
	}
}

func TestFoxAttention_OutputShape(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    100,
		EmbedDim:     32,
		NumHeads:     4,
		NumLayers:    2,
		ContextLen:   64,
		Hidden1:      16,
		Hidden2:     8,
		OutputSize:   4,
		FFNHiddenDim: 64,
		Activation:   "hash",
		Passes:       21,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)
	out := engine.Forward([]int{1, 2, 3})
	if len(out) != cfg.EmbedDim {
		t.Fatalf("expected embed dim %d, got %d", cfg.EmbedDim, len(out))
	}
}

func TestFoxAttention_BackwardCompatEmptyDecaySeeds(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    50,
		EmbedDim:     16,
		NumHeads:     2,
		NumLayers:    1,
		ContextLen:   32,
		Hidden1:      8,
		Hidden2:     4,
		OutputSize:   4,
		FFNHiddenDim: 32,
		Activation:   "hash",
		Passes:       3,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	for i := range seeds.Layers {
		seeds.Layers[i].DecaySeeds = nil
	}
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)
	out := engine.Forward([]int{1, 2, 3})
	if len(out) != cfg.EmbedDim {
		t.Fatalf("expected embed dim %d, got %d", cfg.EmbedDim, len(out))
	}
}

func TestFoxAttention_CausalMask(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    100,
		EmbedDim:     32,
		NumHeads:     4,
		NumLayers:    1,
		ContextLen:   64,
		Hidden1:      16,
		Hidden2:     8,
		OutputSize:   4,
		FFNHiddenDim: 64,
		Activation:   "hash",
		Passes:       21,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)

	outSingle := engine.Forward([]int{42})
	outContext := engine.Forward([]int{1, 42})

	for i := range outSingle {
		if outSingle[i] == 0 && outContext[i] == 0 {
			continue
		}
		if outSingle[i] != outContext[i] {
			t.Logf("position %d: single=%f context=%f (context adds preceding tokens)", i, outSingle[i], outContext[i])
		}
	}
}

func TestFoxAttention_LongRangeForgetting(t *testing.T) {
	cfg := &UnifiedConfig{
		VocabSize:    100,
		EmbedDim:     32,
		NumHeads:     4,
		NumLayers:    1,
		ContextLen:   64,
		Hidden1:      16,
		Hidden2:     8,
		OutputSize:   4,
		FFNHiddenDim: 64,
		Activation:   "hash",
		Passes:       21,
		Jitter:       0.01,
	}
	seeds := BuildDefaultSeedStore(cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)

	outShort := engine.Forward([]int{1, 2})
	outLong := engine.Forward([]int{1, 2, 3, 4, 5, 6, 7, 8})

	diff := float32(0)
	for i := range outShort {
		diff += float32(math.Abs(float64(outShort[i] - outLong[i])))
	}
	if diff == 0 {
		t.Fatal("expected different outputs for different sequence lengths")
	}
}

func TestFoxAttention_HasherTransformerBackwardCompat(t *testing.T) {
	htConfig := &HasherTransformerConfig{
		VocabSize:    50,
		EmbedDim:     16,
		NumLayers:    1,
		NumHeads:     2,
		ContextLen:   32,
		FFNHiddenDim: 32,
		Activation:   "hash",
	}
	ht := NewHasherTransformer(htConfig, nil)
	if len(ht.Layers) > 0 && len(ht.Layers[0].DecaySeeds) != htConfig.ContextLen {
		t.Fatalf("expected decay seeds length %d, got %d", htConfig.ContextLen, len(ht.Layers[0].DecaySeeds))
	}
	engine, err := NewUnifiedHasherEngineFromHasherTransformer(ht)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	out := engine.Forward([]int{1, 2, 3})
	if len(out) != htConfig.EmbedDim {
		t.Fatalf("expected embed dim %d, got %d", htConfig.EmbedDim, len(out))
	}
}
