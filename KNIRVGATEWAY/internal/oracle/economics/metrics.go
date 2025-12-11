package economics

import (
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EconomicMetrics represents a snapshot of economic metrics
type EconomicMetrics struct {
	Timestamp               time.Time `json:"timestamp"`
	TotalSupply             *big.Int  `json:"total_supply"`
	MaxSupply               *big.Int  `json:"max_supply"`
	CirculatingSupply       *big.Int  `json:"circulating_supply"`
	TotalStaked             *big.Int  `json:"total_staked"`
	TotalBurned             *big.Int  `json:"total_burned"`
	TotalFeesCollected      *big.Int  `json:"total_fees_collected"`
	TotalRewardsDistributed *big.Int  `json:"total_rewards_distributed"`
	RewardPoolBalance       *big.Int  `json:"reward_pool_balance"`
	StakingAPY              float64   `json:"staking_apy"`
	BurnRate                float64   `json:"burn_rate"`
}

// MetricsCollector collects and stores economic metrics
type MetricsCollector struct {
	currentMetrics  *EconomicMetrics
	metricsHistory  []EconomicMetrics
	maxHistorySize  int
	logger          *zap.Logger
	mu              sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger *zap.Logger) *MetricsCollector {
	return &MetricsCollector{
		currentMetrics: &EconomicMetrics{
			Timestamp:               time.Now(),
			TotalSupply:             big.NewInt(0),
			MaxSupply:               big.NewInt(0),
			CirculatingSupply:       big.NewInt(0),
			TotalStaked:             big.NewInt(0),
			TotalBurned:             big.NewInt(0),
			TotalFeesCollected:      big.NewInt(0),
			TotalRewardsDistributed: big.NewInt(0),
			RewardPoolBalance:       big.NewInt(0),
			StakingAPY:              0.0,
			BurnRate:                0.0,
		},
		metricsHistory: make([]EconomicMetrics, 0),
		maxHistorySize: 1440, // 24 hours at 1 minute intervals
		logger:         logger,
	}
}

// RecordMetrics records a metrics snapshot
func (mc *MetricsCollector) RecordMetrics(metrics *EconomicMetrics) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Update current metrics
	mc.currentMetrics = metrics

	// Add to history
	mc.metricsHistory = append(mc.metricsHistory, *metrics)

	// Trim history if exceeds max size
	if len(mc.metricsHistory) > mc.maxHistorySize {
		// Remove oldest entries
		excess := len(mc.metricsHistory) - mc.maxHistorySize
		mc.metricsHistory = mc.metricsHistory[excess:]
	}

	mc.logger.Debug("Metrics recorded",
		zap.String("total_supply", metrics.TotalSupply.String()),
		zap.Float64("staking_apy", metrics.StakingAPY),
	)
}

// GetCurrentMetrics returns the current metrics snapshot
func (mc *MetricsCollector) GetCurrentMetrics() *EconomicMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return mc.currentMetrics
}

// GetMetricsHistory returns historical metrics
func (mc *MetricsCollector) GetMetricsHistory(limit int) []EconomicMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if limit <= 0 || limit > len(mc.metricsHistory) {
		limit = len(mc.metricsHistory)
	}

	start := len(mc.metricsHistory) - limit
	history := make([]EconomicMetrics, limit)
	copy(history, mc.metricsHistory[start:])

	return history
}

// GetMetricsAt returns metrics at a specific time (or nearest)
func (mc *MetricsCollector) GetMetricsAt(timestamp time.Time) *EconomicMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.metricsHistory) == 0 {
		return nil
	}

	// Find nearest metrics
	var nearest *EconomicMetrics
	minDiff := time.Hour * 24 * 365 // 1 year

	for _, metrics := range mc.metricsHistory {
		diff := metrics.Timestamp.Sub(timestamp)
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff {
			minDiff = diff
			m := metrics // Create copy
			nearest = &m
		}
	}

	return nearest
}

// GetSupplyMetrics returns supply-related metrics
func (mc *MetricsCollector) GetSupplyMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := mc.currentMetrics

	// Calculate supply percentages
	var circulatingPercent, stakedPercent, burnedPercent float64
	if metrics.TotalSupply.Cmp(big.NewInt(0)) > 0 {
		totalFloat := new(big.Float).SetInt(metrics.TotalSupply)

		circulatingFloat := new(big.Float).SetInt(metrics.CirculatingSupply)
		circulatingPercent, _ = new(big.Float).Quo(circulatingFloat, totalFloat).Float64()
		circulatingPercent *= 100

		stakedFloat := new(big.Float).SetInt(metrics.TotalStaked)
		stakedPercent, _ = new(big.Float).Quo(stakedFloat, totalFloat).Float64()
		stakedPercent *= 100

		burnedFloat := new(big.Float).SetInt(metrics.TotalBurned)
		burnedPercent, _ = new(big.Float).Quo(burnedFloat, totalFloat).Float64()
		burnedPercent *= 100
	}

	return map[string]interface{}{
		"total_supply":        metrics.TotalSupply.String(),
		"max_supply":          metrics.MaxSupply.String(),
		"circulating_supply":  metrics.CirculatingSupply.String(),
		"total_staked":        metrics.TotalStaked.String(),
		"total_burned":        metrics.TotalBurned.String(),
		"circulating_percent": circulatingPercent,
		"staked_percent":      stakedPercent,
		"burned_percent":      burnedPercent,
	}
}

