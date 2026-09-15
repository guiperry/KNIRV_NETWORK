package drq

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ClusterManager orchestrates an error cluster through its lifecycle:
//
//	CLUSTER_ACTIVE -> (converged) -> CLUSTER_TRAINING -> CLUSTER_VALIDATING
//	               -> (validated) -> CLUSTER_RESOLVED -> (rewards distributed)
//
// Every stage is owned by an injected collaborator. A stage whose collaborator
// is absent or refuses the work returns an error and leaves the cluster in its
// current state — it never advances on a fabricated success. That was the
// central defect here: every method used to be a no-op returning nil/false,
// with `isReadyForTraining` hardcoded to false, so the entire loop was
// simultaneously unreachable and pretending to work.
type ClusterManager struct {
	// mu guards the cluster map and cluster field mutations.
	mu       sync.Mutex
	clusters map[string]*ErrorCluster

	// Collaborators, one per lifecycle stage. Each is required only for the
	// stage that uses it, and the check happens at that stage so a
	// misconfigured manager fails at the point of use rather than silently
	// skipping the work.
	loraTrainer LoRATrainerInterface
	skillMinter SkillMinter
	validator   ValidationService

	topology NetworkTopologyInterface
	drqSync  *DRQSyncProtocol

	criteria ConvergenceCriteria
	now      func() time.Time
}

// ClusterDeps are the collaborators and settings a ClusterManager needs.
// Only Criteria is optional (DefaultConvergenceCriteria fills it in).
type ClusterDeps struct {
	Trainer   LoRATrainerInterface
	Minter    SkillMinter
	Validator ValidationService
	Topology  NetworkTopologyInterface
	DRQSync   *DRQSyncProtocol
	Criteria  *ConvergenceCriteria
	Now       func() time.Time
}

