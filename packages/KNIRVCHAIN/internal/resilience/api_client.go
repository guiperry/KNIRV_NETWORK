package resilience

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

type APIClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	retries    int
	logger     *logrus.Logger

	circuitBreaker     map[string]*circuitBreakerState
	circuitBreakerLock sync.RWMutex

	eventBus *EventBus
}

type circuitBreakerState struct {
	failures       int
	lastFailure    time.Time
	isOpen         bool
	cooldownPeriod time.Duration
}

type APIClientOption func(*APIClient)

func WithTimeout(timeout time.Duration) APIClientOption {
	return func(c *APIClient) {
		c.timeout = timeout
	}
}

func WithRetries(retries int) APIClientOption {
	return func(c *APIClient) {
		c.retries = retries
	}
}

func WithLogger(logger *logrus.Logger) APIClientOption {
	return func(c *APIClient) {
		c.logger = logger
	}
}

func WithEventBus(eventBus *EventBus) APIClientOption {
	return func(c *APIClient) {
		c.eventBus = eventBus
	}
}

func NewAPIClient(baseURL string, options ...APIClientOption) *APIClient {
	client := &APIClient{
		baseURL:        baseURL,
		timeout:        30 * time.Second,
		retries:        3,
		logger:         logrus.New(),
		circuitBreaker: make(map[string]*circuitBreakerState),
	}

	for _, option := range options {
		option(client)
	}

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

func (c *APIClient) Get(ctx context.Context, path string, result interface{}) error {
	return c.request(ctx, http.MethodGet, path, nil, result)
}

func (c *APIClient) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.request(ctx, http.MethodPost, path, body, result)
}

func (c *APIClient) Put(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.request(ctx, http.MethodPut, path, body, result)
}

func (c *APIClient) Delete(ctx context.Context, path string, result interface{}) error {
	return c.request(ctx, http.MethodDelete, path, nil, result)
}

func (c *APIClient) request(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	if !c.isCircuitClosed(path) {
		c.publishCircuitEvent(path, true)
		return fmt.Errorf("circuit breaker is open for %s", path)
	}

	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			backoffDuration := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			c.logger.Debugf("Retrying request to %s (attempt %d/%d) after %v", path, attempt, c.retries, backoffDuration)

			select {
			case <-time.After(backoffDuration):
			case <-ctx.Done():
				return fmt.Errorf("request canceled: %w", ctx.Err())
			}
		}

		url := fmt.Sprintf("%s%s", c.baseURL, path)

		var bodyReader io.Reader
		if body != nil {
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(bodyBytes)
			c.logger.Debugf("Request: %s %s, Body: %s", method, url, string(bodyBytes))
		} else {
			c.logger.Debugf("Request: %s %s", method, url)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		startTime := time.Now()

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			c.recordFailure(path)
			c.logger.Errorf("Request failed: %s %s, Error: %v", method, url, err)

			if attempt < c.retries {
				continue
			}
			return lastErr
		}

		c.logger.Debugf("Response: %s %s, Status: %d, Time: %v",
			method, url, resp.StatusCode, time.Since(startTime))

		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errorBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				lastErr = fmt.Errorf("request failed with status %d", resp.StatusCode)
			} else {
				lastErr = fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(errorBytes))
				c.logger.Errorf("Error response: %s", string(errorBytes))
			}

			c.recordFailure(path)

			if attempt < c.retries && (resp.StatusCode == 429 || resp.StatusCode >= 500) {
				continue
			}
			return lastErr
		}

		c.resetCircuitBreaker(path)

		if result != nil {
			responseBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body: %w", err)
			}

			c.logger.Debugf("Response body: %s", string(responseBody))

			if err := json.Unmarshal(responseBody, result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
		}

		return nil
	}

	return lastErr
}

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

	if time.Since(state.lastFailure) > state.cooldownPeriod {
		c.logger.Infof("Circuit breaker cooldown period passed for %s, allowing request", path)
		return true
	}

	return false
}

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

	if state.failures >= 5 {
		if !state.isOpen {
			c.logger.Warnf("Circuit breaker opened for %s after %d failures", path, state.failures)
			state.isOpen = true
			c.publishCircuitEvent(path, true)
		}
	}
}

func (c *APIClient) resetCircuitBreaker(path string) {
	c.circuitBreakerLock.Lock()
	defer c.circuitBreakerLock.Unlock()

	if state, exists := c.circuitBreaker[path]; exists {
		if state.isOpen || state.failures > 0 {
			c.logger.Infof("Circuit breaker reset for %s", path)
			c.publishCircuitEvent(path, false)
		}
		state.failures = 0
		state.isOpen = false
	}
}

func (c *APIClient) publishCircuitEvent(path string, isOpen bool) {
	if c.eventBus == nil {
		return
	}

	eventType := EventTypeCircuitBreakerOpen
	if !isOpen {
		eventType = EventTypeCircuitBreakerClosed
	}

	event := NewEvent(eventType, "APIClient", map[string]interface{}{
		"endpoint": path,
		"is_open":  isOpen,
	})
	c.eventBus.Publish(event)
}

func (c *APIClient) GetCircuitBreakerStatus(path string) (isOpen bool, failures int) {
	c.circuitBreakerLock.RLock()
	defer c.circuitBreakerLock.RUnlock()

	state, exists := c.circuitBreaker[path]
	if !exists {
		return false, 0
	}

	return state.isOpen, state.failures
}
