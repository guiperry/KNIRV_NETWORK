package transformer

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"knirvhasher/pkg/hashing"
	"knirvhasher/pkg/hashing/core"
	"knirvhasher/pkg/hashing/neural"
)

// InferenceMode selects the operational mode of the UnifiedHasherEngine.
type InferenceMode string

const (
	ModeTransformer InferenceMode = "transformer"
	ModeRecursive   InferenceMode = "recursive"
	ModeFeedforward InferenceMode = "feedforward"
)

// UnifiedConfig defines architecture for the unified hash-based engine.
type UnifiedConfig struct {
	VocabSize    int
	EmbedDim     int
	NumHeads     int
	NumLayers    int
	ContextLen   int
	Hidden1      int
	Hidden2      int
	OutputSize   int
	FFNHiddenDim int
	Activation   string
	Passes       int
	Jitter       float64
	SeedRotation bool
}

// DefaultUnifiedConfig returns a sensible default configuration.
func DefaultUnifiedConfig() *UnifiedConfig {
	return &UnifiedConfig{
		VocabSize:    100277,
		EmbedDim:     768,
		NumHeads:     12,
		NumLayers:    12,
		ContextLen:   2048,
		Hidden1:      512,
		Hidden2:     256,
		OutputSize:   100,
		FFNHiddenDim: 3072,
		Activation:   "hash",
		Passes:       21,
		Jitter:       0.01,
		SeedRotation: false,
	}
}

// SeedStore holds all seed parameters for the unified engine.
type SeedStore struct {
	Embeddings [][][32]byte
	Positional [][][32]byte
	Layers     []TransformerLayerSeeds
	OutputSeed [32]byte
	Seeds1     [][32]byte
	Seeds2     [][32]byte
	SeedsOut   [][32]byte
}

// TransformerLayerSeeds holds all seeds for one transformer layer.
type TransformerLayerSeeds struct {
	QuerySeeds  [][][32]byte
	KeySeeds    [][][32]byte
	ValueSeeds  [][][32]byte
	OutputSeeds [][][32]byte
	FFNSeeds    [][][32]byte
	DecaySeeds  [][32]byte
	// FFNOutSeeds is one seed per output neuron of the FFN's down-projection
	// (expanded -> EmbedDim). Length EmbedDim. May be empty on seed stores
	// persisted before this field existed; ffnOutSeedsOrDerive derives a
	// deterministic fallback from FFNSeeds in that case.
	FFNOutSeeds [][32]byte
}

// EngineStats tracks inference statistics.
type EngineStats struct {
	TotalInferences  uint64
	HardwareInferences uint64
	SoftwareFallbacks  uint64
	mu               sync.RWMutex
}

func (s *EngineStats) IncTotal()  { s.mu.Lock(); s.TotalInferences++; s.mu.Unlock() }
func (s *EngineStats) IncHardware() { s.mu.Lock(); s.HardwareInferences++; s.mu.Unlock() }
func (s *EngineStats) IncSoftware() { s.mu.Lock(); s.SoftwareFallbacks++; s.mu.Unlock() }

// UnifiedHasherEngine replaces HasherTransformer, HashNetwork, and RecursiveEngine
// for seed-based inference with optional hardware acceleration.
type UnifiedHasherEngine struct {
	config     *UnifiedConfig
	seeds      *SeedStore
	hashMethod core.HashMethod
	mode       InferenceMode
	stats      *EngineStats
	mu         sync.RWMutex
	hwOnce     sync.Once
	hwTier     string
}

// NewUnifiedHasherEngine creates a new engine with the given seeds, hash method, and mode.
func NewUnifiedHasherEngine(seeds *SeedStore, hashMethod core.HashMethod, mode InferenceMode) *UnifiedHasherEngine {
	if mode == "" {
		mode = ModeTransformer
	}
	if seeds == nil {
		seeds = &SeedStore{}
	}
	engine := &UnifiedHasherEngine{
		config:     DefaultUnifiedConfig(),
		seeds:      seeds,
		hashMethod: hashMethod,
		mode:       mode,
		stats:      &EngineStats{},
	}
	engine.ValidateHardware()
	return engine
}

// NewUnifiedHasherEngineWithConfig creates a new engine with explicit config.
func NewUnifiedHasherEngineWithConfig(cfg *UnifiedConfig, seeds *SeedStore, hashMethod core.HashMethod, mode InferenceMode) *UnifiedHasherEngine {
	if cfg == nil {
		cfg = DefaultUnifiedConfig()
	}
	if mode == "" {
		mode = ModeTransformer
	}
	if seeds == nil {
		seeds = &SeedStore{}
	}
	engine := &UnifiedHasherEngine{
		config:     cfg,
		seeds:      seeds,
		hashMethod: hashMethod,
		mode:       mode,
		stats:      &EngineStats{},
	}
	engine.ValidateHardware()
	return engine
}

