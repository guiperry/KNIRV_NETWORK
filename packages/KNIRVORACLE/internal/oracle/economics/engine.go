package economics

import (
	"context"
	"crypto/sha256"
	"math/big"
	"sync"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// EconomicsEngine manages the token economics of the network
type EconomicsEngine struct {
	nrnToken         *token.NRN
	feeCollector     *FeeCollector
	rewardManager    *RewardManager
	stakingManager   *StakingManager
	burnTracker      *BurnTracker
	metricsCollector *MetricsCollector

	logger *zap.Logger
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// NewEconomicsEngine creates a new economics engine
func NewEconomicsEngine(nrnToken *token.NRN, logger *zap.Logger) *EconomicsEngine {
	ctx, cancel := context.WithCancel(context.Background())
	poolHash := sha256.Sum256([]byte("knirv.oracle.reward-pool.v1"))
	rewardPool, err := types.AddressFromBytes(poolHash[:20])
	if err != nil {
		panic(err)
	}

	ee := &EconomicsEngine{
		nrnToken:         nrnToken,
		feeCollector:     NewFeeCollector(nrnToken, rewardPool, logger),
		rewardManager:    NewRewardManager(nrnToken, rewardPool, logger),
		stakingManager:   NewStakingManager(nrnToken, logger),
		burnTracker:      NewBurnTracker(logger),
		metricsCollector: NewMetricsCollector(logger),
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
	}

	return ee
}

// Start starts the economics engine background processes
func (ee *EconomicsEngine) Start() error {
	ee.logger.Info("Starting economics engine")

	// Start metrics collection goroutine
	go ee.collectMetricsPeriodically()

	// Start reward distribution goroutine
	go ee.distributeRewardsPeriodically()

	return nil
}

// Stop stops the economics engine
func (ee *EconomicsEngine) Stop() error {
	ee.logger.Info("Stopping economics engine")
	ee.cancel()
	return nil
}

// GetFeeCollector returns the fee collector
func (ee *EconomicsEngine) GetFeeCollector() *FeeCollector {
	return ee.feeCollector
}

// GetRewardManager returns the reward manager
func (ee *EconomicsEngine) GetRewardManager() *RewardManager {
	return ee.rewardManager
}

// GetStakingManager returns the staking manager
func (ee *EconomicsEngine) GetStakingManager() *StakingManager {
	return ee.stakingManager
}

func (ee *EconomicsEngine) LockChainBond(chainID string, owner types.Address, amount *big.Int) (*ChainBond, error) {
	return ee.stakingManager.LockChainBond(chainID, owner, amount)
}

func (ee *EconomicsEngine) SlashChainBond(chainID, details string, amount *big.Int) (*ChainBond, *big.Int, error) {
	bond, actual, err := ee.stakingManager.SlashChainBond(chainID, amount)
	if err != nil {
		return nil, nil, err
	}
	if actual.Sign() > 0 {
		ee.burnTracker.RecordBurnWithReason(bond.Owner, actual, BurnReasonSlashing, details)
	}
	return bond, actual, nil
}

func (ee *EconomicsEngine) ReleaseChainBond(chainID string) error {
	return ee.stakingManager.ReleaseChainBond(chainID)
}

// GetBurnTracker returns the burn tracker
func (ee *EconomicsEngine) GetBurnTracker() *BurnTracker {
	return ee.burnTracker
}

// GetMetricsCollector returns the metrics collector
func (ee *EconomicsEngine) GetMetricsCollector() *MetricsCollector {
	return ee.metricsCollector
}

// ProcessFee processes a fee payment
func (ee *EconomicsEngine) ProcessFee(from types.Address, feeType FeeType, amount *big.Int) error {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	// Collect fee
	receipt, err := ee.feeCollector.CollectFee(from, feeType, amount)
	if err != nil {
		return err
	}

	// Track burn
	if receipt.Burned.Sign() > 0 {
		ee.burnTracker.RecordBurn(from, receipt.Burned, string(feeType))
	}

	// Add to reward pool
	if receipt.Rewarded.Sign() > 0 {
		ee.rewardManager.AddToRewardPool(receipt.Rewarded)
	}

	ee.logger.Debug("Fee processed",
		zap.String("from", from.String()),
		zap.String("type", string(feeType)),
		zap.String("amount", amount.String()),
		zap.String("burn", receipt.Burned.String()),
		zap.String("reward", receipt.Rewarded.String()),
		zap.String("tx_hash", receipt.TxHash),
	)

	return nil
}

// GetEconomicMetrics returns current economic metrics
func (ee *EconomicsEngine) GetEconomicMetrics() *EconomicMetrics {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	return ee.metricsCollector.GetCurrentMetrics()
}

// GetEconomicSnapshot returns a snapshot of the economic state
func (ee *EconomicsEngine) GetEconomicSnapshot() map[string]interface{} {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	metrics := ee.metricsCollector.GetCurrentMetrics()
	feeStats := ee.feeCollector.GetFeeStats()
	rewardStats := ee.rewardManager.GetRewardStats()
	stakingStats := ee.stakingManager.GetStakingStats()
	burnStats := ee.burnTracker.GetBurnStats()

	return map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"metrics":   metrics,
		"fees":      feeStats,
		"rewards":   rewardStats,
		"staking":   stakingStats,
		"burns":     burnStats,
	}
}

// collectMetricsPeriodically collects metrics every minute
func (ee *EconomicsEngine) collectMetricsPeriodically() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ee.ctx.Done():
			return
		case <-ticker.C:
			ee.collectMetrics()
		}
	}
}

