package consensus

import (
	"math/big"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE/internal/oracle/types"
)

// BlockHeight represents a block height
type BlockHeight uint64

// VoteType represents the type of vote
type VoteType int

const (
	VoteTypePrevote VoteType = iota
	VoteTypePrecommit
)

func (vt VoteType) String() string {
	switch vt {
	case VoteTypePrevote:
		return "Prevote"
	case VoteTypePrecommit:
		return "Precommit"
	default:
		return "Unknown"
	}
}

// RoundStep represents the current step in consensus
type RoundStep int

const (
	RoundStepNewHeight RoundStep = iota
	RoundStepNewRound
	RoundStepPropose
	RoundStepPrevote
	RoundStepPrevoteWait
	RoundStepPrecommit
	RoundStepPrecommitWait
	RoundStepCommit
)

func (rs RoundStep) String() string {
	switch rs {
	case RoundStepNewHeight:
		return "NewHeight"
	case RoundStepNewRound:
		return "NewRound"
	case RoundStepPropose:
		return "Propose"
	case RoundStepPrevote:
		return "Prevote"
	case RoundStepPrevoteWait:
		return "PrevoteWait"
	case RoundStepPrecommit:
		return "Precommit"
	case RoundStepPrecommitWait:
		return "PrecommitWait"
	case RoundStepCommit:
		return "Commit"
	default:
		return "Unknown"
	}
}

// BlockID uniquely identifies a block
type BlockID struct {
	Hash          []byte        `json:"hash"`
	PartSetHeader PartSetHeader `json:"part_set_header"`
}

// PartSetHeader represents a part set header
type PartSetHeader struct {
	Total uint32 `json:"total"`
	Hash  []byte `json:"hash"`
}

// Block represents a blockchain block
type Block struct {
	Header     BlockHeader  `json:"header"`
	Data       BlockData    `json:"data"`
	Evidence   EvidenceData `json:"evidence"`
	LastCommit *Commit      `json:"last_commit,omitempty"`
}

// BlockHeader contains block metadata
type BlockHeader struct {
	Version            ConsensusVersion `json:"version"`
	ChainID            string           `json:"chain_id"`
	Height             BlockHeight      `json:"height"`
	Time               time.Time        `json:"time"`
	LastBlockID        BlockID          `json:"last_block_id"`
	LastCommitHash     []byte           `json:"last_commit_hash"`
	DataHash           []byte           `json:"data_hash"`
	ValidatorsHash     []byte           `json:"validators_hash"`
	NextValidatorsHash []byte           `json:"next_validators_hash"`
	ConsensusHash      []byte           `json:"consensus_hash"`
	AppHash            []byte           `json:"app_hash"`
	LastResultsHash    []byte           `json:"last_results_hash"`
	EvidenceHash       []byte           `json:"evidence_hash"`
	ProposerAddress    types.Address    `json:"proposer_address"`
}

// ConsensusVersion represents the consensus version
type ConsensusVersion struct {
	Block uint64 `json:"block"`
	App   uint64 `json:"app"`
}

// BlockData contains the block transactions
type BlockData struct {
	Txs [][]byte `json:"txs"`
}

// EvidenceData contains evidence of misbehavior
type EvidenceData struct {
	Evidence []Evidence `json:"evidence"`
}

// Evidence represents evidence of misbehavior
type Evidence interface {
	Height() BlockHeight
	Bytes() []byte
	Hash() []byte
	ValidateBasic() error
}

// Commit represents a commit for a block
type Commit struct {
	Height     BlockHeight `json:"height"`
	Round      int32       `json:"round"`
	BlockID    BlockID     `json:"block_id"`
	Signatures []CommitSig `json:"signatures"`
}

// CommitSig represents a validator's signature on a commit
type CommitSig struct {
	BlockIDFlag      BlockIDFlag   `json:"block_id_flag"`
	ValidatorAddress types.Address `json:"validator_address"`
	Timestamp        time.Time     `json:"timestamp"`
	Signature        []byte        `json:"signature"`
}

// BlockIDFlag indicates which BlockID the signature is for
type BlockIDFlag byte

const (
	BlockIDFlagAbsent BlockIDFlag = iota
	BlockIDFlagCommit
	BlockIDFlagNil
)

// Vote represents a prevote or precommit vote
type Vote struct {
	Type             VoteType      `json:"type"`
	Height           BlockHeight   `json:"height"`
	Round            int32         `json:"round"`
	BlockID          BlockID       `json:"block_id"`
	Timestamp        time.Time     `json:"timestamp"`
	ValidatorAddress types.Address `json:"validator_address"`
	ValidatorIndex   int32         `json:"validator_index"`
	Signature        []byte        `json:"signature"`
}

