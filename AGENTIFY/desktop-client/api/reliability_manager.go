// reliability_manager.go - Reliability improvements with circuit breakers and retry mechanisms
package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// ReliabilityManager coordinates all reliability improvements
type ReliabilityManager struct {
	circuitBreakers map[string]*CircuitBreaker
	retryPolicies   map[string]*RetryPolicy
	healthCheckers  map[string]*HealthChecker
	errorHandler    *ErrorHandler
	enabled         bool
	mutex           sync.RWMutex
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name            string
	maxFailures     int
	timeout         time.Duration
	resetTimeout    time.Duration
	state           CircuitState
	failures        int
	lastFailureTime time.Time
	mutex           sync.RWMutex
	onStateChange   func(string, CircuitState, CircuitState)
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// String returns the string representation of CircuitState
func (cs CircuitState) String() string {
	switch cs {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	maxAttempts     int
	baseDelay       time.Duration
	maxDelay        time.Duration
	backoffFactor   float64
	retryableErrors []error
}

// HealthChecker monitors service health
type HealthChecker struct {
	name             string
	checkFunc        func() error
	interval         time.Duration
	timeout          time.Duration
	isHealthy        bool
	lastCheck        time.Time
	lastError        error
	consecutiveFails int
	maxFails         int
	stopChan         chan bool
	mutex            sync.RWMutex
}

// ErrorHandler provides comprehensive error handling
type ErrorHandler struct {
	errorCounts   map[string]int64
	errorPatterns map[string]ErrorPattern
	fallbackFuncs map[string]func() interface{}
	mutex         sync.RWMutex
}

// ErrorPattern defines error handling patterns
type ErrorPattern struct {
	Pattern    string        `json:"pattern"`
	Action     string        `json:"action"`
	Threshold  int           `json:"threshold"`
	TimeWindow time.Duration `json:"time_window"`
	Fallback   string        `json:"fallback"`
}

// CircuitBreakerStats provides circuit breaker statistics
type CircuitBreakerStats struct {
	Name         string        `json:"name"`
	State        string        `json:"state"`
	Failures     int           `json:"failures"`
	MaxFailures  int           `json:"max_failures"`
	LastFailure  time.Time     `json:"last_failure"`
	Timeout      time.Duration `json:"timeout"`
	ResetTimeout time.Duration `json:"reset_timeout"`
}

// HealthStatus represents the health status of a service
type HealthStatus struct {
	Name             string    `json:"name"`
	IsHealthy        bool      `json:"is_healthy"`
	LastCheck        time.Time `json:"last_check"`
	LastError        string    `json:"last_error"`
	ConsecutiveFails int       `json:"consecutive_fails"`
	MaxFails         int       `json:"max_fails"`
}

// NewReliabilityManager creates a new reliability manager
func NewReliabilityManager() *ReliabilityManager {
	return &ReliabilityManager{
		circuitBreakers: make(map[string]*CircuitBreaker),
		retryPolicies:   make(map[string]*RetryPolicy),
		healthCheckers:  make(map[string]*HealthChecker),
		errorHandler:    NewErrorHandler(),
		enabled:         true,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, maxFailures int, timeout, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:         name,
		maxFailures:  maxFailures,
		timeout:      timeout,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
		failures:     0,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Check if circuit is open
	if cb.state == CircuitOpen {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.setState(CircuitHalfOpen)
		} else {
			return nil, fmt.Errorf("circuit breaker %s is open", cb.name)
		}
	}

	// Execute the function
	result, err := fn()

	if err != nil {
		cb.onFailure()
		return nil, err
	}

	cb.onSuccess()
	return result, nil
}

// onFailure handles failure cases
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.setState(CircuitOpen)
	}
}

