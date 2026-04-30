package economics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"KNIRVGRAPH/internal/nrv"
)

// NRNIntegration handles NRN token economics for KNIRVGRAPH
type NRNIntegration struct {
	knirvRootURL    string
	httpClient      *http.Client
	enabled         bool
	mutex           sync.RWMutex
	nrvSystem       *nrv.NRVSystem
	economicMetrics *EconomicMetrics
	rewardPool      *big.Int
	lastSync        time.Time
}

// EconomicMetrics tracks economic activity in KNIRVGRAPH
type EconomicMetrics struct {
	TotalNRVsCreated   int64     `json:"total_nrvs_created"`
	TotalSkillsInvoked int64     `json:"total_skills_invoked"`
	TotalNRNRewards    *big.Int  `json:"total_nrn_rewards"`
	TotalNRNBurned     *big.Int  `json:"total_nrn_burned"`
	ActiveErrorNodes   int64     `json:"active_error_nodes"`
	ActiveSkillNodes   int64     `json:"active_skill_nodes"`
	NetworkEfficiency  float64   `json:"network_efficiency"`
	LastUpdated        time.Time `json:"last_updated"`
}

// NRVEconomicEvent represents an economic event in the NRV system
type NRVEconomicEvent struct {
	Type      string                 `json:"type"`
	NRVID     string                 `json:"nrv_id"`
	SkillID   string                 `json:"skill_id,omitempty"`
	ErrorID   string                 `json:"error_id,omitempty"`
	UserID    string                 `json:"user_id"`
	Amount    *big.Int               `json:"amount"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SkillInvocationRequest represents a skill invocation request
type SkillInvocationRequest struct {
	UserID  string                 `json:"user_id"`
	SkillID string                 `json:"skill_id"`
	NRVID   string                 `json:"nrv_id"`
	Amount  string                 `json:"amount"`
	Context map[string]interface{} `json:"context"`
}

// RewardDistributionRequest represents a reward distribution request
type RewardDistributionRequest struct {
	RecipientID string                 `json:"recipient_id"`
	Amount      string                 `json:"amount"`
	Reason      string                 `json:"reason"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewNRNIntegration creates a new NRN integration instance
func NewNRNIntegration(knirvRootURL string, nrvSystem *nrv.NRVSystem) *NRNIntegration {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	url := knirvRootURL

	// Handle Unix socket URLs — the URL comes in as "unix://<socket_path>"
	// from resolveKNIRVOracleURL. We need a custom transport to dial the
	// socket, and a dummy base hostname for the HTTP request.
	if strings.HasPrefix(knirvRootURL, "unix://") {
		socketPath := strings.TrimPrefix(knirvRootURL, "unix://")
		url = "http://unix"
		client = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
				},
			},
		}
	}

	return &NRNIntegration{
		knirvRootURL: url,
		httpClient:   client,
		enabled:    true,
		nrvSystem:  nrvSystem,
		rewardPool: big.NewInt(0),
		economicMetrics: &EconomicMetrics{
			TotalNRNRewards: big.NewInt(0),
			TotalNRNBurned:  big.NewInt(0),
			LastUpdated:     time.Now(),
		},
	}
}

// Start initializes the NRN integration
func (ni *NRNIntegration) Start(ctx context.Context) error {
	log.Println("Starting NRN Integration for KNIRVGRAPH...")

	if ni.knirvRootURL == "" {
		log.Println("NRN integration disabled: no compatible KNIRVORACLE endpoint configured")
		ni.enabled = false
		return nil
	}

	// Test connection to KNIRVORACLE
	if err := ni.testConnection(); err != nil {
		log.Printf("Warning: Could not connect to KNIRVORACLE at %s: %v", ni.knirvRootURL, err)
		ni.enabled = false
		return nil // Don't fail startup, just disable economics
	}

	// Start background processes
	go ni.periodicSync(ctx)
	go ni.processEconomicEvents(ctx)

	log.Println("NRN Integration started successfully")
	return nil
}

