package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBasicFramework tests the basic test framework without requiring services
func TestBasicFramework(t *testing.T) {
	t.Run("TestSuiteCreation", func(t *testing.T) {
		suite := NewIntegrationTestSuite()

		assert.NotNil(t, suite)
		assert.Equal(t, "http://localhost:8080", suite.knirvchainURL)
		assert.Equal(t, "http://localhost:8081", suite.knirvgraphURL)
		assert.Equal(t, "http://localhost:3000", suite.knirvnexusFrontendURL)
		assert.Equal(t, "http://localhost:8080", suite.knirvnexusAPIGatewayURL)
		assert.Equal(t, "http://localhost:8083", suite.knirvwalletURL)
		assert.Equal(t, "http://localhost:8084", suite.knirvshellURL)
		assert.Equal(t, "http://localhost:8085", suite.knirvroterURL)
		assert.Equal(t, "http://localhost:8086", suite.knirvGatewayAPIURL) // Updated
		assert.Equal(t, "http://localhost:26657", suite.knirvOracledRPCURL) // New
		assert.Equal(t, "https://rpc.xion-testnet-1.burnt.com:443", suite.xionRPC)
	})

	t.Run("ConfigurationValidation", func(t *testing.T) {
		suite := NewIntegrationTestSuite()

		// Test that all URLs are properly formatted
		urls := []string{
			suite.knirvchainURL,
			suite.knirvgraphURL,
			suite.knirvnexusAPIGatewayURL,
			suite.knirvwalletURL,
			suite.knirvshellURL,
			suite.knirvroterURL,
			suite.knirvGatewayAPIURL,   // Updated
			suite.knirvOracledRPCURL, // New
		}

		for _, url := range urls {
			assert.Contains(t, url, "http://localhost:")
		}

		assert.Contains(t, suite.xionRPC, "https://")
	})
}

// TestFrameworkComponents tests individual framework components
func TestFrameworkComponents(t *testing.T) {
	t.Run("TestWalletStructure", func(t *testing.T) {
		wallet := &TestWallet{ // Uses TestWallet from test_utils.go
			Address:  "test_address_123",
			Mnemonic: "test mnemonic phrase",
		}

		assert.NotEmpty(t, wallet.Address)
		assert.NotEmpty(t, wallet.Mnemonic)
	})

	t.Run("TestRequestDataStructure", func(t *testing.T) {
		// Test LLM data structure
		llmData := map[string]interface{}{
			"name":         "TestLLM",
			"version":      "1.0.0",
			"capabilities": []string{"text-generation"},
		}

		assert.Equal(t, "TestLLM", llmData["name"])
		assert.Equal(t, "1.0.0", llmData["version"])
		assert.IsType(t, []string{}, llmData["capabilities"])

		// Test error data structure
		errorData := map[string]interface{}{
			"error_type":  "test_error",
			"description": "Test error description",
			"severity":    3,
		}

		assert.Equal(t, "test_error", errorData["error_type"])
		assert.Equal(t, "Test error description", errorData["description"])
		assert.Equal(t, 3, errorData["severity"])
	})
}

// TestTestUtilities tests utility functions
func TestTestUtilities(t *testing.T) {
	t.Run("TestSuiteInitialization", func(t *testing.T) {
		suite := NewIntegrationTestSuite() // Uses NewIntegrationTestSuite from test_utils.go

		// Verify all components are initialized
		assert.NotEmpty(t, suite.knirvchainURL)
		assert.NotEmpty(t, suite.knirvgraphURL)
		assert.NotEmpty(t, suite.knirvnexusFrontendURL)
		assert.NotEmpty(t, suite.knirvnexusAPIGatewayURL)
		assert.NotEmpty(t, suite.knirvwalletURL)
		assert.NotEmpty(t, suite.knirvshellURL)
		assert.NotEmpty(t, suite.knirvroterURL)
		assert.NotEmpty(t, suite.knirvGatewayAPIURL)     // Updated
		assert.NotEmpty(t, suite.knirvOracledRPCURL) // New
		assert.NotEmpty(t, suite.xionRPC)
	})

	t.Run("TestDataValidation", func(t *testing.T) {
		// Test that test data structures are valid
		testData := map[string]interface{}{
			"string_field": "test_value",
			"int_field":    42,
			"bool_field":   true,
			"array_field":  []string{"item1", "item2"},
		}

		assert.IsType(t, "", testData["string_field"])
		assert.IsType(t, 0, testData["int_field"])
		assert.IsType(t, true, testData["bool_field"])
		assert.IsType(t, []string{}, testData["array_field"])
	})
}
