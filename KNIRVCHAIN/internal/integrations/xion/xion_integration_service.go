package xion

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"sync"
	"time"

	"KNIRVCHAIN/economics"
)

// XIONIntegrationService orchestrates the complete XION payment flow
type XIONIntegrationService struct {
	config         *XIONIntegrationConfig
	paymentGateway *XIONPaymentGateway
	economicsAPI   *economics.EconomicsAPI
	routerClient   *KNIRVRouterClient
	networkMonitor *XIONNetworkMonitorIntegration
	httpClient     *http.Client
	activePayments map[string]*PaymentFlow
	paymentFlows   []PaymentFlowRecord
	mutex          sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// XIONIntegrationConfig contains configuration for the integration service
type XIONIntegrationConfig struct {
	XIONGateway      XIONGatewayConfig      `json:"xion_payment_gateway"`
	KNIRVIntegration KNIRVIntegrationConfig `json:"knirv_integration"`
	Monitoring       MonitoringConfig       `json:"monitoring"`
	Security         SecurityConfig         `json:"security"`
	Testing          TestingConfig          `json:"testing"`
	Deployment       DeploymentConfig       `json:"deployment"`
}

// KNIRVIntegrationConfig contains KNIRV component integration settings
type KNIRVIntegrationConfig struct {
	Router     RouterConfig     `json:"router"`
	Oracle     OracleConfig     `json:"oracle"`
	Controller ControllerConfig `json:"controller"`
}

// RouterConfig contains KNIRVROUTER integration settings
type RouterConfig struct {
	Enabled    bool             `json:"enabled"`
	Endpoint   string           `json:"endpoint"`
	NRVMinting NRVMintingConfig `json:"nrv_minting"`
}

// NRVMintingConfig contains NRV minting configuration
type NRVMintingConfig struct {
	Enabled              bool               `json:"enabled"`
	QualityBonuses       map[string]float64 `json:"quality_bonuses"`
	CertificationBonuses map[string]float64 `json:"certification_bonuses"`
}

// OracleConfig contains KNIRVCHAIN integration settings
type OracleConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	Treasury struct {
		Endpoint           string `json:"endpoint"`
		ValidationEndpoint string `json:"validation_endpoint"`
	} `json:"treasury"`
}

// ControllerConfig contains KNIRVCONTROLLER integration settings
type ControllerConfig struct {
	Enabled           bool                    `json:"enabled"`
	Endpoint          string                  `json:"endpoint"`
	WalletIntegration WalletIntegrationConfig `json:"wallet_integration"`
}

// WalletIntegrationConfig contains wallet integration settings
type WalletIntegrationConfig struct {
	AutoConnect    bool   `json:"auto_connect"`
	PreferredAuth  string `json:"preferred_auth"`
	GaslessDefault bool   `json:"gasless_default"`
}

// MonitoringConfig contains monitoring settings
type MonitoringConfig struct {
	Enabled          bool                   `json:"enabled"`
	PaymentTracking  PaymentTrackingConfig  `json:"payment_tracking"`
	NRVTracking      NRVTrackingConfig      `json:"nrv_tracking"`
	TreasuryTracking TreasuryTrackingConfig `json:"treasury_tracking"`
}

// PaymentTrackingConfig contains payment tracking settings
type PaymentTrackingConfig struct {
	StatusCheckInterval string `json:"status_check_interval"`
	Timeout             string `json:"timeout"`
	RetryAttempts       int    `json:"retry_attempts"`
}

// NRVTrackingConfig contains NRV tracking settings
type NRVTrackingConfig struct {
	RouteValidation      bool `json:"route_validation"`
	QualityAssessment    bool `json:"quality_assessment"`
	MetadataVerification bool `json:"metadata_verification"`
}

