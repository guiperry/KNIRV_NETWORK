package economics

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// FeeType represents different types of fees
type FeeType string

const (
	FeeTypeTransfer        FeeType = "transfer"
	FeeTypeCrossChain      FeeType = "cross_chain"
	FeeTypeProposal        FeeType = "proposal"
	FeeTypeSkillInvoke     FeeType = "skill_invoke"
	FeeTypeModelTransition FeeType = "model_transition"
	FeeTypeValidatorReward FeeType = "validator_reward"
)

// FeeConfig represents fee configuration
type FeeConfig struct {
	BaseFee       *big.Int `json:"base_fee"`
	FeePercentage float64  `json:"fee_percentage"` // 0-1
	MinFee        *big.Int `json:"min_fee"`
	MaxFee        *big.Int `json:"max_fee"`
}

// BurnConfig represents burn configuration
type BurnConfig struct {
	BurnPercentage float64 `json:"burn_percentage"` // 0-1, percentage of fees to burn
}

// FeeReceipt represents a fee collection receipt
type FeeReceipt struct {
	From      types.Address `json:"from"`
	FeeType   FeeType       `json:"fee_type"`
	Amount    *big.Int      `json:"amount"`
	Burned    *big.Int      `json:"burned"`
	Rewarded  *big.Int      `json:"rewarded"`
	TxHash    string        `json:"tx_hash"`
	Timestamp time.Time     `json:"timestamp"`
}

// FeeCollector manages fee collection
type FeeCollector struct {
	nrnToken   *token.NRN
	feeConfigs map[FeeType]*FeeConfig
	burnConfig *BurnConfig
	totalFees  *big.Int
	feeHistory []FeeReceipt
	logger     *zap.Logger
	rewardPool types.Address
	mu         sync.RWMutex
}

// NewFeeCollector creates a new fee collector
func NewFeeCollector(nrnToken *token.NRN, rewardPool types.Address, logger *zap.Logger) *FeeCollector {
	fc := &FeeCollector{
		nrnToken:   nrnToken,
		feeConfigs: make(map[FeeType]*FeeConfig),
		burnConfig: &BurnConfig{
			BurnPercentage: 0.5, // 50% of fees burned
		},
		totalFees:  big.NewInt(0),
		feeHistory: make([]FeeReceipt, 0),
		logger:     logger,
		rewardPool: rewardPool,
	}

	// Set default fee configs
	fc.setDefaultFeeConfigs()

	return fc
}

// setDefaultFeeConfigs sets default fee configurations
func (fc *FeeCollector) setDefaultFeeConfigs() {
	fc.feeConfigs[FeeTypeTransfer] = &FeeConfig{
		BaseFee:       big.NewInt(100), // 0.0001 NRN
		FeePercentage: 0.001,           // 0.1%
		MinFee:        big.NewInt(10),
		MaxFee:        big.NewInt(1000000),
	}

	fc.feeConfigs[FeeTypeCrossChain] = &FeeConfig{
		BaseFee:       big.NewInt(1000), // 0.001 NRN
		FeePercentage: 0.0025,           // 0.25%
		MinFee:        big.NewInt(100),
		MaxFee:        big.NewInt(10000000),
	}

	fc.feeConfigs[FeeTypeProposal] = &FeeConfig{
		BaseFee:       big.NewInt(100000000), // 100 NRN
		FeePercentage: 0,
		MinFee:        big.NewInt(100000000),
		MaxFee:        big.NewInt(100000000),
	}

	fc.feeConfigs[FeeTypeSkillInvoke] = &FeeConfig{
		BaseFee:       big.NewInt(1000000), // 1 NRN
		FeePercentage: 0,
		MinFee:        big.NewInt(100000),
		MaxFee:        big.NewInt(10000000),
	}

	fc.feeConfigs[FeeTypeModelTransition] = &FeeConfig{
		BaseFee:       big.NewInt(5000000), // 5 NRN
		FeePercentage: 0,
		MinFee:        big.NewInt(1000000),
		MaxFee:        big.NewInt(50000000),
	}

	fc.feeConfigs[FeeTypeValidatorReward] = &FeeConfig{
		BaseFee:       big.NewInt(0),
		FeePercentage: 0.05, // 5% commission
		MinFee:        big.NewInt(0),
		MaxFee:        big.NewInt(1000000000),
	}
}

// CalculateFee calculates the fee for a transaction
func (fc *FeeCollector) CalculateFee(feeType FeeType, amount *big.Int) *big.Int {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	config, exists := fc.feeConfigs[feeType]
	if !exists {
		// Default to transfer fee
		config = fc.feeConfigs[FeeTypeTransfer]
	}

	// Calculate: base_fee + (amount * fee_percentage)
	fee := new(big.Int).Set(config.BaseFee)

	if config.FeePercentage > 0 && amount != nil {
		percentageFee := new(big.Int).Mul(amount, big.NewInt(int64(config.FeePercentage*10000)))
		percentageFee.Div(percentageFee, big.NewInt(10000))
		fee.Add(fee, percentageFee)
	}

	// Enforce min/max
	if fee.Cmp(config.MinFee) < 0 {
		fee.Set(config.MinFee)
	}
	if fee.Cmp(config.MaxFee) > 0 {
		fee.Set(config.MaxFee)
	}

	return fee
}

