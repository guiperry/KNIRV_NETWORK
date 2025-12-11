package economics

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle/token"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle/types"
	"go.uber.org/zap"
)

// RewardType represents different types of rewards
type RewardType string

const (
	RewardTypeStaking    RewardType = "staking"
	RewardTypeValidation RewardType = "validation"
	RewardTypeGovernance RewardType = "governance"
)

// Reward represents a reward distribution
type Reward struct {
	Recipient types.Address `json:"recipient"`
	Amount    *big.Int      `json:"amount"`
	Type      RewardType    `json:"type"`
	Timestamp time.Time     `json:"timestamp"`
	TxHash    string        `json:"tx_hash"`
}

// Staker represents a staker in the reward distribution
type Staker struct {
	Address      types.Address
	StakedAmount *big.Int
}

// RewardManager manages reward distribution
type RewardManager struct {
	nrnToken          *token.NRN
	rewardPool        *big.Int
	totalDistributed  *big.Int
	rewardHistory     []Reward
	logger            *zap.Logger
	mu                sync.RWMutex
}

// NewRewardManager creates a new reward manager
func NewRewardManager(nrnToken *token.NRN, logger *zap.Logger) *RewardManager {
	return &RewardManager{
		nrnToken:         nrnToken,
		rewardPool:       big.NewInt(0),
		totalDistributed: big.NewInt(0),
		rewardHistory:    make([]Reward, 0),
		logger:           logger,
	}
}

// AddToRewardPool adds tokens to the reward pool
func (rm *RewardManager) AddToRewardPool(amount *big.Int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.rewardPool.Add(rm.rewardPool, amount)

	rm.logger.Debug("Added to reward pool",
		zap.String("amount", amount.String()),
		zap.String("new_balance", rm.rewardPool.String()),
	)
}

// DistributeRewards distributes rewards to stakers proportionally
func (rm *RewardManager) DistributeRewards(stakers []Staker, totalAmount *big.Int) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if totalAmount.Cmp(rm.rewardPool) > 0 {
		return fmt.Errorf("insufficient reward pool balance")
	}

	if len(stakers) == 0 {
		return fmt.Errorf("no stakers to distribute rewards to")
	}

	// Calculate total stake
	totalStake := big.NewInt(0)
	for _, staker := range stakers {
		totalStake.Add(totalStake, staker.StakedAmount)
	}

	if totalStake.Cmp(big.NewInt(0)) == 0 {
		return fmt.Errorf("total stake is zero")
	}

	// Distribute proportionally
	for _, staker := range stakers {
		// Calculate share: (staker_stake / total_stake) * total_amount
		share := new(big.Int).Mul(staker.StakedAmount, totalAmount)
		share.Div(share, totalStake)

		if share.Cmp(big.NewInt(0)) == 0 {
			continue // Skip if share rounds to zero
		}

		// Mint rewards (simplified - in production might transfer from pool)
		receipt, err := rm.nrnToken.Mint(staker.Address, share)
		if err != nil {
			rm.logger.Error("Failed to mint reward",
				zap.String("recipient", staker.Address.String()),
				zap.Error(err),
			)
			continue
		}

		// Record reward
		reward := Reward{
			Recipient: staker.Address,
			Amount:    share,
			Type:      RewardTypeStaking,
			Timestamp: time.Now(),
			TxHash:    receipt.TransactionHash,
		}
		rm.rewardHistory = append(rm.rewardHistory, reward)
		rm.totalDistributed.Add(rm.totalDistributed, share)

		rm.logger.Debug("Reward distributed",
			zap.String("recipient", staker.Address.String()),
			zap.String("amount", share.String()),
		)
	}

	// Deduct from pool
	rm.rewardPool.Sub(rm.rewardPool, totalAmount)

	rm.logger.Info("Rewards distributed",
		zap.String("total_amount", totalAmount.String()),
		zap.Int("recipients", len(stakers)),
	)

	return nil
}

