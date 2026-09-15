package drq

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"KNIRVGRAPH/internal/dht"
)

// SkillTowerStore is KNIRVGRAPH's own skill record store — the "tower" the
// skill is minted into.
//
// It is a port rather than a direct dependency so the DRQ pipeline can be
// tested without a live NRVSystem, and so the concrete backing (currently
// internal/nrv's skill registry) can change without touching this package.
type SkillTowerStore interface {
	// PutSkill records a skill node. Implementations must be idempotent per
	// skillID: re-minting an existing skill is not an error, because the
	// chain-side mint this pairs with is itself idempotent per skill ID.
	PutSkill(ctx context.Context, skillID, skillType string, capabilities []string, attributes map[string]interface{}) error
	// DeleteSkill removes a skill record. It is used to roll back a mint that
	// a later verification step rejected. A missing skill is not an error.
	DeleteSkill(ctx context.Context, skillID string) error
	// SkillResolvingError returns the skill minted for errorID, or (nil, nil)
	// when no skill resolves it. Absent is not an error: it means that member
	// error has no minted fix yet, which the corpus builder skips. A non-nil
	// error means the lookup itself failed and the caller must not proceed.
	SkillResolvingError(ctx context.Context, errorID string) (*SkillRecord, error)
}

// SkillRecord is the readable form of a minted skill tower record.
type SkillRecord struct {
	SkillID        string
	Creator        string
	Description    string
	ResolvesErrors []string
	CodePackageURI string
}

// KNIRVGRAPHClient mints — and can roll back — KNIRVGRAPH's own skill record.
type KNIRVGRAPHClient struct {
	store SkillTowerStore
}

// NewKNIRVGRAPHClient builds the client. A nil store makes MintSkillTower fail
// loudly rather than silently discarding the skill.
func NewKNIRVGRAPHClient(store SkillTowerStore) *KNIRVGRAPHClient {
	return &KNIRVGRAPHClient{store: store}
}

