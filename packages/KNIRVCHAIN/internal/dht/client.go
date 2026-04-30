package dht

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/internal/types"

	"github.com/libp2p/go-libp2p/core/peer"
)

// DiscoveryResourceType mirrors types.DiscoveryResourceType for convenience.
type DiscoveryResourceType = types.DiscoveryResourceType

const (
	DiscoveryResourceTypeChain         DiscoveryResourceType = "chain"
	DiscoveryResourceTypeNRN           DiscoveryResourceType = "nrn"
	DiscoveryResourceTypeFile          DiscoveryResourceType = "file"
	DiscoveryResourceTypeAPI           DiscoveryResourceType = "api"
	DiscoveryResourceTypePlugin        DiscoveryResourceType = "plugin"
	DiscoveryResourceTypeGeneratedDoc  DiscoveryResourceType = "generated_doc"
	DiscoveryResourceTypeDataset       DiscoveryResourceType = "dataset"
	DiscoveryResourceTypeModelArtifact DiscoveryResourceType = "model_artifact"
	DiscoveryResourceTypeService       DiscoveryResourceType = "service"
)

// DiscoveryClient handles DHT operations via KNIRVGATEWAY Unix socket.
type DiscoveryClient struct {
	socketPath  string
	httpClient  *http.Client
	source      string
	chainID     string
	nodeRole    string
	mu          sync.RWMutex
	stopChan    chan struct{}
	closeOnce   sync.Once
}

// NewDiscoveryClient creates a new DHT client and registers with KNIRVGATEWAY.
func NewDiscoveryClient(chainID string, p2pPort int, clientOnly bool, isBootnode bool, role config.Role, cfg *config.Config) (*DiscoveryClient, error) {
	socketPath := resolveGatewaySocket(cfg)

	c := &DiscoveryClient{
		socketPath: socketPath,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
				},
			},
		},
		source:   "knirvchain",
		chainID:  chainID,
		nodeRole: role.String(),
		stopChan: make(chan struct{}),
	}

	// Register this node's callback socket with KNIRVGATEWAY
	if cfg != nil && cfg.SocketPath != "" {
		if err := c.registerCallback(cfg.SocketPath); err != nil {
			// Non-fatal: KNIRVGATEWAY may not be running yet
			fmt.Printf("[%s][%s] Warning: Could not register P2P callback with KNIRVGATEWAY: %v\n", role.String(), chainID, err)
		}
	}

	return c, nil
}

// resolveGatewaySocket returns the KNIRVGATEWAY unix socket path.
func resolveGatewaySocket(cfg *config.Config) string {
	if s := os.Getenv("GATEWAY_SOCKET_PATH"); s != "" {
		return s
	}
	// Default XDG path used by KNIRVSERVER
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "knirvserver", "sockets", "gateway.sock")
}

// registerCallback tells KNIRVGATEWAY where to send received blocks/txs.
func (c *DiscoveryClient) registerCallback(chainSocketPath string) error {
	payload := map[string]string{"socket_path": chainSocketPath}
	data, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(
		"http://localhost/p2p/register-callback",
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

// CacheResource sends a resource to KNIRVGATEWAY for DHT announcement.
func (c *DiscoveryClient) CacheResource(ctx context.Context, id, resourceType string) error {
	payload := map[string]interface{}{
		"id":     id,
		"type":   resourceType,
		"source": c.source,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := c.httpClient.Post("http://localhost/dht/cache-resource", "application/json", bytes.NewBuffer(data))
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
func (c *DiscoveryClient) AnnounceGenericResource(id string, resourceType DiscoveryResourceType) error {
	return c.CacheResource(context.Background(), id, string(resourceType))
}

// AnnounceGenericResourceCtx is the context-aware version.
func (c *DiscoveryClient) AnnounceGenericResourceCtx(ctx context.Context, id string, resourceType DiscoveryResourceType) error {
	return c.CacheResource(ctx, id, string(resourceType))
}

// AnnounceMCPCapability caches an MCP capability for DHT announcement.
func (c *DiscoveryClient) AnnounceMCPCapability(ctx context.Context, id, mcpType string) error {
	return c.CacheResource(ctx, id, fmt.Sprintf("mcp:%s", mcpType))
}

// AnnounceMintedResource caches a minted resource for DHT announcement.
func (c *DiscoveryClient) AnnounceMintedResource(ctx context.Context, id string, resourceType DiscoveryResourceType) error {
	return c.CacheResource(ctx, id, string(resourceType))
}

// FindGenericResource implements DiscoveryService (no ctx version).
func (c *DiscoveryClient) FindGenericResource(id string, resourceType DiscoveryResourceType) ([]peer.AddrInfo, error) {
	return c.findResource(context.Background(), id, string(resourceType))
}

// FindResource implements DiscoveryService (ctx version).
func (c *DiscoveryClient) FindResource(ctx context.Context, id string, resourceType DiscoveryResourceType) ([]peer.AddrInfo, error) {
	return c.findResource(ctx, id, string(resourceType))
}

func (c *DiscoveryClient) findResource(ctx context.Context, id string, resourceType string) ([]peer.AddrInfo, error) {
	url := fmt.Sprintf("http://localhost/dht/find?id=%s&type=%s", id, resourceType)

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
	return c.findResource(ctx, id, fmt.Sprintf("mcp:%s", mcpTypeString))
}

// GetPeerID returns KNIRVGATEWAY's peer ID (which acts as our P2P identity).
func (c *DiscoveryClient) GetPeerID() string {
	resp, err := c.httpClient.Get("http://localhost/p2p/peer-id")
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
func (c *DiscoveryClient) GetSelfMultiaddrs() []string {
	resp, err := c.httpClient.Get("http://localhost/p2p/self-addrs")
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

// IsRecentlyRegistered returns false — peer tracking is handled by KNIRVGATEWAY.
func (c *DiscoveryClient) IsRecentlyRegistered(_ string) bool {
	return false
}

// ConnectToPeer is a no-op — connections are managed by KNIRVGATEWAY.
func (c *DiscoveryClient) ConnectToPeer(_ peer.AddrInfo, _ context.Context) error {
	return nil
}

// Run is a no-op — KNIRVGATEWAY manages the P2P lifecycle.
func (c *DiscoveryClient) Run(_ time.Duration) {}

// Close signals the discovery client to stop.
func (c *DiscoveryClient) Close() {
	c.closeOnce.Do(func() {
		close(c.stopChan)
	})
}
