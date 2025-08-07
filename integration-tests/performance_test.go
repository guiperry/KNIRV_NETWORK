package integration_tests

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type PerformanceMetrics struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	MinLatency         time.Duration `json:"min_latency"`
	MaxLatency         time.Duration `json:"max_latency"`
	RequestsPerSecond  float64       `json:"requests_per_second"`
	ErrorRate          float64       `json:"error_rate"`
}

type LoadTestConfig struct {
	ConcurrentUsers int           `json:"concurrent_users"`
	TestDuration    time.Duration `json:"test_duration"`
	RequestsPerUser int           `json:"requests_per_user"`
	RampUpTime      time.Duration `json:"ramp_up_time"`
}

type PerformanceTester struct {
	suite   *IntegrationTestSuite
	metrics *PerformanceMetrics
	mutex   sync.RWMutex
}

func NewPerformanceTester(suite *IntegrationTestSuite) *PerformanceTester {
	return &PerformanceTester{
		suite: suite,
		metrics: &PerformanceMetrics{
			MinLatency: time.Hour, // Initialize to high value
		},
	}
}

func (pt *PerformanceTester) recordRequest(latency time.Duration, success bool) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	atomic.AddInt64(&pt.metrics.TotalRequests, 1)

	if success {
		atomic.AddInt64(&pt.metrics.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&pt.metrics.FailedRequests, 1)
	}

	// Update latency metrics
	if latency < pt.metrics.MinLatency {
		pt.metrics.MinLatency = latency
	}
	if latency > pt.metrics.MaxLatency {
		pt.metrics.MaxLatency = latency
	}
}

func (pt *PerformanceTester) calculateFinalMetrics(testDuration time.Duration) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	if pt.metrics.TotalRequests > 0 {
		pt.metrics.RequestsPerSecond = float64(pt.metrics.TotalRequests) / testDuration.Seconds()
		pt.metrics.ErrorRate = float64(pt.metrics.FailedRequests) / float64(pt.metrics.TotalRequests) * 100
	}
}

// Test KNIRVCHAIN Performance
func (pt *PerformanceTester) TestKNIRVCHAINPerformance(t *testing.T) {
	config := LoadTestConfig{
		ConcurrentUsers: 10,
		TestDuration:    30 * time.Second,
		RequestsPerUser: 50,
		RampUpTime:      5 * time.Second,
	}

	t.Run("TransactionThroughput", func(t *testing.T) {
		pt.resetMetrics()

		ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
		defer cancel()

		var wg sync.WaitGroup
		startTime := time.Now()

		// Ramp up users gradually
		userDelay := config.RampUpTime / time.Duration(config.ConcurrentUsers)

		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				// Stagger user start times
				time.Sleep(time.Duration(userID) * userDelay)

				for j := 0; j < config.RequestsPerUser; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						pt.sendTransaction(t, userID, j)
					}
				}
			}(i)
		}

		wg.Wait()
		testDuration := time.Since(startTime)
		pt.calculateFinalMetrics(testDuration)

		// Assertions
		assert.Greater(t, pt.metrics.SuccessfulRequests, int64(0), "Should have successful requests")
		assert.Less(t, pt.metrics.ErrorRate, 5.0, "Error rate should be less than 5%")
		assert.Greater(t, pt.metrics.RequestsPerSecond, 1.0, "Should process at least 1 request per second")

		t.Logf("KNIRVCHAIN Performance Metrics: %+v", pt.metrics)
	})

	t.Run("LLMRegistrationLoad", func(t *testing.T) {
		pt.resetMetrics()

		ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
		defer cancel()

		var wg sync.WaitGroup
		startTime := time.Now()

		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				for j := 0; j < 5; j++ { // Fewer LLM registrations due to complexity
					select {
					case <-ctx.Done():
						return
					default:
						pt.registerLLM(t, userID, j)
					}
				}
			}(i)
		}

		wg.Wait()
		testDuration := time.Since(startTime)
		pt.calculateFinalMetrics(testDuration)

		assert.Greater(t, pt.metrics.SuccessfulRequests, int64(0))
		assert.Less(t, pt.metrics.ErrorRate, 10.0, "LLM registration error rate should be less than 10%")

		t.Logf("LLM Registration Load Test Metrics: %+v", pt.metrics)
	})
}

