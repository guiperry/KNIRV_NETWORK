package hashing

import (
	"errors"
	"time"
)

// InferenceMode selects the operational mode.
type InferenceMode string

const (
	ModeTransformer InferenceMode = "transformer"
	ModeRecursive   InferenceMode = "recursive"
	ModeFeedforward InferenceMode = "feedforward"
)

var (
	ErrNoValidPasses = errors.New("no valid passes completed")
)

// RecursiveResult contains the complete results from recursive inference.
type RecursiveResult struct {
	Passes      []*InferencePass
	Consensus   *ConsensusResult
	Latency     time.Duration
	ValidPasses int
	TotalPasses int
}

// InferencePass contains the result of a single pass.
type InferencePass struct {
	PassNumber  int
	Prediction  int
	Confidence  float64
	Latency     time.Duration
	PassLatency time.Duration
}

// ConsensusResult contains aggregated results from temporal consensus.
type ConsensusResult struct {
	Prediction        int
	Confidence        float64
	AverageConfidence float64
	VoteCount         int
	Mode              int
}

// StatisticalSummary contains statistics about confidence values.
type StatisticalSummary struct {
	MeanConfidence    float64
	ConfidenceStdDev  float64
	ClassDistribution map[int]int
}
