// Command hands-trainer trains the HANDS inner GPT from a signed .nrv corpus
// (the pipeline handoff from 5_HEAD_TRAINER) or from a legacy token-window
// JSONL file. It is the batch counterpart of POST /models/hands/train: both
// routes funnel through pkg/hands TrainWindows, so only Z3-valid frames and
// bounded step counts are ever trained on.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"knirvmodels/pkg/hands"
	"knirvmodels/pkg/hands/gpt"
	"knirvmodels/pkg/training/nrvtrainer"
)

// Config is the trainer's resolved configuration. ModelConfig is injectable so
// tests can use a tiny architecture instead of the 100k-vocab default.
type Config struct {
	InputPath     string
	JSONLPath     string
	CheckpointDir string
	OutPath       string
	ResumeFrom    string
	VocabPath     string
	Epochs        int
	LearningRate  float64
	MaxSteps      int
	ContextLen    int
	Stride        int
	MaxFrames     int
	ModelConfig   *gpt.Config
}

// Summary is what a pass actually did, emitted as JSON on success.
type Summary struct {
	Input          string `json:"input"`
	Format         string `json:"format"`
	Frames         int    `json:"frames,omitempty"`
	ValidFrames    int    `json:"validFrames,omitempty"`
	SkippedInvalid int    `json:"skippedInvalid,omitempty"`
	Windows        int    `json:"windows"`
	Steps          int    `json:"steps"`
	Checkpoint     string `json:"checkpoint,omitempty"`
}

