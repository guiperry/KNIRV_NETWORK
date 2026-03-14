package payment

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EconomicsEngine handles token economics operations
type EconomicsEngine struct {
	rules        *EconomicRules
	transactions []*EconomicTransaction
	burnEvents   []*BurnEvent
	metrics      *EconomicMetrics
	logger       *zap.Logger
	isRunning    bool
	mu           sync.RWMutex
}

// NewEconomicsEngine creates a new economics engine
func NewEconomicsEngine(logger *zap.Logger) *EconomicsEngine {
	rules := &EconomicRules{
		SkillInvocationCost: big.NewInt(1000000000000000000), // 1 NRN in wei
		LLMRegistrationFee:  big.NewInt(500000000000000000),  // 0.5 NRN in wei
		ValidationReward:    big.NewInt(200000000000000000),  // 0.2 NRN in wei
		BaseGasPrice:        big.NewInt(1000),
	}

	// Convert to strings for JSON
	rules.SkillInvocationCostStr = rules.SkillInvocationCost.String()
	rules.LLMRegistrationFeeStr = rules.LLMRegistrationFee.String()
	rules.ValidationRewardStr = rules.ValidationReward.String()
	rules.BaseGasPriceStr = rules.BaseGasPrice.String()

	totalSupply := new(big.Int)
	totalSupply.SetString("1000000000000000000000000", 10) // 1M NRN

	return &EconomicsEngine{
		rules:        rules,
		transactions: make([]*EconomicTransaction, 0),
		burnEvents:   make([]*BurnEvent, 0),
		metrics: &EconomicMetrics{
			TotalBurned:          big.NewInt(0),
			TransactionVolume:    big.NewInt(0),
			TotalSupply:          totalSupply,
			ServiceMetrics:       make(map[string]*ServiceMetrics),
			TotalBurnedStr:       "0",
			TransactionVolumeStr: "0",
			TotalSupplyStr:       "1000000000000000000000000",
		},
		logger: logger,
	}
}

// Start starts the economics engine
func (ee *EconomicsEngine) Start() error {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	if ee.isRunning {
		return fmt.Errorf("economics engine already running")
	}

	ee.isRunning = true
	ee.logger.Info("Economics engine started")

	return nil
}

// Stop stops the economics engine
func (ee *EconomicsEngine) Stop() error {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	if !ee.isRunning {
		return nil
	}

	ee.isRunning = false
	ee.logger.Info("Economics engine stopped")

	return nil
}

// ProcessSkillInvocation processes a skill invocation
func (ee *EconomicsEngine) ProcessSkillInvocation(req *SkillInvocationRequest) (*EconomicTransaction, error) {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", req.Amount)
	}

	if amount.Cmp(ee.rules.SkillInvocationCost) < 0 {
		return nil, fmt.Errorf("insufficient amount: required %s, provided %s",
			ee.rules.SkillInvocationCost.String(), amount.String())
	}

	tx := &EconomicTransaction{
		ID:        ee.generateTransactionID("skill", req.SkillID),
		Type:      "skill_invocation",
		From:      req.UserID,
		To:        "skill_registry",
		Amount:    amount,
		AmountStr: amount.String(),
		Purpose:   "skill_invocation",
		Metadata: map[string]interface{}{
			"skillId": req.SkillID,
		},
		Status:    "pending",
		Timestamp: time.Now(),
	}

	ee.transactions = append(ee.transactions, tx)

	// Record burn event
	burnEvent := &BurnEvent{
		TxID:      tx.ID,
		User:      req.UserID,
		Amount:    amount,
		AmountStr: amount.String(),
		Purpose:   "skill_invocation",
		SkillID:   req.SkillID,
		Timestamp: time.Now(),
		Validated: false,
	}

	ee.burnEvents = append(ee.burnEvents, burnEvent)

	// Update metrics
	ee.updateServiceMetrics("knirvchain", amount, "spent")
	ee.metrics.TotalBurned.Add(ee.metrics.TotalBurned, amount)
	ee.metrics.TransactionVolume.Add(ee.metrics.TransactionVolume, amount)

	return tx, nil
}

// ProcessLLMRegistration processes an LLM registration
func (ee *EconomicsEngine) ProcessLLMRegistration(req *LLMRegistrationRequest) (*EconomicTransaction, error) {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	fee, ok := new(big.Int).SetString(req.RegistrationFee, 10)
	if !ok {
		return nil, fmt.Errorf("invalid registration fee: %s", req.RegistrationFee)
	}

	if fee.Cmp(ee.rules.LLMRegistrationFee) < 0 {
		return nil, fmt.Errorf("insufficient registration fee: required %s, provided %s",
			ee.rules.LLMRegistrationFee.String(), fee.String())
	}

	tx := &EconomicTransaction{
		ID:        ee.generateTransactionID("llm_reg", req.LLMID),
		Type:      "llm_registration",
		From:      req.UserID,
		To:        "llm_registry",
		Amount:    fee,
		AmountStr: fee.String(),
		Purpose:   "llm_registration",
		Metadata: map[string]interface{}{
			"llmId": req.LLMID,
		},
		Status:    "pending",
		Timestamp: time.Now(),
	}

	ee.transactions = append(ee.transactions, tx)

	// Update metrics
	ee.updateServiceMetrics("knirvchain", fee, "earned")

	return tx, nil
}

