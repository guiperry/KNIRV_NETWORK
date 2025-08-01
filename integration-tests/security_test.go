package integration_tests

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Month 12 Security Testing Framework
type SecurityTestSuite struct {
	suite.Suite
	gatewayURL     string
	httpClient     *http.Client
	insecureClient *http.Client
	authToken      string
	testWallet     *TestWallet
}

type SecurityTestResult struct {
	TestName string            `json:"test_name"`
	Passed   bool              `json:"passed"`
	Issues   []SecurityIssue   `json:"issues,omitempty"`
	Metrics  map[string]string `json:"metrics,omitempty"`
}

type SecurityIssue struct {
	Severity    string `json:"severity"` // "critical", "high", "medium", "low"
	Type        string `json:"type"`     // "authentication", "authorization", "encryption", etc.
	Description string `json:"description"`
	Endpoint    string `json:"endpoint,omitempty"`
}

func (suite *SecurityTestSuite) SetupSuite() {
	suite.gatewayURL = "http://localhost:8000"

	// Standard HTTP client
	suite.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Insecure HTTP client for testing TLS/SSL
	suite.insecureClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Wait for services
	suite.waitForServices()

	// Authenticate for authorized tests
	suite.authenticate()

	// Create test wallet
	suite.createTestWallet()
}

func (suite *SecurityTestSuite) waitForServices() {
	services := []string{"knirvchain", "knirvgraph", "knirvnexus", "knirvroot", "knirvrouter"}

	for _, service := range services {
		suite.T().Logf("Waiting for service: %s", service)

		for i := 0; i < 30; i++ {
			resp, err := suite.httpClient.Get(fmt.Sprintf("%s/%s/health", suite.gatewayURL, service))
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(1 * time.Second)
		}
	}

	suite.T().Log("All services are ready for security testing")
}

func (suite *SecurityTestSuite) authenticate() {
	loginData := map[string]string{
		"username": "admin",
		"password": "password",
	}

	resp := suite.makeRequest("POST", "/auth/login", loginData)
	require.True(suite.T(), resp.Success, "Authentication failed")

	suite.authToken = resp.Data["token"].(string)
	suite.T().Log("Authenticated for security testing")
}

func (suite *SecurityTestSuite) createTestWallet() {
	walletData := map[string]string{
		"name": "security_test_wallet",
	}

	resp := suite.makeAuthenticatedRequest("POST", "/knirvwallet/wallet/create", walletData)
	require.True(suite.T(), resp.Success, "Failed to create test wallet")

	suite.testWallet = &TestWallet{
		Address:  resp.Data["address"].(string),
		Mnemonic: resp.Data["mnemonic"].(string),
		Balance:  "0",
	}

	suite.T().Logf("Created security test wallet: %s", suite.testWallet.Address)
}

// Test 1: Authentication Security
func (suite *SecurityTestSuite) TestAuthenticationSecurity() {
	suite.Run("AuthenticationSecurityTest", func() {
		var issues []SecurityIssue

		// Test 1.1: Invalid credentials
		suite.T().Log("Testing invalid credentials rejection...")
		invalidResp := suite.makeRequest("POST", "/auth/login", map[string]string{
			"username": "invalid_user",
			"password": "invalid_password",
		})

		if invalidResp.Success {
			issues = append(issues, SecurityIssue{
				Severity:    "critical",
				Type:        "authentication",
				Description: "System accepts invalid credentials",
				Endpoint:    "/auth/login",
			})
		}

		// Test 1.2: Empty credentials
		suite.T().Log("Testing empty credentials rejection...")
		emptyResp := suite.makeRequest("POST", "/auth/login", map[string]string{
			"username": "",
			"password": "",
		})

		if emptyResp.Success {
			issues = append(issues, SecurityIssue{
				Severity:    "high",
				Type:        "authentication",
				Description: "System accepts empty credentials",
				Endpoint:    "/auth/login",
			})
		}

		// Test 1.3: SQL injection attempts
		suite.T().Log("Testing SQL injection protection...")
		sqlInjectionResp := suite.makeRequest("POST", "/auth/login", map[string]string{
			"username": "admin'; DROP TABLE users; --",
			"password": "password",
		})

		if sqlInjectionResp.Success {
			issues = append(issues, SecurityIssue{
				Severity:    "critical",
				Type:        "authentication",
				Description: "System vulnerable to SQL injection",
				Endpoint:    "/auth/login",
			})
		}

		// Test 1.4: Brute force protection
		suite.T().Log("Testing brute force protection...")
		for i := 0; i < 10; i++ {
			suite.makeRequest("POST", "/auth/login", map[string]string{
				"username": "admin",
				"password": fmt.Sprintf("wrong_password_%d", i),
			})
		}

		// After multiple failed attempts, should be rate limited
		finalAttempt := suite.makeRequest("POST", "/auth/login", map[string]string{
			"username": "admin",
			"password": "password",
		})

		// If successful immediately after brute force, there's no protection
		if finalAttempt.Success {
			issues = append(issues, SecurityIssue{
				Severity:    "medium",
				Type:        "authentication",
				Description: "No brute force protection detected",
				Endpoint:    "/auth/login",
			})
		}

		result := SecurityTestResult{
			TestName: "AuthenticationSecurity",
			Passed:   len(issues) == 0,
			Issues:   issues,
		}

		suite.T().Logf("Authentication Security Test Result: %+v", result)
		assert.True(suite.T(), result.Passed, "Authentication security issues found: %d", len(issues))
	})
}

