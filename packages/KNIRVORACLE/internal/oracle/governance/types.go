package governance

import (
	"math/big"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// ProposalType represents different types of governance proposals
type ProposalType int

const (
	ProposalTypeNetworkParameter ProposalType = iota
	ProposalTypeValidatorSet
	ProposalTypeTokenEconomics
	ProposalTypeCrossChainConfig
	ProposalTypeEmergencyAction
	ProposalTypeTextProposal
)

func (pt ProposalType) String() string {
	switch pt {
	case ProposalTypeNetworkParameter:
		return "NetworkParameter"
	case ProposalTypeValidatorSet:
		return "ValidatorSet"
	case ProposalTypeTokenEconomics:
		return "TokenEconomics"
	case ProposalTypeCrossChainConfig:
		return "CrossChainConfig"
	case ProposalTypeEmergencyAction:
		return "EmergencyAction"
	case ProposalTypeTextProposal:
		return "TextProposal"
	default:
		return "Unknown"
	}
}

// ProposalStatus represents the current state of a proposal
type ProposalStatus int

const (
	ProposalStatusPending ProposalStatus = iota
	ProposalStatusActive
	ProposalStatusPassed
	ProposalStatusRejected
	ProposalStatusExecuted
	ProposalStatusFailed
	ProposalStatusCancelled
)

func (ps ProposalStatus) String() string {
	switch ps {
	case ProposalStatusPending:
		return "Pending"
	case ProposalStatusActive:
		return "Active"
	case ProposalStatusPassed:
		return "Passed"
	case ProposalStatusRejected:
		return "Rejected"
	case ProposalStatusExecuted:
		return "Executed"
	case ProposalStatusFailed:
		return "Failed"
	case ProposalStatusCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// VoteOption represents a vote choice
type VoteOption int

const (
	VoteOptionYes VoteOption = iota
	VoteOptionNo
	VoteOptionAbstain
	VoteOptionVeto
)

func (vo VoteOption) String() string {
	switch vo {
	case VoteOptionYes:
		return "Yes"
	case VoteOptionNo:
		return "No"
	case VoteOptionAbstain:
		return "Abstain"
	case VoteOptionVeto:
		return "Veto"
	default:
		return "Unknown"
	}
}

// Proposal represents a governance proposal
type Proposal struct {
	ID            string                 `json:"id"`
	Type          ProposalType           `json:"type"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Proposer      types.Address          `json:"proposer"`
	Status        ProposalStatus         `json:"status"`
	SubmitTime    time.Time              `json:"submit_time"`
	VotingStart   time.Time              `json:"voting_start"`
	VotingEnd     time.Time              `json:"voting_end"`
	ExecutionTime *time.Time             `json:"execution_time,omitempty"`
	Deposit       *big.Int               `json:"deposit"`
	Content       map[string]interface{} `json:"content"`
	TallyResult   *TallyResult           `json:"tally_result,omitempty"`
}

// TallyResult contains the results of a proposal vote
type TallyResult struct {
	Yes     *big.Int `json:"yes"`
	No      *big.Int `json:"no"`
	Abstain *big.Int `json:"abstain"`
	Veto    *big.Int `json:"veto"`
	Total   *big.Int `json:"total"`
}

// NewTallyResult creates a new zero-initialized tally result
func NewTallyResult() *TallyResult {
	return &TallyResult{
		Yes:     big.NewInt(0),
		No:      big.NewInt(0),
		Abstain: big.NewInt(0),
		Veto:    big.NewInt(0),
		Total:   big.NewInt(0),
	}
}

// PassRate returns the percentage of Yes votes out of total votes (excluding abstentions)
func (tr *TallyResult) PassRate() float64 {
	activeVotes := new(big.Int).Add(tr.Yes, tr.No)
	activeVotes.Add(activeVotes, tr.Veto)
	if activeVotes.Cmp(big.NewInt(0)) == 0 {
		return 0.0
	}
	yesFloat := new(big.Float).SetInt(tr.Yes)
	activeFloat := new(big.Float).SetInt(activeVotes)
	result, _ := new(big.Float).Quo(yesFloat, activeFloat).Float64()
	return result
}

// VetoRate returns the percentage of Veto votes out of total votes
func (tr *TallyResult) VetoRate() float64 {
	if tr.Total.Cmp(big.NewInt(0)) == 0 {
		return 0.0
	}
	vetoFloat := new(big.Float).SetInt(tr.Veto)
	totalFloat := new(big.Float).SetInt(tr.Total)
	result, _ := new(big.Float).Quo(vetoFloat, totalFloat).Float64()
	return result
}

// ParticipationRate returns the percentage of total stake that participated
func (tr *TallyResult) ParticipationRate(totalStake *big.Int) float64 {
	if totalStake.Cmp(big.NewInt(0)) == 0 {
		return 0.0
	}
	totalFloat := new(big.Float).SetInt(tr.Total)
	stakeFloat := new(big.Float).SetInt(totalStake)
	result, _ := new(big.Float).Quo(totalFloat, stakeFloat).Float64()
	return result
}

// Vote represents a validator's vote on a proposal
type Vote struct {
	ProposalID string        `json:"proposal_id"`
	Voter      types.Address `json:"voter"`
	Option     VoteOption    `json:"option"`
	Weight     *big.Int      `json:"weight"`
	Timestamp  time.Time     `json:"timestamp"`
	Signature  []byte        `json:"signature"`
}

// Validator represents a network validator
type Validator struct {
	Address        types.Address `json:"address"`
	PubKey         []byte        `json:"pub_key"`
	VotingPower    *big.Int      `json:"voting_power"`
	StakedAmount   *big.Int      `json:"staked_amount"`
	CommissionRate float64       `json:"commission_rate"`
	Active         bool          `json:"active"`
	Jailed         bool          `json:"jailed"`
	JailTime       *time.Time    `json:"jail_time,omitempty"`
	JoinedAt       time.Time     `json:"joined_at"`
	LastActiveAt   time.Time     `json:"last_active_at"`
}

// NetworkParameters represents adjustable network parameters
type NetworkParameters struct {
	// Governance parameters
	ProposalDeposit *big.Int `json:"proposal_deposit"`
	VotingPeriod    uint64   `json:"voting_period"`    // in blocks
	QuorumThreshold float64  `json:"quorum_threshold"` // 0-1
	PassThreshold   float64  `json:"pass_threshold"`   // 0-1
	VetoThreshold   float64  `json:"veto_threshold"`   // 0-1

	// Validator parameters
	MinValidatorStake  *big.Int `json:"min_validator_stake"`
	MaxValidators      uint32   `json:"max_validators"`
	UnbondingPeriod    uint64   `json:"unbonding_period"`    // in blocks
	SlashingPercentage float64  `json:"slashing_percentage"` // 0-1

	// Economic parameters
	BaseFeeRate float64 `json:"base_fee_rate"` // 0-1
	BurnRate    float64 `json:"burn_rate"`     // 0-1
	RewardRate  float64 `json:"reward_rate"`   // 0-1

	// Cross-chain parameters
	IBCTimeoutBlocks  uint64   `json:"ibc_timeout_blocks"`
	MinTransferAmount *big.Int `json:"min_transfer_amount"`
	MaxTransferAmount *big.Int `json:"max_transfer_amount"`
}

// DefaultNetworkParameters returns the default network parameters
func DefaultNetworkParameters() *NetworkParameters {
	return &NetworkParameters{
		ProposalDeposit:    big.NewInt(100000000), // 100 NRN
		VotingPeriod:       50400,                 // ~7 days at 5s/block
		QuorumThreshold:    0.33,
		PassThreshold:      0.5,
		VetoThreshold:      0.33,
		MinValidatorStake:  big.NewInt(1000000000), // 1000 NRN
		MaxValidators:      100,
		UnbondingPeriod:    201600, // ~28 days
		SlashingPercentage: 0.05,
		BaseFeeRate:        0.001,
		BurnRate:           0.5,
		RewardRate:         0.5,
		IBCTimeoutBlocks:   1000,
		MinTransferAmount:  big.NewInt(100),
		MaxTransferAmount:  big.NewInt(1000000000000), // 1M NRN
	}
}

// ProposalRequest is used to create a new proposal
type ProposalRequest struct {
	Type        ProposalType           `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Proposer    types.Address          `json:"proposer"`
	Content     map[string]interface{} `json:"content"`
}

// VoteRequest is used to cast a vote
type VoteRequest struct {
	ProposalID string        `json:"proposal_id"`
	Voter      types.Address `json:"voter"`
	Option     VoteOption    `json:"option"`
}

// ValidatorRegistration is used to register a new validator
type ValidatorRegistration struct {
	Address        types.Address `json:"address"`
	PubKey         []byte        `json:"pub_key"`
	StakedAmount   *big.Int      `json:"staked_amount"`
	CommissionRate float64       `json:"commission_rate"`
}
