package core

import (
	"context"
	"fmt"

	"github.com/guiperry/KNIRVCHAIN-CLI/config"
	"github.com/sirupsen/logrus"
)

// KNIRVGraphClient handles communication with KNIRVGRAPH service
type KNIRVGraphClient struct {
	*APIClient
	config      *config.ServiceConfig
	logger      *logrus.Logger
	serviceName string
	connected   bool
	capabilities []string
}

// GraphNode represents a node in the KNIRV graph
type GraphNode struct {
	ID        string                 `json:"id"`
	NodeType  string                 `json:"node_type"`
	Data      map[string]interface{} `json:"data"`
	Parents   []string               `json:"parents"`
	Children  []string               `json:"children"`
	Weight    float64                `json:"weight"`
	Timestamp string                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// GraphEdge represents an edge in the KNIRV graph
type GraphEdge struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Weight    float64                `json:"weight"`
	EdgeType  string                 `json:"edge_type"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp string                 `json:"timestamp"`
}

// SkillNode represents a skill node in the NRV system
type SkillNode struct {
	ID           string                 `json:"id"`
	SkillType    string                 `json:"skill_type"`
	Capabilities []string               `json:"capabilities"`
	Requirements map[string]interface{} `json:"requirements"`
	Performance  *SkillPerformance      `json:"performance,omitempty"`
	Validation   *SkillValidation       `json:"validation,omitempty"`
	Timestamp    string                 `json:"timestamp"`
}

// SkillPerformance represents skill performance metrics
type SkillPerformance struct {
	SuccessRate         float64 `json:"success_rate"`
	AvgResolutionTime   float64 `json:"avg_resolution_time"`
	TotalResolutions    int     `json:"total_resolutions"`
}

// SkillValidation represents skill validation information
type SkillValidation struct {
	IsValidated      bool     `json:"is_validated"`
	ValidatedBy      []string `json:"validated_by"`
	ValidationScore  float64  `json:"validation_score"`
	LastValidated    string   `json:"last_validated"`
}

// ErrorNode represents an error node in the NRV system
type ErrorNode struct {
	ID               string                 `json:"id"`
	ErrorType        string                 `json:"error_type"`
	Description      string                 `json:"description"`
	Context          map[string]interface{} `json:"context"`
	Severity         int                    `json:"severity"`
	Timestamp        string                 `json:"timestamp"`
	ResolutionStatus string                 `json:"resolution_status,omitempty"`
	ResolvedBy       []string               `json:"resolved_by,omitempty"`
}

// NRVVector represents a Network Resolution Vector
type NRVVector struct {
	ID          string                 `json:"id"`
	SourcePeer  string                 `json:"source_peer"`
	TargetHash  string                 `json:"target_hash"`
	Coordinates []float64              `json:"coordinates"`
	Confidence  float64                `json:"confidence"`
	Timestamp   string                 `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// GraphChainStats represents statistics about the graph chain
type GraphChainStats struct {
	Height             int64   `json:"height"`
	TotalNodes         int64   `json:"totalNodes"`
	TotalEdges         int64   `json:"totalEdges"`
	TotalSkillNodes    int64   `json:"totalSkillNodes"`
	TotalErrorNodes    int64   `json:"totalErrorNodes"`
	TotalVectors       int64   `json:"totalVectors"`
	AvgResolutionTime  float64 `json:"avgResolutionTime"`
}

// NewKNIRVGraphClient creates a new KNIRVGRAPH client
func NewKNIRVGraphClient(cfg *config.ServiceConfig, logger *logrus.Logger) *KNIRVGraphClient {
	apiClient := NewAPIClient(cfg.URL, 
		WithTimeout(cfg.Timeout),
		WithRetries(cfg.Retries),
		WithLogger(logger),
	)

	return &KNIRVGraphClient{
		APIClient:   apiClient,
		config:      cfg,
		logger:      logger,
		serviceName: "knirvgraph",
		capabilities: []string{"graph", "nrv", "skills", "errors", "vectors"},
	}
}

// Connect establishes connection to KNIRVGRAPH
func (c *KNIRVGraphClient) Connect(ctx context.Context) error {
	c.logger.Info("Connecting to KNIRVGRAPH service")
	
	// Test connection with a health check
	err := c.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to KNIRVGRAPH: %w", err)
	}
	
	c.connected = true
	c.logger.Info("Successfully connected to KNIRVGRAPH")
	return nil
}

