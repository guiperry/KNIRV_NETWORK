package drq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// uLoRA minting: the bridge from a resolved cluster to a portable adapter
// bundle.
//
// Per ulora_implementation.md §2.2, this sits between discovering the skill
// (step 2) and the canonical chain mint (step 5):
//
//	2.5  gatherClusterDatasets   — each member error's context + its validated fix
//	2.6  CompileCluster          — KNIRVULORA turns the corpus into a `.ulora`
//	2.6a ValidateBundle          — the DVE/cognitive-engine gate (see below)
//	2.7  MintULoRABundle         — KNIRVCHAIN stores the bytes and registers the pointer
//
// KNIRVULORA is a separate, deliberately KNIRV-free module (§0.2), so this talks
// to it over its Unix socket rather than importing it — the same pattern the
// KNIRVCHAIN and KNIRVORACLE clients use. The DTOs below mirror
// packages/KNIRVULORA/internal/api rather than being shared.

const (
	// uloraDefaultSocketPath matches KNIRVULORA's own default (internal/config).
	uloraDefaultSocketPath = "./ulora.sock"
	// uloraCompileRoute is the daemon's cluster-compile endpoint.
	uloraCompileRoute = "/ulora/v1/compile-cluster"
	// uloraCompileTimeout bounds a compile. Adapter compilation runs SFT fitting
	// over the corpus, so it is far slower than an ordinary RPC.
	uloraCompileTimeout = 30 * time.Minute
)

// ULoRAErrorDatasetUnavailable and friends are the failure modes a caller needs
// to distinguish: "not wired yet" versus "the service said no".
var (
	// ErrDVEValidationUnavailable means no bundle-validation backend is
	// configured. The mint must not proceed: §2.2 is explicit that a bundle may
	// not be published on the strength of a placeholder validator.
	ErrDVEValidationUnavailable = errors.New("no bundle validation backend configured")
	// ErrULoRACompilerUnavailable means no KNIRVULORA client is configured.
	ErrULoRACompilerUnavailable = errors.New("no uLoRA compiler configured")
)

// ULoRADatasetRecord is one training example: the error's context as input and
// the validated fix as the target. Mirrors ulora's DatasetRecord.
type ULoRADatasetRecord struct {
	Context             string `json:"context"`
	CorrectedCompletion string `json:"corrected_completion"`
	TargetModel         string `json:"target_model"`
	// TargetModule names the semantic layer role this correction applies to, so
	// the compiler can fit a core per module rather than one shared delta. Left
	// empty when the corpus cannot say which layer a fix belongs to — the
	// compiler then refuses to spread a shared core across every module unless
	// shared-core use is explicitly allowed.
	TargetModule string `json:"target_module,omitempty"`
}

