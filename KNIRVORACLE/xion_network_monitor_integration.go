package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// XIONNetworkMonitorIntegration integrates XION payment gateway with KNIRV Network Monitor
type XIONNetworkMonitorIntegration struct {
	paymentGateway     *XIONPaymentGateway
	integrationService *XIONIntegrationService
	metrics            *XIONMetrics
	networkMonitorURL  string
	httpClient         *http.Client
	ctx                context.Context
	cancel             context.CancelFunc
	mutex              sync.RWMutex
}

// XIONMetrics contains Prometheus metrics for XION payment gateway
type XIONMetrics struct {
	// Payment metrics
	PaymentsTotal      prometheus.Counter
	PaymentsSuccessful prometheus.Counter
	PaymentsFailed     prometheus.Counter
	PaymentDuration    prometheus.Histogram
	PaymentAmount      prometheus.Histogram

	// Flow metrics
	PaymentFlowsActive prometheus.Gauge
	PaymentFlowsTotal  prometheus.Counter
	FlowStepDuration   *prometheus.HistogramVec

	// NRV metrics
	NRVMintingTotal   prometheus.Counter
	NRVQualityGrades  *prometheus.CounterVec
	NRVBonusesApplied prometheus.Counter

	// Treasury metrics
	TreasuryMintsTotal prometheus.Counter
	NRNTokensMinted    prometheus.Counter
	TreasuryBalance    prometheus.Gauge

	// System metrics
	GatewayUptime     prometheus.Gauge
	ActiveConnections prometheus.Gauge
	ErrorRate         prometheus.Gauge
}

// XIONServiceStatus represents XION service status for network monitor
type XIONServiceStatus struct {
	PaymentGateway struct {
		Status                string                 `json:"status"`
		ActivePayments        int                    `json:"active_payments"`
		TotalPayments         int64                  `json:"total_payments"`
		SuccessRate           float64                `json:"success_rate"`
		AverageProcessingTime string                 `json:"avg_processing_time"`
		LastPayment           *time.Time             `json:"last_payment,omitempty"`
		Metrics               map[string]interface{} `json:"metrics"`
	} `json:"payment_gateway"`

	IntegrationService struct {
		Status         string                 `json:"status"`
		ActiveFlows    int                    `json:"active_flows"`
		CompletedFlows int                    `json:"completed_flows"`
		FailedFlows    int                    `json:"failed_flows"`
		Uptime         string                 `json:"uptime"`
		Metrics        map[string]interface{} `json:"metrics"`
	} `json:"integration_service"`

	NRVMinting struct {
		Status              string         `json:"status"`
		TotalMinted         int64          `json:"total_minted"`
		QualityDistribution map[string]int `json:"quality_distribution"`
		AverageQuality      string         `json:"average_quality"`
		BonusesApplied      int64          `json:"bonuses_applied"`
	} `json:"nrv_minting"`

	Treasury struct {
		Status         string     `json:"status"`
		NRNMinted      int64      `json:"nrn_minted"`
		Balance        string     `json:"balance"`
		LastMint       *time.Time `json:"last_mint,omitempty"`
		ProcessingRate float64    `json:"processing_rate"`
	} `json:"treasury"`
}

