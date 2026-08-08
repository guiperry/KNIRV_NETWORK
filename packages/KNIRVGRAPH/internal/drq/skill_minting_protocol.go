package drq

import (
	"context"
	"errors"
	"fmt"
	"time"

	"KNIRVGRAPH/internal/dht"
)

// KNIRVGRAPHClient is a stub for the KNIRVGRAPH client
type KNIRVGRAPHClient struct{}

// MintSkillTower is a stub for minting a skill on KNIRVGRAPH
func (kgc *KNIRVGRAPHClient) MintSkillTower(skillNode *SkillNode, errors []*ErrorNode) error {
	// TODO: Implement actual KNIRVGRAPH minting logic
	_ = skillNode
	_ = errors
	return nil
}

// RevertSkillMinting is a stub for reverting skill minting on KNIRVGRAPH
func (kgc *KNIRVGRAPHClient) RevertSkillMinting(skillID string) {
	// TODO: Implement actual KNIRVGRAPH revert logic
	_ = skillID
}

// MintCanonicalSkill mints (or, if MintSkillNode already minted it,
// re-fetches) skillNode's EventBundleNFT commit-bundle proof on KNIRVCHAIN.
// This is idempotent per skillNode.ID, so calling it after
// SkillDiscoveryEngine.DiscoverSkill has already minted the same skill
// (MintSkillFromCluster's step 2 calls DiscoverSkill, which itself mints via
// MintSkillNode, before this step 5 canonical mint runs) simply returns the
// existing receipt rather than double-minting or double-burning NRN.
func (kc *KNIRVCHAINClient) MintCanonicalSkill(skillNode *SkillNode) error {
	receipt, err := kc.mintSkillEventBundle(skillNode)
	if err != nil {
		return fmt.Errorf("mint canonical skill on KNIRVCHAIN: %w", err)
	}
	skillNode.ValidationProof = receipt
	return nil
}

// SkillOwnershipRights defines rights for skill invocation fees
type SkillOwnershipRights struct {
	AgentID       string
	SkillID       string
	InvocationFee float64
	Perpetual     bool
}

// BurnNRNForSkill, GrantOwnershipRights, and PayBounty are implemented in
// knirvoracle_client.go.

// SkillMintingProtocol coordinates cross-chain minting
type SkillMintingProtocol struct {
	knirvgraph     *KNIRVGRAPHClient
	knirvchain     *KNIRVCHAINClient
	knirvOracle    *KNIRVORACLEClient
	skillDiscovery *SkillDiscoveryEngine
	// dhtClient announces newly canonical skills to KNIRVGATEWAY's DHT
	// (broadcastSkillConsensus). Nil defaults to dht.NewClient() lazily,
	// so existing zero-value/test construction keeps working.
	dhtClient *dht.Client
}

// MintSkillFromCluster creates skill from resolved cluster
func (smp *SkillMintingProtocol) MintSkillFromCluster(
	cluster *ErrorCluster,
) (*SkillNode, error) {
	// 2. Discover skill via HRM WASM model. LoRA-adapter retrieval (former
	// step 1) is deprecated: KNIRV skills are represented as skill.md
	// documents, not LoRA weights (see KNIRV_NETWORK's CLAUDE.md), and the
	// prior step only ever returned a hardcoded dummy adapter — there was
	// no real training-job retrieval behind it to implement.
	skillNode, err := smp.skillDiscovery.DiscoverSkill(cluster, nil)
	if err != nil {
		return nil, err
	}

	// 3. Mint on KNIRVGRAPH (as "tower" in error vector field)
	err = smp.knirvgraph.MintSkillTower(skillNode, cluster.Errors)
	if err != nil {
		return nil, err
	}

	// 4. KNIRV-ORACLE verification
	verified, err := smp.knirvOracle.VerifySkillNode(skillNode)
	if err != nil || !verified {
		// Revert KNIRVGRAPH minting
		smp.knirvgraph.RevertSkillMinting(skillNode.ID)
		return nil, errors.New("KNIRV-ORACLE verification failed")
	}

	// 5. Canonical minting on KNIRVCHAIN
	err = smp.knirvchain.MintCanonicalSkill(skillNode)
	if err != nil {
		// Revert KNIRVGRAPH minting
		smp.knirvgraph.RevertSkillMinting(skillNode.ID)
		return nil, err
	}

	// 6. Trigger NRN burn on KNIRV-ORACLE
	err = smp.knirvOracle.BurnNRNForSkill(skillNode)
	if err != nil {
		return nil, err
	}

	// 7. Distribute rewards
	err = smp.distributeSkillRewards(cluster, skillNode)
	if err != nil {
		return nil, err
	}

	// 8. Update network consensus
	err = smp.broadcastSkillConsensus(skillNode)
	if err != nil {
		return nil, err
	}

	return skillNode, nil
}

