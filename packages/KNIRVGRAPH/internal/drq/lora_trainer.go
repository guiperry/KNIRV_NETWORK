package drq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// LLMModel is the base model the adapter is trained against.
type LLMModel struct {
	Name       string
	ParamCount int64
}

// LoRATrainingBackend is the actual training execution seam.
//
// This is the boundary the DRQ loop stops at until a real training environment
// exists (GPU/DVE training nodes). Splitting it out is what lets the call graph
// — convergence -> train -> validate -> mint -> reward — be genuinely wired and
// tested today, while the training computation itself remains unimplemented.
// A nil backend produces ErrLoRATrainingBackendNotImplemented; it never
// produces a synthetic "trained" adapter.
type LoRATrainingBackend interface {
	// Train runs job to completion and returns the trained adapter bytes.
	Train(ctx context.Context, job *TrainingJob, data []TrainingExample) ([]byte, error)
	// Name identifies the backend for logs and status reporting.
	Name() string
}

// LoRATrainer handles distributed adapter training.
type LoRATrainer struct {
	baseModel      *LLMModel
	gradAggregator *GradientAggregator
	dveClient      *DVEClient

	// backend is nil until a real training environment is configured.
	backend LoRATrainingBackend

	mu            sync.Mutex
	trainingQueue map[string]*TrainingJob

	now func() time.Time
}

// NewLoRATrainer builds a trainer. A nil backend is permitted and produces a
// trainer that refuses to train — which is honest, and keeps every caller of
// TrainLoRAAdapter on a real error path instead of a fabricated success.
func NewLoRATrainer(baseModel *LLMModel, aggregator *GradientAggregator, dveClient *DVEClient, backend LoRATrainingBackend) *LoRATrainer {
	lt := &LoRATrainer{
		baseModel:      baseModel,
		gradAggregator: aggregator,
		dveClient:      dveClient,
		backend:        backend,
		trainingQueue:  make(map[string]*TrainingJob),
		now:            time.Now,
	}
	if lt.gradAggregator == nil {
		lt.gradAggregator = &GradientAggregator{aggregationType: AGGREGATE_MEAN}
	}
	return lt
}

// TrainingStatus defines the status of a LoRA training job.
type TrainingStatus int

const (
	TRAINING_STARTED TrainingStatus = iota
	TRAINING_IN_PROGRESS
	TRAINING_COMPLETE
	TRAINING_FAILED
)

// String renders the status for logs and status endpoints.
func (s TrainingStatus) String() string {
	switch s {
	case TRAINING_STARTED:
		return "started"
	case TRAINING_IN_PROGRESS:
		return "in_progress"
	case TRAINING_COMPLETE:
		return "complete"
	case TRAINING_FAILED:
		return "failed"
	default:
		return fmt.Sprintf("TrainingStatus(%d)", int(s))
	}
}

// TrainingJob represents a single LoRA training job.
type TrainingJob struct {
	ClusterID      string
	ErrorSolutions map[string][]*Solution
	LoRAConfig     LoRAConfiguration
	Status         TrainingStatus
	Checkpoints    []string
	FinalAdapter   []byte

	// Error is set when Status is TRAINING_FAILED, so IsComplete can report
	// *why* instead of returning a bare false.
	Error string
	// Examples is the prepared training set actually handed to the backend.
	Examples int
	// Backend names the backend that ran (or refused to run) the job.
	Backend    string
	StartedAt  time.Time
	FinishedAt time.Time
}

// LoRAConfiguration defines the configuration for LoRA training.
type LoRAConfiguration struct {
	Rank          int
	Alpha         float64
	TargetModules []string
	Dropout       float64
	LearningRate  float64
	BatchSize     int
	Epochs        int
}

// DefaultLoRAConfiguration is the configuration used when a cluster does not
// carry its own.
func DefaultLoRAConfiguration() LoRAConfiguration {
	return LoRAConfiguration{
		Rank:          8,
		Alpha:         16.0,
		TargetModules: []string{"q_proj", "v_proj"},
		Dropout:       0.1,
		LearningRate:  2e-4,
		BatchSize:     32,
		Epochs:        3,
	}
}

