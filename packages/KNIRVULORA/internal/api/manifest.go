package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	SchemaURL   = "https://ulora.org/schema/v1/manifest.json"
	SpecVersion = "1.0.0"
	DefaultTau  = 0.82
)

var ValidTransferPaths = map[string]bool{
	"subspace_svd":      true,
	"synthetic_distill": true,
}

type BaseModelSpec struct {
	Family             string `json:"family"`
	ParamCount         string `json:"param_count"`
	HiddenSize         int    `json:"hidden_size"`
	IntermediateSize   int    `json:"intermediate_size"`
	NumAttentionHeads  int    `json:"num_attention_heads"`
	NumKeyValueHeads   int    `json:"num_key_value_heads"`
	HeadDim            int    `json:"head_dim"`
	NumLayers          int    `json:"num_layers"`
	AttentionMechanism string `json:"attention_mechanism"`
	ActivationFunc     string `json:"activation_func"`
}

func (b *BaseModelSpec) Validate() error {
	if strings.TrimSpace(b.Family) == "" {
		return fmt.Errorf("base_source_model.family is required")
	}
	if b.HiddenSize <= 0 {
		return fmt.Errorf("base_source_model.hidden_size must be positive")
	}
	if b.IntermediateSize <= 0 {
		return fmt.Errorf("base_source_model.intermediate_size must be positive")
	}
	if b.NumAttentionHeads <= 0 {
		return fmt.Errorf("base_source_model.num_attention_heads must be positive")
	}
	if b.NumKeyValueHeads <= 0 {
		return fmt.Errorf("base_source_model.num_key_value_heads must be positive")
	}
	if b.HeadDim <= 0 {
		return fmt.Errorf("base_source_model.head_dim must be positive")
	}
	if b.NumLayers <= 0 {
		return fmt.Errorf("base_source_model.num_layers must be positive")
	}
	if b.AttentionMechanism == "" {
		return fmt.Errorf("base_source_model.attention_mechanism is required")
	}
	if b.ActivationFunc == "" {
		return fmt.Errorf("base_source_model.activation_func is required")
	}
	return nil
}

// CanonicalCore is the model-agnostic half of a uLoRA artifact: the skill or
// behaviour tuning, held in a shared latent space rather than in any one model's
// native coordinate frame.
//
// This is the shape change from the original single projected A/B pair. Standard
// LoRA couples ΔW directly to a base model's (d_in, d_out) and basis, so a raw
// delta pair cannot travel between architectures. Keeping the core in a fixed
// canonical space of dimension K, and pairing it with per-family connector
// matrices (see ModelConnector), makes the artifact portable: a consumer maps it
// into a concrete model's frame with A_tgt = A_canonical · P_in and
// B_tgt = P_out · B_canonical.
type CanonicalCore struct {
	AdapterRank        int     `json:"adapter_rank"`
	ScalingFactorAlpha float64 `json:"scaling_factor_alpha"`
	// CanonicalDim is K: the dimension of the shared, architecture-agnostic
	// space the core tensors live in. Every core tensor is shaped against it
	// (lora_A is r × K, lora_B is K × r), which is what lets one artifact bind
	// to models of differing hidden sizes.
	CanonicalDim int `json:"canonical_dim"`
	// TargetModules are semantic layer roles (e.g. "self_attn.q_proj"), not
	// indices: binding a role to a physical layer is the connector's job, since
	// layer counts and naming differ between architectures, and a bundle cannot
	// know which physical layer a role maps to until it is bound.
	TargetModules []string `json:"target_modules"`
}

func (c *CanonicalCore) Validate() error {
	if c.AdapterRank <= 0 {
		return fmt.Errorf("canonical_core.adapter_rank must be positive")
	}
	if c.ScalingFactorAlpha <= 0 {
		return fmt.Errorf("canonical_core.scaling_factor_alpha must be positive")
	}
	// Without K there is no canonical space, so the core tensors have no defined
	// shape and the artifact cannot be bound to any model. This is required
	// rather than defaulted: guessing K would silently reinterpret every tensor
	// in the file.
	if c.CanonicalDim <= 0 {
		return fmt.Errorf("canonical_core.canonical_dim must be positive")
	}
	if len(c.TargetModules) == 0 {
		return fmt.Errorf("canonical_core.target_modules cannot be empty")
	}
	for _, m := range c.TargetModules {
		if strings.TrimSpace(m) == "" {
			return fmt.Errorf("canonical_core.target_modules contains an empty entry")
		}
	}
	return nil
}

type RoutingPolicy struct {
	SimilarityThreshold     float64  `json:"similarity_threshold"`
	AllowedTransferPaths    []string `json:"allowed_transfer_paths"`
	DistillationAnchorSeeds []string `json:"distillation_anchor_seeds,omitempty"`
}