// onSuccess handles success cases
func (cb *CircuitBreaker) onSuccess() {
	cb.failures = 0
	if cb.state == CircuitHalfOpen {
		cb.setState(CircuitClosed)
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState CircuitState) {
	oldState := cb.state
	cb.state = newState

	log.Printf("Circuit breaker %s state changed: %s -> %s", cb.name, oldState.String(), newState.String())

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, oldState, newState)
	}
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return CircuitBreakerStats{
		Name:         cb.name,
		State:        cb.state.String(),
		Failures:     cb.failures,
		MaxFailures:  cb.maxFailures,
		LastFailure:  cb.lastFailureTime,
		Timeout:      cb.timeout,
		ResetTimeout: cb.resetTimeout,
	}
}

// NewRetryPolicy creates a new retry policy
func NewRetryPolicy(maxAttempts int, baseDelay, maxDelay time.Duration, backoffFactor float64) *RetryPolicy {
	return &RetryPolicy{
		maxAttempts:   maxAttempts,
		baseDelay:     baseDelay,
		maxDelay:      maxDelay,
		backoffFactor: backoffFactor,
		retryableErrors: []error{
			errors.New("connection refused"),
			errors.New("timeout"),
			errors.New("temporary failure"),
		},
	}
}

// Execute executes a function with retry logic
func (rp *RetryPolicy) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	var lastErr error

	for attempt := 1; attempt <= rp.maxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !rp.isRetryable(err) {
			return nil, err
		}

		// Don't retry on last attempt
		if attempt == rp.maxAttempts {
			break
		}

		// Calculate delay with exponential backoff
		delay := rp.calculateDelay(attempt)

		log.Printf("Retry attempt %d/%d failed: %v. Retrying in %v", attempt, rp.maxAttempts, err, delay)

		// Wait for delay or context cancellation
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("all %d retry attempts failed, last error: %w", rp.maxAttempts, lastErr)
}

// isRetryable checks if an error is retryable
func (rp *RetryPolicy) isRetryable(err error) bool {
	for _, retryableErr := range rp.retryableErrors {
		if err.Error() == retryableErr.Error() {
			return true
		}
	}
	return false
}

// calculateDelay calculates the delay for a retry attempt
func (rp *RetryPolicy) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(rp.baseDelay) * float64(attempt) * rp.backoffFactor)
	if delay > rp.maxDelay {
		delay = rp.maxDelay
	}
	return delay
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(name string, checkFunc func() error, interval, timeout time.Duration, maxFails int) *HealthChecker {
	return &HealthChecker{
		name:      name,
		checkFunc: checkFunc,
		interval:  interval,
		timeout:   timeout,
		isHealthy: true,
		maxFails:  maxFails,
		stopChan:  make(chan bool),
	}
}

// Start starts the health checker
func (hc *HealthChecker) Start() {
	go func() {
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				hc.performCheck()
			case <-hc.stopChan:
				return
			}
		}
	}()
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
}

// performCheck performs a health check
func (hc *HealthChecker) performCheck() {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- hc.checkFunc()
	}()

	select {
	case err := <-done:
		hc.lastCheck = time.Now()
		if err != nil {
			hc.lastError = err
			hc.consecutiveFails++
			if hc.consecutiveFails >= hc.maxFails {
				hc.isHealthy = false
			}
			log.Printf("Health check failed for %s: %v (consecutive fails: %d)", hc.name, err, hc.consecutiveFails)
		} else {
			hc.lastError = nil
			hc.consecutiveFails = 0
			hc.isHealthy = true
		}
	case <-ctx.Done():
		hc.lastCheck = time.Now()
		hc.lastError = ctx.Err()
		hc.consecutiveFails++
		if hc.consecutiveFails >= hc.maxFails {
			hc.isHealthy = false
		}
		log.Printf("Health check timeout for %s (consecutive fails: %d)", hc.name, hc.consecutiveFails)
	}
}

