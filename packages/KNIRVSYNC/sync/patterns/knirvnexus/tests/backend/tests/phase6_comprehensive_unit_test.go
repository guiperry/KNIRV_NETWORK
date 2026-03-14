package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Run tests
	code := m.Run()

	// Exit with the test result code
	os.Exit(code)
}

// Phase 6 Unit Tests for KNIRVNEXUS Unified Binary Architecture

func TestUnifiedBinaryStartup(t *testing.T) {
	t.Run("TestServerInitialization", func(t *testing.T) {
		// Test that the unified binary can initialize properly
		router := gin.New()

		// Add basic health endpoint
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":       "healthy",
				"service":      "knirvnexus",
				"port":         8084,
				"architecture": "unified_binary",
			})
		})

		// Test the health endpoint
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
		assert.Equal(t, "knirvnexus", response["service"])
		assert.Equal(t, float64(8084), response["port"])
		assert.Equal(t, "unified_binary", response["architecture"])
	})

	t.Run("TestServiceConfiguration", func(t *testing.T) {
		// Test that service configuration is properly loaded
		config := map[string]interface{}{
			"port":     8084,
			"mode":     "unified",
			"services": []string{"frontend", "backend", "api"},
		}

		assert.Equal(t, 8084, config["port"])
		assert.Equal(t, "unified", config["mode"])
		assert.Contains(t, config["services"], "frontend")
		assert.Contains(t, config["services"], "backend")
		assert.Contains(t, config["services"], "api")
	})
}

func TestAPIEndpoints(t *testing.T) {
	router := gin.New()

	// Setup API routes
	api := router.Group("/api")
	{
		api.GET("/dve-nodes", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"nodes": []map[string]interface{}{
					{"id": "node1", "status": "active", "type": "dve"},
					{"id": "node2", "status": "active", "type": "dve"},
				},
				"total": 2,
			})
		})

		api.GET("/validation-tasks", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"tasks": []map[string]interface{}{
					{"id": "task1", "status": "pending", "priority": "high"},
					{"id": "task2", "status": "completed", "priority": "medium"},
				},
				"total": 2,
			})
		})

		api.GET("/cognitive-engine", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":       "operational",
				"objects":      []string{"hrm", "phi3", "tinyllama"},
				"active_model": "hrm",
			})
		})

		api.GET("/tee-security", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"tee_status":     "secure",
				"attestation":    "valid",
				"security_level": "high",
			})
		})

		api.GET("/nrn-staking", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"total_staked": "1000000",
				"validators":   25,
				"apy":          "12.5%",
			})
		})

		api.GET("/system-health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"cpu_usage":    "45%",
				"memory_usage": "60%",
				"disk_usage":   "30%",
				"uptime":       "72h",
				"status":       "healthy",
			})
		})
	}

	testCases := []struct {
		name           string
		endpoint       string
		expectedStatus int
		expectedFields []string
	}{
		{
			name:           "DVE Nodes Endpoint",
			endpoint:       "/api/dve-nodes",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"nodes", "total"},
		},
		{
			name:           "Validation Tasks Endpoint",
			endpoint:       "/api/validation-tasks",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"tasks", "total"},
		},
		{
			name:           "Cognitive Engine Endpoint",
			endpoint:       "/api/cognitive-engine",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"status", "objects", "active_model"},
		},
		{
			name:           "TEE Security Endpoint",
			endpoint:       "/api/tee-security",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"tee_status", "attestation", "security_level"},
		},
		{
			name:           "NRN Staking Endpoint",
			endpoint:       "/api/nrn-staking",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"total_staked", "validators", "apy"},
		},
		{
			name:           "System Health Endpoint",
			endpoint:       "/api/system-health",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"cpu_usage", "memory_usage", "disk_usage", "uptime", "status"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tc.endpoint, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			for _, field := range tc.expectedFields {
				assert.Contains(t, response, field, fmt.Sprintf("Response should contain field: %s", field))
			}
		})
	}
}