// testConnection tests the connection to KNIRVORACLE
func (ni *NRNIntegration) testConnection() error {
	healthPaths := []string{"/oracle/v3/health", "/health", "/ping"}
	var lastErr error

	for _, healthPath := range healthPaths {
		url := fmt.Sprintf("%s%s", ni.knirvRootURL, healthPath)
		resp, err := ni.httpClient.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return nil
		}

		lastErr = fmt.Errorf("KNIRVORACLE returned status %d for %s", resp.StatusCode, healthPath)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no health endpoints responded")
	}

	return lastErr
}

// ProcessSkillConfirmation handles skill confirmation for KNIRVCHAIN commitment
// Skills are confirmed on KNIRVGRAPH and committed to KNIRVCHAIN for invocation
func (ni *NRNIntegration) ProcessSkillConfirmation(skillID, nrvID, creatorID string) error {
	if !ni.enabled {
		log.Println("NRN integration disabled, skipping skill confirmation processing")
		return nil
	}

	ni.mutex.Lock()
	defer ni.mutex.Unlock()

	// Create skill confirmation request for KNIRVCHAIN commitment
	request := map[string]interface{}{
		"skill_id":   skillID,
		"nrv_id":     nrvID,
		"creator_id": creatorID,
		"source":     "knirvgraph",
		"timestamp":  time.Now().Unix(),
		"action":     "confirm_for_chain_commitment",
	}

	// Send to KNIRVORACLE for KNIRVCHAIN coordination
	url := fmt.Sprintf("%s/api/economics/skill/confirm", ni.knirvRootURL)
	if err := ni.makeRequest("POST", url, request); err != nil {
		return fmt.Errorf("failed to process skill confirmation: %w", err)
	}

	// Update local metrics
	ni.economicMetrics.TotalSkillsInvoked++ // Rename this to TotalSkillsConfirmed in future
	ni.economicMetrics.LastUpdated = time.Now()

	// Create economic event
	event := NRVEconomicEvent{
		Type:      "skill_confirmation",
		NRVID:     nrvID,
		SkillID:   skillID,
		UserID:    creatorID,
		Amount:    big.NewInt(0), // No direct burning on KNIRVGRAPH
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"confirmed_for_chain": true,
		},
	}

	// Process the event
	go ni.handleEconomicEvent(event)

	log.Printf("Processed skill confirmation: creator=%s, skill=%s, nrv=%s for KNIRVCHAIN commitment", creatorID, skillID, nrvID)
	return nil
}

// DistributeRewards distributes NRN rewards for network participation
func (ni *NRNIntegration) DistributeRewards(recipientID string, amount *big.Int, reason string) error {
	if !ni.enabled {
		return nil
	}

	ni.mutex.Lock()
	defer ni.mutex.Unlock()

	// Create reward distribution request
	request := RewardDistributionRequest{
		RecipientID: recipientID,
		Amount:      amount.String(),
		Reason:      reason,
		Metadata: map[string]interface{}{
			"source":    "knirvgraph",
			"timestamp": time.Now().Unix(),
		},
	}

	// Send to KNIRVORACLE for NRN minting/distribution
	url := fmt.Sprintf("%s/api/economics/rewards/distribute", ni.knirvRootURL)
	if err := ni.makeRequest("POST", url, request); err != nil {
		return fmt.Errorf("failed to distribute rewards: %w", err)
	}

	// Update local metrics
	ni.economicMetrics.TotalNRNRewards.Add(ni.economicMetrics.TotalNRNRewards, amount)
	ni.economicMetrics.LastUpdated = time.Now()

	log.Printf("Distributed %s NRN rewards to %s for: %s", amount.String(), recipientID, reason)
	return nil
}