// TreasuryTrackingConfig contains treasury tracking settings
type TreasuryTrackingConfig struct {
	MintValidation     bool `json:"mint_validation"`
	BalanceMonitoring  bool `json:"balance_monitoring"`
	TransactionLogging bool `json:"transaction_logging"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	RateLimiting RateLimitingConfig `json:"rate_limiting"`
	Validation   ValidationConfig   `json:"validation"`
	Encryption   EncryptionConfig   `json:"encryption"`
}

// RateLimitingConfig contains rate limiting settings
type RateLimitingConfig struct {
	Enabled           bool `json:"enabled"`
	RequestsPerMinute int  `json:"requests_per_minute"`
	BurstLimit        int  `json:"burst_limit"`
}

// ValidationConfig contains validation settings
type ValidationConfig struct {
	AddressVerification bool `json:"address_verification"`
	AmountLimits        bool `json:"amount_limits"`
	SignatureRequired   bool `json:"signature_required"`
}

// EncryptionConfig contains encryption settings
type EncryptionConfig struct {
	Enabled     bool   `json:"enabled"`
	Algorithm   string `json:"algorithm"`
	KeyRotation string `json:"key_rotation"`
}

// TestingConfig contains testing settings
type TestingConfig struct {
	TestnetMode      bool          `json:"testnet_mode"`
	MockTransactions bool          `json:"mock_transactions"`
	DebugLogging     bool          `json:"debug_logging"`
	TestAccounts     []TestAccount `json:"test_accounts"`
}

// TestAccount represents a test account
type TestAccount struct {
	Name        string `json:"name"`
	Address     string `json:"address"`
	USDCBalance string `json:"usdc_balance"`
	NRNBalance  string `json:"nrn_balance"`
}

// DeploymentConfig contains deployment settings
type DeploymentConfig struct {
	Environment  string            `json:"environment"`
	AutoStart    bool              `json:"auto_start"`
	HealthChecks HealthCheckConfig `json:"health_checks"`
	Backup       BackupConfig      `json:"backup"`
}

// HealthCheckConfig contains health check settings
type HealthCheckConfig struct {
	Enabled   bool     `json:"enabled"`
	Interval  string   `json:"interval"`
	Endpoints []string `json:"endpoints"`
}

// BackupConfig contains backup settings
type BackupConfig struct {
	Enabled   bool   `json:"enabled"`
	Interval  string `json:"interval"`
	Retention string `json:"retention"`
}

// PaymentFlow represents a complete payment flow from USDC to NRN
type PaymentFlow struct {
	FlowID       string                 `json:"flow_id"`
	PaymentID    string                 `json:"payment_id"`
	UserAddress  string                 `json:"user_address"`
	USDCAmount   string                 `json:"usdc_amount"`
	NRNAmount    string                 `json:"nrn_amount"`
	Status       string                 `json:"status"` // initiated, nrv_minting, treasury_processing, completed, failed
	Steps        []PaymentFlowStep      `json:"steps"`
	CreatedAt    time.Time              `json:"created_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// PaymentFlowStep represents a step in the payment flow
type PaymentFlowStep struct {
	StepName    string                 `json:"step_name"`
	Status      string                 `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// PaymentFlowRecord represents a completed payment flow record
type PaymentFlowRecord struct {
	FlowID         string    `json:"flow_id"`
	PaymentID      string    `json:"payment_id"`
	UserAddress    string    `json:"user_address"`
	USDCAmount     string    `json:"usdc_amount"`
	NRNAmount      string    `json:"nrn_amount"`
	Status         string    `json:"status"`
	Duration       string    `json:"duration"`
	CompletedAt    time.Time `json:"completed_at"`
	NRVMetadata    string    `json:"nrv_metadata,omitempty"`
	TreasuryTxHash string    `json:"treasury_tx_hash,omitempty"`
}

// KNIRVRouterClient handles communication with KNIRVROUTER
type KNIRVRouterClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewXIONIntegrationService creates a new XION integration service
func NewXIONIntegrationService(configPath string, economicsAPI *economics.EconomicsAPI) (*XIONIntegrationService, error) {
	// Load configuration
	config, err := loadXIONIntegrationConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize XION payment gateway
	xionGatewayConfig := &config.XIONGateway

	paymentGateway := NewXIONPaymentGateway(xionGatewayConfig, economicsAPI)

	// Initialize KNIRVROUTER client
	routerClient := &KNIRVRouterClient{
		endpoint: config.KNIRVIntegration.Router.Endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	return &XIONIntegrationService{
		config:         config,
		paymentGateway: paymentGateway,
		economicsAPI:   economicsAPI,
		routerClient:   routerClient,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		activePayments: make(map[string]*PaymentFlow),
		paymentFlows:   make([]PaymentFlowRecord, 0),
		ctx:            ctx,
		cancel:         cancel,
	}, nil
}

// loadXIONIntegrationConfig loads configuration from file
func loadXIONIntegrationConfig(configPath string) (*XIONIntegrationConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config XIONIntegrationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// Start initializes and starts the integration service
func (xis *XIONIntegrationService) Start() error {
	log.Println("Starting XION Integration Service...")

	// Start payment gateway
	if err := xis.paymentGateway.Start(); err != nil {
		return fmt.Errorf("failed to start payment gateway: %w", err)
	}

	// Start background services
	go xis.monitorPaymentFlows()
	go xis.syncWithKNIRVRouter()
	go xis.performHealthChecks()

	log.Println("XION Integration Service started successfully")
	return nil
}

// Stop gracefully shuts down the integration service
func (xis *XIONIntegrationService) Stop() error {
	log.Println("Stopping XION Integration Service...")
	xis.cancel()

	if xis.paymentGateway != nil {
		if err := xis.paymentGateway.Stop(); err != nil {
			log.Printf("Error stopping payment gateway: %v", err)
		}
	}

	log.Println("XION Integration Service stopped")
	return nil
}

// InitiatePaymentFlow starts a complete USDC to NRN payment flow
func (xis *XIONIntegrationService) InitiatePaymentFlow(userAddress, usdcAmount, metaAccountType string, gasless bool) (*PaymentFlow, error) {
	flowID := fmt.Sprintf("flow_%d", time.Now().UnixNano())

	flow := &PaymentFlow{
		FlowID:      flowID,
		UserAddress: userAddress,
		USDCAmount:  usdcAmount,
		Status:      "initiated",
		Steps:       make([]PaymentFlowStep, 0),
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"meta_account_type": metaAccountType,
			"gasless":           gasless,
		},
	}

	// Add initial step
	flow.Steps = append(flow.Steps, PaymentFlowStep{
		StepName:    "payment_initiation",
		Status:      "completed",
		StartedAt:   time.Now(),
		CompletedAt: &[]time.Time{time.Now()}[0],
		Data: map[string]interface{}{
			"user_address": userAddress,
			"usdc_amount":  usdcAmount,
		},
	})

	// Store active payment
	xis.mutex.Lock()
	xis.activePayments[flowID] = flow
	xis.mutex.Unlock()

	// Start processing asynchronously
	go xis.processPaymentFlow(flow)

	log.Printf("Payment flow initiated: %s", flowID)
	return flow, nil
}

// processPaymentFlow processes a complete payment flow
func (xis *XIONIntegrationService) processPaymentFlow(flow *PaymentFlow) {
	log.Printf("Processing payment flow: %s", flow.FlowID)

	// Step 1: Process USDC payment via XION
	if err := xis.processUSDCPayment(flow); err != nil {
		xis.failPaymentFlow(flow, "usdc_payment", err)
		return
	}

	// Step 2: Trigger NRV minting from KNIRVROUTER
	if err := xis.triggerNRVMinting(flow); err != nil {
		xis.failPaymentFlow(flow, "nrv_minting", err)
		return
	}

	// Step 3: Process treasury minting via KNIRVCHAIN
	if err := xis.processTreasuryMinting(flow); err != nil {
		xis.failPaymentFlow(flow, "treasury_minting", err)
		return
	}

	// Step 4: Complete payment flow
	xis.completePaymentFlow(flow)
}

// processUSDCPayment handles the USDC payment via XION
func (xis *XIONIntegrationService) processUSDCPayment(flow *PaymentFlow) error {
	step := xis.addFlowStep(flow, "usdc_payment", "processing")

	// Process payment via XION gateway (this would normally call the actual gateway)
	// For now, simulate the payment processing
	time.Sleep(2 * time.Second) // Simulate processing time

	// Calculate NRN amount using the conversion rate from config
	conversionRate, _ := new(big.Int).SetString(xis.config.XIONGateway.ConversionRate, 10)
	usdcAmount, _ := new(big.Int).SetString(flow.USDCAmount, 10)
	nrnAmount := new(big.Int).Mul(usdcAmount, conversionRate)
	flow.NRNAmount = nrnAmount.String()

	// Generate mock payment ID
	flow.PaymentID = fmt.Sprintf("pay_%d", time.Now().UnixNano())

	// Complete step
	xis.completeFlowStep(flow, step, map[string]interface{}{
		"payment_id": flow.PaymentID,
		"nrn_amount": flow.NRNAmount,
		"status":     "completed",
	})

	log.Printf("USDC payment processed for flow %s: %s USDC -> %s NRN",
		flow.FlowID, flow.USDCAmount, flow.NRNAmount)
	return nil
}

// triggerNRVMinting triggers NRV minting from KNIRVROUTER
func (xis *XIONIntegrationService) triggerNRVMinting(flow *PaymentFlow) error {
	if !xis.config.KNIRVIntegration.Router.Enabled {
		log.Printf("KNIRVROUTER integration disabled, skipping NRV minting for flow %s", flow.FlowID)
		return nil
	}

	step := xis.addFlowStep(flow, "nrv_minting", "processing")
	flow.Status = "nrv_minting"

	// Send request to KNIRVROUTER (simulate for now)
	log.Printf("Triggering NRV minting for flow %s via KNIRVROUTER", flow.FlowID)
	time.Sleep(1 * time.Second) // Simulate processing

	// Generate mock NRV metadata
	nrvMetadata := map[string]interface{}{
		"nrv_id":             fmt.Sprintf("nrv_%d", time.Now().UnixNano()),
		"route_quality":      "A",
		"certification":      "premium",
		"connectivity_score": 0.95,
		"network_metrics": map[string]interface{}{
			"latency":    "50ms",
			"throughput": "1000 TPS",
			"uptime":     "99.9%",
		},
	}

	// Complete step
	xis.completeFlowStep(flow, step, map[string]interface{}{
		"nrv_metadata": nrvMetadata,
		"status":       "completed",
	})

	log.Printf("NRV minting completed for flow %s", flow.FlowID)
	return nil
}

// processTreasuryMinting processes NRN minting via KNIRVCHAIN treasury
func (xis *XIONIntegrationService) processTreasuryMinting(flow *PaymentFlow) error {
	step := xis.addFlowStep(flow, "treasury_minting", "processing")
	flow.Status = "treasury_processing"

	// Process via economics API
	log.Printf("Processing treasury minting for flow %s", flow.FlowID)

	// Simulate treasury processing
	time.Sleep(1 * time.Second)

	// Generate mock treasury transaction
	treasuryTxHash := fmt.Sprintf("treasury_tx_%d", time.Now().UnixNano())

	// Complete step
	xis.completeFlowStep(flow, step, map[string]interface{}{
		"treasury_tx_hash": treasuryTxHash,
		"status":           "completed",
	})

	log.Printf("Treasury minting completed for flow %s: %s", flow.FlowID, treasuryTxHash)
	return nil
}

// completePaymentFlow marks a payment flow as completed
func (xis *XIONIntegrationService) completePaymentFlow(flow *PaymentFlow) {
	now := time.Now()
	flow.Status = "completed"
	flow.CompletedAt = &now

	// Add completion step
	xis.addFlowStep(flow, "flow_completion", "completed")

	// Create flow record
	duration := now.Sub(flow.CreatedAt)
	record := PaymentFlowRecord{
		FlowID:      flow.FlowID,
		PaymentID:   flow.PaymentID,
		UserAddress: flow.UserAddress,
		USDCAmount:  flow.USDCAmount,
		NRNAmount:   flow.NRNAmount,
		Status:      "completed",
		Duration:    duration.String(),
		CompletedAt: now,
	}

	// Store record and remove from active payments
	xis.mutex.Lock()
	xis.paymentFlows = append(xis.paymentFlows, record)
	delete(xis.activePayments, flow.FlowID)
	xis.mutex.Unlock()

	log.Printf("Payment flow completed: %s (duration: %s)", flow.FlowID, duration.String())
}

// Helper methods for payment flow management

// addFlowStep adds a new step to a payment flow
func (xis *XIONIntegrationService) addFlowStep(flow *PaymentFlow, stepName, status string) *PaymentFlowStep {
	step := PaymentFlowStep{
		StepName:  stepName,
		Status:    status,
		StartedAt: time.Now(),
		Data:      make(map[string]interface{}),
	}

	flow.Steps = append(flow.Steps, step)
	return &flow.Steps[len(flow.Steps)-1]
}

// completeFlowStep marks a flow step as completed
func (xis *XIONIntegrationService) completeFlowStep(_ *PaymentFlow, step *PaymentFlowStep, data map[string]interface{}) {
	now := time.Now()
	step.Status = "completed"
	step.CompletedAt = &now

	for k, v := range data {
		step.Data[k] = v
	}
}

// failPaymentFlow marks a payment flow as failed
func (xis *XIONIntegrationService) failPaymentFlow(flow *PaymentFlow, stepName string, err error) {
	now := time.Now()
	flow.Status = "failed"
	flow.ErrorMessage = err.Error()
	flow.CompletedAt = &now

	// Add failure step
	step := PaymentFlowStep{
		StepName:    stepName + "_failure",
		Status:      "failed",
		StartedAt:   time.Now(),
		CompletedAt: &now,
		Error:       err.Error(),
	}
	flow.Steps = append(flow.Steps, step)

	// Remove from active payments
	xis.mutex.Lock()
	delete(xis.activePayments, flow.FlowID)
	xis.mutex.Unlock()

	log.Printf("Payment flow failed: %s at step %s: %v", flow.FlowID, stepName, err)
}

// GetPaymentFlow retrieves a payment flow by ID
func (xis *XIONIntegrationService) GetPaymentFlow(flowID string) (*PaymentFlow, error) {
	xis.mutex.RLock()
	defer xis.mutex.RUnlock()

	if flow, exists := xis.activePayments[flowID]; exists {
		return flow, nil
	}

	// Check completed flows
	for _, record := range xis.paymentFlows {
		if record.FlowID == flowID {
			// Convert record back to flow (simplified)
			flow := &PaymentFlow{
				FlowID:      record.FlowID,
				PaymentID:   record.PaymentID,
				UserAddress: record.UserAddress,
				USDCAmount:  record.USDCAmount,
				NRNAmount:   record.NRNAmount,
				Status:      record.Status,
				CompletedAt: &record.CompletedAt,
			}
			return flow, nil
		}
	}

	return nil, fmt.Errorf("payment flow not found: %s", flowID)
}

// GetActivePaymentFlows returns all active payment flows
func (xis *XIONIntegrationService) GetActivePaymentFlows() []*PaymentFlow {
	xis.mutex.RLock()
	defer xis.mutex.RUnlock()

	flows := make([]*PaymentFlow, 0, len(xis.activePayments))
	for _, flow := range xis.activePayments {
		flows = append(flows, flow)
	}

	return flows
}

// GetPaymentFlowHistory returns completed payment flow records
func (xis *XIONIntegrationService) GetPaymentFlowHistory(limit int) []PaymentFlowRecord {
	xis.mutex.RLock()
	defer xis.mutex.RUnlock()

	if limit <= 0 || limit > len(xis.paymentFlows) {
		limit = len(xis.paymentFlows)
	}

	// Return most recent flows
	start := len(xis.paymentFlows) - limit
	if start < 0 {
		start = 0
	}

	return xis.paymentFlows[start:]
}

// Background monitoring and sync methods

// monitorPaymentFlows monitors active payment flows for timeouts and issues
func (xis *XIONIntegrationService) monitorPaymentFlows() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-xis.ctx.Done():
			return
		case <-ticker.C:
			xis.checkPaymentFlowTimeouts()
		}
	}
}

// checkPaymentFlowTimeouts checks for timed out payment flows
func (xis *XIONIntegrationService) checkPaymentFlowTimeouts() {
	xis.mutex.Lock()
	defer xis.mutex.Unlock()

	timeout := 10 * time.Minute // Configurable timeout
	now := time.Now()

	for flowID, flow := range xis.activePayments {
		if now.Sub(flow.CreatedAt) > timeout {
			flow.Status = "timeout"
			flow.ErrorMessage = "Payment flow timed out"
			flow.CompletedAt = &now

			log.Printf("Payment flow timed out: %s", flowID)
			delete(xis.activePayments, flowID)
		}
	}
}

// syncWithKNIRVRouter syncs with KNIRVROUTER for NRV updates
func (xis *XIONIntegrationService) syncWithKNIRVRouter() {
	if !xis.config.KNIRVIntegration.Router.Enabled {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-xis.ctx.Done():
			return
		case <-ticker.C:
			xis.performRouterSync()
		}
	}
}

// performRouterSync performs sync with KNIRVROUTER
func (xis *XIONIntegrationService) performRouterSync() {
	// In production, this would query KNIRVROUTER for NRV updates
	log.Println("Syncing with KNIRVROUTER...")
	// Implementation would go here
}

// performHealthChecks performs periodic health checks
func (xis *XIONIntegrationService) performHealthChecks() {
	if !xis.config.Deployment.HealthChecks.Enabled {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-xis.ctx.Done():
			return
		case <-ticker.C:
			xis.checkSystemHealth()
		}
	}
}

// checkSystemHealth checks the health of all integrated systems
func (xis *XIONIntegrationService) checkSystemHealth() {
	// Check payment gateway health
	// Check KNIRVROUTER health
	// Check economics API health
	log.Println("Performing system health checks...")
	// Implementation would go here
}
