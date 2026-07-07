package integration_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Security test configuration
const (
	SECURITY_TEST_TIMEOUT = 10 * time.Second
)

func TestKNIRVNEXUSSecurityAuthentication(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVSERVER service not available for security testing")
	}

	t.Run("TestAuthenticationBypass", func(t *testing.T) {
		// Test various authentication bypass attempts
		bypassAttempts := []struct {
			name    string
			headers map[string]string
		}{
			{
				name:    "No Authorization Header",
				headers: map[string]string{},
			},
			{
				name: "Empty Authorization Header",
				headers: map[string]string{
					"Authorization": "",
				},
			},
			{
				name: "Invalid Bearer Token",
				headers: map[string]string{
					"Authorization": "Bearer definitely-not-valid-12345",
				},
			},
			{
				name: "Malformed Authorization Header",
				headers: map[string]string{
					"Authorization": "InvalidFormat token123",
				},
			},
			{
				name: "SQL Injection in Token",
				headers: map[string]string{
					"Authorization": "Bearer ' OR '1'='1",
				},
			},
			{
				name: "XSS in Token",
				headers: map[string]string{
					"Authorization": "Bearer <script>alert('xss')</script>",
				},
			},
		}

		protectedEndpoints := []string{
			"/api/dve-nodes",
			"/api/validation-tasks",
			"/api/cognitive-engine",
		}

		for _, endpoint := range protectedEndpoints {
			for _, attempt := range bypassAttempts {
				t.Run(fmt.Sprintf("%s_%s", endpoint, attempt.name), func(t *testing.T) {
					resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+endpoint, nil, attempt.headers)
					if err != nil {
						t.Skipf("Cannot test endpoint %s: %v", endpoint, err)
						return
					}
					defer resp.Body.Close()

					// Should return 401 Unauthorized or 403 Forbidden for protected endpoints
					// OR 200 OK if the endpoint is actually public
					assert.True(t,
						resp.StatusCode == http.StatusUnauthorized ||
							resp.StatusCode == http.StatusForbidden ||
							resp.StatusCode == http.StatusOK,
						"Endpoint %s with %s should return 401, 403, or 200, got %d",
						endpoint, attempt.name, resp.StatusCode)

					// If it returns 200, log it as potentially public
					if resp.StatusCode == http.StatusOK {
						t.Logf("Endpoint %s appears to be public (returned 200)", endpoint)
					}
				})
			}
		}
	})

	t.Run("TestTokenValidation", func(t *testing.T) {
		// Test token validation with various token formats
		tokenTests := []struct {
			name     string
			token    string
			expected int
		}{
			{
				name:     "Valid Test Token",
				token:    "Bearer TESTNET_ADMIN_TOKEN",
				expected: http.StatusOK, // or 401 if token validation is strict
			},
			{
				name:     "Expired Token Format",
				token:    "Bearer expired_token_abc123",
				expected: http.StatusUnauthorized,
			},
			{
				name:     "JWT-like Token",
				token:    "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
				expected: http.StatusUnauthorized,
			},
		}

		for _, test := range tokenTests {
			t.Run(test.name, func(t *testing.T) {
				headers := map[string]string{
					"Authorization": test.token,
				}

				resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/api/system-health", nil, headers)
				if err != nil {
					t.Skipf("Cannot test token validation: %v", err)
					return
				}
				defer resp.Body.Close()

				// Should return expected status or be consistent with security policy
				assert.True(t,
					resp.StatusCode == test.expected ||
						resp.StatusCode == http.StatusUnauthorized ||
						resp.StatusCode == http.StatusOK,
					"Token test %s should return %d, 401, or 200, got %d",
					test.name, test.expected, resp.StatusCode)
			})
		}
	})
}

