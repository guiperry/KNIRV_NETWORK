package dht

import (
	"context"
	"log"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DHTClientAdapter adapts the new DHT client to the old DHTManagerInterface.
// This allows gradual migration from p2p.DHTManager to dht.Client.
type DHTClientAdapter struct {
	client *Client
}

// NewDHTClientAdapter creates a new adapter wrapping the DHT client.
// This replaces p2p.NewDHTManager in app.go.
func NewDHTClientAdapter(serviceID, chainID string, bootstrapPeers []string, enableAutoRelay bool) (*DHTClientAdapter, error) {
	// Use gateway at localhost:8080
	client := NewClient()

	log.Printf("DHTClientAdapter: created for service=%s, chain=%s", serviceID, chainID)

	return &DHTClientAdapter{client: client}, nil
}

// Start does nothing for the client adapter (connection is lazy)
func (a *DHTClientAdapter) Start() error {
	return nil
}

// Stop does nothing for the client adapter
func (a *DHTClientAdapter) Stop() {
	// Nothing to stop for HTTP client
}

// IsNetworkPaused always returns false for now (KNIRVGATEWAY manages this)
func (a *DHTClientAdapter) IsNetworkPaused() bool {
	return false
}

// AnnounceSkill announces a skill via KNIRVGATEWAY DHT
func (a *DHTClientAdapter) AnnounceSkill(skillID, name, description, category string, metadata map[string]string) error {
	// Build multiaddress from metadata or use default
	multiaddr := "/ip4/127.0.0.1/tcp/1317" // Default KNIRVGRAPH API port
	if ma, ok := metadata["multiaddress"]; ok {
		multiaddr = ma
	}

	return a.client.AnnounceSkill(context.Background(), skillID, multiaddr)
}

// AnnounceCapability announces a capability via KNIRVGATEWAY DHT
func (a *DHTClientAdapter) AnnounceCapability(capabilityID, name, description string, schema interface{}, metadata map[string]string) error {
	multiaddr := "/ip4/127.0.0.1/tcp/1317"
	if ma, ok := metadata["multiaddress"]; ok {
		multiaddr = ma
	}

	return a.client.AnnounceCapability(context.Background(), capabilityID, multiaddr)
}

// AnnounceProperty announces a property via KNIRVGATEWAY DHT
func (a *DHTClientAdapter) AnnounceProperty(propertyID, name, propertyType string, value interface{}, metadata map[string]string) error {
	multiaddr := "/ip4/127.0.0.1/tcp/1317"
	if ma, ok := metadata["multiaddress"]; ok {
		multiaddr = ma
	}

	return a.client.AnnounceProperty(context.Background(), propertyID, multiaddr)
}

// FindGraphServices returns empty (TODO: implement via KNIRVGATEWAY)
func (a *DHTClientAdapter) FindGraphServices() ([]peer.AddrInfo, error) {
	return nil, nil
}

// Publish publishes to a topic via KNIRVGATEWAY (TODO)
func (a *DHTClientAdapter) Publish(topic string, data []byte) error {
	return nil
}

// Subscribe subscribes to a topic via KNIRVGATEWAY (TODO)
func (a *DHTClientAdapter) Subscribe(topic string) (*pubsub.Subscription, error) {
	return nil, nil
}