// GetEconomicHealth returns economic health indicators
func (mc *MetricsCollector) GetEconomicHealth() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := mc.currentMetrics

	// Calculate velocity (fees / circulating supply)
	var velocity float64
	if metrics.CirculatingSupply.Cmp(big.NewInt(0)) > 0 {
		feesFloat := new(big.Float).SetInt(metrics.TotalFeesCollected)
		circulatingFloat := new(big.Float).SetInt(metrics.CirculatingSupply)
		velocity, _ = new(big.Float).Quo(feesFloat, circulatingFloat).Float64()
	}

	// Calculate reward ratio (rewards / staked)
	var rewardRatio float64
	if metrics.TotalStaked.Cmp(big.NewInt(0)) > 0 {
		rewardsFloat := new(big.Float).SetInt(metrics.TotalRewardsDistributed)
		stakedFloat := new(big.Float).SetInt(metrics.TotalStaked)
		rewardRatio, _ = new(big.Float).Quo(rewardsFloat, stakedFloat).Float64()
	}

	// Calculate supply utilization
	var supplyUtilization float64
	if metrics.MaxSupply.Cmp(big.NewInt(0)) > 0 {
		totalFloat := new(big.Float).SetInt(metrics.TotalSupply)
		maxFloat := new(big.Float).SetInt(metrics.MaxSupply)
		supplyUtilization, _ = new(big.Float).Quo(totalFloat, maxFloat).Float64()
		supplyUtilization *= 100
	}

	return map[string]interface{}{
		"staking_apy":        metrics.StakingAPY,
		"burn_rate":          metrics.BurnRate,
		"velocity":           velocity,
		"reward_ratio":       rewardRatio,
		"supply_utilization": supplyUtilization,
		"timestamp":          metrics.Timestamp.Unix(),
	}
}

// GetHistoricalTrend returns trend data over time
func (mc *MetricsCollector) GetHistoricalTrend(dataPoints int) map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.metricsHistory) == 0 {
		return map[string]interface{}{
			"data_points": 0,
			"timestamps":  []int64{},
			"supply":      []string{},
			"staked":      []string{},
			"burned":      []string{},
			"apy":         []float64{},
		}
	}

	// Determine step size
	historyLen := len(mc.metricsHistory)
	step := 1
	if dataPoints > 0 && historyLen > dataPoints {
		step = historyLen / dataPoints
	}

	timestamps := make([]int64, 0)
	supply := make([]string, 0)
	staked := make([]string, 0)
	burned := make([]string, 0)
	apy := make([]float64, 0)

	for i := 0; i < historyLen; i += step {
		m := mc.metricsHistory[i]
		timestamps = append(timestamps, m.Timestamp.Unix())
		supply = append(supply, m.TotalSupply.String())
		staked = append(staked, m.TotalStaked.String())
		burned = append(burned, m.TotalBurned.String())
		apy = append(apy, m.StakingAPY)
	}

	return map[string]interface{}{
		"data_points": len(timestamps),
		"timestamps":  timestamps,
		"supply":      supply,
		"staked":      staked,
		"burned":      burned,
		"apy":         apy,
	}
}

// GetComparison returns comparison between two time periods
func (mc *MetricsCollector) GetComparison(duration time.Duration) map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.metricsHistory) < 2 {
		return map[string]interface{}{
			"error": "insufficient history",
		}
	}

	current := mc.currentMetrics
	past := mc.GetMetricsAt(time.Now().Add(-duration))

	if past == nil {
		return map[string]interface{}{
			"error": "no historical data for comparison",
		}
	}

	// Calculate changes
	supplyChange := new(big.Int).Sub(current.TotalSupply, past.TotalSupply)
	stakedChange := new(big.Int).Sub(current.TotalStaked, past.TotalStaked)
	burnedChange := new(big.Int).Sub(current.TotalBurned, past.TotalBurned)
	apyChange := current.StakingAPY - past.StakingAPY

	return map[string]interface{}{
		"duration_seconds": duration.Seconds(),
		"current": map[string]interface{}{
			"supply": current.TotalSupply.String(),
			"staked": current.TotalStaked.String(),
			"burned": current.TotalBurned.String(),
			"apy":    current.StakingAPY,
		},
		"past": map[string]interface{}{
			"supply": past.TotalSupply.String(),
			"staked": past.TotalStaked.String(),
			"burned": past.TotalBurned.String(),
			"apy":    past.StakingAPY,
		},
		"change": map[string]interface{}{
			"supply": supplyChange.String(),
			"staked": stakedChange.String(),
			"burned": burnedChange.String(),
			"apy":    apyChange,
		},
	}
}

// SetMaxHistorySize sets the maximum history size
func (mc *MetricsCollector) SetMaxHistorySize(size int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if size < 1 {
		size = 1
	}

	mc.maxHistorySize = size

	// Trim if necessary
	if len(mc.metricsHistory) > size {
		excess := len(mc.metricsHistory) - size
		mc.metricsHistory = mc.metricsHistory[excess:]
	}

	mc.logger.Info("Max history size updated",
		zap.Int("size", size),
	)
}
