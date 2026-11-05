package transformer

import (
	crand "crypto/rand"
	"fmt"
	"math"
	"math/rand"

	"knirvmodels/pkg/hashing/core"
)

// ---- HasherTransformer: hash-seed-based transformer (hardware-accelerated path) ----

// HasherTransformerConfig defines architecture for the hash-seed transformer.
// Named HasherTransformerConfig (vs GorgoniteConfig) to distinguish the two implementations.
type HasherTransformerConfig struct {
	VocabSize    int
	EmbedDim     int
	NumLayers    int
	NumHeads     int
	ContextLen   int
	DropoutRate  float32
	FFNHiddenDim int
	Activation   string // "hash", "tanh", "sigmoid"
}

// HasherTransformer implements a transformer whose weights are [32]byte seeds
// operated on via a core.HashMethod for hardware-accelerated inference.
type HasherTransformer struct {
	Config     *HasherTransformerConfig
	Embeddings [][][32]byte
	Positional [][][32]byte
	Layers     []hasherTransformerLayer
	OutputSeed [32]byte
	hashMethod core.HashMethod
}

type hasherTransformerLayer struct {
	QuerySeeds  [][][32]byte
	KeySeeds    [][][32]byte
	ValueSeeds  [][][32]byte
	OutputSeeds [][][32]byte
	FFNSeeds    [][][32]byte
	DecaySeeds  [][32]byte
	FFNOutSeeds [][32]byte
}

// NewHasherTransformer creates a randomly-seeded HasherTransformer.
func NewHasherTransformer(config *HasherTransformerConfig, hashMethod core.HashMethod) *HasherTransformer {
	model := &HasherTransformer{Config: config, hashMethod: hashMethod}

	model.Embeddings = make([][][32]byte, config.VocabSize)
	for i := 0; i < config.VocabSize; i++ {
		model.Embeddings[i] = make([][32]byte, config.EmbedDim)
		for j := 0; j < config.EmbedDim; j++ {
			crand.Read(model.Embeddings[i][j][:])
		}
	}
	model.Positional = make([][][32]byte, config.ContextLen)
	for i := 0; i < config.ContextLen; i++ {
		model.Positional[i] = make([][32]byte, config.EmbedDim)
		for j := 0; j < config.EmbedDim; j++ {
			crand.Read(model.Positional[i][j][:])
		}
	}
	model.Layers = make([]hasherTransformerLayer, config.NumLayers)
	for i := 0; i < config.NumLayers; i++ {
		model.Layers[i] = newHasherLayer(config)
	}
	crand.Read(model.OutputSeed[:])
	return model
}

func newHasherLayer(cfg *HasherTransformerConfig) hasherTransformerLayer {
	layer := hasherTransformerLayer{
		QuerySeeds:  make([][][32]byte, cfg.NumHeads),
		KeySeeds:    make([][][32]byte, cfg.NumHeads),
		ValueSeeds:  make([][][32]byte, cfg.NumHeads),
		OutputSeeds: make([][][32]byte, cfg.EmbedDim),
		FFNSeeds:    make([][][32]byte, cfg.FFNHiddenDim),
		DecaySeeds:  make([][32]byte, cfg.ContextLen),
		FFNOutSeeds: make([][32]byte, cfg.EmbedDim),
	}
	for h := 0; h < cfg.NumHeads; h++ {
		layer.QuerySeeds[h] = make([][32]byte, cfg.EmbedDim)
		layer.KeySeeds[h] = make([][32]byte, cfg.EmbedDim)
		layer.ValueSeeds[h] = make([][32]byte, cfg.EmbedDim)
		for j := 0; j < cfg.EmbedDim; j++ {
			crand.Read(layer.QuerySeeds[h][j][:])
			crand.Read(layer.KeySeeds[h][j][:])
			crand.Read(layer.ValueSeeds[h][j][:])
		}
	}
	for j := 0; j < cfg.EmbedDim; j++ {
		layer.OutputSeeds[j] = make([][32]byte, cfg.EmbedDim)
		for k := 0; k < cfg.EmbedDim; k++ {
			crand.Read(layer.OutputSeeds[j][k][:])
		}
	}
	for j := 0; j < cfg.FFNHiddenDim; j++ {
		layer.FFNSeeds[j] = make([][32]byte, cfg.EmbedDim)
		for k := 0; k < cfg.EmbedDim; k++ {
			crand.Read(layer.FFNSeeds[j][k][:])
		}
	}
	for s := 0; s < cfg.ContextLen; s++ {
		crand.Read(layer.DecaySeeds[s][:])
	}
	for j := 0; j < cfg.EmbedDim; j++ {
		crand.Read(layer.FFNOutSeeds[j][:])
	}
	return layer
}