// Test KNIRVGRAPH Performance
func (pt *PerformanceTester) TestKNIRVGRAPHPerformance(t *testing.T) {
	config := LoadTestConfig{
		ConcurrentUsers: 15,
		TestDuration:    30 * time.Second,
		RequestsPerUser: 30,
		RampUpTime:      3 * time.Second,
	}

	t.Run("NRVCreationThroughput", func(t *testing.T) {
		pt.resetMetrics()

		ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
		defer cancel()

		var wg sync.WaitGroup
		startTime := time.Now()

		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				for j := 0; j < config.RequestsPerUser; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						if j%2 == 0 {
							pt.createErrorNode(t, userID, j)
						} else {
							pt.createSkillNode(t, userID, j)
						}
					}
				}
			}(i)
		}

		wg.Wait()
		testDuration := time.Since(startTime)
		pt.calculateFinalMetrics(testDuration)

		assert.Greater(t, pt.metrics.SuccessfulRequests, int64(0))
		assert.Less(t, pt.metrics.ErrorRate, 5.0)
		assert.Greater(t, pt.metrics.RequestsPerSecond, 2.0)

		t.Logf("KNIRVGRAPH NRV Creation Metrics: %+v", pt.metrics)
	})

	t.Run("VectorResolutionLoad", func(t *testing.T) {
		// First create some vectors to resolve
		for i := 0; i < 10; i++ {
			vectorData := map[string]interface{}{
				"target_hash": fmt.Sprintf("load_test_hash_%d", i),
				"coordinates": []float64{float64(i), float64(i * 2), float64(i * 3)},
				"metadata": map[string]interface{}{
					"type": "load_test_vector",
				},
			}
			pt.suite.makeRequest("POST", pt.suite.knirvgraphURL+"/nrv/vectors", vectorData)
		}

		pt.resetMetrics()

		ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
		defer cancel()

		var wg sync.WaitGroup
		startTime := time.Now()

		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				for j := 0; j < config.RequestsPerUser; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						pt.resolveVector(t, j%10) // Cycle through the 10 vectors we created
					}
				}
			}(i)
		}

		wg.Wait()
		testDuration := time.Since(startTime)
		pt.calculateFinalMetrics(testDuration)

		assert.Greater(t, pt.metrics.SuccessfulRequests, int64(0))
		assert.Less(t, pt.metrics.ErrorRate, 3.0)
		assert.Greater(t, pt.metrics.RequestsPerSecond, 5.0, "Vector resolution should be fast")

		t.Logf("Vector Resolution Load Test Metrics: %+v", pt.metrics)
	})
}

// Test KNIRVNEXUS Performance
func (pt *PerformanceTester) TestKNIRVNEXUSPerformance(t *testing.T) {
	config := LoadTestConfig{
		ConcurrentUsers: 8,
		TestDuration:    25 * time.Second,
		RequestsPerUser: 20,
		RampUpTime:      4 * time.Second,
	}

	t.Run("AgentManagementLoad", func(t *testing.T) {
		pt.resetMetrics()

		ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
		defer cancel()

		var wg sync.WaitGroup
		startTime := time.Now()

		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				for j := 0; j < config.RequestsPerUser; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						switch j % 3 {
						case 0:
							pt.createAgent(t, userID, j)
						case 1:
							pt.listAgents(t)
						case 2:
							pt.getAgentStatus(t, userID)
						}
					}
				}
			}(i)
		}

		wg.Wait()
		testDuration := time.Since(startTime)
		pt.calculateFinalMetrics(testDuration)

		assert.Greater(t, pt.metrics.SuccessfulRequests, int64(0))
		assert.Less(t, pt.metrics.ErrorRate, 8.0)

		t.Logf("KNIRVNEXUS Agent Management Metrics: %+v", pt.metrics)
	})
}