// GetStatus returns the current health status
func (hc *HealthChecker) GetStatus() HealthStatus {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	lastErrorStr := ""
	if hc.lastError != nil {
		lastErrorStr = hc.lastError.Error()
	}

	return HealthStatus{
		Name:             hc.name,
		IsHealthy:        hc.isHealthy,
		LastCheck:        hc.lastCheck,
		LastError:        lastErrorStr,
		ConsecutiveFails: hc.consecutiveFails,
		MaxFails:         hc.maxFails,
	}
}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		errorCounts:   make(map[string]int64),
		errorPatterns: make(map[string]ErrorPattern),
		fallbackFuncs: make(map[string]func() interface{}),
	}
}

// HandleError handles an error with fallback mechanisms
func (eh *ErrorHandler) HandleError(errorType string, err error) interface{} {
	eh.mutex.Lock()
	defer eh.mutex.Unlock()

	// Increment error count
	eh.errorCounts[errorType]++

	// Check if we have a pattern for this error type
	pattern, exists := eh.errorPatterns[errorType]
	if exists {
		// Check if threshold is exceeded
		if eh.errorCounts[errorType] >= int64(pattern.Threshold) {
			log.Printf("Error threshold exceeded for %s: %d >= %d", errorType, eh.errorCounts[errorType], pattern.Threshold)

			// Execute fallback if available
			if fallbackFunc, hasFallback := eh.fallbackFuncs[pattern.Fallback]; hasFallback {
				return fallbackFunc()
			}
		}
	}

	log.Printf("Error handled: %s - %v", errorType, err)
	return nil
}

// RegisterErrorPattern registers an error pattern
func (eh *ErrorHandler) RegisterErrorPattern(errorType string, pattern ErrorPattern) {
	eh.mutex.Lock()
	defer eh.mutex.Unlock()
	eh.errorPatterns[errorType] = pattern
}

// RegisterFallback registers a fallback function
func (eh *ErrorHandler) RegisterFallback(name string, fallbackFunc func() interface{}) {
	eh.mutex.Lock()
	defer eh.mutex.Unlock()
	eh.fallbackFuncs[name] = fallbackFunc
}

// GetErrorStats returns error statistics
func (eh *ErrorHandler) GetErrorStats() map[string]interface{} {
	eh.mutex.RLock()
	defer eh.mutex.RUnlock()

	return map[string]interface{}{
		"error_counts":   eh.errorCounts,
		"error_patterns": len(eh.errorPatterns),
		"fallbacks":      len(eh.fallbackFuncs),
	}
}

// AddCircuitBreaker adds a circuit breaker
func (rm *ReliabilityManager) AddCircuitBreaker(name string, cb *CircuitBreaker) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.circuitBreakers[name] = cb
}

// GetCircuitBreaker returns a circuit breaker by name
func (rm *ReliabilityManager) GetCircuitBreaker(name string) *CircuitBreaker {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return rm.circuitBreakers[name]
}

// AddRetryPolicy adds a retry policy
func (rm *ReliabilityManager) AddRetryPolicy(name string, policy *RetryPolicy) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.retryPolicies[name] = policy
}

// GetRetryPolicy returns a retry policy by name
func (rm *ReliabilityManager) GetRetryPolicy(name string) *RetryPolicy {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return rm.retryPolicies[name]
}

// AddHealthChecker adds a health checker
func (rm *ReliabilityManager) AddHealthChecker(name string, hc *HealthChecker) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.healthCheckers[name] = hc
	hc.Start()
}

// GetHealthChecker returns a health checker by name
func (rm *ReliabilityManager) GetHealthChecker(name string) *HealthChecker {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return rm.healthCheckers[name]
}

// GetErrorHandler returns the error handler
func (rm *ReliabilityManager) GetErrorHandler() *ErrorHandler {
	return rm.errorHandler
}