// DistributeReward distributes a reward to a single recipient
func (rm *RewardManager) DistributeReward(recipient types.Address, amount *big.Int, rewardType RewardType) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if amount.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("reward amount must be positive")
	}

	if amount.Cmp(rm.rewardPool) > 0 {
		return fmt.Errorf("insufficient reward pool balance")
	}

	// Mint reward
	receipt, err := rm.nrnToken.Mint(recipient, amount)
	if err != nil {
		return fmt.Errorf("failed to mint reward: %w", err)
	}

	// Record reward
	reward := Reward{
		Recipient: recipient,
		Amount:    new(big.Int).Set(amount),
		Type:      rewardType,
		Timestamp: time.Now(),
		TxHash:    receipt.TransactionHash,
	}
	rm.rewardHistory = append(rm.rewardHistory, reward)
	rm.totalDistributed.Add(rm.totalDistributed, amount)
	rm.rewardPool.Sub(rm.rewardPool, amount)

	rm.logger.Info("Reward distributed",
		zap.String("recipient", recipient.String()),
		zap.String("amount", amount.String()),
		zap.String("type", string(rewardType)),
	)

	return nil
}

// GetRewardPoolBalance returns the current reward pool balance
func (rm *RewardManager) GetRewardPoolBalance() *big.Int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return new(big.Int).Set(rm.rewardPool)
}

// GetTotalDistributed returns total rewards distributed
func (rm *RewardManager) GetTotalDistributed() *big.Int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return new(big.Int).Set(rm.totalDistributed)
}

// GetRewardHistory returns recent reward history
func (rm *RewardManager) GetRewardHistory(limit int) []Reward {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if limit <= 0 || limit > len(rm.rewardHistory) {
		limit = len(rm.rewardHistory)
	}

	start := len(rm.rewardHistory) - limit
	history := make([]Reward, limit)
	copy(history, rm.rewardHistory[start:])

	return history
}

// GetRecipientRewards returns all rewards for a specific recipient
func (rm *RewardManager) GetRecipientRewards(recipient types.Address) []Reward {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rewards := make([]Reward, 0)
	for _, reward := range rm.rewardHistory {
		if reward.Recipient == recipient {
			rewards = append(rewards, reward)
		}
	}

	return rewards
}

// GetRecipientTotalRewards returns total rewards for a specific recipient
func (rm *RewardManager) GetRecipientTotalRewards(recipient types.Address) *big.Int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	total := big.NewInt(0)
	for _, reward := range rm.rewardHistory {
		if reward.Recipient == recipient {
			total.Add(total, reward.Amount)
		}
	}

	return total
}

// GetRewardStats returns reward statistics
func (rm *RewardManager) GetRewardStats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Count rewards by type
	rewardsByType := make(map[RewardType]*big.Int)
	recipientCount := make(map[types.Address]bool)

	for _, reward := range rm.rewardHistory {
		if _, exists := rewardsByType[reward.Type]; !exists {
			rewardsByType[reward.Type] = big.NewInt(0)
		}
		rewardsByType[reward.Type].Add(rewardsByType[reward.Type], reward.Amount)
		recipientCount[reward.Recipient] = true
	}

	// Convert to map[string]interface{} for JSON
	rewardsByTypeStr := make(map[string]string)
	for rewardType, amount := range rewardsByType {
		rewardsByTypeStr[string(rewardType)] = amount.String()
	}

	return map[string]interface{}{
		"pool_balance":       rm.rewardPool.String(),
		"total_distributed":  rm.totalDistributed.String(),
		"total_transactions": len(rm.rewardHistory),
		"unique_recipients":  len(recipientCount),
		"rewards_by_type":    rewardsByTypeStr,
	}
}

// CalculateStakingReward calculates staking reward for a staker
func (rm *RewardManager) CalculateStakingReward(stakedAmount *big.Int, totalStaked *big.Int, rewardRate float64) *big.Int {
	if totalStaked.Cmp(big.NewInt(0)) == 0 {
		return big.NewInt(0)
	}

	// Calculate: (staked_amount / total_staked) * reward_rate * total_pool
	share := new(big.Int).Mul(stakedAmount, rm.rewardPool)
	share.Div(share, totalStaked)

	// Apply reward rate
	reward := new(big.Int).Mul(share, big.NewInt(int64(rewardRate*10000)))
	reward.Div(reward, big.NewInt(10000))

	return reward
}