// Disconnect closes connection to KNIRVGRAPH
func (c *KNIRVGraphClient) Disconnect() error {
	c.logger.Info("Disconnecting from KNIRVGRAPH service")
	c.connected = false
	return nil
}

// HealthCheck performs a health check on KNIRVGRAPH
func (c *KNIRVGraphClient) HealthCheck(ctx context.Context) error {
	var response interface{}
	return c.Get(ctx, "/health", &response)
}

// GetCapabilities returns the capabilities of KNIRVGRAPH
func (c *KNIRVGraphClient) GetCapabilities() []string {
	return c.capabilities
}

// Subscribe subscribes to events from KNIRVGRAPH
func (c *KNIRVGraphClient) Subscribe(events []string, handler EventHandler) error {
	// TODO: Implement WebSocket subscription
	c.logger.Infof("Subscribing to KNIRVGRAPH events: %v", events)
	return nil
}

// GetServiceName returns the service name
func (c *KNIRVGraphClient) GetServiceName() string {
	return c.serviceName
}

// GetServiceURL returns the service URL
func (c *KNIRVGraphClient) GetServiceURL() string {
	return c.config.URL
}

// IsConnected returns whether the client is connected
func (c *KNIRVGraphClient) IsConnected() bool {
	return c.connected
}

// Graph Operations

// GetCurrentHeight retrieves the current graph chain height
func (c *KNIRVGraphClient) GetCurrentHeight(ctx context.Context) (int64, error) {
	var response map[string]interface{}
	endpoint := "/height"
	
	err := c.Get(ctx, endpoint, &response)
	if err != nil {
		return 0, fmt.Errorf("failed to get current height: %w", err)
	}
	
	height, ok := response["height"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid height response format")
	}
	
	return int64(height), nil
}

// GetNode retrieves a graph node by ID
func (c *KNIRVGraphClient) GetNode(ctx context.Context, nodeID string) (*GraphNode, error) {
	var node GraphNode
	endpoint := fmt.Sprintf("/node/%s", nodeID)
	
	err := c.Get(ctx, endpoint, &node)
	if err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", nodeID, err)
	}
	
	return &node, nil
}

// GetEdge retrieves a graph edge by ID
func (c *KNIRVGraphClient) GetEdge(ctx context.Context, edgeID string) (*GraphEdge, error) {
	var edge GraphEdge
	endpoint := fmt.Sprintf("/edge/%s", edgeID)
	
	err := c.Get(ctx, endpoint, &edge)
	if err != nil {
		return nil, fmt.Errorf("failed to get edge %s: %w", edgeID, err)
	}
	
	return &edge, nil
}

// GetGraphHeads retrieves the current graph heads
func (c *KNIRVGraphClient) GetGraphHeads(ctx context.Context) ([]string, error) {
	var response map[string]interface{}
	endpoint := "/graph/heads"
	
	err := c.Get(ctx, endpoint, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get graph heads: %w", err)
	}
	
	headsInterface, ok := response["heads"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid graph heads response format")
	}
	
	heads := make([]string, len(headsInterface))
	for i, head := range headsInterface {
		heads[i] = head.(string)
	}
	
	return heads, nil
}

