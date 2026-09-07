package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"ulora/internal/api"
	"ulora/internal/config"
)

type Engine struct {
	cfg       *config.Config
	venvPy    string
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

func (e *Engine) runPython(scriptName string, payload any, timeout time.Duration) (string, error) {
	scriptPath := filepath.Join(e.enginesDir, scriptName)
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	cmd := exec.Command(e.venvPy, scriptPath)
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

func (e *Engine) RunPathB(dataset []api.DatasetRecord, targetModel api.BaseModelSpec, rank int, alpha float64, learningRate float64, epochs int, outputPath string) error {
	config := map[string]any{
		"rank":            rank,
		"alpha":           alpha,
		"learning_rate":   learningRate,
		"epochs":          epochs,
		"encode_dim":      min(targetModel.HiddenSize, 512),
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

func (e *Engine) RunPathA(sourceWeightsPath string, sourceModel, targetModel api.BaseModelSpec, rank int, alpha float64, outputPath string) error {
	config := map[string]any{
		"rank":  rank,
		"alpha": alpha,
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
		"family":               m.Family,
		"param_count":          m.ParamCount,
		"hidden_size":          m.HiddenSize,
		"intermediate_size":    m.IntermediateSize,
		"num_attention_heads":  m.NumAttentionHeads,
		"num_key_value_heads":  m.NumKeyValueHeads,
		"head_dim":             m.HeadDim,
		"num_layers":           m.NumLayers,
		"attention_mechanism":  m.AttentionMechanism,
		"activation_func":      m.ActivationFunc,
	}
}
