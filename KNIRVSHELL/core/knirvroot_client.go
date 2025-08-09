package core

import (
	"context"
	"fmt"

	"github.com/guiperry/KNIRVCHAIN-CLI/config"
	"github.com/sirupsen/logrus"
)

// KNIRVRootClient handles communication with KNIRVROOT service
type KNIRVRootClient struct {
	*APIClient
	config       *config.ServiceConfig
	logger       *logrus.Logger
	serviceName  string
	connected    bool
	capabilities []string
}

// BlockchainState represents the blockchain state from KNIRVROOT
type BlockchainState struct {
	Height      int64                  `json:"height"`
	Blocks      []interface{}          `json:"blocks"`
	TxPool      []interface{}          `json:"tx_pool"`
	Reflections []interface{}          `json:"reflections"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// BlockchainTransaction represents a blockchain transaction
type BlockchainTransaction struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Signature string                 `json:"signature"`
	Hash      string                 `json:"hash"`
	Timestamp string                 `json:"timestamp"`
}

// AgentInfo represents agent information
type AgentInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Capabilities []string               `json:"capabilities"`
	Status       string                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// EconomicsData represents economics module data
type EconomicsData struct {
	NRNBalance   string                 `json:"nrn_balance"`
	Skills       []interface{}          `json:"skills"`
	Transactions []interface{}          `json:"transactions"`
	Metrics      map[string]interface{} `json:"metrics"`
}

// NewKNIRVRootClient creates a new KNIRVROOT client
func NewKNIRVRootClient(cfg *config.ServiceConfig, logger *logrus.Logger) *KNIRVRootClient {
	apiClient := NewAPIClient(cfg.URL,
		WithTimeout(cfg.Timeout),
		WithRetries(cfg.Retries),
		WithLogger(logger),
	)

	return &KNIRVRootClient{
		APIClient:    apiClient,
		config:       cfg,
		logger:       logger,
		serviceName:  "knirvroot",
		capabilities: []string{"blockchain", "economics", "agent-management", "tunnel-registry"},
	}
}

// Connect establishes connection to KNIRVROOT
func (c *KNIRVRootClient) Connect(ctx context.Context) error {
	c.logger.Info("Connecting to KNIRVROOT service")

	// Test connection with a ping
	err := c.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to KNIRVROOT: %w", err)
	}

	c.connected = true
	c.logger.Info("Successfully connected to KNIRVROOT")
	return nil
}

// Disconnect closes connection to KNIRVROOT
func (c *KNIRVRootClient) Disconnect() error {
	c.logger.Info("Disconnecting from KNIRVROOT service")
	c.connected = false
	return nil
}

// HealthCheck performs a health check on KNIRVROOT
func (c *KNIRVRootClient) HealthCheck(ctx context.Context) error {
	var response interface{}
	return c.Get(ctx, "/ping", &response)
}

// GetCapabilities returns the capabilities of KNIRVROOT
func (c *KNIRVRootClient) GetCapabilities() []string {
	return c.capabilities
}

// Subscribe subscribes to events from KNIRVROOT
func (c *KNIRVRootClient) Subscribe(events []string, handler EventHandler) error {
	// TODO: Implement WebSocket subscription
	c.logger.Infof("Subscribing to KNIRVROOT events: %v", events)
	return nil
}

// GetServiceName returns the service name
func (c *KNIRVRootClient) GetServiceName() string {
	return c.serviceName
}

// GetServiceURL returns the service URL
func (c *KNIRVRootClient) GetServiceURL() string {
	return c.config.URL
}

// IsConnected returns whether the client is connected
func (c *KNIRVRootClient) IsConnected() bool {
	return c.connected
}

// Blockchain Operations

// GetBlockchainState retrieves the current blockchain state
func (c *KNIRVRootClient) GetBlockchainState(ctx context.Context) (*BlockchainState, error) {
	var state BlockchainState
	endpoint := c.config.Endpoints.API + "/chain"
	if c.config.Endpoints.API == "" {
		endpoint = "/chain"
	}

	err := c.Get(ctx, endpoint, &state)
	if err != nil {
		return nil, fmt.Errorf("failed to get blockchain state: %w", err)
	}

	return &state, nil
}

// SubmitTransaction submits a transaction to the blockchain
func (c *KNIRVRootClient) SubmitTransaction(ctx context.Context, tx *BlockchainTransaction) error {
	endpoint := c.config.Endpoints.API + "/transaction"
	if c.config.Endpoints.API == "" {
		endpoint = "/transaction"
	}

	var response interface{}
	err := c.Post(ctx, endpoint, tx, &response)
	if err != nil {
		return fmt.Errorf("failed to submit transaction: %w", err)
	}

	c.logger.Infof("Transaction submitted successfully: %s", tx.Hash)
	return nil
}

// Agent Management Operations

// GetAgents retrieves the list of connected agents
func (c *KNIRVRootClient) GetAgents(ctx context.Context) ([]AgentInfo, error) {
	var agents []AgentInfo
	endpoint := c.config.Endpoints.API + "/agents"
	if c.config.Endpoints.API == "" {
		endpoint = "/agents"
	}

	err := c.Get(ctx, endpoint, &agents)
	if err != nil {
		return nil, fmt.Errorf("failed to get agents: %w", err)
	}

	return agents, nil
}

// RegisterAgent registers a new agent
func (c *KNIRVRootClient) RegisterAgent(ctx context.Context, agent *AgentInfo) error {
	endpoint := c.config.Endpoints.API + "/agents"
	if c.config.Endpoints.API == "" {
		endpoint = "/agents"
	}

	var response interface{}
	err := c.Post(ctx, endpoint, agent, &response)
	if err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}

	c.logger.Infof("Agent registered successfully: %s", agent.ID)
	return nil
}

// Economics Operations

// GetEconomicsData retrieves economics module data
func (c *KNIRVRootClient) GetEconomicsData(ctx context.Context) (*EconomicsData, error) {
	var data EconomicsData
	endpoint := c.config.Endpoints.Economics
	if endpoint == "" {
		endpoint = "/economics"
	}

	err := c.Get(ctx, endpoint, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to get economics data: %w", err)
	}

	return &data, nil
}

// GetNRNBalance retrieves NRN balance for an address
func (c *KNIRVRootClient) GetNRNBalance(ctx context.Context, address string) (string, error) {
	var response map[string]interface{}
	endpoint := fmt.Sprintf("%s/balance/%s", c.config.Endpoints.Economics, address)
	if c.config.Endpoints.Economics == "" {
		endpoint = fmt.Sprintf("/economics/balance/%s", address)
	}

	err := c.Get(ctx, endpoint, &response)
	if err != nil {
		return "", fmt.Errorf("failed to get NRN balance: %w", err)
	}

	balance, ok := response["balance"].(string)
	if !ok {
		return "", fmt.Errorf("invalid balance response format")
	}

	return balance, nil
}

// RequestNRNFromFaucet requests NRN tokens from the faucet
func (c *KNIRVRootClient) RequestNRNFromFaucet(ctx context.Context, address string, amount string) error {
	request := map[string]interface{}{
		"address": address,
		"amount":  amount,
	}

	endpoint := c.config.Endpoints.Economics + "/faucet"
	if c.config.Endpoints.Economics == "" {
		endpoint = "/economics/faucet"
	}

	var response interface{}
	err := c.Post(ctx, endpoint, request, &response)
	if err != nil {
		return fmt.Errorf("failed to request NRN from faucet: %w", err)
	}

	c.logger.Infof("NRN faucet request successful for address %s, amount %s", address, amount)
	return nil
}
