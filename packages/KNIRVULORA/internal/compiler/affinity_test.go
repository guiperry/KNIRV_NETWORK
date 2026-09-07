package compiler

import (
	"math"
	"testing"

	"ulora/internal/api"
)

func baseModelSpec(family string, hiddenSize, numLayers int) api.BaseModelSpec {
	return api.BaseModelSpec{
		Family:             family,
		ParamCount:         "7B",
		HiddenSize:         hiddenSize,
		IntermediateSize:   11008,
		NumAttentionHeads:  32,
		NumKeyValueHeads:   32,
		HeadDim:            128,
		NumLayers:          numLayers,
		AttentionMechanism: "gqa",
		ActivationFunc:     "silu",
	}
}

func approxEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

func TestComputeAffinityIdenticalModels(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	tgt := baseModelSpec("llama2", 4096, 32)

	affinity := ComputeAffinity(src, tgt)
	if !approxEqual(affinity, 1.0, 1e-9) {
		t.Fatalf("expected affinity 1.0 for identical models, got %f", affinity)
	}
}

func TestComputeAffinityDifferentArchitecture(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	src.AttentionMechanism = "mha"
	src.ActivationFunc = "gelu"
	tgt := baseModelSpec("llama2", 4096, 32)

	affinity := ComputeAffinity(src, tgt)

	archScoreVal := 0.0
	dimScoreVal := 1.0
	layerScoreVal := 1.0
	expected := archScoreVal/3.0 + dimScoreVal/3.0 + layerScoreVal/3.0
	if !approxEqual(affinity, expected, 1e-9) {
		t.Fatalf("expected affinity %f, got %f", expected, affinity)
	}
}

func TestComputeAffinityDimensionDifference(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	tgt := baseModelSpec("llama2", 2048, 32)

	affinity := ComputeAffinity(src, tgt)

	archScoreVal := 1.0
	dimScoreVal := 0.5
	layerScoreVal := 1.0
	expected := archScoreVal/3.0 + dimScoreVal/3.0 + layerScoreVal/3.0
	if !approxEqual(affinity, expected, 1e-9) {
		t.Fatalf("expected affinity %f, got %f", expected, affinity)
	}
}

func TestComputeAffinityLayerDifference(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	tgt := baseModelSpec("llama2", 4096, 16)

	affinity := ComputeAffinity(src, tgt)

	archScoreVal := 1.0
	dimScoreVal := 1.0
	layerScoreVal := 1.0 - 16.0/32.0
	expected := archScoreVal/3.0 + dimScoreVal/3.0 + layerScoreVal/3.0
	if !approxEqual(affinity, expected, 1e-9) {
		t.Fatalf("expected affinity %f, got %f", expected, affinity)
	}
}

func TestComputeAffinityZeroHiddenSize(t *testing.T) {
	src := baseModelSpec("llama2", 0, 32)
	tgt := baseModelSpec("llama2", 4096, 32)

	affinity := ComputeAffinity(src, tgt)

	archScoreVal := 1.0
	dimScoreVal := 0.0
	layerScoreVal := 1.0
	expected := archScoreVal/3.0 + dimScoreVal/3.0 + layerScoreVal/3.0
	if !approxEqual(affinity, expected, 1e-9) {
		t.Fatalf("expected affinity %f, got %f", expected, affinity)
	}
}

func TestComputeAffinityZeroLayers(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 0)
	tgt := baseModelSpec("llama2", 4096, 32)

	affinity := ComputeAffinity(src, tgt)

	archScoreVal := 1.0
	dimScoreVal := 1.0
	layerScoreVal := 0.0
	expected := archScoreVal/3.0 + dimScoreVal/3.0 + layerScoreVal/3.0
	if !approxEqual(affinity, expected, 1e-9) {
		t.Fatalf("expected affinity %f, got %f", expected, affinity)
	}
}

func TestComputeAffinityCaseInsensitive(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	src.AttentionMechanism = "GQA"
	src.ActivationFunc = "SILU"
	tgt := baseModelSpec("llama2", 4096, 32)
	tgt.AttentionMechanism = "gqa"
	tgt.ActivationFunc = "silu"

	affinity := ComputeAffinity(src, tgt)
	if !approxEqual(affinity, 1.0, 1e-9) {
		t.Fatalf("expected case-insensitive match giving affinity 1.0, got %f", affinity)
	}
}

