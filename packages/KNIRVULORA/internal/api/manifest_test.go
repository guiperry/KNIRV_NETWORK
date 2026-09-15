package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func baseModelSpec(family string) BaseModelSpec {
	return BaseModelSpec{
		Family:             family,
		ParamCount:         "7B",
		HiddenSize:         4096,
		IntermediateSize:   11008,
		NumAttentionHeads:  32,
		NumKeyValueHeads:   32,
		HeadDim:            128,
		NumLayers:          32,
		AttentionMechanism: "gqa",
		ActivationFunc:     "silu",
	}
}

func validManifest() Manifest {
	return Manifest{
		Schema:  SchemaURL,
		Name:    "test-ulora",
		Version: "1.0.0",
		Provenance: Provenance{
			SourceID:         "src-001",
			SourceDatasetIDs: []string{"ds-001", "ds-002"},
		},
		BaseSourceModel: baseModelSpec("llama2"),
		CanonicalCore: CanonicalCore{
			AdapterRank:        16,
			ScalingFactorAlpha: 32.0,
			CanonicalDim:       1024, TargetModules: []string{"q_proj", "k_proj", "v_proj"},
		},
		RoutingPolicy: RoutingPolicy{
			SimilarityThreshold:  0.82,
			AllowedTransferPaths: []string{"subspace_svd", "synthetic_distill"},
		},
	}
}

func TestManifestValidateValid(t *testing.T) {
	m := validManifest()
	if err := m.Validate(); err != nil {
		t.Fatalf("expected valid manifest, got: %v", err)
	}
}

func TestManifestValidateInvalidSchema(t *testing.T) {
	m := validManifest()
	m.Schema = "https://example.com/wrong"
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for invalid schema")
	}
	if !strings.Contains(err.Error(), "invalid or missing") {
		t.Fatalf("expected schema error, got: %v", err)
	}
}

func TestManifestValidateEmptyName(t *testing.T) {
	m := validManifest()
	m.Name = ""
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestManifestValidateEmptyVersion(t *testing.T) {
	m := validManifest()
	m.Version = ""
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestManifestValidateBaseModelMissingFamily(t *testing.T) {
	m := validManifest()
	m.BaseSourceModel.Family = ""
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for missing family")
	}
}

func TestManifestValidateBaseModelInvalidHiddenSize(t *testing.T) {
	m := validManifest()
	m.BaseSourceModel.HiddenSize = 0
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for zero hidden_size")
	}
}

func TestManifestValidateCanonicalCoreInvalidRank(t *testing.T) {
	m := validManifest()
	m.CanonicalCore.AdapterRank = 0
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for zero adapter_rank")
	}
}

func TestManifestValidateRoutingPolicyThresholdOutOfRange(t *testing.T) {
	m := validManifest()
	m.RoutingPolicy.SimilarityThreshold = 1.5
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for threshold > 1")
	}
}

func TestManifestValidateRoutingPolicyZeroThresholdDefaults(t *testing.T) {
	m := validManifest()
	m.RoutingPolicy.SimilarityThreshold = 0
	if err := m.RoutingPolicy.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.RoutingPolicy.SimilarityThreshold != DefaultTau {
		t.Fatalf("expected default tau %v, got %v", DefaultTau, m.RoutingPolicy.SimilarityThreshold)
	}
}

func TestManifestValidateRoutingPolicyInvalidPath(t *testing.T) {
	m := validManifest()
	m.RoutingPolicy.AllowedTransferPaths = []string{"invalid_path"}
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for invalid transfer path")
	}
}

func TestManifestValidateProvenanceMissingSourceID(t *testing.T) {
	m := validManifest()
	m.Provenance.SourceID = ""
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for missing source_id")
	}
}

func TestManifestValidateProvenanceEmptyDatasetIDs(t *testing.T) {
	m := validManifest()
	m.Provenance.SourceDatasetIDs = nil
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for empty source_dataset_ids")
	}
}

