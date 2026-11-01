package p2p

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/internal/types"

	"github.com/libp2p/go-libp2p/core/peer"
)

// DiscoveryResourceType mirrors types.DiscoveryResourceType for convenience.
type DiscoveryResourceType = types.DiscoveryResourceType

const (
	callbackRetryInterval = 2 * time.Second
	callbackRetryWindow   = 2 * time.Minute
)

// GatewayClient handles DHT operations via KNIRVGATEWAY HTTP.
type GatewayClient struct {
	httpClient *http.Client
	source     string
	chainID    string
	nodeRole   string
	gatewayURL string
	mu         sync.RWMutex
	stopChan   chan struct{}
	closeOnce  sync.Once
}

// NewGatewayClient creates a new GatewayClient and registers with KNIRVGATEWAY.
func NewGatewayClient(chainID string, p2pPort int, clientOnly bool, isBootnode bool, role config.Role, cfg *config.Config) (*GatewayClient, error) {
	gatewayURL := cfg.Consensus.GatewayURL
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}

	timeout := 10 * time.Second
	if cfg.Consensus.GatewayTimeout != "" {
		if d, err := time.ParseDuration(cfg.Consensus.GatewayTimeout); err == nil {
			timeout = d
		}
	}

	c := &GatewayClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		source:     "knirvchain",
		chainID:    chainID,
		nodeRole:   role.String(),
		gatewayURL: gatewayURL,
		stopChan:   make(chan struct{}),
	}

	if cfg != nil && cfg.SocketPath != "" {
		if err := c.registerCallback(cfg.SocketPath); err != nil {
			// The gateway is started after the chain, so keep trying in the background.
			fmt.Printf("[%s][%s] KNIRVGATEWAY not reachable yet (%v); retrying P2P callback registration in the background\n", role.String(), chainID, err)
			go c.retryRegisterCallback(cfg.SocketPath, role.String())
		}
	}

	return c, nil
}

// retryRegisterCallback retries registerCallback until it succeeds, the client is
// closed, or callbackRetryWindow passes.
func (c *GatewayClient) retryRegisterCallback(chainSocketPath, role string) {
	deadline := time.NewTimer(callbackRetryWindow)
	defer deadline.Stop()
	tick := time.NewTicker(callbackRetryInterval)
	defer tick.Stop()
	for {
		select {
		case <-c.stopChan:
			return
		case <-deadline.C:
			fmt.Printf("[%s][%s] Warning: Could not register P2P callback with KNIRVGATEWAY within %s\n", role, c.chainID, callbackRetryWindow)
			return
		case <-tick.C:
			if err := c.registerCallback(chainSocketPath); err == nil {
				fmt.Printf("[%s][%s] Registered P2P callback with KNIRVGATEWAY\n", role, c.chainID)
				return
			}
		}
	}
}

func (c *GatewayClient) registerCallback(chainSocketPath string) error {
	payload := map[string]string{"socket_path": chainSocketPath}
	data, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(
		c.gatewayURL+"/p2p/register-callback",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register-callback returned %d", resp.StatusCode)
	}
	return nil
}

func (c *GatewayClient) gatewayURLWithoutScheme() string {
	if len(c.gatewayURL) > 7 && c.gatewayURL[:7] == "http://" {
		return c.gatewayURL[7:]
	}
	if len(c.gatewayURL) > 8 && c.gatewayURL[:8] == "https://" {
		return c.gatewayURL[8:]
	}
	return c.gatewayURL
}

// CacheResource sends a resource to KNIRVGATEWAY for DHT announcement.
func (c *GatewayClient) CacheResource(ctx context.Context, id, resourceType string) error {
	payload := map[string]interface{}{
		"id":     id,
		"type":   resourceType,
		"source": c.source,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := c.httpClient.Post(c.gatewayURL+"/dht/cache-resource", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to cache resource: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cache resource returned status %d", resp.StatusCode)
	}

	return nil
}

// AnnounceGenericResource implements DiscoveryService (no ctx version).
func (c *GatewayClient) AnnounceGenericResource(id string, resourceType DiscoveryResourceType) error {
	return c.CacheResource(context.Background(), id, string(resourceType))
}

// AnnounceGenericResourceCtx is the context-aware version.
func (c *GatewayClient) AnnounceGenericResourceCtx(ctx context.Context, id string, resourceType DiscoveryResourceType) error {
	return c.CacheResource(ctx, id, string(resourceType))
}

// AnnounceMCPCapability caches an MCP capability for DHT announcement.
func (c *GatewayClient) AnnounceMCPCapability(ctx context.Context, id, mcpType string) error {
	return c.CacheResource(ctx, id, fmt.Sprintf("mcp:%s", mcpType))
}

// AnnounceMintedResource caches a minted resource for DHT announcement.
func (c *GatewayClient) AnnounceMintedResource(ctx context.Context, id string, resourceType DiscoveryResourceType) error {
	return c.CacheResource(ctx, id, string(resourceType))
}

// FindGenericResource implements DiscoveryService (no ctx version).
func (c *GatewayClient) FindGenericResource(id string, resourceType DiscoveryResourceType) ([]peer.AddrInfo, error) {
	return c.findResource(context.Background(), id, string(resourceType))
}

// FindResource implements DiscoveryService (ctx version).
func (c *GatewayClient) FindResource(ctx context.Context, id string, resourceType DiscoveryResourceType) ([]peer.AddrInfo, error) {
	return c.findResource(ctx, id, string(resourceType))
}

func (c *GatewayClient) findResource(ctx context.Context, id string, resourceType string) ([]peer.AddrInfo, error) {
	url := fmt.Sprintf(c.gatewayURL+"/dht/find?id=%s&type=%s", id, resourceType)

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
func (c *GatewayClient) FindMCPCapabilityProviders(ctx context.Context, id string, mcpTypeString string) ([]peer.AddrInfo, error) {
	return c.findResource(ctx, id, fmt.Sprintf("mcp:%s", mcpTypeString))
}

// GetPeerID returns KNIRVGATEWAY's peer ID (which acts as our P2P identity).
func (c *GatewayClient) GetPeerID() string {
	resp, err := c.httpClient.Get(c.gatewayURL + "/p2p/peer-id")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		PeerID string `json:"peer_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	return result.PeerID
}

// GetSelfMultiaddrs returns KNIRVGATEWAY's multiaddresses.
func (c *GatewayClient) GetSelfMultiaddrs() []string {
	resp, err := c.httpClient.Get(c.gatewayURL + "/p2p/self-addrs")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Addrs []string `json:"addrs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	return result.Addrs
}

// GetGatewaySocket returns the socket path if configured.
func (c *GatewayClient) GetGatewaySocket() string {
	return ""
}

// IsRecentlyRegistered returns false — peer tracking is handled by KNIRVGATEWAY.
func (c *GatewayClient) IsRecentlyRegistered(_ string) bool {
	return false
}

// ConnectToPeer is a no-op — connections are managed by KNIRVGATEWAY.
func (c *GatewayClient) ConnectToPeer(_ peer.AddrInfo, _ context.Context) error {
	return nil
}

// Run is a no-op — KNIRVGATEWAY manages the P2P lifecycle.
func (c *GatewayClient) Run(_ time.Duration) {}

// Close signals the gateway client to stop.
func (c *GatewayClient) Close() {
	c.closeOnce.Do(func() {
		close(c.stopChan)
	})
}
