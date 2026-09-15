package runtime

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ulora/internal/api"
	"ulora/internal/bundling"
	"ulora/internal/safetensors"
)

var errTestLookup = errors.New("connector cache unavailable")

const (
	testK      = 4 // canonical_dim, kept tiny so expected values are checkable by hand
	testRank   = 2
	testHidden = 3
)

func targetSpec() api.BaseModelSpec {
	return api.BaseModelSpec{
		Family:             "llama",
		ParamCount:         "8b",
		HiddenSize:         testHidden,
		IntermediateSize:   8,
		NumAttentionHeads:  2,
		NumKeyValueHeads:   2,
		HeadDim:            2,
		NumLayers:          1,
		AttentionMechanism: "gqa",
		ActivationFunc:     "silu",
	}
}

// testProvider stands in for the runtime's connector cache. Connectors are a
// property of the model, so the binder no longer reads them from the bundle —
// tests must supply them the way the daemon does.
type testProvider struct {
	pIn, pOut [][]float32
	err       error
}

func (p testProvider) ConnectorFor(api.BaseModelSpec, int) ([][]float32, [][]float32, error) {
	return p.pIn, p.pOut, p.err
}

// writeCanonicalBundle assembles a bundle carrying only a canonical core. It
// deliberately contains no connector tensors: those live in the runtime now, and
// a bundle that embedded them would be advertising a projection for exactly one
// model.
func writeCanonicalBundle(t *testing.T, modules []string, core map[string][][]float32) string {
	t.Helper()

	weights := map[string][][]float32{}
	for _, m := range modules {
		weights[m+"/lora_A"] = core[m+"/lora_A"]
		weights[m+"/lora_B"] = core[m+"/lora_B"]
	}

	weightsData, err := safetensors.Write(weights, nil)
	if err != nil {
		t.Fatalf("encode weights: %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, bundling.WeightsFile), weightsData, 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}

	manifest := &api.Manifest{
		Schema: api.SchemaURL, Name: "t", Version: "1.0.0",
		Provenance:      api.Provenance{SourceID: "c1", SourceDatasetIDs: []string{"s1"}},
		BaseSourceModel: targetSpec(),
		CanonicalCore: api.CanonicalCore{
			AdapterRank: testRank, ScalingFactorAlpha: 32,
			CanonicalDim: testK, TargetModules: modules,
		},
		RoutingPolicy: api.RoutingPolicy{SimilarityThreshold: 0.82, AllowedTransferPaths: []string{"subspace_svd"}},
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, bundling.ManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	bundlePath := filepath.Join(t.TempDir(), "bundle.ulora")
	if err := bundling.WriteBundle(dir, bundlePath); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	return bundlePath
}

// identityConnector returns P_in (K × d_in) and P_out (d_out × K) that select the
// first few canonical coordinates, so a projection is an observable selection
// rather than a blur.
func identityConnector() ([][]float32, [][]float32) {
	pIn := make([][]float32, testK)
	for i := range pIn {
		pIn[i] = make([]float32, testHidden)
	}
	for i := 0; i < testHidden && i < testK; i++ {
		pIn[i][i] = 1
	}
	pOut := make([][]float32, testHidden)
	for i := range pOut {
		pOut[i] = make([]float32, testK)
	}
	for i := 0; i < testHidden && i < testK; i++ {
		pOut[i][i] = 1
	}
	return pIn, pOut
}

func canonicalA() [][]float32 {
	out := make([][]float32, testRank)
	for i := range out {
		out[i] = make([]float32, testK)
		for j := range out[i] {
			out[i][j] = float32(i+1) + float32(j)*0.5
		}
	}
	return out
}

func canonicalB() [][]float32 {
	out := make([][]float32, testK)
	for i := range out {
		out[i] = make([]float32, testRank)
		for j := range out[i] {
			out[i][j] = float32(i) * 0.25
		}
	}
	return out
}

func TestBindProjectsCanonicalCoreThroughProvider(t *testing.T) {
	module := "self_attn.q_proj"
	coreA, coreB := canonicalA(), canonicalB()
	bundle := writeCanonicalBundle(t, []string{module},
		map[string][][]float32{module + "/lora_A": coreA, module + "/lora_B": coreB})

	pIn, pOut := identityConnector()
	binder, err := NewBinder(WithBundlePath(bundle), WithConnectorProvider(testProvider{pIn: pIn, pOut: pOut}))
	if err != nil {
		t.Fatalf("open bundle: %v", err)
	}
	layers, err := binder.Bind(targetSpec())
	if err != nil {
		t.Fatalf("bind: %v", err)
	}

	layer, ok := layers[module]
	if !ok {
		t.Fatalf("no layer for %s", module)
	}
	if len(layer.A) != testRank || len(layer.A[0]) != testHidden {
		t.Fatalf("A_target is %dx%d, want %dx%d", len(layer.A), len(layer.A[0]), testRank, testHidden)
	}
	// With an identity projection the first d_in canonical columns survive
	// exactly, so a wrong multiply is visible rather than just shape-correct.
	for i := 0; i < testRank; i++ {
		for j := 0; j < testHidden; j++ {
			if layer.A[i][j] != coreA[i][j] {
				t.Fatalf("A_target[%d][%d] = %v, want %v", i, j, layer.A[i][j], coreA[i][j])
			}
		}
	}
	if len(layer.B) != testHidden || len(layer.B[0]) != testRank {
		t.Fatalf("B_target is %dx%d, want %dx%d", len(layer.B), len(layer.B[0]), testHidden, testRank)
	}
}

// A bundle carries no connectors, so without a provider there is nothing to
// project the core onto and binding must be refused rather than approximated.
func TestBindRefusesWithoutConnectorProvider(t *testing.T) {
	module := "self_attn.q_proj"
	bundle := writeCanonicalBundle(t, []string{module},
		map[string][][]float32{module + "/lora_A": canonicalA(), module + "/lora_B": canonicalB()})

	binder, err := NewBinder(WithBundlePath(bundle))
	if err != nil {
		t.Fatalf("open bundle: %v", err)
	}
	_, err = binder.Bind(targetSpec())
	if err == nil {
		t.Fatal("binding with no connector provider must fail")
	}
	if !strings.Contains(err.Error(), "connector provider") {
		t.Fatalf("error should name the missing provider, got: %v", err)
	}
}

// A model the runtime has not derived a connector for yet is distinct from a
// lookup failure: the message must say what is needed, and it must not invent a
// projection (an invented basis binds without aligning anything).
func TestBindRefusesWhenProviderHasNoConnector(t *testing.T) {
	module := "self_attn.q_proj"
	bundle := writeCanonicalBundle(t, []string{module},
		map[string][][]float32{module + "/lora_A": canonicalA(), module + "/lora_B": canonicalB()})

	binder, err := NewBinder(WithBundlePath(bundle), WithConnectorProvider(testProvider{}))
	if err != nil {
		t.Fatalf("open bundle: %v", err)
	}
	_, err = binder.Bind(targetSpec())
	if err == nil {
		t.Fatal("binding without an available connector must fail")
	}
	if !strings.Contains(err.Error(), "no connector available") {
		t.Fatalf("error should say no connector is available, got: %v", err)
	}
	if !strings.Contains(err.Error(), "weight matrices") {
		t.Fatalf("error should name what deriving one requires, got: %v", err)
	}
}

// A lookup failure is not the same as an absence and must propagate.
func TestBindPropagatesProviderError(t *testing.T) {
	module := "self_attn.q_proj"
	bundle := writeCanonicalBundle(t, []string{module},
		map[string][][]float32{module + "/lora_A": canonicalA(), module + "/lora_B": canonicalB()})

	binder, err := NewBinder(WithBundlePath(bundle), WithConnectorProvider(testProvider{err: errTestLookup}))
	if err != nil {
		t.Fatalf("open bundle: %v", err)
	}
	_, err = binder.Bind(targetSpec())
	if err == nil {
		t.Fatal("a provider error must surface")
	}
	if !strings.Contains(err.Error(), "resolve connector") {
		t.Fatalf("error should say the lookup failed, got: %v", err)
	}
}

func TestBindRefusesMismatchedConnectorShape(t *testing.T) {
	module := "self_attn.q_proj"
	bundle := writeCanonicalBundle(t, []string{module},
		map[string][][]float32{module + "/lora_A": canonicalA(), module + "/lora_B": canonicalB()})

	// P_in should be K × d_in; give it K × (d_in+1).
	badPIn := make([][]float32, testK)
	for i := range badPIn {
		badPIn[i] = make([]float32, testHidden+1)
	}
	_, pOut := identityConnector()

	binder, err := NewBinder(WithBundlePath(bundle), WithConnectorProvider(testProvider{pIn: badPIn, pOut: pOut}))
	if err != nil {
		t.Fatalf("open bundle: %v", err)
	}
	_, err = binder.Bind(targetSpec())
	if err == nil {
		t.Fatal("a connector of the wrong shape must be refused")
	}
	if !strings.Contains(err.Error(), "P_in") {
		t.Fatalf("error should name the offending matrix, got: %v", err)
	}
}

func TestMatMulRejectsDimensionMismatch(t *testing.T) {
	if _, err := matMul([][]float32{{1, 2, 3}}, [][]float32{{1, 2}}, "test"); err == nil {
		t.Fatal("inner-dimension mismatch must be refused")
	}
}

func TestMatMulRejectsRaggedOperands(t *testing.T) {
	if _, err := matMul([][]float32{{1, 2}, {3}}, [][]float32{{1, 2}, {3, 4}}, "test"); err == nil {
		t.Fatal("ragged operands must be refused, not read out of bounds")
	}
}

func TestMatMulComputesCorrectly(t *testing.T) {
	out, err := matMul([][]float32{{2, 3}}, [][]float32{{5}, {7}}, "test")
	if err != nil {
		t.Fatalf("matMul: %v", err)
	}
	if len(out) != 1 || len(out[0]) != 1 || out[0][0] != 31 {
		t.Fatalf("out = %v, want [[31]]", out)
	}
}
