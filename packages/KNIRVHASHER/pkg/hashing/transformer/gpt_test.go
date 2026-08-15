package transformer

import (
	"math"
	"testing"
)

// tinyGorgoniteConfig keeps graphs small enough for a fast, deterministic
// unit test while still exercising every real op (embedding lookup,
// multi-head FoX attention, FFN, cross-entropy loss, backward, SGD step).
func tinyGorgoniteConfig() *GorgoniteConfig {
	return &GorgoniteConfig{
		VocabSize:    16,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    2,
		ContextLen:   6,
		FFNHiddenDim: 16,
		DecayAlpha:   0.95,
	}
}

// TestGPTForward_DependsOnInput guards the Phase 4 fix directly: the
// previous EmbeddingLayer.Forward ignored its token-ID argument and returned
// a fixed matrix, so logits were identical for any input. Two different
// token sequences must now produce different logits.
func TestGPTForward_DependsOnInput(t *testing.T) {
	cfg := tinyGorgoniteConfig()
	model := NewGPT(cfg)

	out1, err := model.Forward([]int{1, 2, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out2, err := model.Forward([]int{7, 9, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out1) != 3*cfg.VocabSize || len(out2) != 3*cfg.VocabSize {
		t.Fatalf("unexpected logits length: %d, %d (want %d)", len(out1), len(out2), 3*cfg.VocabSize)
	}

	same := true
	for i := range out1 {
		if out1[i] != out2[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("expected different logits for different input token sequences — forward pass is not content-dependent")
	}
}

// TestGPTForward_VariesByPosition confirms causal masking + positional
// encoding are real: for the same tokens, the logits at each position
// should not all be identical (previously the graph was position-independent
// too, since it never depended on tokens at all).
func TestGPTForward_VariesByPosition(t *testing.T) {
	cfg := tinyGorgoniteConfig()
	model := NewGPT(cfg)

	out, err := model.Forward([]int{4, 4, 4, 4})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pos0 := out[0:cfg.VocabSize]
	pos3 := out[3*cfg.VocabSize : 4*cfg.VocabSize]

	same := true
	for i := range pos0 {
		if pos0[i] != pos3[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("expected different logits at different positions for a repeated token (positional encoding / causal context should matter)")
	}
}

// TestGPTTrainStep_LossDecreases is the real acceptance gate for Phase 4:
// train on a single, tiny, deterministic (context -> target) example and
// confirm the loss actually goes down. This is the same rigor bar used
// earlier this session for EvolveSeeds: a trainer that "runs without
// erroring" is not evidence it learns anything — only a measured loss
// decrease is.
func TestGPTTrainStep_LossDecreases(t *testing.T) {
	cfg := tinyGorgoniteConfig()
	model := NewGPT(cfg)

	input := []int{1, 2, 3, 4}
	target := []int{2, 3, 4, 5} // next-token-prediction shift by one

	first, err := model.TrainStep(input, target, 0.05)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var last float32
	for step := 0; step < 40; step++ {
		last, err = model.TrainStep(input, target, 0.05)
		if err != nil {
			t.Fatalf("unexpected error at step %d: %v", step, err)
		}
		// NaN/Inf must fail loudly: a `>=` comparison against NaN is always
		// false in IEEE 754, so a naive "did it decrease" check alone would
		// silently let a diverged/NaN'd-out loss pass as "success".
		if math.IsNaN(float64(last)) || math.IsInf(float64(last), 0) {
			t.Fatalf("loss became non-finite at step %d: %v", step, last)
		}
	}

	if last >= first {
		t.Fatalf("expected loss to decrease over training: first=%.4f last=%.4f", first, last)
	}
	t.Logf("loss: first=%.4f last=%.4f (%d steps)", first, last, 41)
}

// TestGPTTrainStep_UpdatesPersistedWeights confirms TrainStep's gradient
// step actually survives past the ephemeral per-call graph — a real risk
// given the "rebuild the graph each call" design: if the post-step value
// isn't copied back into gpt's persisted tensors, training would silently
// no-op every step after the first.
func TestGPTTrainStep_UpdatesPersistedWeights(t *testing.T) {
	cfg := tinyGorgoniteConfig()
	model := NewGPT(cfg)

	before := make([]float32, len(model.embeddings.Data().([]float32)))
	copy(before, model.embeddings.Data().([]float32))

	if _, err := model.TrainStep([]int{1, 2, 3}, []int{2, 3, 4}, 0.1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after := model.embeddings.Data().([]float32)
	changed := false
	for i := range before {
		if before[i] != after[i] {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("expected embeddings to change after a training step")
	}
}

// TestGPTSaveLoad_RoundTrips guards SaveModel/LoadModel against the API
// change (previously walked model.graph.AllNodes(), which no longer exists).
func TestGPTSaveLoad_RoundTrips(t *testing.T) {
	cfg := tinyGorgoniteConfig()
	model := NewGPT(cfg)
	if _, err := model.TrainStep([]int{1, 2, 3}, []int{2, 3, 4}, 0.1); err != nil {
		t.Fatalf("train step: %v", err)
	}

	path := t.TempDir() + "/model.bin"
	if err := SaveModel(model, path); err != nil {
		t.Fatalf("SaveModel: %v", err)
	}

	loaded := NewGPT(cfg)
	if err := LoadModel(loaded, path); err != nil {
		t.Fatalf("LoadModel: %v", err)
	}

	want := model.embeddings.Data().([]float32)
	got := loaded.embeddings.Data().([]float32)
	if len(want) != len(got) {
		t.Fatalf("embeddings length mismatch: %d vs %d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("embeddings[%d] mismatch after round trip: %f vs %f", i, want[i], got[i])
		}
	}
}

// TestGenerate_ProducesRequestedLength is a smoke test for the autoregressive
// path (Generate -> Forward -> SampleToken), which previously indexed into a
// fixed-size, input-independent logits array.
func TestGenerate_ProducesRequestedLength(t *testing.T) {
	cfg := tinyGorgoniteConfig()
	model := NewGPT(cfg)

	out := Generate(model, []int{1, 2}, 5, 1.0, 0)
	if len(out) != 2+5 {
		t.Fatalf("expected %d tokens, got %d", 2+5, len(out))
	}
	for _, tok := range out {
		if tok < 0 || tok >= cfg.VocabSize {
			t.Fatalf("generated token %d out of vocab range [0,%d)", tok, cfg.VocabSize)
		}
	}
}
