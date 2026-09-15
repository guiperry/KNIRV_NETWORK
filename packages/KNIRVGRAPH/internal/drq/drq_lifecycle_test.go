package drq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Test doubles
// ============================================================================

// fakeTrainer is a controllable LoRATrainerInterface.
type fakeTrainer struct {
	trainErr    error
	complete    bool
	completeErr error
	trainCalls  int
}

func (f *fakeTrainer) TrainLoRAAdapter(cluster *ErrorCluster) (*TrainingJob, error) {
	f.trainCalls++
	if f.trainErr != nil {
		return nil, f.trainErr
	}
	return &TrainingJob{ClusterID: cluster.ClusterID, Status: TRAINING_IN_PROGRESS}, nil
}

func (f *fakeTrainer) IsComplete(cluster *ErrorCluster) (bool, error) {
	return f.complete, f.completeErr
}

// fakeValidator is a controllable ValidationService.
type fakeValidator struct {
	submitTask *ValidationTask
	submitErr  error
	proof      *ValidationProof
	proofErr   error
	submitted  int
}

func (f *fakeValidator) SubmitClusterForValidation(_ context.Context, cluster *ErrorCluster) (*ValidationTask, error) {
	f.submitted++
	if f.submitErr != nil {
		return nil, f.submitErr
	}
	if f.submitTask != nil {
		return f.submitTask, nil
	}
	return &ValidationTask{ID: "task-" + cluster.ClusterID, ClusterID: cluster.ClusterID, Status: "in_progress"}, nil
}

func (f *fakeValidator) GetValidationProof(context.Context, string) (*ValidationProof, error) {
	if f.proofErr != nil {
		return nil, f.proofErr
	}
	return f.proof, nil
}

// fakeMinter is a controllable SkillMinter.
type fakeMinter struct {
	mintErr        error
	rewardErr      error
	royaltyErr     error
	mintedSkill    *SkillNode
	mintCalls      int
	rewardCalls    int
	royaltyCalls   int
	lastRewardArgs [2]string
}

func (f *fakeMinter) MintSkillFromCluster(_ context.Context, cluster *ErrorCluster) (*SkillNode, error) {
	f.mintCalls++
	if f.mintErr != nil {
		return nil, f.mintErr
	}
	if f.mintedSkill != nil {
		return f.mintedSkill, nil
	}
	return &SkillNode{
		ID:             "skill-" + cluster.ClusterID,
		Creator:        cluster.OwnerAgent,
		Description:    "real description",
		ResolvesErrors: []string{"e1"},
	}, nil
}

func (f *fakeMinter) DistributeSkillRewards(cluster *ErrorCluster, skillNode *SkillNode) error {
	f.rewardCalls++
	f.lastRewardArgs = [2]string{cluster.ClusterID, skillNode.ID}
	return f.rewardErr
}

func (f *fakeMinter) DistributeRoyalty(cluster *ErrorCluster, skillNode *SkillNode) error {
	f.royaltyCalls++
	return f.royaltyErr
}

func (f *fakeMinter) InvocationFee(*SkillNode) float64 { return 10 }

// fakeTowerStore records skill tower writes.
type fakeTowerStore struct {
	putIDs    []string
	deleted   []string
	putErr    error
	deleteErr error
	// records backs SkillResolvingError, keyed by resolved error id.
	records   map[string]SkillRecord
	lookupErr error
}

func (f *fakeTowerStore) SkillResolvingError(_ context.Context, errorID string) (*SkillRecord, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	record, ok := f.records[errorID]
	if !ok {
		// Absent is not an error: the member simply has no minted fix yet.
		return nil, nil
	}
	return &record, nil
}

func (f *fakeTowerStore) PutSkill(_ context.Context, skillID, _ string, _ []string, _ map[string]interface{}) error {
	if f.putErr != nil {
		return f.putErr
	}
	f.putIDs = append(f.putIDs, skillID)
	return nil
}