// Helper methods for load testing
func (pt *PerformanceTester) sendTransaction(_ *testing.T, userID, requestID int) {
	start := time.Now()

	txData := map[string]interface{}{
		"from":   fmt.Sprintf("user_%d", userID),
		"to":     fmt.Sprintf("recipient_%d", requestID%5),
		"amount": "100000",
		"type":   "transfer",
	}

	_, err := pt.suite.makeRequest("POST", pt.suite.knirvchainURL+"/send_txn", txData)

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func (pt *PerformanceTester) registerLLM(_ *testing.T, userID, requestID int) {
	start := time.Now()

	llmData := map[string]interface{}{
		"name":             fmt.Sprintf("LoadTestLLM_%d_%d", userID, requestID),
		"version":          "1.0.0",
		"capabilities":     []string{"load-testing"},
		"model_data":       "bG9hZCB0ZXN0IGRhdGE=", // base64 encoded "load test data"
		"registration_fee": "1000000",
		"usage_fee":        "100000",
	}

	_, err := pt.suite.makeRequest("POST", pt.suite.knirvchainURL+"/llm/register", llmData)

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func (pt *PerformanceTester) createErrorNode(_ *testing.T, userID, requestID int) {
	start := time.Now()

	errorData := map[string]interface{}{
		"error_type":  fmt.Sprintf("load_test_error_%d", requestID%5),
		"description": fmt.Sprintf("Load test error from user %d request %d", userID, requestID),
		"context": map[string]interface{}{
			"user_id":    userID,
			"request_id": requestID,
		},
		"severity": (requestID % 5) + 1,
	}

	_, err := pt.suite.makeRequest("POST", pt.suite.knirvgraphURL+"/nrv/errors", errorData)

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func (pt *PerformanceTester) createSkillNode(_ *testing.T, userID, requestID int) {
	start := time.Now()

	skillData := map[string]interface{}{
		"skill_type":   fmt.Sprintf("load_test_skill_%d", requestID%3),
		"capabilities": []string{fmt.Sprintf("capability_%d", userID)},
		"requirements": map[string]interface{}{
			"min_confidence": 0.7 + float64(requestID%3)*0.1,
			"max_latency":    fmt.Sprintf("%ds", 3+requestID%5),
		},
	}

	_, err := pt.suite.makeRequest("POST", pt.suite.knirvgraphURL+"/nrv/skills", skillData)

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func (pt *PerformanceTester) resolveVector(_ *testing.T, vectorIndex int) {
	start := time.Now()

	_, err := pt.suite.makeRequest("GET", pt.suite.knirvgraphURL+fmt.Sprintf("/nrv/resolve/load_test_hash_%d", vectorIndex), nil)

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func (pt *PerformanceTester) createAgent(_ *testing.T, userID, requestID int) {
	start := time.Now()

	agentData := map[string]interface{}{
		"name":         fmt.Sprintf("LoadTestAgent_%d_%d", userID, requestID),
		"description":  "Load test agent",
		"type":         "go_plugin",
		"capabilities": []string{"load-testing"},
	}

	_, err := pt.suite.makeRequest("POST", pt.suite.knirvnexusAPIGatewayURL+"/api/v1/agents", agentData)

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func (pt *PerformanceTester) listAgents(_ *testing.T) {
	start := time.Now()

	_, err := pt.suite.makeRequest("GET", pt.suite.knirvnexusAPIGatewayURL+"/api/v1/agents", nil)

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func (pt *PerformanceTester) getAgentStatus(_ *testing.T, userID int) {
	start := time.Now()

	_, err := pt.suite.makeRequest("GET", pt.suite.knirvnexusAPIGatewayURL+fmt.Sprintf("/api/v1/agents/status?user_id=%d", userID), nil)

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func (pt *PerformanceTester) resetMetrics() {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	pt.metrics = &PerformanceMetrics{
		MinLatency: time.Hour,
	}
}

// Month 12 Additional Performance Tests

// Test Gateway Performance under Load
func (pt *PerformanceTester) TestGatewayPerformance(t *testing.T) {
	config := LoadTestConfig{
		ConcurrentUsers: 20,
		TestDuration:    45 * time.Second,
		RequestsPerUser: 100,
		RampUpTime:      5 * time.Second,
	}

	t.Run("GatewayRoutingLoad", func(t *testing.T) {
		pt.resetMetrics()

		ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
		defer cancel()

		var wg sync.WaitGroup
		startTime := time.Now()

		services := []string{"knirvchain", "knirvgraph", "knirvnexus", "knirvroot", "knirvrouter"}

		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				for j := 0; j < config.RequestsPerUser; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						service := services[j%len(services)]
						pt.testGatewayRouting(t, service, userID, j)
					}
				}
			}(i)
		}

		wg.Wait()
		testDuration := time.Since(startTime)
		pt.calculateFinalMetrics(testDuration)

		assert.Greater(t, pt.metrics.SuccessfulRequests, int64(0))
		assert.Less(t, pt.metrics.ErrorRate, 5.0, "Gateway routing error rate should be less than 5%")
		assert.Greater(t, pt.metrics.RequestsPerSecond, 10.0, "Gateway should handle at least 10 requests per second")

		t.Logf("Gateway Performance Metrics: %+v", pt.metrics)
	})
}

// Test Cross-Chain Bridge Performance
func (pt *PerformanceTester) TestBridgePerformance(t *testing.T) {
	config := LoadTestConfig{
		ConcurrentUsers: 5, // Lower concurrency for bridge operations
		TestDuration:    60 * time.Second,
		RequestsPerUser: 10,
		RampUpTime:      10 * time.Second,
	}

	t.Run("BridgeTransferLoad", func(t *testing.T) {
		pt.resetMetrics()

		ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
		defer cancel()

		var wg sync.WaitGroup
		startTime := time.Now()

		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				for j := 0; j < config.RequestsPerUser; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						pt.testBridgeTransfer(t, userID, j)
						time.Sleep(2 * time.Second) // Space out bridge operations
					}
				}
			}(i)
		}

		wg.Wait()
		testDuration := time.Since(startTime)
		pt.calculateFinalMetrics(testDuration)

		assert.Greater(t, pt.metrics.SuccessfulRequests, int64(0))
		assert.Less(t, pt.metrics.ErrorRate, 15.0, "Bridge transfer error rate should be less than 15%")

		t.Logf("Bridge Performance Metrics: %+v", pt.metrics)
	})
}

