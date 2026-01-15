package oracled

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Client is a client for interacting with the knirv-oracle RPC endpoint.
type Client struct {
	BaseURL string
	client  *http.Client
}

// NewClient creates a new OracledClient instance.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second, // 10-second timeout for RPC calls
		},
	}
}

// OracleStatus represents the structure of the status response from knirv-oracle.
// This struct should match the output of KNIRVORACLE/internal/oracle/oracle.go's GetStatus() method.
type OracleStatus struct {
	ChainID   string `json:"chain_id"`
	NetworkID string `json:"network_id"`
	Token     struct {
		Name          string `json:"name"`
		Symbol        string `json:"symbol"`
		TotalSupply   string `json:"total_supply"`
		MaxSupply     string `json:"max_supply"`
		ContractAddress string `json:"contract_address"`
	} `json:"token"`
	Consensus struct {
		LatestBlockHeight int64 `json:"latest_block_height"`
		ValidatorCount    int   `json:"validator_count"`
	} `json:"consensus"`
	Governance map[string]interface{} `json:"governance"` // Assuming this is dynamic
	Economics  map[string]interface{} `json:"economics"`  // Assuming this is dynamic
	P2P        map[string]interface{} `json:"p2p"`        // Assuming this is dynamic
	IBC        struct {
		Enabled bool `json:"enabled"`
	} `json:"ibc"`
}

// GetStatus fetches the current status of the knirv-oracle node.
func (c *Client) GetStatus(ctx context.Context) (*OracleStatus, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/status", c.BaseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to oracled RPC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("oracled RPC returned non-200 status: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from oracled RPC: %w", err)
	}

	var status OracleStatus
	if err := json.Unmarshal(bodyBytes, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal oracled status response: %w", err)
	}

	return &status, nil
}