func (f *fakeTowerStore) DeleteSkill(_ context.Context, skillID string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, skillID)
	return nil
}

// ============================================================================
// Fixtures
// ============================================================================

// convergedCluster builds a cluster that meets every convergence criterion:
// three members, embeddings identical to the centroid (similarity 1.0), one
// validated solution, and a creation time past the maturity threshold.
func convergedCluster(t *testing.T) *ErrorCluster {
	t.Helper()
	embedding := []float64{1, 0, 0, 0}
	members := make([]*ErrorNode, 0, 3)
	for i := 0; i < 3; i++ {
		members = append(members, &ErrorNode{
			Id:             fmt.Sprintf("err-%d", i),
			Description:    fmt.Sprintf("failure %d", i),
			Domain:         "runtime",
			Complexity:     int32(i + 1),
			FailureContext: []byte(fmt.Sprintf("context %d", i)),
			Embedding:      append([]float64(nil), embedding...),
		})
	}
	return &ErrorCluster{
		ClusterID:   "cluster-1",
		Errors:      members,
		Centroid:    append([]float64(nil), embedding...),
		CreatedAt:   time.Now().Add(-time.Hour),
		AgentCounts: map[string]int{"agent-a": 3, "agent-b": 1},
		Solutions: map[string][]*Solution{
			"err-0": {{SolutionID: "s1", ErrorID: "err-0", AgentID: "agent-a", CodePackage: "pkg-a", Validated: true, ValidationScore: 0.9}},
		},
		Status: CLUSTER_ACTIVE,
	}
}

// ============================================================================
// GRAPH-2: convergence
// ============================================================================

func TestIsReadyForTrainingConverged(t *testing.T) {
	cm := NewClusterManager(ClusterDeps{})
	ready, err := cm.isReadyForTraining(convergedCluster(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ready {
		t.Fatal("a cluster meeting every criterion must be ready for training")
	}
}

func TestIsReadyForTrainingBlocksOnEachCriterion(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ErrorCluster)
	}{
		{"too few members", func(c *ErrorCluster) { c.Errors = c.Errors[:2] }},
		{"too young", func(c *ErrorCluster) { c.CreatedAt = time.Now() }},
		{"no validated solutions", func(c *ErrorCluster) {
			c.Solutions = map[string][]*Solution{"err-0": {{SolutionID: "s1", Validated: false}}}
		}},
		{"similarity below threshold", func(c *ErrorCluster) {
			// Orthogonal to the centroid => cosine similarity 0.
			c.Errors[0].Embedding = []float64{0, 1, 0, 0}
			c.Errors[1].Embedding = []float64{0, 1, 0, 0}
			c.Errors[2].Embedding = []float64{0, 1, 0, 0}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cluster := convergedCluster(t)
			tc.mutate(cluster)
			cm := NewClusterManager(ClusterDeps{})
			ready, err := cm.isReadyForTraining(cluster)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ready {
				t.Fatalf("cluster must not be ready: %s", tc.name)
			}
		})
	}
}

// A cluster whose members carry no embeddings cannot be assessed. Returning
// "not ready" would hide a broken ingestion path behind a permanent false.
func TestIsReadyForTrainingFailsLoudlyWithoutEmbeddings(t *testing.T) {
	cluster := convergedCluster(t)
	for _, member := range cluster.Errors {
		member.Embedding = nil
	}
	cm := NewClusterManager(ClusterDeps{})
	ready, err := cm.isReadyForTraining(cluster)
	if err == nil {
		t.Fatal("expected an error when no member carries an embedding")
	}
	if ready {
		t.Fatal("must not report ready")
	}
}

// A dimension mismatch must be an error, not a silent zero-similarity member
// that keeps a healthy cluster permanently unconverged.
func TestIsReadyForTrainingFailsLoudlyOnDimensionMismatch(t *testing.T) {
	cluster := convergedCluster(t)
	cluster.Errors[0].Embedding = []float64{1, 0}
	cm := NewClusterManager(ClusterDeps{})
	if _, err := cm.isReadyForTraining(cluster); err == nil {
		t.Fatal("expected an error for a mismatched embedding dimension")
	}
}

