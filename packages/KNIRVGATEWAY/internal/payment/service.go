package payment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Service represents the payment gateway service
type Service struct {
	config         *Config
	logger         *zap.Logger
	economics      *EconomicsEngine
	stripeClient   *StripeClient
	coinbaseClient *CoinbaseClient
	faucetLimiter  *FaucetLimiter
	networks       map[string]*NetworkConfig
	mu             sync.RWMutex
}

// Config represents the payment service configuration
type Config struct {
	StripeSecretKey     string
	CoinbaseAPIKey      string
	FaucetCooldownHours int
	DefaultNetwork      string
	EconomicsEnabled    bool
}

// NewService creates a new payment service
func NewService(config *Config, logger *zap.Logger) *Service {
	networks := map[string]*NetworkConfig{
		"mainnet": {
			Name:           "Mainnet",
			Token:          "NRN",
			FaucetEnabled:  false,
			PaymentEnabled: true,
		},
		"public-testnet": {
			Name:           "Public Testnet",
			Token:          "NRV",
			FaucetEnabled:  true,
			PaymentEnabled: false,
			FaucetAmount:   1000,
		},
		"private-testnet": {
			Name:           "Private Testnet",
			Token:          "pNRV",
			FaucetEnabled:  true,
			PaymentEnabled: false,
			FaucetAmount:   5000,
		},
	}

	service := &Service{
		config:        config,
		logger:        logger,
		networks:      networks,
		faucetLimiter: NewFaucetLimiter(time.Duration(config.FaucetCooldownHours) * time.Hour),
	}

	if config.EconomicsEnabled {
		service.economics = NewEconomicsEngine(logger)
	}

	if config.StripeSecretKey != "" {
		service.stripeClient = NewStripeClient(config.StripeSecretKey)
	}

	if config.CoinbaseAPIKey != "" {
		service.coinbaseClient = NewCoinbaseClient(config.CoinbaseAPIKey)
	}

	return service
}

// Start starts the payment service
func (s *Service) Start(ctx context.Context) error {
	s.logger.Info("Starting payment gateway service")

	if s.economics != nil {
		if err := s.economics.Start(); err != nil {
			return fmt.Errorf("failed to start economics engine: %w", err)
		}
	}

	return nil
}

// Stop stops the payment service
func (s *Service) Stop(ctx context.Context) error {
	s.logger.Info("Stopping payment gateway service")

	if s.economics != nil {
		if err := s.economics.Stop(); err != nil {
			s.logger.Error("Failed to stop economics engine", zap.Error(err))
		}
	}

	return nil
}

// GetNetworkConfig returns the configuration for a network
func (s *Service) GetNetworkConfig(network string) (*NetworkConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, exists := s.networks[network]
	return config, exists
}

// GetNetworkStatus returns network status information
func (s *Service) GetNetworkStatus(network string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, exists := s.networks[network]
	if !exists {
		return map[string]interface{}{
			"error": "Network not found",
		}
	}

	return map[string]interface{}{
		"network":   network,
		"config":    config,
		"timestamp": time.Now().Unix(),
	}
}

// RequestFaucet handles a faucet request
func (s *Service) RequestFaucet(req *FaucetRequest) *FaucetResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	networkConfig, exists := s.networks[req.Network]
	if !exists || !networkConfig.FaucetEnabled {
		return &FaucetResponse{
			Success: false,
			Error:   "Faucet not available for this network",
		}
	}

	// Validate address format
	if !s.isValidAddress(req.Address) {
		return &FaucetResponse{
			Success: false,
			Error:   "Invalid wallet address format",
		}
	}

	// Check rate limiting
	if !s.faucetLimiter.Allow(req.Network, req.Address) {
		timeLeft := s.faucetLimiter.TimeLeft(req.Network, req.Address)
		hoursLeft := int(timeLeft.Hours())
		return &FaucetResponse{
			Success: false,
			Error:   fmt.Sprintf("Rate limit exceeded. Please wait %d hours before requesting again", hoursLeft),
		}
	}

	// Generate transaction ID
	txID := s.generateTransactionID("faucet", req.Network)

	// Create transaction
	transaction := &Transaction{
		ID:        txID,
		Amount:    fmt.Sprintf("%d", networkConfig.FaucetAmount),
		Token:     networkConfig.Token,
		Recipient: req.Address,
		Network:   networkConfig.Name,
		Timestamp: time.Now(),
	}

	// Update rate limiter
	s.faucetLimiter.RecordRequest(req.Network, req.Address)

	s.logger.Info("Faucet request processed",
		zap.String("network", req.Network),
		zap.String("address", req.Address),
		zap.String("amount", transaction.Amount),
		zap.String("token", transaction.Token),
	)

	return &FaucetResponse{
		Success:     true,
		Transaction: transaction,
		Message:     fmt.Sprintf("Successfully sent %s %s to your wallet", transaction.Amount, transaction.Token),
	}
}

// CheckFaucetStatus checks if an address can request from the faucet
func (s *Service) CheckFaucetStatus(address, network string) map[string]interface{} {
	canRequest := s.faucetLimiter.Allow(network, address)
	timeLeft := s.faucetLimiter.TimeLeft(network, address)

	return map[string]interface{}{
		"canRequest": canRequest,
		"timeLeft":   timeLeft.Milliseconds(),
		"hoursLeft":  int(timeLeft.Hours()),
	}
}