// TrainLoRAAdapter prepares the cluster's training data and starts a training
// job.
//
// It is idempotent per cluster: calling it twice for the same cluster returns
// the existing job rather than launching a second run, so a
// ProcessCluster retry cannot stack up duplicate training jobs.
func (lt *LoRATrainer) TrainLoRAAdapter(cluster *ErrorCluster) (*TrainingJob, error) {
	if cluster == nil {
		return nil, errors.New("cluster is required")
	}
	if strings.TrimSpace(cluster.ClusterID) == "" {
		return nil, errors.New("cluster id is required")
	}

	lt.mu.Lock()
	if existing, ok := lt.trainingQueue[cluster.ClusterID]; ok {
		lt.mu.Unlock()
		return existing, nil
	}
	lt.mu.Unlock()

	trainingData, err := lt.prepareTrainingData(cluster)
	if err != nil {
		return nil, fmt.Errorf("prepare training data for cluster %s: %w", cluster.ClusterID, err)
	}
	if len(trainingData) == 0 {
		return nil, fmt.Errorf("cluster %s produced no training examples; refusing to start a job on an empty dataset", cluster.ClusterID)
	}

	// Refuse before enqueueing, so a backend-less trainer leaves no
	// half-created job behind that IsComplete would then have to explain.
	if lt.backend == nil {
		return nil, fmt.Errorf("%w: cluster %s has %d training examples ready but no training backend is configured",
			ErrLoRATrainingBackendNotImplemented, cluster.ClusterID, len(trainingData))
	}

	job := &TrainingJob{
		ClusterID:      cluster.ClusterID,
		ErrorSolutions: cluster.Solutions,
		LoRAConfig:     DefaultLoRAConfiguration(),
		Status:         TRAINING_STARTED,
		Examples:       len(trainingData),
		Backend:        lt.backend.Name(),
		StartedAt:      lt.now(),
	}

	lt.mu.Lock()
	lt.trainingQueue[cluster.ClusterID] = job
	lt.mu.Unlock()

	go lt.runJob(job, trainingData)
	return job, nil
}

// runJob executes a job against the backend and records the terminal state.
func (lt *LoRATrainer) runJob(job *TrainingJob, data []TrainingExample) {
	lt.mu.Lock()
	job.Status = TRAINING_IN_PROGRESS
	lt.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	adapter, err := lt.backend.Train(ctx, job, data)

	lt.mu.Lock()
	defer lt.mu.Unlock()
	job.FinishedAt = lt.now()
	if err != nil {
		job.Status = TRAINING_FAILED
		job.Error = err.Error()
		return
	}
	if len(adapter) == 0 {
		// A backend that reports success with no adapter bytes has not
		// produced a usable result; treat it as a failure rather than
		// exporting an empty adapter as if it were trained output.
		job.Status = TRAINING_FAILED
		job.Error = "training backend returned success with an empty adapter"
		return
	}
	job.FinalAdapter = adapter
	job.Status = TRAINING_COMPLETE
}

// IsComplete reports whether a cluster's training job finished.
//
// Three outcomes are distinguished, which the previous `return true` could not
// express: still running (false, nil), finished (true, nil), and failed or
// absent (false, error).
func (lt *LoRATrainer) IsComplete(cluster *ErrorCluster) (bool, error) {
	if cluster == nil {
		return false, errors.New("cluster is required")
	}
	if strings.TrimSpace(cluster.ClusterID) == "" {
		return false, errors.New("cluster id is required")
	}

	lt.mu.Lock()
	job, ok := lt.trainingQueue[cluster.ClusterID]
	lt.mu.Unlock()
	if !ok {
		return false, fmt.Errorf("cluster %s has no training job", cluster.ClusterID)
	}

	switch job.Status {
	case TRAINING_COMPLETE:
		return true, nil
	case TRAINING_FAILED:
		return false, fmt.Errorf("training for cluster %s failed: %s", cluster.ClusterID, job.Error)
	default:
		return false, nil
	}
}

// Job returns a copy of a cluster's training job.
func (lt *LoRATrainer) Job(clusterID string) (*TrainingJob, bool) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	job, ok := lt.trainingQueue[clusterID]
	if !ok {
		return nil, false
	}
	copyOfJob := *job
	return &copyOfJob, true
}