// Test KNIRV-ROUTER Connectivity Performance
func (pt *PerformanceTester) TestKNIRVROUTERPerformance(t *testing.T) {
	config := LoadTestConfig{
		ConcurrentUsers: 12,
		TestDuration:    30 * time.Second,
		RequestsPerUser: 25,
		RampUpTime:      3 * time.Second,
	}

	t.Run("ConnectivityProofLoad", func(t *testing.T) {
		pt.resetMetrics()

		ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
		defer cancel()

		var wg sync.WaitGroup
		startTime := time.Now()

		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				for j := 0; j < config.RequestsPerUser; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						if j%5 == 0 {
							pt.testConnectivityProof(t, userID, j)
						} else {
							pt.testConnectivityStatus(t)
						}
					}
				}
			}(i)
		}

		wg.Wait()
		testDuration := time.Since(startTime)
		pt.calculateFinalMetrics(testDuration)

		assert.Greater(t, pt.metrics.SuccessfulRequests, int64(0))
		assert.Less(t, pt.metrics.ErrorRate, 10.0, "KNIRV-ROUTER error rate should be less than 10%")

		t.Logf("KNIRV-ROUTER Performance Metrics: %+v", pt.metrics)
	})
}

// Helper methods for Month 12 performance tests
func (pt *PerformanceTester) testGatewayRouting(_ *testing.T, service string, userID, requestID int) {
	start := time.Now()

	endpoint := fmt.Sprintf("/%s/health", service)
	_, err := pt.suite.makeRequest("GET", endpoint, nil)

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func (pt *PerformanceTester) testBridgeTransfer(_ *testing.T, userID, requestID int) {
	start := time.Now()

	bridgeData := map[string]interface{}{
		"target_chain": "xion",
		"amount":       "100000", // Small amount for load testing
		"recipient":    fmt.Sprintf("test_recipient_%d_%d", userID, requestID),
		"source":       "KNIRVROOT",
	}

	_, err := pt.suite.makeRequest("POST", "/knirvroot/bridge/transfer", bridgeData)

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func (pt *PerformanceTester) testConnectivityProof(_ *testing.T, userID, requestID int) {
	start := time.Now()

	_, err := pt.suite.makeRequest("POST", "/knirvrouter/api/connectivity/proofs", map[string]interface{}{
		"test_id": fmt.Sprintf("perf_test_%d_%d", userID, requestID),
	})

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func (pt *PerformanceTester) testConnectivityStatus(_ *testing.T) {
	start := time.Now()

	_, err := pt.suite.makeRequest("GET", "/knirvrouter/api/connectivity/status", nil)

	latency := time.Since(start)
	pt.recordRequest(latency, err == nil)
}

func TestPerformanceAndLoad(t *testing.T) {
	suite := NewIntegrationTestSuite()
	suite.SetupTest(t)

	tester := NewPerformanceTester(suite)

	t.Run("KNIRVCHAINPerformance", tester.TestKNIRVCHAINPerformance)
	t.Run("KNIRVGRAPHPerformance", tester.TestKNIRVGRAPHPerformance)
	t.Run("KNIRVNEXUSPerformance", tester.TestKNIRVNEXUSPerformance)

	// Month 12 Additional Performance Tests
	t.Run("GatewayPerformance", tester.TestGatewayPerformance)
	t.Run("BridgePerformance", tester.TestBridgePerformance)
	t.Run("KNIRVROUTERPerformance", tester.TestKNIRVROUTERPerformance)
}
