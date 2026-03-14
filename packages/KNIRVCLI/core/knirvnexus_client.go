package core

import (
	"context"
	"fmt"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI/config"
	"github.com/sirupsen/logrus"
)

// KNIRVNexusClient handles communication with KNIRVNEXUS service
type KNIRVNexusClient struct {
	*APIClient
	config      *config.ServiceConfig
	logger      *logrus.Logger
	serviceName string
	connected   bool
	capabilities []string
}

// DVENode represents a Distributed Virtual Environment node
type DVENode struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Status       string                 `json:"status"`
	Capabilities []string               `json:"capabilities"`
	Resources    map[string]interface{} `json:"resources"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

// ValidationTask represents a validation task
type ValidationTask struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output"`
	Validator   string                 `json:"validator"`
	Score       float64                `json:"score"`
	CreatedAt   string                 `json:"created_at"`
	CompletedAt string                 `json:"completed_at"`
}

// InferenceRequest represents an inference request
type InferenceRequest struct {
	Model       string                 `json:"model"`
	Input       string                 `json:"input"`
	Parameters  map[string]interface{} `json:"parameters"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float64                `json:"temperature"`
}

// InferenceResponse represents an inference response
type InferenceResponse struct {
	ID       string                 `json:"id"`
	Output   string                 `json:"output"`
	Metadata map[string]interface{} `json:"metadata"`
	Usage    map[string]interface{} `json:"usage"`
}

// CognitiveEngineRequest represents a cognitive engine request
type CognitiveEngineRequest struct {
	Task        string                 `json:"task"`
	Context     map[string]interface{} `json:"context"`
	Parameters  map[string]interface{} `json:"parameters"`
	Constraints map[string]interface{} `json:"constraints"`
}

// CognitiveEngineResponse represents a cognitive engine response
type CognitiveEngineResponse struct {
	ID       string                 `json:"id"`
	Result   map[string]interface{} `json:"result"`
	Metadata map[string]interface{} `json:"metadata"`
	Status   string                 `json:"status"`
}

// NewKNIRVNexusClient creates a new KNIRVNEXUS client
func NewKNIRVNexusClient(cfg *config.ServiceConfig, logger *logrus.Logger) *KNIRVNexusClient {
	apiClient := NewAPIClient(cfg.URL, 
		WithTimeout(cfg.Timeout),
		WithRetries(cfg.Retries),
		WithLogger(logger),
	)

	return &KNIRVNexusClient{
		APIClient:   apiClient,
		config:      cfg,
		logger:      logger,
		serviceName: "knirvnexus",
		capabilities: []string{"dve", "inference", "validation", "cognitive-engine", "tee-security"},
	}
}

// Connect establishes connection to KNIRVNEXUS
func (c *KNIRVNexusClient) Connect(ctx context.Context) error {
	c.logger.Info("Connecting to KNIRVNEXUS service")
	
	// Test connection with a health check
	err := c.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to KNIRVNEXUS: %w", err)
	}
	
	c.connected = true
	c.logger.Info("Successfully connected to KNIRVNEXUS")
	return nil
}

// Disconnect closes connection to KNIRVNEXUS
func (c *KNIRVNexusClient) Disconnect() error {
	c.logger.Info("Disconnecting from KNIRVNEXUS service")
	c.connected = false
	return nil
}

// HealthCheck performs a health check on KNIRVNEXUS
func (c *KNIRVNexusClient) HealthCheck(ctx context.Context) error {
	var response interface{}
	return c.Get(ctx, "/api/health", &response)
}

// GetCapabilities returns the capabilities of KNIRVNEXUS
func (c *KNIRVNexusClient) GetCapabilities() []string {
	return c.capabilities
}

// Subscribe subscribes to events from KNIRVNEXUS
func (c *KNIRVNexusClient) Subscribe(events []string, handler EventHandler) error {
	// TODO: Implement WebSocket subscription
	c.logger.Infof("Subscribing to KNIRVNEXUS events: %v", events)
	return nil
}

// GetServiceName returns the service name
func (c *KNIRVNexusClient) GetServiceName() string {
	return c.serviceName
}

// GetServiceURL returns the service URL
func (c *KNIRVNexusClient) GetServiceURL() string {
	return c.config.URL
}

// IsConnected returns whether the client is connected
func (c *KNIRVNexusClient) IsConnected() bool {
	return c.connected
}

// DVE Operations

// GetDVENodes retrieves all DVE nodes
func (c *KNIRVNexusClient) GetDVENodes(ctx context.Context) ([]DVENode, error) {
	var nodes []DVENode
	endpoint := "/api/dve-nodes"
	
	err := c.Get(ctx, endpoint, &nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get DVE nodes: %w", err)
	}
	
	return nodes, nil
}

// CreateDVENode creates a new DVE node
func (c *KNIRVNexusClient) CreateDVENode(ctx context.Context, node *DVENode) (*DVENode, error) {
	var createdNode DVENode
	endpoint := "/api/dve-nodes"
	
	err := c.Post(ctx, endpoint, node, &createdNode)
	if err != nil {
		return nil, fmt.Errorf("failed to create DVE node: %w", err)
	}
	
	c.logger.Infof("DVE node created successfully: %s", createdNode.ID)
	return &createdNode, nil
}

// GetDVENode retrieves a specific DVE node by ID
func (c *KNIRVNexusClient) GetDVENode(ctx context.Context, id string) (*DVENode, error) {
	var node DVENode
	endpoint := fmt.Sprintf("/api/dve-nodes/%s", id)
	
	err := c.Get(ctx, endpoint, &node)
	if err != nil {
		return nil, fmt.Errorf("failed to get DVE node %s: %w", id, err)
	}
	
	return &node, nil
}

// Validation Operations

// GetValidationTasks retrieves all validation tasks
func (c *KNIRVNexusClient) GetValidationTasks(ctx context.Context) ([]ValidationTask, error) {
	var tasks []ValidationTask
	endpoint := "/api/validation-tasks"
	
	err := c.Get(ctx, endpoint, &tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to get validation tasks: %w", err)
	}
	
	return tasks, nil
}

// CreateValidationTask creates a new validation task
func (c *KNIRVNexusClient) CreateValidationTask(ctx context.Context, task *ValidationTask) (*ValidationTask, error) {
	var createdTask ValidationTask
	endpoint := "/api/validation-tasks"
	
	err := c.Post(ctx, endpoint, task, &createdTask)
	if err != nil {
		return nil, fmt.Errorf("failed to create validation task: %w", err)
	}
	
	c.logger.Infof("Validation task created successfully: %s", createdTask.ID)
	return &createdTask, nil
}

// GetValidationTask retrieves a specific validation task by ID
func (c *KNIRVNexusClient) GetValidationTask(ctx context.Context, id string) (*ValidationTask, error) {
	var task ValidationTask
	endpoint := fmt.Sprintf("/api/validation-tasks/%s", id)
	
	err := c.Get(ctx, endpoint, &task)
	if err != nil {
		return nil, fmt.Errorf("failed to get validation task %s: %w", id, err)
	}
	
	return &task, nil
}

// Inference Operations

// SubmitInferenceRequest submits an inference request
func (c *KNIRVNexusClient) SubmitInferenceRequest(ctx context.Context, request *InferenceRequest) (*InferenceResponse, error) {
	var response InferenceResponse
	endpoint := c.config.Endpoints.Inference
	if endpoint == "" {
		endpoint = "/api/inference"
	}
	
	err := c.Post(ctx, endpoint, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to submit inference request: %w", err)
	}
	
	c.logger.Infof("Inference request submitted successfully: %s", response.ID)
	return &response, nil
}

// GetInferenceModels retrieves available inference models
func (c *KNIRVNexusClient) GetInferenceModels(ctx context.Context) ([]map[string]interface{}, error) {
	var models []map[string]interface{}
	endpoint := "/api/inference/models"
	
	err := c.Get(ctx, endpoint, &models)
	if err != nil {
		return nil, fmt.Errorf("failed to get inference models: %w", err)
	}
	
	return models, nil
}

// Cognitive Engine Operations

// SubmitCognitiveRequest submits a request to the cognitive engine
func (c *KNIRVNexusClient) SubmitCognitiveRequest(ctx context.Context, request *CognitiveEngineRequest) (*CognitiveEngineResponse, error) {
	var response CognitiveEngineResponse
	endpoint := c.config.Endpoints.Agentic
	if endpoint == "" {
		endpoint = "/api/cognitive-engine"
	}
	
	err := c.Post(ctx, endpoint, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to submit cognitive request: %w", err)
	}
	
	c.logger.Infof("Cognitive request submitted successfully: %s", response.ID)
	return &response, nil
}

// System Operations

// GetSystemHealth retrieves system health status
func (c *KNIRVNexusClient) GetSystemHealth(ctx context.Context) (map[string]interface{}, error) {
	var health map[string]interface{}
	endpoint := "/api/system-health"
	
	err := c.Get(ctx, endpoint, &health)
	if err != nil {
		return nil, fmt.Errorf("failed to get system health: %w", err)
	}
	
	return health, nil
}

// GetSystemMetrics retrieves system metrics
func (c *KNIRVNexusClient) GetSystemMetrics(ctx context.Context) (map[string]interface{}, error) {
	var metrics map[string]interface{}
	endpoint := "/api/system/metrics"
	
	err := c.Get(ctx, endpoint, &metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to get system metrics: %w", err)
	}
	
	return metrics, nil
}
