package economics

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// StakeStatus represents the status of a stake
type StakeStatus int

const (
	StakeStatusActive StakeStatus = iota
	StakeStatusUnbonding
	StakeStatusUnbonded
)

func (ss StakeStatus) String() string {
	switch ss {
	case StakeStatusActive:
		return "Active"
	case StakeStatusUnbonding:
		return "Unbonding"
	case StakeStatusUnbonded:
		return "Unbonded"
	default:
		return "Unknown"
	}
}

// Stake represents a staking position
type Stake struct {
	Staker          types.Address `json:"staker"`
	Amount          *big.Int      `json:"amount"`
	Status          StakeStatus   `json:"status"`
	StakedAt        time.Time     `json:"staked_at"`
	UnbondingAt     *time.Time    `json:"unbonding_at,omitempty"`
	UnbondingHeight uint64        `json:"unbonding_height,omitempty"`
	RewardsEarned   *big.Int      `json:"rewards_earned"`
}

// StakingManager manages staking operations
type StakingManager struct {
	nrnToken        *token.NRN
	stakes          map[types.Address]*Stake
	totalStaked     *big.Int
	minStake        *big.Int
	unbondingPeriod uint64 // in blocks
	logger          *zap.Logger
	mu              sync.RWMutex
}

// NewStakingManager creates a new staking manager
func NewStakingManager(nrnToken *token.NRN, logger *zap.Logger) *StakingManager {
	return &StakingManager{
		nrnToken:        nrnToken,
		stakes:          make(map[types.Address]*Stake),
		totalStaked:     big.NewInt(0),
		minStake:        big.NewInt(1000000000), // 1000 NRN minimum
		unbondingPeriod: 201600,                 // ~28 days at 5s/block
		logger:          logger,
	}
}

// Stake stakes tokens
func (sm *StakingManager) Stake(staker types.Address, amount *big.Int) (*Stake, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate amount
	if amount.Cmp(sm.minStake) < 0 {
		return nil, fmt.Errorf("stake amount below minimum: %s", sm.minStake.String())
	}

	// Check if already has stake
	existingStake, exists := sm.stakes[staker]
	if exists && existingStake.Status == StakeStatusActive {
		// Add to existing stake
		existingStake.Amount.Add(existingStake.Amount, amount)
		sm.totalStaked.Add(sm.totalStaked, amount)

		sm.logger.Info("Stake increased",
			zap.String("staker", staker.String()),
			zap.String("added", amount.String()),
			zap.String("total", existingStake.Amount.String()),
		)

		return existingStake, nil
	}

	// Transfer tokens (lock them)
	// In production, would transfer to staking contract/address
	// For now, just verify balance
	balance := sm.nrnToken.GetBalance(staker)
	if balance.Cmp(amount) < 0 {
		return nil, fmt.Errorf("insufficient balance")
	}

	// Create new stake
	stake := &Stake{
		Staker:        staker,
		Amount:        new(big.Int).Set(amount),
		Status:        StakeStatusActive,
		StakedAt:      time.Now(),
		RewardsEarned: big.NewInt(0),
	}

	sm.stakes[staker] = stake
	sm.totalStaked.Add(sm.totalStaked, amount)

	sm.logger.Info("Tokens staked",
		zap.String("staker", staker.String()),
		zap.String("amount", amount.String()),
	)

	return stake, nil
}

// Unstake initiates unstaking (starts unbonding period)
func (sm *StakingManager) Unstake(staker types.Address, amount *big.Int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stake, exists := sm.stakes[staker]
	if !exists {
		return fmt.Errorf("no stake found")
	}

	if stake.Status != StakeStatusActive {
		return fmt.Errorf("stake not active")
	}

	if amount.Cmp(stake.Amount) > 0 {
		return fmt.Errorf("unstake amount exceeds staked amount")
	}

	// If unstaking full amount, start unbonding
	if amount.Cmp(stake.Amount) == 0 {
		now := time.Now()
		stake.Status = StakeStatusUnbonding
		stake.UnbondingAt = &now
		stake.UnbondingHeight = 0 // Would be current block height in production

		sm.logger.Info("Unbonding started",
			zap.String("staker", staker.String()),
			zap.String("amount", amount.String()),
		)
	} else {
		// Partial unstake - just reduce amount
		stake.Amount.Sub(stake.Amount, amount)
		sm.totalStaked.Sub(sm.totalStaked, amount)

		sm.logger.Info("Stake reduced",
			zap.String("staker", staker.String()),
			zap.String("reduced", amount.String()),
			zap.String("remaining", stake.Amount.String()),
		)
	}

	return nil
}

