package metrics

import (
	"time"
)

// ScalabilityMetrics tracks system performance
type ScalabilityMetrics struct {
	// Clustering metrics
	ClusteringLatencyP50  time.Duration
	ClusteringLatencyP99  time.Duration
	ClusterCount          int
	AvgClusterSize        float64
	
	// Network topology
	NodeCount             int
	EdgeCount             int
	AvgPathLength         float64
	ClusteringCoeff       float64
	
	// DRQ performance
	QValueSyncLatency     time.Duration
	ConvergenceRate       float64
	PhenotypeDrift        float64
	
	// Training throughput
	LoRATrainingTime      time.Duration
	GradientAggregationMS int64
	DVEUtilization        float64
	
	// Consensus performance
	BlockTime             time.Duration
	TxThroughput          float64
	ValidatorCount        int
}