// prepareTrainingData turns a cluster's validated solutions into training
// examples: the error's failure context is the input, the solution that
// resolved it is the target.
//
// This previously returned an empty slice unconditionally, so even a wired
// trainer would have been training on nothing.
func (lt *LoRATrainer) prepareTrainingData(cluster *ErrorCluster) ([]TrainingExample, error) {
	if cluster == nil {
		return nil, errors.New("cluster is required")
	}

	examples := make([]TrainingExample, 0, len(cluster.Errors))
	for _, solutionList := range cluster.Solutions {
		for _, solution := range solutionList {
			if solution == nil || !solution.Validated {
				// Only independently validated solutions are safe as training
				// targets: training on unvalidated answers bakes whatever they
				// got wrong into the skill.
				continue
			}
			output := strings.TrimSpace(solution.CodePackage)
			if output == "" {
				output = strings.TrimSpace(solution.ContributedData.Output)
			}
			if output == "" {
				continue
			}
			input, err := errorContextForSolution(cluster, solution)
			if err != nil {
				return nil, err
			}
			weight := solution.ValidationScore
			if weight <= 0 {
				weight = 1.0
			}
			examples = append(examples, TrainingExample{
				Input:  input,
				Output: output,
				Weight: weight,
			})
		}
	}
	return examples, nil
}

// errorContextForSolution resolves the failure context that a solution fixes.
func errorContextForSolution(cluster *ErrorCluster, solution *Solution) (string, error) {
	if solution == nil {
		return "", errors.New("solution is required")
	}
	if strings.TrimSpace(solution.ErrorID) == "" {
		return "", errors.New("solution has no error id, so its training input cannot be resolved")
	}
	for _, node := range cluster.Errors {
		if node == nil || node.Id != solution.ErrorID {
			continue
		}
		if len(node.FailureContext) > 0 {
			return string(node.FailureContext), nil
		}
		if strings.TrimSpace(node.Description) != "" {
			return node.Description, nil
		}
	}
	return "", fmt.Errorf("solution %s references error %s, which is not a member of cluster %s",
		solution.SolutionID, solution.ErrorID, cluster.ClusterID)
}

// ExportAdapter returns a finished job's trained adapter bytes.
//
// It replaces a stub that returned the literal bytes "dummy_lora_adapter" for
// any input, which would have been published as a real adapter.
func (lt *LoRATrainer) ExportAdapter(clusterID string) ([]byte, error) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	job, ok := lt.trainingQueue[clusterID]
	if !ok {
		return nil, fmt.Errorf("cluster %s has no training job", clusterID)
	}
	switch job.Status {
	case TRAINING_COMPLETE:
		if len(job.FinalAdapter) == 0 {
			return nil, fmt.Errorf("cluster %s reports a complete job with no adapter", clusterID)
		}
		return job.FinalAdapter, nil
	case TRAINING_FAILED:
		return nil, fmt.Errorf("cluster %s training failed: %s", clusterID, job.Error)
	default:
		return nil, fmt.Errorf("cluster %s training is not complete (status %s)", clusterID, job.Status)
	}
}

// ============================================================================
// DVE training-node allocation
// ============================================================================

// DVENode is a rented training node.
type DVENode struct {
	ID          string
	RentalID    string
	Endpoint    string
	AllocatedAt time.Time
}

// DVERentalBackend allocates training nodes on the DVE network. It is a seam
// for the same reason LoRATrainingBackend is: no DVE training-node rental
// client exists in this module yet, and inventing one that returns empty nodes
// is how the previous stub made distributed training look wired.
type DVERentalBackend interface {
	Rent(ctx context.Context, clusterID string, count int) ([]*DVENode, error)
	Release(ctx context.Context, nodes []*DVENode) error
}

// DVEClient rents training nodes from a DVE rental backend.
type DVEClient struct {
	backend DVERentalBackend
}

// NewDVEClient builds a DVE client. A nil backend makes RentNodes fail loudly.
func NewDVEClient(backend DVERentalBackend) *DVEClient {
	return &DVEClient{backend: backend}
}

// RentNodes allocates count training nodes for a cluster.
//
// Previously this returned a hardcoded "Dummy nodes" slice regardless of
// arguments — including on failure — so callers could not tell an allocation
// from a refusal.
func (dc *DVEClient) RentNodes(ctx context.Context, clusterID string, count int) ([]*DVENode, error) {
	if dc == nil || dc.backend == nil {
		return nil, fmt.Errorf("%w: no DVE rental backend configured (cluster %s, %d nodes)",
			ErrLoRATrainingBackendNotImplemented, clusterID, count)
	}
	if count <= 0 {
		return nil, fmt.Errorf("node count must be positive, got %d", count)
	}
	nodes, err := dc.backend.Rent(ctx, clusterID, count)
	if err != nil {
		return nil, fmt.Errorf("rent %d DVE nodes for cluster %s: %w", count, clusterID, err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("DVE rental backend returned no nodes for cluster %s", clusterID)
	}
	return nodes, nil
}
