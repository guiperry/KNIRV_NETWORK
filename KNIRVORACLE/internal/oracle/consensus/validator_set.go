package consensus

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE/internal/oracle/types"
)

// NewValidatorSet creates a new validator set
func NewValidatorSet(validators []*ConsensusValidator) *ValidatorSet {
	vs := &ValidatorSet{
		Validators:       make([]*ConsensusValidator, len(validators)),
		TotalVotingPower: big.NewInt(0),
	}

	// Copy validators
	for i, val := range validators {
		vs.Validators[i] = &ConsensusValidator{
			Address:          val.Address,
			PubKey:           val.PubKey,
			VotingPower:      new(big.Int).Set(val.VotingPower),
			ProposerPriority: big.NewInt(0),
		}
		vs.TotalVotingPower.Add(vs.TotalVotingPower, val.VotingPower)
	}

	// Sort validators by address
	vs.Sort()

	// Initialize proposer priorities
	vs.IncrementProposerPriority(1)

	return vs
}

// Size returns the number of validators
func (vs *ValidatorSet) Size() int {
	return len(vs.Validators)
}

// HasAddress checks if a validator with the given address exists
func (vs *ValidatorSet) HasAddress(address types.Address) bool {
	for _, val := range vs.Validators {
		if val.Address == address {
			return true
		}
	}
	return false
}

// GetByAddress returns a validator by address
func (vs *ValidatorSet) GetByAddress(address types.Address) (*ConsensusValidator, int) {
	for i, val := range vs.Validators {
		if val.Address == address {
			return val, i
		}
	}
	return nil, -1
}

// GetByIndex returns a validator by index
func (vs *ValidatorSet) GetByIndex(index int) *ConsensusValidator {
	if index < 0 || index >= len(vs.Validators) {
		return nil
	}
	return vs.Validators[index]
}

// GetProposer returns the current proposer
func (vs *ValidatorSet) GetProposer() *ConsensusValidator {
	if len(vs.Validators) == 0 {
		return nil
	}

	if vs.Proposer == nil {
		vs.Proposer = vs.findProposer()
	}

	return vs.Proposer
}

// IncrementProposerPriority increments proposer priorities and selects new proposer
func (vs *ValidatorSet) IncrementProposerPriority(times int32) {
	if len(vs.Validators) == 0 {
		return
	}

	for i := int32(0); i < times; i++ {
		vs.incrementProposerPriority()
	}

	vs.Proposer = vs.findProposer()
}

// incrementProposerPriority increments all validator priorities
func (vs *ValidatorSet) incrementProposerPriority() {
	for _, val := range vs.Validators {
		// Increment priority by voting power
		val.ProposerPriority.Add(val.ProposerPriority, val.VotingPower)
	}

	// Decrease proposer priority by total voting power
	proposer := vs.findProposer()
	if proposer != nil {
		proposer.ProposerPriority.Sub(proposer.ProposerPriority, vs.TotalVotingPower)
	}
}

// findProposer finds the validator with the highest proposer priority
func (vs *ValidatorSet) findProposer() *ConsensusValidator {
	if len(vs.Validators) == 0 {
		return nil
	}

	var proposer *ConsensusValidator
	maxPriority := big.NewInt(0).Neg(big.NewInt(1 << 62)) // Very negative number

	for _, val := range vs.Validators {
		if val.ProposerPriority.Cmp(maxPriority) > 0 {
			maxPriority = new(big.Int).Set(val.ProposerPriority)
			proposer = val
		}
	}

	return proposer
}

// Add adds a validator to the set
func (vs *ValidatorSet) Add(validator *ConsensusValidator) error {
	if vs.HasAddress(validator.Address) {
		return fmt.Errorf("validator already exists: %s", validator.Address.String())
	}

	// Add validator
	newVal := &ConsensusValidator{
		Address:          validator.Address,
		PubKey:           validator.PubKey,
		VotingPower:      new(big.Int).Set(validator.VotingPower),
		ProposerPriority: big.NewInt(0),
	}

	vs.Validators = append(vs.Validators, newVal)
	vs.TotalVotingPower.Add(vs.TotalVotingPower, validator.VotingPower)

	// Re-sort
	vs.Sort()

	// Reset proposer
	vs.Proposer = nil

	return nil
}

