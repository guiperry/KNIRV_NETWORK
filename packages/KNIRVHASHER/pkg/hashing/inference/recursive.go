package inference

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"time"

	"knirvhasher/pkg/hashing"
	"knirvhasher/pkg/hashing/core"
	"knirvhasher/pkg/hashing/neural"
	"knirvhasher/pkg/hashing/transformer"
)

var (
	ErrInvalidNetwork = errors.New("invalid network configuration")
	ErrNoValidPasses  = errors.New("no valid passes completed")
)

// RecursiveEngine implements the recursive single-ASIC inference engine
// as specified in HASHER_SDD.md sections 1.2 and 2.3.
type RecursiveEngine struct {
	Network      *neural.HashNetwork
	hashMethod   core.HashMethod
	Passes       int
	Jitter       float64
	SeedRotation bool
	Mode         hashing.InferenceMode
	Tokenizer    transformer.Tokenizer
}

// NewRecursiveEngine creates a new recursive inference engine with software-only hashing
func NewRecursiveEngine(network *neural.HashNetwork, passes int, jitter float64, seedRotation bool) (*RecursiveEngine, error) {
	return NewRecursiveEngineWithHashMethod(network, nil, passes, jitter, seedRotation)
}

// NewRecursiveEngineWithHashMethod creates a new recursive inference engine with optional HashMethod
func NewRecursiveEngineWithHashMethod(network *neural.HashNetwork, hashMethod core.HashMethod, passes int, jitter float64, seedRotation bool) (*RecursiveEngine, error) {
	if network == nil {
		return nil, ErrInvalidNetwork
	}
	if passes <= 0 {
		passes = 21
	}
	if jitter < 0 || jitter > 1 {
		jitter = 0.01
	}

	return &RecursiveEngine{
		Network:      network,
		hashMethod:   hashMethod,
		Passes:       passes,
		Jitter:       jitter,
		SeedRotation: seedRotation,
		Mode:         hashing.ModeRecursive,
	}, nil
}

// SetHashMethod sets the HashMethod for hardware acceleration
func (e *RecursiveEngine) SetHashMethod(method core.HashMethod) {
	e.hashMethod = method
}

// SetMode sets the inference mode.
func (e *RecursiveEngine) SetMode(mode hashing.InferenceMode) {
	e.Mode = mode
}

// IsUsingHardware returns true if the engine is using hardware acceleration
func (e *RecursiveEngine) IsUsingHardware() bool {
	if e == nil {
		return false
	}
	if e.hashMethod == nil {
		return false
	}
	safeCall := func() (result bool) {
		defer func() {
			if r := recover(); r != nil {
				result = false
			}
		}()
		return e.hashMethod.IsAvailable()
	}
	return safeCall()
}

// Infer performs inference on the given input, dispatching to the active mode.
func (e *RecursiveEngine) Infer(input []byte) (*hashing.RecursiveResult, error) {
	switch e.Mode {
	case hashing.ModeTransformer:
		return e.inferTransformer(input)
	case hashing.ModeRecursive:
		return e.inferRecursive(input)
	case hashing.ModeFeedforward:
		return e.inferFeedforward(input)
	default:
		return e.inferRecursive(input)
	}
}

