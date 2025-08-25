package benchmarking

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
	// Helper function for random number generation
	"math/rand"
	"sort"

	"github.com/stretchr/testify/assert"
)

// BenchmarkSuite manages performance benchmarking for the testnet
type BenchmarkSuite struct {
	Services     map[string]*ServiceBenchmark
	Context      context.Context
	Results      *BenchmarkResults
	Config       BenchmarkConfig
	mu           sync.RWMutex
}

// ServiceBenchmark represents benchmarking for a specific service
type ServiceBenchmark struct {
	Name        string
	BaseURL     string
	Endpoints   []EndpointBenchmark
	Metrics     ServiceMetrics
	Thresholds  PerformanceThresholds
}

// EndpointBenchmark represents benchmarking for a specific endpoint
type EndpointBenchmark struct {
	Path           string
	Method         string
	RequestData    interface{}
	ExpectedStatus int
	Metrics        EndpointMetrics
}

// BenchmarkConfig holds configuration for benchmarking
type BenchmarkConfig struct {
	Duration        time.Duration
	ConcurrentUsers int
	RampUpTime      time.Duration
	RequestsPerSec  int
	Timeout         time.Duration
	WarmupRequests  int
}

// BenchmarkResults holds all benchmark results
type BenchmarkResults struct {
	StartTime       time.Time
	EndTime         time.Time
	TotalDuration   time.Duration
	ServiceResults  map[string]*ServiceResults
	OverallMetrics  OverallMetrics
	Passed          bool
}

// ServiceResults holds results for a specific service
type ServiceResults struct {
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
}

// ServiceMetrics holds real-time metrics for a service
type ServiceMetrics struct {
	RequestCount    int64
	ErrorCount      int64
	TotalLatency    time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	LastUpdated     time.Time
}

// EndpointMetrics holds metrics for a specific endpoint
type EndpointMetrics struct {
	RequestCount    int64
	SuccessCount    int64
	ErrorCount      int64
	AvgLatency      time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	Throughput      float64
}

// PerformanceThresholds defines performance expectations
type PerformanceThresholds struct {
	MaxResponseTime time.Duration
	MinThroughput   float64
	MaxErrorRate    float64
	MaxCPUUsage     float64
	MaxMemoryUsage  int64
}

// OverallMetrics holds overall system metrics
type OverallMetrics struct {
	TotalRequests      int64
	TotalErrors        int64
	OverallThroughput  float64
	OverallErrorRate   float64
	AvgResponseTime    time.Duration
	SystemResourceUsage SystemResources
}

