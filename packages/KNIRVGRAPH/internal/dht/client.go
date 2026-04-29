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

// Client handles DHT operations via KNIRVGATEWAY Unix socket.
type Client struct {
	socketPath string
	httpClient  *http.Client
	source      string // "knirvgraph"
}

// NewClient creates a new DHT client for KNIRVGRAPH.
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
				},
			},
		},
		source: "knirvgraph",
	}
}

// CacheResource sends a resource to KNIRVGATEWAY for DHT announcement.
func (c *Client) CacheResource(ctx context.Context, id string, resourceType string, multiaddr string) error {
	url := "http://localhost/dht/cache-resource"

	payload := map[string]interface{}{
		"id":           id,
		"type":         resourceType,
		"multiaddress": multiaddr,
		"source":       c.source,
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

// AnnounceSkill caches a skill resource for DHT announcement.
func (c *Client) AnnounceSkill(ctx context.Context, skillID string, multiaddr string) error {
	return c.CacheResource(ctx, skillID, "skill", multiaddr)
}

// AnnounceCapability caches a capability resource for DHT announcement.
func (c *Client) AnnounceCapability(ctx context.Context, capID string, multiaddr string) error {
	return c.CacheResource(ctx, capID, "capability", multiaddr)
}

// AnnounceProperty caches a property resource for DHT announcement.
func (c *Client) AnnounceProperty(ctx context.Context, propID string, multiaddr string) error {
	return c.CacheResource(ctx, propID, "property", multiaddr)
}

// FindResource finds resource providers via KNIRVGATEWAY DHT query.
func (c *Client) FindResource(ctx context.Context, id string, resourceType string) ([]peer.AddrInfo, error) {
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