func TestEvaluateConvergenceExplainsVerdict(t *testing.T) {
	cm := NewClusterManager(ClusterDeps{})
	cluster := convergedCluster(t)
	cluster.Errors = cluster.Errors[:1]

	report := cm.EvaluateConvergence(cluster)
	if report.Ready {
		t.Fatal("cluster with one member must not be ready")
	}
	if report.Reason == "" || report.Reason == "converged" {
		t.Fatalf("expected an explanatory reason, got %q", report.Reason)
	}
	if report.MemberCount != 1 {
		t.Fatalf("member count = %d, want 1", report.MemberCount)
	}
}

// ============================================================================
// GRAPH-3: the lifecycle
// ============================================================================

func TestProcessClusterFullLifecycle(t *testing.T) {
	trainer := &fakeTrainer{}
	validator := &fakeValidator{proof: &ValidationProof{TaskID: "task-cluster-1", Valid: true, Proof: []byte("proof-bytes")}}
	minter := &fakeMinter{}
	cm := NewClusterManager(ClusterDeps{Trainer: trainer, Minter: minter, Validator: validator})

	cluster := convergedCluster(t)
	if err := cm.Track(cluster); err != nil {
		t.Fatalf("Track: %v", err)
	}

	// Tick 1: converged -> training.
	if err := cm.ProcessCluster(context.Background(), cluster.ClusterID); err != nil {
		t.Fatalf("tick 1: %v", err)
	}
	if cluster.Status != CLUSTER_TRAINING {
		t.Fatalf("status = %v, want CLUSTER_TRAINING", cluster.Status)
	}

	// Tick 2: training still running -> no progress, no error.
	if err := cm.ProcessCluster(context.Background(), cluster.ClusterID); err != nil {
		t.Fatalf("tick 2: %v", err)
	}
	if cluster.Status != CLUSTER_TRAINING {
		t.Fatalf("status = %v, want to remain CLUSTER_TRAINING", cluster.Status)
	}

	// Tick 3: training complete -> validating.
	trainer.complete = true
	if err := cm.ProcessCluster(context.Background(), cluster.ClusterID); err != nil {
		t.Fatalf("tick 3: %v", err)
	}
	if cluster.Status != CLUSTER_VALIDATING {
		t.Fatalf("status = %v, want CLUSTER_VALIDATING", cluster.Status)
	}
	if cluster.ValidationTaskID != "task-cluster-1" {
		t.Fatalf("validation task id = %q", cluster.ValidationTaskID)
	}

	// Tick 4: validated -> minted and resolved.
	if err := cm.ProcessCluster(context.Background(), cluster.ClusterID); err != nil {
		t.Fatalf("tick 4: %v", err)
	}
	if cluster.Status != CLUSTER_RESOLVED {
		t.Fatalf("status = %v, want CLUSTER_RESOLVED", cluster.Status)
	}
	if cluster.MintedSkill == nil || cluster.MintedSkillID == "" {
		t.Fatal("a skill must be recorded on the cluster")
	}
	if len(cluster.ValidationProof) == 0 {
		t.Fatal("the platform proof must be attached to the cluster")
	}
	// Ownership must be established before minting: reward distribution is
	// addressed to OwnerAgent.
	if cluster.OwnerAgent != "agent-a" {
		t.Fatalf("owner = %q, want agent-a (most solutions)", cluster.OwnerAgent)
	}

	// Tick 5: rewards distributed.
	if err := cm.ProcessCluster(context.Background(), cluster.ClusterID); err != nil {
		t.Fatalf("tick 5: %v", err)
	}
	if minter.rewardCalls != 1 {
		t.Fatalf("reward calls = %d, want 1", minter.rewardCalls)
	}

	// Tick 6: a resolved cluster must not be paid twice.
	if err := cm.ProcessCluster(context.Background(), cluster.ClusterID); err != nil {
		t.Fatalf("tick 6: %v", err)
	}
	if minter.rewardCalls != 1 {
		t.Fatalf("reward calls = %d after a second resolved tick, want 1", minter.rewardCalls)
	}
}

