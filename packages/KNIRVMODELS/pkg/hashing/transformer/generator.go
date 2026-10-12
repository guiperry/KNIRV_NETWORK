package transformer

import (
	"context"
	"log"
)

// WASMSourceGenerator generates TinyGo WASM source code for
// guardrail/resolution/patch modules.  Two implementations exist:
//   - InternalGPTGenerator: uses the Gorgonite GPT model (requires training)
//   - ExternalLLMGenerator: delegates to a 3rd-party LLM via callback/HTTP
type WASMSourceGenerator interface {
	GenerateSource(ctx context.Context, wasmType WASMType, inquiry interface{}) (string, error)
}

// TrainingStateProvider reports whether the internal Gorgonite model
// has been trained enough for production use.
type TrainingStateProvider interface {
	// IsReady returns true when the internal model has been trained
	// to a usable accuracy.
	IsReady() bool
	// Progress returns training progress as a fraction [0.0, 1.0].
	Progress() float64
}

// GeneratorType identifies which WASM generator is currently active.
type GeneratorType int

const (
	GeneratorAuto     GeneratorType = iota // let the switcher decide
	GeneratorInternal                      // force internal Gorgonite GPT
	GeneratorExternal                      // force external LLM
)

// GeneratorSwitcher routes WASM generation between the internal
// Gorgonite transformer and an external LLM, based on the model's
// training state.  This provides a bootstrap path: use the external
// LLM until the internal model is fully trained, then switch over.
type GeneratorSwitcher struct {
	internal WASMSourceGenerator
	external WASMSourceGenerator
	training TrainingStateProvider
	force    GeneratorType
}

// NewGeneratorSwitcher creates a new switcher.  Pass nil for external
// to always use the internal generator.
func NewGeneratorSwitcher(internal, external WASMSourceGenerator, training TrainingStateProvider) *GeneratorSwitcher {
	return &GeneratorSwitcher{
		internal: internal,
		external: external,
		training: training,
	}
}

// SetForce overrides the auto-detection mode.
func (gs *GeneratorSwitcher) SetForce(mode GeneratorType) {
	gs.force = mode
}

// ActiveGenerator returns which generator is currently selected.
func (gs *GeneratorSwitcher) ActiveGenerator() GeneratorType {
	if gs.force == GeneratorInternal || gs.force == GeneratorExternal {
		return gs.force
	}
	if gs.external == nil {
		return GeneratorInternal
	}
	if gs.training != nil && gs.training.IsReady() {
		return GeneratorInternal
	}
	return GeneratorExternal
}

// GenerateSource delegates to the appropriate generator.
func (gs *GeneratorSwitcher) GenerateSource(ctx context.Context, wasmType WASMType, inquiry interface{}) (string, error) {
	active := gs.ActiveGenerator()
	switch active {
	case GeneratorExternal:
		log.Printf("[generator-switcher] using external LLM for %s", wasmType)
		return gs.external.GenerateSource(ctx, wasmType, inquiry)
	default:
		log.Printf("[generator-switcher] using internal GPT for %s", wasmType)
		return gs.internal.GenerateSource(ctx, wasmType, inquiry)
	}
}

// TrainingProgress returns the training progress [0.0, 1.0].
func (gs *GeneratorSwitcher) TrainingProgress() float64 {
	if gs.training == nil {
		return 1.0
	}
	return gs.training.Progress()
}

// InternalGPTGenerator wraps the HEARTService's existing Gorgonite
// pipeline stages into a WASMSourceGenerator.
type InternalGPTGenerator struct {
	service *HEARTService
}

// NewInternalGPTGenerator creates an internal generator backed by the
// HEARTService's Gorgonite stages.
func NewInternalGPTGenerator(svc *HEARTService) *InternalGPTGenerator {
	return &InternalGPTGenerator{service: svc}
}

// GenerateSource runs the full Gorgonite pipeline (stages 1-4) and
// returns the generated TinyGo WASM source code.
func (g *InternalGPTGenerator) GenerateSource(ctx context.Context, wasmType WASMType, inquiry interface{}) (string, error) {
	s1 := g.service.stage1(wasmType, inquiry)
	s2, err := g.service.stage2(ctx, s1, nil)
	if err != nil {
		return "", err
	}
	s3, err := g.service.stage3(ctx, s1, s2)
	if err != nil {
		return "", err
	}
	s4, err := g.service.stage4(ctx, s1, s2, s3)
	if err != nil {
		return "", err
	}
	return s4.Source, nil
}
