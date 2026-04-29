package dht

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// DiscoveryResourceType represents a DHT resource type.
type DiscoveryResourceType string

const (
	DiscoveryResourceTypeChain         DiscoveryResourceType = "chain"
	DiscoveryResourceTypeNRN          DiscoveryResourceType = "nrn"
	DiscoveryResourceTypeFile          DiscoveryResourceType = "file"
	DiscoveryResourceTypeAPI           DiscoveryResourceType = "api"
	DiscoveryResourceTypePlugin        DiscoveryResourceType = "plugin"
	DiscoveryResourceTypeGeneratedDoc DiscoveryResourceType = "generated_doc"
	DiscoveryResourceTypeDataset       DiscoveryResourceType = "dataset"
	DiscoveryResourceTypeModelArtifact DiscoveryResourceType = "model_artifact"
	DiscoveryResourceTypeService       DiscoveryResourceType = "service"
)

// DiscoveryClient handles DHT operations via KNIRVGATEWAY Unix socket.
type DiscoveryClient struct {
	socketPath string
	httpClient  *http.Client
	source      string // "knirvchain"
}

// NewDiscoveryClient creates a new DHT client for KNIRVCHAIN.
func NewDiscoveryClient(socketPath string) *DiscoveryClient {
	return &DiscoveryClient{
		socketPath: socketPath,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
				},
			},
		},
		source: "knirvchain",
	}
}

// CacheResource sends a resource to KNIRVGATEWAY for DHT announcement.
func (c *DiscoveryClient) CacheResource(ctx context.Context, id, resourceType string) error {
	url := "http://localhost/dht/cache-resource"

	payload := map[string]interface{}{
		"id":     id,
		"type":   resourceType,
		"source": c.source,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to cache resource: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cache resource returned status %d", resp.StatusCode)
	}

	return nil
}

// AnnounceGenericResource caches a resource for DHT announcement via broadcast system.
func (c *DiscoveryClient) AnnounceGenericResource(ctx context.Context, id string, resourceType DiscoveryResourceType) error {
	return c.CacheResource(ctx, id, string(resourceType))
}

// AnnounceMCPCapability caches an MCP capability for DHT announcement.
func (c *DiscoveryClient) AnnounceMCPCapability(ctx context.Context, id, mcpType string) error {
	resourceType := fmt.Sprintf("mcp:%s", mcpType)
	return c.CacheResource(ctx, id, resourceType)
}

// AnnounceMintedResource caches a minted resource for DHT announcement.
func (c *DiscoveryClient) AnnounceMintedResource(ctx context.Context, id string, resourceType DiscoveryResourceType) error {
	return c.CacheResource(ctx, id, string(resourceType))
}

// FindGenericResource finds resource providers via KNIRVGATEWAY DHT query.
func (c *DiscoveryClient) FindGenericResource(ctx context.Context, id string, resourceType DiscoveryResourceType) ([]peer.AddrInfo, error) {
	url := fmt.Sprintf("http://localhost/dht/find?id=%s&type=%s", id, string(resourceType))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("DHT find failed: %w", err)
	}
	defer resp.Body.Close()

	var peers []peer.AddrInfo
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, err
	}

	return peers, nil
}

// FindMCPCapabilityProviders finds MCP capability providers.
func (c *DiscoveryClient) FindMCPCapabilityProviders(ctx context.Context, id string, mcpTypeString string) ([]peer.AddrInfo, error) {
	resourceType := fmt.Sprintf("mcp:%s", mcpTypeString)
	return c.FindGenericResource(ctx, id, DiscoveryResourceType(resourceType))
}
