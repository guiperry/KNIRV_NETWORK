package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// EconomicsTestConfig holds configuration for economics integration tests
type EconomicsTestConfig struct {
	EconomicsURL string
	Timeout      time.Duration
}

// EconomicsResponse represents a response from the economics service
type EconomicsResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// SkillInvocationRequest represents a skill invocation request
type SkillInvocationRequest struct {
	UserID  string `json:"user_id"`
	SkillID string `json:"skill_id"`
	Amount  string `json:"amount"`
}

// LLMRegistrationRequest represents an LLM registration request
type LLMRegistrationRequest struct {
	UserID          string `json:"user_id"`
	LLMID           string `json:"llm_id"`
	RegistrationFee string `json:"registration_fee"`
}

// ValidationRewardRequest represents a validation reward request
type ValidationRewardRequest struct {
	ValidatorID      string `json:"validator_id"`
	TargetID         string `json:"target_id"`
	ValidationResult bool   `json:"validation_result"`
}

// NetworkFeesRequest represents a network fees calculation request
type NetworkFeesRequest struct {
	GasUsed  uint64 `json:"gas_used"`
	Priority string `json:"priority"`
}

// getEconomicsConfig returns the configuration for economics tests
func getEconomicsConfig() *EconomicsTestConfig {
	economicsURL := os.Getenv("ECONOMICS_SERVICE_URL")
	if economicsURL == "" {
		economicsURL = "http://localhost:8090"
	}

	return &EconomicsTestConfig{
		EconomicsURL: economicsURL,
		Timeout:      30 * time.Second,
	}
}

// makeEconomicsRequest makes an HTTP request to the economics service
func makeEconomicsRequest(t *testing.T, method, endpoint string, data interface{}) (*EconomicsResponse, error) {
	config := getEconomicsConfig()
	url := config.EconomicsURL + endpoint

	var reqBody []byte
	var err error
	if data != nil {
		reqBody, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	client := &http.Client{Timeout: config.Timeout}

	var req *http.Request
	if reqBody != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var economicsResp EconomicsResponse
	if err := json.Unmarshal(body, &economicsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &economicsResp, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, economicsResp.Error)
	}

	return &economicsResp, nil
}

// TestEconomicsServiceHealth tests the economics service health endpoint
func TestEconomicsServiceHealth(t *testing.T) {
	resp, err := makeEconomicsRequest(t, "GET", "/economics/health", nil)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Health check returned success=false: %s", resp.Error)
	}

	t.Logf("Economics service health check passed")
}

// TestEconomicsServiceInfo tests the economics service info endpoint
func TestEconomicsServiceInfo(t *testing.T) {
	resp, err := makeEconomicsRequest(t, "GET", "/economics/info", nil)
	if err != nil {
		t.Fatalf("Service info failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Service info returned success=false: %s", resp.Error)
	}

	// Check for expected fields
	if resp.Data["service"] == nil {
		t.Errorf("Service info missing 'service' field")
	}
	if resp.Data["version"] == nil {
		t.Errorf("Service info missing 'version' field")
	}

	t.Logf("Economics service info check passed")
}

// TestEconomicsMetrics tests the economics metrics endpoint
func TestEconomicsMetrics(t *testing.T) {
	resp, err := makeEconomicsRequest(t, "GET", "/economics/metrics", nil)
	if err != nil {
		t.Fatalf("Metrics request failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Metrics returned success=false: %s", resp.Error)
	}

	// Check for expected metric fields
	expectedFields := []string{"total_supply", "circulating_supply", "total_burned", "last_updated"}
	for _, field := range expectedFields {
		if resp.Data[field] == nil {
			t.Errorf("Metrics missing expected field: %s", field)
		}
	}

	t.Logf("Economics metrics check passed")
}

// TestSkillInvocation tests the skill invocation endpoint
func TestSkillInvocation(t *testing.T) {
	request := SkillInvocationRequest{
		UserID:  "test_user_integration",
		SkillID: "test_skill_integration",
		Amount:  "100000",
	}

	resp, err := makeEconomicsRequest(t, "POST", "/economics/skill/invoke", request)
	if err != nil {
		t.Fatalf("Skill invocation failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Skill invocation returned success=false: %s", resp.Error)
	}

	// Check for transaction ID
	if resp.Data["transaction_id"] == nil {
		t.Errorf("Skill invocation response missing transaction_id")
	}

	t.Logf("Skill invocation test passed, transaction ID: %v", resp.Data["transaction_id"])
}

