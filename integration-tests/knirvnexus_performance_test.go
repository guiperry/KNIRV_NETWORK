package integration_tests

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Performance test configuration
const (
	PERFORMANCE_TEST_DURATION = 30 * time.Second
	MAX_CONCURRENT_USERS      = 50
	LOAD_TEST_REQUESTS        = 1000
	MEMORY_CHECK_INTERVAL     = 5 * time.Second
)

// Phase6PerformanceMetrics holds performance test results
type Phase6PerformanceMetrics struct {
	TotalRequests     int
	SuccessfulReqs    int
	FailedReqs        int
	AverageLatency    time.Duration
	MinLatency        time.Duration
	MaxLatency        time.Duration
	RequestsPerSecond float64
	ErrorRate         float64
}

// RequestResult holds individual request results
type RequestResult struct {
	Success bool
	Latency time.Duration
	Status  int
	Error   error
}

func TestKNIRVNEXUSPerformanceLoad(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVNEXUS service not available for performance testing")
	}

	t.Run("TestBasicLoadTesting", func(t *testing.T) {
		// Test basic load handling
		concurrency := 10
		requestsPerWorker := 50

		metrics := runLoadTest(t, "/health", concurrency, requestsPerWorker)

		// Performance assertions
		assert.Greater(t, metrics.SuccessfulReqs, 0, "Should have successful requests")
		assert.Less(t, metrics.ErrorRate, 0.05, "Error rate should be less than 5%")
		assert.Less(t, metrics.AverageLatency, 2*time.Second, "Average latency should be under 2s")
		assert.Greater(t, metrics.RequestsPerSecond, 5.0, "Should handle at least 5 requests per second")

		t.Logf("Load Test Results:")
		t.Logf("  Total Requests: %d", metrics.TotalRequests)
		t.Logf("  Successful: %d", metrics.SuccessfulReqs)
		t.Logf("  Failed: %d", metrics.FailedReqs)
		t.Logf("  Error Rate: %.2f%%", metrics.ErrorRate*100)
		t.Logf("  Average Latency: %v", metrics.AverageLatency)
		t.Logf("  Min Latency: %v", metrics.MinLatency)
		t.Logf("  Max Latency: %v", metrics.MaxLatency)
		t.Logf("  Requests/Second: %.2f", metrics.RequestsPerSecond)
	})

	t.Run("TestAPIEndpointPerformance", func(t *testing.T) {
		// Test performance of different API endpoints
		endpoints := []string{
			"/health",
			"/api/system-health",
			"/api/dve-nodes",
			"/api/validation-tasks",
		}

		concurrency := 5
		requestsPerWorker := 20

		for _, endpoint := range endpoints {
			t.Run(fmt.Sprintf("Endpoint_%s", endpoint), func(t *testing.T) {
				metrics := runLoadTest(t, endpoint, concurrency, requestsPerWorker)

				// Endpoint-specific performance requirements
				maxLatency := 3 * time.Second
				if endpoint == "/health" {
					maxLatency = 1 * time.Second // Health endpoint should be faster
				}

				assert.Less(t, metrics.AverageLatency, maxLatency,
					"Average latency for %s should be under %v", endpoint, maxLatency)
				assert.Less(t, metrics.ErrorRate, 0.1,
					"Error rate for %s should be less than 10%%", endpoint)

				t.Logf("Endpoint %s Performance:", endpoint)
				t.Logf("  Average Latency: %v", metrics.AverageLatency)
				t.Logf("  Requests/Second: %.2f", metrics.RequestsPerSecond)
				t.Logf("  Error Rate: %.2f%%", metrics.ErrorRate*100)
			})
		}
	})
}

func TestKNIRVNEXUSConcurrentUsers(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVNEXUS service not available for concurrent user testing")
	}

	t.Run("TestConcurrentUserScaling", func(t *testing.T) {
		// Test scaling with increasing concurrent users
		userCounts := []int{1, 5, 10, 20, 30}
		requestsPerUser := 10

		for _, userCount := range userCounts {
			t.Run(fmt.Sprintf("Users_%d", userCount), func(t *testing.T) {
				metrics := runLoadTest(t, "/health", userCount, requestsPerUser)

				// Performance should degrade gracefully
				maxLatency := time.Duration(userCount*100) * time.Millisecond
				if maxLatency > 5*time.Second {
					maxLatency = 5 * time.Second
				}

				assert.Less(t, metrics.AverageLatency, maxLatency,
					"Average latency with %d users should be under %v", userCount, maxLatency)
				assert.Less(t, metrics.ErrorRate, 0.15,
					"Error rate with %d users should be less than 15%%", userCount)

				t.Logf("%d Concurrent Users:", userCount)
				t.Logf("  Average Latency: %v", metrics.AverageLatency)
				t.Logf("  Requests/Second: %.2f", metrics.RequestsPerSecond)
				t.Logf("  Error Rate: %.2f%%", metrics.ErrorRate*100)
			})
		}
	})
}

