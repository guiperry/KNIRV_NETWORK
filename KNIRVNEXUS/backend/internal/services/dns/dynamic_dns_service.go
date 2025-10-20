package dns

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	// dataengine "backend_server/internal/services/data-engine" // TODO: Re-enable when data-engine is available
	dataengine "backend_server/internal/data-engine"
	"backend_server/pkg/cloudflare"
)

// DynamicDNSService manages dynamic DNS updates
type DynamicDNSService struct {
	// CloudFlare DNS manager
	dnsManager *cloudflare.DNSManager
	dataEngine *dataengine.BuntDBDataEngine

	// Configuration
	config DNSConfig

	// State management
	isRunning bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc

	// Current state
	currentIP   string
	lastUpdate  time.Time
	updateCount int64
	errorCount  int64
}

// DNSConfig contains configuration for the DNS service
type DNSConfig struct {
	// CloudFlare settings
	CloudFlareAPIToken string `yaml:"cloudflare_api_token"`
	ZoneName           string `yaml:"zone_name"`

	// Update settings
	UpdateInterval      time.Duration `yaml:"update_interval"`
	ForceUpdateInterval time.Duration `yaml:"force_update_interval"`

	// DNS records to manage
	Records []DNSRecordConfig `yaml:"records"`

	// Health check settings
	EnableHealthCheck   bool          `yaml:"enable_health_check"`
	HealthCheckURL      string        `yaml:"health_check_url"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`

	// Retry settings
	MaxRetries    int           `yaml:"max_retries"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
	BackoffFactor float64       `yaml:"backoff_factor"`
}

// DNSRecordConfig represents a DNS record configuration
type DNSRecordConfig struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	TTL      int    `yaml:"ttl"`
	Proxied  bool   `yaml:"proxied"`
	Priority int    `yaml:"priority,omitempty"`
	Comment  string `yaml:"comment"`

	// Dynamic update settings
	UpdateWithIP  bool   `yaml:"update_with_ip"`
	StaticContent string `yaml:"static_content,omitempty"`
}

// NewDynamicDNSService creates a new dynamic DNS service
func NewDynamicDNSService(dataEngine *dataengine.BuntDBDataEngine, config DNSConfig) (*DynamicDNSService, error) {
	// Allow development mode with placeholder token
	if config.CloudFlareAPIToken == "" {
		return nil, fmt.Errorf("CloudFlare API token is required")
	}

	if config.ZoneName == "" {
		return nil, fmt.Errorf("zone name is required")
	}

	// Set default values
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 5 * time.Minute
	}
	if config.ForceUpdateInterval == 0 {
		config.ForceUpdateInterval = 1 * time.Hour
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 30 * time.Second
	}
	if config.BackoffFactor == 0 {
		config.BackoffFactor = 2.0
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize DNS manager with the provided API token
	dnsManager := cloudflare.NewDNSManager(config.CloudFlareAPIToken)

	service := &DynamicDNSService{
		dnsManager: dnsManager,
		dataEngine: dataEngine,
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
	}

	return service, nil
}

// Start starts the dynamic DNS service
func (dds *DynamicDNSService) Start() error {
	dds.mu.Lock()
	defer dds.mu.Unlock()

	if dds.isRunning {
		return fmt.Errorf("dynamic DNS service is already running")
	}

	// Start monitoring loops
	go dds.ipMonitoringLoop()
	go dds.healthCheckLoop()
	go dds.forceUpdateLoop()

	dds.isRunning = true
	log.Println("DynamicDNSService: Started successfully")

	return nil
}

// Stop stops the dynamic DNS service
func (dds *DynamicDNSService) Stop() error {
	dds.mu.Lock()
	defer dds.mu.Unlock()

	if !dds.isRunning {
		return nil
	}

	dds.cancel()
	dds.isRunning = false

	log.Println("DynamicDNSService: Stopped successfully")

	return nil
}

// ipMonitoringLoop monitors IP changes and updates DNS records
func (dds *DynamicDNSService) ipMonitoringLoop() {
	ticker := time.NewTicker(dds.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dds.ctx.Done():
			return
		case <-ticker.C:
			dds.checkAndUpdateIP()
		}
	}
}

// checkAndUpdateIP checks for IP changes and updates DNS records if needed
func (dds *DynamicDNSService) checkAndUpdateIP() {
	// Get current public IP
	currentIP, err := dds.getCurrentPublicIP()
	if err != nil {
		log.Printf("DynamicDNSService: Failed to get current IP: %v", err)
		dds.errorCount++
		return
	}

	dds.mu.Lock()
	ipChanged := currentIP != dds.currentIP
	dds.currentIP = currentIP
	dds.mu.Unlock()

	if ipChanged {
		log.Printf("DynamicDNSService: IP changed to %s, updating DNS records", currentIP)
		if err := dds.updateDNSRecords(currentIP); err != nil {
			log.Printf("DynamicDNSService: Failed to update DNS records: %v", err)
			dds.errorCount++
		} else {
			dds.mu.Lock()
			dds.lastUpdate = time.Now()
			dds.updateCount++
			dds.mu.Unlock()

			// Log metrics
			if dds.dataEngine != nil {
				dds.dataEngine.ProcessMetricEvent(
					"dns-service",
					"ip_updated",
					1.0,
					"count",
					map[string]string{
						"new_ip": currentIP,
						"zone":   dds.config.ZoneName,
					},
				)
			}
		}
	}
}

// updateDNSRecords updates all configured DNS records
func (dds *DynamicDNSService) updateDNSRecords(newIP string) error {
	// Get zone
	zone, err := dds.dnsManager.GetZoneByName(dds.config.ZoneName)
	if err != nil {
		return fmt.Errorf("failed to get zone: %w", err)
	}

	// Update each configured record
	for _, recordConfig := range dds.config.Records {
		var content string
		if recordConfig.UpdateWithIP {
			content = newIP
		} else if recordConfig.StaticContent != "" {
			content = recordConfig.StaticContent
		} else {
			continue // Skip records without content
		}

		record := cloudflare.DNSRecord{
			Type:     recordConfig.Type,
			Name:     recordConfig.Name,
			Content:  content,
			TTL:      recordConfig.TTL,
			Proxied:  recordConfig.Proxied,
			Priority: recordConfig.Priority,
		}

		if err := dds.updateRecordWithRetry(zone.ID, record); err != nil {
			log.Printf("DynamicDNSService: Failed to update record %s: %v", recordConfig.Name, err)
			continue
		}

		log.Printf("DynamicDNSService: Updated record %s (%s) to %s", recordConfig.Name, recordConfig.Type, content)
	}

	return nil
}

// updateRecordWithRetry updates a DNS record with retry logic
func (dds *DynamicDNSService) updateRecordWithRetry(zoneID string, record cloudflare.DNSRecord) error {
	var lastErr error
	delay := dds.config.RetryDelay

	for attempt := 0; attempt <= dds.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * dds.config.BackoffFactor)
		}

		_, err := dds.dnsManager.UpdateOrCreateDNSRecord(zoneID, record)
		if err == nil {
			return nil
		}

		lastErr = err
		log.Printf("DynamicDNSService: Attempt %d failed for record %s: %v", attempt+1, record.Name, err)
	}

	return fmt.Errorf("failed after %d attempts: %w", dds.config.MaxRetries+1, lastErr)
}

// healthCheckLoop performs periodic health checks
func (dds *DynamicDNSService) healthCheckLoop() {
	if !dds.config.EnableHealthCheck || dds.config.HealthCheckURL == "" {
		return
	}

	ticker := time.NewTicker(dds.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dds.ctx.Done():
			return
		case <-ticker.C:
			dds.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check
func (dds *DynamicDNSService) performHealthCheck() {
	// Perform HTTP health check if URL is configured
	if dds.config.HealthCheckURL != "" {
		if err := dds.performHTTPHealthCheck(); err != nil {
			log.Printf("DynamicDNSService: Health check failed: %v", err)
			dds.errorCount++

			// Log failed health check
			if dds.dataEngine != nil {
				dds.dataEngine.ProcessMetricEvent(
					"dns-service",
					"health_check_failed",
					1.0,
					"count",
					map[string]string{
						"url":   dds.config.HealthCheckURL,
						"error": err.Error(),
					},
				)
			}
			return
		}
	}

	// Perform DNS propagation check
	if err := dds.performDNSPropagationCheck(); err != nil {
		log.Printf("DynamicDNSService: DNS propagation check failed: %v", err)
		// Don't increment error count for propagation issues as they may be temporary
	}

	log.Printf("DynamicDNSService: Health check completed successfully")

	// Log successful health check
	if dds.dataEngine != nil {
		dds.dataEngine.ProcessMetricEvent(
			"dns-service",
			"health_check_success",
			1.0,
			"count",
			map[string]string{
				"url": dds.config.HealthCheckURL,
			},
		)
	}
}

// forceUpdateLoop performs periodic forced updates
func (dds *DynamicDNSService) forceUpdateLoop() {
	ticker := time.NewTicker(dds.config.ForceUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dds.ctx.Done():
			return
		case <-ticker.C:
			dds.forceUpdate()
		}
	}
}

// forceUpdate forces an update of all DNS records
func (dds *DynamicDNSService) forceUpdate() {
	log.Println("DynamicDNSService: Performing forced update")

	currentIP, err := dds.getCurrentPublicIP()
	if err != nil {
		log.Printf("DynamicDNSService: Failed to get current IP for forced update: %v", err)
		return
	}

	if err := dds.updateDNSRecords(currentIP); err != nil {
		log.Printf("DynamicDNSService: Failed forced update: %v", err)
		dds.errorCount++
	} else {
		dds.mu.Lock()
		dds.lastUpdate = time.Now()
		dds.updateCount++
		dds.mu.Unlock()
	}
}

// getCurrentPublicIP gets the current public IP address
func (dds *DynamicDNSService) getCurrentPublicIP() (string, error) {
	// Use the CloudFlare DNS manager's method for IP detection
	return dds.dnsManager.GetCurrentPublicIP()
}

// GetStatus returns the current status of the DNS service
func (dds *DynamicDNSService) GetStatus() map[string]interface{} {
	dds.mu.RLock()
	defer dds.mu.RUnlock()

	return map[string]interface{}{
		"running":      dds.isRunning,
		"current_ip":   dds.currentIP,
		"last_update":  dds.lastUpdate,
		"update_count": dds.updateCount,
		"error_count":  dds.errorCount,
		"zone_name":    dds.config.ZoneName,
		"record_count": len(dds.config.Records),
	}
}

// performHTTPHealthCheck performs an HTTP health check
func (dds *DynamicDNSService) performHTTPHealthCheck() error {
	// Use a standard HTTP client to check the health URL
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(dds.config.HealthCheckURL)
	if err != nil {
		return fmt.Errorf("HTTP health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP health check returned status %d", resp.StatusCode)
	}

	return nil
}

// performDNSPropagationCheck verifies DNS record propagation
func (dds *DynamicDNSService) performDNSPropagationCheck() error {
	// Get current IP
	currentIP, err := dds.getCurrentPublicIP()
	if err != nil {
		return fmt.Errorf("failed to get current IP for propagation check: %w", err)
	}

	// Check if our DNS records match the current IP
	zone, err := dds.dnsManager.GetZoneByName(dds.config.ZoneName)
	if err != nil {
		return fmt.Errorf("failed to get zone for propagation check: %w", err)
	}

	for _, recordConfig := range dds.config.Records {
		if !recordConfig.UpdateWithIP {
			continue // Only check records that should have the current IP
		}

		record, err := dds.dnsManager.GetDNSRecord(zone.ID, recordConfig.Name, recordConfig.Type)
		if err != nil {
			return fmt.Errorf("failed to get DNS record %s for propagation check: %w", recordConfig.Name, err)
		}

		if record.Content != currentIP {
			return fmt.Errorf("DNS record %s has content %s, expected %s", recordConfig.Name, record.Content, currentIP)
		}
	}

	return nil
}

// IsRunning returns whether the DNS service is running
func (dds *DynamicDNSService) IsRunning() bool {
	dds.mu.RLock()
	defer dds.mu.RUnlock()
	return dds.isRunning
}
