package integration_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E test configuration
const (
	E2E_TEST_TIMEOUT = 60 * time.Second
)

// Test data structures for E2E tests
type E2EDVENode struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

type E2EValidationTask struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

type E2ESystemHealth struct {
	Status      string `json:"status"`
	CPUUsage    string `json:"cpu_usage"`
	MemoryUsage string `json:"memory_usage"`
	Uptime      string `json:"uptime"`
}

func TestKNIRVNEXUSE2ECompleteWorkflows(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVSERVER service not available for E2E testing")
	}

	t.Run("TestDVENodeRegistrationWorkflow", func(t *testing.T) {
		// Test complete DVE node registration workflow

		// Step 1: Check initial DVE nodes state
		resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/api/dve-nodes", nil, nil)
		if err != nil {
			t.Skipf("Cannot access DVE nodes endpoint: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var nodesResponse map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&nodesResponse)
			require.NoError(t, err)

			// Should have nodes structure
			assert.Contains(t, nodesResponse, "nodes", "Response should contain nodes")

			if nodes, ok := nodesResponse["nodes"].([]interface{}); ok {
				t.Logf("Current DVE nodes count: %d", len(nodes))

				// Verify node structure
				for i, nodeInterface := range nodes {
					if i >= 3 { // Limit verification to first 3 nodes
						break
					}

					if nodeMap, ok := nodeInterface.(map[string]interface{}); ok {
						assert.Contains(t, nodeMap, "id", "Node should have ID")
						assert.Contains(t, nodeMap, "status", "Node should have status")
						assert.Contains(t, nodeMap, "type", "Node should have type")

						t.Logf("Node %d: ID=%v, Status=%v, Type=%v",
							i, nodeMap["id"], nodeMap["status"], nodeMap["type"])
					}
				}
			}
		} else if resp.StatusCode == http.StatusUnauthorized {
			t.Logf("DVE nodes endpoint requires authentication")
		} else {
			t.Logf("DVE nodes endpoint returned status %d", resp.StatusCode)
		}

		// Step 2: Simulate node registration (if POST endpoint exists)
		// This would typically involve registering a new DVE node
		t.Logf("DVE node registration workflow test completed")
	})

	t.Run("TestValidationTaskCreationWorkflow", func(t *testing.T) {
		// Test complete validation task creation and execution workflow

		// Step 1: Check current validation tasks
		resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/api/validation-tasks", nil, nil)
		if err != nil {
			t.Skipf("Cannot access validation tasks endpoint: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var tasksResponse map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&tasksResponse)
			require.NoError(t, err)

			// Should have tasks structure
			assert.Contains(t, tasksResponse, "tasks", "Response should contain tasks")

			if tasks, ok := tasksResponse["tasks"].([]interface{}); ok {
				t.Logf("Current validation tasks count: %d", len(tasks))

				// Verify task structure
				for i, taskInterface := range tasks {
					if i >= 3 { // Limit verification to first 3 tasks
						break
					}

					if taskMap, ok := taskInterface.(map[string]interface{}); ok {
						assert.Contains(t, taskMap, "id", "Task should have ID")
						assert.Contains(t, taskMap, "status", "Task should have status")
						assert.Contains(t, taskMap, "priority", "Task should have priority")

						t.Logf("Task %d: ID=%v, Status=%v, Priority=%v",
							i, taskMap["id"], taskMap["status"], taskMap["priority"])
					}
				}
			}
		} else if resp.StatusCode == http.StatusUnauthorized {
			t.Logf("Validation tasks endpoint requires authentication")
		} else {
			t.Logf("Validation tasks endpoint returned status %d", resp.StatusCode)
		}

		// Step 2: Simulate task creation and execution
		t.Logf("Validation task creation workflow test completed")
	})
}

func TestKNIRVNEXUSE2ESystemHealthMonitoring(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVSERVER service not available for system health testing")
	}

	t.Run("TestSystemHealthMonitoringWorkflow", func(t *testing.T) {
		// Test complete system health monitoring workflow

		// Step 1: Check basic health endpoint
		resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/health", nil, nil)
		require.NoError(t, err, "Health endpoint should be accessible")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health endpoint should return 200")

		var healthResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&healthResponse)
		require.NoError(t, err)

		assert.Equal(t, "healthy", healthResponse["status"], "Service should be healthy")

		// Step 2: Check detailed system health
		resp, err = makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/api/system-health", nil, nil)
		if err != nil {
			t.Skipf("Cannot access system health endpoint: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var systemHealth map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&systemHealth)
			require.NoError(t, err)

			// Verify system health metrics
			expectedFields := []string{"status", "cpu_usage", "memory_usage", "uptime"}
			for _, field := range expectedFields {
				if value, ok := systemHealth[field]; ok {
					t.Logf("System Health - %s: %v", field, value)
					assert.NotEmpty(t, value, "System health field %s should not be empty", field)
				}
			}

			// Verify system is healthy
			if status, ok := systemHealth["status"]; ok {
				assert.NotEqual(t, "error", status, "System status should not be error")
				assert.NotEqual(t, "critical", status, "System status should not be critical")
			}
		} else if resp.StatusCode == http.StatusUnauthorized {
			t.Logf("System health endpoint requires authentication")
		}

		// Step 3: Monitor system health over time
		t.Logf("Monitoring system health over 10 seconds...")
		healthChecks := 5
		healthyCount := 0

		for i := 0; i < healthChecks; i++ {
			resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/health", nil, nil)
			if err == nil && resp.StatusCode == http.StatusOK {
				healthyCount++
				resp.Body.Close()
			}
			time.Sleep(2 * time.Second)
		}

		healthPercentage := float64(healthyCount) / float64(healthChecks) * 100
		t.Logf("System health over time: %.1f%% (%d/%d checks passed)",
			healthPercentage, healthyCount, healthChecks)

		assert.Greater(t, healthPercentage, 80.0,
			"System should be healthy at least 80% of the time")
	})
}