// NewClusterManager builds a ClusterManager.
func NewClusterManager(deps ClusterDeps) *ClusterManager {
	criteria := DefaultConvergenceCriteria()
	if deps.Criteria != nil {
		criteria = *deps.Criteria
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &ClusterManager{
		clusters:    make(map[string]*ErrorCluster),
		loraTrainer: deps.Trainer,
		skillMinter: deps.Minter,
		validator:   deps.Validator,
		topology:    deps.Topology,
		drqSync:     deps.DRQSync,
		criteria:    criteria,
		now:         now,
	}
}

// Track registers a cluster with the manager so ProcessCluster can find it.
func (cm *ClusterManager) Track(cluster *ErrorCluster) error {
	if cluster == nil {
		return errors.New("cluster is required")
	}
	if strings.TrimSpace(cluster.ClusterID) == "" {
		return errors.New("cluster id is required")
	}
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clusters[cluster.ClusterID] = cluster
	return nil
}

// Cluster returns a tracked cluster by id.
func (cm *ClusterManager) Cluster(clusterID string) (*ErrorCluster, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cluster, ok := cm.clusters[clusterID]
	return cluster, ok
}

// TrackedClusterIDs returns the ids of every cluster this manager is driving,
// in a deterministic order.
func (cm *ClusterManager) TrackedClusterIDs() []string {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	ids := make([]string, 0, len(cm.clusters))
	for id := range cm.clusters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Criteria returns the convergence thresholds in force.
func (cm *ClusterManager) Criteria() ConvergenceCriteria { return cm.criteria }

// ProcessCluster advances one cluster by one lifecycle step. It is intended to
// be called repeatedly (each DRQ tick); each call either makes progress or
// returns nil because nothing was actionable yet.
//
// "Nothing actionable yet" (cluster not converged, validation still pending) is
// deliberately distinct from failure: the former returns nil, the latter
// returns an error.
func (cm *ClusterManager) ProcessCluster(ctx context.Context, clusterID string) error {
	cm.mu.Lock()
	cluster, exists := cm.clusters[clusterID]
	cm.mu.Unlock()

	if !exists {
		return fmt.Errorf("cluster %s is not tracked", clusterID)
	}
	if cluster == nil {
		return fmt.Errorf("cluster %s is nil", clusterID)
	}

	switch cluster.Status {
	case CLUSTER_ACTIVE:
		ready, err := cm.isReadyForTraining(cluster)
		if err != nil {
			return fmt.Errorf("evaluate convergence for cluster %s: %w", clusterID, err)
		}
		if !ready {
			return nil
		}
		return cm.initiateLoRATraining(cluster)

	case CLUSTER_TRAINING:
		if cm.loraTrainer == nil {
			return fmt.Errorf("cluster %s is training but no LoRA trainer is configured", clusterID)
		}
		complete, err := cm.loraTrainer.IsComplete(cluster)
		if err != nil {
			return fmt.Errorf("check LoRA training for cluster %s: %w", clusterID, err)
		}
		if !complete {
			return nil
		}
		cluster.LoRAInProgress = false
		cluster.Status = CLUSTER_VALIDATING
		return cm.submitForValidation(cluster)

	case CLUSTER_VALIDATING:
		valid, err := cm.isValidated(cluster)
		if err != nil {
			return fmt.Errorf("check validation for cluster %s: %w", clusterID, err)
		}
		if !valid {
			return nil
		}
		return cm.mintSkill(ctx, cluster)

	case CLUSTER_RESOLVED:
		return cm.distributeRewards(cluster)

	case CLUSTER_ARCHIVED:
		return nil
	}

	return fmt.Errorf("cluster %s has unknown status %v", clusterID, cluster.Status)
}

// isReadyForTraining decides whether a cluster has converged enough to be worth
// training on (GRAPH-2).
//
// This was previously `return false // Not ready for stub`, which made every
// stage downstream of it unreachable — the DRQ loop could never mint anything.
//
// The four criteria are deliberately conservative, and are the ones the
// ConvergenceCriteria doc block commits to:
//
//   - size:      at least MinClusterSize member errors, or there is nothing to
//     learn from.
//   - tightness: the members' embeddings average at least MinAvgSimilarity
//     cosine similarity to the centroid. This is the actual
//     "have these errors converged on a single concept?" test, and it reuses
//     the same cosineSimilarity the clustering itself uses.
//   - maturity:  the cluster has existed for at least MinTimeInCluster, so a
//     burst of near-identical errors arriving in one instant cannot trigger
//     training before the cluster has been observed to persist.
//   - evidence:  at least MinValidatedSolutions independently validated
//     solutions exist, so training is not run on unproven answers.
func (cm *ClusterManager) isReadyForTraining(cluster *ErrorCluster) (bool, error) {
	if cluster == nil {
		return false, errors.New("cluster is required")
	}
	if cluster.Status != CLUSTER_ACTIVE {
		return false, nil
	}

	if len(cluster.Errors) < cm.criteria.MinClusterSize {
		return false, nil
	}

	if cm.criteria.MinTimeInCluster > 0 {
		if cluster.CreatedAt.IsZero() {
			return false, errors.New("cluster has no creation time, so its maturity cannot be evaluated")
		}
		if cm.now().Sub(cluster.CreatedAt) < cm.criteria.MinTimeInCluster {
			return false, nil
		}
	}

	if cm.countValidatedSolutions(cluster) < cm.criteria.MinValidatedSolutions {
		return false, nil
	}

	meanSim, members, err := meanCentroidSimilarity(cluster)
	if err != nil {
		return false, err
	}
	if members == 0 {
		// No member carries an embedding: convergence is unmeasurable, and
		// silently returning "not ready" would hide a broken ingestion path
		// forever.
		return false, errors.New("no cluster member carries an embedding; convergence cannot be measured")
	}
	if meanSim < cm.criteria.MinAvgSimilarity {
		return false, nil
	}

	return true, nil
}

// meanCentroidSimilarity returns the mean cosine similarity between each member
// error's embedding and the cluster centroid, plus how many members were
// measured.
//
// A member whose embedding dimension disagrees with the centroid is a hard
// error rather than a skipped member: cosineSimilarity returns 0 for
// mismatched lengths, which would quietly depress the average and keep a
// healthy cluster permanently "not converged".
func meanCentroidSimilarity(cluster *ErrorCluster) (float64, int, error) {
	if cluster == nil {
		return 0, 0, errors.New("cluster is required")
	}
	if len(cluster.Centroid) == 0 {
		return 0, 0, errors.New("cluster has no centroid to measure members against")
	}

	var sum float64
	var measured int
	for _, member := range cluster.Errors {
		if member == nil || len(member.Embedding) == 0 {
			continue
		}
		if len(member.Embedding) != len(cluster.Centroid) {
			return 0, 0, fmt.Errorf(
				"error %s has a %d-dim embedding but the cluster centroid is %d-dim",
				member.Id, len(member.Embedding), len(cluster.Centroid))
		}
		sum += cosineSimilarity(member.Embedding, cluster.Centroid)
		measured++
	}
	if measured == 0 {
		return 0, 0, nil
	}
	return sum / float64(measured), measured, nil
}

// countValidatedSolutions counts solutions across the cluster that have passed
// platform validation.
func (cm *ClusterManager) countValidatedSolutions(cluster *ErrorCluster) int {
	if cluster == nil {
		return 0
	}
	validated := 0
	for _, solutions := range cluster.Solutions {
		for _, solution := range solutions {
			if solution != nil && solution.Validated {
				validated++
			}
		}
	}
	return validated
}

// initiateLoRATraining kicks off adapter training for a converged cluster.
//
// A trainer that cannot actually train (no backend configured) must return an
// error; the cluster then stays CLUSTER_ACTIVE and is retried next tick rather
// than being advanced on a fabricated success.
func (cm *ClusterManager) initiateLoRATraining(cluster *ErrorCluster) error {
	if cluster == nil {
		return errors.New("cluster is required")
	}
	if cm.loraTrainer == nil {
		return fmt.Errorf("%w: no LoRA trainer configured for cluster %s",
			ErrLoRATrainingBackendNotImplemented, cluster.ClusterID)
	}

	job, err := cm.loraTrainer.TrainLoRAAdapter(cluster)
	if err != nil {
		cluster.LoRAInProgress = false
		return fmt.Errorf("start LoRA training for cluster %s: %w", cluster.ClusterID, err)
	}
	if job == nil {
		cluster.LoRAInProgress = false
		return fmt.Errorf("LoRA trainer returned no job for cluster %s", cluster.ClusterID)
	}

	cluster.LoRAInProgress = true
	cluster.TrainingJobID = job.ClusterID
	if strings.TrimSpace(cluster.TrainingJobID) == "" {
		cluster.TrainingJobID = cluster.ClusterID
	}
	cluster.Status = CLUSTER_TRAINING
	return nil
}

// submitForValidation hands a trained cluster to the platform's DVE validation
// service — the same machinery paying customers' skills are validated through,
// so mined skills are not held to a weaker standard than sold ones.
func (cm *ClusterManager) submitForValidation(cluster *ErrorCluster) error {
	if cluster == nil {
		return errors.New("cluster is required")
	}
	if cm.validator == nil {
		return fmt.Errorf("no validation service configured; cannot validate cluster %s", cluster.ClusterID)
	}

	task, err := cm.validator.SubmitClusterForValidation(context.Background(), cluster)
	if err != nil {
		return fmt.Errorf("submit cluster %s for validation: %w", cluster.ClusterID, err)
	}
	if task == nil {
		return fmt.Errorf("validation service returned no task for cluster %s", cluster.ClusterID)
	}
	if strings.TrimSpace(task.ID) == "" {
		return fmt.Errorf("validation service returned a task with no id for cluster %s", cluster.ClusterID)
	}

	cluster.ValidationTaskID = task.ID
	cluster.ValidationDegraded = task.Degraded
	cluster.Status = CLUSTER_VALIDATING
	return nil
}

// isValidated reports whether the platform has validated this cluster.
//
// A degraded validator response is never treated as validation: minting a skill
// off a fallback score is exactly the "silently ships a lie" failure mode this
// pipeline exists to prevent, so it surfaces as an error.
func (cm *ClusterManager) isValidated(cluster *ErrorCluster) (bool, error) {
	if cluster == nil {
		return false, errors.New("cluster is required")
	}
	if cm.validator == nil {
		return false, fmt.Errorf("no validation service configured; cannot check cluster %s", cluster.ClusterID)
	}
	if strings.TrimSpace(cluster.ValidationTaskID) == "" {
		return false, fmt.Errorf("cluster %s has no validation task to check", cluster.ClusterID)
	}

	proof, err := cm.validator.GetValidationProof(context.Background(), cluster.ValidationTaskID)
	if err != nil {
		return false, fmt.Errorf("fetch validation proof for cluster %s: %w", cluster.ClusterID, err)
	}
	if proof == nil {
		// Still pending.
		return false, nil
	}
	if proof.Degraded {
		return false, fmt.Errorf("%w: cluster %s was validated by a degraded validator",
			ErrValidationProofMissing, cluster.ClusterID)
	}
	if !proof.Valid || len(proof.Proof) == 0 {
		return false, nil
	}

	cluster.ValidationProof = proof.Proof
	cluster.ValidationDegraded = false
	return true, nil
}

// mintSkill mints a canonical skill from a validated cluster.
func (cm *ClusterManager) mintSkill(ctx context.Context, cluster *ErrorCluster) error {
	if cluster == nil {
		return errors.New("cluster is required")
	}
	if cm.skillMinter == nil {
		return fmt.Errorf("no skill minter configured; cannot mint for cluster %s", cluster.ClusterID)
	}
	// Ownership must be established before minting: reward distribution is
	// addressed to cluster.OwnerAgent, and an empty owner would send the
	// invocation-fee entitlement nowhere.
	cm.computeOwnership(cluster)

	skill, err := cm.skillMinter.MintSkillFromCluster(ctx, cluster)
	if err != nil {
		return fmt.Errorf("mint skill from cluster %s: %w", cluster.ClusterID, err)
	}
	if skill == nil {
		return fmt.Errorf("minter returned no skill for cluster %s", cluster.ClusterID)
	}
	if strings.TrimSpace(skill.ID) == "" {
		return fmt.Errorf("minter returned a skill with no id for cluster %s", cluster.ClusterID)
	}

	cluster.MintedSkillID = skill.ID
	cluster.MintedSkill = skill
	cluster.Status = CLUSTER_RESOLVED
	return nil
}

// distributeRewards pays out a resolved cluster's rewards, at most once for the
// bounty and until-successfully for the royalty.
//
// The lifecycle deliberately calls this on every tick while the cluster is
// CLUSTER_RESOLVED, so the once-only bounty guard is load-bearing.
func (cm *ClusterManager) distributeRewards(cluster *ErrorCluster) error {
	if cluster == nil {
		return errors.New("cluster is required")
	}
	if cm.skillMinter == nil {
		return fmt.Errorf("no skill minter configured; cannot distribute rewards for cluster %s", cluster.ClusterID)
	}
	if cluster.MintedSkill == nil {
		return fmt.Errorf("cluster %s has no minted skill to distribute rewards for", cluster.ClusterID)
	}

	// The bounty payout is once-only: the lifecycle re-enters this branch on
	// every tick while the cluster stays CLUSTER_RESOLVED, so without the guard
	// a single resolved cluster would be paid repeatedly.
	if !cluster.RewardsDistributed {
		if err := cm.skillMinter.DistributeSkillRewards(cluster, cluster.MintedSkill); err != nil {
			return fmt.Errorf("distribute rewards for cluster %s: %w", cluster.ClusterID, err)
		}
		cluster.RewardsDistributed = true
	}

	// GRAPH-6: perpetual ownership rights earn an invocation royalty, paid by
	// KNIRVCHAIN's royalty engine through the minting protocol. A missing or
	// unavailable engine is an explicit error, never a silent skip.
	//
	// This runs outside the once-only guard on purpose: a royalty shortfall is
	// recoverable, so it is retried until it lands rather than being dropped
	// after the first attempt. Double-payment is prevented by the request's
	// RefID (the cluster id), which makes the royalty idempotent per cluster.
	if err := cm.skillMinter.DistributeRoyalty(cluster, cluster.MintedSkill); err != nil {
		return fmt.Errorf("distribute royalty for skill %s: %w", cluster.MintedSkill.ID, err)
	}

	return nil
}

// computeOwnership sets OwnerAgent to the agent with the most solutions in the
// cluster and SecondPlace to the runner-up, so ownership rewards have a real
// recipient. Ties break on agent ID for determinism.
func (cm *ClusterManager) computeOwnership(cluster *ErrorCluster) {
	if cluster == nil || len(cluster.AgentCounts) == 0 {
		return
	}
	owner, second := "", ""
	ownerCount, secondCount := -1, -1
	for agentID, count := range cluster.AgentCounts {
		switch {
		case count > ownerCount, count == ownerCount && agentID < owner:
			second, secondCount = owner, ownerCount
			owner, ownerCount = agentID, count
		case count > secondCount, count == secondCount && agentID < second:
			second, secondCount = agentID, count
		}
	}
	cluster.OwnerAgent = owner
	cluster.SecondPlace = second
}

// countUniqueApproaches counts the distinct solution approaches in a cluster
// (GRAPH-5).
//
// An "approach" is the solution's code package when present — two agents
// submitting byte-identical packages solved the problem the same way — falling
// back to the agent's identity when no package is recorded, since otherwise
// every packageless solution would collapse into a single approach.
func (cm *ClusterManager) countUniqueApproaches(cluster *ErrorCluster) int {
	if cluster == nil {
		return 0
	}
	seen := make(map[string]struct{})
	for _, solutions := range cluster.Solutions {
		for _, solution := range solutions {
			if solution == nil {
				continue
			}
			key := strings.TrimSpace(solution.CodePackage)
			if key == "" {
				key = "agent:" + strings.TrimSpace(solution.AgentID)
			}
			if key == "agent:" {
				continue
			}
			seen[key] = struct{}{}
		}
	}
	return len(seen)
}

// calculateValidationRate returns the fraction of a cluster's solutions that
// passed validation (GRAPH-5). A cluster with no solutions rates 0, not 1: an
// empty cluster has proven nothing.
func (cm *ClusterManager) calculateValidationRate(cluster *ErrorCluster) float64 {
	if cluster == nil {
		return 0
	}
	var total, validated int
	for _, solutions := range cluster.Solutions {
		for _, solution := range solutions {
			if solution == nil {
				continue
			}
			total++
			if solution.Validated {
				validated++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(validated) / float64(total)
}

// Convergence report used by callers that want to explain why a cluster is not
// ready yet, without duplicating the criteria logic.
type ConvergenceReport struct {
	Ready              bool
	Reason             string
	MemberCount        int
	MeanSimilarity     float64
	ValidatedCount     int
	Age                time.Duration
	RequiredMinSize    int
	RequiredSimilarity float64
	RequiredMinAge     time.Duration
	RequiredValidated  int
}

// EvaluateConvergence explains the current convergence verdict for a cluster.
func (cm *ClusterManager) EvaluateConvergence(cluster *ErrorCluster) ConvergenceReport {
	report := ConvergenceReport{
		RequiredMinSize:    cm.criteria.MinClusterSize,
		RequiredSimilarity: cm.criteria.MinAvgSimilarity,
		RequiredMinAge:     cm.criteria.MinTimeInCluster,
		RequiredValidated:  cm.criteria.MinValidatedSolutions,
	}
	if cluster == nil {
		report.Reason = "cluster is nil"
		return report
	}
	report.MemberCount = len(cluster.Errors)
	report.ValidatedCount = cm.countValidatedSolutions(cluster)
	if !cluster.CreatedAt.IsZero() {
		report.Age = cm.now().Sub(cluster.CreatedAt)
	}
	if meanSim, members, err := meanCentroidSimilarity(cluster); err == nil {
		report.MeanSimilarity = meanSim
		report.MemberCount = members
	} else {
		report.Reason = err.Error()
		return report
	}

	ready, err := cm.isReadyForTraining(cluster)
	report.Ready = ready
	if err != nil {
		report.Reason = err.Error()
		return report
	}
	if ready {
		report.Reason = "converged"
		return report
	}
	switch {
	case len(cluster.Errors) < cm.criteria.MinClusterSize:
		report.Reason = fmt.Sprintf("needs %d members, has %d", cm.criteria.MinClusterSize, len(cluster.Errors))
	case cm.criteria.MinTimeInCluster > 0 && report.Age < cm.criteria.MinTimeInCluster:
		report.Reason = fmt.Sprintf("needs to age %v, is %v", cm.criteria.MinTimeInCluster, report.Age)
	case report.ValidatedCount < cm.criteria.MinValidatedSolutions:
		report.Reason = fmt.Sprintf("needs %d validated solutions, has %d", cm.criteria.MinValidatedSolutions, report.ValidatedCount)
	case report.MeanSimilarity < cm.criteria.MinAvgSimilarity:
		report.Reason = fmt.Sprintf("mean similarity %.4f below threshold %.4f", report.MeanSimilarity, cm.criteria.MinAvgSimilarity)
	default:
		report.Reason = "not converged"
	}
	return report
}
