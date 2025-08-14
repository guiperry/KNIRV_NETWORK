package crossservice

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CrossServiceIntegrationSuite manages cross-service integration testing
type CrossServiceIntegrationSuite struct {
	Services    map[string]*ServiceClient
	Context     context.Context
	TestData    *IntegrationTestData
	EventBus    *EventBus
	mu          sync.RWMutex
}

// ServiceClient represents a client for a specific service
type ServiceClient struct {
	Name     string
	BaseURL  string
	Healthy  bool
	LastPing time.Time
}

// IntegrationTestData holds test data for integration scenarios
type IntegrationTestData struct {
	Users       []TestUser       `json:"users"`
	Skills      []TestSkill      `json:"skills"`
	Agents      []TestAgent      `json:"agents"`
	Transactions []TestTransaction `json:"transactions"`
}

// TestUser represents a user in integration tests
type TestUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Wallet   string `json:"wallet"`
	Balance  float64 `json:"balance"`
}

// TestSkill represents a skill in integration tests
type TestSkill struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Creator     string  `json:"creator"`
	Price       float64 `json:"price"`
	Status      string  `json:"status"`
	BlockchainTx string `json:"blockchain_tx"`
	GraphNode   string  `json:"graph_node"`
}

// TestAgent represents an agent in integration tests
type TestAgent struct {
	ID           string   `json:"id"`
	Owner        string   `json:"owner"`
	Type         string   `json:"type"`
	Capabilities []string `json:"capabilities"`
	Status       string   `json:"status"`
}