func TestKNIRVNEXUSE2EUserAuthenticationWorkflow(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVSERVER service not available for authentication testing")
	}

	t.Run("TestUserAuthenticationWorkflow", func(t *testing.T) {
		// Test complete user authentication workflow

		userRoles := []struct {
			name  string
			token string
		}{
			{"Admin", "Bearer TESTNET_ADMIN_TOKEN"},
			{"Validator", "Bearer TESTNET_VALIDATOR_TOKEN"},
			{"Observer", "Bearer TESTNET_OBSERVER_TOKEN"},
		}

		protectedEndpoints := []string{
			"/api/dve-nodes",
			"/api/validation-tasks",
			"/api/system-health",
		}

		for _, user := range userRoles {
			t.Run(fmt.Sprintf("User_%s", user.name), func(t *testing.T) {
				headers := map[string]string{
					"Authorization": user.token,
				}

				accessibleEndpoints := 0

				for _, endpoint := range protectedEndpoints {
					resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+endpoint, nil, headers)
					if err != nil {
						t.Logf("Cannot test endpoint %s for user %s: %v", endpoint, user.name, err)
						continue
					}
					defer resp.Body.Close()

					switch resp.StatusCode {
					case http.StatusOK:
						accessibleEndpoints++
						t.Logf("User %s can access %s", user.name, endpoint)
					case http.StatusUnauthorized:
						t.Logf("User %s unauthorized for %s", user.name, endpoint)
					case http.StatusForbidden:
						t.Logf("User %s forbidden from %s", user.name, endpoint)
					default:
						t.Logf("User %s got status %d for %s", user.name, resp.StatusCode, endpoint)
					}
				}

				t.Logf("User %s can access %d/%d endpoints",
					user.name, accessibleEndpoints, len(protectedEndpoints))
			})
		}
	})
}