// The previous lifecycle reported success with no trainer at all. A missing
// backend must leave the cluster untouched and say so.
func TestProcessClusterRefusesWithoutTrainer(t *testing.T) {
	cm := NewClusterManager(ClusterDeps{Minter: &fakeMinter{}, Validator: &fakeValidator{}})
	cluster := convergedCluster(t)
	if err := cm.Track(cluster); err != nil {
		t.Fatalf("Track: %v", err)
	}

	err := cm.ProcessCluster(context.Background(), cluster.ClusterID)
	if err == nil {
		t.Fatal("expected an error when no LoRA trainer is configured")
	}
	if !errors.Is(err, ErrLoRATrainingBackendNotImplemented) {
		t.Fatalf("error = %v, want it to wrap ErrLoRATrainingBackendNotImplemented", err)
	}
	if cluster.Status != CLUSTER_ACTIVE {
		t.Fatalf("status = %v, want to remain CLUSTER_ACTIVE", cluster.Status)
	}
}

func TestProcessClusterPropagatesTrainingFailure(t *testing.T) {
	trainer := &fakeTrainer{}
	cm := NewClusterManager(ClusterDeps{Trainer: trainer, Minter: &fakeMinter{}, Validator: &fakeValidator{}})
	cluster := convergedCluster(t)
	_ = cm.Track(cluster)

	if err := cm.ProcessCluster(context.Background(), cluster.ClusterID); err != nil {
		t.Fatalf("tick 1: %v", err)
	}
	// Status is now TRAINING; report a hard failure from the trainer.
	trainer.completeErr = errors.New("GPU node lost")
	err := cm.ProcessCluster(context.Background(), cluster.ClusterID)
	if err == nil {
		t.Fatal("expected the training failure to surface")
	}
	if cluster.Status != CLUSTER_TRAINING {
		t.Fatalf("status = %v, want to remain CLUSTER_TRAINING on failure", cluster.Status)
	}
}

// A degraded validator must never yield a minted skill.
func TestProcessClusterDegradedValidatorBlocksMinting(t *testing.T) {
	trainer := &fakeTrainer{complete: true}
	validator := &fakeValidator{proof: &ValidationProof{TaskID: "t1", Valid: true, Proof: []byte("p"), Degraded: true}}
	minter := &fakeMinter{}
	cm := NewClusterManager(ClusterDeps{Trainer: trainer, Minter: minter, Validator: validator})

	cluster := convergedCluster(t)
	cluster.Status = CLUSTER_VALIDATING
	cluster.ValidationTaskID = "t1"
	_ = cm.Track(cluster)

	err := cm.ProcessCluster(context.Background(), cluster.ClusterID)
	if err == nil {
		t.Fatal("expected a degraded validator to block the lifecycle")
	}
	if !errors.Is(err, ErrValidationProofMissing) {
		t.Fatalf("error = %v, want it to wrap ErrValidationProofMissing", err)
	}
	if minter.mintCalls != 0 {
		t.Fatalf("mint calls = %d, want 0", minter.mintCalls)
	}
	if cluster.Status != CLUSTER_VALIDATING {
		t.Fatalf("status = %v, want to remain CLUSTER_VALIDATING", cluster.Status)
	}
}