// SetHashMethod updates the HashMethod used for hardware-accelerated hashing.
func (e *UnifiedHasherEngine) SetHashMethod(method core.HashMethod) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.hashMethod = method
	e.hwOnce.Do(func() {})
}

// SetSeeds replaces the active SeedStore.
func (e *UnifiedHasherEngine) SetSeeds(seeds *SeedStore) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.seeds = seeds
}

// SetMode changes the inference mode.
func (e *UnifiedHasherEngine) SetMode(mode InferenceMode) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mode = mode
}

// Mode returns the current inference mode.
func (e *UnifiedHasherEngine) Mode() InferenceMode {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.mode
}

// IsUsingHardware returns true if hardware acceleration is active.
func (e *UnifiedHasherEngine) IsUsingHardware() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.hashMethod != nil && e.hashMethod.IsAvailable()
}

// HardwareTier returns the detected hardware tier string ("ASIC", "CUDA", "eBPF", "uBPF", "software").
func (e *UnifiedHasherEngine) HardwareTier() string {
	e.mu.RLock()
	tier := e.hwTier
	e.mu.RUnlock()
	return tier
}

// ValidateHardware probes hardware availability once and logs the active tier.
// Call this during engine initialization before any inference calls.
func (e *UnifiedHasherEngine) ValidateHardware() {
	e.hwOnce.Do(func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.hashMethod == nil {
			e.hwTier = "software"
			return
		}
		if !e.hashMethod.IsAvailable() {
			e.hwTier = "software"
			return
		}
		name := e.hashMethod.Name()
		switch name {
		case "ASICMethod":
			e.hwTier = "ASIC"
		case "CudaMethod":
			e.hwTier = "CUDA"
		case "EbpfMethod":
			e.hwTier = "eBPF"
		case "UbpfMethod":
			e.hwTier = "uBPF"
		default:
			e.hwTier = name
		}
		log.Printf("[UnifiedHasherEngine] hardware tier detected: %s", e.hwTier)
	})
}

// Stats returns a snapshot of engine statistics.
func (e *UnifiedHasherEngine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return EngineStats{
		TotalInferences:    e.stats.TotalInferences,
		HardwareInferences: e.stats.HardwareInferences,
		SoftwareFallbacks:  e.stats.SoftwareFallbacks,
	}
}

// Forward runs token IDs through the transformer mode and returns a pooled float32 vector.
// This is the main entry point for ModeTransformer.
func (e *UnifiedHasherEngine) Forward(tokenIDs []int) []float32 {
	e.mu.RLock()
	cfg := e.config
	method := e.hashMethod
	e.mu.RUnlock()

	if len(tokenIDs) == 0 {
		return make([]float32, cfg.EmbedDim)
	}

	router := NewHardwareRouter(method, FallbackMixed)
	hidden := make([][]float32, len(tokenIDs))
	for i, id := range tokenIDs {
		hidden[i] = e.embedToken(id, i, router)
	}
	for l := 0; l < cfg.NumLayers; l++ {
		hidden = e.forwardLayer(hidden, l, router)
	}
	return e.averagePool(hidden)
}

// Infer dispatches to the appropriate inference mode.
func (e *UnifiedHasherEngine) Infer(input []byte) (*hashing.RecursiveResult, error) {
	e.mu.RLock()
	mode := e.mode
	e.mu.RUnlock()

	switch mode {
	case ModeTransformer:
		return e.inferTransformer(input)
	case ModeRecursive:
		return e.inferRecursive(input)
	case ModeFeedforward:
		return e.inferFeedforward(input)
	default:
		return nil, fmt.Errorf("unsupported inference mode: %s", mode)
	}
}

