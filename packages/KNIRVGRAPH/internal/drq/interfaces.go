package drq

// NetworkTopologyInterface defines the necessary methods from NetworkTopology
// that DRQ components need to interact with.
type NetworkTopologyInterface interface {
	GetClusterLoad(domain string) float64 // This method is used by calculatePriority
	// Add other methods from NetworkTopology that DRQ components need to call
}

// LoRATrainerInterface defines the necessary methods from LoRATrainer
// that DRQ components need to interact with.
type LoRATrainerInterface interface {
	IsComplete(clusterID string) bool
	TrainLoRAAdapter(cluster *ErrorCluster) (*TrainingJob, error)
}