// MintSkillTower records skillNode in KNIRVGRAPH's skill tower.
//
// This was `return nil` — a no-op that reported success, so MintSkillFromCluster
// believed the skill had been minted on KNIRVGRAPH when nothing had been
// recorded anywhere, and a later revert attempt had nothing to roll back.
func (kgc *KNIRVGRAPHClient) MintSkillTower(skillNode *SkillNode, errorNodes []*ErrorNode) error {
	if skillNode == nil {
		return errors.New("skill node is required")
	}
	skillID := strings.TrimSpace(skillNode.ID)
	if skillID == "" {
		return errors.New("skill node id is required to mint a skill tower")
	}
	if kgc == nil || kgc.store == nil {
		return fmt.Errorf("no KNIRVGRAPH skill tower store configured; cannot mint skill %s", skillID)
	}
	if strings.TrimSpace(skillNode.Creator) == "" {
		return fmt.Errorf("skill %s has no creator to attribute the tower record to", skillID)
	}

	resolvedIDs := extractErrorIDs(errorNodes)
	capabilities := append([]string(nil), resolvedIDs...)
	capabilities = append(capabilities, "errors:"+strings.Join(resolvedIDs, ","))

	attributes := map[string]interface{}{
		"creator":          skillNode.Creator,
		"description":      skillNode.Description,
		"resolves_errors":  skillNode.ResolvesErrors,
		"code_package_uri": skillNode.CodePackageURI,
		"source":           "knirvgraph-drq",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := kgc.store.PutSkill(ctx, skillID, "mined", capabilities, attributes); err != nil {
		return fmt.Errorf("mint skill %s into KNIRVGRAPH tower: %w", skillID, err)
	}
	return nil
}

// RevertSkillMinting rolls back a skill tower mint.
//
// This was `func (kgc *KNIRVGRAPHClient) RevertSkillMinting(skillID string)`
// with an empty body and no return, so MintSkillFromCluster's failure paths
// "reverted" a mint by doing nothing and could not report that the rollback
// itself had failed.
func (kgc *KNIRVGRAPHClient) RevertSkillMinting(skillID string) error {
	skillID = strings.TrimSpace(skillID)
	if skillID == "" {
		return errors.New("skill id is required to revert a mint")
	}
	if kgc == nil || kgc.store == nil {
		return fmt.Errorf("no KNIRVGRAPH skill tower store configured; cannot revert skill %s", skillID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := kgc.store.DeleteSkill(ctx, skillID); err != nil {
		return fmt.Errorf("revert skill %s in KNIRVGRAPH tower: %w", skillID, err)
	}
	return nil
}

// MintCanonicalSkill mints the KNIRVCHAIN EventBundleNFT commit-bundle proof
// for a skill (see knirvchain_client.go). Idempotent per skillNode.ID.
func (kc *KNIRVCHAINClient) MintCanonicalSkill(skillNode *SkillNode) error {
	receipt, err := kc.mintSkillEventBundle(skillNode)
	if err != nil {
		return fmt.Errorf("mint canonical skill on KNIRVCHAIN: %w", err)
	}
	skillNode.ValidationProof = receipt
	return nil
}

// SkillOwnershipRights defines rights for skill invocation fees.
type SkillOwnershipRights struct {
	AgentID       string
	SkillID       string
	InvocationFee float64
	Perpetual     bool
}

// SkillMintingProtocol coordinates cross-chain minting.
type SkillMintingProtocol struct {
	knirvgraph     *KNIRVGRAPHClient
	knirvchain     *KNIRVCHAINClient
	knirvOracle    *KNIRVORACLEClient
	skillDiscovery *SkillDiscoveryEngine
	// royalty pays the perpetual invocation entitlement. A nil distributor is
	// reported as ErrRoyaltyBackendUnavailable rather than skipped.
	royalty RoyaltyDistributor
	// dhtClient announces newly canonical skills to KNIRVGATEWAY's DHT.
	dhtClient *dht.Client

	// uLoRA adapter minting. A nil compiler means this deployment does not mint
	// adapters, and the skill.md mint above still completes; once set, every
	// stage must succeed and the validation gate is never bypassed.
	uloraCompiler   ULoRACompiler
	bundleValidator BundleValidator
	skillDocs       SkillDocSource
	targetModels    map[string]ULoRATargetModel
	manifestVersion string
}

// SkillMintingDeps are the collaborators minting needs.
type SkillMintingDeps struct {
	Graph    *KNIRVGRAPHClient
	Chain    *KNIRVCHAINClient
	Oracle   *KNIRVORACLEClient
	Discover *SkillDiscoveryEngine
	Royalty  RoyaltyDistributor
	DHT      *dht.Client

	// uLoRA adapter compilation (optional as a group). When Compiler is set the
	// whole compile -> validate -> register sequence runs and any failure aborts
	// the mint. When it is nil the deployment simply does not mint adapters, and
	// the skill.md mint above still completes.
	Compiler ULoRACompiler
	// Validator is the step 2.6a gate. Required whenever Compiler is set: a
	// configured compiler with no validator stops with
	// ErrDVEValidationUnavailable rather than publishing unvalidated bytes.
	Validator BundleValidator
	// SkillDocs resolves each member error's validated fix (the
	// CorrectedCompletion half of a training record).
	SkillDocs SkillDocSource
	// TargetModels maps a model_origin to its architecture, which compilation
	// needs in full.
	TargetModels map[string]ULoRATargetModel
	// ManifestVersion is recorded on the registered bundle pointer.
	ManifestVersion string
}

// NewSkillMintingProtocol builds the minting protocol.
func NewSkillMintingProtocol(deps SkillMintingDeps) *SkillMintingProtocol {
	return &SkillMintingProtocol{
		knirvgraph:      deps.Graph,
		knirvchain:      deps.Chain,
		knirvOracle:     deps.Oracle,
		skillDiscovery:  deps.Discover,
		royalty:         deps.Royalty,
		dhtClient:       deps.DHT,
		uloraCompiler:   deps.Compiler,
		bundleValidator: deps.Validator,
		skillDocs:       deps.SkillDocs,
		targetModels:    deps.TargetModels,
		manifestVersion: deps.ManifestVersion,
	}
}

// MintSkillFromCluster creates a skill from a resolved cluster.
func (smp *SkillMintingProtocol) MintSkillFromCluster(ctx context.Context, cluster *ErrorCluster) (*SkillNode, error) {
	if cluster == nil {
		return nil, errors.New("cluster is required")
	}
	if smp.skillDiscovery == nil {
		return nil, errors.New("no skill discovery engine configured")
	}
	if smp.knirvgraph == nil {
		return nil, errors.New("no KNIRVGRAPH client configured")
	}
	if smp.knirvOracle == nil {
		return nil, errors.New("no KNIRVORACLE client configured")
	}
	if smp.knirvchain == nil {
		return nil, errors.New("no KNIRVCHAIN client configured")
	}

	// 2. Discover the skill document. LoRA-adapter retrieval is deprecated:
	// KNIRV skills are skill.md documents, not LoRA weights.
	skillNode, err := smp.skillDiscovery.DiscoverSkill(cluster)
	if err != nil {
		return nil, fmt.Errorf("discover skill for cluster %s: %w", cluster.ClusterID, err)
	}

	// 3. Mint on KNIRVGRAPH ("tower" in the error vector field).
	if err := smp.knirvgraph.MintSkillTower(skillNode, cluster.Errors); err != nil {
		return nil, err
	}

	// 4. KNIRVORACLE verification. A failure here rolls back step 3; if the
	// rollback also fails, both errors are reported so the orphaned tower
	// record is not silently left behind.
	verified, err := smp.knirvOracle.VerifySkillNode(skillNode)
	if err != nil {
		return nil, fmt.Errorf("verify skill %s with KNIRVORACLE: %w (rollback: %v)",
			skillNode.ID, err, smp.knirvgraph.RevertSkillMinting(skillNode.ID))
	}
	if !verified {
		return nil, fmt.Errorf("KNIRVORACLE rejected skill %s (rollback: %v)",
			skillNode.ID, smp.knirvgraph.RevertSkillMinting(skillNode.ID))
	}

	// 5. Compile, validate and register the cluster's portable uLoRA adapter.
	//    Skipped only when no compiler is configured for this deployment; once one
	//    is, every stage must succeed and a failure rolls back the mint. The
	//    validation gate is never bypassed (ulora_implementation.md §2.2).
	if smp.uloraCompiler != nil {
		if err := smp.mintULoRABundle(ctx, cluster, skillNode); err != nil {
			return nil, fmt.Errorf("%w (rollback: %v)", err, smp.knirvgraph.RevertSkillMinting(skillNode.ID))
		}
	}

	// 6. Canonical minting on KNIRVCHAIN.
	if err := smp.knirvchain.MintCanonicalSkill(skillNode); err != nil {
		return nil, fmt.Errorf("%w (rollback: %v)", err, smp.knirvgraph.RevertSkillMinting(skillNode.ID))
	}

	// 7. Burn NRN on KNIRVORACLE.
	if err := smp.knirvOracle.BurnNRNForSkill(skillNode); err != nil {
		return nil, err
	}

	// 8. Distribute rewards.
	if err := smp.distributeSkillRewards(cluster, skillNode); err != nil {
		return nil, err
	}

	// 9. Announce to the network.
	if err := smp.broadcastSkillConsensus(skillNode); err != nil {
		return nil, err
	}

	return skillNode, nil
}

// mintULoRABundle runs the adapter half of the mint: gather the corpus, compile
// it into a `.ulora` bundle, gate it on validation, then register it on
// KNIRVCHAIN.
//
// Ordering matters — the bundle is only registered after validation passes, and
// registration is what makes it fetchable, so an unvalidated adapter cannot
// become reachable by failing later instead of earlier.
func (smp *SkillMintingProtocol) mintULoRABundle(ctx context.Context, cluster *ErrorCluster, skillNode *SkillNode) error {
	if smp.uloraCompiler == nil {
		return ErrULoRACompilerUnavailable
	}
	// A configured compiler with no validator would have to either skip the gate
	// or invent a verdict. Neither is acceptable, so refuse.
	if smp.bundleValidator == nil {
		return ErrDVEValidationUnavailable
	}
	if smp.manifestVersion == "" {
		return errors.New("no uLoRA manifest version configured")
	}

	dataset, sourceSkillIDs, err := gatherClusterDatasets(cluster, smp.skillDocs)
	if err != nil {
		return fmt.Errorf("gather uLoRA dataset for cluster %s: %w", cluster.ClusterID, err)
	}
	targetModels, err := targetModelsForCorpus(dataset, smp.targetModels)
	if err != nil {
		return fmt.Errorf("resolve target models for cluster %s: %w", cluster.ClusterID, err)
	}

	result, err := smp.uloraCompiler.CompileCluster(ctx, cluster.ClusterID, dataset, targetModels, ULoRAProvenance{
		SourceID:         cluster.ClusterID,
		SourceDatasetIDs: sourceSkillIDs,
		Extensions: map[string]any{
			// The underlying error nodes, so a bundle can be traced back to the
			// errors it was derived from.
			"knirvgraph_error_ids": ResolvedErrorIDs(cluster),
		},
	})
	if err != nil {
		return fmt.Errorf("compile uLoRA bundle for cluster %s: %w", cluster.ClusterID, err)
	}

	if err := smp.bundleValidator.ValidateBundle(ctx, cluster, result, dataset); err != nil {
		return fmt.Errorf("validate uLoRA bundle for cluster %s: %w", cluster.ClusterID, err)
	}

	bundleBytes, err := ReadCompiledBundle(result)
	if err != nil {
		return fmt.Errorf("read compiled uLoRA bundle for cluster %s: %w", cluster.ClusterID, err)
	}

	if _, err := smp.knirvchain.MintULoRABundle(cluster.ClusterID, smp.manifestVersion, bundleBytes, sourceSkillIDs, result.TargetModels, result.ContentHash); err != nil {
		return fmt.Errorf("register uLoRA bundle for cluster %s: %w", cluster.ClusterID, err)
	}

	// Record where the adapter can be fetched from, so the skill and its adapter
	// are not two unlinked artifacts.
	if skillNode != nil {
		skillNode.CodePackageURI = fmt.Sprintf("knirv://network/ulora_%s", result.ContentHash)
	}
	return nil
}

// DistributeSkillRewards is the exported form of distributeSkillRewards, so
// the cluster lifecycle can drive payouts through the SkillMinter port.
func (smp *SkillMintingProtocol) DistributeSkillRewards(cluster *ErrorCluster, skillNode *SkillNode) error {
	return smp.distributeSkillRewards(cluster, skillNode)
}

// InvocationFee exposes the perpetual per-invocation fee for a skill.
func (smp *SkillMintingProtocol) InvocationFee(skillNode *SkillNode) float64 {
	return smp.calculateInvocationFee(skillNode)
}

// DistributeRoyalty pays the perpetual invocation entitlement for a minted
// skill via KNIRVCHAIN's royalty engine.
//
// Deliberately a separate step from distributeSkillRewards: the bounty payout
// is guarded by the cluster's once-only flag, and folding a not-yet-available
// royalty backend into that same call would either mask the shortfall or
// re-pay every bounty on each retry. Returning the error here surfaces the gap
// without either.
func (smp *SkillMintingProtocol) DistributeRoyalty(cluster *ErrorCluster, skillNode *SkillNode) error {
	if cluster == nil || skillNode == nil {
		return errors.New("cluster and skill node are required")
	}
	if smp.royalty == nil {
		return fmt.Errorf("%w: skill %s is held under perpetual ownership rights but no royalty distributor is configured",
			ErrRoyaltyBackendUnavailable, skillNode.ID)
	}
	owner := strings.TrimSpace(cluster.OwnerAgent)
	if owner == "" {
		return fmt.Errorf("skill %s has no owner agent to pay a royalty to", skillNode.ID)
	}
	request := RoyaltyRequest{
		SkillID:    skillNode.ID,
		OwnerAgent: owner,
		Fee:        smp.calculateInvocationFee(skillNode),
		RefID:      cluster.ClusterID,
	}
	if _, err := smp.royalty.DistributeRoyalty(context.Background(), request); err != nil {
		return fmt.Errorf("distribute royalty for skill %s: %w", skillNode.ID, err)
	}
	return nil
}

// distributeSkillRewards handles cluster ownership rewards.
func (smp *SkillMintingProtocol) distributeSkillRewards(
	cluster *ErrorCluster,
	skillNode *SkillNode,
) error {
	if cluster == nil || skillNode == nil {
		return errors.New("cluster and skill node are required")
	}
	if smp.knirvOracle == nil {
		return errors.New("no KNIRVORACLE client configured")
	}
	if strings.TrimSpace(cluster.OwnerAgent) == "" {
		// The ownership grant is addressed to the owner; an empty owner would
		// grant the perpetual fee entitlement to nobody.
		return fmt.Errorf("cluster %s has no owner agent; cannot grant skill ownership", cluster.ClusterID)
	}

	ownerReward := SkillOwnershipRights{
		AgentID:       cluster.OwnerAgent,
		SkillID:       skillNode.ID,
		InvocationFee: smp.calculateInvocationFee(skillNode),
		Perpetual:     true,
	}
	if err := smp.knirvOracle.GrantOwnershipRights(ownerReward); err != nil {
		return err
	}

	// Distribute the bounty among contributors, proportional to each agent's
	// share of the cluster's total resolved-error solutions.
	var totalSolutions uint64
	for _, count := range cluster.AgentCounts {
		totalSolutions += uint64(count)
	}
	for agentID, solutionCount := range cluster.AgentCounts {
		share := smp.calculateBountyShare(
			cluster.TotalBounty,
			uint64(solutionCount),
			totalSolutions,
		)
		if share == 0 {
			continue
		}
		if err := smp.knirvOracle.PayBounty(agentID, skillNode.ID, share); err != nil {
			return err
		}
	}

	return nil
}

// Invocation-fee bounds (in NRN), modeled on KNIRVCHAIN's
// capability_minting.go calculateNRNCost base+multiplier+clamp shape and
// KNIRVORACLE economics/fees.go's FeeTypeSkillInvoke bounds.
const (
	baseInvocationFeeNRN = 10.0
	minInvocationFeeNRN  = 1.0
	maxInvocationFeeNRN  = 100.0
)

// calculateInvocationFee prices a skill's perpetual per-invocation fee.
func (smp *SkillMintingProtocol) calculateInvocationFee(skillNode *SkillNode) float64 {
	if skillNode == nil {
		return minInvocationFeeNRN
	}
	complexityMultiplier := 1.0 + 0.1*float64(len(skillNode.ResolvesErrors))
	fee := baseInvocationFeeNRN * complexityMultiplier
	switch {
	case fee < minInvocationFeeNRN:
		return minInvocationFeeNRN
	case fee > maxInvocationFeeNRN:
		return maxInvocationFeeNRN
	default:
		return fee
	}
}

// calculateBountyShare splits totalBounty proportionally to solutionCount's
// share of totalSolutionsInCluster.
func (smp *SkillMintingProtocol) calculateBountyShare(totalBounty, solutionCount, totalSolutionsInCluster uint64) uint64 {
	if totalSolutionsInCluster == 0 || solutionCount == 0 {
		return 0
	}
	return totalBounty * solutionCount / totalSolutionsInCluster
}

// broadcastSkillConsensusDefaultMultiaddr mirrors DHTClientAdapter's default.
const broadcastSkillConsensusDefaultMultiaddr = "/ip4/127.0.0.1/tcp/1317"

// broadcastSkillConsensus announces the newly canonical skill to the network
// via KNIRVGATEWAY's DHT resource cache. This is a discovery announcement
// ("this skill exists, here is where to find it"), not validator-consensus
// gossip: KNIRVGATEWAY exposes no pub/sub route (see internal/dht/adapter.go).
func (smp *SkillMintingProtocol) broadcastSkillConsensus(skillNode *SkillNode) error {
	if skillNode == nil {
		return errors.New("skill node is required")
	}
	client := smp.dhtClient
	if client == nil {
		client = dht.NewClient()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.AnnounceSkill(ctx, skillNode.ID, broadcastSkillConsensusDefaultMultiaddr); err != nil {
		return fmt.Errorf("announce skill to KNIRVGATEWAY DHT: %w", err)
	}
	log.Printf("DRQ: announced skill %s to KNIRVGATEWAY DHT", skillNode.ID)
	return nil
}