// FindPath finds a path between two nodes
func (c *KNIRVGraphClient) FindPath(ctx context.Context, fromID, toID string, maxDepth int) ([]string, error) {
	var response map[string]interface{}
	endpoint := fmt.Sprintf("/graph/path/%s/%s?max_depth=%d", fromID, toID, maxDepth)
	
	err := c.Get(ctx, endpoint, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to find path from %s to %s: %w", fromID, toID, err)
	}
	
	pathInterface, ok := response["path"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid path response format")
	}
	
	path := make([]string, len(pathInterface))
	for i, node := range pathInterface {
		path[i] = node.(string)
	}
	
	return path, nil
}

// NRV System Operations

// GetAllSkills retrieves all skill nodes
func (c *KNIRVGraphClient) GetAllSkills(ctx context.Context) ([]SkillNode, error) {
	var skills []SkillNode
	endpoint := c.config.Endpoints.NRV + "/skills"
	if c.config.Endpoints.NRV == "" {
		endpoint = "/nrv/skills"
	}
	
	err := c.Get(ctx, endpoint, &skills)
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}
	
	return skills, nil
}

// SubmitSkillNode submits a new skill node
func (c *KNIRVGraphClient) SubmitSkillNode(ctx context.Context, skill *SkillNode) error {
	endpoint := c.config.Endpoints.NRV + "/skills"
	if c.config.Endpoints.NRV == "" {
		endpoint = "/nrv/skills"
	}
	
	var response interface{}
	err := c.Post(ctx, endpoint, skill, &response)
	if err != nil {
		return fmt.Errorf("failed to submit skill node: %w", err)
	}
	
	c.logger.Infof("Skill node submitted successfully: %s", skill.ID)
	return nil
}

// GetAllErrors retrieves all error nodes
func (c *KNIRVGraphClient) GetAllErrors(ctx context.Context) ([]ErrorNode, error) {
	var errors []ErrorNode
	endpoint := c.config.Endpoints.NRV + "/errors"
	if c.config.Endpoints.NRV == "" {
		endpoint = "/nrv/errors"
	}
	
	err := c.Get(ctx, endpoint, &errors)
	if err != nil {
		return nil, fmt.Errorf("failed to get errors: %w", err)
	}
	
	return errors, nil
}

// SubmitErrorNode submits a new error node
func (c *KNIRVGraphClient) SubmitErrorNode(ctx context.Context, errorNode *ErrorNode) error {
	endpoint := c.config.Endpoints.NRV + "/errors"
	if c.config.Endpoints.NRV == "" {
		endpoint = "/nrv/errors"
	}
	
	var response interface{}
	err := c.Post(ctx, endpoint, errorNode, &response)
	if err != nil {
		return fmt.Errorf("failed to submit error node: %w", err)
	}
	
	c.logger.Infof("Error node submitted successfully: %s", errorNode.ID)
	return nil
}

// QuerySkillsForError queries skills that can resolve a specific error
func (c *KNIRVGraphClient) QuerySkillsForError(ctx context.Context, errorType string, context map[string]interface{}) ([]SkillNode, error) {
	request := map[string]interface{}{
		"error_type": errorType,
		"context":    context,
	}
	
	var skills []SkillNode
	endpoint := c.config.Endpoints.NRV + "/query/skills-for-error"
	if c.config.Endpoints.NRV == "" {
		endpoint = "/nrv/query/skills-for-error"
	}
	
	err := c.Post(ctx, endpoint, request, &skills)
	if err != nil {
		return nil, fmt.Errorf("failed to query skills for error: %w", err)
	}
	
	return skills, nil
}

// GetGraphStats retrieves graph chain statistics
func (c *KNIRVGraphClient) GetGraphStats(ctx context.Context) (*GraphChainStats, error) {
	var stats GraphChainStats
	endpoint := "/stats"
	
	err := c.Get(ctx, endpoint, &stats)
	if err != nil {
		return nil, fmt.Errorf("failed to get graph stats: %w", err)
	}
	
	return &stats, nil
}
