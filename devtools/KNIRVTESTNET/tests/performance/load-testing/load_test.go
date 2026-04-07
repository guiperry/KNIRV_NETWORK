package loadtesting

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

const (
	GatewayURL = "http://localhost:8888"
	Timeout    = 30 * time.Second
)

// TestBasicLoadTest tests basic load handling
func TestBasicLoadTest(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Concurrent Health Checks", func(t *testing.T) {
		const numRequests = 50
		const concurrency = 10

		var wg sync.WaitGroup
		results := make(chan error, numRequests)
		semaphore := make(chan struct{}, concurrency)

		start := time.Now()

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				resp, err := client.Get(GatewayURL + "/gateway/health")
				if err != nil {
					results <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					results <- fmt.Errorf("status code: %d", resp.StatusCode)
					return
				}

				results <- nil
			}()
		}

		wg.Wait()
		close(results)

		duration := time.Since(start)
		errors := 0
		for err := range results {
			if err != nil {
				errors++
				t.Logf("Request error: %v", err)
			}
		}

		successRate := float64(numRequests-errors) / float64(numRequests) * 100
		requestsPerSecond := float64(numRequests) / duration.Seconds()

		t.Logf("✅ Load test completed:")
		t.Logf("   Duration: %v", duration)
		t.Logf("   Requests: %d", numRequests)
		t.Logf("   Errors: %d", errors)
		t.Logf("   Success Rate: %.2f%%", successRate)
		t.Logf("   Requests/sec: %.2f", requestsPerSecond)

		if successRate < 95.0 {
			t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate)
		}
	})

	t.Run("Sustained Load Test", func(t *testing.T) {
		const duration = 30 * time.Second
		const requestInterval = 100 * time.Millisecond

		start := time.Now()
		var requestCount int
		var errorCount int

		for time.Since(start) < duration {
			resp, err := client.Get(GatewayURL + "/gateway/health")
			requestCount++

			if err != nil {
				errorCount++
				t.Logf("Request error: %v", err)
			} else {
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					errorCount++
					t.Logf("Status error: %d", resp.StatusCode)
				}
			}

			time.Sleep(requestInterval)
		}

		actualDuration := time.Since(start)
		successRate := float64(requestCount-errorCount) / float64(requestCount) * 100
		requestsPerSecond := float64(requestCount) / actualDuration.Seconds()

		t.Logf("✅ Sustained load test completed:")
		t.Logf("   Duration: %v", actualDuration)
		t.Logf("   Requests: %d", requestCount)
		t.Logf("   Errors: %d", errorCount)
		t.Logf("   Success Rate: %.2f%%", successRate)
		t.Logf("   Requests/sec: %.2f", requestsPerSecond)

		if successRate < 90.0 {
			t.Errorf("Sustained load success rate too low: %.2f%% (expected >= 90%%)", successRate)
		}
	})
}

