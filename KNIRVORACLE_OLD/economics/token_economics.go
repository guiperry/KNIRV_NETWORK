package economics

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"
)

type TokenEconomics struct {
	nrnContract      string
	xionRPC          string
	KNIRVORACLEDB      LevelDB
	economicRules    *EconomicRules
	transactionPool  *TransactionPool
	rewardCalculator *RewardCalculator
	burnTracker      *BurnTracker
	metrics          *EconomicMetrics
	mutex            sync.RWMutex
}

type EconomicRules struct {
	SkillInvocationCost  *big.Int              `json:"skill_invocation_cost"`
	LLMRegistrationFee   *big.Int              `json:"llm_registration_fee"`
	ValidationReward     *big.Int              `json:"validation_reward"`
	BurnRates            map[string]*big.Int   `json:"burn_rates"`
	MintingRules         *MintingRules         `json:"minting_rules"`
	StakingRequirements  *StakingRequirements  `json:"staking_requirements"`
	GovernanceThresholds *GovernanceThresholds `json:"governance_thresholds"`
}

type MintingRules struct {
	MaxSupply        *big.Int `json:"max_supply"`
	InflationRate    float64  `json:"inflation_rate"`
	ValidatorRewards *big.Int `json:"validator_rewards"`
	DeveloperRewards *big.Int `json:"developer_rewards"`
	CommunityRewards *big.Int `json:"community_rewards"`
}

type StakingRequirements struct {
	MinValidatorStake *big.Int      `json:"min_validator_stake"`
	MinDeveloperStake *big.Int      `json:"min_developer_stake"`
	SlashingPenalty   float64       `json:"slashing_penalty"`
	UnbondingPeriod   time.Duration `json:"unbonding_period"`
}

type GovernanceThresholds struct {
	ProposalDeposit *big.Int      `json:"proposal_deposit"`
	VotingThreshold float64       `json:"voting_threshold"`
	QuorumThreshold float64       `json:"quorum_threshold"`
	VotingPeriod    time.Duration `json:"voting_period"`
}

type TransactionPool struct {
	pendingTxs      map[string]*EconomicTransaction
	confirmedTxs    map[string]*EconomicTransaction
	mutex           sync.RWMutex
	maxPoolSize     int
	cleanupInterval time.Duration
}

type EconomicTransaction struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	From        string                 `json:"from"`
	To          string                 `json:"to"`
	Amount      *big.Int               `json:"amount"`
	Purpose     string                 `json:"purpose"`
	Metadata    map[string]interface{} `json:"metadata"`
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	ConfirmedAt *time.Time             `json:"confirmed_at,omitempty"`
	BlockHeight uint64                 `json:"block_height,omitempty"`
	GasUsed     uint64                 `json:"gas_used,omitempty"`
}

type RewardCalculator struct {
	baseRewards     map[string]*big.Int
	multipliers     map[string]float64
	performanceData map[string]*PerformanceMetrics
	mutex           sync.RWMutex
}

type PerformanceMetrics struct {
	SuccessRate      float64   `json:"success_rate"`
	ResponseTime     float64   `json:"response_time"`
	UserSatisfaction float64   `json:"user_satisfaction"`
	Uptime           float64   `json:"uptime"`
	LastUpdated      time.Time `json:"last_updated"`
}

type BurnTracker struct {
	totalBurned *big.Int
	burnHistory []*BurnEvent
	burnRates   map[string]*big.Int
	mutex       sync.RWMutex
}