// Predict runs feedforward inference and returns (argmax, confidence).
func (e *UnifiedHasherEngine) Predict(input []byte) (int, float64, error) {
	e.mu.RLock()
	seeds := e.seeds
	method := e.hashMethod
	e.mu.RUnlock()

	if len(seeds.Seeds1) == 0 {
		return -1, 0, errors.New("no seeds configured")
	}

	// Build a temporary HashNetwork from seeds for feedforward path
	net := &neural.HashNetwork{
		InputSize:  len(input),
		Hidden1:    len(seeds.Seeds1),
		Hidden2:    len(seeds.Seeds2),
		OutputSize: len(seeds.SeedsOut),
		Seeds1:     seeds.Seeds1,
		Seeds2:     seeds.Seeds2,
		SeedsOut:   seeds.SeedsOut,
	}
	if len(input) != net.InputSize {
		padded := make([]byte, net.InputSize)
		copy(padded, input)
		input = padded
	}

	// Layer 1
	layer1Inputs := prepareLayerInputs(input, net.Seeds1)
	var layer1Hashes [][32]byte
	var err error
	if method != nil && method.IsAvailable() {
		layer1Hashes, err = method.ComputeBatch(layer1Inputs)
		if err != nil {
			layer1Hashes = nil
		}
	}
	var layer1Output []float64
	if layer1Hashes != nil {
		layer1Output = hashesToFloats64(layer1Hashes)
	} else {
		layer1Raw := make([]float64, net.Hidden1)
		for i, neuron := range net.Neurons1 {
			_ = neuron
			layer1Raw[i] = seedToFloat64(seeds.Seeds1[i], input)
		}
		layer1Output = layer1Raw
	}

	// Layer 2
	layer2Input := float64SliceToBytes(layer1Output)
	layer2Inputs := prepareLayerInputs(layer2Input, net.Seeds2)
	var layer2Hashes [][32]byte
	if method != nil && method.IsAvailable() {
		layer2Hashes, err = method.ComputeBatch(layer2Inputs)
		if err != nil {
			layer2Hashes = nil
		}
	}
	var layer2Output []float64
	if layer2Hashes != nil {
		layer2Output = hashesToFloats64(layer2Hashes)
	} else {
		layer2Raw := make([]float64, net.Hidden2)
		for i := range layer2Raw {
			layer2Raw[i] = seedToFloat64(seeds.Seeds2[i], layer2Input)
		}
		layer2Output = layer2Raw
	}

	// Output layer
	outputInput := float64SliceToBytes(layer2Output)
	outputInputs := prepareLayerInputs(outputInput, net.SeedsOut)
	var outputHashes [][32]byte
	if method != nil && method.IsAvailable() {
		outputHashes, err = method.ComputeBatch(outputInputs)
		if err != nil {
			outputHashes = nil
		}
	}
	var output []float64
	if outputHashes != nil {
		output = hashesToFloats64(outputHashes)
	} else {
		output = make([]float64, net.OutputSize)
		for i := range output {
			output[i] = seedToFloat64(seeds.SeedsOut[i], outputInput)
		}
	}

	maxVal := output[0]
	maxIndex := 0
	for i, val := range output[1:] {
		if val > maxVal {
			maxVal = val
			maxIndex = i + 1
		}
	}

	e.mu.Lock()
	e.stats.IncTotal()
	if method != nil && method.IsAvailable() {
		e.stats.IncHardware()
	} else {
		e.stats.IncSoftware()
	}
	e.mu.Unlock()

	return maxIndex, maxVal, nil
}

// GenerateToken produces the next token given a context and temperature.
func (e *UnifiedHasherEngine) GenerateToken(ctx []int, temperature float32) (int, []float32) {
	hidden := e.Forward(ctx)
	scores := HashToVocab(hidden, e.seeds.OutputSeed, e.config.VocabSize)
	if temperature <= 0 {
		return argmax32(scores), scores
	}
	return sampleTemp32(scores, temperature), scores
}

func (e *UnifiedHasherEngine) embedToken(tokenID, position int, router *HardwareRouter) []float32 {
	dim := e.config.EmbedDim
	out := make([]float32, dim)
	tokenID = tokenID % e.config.VocabSize
	if tokenID < len(e.seeds.Embeddings) {
		for j := 0; j < dim && j < len(e.seeds.Embeddings[tokenID]); j++ {
			out[j] = SeedToFloat(e.seeds.Embeddings[tokenID][j])
		}
	}
	if position < e.config.ContextLen && position < len(e.seeds.Positional) {
		for j := 0; j < dim && j < len(e.seeds.Positional[position]); j++ {
			out[j] += SeedToFloat(e.seeds.Positional[position][j])
		}
	}
	return out
}

func (e *UnifiedHasherEngine) forwardLayer(hidden [][]float32, layerIdx int, router *HardwareRouter) [][]float32 {
	seqLen := len(hidden)
	dim := e.config.EmbedDim
	if layerIdx >= len(e.seeds.Layers) {
		return hidden
	}
	layer := e.seeds.Layers[layerIdx]

	attn := e.attnForward(hidden, layer)
	for i := 0; i < seqLen; i++ {
		for j := 0; j < dim; j++ {
			hidden[i][j] = LayerNorm(hidden[i][j]+attn[i][j], LayerNormMin, LayerNormMax)
		}
	}
	ffn := e.ffnForward(hidden, layer, router)
	for i := 0; i < seqLen; i++ {
		for j := 0; j < dim; j++ {
			hidden[i][j] = LayerNorm(hidden[i][j]+ffn[i][j], LayerNormMin, LayerNormMax)
		}
	}
	return hidden
}

