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
	chainBonds      map[string]*ChainBond
	bondedByStaker  map[types.Address]*big.Int
	totalStaked     *big.Int
	minStake        *big.Int
	unbondingPeriod uint64 // in blocks
	logger          *zap.Logger
	mu              sync.RWMutex
}

// ChainBond reserves part of an active staking position for checkpoint
// obligations. Remaining can only decrease through slashing or explicit
// release; the original amount is retained for auditability.
type ChainBond struct {
	ChainID   string        `json:"chain_id"`
	Owner     types.Address `json:"owner"`
	Original  *big.Int      `json:"original"`
	Remaining *big.Int      `json:"remaining"`
	Slashed   *big.Int      `json:"slashed"`
	LockedAt  time.Time     `json:"locked_at"`
}

// NewStakingManager creates a new staking manager
func NewStakingManager(nrnToken *token.NRN, logger *zap.Logger) *StakingManager {
	return &StakingManager{
		nrnToken:        nrnToken,
		stakes:          make(map[types.Address]*Stake),
		chainBonds:      make(map[string]*ChainBond),
		bondedByStaker:  make(map[types.Address]*big.Int),
		totalStaked:     big.NewInt(0),
		minStake:        big.NewInt(1000000000), // 1000 NRN minimum
		unbondingPeriod: 201600,                 // ~28 days at 5s/block
		logger:          logger,
	}
}

// LockChainBond reserves amount from an active stake. Tokens remain part of
// totalStaked until slashed, but cannot be reused for another chain bond.
func (sm *StakingManager) LockChainBond(chainID string, owner types.Address, amount *big.Int) (*ChainBond, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if chainID == "" || amount == nil || amount.Sign() <= 0 {
		return nil, fmt.Errorf("positive chain bond and chain id required")
	}
	if _, exists := sm.chainBonds[chainID]; exists {
		return nil, fmt.Errorf("chain %s already has a bond", chainID)
	}
	stake, ok := sm.stakes[owner]
	if !ok || stake.Status != StakeStatusActive {
		return nil, fmt.Errorf("bond owner %s has no active stake", owner.String())
	}
	reserved := sm.bondedByStaker[owner]
	if reserved == nil {
		reserved = big.NewInt(0)
	}
	available := new(big.Int).Sub(stake.Amount, reserved)
	if available.Cmp(amount) < 0 {
		return nil, fmt.Errorf("insufficient unbonded stake: have %s need %s", available, amount)
	}
	bond := &ChainBond{
		ChainID: chainID, Owner: owner,
		Original: new(big.Int).Set(amount), Remaining: new(big.Int).Set(amount),
		Slashed: big.NewInt(0), LockedAt: time.Now().UTC(),
	}
	sm.chainBonds[chainID] = bond
	sm.bondedByStaker[owner] = new(big.Int).Add(reserved, amount)
	return cloneChainBond(bond), nil
}

// SlashChainBond burns up to amount from a locked bond and its underlying
// stake, returning the amount actually slashed.
func (sm *StakingManager) SlashChainBond(chainID string, amount *big.Int) (*ChainBond, *big.Int, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	bond, ok := sm.chainBonds[chainID]
	if !ok {
		return nil, nil, fmt.Errorf("no chain bond for %s", chainID)
	}
	if amount == nil || amount.Sign() <= 0 {
		return nil, nil, fmt.Errorf("slash amount must be positive")
	}
	actual := new(big.Int).Set(amount)
	if actual.Cmp(bond.Remaining) > 0 {
		actual.Set(bond.Remaining)
	}
	if actual.Sign() == 0 {
		return cloneChainBond(bond), actual, nil
	}
	bond.Remaining.Sub(bond.Remaining, actual)
	bond.Slashed.Add(bond.Slashed, actual)
	sm.bondedByStaker[bond.Owner].Sub(sm.bondedByStaker[bond.Owner], actual)
	if stake := sm.stakes[bond.Owner]; stake != nil {
		stake.Amount.Sub(stake.Amount, actual)
	}
	sm.totalStaked.Sub(sm.totalStaked, actual)
	return cloneChainBond(bond), new(big.Int).Set(actual), nil
}

func (sm *StakingManager) GetChainBond(chainID string) (*ChainBond, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	bond, ok := sm.chainBonds[chainID]
	if !ok {
		return nil, fmt.Errorf("no chain bond for %s", chainID)
	}
	return cloneChainBond(bond), nil
}

// ReleaseChainBond unlocks an unslashed bond. It is used to roll back a failed
// registration and for future governance-authorized chain retirement.
func (sm *StakingManager) ReleaseChainBond(chainID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	bond, ok := sm.chainBonds[chainID]
	if !ok {
		return fmt.Errorf("no chain bond for %s", chainID)
	}
	if reserved := sm.bondedByStaker[bond.Owner]; reserved != nil {
		reserved.Sub(reserved, bond.Remaining)
	}
	delete(sm.chainBonds, chainID)
	return nil
}

func cloneChainBond(bond *ChainBond) *ChainBond {
	if bond == nil {
		return nil
	}
	out := *bond
	out.Original = new(big.Int).Set(bond.Original)
	out.Remaining = new(big.Int).Set(bond.Remaining)
	out.Slashed = new(big.Int).Set(bond.Slashed)
	return &out
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
	if amount == nil || amount.Sign() <= 0 {
		return fmt.Errorf("unstake amount must be positive")
	}

	stake, exists := sm.stakes[staker]
	if !exists {
		return fmt.Errorf("no stake found")
	}

	if stake.Status != StakeStatusActive {
		return fmt.Errorf("stake not active")
	}

	reserved := sm.bondedByStaker[staker]
	if reserved == nil {
		reserved = big.NewInt(0)
	}
	available := new(big.Int).Sub(stake.Amount, reserved)
	if amount.Cmp(available) > 0 {
		return fmt.Errorf("unstake amount exceeds unbonded stake: available %s, bonded %s", available, reserved)
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
	if reserved := sm.bondedByStaker[staker]; reserved != nil && reserved.Sign() > 0 {
		return fmt.Errorf("stake has %s locked in chain bonds", reserved)
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
