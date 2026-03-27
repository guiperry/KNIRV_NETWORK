package governance

import (
	"fmt"
	"math/big"
	"time"

	"backend_server/internal/oracle/types"
	"go.uber.org/zap"
)

// RegisterValidator registers a new validator
func (gs *GovernanceSystem) RegisterValidator(reg *ValidatorRegistration) (*Validator, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	// Check if validator already exists
	if _, exists := gs.validators[reg.Address]; exists {
		return nil, fmt.Errorf("validator already registered")
	}

	// Check minimum stake
	if reg.StakedAmount.Cmp(gs.params.MinValidatorStake) < 0 {
		return nil, ErrInsufficientStake
	}

	// Check commission rate
	if reg.CommissionRate < 0 || reg.CommissionRate > 1 {
		return nil, ErrInvalidCommissionRate
	}

	// Check max validators
	activeCount := gs.getActiveValidatorCount()
	if activeCount >= int(gs.params.MaxValidators) {
		return nil, ErrMaxValidatorsReached
	}

	now := time.Now()
	validator := &Validator{
		Address:        reg.Address,
		PubKey:         reg.PubKey,
		VotingPower:    gs.calculateVotingPower(reg.StakedAmount),
		StakedAmount:   new(big.Int).Set(reg.StakedAmount),
		CommissionRate: reg.CommissionRate,
		Active:         true,
		Jailed:         false,
		JoinedAt:       now,
		LastActiveAt:   now,
	}

	gs.validators[reg.Address] = validator

	gs.logger.Info("Validator registered",
		zap.String("address", validator.Address.String()),
		zap.String("staked_amount", validator.StakedAmount.String()),
		zap.String("voting_power", validator.VotingPower.String()),
	)

	return validator, nil
}

// GetValidator retrieves a validator by address
func (gs *GovernanceSystem) GetValidator(address types.Address) (*Validator, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	validator, exists := gs.validators[address]
	if !exists {
		return nil, ErrValidatorNotFound
	}

	return validator, nil
}

// ListValidators returns all validators
func (gs *GovernanceSystem) ListValidators() []*Validator {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	validators := make([]*Validator, 0, len(gs.validators))
	for _, validator := range gs.validators {
		validators = append(validators, validator)
	}

	return validators
}

// ListActiveValidators returns only active validators
func (gs *GovernanceSystem) ListActiveValidators() []*Validator {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	validators := make([]*Validator, 0)
	for _, validator := range gs.validators {
		if validator.Active && !validator.Jailed {
			validators = append(validators, validator)
		}
	}

	return validators
}

// UpdateValidatorStake updates a validator's stake and recalculates voting power
func (gs *GovernanceSystem) UpdateValidatorStake(address types.Address, newStake *big.Int) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	validator, exists := gs.validators[address]
	if !exists {
		return ErrValidatorNotFound
	}

	// Check minimum stake
	if newStake.Cmp(gs.params.MinValidatorStake) < 0 {
		return ErrInsufficientStake
	}

	oldStake := new(big.Int).Set(validator.StakedAmount)
	validator.StakedAmount = new(big.Int).Set(newStake)
	validator.VotingPower = gs.calculateVotingPower(newStake)
	validator.LastActiveAt = time.Now()

	gs.logger.Info("Validator stake updated",
		zap.String("address", address.String()),
		zap.String("old_stake", oldStake.String()),
		zap.String("new_stake", newStake.String()),
		zap.String("voting_power", validator.VotingPower.String()),
	)

	return nil
}

// ActivateValidator activates a validator
func (gs *GovernanceSystem) ActivateValidator(address types.Address) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	validator, exists := gs.validators[address]
	if !exists {
		return ErrValidatorNotFound
	}

	if validator.Jailed {
		return fmt.Errorf("cannot activate jailed validator")
	}

	validator.Active = true
	validator.LastActiveAt = time.Now()

	gs.logger.Info("Validator activated",
		zap.String("address", address.String()),
	)

	return nil
}

// DeactivateValidator deactivates a validator
func (gs *GovernanceSystem) DeactivateValidator(address types.Address) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	validator, exists := gs.validators[address]
	if !exists {
		return ErrValidatorNotFound
	}

	validator.Active = false

	gs.logger.Info("Validator deactivated",
		zap.String("address", address.String()),
	)

	return nil
}

// JailValidator jails a validator (punishment for misbehavior)
func (gs *GovernanceSystem) JailValidator(address types.Address, reason string) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	validator, exists := gs.validators[address]
	if !exists {
		return ErrValidatorNotFound
	}

	now := time.Now()
	validator.Jailed = true
	validator.Active = false
	validator.JailTime = &now

	gs.logger.Warn("Validator jailed",
		zap.String("address", address.String()),
		zap.String("reason", reason),
	)

	return nil
}