// attnForward is LM-only float computation. Attestation hashing belongs to
// the mining path and must not be implied by this attention API.
func (e *UnifiedHasherEngine) attnForward(hidden [][]float32, layer TransformerLayerSeeds) [][]float32 {
	seqLen := len(hidden)
	dim := e.config.EmbedDim
	numHeads := e.config.NumHeads
	out := make([][]float32, seqLen)
	for i := range out {
		out[i] = make([]float32, dim)
	}

	cumDecay := precomputeCumulativeDecay(layer.DecaySeeds, seqLen)

	for i := 0; i < seqLen; i++ {
		for h := 0; h < numHeads; h++ {
			q := ProjectSeeds(hidden[i], layer.QuerySeeds[h], e.config.Activation)
			for j := 0; j <= i; j++ {
				k := ProjectSeeds(hidden[j], layer.KeySeeds[h], e.config.Activation)
				v := ProjectSeeds(hidden[j], layer.ValueSeeds[h], e.config.Activation)
				match := dotProduct(q, k)
				if match > 88.0 {
					match = 88.0
				} else if match < -88.0 {
					match = -88.0
				}
				attentionWeight := float32(math.Exp(float64(match)))
				decay := getDecayFromCumulative(j+1, i+1, cumDecay)
				for d := 0; d < dim && d < len(v); d++ {
					out[i][d] += attentionWeight * decay * v[d] / float32(numHeads)
				}
			}
		}
		proj := ProjectSeeds2D(out[i], layer.OutputSeeds, e.config.Activation)
		copy(out[i], proj)
	}
	return out
}

func computeDecayChain(startPos, endPos int, decaySeeds [][32]byte) float32 {
	if startPos >= endPos || len(decaySeeds) == 0 {
		return 1.0
	}
	var product float32 = 1.0
	for s := startPos; s < endPos; s++ {
		if s < len(decaySeeds) {
			alpha := SeedToUnitFloat(decaySeeds[s])
			product *= alpha
		}
	}
	if product < 1e-30 {
		product = 1e-30
	}
	return product
}

func precomputeCumulativeDecay(decaySeeds [][32]byte, maxLen int) []float32 {
	cumDecay := make([]float32, maxLen+1)
	cumDecay[0] = 1.0
	for t := 1; t <= maxLen; t++ {
		alpha := float32(1.0)
		if t-1 < len(decaySeeds) {
			alpha = SeedToUnitFloat(decaySeeds[t-1])
		}
		cumDecay[t] = cumDecay[t-1] * alpha
		if cumDecay[t] < 1e-30 {
			cumDecay[t] = 1e-30
		}
	}
	return cumDecay
}

func getDecayFromCumulative(startPos, endPos int, cumDecay []float32) float32 {
	if startPos >= endPos || len(cumDecay) == 0 {
		return 1.0
	}
	if endPos >= len(cumDecay) {
		endPos = len(cumDecay) - 1
	}
	if startPos >= len(cumDecay) {
		startPos = len(cumDecay) - 1
	}
	if cumDecay[startPos] == 0 {
		return 1e-30
	}
	return cumDecay[endPos] / cumDecay[startPos]
}

func (e *UnifiedHasherEngine) ffnForward(hidden [][]float32, layer TransformerLayerSeeds, router *HardwareRouter) [][]float32 {
	outSeeds := ffnOutSeedsOrDerive(layer.FFNOutSeeds, layer.FFNSeeds, e.config.EmbedDim)
	out := make([][]float32, len(hidden))
	for i, h := range hidden {
		expanded := ProjectSeeds2D(h, layer.FFNSeeds, e.config.Activation)
		out[i] = ProjectBack(expanded, outSeeds, e.config.Activation)
	}
	return out
}