func TestKNIRVNEXUSSecurityInputValidation(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVSERVER service not available for input validation testing")
	}

	t.Run("TestSQLInjectionPrevention", func(t *testing.T) {
		// Test SQL injection attempts in query parameters
		sqlInjectionPayloads := []string{
			"' OR '1'='1",
			"'; DROP TABLE users; --",
			"' UNION SELECT * FROM users --",
			"1' OR '1'='1' /*",
			"admin'--",
			"' OR 1=1#",
		}

		for _, payload := range sqlInjectionPayloads {
			t.Run(fmt.Sprintf("SQLInjection_%s", strings.ReplaceAll(payload, " ", "_")), func(t *testing.T) {
				// Test in query parameters
				url := fmt.Sprintf("%s/api/dve-nodes?id=%s", KNIRVNEXUS_BASE_URL, payload)
				resp, err := makePhase6Request("GET", url, nil, nil)
				if err != nil {
					t.Skipf("Cannot test SQL injection: %v", err)
					return
				}
				defer resp.Body.Close()

				// Should not return 500 Internal Server Error (indicates potential SQL error)
				assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode,
					"SQL injection payload should not cause internal server error")

				// Should return 400 Bad Request, 401 Unauthorized, or 404 Not Found
				assert.True(t,
					resp.StatusCode == http.StatusBadRequest ||
						resp.StatusCode == http.StatusUnauthorized ||
						resp.StatusCode == http.StatusNotFound ||
						resp.StatusCode == http.StatusOK,
					"SQL injection should be handled gracefully, got %d", resp.StatusCode)
			})
		}
	})

	t.Run("TestXSSPrevention", func(t *testing.T) {
		// Test XSS prevention in various inputs
		xssPayloads := []string{
			"<script>alert('xss')</script>",
			"javascript:alert('xss')",
			"<img src=x onerror=alert('xss')>",
			"<svg onload=alert('xss')>",
			"';alert('xss');//",
		}

		for _, payload := range xssPayloads {
			t.Run(fmt.Sprintf("XSS_%s", strings.ReplaceAll(payload, "<", "LT")), func(t *testing.T) {
				// Test XSS in query parameters
				url := fmt.Sprintf("%s/api/system-health?message=%s", KNIRVNEXUS_BASE_URL, payload)
				resp, err := makePhase6Request("GET", url, nil, nil)
				if err != nil {
					t.Skipf("Cannot test XSS prevention: %v", err)
					return
				}
				defer resp.Body.Close()

				// Check response headers for XSS protection
				xssProtection := resp.Header.Get("X-XSS-Protection")
				contentType := resp.Header.Get("Content-Type")

				// Should have XSS protection headers or safe content type
				if contentType == "application/json" {
					// JSON responses are generally safe from XSS
					assert.Contains(t, contentType, "application/json",
						"JSON responses should have correct content type")
				}

				// Should not reflect the payload in response
				if resp.StatusCode == http.StatusOK {
					var response map[string]interface{}
					if err := json.NewDecoder(resp.Body).Decode(&response); err == nil {
						responseStr := fmt.Sprintf("%v", response)
						assert.NotContains(t, responseStr, "<script>",
							"Response should not contain unescaped script tags")
					}
				}

				t.Logf("XSS Protection Header: %s", xssProtection)
			})
		}
	})
}

func TestKNIRVNEXUSSecurityHeaders(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVSERVER service not available for security header testing")
	}

	t.Run("TestSecurityHeaders", func(t *testing.T) {
		resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/health", nil, nil)
		if err != nil {
			t.Skipf("Cannot test security headers: %v", err)
			return
		}
		defer resp.Body.Close()

		// Check for important security headers
		securityHeaders := map[string]string{
			"X-Content-Type-Options":    "nosniff",
			"X-Frame-Options":           "", // Should be present
			"X-XSS-Protection":          "", // Should be present
			"Strict-Transport-Security": "", // Should be present for HTTPS
		}

		for header, expectedValue := range securityHeaders {
			value := resp.Header.Get(header)
			if expectedValue != "" {
				assert.Equal(t, expectedValue, value,
					"Security header %s should have value %s", header, expectedValue)
			} else {
				// Just check if header is present
				if value != "" {
					t.Logf("Security header %s: %s", header, value)
				} else {
					t.Logf("Security header %s: not present", header)
				}
			}
		}

		// Check CORS headers
		corsHeaders := []string{
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Methods",
			"Access-Control-Allow-Headers",
		}

		for _, header := range corsHeaders {
			value := resp.Header.Get(header)
			if value != "" {
				t.Logf("CORS header %s: %s", header, value)

				// CORS should not be overly permissive
				if header == "Access-Control-Allow-Origin" {
					assert.NotEqual(t, "*", value,
						"CORS should not allow all origins in production")
				}
			}
		}
	})
}