func TestProcessClusterPendingValidationMakesNoProgress(t *testing.T) {
	validator := &fakeValidator{proof: nil} // still pending
	cm := NewClusterManager(ClusterDeps{Trainer: &fakeTrainer{}, Minter: &fakeMinter{}, Validator: validator})
	cluster := convergedCluster(t)
	cluster.Status = CLUSTER_VALIDATING
	cluster.ValidationTaskID = "t1"
	_ = cm.Track(cluster)

	if err := cm.ProcessCluster(context.Background(), cluster.ClusterID); err != nil {
		t.Fatalf("pending validation is not an error: %v", err)
	}
	if cluster.Status != CLUSTER_VALIDATING {
		t.Fatalf("status = %v, want CLUSTER_VALIDATING", cluster.Status)
	}
}

func TestProcessClusterUnknownClusterIsAnError(t *testing.T) {
	cm := NewClusterManager(ClusterDeps{})
	if err := cm.ProcessCluster(context.Background(), "nope"); err == nil {
		t.Fatal("expected an error for an untracked cluster")
	}
}

// A royalty shortfall must be reported without re-paying the bounty.
func TestProcessClusterRoyaltyShortfallDoesNotDoublePayBounty(t *testing.T) {
	minter := &fakeMinter{royaltyErr: ErrRoyaltyBackendUnavailable}
	cm := NewClusterManager(ClusterDeps{Minter: minter})
	cluster := convergedCluster(t)
	cluster.Status = CLUSTER_RESOLVED
	cluster.MintedSkill = &SkillNode{ID: "skill-1"}
	cluster.OwnerAgent = "agent-a"
	_ = cm.Track(cluster)

	err := cm.ProcessCluster(context.Background(), cluster.ClusterID)
	if err == nil {
		t.Fatal("expected the missing royalty backend to surface")
	}
	if minter.rewardCalls != 1 {
		t.Fatalf("reward calls = %d, want 1", minter.rewardCalls)
	}
	// Second tick retries the royalty but must not re-pay the bounty.
	if err := cm.ProcessCluster(context.Background(), cluster.ClusterID); err == nil {
		t.Fatal("expected the royalty error again")
	}
	if minter.rewardCalls != 1 {
		t.Fatalf("reward calls = %d after retry, want 1 (bounty must not be re-paid)", minter.rewardCalls)
	}
}

// ============================================================================
// GRAPH-5: helpers
// ============================================================================

func TestCountUniqueApproaches(t *testing.T) {
	cm := NewClusterManager(ClusterDeps{})
	cluster := &ErrorCluster{
		Solutions: map[string][]*Solution{
			"e1": {
				{SolutionID: "s1", AgentID: "a", CodePackage: "pkg-a"},
				{SolutionID: "s2", AgentID: "b", CodePackage: "pkg-a"}, // same approach
				{SolutionID: "s3", AgentID: "c", CodePackage: "pkg-b"},
			},
			"e2": {
				{SolutionID: "s4", AgentID: "d", CodePackage: ""}, // falls back to agent identity
				{SolutionID: "s5", AgentID: "e", CodePackage: ""},
			},
		},
	}
	if got := cm.countUniqueApproaches(cluster); got != 4 {
		t.Fatalf("unique approaches = %d, want 4 (pkg-a, pkg-b, agent:d, agent:e)", got)
	}
	if got := cm.countUniqueApproaches(nil); got != 0 {
		t.Fatalf("nil cluster approaches = %d, want 0", got)
	}
}

func TestCalculateValidationRate(t *testing.T) {
	cm := NewClusterManager(ClusterDeps{})
	cluster := &ErrorCluster{
		Solutions: map[string][]*Solution{
			"e1": {
				{SolutionID: "s1", Validated: true},
				{SolutionID: "s2", Validated: true},
				{SolutionID: "s3", Validated: false},
				{SolutionID: "s4", Validated: false},
			},
		},
	}
	if got := cm.calculateValidationRate(cluster); got != 0.5 {
		t.Fatalf("validation rate = %v, want 0.5", got)
	}
	// An empty cluster has proven nothing; the rate is 0, not 1.
	if got := cm.calculateValidationRate(&ErrorCluster{}); got != 0 {
		t.Fatalf("empty cluster validation rate = %v, want 0", got)
	}
}

