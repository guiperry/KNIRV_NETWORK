package nrvtrainer

import (
	"context"
	"fmt"
)

// HandsOptions bounds one HANDS training pass over token windows.
type HandsOptions struct {
	Epochs       int
	LearningRate float64
	// MaxSteps 0 derives the step count from the dataset (windows x epochs),
	// the same as the JSONL path; a positive value is a hard cap.
	MaxSteps   int
	ContextLen int
	Stride     int
	// VocabSize, when positive, makes Train reject any token ID outside
	// [0,VocabSize) instead of letting the model wrap it. Zero skips the check.
	VocabSize int
}

const (
	defaultHandsEpochs       = 1
	hardHandsEpochs          = 100
	defaultHandsLearningRate = 3e-4
	hardHandsMaxSteps        = 100_000
	defaultHandsContextLen   = 64
	hardHandsContextLen      = 4096
)

// HandsStats is what the model-side trainer reports back.
type HandsStats struct {
	Windows    int    `json:"windows"`
	Steps      int    `json:"steps"`
	Checkpoint string `json:"checkpoint,omitempty"`
}

// HandsTrainFunc runs gradient descent over token-ID windows. It is supplied by
// pkg/models (production) or a test stand-in, keeping this package free of the
// GPT implementation.
type HandsTrainFunc func(ctx context.Context, windows [][]int, opts HandsOptions) (HandsStats, error)

// HandsTrainer feeds TokenWindow payloads to the HANDS inner GPT.
type HandsTrainer struct {
	Fn HandsTrainFunc
}

func (t *HandsTrainer) Target() Target { return TargetHANDS }

func (t *HandsTrainer) Train(ctx context.Context, asset *Asset, opts Options) (TargetResult, error) {
	if t.Fn == nil {
		return TargetResult{}, fmt.Errorf("hands trainer is not configured")
	}
	hands := normalizeHands(opts.Hands)
	if hands.VocabSize > 0 {
		if err := asset.CheckTokenRange(hands.VocabSize); err != nil {
			return TargetResult{}, fmt.Errorf("refusing to train: %w", err)
		}
	}
	windows := asset.TokenWindows(hands.ContextLen, hands.Stride)
	if len(windows) == 0 {
		return TargetResult{}, fmt.Errorf("asset has no usable token_window payloads (%d valid frame(s))", len(asset.Frames))
	}
	stats, err := t.Fn(ctx, windows, hands)
	if err != nil {
		return TargetResult{}, err
	}
	detail := fmt.Sprintf("%d windows, %d steps", stats.Windows, stats.Steps)
	if stats.Checkpoint != "" {
		detail += ", checkpoint " + stats.Checkpoint
	}
	return TargetResult{
		Frames:  len(asset.Frames),
		Windows: len(windows),
		Steps:   stats.Steps,
		Detail:  detail,
	}, nil
}

func normalizeHands(in HandsOptions) HandsOptions {
	out := in
	if out.Epochs <= 0 {
		out.Epochs = defaultHandsEpochs
	}
	if out.Epochs > hardHandsEpochs {
		out.Epochs = hardHandsEpochs
	}
	if out.LearningRate <= 0 {
		out.LearningRate = defaultHandsLearningRate
	}
	// MaxSteps <= 0 is left as 0: the engine derives steps from the dataset.
	if out.MaxSteps > hardHandsMaxSteps {
		out.MaxSteps = hardHandsMaxSteps
	}
	if out.ContextLen <= 0 {
		out.ContextLen = defaultHandsContextLen
	}
	if out.ContextLen > hardHandsContextLen {
		out.ContextLen = hardHandsContextLen
	}
	if out.Stride <= 0 || out.Stride > out.ContextLen {
		out.Stride = out.ContextLen
	}
	return out
}
