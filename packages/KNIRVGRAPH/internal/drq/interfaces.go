package drq

import (
	"context"
	"time"
)

// NetworkTopologyInterface defines the necessary methods from NetworkTopology
// that DRQ components need to interact with.
type NetworkTopologyInterface interface {
	GetClusterLoad(domain string) float64 // This method is used by calculatePriority
	// Add other methods from NetworkTopology that DRQ components need to call
}

// LoRATrainerInterface defines the necessary methods from LoRATrainer
// that DRQ components need to interact with.
//
// The concrete *LoRATrainer satisfies this; so does any test double. The
// production implementation refuses to report a completed job unless a real
// training backend is configured (see ErrLoRATrainingBackendNotImplemented) —
// it never manufactures training success.
type LoRATrainerInterface interface {
	// IsComplete reports whether the cluster's training job has finished. It
	// returns an error rather than a bare bool so "training genuinely failed"
	// and "training has not finished" cannot be confused; the previous
	// signature returned a hardcoded true, which advanced every cluster
	// straight to validation without any training having occurred.
	IsComplete(cluster *ErrorCluster) (bool, error)
	TrainLoRAAdapter(cluster *ErrorCluster) (*TrainingJob, error)
}

// SkillDiscoveryService turns a converged ErrorCluster into a candidate
// SkillNode. Implemented by *SkillDiscoveryEngine.
//
// Discovery is side-effect free: it builds the node (description, stable ID,
// resolved error set, content-addressed skill document URI) but performs no
// KNIRVGRAPH/KNIRVORACLE/KNIRVCHAIN calls. All minting is orchestrated by
// SkillMintingProtocol.MintSkillFromCluster (GRAPH-1).
type SkillDiscoveryService interface {
	DiscoverSkill(cluster *ErrorCluster) (*SkillNode, error)
}

// SkillTowerMinter mints — and can roll back — KNIRVGRAPH's own skill record.
// Implemented by *KNIRVGRAPHClient.
type SkillTowerMinter interface {
	MintSkillTower(skillNode *SkillNode, errors []*ErrorNode) error
	RevertSkillMinting(skillID string) error
}

// CanonicalSkillMinter mints the KNIRVCHAIN EventBundleNFT commit-bundle proof
// for a skill. Implemented by *KNIRVCHAINClient.
type CanonicalSkillMinter interface {
	MintCanonicalSkill(skillNode *SkillNode) error
}

// SkillVerifier is KNIRVORACLE's skill-economics surface (verify + burn +
// ownership + bounty). Implemented by *KNIRVORACLEClient.
type SkillVerifier interface {
	VerifySkillNode(skillNode *SkillNode) (bool, error)
	BurnNRNForSkill(skillNode *SkillNode) error
	GrantOwnershipRights(rights SkillOwnershipRights) error
	PayBounty(agentID, skillID string, amount uint64) error
}

// SkillMinter mints a validated skill from a resolved cluster and later
// distributes that cluster's rewards. Implemented by *SkillMintingProtocol.
type SkillMinter interface {
	MintSkillFromCluster(ctx context.Context, cluster *ErrorCluster) (*SkillNode, error)
	DistributeSkillRewards(cluster *ErrorCluster, skillNode *SkillNode) error
	// DistributeRoyalty pays the perpetual invocation entitlement. It is a
	// separate step from DistributeSkillRewards because the bounty payout is
	// once-only guarded, and a missing royalty backend must surface without
	// re-triggering the bounty.
	DistributeRoyalty(cluster *ErrorCluster, skillNode *SkillNode) error
	// InvocationFee is the perpetual per-invocation fee this skill commands,
	// used to price the owner's royalty entitlement.
	InvocationFee(skillNode *SkillNode) float64
}

// ConsensusAnnouncer announces a newly canonical skill to the network.
// The production implementation is backed by KNIRVGATEWAY's DHT resource
// cache; tests inject a double so the DRQ call graph can be exercised without
// a live gateway.
type ConsensusAnnouncer interface {
	AnnounceSkill(ctx context.Context, skillNode *SkillNode) error
}

// ValidationService is the platform DVE validation service: the same machinery
// paying customers' skills are validated through. submitForValidation creates a
// validation task for a cluster; isValidated re-reads the task and checks the
// validation service's own Proof certificate — DRQ does not re-implement
// validation policy locally.
type ValidationService interface {
	SubmitClusterForValidation(ctx context.Context, cluster *ErrorCluster) (*ValidationTask, error)
	GetValidationProof(ctx context.Context, taskID string) (*ValidationProof, error)
}

// RoyaltyDistributor distributes royalties for a skill held under perpetual
// ownership rights. Implemented by *KNIRVCHAINRoyaltyClient (unix socket to
// KNIRVCHAIN's royalty engine); a missing/unavailable backend returns
// ErrRoyaltyBackendUnavailable rather than silently no-op'ing.
type RoyaltyDistributor interface {
	DistributeRoyalty(ctx context.Context, req RoyaltyRequest) (*RoyaltyReceipt, error)
}

// ConvergenceCriteria are the thresholds a CLUSTER_ACTIVE error cluster must
// meet before DRQ treats it as converged and starts LoRA training.
//
// There is no existing DRQ config struct for these (the DRQ/Clustering config
// blocks live in internal/app, which a leaf package like drq must not import
// back), so the field set is defined here and the production defaults mirror
// the values internal/app.NewApp() installs:
//   - MinAvgSimilarity mirrors app.ClusteringConfig.SimilarityThreshold (0.85),
//     which is also DRQClusterManager.similarityThresh — the same threshold the
//     cosine-similarity clustering itself uses to decide cluster membership.
//   - MaxClusterSize mirrors app.ClusteringConfig.MaxClusterSize (100) and is
//     read from DRQClusterManager.maxClusterSize.
//   - MinValidatedSolutions mirrors app.ValidationConfig.MinAttestations (5)
//     scaled down: at least one independently validated solution must exist
//     before a cluster is worth training on (MinAttestations is the DVE's
//     per-solution quorum, not a cluster gate).
type ConvergenceCriteria struct {
	MinClusterSize        int           // minimum number of errors in the cluster
	MinAvgSimilarity      float64       // min average cosine similarity of member embeddings to the centroid
	MinTimeInCluster      time.Duration // minimum age of the cluster since CreatedAt
	MinValidatedSolutions int           // minimum validated solutions available for training
}

// DefaultConvergenceCriteria returns the thresholds used when ClusterManager is
// constructed without an explicit override.
func DefaultConvergenceCriteria() ConvergenceCriteria {
	return ConvergenceCriteria{
		MinClusterSize:        3,
		MinAvgSimilarity:      0.85,
		MinTimeInCluster:      1 * time.Minute,
		MinValidatedSolutions: 1,
	}
}

// Compile-time assertions that the concrete types satisfy the ports the
// ClusterManager lifecycle depends on. Without these, a signature drift on
// either side silently turns a wired stage back into a no-op — which is
// exactly how the DRQ loop previously ended up doing nothing while reporting
// success everywhere.
var (
	_ SkillMinter           = (*SkillMintingProtocol)(nil)
	_ SkillTowerMinter      = (*KNIRVGRAPHClient)(nil)
	_ SkillDiscoveryService = (*SkillDiscoveryEngine)(nil)
	_ LoRATrainerInterface  = (*LoRATrainer)(nil)
	_ SkillTowerStore       = (*NRVSkillTowerStore)(nil)
)
