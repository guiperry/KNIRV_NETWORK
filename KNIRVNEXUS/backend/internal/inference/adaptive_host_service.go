package inference

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	dataengine "backend-server/internal/data-engine"
	"backend-server/pkg/host"
)

// AdaptiveHostService extends the inference service with host integration
type AdaptiveHostService struct {
	*InferenceService

	// Host integration
	hostController *host.HostController

	// Data engine integration
	dataEngine *dataengine.BuntDBDataEngine

	// Fine-tuning components
	fineTuner     *FineTuningManager
	modelRegistry *ModelRegistry

	// Adaptive features
	resourceMonitor *ResourceMonitor
	loadBalancer    *InferenceLoadBalancer

	// Configuration
	config AdaptiveHostConfig

	// State management
	isRunning bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// AdaptiveHostConfig contains configuration for the adaptive host service
type AdaptiveHostConfig struct {
	// Host monitoring
	EnableHostMonitoring bool          `yaml:"enable_host_monitoring"`
	MonitoringInterval   time.Duration `yaml:"monitoring_interval"`

	// Resource management
	MaxCPUUsage    float64 `yaml:"max_cpu_usage"`
	MaxMemoryUsage float64 `yaml:"max_memory_usage"`
	AutoScaling    bool    `yaml:"auto_scaling"`

	// Model management
	EnableFineTuning   bool    `yaml:"enable_fine_tuning"`
	ModelCacheSize     int     `yaml:"model_cache_size"`
	ModelSwapThreshold float64 `yaml:"model_swap_threshold"`

	// Data engine integration
	EnableDataLogging bool          `yaml:"enable_data_logging"`
	MetricsInterval   time.Duration `yaml:"metrics_interval"`

	// Load balancing
	EnableLoadBalancing   bool          `yaml:"enable_load_balancing"`
	MaxConcurrentRequests int           `yaml:"max_concurrent_requests"`
	RequestTimeout        time.Duration `yaml:"request_timeout"`
}

// ResourceMonitor monitors system resources and inference performance
type ResourceMonitor struct {
	hostController *host.HostController
	dataEngine     *dataengine.BuntDBDataEngine

	// Metrics
	cpuUsage     float64
	memoryUsage  float64
	requestCount int64
	errorCount   int64
	avgLatency   time.Duration

	// Thresholds
	cpuThreshold    float64
	memoryThreshold float64

	mu sync.RWMutex
}

// InferenceLoadBalancer manages load balancing across inference objects
type InferenceLoadBalancer struct {
	objects       map[string]*ModelInstance
	requestQueue  chan *InferenceRequest
	responseQueue chan *InferenceResponse

	maxConcurrent  int
	activeRequests int

	mu sync.RWMutex
}

// ModelInstance represents a model instance with performance metrics
type ModelInstance struct {
	Name         string
	Provider     string
	IsAvailable  bool
	LoadFactor   float64
	AvgLatency   time.Duration
	ErrorRate    float64
	LastUsed     time.Time
	RequestCount int64

	// Resource usage
	CPUUsage    float64
	MemoryUsage uint64
}

// InferenceRequest represents an inference request
type InferenceRequest struct {
	ID          string
	ModelName   string
	Prompt      string
	Instruction string
	Context     context.Context
	StartTime   time.Time

	// Response channel
	ResponseChan chan *InferenceResponse
}

// InferenceResponse represents an inference response
type InferenceResponse struct {
	ID         string
	Result     string
	Error      error
	Latency    time.Duration
	ModelUsed  string
	TokensUsed int

	// Performance metrics
	CPUUsage    float64
	MemoryUsage uint64
}

// NewAdaptiveHostService creates a new adaptive host service
func NewAdaptiveHostService(config AdaptiveHostConfig, hostController *host.HostController, dataEngine *dataengine.BuntDBDataEngine) (*AdaptiveHostService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create base inference service
	baseService, err := NewInferenceService(nil) // Pass nil for now, will integrate later
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create base inference service: %w", err)
	}

	service := &AdaptiveHostService{
		InferenceService: baseService,
		hostController:   hostController,
		dataEngine:       dataEngine,
		config:           config,
		ctx:              ctx,
		cancel:           cancel,
	}

	// Initialize components
	if err := service.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return service, nil
}