func TestKNIRVNEXUSMemoryUsage(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVNEXUS service not available for memory testing")
	}

	t.Run("TestMemoryUsageUnderLoad", func(t *testing.T) {
		// Monitor memory usage during load testing
		var memStats []runtime.MemStats
		var memMutex sync.Mutex

		// Start memory monitoring
		stopMonitoring := make(chan bool)
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					var m runtime.MemStats
					runtime.ReadMemStats(&m)

					memMutex.Lock()
					memStats = append(memStats, m)
					memMutex.Unlock()
				case <-stopMonitoring:
					return
				}
			}
		}()

		// Run load test while monitoring memory
		metrics := runLoadTest(t, "/health", 20, 25)

		// Stop monitoring
		stopMonitoring <- true
		time.Sleep(100 * time.Millisecond) // Allow final measurement

		// Analyze memory usage
		memMutex.Lock()
		defer memMutex.Unlock()

		if len(memStats) > 0 {
			initialMem := memStats[0].Alloc
			maxMem := initialMem
			finalMem := memStats[len(memStats)-1].Alloc

			for _, stat := range memStats {
				if stat.Alloc > maxMem {
					maxMem = stat.Alloc
				}
			}

			memoryGrowth := float64(maxMem-initialMem) / float64(initialMem) * 100

			t.Logf("Memory Usage Analysis:")
			t.Logf("  Initial Memory: %d bytes", initialMem)
			t.Logf("  Max Memory: %d bytes", maxMem)
			t.Logf("  Final Memory: %d bytes", finalMem)
			t.Logf("  Memory Growth: %.2f%%", memoryGrowth)
			t.Logf("  GC Cycles: %d", memStats[len(memStats)-1].NumGC-memStats[0].NumGC)

			// Memory growth should be reasonable
			assert.Less(t, memoryGrowth, 200.0,
				"Memory growth should be less than 200%% during load test")
		}

		// Load test should still perform well
		assert.Greater(t, metrics.SuccessfulReqs, 0, "Should have successful requests")
		assert.Less(t, metrics.ErrorRate, 0.1, "Error rate should be less than 10%")
	})
}

func TestKNIRVNEXUSNetworkLatency(t *testing.T) {
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", 10*time.Second) {
		t.Skip("KNIRVNEXUS service not available for latency testing")
	}

	t.Run("TestNetworkLatencyMeasurement", func(t *testing.T) {
		// Measure network latency components
		iterations := 50
		latencies := make([]time.Duration, iterations)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/health", nil, nil)
			latency := time.Since(start)

			if err == nil && resp.StatusCode == http.StatusOK {
				latencies[i] = latency
				resp.Body.Close()
			} else {
				t.Logf("Request %d failed: %v", i, err)
			}

			// Small delay between requests
			time.Sleep(10 * time.Millisecond)
		}

		// Calculate latency statistics
		var totalLatency time.Duration
		minLatency := time.Hour
		maxLatency := time.Duration(0)
		validMeasurements := 0

		for _, latency := range latencies {
			if latency > 0 {
				totalLatency += latency
				validMeasurements++

				if latency < minLatency {
					minLatency = latency
				}
				if latency > maxLatency {
					maxLatency = latency
				}
			}
		}

		if validMeasurements > 0 {
			avgLatency := totalLatency / time.Duration(validMeasurements)

			t.Logf("Network Latency Analysis (%d measurements):", validMeasurements)
			t.Logf("  Average Latency: %v", avgLatency)
			t.Logf("  Min Latency: %v", minLatency)
			t.Logf("  Max Latency: %v", maxLatency)
			t.Logf("  Latency Range: %v", maxLatency-minLatency)

			// Latency requirements
			assert.Less(t, avgLatency, 1*time.Second,
				"Average latency should be under 1 second")
			assert.Less(t, maxLatency, 5*time.Second,
				"Max latency should be under 5 seconds")
			assert.Greater(t, validMeasurements, iterations*8/10,
				"At least 80%% of requests should succeed")
		} else {
			t.Error("No valid latency measurements obtained")
		}
	})
}

// runLoadTest executes a load test and returns performance metrics
func runLoadTest(t *testing.T, endpoint string, concurrency, requestsPerWorker int) Phase6PerformanceMetrics {
	totalRequests := concurrency * requestsPerWorker
	results := make(chan RequestResult, totalRequests)

	var wg sync.WaitGroup
	startTime := time.Now()

	// Launch concurrent workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < requestsPerWorker; j++ {
				reqStart := time.Now()
				resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+endpoint, nil, nil)
				latency := time.Since(reqStart)

				result := RequestResult{
					Latency: latency,
					Error:   err,
				}

				if err == nil {
					result.Status = resp.StatusCode
					result.Success = resp.StatusCode == http.StatusOK
					resp.Body.Close()
				}

				results <- result
			}
		}()
	}

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	totalDuration := time.Since(startTime)

	// Analyze results
	metrics := Phase6PerformanceMetrics{
		TotalRequests: totalRequests,
		MinLatency:    time.Hour,
		MaxLatency:    0,
	}

	var totalLatency time.Duration

	for result := range results {
		if result.Success {
			metrics.SuccessfulReqs++
		} else {
			metrics.FailedReqs++
		}

		if result.Latency > 0 {
			totalLatency += result.Latency

			if result.Latency < metrics.MinLatency {
				metrics.MinLatency = result.Latency
			}
			if result.Latency > metrics.MaxLatency {
				metrics.MaxLatency = result.Latency
			}
		}
	}

	if metrics.TotalRequests > 0 {
		metrics.ErrorRate = float64(metrics.FailedReqs) / float64(metrics.TotalRequests)
		metrics.RequestsPerSecond = float64(metrics.TotalRequests) / totalDuration.Seconds()
	}

	if metrics.SuccessfulReqs > 0 {
		metrics.AverageLatency = totalLatency / time.Duration(metrics.SuccessfulReqs)
	}

	return metrics
}