func TestComputeOwnership(t *testing.T) {
	cm := NewClusterManager(ClusterDeps{})
	cluster := &ErrorCluster{AgentCounts: map[string]int{"a": 1, "b": 5, "c": 3}}
	cm.computeOwnership(cluster)
	if cluster.OwnerAgent != "b" {
		t.Fatalf("owner = %q, want b", cluster.OwnerAgent)
	}
	if cluster.SecondPlace != "c" {
		t.Fatalf("second = %q, want c", cluster.SecondPlace)
	}
}

// ============================================================================
// LoRA trainer: no fabricated success
// ============================================================================

type fakeBackend struct {
	adapter []byte
	err     error
}

func (f *fakeBackend) Train(context.Context, *TrainingJob, []TrainingExample) ([]byte, error) {
	return f.adapter, f.err
}
func (f *fakeBackend) Name() string { return "fake" }

func trainerCluster() *ErrorCluster {
	return &ErrorCluster{
		ClusterID: "c1",
		Errors:    []*ErrorNode{{Id: "e1", FailureContext: []byte("boom")}},
		Solutions: map[string][]*Solution{
			"e1": {{SolutionID: "s1", ErrorID: "e1", AgentID: "a", CodePackage: "fix", Validated: true, ValidationScore: 0.9}},
		},
	}
}

// The old implementation returned a hardcoded `true` from IsComplete and the
// bytes "dummy_lora_adapter" from the exporter.
func TestTrainerRefusesWithoutBackend(t *testing.T) {
	trainer := NewLoRATrainer(&LLMModel{Name: "base"}, nil, nil, nil)
	_, err := trainer.TrainLoRAAdapter(trainerCluster())
	if err == nil {
		t.Fatal("expected an error when no training backend is configured")
	}
	if !errors.Is(err, ErrLoRATrainingBackendNotImplemented) {
		t.Fatalf("error = %v, want it to wrap ErrLoRATrainingBackendNotImplemented", err)
	}
}

func TestTrainerRefusesOnEmptyDataset(t *testing.T) {
	trainer := NewLoRATrainer(&LLMModel{Name: "base"}, nil, nil, &fakeBackend{adapter: []byte("ok")})
	cluster := &ErrorCluster{ClusterID: "c1", Errors: []*ErrorNode{{Id: "e1"}}}
	_, err := trainer.TrainLoRAAdapter(cluster)
	if err == nil {
		t.Fatal("expected an error for a cluster with no usable training examples")
	}
}

// Only independently validated solutions may become training targets.
func TestPrepareTrainingDataSkipsUnvalidated(t *testing.T) {
	trainer := NewLoRATrainer(&LLMModel{Name: "base"}, nil, nil, nil)
	cluster := &ErrorCluster{
		ClusterID: "c1",
		Errors:    []*ErrorNode{{Id: "e1", FailureContext: []byte("boom")}},
		Solutions: map[string][]*Solution{
			"e1": {
				{SolutionID: "s1", ErrorID: "e1", CodePackage: "good", Validated: true},
				{SolutionID: "s2", ErrorID: "e1", CodePackage: "bad", Validated: false},
			},
		},
	}
	examples, err := trainer.prepareTrainingData(cluster)
	if err != nil {
		t.Fatalf("prepareTrainingData: %v", err)
	}
	if len(examples) != 1 {
		t.Fatalf("examples = %d, want 1 (unvalidated solutions must be excluded)", len(examples))
	}
	if examples[0].Output != "good" {
		t.Fatalf("output = %q, want the validated solution", examples[0].Output)
	}
	if examples[0].Input != "boom" {
		t.Fatalf("input = %q, want the error's failure context", examples[0].Input)
	}
}

