package loadtesting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LoadTestSuite manages load testing for the testnet
type LoadTestSuite struct {
	Config      LoadTestConfig
	Services    map[string]*ServiceTarget
	Metrics     *LoadTestMetrics
	Results     *LoadTestResults
	Context     context.Context
	Cancel      context.CancelFunc
	mu          sync.RWMutex
}

// LoadTestConfig holds load test configuration
type LoadTestConfig struct {
	Duration        time.Duration
	MaxUsers        int
	RampUpTime      time.Duration
	RampDownTime    time.Duration
	RequestRate     int
	Timeout         time.Duration
	ThinkTime       time.Duration
	FailureThreshold float64
}

// ServiceTarget represents a service to load test
type ServiceTarget struct {
	Name      string
	BaseURL   string
	Endpoints []EndpointTarget
	Weight    float64 // Percentage of total load
}

// EndpointTarget represents an endpoint to test
type EndpointTarget struct {
	Path         string
	Method       string
	RequestData  interface{}
	Weight       float64
	ExpectedCode int
}

// LoadTestMetrics holds real-time metrics
type LoadTestMetrics struct {
	TotalRequests    int64
	SuccessfulReqs   int64
	FailedRequests   int64
	TotalLatency     int64 // in milliseconds
	MinLatency       int64
	MaxLatency       int64
	ActiveUsers      int64
	RequestsPerSec   float64
	ErrorRate        float64
	LastUpdated      time.Time
	mu               sync.RWMutex
}

// LoadTestResults holds final test results
type LoadTestResults struct {
	StartTime       time.Time
	EndTime         time.Time
	TotalDuration   time.Duration
	ServiceResults  map[string]*ServiceLoadResults
	OverallMetrics  OverallLoadMetrics
	Passed          bool
	FailureReasons  []string
}

// ServiceLoadResults holds results for a specific service
type ServiceLoadResults struct {
	ServiceName     string
	TotalRequests   int64
	SuccessfulReqs  int64
	FailedRequests  int64
	AvgResponseTime time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration
	Throughput      float64
	ErrorRate       float64
	Percentiles     map[string]time.Duration
	EndpointResults map[string]*EndpointLoadResults
}

// EndpointLoadResults holds results for a specific endpoint
type EndpointLoadResults struct {
	Path            string
	Method          string
	TotalRequests   int64
	SuccessfulReqs  int64
	FailedRequests  int64
	AvgResponseTime time.Duration
	Throughput      float64
	ErrorRate       float64
}

// OverallLoadMetrics holds overall system metrics
type OverallLoadMetrics struct {
	TotalRequests      int64
	TotalErrors        int64
	OverallThroughput  float64
	OverallErrorRate   float64
	AvgResponseTime    time.Duration
	PeakThroughput     float64
	SystemStability    float64
}

// UserSession represents a virtual user session
type UserSession struct {
	ID          int
	StartTime   time.Time
	EndTime     time.Time
	Requests    int64
	Errors      int64
	TotalLatency time.Duration
	Active      bool
}

