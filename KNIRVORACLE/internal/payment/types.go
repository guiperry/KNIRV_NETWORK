package payment

import (
	"math/big"
	"time"
)

// NetworkConfig represents network-specific configuration
type NetworkConfig struct {
	Name           string `json:"name"`
	Token          string `json:"token"`
	FaucetEnabled  bool   `json:"faucetEnabled"`
	PaymentEnabled bool   `json:"paymentEnabled"`
	FaucetAmount   int64  `json:"faucetAmount,omitempty"`
}

// PaymentRequest represents a payment request
type PaymentRequest struct {
	Amount  float64 `json:"amount"`
	Address string  `json:"address"`
	Network string  `json:"network"`
}

// FaucetRequest represents a faucet request
type FaucetRequest struct {
	Address string `json:"address"`
	Network string `json:"network"`
}

// FaucetResponse represents a faucet response
type FaucetResponse struct {
	Success     bool         `json:"success"`
	Transaction *Transaction `json:"transaction,omitempty"`
	Message     string       `json:"message,omitempty"`
	Error       string       `json:"error,omitempty"`
}

// Transaction represents a payment/transaction
type Transaction struct {
	ID        string    `json:"id"`
	Amount    string    `json:"amount"` // BigInt as string
	Token     string    `json:"token"`
	Recipient string    `json:"recipient"`
	Network   string    `json:"network"`
	Timestamp time.Time `json:"timestamp"`
}

// SkillInvocationRequest represents a skill invocation request
type SkillInvocationRequest struct {
	UserID  string `json:"userId"`
	SkillID string `json:"skillId"`
	Amount  string `json:"amount"` // BigInt as string
}

// LLMRegistrationRequest represents an LLM registration request
type LLMRegistrationRequest struct {
	UserID          string `json:"userId"`
	LLMID           string `json:"llmId"`
	RegistrationFee string `json:"registrationFee"` // BigInt as string
}

// ValidationRewardRequest represents a validation reward request
type ValidationRewardRequest struct {
	ValidatorID      string `json:"validatorId"`
	TargetID         string `json:"targetId"`
	ValidationResult bool   `json:"validationResult"`
}

// FeeCalculationRequest represents a fee calculation request
type FeeCalculationRequest struct {
	GasUsed  int64  `json:"gasUsed"`
	Priority string `json:"priority,omitempty"` // "low", "medium", "high"
}

// FeeCalculationResponse represents a fee calculation response
type FeeCalculationResponse struct {
	GasUsed  int64  `json:"gasUsed"`
	Priority string `json:"priority"`
	TotalFee string `json:"totalFee"` // BigInt as string
	GasPrice string `json:"gasPrice"` // BigInt as string
}

// EconomicTransaction represents an economic transaction
type EconomicTransaction struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Amount    *big.Int               `json:"-"`
	AmountStr string                 `json:"amount"`
	Purpose   string                 `json:"purpose"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
}

// BurnEvent represents a token burn event
type BurnEvent struct {
	TxID      string    `json:"txId"`
	User      string    `json:"user"`
	Amount    *big.Int  `json:"-"`
	AmountStr string    `json:"amount"`
	Purpose   string    `json:"purpose"`
	SkillID   string    `json:"skillId,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Validated bool      `json:"validated"`
}

// EconomicMetrics represents economic metrics
type EconomicMetrics struct {
	TotalBurned          *big.Int                   `json:"-"`
	TotalBurnedStr       string                     `json:"totalBurned"`
	TransactionVolume    *big.Int                   `json:"-"`
	TransactionVolumeStr string                     `json:"transactionVolume"`
	TotalSupply          *big.Int                   `json:"-"`
	TotalSupplyStr       string                     `json:"totalSupply"`
	ServiceMetrics       map[string]*ServiceMetrics `json:"serviceMetrics"`
}

// ServiceMetrics represents metrics for a specific service
type ServiceMetrics struct {
	Revenue          *big.Int  `json:"-"`
	RevenueStr       string    `json:"revenue"`
	Costs            *big.Int  `json:"-"`
	CostsStr         string    `json:"costs"`
	Profit           *big.Int  `json:"-"`
	ProfitStr        string    `json:"profit"`
	TokensEarned     *big.Int  `json:"-"`
	TokensEarnedStr  string    `json:"tokensEarned"`
	TokensSpent      *big.Int  `json:"-"`
	TokensSpentStr   string    `json:"tokensSpent"`
	UserCount        int64     `json:"userCount"`
	TransactionCount int64     `json:"transactionCount"`
	LastUpdated      time.Time `json:"lastUpdated"`
}

// EconomicRules represents the economic rules configuration
type EconomicRules struct {
	SkillInvocationCost    *big.Int `json:"-"`
	SkillInvocationCostStr string   `json:"skillInvocationCost"`
	LLMRegistrationFee     *big.Int `json:"-"`
	LLMRegistrationFeeStr  string   `json:"llmRegistrationFee"`
	ValidationReward       *big.Int `json:"-"`
	ValidationRewardStr    string   `json:"validationReward"`
	BaseGasPrice           *big.Int `json:"-"`
	BaseGasPriceStr        string   `json:"baseGasPrice"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// StripeCheckoutSession represents a Stripe checkout session
type StripeCheckoutSession struct {
	ID            string `json:"id"`
	URL           string `json:"url,omitempty"`
	PaymentStatus string `json:"payment_status,omitempty"`
}

// CoinbaseCharge represents a Coinbase Commerce charge
type CoinbaseCharge struct {
	ID     string                 `json:"id"`
	Code   string                 `json:"code"`
	Name   string                 `json:"name"`
	Status string                 `json:"status"`
	Data   map[string]interface{} `json:"data,omitempty"`
}