func loadConfig() (*Config, error) {
	defaults := gpt.DefaultConfig()
	cfg := &Config{ModelConfig: defaults}
	flag.StringVar(&cfg.InputPath, "input", "", "Path to a signed .nrv corpus (mutually exclusive with --jsonl)")
	flag.StringVar(&cfg.JSONLPath, "jsonl", "", "Legacy token-window JSONL fallback, one window per line")
	flag.StringVar(&cfg.CheckpointDir, "checkpoint-dir", "checkpoints", "Directory for the trained model")
	flag.StringVar(&cfg.OutPath, "out", "", "Checkpoint path (default <checkpoint-dir>/model-latest)")
	flag.StringVar(&cfg.ResumeFrom, "resume", "", "Optional checkpoint to resume from (architecture must match)")
	flag.StringVar(&cfg.VocabPath, "vocab", "", "Optional BPE vocab path for generation after training")
	flag.IntVar(&cfg.Epochs, "epochs", 1, "Training epochs")
	flag.Float64Var(&cfg.LearningRate, "lr", 3e-4, "Learning rate (<= 1e-3 recommended)")
	flag.IntVar(&cfg.MaxSteps, "max-steps", 512, "Hard cap on optimizer steps (0 = derive from dataset)")
	flag.IntVar(&cfg.ContextLen, "context-len", 64, "Token window length")
	flag.IntVar(&cfg.Stride, "stride", 0, "Window stride (default: context length)")
	flag.IntVar(&cfg.MaxFrames, "max-frames", nrvtrainer.MaxFramesDefault(), "Cap on frames read from the asset")
	flag.IntVar(&cfg.ModelConfig.VocabSize, "vocab-size", defaults.VocabSize, "Model vocabulary size")
	flag.IntVar(&cfg.ModelConfig.EmbedDim, "embed-dim", defaults.EmbedDim, "Model embedding width")
	flag.IntVar(&cfg.ModelConfig.NumHeads, "heads", defaults.NumHeads, "Attention heads")
	flag.IntVar(&cfg.ModelConfig.NumLayers, "layers", defaults.NumLayers, "Transformer layers")
	flag.IntVar(&cfg.ModelConfig.ContextLen, "model-context", defaults.ContextLen, "Model positional context")
	flag.IntVar(&cfg.ModelConfig.FFNHiddenDim, "ffn-dim", defaults.FFNHiddenDim, "FFN hidden width")
	flag.Float64Var(&cfg.ModelConfig.DropoutRate, "dropout", defaults.DropoutRate, "Reserved dropout rate")
	flag.Parse()

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func validateConfig(cfg *Config) error {
	if (cfg.InputPath == "") == (cfg.JSONLPath == "") {
		return fmt.Errorf("exactly one of --input or --jsonl is required")
	}
	if cfg.Epochs <= 0 {
		return fmt.Errorf("epochs must be > 0")
	}
	if cfg.LearningRate <= 0 || cfg.LearningRate > 1 {
		return fmt.Errorf("learning rate must be in (0,1]")
	}
	if cfg.MaxSteps < 0 {
		return fmt.Errorf("max-steps must be >= 0")
	}
	if cfg.ContextLen < 2 {
		return fmt.Errorf("context-len must be >= 2")
	}
	if cfg.Stride == 0 {
		cfg.Stride = cfg.ContextLen
	}
	if cfg.Stride < 1 || cfg.Stride > cfg.ContextLen {
		return fmt.Errorf("stride must be in [1,context-len]")
	}
	if cfg.ModelConfig == nil {
		return fmt.Errorf("model config is required")
	}
	if cfg.ContextLen > cfg.ModelConfig.ContextLen {
		return fmt.Errorf("context-len %d exceeds model context %d", cfg.ContextLen, cfg.ModelConfig.ContextLen)
	}
	if cfg.MaxFrames < 0 {
		return fmt.Errorf("max-frames must be >= 0")
	}
	if cfg.OutPath == "" {
		cfg.OutPath = filepath.Join(cfg.CheckpointDir, "model-latest")
	}
	return nil
}

func Run(ctx context.Context, cfg *Config) (*Summary, error) {
	eng := hands.NewEngine(cfg.ModelConfig, cfg.VocabPath)
	if cfg.ResumeFrom != "" {
		if err := eng.Load(cfg.ResumeFrom); err != nil {
			return nil, fmt.Errorf("resume %s: %w", cfg.ResumeFrom, err)
		}
	}

	if cfg.InputPath != "" {
		return runNRV(ctx, cfg, eng)
	}
	return runJSONL(cfg, eng)
}

func runNRV(ctx context.Context, cfg *Config, eng *hands.Engine) (*Summary, error) {
	trainer := &nrvtrainer.HandsTrainer{Fn: func(_ context.Context, windows [][]int, opts nrvtrainer.HandsOptions) (nrvtrainer.HandsStats, error) {
		result, err := eng.TrainWindows(windows, hands.TrainOptions{
			Epochs:       opts.Epochs,
			LearningRate: opts.LearningRate,
			MaxSteps:     opts.MaxSteps,
			SavePath:     cfg.OutPath,
		})
		return nrvtrainer.HandsStats{
			Windows:    result.Windows,
			Steps:      result.Steps,
			Checkpoint: result.Checkpoint,
		}, err
	}}
	dispatcher := nrvtrainer.Default(trainer, nil)
	result, err := dispatcher.Train(ctx, cfg.InputPath, nrvtrainer.Options{
		Targets:   []nrvtrainer.Target{nrvtrainer.TargetHANDS},
		MaxFrames: cfg.MaxFrames,
		Hands: nrvtrainer.HandsOptions{
			Epochs:       cfg.Epochs,
			LearningRate: cfg.LearningRate,
			MaxSteps:     cfg.MaxSteps,
			ContextLen:   cfg.ContextLen,
			Stride:       cfg.Stride,
			VocabSize:    cfg.ModelConfig.VocabSize,
		},
	})
	if err != nil {
		return nil, err
	}
	summary := &Summary{
		Input:          cfg.InputPath,
		Format:         string(result.Asset.Format),
		Frames:         result.Asset.Frames,
		ValidFrames:    result.Asset.ValidFrames,
		SkippedInvalid: result.Asset.SkippedInvalid,
	}
	for _, target := range result.Targets {
		if target.Target == nrvtrainer.TargetHANDS {
			summary.Windows = target.Windows
			summary.Steps = target.Steps
			for _, detail := range strings.Split(target.Detail, ",") {
				if value, ok := strings.CutPrefix(strings.TrimSpace(detail), "checkpoint "); ok {
					summary.Checkpoint = value
				}
			}
		}
	}
	if summary.Checkpoint == "" && fileExists(cfg.OutPath) {
		summary.Checkpoint = cfg.OutPath
	}
	return summary, nil
}

func runJSONL(cfg *Config, eng *hands.Engine) (*Summary, error) {
	windows, err := readJSONLWindows(cfg.JSONLPath, cfg.ModelConfig.VocabSize)
	if err != nil {
		return nil, err
	}
	if len(windows) > cfg.MaxFrames && cfg.MaxFrames > 0 {
		windows = windows[:cfg.MaxFrames]
	}
	result, err := eng.TrainWindows(windows, hands.TrainOptions{
		Epochs:       cfg.Epochs,
		LearningRate: cfg.LearningRate,
		MaxSteps:     cfg.MaxSteps,
		SavePath:     cfg.OutPath,
	})
	if err != nil {
		return nil, err
	}
	return &Summary{
		Input:      cfg.JSONLPath,
		Format:     "jsonl",
		Windows:    result.Windows,
		Steps:      result.Steps,
		Checkpoint: result.Checkpoint,
	}, nil
}

// readJSONLWindows accepts the legacy corpus shapes in circulation: a bare
// token array, {"token_ids": [...]}, {"tokens": [...]} or {"window": [...]}.
func readJSONLWindows(path string, vocabSize int) ([][]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var windows [][]int
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var tokens []int
		if strings.HasPrefix(raw, "[") {
			if err := json.Unmarshal([]byte(raw), &tokens); err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
		} else {
			var record struct {
				TokenIDs []int `json:"token_ids"`
				Tokens   []int `json:"tokens"`
				Window   []int `json:"window"`
			}
			if err := json.Unmarshal([]byte(raw), &record); err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			switch {
			case len(record.TokenIDs) > 0:
				tokens = record.TokenIDs
			case len(record.Tokens) > 0:
				tokens = record.Tokens
			default:
				tokens = record.Window
			}
		}
		if len(tokens) < 2 {
			return nil, fmt.Errorf("line %d: window needs at least 2 tokens", line)
		}
		for _, token := range tokens {
			if token < 0 || token >= vocabSize {
				return nil, fmt.Errorf("line %d: token %d outside vocabulary [0,%d)", line, token, vocabSize)
			}
		}
		windows = append(windows, tokens)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(windows) == 0 {
		return nil, fmt.Errorf("%s: no token windows", path)
	}
	return windows, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(2)
	}
	summary, err := Run(context.Background(), cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "train:", err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, "summary:", err)
		os.Exit(1)
	}
}
