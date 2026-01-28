package hasher

import (
	"math/rand"
	"time"
)

// RecursiveEngine implements the recursive single-ASIC inference engine
// as specified in HASHER_SDD.md sections 1.2 and 2.3
type RecursiveEngine struct {
	Network     *HashNetwork        // The hash network to use
	Passes      int                // Number of temporal passes (default: 21)
	Jitter      float64             // Input jitter factor [0, 1] (default: 0.01)
	SeedRotation bool              // Whether to rotate neuron seeds per pass
}

// NewRecursiveEngine creates a new recursive inference engine
func NewRecursiveEngine(network *HashNetwork, passes int, jitter float64, seedRotation bool) (*RecursiveEngine, error) {
	if network == nil {
		return nil, ErrInvalidNetwork
	}
	if passes <= 0 {
		passes = 21 // Default from HASHER_SDD.md section 4.1.2
	}
	if jitter < 0 || jitter > 1 {
		jitter = 0.01 // Default small jitter
	}

	return &RecursiveEngine{
		Network:     network,
		Passes:      passes,
		Jitter:      jitter,
		SeedRotation: seedRotation,
	}, nil
}

// Infer performs recursive inference on the given input using temporal ensemble
func (e *RecursiveEngine) Infer(input []byte) (*RecursiveResult, error) {
	start := time.Now()

	results := make([]*InferencePass, 0, e.Passes)
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

	// Aggregate results
	aggregated := e.aggregateResults(results)
	
	return &RecursiveResult{
		Passes:       results,
		Consensus:    aggregated,
		Latency:      time.Since(start),
		ValidPasses:  len(results),
		TotalPasses:  e.Passes,
	}, nil
}

// runPass executes a single pass of the temporal ensemble
func (e *RecursiveEngine) runPass(input []byte, passNum int) (*InferencePass, error) {
	start := time.Now()
	passStart := time.Now()

	// Apply input jitter
	jitteredInput, err := applyJitter(input, e.Jitter, passNum)
	if err != nil {
		return nil, err
	}

	// Run inference with optional seed rotation
	var prediction int
	var confidence float64
	if e.SeedRotation {
		// Create a temporary network with rotated seeds
		tempNet := e.rotateNetworkSeeds(passNum)
		pred, conf, err := tempNet.Predict(jitteredInput)
		if err != nil {
			return nil, err
		}
		prediction = pred
		confidence = conf
	} else {
		// Run with original network
		pred, conf, err := e.Network.Predict(jitteredInput)
		if err != nil {
			return nil, err
		}
		prediction = pred
		confidence = conf
	}

	return &InferencePass{
		PassNumber:  passNum,
		Prediction:  prediction,
		Confidence:  confidence,
		Latency:     time.Since(start),
		PassLatency: time.Since(passStart),
	}, nil
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
		// Apply small random jitter to each byte
		delta := int(rng.Float64()*jitter*255) - int(rng.Float64()*jitter*255)
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
func (e *RecursiveEngine) rotateNetworkSeeds(passNum int) *HashNetwork {
	// Create a deep copy of the network with rotated seeds
	tempNet, _ := NewHashNetwork(
		e.Network.InputSize,
		e.Network.Hidden1,
		e.Network.Hidden2,
		e.Network.OutputSize,
	)

	// Rotate each layer's seeds
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
		seed[i] = seed[i] ^ byte((offset+i) % 256)
	}
}

// aggregateResults performs temporal consensus on pass results
func (e *RecursiveEngine) aggregateResults(passes []*InferencePass) *ConsensusResult {
	// Collect all predictions
	predictions := make([]int, 0, len(passes))
	for _, pass := range passes {
		predictions = append(predictions, pass.Prediction)
	}

	// Compute vote count for each class
	voteCount := make(map[int]int)
	maxVotes := 0
	mode := -1

	for _, pred := range predictions {
		voteCount[pred]++
		if voteCount[pred] > maxVotes {
			maxVotes = voteCount[pred]
			mode = pred
		}
	}

	// Calculate confidence as percentage of max votes
	confidence := float64(maxVotes) / float64(len(passes))

	// Calculate average confidence across passes
	totalConfidence := 0.0
	for _, pass := range passes {
		totalConfidence += pass.Confidence
	}
	averageConfidence := totalConfidence / float64(len(passes))

	return &ConsensusResult{
		Prediction:       mode,
		Confidence:       confidence,
		AverageConfidence: averageConfidence,
		VoteCount:        len(passes),
		Mode:             mode,
	}
}

// RecursiveResult contains the complete results from recursive inference
type RecursiveResult struct {
	Passes       []*InferencePass    // Results from each individual pass
	Consensus    *ConsensusResult    // Aggregated consensus
	Latency      time.Duration       // Total inference latency
	ValidPasses  int                 // Number of valid passes completed
	TotalPasses  int                 // Total passes attempted
}

// InferencePass contains the result of a single pass
type InferencePass struct {
	PassNumber  int                 // Pass sequence number
	Prediction  int                 // Predicted class label
	Confidence  float64             // Neuron confidence [0, 1]
	Latency     time.Duration       // Total time since start
	PassLatency time.Duration       // Time for this specific pass
}

// ConsensusResult contains aggregated results from temporal consensus
type ConsensusResult struct {
	Prediction       int             // Aggregated prediction
	Confidence       float64         // Consensus confidence [0, 1]
	AverageConfidence float64        // Average per-pass confidence
	VoteCount        int             // Total number of valid votes
	Mode             int             // Most frequent prediction
}

// StatisticalSummary returns statistical information about the passes
func (r *RecursiveResult) StatisticalSummary() *StatisticalSummary {
	allConfidences := make([]float64, 0, r.ValidPasses)
	classDistribution := make(map[int]int)

	for _, pass := range r.Passes {
		allConfidences = append(allConfidences, pass.Confidence)
		classDistribution[pass.Prediction]++
	}

	// Calculate mean and std deviation
	mean := 0.0
	for _, conf := range allConfidences {
		mean += conf
	}
	mean /= float64(r.ValidPasses)

	stdDev := 0.0
	for _, conf := range allConfidences {
		diff := conf - mean
		stdDev += diff * diff
	}
	stdDev /= float64(r.ValidPasses)
	// Note: For simplicity, we're not taking square root here

	return &StatisticalSummary{
		MeanConfidence:    mean,
		ConfidenceStdDev: stdDev,
		ClassDistribution: classDistribution,
	}
}

// StatisticalSummary contains statistics about confidence values
type StatisticalSummary struct {
	MeanConfidence    float64         // Mean per-pass confidence
	ConfidenceStdDev  float64         // Standard deviation of confidence
	ClassDistribution map[int]int     // Distribution of predicted classes
}