// initializeComponents initializes all service components
func (s *AdaptiveHostService) initializeComponents() error {
	// Initialize resource monitor
	if s.config.EnableHostMonitoring {
		s.resourceMonitor = &ResourceMonitor{
			hostController:  s.hostController,
			dataEngine:      s.dataEngine,
			cpuThreshold:    s.config.MaxCPUUsage,
			memoryThreshold: s.config.MaxMemoryUsage,
		}
	}

	// Initialize load balancer
	if s.config.EnableLoadBalancing {
		s.loadBalancer = &InferenceLoadBalancer{
			objects:       make(map[string]*ModelInstance),
			requestQueue:  make(chan *InferenceRequest, s.config.MaxConcurrentRequests),
			responseQueue: make(chan *InferenceResponse, s.config.MaxConcurrentRequests),
			maxConcurrent: s.config.MaxConcurrentRequests,
		}
	}

	// Initialize fine-tuning manager
	if s.config.EnableFineTuning {
		var err error
		s.fineTuner, err = NewFineTuningManager(s.dataEngine)
		if err != nil {
			return fmt.Errorf("failed to create fine-tuning manager: %w", err)
		}
	}

	// Initialize model registry
	s.modelRegistry = NewModelRegistry(s.config.ModelCacheSize)

	return nil
}

// Start starts the adaptive host service
func (s *AdaptiveHostService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("adaptive host service is already running")
	}

	// Start base inference service
	if err := s.InferenceService.Start(); err != nil {
		return fmt.Errorf("failed to start base inference service: %w", err)
	}

	// Start resource monitoring
	if s.resourceMonitor != nil {
		go s.resourceMonitoringLoop()
	}

	// Start load balancer
	if s.loadBalancer != nil {
		go s.loadBalancingLoop()
	}

	// Start metrics collection
	if s.config.EnableDataLogging {
		go s.metricsCollectionLoop()
	}

	// Start model management
	go s.modelManagementLoop()

	s.isRunning = true
	log.Println("AdaptiveHostService: Started successfully")

	return nil
}

// modelManagementLoop manages model lifecycle and performance monitoring
func (s *AdaptiveHostService) modelManagementLoop() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.performModelHealthCheck()
		}
	}
}

// performModelHealthCheck checks model performance and switches if needed
func (s *AdaptiveHostService) performModelHealthCheck() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if current model is performing well
	// This is a placeholder for actual model performance monitoring
	log.Println("AdaptiveHostService: Performing model health check")

	// TODO: Implement actual model performance metrics
	// - Response time monitoring
	// - Error rate tracking
	// - Quality assessment
	// - Resource usage monitoring
}

// Stop stops the adaptive host service
func (s *AdaptiveHostService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	// Cancel context to stop all goroutines
	s.cancel()

	// Stop base inference service
	if err := s.InferenceService.Stop(); err != nil {
		log.Printf("Error stopping base inference service: %v", err)
	}

	// Stop fine-tuning manager
	if s.fineTuner != nil {
		s.fineTuner.Stop()
	}

	s.isRunning = false
	log.Println("AdaptiveHostService: Stopped successfully")

	return nil
}

// GenerateTextAdaptive generates text with adaptive resource management
func (s *AdaptiveHostService) GenerateTextAdaptive(modelName, prompt, instruction string) (string, error) {
	s.mu.RLock()
	if !s.isRunning {
		s.mu.RUnlock()
		return "", fmt.Errorf("adaptive host service is not running")
	}
	s.mu.RUnlock()

	// Create inference request
	request := &InferenceRequest{
		ID:           fmt.Sprintf("req-%d", time.Now().UnixNano()),
		ModelName:    modelName,
		Prompt:       prompt,
		Instruction:  instruction,
		Context:      s.ctx,
		StartTime:    time.Now(),
		ResponseChan: make(chan *InferenceResponse, 1),
	}

	// Check resource availability
	if s.resourceMonitor != nil {
		if err := s.checkResourceAvailability(); err != nil {
			return "", fmt.Errorf("resource check failed: %w", err)
		}
	}

	// Route through load balancer if enabled
	if s.loadBalancer != nil {
		return s.processRequestWithLoadBalancing(request)
	}

	// Direct processing
	return s.processRequestDirect(request)
}

// processRequestWithLoadBalancing processes request through load balancer
func (s *AdaptiveHostService) processRequestWithLoadBalancing(request *InferenceRequest) (string, error) {
	// Add to request queue
	select {
	case s.loadBalancer.requestQueue <- request:
		// Request queued successfully
	case <-time.After(s.config.RequestTimeout):
		return "", fmt.Errorf("request timeout: queue is full")
	}

	// Wait for response
	select {
	case response := <-request.ResponseChan:
		if response.Error != nil {
			return "", response.Error
		}

		// Log metrics
		s.logInferenceMetrics(response)

		return response.Result, nil
	case <-time.After(s.config.RequestTimeout):
		return "", fmt.Errorf("request timeout: no response received")
	}
}

