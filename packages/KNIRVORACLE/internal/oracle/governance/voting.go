package governance

import (
	"fmt"
	"math/big"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// CastVote casts a vote on a proposal
func (gs *GovernanceSystem) CastVote(req *VoteRequest, privateKey string) (*Vote, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	// Get proposal
	proposal, exists := gs.proposals[req.ProposalID]
	if !exists {
		return nil, ErrProposalNotFound
	}

	// Check voting period
	now := time.Now()
	if now.Before(proposal.VotingStart) {
		return nil, ErrVotingNotActive
	}
	if now.After(proposal.VotingEnd) {
		return nil, ErrVotingEnded
	}

	// Update proposal status if needed
	if proposal.Status == ProposalStatusPending {
		proposal.Status = ProposalStatusActive
	}

	// Check if already voted
	if _, exists := gs.votes[req.ProposalID][req.Voter]; exists {
		return nil, ErrAlreadyVoted
	}

	// Get validator
	validator, exists := gs.validators[req.Voter]
	if !exists {
		return nil, ErrValidatorNotFound
	}

	// Check validator is active
	if !validator.Active || validator.Jailed {
		return nil, ErrValidatorNotActive
	}

	// Create vote with stake weight
	vote := &Vote{
		ProposalID: req.ProposalID,
		Voter:      req.Voter,
		Option:     req.Option,
		Weight:     new(big.Int).Set(validator.StakedAmount),
		Timestamp:  now,
	}

	// Sign vote
	voteData := fmt.Sprintf("%s:%s:%s:%s",
		vote.ProposalID,
		vote.Voter.String(),
		vote.Option.String(),
		vote.Weight.String(),
	)
	signatureHex, err := crypto.SignMessage(privateKey, []byte(voteData))
	if err != nil {
		return nil, fmt.Errorf("failed to sign vote: %w", err)
	}
	// Convert hex signature to bytes
	vote.Signature = []byte(signatureHex)

	// Store vote
	gs.votes[req.ProposalID][req.Voter] = vote

	// Update tally
	gs.updateTally(proposal, vote)

	gs.logger.Info("Vote cast",
		zap.String("proposal_id", req.ProposalID),
		zap.String("voter", req.Voter.String()),
		zap.String("option", req.Option.String()),
		zap.String("weight", vote.Weight.String()),
	)

	return vote, nil
}

// GetVote retrieves a vote
func (gs *GovernanceSystem) GetVote(proposalID string, voter types.Address) (*Vote, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	proposalVotes, exists := gs.votes[proposalID]
	if !exists {
		return nil, ErrProposalNotFound
	}

	vote, exists := proposalVotes[voter]
	if !exists {
		return nil, fmt.Errorf("vote not found")
	}

	return vote, nil
}

// GetProposalVotes returns all votes for a proposal
func (gs *GovernanceSystem) GetProposalVotes(proposalID string) ([]*Vote, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	proposalVotes, exists := gs.votes[proposalID]
	if !exists {
		return nil, ErrProposalNotFound
	}

	votes := make([]*Vote, 0, len(proposalVotes))
	for _, vote := range proposalVotes {
		votes = append(votes, vote)
	}

	return votes, nil
}

// HasVoted checks if an address has voted on a proposal
func (gs *GovernanceSystem) HasVoted(proposalID string, voter types.Address) bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	proposalVotes, exists := gs.votes[proposalID]
	if !exists {
		return false
	}

	_, exists = proposalVotes[voter]
	return exists
}