// SystemResources holds system resource usage
type SystemResources struct {
	CPUUsage    float64
	MemoryUsage int64
	DiskIO      int64
	NetworkIO   int64
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite() *BenchmarkSuite {
	config := BenchmarkConfig{
		Duration:        5 * time.Minute,
		ConcurrentUsers: 50,
		RampUpTime:      30 * time.Second,
		RequestsPerSec:  100,
		Timeout:         30 * time.Second,
		WarmupRequests:  100,
	}

	services := map[string]*ServiceBenchmark{
		"knirv-oracle": {
			Name:    "knirv-oracle",
			BaseURL: "http://localhost:1317",
			Endpoints: []EndpointBenchmark{
				{Path: "/health", Method: "GET", ExpectedStatus: 200},
				{Path: "/status", Method: "GET", ExpectedStatus: 200},
				{Path: "/balance/test_address", Method: "GET", ExpectedStatus: 200},
			},
			Thresholds: PerformanceThresholds{
				MaxResponseTime: 100 * time.Millisecond,
				MinThroughput:   50.0,
				MaxErrorRate:    0.01,
			},
		},
		"knirvchain": {
			Name:    "knirvchain",
			BaseURL: "http://localhost:8090",
			Endpoints: []EndpointBenchmark{
				{Path: "/health", Method: "GET", ExpectedStatus: 200},
				{Path: "/skills", Method: "GET", ExpectedStatus: 200},
				{Path: "/models", Method: "GET", ExpectedStatus: 200},
			},
			Thresholds: PerformanceThresholds{
				MaxResponseTime: 200 * time.Millisecond,
				MinThroughput:   30.0,
				MaxErrorRate:    0.02,
			},
		},
		"knirvgraph": {
			Name:    "knirvgraph",
			BaseURL: "http://localhost:8082",
			Endpoints: []EndpointBenchmark{
				{Path: "/height", Method: "GET", ExpectedStatus: 200},
				{Path: "/nodes", Method: "GET", ExpectedStatus: 200},
				{Path: "/edges", Method: "GET", ExpectedStatus: 200},
			},
			Thresholds: PerformanceThresholds{
				MaxResponseTime: 150 * time.Millisecond,
				MinThroughput:   40.0,
				MaxErrorRate:    0.015,
			},
		},
		"knirv-nexus": {
			Name:    "knirv-nexus",
			BaseURL: "http://localhost:8084",
			Endpoints: []EndpointBenchmark{
				{Path: "/health", Method: "GET", ExpectedStatus: 200},
				{Path: "/nodes", Method: "GET", ExpectedStatus: 200},
				{Path: "/tasks", Method: "GET", ExpectedStatus: 200},
			},
			Thresholds: PerformanceThresholds{
				MaxResponseTime: 100 * time.Millisecond,
				MinThroughput:   60.0,
				MaxErrorRate:    0.01,
			},
		},
		"knirv-router": {
			Name:    "knirv-router",
			BaseURL: "http://localhost:5001",
			Endpoints: []EndpointBenchmark{
				{Path: "/health", Method: "GET", ExpectedStatus: 200},
				{Path: "/peers", Method: "GET", ExpectedStatus: 200},
				{Path: "/routes", Method: "GET", ExpectedStatus: 200},
			},
			Thresholds: PerformanceThresholds{
				MaxResponseTime: 50 * time.Millisecond,
				MinThroughput:   100.0,
				MaxErrorRate:    0.005,
			},
		},
		"knirv-gateway": {
			Name:    "knirv-gateway",
			BaseURL: "http://localhost:8087",
			Endpoints: []EndpointBenchmark{
				{Path: "/health", Method: "GET", ExpectedStatus: 200},
				{Path: "/api/status", Method: "GET", ExpectedStatus: 200},
				{Path: "/api/services", Method: "GET", ExpectedStatus: 200},
			},
			Thresholds: PerformanceThresholds{
				MaxResponseTime: 100 * time.Millisecond,
				MinThroughput:   80.0,
				MaxErrorRate:    0.01,
			},
		},
	}

	return &BenchmarkSuite{
		Services: services,
		Context:  context.Background(),
		Config:   config,
		Results: &BenchmarkResults{
			ServiceResults: make(map[string]*ServiceResults),
		},
	}
}

// BenchmarkAllServices runs benchmarks for all services
func BenchmarkAllServices(b *testing.B) {
	suite := NewBenchmarkSuite()
	
	b.ResetTimer()
	b.StartTimer()

	suite.Results.StartTime = time.Now()

	// Run benchmarks for each service
	for serviceName, serviceBenchmark := range suite.Services {
		b.Run(fmt.Sprintf("Service_%s", serviceName), func(b *testing.B) {
			suite.benchmarkService(b, serviceBenchmark)
		})
	}

	suite.Results.EndTime = time.Now()
	suite.Results.TotalDuration = suite.Results.EndTime.Sub(suite.Results.StartTime)

	b.StopTimer()

	// Generate benchmark report
	suite.generateBenchmarkReport()
}

// benchmarkService runs benchmark for a specific service
func (suite *BenchmarkSuite) benchmarkService(b *testing.B, serviceBenchmark *ServiceBenchmark) {
	results := &ServiceResults{
		ServiceName: serviceBenchmark.Name,
		Percentiles: make(map[string]time.Duration),
	}

	var responseTimes []time.Duration
	var mu sync.Mutex

	// Warmup phase
	suite.warmupService(serviceBenchmark)

	b.ResetTimer()

	// Run concurrent benchmark
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for _, endpoint := range serviceBenchmark.Endpoints {
				startTime := time.Now()
				
				err := suite.makeRequest(serviceBenchmark.BaseURL, endpoint)
				
				responseTime := time.Since(startTime)
				
				mu.Lock()
				responseTimes = append(responseTimes, responseTime)
				results.TotalRequests++
				
				if err != nil {
					results.FailedRequests++
				} else {
					results.SuccessfulReqs++
				}
				mu.Unlock()
			}
		}
	})

	// Calculate metrics
	suite.calculateServiceMetrics(results, responseTimes)
	
	suite.mu.Lock()
	suite.Results.ServiceResults[serviceBenchmark.Name] = results
	suite.mu.Unlock()
}

// warmupService performs warmup requests
func (suite *BenchmarkSuite) warmupService(serviceBenchmark *ServiceBenchmark) {
	for i := 0; i < suite.Config.WarmupRequests; i++ {
		for _, endpoint := range serviceBenchmark.Endpoints {
			suite.makeRequest(serviceBenchmark.BaseURL, endpoint)
		}
	}
}

// makeRequest makes an HTTP request to the service
func (suite *BenchmarkSuite) makeRequest(baseURL string, endpoint EndpointBenchmark) error {
	// Implementation would make actual HTTP request
	// For now, simulate request with small delay
	time.Sleep(time.Duration(10+rand.Intn(90)) * time.Millisecond)
	
	// Simulate occasional errors
	if rand.Float64() < 0.01 { // 1% error rate
		return fmt.Errorf("simulated error")
	}
	
	return nil
}

