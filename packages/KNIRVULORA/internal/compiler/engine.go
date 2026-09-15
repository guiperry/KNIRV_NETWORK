package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"ulora/internal/api"
	"ulora/internal/config"
)

type Engine struct {
	cfg        *config.Config
	venvPy     string
	enginesDir string
}

func NewEngine(cfg *config.Config) *Engine {
	venvPy := cfg.PythonBin
	if cfg.VenvDir != "" {
		venvPython := filepath.Join(cfg.VenvDir, "bin", "python")
		if _, err := os.Stat(venvPython); err == nil {
			venvPy = venvPython
		}
	}
	enginesDir := filepath.Join("internal", "compiler", "engines")
	return &Engine{
		cfg:        cfg,
		venvPy:     venvPy,
		enginesDir: enginesDir,
	}
}

func (e *Engine) SetEnginesDir(dir string) {
	e.enginesDir = dir
}

// pythonEnv returns the child environment with PYTHONHASHSEED pinned.
//
// path_b_sft.py encodes corpus text with Python's builtin hash(), which CPython
// salts per interpreter process for str keys. Without a fixed seed the same
// corpus encodes to different vectors on every compile, so the same input
// produces different adapter weights and therefore a different bundle content
// hash.
//
// That breaks the property this whole design rests on: content_hash is what the
// mint route verifies, what the bind route refuses to mismatch, and what a
// validation verdict is attributed to. A bundle that is not reproducible is not
// content-addressed. The mint and bind hash checks cannot catch this — they
// verify that what was sent matches what was stored, which stays true while the
// artifact itself varies run to run.
func pythonEnv() []string {
	const key = "PYTHONHASHSEED="
	inherited := os.Environ()
	env := make([]string, 0, len(inherited)+1)
	for _, entry := range inherited {
		// Drop any inherited value so ours is the only one, rather than
		// depending on how duplicate entries are resolved.
		if strings.HasPrefix(entry, key) {
			continue
		}
		env = append(env, entry)
	}
	// 0 disables hash randomisation, making hash() stable for str.
	return append(env, key+"0")
}

func (e *Engine) runPython(scriptName string, payload any, timeout time.Duration) (string, error) {
	scriptPath := filepath.Join(e.enginesDir, scriptName)
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	cmd := exec.Command(e.venvPy, scriptPath)
	cmd.Env = pythonEnv()
	cmd.Stdin = bytes.NewReader(data)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		return "", fmt.Errorf("python engine %s timed out after %v", scriptName, timeout)
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("python engine %s failed: %s", scriptName, stderr.String())
		}
	}

	return stdout.String(), nil
}

// RunPathB drives the direct-fit engine, which produces the canonical core
// in the shared K-space rather than a delta fitted to one target.
//
// canonicalDim and targetModules are part of the input contract: the core lives
// in the canonical space, so K defines every tensor's shape, and a core is
// produced per semantic module. The engine refuses without them.
func (e *Engine) RunPathB(dataset []api.DatasetRecord, targetModel api.BaseModelSpec, rank int, alpha float64, learningRate float64, epochs int, canonicalDim int, targetModules []string, allowSharedCore bool, outputPath string) error {
	config := map[string]any{
		"rank":              rank,
		"alpha":             alpha,
		"learning_rate":     learningRate,
		"epochs":            epochs,
		"canonical_dim":     canonicalDim,
		"target_modules":    targetModules,
		"allow_shared_core": allowSharedCore,
		// encode_dim is deliberately not sent: the core acts in canonical space,
		// so the embedding must be K-wide. The engine defaults it to K and
		// refuses a mismatch, rather than this caller silently picking 512.
	}

	payload := map[string]any{
		"dataset":      dataset,
		"target_model": modelSpecToMap(targetModel),
		"config":       config,
		"output_path":  outputPath,
	}

	_, err := e.runPython("path_b_sft.py", payload, e.cfg.RequestTimeout)
	return err
}

// RunPathA drives the subspace-transfer engine, which emits the canonical core
// plus a per-family connector rather than a delta fitted to one model.
//
// canonicalDim, targetModules and tensorPrefix are part of the input contract:
// the engine refuses to run without K (the shared-space dimension), the semantic
// module list it should produce a core for, and the prefix its connector tensors
// are namespaced under. Without them it raises PathAError rather than guessing,
// because guessing K would reinterpret every tensor it writes.
func (e *Engine) RunPathA(sourceWeightsPath string, sourceModel, targetModel api.BaseModelSpec, rank int, alpha float64, canonicalDim int, targetModules []string, tensorPrefix string, outputPath string) error {
	config := map[string]any{
		"rank":           rank,
		"alpha":          alpha,
		"canonical_dim":  canonicalDim,
		"target_modules": targetModules,
		"tensor_prefix":  tensorPrefix,
	}

	payload := map[string]any{
		"source_weights_path": sourceWeightsPath,
		"source_model":        modelSpecToMap(sourceModel),
		"target_model":        modelSpecToMap(targetModel),
		"config":              config,
		"output_path":         outputPath,
	}

	_, err := e.runPython("path_a_svd.py", payload, e.cfg.RequestTimeout)
	return err
}

func (e *Engine) WriteParquet(records []api.DatasetRecord, outputPath string) error {
	payload := map[string]any{
		"records":     records,
		"output_path": outputPath,
	}
	_, err := e.runPython("write_parquet.py", payload, e.cfg.RequestTimeout)
	return err
}

func (e *Engine) MergeSafetensors(inputFiles []map[string]any, outputPath string) error {
	payload := map[string]any{
		"inputs":      inputFiles,
		"output_path": outputPath,
	}
	_, err := e.runPython("merge_safetensors.py", payload, e.cfg.RequestTimeout)
	return err
}

func modelSpecToMap(m api.BaseModelSpec) map[string]any {
	return map[string]any{
		"family":              m.Family,
		"param_count":         m.ParamCount,
		"hidden_size":         m.HiddenSize,
		"intermediate_size":   m.IntermediateSize,
		"num_attention_heads": m.NumAttentionHeads,
		"num_key_value_heads": m.NumKeyValueHeads,
		"head_dim":            m.HeadDim,
		"num_layers":          m.NumLayers,
		"attention_mechanism": m.AttentionMechanism,
		"activation_func":     m.ActivationFunc,
	}
}

// DeriveConnector runs the standalone connector engine for a model.
//
// It takes the model's weight matrix because a connector is built from that
// model's principal subspaces; there is no fallback, since an invented basis
// would bind without aligning anything. Used to populate the runtime's per-model
// connector cache.
func (e *Engine) DeriveConnector(targetModel api.BaseModelSpec, weightMatrix [][]float64, canonicalDim, rank int, outputPath string) error {
	spec := modelSpecToMap(targetModel)
	spec["weight_matrix"] = weightMatrix

	config := map[string]any{"canonical_dim": canonicalDim}
	if rank > 0 {
		config["rank"] = rank
	}

	payload := map[string]any{
		"target_model": spec,
		"config":       config,
		"output_path":  outputPath,
	}

	_, err := e.runPython("derive_connector.py", payload, e.cfg.RequestTimeout)
	return err
}