// TestLLMRegistration tests the LLM registration endpoint
func TestLLMRegistration(t *testing.T) {
	request := LLMRegistrationRequest{
		UserID:          "test_user_integration",
		LLMID:           "test_llm_integration",
		RegistrationFee: "1000000",
	}

	resp, err := makeEconomicsRequest(t, "POST", "/economics/llm/register", request)
	if err != nil {
		t.Fatalf("LLM registration failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("LLM registration returned success=false: %s", resp.Error)
	}

	// Check for transaction ID
	if resp.Data["transaction_id"] == nil {
		t.Errorf("LLM registration response missing transaction_id")
	}

	t.Logf("LLM registration test passed, transaction ID: %v", resp.Data["transaction_id"])
}

// TestValidationReward tests the validation reward endpoint
func TestValidationReward(t *testing.T) {
	request := ValidationRewardRequest{
		ValidatorID:      "test_validator_integration",
		TargetID:         "test_target_integration",
		ValidationResult: true,
	}

	resp, err := makeEconomicsRequest(t, "POST", "/economics/validation/reward", request)
	if err != nil {
		t.Fatalf("Validation reward failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Validation reward returned success=false: %s", resp.Error)
	}

	// Check for transaction ID
	if resp.Data["transaction_id"] == nil {
		t.Errorf("Validation reward response missing transaction_id")
	}

	t.Logf("Validation reward test passed, transaction ID: %v", resp.Data["transaction_id"])
}

// TestNetworkFeeCalculation tests the network fee calculation endpoint
func TestNetworkFeeCalculation(t *testing.T) {
	request := NetworkFeesRequest{
		GasUsed:  21000,
		Priority: "medium",
	}

	resp, err := makeEconomicsRequest(t, "POST", "/economics/fees/calculate", request)
	if err != nil {
		t.Fatalf("Network fee calculation failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Network fee calculation returned success=false: %s", resp.Error)
	}

	// Check for expected fields
	if resp.Data["total_fee"] == nil {
		t.Errorf("Fee calculation response missing total_fee")
	}
	if resp.Data["gas_price"] == nil {
		t.Errorf("Fee calculation response missing gas_price")
	}

	t.Logf("Network fee calculation test passed, total fee: %v", resp.Data["total_fee"])
}

// TestEconomicsRules tests the economics rules endpoints
func TestEconomicsRules(t *testing.T) {
	// Test GET rules
	resp, err := makeEconomicsRequest(t, "GET", "/economics/rules", nil)
	if err != nil {
		t.Fatalf("Get economics rules failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Get economics rules returned success=false: %s", resp.Error)
	}

	// Check for expected rule fields
	expectedFields := []string{"skill_invocation_cost", "llm_registration_fee", "validation_reward"}
	for _, field := range expectedFields {
		if resp.Data[field] == nil {
			t.Errorf("Economics rules missing expected field: %s", field)
		}
	}

	t.Logf("Economics rules retrieval test passed")
}

// TestTransactionHistory tests the transaction history endpoints
func TestTransactionHistory(t *testing.T) {
	// Test get transactions
	resp, err := makeEconomicsRequest(t, "GET", "/economics/transactions?limit=5", nil)
	if err != nil {
		t.Fatalf("Get transactions failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Get transactions returned success=false: %s", resp.Error)
	}

	// Check for transactions array
	if resp.Data["transactions"] == nil {
		t.Errorf("Transactions response missing transactions array")
	}

	t.Logf("Transaction history test passed")
}

// TestBurnTracking tests the burn tracking endpoints
func TestBurnTracking(t *testing.T) {
	// Test get burn history
	resp, err := makeEconomicsRequest(t, "GET", "/economics/burn/history?limit=5", nil)
	if err != nil {
		t.Fatalf("Get burn history failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Get burn history returned success=false: %s", resp.Error)
	}

	// Test get total burned
	resp, err = makeEconomicsRequest(t, "GET", "/economics/burn/total", nil)
	if err != nil {
		t.Fatalf("Get total burned failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Get total burned returned success=false: %s", resp.Error)
	}

	if resp.Data["total_burned"] == nil {
		t.Errorf("Total burned response missing total_burned field")
	}

	t.Logf("Burn tracking test passed")
}

// TestIntegrationStatus tests the integration status endpoint
func TestIntegrationStatus(t *testing.T) {
	resp, err := makeEconomicsRequest(t, "GET", "/economics/integration/status", nil)
	if err != nil {
		t.Fatalf("Integration status failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("Integration status returned success=false: %s", resp.Error)
	}

	// Check for expected status fields
	if resp.Data["status"] == nil {
		t.Errorf("Integration status missing status field")
	}

	t.Logf("Integration status test passed")
}

// TestEconomicsWorkflow tests a complete economics workflow
func TestEconomicsWorkflow(t *testing.T) {
	t.Run("Complete Economics Workflow", func(t *testing.T) {
		// 1. Check service health
		t.Run("Health Check", func(t *testing.T) {
			TestEconomicsServiceHealth(t)
		})

		// 2. Get initial metrics
		t.Run("Initial Metrics", func(t *testing.T) {
			TestEconomicsMetrics(t)
		})

		// 3. Process skill invocation
		t.Run("Skill Invocation", func(t *testing.T) {
			TestSkillInvocation(t)
		})

		// 4. Register LLM
		t.Run("LLM Registration", func(t *testing.T) {
			TestLLMRegistration(t)
		})

		// 5. Process validation reward
		t.Run("Validation Reward", func(t *testing.T) {
			TestValidationReward(t)
		})

		// 6. Check updated metrics
		t.Run("Updated Metrics", func(t *testing.T) {
			TestEconomicsMetrics(t)
		})

		// 7. Check transaction history
		t.Run("Transaction History", func(t *testing.T) {
			TestTransactionHistory(t)
		})

		// 8. Check burn tracking
		t.Run("Burn Tracking", func(t *testing.T) {
			TestBurnTracking(t)
		})
	})
}
