package compiler

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ulora/internal/api"
	"ulora/internal/bundling"
	"ulora/internal/config"
	"ulora/internal/connector"
	"ulora/internal/safetensors"
)

const (
	PathASVD = "subspace_svd"
	PathBSFT = "synthetic_distill"
)

type Compiler struct {
	cfg    *config.Config
	engine *Engine
}

func NewCompiler(cfg *config.Config) *Compiler {
	return &Compiler{
		cfg:    cfg,
		engine: NewEngine(cfg),
	}
}

func (c *Compiler) SetEnginesDir(dir string) {
	c.engine.SetEnginesDir(dir)
}

// CompileParams carries the caller's training settings.
//
// These used to be hardcoded inside CompileCluster: the daemon parsed rank,
// alpha, epochs and learning_rate from the request, validated and defaulted them,
// and then called CompileCluster without them — so a caller's settings were
// silently discarded and every bundle came out at rank 16 / alpha 32 / 50 epochs
// regardless of what was asked for.
type CompileParams struct {
	Rank         int
	Alpha        float64
	LearningRate float64
	Epochs       int
}

// withDefaults fills any unset field with the protocol default, so a caller that
// omits something gets the documented value rather than a zero.
func (p CompileParams) withDefaults() CompileParams {
	if p.Rank <= 0 {
		p.Rank = defaultRank
	}
	if p.Alpha <= 0 {
		p.Alpha = defaultAlpha
	}
	if p.LearningRate <= 0 {
		p.LearningRate = defaultLearningRate
	}
	if p.Epochs <= 0 {
		p.Epochs = defaultEpochs
	}
	return p
}

const (
	defaultRank         = 16
	defaultAlpha        = 32.0
	defaultLearningRate = 0.01
	defaultEpochs       = 50
)

