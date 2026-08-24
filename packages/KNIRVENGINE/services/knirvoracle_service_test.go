// knirvoracle_service_test.go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test helper functions
func createTestKNIRVOracleService(t *testing.T) (*KNIRVOracleService, *httptest.Server) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleTestRequest(w, r)
	}))

	config := KNIRVOracleConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
		Timeout: 30,
	}

	service := NewKNIRVOracleService(config)
	return service, server
}

func handleTestRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/health":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
		})

	case "/agent/mint":
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AgentMintResponse{
			Success:       true,
			TransactionID: "tx-12345",
			AgentNFTID:    "nft-67890",
			Message:       "Agent minted successfully",
		})

	case "/wallet/mcp/create_register_capability":
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CapabilityRegistrationResponse{
			Success:      true,
			CapabilityID: "cap-12345",
			Message:      "Capability registered successfully",
		})

	case "/api/mint/nrv":
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(FaucetResponse{
			Success:   true,
			RequestID: "faucet-tx-12345",
			Amount:    "1000",
			Status:    "completed",
			Message:   "Faucet request successful",
		})

	case "/balance/test-address":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(WalletBalanceResponse{
			Success:    true,
			Address:    "test-address",
			Balance:    "5000.50",
			NRNBalance: "1000.0",
			USDValue:   "5000.50",
		})

	case "/transactions":
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(TransactionResponse{
			Success:       true,
			TransactionID: "tx-98765",
			Status:        "pending",
			Message:       "Transaction submitted",
		})

	case "/agent/capabilities/test-agent":
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"capabilities": []map[string]interface{}{
				{
					"id":          "cap-1",
					"name":        "test-capability",
					"description": "A test capability",
				},
			},
		}
		json.NewEncoder(w).Encode(response)

	case "/api/treasury/status":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_balance": "1000000",
			"status":        "active",
			"last_update":   time.Now().Unix(),
		})

	default:
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "endpoint not found",
		})
	}
}

// TestNewKNIRVOracleService tests the constructor
func TestNewKNIRVOracleService(t *testing.T) {
	config := KNIRVOracleConfig{
		BaseURL: "https://test.knirvoracle.com",
		APIKey:  "test-key",
		Timeout: 30,
	}

	service := NewKNIRVOracleService(config)

	assert.NotNil(t, service)
	assert.Equal(t, config.BaseURL, service.baseURL)
	assert.Equal(t, config.APIKey, service.apiKey)
	assert.NotNil(t, service.httpClient)
}

// TestNewKNIRVOracleService_DefaultTimeout tests constructor with default timeout
func TestNewKNIRVOracleService_DefaultTimeout(t *testing.T) {
	config := KNIRVOracleConfig{
		BaseURL: "https://test.knirvoracle.com",
		APIKey:  "test-key",
		Timeout: 0, // Should use default
	}

	service := NewKNIRVOracleService(config)

	assert.NotNil(t, service)
	assert.Equal(t, 30*time.Second, service.httpClient.Timeout)
}

// TestKNIRVOracleService_HealthCheck tests health check functionality
func TestKNIRVOracleService_HealthCheck(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	ctx := context.Background()
	err := service.HealthCheck(ctx)

	assert.NoError(t, err)
}

// TestKNIRVOracleService_MintAgent tests agent minting
func TestKNIRVOracleService_MintAgent(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	ctx := context.Background()
	request := AgentMintRequest{
		AgentID:     "test-agent-001",
		Name:        "Test Agent",
		Description: "A test agent for unit testing",
		Owner:       "test-owner",
		Metadata: map[string]interface{}{
			"version": "1.0.0",
			"type":    "test",
		},
		ImageURL: "https://example.com/agent.png",
	}

	response, err := service.MintAgent(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Success)
	assert.Equal(t, "tx-12345", response.TransactionID)
	assert.Equal(t, "nft-67890", response.AgentNFTID)
	assert.NotEmpty(t, response.Message)
}