func TestManifestValidateProvenanceEmptyDatasetIDEntry(t *testing.T) {
	m := validManifest()
	m.Provenance.SourceDatasetIDs = []string{""}
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for empty dataset ID entry")
	}
}

func TestManifestValidateCanonicalCoreEmptyTargetModules(t *testing.T) {
	m := validManifest()
	m.CanonicalCore.TargetModules = nil
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for empty target_modules")
	}
}

func TestManifestValidateCanonicalCoreEmptyModuleEntry(t *testing.T) {
	m := validManifest()
	m.CanonicalCore.TargetModules = []string{""}
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for empty module entry")
	}
}

func TestManifestFromJSONValid(t *testing.T) {
	m := validManifest()
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	result, err := ManifestFromJSON(data)
	if err != nil {
		t.Fatalf("expected valid manifest, got: %v", err)
	}
	if result.Name != m.Name {
		t.Fatalf("expected name %s, got %s", m.Name, result.Name)
	}
}

func TestManifestFromJSONInvalidJSON(t *testing.T) {
	_, err := ManifestFromJSON([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestManifestFromJSONInvalidManifest(t *testing.T) {
	m := validManifest()
	m.Name = ""
	data, _ := json.Marshal(m)
	_, err := ManifestFromJSON(data)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestHealthResponseNow(t *testing.T) {
	resp := HealthResponseNow()
	if resp.Status != "ok" {
		t.Fatalf("expected status 'ok', got %s", resp.Status)
	}
	if resp.Version != Version {
		t.Fatalf("expected version %s, got %s", Version, resp.Version)
	}
	if resp.Timestamp == 0 {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestValidTransferPaths(t *testing.T) {
	if !ValidTransferPaths["subspace_svd"] {
		t.Fatal("expected subspace_svd to be valid")
	}
	if !ValidTransferPaths["synthetic_distill"] {
		t.Fatal("expected synthetic_distill to be valid")
	}
	if ValidTransferPaths["invalid"] {
		t.Fatal("expected invalid to be false")
	}
}

func TestBaseModelSpecValidateAllFields(t *testing.T) {
	tests := []struct {
		name     string
		setField func(*BaseModelSpec)
	}{
		{"empty_family", func(b *BaseModelSpec) { b.Family = "" }},
		{"zero_hidden_size", func(b *BaseModelSpec) { b.HiddenSize = 0 }},
		{"zero_intermediate_size", func(b *BaseModelSpec) { b.IntermediateSize = 0 }},
		{"zero_num_attention_heads", func(b *BaseModelSpec) { b.NumAttentionHeads = 0 }},
		{"zero_num_key_value_heads", func(b *BaseModelSpec) { b.NumKeyValueHeads = 0 }},
		{"zero_head_dim", func(b *BaseModelSpec) { b.HeadDim = 0 }},
		{"zero_num_layers", func(b *BaseModelSpec) { b.NumLayers = 0 }},
		{"empty_attention_mechanism", func(b *BaseModelSpec) { b.AttentionMechanism = "" }},
		{"empty_activation_func", func(b *BaseModelSpec) { b.ActivationFunc = "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := baseModelSpec("test")
			tt.setField(&b)
			if err := b.Validate(); err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

func TestCanonicalCoreValidateAllFields(t *testing.T) {
	tests := []struct {
		name     string
		setField func(*CanonicalCore)
	}{
		{"zero_adapter_rank", func(c *CanonicalCore) { c.AdapterRank = 0 }},
		{"zero_scaling_factor", func(c *CanonicalCore) { c.ScalingFactorAlpha = 0 }},
		{"empty_target_modules", func(c *CanonicalCore) { c.TargetModules = nil }},
		{"empty_module_entry", func(c *CanonicalCore) { c.TargetModules = []string{""} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CanonicalCore{
				AdapterRank:        16,
				ScalingFactorAlpha: 32.0,
				CanonicalDim:       1024, TargetModules: []string{"q_proj", "v_proj"},
			}
			tt.setField(&c)
			if err := c.Validate(); err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}