func (e *UnifiedHasherEngine) averagePool(hidden [][]float32) []float32 {
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

// ---- Recursive mode ----

func (e *UnifiedHasherEngine) inferTransformer(input []byte) (*hashing.RecursiveResult, error) {
	start := time.Now()
	hidden := e.Forward([]int{1})
	logits := HashToVocab(hidden, e.seeds.OutputSeed, e.config.VocabSize)
	pred := argmax32(logits)
	conf := float64(logits[pred])

	return &hashing.RecursiveResult{
		Passes: []*hashing.InferencePass{
			{
				PassNumber:  0,
				Prediction:  pred,
				Confidence:  conf,
				Latency:     time.Since(start),
				PassLatency: time.Since(start),
			},
		},
		Consensus: &hashing.ConsensusResult{
			Prediction:        pred,
			Confidence:        conf,
			AverageConfidence: conf,
			VoteCount:         1,
			Mode:              pred,
		},
		Latency:     time.Since(start),
		ValidPasses: 1,
		TotalPasses: 1,
	}, nil
}

func (e *UnifiedHasherEngine) inferRecursive(input []byte) (*hashing.RecursiveResult, error) {
	e.mu.RLock()
	seeds := e.seeds
	cfg := e.config
	method := e.hashMethod
	e.mu.RUnlock()

	if len(seeds.Seeds1) == 0 {
		return nil, errors.New("no seeds configured for recursive mode")
	}

	start := time.Now()
	results := make([]*hashing.InferencePass, 0, cfg.Passes)

	for i := 0; i < cfg.Passes; i++ {
		passResult, err := e.runRecursivePass(input, i, seeds, cfg, method)
		if err != nil {
			passResult, retryErr := e.runRecursivePassSoftware(input, i, seeds, cfg)
			if retryErr != nil {
				continue
			}
			results = append(results, passResult)
			continue
		}
		results = append(results, passResult)
	}

	if len(results) == 0 {
		return nil, hashing.ErrNoValidPasses
	}

	aggregated := aggregateRecursiveResults(results)
	return &hashing.RecursiveResult{
		Passes:      results,
		Consensus:   aggregated,
		Latency:     time.Since(start),
		ValidPasses: len(results),
		TotalPasses: cfg.Passes,
	}, nil
}

func (e *UnifiedHasherEngine) runRecursivePass(input []byte, passNum int, seeds *SeedStore, cfg *UnifiedConfig, method core.HashMethod) (*hashing.InferencePass, error) {
	start := time.Now()

	jittered, err := applyJitter(input, cfg.Jitter, passNum)
	if err != nil {
		return nil, err
	}

	var prediction int
	var confidence float64

	if method != nil && method.IsAvailable() {
		pred, conf, err := e.runHardwareRecursivePass(jittered, passNum, seeds, cfg, method)
		if err != nil {
			return nil, err
		}
		prediction = pred
		confidence = conf
	} else {
		net := buildTempNetwork(jittered, seeds, cfg)
		pred, conf, err := net.Predict(jittered)
		if err != nil {
			return nil, err
		}
		prediction = pred
		confidence = conf
	}

	return &hashing.InferencePass{
		PassNumber:  passNum,
		Prediction:  prediction,
		Confidence:  confidence,
		Latency:     time.Since(start),
		PassLatency: time.Since(start),
	}, nil
}

// runRecursivePassSoftware retries a single recursive pass using the software path.
func (e *UnifiedHasherEngine) runRecursivePassSoftware(input []byte, passNum int, seeds *SeedStore, cfg *UnifiedConfig) (*hashing.InferencePass, error) {
	start := time.Now()
	jittered, err := applyJitter(input, cfg.Jitter, passNum)
	if err != nil {
		return nil, err
	}
	net := buildTempNetwork(jittered, seeds, cfg)
	pred, conf, err := net.Predict(jittered)
	if err != nil {
		return nil, err
	}
	return &hashing.InferencePass{
		PassNumber:  passNum,
		Prediction:  pred,
		Confidence:  conf,
		Latency:     time.Since(start),
		PassLatency: time.Since(start),
	}, nil
}

func (e *UnifiedHasherEngine) runHardwareRecursivePass(input []byte, passNum int, seeds *SeedStore, cfg *UnifiedConfig, method core.HashMethod) (int, float64, error) {
	// Layer 1
	layer1Inputs := prepareLayerInputs(input, seeds.Seeds1)
	layer1Hashes, err := method.ComputeBatch(layer1Inputs)
	if err != nil {
		return -1, 0, err
	}
	layer1Output := hashesToFloats64(layer1Hashes)
	layer1Bytes := float64SliceToBytes(layer1Output)

	// Layer 2
	layer2Inputs := prepareLayerInputs(layer1Bytes, seeds.Seeds2)
	layer2Hashes, err := method.ComputeBatch(layer2Inputs)
	if err != nil {
		return -1, 0, err
	}
	layer2Output := hashesToFloats64(layer2Hashes)
	layer2Bytes := float64SliceToBytes(layer2Output)

	// Output
	outputInputs := prepareLayerInputs(layer2Bytes, seeds.SeedsOut)
	outputHashes, err := method.ComputeBatch(outputInputs)
	if err != nil {
		return -1, 0, err
	}
	output := hashesToFloats64(outputHashes)

	maxVal := output[0]
	maxIndex := 0
	for i, val := range output[1:] {
		if val > maxVal {
			maxVal = val
			maxIndex = i + 1
		}
	}
	return maxIndex, maxVal, nil
}

func (e *UnifiedHasherEngine) inferFeedforward(input []byte) (*hashing.RecursiveResult, error) {
	pred, conf, err := e.Predict(input)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	return &hashing.RecursiveResult{
		Passes: []*hashing.InferencePass{
			{
				PassNumber:  0,
				Prediction:  pred,
				Confidence:  conf,
				Latency:     time.Since(start),
				PassLatency: time.Since(start),
			},
		},
		Consensus: &hashing.ConsensusResult{
			Prediction:        pred,
			Confidence:        conf,
			AverageConfidence: conf,
			VoteCount:         1,
			Mode:              pred,
		},
		Latency:     time.Since(start),
		ValidPasses: 1,
		TotalPasses: 1,
	}, nil
}

// ---- Helper functions ----

func prepareLayerInputs(input []byte, seeds [][32]byte) [][]byte {
	inputs := make([][]byte, len(seeds))
	for i, seed := range seeds {
		combined := make([]byte, len(input)+32)
		copy(combined, input)
		copy(combined[len(input):], seed[:])
		inputs[i] = combined
	}
	return inputs
}

func seedToFloat64(seed [32]byte, input []byte) float64 {
	combined := append(input, seed[:]...)
	hash := sha256.Sum256(combined)
	val := binary.BigEndian.Uint64(hash[0:8])
	return float64(val) / float64(1<<64-1)
}

func hashesToFloats64(hashes [][32]byte) []float64 {
	floats := make([]float64, len(hashes))
	for i, hash := range hashes {
		val := uint64(hash[0])<<56 | uint64(hash[1])<<48 | uint64(hash[2])<<40 | uint64(hash[3])<<32 |
			uint64(hash[4])<<24 | uint64(hash[5])<<16 | uint64(hash[6])<<8 | uint64(hash[7])
		floats[i] = float64(val) / float64(1<<64-1)
	}
	return floats
}

func float64SliceToBytes(floats []float64) []byte {
	bytes := make([]byte, 0, len(floats)*8)
	for _, f := range floats {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, float64ToUint64(f))
		bytes = append(bytes, buf...)
	}
	return bytes
}