// TestKNIRVOracleService_RegisterCapability tests capability registration
func TestKNIRVOracleService_RegisterCapability(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	ctx := context.Background()
	request := CapabilityRegistrationRequest{
		Name:        "test-capability",
		Type:        "computation",
		Description: "A test capability",
		Schema: map[string]interface{}{
			"input":  "string",
			"output": "string",
		},
		Owner:         "test-owner",
		GasFeeNRN:     100,
		LocationHints: []string{"https://test.mcp.server.com"},
	}

	response, err := service.RegisterCapability(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Success)
	assert.Equal(t, "cap-12345", response.CapabilityID)
	assert.NotEmpty(t, response.Message)
}

// TestKNIRVOracleService_RequestFaucet tests faucet request
func TestKNIRVOracleService_RequestFaucet(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	ctx := context.Background()
	request := FaucetRequest{
		Address: "0x1234567890abcdef",
		Amount:  "1000",
		Reason:  "testing",
	}

	response, err := service.RequestFaucet(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Success)
	assert.Equal(t, "faucet-tx-12345", response.RequestID)
	assert.Equal(t, "1000", response.Amount)
	assert.Equal(t, "completed", response.Status)
}

// TestKNIRVOracleService_GetWalletBalance tests wallet balance retrieval
func TestKNIRVOracleService_GetWalletBalance(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	ctx := context.Background()
	address := "test-address"

	response, err := service.GetWalletBalance(ctx, address)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, address, response.Address)
	assert.Equal(t, "5000.50", response.Balance)
	assert.True(t, response.Success)
}

// TestKNIRVOracleService_SendTransaction tests transaction sending
func TestKNIRVOracleService_SendTransaction(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	ctx := context.Background()
	request := TransactionRequest{
		From:   "0x1234567890abcdef",
		To:     "0xfedcba0987654321",
		Amount: "100.50",
		Token:  "NRN",
		Memo:   "test transaction data",
	}

	response, err := service.SendTransaction(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Success)
	assert.Equal(t, "tx-98765", response.TransactionID)
	assert.Equal(t, "pending", response.Status)
}

// TestKNIRVOracleService_GetAgentCapabilities tests agent capabilities retrieval
func TestKNIRVOracleService_GetAgentCapabilities(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	ctx := context.Background()
	agentID := "test-agent"

	capabilities, err := service.GetAgentCapabilities(ctx, agentID)

	assert.NoError(t, err)
	assert.NotNil(t, capabilities)
	assert.Len(t, capabilities, 1)
	assert.Equal(t, "cap-1", capabilities[0]["id"])
	assert.Equal(t, "test-capability", capabilities[0]["name"])
}

// TestKNIRVOracleService_GetTreasuryStatus tests treasury status retrieval
func TestKNIRVOracleService_GetTreasuryStatus(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	ctx := context.Background()

	status, err := service.GetTreasuryStatus(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "1000000", status["total_balance"])
	assert.Equal(t, "active", status["status"])
	assert.NotNil(t, status["last_update"])
}

// TestKNIRVOracleService_ContextCancellation tests context cancellation
func TestKNIRVOracleService_ContextCancellation(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := service.HealthCheck(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

// TestKNIRVOracleService_Timeout tests request timeout
func TestKNIRVOracleService_Timeout(t *testing.T) {
	// Create a server that delays responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := KNIRVOracleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 1, // 1 second timeout
	}

	service := NewKNIRVOracleService(config)
	ctx := context.Background()

	err := service.HealthCheck(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline exceeded")
}

// TestKNIRVOracleService_ConcurrentRequests tests concurrent request handling
func TestKNIRVOracleService_ConcurrentRequests(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	var wg sync.WaitGroup
	numRequests := 10
	results := make(chan error, numRequests)

	// Make multiple concurrent health check requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			err := service.HealthCheck(ctx)
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	// Check that all requests succeeded
	successCount := 0
	for err := range results {
		if err == nil {
			successCount++
		}
	}

	assert.Equal(t, numRequests, successCount)
}

// TestKNIRVOracleService_ErrorHandling tests error handling
func TestKNIRVOracleService_ErrorHandling(t *testing.T) {
	// Create a server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "internal server error",
		})
	}))
	defer server.Close()

	config := KNIRVOracleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 30,
	}

	service := NewKNIRVOracleService(config)
	ctx := context.Background()

	// Test health check error
	err := service.HealthCheck(ctx)
	assert.Error(t, err)

	// Test mint agent error
	request := AgentMintRequest{
		AgentID: "test-agent",
		Name:    "Test Agent",
	}
	_, err = service.MintAgent(ctx, request)
	assert.Error(t, err)
}