// GetEconomicMetrics returns current economic metrics
func (ni *NRNIntegration) GetEconomicMetrics() *EconomicMetrics {
	ni.mutex.RLock()
	defer ni.mutex.RUnlock()

	// Create a copy to avoid race conditions
	metrics := &EconomicMetrics{
		TotalNRVsCreated:   ni.economicMetrics.TotalNRVsCreated,
		TotalSkillsInvoked: ni.economicMetrics.TotalSkillsInvoked,
		TotalNRNRewards:    new(big.Int).Set(ni.economicMetrics.TotalNRNRewards),
		TotalNRNBurned:     new(big.Int).Set(ni.economicMetrics.TotalNRNBurned),
		ActiveErrorNodes:   ni.economicMetrics.ActiveErrorNodes,
		ActiveSkillNodes:   ni.economicMetrics.ActiveSkillNodes,
		NetworkEfficiency:  ni.economicMetrics.NetworkEfficiency,
		LastUpdated:        ni.economicMetrics.LastUpdated,
	}

	return metrics
}

// makeRequest makes an HTTP request to KNIRVORACLE
func (ni *NRNIntegration) makeRequest(method, url string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ni.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return nil
}

// periodicSync performs periodic synchronization with KNIRVORACLE
func (ni *NRNIntegration) periodicSync(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ni.syncWithKNIRVRoot()
		}
	}
}

// syncWithKNIRVRoot synchronizes state with KNIRVORACLE
func (ni *NRNIntegration) syncWithKNIRVRoot() {
	if !ni.enabled {
		return
	}

	// Update network statistics
	ni.updateNetworkStatistics()

	// Sync economic metrics
	url := fmt.Sprintf("%s/api/economics/sync", ni.knirvRootURL)
	metrics := ni.GetEconomicMetrics()

	if err := ni.makeRequest("POST", url, metrics); err != nil {
		log.Printf("Failed to sync with KNIRVORACLE: %v", err)
	}

	ni.lastSync = time.Now()
}

// updateNetworkStatistics updates network efficiency and node counts
func (ni *NRNIntegration) updateNetworkStatistics() {
	ni.mutex.Lock()
	defer ni.mutex.Unlock()

	// Get current NRV system statistics
	if ni.nrvSystem != nil {
		// Update active node counts (placeholder - would get from NRV system)
		ni.economicMetrics.ActiveErrorNodes = 0 // Would get from nrvSystem.GetActiveErrorNodes()
		ni.economicMetrics.ActiveSkillNodes = 0 // Would get from nrvSystem.GetActiveSkillNodes()

		// Calculate network efficiency based on resolution success rate
		if ni.economicMetrics.TotalNRVsCreated > 0 {
			ni.economicMetrics.NetworkEfficiency = float64(ni.economicMetrics.TotalSkillsInvoked) / float64(ni.economicMetrics.TotalNRVsCreated)
		}
	}

	ni.economicMetrics.LastUpdated = time.Now()
}

// processEconomicEvents processes economic events in the background
func (ni *NRNIntegration) processEconomicEvents(ctx context.Context) {
	// This would listen for events from the NRV system and process them economically
	// For now, it's a placeholder for future event-driven economics
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Minute):
			// Process any pending economic events
			ni.processPendingEvents()
		}
	}
}

// handleEconomicEvent handles a single economic event
func (ni *NRNIntegration) handleEconomicEvent(event NRVEconomicEvent) {
	log.Printf("Processing economic event: %s for %s", event.Type, event.NRVID)

	// Update metrics based on event type
	ni.mutex.Lock()
	defer ni.mutex.Unlock()

	switch event.Type {
	case "nrv_created":
		ni.economicMetrics.TotalNRVsCreated++
	case "skill_invocation":
		// Already handled in ProcessSkillInvocation
	case "error_resolved":
		// Could trigger reward distribution
	}

	ni.economicMetrics.LastUpdated = time.Now()
}

// processPendingEvents processes any pending economic events
func (ni *NRNIntegration) processPendingEvents() {
	// Placeholder for processing pending events
	// In a full implementation, this would process queued events
}

// IsEnabled returns whether NRN integration is enabled
func (ni *NRNIntegration) IsEnabled() bool {
	ni.mutex.RLock()
	defer ni.mutex.RUnlock()
	return ni.enabled
}

// SetEnabled enables or disables NRN integration
func (ni *NRNIntegration) SetEnabled(enabled bool) {
	ni.mutex.Lock()
	defer ni.mutex.Unlock()
	ni.enabled = enabled
	log.Printf("NRN integration enabled: %v", enabled)
}