// ULoRATargetModel mirrors ulora's BaseModelSpec: the architecture the adapter
// is being projected onto, which compilation needs in full.
//
// AttentionMechanism and ActivationFunc are required by BaseModelSpec.Validate,
// which ManifestFromJSON runs when a bundle is opened. Without them the compiled
// manifest is rejected at bind time — so a compile can appear to succeed and the
// resulting bundle still be unusable. There is deliberately no default: a
// guessed attention mechanism projects the adapter onto the wrong topology,
// which is worse than refusing.
type ULoRATargetModel struct {
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

// validate rejects a spec that would compile into an unbindable manifest.
func (m ULoRATargetModel) validate() error {
	switch {
	case strings.TrimSpace(m.Family) == "":
		return errors.New("family is required")
	case strings.TrimSpace(m.AttentionMechanism) == "":
		return errors.New("attention_mechanism is required (ulora validates it; a bundle without it cannot be bound)")
	case strings.TrimSpace(m.ActivationFunc) == "":
		return errors.New("activation_func is required (ulora validates it; a bundle without it cannot be bound)")
	case m.HiddenSize <= 0:
		return fmt.Errorf("hidden_size must be positive, got %d", m.HiddenSize)
	case m.IntermediateSize <= 0:
		return fmt.Errorf("intermediate_size must be positive, got %d", m.IntermediateSize)
	case m.NumLayers <= 0:
		return fmt.Errorf("num_layers must be positive, got %d", m.NumLayers)
	}
	return nil
}

type ULoRAProvenance struct {
	SourceID         string         `json:"source_id"`
	SourceDatasetIDs []string       `json:"source_dataset_ids"`
	Extensions       map[string]any `json:"extensions,omitempty"`
}

type uloraCompileRequest struct {
	Provenance   ULoRAProvenance      `json:"provenance"`
	Dataset      []ULoRADatasetRecord `json:"dataset"`
	TargetModels []ULoRATargetModel   `json:"target_models"`
	Alpha        float64              `json:"alpha,omitempty"`
	Rank         int                  `json:"rank,omitempty"`
	NumEpochs    int                  `json:"num_epochs,omitempty"`
	LearningRate float64              `json:"learning_rate,omitempty"`
}

// ULoRACompileResult is what the compiler reports. Paths are on the shared
// app-data filesystem that KNIRVSERVER gives ulorad and KNIRVGRAPH, which is how
// the bundle bytes reach the chain's mint route (see ULoRABundleBytes).
type ULoRACompileResult struct {
	BundleID     string   `json:"bundle_id"`
	ContentHash  string   `json:"content_hash"`
	ManifestPath string   `json:"manifest_path"`
	WeightsPath  string   `json:"weights_path"`
	AnchorsPath  string   `json:"anchors_path,omitempty"`
	PathUsed     string   `json:"path_used"`
	TargetModels []string `json:"target_models"`
}

// ULoRACompiler is the port the minting protocol compiles through.
type ULoRACompiler interface {
	CompileCluster(ctx context.Context, clusterID string, dataset []ULoRADatasetRecord, targetModels []ULoRATargetModel, provenance ULoRAProvenance) (*ULoRACompileResult, error)
}

// BundleValidator is the step 2.6a gate. Implementations decide whether a
// compiled bundle may be published.
//
// The real implementation runs the bundle through KNIRV's DVE inference engine
// against held-out cases and reports an objects.ValidationResult, with the
// cognitive engine's verdict deciding. Until that exists, callers get
// ErrDVEValidationUnavailable and the mint stops — deliberately, because §2.2
// forbids publishing a bundle on a placeholder validator.
type BundleValidator interface {
	ValidateBundle(ctx context.Context, cluster *ErrorCluster, result *ULoRACompileResult, dataset []ULoRADatasetRecord) error
}

// SkillDocSource resolves the skill.md document a member error resolved into.
// This is the CorrectedCompletion half of each training record.
type SkillDocSource interface {
	SkillDocForError(errorID string) (string, error)
}

// KNIRVULORAClient talks to the ulorad daemon over its Unix socket.
type KNIRVULORAClient struct {
	socketPath string
	http       *http.Client
}

// NewKNIRVULORAClient builds a compile client. An empty socketPath falls back to
// ULORA_SOCKET_PATH, then to the daemon's own default.
func NewKNIRVULORAClient(socketPath string) *KNIRVULORAClient {
	path := strings.TrimSpace(socketPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("ULORA_SOCKET_PATH"))
	}
	if path == "" {
		path = uloraDefaultSocketPath
	}
	return &KNIRVULORAClient{
		socketPath: path,
		http: &http.Client{
			Timeout: uloraCompileTimeout,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", path)
				},
			},
		},
	}
}

// CompileCluster submits a corpus to KNIRVULORA and returns the bundle it built.
func (c *KNIRVULORAClient) CompileCluster(ctx context.Context, clusterID string, dataset []ULoRADatasetRecord, targetModels []ULoRATargetModel, provenance ULoRAProvenance) (*ULoRACompileResult, error) {
	if c == nil {
		return nil, ErrULoRACompilerUnavailable
	}
	if len(dataset) == 0 {
		return nil, errors.New("uLoRA compile requires at least one dataset record")
	}
	if len(targetModels) == 0 {
		return nil, errors.New("uLoRA compile requires at least one target model")
	}
	if strings.TrimSpace(provenance.SourceID) == "" {
		provenance.SourceID = clusterID
	}

	body, err := json.Marshal(uloraCompileRequest{
		Provenance:   provenance,
		Dataset:      dataset,
		TargetModels: targetModels,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal uLoRA compile request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://ulora"+uloraCompileRoute, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build uLoRA compile request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("contact KNIRVULORA at %s: %w", c.socketPath, err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read uLoRA compile response: %w", err)
	}
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uLoRA compile failed (HTTP %d): %s", response.StatusCode, strings.TrimSpace(string(payload)))
	}

	var result ULoRACompileResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode uLoRA compile response: %w", err)
	}
	// A compile that reports no output paths would send the caller looking for
	// bundle files that were never produced.
	if strings.TrimSpace(result.ContentHash) == "" {
		return nil, errors.New("uLoRA compile returned no content hash")
	}
	if strings.TrimSpace(result.WeightsPath) == "" || strings.TrimSpace(result.ManifestPath) == "" {
		return nil, fmt.Errorf("uLoRA compile returned incomplete output paths (manifest=%q weights=%q)",
			result.ManifestPath, result.WeightsPath)
	}
	return &result, nil
}