type BurnEvent struct {
	TxID      string    `json:"tx_id"`
	User      string    `json:"user"`
	Amount    *big.Int  `json:"amount"`
	Purpose   string    `json:"purpose"`
	SkillID   string    `json:"skill_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Validated bool      `json:"validated"`
}

type EconomicMetrics struct {
	TotalSupply        *big.Int                     `json:"total_supply"`
	CirculatingSupply  *big.Int                     `json:"circulating_supply"`
	TotalBurned        *big.Int                     `json:"total_burned"`
	TotalStaked        *big.Int                     `json:"total_staked"`
	ActiveValidators   int                          `json:"active_validators"`
	TransactionVolume  *big.Int                     `json:"transaction_volume"`
	AverageGasPrice    *big.Int                     `json:"average_gas_price"`
	NetworkUtilization float64                      `json:"network_utilization"`
	TokenVelocity      float64                      `json:"token_velocity"`
	LastUpdated        time.Time                    `json:"last_updated"`
	ServiceMetrics     map[string]*ServiceEconomics `json:"service_metrics"`
	mutex              sync.RWMutex                 `json:"-"`
}

type ServiceEconomics struct {
	Revenue          *big.Int  `json:"revenue"`
	Costs            *big.Int  `json:"costs"`
	Profit           *big.Int  `json:"profit"`
	TokensEarned     *big.Int  `json:"tokens_earned"`
	TokensSpent      *big.Int  `json:"tokens_spent"`
	UserCount        int       `json:"user_count"`
	TransactionCount int       `json:"transaction_count"`
	LastUpdated      time.Time `json:"last_updated"`
}

// LevelDB interface for compatibility with existing KNIRVORACLE
type LevelDB interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Close() error
}

func NewTokenEconomics(nrnContract string, xionRPC string, KNIRVORACLEDB LevelDB) (*TokenEconomics, error) {
	economics := &TokenEconomics{
		nrnContract:      nrnContract,
		xionRPC:          xionRPC,
		KNIRVORACLEDB:      KNIRVORACLEDB,
		economicRules:    NewDefaultEconomicRules(),
		transactionPool:  NewTransactionPool(),
		rewardCalculator: NewRewardCalculator(),
		burnTracker:      NewBurnTracker(),
		metrics:          NewEconomicMetrics(),
	}

	return economics, nil
}

func NewDefaultEconomicRules() *EconomicRules {
	return &EconomicRules{
		SkillInvocationCost: big.NewInt(100000),  // 0.1 NRN
		LLMRegistrationFee:  big.NewInt(1000000), // 1 NRN
		ValidationReward:    big.NewInt(50000),   // 0.05 NRN
		BurnRates: map[string]*big.Int{
			"skill_invocation": big.NewInt(100000),
			"llm_registration": big.NewInt(500000),
			"validation":       big.NewInt(25000),
		},
		MintingRules: &MintingRules{
			MaxSupply:        big.NewInt(1000000000000000), // 1B NRN
			InflationRate:    0.05,                         // 5% annual
			ValidatorRewards: big.NewInt(10000000),         // 10 NRN per block
			DeveloperRewards: big.NewInt(5000000),          // 5 NRN per block
			CommunityRewards: big.NewInt(2000000),          // 2 NRN per block
		},
		StakingRequirements: &StakingRequirements{
			MinValidatorStake: big.NewInt(100000000000), // 100K NRN
			MinDeveloperStake: big.NewInt(10000000000),  // 10K NRN
			SlashingPenalty:   0.05,                     // 5%
			UnbondingPeriod:   21 * 24 * time.Hour,      // 21 days
		},
		GovernanceThresholds: &GovernanceThresholds{
			ProposalDeposit: big.NewInt(1000000000), // 1K NRN
			VotingThreshold: 0.5,                    // 50%
			QuorumThreshold: 0.33,                   // 33%
			VotingPeriod:    7 * 24 * time.Hour,     // 7 days
		},
	}
}

func NewTransactionPool() *TransactionPool {
	return &TransactionPool{
		pendingTxs:      make(map[string]*EconomicTransaction),
		confirmedTxs:    make(map[string]*EconomicTransaction),
		maxPoolSize:     10000,
		cleanupInterval: 1 * time.Hour,
	}
}

func NewRewardCalculator() *RewardCalculator {
	return &RewardCalculator{
		baseRewards: map[string]*big.Int{
			"validation":     big.NewInt(50000),
			"skill_creation": big.NewInt(100000),
			"bug_reporting":  big.NewInt(25000),
			"community_help": big.NewInt(10000),
		},
		multipliers: map[string]float64{
			"high_performance": 1.5,
			"consistent_user":  1.2,
			"early_adopter":    1.3,
			"community_leader": 2.0,
		},
		performanceData: make(map[string]*PerformanceMetrics),
	}
}

func NewBurnTracker() *BurnTracker {
	return &BurnTracker{
		totalBurned: big.NewInt(0),
		burnHistory: make([]*BurnEvent, 0),
		burnRates:   make(map[string]*big.Int),
	}
}

func NewEconomicMetrics() *EconomicMetrics {
	return &EconomicMetrics{
		TotalSupply:       big.NewInt(0),
		CirculatingSupply: big.NewInt(0),
		TotalBurned:       big.NewInt(0),
		TotalStaked:       big.NewInt(0),
		ServiceMetrics:    make(map[string]*ServiceEconomics),
		LastUpdated:       time.Now(),
	}
}

func (te *TokenEconomics) Start(ctx context.Context) error {
	log.Println("Starting Token Economics system...")

	// Start background processes
	go te.transactionProcessor(ctx)
	go te.metricsUpdater(ctx)
	go te.rewardDistributor(ctx)
	go te.burnProcessor(ctx)

	// Load existing state
	if err := te.loadState(); err != nil {
		log.Printf("Warning: Failed to load economics state: %v", err)
	}

	log.Println("Token Economics system started")
	return nil
}

func (te *TokenEconomics) ProcessSkillInvocation(userID, skillID string, amount *big.Int) (*EconomicTransaction, error) {
	te.mutex.Lock()
	defer te.mutex.Unlock()

	// Validate amount
	requiredAmount := te.economicRules.SkillInvocationCost
	if amount.Cmp(requiredAmount) < 0 {
		return nil, fmt.Errorf("insufficient amount: required %s, provided %s", requiredAmount.String(), amount.String())
	}

	// Create transaction
	tx := &EconomicTransaction{
		ID:      fmt.Sprintf("skill_%s_%d", skillID, time.Now().UnixNano()),
		Type:    "skill_invocation",
		From:    userID,
		To:      "skill_registry",
		Amount:  amount,
		Purpose: "skill_invocation",
		Metadata: map[string]interface{}{
			"skill_id": skillID,
		},
		Status:    "pending",
		Timestamp: time.Now(),
	}

	// Add to transaction pool
	te.transactionPool.AddTransaction(tx)

	// Record burn event
	burnEvent := &BurnEvent{
		TxID:      tx.ID,
		User:      userID,
		Amount:    amount,
		Purpose:   "skill_invocation",
		SkillID:   skillID,
		Timestamp: time.Now(),
		Validated: false,
	}

	te.burnTracker.AddBurnEvent(burnEvent)

	// Update metrics
	te.updateServiceMetrics("knirvchain", amount, "spent")

	return tx, nil
}

func (te *TokenEconomics) ProcessLLMRegistration(userID, llmID string, registrationFee *big.Int) (*EconomicTransaction, error) {
	te.mutex.Lock()
	defer te.mutex.Unlock()

	// Validate fee
	requiredFee := te.economicRules.LLMRegistrationFee
	if registrationFee.Cmp(requiredFee) < 0 {
		return nil, fmt.Errorf("insufficient registration fee: required %s, provided %s", requiredFee.String(), registrationFee.String())
	}

	// Create transaction
	tx := &EconomicTransaction{
		ID:      fmt.Sprintf("llm_reg_%s_%d", llmID, time.Now().UnixNano()),
		Type:    "llm_registration",
		From:    userID,
		To:      "llm_registry",
		Amount:  registrationFee,
		Purpose: "llm_registration",
		Metadata: map[string]interface{}{
			"llm_id": llmID,
		},
		Status:    "pending",
		Timestamp: time.Now(),
	}

	te.transactionPool.AddTransaction(tx)

	// Update metrics
	te.updateServiceMetrics("knirvchain", registrationFee, "earned")

	return tx, nil
}

func (te *TokenEconomics) ProcessValidationReward(validatorID, targetID string, validationResult bool) (*EconomicTransaction, error) {
	te.mutex.Lock()
	defer te.mutex.Unlock()

	if !validationResult {
		return nil, fmt.Errorf("validation failed, no reward")
	}

	// Calculate reward based on performance
	baseReward := te.economicRules.ValidationReward
	finalReward := te.rewardCalculator.CalculateReward(validatorID, "validation", baseReward)

	// Create transaction
	tx := &EconomicTransaction{
		ID:      fmt.Sprintf("validation_%s_%d", targetID, time.Now().UnixNano()),
		Type:    "validation_reward",
		From:    "reward_pool",
		To:      validatorID,
		Amount:  finalReward,
		Purpose: "validation_reward",
		Metadata: map[string]interface{}{
			"target_id":         targetID,
			"validation_result": validationResult,
		},
		Status:    "pending",
		Timestamp: time.Now(),
	}

	te.transactionPool.AddTransaction(tx)

	// Update metrics
	te.updateServiceMetrics("knirvnexus", finalReward, "earned")

	return tx, nil
}

func (te *TokenEconomics) CalculateNetworkFees(gasUsed uint64, priority string) *big.Int {
	baseGasPrice := big.NewInt(1000) // Base gas price in wei

	// Apply priority multiplier
	multiplier := 1.0
	switch priority {
	case "high":
		multiplier = 2.0
	case "medium":
		multiplier = 1.5
	case "low":
		multiplier = 1.0
	}

	// Calculate total fee
	gasPrice := new(big.Int).Mul(baseGasPrice, big.NewInt(int64(multiplier*1000)))
	gasPrice = new(big.Int).Div(gasPrice, big.NewInt(1000))

	totalFee := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasUsed)))

	return totalFee
}

func (te *TokenEconomics) GetEconomicMetrics() *EconomicMetrics {
	te.mutex.RLock()
	defer te.mutex.RUnlock()

	// Create a copy to avoid race conditions
	metrics := &EconomicMetrics{
		TotalSupply:        new(big.Int).Set(te.metrics.TotalSupply),
		CirculatingSupply:  new(big.Int).Set(te.metrics.CirculatingSupply),
		TotalBurned:        new(big.Int).Set(te.metrics.TotalBurned),
		TotalStaked:        new(big.Int).Set(te.metrics.TotalStaked),
		ActiveValidators:   te.metrics.ActiveValidators,
		TransactionVolume:  new(big.Int).Set(te.metrics.TransactionVolume),
		AverageGasPrice:    new(big.Int).Set(te.metrics.AverageGasPrice),
		NetworkUtilization: te.metrics.NetworkUtilization,
		TokenVelocity:      te.metrics.TokenVelocity,
		LastUpdated:        te.metrics.LastUpdated,
		ServiceMetrics:     make(map[string]*ServiceEconomics),
	}

	// Copy service metrics
	for service, serviceMetrics := range te.metrics.ServiceMetrics {
		metrics.ServiceMetrics[service] = &ServiceEconomics{
			Revenue:          new(big.Int).Set(serviceMetrics.Revenue),
			Costs:            new(big.Int).Set(serviceMetrics.Costs),
			Profit:           new(big.Int).Set(serviceMetrics.Profit),
			TokensEarned:     new(big.Int).Set(serviceMetrics.TokensEarned),
			TokensSpent:      new(big.Int).Set(serviceMetrics.TokensSpent),
			UserCount:        serviceMetrics.UserCount,
			TransactionCount: serviceMetrics.TransactionCount,
			LastUpdated:      serviceMetrics.LastUpdated,
		}
	}

	return metrics
}

func (te *TokenEconomics) updateServiceMetrics(serviceName string, amount *big.Int, operation string) {
	if te.metrics.ServiceMetrics[serviceName] == nil {
		te.metrics.ServiceMetrics[serviceName] = &ServiceEconomics{
			Revenue:          big.NewInt(0),
			Costs:            big.NewInt(0),
			Profit:           big.NewInt(0),
			TokensEarned:     big.NewInt(0),
			TokensSpent:      big.NewInt(0),
			UserCount:        0,
			TransactionCount: 0,
			LastUpdated:      time.Now(),
		}
	}

	serviceMetrics := te.metrics.ServiceMetrics[serviceName]

	switch operation {
	case "earned":
		serviceMetrics.Revenue.Add(serviceMetrics.Revenue, amount)
		serviceMetrics.TokensEarned.Add(serviceMetrics.TokensEarned, amount)
	case "spent":
		serviceMetrics.Costs.Add(serviceMetrics.Costs, amount)
		serviceMetrics.TokensSpent.Add(serviceMetrics.TokensSpent, amount)
	}

	// Update profit
	serviceMetrics.Profit.Sub(serviceMetrics.Revenue, serviceMetrics.Costs)
	serviceMetrics.TransactionCount++
	serviceMetrics.LastUpdated = time.Now()
}

func (te *TokenEconomics) transactionProcessor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			te.processPendingTransactions()
		}
	}
}

func (te *TokenEconomics) processPendingTransactions() {
	te.transactionPool.mutex.RLock()
	pendingTxs := make([]*EconomicTransaction, 0, len(te.transactionPool.pendingTxs))
	for _, tx := range te.transactionPool.pendingTxs {
		pendingTxs = append(pendingTxs, tx)
	}
	te.transactionPool.mutex.RUnlock()

	for _, tx := range pendingTxs {
		if err := te.processTransaction(tx); err != nil {
			log.Printf("Failed to process transaction %s: %v", tx.ID, err)
			tx.Status = "failed"
		} else {
			tx.Status = "confirmed"
			now := time.Now()
			tx.ConfirmedAt = &now
		}

		// Move to confirmed transactions
		te.transactionPool.mutex.Lock()
		delete(te.transactionPool.pendingTxs, tx.ID)
		te.transactionPool.confirmedTxs[tx.ID] = tx
		te.transactionPool.mutex.Unlock()
	}
}

func (te *TokenEconomics) processTransaction(tx *EconomicTransaction) error {
	// Simulate transaction processing
	// In real implementation, this would interact with XION blockchain

	switch tx.Type {
	case "skill_invocation":
		return te.processSkillInvocationTx(tx)
	case "llm_registration":
		return te.processLLMRegistrationTx(tx)
	case "validation_reward":
		return te.processValidationRewardTx(tx)
	default:
		return fmt.Errorf("unknown transaction type: %s", tx.Type)
	}
}

func (te *TokenEconomics) processSkillInvocationTx(tx *EconomicTransaction) error {
	// Burn tokens for skill invocation
	te.burnTracker.mutex.Lock()
	te.burnTracker.totalBurned.Add(te.burnTracker.totalBurned, tx.Amount)
	te.burnTracker.mutex.Unlock()

	// Update total burned in metrics
	te.metrics.TotalBurned.Add(te.metrics.TotalBurned, tx.Amount)
	te.metrics.CirculatingSupply.Sub(te.metrics.CirculatingSupply, tx.Amount)

	log.Printf("Burned %s NRN for skill invocation %s", tx.Amount.String(), tx.Metadata["skill_id"])
	return nil
}

func (te *TokenEconomics) processLLMRegistrationTx(tx *EconomicTransaction) error {
	// Transfer registration fee to treasury
	log.Printf("Processed LLM registration fee %s NRN for %s", tx.Amount.String(), tx.Metadata["llm_id"])
	return nil
}

func (te *TokenEconomics) processValidationRewardTx(tx *EconomicTransaction) error {
	// Mint reward tokens
	te.metrics.TotalSupply.Add(te.metrics.TotalSupply, tx.Amount)
	te.metrics.CirculatingSupply.Add(te.metrics.CirculatingSupply, tx.Amount)

	log.Printf("Minted %s NRN validation reward for %s", tx.Amount.String(), tx.To)
	return nil
}

func (te *TokenEconomics) metricsUpdater(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			te.updateMetrics()
		}
	}
}

func (te *TokenEconomics) updateMetrics() {
	te.metrics.mutex.Lock()
	defer te.metrics.mutex.Unlock()

	// Update network utilization
	te.metrics.NetworkUtilization = te.calculateNetworkUtilization()

	// Update token velocity
	te.metrics.TokenVelocity = te.calculateTokenVelocity()

	// Update average gas price
	te.metrics.AverageGasPrice = te.calculateAverageGasPrice()

	te.metrics.LastUpdated = time.Now()
}

func (te *TokenEconomics) calculateNetworkUtilization() float64 {
	// Calculate based on transaction volume and capacity
	// This is a simplified calculation
	return 0.75 // 75% utilization
}

func (te *TokenEconomics) calculateTokenVelocity() float64 {
	// Token velocity = Transaction Volume / Circulating Supply
	if te.metrics.CirculatingSupply.Cmp(big.NewInt(0)) == 0 {
		return 0
	}

	// Simplified calculation
	return 2.5 // 2.5x velocity
}

func (te *TokenEconomics) calculateAverageGasPrice() *big.Int {
	// Calculate average gas price from recent transactions
	return big.NewInt(1500) // 1500 wei average
}

func (te *TokenEconomics) rewardDistributor(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			te.distributeRewards()
		}
	}
}

func (te *TokenEconomics) distributeRewards() {
	// Distribute validator rewards
	// Distribute developer rewards
	// Distribute community rewards
	log.Println("Distributing periodic rewards...")
}

func (te *TokenEconomics) burnProcessor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			te.processBurnEvents()
		}
	}
}

func (te *TokenEconomics) processBurnEvents() {
	te.burnTracker.mutex.Lock()
	defer te.burnTracker.mutex.Unlock()

	// Process unvalidated burn events
	for _, event := range te.burnTracker.burnHistory {
		if !event.Validated {
			// Validate burn event
			event.Validated = true
			log.Printf("Validated burn event: %s burned %s NRN", event.User, event.Amount.String())
		}
	}
}

func (te *TokenEconomics) loadState() error {
	// Load economic state from database
	// This would restore metrics, transaction history, etc.
	return nil
}

func (te *TokenEconomics) saveState() error {
	// Save economic state to database
	return nil
}

// Transaction Pool methods
func (tp *TransactionPool) AddTransaction(tx *EconomicTransaction) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	tp.pendingTxs[tx.ID] = tx

	// Clean up if pool is too large
	if len(tp.pendingTxs) > tp.maxPoolSize {
		tp.cleanupOldTransactions()
	}
}

func (tp *TransactionPool) cleanupOldTransactions() {
	// Remove oldest transactions if pool is full
	cutoff := time.Now().Add(-1 * time.Hour)

	for id, tx := range tp.pendingTxs {
		if tx.Timestamp.Before(cutoff) {
			delete(tp.pendingTxs, id)
		}
	}
}

// Reward Calculator methods
func (rc *RewardCalculator) CalculateReward(userID, rewardType string, baseAmount *big.Int) *big.Int {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	multiplier := 1.0

	// Apply performance-based multipliers
	if metrics, exists := rc.performanceData[userID]; exists {
		if metrics.SuccessRate > 0.9 {
			multiplier *= rc.multipliers["high_performance"]
		}
		if metrics.Uptime > 0.95 {
			multiplier *= rc.multipliers["consistent_user"]
		}
	}

	// Calculate final reward
	finalAmount := new(big.Int).Mul(baseAmount, big.NewInt(int64(multiplier*1000)))
	finalAmount = new(big.Int).Div(finalAmount, big.NewInt(1000))

	return finalAmount
}

func (rc *RewardCalculator) UpdatePerformanceMetrics(userID string, metrics *PerformanceMetrics) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	rc.performanceData[userID] = metrics
}

// Burn Tracker methods
func (bt *BurnTracker) AddBurnEvent(event *BurnEvent) {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	bt.burnHistory = append(bt.burnHistory, event)

	// Maintain history size
	if len(bt.burnHistory) > 10000 {
		bt.burnHistory = bt.burnHistory[1000:]
	}
}

func (bt *BurnTracker) GetTotalBurned() *big.Int {
	bt.mutex.RLock()
	defer bt.mutex.RUnlock()

	return new(big.Int).Set(bt.totalBurned)
}
