package hands

import (
	"errors"

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
	path  string
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
	e.path = path
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
func (e *Engine) Path() string { return e.path }

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