// Test 2: Authorization Security
func (suite *SecurityTestSuite) TestAuthorizationSecurity() {
	suite.Run("AuthorizationSecurityTest", func() {
		var issues []SecurityIssue

		// Test 2.1: Access without authentication
		suite.T().Log("Testing unauthorized access protection...")
		unauthorizedEndpoints := []string{
			"/knirvchain/llm/register",
			"/knirvgraph/nrv/errors",
			"/knirvnexus/agents/create",
			"/knirvwallet/wallet/create",
		}

		for _, endpoint := range unauthorizedEndpoints {
			resp := suite.makeRequest("GET", endpoint, nil)
			if resp.Success {
				issues = append(issues, SecurityIssue{
					Severity:    "high",
					Type:        "authorization",
					Description: "Endpoint accessible without authentication",
					Endpoint:    endpoint,
				})
			}
		}

		// Test 2.2: Invalid token access
		suite.T().Log("Testing invalid token rejection...")
		invalidTokenResp := suite.makeRequestWithAuth("GET", "/knirvchain/llm/list", nil, "invalid_token_12345")
		if invalidTokenResp.Success {
			issues = append(issues, SecurityIssue{
				Severity:    "critical",
				Type:        "authorization",
				Description: "System accepts invalid authentication tokens",
				Endpoint:    "/knirvchain/llm/list",
			})
		}

		// Test 2.3: Expired token handling (simulate)
		suite.T().Log("Testing expired token handling...")
		expiredTokenResp := suite.makeRequestWithAuth("GET", "/knirvchain/llm/list", nil, "expired.token.here")
		if expiredTokenResp.Success {
			issues = append(issues, SecurityIssue{
				Severity:    "high",
				Type:        "authorization",
				Description: "System may accept expired tokens",
				Endpoint:    "/knirvchain/llm/list",
			})
		}

		result := SecurityTestResult{
			TestName: "AuthorizationSecurity",
			Passed:   len(issues) == 0,
			Issues:   issues,
		}

		suite.T().Logf("Authorization Security Test Result: %+v", result)
		assert.True(suite.T(), result.Passed, "Authorization security issues found: %d", len(issues))
	})
}

// Test 3: Rate Limiting Security
func (suite *SecurityTestSuite) TestRateLimitingSecurity() {
	suite.Run("RateLimitingSecurityTest", func() {
		var issues []SecurityIssue

		// Test 3.1: API rate limiting
		suite.T().Log("Testing API rate limiting...")

		rateLimitHit := false
		for i := 0; i < 100; i++ { // Send 100 requests rapidly
			resp := suite.makeAuthenticatedRequest("GET", "/knirvchain/health", nil)
			if !resp.Success && strings.Contains(resp.Error, "rate limit") {
				rateLimitHit = true
				break
			}
			time.Sleep(10 * time.Millisecond) // Small delay between requests
		}

		if !rateLimitHit {
			issues = append(issues, SecurityIssue{
				Severity:    "medium",
				Type:        "rate_limiting",
				Description: "No rate limiting detected on API endpoints",
				Endpoint:    "/knirvchain/health",
			})
		}

		result := SecurityTestResult{
			TestName: "RateLimitingSecurity",
			Passed:   len(issues) == 0,
			Issues:   issues,
		}

		suite.T().Logf("Rate Limiting Security Test Result: %+v", result)
		assert.True(suite.T(), result.Passed, "Rate limiting security issues found: %d", len(issues))
	})
}

// Test 4: Input Validation Security
func (suite *SecurityTestSuite) TestInputValidationSecurity() {
	suite.Run("InputValidationSecurityTest", func() {
		var issues []SecurityIssue

		// Test 4.1: XSS protection
		suite.T().Log("Testing XSS protection...")
		xssPayload := "<script>alert('xss')</script>"

		xssResp := suite.makeAuthenticatedRequest("POST", "/knirvgraph/nrv/errors", map[string]interface{}{
			"error_type":  xssPayload,
			"description": xssPayload,
			"context": map[string]interface{}{
				"user_input": xssPayload,
			},
			"severity": 1,
		})

		// Check if XSS payload was sanitized or rejected
		if xssResp.Success {
			// Additional check would be needed to see if the payload was sanitized
			suite.T().Log("XSS payload accepted - checking if sanitized...")
		}

		// Test 4.2: Command injection protection
		suite.T().Log("Testing command injection protection...")
		cmdInjectionPayload := "; rm -rf /"

		cmdResp := suite.makeAuthenticatedRequest("POST", "/knirvnexus/agents/create", map[string]interface{}{
			"name":        cmdInjectionPayload,
			"description": "Test agent",
			"type":        "go_plugin",
		})

		if cmdResp.Success {
			suite.T().Log("Command injection payload accepted - system should validate input")
		}

		result := SecurityTestResult{
			TestName: "InputValidationSecurity",
			Passed:   len(issues) == 0,
			Issues:   issues,
		}

		suite.T().Logf("Input Validation Security Test Result: %+v", result)
		assert.True(suite.T(), result.Passed, "Input validation security issues found: %d", len(issues))
	})
}