// ExecuteWithReliability executes a function with full reliability protection
func (rm *ReliabilityManager) ExecuteWithReliability(ctx context.Context, serviceName string, fn func() (interface{}, error)) (interface{}, error) {
	if !rm.enabled {
		return fn()
	}

	// Get circuit breaker
	cb := rm.GetCircuitBreaker(serviceName)
	if cb == nil {
		// Create default circuit breaker
		cb = NewCircuitBreaker(serviceName, 5, 30*time.Second, 60*time.Second)
		rm.AddCircuitBreaker(serviceName, cb)
	}

	// Get retry policy
	retryPolicy := rm.GetRetryPolicy(serviceName)
	if retryPolicy == nil {
		// Create default retry policy
		retryPolicy = NewRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0)
		rm.AddRetryPolicy(serviceName, retryPolicy)
	}

	// Execute with circuit breaker and retry
	return cb.Execute(func() (interface{}, error) {
		return retryPolicy.Execute(ctx, fn)
	})
}

// GetStats returns comprehensive reliability statistics
func (rm *ReliabilityManager) GetStats() map[string]interface{} {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	circuitBreakerStats := make(map[string]CircuitBreakerStats)
	for name, cb := range rm.circuitBreakers {
		circuitBreakerStats[name] = cb.GetStats()
	}

	healthStatuses := make(map[string]HealthStatus)
	for name, hc := range rm.healthCheckers {
		healthStatuses[name] = hc.GetStatus()
	}

	return map[string]interface{}{
		"enabled":          rm.enabled,
		"circuit_breakers": circuitBreakerStats,
		"health_checkers":  healthStatuses,
		"retry_policies":   len(rm.retryPolicies),
		"error_stats":      rm.errorHandler.GetErrorStats(),
		"timestamp":        time.Now().UTC(),
	}
}

// Enable enables reliability features
func (rm *ReliabilityManager) Enable() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.enabled = true
	log.Println("Reliability manager enabled")
}

// Disable disables reliability features
func (rm *ReliabilityManager) Disable() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.enabled = false
	log.Println("Reliability manager disabled")
}

// Stop stops all reliability components
func (rm *ReliabilityManager) Stop() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	for _, hc := range rm.healthCheckers {
		hc.Stop()
	}

	log.Println("Reliability manager stopped")
}

// SetupDefaultReliability sets up default reliability configurations
func (rm *ReliabilityManager) SetupDefaultReliability() {
	// Add default circuit breakers
	rm.AddCircuitBreaker("database", NewCircuitBreaker("database", 5, 30*time.Second, 60*time.Second))
	rm.AddCircuitBreaker("llm_service", NewCircuitBreaker("llm_service", 3, 10*time.Second, 30*time.Second))
	rm.AddCircuitBreaker("external_api", NewCircuitBreaker("external_api", 10, 60*time.Second, 120*time.Second))

	// Add default retry policies
	rm.AddRetryPolicy("database", NewRetryPolicy(3, 100*time.Millisecond, 2*time.Second, 2.0))
	rm.AddRetryPolicy("llm_service", NewRetryPolicy(2, 500*time.Millisecond, 5*time.Second, 1.5))
	rm.AddRetryPolicy("external_api", NewRetryPolicy(5, 1*time.Second, 30*time.Second, 2.0))

	// Add default health checkers
	dbHealthChecker := NewHealthChecker("database", func() error {
		// Simplified database health check
		return nil // In real implementation, ping database
	}, 30*time.Second, 5*time.Second, 3)
	rm.AddHealthChecker("database", dbHealthChecker)

	// Register default error patterns
	rm.errorHandler.RegisterErrorPattern("database_error", ErrorPattern{
		Pattern:    "database connection failed",
		Action:     "fallback",
		Threshold:  5,
		TimeWindow: 5 * time.Minute,
		Fallback:   "cache_fallback",
	})

	// Register default fallbacks
	rm.errorHandler.RegisterFallback("cache_fallback", func() interface{} {
		return map[string]interface{}{
			"status":  "degraded",
			"message": "Using cached data due to database issues",
		}
	})

	log.Println("Default reliability configurations set up")
}
