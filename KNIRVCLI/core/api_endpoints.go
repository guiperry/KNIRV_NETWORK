package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// HealthCheck checks if the node is healthy
func (c *APIClient) HealthCheck(ctx context.Context) error {
	var result map[string]interface{}
	err := c.Get(ctx, "/health", &result)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Check if status is OK
	status, ok := result["status"].(string)
	if !ok || status != "ok" {
		return fmt.Errorf("node is not healthy: %v", result)
	}

	return nil
}

// GetNodeInfo gets information about the node
func (c *APIClient) GetNodeInfo(ctx context.Context) (*NodeInfo, error) {
	var result NodeInfo
	err := c.Get(ctx, "/info", &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %w", err)
	}

	return &result, nil
}

// CheckVersionCompatibility checks if the node version is compatible
func (c *APIClient) CheckVersionCompatibility(ctx context.Context, clientVersion string) error {
	nodeInfo, err := c.GetNodeInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to check version compatibility: %w", err)
	}

	// TODO: Implement proper version compatibility check
	// For now, just log the versions
	c.logger.Infof("Client version: %s, Node version: %s", clientVersion, nodeInfo.Version)

	return nil
}

// GetChainInfo gets information about the blockchain
func (c *APIClient) GetChainInfo(ctx context.Context) (*ChainInfo, error) {
	var result ChainInfo
	err := c.Get(ctx, "/chain", &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain info: %w", err)
	}

	return &result, nil
}

// GetBlockInfo gets information about a block
func (c *APIClient) GetBlockInfo(ctx context.Context, height uint64) (*BlockInfo, error) {
	var result BlockInfo
	err := c.Get(ctx, fmt.Sprintf("/block/%d", height), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get block info: %w", err)
	}

	return &result, nil
}

// GetBlockByHash gets information about a block by its hash
func (c *APIClient) GetBlockByHash(ctx context.Context, hash string) (*BlockInfo, error) {
	var result BlockInfo
	err := c.Get(ctx, fmt.Sprintf("/block/hash/%s", hash), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get block by hash: %w", err)
	}

	return &result, nil
}

// GetTransactionInfo gets information about a transaction
func (c *APIClient) GetTransactionInfo(ctx context.Context, hash string) (*TransactionInfo, error) {
	var result TransactionInfo
	err := c.Get(ctx, fmt.Sprintf("/transaction/%s", hash), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info: %w", err)
	}

	return &result, nil
}

// GetTransactionPoolInfo gets information about the transaction pool
func (c *APIClient) GetTransactionPoolInfo(ctx context.Context) (*TransactionPoolInfo, error) {
	var result TransactionPoolInfo
	err := c.Get(ctx, "/txn_pool", &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction pool info: %w", err)
	}

	return &result, nil
}

// PrepareCapabilityRegistration prepares a capability registration
func (c *APIClient) PrepareCapabilityRegistration(
	ctx context.Context,
	request MCPPrepareCapabilityRegistrationRequest,
	pluginFilePath string,
	manifestFilePath string,
) (*MCPPrepareCapabilityRegistrationResponse, error) {
	// Validate plugin and manifest files exist
	if err := ValidateFileExists(pluginFilePath); err != nil {
		return nil, fmt.Errorf("plugin file validation failed: %w", err)
	}
	if err := ValidateFileExists(manifestFilePath); err != nil {
		return nil, fmt.Errorf("manifest file validation failed: %w", err)
	}

	// Generate file references
	pluginRef, err := generateFileReference(pluginFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plugin file reference: %w", err)
	}

	manifestRef, err := generateFileReference(manifestFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate manifest file reference: %w", err)
	}

	// Add file references to descriptor schema
	if request.Descriptor == nil {
		request.Descriptor = make(map[string]interface{})
	}

	// Add schema with file references
	schema := map[string]interface{}{
		"manifestFile":   manifestRef.RelativePath,
		"executableFile": pluginRef.RelativePath,
		"locationHints":  []string{manifestRef.LocationHint, pluginRef.LocationHint},
	}

	// Add schema to descriptor
	request.Descriptor["schema"] = schema

	// Make API request
	var response MCPPrepareCapabilityRegistrationResponse
	err = c.Post(ctx, "/mcp/capability/prepare_registration", request, &response)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	return &response, nil
}

// SubmitTransaction submits a transaction to the blockchain
func (c *APIClient) SubmitTransaction(ctx context.Context, tx *Transaction) (*SubmitTransactionResponse, error) {
	// Convert Transaction to SubmitTransactionRequest
	request := SubmitTransactionRequest{
		From:            tx.From,
		To:              tx.To,
		Value:           tx.Value,
		Data:            tx.Data,
		Timestamp:       tx.Timestamp,
		Fee:             tx.Fee,
		Type:            tx.Type,
		PublicKey:       tx.PublicKey,
		Signature:       tx.Signature,
		TransactionHash: tx.TransactionHash,
	}

	var response SubmitTransactionResponse
	err := c.Post(ctx, "/transaction", request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to submit transaction: %w", err)
	}

	return &response, nil
}

// QueryAgents queries agents from the blockchain
func QueryAgents(nodeURL string, queryParams map[string]string) ([]map[string]interface{}, error) {
	client := NewAPIClient(nodeURL)
	ctx := context.Background()

	// Build query string
	path := "/agents"
	if len(queryParams) > 0 {
		path += "?"
		first := true
		for key, value := range queryParams {
			if !first {
				path += "&"
			}
			path += fmt.Sprintf("%s=%s", key, value)
			first = false
		}
	}

	var result struct {
		Agents []map[string]interface{} `json:"agents"`
	}

	err := client.Get(ctx, path, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}

	return result.Agents, nil
}

// QueryAgentByID queries a specific agent by ID
func QueryAgentByID(nodeURL string, agentID string) (map[string]interface{}, error) {
	client := NewAPIClient(nodeURL)
	ctx := context.Background()

	var result map[string]interface{}
	err := client.Get(ctx, fmt.Sprintf("/agents/%s", agentID), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to query agent: %w", err)
	}

	return result, nil
}

// QueryAgentPlugins queries plugins for a specific agent
func QueryAgentPlugins(nodeURL string, agentID string) ([]map[string]interface{}, error) {
	client := NewAPIClient(nodeURL)
	ctx := context.Background()

	var result struct {
		Plugins []map[string]interface{} `json:"plugins"`
	}

	err := client.Get(ctx, fmt.Sprintf("/agents/%s/plugins", agentID), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to query agent plugins: %w", err)
	}

	return result.Plugins, nil
}

// generateFileReference generates a file reference for a file
func generateFileReference(filePath string) (*FileReference, error) {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate content hash
	contentHash, err := calculateContentHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate content hash: %w", err)
	}

	// Get relative path (for now, just use the base name)
	relativePath := filepath.Base(filePath)

	// Create location hint (for now, just use a placeholder)
	// In a real implementation, this would be a URL where the file can be accessed
	locationHint := fmt.Sprintf("file://%s", relativePath)

	return &FileReference{
		OriginalPath: filePath,
		RelativePath: relativePath,
		ContentHash:  contentHash,
		LocationHint: locationHint,
		FileSize:     info.Size(),
		LastModified: info.ModTime(),
	}, nil
}

// CalculateHash calculates the SHA256 hash of byte data
func CalculateHash(data []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	return hasher.Sum(nil)
}
