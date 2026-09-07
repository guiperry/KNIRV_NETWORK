package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ulora/internal/api"
	"ulora/internal/bundling"
)

func validManifestForTest() api.Manifest {
	return api.Manifest{
		Schema:  api.SchemaURL,
		Name:    "test-bundle",
		Version: "1.0.0",
		Provenance: api.Provenance{
			SourceID:         "src-001",
			SourceDatasetIDs: []string{"ds-001"},
		},
		BaseSourceModel: api.BaseModelSpec{
			Family:             "llama2",
			ParamCount:         "7B",
			HiddenSize:         4096,
			IntermediateSize:   11008,
			NumAttentionHeads:  32,
			NumKeyValueHeads:   32,
			HeadDim:            128,
			NumLayers:          32,
			AttentionMechanism: "gqa",
			ActivationFunc:     "silu",
		},
		CanonicalCore: api.CanonicalCore{
			AdapterRank:        16,
			ScalingFactorAlpha: 32.0,
			TargetModules:      []string{"q_proj", "v_proj"},
		},
		RoutingPolicy: api.RoutingPolicy{
			SimilarityThreshold:  0.82,
			AllowedTransferPaths: []string{"subspace_svd", "synthetic_distill"},
		},
	}
}

func createTestBundle(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "bundle_contents")
	if err := bundling.EnsureDir(bundleDir); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}

	m := validManifestForTest()
	manifestJSON, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, bundling.ManifestFile), manifestJSON, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	weightsData := []byte("fake weights binary data")
	if err := os.WriteFile(filepath.Join(bundleDir, bundling.WeightsFile), weightsData, 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}

	bundlePath := filepath.Join(tmpDir, "test.ulora")
	if err := bundling.WriteBundle(bundleDir, bundlePath); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	return bundlePath
}

func TestNewBinderValid(t *testing.T) {
	bundlePath := createTestBundle(t)

	b, err := NewBinder(WithBundlePath(bundlePath))
	if err != nil {
		t.Fatalf("new binder: %v", err)
	}
	if b == nil {
		t.Fatal("expected non-nil binder")
	}
	if b.manifest == nil {
		t.Fatal("expected manifest to be set")
	}
}

func TestNewBinderNoBundlePath(t *testing.T) {
	_, err := NewBinder()
	if err == nil {
		t.Fatal("expected error for missing bundle path")
	}
}

func TestNewBinderInvalidBundle(t *testing.T) {
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "nonexistent.ulora")
	_, err := NewBinder(WithBundlePath(invalidPath))
	if err == nil {
		t.Fatal("expected error for nonexistent bundle")
	}
}

