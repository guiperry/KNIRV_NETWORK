package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"KNIRVENGINE/desktop-client/agent"
	"KNIRVENGINE/desktop-client/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKNIRVOracleIntegration tests the complete KNIRVORACLE integration
func TestKNIRVOracleIntegration(t *testing.T) {
	// Skip if KNIRVORACLE_URL is not set (for CI/CD environments)
	knirvoracleURL := os.Getenv("KNIRVORACLE_URL")
	if knirvoracleURL == "" {
		t.Skip("KNIRVORACLE_URL not set, skipping integration tests")
	}

	// Initialize KNIRVORACLE service
	config := services.KNIRVOracleConfig{
		BaseURL: knirvoracleURL,
		APIKey:  os.Getenv("KNIRVORACLE_API_KEY"),
		Timeout: 30,
	}
	knirvoracleService := services.NewKNIRVOracleService(config)

	t.Run("HealthCheck", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := knirvoracleService.HealthCheck(ctx)
		assert.NoError(t, err, "KNIRVORACLE health check should pass")
	})

	t.Run("AgentMinting", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create test agent configuration
		agentConfig := agent.AgentConfig{
			AgentID:     "test-agent-" + time.Now().Format("20060102150405"),
			Name:        "Test Agent",
			Description: "Test agent for integration testing",
			AgentType:   "assistant",
			Model:       "gpt-4",
		}

		// Test agent minting
		mintRequest := services.AgentMintRequest{
			AgentID:     agentConfig.AgentID,
			Name:        agentConfig.Name,
			Description: agentConfig.Description,
			Owner:       "test-owner",
			Metadata: map[string]interface{}{
				"agent_type": agentConfig.AgentType,
				"model":      agentConfig.Model,
				"test":       true,
			},
		}

		response, err := knirvoracleService.MintAgent(ctx, mintRequest)
		require.NoError(t, err, "Agent minting should not fail")
		assert.True(t, response.Success, "Agent minting should succeed")
		assert.NotEmpty(t, response.TransactionID, "Transaction ID should be provided")
		assert.NotEmpty(t, response.AgentNFTID, "Agent NFT ID should be provided")
	})

	t.Run("CapabilityRegistration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Test capability registration
		capabilityRequest := services.CapabilityRegistrationRequest{
			Name:        "Test Capability " + time.Now().Format("15:04:05"),
			Type:        "mcp_capability",
			Description: "Test capability for integration testing",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{
						"type":        "string",
						"description": "Input data for the capability",
					},
				},
				"required": []string{"input"},
			},
			Owner:     "test-owner",
			GasFeeNRN: 1000,
		}

		response, err := knirvoracleService.RegisterCapability(ctx, capabilityRequest)
		require.NoError(t, err, "Capability registration should not fail")
		assert.True(t, response.Success, "Capability registration should succeed")
		assert.NotEmpty(t, response.CapabilityID, "Capability ID should be provided")
		assert.NotEmpty(t, response.TxHash, "Transaction hash should be provided")
	})

	t.Run("WalletOperations", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Test wallet address (use a test address)
		testAddress := "0x742d35Cc6aa34567890abcdef1234567890abcdef"

		// Test faucet request
		faucetRequest := services.FaucetRequest{
			Address: testAddress,
			Amount:  "100",
			Reason:  "Integration test",
		}

		faucetResponse, err := knirvoracleService.RequestFaucet(ctx, faucetRequest)
		if err != nil {
			t.Logf("Faucet request failed (may be expected): %v", err)
		} else {
			assert.True(t, faucetResponse.Success, "Faucet request should succeed")
			assert.NotEmpty(t, faucetResponse.RequestID, "Request ID should be provided")
		}

		// Test wallet balance retrieval
		balanceResponse, err := knirvoracleService.GetWalletBalance(ctx, testAddress)
		if err != nil {
			t.Logf("Balance retrieval failed (may be expected for test address): %v", err)
		} else {
			assert.True(t, balanceResponse.Success, "Balance retrieval should succeed")
			assert.Equal(t, testAddress, balanceResponse.Address, "Address should match")
		}
	})

	t.Run("TreasuryStatus", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		treasuryStatus, err := knirvoracleService.GetTreasuryStatus(ctx)
		if err != nil {
			t.Logf("Treasury status retrieval failed (may be expected): %v", err)
		} else {
			assert.NotNil(t, treasuryStatus, "Treasury status should be returned")
		}
	})
}