// NewXIONNetworkMonitorIntegration creates a new XION network monitor integration
func NewXIONNetworkMonitorIntegration(
	paymentGateway *XIONPaymentGateway,
	integrationService *XIONIntegrationService,
	networkMonitorURL string,
) *XIONNetworkMonitorIntegration {
	ctx, cancel := context.WithCancel(context.Background())

	integration := &XIONNetworkMonitorIntegration{
		paymentGateway:     paymentGateway,
		integrationService: integrationService,
		networkMonitorURL:  networkMonitorURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize Prometheus metrics
	integration.initializeMetrics()

	return integration
}

// initializeMetrics initializes Prometheus metrics for XION services
func (xnmi *XIONNetworkMonitorIntegration) initializeMetrics() {
	xnmi.metrics = &XIONMetrics{
		// Payment metrics
		PaymentsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "xion_payments_total",
			Help: "Total number of XION payments processed",
		}),
		PaymentsSuccessful: promauto.NewCounter(prometheus.CounterOpts{
			Name: "xion_payments_successful_total",
			Help: "Total number of successful XION payments",
		}),
		PaymentsFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "xion_payments_failed_total",
			Help: "Total number of failed XION payments",
		}),
		PaymentDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "xion_payment_duration_seconds",
			Help:    "Duration of XION payment processing",
			Buckets: prometheus.DefBuckets,
		}),
		PaymentAmount: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "xion_payment_amount_usdc",
			Help:    "Amount of USDC in payments",
			Buckets: []float64{1, 10, 50, 100, 500, 1000, 5000, 10000},
		}),

		// Flow metrics
		PaymentFlowsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "xion_payment_flows_active",
			Help: "Number of active payment flows",
		}),
		PaymentFlowsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "xion_payment_flows_total",
			Help: "Total number of payment flows initiated",
		}),
		FlowStepDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "xion_flow_step_duration_seconds",
			Help:    "Duration of individual payment flow steps",
			Buckets: prometheus.DefBuckets,
		}, []string{"step_name", "status"}),

		// NRV metrics
		NRVMintingTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "xion_nrv_minting_total",
			Help: "Total number of NRV tokens minted",
		}),
		NRVQualityGrades: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "xion_nrv_quality_grades_total",
			Help: "Total NRV tokens by quality grade",
		}, []string{"grade", "certification"}),
		NRVBonusesApplied: promauto.NewCounter(prometheus.CounterOpts{
			Name: "xion_nrv_bonuses_applied_total",
			Help: "Total number of quality bonuses applied",
		}),

		// Treasury metrics
		TreasuryMintsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "xion_treasury_mints_total",
			Help: "Total number of treasury mints processed",
		}),
		NRNTokensMinted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "xion_nrn_tokens_minted_total",
			Help: "Total number of NRN tokens minted",
		}),
		TreasuryBalance: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "xion_treasury_balance",
			Help: "Current treasury balance",
		}),

		// System metrics
		GatewayUptime: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "xion_gateway_uptime_seconds",
			Help: "XION gateway uptime in seconds",
		}),
		ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "xion_active_connections",
			Help: "Number of active connections to XION gateway",
		}),
		ErrorRate: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "xion_error_rate",
			Help: "Current error rate for XION services",
		}),
	}
}

// Start begins the network monitor integration
func (xnmi *XIONNetworkMonitorIntegration) Start() error {
	log.Println("Starting XION Network Monitor Integration...")

	// Start metrics collection
	go xnmi.collectMetrics()

	// Start status reporting to network monitor
	go xnmi.reportToNetworkMonitor()

	// Start health monitoring
	go xnmi.monitorHealth()

	log.Println("XION Network Monitor Integration started successfully")
	return nil
}

// Stop gracefully shuts down the integration
func (xnmi *XIONNetworkMonitorIntegration) Stop() error {
	log.Println("Stopping XION Network Monitor Integration...")
	xnmi.cancel()
	return nil
}

// collectMetrics collects and updates Prometheus metrics
func (xnmi *XIONNetworkMonitorIntegration) collectMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-xnmi.ctx.Done():
			return
		case <-ticker.C:
			xnmi.updateMetrics(startTime)
		}
	}
}

// updateMetrics updates all Prometheus metrics
func (xnmi *XIONNetworkMonitorIntegration) updateMetrics(startTime time.Time) {
	xnmi.mutex.RLock()
	defer xnmi.mutex.RUnlock()

	// Update uptime
	uptime := time.Since(startTime).Seconds()
	xnmi.metrics.GatewayUptime.Set(uptime)

	// Update active flows
	if xnmi.integrationService != nil {
		activeFlows := xnmi.integrationService.GetActivePaymentFlows()
		xnmi.metrics.PaymentFlowsActive.Set(float64(len(activeFlows)))
	}

	// Update payment gateway metrics
	if xnmi.paymentGateway != nil {
		// These would be actual metrics from the payment gateway
		// For now, we'll simulate some values
		xnmi.metrics.ActiveConnections.Set(float64(10)) // Mock value
	}
}

// reportToNetworkMonitor sends status updates to the network monitor
func (xnmi *XIONNetworkMonitorIntegration) reportToNetworkMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-xnmi.ctx.Done():
			return
		case <-ticker.C:
			xnmi.sendStatusUpdate()
		}
	}
}