// NewLoadTestSuite creates a new load test suite
func NewLoadTestSuite() *LoadTestSuite {
	config := LoadTestConfig{
		Duration:         5 * time.Minute,
		MaxUsers:         100,
		RampUpTime:       30 * time.Second,
		RampDownTime:     30 * time.Second,
		RequestRate:      10, // requests per user per second
		Timeout:          30 * time.Second,
		ThinkTime:        1 * time.Second,
		FailureThreshold: 0.05, // 5% error rate threshold
	}

	services := map[string]*ServiceTarget{
		"knirv-root": {
			Name:    "knirv-root",
			BaseURL: "http://localhost:1317",
			Weight:  0.2,
			Endpoints: []EndpointTarget{
				{Path: "/health", Method: "GET", Weight: 0.3, ExpectedCode: 200},
				{Path: "/status", Method: "GET", Weight: 0.3, ExpectedCode: 200},
				{Path: "/balance/test_address", Method: "GET", Weight: 0.4, ExpectedCode: 200},
			},
		},
		"knirvchain": {
			Name:    "knirvchain",
			BaseURL: "http://localhost:8090",
			Weight:  0.25,
			Endpoints: []EndpointTarget{
				{Path: "/health", Method: "GET", Weight: 0.2, ExpectedCode: 200},
				{Path: "/skills", Method: "GET", Weight: 0.4, ExpectedCode: 200},
				{Path: "/models", Method: "GET", Weight: 0.4, ExpectedCode: 200},
			},
		},
		"knirvgraph": {
			Name:    "knirvgraph",
			BaseURL: "http://localhost:8082",
			Weight:  0.2,
			Endpoints: []EndpointTarget{
				{Path: "/height", Method: "GET", Weight: 0.3, ExpectedCode: 200},
				{Path: "/nodes", Method: "GET", Weight: 0.35, ExpectedCode: 200},
				{Path: "/edges", Method: "GET", Weight: 0.35, ExpectedCode: 200},
			},
		},
		"knirv-nexus": {
			Name:    "knirv-nexus",
			BaseURL: "http://localhost:8084",
			Weight:  0.15,
			Endpoints: []EndpointTarget{
				{Path: "/health", Method: "GET", Weight: 0.4, ExpectedCode: 200},
				{Path: "/nodes", Method: "GET", Weight: 0.3, ExpectedCode: 200},
				{Path: "/tasks", Method: "GET", Weight: 0.3, ExpectedCode: 200},
			},
		},
		"knirv-router": {
			Name:    "knirv-router",
			BaseURL: "http://localhost:5001",
			Weight:  0.1,
			Endpoints: []EndpointTarget{
				{Path: "/health", Method: "GET", Weight: 0.4, ExpectedCode: 200},
				{Path: "/peers", Method: "GET", Weight: 0.3, ExpectedCode: 200},
				{Path: "/routes", Method: "GET", Weight: 0.3, ExpectedCode: 200},
			},
		},
		"knirv-gateway": {
			Name:    "knirv-gateway",
			BaseURL: "http://localhost:8087",
			Weight:  0.1,
			Endpoints: []EndpointTarget{
				{Path: "/health", Method: "GET", Weight: 0.4, ExpectedCode: 200},
				{Path: "/api/status", Method: "GET", Weight: 0.3, ExpectedCode: 200},
				{Path: "/api/services", Method: "GET", Weight: 0.3, ExpectedCode: 200},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &LoadTestSuite{
		Config:   config,
		Services: services,
		Metrics:  NewLoadTestMetrics(),
		Results: &LoadTestResults{
			ServiceResults: make(map[string]*ServiceLoadResults),
		},
		Context: ctx,
		Cancel:  cancel,
	}
}

// NewLoadTestMetrics creates new load test metrics
func NewLoadTestMetrics() *LoadTestMetrics {
	return &LoadTestMetrics{
		MinLatency:  999999,
		MaxLatency:  0,
		LastUpdated: time.Now(),
	}
}

// TestLoadCapacity tests system capacity under various load levels
func TestLoadCapacity(t *testing.T) {
	suite := NewLoadTestSuite()
	defer suite.Cancel()

	loadLevels := []struct {
		users    int
		duration time.Duration
		name     string
	}{
		{10, 2 * time.Minute, "Light_Load"},
		{25, 2 * time.Minute, "Medium_Load"},
		{50, 3 * time.Minute, "Heavy_Load"},
		{100, 3 * time.Minute, "Peak_Load"},
	}

	for _, level := range loadLevels {
		t.Run(level.name, func(t *testing.T) {
			// Configure test for this load level
			suite.Config.MaxUsers = level.users
			suite.Config.Duration = level.duration

			// Run load test
			results, err := suite.RunLoadTest()
			require.NoError(t, err)

			// Validate results
			assert.Less(t, results.OverallMetrics.OverallErrorRate, suite.Config.FailureThreshold,
				"Error rate exceeds threshold")
			assert.Greater(t, results.OverallMetrics.OverallThroughput, float64(level.users)*0.5,
				"Throughput too low for load level")
			assert.Less(t, results.OverallMetrics.AvgResponseTime, 2*time.Second,
				"Average response time too high")

			// Log results
			t.Logf("Load Level: %s", level.name)
			t.Logf("Users: %d, Duration: %s", level.users, level.duration)
			t.Logf("Total Requests: %d", results.OverallMetrics.TotalRequests)
			t.Logf("Error Rate: %.4f", results.OverallMetrics.OverallErrorRate)
			t.Logf("Throughput: %.2f req/s", results.OverallMetrics.OverallThroughput)
			t.Logf("Avg Response Time: %s", results.OverallMetrics.AvgResponseTime)
		})
	}
}

// RunLoadTest executes the load test
func (suite *LoadTestSuite) RunLoadTest() (*LoadTestResults, error) {
	suite.Results.StartTime = time.Now()

	// Start metrics collection
	go suite.collectMetrics()

	// Start user sessions
	var wg sync.WaitGroup
	userSessions := make([]*UserSession, suite.Config.MaxUsers)

	// Ramp up users
	rampUpInterval := suite.Config.RampUpTime / time.Duration(suite.Config.MaxUsers)

	for i := 0; i < suite.Config.MaxUsers; i++ {
		userSessions[i] = &UserSession{
			ID:        i + 1,
			StartTime: time.Now(),
			Active:    true,
		}

		wg.Add(1)
		go suite.runUserSession(&wg, userSessions[i])

		// Ramp up delay
		if i < suite.Config.MaxUsers-1 {
			time.Sleep(rampUpInterval)
		}
	}

	// Wait for test duration
	testTimer := time.NewTimer(suite.Config.Duration)
	<-testTimer.C

	// Signal shutdown
	suite.Cancel()

	// Wait for all users to finish
	wg.Wait()

	suite.Results.EndTime = time.Now()
	suite.Results.TotalDuration = suite.Results.EndTime.Sub(suite.Results.StartTime)

	// Calculate final results
	suite.calculateFinalResults()

	return suite.Results, nil
}

// runUserSession simulates a single user session
func (suite *LoadTestSuite) runUserSession(wg *sync.WaitGroup, session *UserSession) {
	defer wg.Done()
	defer func() {
		session.EndTime = time.Now()
		session.Active = false
		atomic.AddInt64(&suite.Metrics.ActiveUsers, -1)
	}()

	atomic.AddInt64(&suite.Metrics.ActiveUsers, 1)

	client := &http.Client{
		Timeout: suite.Config.Timeout,
	}

	requestInterval := time.Second / time.Duration(suite.Config.RequestRate)

	for {
		select {
		case <-suite.Context.Done():
			return
		default:
			// Select service and endpoint based on weights
			service, endpoint := suite.selectTarget()

			// Make request
			startTime := time.Now()
			err := suite.makeRequest(client, service, endpoint)
			latency := time.Since(startTime)

			// Update metrics
			suite.updateMetrics(service.Name, endpoint.Path, latency, err == nil)
			session.Requests++
			session.TotalLatency += latency

			if err != nil {
				session.Errors++
			}

			// Think time
			time.Sleep(suite.Config.ThinkTime)

			// Request rate limiting
			time.Sleep(requestInterval)
		}
	}
}

// selectTarget selects a service and endpoint based on weights
func (suite *LoadTestSuite) selectTarget() (*ServiceTarget, *EndpointTarget) {
	// Simple random selection for now
	// In a real implementation, this would use weighted selection
	serviceNames := make([]string, 0, len(suite.Services))
	for name := range suite.Services {
		serviceNames = append(serviceNames, name)
	}

	serviceName := serviceNames[time.Now().UnixNano()%int64(len(serviceNames))]
	service := suite.Services[serviceName]

	endpointIndex := time.Now().UnixNano() % int64(len(service.Endpoints))
	endpoint := &service.Endpoints[endpointIndex]

	return service, endpoint
}

// makeRequest makes an HTTP request
func (suite *LoadTestSuite) makeRequest(client *http.Client, service *ServiceTarget, endpoint *EndpointTarget) error {
	url := service.BaseURL + endpoint.Path

	req, err := http.NewRequestWithContext(suite.Context, endpoint.Method, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != endpoint.ExpectedCode {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// updateMetrics updates load test metrics
func (suite *LoadTestSuite) updateMetrics(serviceName, endpoint string, latency time.Duration, success bool) {
	suite.Metrics.mu.Lock()
	defer suite.Metrics.mu.Unlock()

	latencyMs := latency.Milliseconds()

	atomic.AddInt64(&suite.Metrics.TotalRequests, 1)
	atomic.AddInt64(&suite.Metrics.TotalLatency, latencyMs)

	if success {
		atomic.AddInt64(&suite.Metrics.SuccessfulReqs, 1)
	} else {
		atomic.AddInt64(&suite.Metrics.FailedRequests, 1)
	}

	// Update min/max latency
	if latencyMs < suite.Metrics.MinLatency {
		suite.Metrics.MinLatency = latencyMs
	}
	if latencyMs > suite.Metrics.MaxLatency {
		suite.Metrics.MaxLatency = latencyMs
	}

	suite.Metrics.LastUpdated = time.Now()
}

// collectMetrics collects real-time metrics
func (suite *LoadTestSuite) collectMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-suite.Context.Done():
			return
		case <-ticker.C:
			suite.calculateRealTimeMetrics()
		}
	}
}

// calculateRealTimeMetrics calculates real-time metrics
func (suite *LoadTestSuite) calculateRealTimeMetrics() {
	suite.Metrics.mu.Lock()
	defer suite.Metrics.mu.Unlock()

	totalReqs := atomic.LoadInt64(&suite.Metrics.TotalRequests)
	failedReqs := atomic.LoadInt64(&suite.Metrics.FailedRequests)

	if totalReqs > 0 {
		suite.Metrics.ErrorRate = float64(failedReqs) / float64(totalReqs)

		// Calculate requests per second
		elapsed := time.Since(suite.Results.StartTime).Seconds()
		if elapsed > 0 {
			suite.Metrics.RequestsPerSec = float64(totalReqs) / elapsed
		}
	}
}

// calculateFinalResults calculates final test results
func (suite *LoadTestSuite) calculateFinalResults() {
	// Calculate overall metrics
	totalReqs := atomic.LoadInt64(&suite.Metrics.TotalRequests)
	totalErrors := atomic.LoadInt64(&suite.Metrics.FailedRequests)
	totalLatency := atomic.LoadInt64(&suite.Metrics.TotalLatency)

	suite.Results.OverallMetrics = OverallLoadMetrics{
		TotalRequests:     totalReqs,
		TotalErrors:       totalErrors,
		OverallThroughput: float64(totalReqs) / suite.Results.TotalDuration.Seconds(),
		OverallErrorRate:  float64(totalErrors) / float64(totalReqs),
		AvgResponseTime:   time.Duration(totalLatency/totalReqs) * time.Millisecond,
		PeakThroughput:    suite.Metrics.RequestsPerSec,
		SystemStability:   1.0 - (float64(totalErrors)/float64(totalReqs)),
	}

	// Check if test passed
	suite.Results.Passed = suite.Results.OverallMetrics.OverallErrorRate <= suite.Config.FailureThreshold

	if !suite.Results.Passed {
		suite.Results.FailureReasons = append(suite.Results.FailureReasons,
			fmt.Sprintf("Error rate %.4f exceeds threshold %.4f",
				suite.Results.OverallMetrics.OverallErrorRate,
				suite.Config.FailureThreshold))
	}
}

// TestStressLimits tests system behavior at stress limits
func TestStressLimits(t *testing.T) {
	suite := NewLoadTestSuite()
	defer suite.Cancel()

	stressLevels := []struct {
		users       int
		requestRate int
		name        string
	}{
		{200, 20, "High_Stress"},
		{500, 30, "Extreme_Stress"},
		{1000, 50, "Breaking_Point"},
	}

	for _, level := range stressLevels {
		t.Run(level.name, func(t *testing.T) {
			suite.Config.MaxUsers = level.users
			suite.Config.RequestRate = level.requestRate
			suite.Config.Duration = 2 * time.Minute

			results, err := suite.RunLoadTest()
			require.NoError(t, err)

			// For stress tests, we expect higher error rates
			maxErrorRate := 0.1 // 10% for stress tests
			assert.Less(t, results.OverallMetrics.OverallErrorRate, maxErrorRate,
				"Error rate too high even for stress test")

			// System should still be responsive
			assert.Less(t, results.OverallMetrics.AvgResponseTime, 5*time.Second,
				"Response time too high under stress")

			t.Logf("Stress Level: %s", level.name)
			t.Logf("Users: %d, Request Rate: %d", level.users, level.requestRate)
			t.Logf("Error Rate: %.4f", results.OverallMetrics.OverallErrorRate)
			t.Logf("Avg Response Time: %s", results.OverallMetrics.AvgResponseTime)
			t.Logf("System Stability: %.4f", results.OverallMetrics.SystemStability)
		})
	}
}

// TestRecoveryAfterLoad tests system recovery after load
func TestRecoveryAfterLoad(t *testing.T) {
	suite := NewLoadTestSuite()
	defer suite.Cancel()

	// Run high load test
	suite.Config.MaxUsers = 100
	suite.Config.Duration = 2 * time.Minute

	t.Log("Running high load test...")
	results, err := suite.RunLoadTest()
	require.NoError(t, err)

	// Wait for system to recover
	t.Log("Waiting for system recovery...")
	time.Sleep(30 * time.Second)

	// Test system responsiveness after load
	t.Log("Testing system responsiveness after load...")
	client := &http.Client{Timeout: 10 * time.Second}

	for serviceName, service := range suite.Services {
		for _, endpoint := range service.Endpoints {
			startTime := time.Now()
			err := suite.makeRequest(client, service, &endpoint)
			responseTime := time.Since(startTime)

			assert.NoError(t, err, "Service %s endpoint %s should be responsive after load", serviceName, endpoint.Path)
			assert.Less(t, responseTime, 1*time.Second, "Response time should be normal after recovery")
		}
	}

	t.Log("System recovery test completed successfully")
}

// GenerateLoadTestReport generates a comprehensive load test report
func (suite *LoadTestSuite) GenerateLoadTestReport() string {
	report := fmt.Sprintf(`
KNIRV TESTNET Load Test Report
==============================
Test Duration: %s
Max Users: %d
Request Rate: %d req/user/sec

Overall Results:
- Total Requests: %d
- Successful: %d (%.2f%%)
- Failed: %d (%.2f%%)
- Throughput: %.2f req/s
- Avg Response Time: %s
- Error Rate: %.4f
- System Stability: %.4f

`,
		suite.Config.Duration,
		suite.Config.MaxUsers,
		suite.Config.RequestRate,
		suite.Results.OverallMetrics.TotalRequests,
		suite.Results.OverallMetrics.TotalRequests-suite.Results.OverallMetrics.TotalErrors,
		(float64(suite.Results.OverallMetrics.TotalRequests-suite.Results.OverallMetrics.TotalErrors)/float64(suite.Results.OverallMetrics.TotalRequests))*100,
		suite.Results.OverallMetrics.TotalErrors,
		suite.Results.OverallMetrics.OverallErrorRate*100,
		suite.Results.OverallMetrics.OverallThroughput,
		suite.Results.OverallMetrics.AvgResponseTime,
		suite.Results.OverallMetrics.OverallErrorRate,
		suite.Results.OverallMetrics.SystemStability)

	if suite.Results.Passed {
		report += "Test Status: PASSED\n"
	} else {
		report += "Test Status: FAILED\n"
		report += "Failure Reasons:\n"
		for _, reason := range suite.Results.FailureReasons {
			report += fmt.Sprintf("- %s\n", reason)
		}
	}

	return report
}