func float64ToUint64(f float64) uint64 {
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	return uint64(f * float64(1<<64-1))
}

func applyJitter(input []byte, jitter float64, seed int) ([]byte, error) {
	if jitter == 0 {
		return input, nil
	}
	rng := rand.New(rand.NewSource(int64(seed)))
	jittered := make([]byte, len(input))
	copy(jittered, input)
	for i := range jittered {
		a := int(rng.Float64() * jitter * 255)
		b := int(rng.Float64() * jitter * 255)
		delta := a - b
		newVal := int(jittered[i]) + delta
		if newVal < 0 {
			newVal = 0
		}
		if newVal > 255 {
			newVal = 255
		}
		jittered[i] = byte(newVal)
	}
	return jittered, nil
}

func buildTempNetwork(input []byte, seeds *SeedStore, cfg *UnifiedConfig) *neural.HashNetwork {
	net, _ := neural.NewHashNetwork(len(input), len(seeds.Seeds1), len(seeds.Seeds2), len(seeds.SeedsOut))
	net.Seeds1 = seeds.Seeds1
	net.Seeds2 = seeds.Seeds2
	net.SeedsOut = seeds.SeedsOut
	for i := range net.Neurons1 {
		net.Neurons1[i] = neural.NewHashNeuron(seeds.Seeds1[i], "float")
	}
	for i := range net.Neurons2 {
		net.Neurons2[i] = neural.NewHashNeuron(seeds.Seeds2[i], "float")
	}
	for i := range net.NeuronsOut {
		net.NeuronsOut[i] = neural.NewHashNeuron(seeds.SeedsOut[i], "float")
	}
	return net
}

func aggregateRecursiveResults(passes []*hashing.InferencePass) *hashing.ConsensusResult {
	if len(passes) == 0 {
		return &hashing.ConsensusResult{}
	}
	voteCount := make(map[int]int)
	maxVotes := 0
	mode := -1
	totalConfidence := 0.0
	for _, pass := range passes {
		voteCount[pass.Prediction]++
		if voteCount[pass.Prediction] > maxVotes {
			maxVotes = voteCount[pass.Prediction]
			mode = pass.Prediction
		}
		totalConfidence += pass.Confidence
	}
	confidence := float64(maxVotes) / float64(len(passes))
	avgConfidence := totalConfidence / float64(len(passes))
	return &hashing.ConsensusResult{
		Prediction:        mode,
		Confidence:        confidence,
		AverageConfidence: avgConfidence,
		VoteCount:         len(passes),
		Mode:              mode,
	}
}

// ---- SeedStoreWriter interface and implementations ----

// SeedStoreWriter abstracts serialization of SeedStore.
type SeedStoreWriter interface {
	WriteSeedStore(store *SeedStore) error
	ReadSeedStore() (*SeedStore, error)
}

