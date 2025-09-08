package core

import (
	"context"
	"fmt"

	"github.com/guiperry/KNIRVCHAIN-CLI/config"
	"github.com/sirupsen/logrus"
)

// KNIRVGatewayClient handles communication with KNIRVGATEWAY service
type KNIRVGatewayClient struct {
	*APIClient
	config      *config.ServiceConfig
	logger      *logrus.Logger
	serviceName string
	connected   bool
	capabilities []string
}

// GatewayHealth represents gateway health status
type GatewayHealth struct {
	Status    string                 `json:"status"`
	Services  map[string]interface{} `json:"services"`
	Metrics   map[string]interface{} `json:"metrics"`
	Timestamp string                 `json:"timestamp"`
}

// ServiceInfo represents information about available services
type ServiceInfo struct {
	Name         string                 `json:"name"`
	URL          string                 `json:"url"`
	Status       string                 `json:"status"`
	Capabilities []string               `json:"capabilities"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// AuthToken represents an authentication token
type AuthToken struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	Type      string `json:"type"`
}

// PoAuDResult represents a Proof of Audit result
type PoAuDResult struct {
	ID          string                 `json:"id"`
	Status      string                 `json:"status"`
	Score       float64                `json:"score"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   string                 `json:"timestamp"`
	ValidatedBy string                 `json:"validated_by"`
}

// NewKNIRVGatewayClient creates a new KNIRVGATEWAY client
func NewKNIRVGatewayClient(cfg *config.ServiceConfig, logger *logrus.Logger) *KNIRVGatewayClient {
	apiClient := NewAPIClient(cfg.URL, 
		WithTimeout(cfg.Timeout),
		WithRetries(cfg.Retries),
		WithLogger(logger),
	)

	return &KNIRVGatewayClient{
		APIClient:   apiClient,
		config:      cfg,
		logger:      logger,
		serviceName: "knirvgateway",
		capabilities: []string{"gateway", "proxy", "health-monitoring", "authentication", "poaud"},
	}
}

// Connect establishes connection to KNIRVGATEWAY
func (c *KNIRVGatewayClient) Connect(ctx context.Context) error {
	c.logger.Info("Connecting to KNIRVGATEWAY service")
	
	// Test connection with a health check
	err := c.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to KNIRVGATEWAY: %w", err)
	}
	
	c.connected = true
	c.logger.Info("Successfully connected to KNIRVGATEWAY")
	return nil
}

// Disconnect closes connection to KNIRVGATEWAY
func (c *KNIRVGatewayClient) Disconnect() error {
	c.logger.Info("Disconnecting from KNIRVGATEWAY service")
	c.connected = false
	return nil
}

// HealthCheck performs a health check on KNIRVGATEWAY
func (c *KNIRVGatewayClient) HealthCheck(ctx context.Context) error {
	endpoint := c.config.Endpoints.Health
	if endpoint == "" {
		endpoint = "/gateway/health"
	}
	
	var response interface{}
	return c.Get(ctx, endpoint, &response)
}

// GetCapabilities returns the capabilities of KNIRVGATEWAY
func (c *KNIRVGatewayClient) GetCapabilities() []string {
	return c.capabilities
}

// Subscribe subscribes to events from KNIRVGATEWAY
func (c *KNIRVGatewayClient) Subscribe(events []string, handler EventHandler) error {
	// TODO: Implement SSE subscription
	c.logger.Infof("Subscribing to KNIRVGATEWAY events: %v", events)
	return nil
}

// GetServiceName returns the service name
func (c *KNIRVGatewayClient) GetServiceName() string {
	return c.serviceName
}

// GetServiceURL returns the service URL
func (c *KNIRVGatewayClient) GetServiceURL() string {
	return c.config.URL
}

// IsConnected returns whether the client is connected
func (c *KNIRVGatewayClient) IsConnected() bool {
	return c.connected
}

// Gateway Operations

// GetGatewayHealth retrieves gateway health status
func (c *KNIRVGatewayClient) GetGatewayHealth(ctx context.Context) (*GatewayHealth, error) {
	var health GatewayHealth
	endpoint := "/gateway/health"
	
	err := c.Get(ctx, endpoint, &health)
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway health: %w", err)
	}
	
	return &health, nil
}