// TestTransaction represents a transaction in integration tests
type TestTransaction struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// EventBus manages cross-service event communication
type EventBus struct {
	events   []ServiceEvent
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

// ServiceEvent represents an event between services
type ServiceEvent struct {
	ID        string                 `json:"id"`
	Source    string                 `json:"source"`
	Target    string                 `json:"target"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// EventHandler handles service events
type EventHandler func(event ServiceEvent) error

// NewCrossServiceIntegrationSuite creates a new integration test suite
func NewCrossServiceIntegrationSuite() *CrossServiceIntegrationSuite {
	services := map[string]*ServiceClient{
		"knirv-root":    {Name: "knirv-root", BaseURL: "http://localhost:1317"},
		"knirvchain":    {Name: "knirvchain", BaseURL: "http://localhost:8090"},
		"knirvgraph":    {Name: "knirvgraph", BaseURL: "http://localhost:8082"},
		"knirv-nexus":   {Name: "knirv-nexus", BaseURL: "http://localhost:8084"},
		"knirv-router":  {Name: "knirv-router", BaseURL: "http://localhost:5001"},
		"knirv-gateway": {Name: "knirv-gateway", BaseURL: "http://localhost:8087"},
	}

	return &CrossServiceIntegrationSuite{
		Services: services,
		Context:  context.Background(),
		EventBus: NewEventBus(),
		TestData: &IntegrationTestData{},
	}
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		events:   make([]ServiceEvent, 0),
		handlers: make(map[string][]EventHandler),
	}
}

// TestCompleteDataFlow tests data flow across all services
func TestCompleteDataFlow(t *testing.T) {
	suite := NewCrossServiceIntegrationSuite()
	require.NoError(t, suite.InitializeServices())

	t.Run("Skill Registration Flow", func(t *testing.T) {
		// Test complete skill registration across services
		skillData := map[string]interface{}{
			"name":        "Advanced ML Model",
			"description": "Machine learning model for prediction",
			"creator":     "developer_001",
			"price":       100.0,
			"category":    "machine_learning",
		}

		// Step 1: Register skill on KNIRVCHAIN
		chainResponse, err := suite.callService("knirvchain", "POST", "/skills/register", skillData)
		require.NoError(t, err)

		var chainResult struct {
			Success   bool   `json:"success"`
			SkillID   string `json:"skill_id"`
			TxHash    string `json:"tx_hash"`
			Status    string `json:"status"`
		}
		err = json.Unmarshal(chainResponse, &chainResult)
		require.NoError(t, err)
		assert.True(t, chainResult.Success)
		assert.NotEmpty(t, chainResult.SkillID)

		// Step 2: Update knowledge graph in KNIRVGRAPH
		graphData := map[string]interface{}{
			"skill_id":      chainResult.SkillID,
			"relationships": []string{"machine_learning", "prediction", "ai"},
			"metadata":      skillData,
		}

		graphResponse, err := suite.callService("knirvgraph", "POST", "/graph/update", graphData)
		require.NoError(t, err)

		var graphResult struct {
			Success bool   `json:"success"`
			NodeID  string `json:"node_id"`
			Edges   int    `json:"edges_created"`
		}
		err = json.Unmarshal(graphResponse, &graphResult)
		require.NoError(t, err)
		assert.True(t, graphResult.Success)
		assert.NotEmpty(t, graphResult.NodeID)

		// Step 3: Submit for validation in KNIRV-NEXUS
		validationData := map[string]interface{}{
			"skill_id":   chainResult.SkillID,
			"tx_hash":    chainResult.TxHash,
			"graph_node": graphResult.NodeID,
		}

		nexusResponse, err := suite.callService("knirv-nexus", "POST", "/validation/submit", validationData)
		require.NoError(t, err)

		var nexusResult struct {
			Success      bool   `json:"success"`
			ValidationID string `json:"validation_id"`
			Status       string `json:"status"`
		}
		err = json.Unmarshal(nexusResponse, &nexusResult)
		require.NoError(t, err)
		assert.True(t, nexusResult.Success)
		assert.Equal(t, "pending", nexusResult.Status)

		// Step 4: Record transaction in KNIRV-ROOT
		rootData := map[string]interface{}{
			"type":        "skill_registration",
			"skill_id":    chainResult.SkillID,
			"creator":     "developer_001",
			"amount":      100.0,
			"tx_hash":     chainResult.TxHash,
		}

		rootResponse, err := suite.callService("knirv-root", "POST", "/transactions/record", rootData)
		require.NoError(t, err)

		var rootResult struct {
			Success       bool   `json:"success"`
			TransactionID string `json:"transaction_id"`
			BlockHeight   int64  `json:"block_height"`
		}
		err = json.Unmarshal(rootResponse, &rootResult)
		require.NoError(t, err)
		assert.True(t, rootResult.Success)
		assert.Greater(t, rootResult.BlockHeight, int64(0))

		// Step 5: Verify data consistency across services
		err = suite.verifySkillConsistency(chainResult.SkillID)
		require.NoError(t, err)
	})

	t.Run("Agent Collaboration Flow", func(t *testing.T) {
		// Test agent collaboration across services
		collaborationData := map[string]interface{}{
			"primary_agent":   "agent_001",
			"collaborators":   []string{"agent_002", "agent_003"},
			"task_type":       "complex_analysis",
			"coordination_mode": "hierarchical",
		}

		// Step 1: Initialize collaboration through KNIRV-ROUTER
		routerResponse, err := suite.callService("knirv-router", "POST", "/collaboration/init", collaborationData)
		require.NoError(t, err)

		var routerResult struct {
			Success   bool   `json:"success"`
			SessionID string `json:"session_id"`
			Status    string `json:"status"`
		}
		err = json.Unmarshal(routerResponse, &routerResult)
		require.NoError(t, err)
		assert.True(t, routerResult.Success)
		assert.NotEmpty(t, routerResult.SessionID)

		// Step 2: Share knowledge through KNIRVGRAPH
		knowledgeData := map[string]interface{}{
			"session_id": routerResult.SessionID,
			"knowledge_type": "collaborative_context",
			"participants": collaborationData["collaborators"],
		}

		graphResponse, err := suite.callService("knirvgraph", "POST", "/knowledge/share", knowledgeData)
		require.NoError(t, err)

		var graphResult struct {
			Success     bool   `json:"success"`
			ContextID   string `json:"context_id"`
			SharedNodes int    `json:"shared_nodes"`
		}
		err = json.Unmarshal(graphResponse, &graphResult)
		require.NoError(t, err)
		assert.True(t, graphResult.Success)
		assert.Greater(t, graphResult.SharedNodes, 0)

		// Step 3: Execute collaborative task through KNIRV-NEXUS
		taskData := map[string]interface{}{
			"session_id":  routerResult.SessionID,
			"context_id":  graphResult.ContextID,
			"task_type":   "complex_analysis",
			"participants": collaborationData["collaborators"],
		}

		nexusResponse, err := suite.callService("knirv-nexus", "POST", "/tasks/collaborative", taskData)
		require.NoError(t, err)

		var nexusResult struct {
			Success bool   `json:"success"`
			TaskID  string `json:"task_id"`
			Status  string `json:"status"`
		}
		err = json.Unmarshal(nexusResponse, &nexusResult)
		require.NoError(t, err)
		assert.True(t, nexusResult.Success)
		assert.Equal(t, "executing", nexusResult.Status)

		// Step 4: Record collaboration rewards in KNIRV-ROOT
		rewardData := map[string]interface{}{
			"session_id":   routerResult.SessionID,
			"task_id":      nexusResult.TaskID,
			"participants": collaborationData["collaborators"],
			"reward_type":  "collaboration",
		}

		rootResponse, err := suite.callService("knirv-root", "POST", "/rewards/distribute", rewardData)
		require.NoError(t, err)

		var rootResult struct {
			Success      bool                   `json:"success"`
			TotalReward  float64                `json:"total_reward"`
			Distribution map[string]float64     `json:"distribution"`
		}
		err = json.Unmarshal(rootResponse, &rootResult)
		require.NoError(t, err)
		assert.True(t, rootResult.Success)
		assert.Greater(t, rootResult.TotalReward, 0.0)
	})

	t.Run("Economic Transaction Flow", func(t *testing.T) {
		// Test economic transactions across services
		transactionData := map[string]interface{}{
			"from":     "user_001",
			"to":       "developer_001",
			"amount":   50.0,
			"type":     "skill_purchase",
			"skill_id": "skill_001",
		}

		// Step 1: Initiate transaction through KNIRV-GATEWAY
		gatewayResponse, err := suite.callService("knirv-gateway", "POST", "/transactions/initiate", transactionData)
		require.NoError(t, err)

		var gatewayResult struct {
			Success       bool   `json:"success"`
			TransactionID string `json:"transaction_id"`
			Status        string `json:"status"`
		}
		err = json.Unmarshal(gatewayResponse, &gatewayResult)
		require.NoError(t, err)
		assert.True(t, gatewayResult.Success)
		assert.NotEmpty(t, gatewayResult.TransactionID)

		// Step 2: Process payment through KNIRV-ROOT
		paymentData := map[string]interface{}{
			"transaction_id": gatewayResult.TransactionID,
			"from":          transactionData["from"],
			"to":            transactionData["to"],
			"amount":        transactionData["amount"],
		}

		rootResponse, err := suite.callService("knirv-root", "POST", "/payments/process", paymentData)
		require.NoError(t, err)

		var rootResult struct {
			Success   bool   `json:"success"`
			PaymentID string `json:"payment_id"`
			Status    string `json:"status"`
		}
		err = json.Unmarshal(rootResponse, &rootResult)
		require.NoError(t, err)
		assert.True(t, rootResult.Success)
		assert.Equal(t, "completed", rootResult.Status)

		// Step 3: Execute skill through KNIRVCHAIN
		executionData := map[string]interface{}{
			"transaction_id": gatewayResult.TransactionID,
			"skill_id":       transactionData["skill_id"],
			"user":           transactionData["from"],
			"payment_id":     rootResult.PaymentID,
		}

		chainResponse, err := suite.callService("knirvchain", "POST", "/skills/execute", executionData)
		require.NoError(t, err)

		var chainResult struct {
			Success     bool        `json:"success"`
			ExecutionID string      `json:"execution_id"`
			Result      interface{} `json:"result"`
			Status      string      `json:"status"`
		}
		err = json.Unmarshal(chainResponse, &chainResult)
		require.NoError(t, err)
		assert.True(t, chainResult.Success)
		assert.NotEmpty(t, chainResult.ExecutionID)

		// Step 4: Update usage statistics in KNIRVGRAPH
		usageData := map[string]interface{}{
			"skill_id":     transactionData["skill_id"],
			"execution_id": chainResult.ExecutionID,
			"user":         transactionData["from"],
			"success":      true,
		}

		graphResponse, err := suite.callService("knirvgraph", "POST", "/usage/update", usageData)
		require.NoError(t, err)

		var graphResult struct {
			Success     bool `json:"success"`
			UsageCount  int  `json:"usage_count"`
			UpdatedEdges int  `json:"updated_edges"`
		}
		err = json.Unmarshal(graphResponse, &graphResult)
		require.NoError(t, err)
		assert.True(t, graphResult.Success)
		assert.Greater(t, graphResult.UsageCount, 0)
	})
}

// TestServiceCommunication tests inter-service communication
func TestServiceCommunication(t *testing.T) {
	suite := NewCrossServiceIntegrationSuite()
	require.NoError(t, suite.InitializeServices())

	t.Run("Event Propagation", func(t *testing.T) {
		// Test event propagation between services
		event := ServiceEvent{
			ID:     "event_001",
			Source: "knirvchain",
			Target: "knirvgraph",
			Type:   "skill_created",
			Data: map[string]interface{}{
				"skill_id": "skill_001",
				"creator":  "developer_001",
			},
			Timestamp: time.Now(),
		}

		err := suite.EventBus.PublishEvent(event)
		require.NoError(t, err)

		// Wait for event processing
		time.Sleep(2 * time.Second)

		// Verify event was processed
		events := suite.EventBus.GetEvents()
		assert.Len(t, events, 1)
		assert.Equal(t, "skill_created", events[0].Type)
	})

	t.Run("Service Health Monitoring", func(t *testing.T) {
		// Test health monitoring across services
		for serviceName, client := range suite.Services {
			t.Run(fmt.Sprintf("Health_%s", serviceName), func(t *testing.T) {
				response, err := suite.callService(serviceName, "GET", "/health", nil)
				require.NoError(t, err)

				var healthResult struct {
					Status    string `json:"status"`
					Timestamp string `json:"timestamp"`
					Version   string `json:"version"`
				}
				err = json.Unmarshal(response, &healthResult)
				require.NoError(t, err)
				assert.Equal(t, "healthy", healthResult.Status)

				client.Healthy = true
				client.LastPing = time.Now()
			})
		}
	})

	t.Run("Data Consistency Validation", func(t *testing.T) {
		// Test data consistency across services
		skillID := "skill_consistency_test"

		// Create skill data in multiple services
		services := []string{"knirvchain", "knirvgraph", "knirv-nexus"}
		
		for _, serviceName := range services {
			skillData := map[string]interface{}{
				"skill_id": skillID,
				"name":     "Consistency Test Skill",
				"creator":  "test_user",
			}

			_, err := suite.callService(serviceName, "POST", "/test/create_skill", skillData)
			require.NoError(t, err)
		}

		// Verify consistency across services
		err := suite.verifySkillConsistency(skillID)
		require.NoError(t, err)
	})
}

// Helper methods

// InitializeServices initializes all service clients
func (suite *CrossServiceIntegrationSuite) InitializeServices() error {
	suite.mu.Lock()
	defer suite.mu.Unlock()

	for _, client := range suite.Services {
		// Ping service to verify availability
		_, err := suite.callService(client.Name, "GET", "/health", nil)
		if err != nil {
			return fmt.Errorf("service %s not available: %w", client.Name, err)
		}
		client.Healthy = true
		client.LastPing = time.Now()
	}

	return nil
}

// callService makes a call to a specific service
func (suite *CrossServiceIntegrationSuite) callService(serviceName, method, endpoint string, data interface{}) ([]byte, error) {
	// Implementation would make actual HTTP request
	// For now, return mock response based on service and endpoint
	
	mockResponse := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Mock response from %s %s", serviceName, endpoint),
	}

	// Add service-specific mock data
	switch serviceName {
	case "knirvchain":
		if endpoint == "/skills/register" {
			mockResponse["skill_id"] = "skill_001"
			mockResponse["tx_hash"] = "0x123456789"
			mockResponse["status"] = "registered"
		}
	case "knirvgraph":
		if endpoint == "/graph/update" {
			mockResponse["node_id"] = "node_001"
			mockResponse["edges_created"] = 3
		}
	case "knirv-nexus":
		if endpoint == "/validation/submit" {
			mockResponse["validation_id"] = "val_001"
			mockResponse["status"] = "pending"
		}
	case "knirv-root":
		if endpoint == "/transactions/record" {
			mockResponse["transaction_id"] = "tx_001"
			mockResponse["block_height"] = 12345
		}
	}

	return json.Marshal(mockResponse)
}

// verifySkillConsistency verifies skill data consistency across services
func (suite *CrossServiceIntegrationSuite) verifySkillConsistency(skillID string) error {
	// Get skill data from each service
	services := []string{"knirvchain", "knirvgraph", "knirv-nexus"}
	skillData := make(map[string]interface{})

	for _, serviceName := range services {
		response, err := suite.callService(serviceName, "GET", fmt.Sprintf("/skills/%s", skillID), nil)
		if err != nil {
			return fmt.Errorf("failed to get skill from %s: %w", serviceName, err)
		}

		var serviceData map[string]interface{}
		err = json.Unmarshal(response, &serviceData)
		if err != nil {
			return fmt.Errorf("failed to parse response from %s: %w", serviceName, err)
		}

		skillData[serviceName] = serviceData
	}

	// Verify consistency (simplified check)
	// In a real implementation, this would check specific fields
	for serviceName, data := range skillData {
		if serviceData, ok := data.(map[string]interface{}); ok {
			if !serviceData["success"].(bool) {
				return fmt.Errorf("skill not found in service %s", serviceName)
			}
		}
	}

	return nil
}

// PublishEvent publishes an event to the event bus
func (eb *EventBus) PublishEvent(event ServiceEvent) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.events = append(eb.events, event)

	// Trigger handlers
	if handlers, exists := eb.handlers[event.Type]; exists {
		for _, handler := range handlers {
			go handler(event)
		}
	}

	return nil
}

// GetEvents returns all events from the event bus
func (eb *EventBus) GetEvents() []ServiceEvent {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	return append([]ServiceEvent{}, eb.events...)
}

// RegisterHandler registers an event handler
func (eb *EventBus) RegisterHandler(eventType string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.handlers[eventType] == nil {
		eb.handlers[eventType] = make([]EventHandler, 0)
	}
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}