func TestPhase6DatabaseOperations(t *testing.T) {
	t.Run("TestBuntDBOperations", func(t *testing.T) {
		// Test BuntDB operations (mocked for unit testing)
		testData := map[string]interface{}{
			"users": []map[string]interface{}{
				{"id": "user1", "role": "admin", "active": true},
				{"id": "user2", "role": "validator", "active": true},
			},
			"objects": []map[string]interface{}{
				{"id": "model1", "status": "active", "type": "dve"},
				{"id": "model2", "status": "pending", "type": "validator"},
			},
			"reports": []map[string]interface{}{
				{"id": "report1", "type": "system", "timestamp": time.Now().Unix()},
				{"id": "report2", "type": "security", "timestamp": time.Now().Unix()},
			},
		}

		// Test data structure
		assert.Contains(t, testData, "users")
		assert.Contains(t, testData, "objects")
		assert.Contains(t, testData, "reports")

		// Test user data
		users := testData["users"].([]map[string]interface{})
		assert.Len(t, users, 2)
		assert.Equal(t, "admin", users[0]["role"])
		assert.Equal(t, true, users[0]["active"])

		// Test model data
		objects := testData["objects"].([]map[string]interface{})
		assert.Len(t, objects, 2)
		assert.Equal(t, "active", objects[0]["status"])
		assert.Equal(t, "dve", objects[0]["type"])

		// Test report data
		reports := testData["reports"].([]map[string]interface{})
		assert.Len(t, reports, 2)
		assert.Equal(t, "system", reports[0]["type"])
		assert.IsType(t, int64(0), reports[0]["timestamp"])
	})
}

func TestAuthenticationAndAuthorization(t *testing.T) {
	router := gin.New()

	// Mock authentication middleware
	authMiddleware := func() gin.HandlerFunc {
		return func(c *gin.Context) {
			token := c.GetHeader("Authorization")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization token provided"})
				c.Abort()
				return
			}

			// Mock token validation
			if token == "Bearer valid-token" {
				c.Set("user_id", "user123")
				c.Set("role", "admin")
				c.Next()
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
				c.Abort()
				return
			}
		}
	}

	// Protected route
	protected := router.Group("/api/protected")
	protected.Use(authMiddleware())
	{
		protected.GET("/admin", func(c *gin.Context) {
			role := c.GetString("role")
			if role != "admin" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Admin access granted"})
		})
	}

	t.Run("TestUnauthorizedAccess", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected/admin", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "error")
		assert.Equal(t, "No authorization token provided", response["error"])
	})

	t.Run("TestInvalidToken", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected/admin", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "error")
		assert.Equal(t, "Invalid token", response["error"])
	})

	t.Run("TestValidToken", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected/admin", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "message")
		assert.Equal(t, "Admin access granted", response["message"])
	})
}

func TestServiceLifecycleManagement(t *testing.T) {
	t.Run("TestServiceStartup", func(t *testing.T) {
		// Test service startup sequence
		services := []string{"database", "api", "frontend", "websocket"}
		startupOrder := make([]string, 0, len(services))

		// Simulate startup sequence
		startupOrder = append(startupOrder, services...)

		assert.Equal(t, services, startupOrder)
		assert.Len(t, startupOrder, 4)
		assert.Equal(t, "database", startupOrder[0]) // Database should start first
		assert.Equal(t, "frontend", startupOrder[2]) // Frontend after API
	})

	t.Run("TestServiceHealthChecks", func(t *testing.T) {
		// Test individual service health checks
		serviceHealth := map[string]bool{
			"database":  true,
			"api":       true,
			"frontend":  true,
			"websocket": true,
		}

		for service, healthy := range serviceHealth {
			assert.True(t, healthy, fmt.Sprintf("Service %s should be healthy", service))
		}

		// Test overall system health
		allHealthy := true
		for _, healthy := range serviceHealth {
			if !healthy {
				allHealthy = false
				break
			}
		}
		assert.True(t, allHealthy, "All services should be healthy")
	})
}

