package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// APIClient is a client for the KNIRVCHAIN API
type APIClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	retries    int
	logger     *logrus.Logger

	// Circuit breaker
	circuitBreaker     map[string]*circuitBreakerState
	circuitBreakerLock sync.RWMutex
}

// circuitBreakerState tracks the state of a circuit breaker for an endpoint
type circuitBreakerState struct {
	failures       int
	lastFailure    time.Time
	isOpen         bool
	cooldownPeriod time.Duration
}

// APIClientOption is a function that configures an APIClient
type APIClientOption func(*APIClient)

// WithTimeout sets the timeout for the API client
func WithTimeout(timeout time.Duration) APIClientOption {
	return func(c *APIClient) {
		c.timeout = timeout
	}
}

// WithRetries sets the number of retries for the API client
func WithRetries(retries int) APIClientOption {
	return func(c *APIClient) {
		c.retries = retries
	}
}

// WithLogger sets the logger for the API client
func WithLogger(logger *logrus.Logger) APIClientOption {
	return func(c *APIClient) {
		c.logger = logger
	}
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string, options ...APIClientOption) *APIClient {
	client := &APIClient{
		baseURL:        baseURL,
		timeout:        30 * time.Second,
		retries:        3,
		logger:         logrus.New(),
		circuitBreaker: make(map[string]*circuitBreakerState),
	}

	// Apply options
	for _, option := range options {
		option(client)
	}

	// Configure HTTP client
	client.httpClient = &http.Client{
		Timeout: client.timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	return client
}

// Get performs a GET request with context
func (c *APIClient) Get(ctx context.Context, path string, result interface{}) error {
	return c.request(ctx, http.MethodGet, path, nil, result)
}

// Post performs a POST request with context
func (c *APIClient) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.request(ctx, http.MethodPost, path, body, result)
}

// request performs an HTTP request with retry logic and circuit breaker
func (c *APIClient) request(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	// Check circuit breaker
	if !c.isCircuitClosed(path) {
		return fmt.Errorf("circuit breaker is open for %s", path)
	}

	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		// If this is a retry, wait with exponential backoff
		if attempt > 0 {
			backoffDuration := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			c.logger.Debugf("Retrying request to %s (attempt %d/%d) after %v", path, attempt, c.retries, backoffDuration)

			select {
			case <-time.After(backoffDuration):
				// Continue with retry
			case <-ctx.Done():
				return fmt.Errorf("request canceled: %w", ctx.Err())
			}
		}

		// Create URL
		url := fmt.Sprintf("%s%s", c.baseURL, path)

		// Create request body
		var bodyReader io.Reader
		if body != nil {
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(bodyBytes)

			// Log request details
			c.logger.Debugf("Request: %s %s, Body: %s", method, url, string(bodyBytes))
		} else {
			c.logger.Debugf("Request: %s %s", method, url)
		}

		// Create request with context
		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		// Record start time for logging
		startTime := time.Now()

		// Send request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			c.recordFailure(path)

			// Log error
			c.logger.Errorf("Request failed: %s %s, Error: %v", method, url, err)

			// Check if we should retry
			if attempt < c.retries {
				continue
			}
			return lastErr
		}

		// Log response time
		c.logger.Debugf("Response: %s %s, Status: %d, Time: %v",
			method, url, resp.StatusCode, time.Since(startTime))

		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// Read error response
			errorBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				lastErr = fmt.Errorf("request failed with status %d", resp.StatusCode)
			} else {
				lastErr = fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(errorBytes))
				c.logger.Errorf("Error response: %s", string(errorBytes))
			}

			c.recordFailure(path)

			// Check if we should retry based on status code
			if attempt < c.retries && (resp.StatusCode == 429 || resp.StatusCode >= 500) {
				continue
			}
			return lastErr
		}

		// Reset circuit breaker on success
		c.resetCircuitBreaker(path)

		// Parse response
		if result != nil {
			responseBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body: %w", err)
			}

			// Log response body for debugging
			c.logger.Debugf("Response body: %s", string(responseBody))

			if err := json.Unmarshal(responseBody, result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
		}

		return nil
	}

	return lastErr
}

// isCircuitClosed checks if the circuit breaker is closed for the given path
func (c *APIClient) isCircuitClosed(path string) bool {
	c.circuitBreakerLock.RLock()
	defer c.circuitBreakerLock.RUnlock()

	state, exists := c.circuitBreaker[path]
	if !exists {
		return true
	}

	if !state.isOpen {
		return true
	}

	// Check if cooldown period has passed
	if time.Since(state.lastFailure) > state.cooldownPeriod {
		// Allow one request to try again
		c.logger.Infof("Circuit breaker cooldown period passed for %s, allowing request", path)
		return true
	}

	return false
}

// recordFailure records a failure for the circuit breaker
func (c *APIClient) recordFailure(path string) {
	c.circuitBreakerLock.Lock()
	defer c.circuitBreakerLock.Unlock()

	state, exists := c.circuitBreaker[path]
	if !exists {
		state = &circuitBreakerState{
			cooldownPeriod: 30 * time.Second,
		}
		c.circuitBreaker[path] = state
	}

	state.failures++
	state.lastFailure = time.Now()

	// Open circuit after 5 consecutive failures
	if state.failures >= 5 {
		if !state.isOpen {
			c.logger.Warnf("Circuit breaker opened for %s after %d failures", path, state.failures)
			state.isOpen = true
		}
	}
}

// resetCircuitBreaker resets the circuit breaker for the given path
func (c *APIClient) resetCircuitBreaker(path string) {
	c.circuitBreakerLock.Lock()
	defer c.circuitBreakerLock.Unlock()

	if state, exists := c.circuitBreaker[path]; exists {
		if state.isOpen || state.failures > 0 {
			c.logger.Infof("Circuit breaker reset for %s", path)
		}
		state.failures = 0
		state.isOpen = false
	}
}
