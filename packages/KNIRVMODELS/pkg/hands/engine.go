package hands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"knirvmodels/pkg/hands/gpt"
	handstok "knirvmodels/pkg/hands/tokenizer"
)

// Engine is the public facade for the HANDS inner GPT. It keeps the gpt package
// internals and the tokenizer wiring behind one type that Phase 2/3 model
// registration code talks to. It is intentionally thin in Phase 1: load,
// generate, and readiness only.
type Engine struct {
	model *gpt.GPT
	tok   handstok.Tokenizer

	// pathMu guards path. TrainWindows (training goroutine) writes it while
	// readers such as the weights handler call Path() concurrently.
	pathMu sync.RWMutex
	path   string
}

// NewEngine builds a HANDS engine from an architecture config. vocabPath is the
// BPE vocab file; pass "" for the embedded defaults tokenizer.
func NewEngine(cfg *gpt.Config, vocabPath string) *Engine {
	e := &Engine{model: gpt.NewGPT(cfg)}
	if vocabPath != "" {
		e.tok = handstok.NewBPETokenizer(vocabPath)
	} else {
		e.tok = handstok.NewBPETokenizer("")
	}
	return e
}

// Load reads a previously trained model from path.
func (e *Engine) Load(path string) error {
	if err := gpt.LoadModel(e.model, path); err != nil {
		return err
	}
	// A loaded checkpoint is a trained model: it is ready to generate.
	e.model.SetTrained(true)
	e.setPath(path)
	return nil
}

// Save writes the current model to path.
func (e *Engine) Save(path string) error { return gpt.SaveModel(e.model, path) }

// Generate autoregressively extends start by n tokens.
func (e *Engine) Generate(start []int, n int, temperature float32, topK int) []int {
	return gpt.Generate(e.model, start, n, temperature, topK)
}

// Model exposes the underlying GPT for callers that need raw forward passes
// (e.g. HEART's Gorgonite inference path in pkg/hashing/transformer).
func (e *Engine) Model() *gpt.GPT { return e.model }

// IsReady reports whether a usable model is loaded/initialized.
func (e *Engine) IsReady() bool { return e.model != nil && e.model.IsReady() }

// Progress returns training progress in [0,1] (0 when untrained).
func (e *Engine) Progress() float64 { return e.model.Progress() }

// Config exposes the architecture in use.
func (e *Engine) Config() *gpt.Config { return e.model.Config() }

// ErrNotReady is returned by generation calls before a model is loaded.
var ErrNotReady = errors.New("hands engine not ready")

// Path returns the checkpoint path the engine last loaded from, if any.
func (e *Engine) Path() string {
	e.pathMu.RLock()
	defer e.pathMu.RUnlock()
	return e.path
}

func (e *Engine) setPath(path string) {
	e.pathMu.Lock()
	defer e.pathMu.Unlock()
	e.path = path
}

// GenerateText encodes prompt, autoregressively extends it by n tokens, and
// decodes the result back to text. Returns ErrNotReady if no model is loaded.
func (e *Engine) GenerateText(prompt string, n int, temperature float32, topK int) (string, error) {
	if e.model == nil || !e.model.IsReady() {
		return "", ErrNotReady
	}
	start := e.tok.Encode(prompt)
	out := gpt.Generate(e.model, start, n, temperature, topK)
	return e.tok.Decode(out), nil
}

// TrainOptions bounds one training pass over pre-tokenised windows.
type TrainOptions struct {
	Epochs       int
	LearningRate float64
	MaxSteps     int
	// SavePath, when set, receives the checkpoint atomically after the pass.
	SavePath string
}

// TrainResult reports what a pass actually did.
type TrainResult struct {
	Windows    int
	Steps      int
	Checkpoint string
}

// TrainWindows runs gradient descent over token-ID windows. The caller is
// responsible for serialising this against generation: TrainWindows mutates the
// model's persisted tensors in place, so a concurrent Forward would observe a
// half-updated weight set.
func (e *Engine) TrainWindows(windows [][]int, opts TrainOptions) (TrainResult, error) {
	if e == nil || e.model == nil {
		return TrainResult{}, ErrNotReady
	}
	dataset := gpt.DatasetFromWindows(windows)
	if dataset.Len() == 0 {
		return TrainResult{}, fmt.Errorf("TrainWindows: no windows with at least two tokens")
	}
	epochs := opts.Epochs
	if epochs <= 0 {
		epochs = 1
	}
	steps := dataset.Len() * epochs
	if opts.MaxSteps > 0 && opts.MaxSteps < steps {
		steps = opts.MaxSteps
	}
	if err := gpt.TrainModel(e.model, dataset, gpt.TrainConfig{
		LearningRate: opts.LearningRate,
		NumEpochs:    epochs,
		BatchSize:    1,
		GradientClip: 1.0,
		MaxSteps:     opts.MaxSteps,
	}); err != nil {
		return TrainResult{}, err
	}
	e.model.SetTrained(true)

	result := TrainResult{Windows: len(windows), Steps: steps}
	if opts.SavePath != "" {
		if err := e.SaveAtomic(opts.SavePath); err != nil {
			return result, err
		}
		result.Checkpoint = opts.SavePath
		e.setPath(opts.SavePath)
	}
	return result, nil
}

// SaveAtomic writes the checkpoint to a temporary sibling and renames it into
// place, so a reader never observes a partially-written model-latest.
func (e *Engine) SaveAtomic(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create checkpoint dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary checkpoint: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := e.Save(tmpPath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("promote checkpoint: %w", err)
	}
	return nil
}
