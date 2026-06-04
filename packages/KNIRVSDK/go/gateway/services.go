// Additional Gateway Services Implementation for KNIRV Gateway SDK

package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Gateway Service Types

// RoutesService handles API gateway routing
type RoutesService struct {
	client *Client
}

// StatusService handles gateway status
type StatusService struct {
	client *Client
}

// Route represents an API gateway route
type Route struct {
	Path        string   `json:"path"`
	Methods     []string `json:"methods"`
	Target      string   `json:"target"`
	AuthRequired bool    `json:"auth_required"`
	RateLimit   int      `json:"rate_limit"`
}

// GatewayStatus represents gateway status information
type GatewayStatus struct {
	Status      string            `json:"status"`
	Version     string            `json:"version"`
	Uptime      time.Duration     `json:"uptime"`
	Services    map[string]string `json:"services"`
	LastUpdated time.Time         `json:"last_updated"`
}

// IntegrationStatus represents integration status
type IntegrationStatus struct {
	KNIRVChainURL string    `json:"knirvchain_url"`
	KNIRVNexusURL string    `json:"knirvnexus_url"`
	KNIRVRootURL  string    `json:"knirvoracle_url"`
	KNIRVGraphURL string    `json:"knirvgraph_url"`
	LastSync      time.Time `json:"last_sync"`
	Status        string    `json:"status"`
}

// Transactions Service Methods (continued)

// Get retrieves a specific transaction by ID
func (s *TransactionsService) Get(ctx context.Context, transactionID string) (*Transaction, error) {
	path := fmt.Sprintf("/economics/transaction/%s", transactionID)
	resp, err := s.client.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("transaction retrieval failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool         `json:"success"`
		Data    *Transaction `json:"data"`
		Error   string       `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("transaction retrieval failed: %s", result.Error)
	}

	return result.Data, nil
}

// List retrieves a list of transactions with optional filters
func (s *TransactionsService) List(ctx context.Context, limit int, status string) ([]*Transaction, error) {
	path := fmt.Sprintf("/economics/transactions?limit=%d", limit)
	if status != "" {
		path += fmt.Sprintf("&status=%s", status)
	}

	resp, err := s.client.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("transactions list retrieval failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Transactions []*Transaction `json:"transactions"`
			Count        int            `json:"count"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("transactions list retrieval failed: %s", result.Error)
	}

	return result.Data.Transactions, nil
}

// Burn Service Methods

// GetHistory retrieves burn event history
func (s *BurnService) GetHistory(ctx context.Context, limit int) ([]*BurnEvent, error) {
	path := fmt.Sprintf("/economics/burn/history?limit=%d", limit)
	resp, err := s.client.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("burn history retrieval failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			BurnEvents []*BurnEvent `json:"burn_events"`
			Count      int          `json:"count"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("burn history retrieval failed: %s", result.Error)
	}

	return result.Data.BurnEvents, nil
}

// GetTotal retrieves the total amount of burned tokens
func (s *BurnService) GetTotal(ctx context.Context) (string, error) {
	resp, err := s.client.Get(ctx, "/economics/burn/total")
	if err != nil {
		return "", fmt.Errorf("total burned retrieval failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			TotalBurned string    `json:"total_burned"`
			Timestamp   time.Time `json:"timestamp"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("total burned retrieval failed: %s", result.Error)
	}

	return result.Data.TotalBurned, nil
}

// Rules Service Methods

// Get retrieves current economic rules
func (s *RulesService) Get(ctx context.Context) (*EconomicRules, error) {
	resp, err := s.client.Get(ctx, "/economics/rules")
	if err != nil {
		return nil, fmt.Errorf("rules retrieval failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool           `json:"success"`
		Data    *EconomicRules `json:"data"`
		Error   string         `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("rules retrieval failed: %s", result.Error)
	}

	return result.Data, nil
}

// Update updates economic rules (requires admin privileges)
func (s *RulesService) Update(ctx context.Context, rules *EconomicRules) (*EconomicRules, error) {
	resp, err := s.client.Put(ctx, "/economics/rules", rules)
	if err != nil {
		return nil, fmt.Errorf("rules update failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Message string         `json:"message"`
			Rules   *EconomicRules `json:"rules"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("rules update failed: %s", result.Error)
	}

	return result.Data.Rules, nil
}

// Gateway Service Methods

// GetRoutes retrieves current gateway routes
func (s *RoutesService) GetRoutes(ctx context.Context) ([]*Route, error) {
	resp, err := s.client.Get(ctx, "/gateway/routes")
	if err != nil {
		return nil, fmt.Errorf("routes retrieval failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool     `json:"success"`
		Data    []*Route `json:"data"`
		Error   string   `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("routes retrieval failed: %s", result.Error)
	}

	return result.Data, nil
}

// GetStatus retrieves gateway status
func (s *StatusService) GetStatus(ctx context.Context) (*GatewayStatus, error) {
	resp, err := s.client.Get(ctx, "/gateway/status")
	if err != nil {
		return nil, fmt.Errorf("status retrieval failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool           `json:"success"`
		Data    *GatewayStatus `json:"data"`
		Error   string         `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("status retrieval failed: %s", result.Error)
	}

	return result.Data, nil
}

// Health Service Methods

// Check performs a health check on the gateway services
func (s *HealthService) Check(ctx context.Context) (bool, error) {
	resp, err := s.client.Get(ctx, "/economics/health")
	if err != nil {
		return false, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Success, nil
}

// Integration Service Methods

// GetStatus retrieves integration status with KNIRV components
func (s *IntegrationService) GetStatus(ctx context.Context) (*IntegrationStatus, error) {
	resp, err := s.client.Get(ctx, "/economics/integration/status")
	if err != nil {
		return nil, fmt.Errorf("integration status retrieval failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool               `json:"success"`
		Data    *IntegrationStatus `json:"data"`
		Error   string             `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("integration status retrieval failed: %s", result.Error)
	}

	return result.Data, nil
}