// SetHashMethod updates the HashMethod used for hardware-accelerated inference.
func (ht *HasherTransformer) SetHashMethod(method core.HashMethod) { ht.hashMethod = method }

// Forward runs token IDs through all layers and returns a pooled float32 vector.
// Uses HardwareRouter for hardware-accelerated projections when a HashMethod is available.
func (ht *HasherTransformer) Forward(tokenIDs []int) []float32 {
	if len(tokenIDs) == 0 {
		return make([]float32, ht.Config.EmbedDim)
	}
	router := NewHardwareRouter(ht.hashMethod, FallbackMixed)
	hidden := make([][]float32, len(tokenIDs))
	for i, id := range tokenIDs {
		hidden[i] = ht.embedToken(id, i, router)
	}
	for l := 0; l < ht.Config.NumLayers; l++ {
		hidden = ht.forwardHasherLayer(hidden, l, router)
	}
	return ht.averagePool(hidden)
}

func (ht *HasherTransformer) embedToken(tokenID, position int, router *HardwareRouter) []float32 {
	dim := ht.Config.EmbedDim
	out := make([]float32, dim)
	tokenID = tokenID % ht.Config.VocabSize
	if tokenID < len(ht.Embeddings) {
		seeds := ht.Embeddings[tokenID]
		proj, err := router.Project(make([]float32, dim), seeds)
		if err == nil {
			copy(out, proj)
		}
	}
	if position < ht.Config.ContextLen && position < len(ht.Positional) {
		seeds := ht.Positional[position]
		add, err := router.Project(make([]float32, dim), seeds)
		if err == nil {
			for j := 0; j < dim; j++ {
				out[j] += add[j]
			}
		}
	}
	return out
}

func (ht *HasherTransformer) forwardHasherLayer(hidden [][]float32, layerIdx int, router *HardwareRouter) [][]float32 {
	seqLen := len(hidden)
	dim := ht.Config.EmbedDim
	layer := ht.Layers[layerIdx]

	attn := ht.hasherMultiHeadAttention(hidden, layer, router)
	for i := 0; i < seqLen; i++ {
		for j := 0; j < dim; j++ {
			hidden[i][j] = ht.hasherLayerNorm(hidden[i][j] + attn[i][j])
		}
	}
	ffn := ht.hasherFFN(hidden, layer, router)
	for i := 0; i < seqLen; i++ {
		for j := 0; j < dim; j++ {
			hidden[i][j] = ht.hasherLayerNorm(hidden[i][j] + ffn[i][j])
		}
	}
	return hidden
}

func (ht *HasherTransformer) hasherMultiHeadAttention(hidden [][]float32, layer hasherTransformerLayer, router *HardwareRouter) [][]float32 {
	seqLen := len(hidden)
	dim := ht.Config.EmbedDim
	out := make([][]float32, seqLen)
	for i := range out {
		out[i] = make([]float32, dim)
	}

	cumDecay := precomputeCumulativeDecay(layer.DecaySeeds, seqLen)

	for i := 0; i < seqLen; i++ {
		for h := 0; h < ht.Config.NumHeads; h++ {
			q := ht.projectSeeds(hidden[i], layer.QuerySeeds[h], router)
			for j := 0; j <= i; j++ {
				k := ht.projectSeeds(hidden[j], layer.KeySeeds[h], router)
				v := ht.projectSeeds(hidden[j], layer.ValueSeeds[h], router)
				match := dotProduct(q, k)
				if match > 88.0 {
					match = 88.0
				} else if match < -88.0 {
					match = -88.0
				}
				attentionWeight := float32(math.Exp(float64(match)))
				decay := getDecayFromCumulative(j+1, i+1, cumDecay)
				for d := 0; d < dim && d < len(v); d++ {
					out[i][d] += attentionWeight * decay * v[d] / float32(ht.Config.NumHeads)
				}
			}
		}
		out[i] = ht.projectSeeds2D(out[i], layer.OutputSeeds, router)
	}
	return out
}

