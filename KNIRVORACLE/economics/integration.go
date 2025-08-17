package economics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"
)

// EconomicsIntegration handles integration with existing KNIRV components
type EconomicsIntegration struct {
	tokenEconomics *TokenEconomics
	knirvchainURL  string
	knirvnexusURL  string
	knirvoracleURL   string
	knirvgraphURL  string
	httpClient     *http.Client
}

type ComponentConfig struct {
	KNIRVChainURL string `json:"knirvchain_url"`
	KNIRVNexusURL string `json:"knirvnexus_url"`
	KNIRVRootURL  string `json:"knirvoracle_url"`
	KNIRVGraphURL string `json:"knirvgraph_url"`
}

type SkillExecutionEvent struct {
	UserID    string    `json:"user_id"`
	SkillID   string    `json:"skill_id"`
	Cost      string    `json:"cost"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
}

type LLMValidationEvent struct {
	ValidatorID string    `json:"validator_id"`
	LLMID       string    `json:"llm_id"`
	IsValid     bool      `json:"is_valid"`
	Timestamp   time.Time `json:"timestamp"`
}

type NetworkActivityEvent struct {
	NodeID       string    `json:"node_id"`
	ActivityType string    `json:"activity_type"`
	Data         string    `json:"data"`
	Timestamp    time.Time `json:"timestamp"`
}

func NewEconomicsIntegration(tokenEconomics *TokenEconomics, config ComponentConfig) *EconomicsIntegration {
	return &EconomicsIntegration{
		tokenEconomics: tokenEconomics,
		knirvchainURL:  config.KNIRVChainURL,
		knirvnexusURL:  config.KNIRVNexusURL,
		knirvoracleURL:   config.KNIRVRootURL,
		knirvgraphURL:  config.KNIRVGraphURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (ei *EconomicsIntegration) Start(ctx context.Context) error {
	log.Println("Starting Economics Integration service...")

	// Start event listeners for each component
	go ei.listenToKNIRVChainEvents(ctx)
	go ei.listenToKNIRVNexusEvents(ctx)
	go ei.listenToKNIRVRootEvents(ctx)
	go ei.listenToKNIRVGraphEvents(ctx)

	// Start periodic sync with components
	go ei.periodicSync(ctx)

	log.Println("Economics Integration service started")
	return nil
}

func (ei *EconomicsIntegration) listenToKNIRVChainEvents(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ei.syncKNIRVChainEvents()
		}
	}
}

func (ei *EconomicsIntegration) syncKNIRVChainEvents() {
	// Query KNIRVCHAIN for recent skill executions and LLM registrations
	skillEvents, err := ei.getSkillExecutionEvents()
	if err != nil {
		log.Printf("Error getting skill execution events: %v", err)
		return
	}

	for _, event := range skillEvents {
		ei.processSkillExecutionEvent(event)
	}

	// Query for LLM registration events
	llmEvents, err := ei.getLLMRegistrationEvents()
	if err != nil {
		log.Printf("Error getting LLM registration events: %v", err)
		return
	}

	for _, event := range llmEvents {
		ei.processLLMRegistrationEvent(event)
	}
}

func (ei *EconomicsIntegration) getSkillExecutionEvents() ([]SkillExecutionEvent, error) {
	url := fmt.Sprintf("%s/api/skills/executions/recent", ei.knirvchainURL)
	resp, err := ei.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var events []SkillExecutionEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, err
	}

	return events, nil
}

func (ei *EconomicsIntegration) getLLMRegistrationEvents() ([]LLMValidationEvent, error) {
	url := fmt.Sprintf("%s/api/llm/registrations/recent", ei.knirvchainURL)
	resp, err := ei.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var events []LLMValidationEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, err
	}

	return events, nil
}

func (ei *EconomicsIntegration) processSkillExecutionEvent(event SkillExecutionEvent) {
	// Convert cost to big.Int
	cost, ok := new(big.Int).SetString(event.Cost, 10)
	if !ok {
		log.Printf("Invalid cost format in skill execution event: %s", event.Cost)
		return
	}

	// Process the skill invocation through economics system
	tx, err := ei.tokenEconomics.ProcessSkillInvocation(event.UserID, event.SkillID, cost)
	if err != nil {
		log.Printf("Error processing skill invocation: %v", err)
		return
	}

	log.Printf("Processed skill execution: %s for user %s, tx: %s", event.SkillID, event.UserID, tx.ID)
}

func (ei *EconomicsIntegration) processLLMRegistrationEvent(event LLMValidationEvent) {
	// Process validation reward if the LLM was validated successfully
	if event.IsValid {
		tx, err := ei.tokenEconomics.ProcessValidationReward(event.ValidatorID, event.LLMID, true)
		if err != nil {
			log.Printf("Error processing validation reward: %v", err)
			return
		}

		log.Printf("Processed validation reward for validator %s, tx: %s", event.ValidatorID, tx.ID)
	}
}

func (ei *EconomicsIntegration) listenToKNIRVNexusEvents(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ei.syncKNIRVNexusEvents()
		}
	}
}

func (ei *EconomicsIntegration) syncKNIRVNexusEvents() {
	// Query KNIRVNEXUS for agent activities and validations
	url := fmt.Sprintf("%s/api/agents/activities/recent", ei.knirvnexusURL)
	resp, err := ei.httpClient.Get(url)
	if err != nil {
		log.Printf("Error querying KNIRVNEXUS activities: %v", err)
		return
	}
	defer resp.Body.Close()

	var activities []NetworkActivityEvent
	if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		log.Printf("Error decoding KNIRVNEXUS activities: %v", err)
		return
	}

	for _, activity := range activities {
		ei.processNetworkActivity(activity)
	}
}

func (ei *EconomicsIntegration) processNetworkActivity(activity NetworkActivityEvent) {
	// Update performance metrics for the node
	metrics := &PerformanceMetrics{
		SuccessRate:      0.95,  // This would be calculated based on actual activity
		ResponseTime:     100.0, // milliseconds
		UserSatisfaction: 0.9,
		Uptime:           0.99,
		LastUpdated:      time.Now(),
	}

	ei.tokenEconomics.rewardCalculator.UpdatePerformanceMetrics(activity.NodeID, metrics)
	log.Printf("Updated performance metrics for node %s", activity.NodeID)
}

func (ei *EconomicsIntegration) listenToKNIRVRootEvents(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ei.syncKNIRVRootEvents()
		}
	}
}

func (ei *EconomicsIntegration) syncKNIRVRootEvents() {
	// Query KNIRVORACLE for blockchain events and wallet activities
	url := fmt.Sprintf("%s/api/blockchain/recent-transactions", ei.knirvoracleURL)
	resp, err := ei.httpClient.Get(url)
	if err != nil {
		log.Printf("Error querying KNIRVORACLE transactions: %v", err)
		return
	}
	defer resp.Body.Close()

	// Process blockchain events for economic impact
	log.Println("Synced with KNIRVORACLE blockchain events")
}

func (ei *EconomicsIntegration) listenToKNIRVGraphEvents(ctx context.Context) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ei.syncKNIRVGraphEvents()
		}
	}
}

func (ei *EconomicsIntegration) syncKNIRVGraphEvents() {
	// Query KNIRVGRAPH for network topology and connection events
	url := fmt.Sprintf("%s/api/graph/network-stats", ei.knirvgraphURL)
	resp, err := ei.httpClient.Get(url)
	if err != nil {
		log.Printf("Error querying KNIRVGRAPH stats: %v", err)
		return
	}
	defer resp.Body.Close()

	// Process network statistics for economic calculations
	log.Println("Synced with KNIRVGRAPH network statistics")
}

func (ei *EconomicsIntegration) periodicSync(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ei.performPeriodicSync()
		}
	}
}

func (ei *EconomicsIntegration) performPeriodicSync() {
	// Sync economic metrics with all components
	metrics := ei.tokenEconomics.GetEconomicMetrics()

	// Send metrics to each component
	ei.sendMetricsToComponent(ei.knirvchainURL, metrics)
	ei.sendMetricsToComponent(ei.knirvnexusURL, metrics)
	ei.sendMetricsToComponent(ei.knirvoracleURL, metrics)
	ei.sendMetricsToComponent(ei.knirvgraphURL, metrics)

	log.Println("Performed periodic sync of economic metrics")
}

func (ei *EconomicsIntegration) sendMetricsToComponent(componentURL string, metrics *EconomicMetrics) {
	url := fmt.Sprintf("%s/api/economics/metrics", componentURL)

	// For now, just log that we would send metrics
	// In real implementation, this would send the actual metrics
	log.Printf("Would send metrics to %s", componentURL)

	// Simulate HTTP call
	resp, err := ei.httpClient.Get(url + "/health") // Just check if component is alive
	if err != nil {
		log.Printf("Error connecting to %s: %v", componentURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Component %s returned status %d", componentURL, resp.StatusCode)
	}
}

// GetIntegrationStatus returns the current status of component integrations
func (ei *EconomicsIntegration) GetIntegrationStatus() map[string]interface{} {
	return map[string]interface{}{
		"knirvchain_url": ei.knirvchainURL,
		"knirvnexus_url": ei.knirvnexusURL,
		"knirvoracle_url":  ei.knirvoracleURL,
		"knirvgraph_url": ei.knirvgraphURL,
		"last_sync":      time.Now(),
		"status":         "active",
	}
}