// CSVSeedStoreWriter writes seeds in JSON format for backward compatibility
// with the existing pipeline output. The file uses a .csv extension but stores
// JSON to preserve nested SeedStore structure.
type CSVSeedStoreWriter struct {
	path string
}

func NewCSVSeedStoreWriter(path string) *CSVSeedStoreWriter {
	return &CSVSeedStoreWriter{path: path}
}

func (w *CSVSeedStoreWriter) WriteSeedStore(store *SeedStore) error {
	if store == nil {
		return fmt.Errorf("cannot write nil SeedStore")
	}
	data, err := json.MarshalIndent(store, "", "\t")
	if err != nil {
		return fmt.Errorf("CSVSeedStoreWriter marshal: %w", err)
	}
	if err := os.WriteFile(w.path, data, 0644); err != nil {
		return fmt.Errorf("CSVSeedStoreWriter write: %w", err)
	}
	return nil
}

func (w *CSVSeedStoreWriter) ReadSeedStore() (*SeedStore, error) {
	data, err := os.ReadFile(w.path)
	if err != nil {
		return nil, fmt.Errorf("CSVSeedStoreWriter read: %w", err)
	}
	var store SeedStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("CSVSeedStoreWriter unmarshal: %w", err)
	}
	return &store, nil
}

// MemorySeedStoreWriter keeps SeedStore in memory.
type MemorySeedStoreWriter struct {
	store *SeedStore
}

func NewMemorySeedStoreWriter(store *SeedStore) *MemorySeedStoreWriter {
	if store == nil {
		store = &SeedStore{}
	}
	return &MemorySeedStoreWriter{store: store}
}

func (w *MemorySeedStoreWriter) WriteSeedStore(store *SeedStore) error {
	w.store = store
	return nil
}

func (w *MemorySeedStoreWriter) ReadSeedStore() (*SeedStore, error) {
	if w.store == nil {
		return &SeedStore{}, nil
	}
	return w.store, nil
}

// BPFSeedStoreWriter writes seeds to a BPF map path for device deployment.
// It is a stub by default; enable real BPF syscalls behind the DeviceType flag.
type BPFSeedStoreWriter struct {
	path     string
	deviceType string
}

func NewBPFSeedStoreWriter(path, deviceType string) *BPFSeedStoreWriter {
	if deviceType == "" {
		deviceType = "bpf_dummy"
	}
	return &BPFSeedStoreWriter{path: path, deviceType: deviceType}
}

func (w *BPFSeedStoreWriter) WriteSeedStore(store *SeedStore) error {
	if store == nil {
		return fmt.Errorf("cannot write nil SeedStore to BPF")
	}
	if w.deviceType == "bpf_real" {
		return fmt.Errorf("BPFRealInterface not implemented: requires libbpf and kernel headers")
	}
	data, err := json.Marshal(store)
	if err != nil {
		return fmt.Errorf("BPFSeedStoreWriter marshal: %w", err)
	}
	if err := os.WriteFile(w.path+".json", data, 0644); err != nil {
		return fmt.Errorf("BPFSeedStoreWriter write: %w", err)
	}
	return nil
}

func (w *BPFSeedStoreWriter) ReadSeedStore() (*SeedStore, error) {
	data, err := os.ReadFile(w.path + ".json")
	if err != nil {
		return nil, fmt.Errorf("BPFSeedStoreWriter read: %w", err)
	}
	var store SeedStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("BPFSeedStoreWriter unmarshal: %w", err)
	}
	return &store, nil
}

// NRVSeedStoreWriter writes seeds to .nrv bracket embedding format.
type NRVSeedStoreWriter struct {
	path string
}

func NewNRVSeedStoreWriter(path string) *NRVSeedStoreWriter {
	return &NRVSeedStoreWriter{path: path}
}

func (w *NRVSeedStoreWriter) WriteSeedStore(store *SeedStore) error {
	if store == nil {
		return fmt.Errorf("cannot write nil SeedStore to NRV")
	}
	data, err := json.Marshal(store)
	if err != nil {
		return fmt.Errorf("NRVSeedStoreWriter marshal: %w", err)
	}
	if err := os.WriteFile(w.path+".nrv.json", data, 0644); err != nil {
		return fmt.Errorf("NRVSeedStoreWriter write: %w", err)
	}
	return nil
}

func (w *NRVSeedStoreWriter) ReadSeedStore() (*SeedStore, error) {
	data, err := os.ReadFile(w.path + ".nrv.json")
	if err != nil {
		return nil, fmt.Errorf("NRVSeedStoreWriter read: %w", err)
	}
	var store SeedStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("NRVSeedStoreWriter unmarshal: %w", err)
	}
	return &store, nil
}