// UnjailValidator unjails a validator after unbonding period
func (gs *GovernanceSystem) UnjailValidator(address types.Address) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	validator, exists := gs.validators[address]
	if !exists {
		return ErrValidatorNotFound
	}

	if !validator.Jailed {
		return fmt.Errorf("validator is not jailed")
	}

	// Check unbonding period (simplified - would need block height tracking)
	if validator.JailTime != nil {
		unbondingDuration := time.Duration(gs.params.UnbondingPeriod) * 5 * time.Second
		if time.Since(*validator.JailTime) < unbondingDuration {
			return fmt.Errorf("unbonding period not yet elapsed")
		}
	}

	validator.Jailed = false
	validator.JailTime = nil
	// Note: validator remains inactive until explicitly activated

	gs.logger.Info("Validator unjailed",
		zap.String("address", address.String()),
	)

	return nil
}

// SlashValidator slashes a validator's stake (punishment for misbehavior)
func (gs *GovernanceSystem) SlashValidator(address types.Address, reason string) (*big.Int, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	validator, exists := gs.validators[address]
	if !exists {
		return nil, ErrValidatorNotFound
	}

	// Calculate slash amount
	slashPercentageFloat := gs.params.SlashingPercentage
	slashAmount := new(big.Int).Mul(validator.StakedAmount, big.NewInt(int64(slashPercentageFloat*10000)))
	slashAmount.Div(slashAmount, big.NewInt(10000))

	// Apply slash
	validator.StakedAmount.Sub(validator.StakedAmount, slashAmount)
	validator.VotingPower = gs.calculateVotingPower(validator.StakedAmount)

	gs.logger.Warn("Validator slashed",
		zap.String("address", address.String()),
		zap.String("slash_amount", slashAmount.String()),
		zap.String("remaining_stake", validator.StakedAmount.String()),
		zap.String("reason", reason),
	)

	return slashAmount, nil
}

// RemoveValidator removes a validator from the set
func (gs *GovernanceSystem) RemoveValidator(address types.Address) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if _, exists := gs.validators[address]; !exists {
		return ErrValidatorNotFound
	}

	delete(gs.validators, address)

	gs.logger.Info("Validator removed",
		zap.String("address", address.String()),
	)

	return nil
}

// GetValidatorSet returns the current active validator set sorted by voting power
func (gs *GovernanceSystem) GetValidatorSet() []*Validator {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	validators := make([]*Validator, 0)
	for _, validator := range gs.validators {
		if validator.Active && !validator.Jailed {
			validators = append(validators, validator)
		}
	}

	// Sort by voting power (descending)
	// Simple bubble sort for now - could optimize with sort.Slice
	for i := 0; i < len(validators)-1; i++ {
		for j := i + 1; j < len(validators); j++ {
			if validators[i].VotingPower.Cmp(validators[j].VotingPower) < 0 {
				validators[i], validators[j] = validators[j], validators[i]
			}
		}
	}

	return validators
}

// GetValidatorStats returns statistics about the validator set
func (gs *GovernanceSystem) GetValidatorStats() map[string]interface{} {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	totalValidators := len(gs.validators)
	activeCount := 0
	jailedCount := 0
	totalStake := big.NewInt(0)
	totalVotingPower := big.NewInt(0)

	for _, validator := range gs.validators {
		if validator.Active {
			activeCount++
		}
		if validator.Jailed {
			jailedCount++
		}
		totalStake.Add(totalStake, validator.StakedAmount)
		if validator.Active && !validator.Jailed {
			totalVotingPower.Add(totalVotingPower, validator.VotingPower)
		}
	}

	return map[string]interface{}{
		"total_validators":   totalValidators,
		"active_validators":  activeCount,
		"jailed_validators":  jailedCount,
		"max_validators":     gs.params.MaxValidators,
		"total_stake":        totalStake.String(),
		"total_voting_power": totalVotingPower.String(),
		"min_stake":          gs.params.MinValidatorStake.String(),
	}
}

// Helper: calculate voting power from stake (1:1 for now)
func (gs *GovernanceSystem) calculateVotingPower(stake *big.Int) *big.Int {
	return new(big.Int).Set(stake)
}

// Helper: get active validator count
func (gs *GovernanceSystem) getActiveValidatorCount() int {
	count := 0
	for _, validator := range gs.validators {
		if validator.Active && !validator.Jailed {
			count++
		}
	}
	return count
}
