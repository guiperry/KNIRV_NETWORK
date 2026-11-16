//go:debug randseednop=0

package gpt

// Fixed-seed golden test for GPT logits (PendingImplementations.md 9.1,
// key [KNIRVMODELS-HANDS]).
//
// Purpose: give a later move/refactor of pkg/hands a before/after comparison
// point. The test builds a small, CPU-only config, seeds the package's own
// weight initialiser (randDense, which draws from the global math/rand
// source), runs Forward on a fixed token sequence and compares the flattened
// [seqLen x VocabSize] logits with a committed fixture.
//
// Determinism: `//go:debug randseednop=0` restores the pre-Go-1.24 behaviour
// of rand.Seed so that the global source, and therefore randDense, is
// reproducible. Without it rand.Seed is a no-op on Go 1.24+ and the golden
// would not be stable.
//
// Tolerance: |got - want| <= goldenAbsTol + goldenRelTol*|want| per logit.
// The forward pass is pure float32 CPU code with a fixed op order, so a
// same-platform run is expected to match exactly; the tolerance only absorbs
// cross-platform float32 differences (FMA, libm exp/log).
//
// Update: `go test ./pkg/hands/gpt -run TestGPTGoldenLogits -update`
// rewrites testdata/gpt_logits_golden.json and nothing else. Do not run
// -update after an intentional numerical change without reviewing the diff.

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

var updateGoldenLogits = flag.Bool("update", false,
	"rewrite testdata/gpt_logits_golden.json from the current Forward pass (testdata only)")

const (
	goldenFixturePath = "testdata/gpt_logits_golden.json"
	goldenSeed        = int64(9101)
	goldenAbsTol      = 1e-5
	goldenRelTol      = 1e-4
)

// goldenTokens is the fixed input sequence. Length equals ContextLen so the
// causal mask and positional encoding are fully exercised.
var goldenTokens = []int{3, 1, 4, 1, 5, 9}

// goldenConfig is the small CPU config used for the golden. It matches the
// shape used by the existing unit tests (tinyGorgoniteConfig) on purpose so
// that failures are easy to reason about.
func goldenConfig() *Config {
	return &Config{
		VocabSize:    16,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    2,
		ContextLen:   6,
		FFNHiddenDim: 16,
		DecayAlpha:   0.95,
	}
}

type goldenFixture struct {
	Description string    `json:"description"`
	Seed        int64     `json:"seed"`
	Tokens      []int     `json:"tokens"`
	AbsTol      float64   `json:"abs_tol"`
	RelTol      float64   `json:"rel_tol"`
	Config      Config    `json:"config"`
	Shape       []int     `json:"shape"`
	Logits      []float32 `json:"logits"`
}

// goldenForward seeds the global source, builds a fresh model with the
// package's own initialiser and returns the logits for goldenTokens.
func goldenForward(t *testing.T) ([]float32, *Config) {
	t.Helper()
	cfg := goldenConfig()
	rand.Seed(goldenSeed)
	model := NewGPT(cfg)
	logits, err := model.Forward(goldenTokens)
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	return logits, cfg
}

func TestGPTGoldenLogits(t *testing.T) {
	got, cfg := goldenForward(t)
	wantLen := len(goldenTokens) * cfg.VocabSize
	if len(got) != wantLen {
		t.Fatalf("logits length = %d, want %d", len(got), wantLen)
	}

	if *updateGoldenLogits {
		writeGoldenFixture(t, got, cfg)
		t.Logf("rewrote %s (%d logits)", goldenFixturePath, len(got))
		return
	}

	raw, err := os.ReadFile(goldenFixturePath)
	if err != nil {
		t.Fatalf("read golden fixture %s: %v (generate with -update)", goldenFixturePath, err)
	}
	var fx goldenFixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatalf("parse golden fixture: %v", err)
	}
	if fx.Seed != goldenSeed {
		t.Fatalf("fixture seed = %d, want %d", fx.Seed, goldenSeed)
	}
	if fx.Config != *cfg {
		t.Fatalf("fixture config = %+v, want %+v", fx.Config, *cfg)
	}
	if fmt.Sprint(fx.Tokens) != fmt.Sprint(goldenTokens) {
		t.Fatalf("fixture tokens = %v, want %v", fx.Tokens, goldenTokens)
	}
	if len(fx.Logits) != len(got) {
		t.Fatalf("fixture has %d logits, forward produced %d", len(fx.Logits), len(got))
	}

	var worst float64
	worstIdx := -1
	for i := range got {
		diff := math.Abs(float64(got[i]) - float64(fx.Logits[i]))
		limit := fx.AbsTol + fx.RelTol*math.Abs(float64(fx.Logits[i]))
		if diff > limit {
			t.Errorf("logit[pos=%d,vocab=%d] = %.9g, golden %.9g (|diff| %.3g > tol %.3g)",
				i/cfg.VocabSize, i%cfg.VocabSize, got[i], fx.Logits[i], diff, limit)
			if worstIdx < 0 {
				worstIdx = i
			}
		}
		if diff > worst {
			worst = diff
		}
	}
	t.Logf("max |diff| vs golden = %.3g over %d logits", worst, len(got))
}

// TestGPTGoldenLogitsDeterministic checks that two fresh builds with the same
// seed produce bit-identical logits within one process.
func TestGPTGoldenLogitsDeterministic(t *testing.T) {
	a, _ := goldenForward(t)
	b, _ := goldenForward(t)
	if len(a) != len(b) {
		t.Fatalf("length mismatch %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("logit %d differs between identical seeded runs: %v vs %v", i, a[i], b[i])
		}
	}
}

func writeGoldenFixture(t *testing.T, logits []float32, cfg *Config) {
	t.Helper()
	fx := goldenFixture{
		Description: "Fixed-seed GPT logits for pkg/hands/gpt (9.1). Generated by TestGPTGoldenLogits -update. " +
			"Tolerance: |got-want| <= abs_tol + rel_tol*|want|.",
		Seed:   goldenSeed,
		Tokens: goldenTokens,
		AbsTol: goldenAbsTol,
		RelTol: goldenRelTol,
		Config: *cfg,
		Shape:  []int{len(goldenTokens), cfg.VocabSize},
		Logits: logits,
	}
	buf, err := json.MarshalIndent(fx, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	buf = append(buf, '\n')
	// Only testdata/ is written; the path is fixed and relative to the package.
	if err := os.MkdirAll(filepath.Dir(goldenFixturePath), 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	if err := os.WriteFile(goldenFixturePath, buf, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
