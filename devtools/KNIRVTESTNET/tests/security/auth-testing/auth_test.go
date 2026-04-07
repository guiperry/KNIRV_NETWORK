package authtesting

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	GatewayURL = "http://localhost:8888"
	Timeout    = 30 * time.Second
)

// TestAuthenticationEndpoints tests authentication-related endpoints
func TestAuthenticationEndpoints(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Testnet Tokens Endpoint", func(t *testing.T) {
		resp, err := client.Get(GatewayURL + "/auth/testnet-tokens")
		if err != nil {
			t.Fatalf("Failed to get testnet tokens: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Testnet tokens endpoint failed with status: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var tokenResponse map[string]interface{}
		if err := json.Unmarshal(body, &tokenResponse); err != nil {
			t.Fatalf("Failed to parse token response: %v", err)
		}

		// Verify tokens structure
		if tokens, ok := tokenResponse["tokens"]; ok {
			if tokenMap, ok := tokens.(map[string]interface{}); ok {
				expectedTokens := []string{"admin", "developer", "guest"}
				for _, tokenType := range expectedTokens {
					if token, exists := tokenMap[tokenType]; exists {
						if tokenStr, ok := token.(string); ok && tokenStr != "" {
							t.Logf("✅ %s token available: %s", tokenType, tokenStr)
						} else {
							t.Errorf("❌ %s token is empty or invalid", tokenType)
						}
					} else {
						t.Errorf("❌ %s token not found", tokenType)
					}
				}
			}
		} else {
			t.Error("❌ No tokens field in response")
		}

		// Verify testnet flag
		if testnet, ok := tokenResponse["testnet"]; !ok || testnet != true {
			t.Error("❌ Testnet flag not set correctly")
		} else {
			t.Log("✅ Testnet flag verified")
		}
	})

	t.Run("Authentication Validation", func(t *testing.T) {
		// First get tokens
		resp, err := client.Get(GatewayURL + "/auth/testnet-tokens")
		if err != nil {
			t.Fatalf("Failed to get testnet tokens: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var tokenResponse map[string]interface{}
		if err := json.Unmarshal(body, &tokenResponse); err != nil {
			t.Fatalf("Failed to parse token response: %v", err)
		}

		// Test validation endpoint if it exists
		validationResp, err := client.Get(GatewayURL + "/auth/validate")
		if err != nil {
			t.Logf("⚠️ Auth validation endpoint not available: %v", err)
			return
		}
		defer validationResp.Body.Close()

		if validationResp.StatusCode == http.StatusOK {
			t.Log("✅ Auth validation endpoint is accessible")
		} else {
			t.Logf("⚠️ Auth validation endpoint returned status: %d", validationResp.StatusCode)
		}
	})
}

// TestSecurityHeaders tests security-related HTTP headers
func TestSecurityHeaders(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	endpoints := []string{
		"/gateway/health",
		"/gateway/services",
		"/auth/testnet-tokens",
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("Security Headers for %s", endpoint), func(t *testing.T) {
			resp, err := client.Get(GatewayURL + endpoint)
			if err != nil {
				t.Fatalf("Failed to get %s: %v", endpoint, err)
			}
			defer resp.Body.Close()

			// Check for security headers
			headers := resp.Header

			// Check for CORS headers
			if corsOrigin := headers.Get("Access-Control-Allow-Origin"); corsOrigin != "" {
				t.Logf("✅ CORS Origin header present: %s", corsOrigin)
			}

			// Check for content type
			if contentType := headers.Get("Content-Type"); contentType != "" {
				t.Logf("✅ Content-Type header present: %s", contentType)
			}

			// Check for server header (should be minimal for security)
			if server := headers.Get("Server"); server != "" {
				t.Logf("⚠️ Server header present: %s (consider removing for security)", server)
			} else {
				t.Log("✅ Server header not exposed")
			}

			// Check for X-Powered-By (should not be present)
			if poweredBy := headers.Get("X-Powered-By"); poweredBy != "" {
				t.Errorf("❌ X-Powered-By header exposed: %s (security risk)", poweredBy)
			} else {
				t.Log("✅ X-Powered-By header not exposed")
			}
		})
	}
}

// TestInputValidation tests input validation and sanitization
func TestInputValidation(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("SQL Injection Attempts", func(t *testing.T) {
		maliciousInputs := []string{
			"'; DROP TABLE users; --",
			"1' OR '1'='1",
			"admin'--",
			"' UNION SELECT * FROM users --",
		}

		for _, input := range maliciousInputs {
			// Test against endpoints that might accept parameters
			testURL := fmt.Sprintf("%s/gateway/services?query=%s", GatewayURL, input)
			resp, err := client.Get(testURL)
			if err != nil {
				t.Logf("⚠️ Request with malicious input failed (expected): %v", err)
				continue
			}
			defer resp.Body.Close()

			// The service should handle malicious input gracefully
			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				bodyStr := string(body)
				
				// Check that the malicious input is not reflected in the response
				if strings.Contains(bodyStr, input) {
					t.Errorf("❌ Malicious input reflected in response: %s", input)
				} else {
					t.Logf("✅ Malicious input handled safely: %s", input)
				}
			}
		}
	})

	t.Run("XSS Prevention", func(t *testing.T) {
		xssPayloads := []string{
			"<script>alert('xss')</script>",
			"javascript:alert('xss')",
			"<img src=x onerror=alert('xss')>",
			"'><script>alert('xss')</script>",
		}

		for _, payload := range xssPayloads {
			testURL := fmt.Sprintf("%s/gateway/services?search=%s", GatewayURL, payload)
			resp, err := client.Get(testURL)
			if err != nil {
				t.Logf("⚠️ Request with XSS payload failed (expected): %v", err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				bodyStr := string(body)
				
				// Check that XSS payload is not reflected unescaped
				if strings.Contains(bodyStr, payload) {
					t.Errorf("❌ XSS payload reflected unescaped: %s", payload)
				} else {
					t.Logf("✅ XSS payload handled safely: %s", payload)
				}
			}
		}
	})
}

// TestRateLimiting tests rate limiting mechanisms
func TestRateLimiting(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Rapid Request Test", func(t *testing.T) {
		const rapidRequests = 100
		const timeWindow = 10 * time.Second

		start := time.Now()
		successCount := 0
		rateLimitedCount := 0

		for i := 0; i < rapidRequests && time.Since(start) < timeWindow; i++ {
			resp, err := client.Get(GatewayURL + "/gateway/health")
			if err != nil {
				t.Logf("Request %d failed: %v", i, err)
				continue
			}
			resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusOK:
				successCount++
			case http.StatusTooManyRequests:
				rateLimitedCount++
				t.Logf("Rate limited at request %d", i)
			default:
				t.Logf("Unexpected status %d at request %d", resp.StatusCode, i)
			}
		}

		t.Logf("✅ Rapid request test results:")
		t.Logf("   Successful requests: %d", successCount)
		t.Logf("   Rate limited requests: %d", rateLimitedCount)
		t.Logf("   Total requests: %d", successCount+rateLimitedCount)

		if rateLimitedCount > 0 {
			t.Log("✅ Rate limiting is active")
		} else {
			t.Log("⚠️ No rate limiting detected (may be intentional for testnet)")
		}
	})
}