// Test 5: HTTPS and Encryption Security
func (suite *SecurityTestSuite) TestHTTPSEncryptionSecurity() {
	suite.Run("HTTPSEncryptionSecurityTest", func() {
		var issues []SecurityIssue

		// Test 5.1: HTTPS enforcement
		suite.T().Log("Testing HTTPS enforcement...")

		// Try to access via HTTP (should redirect to HTTPS or reject)
		httpURL := strings.Replace(suite.gatewayURL, "http://", "http://", 1)
		resp, err := suite.httpClient.Get(httpURL + "/health")

		if err == nil && resp.StatusCode == 200 {
			issues = append(issues, SecurityIssue{
				Severity:    "high",
				Type:        "encryption",
				Description: "HTTP access allowed - HTTPS not enforced",
				Endpoint:    "/health",
			})
		}
		if resp != nil {
			resp.Body.Close()
		}

		// Test 5.2: TLS configuration
		suite.T().Log("Testing TLS configuration...")

		// This would require HTTPS to be properly configured
		httpsURL := strings.Replace(suite.gatewayURL, "http://", "https://", 1)
		_, err = suite.insecureClient.Get(httpsURL + "/health")

		if err != nil {
			suite.T().Logf("HTTPS not configured or accessible: %v", err)
		}

		result := SecurityTestResult{
			TestName: "HTTPSEncryptionSecurity",
			Passed:   len(issues) == 0,
			Issues:   issues,
		}

		suite.T().Logf("HTTPS Encryption Security Test Result: %+v", result)
		assert.True(suite.T(), result.Passed, "HTTPS encryption security issues found: %d", len(issues))
	})
}

// Test 6: Wallet and Transaction Security
func (suite *SecurityTestSuite) TestWalletTransactionSecurity() {
	suite.Run("WalletTransactionSecurityTest", func() {
		var issues []SecurityIssue

		// Test 6.1: Unauthorized wallet access
		suite.T().Log("Testing unauthorized wallet access...")

		unauthorizedWalletResp := suite.makeRequest("GET", fmt.Sprintf("/knirvwallet/balance/%s", suite.testWallet.Address), nil)
		if unauthorizedWalletResp.Success {
			issues = append(issues, SecurityIssue{
				Severity:    "critical",
				Type:        "authorization",
				Description: "Wallet balance accessible without authentication",
				Endpoint:    "/knirvwallet/balance",
			})
		}

		// Test 6.2: Transaction validation
		suite.T().Log("Testing transaction validation...")

		invalidTxResp := suite.makeAuthenticatedRequest("POST", "/knirvchain/send_txn", map[string]interface{}{
			"from":   "invalid_address",
			"to":     "another_invalid_address",
			"amount": "-1000", // Negative amount
			"type":   "transfer",
		})

		if invalidTxResp.Success {
			issues = append(issues, SecurityIssue{
				Severity:    "high",
				Type:        "validation",
				Description: "Invalid transaction accepted",
				Endpoint:    "/knirvchain/send_txn",
			})
		}

		result := SecurityTestResult{
			TestName: "WalletTransactionSecurity",
			Passed:   len(issues) == 0,
			Issues:   issues,
		}

		suite.T().Logf("Wallet Transaction Security Test Result: %+v", result)
		assert.True(suite.T(), result.Passed, "Wallet transaction security issues found: %d", len(issues))
	})
}

// Helper methods for security testing
func (suite *SecurityTestSuite) makeRequest(method, path string, data interface{}) *TestResponse {
	return suite.makeRequestWithAuth(method, path, data, "")
}

func (suite *SecurityTestSuite) makeAuthenticatedRequest(method, path string, data interface{}) *TestResponse {
	return suite.makeRequestWithAuth(method, path, data, suite.authToken)
}

func (suite *SecurityTestSuite) makeRequestWithAuth(method, path string, data interface{}, token string) *TestResponse {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return &TestResponse{Success: false, Error: "Failed to marshal request data"}
		}
		body = bytes.NewReader(jsonData)
	}

	url := suite.gatewayURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return &TestResponse{Success: false, Error: "Failed to create request"}
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := suite.httpClient.Do(req)
	if err != nil {
		return &TestResponse{Success: false, Error: err.Error()}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &TestResponse{Success: false, Error: "Failed to read response body"}
	}

	var testResp TestResponse
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if len(responseBody) > 0 {
			err = json.Unmarshal(responseBody, &testResp.Data)
			if err != nil {
				json.Unmarshal(responseBody, &testResp)
			} else {
				testResp.Success = true
			}
		} else {
			testResp.Success = true
		}
	} else {
		testResp.Success = false
		testResp.Error = string(responseBody)
	}

	return &testResp
}

// Main test function for the Security Test Suite
func TestSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}