// gatherClusterDatasets builds the training corpus for a cluster: for each
// member error, its context as input and the validated fix from that member's
// already-minted skill.md as the target.
//
// A member with no resolvable fix is skipped rather than filled with a
// placeholder: training on invented targets would transmute fabricated data into
// the adapter's weights, which is the whole thing this pipeline exists to avoid.
// If nothing is left, that is an error, not an empty corpus.
func gatherClusterDatasets(cluster *ErrorCluster, docs SkillDocSource) ([]ULoRADatasetRecord, []string, error) {
	if cluster == nil {
		return nil, nil, errors.New("cluster is required")
	}
	if docs == nil {
		return nil, nil, errors.New("no skill document source configured")
	}

	// Deterministic order, so the same cluster always compiles from the same
	// corpus and produces the same content hash.
	members := make([]*ErrorNode, 0, len(cluster.Errors))
	for _, member := range cluster.Errors {
		if member != nil {
			members = append(members, member)
		}
	}
	sort.Slice(members, func(i, j int) bool { return members[i].GetId() < members[j].GetId() })

	var (
		records  []ULoRADatasetRecord
		skillIDs []string
		skipped  []string
	)
	for _, member := range members {
		errorID := strings.TrimSpace(member.GetId())
		if errorID == "" {
			continue
		}
		fix, err := docs.SkillDocForError(errorID)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve skill document for error %s: %w", errorID, err)
		}
		if strings.TrimSpace(fix) == "" {
			skipped = append(skipped, errorID)
			continue
		}
		records = append(records, ULoRADatasetRecord{
			Context:             errorContextForTraining(member),
			CorrectedCompletion: fix,
			TargetModel:         member.GetModelOrigin(),
		})
		skillIDs = append(skillIDs, errorID)
	}

	if len(records) == 0 {
		return nil, nil, fmt.Errorf("cluster %s has no member with a resolvable skill document (%d member(s) checked)",
			cluster.ClusterID, len(members))
	}
	return records, skillIDs, nil
}

// errorContextForTraining renders a member error's context as the input half of
// a training record. It carries the error's identity and description alongside
// the structured context, because embedding the context map alone makes
// structurally similar errors indistinguishable — the same reasoning the DRQ
// loop's failure-context builder uses.
func errorContextForTraining(member *ErrorNode) string {
	if member == nil {
		return ""
	}
	context := map[string]any{
		"error_type":  member.GetErrorType(),
		"domain":      member.GetDomain(),
		"description": member.GetDescription(),
	}
	if fields := member.GetContext(); fields != nil {
		context["context"] = fields.AsMap()
	}
	if raw := member.GetFailureContext(); len(raw) > 0 {
		context["failure_context"] = string(raw)
	}
	encoded, err := json.Marshal(context)
	if err != nil {
		// Fall back to the human-readable fields rather than dropping the record.
		return member.GetErrorType() + " " + member.GetDomain() + " " + member.GetDescription()
	}
	return string(encoded)
}

// targetModelsForCorpus returns the distinct architectures the corpus targets.
// Compilation needs each model's full architecture spec, so an unknown model is
// an error: guessing hidden sizes would produce an adapter for a model that does
// not exist.
func targetModelsForCorpus(dataset []ULoRADatasetRecord, specs map[string]ULoRATargetModel) ([]ULoRATargetModel, error) {
	seen := make(map[string]bool)
	var names []string
	for _, record := range dataset {
		name := strings.TrimSpace(record.TargetModel)
		if name == "" {
			return nil, errors.New("dataset record has no target model")
		}
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	sort.Strings(names)

	models := make([]ULoRATargetModel, 0, len(names))
	var missing []string
	for _, name := range names {
		spec, ok := specs[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		if err := spec.validate(); err != nil {
			// Fail here rather than at bind: ulora validates the manifest when a
			// bundle is opened, so an incomplete spec compiles into a bundle that
			// can never be bound — the compile reports success and the artifact
			// is dead on arrival.
			return nil, fmt.Errorf("target model %q has an invalid architecture spec: %w", name, err)
		}
		models = append(models, spec)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("no architecture spec configured for target model(s): %s", strings.Join(missing, ", "))
	}
	return models, nil
}