// TestHTTPSRedirection tests HTTPS redirection (if applicable)
func TestHTTPSRedirection(t *testing.T) {
	client := &http.Client{
		Timeout: Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	t.Run("HTTPS Redirection Check", func(t *testing.T) {
		// Test if HTTP redirects to HTTPS (in production environments)
		resp, err := client.Get("http://localhost:8888/gateway/health")
		if err != nil {
			t.Fatalf("Failed to test HTTPS redirection: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
			location := resp.Header.Get("Location")
			if strings.HasPrefix(location, "https://") {
				t.Log("✅ HTTPS redirection is configured")
			} else {
				t.Log("⚠️ Redirection present but not to HTTPS")
			}
		} else {
			t.Log("⚠️ No HTTPS redirection (expected for testnet)")
		}
	})
}

// TestErrorHandling tests error handling and information disclosure
func TestErrorHandling(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("404 Error Handling", func(t *testing.T) {
		resp, err := client.Get(GatewayURL + "/nonexistent/endpoint")
		if err != nil {
			t.Fatalf("Failed to test 404 handling: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)
			
			// Check that error doesn't expose sensitive information
			sensitiveInfo := []string{
				"stack trace",
				"internal error",
				"database",
				"sql",
				"password",
				"token",
			}
			
			exposedInfo := false
			for _, info := range sensitiveInfo {
				if strings.Contains(strings.ToLower(bodyStr), info) {
					t.Errorf("❌ 404 error exposes sensitive information: %s", info)
					exposedInfo = true
				}
			}
			
			if !exposedInfo {
				t.Log("✅ 404 error handling is secure")
			}
		} else {
			t.Logf("⚠️ Expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("Method Not Allowed Handling", func(t *testing.T) {
		resp, err := client.Post(GatewayURL+"/gateway/health", "application/json", strings.NewReader("{}"))
		if err != nil {
			t.Fatalf("Failed to test method not allowed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusMethodNotAllowed {
			t.Log("✅ Method not allowed handling is working")
		} else {
			t.Logf("⚠️ Expected 405, got %d", resp.StatusCode)
		}
	})
}