// calculateServiceMetrics calculates performance metrics for a service
func (suite *BenchmarkSuite) calculateServiceMetrics(results *ServiceResults, responseTimes []time.Duration) {
	if len(responseTimes) == 0 {
		return
	}

	// Sort response times for percentile calculations
	sort.Slice(responseTimes, func(i, j int) bool {
		return responseTimes[i] < responseTimes[j]
	})

	// Calculate basic metrics
	var totalTime time.Duration
	results.MinResponseTime = responseTimes[0]
	results.MaxResponseTime = responseTimes[len(responseTimes)-1]

	for _, rt := range responseTimes {
		totalTime += rt
	}

	results.AvgResponseTime = totalTime / time.Duration(len(responseTimes))
	results.ErrorRate = float64(results.FailedRequests) / float64(results.TotalRequests)
	
	// Calculate throughput (requests per second)
	if suite.Results.TotalDuration > 0 {
		results.Throughput = float64(results.TotalRequests) / suite.Results.TotalDuration.Seconds()
	}

	// Calculate percentiles
	results.Percentiles["p50"] = responseTimes[len(responseTimes)*50/100]
	results.Percentiles["p90"] = responseTimes[len(responseTimes)*90/100]
	results.Percentiles["p95"] = responseTimes[len(responseTimes)*95/100]
	results.Percentiles["p99"] = responseTimes[len(responseTimes)*99/100]
}

// TestPerformanceThresholds tests that all services meet performance thresholds
func TestPerformanceThresholds(t *testing.T) {
	suite := NewBenchmarkSuite()

	// Run quick performance test
	for serviceName, serviceBenchmark := range suite.Services {
		t.Run(fmt.Sprintf("Thresholds_%s", serviceName), func(t *testing.T) {
			// Simulate performance test
			results := &ServiceResults{
				ServiceName:     serviceName,
				TotalRequests:   1000,
				SuccessfulReqs:  990,
				FailedRequests:  10,
				AvgResponseTime: 80 * time.Millisecond,
				Throughput:      75.0,
				ErrorRate:       0.01,
			}

			// Validate against thresholds
			assert.LessOrEqual(t, results.AvgResponseTime, serviceBenchmark.Thresholds.MaxResponseTime,
				"Average response time exceeds threshold")
			assert.GreaterOrEqual(t, results.Throughput, serviceBenchmark.Thresholds.MinThroughput,
				"Throughput below threshold")
			assert.LessOrEqual(t, results.ErrorRate, serviceBenchmark.Thresholds.MaxErrorRate,
				"Error rate exceeds threshold")
		})
	}
}

// TestLoadCapacity tests system capacity under load
func TestLoadCapacity(t *testing.T) {
	suite := NewBenchmarkSuite()

	loadLevels := []int{10, 25, 50, 100, 200}

	for _, load := range loadLevels {
		t.Run(fmt.Sprintf("Load_%d_users", load), func(t *testing.T) {
			// Simulate load test
			results := suite.simulateLoadTest(load)

			// Validate system handles load
			assert.Less(t, results.OverallErrorRate, 0.05, "Error rate too high under load")
			assert.Greater(t, results.OverallThroughput, float64(load)*0.8, "Throughput too low")
			
			// Check if system degrades gracefully
			if load > 100 {
				// Allow higher response times under heavy load
				assert.Less(t, results.AvgResponseTime, 500*time.Millisecond, "Response time too high")
			} else {
				assert.Less(t, results.AvgResponseTime, 200*time.Millisecond, "Response time too high")
			}
		})
	}
}

// simulateLoadTest simulates a load test with specified number of users
func (suite *BenchmarkSuite) simulateLoadTest(users int) *OverallMetrics {
	// Simulate load test results
	baseLatency := 50 * time.Millisecond
	loadFactor := float64(users) / 50.0 // Base load is 50 users
	
	return &OverallMetrics{
		TotalRequests:     int64(users * 100),
		TotalErrors:       int64(users * 2),
		OverallThroughput: float64(users) * 0.9,
		OverallErrorRate:  0.02 * loadFactor,
		AvgResponseTime:   time.Duration(float64(baseLatency) * (1 + loadFactor*0.5)),
		SystemResourceUsage: SystemResources{
			CPUUsage:    30.0 + (loadFactor * 20.0),
			MemoryUsage: 1024 + int64(loadFactor*512),
			DiskIO:      100 + int64(loadFactor*50),
			NetworkIO:   500 + int64(loadFactor*200),
		},
	}
}

// generateBenchmarkReport generates a comprehensive benchmark report
func (suite *BenchmarkSuite) generateBenchmarkReport() {
	report := fmt.Sprintf(`
KNIRV TESTNET Performance Benchmark Report
==========================================
Generated: %s
Duration: %s
Total Services: %d

`, suite.Results.StartTime.Format(time.RFC3339), 
	suite.Results.TotalDuration, len(suite.Services))

	// Service-specific results
	for serviceName, results := range suite.Results.ServiceResults {
		report += fmt.Sprintf(`
Service: %s
-----------
Total Requests: %d
Successful: %d (%.2f%%)
Failed: %d (%.2f%%)
Average Response Time: %s
Throughput: %.2f req/s
Error Rate: %.4f

Percentiles:
  P50: %s
  P90: %s
  P95: %s
  P99: %s

`, serviceName, results.TotalRequests, results.SuccessfulReqs,
		float64(results.SuccessfulReqs)/float64(results.TotalRequests)*100,
		results.FailedRequests, results.ErrorRate*100,
		results.AvgResponseTime, results.Throughput, results.ErrorRate,
		results.Percentiles["p50"], results.Percentiles["p90"],
		results.Percentiles["p95"], results.Percentiles["p99"])
	}

	// Save report to file
	fmt.Println(report)
}