func (c *Compiler) CompileCluster(clusterID string, datasets []api.DatasetRecord, targetModels []api.BaseModelSpec, provenance api.CompileProvenance, params CompileParams) (*api.CompileResponse, error) {
	if len(targetModels) == 0 {
		return nil, fmt.Errorf("at least one target model is required")
	}
	if len(datasets) == 0 {
		return nil, fmt.Errorf("at least one dataset record is required")
	}

	workDir, err := os.MkdirTemp("", "ulora-compile-*")
	if err != nil {
		return nil, fmt.Errorf("create work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	params = params.withDefaults()
	sourceModel := targetModels[0]
	rank := params.Rank
	alpha := params.Alpha
	lr := params.LearningRate
	epochs := params.Epochs

	sourceWeightsPath := filepath.Join(workDir, "source_weights.safetensors")
	if err := c.engine.RunPathB(datasets, sourceModel, rank, alpha, lr, epochs, DefaultCanonicalDim, targetModuleRoles, AllowSharedCore, sourceWeightsPath); err != nil {
		return nil, fmt.Errorf("path B SFT training failed: %w", err)
	}

	manifest := buildManifest(clusterID, provenance, sourceModel, rank, alpha, targetModels)

	routing := api.RoutingPolicy{
		SimilarityThreshold:  api.DefaultTau,
		AllowedTransferPaths: []string{"subspace_svd", "synthetic_distill"},
	}

	usedPath := PathBSFT
	// No merge prefix on the source output: Path B writes the canonical core
	// already namespaced by semantic module ("self_attn.q_proj/lora_A"), so
	// prefixing would produce a name no binder looks for.
	inputFiles := []map[string]any{
		{"path": sourceWeightsPath, "prefix": ""},
	}

	if len(targetModels) > 1 {
		if AllAboveThreshold(sourceModel, targetModels, routing.SimilarityThreshold) {
			for i := 1; i < len(targetModels); i++ {
				tgt := targetModels[i]
				tgtPath := filepath.Join(workDir, fmt.Sprintf("weights_%d.safetensors", i))
				if err := c.engine.RunPathA(sourceWeightsPath, sourceModel, tgt, rank, alpha, DefaultCanonicalDim, targetModuleRoles, modelPrefix(tgt), tgtPath); err != nil {
					return nil, fmt.Errorf("path A transfer to %s failed: %w", tgt.Family, err)
				}
				usedPath = PathASVD
				// No merge prefix: the engine already namespaces the core by
				// semantic module and the connector by tensor_prefix.
				inputFiles = append(inputFiles, map[string]any{
					"path":   tgtPath,
					"prefix": "",
				})
			}
		} else {
			for i := 1; i < len(targetModels); i++ {
				tgt := targetModels[i]
				tgtPath := filepath.Join(workDir, fmt.Sprintf("weights_%d.safetensors", i))
				if err := c.engine.RunPathB(datasets, tgt, rank, alpha, lr, epochs, DefaultCanonicalDim, targetModuleRoles, AllowSharedCore, tgtPath); err != nil {
					return nil, fmt.Errorf("path B SFT for target %s failed: %w", tgt.Family, err)
				}
				inputFiles = append(inputFiles, map[string]any{
					"path":   tgtPath,
					"prefix": modelPrefix(tgt),
				})
			}
		}
	}

	bundleDir, err := createBundleDir(c.cfg.DataDir, clusterID, provenance.SourceID)
	if err != nil {
		return nil, fmt.Errorf("create bundle directory: %w", err)
	}

	manifestPath := filepath.Join(bundleDir, bundling.ManifestFile)
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestJSON, 0o644); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	combinedWeightsPath := filepath.Join(bundleDir, bundling.WeightsFile)
	if err := c.engine.MergeSafetensors(inputFiles, combinedWeightsPath); err != nil {
		return nil, fmt.Errorf("merge weights into bundle: %w", err)
	}

	anchorsPath := filepath.Join(bundleDir, bundling.AnchorsFile)
	if err := c.engine.WriteParquet(datasets, anchorsPath); err != nil {
		return nil, fmt.Errorf("write calibration anchors: %w", err)
	}

	bundlePath := filepath.Join(c.cfg.DataDir, "bundles", fmt.Sprintf("ulora-%s.ulora", clusterID))
	if err := bundling.WriteBundle(bundleDir, bundlePath); err != nil {
		return nil, fmt.Errorf("write bundle: %w", err)
	}

	contentHash, err := bundling.ContentHashOfFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("hash bundle: %w", err)
	}

	return &api.CompileResponse{
		BundleID:     clusterID,
		ContentHash:  contentHash,
		ManifestPath: manifestPath,
		WeightsPath:  combinedWeightsPath,
		AnchorsPath:  anchorsPath,
		PathUsed:     usedPath,
		TargetModels: modelNames(targetModels),
	}, nil
}