func (ht *HasherTransformer) hasherFFN(hidden [][]float32, layer hasherTransformerLayer, router *HardwareRouter) [][]float32 {
	outSeeds := ffnOutSeedsOrDerive(layer.FFNOutSeeds, layer.FFNSeeds, ht.Config.EmbedDim)
	out := make([][]float32, len(hidden))
	for i, h := range hidden {
		expanded := ht.projectSeeds2D(h, layer.FFNSeeds, router)
		out[i] = ht.projectBack(expanded, outSeeds, router)
	}
	return out
}

func (ht *HasherTransformer) projectSeeds(input []float32, seeds [][32]byte, router *HardwareRouter) []float32 {
	if router != nil {
		if out, err := router.Project(input, seeds); err == nil && len(out) > 0 {
			return out
		}
	}
	return ht.projectSeedsFallback(input, seeds)
}

func (ht *HasherTransformer) projectSeedsFallback(input []float32, seeds [][32]byte) []float32 {
	return ProjectSeeds(input, seeds, ht.Config.Activation)
}

func (ht *HasherTransformer) projectSeeds2D(input []float32, seeds [][][32]byte, router *HardwareRouter) []float32 {
	if router != nil {
		if out, err := router.ProjectBatch2D(input, seeds); err == nil && len(out) > 0 {
			return out
		}
	}
	return ht.projectSeeds2DFallback(input, seeds)
}

func (ht *HasherTransformer) projectSeeds2DFallback(input []float32, seeds [][][32]byte) []float32 {
	return ProjectSeeds2D(input, seeds, ht.Config.Activation)
}

// projectBack projects an FFN-expanded vector back down to len(seeds) output
// dimensions. Earlier this discarded seeds entirely on the software path
// (averaged every input into one scalar and broadcast it to every output —
// see ProjectBack in seed_utils.go for why that was a rank-1 bottleneck), and
// on the "hardware" path treated the raw float32 bit-patterns of a zeroed
// input as fake seeds, which was disconnected from any real seed material.
// Both paths now go through the same per-output-neuron seeds as every other
// projection.
func (ht *HasherTransformer) projectBack(input []float32, seeds [][32]byte, router *HardwareRouter) []float32 {
	if router != nil {
		if out, err := router.Project(input, seeds); err == nil && len(out) > 0 {
			return out
		}
	}
	return ht.projectBackFallback(input, seeds)
}

func (ht *HasherTransformer) projectBackFallback(input []float32, seeds [][32]byte) []float32 {
	return ProjectBack(input, seeds, ht.Config.Activation)
}

func (ht *HasherTransformer) averagePool(hidden [][]float32) []float32 {
	if len(hidden) == 0 {
		return nil
	}
	out := make([]float32, len(hidden[0]))
	for _, row := range hidden {
		for j, v := range row {
			out[j] += v
		}
	}
	n := float32(len(hidden))
	for j := range out {
		out[j] /= n
	}
	return out
}

// seedToFloat delegates to the shared, signed [-1,1] SeedToFloat (seed_utils.go)
// so HasherTransformer and UnifiedHasherEngine can never drift apart on what a
// seed means numerically.
func (ht *HasherTransformer) seedToFloat(seed [32]byte) float32 {
	return SeedToFloat(seed)
}

func (ht *HasherTransformer) activate(x float32) float32 {
	return Activate(x, ht.Config.Activation)
}

func (ht *HasherTransformer) hasherLayerNorm(x float32) float32 {
	if x < -10 {
		return -10
	}
	if x > 10 {
		return 10
	}
	return x
}

// GenerateToken produces the next token given a context and temperature.
func (ht *HasherTransformer) GenerateToken(ctx []int, temperature float32) (int, []float32) {
	hidden := ht.Forward(ctx)
	router := NewHardwareRouter(ht.hashMethod, FallbackMixed)
	scores := router.HashToVocab(hidden, ht.OutputSeed, ht.Config.VocabSize)
	if temperature <= 0 {
		return argmax32(scores), scores
	}
	return sampleTemp32(scores, temperature), scores
}