// CompleteUnbonding completes the unbonding process
func (sm *StakingManager) CompleteUnbonding(staker types.Address, currentHeight uint64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stake, exists := sm.stakes[staker]
	if !exists {
		return fmt.Errorf("no stake found")
	}

	if stake.Status != StakeStatusUnbonding {
		return fmt.Errorf("stake not unbonding")
	}

	// Check if unbonding period elapsed (simplified - uses time instead of blocks)
	if stake.UnbondingAt != nil {
		unbondingDuration := time.Duration(sm.unbondingPeriod) * 5 * time.Second
		if time.Since(*stake.UnbondingAt) < unbondingDuration {
			return fmt.Errorf("unbonding period not elapsed")
		}
	}

	// Complete unbonding
	stake.Status = StakeStatusUnbonded
	sm.totalStaked.Sub(sm.totalStaked, stake.Amount)

	// In production, would transfer tokens back to user
	// For now, just mark as unbonded

	sm.logger.Info("Unbonding completed",
		zap.String("staker", staker.String()),
		zap.String("amount", stake.Amount.String()),
	)

	return nil
}

// GetStake retrieves a stake
func (sm *StakingManager) GetStake(staker types.Address) (*Stake, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stake, exists := sm.stakes[staker]
	if !exists {
		return nil, fmt.Errorf("no stake found")
	}

	return stake, nil
}

// GetAllStakers returns all active stakers
func (sm *StakingManager) GetAllStakers() []Staker {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stakers := make([]Staker, 0)
	for _, stake := range sm.stakes {
		if stake.Status == StakeStatusActive {
			stakers = append(stakers, Staker{
				Address:      stake.Staker,
				StakedAmount: new(big.Int).Set(stake.Amount),
			})
		}
	}

	return stakers
}

// GetTotalStaked returns total staked amount
func (sm *StakingManager) GetTotalStaked() *big.Int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return new(big.Int).Set(sm.totalStaked)
}

// UpdateRewards updates rewards earned by a staker
func (sm *StakingManager) UpdateRewards(staker types.Address, rewardAmount *big.Int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stake, exists := sm.stakes[staker]
	if !exists {
		return fmt.Errorf("no stake found")
	}

	stake.RewardsEarned.Add(stake.RewardsEarned, rewardAmount)

	sm.logger.Debug("Rewards updated",
		zap.String("staker", staker.String()),
		zap.String("reward", rewardAmount.String()),
		zap.String("total_earned", stake.RewardsEarned.String()),
	)

	return nil
}

// SetMinStake sets the minimum stake amount
func (sm *StakingManager) SetMinStake(amount *big.Int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if amount.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("minimum stake must be positive")
	}

	sm.minStake = new(big.Int).Set(amount)

	sm.logger.Info("Minimum stake updated",
		zap.String("min_stake", amount.String()),
	)

	return nil
}

// SetUnbondingPeriod sets the unbonding period
func (sm *StakingManager) SetUnbondingPeriod(blocks uint64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if blocks == 0 {
		return fmt.Errorf("unbonding period must be positive")
	}

	sm.unbondingPeriod = blocks

	sm.logger.Info("Unbonding period updated",
		zap.Uint64("blocks", blocks),
	)

	return nil
}

// GetStakingStats returns staking statistics
func (sm *StakingManager) GetStakingStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	activeCount := 0
	unbondingCount := 0
	totalRewards := big.NewInt(0)

	for _, stake := range sm.stakes {
		switch stake.Status {
		case StakeStatusActive:
			activeCount++
		case StakeStatusUnbonding:
			unbondingCount++
		}
		totalRewards.Add(totalRewards, stake.RewardsEarned)
	}

	return map[string]interface{}{
		"total_staked":     sm.totalStaked.String(),
		"total_stakers":    len(sm.stakes),
		"active_stakers":   activeCount,
		"unbonding_stakes": unbondingCount,
		"total_rewards":    totalRewards.String(),
		"min_stake":        sm.minStake.String(),
		"unbonding_period": sm.unbondingPeriod,
	}
}

// ListActiveStakes returns all active stakes
func (sm *StakingManager) ListActiveStakes() []*Stake {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stakes := make([]*Stake, 0)
	for _, stake := range sm.stakes {
		if stake.Status == StakeStatusActive {
			stakes = append(stakes, stake)
		}
	}

	return stakes
}

// ListUnbondingStakes returns all unbonding stakes
func (sm *StakingManager) ListUnbondingStakes() []*Stake {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stakes := make([]*Stake, 0)
	for _, stake := range sm.stakes {
		if stake.Status == StakeStatusUnbonding {
			stakes = append(stakes, stake)
		}
	}

	return stakes
}