// CreateStripeCheckoutSession creates a Stripe checkout session
func (s *Service) CreateStripeCheckoutSession(req *PaymentRequest) (*StripeCheckoutSession, error) {
	if s.stripeClient == nil {
		return nil, fmt.Errorf("Stripe client not configured")
	}

	networkConfig, exists := s.networks[req.Network]
	if !exists || !networkConfig.PaymentEnabled {
		return nil, fmt.Errorf("payments not available for this network")
	}

	if !s.isValidAddress(req.Address) {
		return nil, fmt.Errorf("invalid wallet address")
	}

	return s.stripeClient.CreateCheckoutSession(req, networkConfig)
}

// CreateCoinbaseCharge creates a Coinbase Commerce charge
func (s *Service) CreateCoinbaseCharge(req *PaymentRequest) (*CoinbaseCharge, error) {
	if s.coinbaseClient == nil {
		return nil, fmt.Errorf("Coinbase client not configured")
	}

	networkConfig, exists := s.networks[req.Network]
	if !exists || !networkConfig.PaymentEnabled {
		return nil, fmt.Errorf("payments not available for this network")
	}

	if !s.isValidAddress(req.Address) {
		return nil, fmt.Errorf("invalid wallet address")
	}

	return s.coinbaseClient.CreateCharge(req, networkConfig)
}

// ProcessSkillInvocation processes a skill invocation
func (s *Service) ProcessSkillInvocation(req *SkillInvocationRequest) (*EconomicTransaction, error) {
	if s.economics == nil {
		return nil, fmt.Errorf("economics engine not enabled")
	}

	return s.economics.ProcessSkillInvocation(req)
}

// ProcessLLMRegistration processes an LLM registration
func (s *Service) ProcessLLMRegistration(req *LLMRegistrationRequest) (*EconomicTransaction, error) {
	if s.economics == nil {
		return nil, fmt.Errorf("economics engine not enabled")
	}

	return s.economics.ProcessLLMRegistration(req)
}

// ProcessValidationReward processes a validation reward
func (s *Service) ProcessValidationReward(req *ValidationRewardRequest) (*EconomicTransaction, error) {
	if s.economics == nil {
		return nil, fmt.Errorf("economics engine not enabled")
	}

	return s.economics.ProcessValidationReward(req)
}

// ProcessValidationReport processes a public validation report payment
func (s *Service) ProcessValidationReport(req *ValidationReportRequest) (*EconomicTransaction, error) {
	if s.economics == nil {
		return nil, fmt.Errorf("economics engine not enabled")
	}

	return s.economics.ProcessValidationReport(req)
}

// CalculateNetworkFees calculates network fees
func (s *Service) CalculateNetworkFees(req *FeeCalculationRequest) *FeeCalculationResponse {
	if s.economics == nil {
		// Return basic calculation if economics engine is disabled
		baseGasPrice := big.NewInt(1000)
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

	return s.economics.CalculateNetworkFees(req)
}

// GetEconomicMetrics returns economic metrics
func (s *Service) GetEconomicMetrics() *EconomicMetrics {
	if s.economics == nil {
		return &EconomicMetrics{
			ServiceMetrics: make(map[string]*ServiceMetrics),
		}
	}

	return s.economics.GetMetrics()
}

// GetEconomicRules returns economic rules
func (s *Service) GetEconomicRules() *EconomicRules {
	if s.economics == nil {
		return &EconomicRules{}
	}

	return s.economics.GetRules()
}

// GetTransactions returns transactions with optional filtering
func (s *Service) GetTransactions(limit int, status string) []*EconomicTransaction {
	if s.economics == nil {
		return []*EconomicTransaction{}
	}

	return s.economics.GetTransactions(limit, status)
}

// GetTransaction retrieves a transaction from the configured economics ledger.
func (s *Service) GetTransaction(id string) (*EconomicTransaction, bool) {
	if s.economics == nil {
		return nil, false
	}
	return s.economics.GetTransaction(id)
}

// GetBurnHistory returns burn history
func (s *Service) GetBurnHistory(limit int, user, purpose string) []*BurnEvent {
	if s.economics == nil {
		return []*BurnEvent{}
	}

	return s.economics.GetBurnHistory(limit, user, purpose)
}

// GetTotalBurned returns total burned amount
func (s *Service) GetTotalBurned() string {
	if s.economics == nil {
		return "0"
	}

	return s.economics.GetTotalBurned().String()
}

// GetServiceMetrics returns metrics for a specific service
func (s *Service) GetServiceMetrics(serviceName string) *ServiceMetrics {
	if s.economics == nil {
		return &ServiceMetrics{}
	}

	return s.economics.GetServiceMetrics(serviceName)
}

// Helper functions

func (s *Service) isValidAddress(address string) bool {
	// Basic validation: starts with "knirv" or "0x"
	return len(address) > 0 && (address[:5] == "knirv" || address[:2] == "0x")
}

func (s *Service) generateTransactionID(prefix, network string) string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	randomStr := hex.EncodeToString(bytes)
	return fmt.Sprintf("%s_%s_%d_%s", prefix, network, time.Now().Unix(), randomStr)
}