func argmax32(s []float32) int {
	best := 0
	for i, v := range s {
		if v > s[best] {
			best = i
		}
	}
	return best
}

func sampleTemp32(scores []float32, temp float32) int {
	maxS := scores[0]
	for _, v := range scores {
		if v > maxS {
			maxS = v
		}
	}
	var sum float32
	probs := make([]float32, len(scores))
	for i, v := range scores {
		probs[i] = float32(math.Exp(float64((v - maxS) / temp)))
		sum += probs[i]
	}
	for i := range probs {
		probs[i] /= sum
	}
	var cum float32
	for i, p := range probs {
		cum += p
		if rand.Float32() < cum {
			return i
		}
	}
	return len(probs) - 1
}

// ---- HasherTransformer backward-compatibility wrappers ----

// ForwardWrapper runs Forward() via the unified engine for backward compatibility.
// When hashMethod is non-nil, the unified engine attempts hardware acceleration.
func (ht *HasherTransformer) ForwardWrapper(tokenIDs []int) []float32 {
	engine, err := NewUnifiedHasherEngineFromHasherTransformer(ht)
	if err != nil {
		return ht.Forward(tokenIDs)
	}
	return engine.Forward(tokenIDs)
}

// GenerateTokenWrapper produces the next token via the unified engine.
func (ht *HasherTransformer) GenerateTokenWrapper(ctx []int, temperature float32) (int, []float32) {
	engine, err := NewUnifiedHasherEngineFromHasherTransformer(ht)
	if err != nil {
		return ht.GenerateToken(ctx, temperature)
	}
	return engine.GenerateToken(ctx, temperature)
}

// hasherTransformerConfigToUnified converts a HasherTransformerConfig to the
// equivalent UnifiedConfig.
func hasherTransformerConfigToUnified(cfg *HasherTransformerConfig) *UnifiedConfig {
	return &UnifiedConfig{
		VocabSize:    cfg.VocabSize,
		EmbedDim:     cfg.EmbedDim,
		NumHeads:     cfg.NumHeads,
		NumLayers:    cfg.NumLayers,
		ContextLen:   cfg.ContextLen,
		FFNHiddenDim: cfg.FFNHiddenDim,
		Activation:   cfg.Activation,
		Passes:       21,
		Jitter:       0.01,
		SeedRotation: false,
	}
}

// hasherTransformerToSeedStore extracts ht's seeds into a standalone
// SeedStore, used by NewUnifiedHasherEngineFromHasherTransformer for the
// HasherTransformer → UnifiedHasherEngine conversion path.
func hasherTransformerToSeedStore(ht *HasherTransformer) *SeedStore {
	seeds := &SeedStore{
		Embeddings: ht.Embeddings,
		Positional: ht.Positional,
		Layers:     make([]TransformerLayerSeeds, len(ht.Layers)),
		OutputSeed: ht.OutputSeed,
	}
	for i, l := range ht.Layers {
		seeds.Layers[i] = TransformerLayerSeeds{
			QuerySeeds:  l.QuerySeeds,
			KeySeeds:    l.KeySeeds,
			ValueSeeds:  l.ValueSeeds,
			OutputSeeds: l.OutputSeeds,
			FFNSeeds:    l.FFNSeeds,
			DecaySeeds:  l.DecaySeeds,
			FFNOutSeeds: l.FFNOutSeeds,
		}
	}
	return seeds
}

// NewUnifiedHasherEngineFromHasherTransformer converts a legacy HasherTransformer
// into a UnifiedHasherEngine in transformer mode.
func NewUnifiedHasherEngineFromHasherTransformer(ht *HasherTransformer) (*UnifiedHasherEngine, error) {
	if ht == nil || ht.Config == nil {
		return nil, fmt.Errorf("nil HasherTransformer")
	}
	cfg := hasherTransformerConfigToUnified(ht.Config)
	seeds := hasherTransformerToSeedStore(ht)
	return NewUnifiedHasherEngineWithConfig(cfg, seeds, ht.hashMethod, ModeTransformer), nil
}