// Proposal represents a block proposal
type Proposal struct {
	Type      byte        `json:"type"`
	Height    BlockHeight `json:"height"`
	Round     int32       `json:"round"`
	POLRound  int32       `json:"pol_round"`
	BlockID   BlockID     `json:"block_id"`
	Timestamp time.Time   `json:"timestamp"`
	Signature []byte      `json:"signature"`
}

// ConsensusValidator represents a validator in the consensus
type ConsensusValidator struct {
	Address          types.Address `json:"address"`
	PubKey           []byte        `json:"pub_key"`
	VotingPower      *big.Int      `json:"voting_power"`
	ProposerPriority *big.Int      `json:"proposer_priority"`
}

// ValidatorSet represents a set of validators
type ValidatorSet struct {
	Validators       []*ConsensusValidator `json:"validators"`
	Proposer         *ConsensusValidator   `json:"proposer"`
	TotalVotingPower *big.Int              `json:"total_voting_power"`
}

// RoundState represents the current state of consensus
type RoundState struct {
	Height                    BlockHeight   `json:"height"`
	Round                     int32         `json:"round"`
	Step                      RoundStep     `json:"step"`
	StartTime                 time.Time     `json:"start_time"`
	CommitTime                time.Time     `json:"commit_time"`
	Validators                *ValidatorSet `json:"validators"`
	Proposal                  *Proposal     `json:"proposal,omitempty"`
	ProposalBlock             *Block        `json:"proposal_block,omitempty"`
	ProposalBlockParts        interface{}   `json:"proposal_block_parts,omitempty"`
	LockedRound               int32         `json:"locked_round"`
	LockedBlock               *Block        `json:"locked_block,omitempty"`
	ValidRound                int32         `json:"valid_round"`
	ValidBlock                *Block        `json:"valid_block,omitempty"`
	Votes                     interface{}   `json:"votes,omitempty"`
	CommitRound               int32         `json:"commit_round"`
	LastCommit                *Commit       `json:"last_commit,omitempty"`
	LastValidators            *ValidatorSet `json:"last_validators,omitempty"`
	TriggeredTimeoutPrecommit bool          `json:"triggered_timeout_precommit"`
}

// ConsensusParams represents consensus parameters
type ConsensusParams struct {
	Block     BlockParams     `json:"block"`
	Evidence  EvidenceParams  `json:"evidence"`
	Validator ValidatorParams `json:"validator"`
	Version   VersionParams   `json:"version"`
}

// BlockParams defines limits on the block size
type BlockParams struct {
	MaxBytes int64 `json:"max_bytes"`
	MaxGas   int64 `json:"max_gas"`
}

// EvidenceParams defines limits on evidence
type EvidenceParams struct {
	MaxAgeNumBlocks int64         `json:"max_age_num_blocks"`
	MaxAgeDuration  time.Duration `json:"max_age_duration"`
	MaxBytes        int64         `json:"max_bytes"`
}

// ValidatorParams defines limits on validators
type ValidatorParams struct {
	PubKeyTypes []string `json:"pub_key_types"`
}

// VersionParams defines the ABCI version
type VersionParams struct {
	App uint64 `json:"app"`
}

// DefaultConsensusParams returns default consensus parameters
func DefaultConsensusParams() *ConsensusParams {
	return &ConsensusParams{
		Block: BlockParams{
			MaxBytes: 1048576,  // 1 MB
			MaxGas:   10000000, // 10M gas
		},
		Evidence: EvidenceParams{
			MaxAgeNumBlocks: 100000,
			MaxAgeDuration:  48 * time.Hour,
			MaxBytes:        1048576, // 1 MB
		},
		Validator: ValidatorParams{
			PubKeyTypes: []string{"ed25519", "secp256k1"},
		},
		Version: VersionParams{
			App: 1,
		},
	}
}

// State represents the blockchain state
type State struct {
	Version                          ConsensusVersion `json:"version"`
	ChainID                          string           `json:"chain_id"`
	InitialHeight                    BlockHeight      `json:"initial_height"`
	LastBlockHeight                  BlockHeight      `json:"last_block_height"`
	LastBlockID                      BlockID          `json:"last_block_id"`
	LastBlockTime                    time.Time        `json:"last_block_time"`
	NextValidators                   *ValidatorSet    `json:"next_validators"`
	Validators                       *ValidatorSet    `json:"validators"`
	LastValidators                   *ValidatorSet    `json:"last_validators"`
	LastHeightValidatorsChanged      BlockHeight      `json:"last_height_validators_changed"`
	ConsensusParams                  ConsensusParams  `json:"consensus_params"`
	LastHeightConsensusParamsChanged BlockHeight      `json:"last_height_consensus_params_changed"`
	LastResultsHash                  []byte           `json:"last_results_hash"`
	AppHash                          []byte           `json:"app_hash"`
}
