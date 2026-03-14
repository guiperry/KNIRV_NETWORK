package governance

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle/crypto"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle/types"
	"go.uber.org/zap"
)

var (
	ErrProposalNotFound      = fmt.Errorf("proposal not found")
	ErrInvalidProposal       = fmt.Errorf("invalid proposal")
	ErrInsufficientDeposit   = fmt.Errorf("insufficient deposit")
	ErrVotingNotActive       = fmt.Errorf("voting period not active")
	ErrVotingEnded           = fmt.Errorf("voting period has ended")
	ErrAlreadyVoted          = fmt.Errorf("already voted on this proposal")
	ErrValidatorNotFound     = fmt.Errorf("validator not found")
	ErrValidatorNotActive    = fmt.Errorf("validator not active")
	ErrInsufficientStake     = fmt.Errorf("insufficient stake")
	ErrInvalidCommissionRate = fmt.Errorf("invalid commission rate")
	ErrMaxValidatorsReached  = fmt.Errorf("maximum validators reached")
)

// GovernanceSystem manages network governance
type GovernanceSystem struct {
	proposals  map[string]*Proposal
	votes      map[string]map[types.Address]*Vote // proposalID -> voter -> vote
	validators map[types.Address]*Validator
	params     *NetworkParameters
	logger     *zap.Logger
	mu         sync.RWMutex
}

// NewGovernanceSystem creates a new governance system
func NewGovernanceSystem(logger *zap.Logger) *GovernanceSystem {
	return &GovernanceSystem{
		proposals:  make(map[string]*Proposal),
		votes:      make(map[string]map[types.Address]*Vote),
		validators: make(map[types.Address]*Validator),
		params:     DefaultNetworkParameters(),
		logger:     logger,
	}
}

// CreateProposal creates a new governance proposal
func (gs *GovernanceSystem) CreateProposal(req *ProposalRequest, deposit *big.Int) (*Proposal, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	// Validate proposal
	if err := gs.validateProposalRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidProposal, err)
	}

	// Check deposit
	if deposit.Cmp(gs.params.ProposalDeposit) < 0 {
		return nil, ErrInsufficientDeposit
	}

	// Generate proposal ID
	proposalID, err := generateProposalID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate proposal ID: %w", err)
	}

	now := time.Now()
	proposal := &Proposal{
		ID:          proposalID,
		Type:        req.Type,
		Title:       req.Title,
		Description: req.Description,
		Proposer:    req.Proposer,
		Status:      ProposalStatusPending,
		SubmitTime:  now,
		VotingStart: now.Add(1 * time.Hour), // 1 hour delay before voting starts
		VotingEnd:   now.Add(1*time.Hour + time.Duration(gs.params.VotingPeriod)*5*time.Second),
		Deposit:     new(big.Int).Set(deposit),
		Content:     req.Content,
		TallyResult: NewTallyResult(),
	}

	gs.proposals[proposalID] = proposal
	gs.votes[proposalID] = make(map[types.Address]*Vote)

	gs.logger.Info("Proposal created",
		zap.String("proposal_id", proposalID),
		zap.String("type", proposal.Type.String()),
		zap.String("proposer", proposal.Proposer.String()),
	)

	return proposal, nil
}

// GetProposal retrieves a proposal by ID
func (gs *GovernanceSystem) GetProposal(proposalID string) (*Proposal, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	proposal, exists := gs.proposals[proposalID]
	if !exists {
		return nil, ErrProposalNotFound
	}

	return proposal, nil
}

// ListProposals returns all proposals
func (gs *GovernanceSystem) ListProposals() []*Proposal {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	proposals := make([]*Proposal, 0, len(gs.proposals))
	for _, proposal := range gs.proposals {
		proposals = append(proposals, proposal)
	}

	return proposals
}

// ListActiveProposals returns proposals currently in voting period
func (gs *GovernanceSystem) ListActiveProposals() []*Proposal {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	now := time.Now()
	proposals := make([]*Proposal, 0)
	for _, proposal := range gs.proposals {
		if proposal.Status == ProposalStatusActive ||
			(proposal.Status == ProposalStatusPending && now.After(proposal.VotingStart)) {
			proposals = append(proposals, proposal)
		}
	}

	return proposals
}

// UpdateProposalStatus updates proposal status based on current time and votes
func (gs *GovernanceSystem) UpdateProposalStatus(proposalID string) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	proposal, exists := gs.proposals[proposalID]
	if !exists {
		return ErrProposalNotFound
	}

	now := time.Now()

	// Transition from Pending to Active
	if proposal.Status == ProposalStatusPending && now.After(proposal.VotingStart) {
		proposal.Status = ProposalStatusActive
		gs.logger.Info("Proposal voting started",
			zap.String("proposal_id", proposalID),
		)
	}

	// Check if voting period ended
	if proposal.Status == ProposalStatusActive && now.After(proposal.VotingEnd) {
		// Tally votes and determine outcome
		totalStake := gs.getTotalActiveStake()
		passed := gs.evaluateProposal(proposal, totalStake)

		if passed {
			proposal.Status = ProposalStatusPassed
			gs.logger.Info("Proposal passed",
				zap.String("proposal_id", proposalID),
				zap.Float64("pass_rate", proposal.TallyResult.PassRate()),
			)
		} else {
			proposal.Status = ProposalStatusRejected
			gs.logger.Info("Proposal rejected",
				zap.String("proposal_id", proposalID),
				zap.Float64("pass_rate", proposal.TallyResult.PassRate()),
			)
		}
	}

	return nil
}

