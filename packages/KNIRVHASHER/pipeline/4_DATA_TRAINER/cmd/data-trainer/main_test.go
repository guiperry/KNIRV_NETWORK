package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"knirvhasher/pkg/hashing/schema"
	"knirvhasher/pkg/hashing/semanticmemory"
	"knirvhasher/pkg/hashing/transformer"
)

func tinyModelConfig() *transformer.GorgoniteConfig {
	return &transformer.GorgoniteConfig{
		VocabSize:    16,
		EmbedDim:     8,
		NumHeads:     2,
		NumLayers:    2,
		ContextLen:   6,
		FFNHiddenDim: 16,
		DecayAlpha:   0.95,
	}
}

func writeTestFrames(t *testing.T, dir string, frames []schema.TrainingFrame) string {
	t.Helper()
	path := filepath.Join(dir, "training_frames.json")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test frames: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(frames); err != nil {
		t.Fatalf("encode test frames: %v", err)
	}
	return path
}

func TestRun_TrainsAndSavesCheckpoint(t *testing.T) {
	dir := t.TempDir()
	frames := []schema.TrainingFrame{
		{SourceFile: "a.txt", TokenSequence: []int32{1, 2, 3}, TargetTokenID: 4},
		{SourceFile: "b.txt", TokenSequence: []int32{5, 6, 7}, TargetTokenID: 8},
	}
	inputPath := writeTestFrames(t, dir, frames)

	cfg := &Config{
		InputPath:     inputPath,
		CheckpointDir: filepath.Join(dir, "ckpt"),
		NumEpochs:     2,
		LearningRate:  0.05,
		SaveFreq:      1,
		Mode:          "gpt",
		ModelConfig:   tinyModelConfig(),
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	latest := filepath.Join(cfg.CheckpointDir, "model_latest.bin")
	if _, err := os.Stat(latest); err != nil {
		t.Fatalf("expected latest checkpoint at %s: %v", latest, err)
	}

	epoch1 := filepath.Join(cfg.CheckpointDir, "checkpoint_epoch_1.bin")
	if _, err := os.Stat(epoch1); err != nil {
		t.Fatalf("expected epoch 1 checkpoint at %s: %v", epoch1, err)
	}
}

func TestRun_ResumeFromCheckpoint(t *testing.T) {
	dir := t.TempDir()
	frames := []schema.TrainingFrame{
		{SourceFile: "a.txt", TokenSequence: []int32{1, 2, 3}, TargetTokenID: 4},
	}

	// First run: train 1 epoch and save.
	cfg1 := &Config{
		InputPath:     writeTestFrames(t, dir, frames),
		CheckpointDir: filepath.Join(dir, "ckpt1"),
		NumEpochs:     1,
		LearningRate:  0.05,
		SaveFreq:      1,
		Mode:          "gpt",
		ModelConfig:   tinyModelConfig(),
	}
	if err := Run(cfg1); err != nil {
		t.Fatalf("initial Run: %v", err)
	}

	// Second run: resume from the checkpoint and train 1 more epoch.
	cfg2 := &Config{
		InputPath:     writeTestFrames(t, dir, frames),
		CheckpointDir: filepath.Join(dir, "ckpt2"),
		NumEpochs:     1,
		LearningRate:  0.05,
		SaveFreq:      1,
		Mode:          "gpt",
		ResumeFrom:    filepath.Join(cfg1.CheckpointDir, "model_latest.bin"),
		ModelConfig:   tinyModelConfig(),
	}
	if err := Run(cfg2); err != nil {
		t.Fatalf("resume Run: %v", err)
	}
}

func TestRun_SemanticStreamsAndSavesMemory(t *testing.T) {
	oldDecoder := newTokenContextDecoder
	newTokenContextDecoder = func() (func([]int32) string, error) {
		return func(tokens []int32) string { return "semantic training test context" }, nil
	}
	t.Cleanup(func() { newTokenContextDecoder = oldDecoder })

	dir := t.TempDir()
	frames := []schema.TrainingFrame{
		{SourceFile: "a.txt", TokenSequence: []int32{9906, 1917}, TargetTokenID: 374},
		{SourceFile: "b.txt", TokenSequence: []int32{374, 701}, TargetTokenID: 836},
	}
	cfg := &Config{
		InputPath:     writeTestFrames(t, dir, frames),
		CheckpointDir: filepath.Join(dir, "semantic"),
		NumEpochs:     1,
		LearningRate:  0.01,
		SaveFreq:      1,
		Mode:          "semantic",
		MaxPrototypes: 8,
	}
	if err := Run(cfg); err != nil {
		t.Fatalf("Run semantic: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.CheckpointDir, "semantic_memory.json")); err != nil {
		t.Fatalf("expected semantic memory checkpoint: %v", err)
	}
}

func TestRun_SemanticDoesNotApplyPublishedBatchTwice(t *testing.T) {
	oldDecoder := newTokenContextDecoder
	newTokenContextDecoder = func() (func([]int32) string, error) { return func([]int32) string { return "idempotent batch" }, nil }
	t.Cleanup(func() { newTokenContextDecoder = oldDecoder })

	framesDir := t.TempDir()
	batchID := "batch-idempotent"
	batchDir := filepath.Join(framesDir, "batches", batchID)
	if err := os.MkdirAll(batchDir, 0755); err != nil {
		t.Fatal(err)
	}
	framesPath := writeTestFrames(t, batchDir, []schema.TrainingFrame{{SourceFile: "a", TokenSequence: []int32{1, 2}, TargetTokenID: 3}})
	data, err := os.ReadFile(framesPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	manifest := fmt.Sprintf(`{"version":1,"batch_id":%q,"artifacts":{"json":"training_frames.json"},"sha256":{"json":%q}}`, batchID, fmt.Sprintf("%x", sum))
	if err := os.WriteFile(filepath.Join(framesDir, "latest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{InputPath: filepath.Join(framesDir, "latest.json"), CheckpointDir: filepath.Join(framesDir, "trainer-checkpoints"), NumEpochs: 1, LearningRate: 0.01, Mode: "semantic", MaxPrototypes: 8}
	if err := Run(cfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := Run(cfg); err != nil {
		t.Fatalf("second run: %v", err)
	}
	model, err := semanticmemory.Load(filepath.Join(cfg.CheckpointDir, "semantic_memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	if model.FramesSeen != 1 || !model.HasAppliedBatch(batchID) {
		t.Fatalf("batch was applied more than once: frames=%d batches=%v", model.FramesSeen, model.AppliedBatches)
	}
}

func TestRun_NoFrames(t *testing.T) {
	dir := t.TempDir()
	writeTestFrames(t, dir, []schema.TrainingFrame{})

	cfg := &Config{
		InputPath:     filepath.Join(dir, "training_frames.json"),
		CheckpointDir: filepath.Join(dir, "ckpt"),
		NumEpochs:     1,
		LearningRate:  0.01,
		Mode:          "gpt",
	}
	if err := Run(cfg); err == nil {
		t.Fatal("expected error when no frames are loaded")
	}
}

func TestRun_MissingInput(t *testing.T) {
	cfg := &Config{
		InputPath:     filepath.Join(t.TempDir(), "nonexistent.json"),
		CheckpointDir: filepath.Join(t.TempDir(), "ckpt"),
		NumEpochs:     1,
		LearningRate:  0.01,
		Mode:          "gpt",
	}
	if err := Run(cfg); err == nil {
		t.Fatal("expected error for missing input file")
	}
}