// processRequestDirect processes request directly
func (s *AdaptiveHostService) processRequestDirect(request *InferenceRequest) (string, error) {
	startTime := time.Now()

	// Get system metrics before inference
	var beforeCPU, beforeMemory float64
	if s.resourceMonitor != nil {
		beforeCPU = s.resourceMonitor.cpuUsage
		beforeMemory = s.resourceMonitor.memoryUsage
	}

	// Perform inference
	result, err := s.InferenceService.GenerateText(request.ModelName, request.Prompt, request.Instruction)
	if err != nil {
		// Log error metrics
		s.logErrorMetrics(request.ModelName, err)
		return "", err
	}

	// Calculate metrics
	latency := time.Since(startTime)

	// Get system metrics after inference
	var afterCPU, afterMemory float64
	if s.resourceMonitor != nil {
		afterCPU = s.resourceMonitor.cpuUsage
		afterMemory = s.resourceMonitor.memoryUsage
	}

	// Create response
	response := &InferenceResponse{
		ID:          request.ID,
		Result:      result,
		Latency:     latency,
		ModelUsed:   request.ModelName,
		CPUUsage:    afterCPU - beforeCPU,
		MemoryUsage: uint64((afterMemory - beforeMemory) * 1024 * 1024), // Convert to bytes
	}

	// Log metrics
	s.logInferenceMetrics(response)

	return result, nil
}

// checkResourceAvailability checks if system resources are available for inference
func (s *AdaptiveHostService) checkResourceAvailability() error {
	s.resourceMonitor.mu.RLock()
	defer s.resourceMonitor.mu.RUnlock()

	if s.resourceMonitor.cpuUsage > s.resourceMonitor.cpuThreshold {
		return fmt.Errorf("CPU usage too high: %.2f%% > %.2f%%",
			s.resourceMonitor.cpuUsage, s.resourceMonitor.cpuThreshold)
	}

	if s.resourceMonitor.memoryUsage > s.resourceMonitor.memoryThreshold {
		return fmt.Errorf("memory usage too high: %.2f%% > %.2f%%",
			s.resourceMonitor.memoryUsage, s.resourceMonitor.memoryThreshold)
	}

	return nil
}

// logInferenceMetrics logs inference metrics to the data engine
func (s *AdaptiveHostService) logInferenceMetrics(response *InferenceResponse) {
	if s.dataEngine == nil || !s.config.EnableDataLogging {
		return
	}

	// Log latency metric
	s.dataEngine.ProcessMetricEvent(
		"inference-service",
		"inference_latency",
		float64(response.Latency.Milliseconds()),
		"milliseconds",
		map[string]string{
			"model":      response.ModelUsed,
			"request_id": response.ID,
		},
	)

	// Log CPU usage metric
	s.dataEngine.ProcessMetricEvent(
		"inference-service",
		"cpu_usage",
		response.CPUUsage,
		"percent",
		map[string]string{
			"model":      response.ModelUsed,
			"request_id": response.ID,
		},
	)

	// Log memory usage metric
	s.dataEngine.ProcessMetricEvent(
		"inference-service",
		"memory_usage",
		float64(response.MemoryUsage),
		"bytes",
		map[string]string{
			"model":      response.ModelUsed,
			"request_id": response.ID,
		},
	)
}

// logErrorMetrics logs error metrics to the data engine
func (s *AdaptiveHostService) logErrorMetrics(modelName string, err error) {
	if s.dataEngine == nil || !s.config.EnableDataLogging {
		return
	}

	// Log error count metric
	s.dataEngine.ProcessMetricEvent(
		"inference-service",
		"error_count",
		1.0,
		"count",
		map[string]string{
			"model":      modelName,
			"error_type": fmt.Sprintf("%T", err),
		},
	)

	// Log error alert
	s.dataEngine.ProcessAlertEvent(
		"Inference Error",
		fmt.Sprintf("Inference failed for model %s: %v", modelName, err),
		"warning",
		"inference-service",
		map[string]string{
			"model": modelName,
		},
	)
}

// resourceMonitoringLoop monitors system resources
func (s *AdaptiveHostService) resourceMonitoringLoop() {
	ticker := time.NewTicker(s.config.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.updateResourceMetrics()
		}
	}
}

