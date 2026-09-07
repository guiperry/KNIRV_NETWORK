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
)

const (
	PathASVD       = "subspace_svd"
	PathBSFT       = "synthetic_distill"
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

func (c *Compiler) CompileCluster(clusterID string, datasets []api.DatasetRecord, targetModels []api.BaseModelSpec, provenance api.CompileProvenance) (*api.CompileResponse, error) {
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

	sourceModel := targetModels[0]
	rank := 16
	alpha := 32.0
	lr := 0.01
	epochs := 50

	sourceWeightsPath := filepath.Join(workDir, "source_weights.safetensors")
	if err := c.engine.RunPathB(datasets, sourceModel, rank, alpha, lr, epochs, sourceWeightsPath); err != nil {
		return nil, fmt.Errorf("path B SFT training failed: %w", err)
	}

	manifest := buildManifest(clusterID, provenance, sourceModel, rank, alpha, targetModels)

	routing := api.RoutingPolicy{
		SimilarityThreshold:  api.DefaultTau,
		AllowedTransferPaths: []string{"subspace_svd", "synthetic_distill"},
	}

	usedPath := PathBSFT
	inputFiles := []map[string]any{
		{"path": sourceWeightsPath, "prefix": "source"},
	}

	if len(targetModels) > 1 {
		if AllAboveThreshold(sourceModel, targetModels, routing.SimilarityThreshold) {
			for i := 1; i < len(targetModels); i++ {
				tgt := targetModels[i]
				tgtPath := filepath.Join(workDir, fmt.Sprintf("weights_%d.safetensors", i))
				if err := c.engine.RunPathA(sourceWeightsPath, sourceModel, tgt, rank, alpha, tgtPath); err != nil {
					return nil, fmt.Errorf("path A transfer to %s failed: %w", tgt.Family, err)
				}
				usedPath = PathASVD
				inputFiles = append(inputFiles, map[string]any{
					"path":   tgtPath,
					"prefix": modelPrefix(tgt),
				})
			}
		} else {
			for i := 1; i < len(targetModels); i++ {
				tgt := targetModels[i]
				tgtPath := filepath.Join(workDir, fmt.Sprintf("weights_%d.safetensors", i))
				if err := c.engine.RunPathB(datasets, tgt, rank, alpha, lr, epochs, tgtPath); err != nil {
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
			if err := c.engine.RunPathA(sourceWeightsPath, sourceModel, tgt, rank, alpha, tgtPath); err != nil {
				return nil, fmt.Errorf("path A transfer to %s failed: %w", tgt.Family, err)
			}
			inputFiles = append(inputFiles, map[string]any{
				"path":   tgtPath,
				"prefix": modelPrefix(tgt),
			})
		}
	} else {
		for i, tgt := range targetModels {
			tgtPath := filepath.Join(workDir, fmt.Sprintf("transfer_%d.safetensors", i))
			if err := c.engine.RunPathA(sourceWeightsPath, sourceModel, tgt, rank, alpha, tgtPath); err != nil {
				return nil, fmt.Errorf("path A transfer to %s failed: %w", tgt.Family, err)
			}
			inputFiles = append(inputFiles, map[string]any{
				"path":   tgtPath,
				"prefix": modelPrefix(tgt),
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

func buildManifest(clusterID string, provenance api.CompileProvenance, sourceModel api.BaseModelSpec, rank int, alpha float64, targets []api.BaseModelSpec) *api.Manifest {
	targetModuleList := []string{
		"self_attn.q_proj", "self_attn.k_proj", "self_attn.v_proj", "self_attn.o_proj",
		"mlp.gate_proj", "mlp.up_proj", "mlp.down_proj",
	}

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
			TargetModules:      targetModuleList,
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