func (e *RecursiveEngine) inferTransformer(input []byte) (*hashing.RecursiveResult, error) {
	if e.Tokenizer == nil {
		return nil, errors.New("tokenizer required for transformer mode")
	}
	tokenIDs := e.Tokenizer.Encode(string(input))
	cfg := &transformer.UnifiedConfig{
		VocabSize:    100,
		EmbedDim:     32,
		NumHeads:     4,
		NumLayers:    2,
		ContextLen:   64,
		FFNHiddenDim: 64,
		Activation:   "hash",
		Passes:       e.Passes,
		Jitter:       e.Jitter,
		SeedRotation: e.SeedRotation,
	}
	seeds := transformer.BuildDefaultSeedStore(cfg)
	engine := transformer.NewUnifiedHasherEngineWithConfig(cfg, seeds, e.hashMethod, transformer.ModeTransformer)
	hidden := engine.Forward(tokenIDs)
	logits := transformer.HashToVocab(hidden, seeds.OutputSeed, cfg.VocabSize)
	pred := transformer.Argmax32(logits)
	conf := float64(logits[pred])
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

func (e *RecursiveEngine) inferRecursive(input []byte) (*hashing.RecursiveResult, error) {
	start := time.Now()

	results := make([]*hashing.InferencePass, 0, e.Passes)
	for i := 0; i < e.Passes; i++ {
		passResult, err := e.runPass(input, i)
		if err != nil {
			continue
		}
		results = append(results, passResult)
	}

	if len(results) == 0 {
		return nil, ErrNoValidPasses
	}

	aggregated := aggregateResults(results)

	return &hashing.RecursiveResult{
		Passes:      results,
		Consensus:   aggregated,
		Latency:     time.Since(start),
		ValidPasses: len(results),
		TotalPasses: e.Passes,
	}, nil
}

func (e *RecursiveEngine) inferFeedforward(input []byte) (*hashing.RecursiveResult, error) {
	start := time.Now()
	pred, conf, err := e.Network.Predict(input)
	if err != nil {
		return nil, err
	}

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

// runPass executes a single pass of the temporal ensemble
func (e *RecursiveEngine) runPass(input []byte, passNum int) (*hashing.InferencePass, error) {
	start := time.Now()
	passStart := time.Now()

	jitteredInput, err := applyJitter(input, e.Jitter, passNum)
	if err != nil {
		return nil, err
	}

	var prediction int
	var confidence float64

	if e.hashMethod != nil && e.hashMethod.IsAvailable() {
		pred, conf, err := e.runHardwareInference(jitteredInput, passNum)
		if err != nil {
			return nil, err
		}
		prediction = pred
		confidence = conf
	} else {
		if e.SeedRotation {
			tempNet := e.rotateNetworkSeeds(passNum)
			pred, conf, err := tempNet.Predict(jitteredInput)
			if err != nil {
				return nil, err
			}
			prediction = pred
			confidence = conf
		} else {
			pred, conf, err := e.Network.Predict(jitteredInput)
			if err != nil {
				return nil, err
			}
			prediction = pred
			confidence = conf
		}
	}

	return &hashing.InferencePass{
		PassNumber:  passNum,
		Prediction:  prediction,
		Confidence:  confidence,
		Latency:     time.Since(start),
		PassLatency: time.Since(passStart),
	}, nil
}

// runHardwareInference runs a single inference pass using HashMethod
func (e *RecursiveEngine) runHardwareInference(input []byte, passNum int) (int, float64, error) {
	network := e.Network
	if e.SeedRotation {
		network = e.rotateNetworkSeeds(passNum)
	}

	layer1Inputs := prepareLayerInputs(input, network.Seeds1)
	layer1Hashes, err := e.hashMethod.ComputeBatch(layer1Inputs)
	if err != nil {
		return -1, 0, err
	}
	layer1Output := hashesToFloats(layer1Hashes)
	layer1Bytes := floatSliceToBytes(layer1Output)

	layer2Inputs := prepareLayerInputs(layer1Bytes, network.Seeds2)
	layer2Hashes, err := e.hashMethod.ComputeBatch(layer2Inputs)
	if err != nil {
		return -1, 0, err
	}
	layer2Output := hashesToFloats(layer2Hashes)
	layer2Bytes := floatSliceToBytes(layer2Output)

	outputInputs := prepareLayerInputs(layer2Bytes, network.SeedsOut)
	outputHashes, err := e.hashMethod.ComputeBatch(outputInputs)
	if err != nil {
		return -1, 0, err
	}
	output := hashesToFloats(outputHashes)

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

// prepareLayerInputs prepares inputs for a neural network layer
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

// hashesToFloats converts hash outputs to float64 values [0, 1]
func hashesToFloats(hashes [][32]byte) []float64 {
	floats := make([]float64, len(hashes))
	for i, hash := range hashes {
		val := uint64(hash[0])<<56 | uint64(hash[1])<<48 | uint64(hash[2])<<40 | uint64(hash[3])<<32 |
			uint64(hash[4])<<24 | uint64(hash[5])<<16 | uint64(hash[6])<<8 | uint64(hash[7])
		floats[i] = float64(val) / float64(1<<64-1)
	}
	return floats
}

// floatSliceToBytes converts a slice of float64 to a byte slice for hashing
func floatSliceToBytes(floats []float64) []byte {
	bytes := make([]byte, 0, len(floats)*8)
	for _, f := range floats {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, float64ToUint64(f))
		bytes = append(bytes, buf...)
	}
	return bytes
}

// float64ToUint64 converts float64 to uint64 for hashing purposes
func float64ToUint64(f float64) uint64 {
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	return uint64(f * float64(1<<64-1))
}

// applyJitter adds controlled jitter to the input
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

// rotateNetworkSeeds creates a temporary network with rotated seeds for passNum
func (e *RecursiveEngine) rotateNetworkSeeds(passNum int) *neural.HashNetwork {
	tempNet, _ := neural.NewHashNetwork(
		e.Network.InputSize,
		e.Network.Hidden1,
		e.Network.Hidden2,
		e.Network.OutputSize,
	)

	for i := range tempNet.Seeds1 {
		rotateSeed(tempNet.Seeds1[i][:], passNum)
	}
	for i := range tempNet.Seeds2 {
		rotateSeed(tempNet.Seeds2[i][:], passNum)
	}
	for i := range tempNet.SeedsOut {
		rotateSeed(tempNet.SeedsOut[i][:], passNum)
	}

	return tempNet
}

// rotateSeed performs a deterministic seed rotation based on pass number
func rotateSeed(seed []byte, offset int) {
	for i := range seed {
		seed[i] = seed[i] ^ byte((offset+i)%256)
	}
}

// aggregateResults performs temporal consensus on pass results
func aggregateResults(passes []*hashing.InferencePass) *hashing.ConsensusResult {
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