// TallyVotes recalculates tally for a proposal
func (gs *GovernanceSystem) TallyVotes(proposalID string) (*TallyResult, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	proposal, exists := gs.proposals[proposalID]
	if !exists {
		return nil, ErrProposalNotFound
	}

	// Reset tally
	tally := NewTallyResult()

	// Sum votes by option
	proposalVotes := gs.votes[proposalID]
	for _, vote := range proposalVotes {
		switch vote.Option {
		case VoteOptionYes:
			tally.Yes.Add(tally.Yes, vote.Weight)
		case VoteOptionNo:
			tally.No.Add(tally.No, vote.Weight)
		case VoteOptionAbstain:
			tally.Abstain.Add(tally.Abstain, vote.Weight)
		case VoteOptionVeto:
			tally.Veto.Add(tally.Veto, vote.Weight)
		}
		tally.Total.Add(tally.Total, vote.Weight)
	}

	proposal.TallyResult = tally

	gs.logger.Info("Votes tallied",
		zap.String("proposal_id", proposalID),
		zap.String("yes", tally.Yes.String()),
		zap.String("no", tally.No.String()),
		zap.String("abstain", tally.Abstain.String()),
		zap.String("veto", tally.Veto.String()),
		zap.String("total", tally.Total.String()),
	)

	return tally, nil
}

// GetVoteStats returns voting statistics for a proposal
func (gs *GovernanceSystem) GetVoteStats(proposalID string) (map[string]interface{}, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	proposal, exists := gs.proposals[proposalID]
	if !exists {
		return nil, ErrProposalNotFound
	}

	proposalVotes := gs.votes[proposalID]
	totalStake := gs.getTotalActiveStake()

	stats := map[string]interface{}{
		"proposal_id":        proposalID,
		"status":             proposal.Status.String(),
		"total_voters":       len(proposalVotes),
		"total_validators":   len(gs.validators),
		"yes":                proposal.TallyResult.Yes.String(),
		"no":                 proposal.TallyResult.No.String(),
		"abstain":            proposal.TallyResult.Abstain.String(),
		"veto":               proposal.TallyResult.Veto.String(),
		"total_votes":        proposal.TallyResult.Total.String(),
		"total_stake":        totalStake.String(),
		"pass_rate":          proposal.TallyResult.PassRate(),
		"veto_rate":          proposal.TallyResult.VetoRate(),
		"participation_rate": proposal.TallyResult.ParticipationRate(totalStake),
		"quorum_threshold":   gs.params.QuorumThreshold,
		"pass_threshold":     gs.params.PassThreshold,
		"veto_threshold":     gs.params.VetoThreshold,
	}

	// Check if thresholds met
	participationRate := proposal.TallyResult.ParticipationRate(totalStake)
	stats["quorum_met"] = participationRate >= gs.params.QuorumThreshold
	stats["pass_threshold_met"] = proposal.TallyResult.PassRate() >= gs.params.PassThreshold
	stats["veto_threshold_met"] = proposal.TallyResult.VetoRate() >= gs.params.VetoThreshold

	return stats, nil
}

// VerifyVote verifies a vote signature
func (gs *GovernanceSystem) VerifyVote(vote *Vote) (bool, error) {
	// Get validator
	validator, exists := gs.validators[vote.Voter]
	if !exists {
		return false, ErrValidatorNotFound
	}

	// Reconstruct vote data
	voteData := fmt.Sprintf("%s:%s:%s:%s",
		vote.ProposalID,
		vote.Voter.String(),
		vote.Option.String(),
		vote.Weight.String(),
	)
	voteHash := crypto.Keccak256([]byte(voteData))

	// Verify signature
	valid := crypto.VerifySignature(validator.PubKey, voteHash, vote.Signature)
	return valid, nil
}

// Helper: update tally with new vote
func (gs *GovernanceSystem) updateTally(proposal *Proposal, vote *Vote) {
	switch vote.Option {
	case VoteOptionYes:
		proposal.TallyResult.Yes.Add(proposal.TallyResult.Yes, vote.Weight)
	case VoteOptionNo:
		proposal.TallyResult.No.Add(proposal.TallyResult.No, vote.Weight)
	case VoteOptionAbstain:
		proposal.TallyResult.Abstain.Add(proposal.TallyResult.Abstain, vote.Weight)
	case VoteOptionVeto:
		proposal.TallyResult.Veto.Add(proposal.TallyResult.Veto, vote.Weight)
	}
	proposal.TallyResult.Total.Add(proposal.TallyResult.Total, vote.Weight)
}