// distributeSkillRewards handles cluster ownership rewards
func (smp *SkillMintingProtocol) distributeSkillRewards(
	cluster *ErrorCluster,
	skillNode *SkillNode,
) error {
	// Owner gets skill invocation fees
	ownerReward := SkillOwnershipRights{
		AgentID:       cluster.OwnerAgent,
		SkillID:       skillNode.ID,
		InvocationFee: smp.calculateInvocationFee(skillNode),
		Perpetual:     true,
	}

	err := smp.knirvOracle.GrantOwnershipRights(ownerReward)
	if err != nil {
		return err
	}

	// Distribute bounty among all contributors, proportional to each
	// agent's share of the cluster's total resolved-error solutions.
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

		err = smp.knirvOracle.PayBounty(agentID, skillNode.ID, share)
		if err != nil {
			return err
		}
	}

	return nil
}

// Invocation-fee bounds (in NRN), modeled on KNIRVCHAIN's
// capability_minting.go calculateNRNCost base+multiplier+clamp shape and
// KNIRVORACLE economics/fees.go's FeeTypeSkillInvoke bounds — there is no
// shared pricing service to call for this, so this is a local calculation.
const (
	baseInvocationFeeNRN = 10.0
	minInvocationFeeNRN  = 1.0
	maxInvocationFeeNRN  = 100.0
)

// calculateInvocationFee prices a skill's perpetual per-invocation fee.
// A skill that resolves more distinct errors has broader applicability and
// replacement value, so it can command a higher fee — bounded so a skill
// can't price itself out of ever being invoked.
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
// share of totalSolutionsInCluster, the same proportional-share shape as
// KNIRVORACLE's RewardManager.DistributeRewards (stake / totalStake *
// amount) — here, an agent's solutions stand in for stake.
func (smp *SkillMintingProtocol) calculateBountyShare(totalBounty, solutionCount, totalSolutionsInCluster uint64) uint64 {
	if totalSolutionsInCluster == 0 || solutionCount == 0 {
		return 0
	}
	return totalBounty * solutionCount / totalSolutionsInCluster
}

// broadcastSkillConsensusDefaultMultiaddr mirrors DHTClientAdapter's own
// default (internal/dht/adapter.go) — KNIRVGRAPH's local API port — used
// whenever a more specific network address for the skill isn't known.
const broadcastSkillConsensusDefaultMultiaddr = "/ip4/127.0.0.1/tcp/1317"

// broadcastSkillConsensus announces the newly canonical skill to the
// network via KNIRVGATEWAY's DHT resource cache (dht.Client.AnnounceSkill —
// the same real, production-wired call App.AnnounceSkill uses through
// DHTClientAdapter). This is a discovery announcement ("this skill exists,
// here's where to find it"), not a validator-consensus gossip broadcast —
// no generic pub/sub consensus mechanism exists in this codebase to call
// instead (KNIRVGATEWAY's DHTManager.PublishAnnouncement exists but isn't
// exposed over any HTTP route, and dht.Client.Publish/DHTClientAdapter's
// own Publish/Subscribe are unwired no-ops/404s).
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
	return nil
}