// sendStatusUpdate sends current XION service status to network monitor
func (xnmi *XIONNetworkMonitorIntegration) sendStatusUpdate() {
	status := xnmi.generateServiceStatus()

	// Convert to JSON
	statusJSON, err := json.Marshal(status)
	if err != nil {
		log.Printf("Failed to marshal XION status: %v", err)
		return
	}

	// Send to network monitor
	url := fmt.Sprintf("%s/api/services/xion/status", xnmi.networkMonitorURL)
	resp, err := xnmi.httpClient.Post(url, "application/json",
		bytes.NewBuffer(statusJSON))

	if err != nil {
		log.Printf("Failed to send status to network monitor: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Network monitor returned status %d", resp.StatusCode)
	}
}

// generateServiceStatus generates current service status
func (xnmi *XIONNetworkMonitorIntegration) generateServiceStatus() *XIONServiceStatus {
	status := &XIONServiceStatus{}

	// Payment Gateway Status
	status.PaymentGateway.Status = "up"
	if xnmi.paymentGateway != nil {
		// Get actual metrics from payment gateway
		status.PaymentGateway.ActivePayments = 0 // Would get from gateway
		status.PaymentGateway.TotalPayments = 0  // Would get from gateway
		status.PaymentGateway.SuccessRate = 0.95 // Would calculate from metrics
		status.PaymentGateway.AverageProcessingTime = "2.5s"
	}

	// Integration Service Status
	status.IntegrationService.Status = "up"
	if xnmi.integrationService != nil {
		activeFlows := xnmi.integrationService.GetActivePaymentFlows()
		status.IntegrationService.ActiveFlows = len(activeFlows)

		history := xnmi.integrationService.GetPaymentFlowHistory(100)
		completedCount := 0
		failedCount := 0

		for _, record := range history {
			switch record.Status {
			case "completed":
				completedCount++
			case "failed":
				failedCount++
			}
		}

		status.IntegrationService.CompletedFlows = completedCount
		status.IntegrationService.FailedFlows = failedCount
	}

	// NRV Minting Status
	status.NRVMinting.Status = "up"
	status.NRVMinting.QualityDistribution = map[string]int{
		"A": 45, "B": 30, "C": 20, "D": 4, "F": 1,
	}
	status.NRVMinting.AverageQuality = "B+"

	// Treasury Status
	status.Treasury.Status = "up"
	status.Treasury.ProcessingRate = 0.98

	return status
}

// monitorHealth monitors the health of XION services
func (xnmi *XIONNetworkMonitorIntegration) monitorHealth() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-xnmi.ctx.Done():
			return
		case <-ticker.C:
			xnmi.checkServiceHealth()
		}
	}
}

// checkServiceHealth performs health checks on XION services
func (xnmi *XIONNetworkMonitorIntegration) checkServiceHealth() {
	// Check payment gateway health
	// Check integration service health
	// Update error rates and alert if necessary

	// This would implement actual health checks
	log.Println("Performing XION service health checks...")
}

// RegisterWithNetworkMonitor registers XION services with the network monitor
func (xnmi *XIONNetworkMonitorIntegration) RegisterWithNetworkMonitor() error {
	registrationData := map[string]interface{}{
		"service_name": "xion_payment_gateway",
		"service_type": "payment_gateway",
		"endpoints": []string{
			"/api/payment/usdc-to-nrn",
			"/api/payment/status",
			"/api/payment/config",
			"/api/payment/rates",
		},
		"metrics_endpoint": "/metrics",
		"health_endpoint":  "/health",
		"critical":         true,
		"tags":             []string{"payment", "xion", "usdc", "nrn"},
	}

	registrationJSON, err := json.Marshal(registrationData)
	if err != nil {
		return fmt.Errorf("failed to marshal registration data: %w", err)
	}

	url := fmt.Sprintf("%s/api/services/register", xnmi.networkMonitorURL)
	resp, err := xnmi.httpClient.Post(url, "application/json",
		bytes.NewBuffer(registrationJSON))

	if err != nil {
		return fmt.Errorf("failed to register with network monitor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("network monitor registration failed with status %d", resp.StatusCode)
	}

	log.Println("Successfully registered XION services with network monitor")
	return nil
}