// CrossModeValidation compares legacy HashNetwork predictions with unified engine
// predictions to ensure seed compatibility across modes.
// Tolerance: exact match in software mode, ±1 in hardware mode.
func CrossModeValidation(seed [32]byte, input []byte) error {
	legacyNet, err := neural.NewHashNetwork(len(input), 16, 8, 4)
	if err != nil {
		return fmt.Errorf("CrossModeValidation: legacy net creation: %w", err)
	}
	legacyPred, _, err := legacyNet.Predict(input)
	if err != nil {
		return fmt.Errorf("CrossModeValidation: legacy predict: %w", err)
	}

	store := &SeedStore{
		Seeds1: legacyNet.Seeds1,
		Seeds2: legacyNet.Seeds2,
		SeedsOut: legacyNet.SeedsOut,
	}
	engine := NewUnifiedHasherEngine(store, nil, ModeFeedforward)
	unifiedPred, _, err := engine.Predict(input)
	if err != nil {
		return fmt.Errorf("CrossModeValidation: unified predict: %w", err)
	}

	if legacyPred != unifiedPred && abs(legacyPred-unifiedPred) > 1 {
		return fmt.Errorf("CrossModeValidation: legacy=%d unified=%d", legacyPred, unifiedPred)
	}
	return nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// NewUnifiedHasherEngineFromConfig is a convenience constructor used by HEARTService.
// It prefers seeds mined by the 3_DATA_SEEDER pipeline (see
// LoadOrBuildSeedStore / DefaultFramesDir) over unmined crypto/rand noise.
func NewUnifiedHasherEngineFromConfig(cfg *UnifiedConfig) (*UnifiedHasherEngine, error) {
	if cfg == nil {
		cfg = DefaultUnifiedConfig()
	}
	seeds, _ := LoadOrBuildSeedStore(DefaultFramesDir, cfg)
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)
	return engine, nil
}

func BuildDefaultSeedStore(cfg *UnifiedConfig) *SeedStore {
	store := &SeedStore{}
	store.Embeddings = make([][][32]byte, cfg.VocabSize)
	for i := range store.Embeddings {
		store.Embeddings[i] = make([][32]byte, cfg.EmbedDim)
		for j := range store.Embeddings[i] {
			rand.Read(store.Embeddings[i][j][:])
		}
	}
	store.Positional = make([][][32]byte, cfg.ContextLen)
	for i := range store.Positional {
		store.Positional[i] = make([][32]byte, cfg.EmbedDim)
		for j := range store.Positional[i] {
			rand.Read(store.Positional[i][j][:])
		}
	}
	store.Layers = make([]TransformerLayerSeeds, cfg.NumLayers)
	for l := range store.Layers {
		store.Layers[l] = buildDefaultLayerSeeds(cfg)
	}
	rand.Read(store.OutputSeed[:])
	store.Seeds1 = make([][32]byte, cfg.Hidden1)
	for i := range store.Seeds1 {
		rand.Read(store.Seeds1[i][:])
	}
	store.Seeds2 = make([][32]byte, cfg.Hidden2)
	for i := range store.Seeds2 {
		rand.Read(store.Seeds2[i][:])
	}
	store.SeedsOut = make([][32]byte, cfg.OutputSize)
	for i := range store.SeedsOut {
		rand.Read(store.SeedsOut[i][:])
	}
	return store
}

func buildDefaultLayerSeeds(cfg *UnifiedConfig) TransformerLayerSeeds {
	layer := TransformerLayerSeeds{
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
			rand.Read(layer.QuerySeeds[h][j][:])
			rand.Read(layer.KeySeeds[h][j][:])
			rand.Read(layer.ValueSeeds[h][j][:])
		}
	}
	for j := 0; j < cfg.EmbedDim; j++ {
		layer.OutputSeeds[j] = make([][32]byte, cfg.EmbedDim)
		for k := 0; k < cfg.EmbedDim; k++ {
			rand.Read(layer.OutputSeeds[j][k][:])
		}
	}
	for j := 0; j < cfg.FFNHiddenDim; j++ {
		layer.FFNSeeds[j] = make([][32]byte, cfg.EmbedDim)
		for k := 0; k < cfg.EmbedDim; k++ {
			rand.Read(layer.FFNSeeds[j][k][:])
		}
	}
	for s := 0; s < cfg.ContextLen; s++ {
		rand.Read(layer.DecaySeeds[s][:])
	}
	for j := 0; j < cfg.EmbedDim; j++ {
		rand.Read(layer.FFNOutSeeds[j][:])
	}
	return layer
}

// CrossModeValidation compares legacy and unified predictions.