// CancelProposal cancels a proposal (only by proposer before voting starts)
func (gs *GovernanceSystem) CancelProposal(proposalID string, requester types.Address) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	proposal, exists := gs.proposals[proposalID]
	if !exists {
		return ErrProposalNotFound
	}

	// Can only cancel pending proposals
	if proposal.Status != ProposalStatusPending {
		return fmt.Errorf("can only cancel pending proposals")
	}

	// Only proposer can cancel
	if proposal.Proposer != requester {
		return fmt.Errorf("only proposer can cancel proposal")
	}

	proposal.Status = ProposalStatusCancelled

	gs.logger.Info("Proposal cancelled",
		zap.String("proposal_id", proposalID),
		zap.String("proposer", requester.String()),
	)

	return nil
}

// GetNetworkParameters returns current network parameters
func (gs *GovernanceSystem) GetNetworkParameters() *NetworkParameters {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return gs.params
}

// UpdateNetworkParameters updates network parameters (through governance)
func (gs *GovernanceSystem) UpdateNetworkParameters(params *NetworkParameters) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	// Validate parameters
	if err := gs.validateNetworkParameters(params); err != nil {
		return err
	}

	gs.params = params

	gs.logger.Info("Network parameters updated")

	return nil
}

// Helper: validate proposal request
func (gs *GovernanceSystem) validateProposalRequest(req *ProposalRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if req.Description == "" {
		return fmt.Errorf("description cannot be empty")
	}
	if req.Proposer.IsZero() {
		return fmt.Errorf("proposer address cannot be zero")
	}
	if req.Content == nil {
		return fmt.Errorf("content cannot be nil")
	}

	// Type-specific validation
	switch req.Type {
	case ProposalTypeNetworkParameter:
		// Validate parameter update content
		if _, ok := req.Content["parameters"]; !ok {
			return fmt.Errorf("network parameter proposal must include 'parameters' field")
		}
	case ProposalTypeValidatorSet:
		// Validate validator set change content
		if _, ok := req.Content["validator"]; !ok {
			return fmt.Errorf("validator set proposal must include 'validator' field")
		}
	}

	return nil
}

// Helper: evaluate if proposal passed
func (gs *GovernanceSystem) evaluateProposal(proposal *Proposal, totalStake *big.Int) bool {
	tally := proposal.TallyResult

	// Check quorum (participation threshold)
	participationRate := tally.ParticipationRate(totalStake)
	if participationRate < gs.params.QuorumThreshold {
		gs.logger.Info("Proposal failed quorum",
			zap.String("proposal_id", proposal.ID),
			zap.Float64("participation", participationRate),
			zap.Float64("required", gs.params.QuorumThreshold),
		)
		return false
	}

	// Check veto threshold
	vetoRate := tally.VetoRate()
	if vetoRate >= gs.params.VetoThreshold {
		gs.logger.Info("Proposal vetoed",
			zap.String("proposal_id", proposal.ID),
			zap.Float64("veto_rate", vetoRate),
		)
		return false
	}

	// Check pass threshold
	passRate := tally.PassRate()
	if passRate < gs.params.PassThreshold {
		gs.logger.Info("Proposal failed pass threshold",
			zap.String("proposal_id", proposal.ID),
			zap.Float64("pass_rate", passRate),
			zap.Float64("required", gs.params.PassThreshold),
		)
		return false
	}

	return true
}

// Helper: get total active stake
func (gs *GovernanceSystem) getTotalActiveStake() *big.Int {
	total := big.NewInt(0)
	for _, validator := range gs.validators {
		if validator.Active && !validator.Jailed {
			total.Add(total, validator.StakedAmount)
		}
	}
	return total
}

// Helper: validate network parameters
func (gs *GovernanceSystem) validateNetworkParameters(params *NetworkParameters) error {
	if params.QuorumThreshold < 0 || params.QuorumThreshold > 1 {
		return fmt.Errorf("quorum threshold must be between 0 and 1")
	}
	if params.PassThreshold < 0 || params.PassThreshold > 1 {
		return fmt.Errorf("pass threshold must be between 0 and 1")
	}
	if params.VetoThreshold < 0 || params.VetoThreshold > 1 {
		return fmt.Errorf("veto threshold must be between 0 and 1")
	}
	if params.SlashingPercentage < 0 || params.SlashingPercentage > 1 {
		return fmt.Errorf("slashing percentage must be between 0 and 1")
	}
	if params.ProposalDeposit.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("proposal deposit must be positive")
	}
	if params.MinValidatorStake.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("minimum validator stake must be positive")
	}

	return nil
}

// Helper: generate random proposal ID
func generateProposalID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	hash := crypto.Keccak256(bytes)
	return hex.EncodeToString(hash[:8]), nil
}
