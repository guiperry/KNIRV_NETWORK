package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"knirvmodels/pkg/hands/gpt"
)

func tinyModel() *gpt.Config {
	return &gpt.Config{
		VocabSize: 16, EmbedDim: 8, NumHeads: 2, NumLayers: 1,
		ContextLen: 16, FFNHiddenDim: 16, DecayAlpha: 1.0,
	}
}

func baseConfig(t *testing.T) *Config {
	t.Helper()
	return &Config{
		CheckpointDir: t.TempDir(),
		OutPath:       filepath.Join(t.TempDir(), "model-latest"),
		Epochs:        1,
		LearningRate:  1e-3,
		MaxSteps:      2,
		ContextLen:    4,
		Stride:        4,
		MaxFrames:     100,
		ModelConfig:   tinyModel(),
	}
}

func TestRunJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.jsonl")
	content := `[1,2,3,4]
{"token_ids":[4,3,2,1]}
{"tokens":[1,2,3,4]}
{"window":[2,3,4,5]}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := baseConfig(t)
	cfg.JSONLPath = path
	summary, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.Windows != 4 || summary.Steps == 0 || summary.Steps > cfg.MaxSteps {
		t.Fatalf("summary = %+v, want 4 windows and <= %d steps", summary, cfg.MaxSteps)
	}
	if _, err := os.Stat(cfg.OutPath); err != nil {
		t.Fatalf("checkpoint missing: %v", err)
	}
}

func TestRunNRVRoutesOnlyValidFrames(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "corpus.nrv")
	writeAsset(t, asset, []assetFrame{
		{id: "valid", valid: true, tokens: []int{1, 2, 3, 4, 5, 6}},
		{id: "invalid", valid: false, tokens: []int{6, 5, 4, 3, 2, 1}},
	})
	cfg := baseConfig(t)
	cfg.InputPath = asset
	summary, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.Format != "nrv3" || summary.Frames != 2 || summary.ValidFrames != 1 || summary.SkippedInvalid != 1 {
		t.Fatalf("summary = %+v, want nrv3 with 1/2 valid", summary)
	}
	if summary.Steps == 0 {
		t.Fatalf("summary = %+v, want training steps", summary)
	}
	if _, err := os.Stat(cfg.OutPath); err != nil {
		t.Fatalf("checkpoint missing: %v", err)
	}
}

func TestRunNRVSummaryWindowsCountsTokenWindows(t *testing.T) {
	asset := filepath.Join(t.TempDir(), "windows.nrv")
	writeAsset(t, asset, []assetFrame{
		{id: "a", valid: true, tokens: []int{1, 2, 3, 4, 5, 6, 7, 8}}, // 2 windows at len 4
		{id: "b", valid: true, tokens: []int{1, 2, 3, 4}},             // 1 window
		{id: "c", valid: false, tokens: []int{1, 2, 3, 4, 5, 6, 7, 8}},
	})
	cfg := baseConfig(t)
	cfg.InputPath = asset
	summary, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.ValidFrames != 2 {
		t.Fatalf("valid frames = %d, want 2", summary.ValidFrames)
	}
	if summary.Windows != 3 {
		t.Fatalf("summary.Windows = %d, want 3 token windows (not the 2 valid frames)", summary.Windows)
	}
}

func TestRunNRVMaxStepsZeroDerivesFromDataset(t *testing.T) {
	// 600 windows at len 4 / stride 4 exceeds the old implicit 512-step cap.
	tokens := make([]int, 2400)
	for i := range tokens {
		tokens[i] = i % 16
	}
	asset := filepath.Join(t.TempDir(), "long.nrv")
	writeAsset(t, asset, []assetFrame{{id: "long", valid: true, tokens: tokens}})
	cfg := baseConfig(t)
	cfg.InputPath = asset
	cfg.MaxSteps = 0
	summary, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.Windows != 600 || summary.Steps != 600 {
		t.Fatalf("summary = %+v, want 600 windows and 600 derived steps (max-steps 0 = dataset)", summary)
	}
}

func TestRunNRVRejectsOutOfVocabularyTokens(t *testing.T) {
	asset := filepath.Join(t.TempDir(), "oov.nrv")
	writeAsset(t, asset, []assetFrame{
		{id: "good", valid: true, tokens: []int{1, 2, 3, 4}},
		{id: "bad", valid: true, tokens: []int{1, 2, 99, 4}}, // vocab is 16
	})
	cfg := baseConfig(t)
	cfg.InputPath = asset
	if _, err := Run(context.Background(), cfg); err == nil {
		t.Fatal("valid frame with out-of-vocabulary token must be rejected, not wrapped")
	}
	if _, err := os.Stat(cfg.OutPath); err == nil {
		t.Fatal("no checkpoint may be written when the corpus is rejected")
	}

	// An invalid frame carrying bad tokens never reaches training, so it must not fail the run.
	writeAsset(t, asset, []assetFrame{
		{id: "good", valid: true, tokens: []int{1, 2, 3, 4}},
		{id: "bad-invalid", valid: false, tokens: []int{1, 2, 99, 4}},
	})
	cfg = baseConfig(t)
	cfg.InputPath = asset
	if _, err := Run(context.Background(), cfg); err != nil {
		t.Fatalf("invalid frame should be ignored, got %v", err)
	}
}

func TestRunRejectsAssetWithoutValidFrames(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "invalid.nrv")
	writeAsset(t, asset, []assetFrame{{id: "invalid", valid: false, tokens: []int{1, 2, 3, 4}}})
	cfg := baseConfig(t)
	cfg.InputPath = asset
	if _, err := Run(context.Background(), cfg); err == nil {
		t.Fatal("expected an error for a corpus with no valid frames")
	}
}

func TestValidateConfig(t *testing.T) {
	cfg := baseConfig(t)
	cfg.InputPath = "a.nrv"
	cfg.JSONLPath = "b.jsonl"
	if err := validateConfig(cfg); err == nil {
		t.Fatal("both inputs must be rejected")
	}
	cfg = baseConfig(t)
	if err := validateConfig(cfg); err == nil {
		t.Fatal("missing input must be rejected")
	}
	cfg = baseConfig(t)
	cfg.JSONLPath = "b.jsonl"
	cfg.ContextLen = cfg.ModelConfig.ContextLen + 1
	if err := validateConfig(cfg); err == nil {
		t.Fatal("context longer than model context must be rejected")
	}
}

func TestReadJSONLWindowsRejectsBadTokens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	if err := os.WriteFile(path, []byte(`{"token_ids":[1,99]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readJSONLWindows(path, 16); err == nil {
		t.Fatal("out-of-vocabulary token must be rejected")
	}
	if err := os.WriteFile(path, []byte(`{"token_ids":[1]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readJSONLWindows(path, 16); err == nil {
		t.Fatal("single-token window must be rejected")
	}
}

type assetFrame struct {
	id     string
	valid  bool
	tokens []int
}

func writeAsset(t *testing.T, path string, frames []assetFrame) {
	t.Helper()
	entries := make([]map[string]any, 0, len(frames))
	for _, frame := range frames {
		entries = append(entries, map[string]any{
			"id":             frame.id,
			"timestamp_unix": int64(1),
			"count":          0,
			"offset":         0,
			"z3": map[string]any{
				"constraint_count": 1,
				"satisfiable":      frame.valid,
			},
			"payload": map[string]any{
				"token_window": map[string]any{"tokens": frame.tokens},
			},
		})
	}
	registry, err := json.Marshal(map[string]any{"frames": entries})
	if err != nil {
		t.Fatal(err)
	}
	buf := bytes.NewBuffer(nil)
	buf.WriteString("NRV3")
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(registry))); err != nil {
		t.Fatal(err)
	}
	buf.Write(registry)
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}