// ProcessValidationReward processes a validation reward
func (ee *EconomicsEngine) ProcessValidationReward(req *ValidationRewardRequest) (*EconomicTransaction, error) {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	if !req.ValidationResult {
		return nil, fmt.Errorf("validation failed, no reward")
	}

	reward := new(big.Int).Set(ee.rules.ValidationReward)

	tx := &EconomicTransaction{
		ID:        ee.generateTransactionID("validation", req.TargetID),
		Type:      "validation_reward",
		From:      "reward_pool",
		To:        req.ValidatorID,
		Amount:    reward,
		AmountStr: reward.String(),
		Purpose:   "validation_reward",
		Metadata: map[string]interface{}{
			"targetId":         req.TargetID,
			"validationResult": req.ValidationResult,
		},
		Status:    "pending",
		Timestamp: time.Now(),
	}

	ee.transactions = append(ee.transactions, tx)

	// Update metrics
	ee.updateServiceMetrics("knirvserver", reward, "earned")
	ee.metrics.TotalSupply.Add(ee.metrics.TotalSupply, reward)
	ee.metrics.TransactionVolume.Add(ee.metrics.TransactionVolume, reward)

	return tx, nil
}

// CalculateNetworkFees calculates network fees
func (ee *EconomicsEngine) CalculateNetworkFees(req *FeeCalculationRequest) *FeeCalculationResponse {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	baseGasPrice := new(big.Int).Set(ee.rules.BaseGasPrice)
	multiplier := 1000 // Default medium priority

	switch req.Priority {
	case "high":
		multiplier = 2000
	case "low":
		multiplier = 500
	}

	gasPrice := new(big.Int).Mul(baseGasPrice, big.NewInt(int64(multiplier)))
	gasPrice.Div(gasPrice, big.NewInt(1000))

	totalFee := new(big.Int).Mul(gasPrice, big.NewInt(req.GasUsed))

	return &FeeCalculationResponse{
		GasUsed:  req.GasUsed,
		Priority: req.Priority,
		TotalFee: totalFee.String(),
		GasPrice: gasPrice.String(),
	}
}

// GetMetrics returns economic metrics
func (ee *EconomicsEngine) GetMetrics() *EconomicMetrics {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	// Update string representations
	ee.metrics.TotalBurnedStr = ee.metrics.TotalBurned.String()
	ee.metrics.TransactionVolumeStr = ee.metrics.TransactionVolume.String()
	ee.metrics.TotalSupplyStr = ee.metrics.TotalSupply.String()

	return ee.metrics
}

// GetRules returns economic rules
func (ee *EconomicsEngine) GetRules() *EconomicRules {
	ee.mu.RLock()
	defer ee.mu.RUnlock()
	return ee.rules
}

// GetTransactions returns transactions with optional filtering
func (ee *EconomicsEngine) GetTransactions(limit int, status string) []*EconomicTransaction {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	var filtered []*EconomicTransaction

	for _, tx := range ee.transactions {
		if status != "" && tx.Status != status {
			continue
		}
		filtered = append(filtered, tx)
	}

	// Return last 'limit' transactions
	if len(filtered) > limit && limit > 0 {
		start := len(filtered) - limit
		filtered = filtered[start:]
	}

	return filtered
}

// GetBurnHistory returns burn history
func (ee *EconomicsEngine) GetBurnHistory(limit int, user, purpose string) []*BurnEvent {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	var filtered []*BurnEvent

	for _, event := range ee.burnEvents {
		if user != "" && event.User != user {
			continue
		}
		if purpose != "" && event.Purpose != purpose {
			continue
		}
		filtered = append(filtered, event)
	}

	// Return last 'limit' events
	if len(filtered) > limit && limit > 0 {
		start := len(filtered) - limit
		filtered = filtered[start:]
	}

	return filtered
}

// GetTotalBurned returns total burned amount
func (ee *EconomicsEngine) GetTotalBurned() *big.Int {
	ee.mu.RLock()
	defer ee.mu.RUnlock()
	return new(big.Int).Set(ee.metrics.TotalBurned)
}

// GetServiceMetrics returns metrics for a specific service
func (ee *EconomicsEngine) GetServiceMetrics(serviceName string) *ServiceMetrics {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	metrics, exists := ee.metrics.ServiceMetrics[serviceName]
	if !exists {
		return &ServiceMetrics{}
	}

	// Update string representations
	metrics.RevenueStr = metrics.Revenue.String()
	metrics.CostsStr = metrics.Costs.String()
	metrics.ProfitStr = metrics.Profit.String()
	metrics.TokensEarnedStr = metrics.TokensEarned.String()
	metrics.TokensSpentStr = metrics.TokensSpent.String()

	return metrics
}

// Helper methods

func (ee *EconomicsEngine) generateTransactionID(prefix, identifier string) string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	randomStr := hex.EncodeToString(bytes)
	return fmt.Sprintf("%s_%s_%d_%s", prefix, identifier, time.Now().Unix(), randomStr)
}

func (ee *EconomicsEngine) updateServiceMetrics(serviceName string, amount *big.Int, operation string) {
	metrics, exists := ee.metrics.ServiceMetrics[serviceName]
	if !exists {
		metrics = &ServiceMetrics{
			Revenue:          big.NewInt(0),
			Costs:            big.NewInt(0),
			Profit:           big.NewInt(0),
			TokensEarned:     big.NewInt(0),
			TokensSpent:      big.NewInt(0),
			UserCount:        0,
			TransactionCount: 0,
			LastUpdated:      time.Now(),
		}
		ee.metrics.ServiceMetrics[serviceName] = metrics
	}

	switch operation {
	case "earned":
		metrics.Revenue.Add(metrics.Revenue, amount)
		metrics.TokensEarned.Add(metrics.TokensEarned, amount)
	case "spent":
		metrics.Costs.Add(metrics.Costs, amount)
		metrics.TokensSpent.Add(metrics.TokensSpent, amount)
	}

	// Recalculate profit
	metrics.Profit.Sub(metrics.Revenue, metrics.Costs)
	metrics.TransactionCount++
	metrics.LastUpdated = time.Now()
}