// collectMetrics collects current economic metrics
func (ee *EconomicsEngine) collectMetrics() {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	// Get NRN token info
	tokenInfo := ee.nrnToken.Info()

	// Get fee stats
	feeStats := ee.feeCollector.GetFeeStats()
	totalFeesCollected, _ := new(big.Int).SetString(feeStats["total_fees_collected"].(string), 10)

	// Get reward stats
	rewardStats := ee.rewardManager.GetRewardStats()
	totalRewardsDistributed, _ := new(big.Int).SetString(rewardStats["total_distributed"].(string), 10)
	rewardPool, _ := new(big.Int).SetString(rewardStats["pool_balance"].(string), 10)

	// Get staking stats
	stakingStats := ee.stakingManager.GetStakingStats()
	totalStaked, _ := new(big.Int).SetString(stakingStats["total_staked"].(string), 10)

	// Get burn stats
	burnStats := ee.burnTracker.GetBurnStats()
	totalBurned, _ := new(big.Int).SetString(burnStats["total_burned"].(string), 10)

	// Parse token supply strings
	totalSupply, _ := new(big.Int).SetString(tokenInfo.TotalSupply, 10)
	maxSupply, _ := new(big.Int).SetString(tokenInfo.MaxSupply, 10)

	// Create metrics snapshot
	metrics := &EconomicMetrics{
		Timestamp:               time.Now(),
		TotalSupply:             totalSupply,
		MaxSupply:               maxSupply,
		CirculatingSupply:       new(big.Int).Sub(totalSupply, totalStaked),
		TotalStaked:             totalStaked,
		TotalBurned:             totalBurned,
		TotalFeesCollected:      totalFeesCollected,
		TotalRewardsDistributed: totalRewardsDistributed,
		RewardPoolBalance:       rewardPool,
		StakingAPY:              ee.calculateStakingAPY(totalStaked, totalRewardsDistributed),
		BurnRate:                ee.calculateBurnRate(totalBurned, totalSupply),
	}

	ee.metricsCollector.RecordMetrics(metrics)

	ee.logger.Debug("Metrics collected",
		zap.String("total_supply", metrics.TotalSupply.String()),
		zap.String("circulating", metrics.CirculatingSupply.String()),
		zap.Float64("staking_apy", metrics.StakingAPY),
	)
}

// distributeRewardsPeriodically distributes rewards every hour
func (ee *EconomicsEngine) distributeRewardsPeriodically() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ee.ctx.Done():
			return
		case <-ticker.C:
			ee.distributeRewards()
		}
	}
}

// distributeRewards distributes pending rewards to stakers
func (ee *EconomicsEngine) distributeRewards() {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	// Get all stakers
	stakers := ee.stakingManager.GetAllStakers()
	if len(stakers) == 0 {
		return
	}

	// Get reward pool balance
	rewardStats := ee.rewardManager.GetRewardStats()
	poolBalance, _ := new(big.Int).SetString(rewardStats["pool_balance"].(string), 10)

	if poolBalance.Cmp(big.NewInt(0)) == 0 {
		return
	}

	// Calculate hourly distribution amount (e.g., 1% of pool per hour)
	distributionAmount := new(big.Int).Div(poolBalance, big.NewInt(100))

	// Distribute proportionally to stakers
	err := ee.rewardManager.DistributeRewards(stakers, distributionAmount)
	if err != nil {
		ee.logger.Error("Failed to distribute rewards", zap.Error(err))
		return
	}

	ee.logger.Info("Rewards distributed",
		zap.String("amount", distributionAmount.String()),
		zap.Int("stakers", len(stakers)),
	)
}

// Helper: calculate staking APY
func (ee *EconomicsEngine) calculateStakingAPY(totalStaked, totalRewards *big.Int) float64 {
	if totalStaked.Cmp(big.NewInt(0)) == 0 {
		return 0.0
	}

	// Simple calculation: (total_rewards / total_staked) * 100
	rewardsFloat := new(big.Float).SetInt(totalRewards)
	stakedFloat := new(big.Float).SetInt(totalStaked)
	apy, _ := new(big.Float).Quo(rewardsFloat, stakedFloat).Float64()

	return apy * 100 // Convert to percentage
}

// Helper: calculate burn rate
func (ee *EconomicsEngine) calculateBurnRate(totalBurned, totalSupply *big.Int) float64 {
	if totalSupply.Cmp(big.NewInt(0)) == 0 {
		return 0.0
	}

	// Calculate: (total_burned / total_supply) * 100
	burnedFloat := new(big.Float).SetInt(totalBurned)
	supplyFloat := new(big.Float).SetInt(totalSupply)
	rate, _ := new(big.Float).Quo(burnedFloat, supplyFloat).Float64()

	return rate * 100 // Convert to percentage
}