func TestTrainerIsCompleteDistinguishesOutcomes(t *testing.T) {
	trainer := NewLoRATrainer(&LLMModel{Name: "base"}, nil, nil, &fakeBackend{adapter: []byte("trained")})
	cluster := trainerCluster()

	// No job at all.
	if _, err := trainer.IsComplete(cluster); err == nil {
		t.Fatal("expected an error for a cluster with no training job")
	}

	job, err := trainer.TrainLoRAAdapter(cluster)
	if err != nil {
		t.Fatalf("TrainLoRAAdapter: %v", err)
	}
	if job == nil {
		t.Fatal("expected a job")
	}

	// Wait for the background run to finish.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		complete, err := trainer.IsComplete(cluster)
		if err != nil {
			t.Fatalf("IsComplete: %v", err)
		}
		if complete {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	complete, err := trainer.IsComplete(cluster)
	if err != nil {
		t.Fatalf("IsComplete: %v", err)
	}
	if !complete {
		t.Fatal("expected training to complete")
	}

	adapter, err := trainer.ExportAdapter(cluster.ClusterID)
	if err != nil {
		t.Fatalf("ExportAdapter: %v", err)
	}
	if string(adapter) != "trained" {
		t.Fatalf("adapter = %q, want the backend's real output", string(adapter))
	}
}

func TestTrainerReportsBackendFailure(t *testing.T) {
	trainer := NewLoRATrainer(&LLMModel{Name: "base"}, nil, nil, &fakeBackend{err: errors.New("node died")})
	cluster := trainerCluster()
	if _, err := trainer.TrainLoRAAdapter(cluster); err != nil {
		t.Fatalf("TrainLoRAAdapter: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		_, err := trainer.IsComplete(cluster)
		if err != nil {
			lastErr = err
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if lastErr == nil {
		t.Fatal("expected the backend failure to be reported")
	}
	if _, err := trainer.ExportAdapter(cluster.ClusterID); err == nil {
		t.Fatal("expected ExportAdapter to fail for a failed job")
	}
}

func TestTrainerIsIdempotentPerCluster(t *testing.T) {
	backend := &fakeBackend{adapter: []byte("trained")}
	trainer := NewLoRATrainer(&LLMModel{Name: "base"}, nil, nil, backend)
	cluster := trainerCluster()

	first, err := trainer.TrainLoRAAdapter(cluster)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := trainer.TrainLoRAAdapter(cluster)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first != second {
		t.Fatal("a second call must return the existing job, not start another")
	}
}

func TestDVENodeRentalRefusesWithoutBackend(t *testing.T) {
	client := NewDVEClient(nil)
	if _, err := client.RentNodes(context.Background(), "c1", 4); err == nil {
		t.Fatal("expected an error when no DVE rental backend is configured")
	}
}

// ============================================================================
// Discovery: deterministic, side-effect free, non-fabricated
// ============================================================================

func TestDiscoverSkillIsDeterministicAndReal(t *testing.T) {
	engine := NewSkillDiscoveryEngine(nil)
	cluster := convergedCluster(t)
	cluster.OwnerAgent = "agent-a"

	first, err := engine.DiscoverSkill(cluster)
	if err != nil {
		t.Fatalf("DiscoverSkill: %v", err)
	}
	second, err := engine.DiscoverSkill(cluster)
	if err != nil {
		t.Fatalf("DiscoverSkill (second): %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("skill id is not deterministic: %q vs %q", first.ID, second.ID)
	}
	if first.ID == "skill-cluster-1" {
		t.Fatal("skill id must be content-addressed, not just the cluster id")
	}
	if len(first.ResolvesErrors) != 3 {
		t.Fatalf("resolves %d errors, want the cluster's 3", len(first.ResolvesErrors))
	}
	for _, forbidden := range []string{"Dummy", "dummy"} {
		if contains(first.Description, forbidden) {
			t.Fatalf("description contains fabricated text: %q", first.Description)
		}
		if contains(first.CodePackageURI, forbidden) {
			t.Fatalf("code package URI contains fabricated text: %q", first.CodePackageURI)
		}
	}
	if contains(first.CodePackageURI, "ipfs://") {
		t.Fatalf("code package URI claims an IPFS location: %q", first.CodePackageURI)
	}
	if first.Creator != "agent-a" {
		t.Fatalf("creator = %q, want the cluster owner", first.Creator)
	}
}

func TestDiscoverSkillRejectsUnusableClusters(t *testing.T) {
	engine := NewSkillDiscoveryEngine(nil)
	cases := []struct {
		name    string
		cluster *ErrorCluster
	}{
		{"nil cluster", nil},
		{"no cluster id", &ErrorCluster{}},
		{"no members", &ErrorCluster{ClusterID: "c1", OwnerAgent: "a"}},
		{"no owner", func() *ErrorCluster { c := convergedCluster(t); c.OwnerAgent = ""; return c }()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := engine.DiscoverSkill(tc.cluster); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

// ============================================================================
// GRAPH-4: skill tower mint and revert
// ============================================================================

func TestMintSkillTowerRecordsAndReverts(t *testing.T) {
	store := &fakeTowerStore{}
	client := NewKNIRVGRAPHClient(store)
	node := &SkillNode{ID: "skill-1", Creator: "agent-a", Description: "d", ResolvesErrors: []string{"e1"}}

	if err := client.MintSkillTower(node, []*ErrorNode{{Id: "e1"}}); err != nil {
		t.Fatalf("MintSkillTower: %v", err)
	}
	if len(store.putIDs) != 1 || store.putIDs[0] != "skill-1" {
		t.Fatalf("put ids = %v, want [skill-1]", store.putIDs)
	}
	if err := client.RevertSkillMinting("skill-1"); err != nil {
		t.Fatalf("RevertSkillMinting: %v", err)
	}
	if len(store.deleted) != 1 || store.deleted[0] != "skill-1" {
		t.Fatalf("deleted = %v, want [skill-1]", store.deleted)
	}
}

// The old MintSkillTower was `return nil` with no store at all.
func TestMintSkillTowerFailsLoudlyWithoutStore(t *testing.T) {
	client := NewKNIRVGRAPHClient(nil)
	node := &SkillNode{ID: "skill-1", Creator: "agent-a"}
	if err := client.MintSkillTower(node, nil); err == nil {
		t.Fatal("expected an error when no tower store is configured")
	}
	if err := client.RevertSkillMinting("skill-1"); err == nil {
		t.Fatal("expected an error when no tower store is configured")
	}
}

func TestMintSkillTowerValidatesInput(t *testing.T) {
	client := NewKNIRVGRAPHClient(&fakeTowerStore{})
	if err := client.MintSkillTower(nil, nil); err == nil {
		t.Fatal("expected an error for a nil skill node")
	}
	if err := client.MintSkillTower(&SkillNode{}, nil); err == nil {
		t.Fatal("expected an error for a skill with no id")
	}
	if err := client.MintSkillTower(&SkillNode{ID: "s1"}, nil); err == nil {
		t.Fatal("expected an error for a skill with no creator")
	}
}

func TestSkillMintingProtocolRequiresCollaborators(t *testing.T) {
	// MintSkillFromCluster previously could not fail on missing collaborators
	// because every step was a no-op.
	smp := NewSkillMintingProtocol(SkillMintingDeps{})
	if _, err := smp.MintSkillFromCluster(context.Background(), convergedCluster(t)); err == nil {
		t.Fatal("expected an error with no collaborators configured")
	}
}

func TestDistributeRoyaltyRefusesWithoutBackend(t *testing.T) {
	smp := NewSkillMintingProtocol(SkillMintingDeps{})
	err := smp.DistributeRoyalty(convergedCluster(t), &SkillNode{ID: "s1"})
	if err == nil {
		t.Fatal("expected an error when no royalty distributor is configured")
	}
	if !errors.Is(err, ErrRoyaltyBackendUnavailable) {
		t.Fatalf("error = %v, want it to wrap ErrRoyaltyBackendUnavailable", err)
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