// Update updates a validator's voting power
func (vs *ValidatorSet) Update(validator *ConsensusValidator) error {
	val, idx := vs.GetByAddress(validator.Address)
	if val == nil {
		return fmt.Errorf("validator not found: %s", validator.Address.String())
	}

	// Update voting power
	oldPower := new(big.Int).Set(val.VotingPower)
	val.VotingPower = new(big.Int).Set(validator.VotingPower)
	val.PubKey = validator.PubKey

	// Update total voting power
	vs.TotalVotingPower.Sub(vs.TotalVotingPower, oldPower)
	vs.TotalVotingPower.Add(vs.TotalVotingPower, validator.VotingPower)

	// Update in array
	vs.Validators[idx] = val

	// Reset proposer
	vs.Proposer = nil

	return nil
}

// Remove removes a validator from the set
func (vs *ValidatorSet) Remove(address types.Address) error {
	val, idx := vs.GetByAddress(address)
	if val == nil {
		return fmt.Errorf("validator not found: %s", address.String())
	}

	// Remove from array
	vs.Validators = append(vs.Validators[:idx], vs.Validators[idx+1:]...)

	// Update total voting power
	vs.TotalVotingPower.Sub(vs.TotalVotingPower, val.VotingPower)

	// Reset proposer
	vs.Proposer = nil

	return nil
}

// Sort sorts validators by address
func (vs *ValidatorSet) Sort() {
	sort.Slice(vs.Validators, func(i, j int) bool {
		return compareAddresses(vs.Validators[i].Address, vs.Validators[j].Address) < 0
	})
}

// Copy returns a deep copy of the validator set
func (vs *ValidatorSet) Copy() *ValidatorSet {
	validators := make([]*ConsensusValidator, len(vs.Validators))
	for i, val := range vs.Validators {
		validators[i] = &ConsensusValidator{
			Address:          val.Address,
			PubKey:           val.PubKey,
			VotingPower:      new(big.Int).Set(val.VotingPower),
			ProposerPriority: new(big.Int).Set(val.ProposerPriority),
		}
	}

	newVS := &ValidatorSet{
		Validators:       validators,
		TotalVotingPower: new(big.Int).Set(vs.TotalVotingPower),
	}

	if vs.Proposer != nil {
		newVS.Proposer = &ConsensusValidator{
			Address:          vs.Proposer.Address,
			PubKey:           vs.Proposer.PubKey,
			VotingPower:      new(big.Int).Set(vs.Proposer.VotingPower),
			ProposerPriority: new(big.Int).Set(vs.Proposer.ProposerPriority),
		}
	}

	return newVS
}

// VerifyCommit verifies a commit against the validator set
func (vs *ValidatorSet) VerifyCommit(chainID string, blockID BlockID, height BlockHeight, commit *Commit) error {
	if commit.Height != height {
		return fmt.Errorf("commit height mismatch: expected %d, got %d", height, commit.Height)
	}

	// Count voting power
	votingPower := big.NewInt(0)
	seenVals := make(map[string]bool)

	for _, sig := range commit.Signatures {
		if sig.BlockIDFlag == BlockIDFlagAbsent {
			continue
		}

		// Check for duplicates
		addrStr := sig.ValidatorAddress.String()
		if seenVals[addrStr] {
			return fmt.Errorf("duplicate validator: %s", addrStr)
		}
		seenVals[addrStr] = true

		// Get validator
		val, _ := vs.GetByAddress(sig.ValidatorAddress)
		if val == nil {
			return fmt.Errorf("validator not in set: %s", addrStr)
		}

		// Add voting power
		votingPower.Add(votingPower, val.VotingPower)
	}

	// Check 2/3+ majority
	requiredPower := new(big.Int).Mul(vs.TotalVotingPower, big.NewInt(2))
	requiredPower.Div(requiredPower, big.NewInt(3))

	if votingPower.Cmp(requiredPower) <= 0 {
		return fmt.Errorf("insufficient voting power: %s <= %s", votingPower.String(), requiredPower.String())
	}

	return nil
}

// Helper: compare addresses
func compareAddresses(a, b types.Address) int {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