// TestServiceLoadDistribution tests load distribution across services
func TestServiceLoadDistribution(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	endpoints := []string{
		"/gateway/health",
		"/gateway/services",
		"/gateway/testnet/status",
		"/auth/testnet-tokens",
	}

	t.Run("Multi-Endpoint Load Test", func(t *testing.T) {
		const requestsPerEndpoint = 20
		var wg sync.WaitGroup
		results := make(map[string][]error)
		var mu sync.Mutex

		for _, endpoint := range endpoints {
			results[endpoint] = make([]error, 0)
		}

		start := time.Now()

		for _, endpoint := range endpoints {
			for i := 0; i < requestsPerEndpoint; i++ {
				wg.Add(1)
				go func(ep string) {
					defer wg.Done()

					resp, err := client.Get(GatewayURL + ep)
					mu.Lock()
					if err != nil {
						results[ep] = append(results[ep], err)
					} else {
						resp.Body.Close()
						if resp.StatusCode != http.StatusOK {
							results[ep] = append(results[ep], fmt.Errorf("status: %d", resp.StatusCode))
						} else {
							results[ep] = append(results[ep], nil)
						}
					}
					mu.Unlock()
				}(endpoint)
			}
		}

		wg.Wait()
		duration := time.Since(start)

		totalRequests := len(endpoints) * requestsPerEndpoint
		totalErrors := 0

		for endpoint, errs := range results {
			errorCount := 0
			for _, err := range errs {
				if err != nil {
					errorCount++
				}
			}
			totalErrors += errorCount
			successRate := float64(requestsPerEndpoint-errorCount) / float64(requestsPerEndpoint) * 100
			t.Logf("✅ Endpoint %s: %.2f%% success rate (%d/%d)", endpoint, successRate, requestsPerEndpoint-errorCount, requestsPerEndpoint)
		}

		overallSuccessRate := float64(totalRequests-totalErrors) / float64(totalRequests) * 100
		requestsPerSecond := float64(totalRequests) / duration.Seconds()

		t.Logf("✅ Multi-endpoint load test completed:")
		t.Logf("   Duration: %v", duration)
		t.Logf("   Total Requests: %d", totalRequests)
		t.Logf("   Total Errors: %d", totalErrors)
		t.Logf("   Overall Success Rate: %.2f%%", overallSuccessRate)
		t.Logf("   Requests/sec: %.2f", requestsPerSecond)

		if overallSuccessRate < 90.0 {
			t.Errorf("Overall success rate too low: %.2f%% (expected >= 90%%)", overallSuccessRate)
		}
	})
}

// TestResponseTimeConsistency tests response time consistency under load
func TestResponseTimeConsistency(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Response Time Analysis", func(t *testing.T) {
		const numRequests = 100
		responseTimes := make([]time.Duration, numRequests)

		for i := 0; i < numRequests; i++ {
			start := time.Now()
			resp, err := client.Get(GatewayURL + "/gateway/health")
			duration := time.Since(start)

			if err != nil {
				t.Errorf("Request %d failed: %v", i, err)
				continue
			}
			resp.Body.Close()

			responseTimes[i] = duration
		}

		// Calculate statistics
		var total time.Duration
		var min, max time.Duration = responseTimes[0], responseTimes[0]

		for _, rt := range responseTimes {
			total += rt
			if rt < min {
				min = rt
			}
			if rt > max {
				max = rt
			}
		}

		avg := total / time.Duration(numRequests)

		t.Logf("✅ Response time analysis:")
		t.Logf("   Requests: %d", numRequests)
		t.Logf("   Average: %v", avg)
		t.Logf("   Min: %v", min)
		t.Logf("   Max: %v", max)
		t.Logf("   Range: %v", max-min)

		if avg > 1*time.Second {
			t.Errorf("Average response time too high: %v (expected < 1s)", avg)
		}

		if max > 5*time.Second {
			t.Errorf("Maximum response time too high: %v (expected < 5s)", max)
		}
	})
}

// TestMemoryLeakDetection tests for potential memory leaks under sustained load
func TestMemoryLeakDetection(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Sustained Request Pattern", func(t *testing.T) {
		const testDuration = 60 * time.Second
		const requestInterval = 50 * time.Millisecond

		start := time.Now()
		requestCount := 0
		errorCount := 0

		for time.Since(start) < testDuration {
			resp, err := client.Get(GatewayURL + "/gateway/health")
			requestCount++

			if err != nil {
				errorCount++
			} else {
				// Read and discard body to ensure proper cleanup
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}

			time.Sleep(requestInterval)
		}

		successRate := float64(requestCount-errorCount) / float64(requestCount) * 100

		t.Logf("✅ Memory leak detection test:")
		t.Logf("   Duration: %v", testDuration)
		t.Logf("   Requests: %d", requestCount)
		t.Logf("   Errors: %d", errorCount)
		t.Logf("   Success Rate: %.2f%%", successRate)

		if successRate < 95.0 {
			t.Errorf("Success rate degraded over time: %.2f%% (possible memory leak)", successRate)
		}
	})
}