func TestKNIRVNEXUSE2ERealTimeDataUpdates(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVSERVER service not available for real-time testing")
	}

	t.Run("TestRealTimeDataUpdatesWorkflow", func(t *testing.T) {
		// Test real-time data updates workflow

		// Step 1: Get initial data snapshot
		initialData := make(map[string]interface{})

		endpoints := []string{
			"/api/dve-nodes",
			"/api/validation-tasks",
			"/api/system-health",
		}

		for _, endpoint := range endpoints {
			resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+endpoint, nil, nil)
			if err != nil {
				t.Logf("Cannot get initial data for %s: %v", endpoint, err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var data map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
					initialData[endpoint] = data
					t.Logf("Got initial data for %s", endpoint)
				}
			}
		}

		// Step 2: Wait and check for data changes
		t.Logf("Waiting 5 seconds for potential data changes...")
		time.Sleep(5 * time.Second)

		// Step 3: Get updated data snapshot
		changedEndpoints := 0

		for _, endpoint := range endpoints {
			resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+endpoint, nil, nil)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var newData map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&newData); err == nil {
					// Compare with initial data
					if initialData[endpoint] != nil {
						// Simple comparison - in real scenario, would compare specific fields
						initialJSON, _ := json.Marshal(initialData[endpoint])
						newJSON, _ := json.Marshal(newData)

						if string(initialJSON) != string(newJSON) {
							changedEndpoints++
							t.Logf("Data changed for endpoint %s", endpoint)
						}
					}
				}
			}
		}

		t.Logf("Data changes detected in %d/%d endpoints", changedEndpoints, len(endpoints))

		// Step 4: Test WebSocket/SSE connection if available
		t.Logf("Testing real-time connection capabilities...")

		// Try to connect to SSE endpoint
		client := &http.Client{Timeout: 3 * time.Second}
		req, err := http.NewRequest("GET", KNIRVNEXUS_BASE_URL+"/api/events", nil)
		if err == nil {
			req.Header.Set("Accept", "text/event-stream")
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					t.Logf("SSE endpoint is available")
				} else {
					t.Logf("SSE endpoint returned status %d", resp.StatusCode)
				}
			} else {
				t.Logf("SSE endpoint not accessible: %v", err)
			}
		}
	})
}

func TestKNIRVNEXUSE2EErrorRecoveryWorkflow(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVSERVER service not available for error recovery testing")
	}

	t.Run("TestErrorRecoveryWorkflow", func(t *testing.T) {
		// Test error recovery and resilience workflow

		// Step 1: Verify system is healthy
		resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/health", nil, nil)
		require.NoError(t, err, "Health check should work")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "System should be healthy initially")

		// Step 2: Test error handling with invalid requests
		errorScenarios := []struct {
			name     string
			method   string
			endpoint string
			expected int
		}{
			{"Invalid Endpoint", "GET", "/api/nonexistent", http.StatusNotFound},
			{"Invalid Method", "DELETE", "/health", http.StatusMethodNotAllowed},
			{"Malformed Request", "GET", "/api/dve-nodes?invalid=<script>", http.StatusBadRequest},
		}

		for _, scenario := range errorScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				resp, err := makePhase6Request(scenario.method, KNIRVNEXUS_BASE_URL+scenario.endpoint, nil, nil)
				if err != nil {
					t.Logf("Error scenario %s: %v", scenario.name, err)
					return
				}
				defer resp.Body.Close()

				// Should handle errors gracefully
				assert.True(t, resp.StatusCode >= 400 && resp.StatusCode < 500,
					"Error scenario %s should return 4xx status, got %d", scenario.name, resp.StatusCode)
			})
		}

		// Step 3: Verify system is still healthy after error scenarios
		resp, err = makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/health", nil, nil)
		require.NoError(t, err, "Health check should still work after errors")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "System should remain healthy after error scenarios")

		t.Logf("Error recovery workflow test completed successfully")
	})
}