func (r *RoutingPolicy) Validate() error {
	if r.SimilarityThreshold < 0 || r.SimilarityThreshold > 1 {
		return fmt.Errorf("routing_policy.similarity_threshold must be between 0 and 1")
	}
	if len(r.AllowedTransferPaths) == 0 {
		return fmt.Errorf("routing_policy.allowed_transfer_paths cannot be empty")
	}
	for _, p := range r.AllowedTransferPaths {
		if !ValidTransferPaths[p] {
			return fmt.Errorf("routing_policy.allowed_transfer_paths contains invalid path: %s", p)
		}
	}
	if r.SimilarityThreshold == 0 {
		r.SimilarityThreshold = DefaultTau
	}
	return nil
}

type Provenance struct {
	SourceID         string         `json:"source_id"`
	SourceDatasetIDs []string       `json:"source_dataset_ids"`
	Extensions       map[string]any `json:"extensions,omitempty"`
}

func (p *Provenance) Validate() error {
	if strings.TrimSpace(p.SourceID) == "" {
		return fmt.Errorf("provenance.source_id is required")
	}
	if len(p.SourceDatasetIDs) == 0 {
		return fmt.Errorf("provenance.source_dataset_ids cannot be empty")
	}
	for _, id := range p.SourceDatasetIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("provenance.source_dataset_ids contains an empty entry")
		}
	}
	return nil
}

type Manifest struct {
	Schema          string        `json:"$schema"`
	Name            string        `json:"name"`
	Version         string        `json:"version"`
	Provenance      Provenance    `json:"provenance"`
	BaseSourceModel BaseModelSpec `json:"base_source_model"`
	CanonicalCore   CanonicalCore `json:"canonical_core"`
	RoutingPolicy   RoutingPolicy `json:"routing_policy"`
}

func (m *Manifest) Validate() error {
	if strings.TrimSpace(m.Schema) != SchemaURL {
		return fmt.Errorf("invalid or missing $schema: expected %s", SchemaURL)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("version is required")
	}
	if err := m.Provenance.Validate(); err != nil {
		return fmt.Errorf("provenance: %w", err)
	}
	if err := m.BaseSourceModel.Validate(); err != nil {
		return fmt.Errorf("base_source_model: %w", err)
	}
	if err := m.CanonicalCore.Validate(); err != nil {
		return fmt.Errorf("canonical_core: %w", err)
	}
	if err := m.RoutingPolicy.Validate(); err != nil {
		return fmt.Errorf("routing_policy: %w", err)
	}
	return nil
}

type DatasetRecord struct {
	Context             string `json:"context"`
	CorrectedCompletion string `json:"corrected_completion"`
	TargetModel         string `json:"target_model"`
	// TargetModule names the semantic layer role this correction applies to
	// (e.g. "self_attn.q_proj"), which is what lets the compiler fit a core per
	// module instead of one shared delta. Optional: a record without it cannot be
	// attributed to a layer, and the compiler refuses to spread a single shared
	// core across every module unless the caller explicitly opts in.
	TargetModule string `json:"target_module,omitempty"`
}

type CompileRequest struct {
	Provenance   CompileProvenance `json:"provenance"`
	Dataset      []DatasetRecord   `json:"dataset"`
	TargetModels []BaseModelSpec   `json:"target_models"`
	Alpha        float64           `json:"alpha,omitempty"`
	Rank         int               `json:"rank,omitempty"`
	NumEpochs    int               `json:"num_epochs,omitempty"`
	LearningRate float64           `json:"learning_rate,omitempty"`
}

type CompileProvenance struct {
	SourceID         string         `json:"source_id"`
	SourceDatasetIDs []string       `json:"source_dataset_ids"`
	Extensions       map[string]any `json:"extensions,omitempty"`
}

type CompileResponse struct {
	BundleID     string   `json:"bundle_id"`
	ContentHash  string   `json:"content_hash"`
	ManifestPath string   `json:"manifest_path"`
	WeightsPath  string   `json:"weights_path"`
	AnchorsPath  string   `json:"anchors_path,omitempty"`
	PathUsed     string   `json:"path_used"`
	TargetModels []string `json:"target_models"`
}

type TransferRequest struct {
	SourceBundlePath string          `json:"source_bundle_path"`
	TargetModels     []BaseModelSpec `json:"target_models"`
}

type TransferResponse struct {
	BundleID     string   `json:"bundle_id"`
	ContentHash  string   `json:"content_hash"`
	ManifestPath string   `json:"manifest_path"`
	WeightsPath  string   `json:"weights_path"`
	PathUsed     string   `json:"path_used"`
	TargetModels []string `json:"target_models"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
}

var Version = "0.1.0"

func HealthResponseNow() HealthResponse {
	return HealthResponse{
		Status:    "ok",
		Version:   Version,
		Timestamp: time.Now().Unix(),
	}
}

func ManifestFromJSON(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}
