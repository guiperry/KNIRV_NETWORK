// PoAu-D (Proof of Authority using Delegation) Service Client
// This file provides SDK methods for interacting with the PoAu-D consensus management API

package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway/internal/requestconfig"
	"github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway/option"
)

// PoAuDStatus represents the status of the PoAu-D consensus mechanism
type PoAuDStatus struct {
	Enabled                bool                   `json:"enabled"`
	NetworkAuthorsCount    int                    `json:"network_authors_count,omitempty"`
	MainPoolSize          int                    `json:"main_pool_size,omitempty"`
	PasPoolSize           int                    `json:"pas_pool_size,omitempty"`
	DelegatedTransactions int                    `json:"delegated_transactions,omitempty"`
	DelegationStats       map[string]interface{} `json:"delegation_stats,omitempty"`
}

// NetworkAuthor represents a Network Author Peer
type NetworkAuthor struct {
	Address string `json:"address"`
}

// NetworkAuthorsResponse represents the response from the network authors endpoint
type NetworkAuthorsResponse struct {
	NetworkAuthors []string `json:"network_authors"`
	Count          int      `json:"count"`
}

// PoAuDResponse represents a standard PoAu-D API response
type PoAuDResponse struct {
	Success bool   `json:"success,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
	Message string `json:"message,omitempty"`
	Address string `json:"address,omitempty"`
}

// Enable enables the PoAu-D consensus mechanism
func (s *PoAuDService) Enable(ctx context.Context, opts ...option.RequestOption) (*PoAuDResponse, error) {
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodPost, "/poaud/enable", nil, &PoAuDResponse{}, opts...)
	return requestconfig.ExecuteNewRequest[*PoAuDResponse](cfg)
}

// Disable disables the PoAu-D consensus mechanism (fallback to PoW)
func (s *PoAuDService) Disable(ctx context.Context, opts ...option.RequestOption) (*PoAuDResponse, error) {
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodPost, "/poaud/disable", nil, &PoAuDResponse{}, opts...)
	return requestconfig.ExecuteNewRequest[*PoAuDResponse](cfg)
}

// GetStatus retrieves the current PoAu-D status and statistics
func (s *PoAuDService) GetStatus(ctx context.Context, opts ...option.RequestOption) (*PoAuDStatus, error) {
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodGet, "/poaud/status", nil, &PoAuDStatus{}, opts...)
	return requestconfig.ExecuteNewRequest[*PoAuDStatus](cfg)
}

// AddNetworkAuthor adds an address to the Network Authors set
func (s *NetworkAuthorsService) Add(ctx context.Context, address string, opts ...option.RequestOption) (*PoAuDResponse, error) {
	body := NetworkAuthor{Address: address}
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodPost, "/poaud/network-authors/add", body, &PoAuDResponse{}, opts...)
	return requestconfig.ExecuteNewRequest[*PoAuDResponse](cfg)
}

// RemoveNetworkAuthor removes an address from the Network Authors set
func (s *NetworkAuthorsService) Remove(ctx context.Context, address string, opts ...option.RequestOption) (*PoAuDResponse, error) {
	body := NetworkAuthor{Address: address}
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodPost, "/poaud/network-authors/remove", body, &PoAuDResponse{}, opts...)
	return requestconfig.ExecuteNewRequest[*PoAuDResponse](cfg)
}

// List retrieves all current Network Authors
func (s *NetworkAuthorsService) List(ctx context.Context, opts ...option.RequestOption) (*NetworkAuthorsResponse, error) {
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodGet, "/poaud/network-authors", nil, &NetworkAuthorsResponse{}, opts...)
	return requestconfig.ExecuteNewRequest[*NetworkAuthorsResponse](cfg)
}

// PoAuDClient provides a convenient interface for PoAu-D operations
type PoAuDClient struct {
	service *PoAuDService
}

// NewPoAuDClient creates a new PoAu-D client
func NewPoAuDClient(opts ...option.RequestOption) *PoAuDClient {
	client := NewClient(opts...)
	return &PoAuDClient{
		service: &client.PoAuD,
	}
}

// EnableConsensus enables PoAu-D consensus mechanism
func (c *PoAuDClient) EnableConsensus(ctx context.Context) (*PoAuDResponse, error) {
	return c.service.Enable(ctx)
}

// DisableConsensus disables PoAu-D consensus mechanism
func (c *PoAuDClient) DisableConsensus(ctx context.Context) (*PoAuDResponse, error) {
	return c.service.Disable(ctx)
}

// GetConsensusStatus gets the current PoAu-D status
func (c *PoAuDClient) GetConsensusStatus(ctx context.Context) (*PoAuDStatus, error) {
	return c.service.GetStatus(ctx)
}

// AddNetworkAuthor adds a Network Author Peer
func (c *PoAuDClient) AddNetworkAuthor(ctx context.Context, address string) (*PoAuDResponse, error) {
	return c.service.NetworkAuthors.Add(ctx, address)
}

// RemoveNetworkAuthor removes a Network Author Peer
func (c *PoAuDClient) RemoveNetworkAuthor(ctx context.Context, address string) (*PoAuDResponse, error) {
	return c.service.NetworkAuthors.Remove(ctx, address)
}

// ListNetworkAuthors lists all Network Author Peers
func (c *PoAuDClient) ListNetworkAuthors(ctx context.Context) (*NetworkAuthorsResponse, error) {
	return c.service.NetworkAuthors.List(ctx)
}

// IsPoAuDEnabled checks if PoAu-D is currently enabled
func (c *PoAuDClient) IsPoAuDEnabled(ctx context.Context) (bool, error) {
	status, err := c.GetConsensusStatus(ctx)
	if err != nil {
		return false, err
	}
	return status.Enabled, nil
}

// GetNetworkAuthorCount returns the number of current Network Authors
func (c *PoAuDClient) GetNetworkAuthorCount(ctx context.Context) (int, error) {
	authors, err := c.ListNetworkAuthors(ctx)
	if err != nil {
		return 0, err
	}
	return authors.Count, nil
}

// IsNetworkAuthor checks if an address is a Network Author
func (c *PoAuDClient) IsNetworkAuthor(ctx context.Context, address string) (bool, error) {
	authors, err := c.ListNetworkAuthors(ctx)
	if err != nil {
		return false, err
	}
	
	for _, author := range authors.NetworkAuthors {
		if author == address {
			return true, nil
		}
	}
	return false, nil
}

// GetDelegationStatistics returns delegation statistics
func (c *PoAuDClient) GetDelegationStatistics(ctx context.Context) (map[string]interface{}, error) {
	status, err := c.GetConsensusStatus(ctx)
	if err != nil {
		return nil, err
	}
	
	stats := map[string]interface{}{
		"enabled":                 status.Enabled,
		"network_authors_count":   status.NetworkAuthorsCount,
		"main_pool_size":         status.MainPoolSize,
		"pas_pool_size":          status.PasPoolSize,
		"delegated_transactions": status.DelegatedTransactions,
	}
	
	if status.DelegationStats != nil {
		for k, v := range status.DelegationStats {
			stats[k] = v
		}
	}
	
	return stats, nil
}

// ValidateNetworkAuthor validates that an address is properly formatted for use as a Network Author
func ValidateNetworkAuthor(address string) error {
	if address == "" {
		return fmt.Errorf("network author address cannot be empty")
	}
	
	if len(address) < 10 {
		return fmt.Errorf("network author address too short: %s", address)
	}
	
	// Add more validation as needed based on KNIRV address format
	return nil
}

// PoAuDConfig represents PoAu-D configuration options
type PoAuDConfig struct {
	Enabled                 bool   `json:"enabled"`
	DelegationInterval      string `json:"delegation_interval"`
	MaxSubpoolStaleTime     string `json:"max_subpool_stale_time"`
	MaxPapSubpoolQueue      int    `json:"max_pap_subpool_queue"`
	StatusAdvertiseInterval string `json:"status_advertise_interval"`
}

// GetDefaultPoAuDConfig returns default PoAu-D configuration
func GetDefaultPoAuDConfig() *PoAuDConfig {
	return &PoAuDConfig{
		Enabled:                 false,
		DelegationInterval:      "10s",
		MaxSubpoolStaleTime:     "5m",
		MaxPapSubpoolQueue:      100,
		StatusAdvertiseInterval: "30m",
	}
}