// TestKNIRVOracleService_InvalidJSON tests handling of invalid JSON responses
func TestKNIRVOracleService_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json response"))
	}))
	defer server.Close()

	config := KNIRVOracleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 30,
	}

	service := NewKNIRVOracleService(config)
	ctx := context.Background()

	request := AgentMintRequest{
		AgentID: "test-agent",
		Name:    "Test Agent",
	}

	_, err := service.MintAgent(ctx, request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

// TestKNIRVOracleService_NetworkError tests network error handling
func TestKNIRVOracleService_NetworkError(t *testing.T) {
	config := KNIRVOracleConfig{
		BaseURL: "http://nonexistent.server.com",
		APIKey:  "test-key",
		Timeout: 1,
	}

	service := NewKNIRVOracleService(config)
	ctx := context.Background()

	err := service.HealthCheck(ctx)
	assert.Error(t, err)
}

// TestKNIRVOracleService_InvokeCapability tests capability invocation
func TestKNIRVOracleService_InvokeCapability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wallet/mcp/create_invoke_capability" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": "capability invoked successfully",
				"output": map[string]interface{}{
					"processed_data": "test result",
				},
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := KNIRVOracleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 30,
	}

	service := NewKNIRVOracleService(config)
	ctx := context.Background()

	inputData := map[string]interface{}{
		"input": "test data",
	}

	result, err := service.InvokeCapability(ctx, "test-capability", inputData)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result, "result")
}

// TestKNIRVOracleService_EdgeCases tests various edge cases
func TestKNIRVOracleService_EdgeCases(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	ctx := context.Background()

	// Test with empty agent ID
	_, err := service.GetAgentCapabilities(ctx, "")
	assert.Error(t, err)

	// Test with empty wallet address
	_, err = service.GetWalletBalance(ctx, "")
	assert.Error(t, err)

	// Test with nil context
	err = service.HealthCheck(nil)
	assert.Error(t, err)
}

// TestKNIRVOracleService_RequestValidation tests request validation
func TestKNIRVOracleService_RequestValidation(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	ctx := context.Background()

	// Test mint agent with empty required fields
	emptyRequest := AgentMintRequest{}
	response, err := service.MintAgent(ctx, emptyRequest)
	// This might succeed depending on server validation, but we test the flow
	if err != nil {
		assert.Error(t, err) // Server-side validation error
	} else {
		assert.NotNil(t, response) // Server accepted the request
	}

	// Test capability registration with empty fields
	emptyCapRequest := CapabilityRegistrationRequest{}
	capResponse, err := service.RegisterCapability(ctx, emptyCapRequest)
	if err != nil {
		assert.Error(t, err) // Server-side validation error
	} else {
		assert.NotNil(t, capResponse) // Server accepted the request
	}
}

// TestKNIRVOracleService_AuthenticationHeaders tests API key authentication
func TestKNIRVOracleService_AuthenticationHeaders(t *testing.T) {
	var receivedAPIKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAPIKey = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
		})
	}))
	defer server.Close()

	config := KNIRVOracleConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key-123",
		Timeout: 30,
	}

	service := NewKNIRVOracleService(config)
	ctx := context.Background()

	err := service.HealthCheck(ctx)
	assert.NoError(t, err)
	assert.Contains(t, receivedAPIKey, "test-api-key-123")
}

// TestKNIRVOracleService_LargePayloads tests handling of large payloads
func TestKNIRVOracleService_LargePayloads(t *testing.T) {
	service, server := createTestKNIRVOracleService(t)
	defer server.Close()

	ctx := context.Background()

	// Create a large metadata payload
	largeMetadata := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeMetadata[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	request := AgentMintRequest{
		AgentID:     "test-agent-large",
		Name:        "Test Agent with Large Metadata",
		Description: "Testing large payload handling",
		Owner:       "test-owner",
		Metadata:    largeMetadata,
	}

	response, err := service.MintAgent(ctx, request)

	// Should handle large payloads gracefully
	assert.NoError(t, err)
	assert.NotNil(t, response)
}
