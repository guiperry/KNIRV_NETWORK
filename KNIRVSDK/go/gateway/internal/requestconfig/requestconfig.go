// Request configuration for KNIRV Gateway SDK

package requestconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RequestConfig holds configuration for HTTP requests
type RequestConfig struct {
	// Basic request configuration
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string

	// Service URLs
	BaseURL      string
	EconomicsURL string
	ServiceURLs  map[string]string

	// Authentication
	APIKey      string
	NRNContract string
	BasicAuth   *BasicAuth

	// Client configuration
	HTTPClient *http.Client
	Timeout    time.Duration

	// Retry configuration
	Retries    int
	RetryDelay time.Duration

	// Environment and debugging
	Environment string
	Debug       bool
	Verbose     bool

	// Advanced features
	RateLimiting           bool
	RequestsPerSecond      int
	CircuitBreaker         bool
	FailureThreshold       int
	CircuitBreakerTimeout  time.Duration
	MetricsEnabled         bool
	TracingEnabled         bool
	ServiceName            string
	HealthCheckEnabled     bool
	HealthCheckInterval    time.Duration
}

// BasicAuth holds basic authentication credentials
type BasicAuth struct {
	Username string
	Password string
}

// Build creates an HTTP request from the configuration
func (cfg *RequestConfig) Build(ctx context.Context) (*http.Request, error) {
	// Determine the base URL to use
	baseURL := cfg.getBaseURL()
	if baseURL == "" {
		return nil, fmt.Errorf("no base URL configured")
	}

	// Build the full URL
	fullURL, err := cfg.buildURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Prepare the request body
	var bodyReader *bytes.Reader
	if cfg.Body != nil {
		bodyBytes, err := cfg.marshalBody()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create the HTTP request
	var req *http.Request
	if bodyReader != nil {
		req, err = http.NewRequestWithContext(ctx, cfg.Method, fullURL, bodyReader)
	} else {
		req, err = http.NewRequestWithContext(ctx, cfg.Method, fullURL, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	cfg.setHeaders(req)

	// Set authentication
	cfg.setAuth(req)

	return req, nil
}

// getBaseURL determines which base URL to use based on the path
func (cfg *RequestConfig) getBaseURL() string {
	// If path starts with /economics, use economics URL
	if strings.HasPrefix(cfg.Path, "/economics") {
		if cfg.EconomicsURL != "" {
			return cfg.EconomicsURL
		}
	}

	// Otherwise use the main base URL
	if cfg.BaseURL != "" {
		return cfg.BaseURL
	}

	// Fallback to default development URLs
	if strings.HasPrefix(cfg.Path, "/economics") {
		return "http://localhost:8090"
	}
	return "http://localhost:8000"
}

// buildURL constructs the full URL with query parameters
func (cfg *RequestConfig) buildURL(baseURL string) (string, error) {
	// Parse the base URL
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Add the path
	u.Path = strings.TrimSuffix(u.Path, "/") + cfg.Path

	// Add query parameters
	if len(cfg.QueryParams) > 0 {
		q := u.Query()
		for key, value := range cfg.QueryParams {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}

// marshalBody converts the body to JSON bytes
func (cfg *RequestConfig) marshalBody() ([]byte, error) {
	if cfg.Body == nil {
		return nil, nil
	}

	// If it's already bytes, return as-is
	if bytes, ok := cfg.Body.([]byte); ok {
		return bytes, nil
	}

	// If it's a string, convert to bytes
	if str, ok := cfg.Body.(string); ok {
		return []byte(str), nil
	}

	// Otherwise, marshal as JSON
	return json.Marshal(cfg.Body)
}

// setHeaders sets the request headers
func (cfg *RequestConfig) setHeaders(req *http.Request) {
	// Set default headers
	req.Header.Set("User-Agent", "KNIRV-Gateway-SDK/1.0.0")
	req.Header.Set("Accept", "application/json")

	// Set Content-Type for requests with body
	if cfg.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}

	// Set API key header if provided
	if cfg.APIKey != "" {
		req.Header.Set("X-API-Key", cfg.APIKey)
	}

	// Set NRN contract header if provided
	if cfg.NRNContract != "" {
		req.Header.Set("X-NRN-Contract", cfg.NRNContract)
	}

	// Set service identification headers
	if cfg.ServiceName != "" {
		req.Header.Set("X-Service-Name", cfg.ServiceName)
	}

	// Set environment header
	if cfg.Environment != "" {
		req.Header.Set("X-Environment", cfg.Environment)
	}

	// Set debug headers
	if cfg.Debug {
		req.Header.Set("X-Debug", "true")
	}
	if cfg.Verbose {
		req.Header.Set("X-Verbose", "true")
	}
}

// setAuth sets authentication on the request
func (cfg *RequestConfig) setAuth(req *http.Request) {
	// Set basic authentication if provided
	if cfg.BasicAuth != nil {
		req.SetBasicAuth(cfg.BasicAuth.Username, cfg.BasicAuth.Password)
	}
}

// GetHTTPClient returns the HTTP client to use for requests
func (cfg *RequestConfig) GetHTTPClient() *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}

	// Create default client with timeout
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &http.Client{
		Timeout: timeout,
	}
}

// Clone creates a copy of the request configuration
func (cfg *RequestConfig) Clone() *RequestConfig {
	clone := &RequestConfig{
		Method:                 cfg.Method,
		Path:                   cfg.Path,
		Body:                   cfg.Body,
		BaseURL:                cfg.BaseURL,
		EconomicsURL:           cfg.EconomicsURL,
		APIKey:                 cfg.APIKey,
		NRNContract:            cfg.NRNContract,
		HTTPClient:             cfg.HTTPClient,
		Timeout:                cfg.Timeout,
		Retries:                cfg.Retries,
		RetryDelay:             cfg.RetryDelay,
		Environment:            cfg.Environment,
		Debug:                  cfg.Debug,
		Verbose:                cfg.Verbose,
		RateLimiting:           cfg.RateLimiting,
		RequestsPerSecond:      cfg.RequestsPerSecond,
		CircuitBreaker:         cfg.CircuitBreaker,
		FailureThreshold:       cfg.FailureThreshold,
		CircuitBreakerTimeout:  cfg.CircuitBreakerTimeout,
		MetricsEnabled:         cfg.MetricsEnabled,
		TracingEnabled:         cfg.TracingEnabled,
		ServiceName:            cfg.ServiceName,
		HealthCheckEnabled:     cfg.HealthCheckEnabled,
		HealthCheckInterval:    cfg.HealthCheckInterval,
	}

	// Deep copy maps
	if cfg.Headers != nil {
		clone.Headers = make(map[string]string)
		for k, v := range cfg.Headers {
			clone.Headers[k] = v
		}
	}

	if cfg.QueryParams != nil {
		clone.QueryParams = make(map[string]string)
		for k, v := range cfg.QueryParams {
			clone.QueryParams[k] = v
		}
	}

	if cfg.ServiceURLs != nil {
		clone.ServiceURLs = make(map[string]string)
		for k, v := range cfg.ServiceURLs {
			clone.ServiceURLs[k] = v
		}
	}

	// Deep copy BasicAuth
	if cfg.BasicAuth != nil {
		clone.BasicAuth = &BasicAuth{
			Username: cfg.BasicAuth.Username,
			Password: cfg.BasicAuth.Password,
		}
	}

	return clone
}

// Validate checks if the configuration is valid
func (cfg *RequestConfig) Validate() error {
	if cfg.Method == "" {
		return fmt.Errorf("HTTP method is required")
	}

	if cfg.Path == "" {
		return fmt.Errorf("request path is required")
	}

	if cfg.BaseURL == "" && cfg.EconomicsURL == "" {
		return fmt.Errorf("at least one base URL must be configured")
	}

	return nil
}