// GetAvailableServices retrieves list of available services
func (c *KNIRVGatewayClient) GetAvailableServices(ctx context.Context) ([]ServiceInfo, error) {
	var services []ServiceInfo
	endpoint := "/gateway/services"
	
	err := c.Get(ctx, endpoint, &services)
	if err != nil {
		return nil, fmt.Errorf("failed to get available services: %w", err)
	}
	
	return services, nil
}

// GetGatewayMetrics retrieves gateway performance metrics
func (c *KNIRVGatewayClient) GetGatewayMetrics(ctx context.Context) (map[string]interface{}, error) {
	var metrics map[string]interface{}
	endpoint := "/gateway/metrics"
	
	err := c.Get(ctx, endpoint, &metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway metrics: %w", err)
	}
	
	return metrics, nil
}

// Authentication Operations

// Login authenticates with the gateway
func (c *KNIRVGatewayClient) Login(ctx context.Context, credentials map[string]interface{}) (*AuthToken, error) {
	var token AuthToken
	endpoint := "/auth/login"
	
	err := c.Post(ctx, endpoint, credentials, &token)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	
	c.logger.Info("Successfully authenticated with KNIRVGATEWAY")
	return &token, nil
}

// VerifyToken verifies an authentication token
func (c *KNIRVGatewayClient) VerifyToken(ctx context.Context, token string) (bool, error) {
	request := map[string]interface{}{
		"token": token,
	}
	
	var response map[string]interface{}
	endpoint := "/auth/verify"
	
	err := c.Post(ctx, endpoint, request, &response)
	if err != nil {
		return false, fmt.Errorf("failed to verify token: %w", err)
	}
	
	valid, ok := response["valid"].(bool)
	if !ok {
		return false, fmt.Errorf("invalid token verification response")
	}
	
	return valid, nil
}

// Economics Operations

// GetEconomicsData retrieves economics data through the gateway
func (c *KNIRVGatewayClient) GetEconomicsData(ctx context.Context) (map[string]interface{}, error) {
	var data map[string]interface{}
	endpoint := c.config.Endpoints.Economics
	if endpoint == "" {
		endpoint = "/economics"
	}
	
	err := c.Get(ctx, endpoint, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to get economics data: %w", err)
	}
	
	return data, nil
}

// PoAuD Operations

// SubmitPoAuDRequest submits a Proof of Audit request
func (c *KNIRVGatewayClient) SubmitPoAuDRequest(ctx context.Context, request map[string]interface{}) (*PoAuDResult, error) {
	var result PoAuDResult
	endpoint := c.config.Endpoints.PoAuD
	if endpoint == "" {
		endpoint = "/poaud"
	}
	
	err := c.Post(ctx, endpoint, request, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to submit PoAuD request: %w", err)
	}
	
	c.logger.Infof("PoAuD request submitted successfully: %s", result.ID)
	return &result, nil
}

// GetPoAuDResult retrieves a PoAuD result by ID
func (c *KNIRVGatewayClient) GetPoAuDResult(ctx context.Context, id string) (*PoAuDResult, error) {
	var result PoAuDResult
	endpoint := fmt.Sprintf("%s/%s", c.config.Endpoints.PoAuD, id)
	if c.config.Endpoints.PoAuD == "" {
		endpoint = fmt.Sprintf("/poaud/%s", id)
	}
	
	err := c.Get(ctx, endpoint, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get PoAuD result: %w", err)
	}
	
	return &result, nil
}

// Proxy Operations

// ProxyRequest proxies a request to another KNIRV service through the gateway
func (c *KNIRVGatewayClient) ProxyRequest(ctx context.Context, service string, path string, method string, body interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	endpoint := fmt.Sprintf("/api/%s%s", service, path)
	
	var err error
	switch method {
	case "GET":
		err = c.Get(ctx, endpoint, &response)
	case "POST":
		err = c.Post(ctx, endpoint, body, &response)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to proxy request to %s: %w", service, err)
	}
	
	return response, nil
}