func TestSelectPathAboveThreshold(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	tgt := baseModelSpec("llama2", 4096, 32)
	rp := api.RoutingPolicy{SimilarityThreshold: 0.5}

	path := SelectPath(ComputeAffinity(src, tgt), rp)
	if path != "subspace_svd" {
		t.Fatalf("expected subspace_svd for high affinity, got %s", path)
	}
}

func TestSelectPathBelowThreshold(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	src.AttentionMechanism = "mha"
	src.ActivationFunc = "gelu"
	tgt := baseModelSpec("llama2", 4096, 32)
	rp := api.RoutingPolicy{SimilarityThreshold: 0.82}

	path := SelectPath(ComputeAffinity(src, tgt), rp)
	if path != "synthetic_distill" {
		t.Fatalf("expected synthetic_distill for low affinity, got %s", path)
	}
}

func TestSelectPathExactThreshold(t *testing.T) {
	rp := api.RoutingPolicy{SimilarityThreshold: 0.5}
	path := SelectPath(0.5, rp)
	if path != "subspace_svd" {
		t.Fatalf("expected subspace_svd at exact threshold, got %s", path)
	}
}

func TestComputePairwiseAffinities(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	targets := []api.BaseModelSpec{
		baseModelSpec("llama2", 4096, 32),
		baseModelSpec("llama3", 2048, 16),
	}

	result := ComputePairwiseAffinities(src, targets)
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	key1 := "llama2:7B"
	if val, ok := result[key1]; !ok || !approxEqual(val, 1.0, 1e-9) {
		t.Fatalf("expected affinity 1.0 for key %s, got %v", key1, val)
	}

	key2 := "llama3:7B"
	if _, ok := result[key2]; !ok {
		t.Fatalf("expected key %s in results, got %v", key2, result)
	}
}

func TestAllAboveThresholdAllAbove(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	targets := []api.BaseModelSpec{
		baseModelSpec("llama2", 4096, 32),
		baseModelSpec("llama2", 4090, 32),
	}
	threshold := 0.82

	if !AllAboveThreshold(src, targets, threshold) {
		t.Fatal("expected all above threshold")
	}
}

func TestAllAboveThresholdOneBelow(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	src.AttentionMechanism = "mha"
	src.ActivationFunc = "gelu"
	targets := []api.BaseModelSpec{
		baseModelSpec("llama2", 4096, 32),
	}

	if AllAboveThreshold(src, targets, 0.82) {
		t.Fatal("expected not all above threshold")
	}
}

func TestAllAboveThresholdEmptyTargets(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	targets := []api.BaseModelSpec{}

	if !AllAboveThreshold(src, targets, 0.82) {
		t.Fatal("expected empty targets to return true")
	}
}

func TestKeyForModel(t *testing.T) {
	m := api.BaseModelSpec{Family: "Llama-2", ParamCount: "7B"}
	key := keyForModel(m)
	expected := "llama-2:7B"
	if key != expected {
		t.Fatalf("expected '%s', got '%s'", expected, key)
	}
}

func TestArchScoreMatch(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	tgt := baseModelSpec("llama2", 4096, 32)
	if archScore(src, tgt) != 1.0 {
		t.Fatal("expected arch score 1.0 for matching arch")
	}
}

func TestArchScoreMismatch(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	src.AttentionMechanism = "mha"
	tgt := baseModelSpec("llama2", 4096, 32)
	if archScore(src, tgt) != 0.0 {
		t.Fatal("expected arch score 0.0 for mismatched arch")
	}
}

func TestDimScore(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	tgt := baseModelSpec("llama2", 2048, 32)
	score := dimScore(src, tgt)
	if !approxEqual(score, 0.5, 1e-9) {
		t.Fatalf("expected dim score 0.5, got %f", score)
	}
}

func TestLayerScoreIdentical(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	tgt := baseModelSpec("llama2", 4096, 32)
	if layerScore(src, tgt) != 1.0 {
		t.Fatal("expected layer score 1.0 for identical layers")
	}
}

func TestLayerScoreDifference(t *testing.T) {
	src := baseModelSpec("llama2", 4096, 32)
	tgt := baseModelSpec("llama2", 4096, 16)
	expected := 1.0 - 16.0/32.0
	score := layerScore(src, tgt)
	if !approxEqual(score, expected, 1e-9) {
		t.Fatalf("expected layer score %f, got %f", expected, score)
	}
}
