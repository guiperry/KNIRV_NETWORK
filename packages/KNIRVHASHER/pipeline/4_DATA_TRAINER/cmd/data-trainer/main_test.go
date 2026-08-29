package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"knirvhasher/pkg/hashing/schema"
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
		ResumeFrom:    filepath.Join(cfg1.CheckpointDir, "model_latest.bin"),
		ModelConfig:   tinyModelConfig(),
	}
	if err := Run(cfg2); err != nil {
		t.Fatalf("resume Run: %v", err)
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
	}
	if err := Run(cfg); err == nil {
		t.Fatal("expected error for missing input file")
	}
}