func (c *Compiler) Transfer(sourceBundlePath string, targetModels []api.BaseModelSpec) (*api.TransferResponse, error) {
	if len(targetModels) == 0 {
		return nil, fmt.Errorf("at least one target model is required")
	}

	workDir, err := os.MkdirTemp("", "ulora-transfer-*")
	if err != nil {
		return nil, fmt.Errorf("create work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	bundleDir := filepath.Join(workDir, "extracted")
	if err := bundling.ExtractBundle(sourceBundlePath, bundleDir); err != nil {
		return nil, fmt.Errorf("extract source bundle: %w", err)
	}

	manifestData, err := bundling.ReadFile(bundleDir, bundling.ManifestFile)
	if err != nil {
		return nil, fmt.Errorf("read manifest from bundle: %w", err)
	}
	manifest, err := api.ManifestFromJSON(manifestData)
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	sourceModel := manifest.BaseSourceModel
	rank := manifest.CanonicalCore.AdapterRank
	alpha := manifest.CanonicalCore.ScalingFactorAlpha

	sourceWeightsPath := filepath.Join(bundleDir, bundling.WeightsFile)

	usedPath := PathASVD
	inputFiles := []map[string]any{}

	if AllAboveThreshold(sourceModel, targetModels, manifest.RoutingPolicy.SimilarityThreshold) {
		for i, tgt := range targetModels {
			tgtPath := filepath.Join(workDir, fmt.Sprintf("transfer_%d.safetensors", i))
			if err := c.engine.RunPathA(sourceWeightsPath, sourceModel, tgt, rank, alpha, DefaultCanonicalDim, targetModuleRoles, modelPrefix(tgt), tgtPath); err != nil {
				return nil, fmt.Errorf("path A transfer to %s failed: %w", tgt.Family, err)
			}
			// No merge prefix: the engine already namespaces the core by
			// semantic module and the connector by tensor_prefix.
			inputFiles = append(inputFiles, map[string]any{
				"path":   tgtPath,
				"prefix": "",
			})
		}
	} else {
		for i, tgt := range targetModels {
			tgtPath := filepath.Join(workDir, fmt.Sprintf("transfer_%d.safetensors", i))
			if err := c.engine.RunPathA(sourceWeightsPath, sourceModel, tgt, rank, alpha, DefaultCanonicalDim, targetModuleRoles, modelPrefix(tgt), tgtPath); err != nil {
				return nil, fmt.Errorf("path A transfer to %s failed: %w", tgt.Family, err)
			}
			// No merge prefix: the engine already namespaces the core by
			// semantic module and the connector by tensor_prefix.
			inputFiles = append(inputFiles, map[string]any{
				"path":   tgtPath,
				"prefix": "",
			})
		}
	}

	transferID := fmt.Sprintf("transfer-%d", time.Now().UnixNano())
	transferDir, err := createBundleDir(c.cfg.DataDir, transferID, "transfer")
	if err != nil {
		return nil, fmt.Errorf("create transfer directory: %w", err)
	}

	manifestPath := filepath.Join(transferDir, bundling.ManifestFile)
	updatedManifest := *manifest
	updatedManifest.Name = fmt.Sprintf("ulora-transfer-%s", transferID)
	manifestJSON, err := json.MarshalIndent(&updatedManifest, "", " ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestJSON, 0o644); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	combinedWeightsPath := filepath.Join(transferDir, bundling.WeightsFile)
	if err := c.engine.MergeSafetensors(inputFiles, combinedWeightsPath); err != nil {
		return nil, fmt.Errorf("merge weights into transfer bundle: %w", err)
	}

	bundlePath := filepath.Join(c.cfg.DataDir, "bundles", fmt.Sprintf("%s.ulora", transferID))
	if err := bundling.WriteBundle(transferDir, bundlePath); err != nil {
		return nil, fmt.Errorf("write transfer bundle: %w", err)
	}

	contentHash, err := bundling.ContentHashOfFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("hash bundle: %w", err)
	}

	return &api.TransferResponse{
		BundleID:     transferID,
		ContentHash:  contentHash,
		ManifestPath: manifestPath,
		WeightsPath:  combinedWeightsPath,
		PathUsed:     usedPath,
		TargetModels: modelNames(targetModels),
	}, nil
}

// AllowSharedCore lets the direct-fit engine serve modules the corpus does not
// attribute with one shared core.
//
// It is true because a bundle must carry a core for every module its manifest
// advertises (targetModuleRoles), or a binder looks up tensors that were never
// written — and a single error cluster is unlikely to attribute records to all
// seven. The engine records exactly which modules were served this way
// (modules_with_shared_core, core_shared_across_modules), so the limitation
// travels with the artifact instead of being hidden by it.
//
// The better behaviour, once the corpus can attribute per layer, is to narrow
// the advertised module set to what is actually attributed rather than to widen
// the artifact with a shared core.
const AllowSharedCore = true

// targetModuleRoles are the semantic layer roles a bundle carries a canonical
// core for. Declared once and shared: the manifest advertises the roles and the
// Path A engine writes a core tensor per role, so if the two lists diverged the
// binder would look for tensors that were never written.
var targetModuleRoles = []string{
	"self_attn.q_proj", "self_attn.k_proj", "self_attn.v_proj", "self_attn.o_proj",
	"mlp.gate_proj", "mlp.up_proj", "mlp.down_proj",
}

// DefaultCanonicalDim is K: the dimension of the shared latent space the
// canonical core lives in. It is fixed per protocol rather than per model —
// that is the whole point, since every binder projects into this same space —
// but it is carried in the manifest so a bundle stays interpretable if the
// protocol default ever changes.
const DefaultCanonicalDim = 1024

func buildManifest(clusterID string, provenance api.CompileProvenance, sourceModel api.BaseModelSpec, rank int, alpha float64, targets []api.BaseModelSpec) *api.Manifest {
	return &api.Manifest{
		Schema:  api.SchemaURL,
		Name:    fmt.Sprintf("ulora-cluster-%s", clusterID),
		Version: api.SpecVersion,
		Provenance: api.Provenance{
			SourceID:         provenance.SourceID,
			SourceDatasetIDs: provenance.SourceDatasetIDs,
			Extensions:       provenance.Extensions,
		},
		BaseSourceModel: sourceModel,
		CanonicalCore: api.CanonicalCore{
			AdapterRank:        rank,
			ScalingFactorAlpha: alpha,
			CanonicalDim:       DefaultCanonicalDim,
			TargetModules:      targetModuleRoles,
		},
		RoutingPolicy: api.RoutingPolicy{
			SimilarityThreshold:  api.DefaultTau,
			AllowedTransferPaths: []string{"subspace_svd", "synthetic_distill"},
		},
	}
}

func createBundleDir(dataDir, clusterID, sourceID string) (string, error) {
	dirName := fmt.Sprintf("%s-%s", clusterID, hashShort(sourceID))
	dir := filepath.Join(dataDir, "bundles", dirName)
	if err := bundling.EnsureDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func hashShort(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:8])
}

func modelPrefix(m api.BaseModelSpec) string {
	return fmt.Sprintf("%s-%s", m.Family, m.ParamCount)
}

func modelNames(models []api.BaseModelSpec) []string {
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = fmt.Sprintf("%s-%s", m.Family, m.ParamCount)
	}
	return names
}