// TestAgentBuilderKNIRVOracleIntegration tests the agent builder integration with KNIRVORACLE
func TestAgentBuilderKNIRVOracleIntegration(t *testing.T) {
	// Skip if KNIRVORACLE_URL is not set
	if os.Getenv("KNIRVORACLE_URL") == "" {
		t.Skip("KNIRVORACLE_URL not set, skipping integration tests")
	}

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Initialize agent builder
	builder, err := agent.NewAgentBuilderWithStorage(nil, "../../agent/templates", tempDir)
	require.NoError(t, err, "Agent builder initialization should not fail")

	t.Run("BuildAgentWithMinting", func(t *testing.T) {
		// Create test agent configuration
		agentConfig := agent.AgentConfig{
			Name:        "Integration Test Agent",
			Description: "Agent created during integration testing",
			AgentType:   "assistant",
			Model:       "gpt-4",
			ExtraParams: map[string]interface{}{
				"test": true,
			},
		}

		// Build the agent (this should trigger minting)
		agentID, err := builder.BuildAgent(agentConfig)
		require.NoError(t, err, "Agent building should not fail")
		assert.NotEmpty(t, agentID, "Agent ID should be provided")

		// Wait a bit for the minting process to complete
		time.Sleep(5 * time.Second)

		// Verify the agent was created
		retrievedAgent, err := builder.GetAgent(agentID)
		require.NoError(t, err, "Agent retrieval should not fail")
		assert.Equal(t, agentConfig.Name, retrievedAgent.Name, "Agent name should match")
		assert.Equal(t, agentConfig.Description, retrievedAgent.Description, "Agent description should match")
	})
}

// TestKNIRVOracleServiceConfiguration tests service configuration
func TestKNIRVOracleServiceConfiguration(t *testing.T) {
	t.Run("DefaultConfiguration", func(t *testing.T) {
		config := services.KNIRVOracleConfig{
			BaseURL: "http://localhost:8080",
			Timeout: 30,
		}

		service := services.NewKNIRVOracleService(config)
		assert.NotNil(t, service, "Service should be created")
	})

	t.Run("ConfigurationWithAPIKey", func(t *testing.T) {
		config := services.KNIRVOracleConfig{
			BaseURL: "http://localhost:8080",
			APIKey:  "test-api-key",
			Timeout: 60,
		}

		service := services.NewKNIRVOracleService(config)
		assert.NotNil(t, service, "Service should be created with API key")
	})
}

// TestKNIRVOracleErrorHandling tests error handling scenarios
func TestKNIRVOracleErrorHandling(t *testing.T) {
	// Create service with invalid URL
	config := services.KNIRVOracleConfig{
		BaseURL: "http://invalid-url:9999",
		Timeout: 5,
	}
	service := services.NewKNIRVOracleService(config)

	t.Run("HealthCheckFailure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := service.HealthCheck(ctx)
		assert.Error(t, err, "Health check should fail with invalid URL")
	})

	t.Run("AgentMintingFailure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		mintRequest := services.AgentMintRequest{
			AgentID:     "test-agent",
			Name:        "Test Agent",
			Description: "Test agent",
			Owner:       "test-owner",
			Metadata:    map[string]interface{}{},
		}

		_, err := service.MintAgent(ctx, mintRequest)
		assert.Error(t, err, "Agent minting should fail with invalid URL")
	})
}

// BenchmarkKNIRVOracleOperations benchmarks KNIRVORACLE operations
func BenchmarkKNIRVOracleOperations(b *testing.B) {
	// Skip if KNIRVORACLE_URL is not set
	if os.Getenv("KNIRVORACLE_URL") == "" {
		b.Skip("KNIRVORACLE_URL not set, skipping benchmarks")
	}

	config := services.KNIRVOracleConfig{
		BaseURL: os.Getenv("KNIRVORACLE_URL"),
		APIKey:  os.Getenv("KNIRVORACLE_API_KEY"),
		Timeout: 30,
	}
	service := services.NewKNIRVOracleService(config)

	b.Run("HealthCheck", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			service.HealthCheck(ctx)
			cancel()
		}
	})

	b.Run("GetTreasuryStatus", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			service.GetTreasuryStatus(ctx)
			cancel()
		}
	})
}
