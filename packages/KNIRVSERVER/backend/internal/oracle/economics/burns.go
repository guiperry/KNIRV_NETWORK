package economics

import (
	"math/big"
	"sync"
	"time"

	"backend_server/internal/oracle/types"
	"go.uber.org/zap"
)

// BurnReason represents the reason for burning tokens
type BurnReason string

const (
	BurnReasonFee             BurnReason = "fee"
	BurnReasonSkillInvocation BurnReason = "skill_invocation"
	BurnReasonModelTransition BurnReason = "model_transition"
	BurnReasonGovernance      BurnReason = "governance"
	BurnReasonSlashing        BurnReason = "slashing"
)

// BurnEvent represents a token burn event
type BurnEvent struct {
	From      types.Address `json:"from"`
	Amount    *big.Int      `json:"amount"`
	Reason    BurnReason    `json:"reason"`
	Details   string        `json:"details"`
	Timestamp time.Time     `json:"timestamp"`
}

// BurnTracker tracks token burns
type BurnTracker struct {
	totalBurned   *big.Int
	burnEvents    []BurnEvent
	burnsByReason map[BurnReason]*big.Int
	logger        *zap.Logger
	mu            sync.RWMutex
}

// NewBurnTracker creates a new burn tracker
func NewBurnTracker(logger *zap.Logger) *BurnTracker {
	return &BurnTracker{
		totalBurned:   big.NewInt(0),
		burnEvents:    make([]BurnEvent, 0),
		burnsByReason: make(map[BurnReason]*big.Int),
		logger:        logger,
	}
}

// RecordBurn records a burn event
func (bt *BurnTracker) RecordBurn(from types.Address, amount *big.Int, details string) {
	bt.RecordBurnWithReason(from, amount, BurnReasonFee, details)
}

// RecordBurnWithReason records a burn event with a specific reason
func (bt *BurnTracker) RecordBurnWithReason(from types.Address, amount *big.Int, reason BurnReason, details string) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	event := BurnEvent{
		From:      from,
		Amount:    new(big.Int).Set(amount),
		Reason:    reason,
		Details:   details,
		Timestamp: time.Now(),
	}

	bt.burnEvents = append(bt.burnEvents, event)
	bt.totalBurned.Add(bt.totalBurned, amount)

	// Track by reason
	if _, exists := bt.burnsByReason[reason]; !exists {
		bt.burnsByReason[reason] = big.NewInt(0)
	}
	bt.burnsByReason[reason].Add(bt.burnsByReason[reason], amount)

	bt.logger.Debug("Burn recorded",
		zap.String("from", from.String()),
		zap.String("amount", amount.String()),
		zap.String("reason", string(reason)),
	)
}

// GetTotalBurned returns total tokens burned
func (bt *BurnTracker) GetTotalBurned() *big.Int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	return new(big.Int).Set(bt.totalBurned)
}

// GetBurnsByReason returns total burned for a specific reason
func (bt *BurnTracker) GetBurnsByReason(reason BurnReason) *big.Int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	amount, exists := bt.burnsByReason[reason]
	if !exists {
		return big.NewInt(0)
	}

	return new(big.Int).Set(amount)
}

// GetBurnHistory returns recent burn history
func (bt *BurnTracker) GetBurnHistory(limit int) []BurnEvent {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	if limit <= 0 || limit > len(bt.burnEvents) {
		limit = len(bt.burnEvents)
	}

	start := len(bt.burnEvents) - limit
	history := make([]BurnEvent, limit)
	copy(history, bt.burnEvents[start:])

	return history
}

// GetBurnsByAddress returns all burns from a specific address
func (bt *BurnTracker) GetBurnsByAddress(address types.Address) []BurnEvent {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	burns := make([]BurnEvent, 0)
	for _, event := range bt.burnEvents {
		if event.From == address {
			burns = append(burns, event)
		}
	}

	return burns
}

// GetAddressTotalBurned returns total burned by a specific address
func (bt *BurnTracker) GetAddressTotalBurned(address types.Address) *big.Int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	total := big.NewInt(0)
	for _, event := range bt.burnEvents {
		if event.From == address {
			total.Add(total, event.Amount)
		}
	}

	return total
}

// GetBurnStats returns burn statistics
func (bt *BurnTracker) GetBurnStats() map[string]interface{} {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	// Convert burns by reason to string map
	burnsByReasonStr := make(map[string]string)
	for reason, amount := range bt.burnsByReason {
		burnsByReasonStr[string(reason)] = amount.String()
	}

	// Count unique burners
	uniqueBurners := make(map[types.Address]bool)
	for _, event := range bt.burnEvents {
		uniqueBurners[event.From] = true
	}

	return map[string]interface{}{
		"total_burned":    bt.totalBurned.String(),
		"total_events":    len(bt.burnEvents),
		"unique_burners":  len(uniqueBurners),
		"burns_by_reason": burnsByReasonStr,
	}
}

// GetBurnRate calculates burn rate over time period
func (bt *BurnTracker) GetBurnRate(duration time.Duration) *big.Int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	burned := big.NewInt(0)

	for _, event := range bt.burnEvents {
		if event.Timestamp.After(cutoff) {
			burned.Add(burned, event.Amount)
		}
	}

	return burned
}

// GetDailyBurnRate returns burn rate over last 24 hours
func (bt *BurnTracker) GetDailyBurnRate() *big.Int {
	return bt.GetBurnRate(24 * time.Hour)
}

// GetWeeklyBurnRate returns burn rate over last 7 days
func (bt *BurnTracker) GetWeeklyBurnRate() *big.Int {
	return bt.GetBurnRate(7 * 24 * time.Hour)
}

// GetMonthlyBurnRate returns burn rate over last 30 days
func (bt *BurnTracker) GetMonthlyBurnRate() *big.Int {
	return bt.GetBurnRate(30 * 24 * time.Hour)
}

// GetBurnTrend returns burn trend data
func (bt *BurnTracker) GetBurnTrend(intervals int, duration time.Duration) []map[string]interface{} {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	trend := make([]map[string]interface{}, intervals)
	intervalDuration := duration / time.Duration(intervals)

	for i := 0; i < intervals; i++ {
		start := time.Now().Add(-duration + time.Duration(i)*intervalDuration)
		end := start.Add(intervalDuration)

		burned := big.NewInt(0)
		eventCount := 0

		for _, event := range bt.burnEvents {
			if event.Timestamp.After(start) && event.Timestamp.Before(end) {
				burned.Add(burned, event.Amount)
				eventCount++
			}
		}

		trend[i] = map[string]interface{}{
			"start":       start.Unix(),
			"end":         end.Unix(),
			"burned":      burned.String(),
			"event_count": eventCount,
		}
	}

	return trend
}