// DeriveConnector derives the connector for one model and returns it.
//
// The engine writes safetensors and this reads it back with the same reader the
// binder uses, so a connector can never be cached in a form the binder could not
// parse — the two would otherwise drift and fail only at bind time.
func (c *Compiler) DeriveConnector(targetModel api.BaseModelSpec, weightMatrix [][]float64, canonicalDim, rank int) (*connector.Connector, error) {
	if canonicalDim <= 0 {
		canonicalDim = DefaultCanonicalDim
	}

	workDir, err := os.MkdirTemp(c.cfg.DataDir, "connector-derive-")
	if err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	outPath := filepath.Join(workDir, "connector.safetensors")
	if err := c.engine.DeriveConnector(targetModel, weightMatrix, canonicalDim, rank, outPath); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("read derived connector: %w", err)
	}
	file, err := safetensors.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse derived connector: %w", err)
	}

	pIn, err := connectorMatrix(file, connector.TensorPIn)
	if err != nil {
		return nil, err
	}
	pOut, err := connectorMatrix(file, connector.TensorPOut)
	if err != nil {
		return nil, err
	}

	return &connector.Connector{
		Family:       targetModel.Family,
		ParamCount:   targetModel.ParamCount,
		CanonicalDim: canonicalDim,
		PIn:          pIn,
		POut:         pOut,
	}, nil
}

func connectorMatrix(file *safetensors.File, name string) ([][]float32, error) {
	tensor, ok := file.Tensor(name)
	if !ok {
		return nil, fmt.Errorf("derived connector has no tensor %q", name)
	}
	return tensor.Float32Matrix()
}
