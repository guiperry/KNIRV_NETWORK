package p2pconsensus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// GatewayProxy is a client that proxies P2P operations through KNIRVGATEWAY.
type GatewayProxy struct {
	gatewayURL    string
	networkID     string
	networkSecret string
	callbackPath  string
	client        *http.Client
	mu            sync.RWMutex
	stopChan      chan struct{}
}

// NewGatewayProxy creates a new gateway proxy client. networkSecret is an
// optional pre-shared key used to authenticate messages to/from the gateway.
func NewGatewayProxy(gatewayURL, networkID, callbackPath string, networkSecret ...string) *GatewayProxy {
	secret := ""
	if len(networkSecret) > 0 {
		secret = networkSecret[0]
	}
	return &GatewayProxy{
		gatewayURL:    gatewayURL,
		networkID:     networkID,
		networkSecret: secret,
		callbackPath:  callbackPath,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

// Register tells KNIRVGATEWAY where to send CRDT operations.
func (g *GatewayProxy) Register() error {
	payload := map[string]string{
		"socket_path": g.callbackPath,
		"network_id":  g.networkID,
		"topic":       Topic(g.networkID, g.networkSecret),
	}
	data, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost,
		g.gatewayURL+"/knirvbase/register-callback", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("register callback: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("register callback: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register-callback returned %d", resp.StatusCode)
	}
	return nil
}

// PublishOperation sends a CRDT operation to the gateway for broadcast.
func (g *GatewayProxy) PublishOperation(ctx context.Context, op OperationEnvelope) error {
	data, err := json.Marshal(op)
	if err != nil {
		return fmt.Errorf("marshal operation: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		g.gatewayURL+"/knirvbase/publish-op", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Authenticate the operation payload with the network secret.
	if sig := SignMessage(g.networkID, g.networkSecret, data); sig != "" {
		req.Header.Set("X-KNIRV-Signature", sig)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("publish op: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("publish-op returned %d", resp.StatusCode)
	}
	return nil
}

// DiscoverPeers asks the gateway for peers in the given network.
func (g *GatewayProxy) DiscoverPeers(query string) ([]PeerInfo, error) {
	url := fmt.Sprintf("%s/knirvbase/discover?network_id=%s", g.gatewayURL, g.networkID)
	if query != "" {
		url += "&query=" + query
	}
	resp, err := g.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("discover peers: %w", err)
	}
	defer resp.Body.Close()
	var peers []PeerInfo
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, fmt.Errorf("decode peers: %w", err)
	}
	return peers, nil
}

// Health checks if the gateway is reachable.
func (g *GatewayProxy) Health() bool {
	timeout, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(timeout, http.MethodGet, g.gatewayURL+"/p2p/health", nil)
	if err != nil {
		return false
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Close shuts down the proxy.
func (g *GatewayProxy) Close() {
	select {
	case <-g.stopChan:
	default:
		close(g.stopChan)
	}
}

// UnixSocketDialer returns an http.Client that dials the given Unix socket path.
func UnixSocketDialer(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
		Timeout: 30 * time.Second,
	}
}