// CollectFee collects a fee from an address
func (fc *FeeCollector) CollectFee(from types.Address, feeType FeeType, amount *big.Int) (*FeeReceipt, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Validate amount
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("fee amount must be positive")
	}

	burnAmount := new(big.Int).Mul(amount, big.NewInt(int64(fc.burnConfig.BurnPercentage*10000)))
	burnAmount.Div(burnAmount, big.NewInt(10000))
	rewardAmount := new(big.Int).Sub(amount, burnAmount)
	if err := fc.nrnToken.CollectFee(from, fc.rewardPool, burnAmount, rewardAmount); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	txHash := sha256.Sum256([]byte(fmt.Sprintf("fee:%s:%s:%s:%d", from, feeType, amount, now.UnixNano())))

	// Create fee receipt
	feeReceipt := FeeReceipt{
		From:      from,
		FeeType:   feeType,
		Amount:    new(big.Int).Set(amount),
		Burned:    new(big.Int).Set(burnAmount),
		Rewarded:  new(big.Int).Set(rewardAmount),
		TxHash:    fmt.Sprintf("%X", txHash),
		Timestamp: now,
	}

	// Update totals
	fc.totalFees.Add(fc.totalFees, amount)
	fc.feeHistory = append(fc.feeHistory, feeReceipt)

	fc.logger.Debug("Fee collected",
		zap.String("from", from.String()),
		zap.String("type", string(feeType)),
		zap.String("amount", amount.String()),
	)

	return &feeReceipt, nil
}

// SetFeeConfig sets fee configuration for a type
func (fc *FeeCollector) SetFeeConfig(feeType FeeType, config *FeeConfig) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Validate config
	if config.BaseFee.Cmp(big.NewInt(0)) < 0 {
		return fmt.Errorf("base fee cannot be negative")
	}
	if config.FeePercentage < 0 || config.FeePercentage > 1 {
		return fmt.Errorf("fee percentage must be between 0 and 1")
	}
	if config.MinFee.Cmp(config.MaxFee) > 0 {
		return fmt.Errorf("min fee cannot exceed max fee")
	}

	fc.feeConfigs[feeType] = config

	fc.logger.Info("Fee config updated",
		zap.String("type", string(feeType)),
		zap.String("base_fee", config.BaseFee.String()),
		zap.Float64("percentage", config.FeePercentage),
	)

	return nil
}

// GetFeeConfig gets fee configuration for a type
func (fc *FeeCollector) GetFeeConfig(feeType FeeType) *FeeConfig {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	return fc.feeConfigs[feeType]
}

// SetBurnConfig sets the burn configuration
func (fc *FeeCollector) SetBurnConfig(config *BurnConfig) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if config.BurnPercentage < 0 || config.BurnPercentage > 1 {
		return fmt.Errorf("burn percentage must be between 0 and 1")
	}

	fc.burnConfig = config

	fc.logger.Info("Burn config updated",
		zap.Float64("burn_percentage", config.BurnPercentage),
	)

	return nil
}

// GetBurnConfig gets the burn configuration
func (fc *FeeCollector) GetBurnConfig() *BurnConfig {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	return fc.burnConfig
}

// GetTotalFees returns total fees collected
func (fc *FeeCollector) GetTotalFees() *big.Int {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	return new(big.Int).Set(fc.totalFees)
}

// GetFeeHistory returns recent fee history
func (fc *FeeCollector) GetFeeHistory(limit int) []FeeReceipt {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	if limit <= 0 || limit > len(fc.feeHistory) {
		limit = len(fc.feeHistory)
	}

	start := len(fc.feeHistory) - limit
	history := make([]FeeReceipt, limit)
	copy(history, fc.feeHistory[start:])

	return history
}

// GetFeeStats returns fee statistics
func (fc *FeeCollector) GetFeeStats() map[string]interface{} {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	// Count fees by type
	feesByType := make(map[FeeType]*big.Int)
	for _, receipt := range fc.feeHistory {
		if _, exists := feesByType[receipt.FeeType]; !exists {
			feesByType[receipt.FeeType] = big.NewInt(0)
		}
		feesByType[receipt.FeeType].Add(feesByType[receipt.FeeType], receipt.Amount)
	}

	// Convert to map[string]interface{} for JSON
	feesByTypeStr := make(map[string]string)
	for feeType, amount := range feesByType {
		feesByTypeStr[string(feeType)] = amount.String()
	}

	return map[string]interface{}{
		"total_fees_collected": fc.totalFees.String(),
		"total_transactions":   len(fc.feeHistory),
		"fees_by_type":         feesByTypeStr,
		"burn_percentage":      fc.burnConfig.BurnPercentage,
	}
}