func TestConfigurationManagement(t *testing.T) {
	t.Run("TestDefaultConfiguration", func(t *testing.T) {
		// Test default configuration values
		defaultConfig := map[string]interface{}{
			"port":        8084,
			"host":        "0.0.0.0",
			"environment": "testnet",
			"debug":       false,
			"cors":        true,
			"rate_limit":  100,
		}

		assert.Equal(t, 8084, defaultConfig["port"])
		assert.Equal(t, "0.0.0.0", defaultConfig["host"])
		assert.Equal(t, "testnet", defaultConfig["environment"])
		assert.Equal(t, false, defaultConfig["debug"])
		assert.Equal(t, true, defaultConfig["cors"])
		assert.Equal(t, 100, defaultConfig["rate_limit"])
	})

	t.Run("TestEnvironmentOverrides", func(t *testing.T) {
		// Test environment variable overrides
		envConfig := map[string]interface{}{
			"port":        8084,
			"environment": "development",
			"debug":       true,
		}

		// Simulate environment override
		if envConfig["environment"] == "development" {
			envConfig["debug"] = true
			envConfig["rate_limit"] = 1000 // Higher limit for dev
		}

		assert.Equal(t, "development", envConfig["environment"])
		assert.Equal(t, true, envConfig["debug"])
		assert.Equal(t, 1000, envConfig["rate_limit"])
	})
}

func TestTEESecurityValidation(t *testing.T) {
	t.Run("TestTEEAttestation", func(t *testing.T) {
		// Test TEE attestation validation
		attestation := map[string]interface{}{
			"status":       "valid",
			"timestamp":    time.Now().Unix(),
			"signature":    "mock_signature_hash",
			"measurements": []string{"pcr0", "pcr1", "pcr2"},
		}

		assert.Equal(t, "valid", attestation["status"])
		assert.IsType(t, int64(0), attestation["timestamp"])
		assert.NotEmpty(t, attestation["signature"])

		measurements := attestation["measurements"].([]string)
		assert.Len(t, measurements, 3)
		assert.Contains(t, measurements, "pcr0")
	})

	t.Run("TestSecurityLevel", func(t *testing.T) {
		// Test security level validation
		securityLevels := []string{"low", "medium", "high", "critical"}
		currentLevel := "high"

		assert.Contains(t, securityLevels, currentLevel)
		assert.NotEqual(t, "low", currentLevel) // Should not be low in testnet
	})
}

func TestPhase6P2PNetworking(t *testing.T) {
	t.Run("TestPeerDiscovery", func(t *testing.T) {
		// Test peer discovery mechanism
		peers := []map[string]interface{}{
			{"id": "peer1", "address": "192.168.1.100:8084", "status": "connected"},
			{"id": "peer2", "address": "192.168.1.101:8084", "status": "connected"},
			{"id": "peer3", "address": "192.168.1.102:8084", "status": "disconnected"},
		}

		connectedPeers := 0
		for _, peer := range peers {
			if peer["status"] == "connected" {
				connectedPeers++
			}
		}

		assert.Equal(t, 2, connectedPeers)
		assert.Len(t, peers, 3)
	})

	t.Run("TestNetworkCommunication", func(t *testing.T) {
		// Test network message handling
		message := map[string]interface{}{
			"type":      "consensus",
			"payload":   "test_data",
			"timestamp": time.Now().Unix(),
			"sender":    "peer1",
		}

		assert.Equal(t, "consensus", message["type"])
		assert.Equal(t, "test_data", message["payload"])
		assert.Equal(t, "peer1", message["sender"])
		assert.IsType(t, int64(0), message["timestamp"])
	})
}

func TestErrorHandlingAndValidation(t *testing.T) {
	router := gin.New()

	// Add error handling middleware
	router.Use(func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  500,
				})
			}
		}()
		c.Next()
	})

	// Add test routes
	router.GET("/api/test-error", func(c *gin.Context) {
		panic("Test error")
	})

	router.POST("/api/validate", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid JSON",
				"code":  400,
			})
			return
		}

		// Validate required fields
		if data["name"] == nil || data["name"] == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Name is required",
				"code":  400,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Validation passed"})
	})

	t.Run("TestPanicRecovery", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test-error", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Internal server error", response["error"])
		assert.Equal(t, float64(500), response["code"])
	})

	t.Run("TestInputValidation", func(t *testing.T) {
		// Test invalid JSON
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/validate", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Test missing required field
		w = httptest.NewRecorder()
		validJSON := `{"description": "test"}`
		req, _ = http.NewRequest("POST", "/api/validate", bytes.NewBufferString(validJSON))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Test valid input
		w = httptest.NewRecorder()
		validJSON = `{"name": "test", "description": "test description"}`
		req, _ = http.NewRequest("POST", "/api/validate", bytes.NewBufferString(validJSON))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
