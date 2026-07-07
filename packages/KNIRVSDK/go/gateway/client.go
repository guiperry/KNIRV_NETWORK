// KNIRV Gateway SDK Client
// This client provides access to KNIRVGATEWAY services including Economics and API Gateway

package gateway

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/gateway/internal/requestconfig"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/gateway/option"
)

// Client creates a struct with services and top level methods that help with
// interacting with the KNIRVGATEWAY API. You should not instantiate
// this client directly, and instead use the [NewClient] method instead.
type Client struct {
	Options     []option.RequestOption
	Economics   EconomicsService
	Gateway     GatewayService
	Health      HealthService
	Integration IntegrationService
	PoAuD       PoAuDService
}

// EconomicsService provides access to the Economics API
type EconomicsService struct {
	Skills       SkillsService
	LLM          LLMService
	Validation   ValidationService
	Fees         FeesService
	Metrics      MetricsService
	Transactions TransactionsService
	Burn         BurnService
	Rules        RulesService
}

// GatewayService provides access to the API Gateway functionality
type GatewayService struct {
	Routes RoutesService
	Health HealthService
	Status StatusService
}

// HealthService provides health check functionality
type HealthService struct {
	client *Client
}

// IntegrationService provides integration status and management
type IntegrationService struct {
	client *Client
}

// PoAuDService provides access to the PoAu-D consensus management API
type PoAuDService struct {
	client         *Client
	NetworkAuthors NetworkAuthorsService
}

// NetworkAuthorsService provides Network Authors management
type NetworkAuthorsService struct {
	client *Client
}

// DefaultClientOptions read from the environment
// (KNIRVGATEWAY_API_KEY, KNIRVGATEWAY_BASE_URL). This
// should be used to initialize new clients.
func DefaultClientOptions() []option.RequestOption {
	defaults := []option.RequestOption{option.WithEnvironmentProduction()}

	// Gateway base URL
	if o, ok := os.LookupEnv("KNIRVGATEWAY_BASE_URL"); ok {
		defaults = append(defaults, option.WithBaseURL(o))
	} else if o, ok := os.LookupEnv("GATEWAY_SERVICE_URL"); ok {
		defaults = append(defaults, option.WithBaseURL(o))
	} else {
		defaults = append(defaults, option.WithBaseURL("http://localhost:8000"))
	}

	// Economics service URL
	if o, ok := os.LookupEnv("ECONOMICS_SERVICE_URL"); ok {
		defaults = append(defaults, option.WithEconomicsURL(o))
	} else {
		defaults = append(defaults, option.WithEconomicsURL("http://localhost:8090"))
	}

	// API Key
	if o, ok := os.LookupEnv("KNIRVGATEWAY_API_KEY"); ok {
		defaults = append(defaults, option.WithAPIKey(o))
	}

	// NRN Contract
	if o, ok := os.LookupEnv("NRN_CONTRACT"); ok {
		defaults = append(defaults, option.WithNRNContract(o))
	}

	return defaults
}

// NewClient generates a new client with the default option read from the
// environment (KNIRVGATEWAY_API_KEY, KNIRVGATEWAY_BASE_URL). The option passed in as arguments are
// applied after these default arguments, and all option will be passed down to the
// services and requests that this client makes.
func NewClient(opts ...option.RequestOption) (*Client, error) {
	options := DefaultClientOptions()
	options = append(options, opts...)

	client := &Client{
		Options: options,
	}

	// Initialize services
	client.Economics = EconomicsService{
		Skills:       SkillsService{client: client},
		LLM:          LLMService{client: client},
		Validation:   ValidationService{client: client},
		Fees:         FeesService{client: client},
		Metrics:      MetricsService{client: client},
		Transactions: TransactionsService{client: client},
		Burn:         BurnService{client: client},
		Rules:        RulesService{client: client},
	}

	client.Gateway = GatewayService{
		Routes: RoutesService{client: client},
		Health: HealthService{client: client},
		Status: StatusService{client: client},
	}

	client.Health = HealthService{client: client}
	client.Integration = IntegrationService{client: client}

	client.PoAuD = PoAuDService{
		client:         client,
		NetworkAuthors: NetworkAuthorsService{client: client},
	}

	return client, nil
}