// updateResourceMetrics updates resource usage metrics
func (s *AdaptiveHostService) updateResourceMetrics() {
	if s.resourceMonitor == nil {
		return
	}

	// Get system info from host controller
	systemInfo, err := s.hostController.GetSystemInfo()
	if err != nil {
		log.Printf("Error getting system info: %v", err)
		return
	}

	s.resourceMonitor.mu.Lock()
	if systemInfo.CPUInfo != nil {
		s.resourceMonitor.cpuUsage = systemInfo.CPUInfo.Usage
	}
	if systemInfo.MemoryInfo != nil {
		s.resourceMonitor.memoryUsage = systemInfo.MemoryInfo.Usage
	}
	s.resourceMonitor.mu.Unlock()

	// Log metrics to data engine
	if s.config.EnableDataLogging && s.dataEngine != nil {
		s.dataEngine.ProcessMetricEvent(
			"adaptive-host",
			"cpu_usage",
			s.resourceMonitor.cpuUsage,
			"percent",
			map[string]string{
				"component": "inference-service",
			},
		)

		s.dataEngine.ProcessMetricEvent(
			"adaptive-host",
			"memory_usage",
			s.resourceMonitor.memoryUsage,
			"percent",
			map[string]string{
				"component": "inference-service",
			},
		)
	}
}

// loadBalancingLoop processes load balancing requests
func (s *AdaptiveHostService) loadBalancingLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case request := <-s.loadBalancer.requestQueue:
			go s.processLoadBalancedRequest(request)
		}
	}
}

// processLoadBalancedRequest processes a request through load balancing
func (s *AdaptiveHostService) processLoadBalancedRequest(request *InferenceRequest) {
	// Check if we can accept more requests
	s.loadBalancer.mu.Lock()
	if s.loadBalancer.activeRequests >= s.loadBalancer.maxConcurrent {
		s.loadBalancer.mu.Unlock()

		// Send error response
		request.ResponseChan <- &InferenceResponse{
			ID:    request.ID,
			Error: fmt.Errorf("too many concurrent requests"),
		}
		return
	}
	s.loadBalancer.activeRequests++
	s.loadBalancer.mu.Unlock()

	// Process the request
	result, err := s.processRequestDirect(request)

	// Create response
	response := &InferenceResponse{
		ID:        request.ID,
		Result:    result,
		Error:     err,
		Latency:   time.Since(request.StartTime),
		ModelUsed: request.ModelName,
	}

	// Send response
	request.ResponseChan <- response

	// Decrement active requests
	s.loadBalancer.mu.Lock()
	s.loadBalancer.activeRequests--
	s.loadBalancer.mu.Unlock()
}

// metricsCollectionLoop collects and reports metrics
func (s *AdaptiveHostService) metricsCollectionLoop() {
	ticker := time.NewTicker(s.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.collectAndReportMetrics()
		}
	}
}

// collectAndReportMetrics collects and reports service metrics
func (s *AdaptiveHostService) collectAndReportMetrics() {
	if s.dataEngine == nil {
		return
	}

	// Collect model registry metrics
	objects := s.modelRegistry.ListModels()

	for _, model := range objects {
		// Report model metrics
		s.dataEngine.ProcessMetricEvent(
			"model-registry",
			"model_usage_count",
			float64(model.UsageCount),
			"count",
			map[string]string{
				"model":    model.Name,
				"provider": model.Provider,
				"status":   string(model.Status),
			},
		)

		s.dataEngine.ProcessMetricEvent(
			"model-registry",
			"model_health_score",
			model.HealthScore,
			"score",
			map[string]string{
				"model":    model.Name,
				"provider": model.Provider,
			},
		)

		if model.AvgLatency > 0 {
			s.dataEngine.ProcessMetricEvent(
				"model-registry",
				"model_avg_latency",
				float64(model.AvgLatency.Milliseconds()),
				"milliseconds",
				map[string]string{
					"model":    model.Name,
					"provider": model.Provider,
				},
			)
		}

		s.dataEngine.ProcessMetricEvent(
			"model-registry",
			"model_error_rate",
			model.ErrorRate,
			"percent",
			map[string]string{
				"model":    model.Name,
				"provider": model.Provider,
			},
		)
	}

	// Report load balancer metrics
	if s.loadBalancer != nil {
		s.loadBalancer.mu.RLock()
		activeRequests := s.loadBalancer.activeRequests
		s.loadBalancer.mu.RUnlock()

		s.dataEngine.ProcessMetricEvent(
			"load-balancer",
			"active_requests",
			float64(activeRequests),
			"count",
			map[string]string{
				"component": "inference-service",
			},
		)
	}
}