func TestNewBinderInvalidManifest(t *testing.T) {
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "bundle_contents")
	if err := bundling.EnsureDir(bundleDir); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(bundleDir, bundling.ManifestFile), []byte(`{"invalid":true}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, bundling.WeightsFile), []byte("weights"), 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}

	bundlePath := filepath.Join(tmpDir, "bad.ulora")
	if err := bundling.WriteBundle(bundleDir, bundlePath); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	_, err := NewBinder(WithBundlePath(bundlePath))
	if err == nil {
		t.Fatal("expected error for invalid manifest in bundle")
	}
}

func TestBinderManifest(t *testing.T) {
	bundlePath := createTestBundle(t)

	b, err := NewBinder(WithBundlePath(bundlePath))
	if err != nil {
		t.Fatalf("new binder: %v", err)
	}

	m := b.Manifest()
	if m == nil {
		t.Fatal("expected manifest to be non-nil")
	}
	if m.Name != "test-bundle" {
		t.Fatalf("expected name 'test-bundle', got %s", m.Name)
	}
	if m.BaseSourceModel.Family != "llama2" {
		t.Fatalf("expected family 'llama2', got %s", m.BaseSourceModel.Family)
	}
}

func TestBinderValidateTopologyMatch(t *testing.T) {
	bundlePath := createTestBundle(t)
	b, err := NewBinder(WithBundlePath(bundlePath))
	if err != nil {
		t.Fatalf("new binder: %v", err)
	}

	target := b.manifest.BaseSourceModel
	if err := b.validateTopology(target); err != nil {
		t.Fatalf("expected no error for matching topology, got: %v", err)
	}
}

func TestBinderValidateTopologyMismatch(t *testing.T) {
	bundlePath := createTestBundle(t)
	b, err := NewBinder(WithBundlePath(bundlePath))
	if err != nil {
		t.Fatalf("new binder: %v", err)
	}

	target := b.manifest.BaseSourceModel
	target.AttentionMechanism = "mha"
	if err := b.validateTopology(target); err == nil {
		t.Fatal("expected error for mismatched topology")
	}

	target = b.manifest.BaseSourceModel
	target.ActivationFunc = "gelu"
	if err := b.validateTopology(target); err == nil {
		t.Fatal("expected error for mismatched topology")
	}
}

func TestBinderBindMismatchTopology(t *testing.T) {
	bundlePath := createTestBundle(t)
	b, err := NewBinder(WithBundlePath(bundlePath))
	if err != nil {
		t.Fatalf("new binder: %v", err)
	}

	target := b.manifest.BaseSourceModel
	target.AttentionMechanism = "mha"

	_, err = b.Bind(target)
	if err == nil {
		t.Fatal("expected error for mismatched topology in Bind")
	}
}

func TestDeterminePrefix(t *testing.T) {
	target := api.BaseModelSpec{
		Family:     "Llama-2",
		ParamCount: "7B",
	}
	prefix := determinePrefix("q_proj", target)
	expected := "Llama-2-7B"
	if prefix != expected {
		t.Fatalf("expected prefix '%s', got '%s'", expected, prefix)
	}
}

func TestExtractTensor(t *testing.T) {
	tensors := map[string]interface{}{
		"prefix/lora_A": [][]float32{{1.0, 2.0}, {3.0, 4.0}},
		"prefix/lora_B": [][]float32{{5.0}, {6.0}},
	}

	a, b := extractTensor(tensors, "prefix/lora_A", "prefix/lora_B")
	if len(a) != 2 {
		t.Fatalf("expected 2 rows in lora_A, got %d", len(a))
	}
	if a[0][0] != 1.0 {
		t.Fatalf("expected a[0][0]=1.0, got %f", a[0][0])
	}
	if len(b) != 2 {
		t.Fatalf("expected 2 rows in lora_B, got %d", len(b))
	}

	a, b = extractTensor(tensors, "nonexistent_A", "nonexistent_B")
	if len(a) != 0 {
		t.Fatal("expected nil for nonexistent A")
	}
	if len(b) != 0 {
		t.Fatal("expected nil for nonexistent B")
	}
}

func TestProjectMatrixIdentity(t *testing.T) {
	mat := [][]float32{
		{1.0, 2.0, 3.0},
		{4.0, 5.0, 6.0},
	}
	result := projectMatrix(mat, 2, "input")
	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
}

func TestProjectMatrixExpand(t *testing.T) {
	mat := [][]float32{{1.0, 2.0}}
	result := projectMatrix(mat, 4, "output")
	if len(result) != 4 {
		t.Fatalf("expected 4 rows after expansion, got %d", len(result))
	}
	if len(result[0]) != 2 {
		t.Fatalf("expected 2 cols, got %d", len(result[0]))
	}
	for i := 1; i < 4; i++ {
		if len(result[i]) != 2 {
			t.Fatalf("expected 2 cols for padding row %d, got %d", i, len(result[i]))
		}
	}
}

func TestProjectMatrixShrink(t *testing.T) {
	mat := [][]float32{
		{1.0, 2.0},
		{3.0, 4.0},
		{5.0, 6.0},
		{7.0, 8.0},
	}
	result := projectMatrix(mat, 2, "input")
	if len(result) != 2 {
		t.Fatalf("expected 2 rows after shrink, got %d", len(result))
	}
}

func TestProjectMatrixEmpty(t *testing.T) {
	result := projectMatrix([][]float32{}, 4, "input")
	if len(result) != 0 {
		t.Fatalf("expected 0 rows for empty matrix, got %d", len(result))
	}
}

func TestBindResultToJSON(t *testing.T) {
	br := &BindResult{
		TargetModel: api.BaseModelSpec{
			Family:            "llama2",
			ParamCount:        "7B",
			HiddenSize:        4096,
			NumLayers:         32,
			AttentionMechanism: "gqa",
			ActivationFunc:     "silu",
		},
		LayerWeights: map[string]LayerWeight{
			"q_proj": {
				A: [][]float32{{1.0, 2.0}},
				B: [][]float32{{3.0}, {4.0}},
			},
		},
	}

	data, err := br.ToJSON()
	if err != nil {
		t.Fatalf("toJSON: %v", err)
	}

	var decoded struct {
		TargetModel struct {
			Family string `json:"family"`
		} `json:"target_model"`
		LayerWeights map[string]LayerWeight `json:"layer_weights"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.TargetModel.Family != "llama2" {
		t.Fatalf("expected family 'llama2', got %s", decoded.TargetModel.Family)
	}
	if len(decoded.LayerWeights) != 1 {
		t.Fatalf("expected 1 layer weight, got %d", len(decoded.LayerWeights))
	}
}

func TestBinderBindBytes(t *testing.T) {
	bundlePath := createTestBundle(t)
	b, err := NewBinder(WithBundlePath(bundlePath))
	if err != nil {
		t.Fatalf("new binder: %v", err)
	}

	_, err = b.BindBytes(b.manifest.BaseSourceModel)
	if err == nil {
		t.Fatal("expected error from BindBytes")
	}
	if err.Error() != "BindBytes not implemented; use Bind instead" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestLayerWeightJSON(t *testing.T) {
	lw := LayerWeight{
		A: [][]float32{{1.0, 2.0}, {3.0, 4.0}},
		B: [][]float32{{5.0, 6.0}, {7.0, 8.0}},
	}
	data, err := json.Marshal(lw)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var result LayerWeight
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.A) != 2 || len(result.B) != 2 {
		t.Fatal("expected 2x2 matrices")
	}
}