// NewEconomicsClient creates a client specifically for the Economics Service
func NewEconomicsClient(opts ...option.RequestOption) *Client {
	// Set economics-specific defaults
	economicsOpts := []option.RequestOption{
		option.WithBaseURL(os.Getenv("ECONOMICS_SERVICE_URL")),
	}
	if economicsOpts[0] == nil {
		economicsOpts[0] = option.WithBaseURL("http://localhost:8090")
	}

	economicsOpts = append(economicsOpts, opts...)
	client, _ := NewClient(economicsOpts...)
	return client
}

// NewGatewayClient creates a client specifically for the API Gateway
func NewGatewayClient(opts ...option.RequestOption) *Client {
	// Set gateway-specific defaults
	gatewayOpts := []option.RequestOption{
		option.WithBaseURL(os.Getenv("GATEWAY_SERVICE_URL")),
	}
	if gatewayOpts[0] == nil {
		gatewayOpts[0] = option.WithBaseURL("http://localhost:8000")
	}

	gatewayOpts = append(gatewayOpts, opts...)
	client, _ := NewClient(gatewayOpts...)
	return client
}

// Execute makes an HTTP request with the given context and request configuration
func (c *Client) Execute(ctx context.Context, cfg requestconfig.RequestConfig) (*http.Response, error) {
	req, err := cfg.Build(ctx)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return client.Do(req)
}

// Get makes a GET request to the specified path
func (c *Client) Get(ctx context.Context, path string, opts ...option.RequestOption) (*http.Response, error) {
	cfg := requestconfig.RequestConfig{
		Method: http.MethodGet,
		Path:   path,
	}

	for _, opt := range append(c.Options, opts...) {
		opt(&cfg)
	}

	return c.Execute(ctx, cfg)
}

// Post makes a POST request to the specified path with the given body
func (c *Client) Post(ctx context.Context, path string, body interface{}, opts ...option.RequestOption) (*http.Response, error) {
	cfg := requestconfig.RequestConfig{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	}

	for _, opt := range append(c.Options, opts...) {
		opt(&cfg)
	}

	return c.Execute(ctx, cfg)
}

// Put makes a PUT request to the specified path with the given body
func (c *Client) Put(ctx context.Context, path string, body interface{}, opts ...option.RequestOption) (*http.Response, error) {
	cfg := requestconfig.RequestConfig{
		Method: http.MethodPut,
		Path:   path,
		Body:   body,
	}

	for _, opt := range append(c.Options, opts...) {
		opt(&cfg)
	}

	return c.Execute(ctx, cfg)
}

// Delete makes a DELETE request to the specified path
func (c *Client) Delete(ctx context.Context, path string, opts ...option.RequestOption) (*http.Response, error) {
	cfg := requestconfig.RequestConfig{
		Method: http.MethodDelete,
		Path:   path,
	}

	for _, opt := range append(c.Options, opts...) {
		opt(&cfg)
	}

	return c.Execute(ctx, cfg)
}

// NewRequest creates a new HTTP request with the specified method, path, and body
func (c *Client) NewRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	cfg := requestconfig.RequestConfig{
		Method: method,
		Path:   path,
		Body:   body,
	}

	for _, opt := range c.Options {
		opt(&cfg)
	}

	return cfg.Build(ctx)
}

// Do executes an HTTP request and returns the response
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// Create a temporary config to get the configured HTTP client and retry settings
	cfg := &requestconfig.RequestConfig{}
	for _, opt := range c.Options {
		opt(cfg)
	}

	// Use the configured HTTP client
	httpClient := cfg.GetHTTPClient()

	// Implement retry logic
	maxRetries := cfg.Retries
	if maxRetries == 0 {
		maxRetries = 1 // At least one attempt
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := httpClient.Do(req)
		if err == nil && resp.StatusCode < 500 {
			// Success or client error (don't retry client errors)
			return resp, nil
		}

		// Store the error for potential return
		if err != nil {
			lastErr = err
		} else {
			resp.Body.Close() // Close the body before retrying
		}

		// Don't sleep after the last attempt
		if attempt < maxRetries-1 && cfg.RetryDelay > 0 {
			time.Sleep(cfg.RetryDelay)
		}
	}

	// Return the last error if all retries failed
	if lastErr != nil {
		return nil, lastErr
	}

	// This shouldn't happen, but just in case
	return nil, http.ErrServerClosed
}