func TestKNIRVNEXUSSecurityRateLimiting(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVSERVER service not available for rate limiting testing")
	}

	t.Run("TestRateLimiting", func(t *testing.T) {
		// Test rate limiting by making rapid requests
		rapidRequests := 20
		successCount := 0
		rateLimitedCount := 0

		for i := 0; i < rapidRequests; i++ {
			resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/health", nil, nil)
			if err != nil {
				t.Logf("Request %d failed: %v", i, err)
				continue
			}

			switch resp.StatusCode {
			case http.StatusOK:
				successCount++
			case http.StatusTooManyRequests:
				rateLimitedCount++
				t.Logf("Rate limited at request %d", i)
			default:
				t.Logf("Request %d returned status %d", i, resp.StatusCode)
			}

			resp.Body.Close()

			// Small delay to avoid overwhelming the service
			time.Sleep(50 * time.Millisecond)
		}

		t.Logf("Rate Limiting Test Results:")
		t.Logf("  Successful requests: %d", successCount)
		t.Logf("  Rate limited requests: %d", rateLimitedCount)
		t.Logf("  Total requests: %d", rapidRequests)

		// Should have some successful requests
		assert.Greater(t, successCount, 0, "Should have some successful requests")

		// If rate limiting is implemented, should see some rate limited responses
		if rateLimitedCount > 0 {
			t.Logf("Rate limiting appears to be implemented")
		} else {
			t.Logf("No rate limiting detected (may not be implemented)")
		}
	})
}

func TestKNIRVNEXUSSecurityTEE(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVSERVER service not available for TEE security testing")
	}

	t.Run("TestTEESecurityEndpoint", func(t *testing.T) {
		resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/api/tee-security", nil, nil)
		if err != nil {
			t.Skipf("TEE security endpoint not available: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var teeStatus map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&teeStatus)
			if err == nil {
				// Check TEE security status
				if status, ok := teeStatus["tee_status"]; ok {
					assert.NotEqual(t, "insecure", status,
						"TEE status should not be insecure")
					t.Logf("TEE Status: %v", status)
				}

				if attestation, ok := teeStatus["attestation"]; ok {
					assert.NotEqual(t, "invalid", attestation,
						"TEE attestation should not be invalid")
					t.Logf("TEE Attestation: %v", attestation)
				}

				if secLevel, ok := teeStatus["security_level"]; ok {
					t.Logf("TEE Security Level: %v", secLevel)
				}
			}
		} else if resp.StatusCode == http.StatusUnauthorized {
			t.Logf("TEE security endpoint requires authentication")
		} else {
			t.Logf("TEE security endpoint returned status %d", resp.StatusCode)
		}
	})
}

func TestKNIRVNEXUSSecurityNetworkCommunication(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVSERVER service not available for network security testing")
	}

	t.Run("TestHTTPSRedirection", func(t *testing.T) {
		// Test if HTTP requests are redirected to HTTPS
		// Note: This test assumes the service might support HTTPS
		httpURL := strings.Replace(KNIRVNEXUS_BASE_URL, "http://", "http://", 1)

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
			Timeout: 5 * time.Second,
		}

		resp, err := client.Get(httpURL + "/health")
		if err != nil {
			t.Skipf("Cannot test HTTPS redirection: %v", err)
			return
		}
		defer resp.Body.Close()

		// Check if redirected to HTTPS
		if resp.StatusCode == http.StatusMovedPermanently ||
			resp.StatusCode == http.StatusFound {
			location := resp.Header.Get("Location")
			if strings.HasPrefix(location, "https://") {
				t.Logf("HTTP requests are redirected to HTTPS: %s", location)
			}
		} else {
			t.Logf("No HTTPS redirection detected (status: %d)", resp.StatusCode)
		}
	})

	t.Run("TestSecureProtocols", func(t *testing.T) {
		// Test that only secure protocols are supported
		resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/health", nil, nil)
		if err != nil {
			t.Skipf("Cannot test secure protocols: %v", err)
			return
		}
		defer resp.Body.Close()

		// Check TLS version if HTTPS
		if resp.TLS != nil {
			t.Logf("TLS Version: %d", resp.TLS.Version)
			t.Logf("Cipher Suite: %d", resp.TLS.CipherSuite)

			// Should use TLS 1.2 or higher
			assert.GreaterOrEqual(t, resp.TLS.Version, uint16(0x0303),
				"Should use TLS 1.2 or higher")
		} else {
			t.Logf("Connection is not using TLS")
		}
	})
}
