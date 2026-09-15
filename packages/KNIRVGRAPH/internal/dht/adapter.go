package dht

import (
	"context"
	"errors"
	"fmt"
	"sort"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DHTClientAdapter adapts the DHT client to the old DHTManagerInterface.
type DHTClientAdapter struct {
	client *Client
}

// NewDHTClientAdapter creates a new adapter wrapping the DHT client.
func NewDHTClientAdapter(serviceID, chainID string, bootstrapPeers []string, enableAutoRelay bool) (*DHTClientAdapter, error) {
	client := NewClient()
	return &DHTClientAdapter{client: client}, nil
}

// Start does nothing for the client adapter (connection is lazy).
func (a *DHTClientAdapter) Start() error { return nil }

// Stop does nothing for the client adapter.
func (a *DHTClientAdapter) Stop() {}

// IsNetworkPaused always returns false (KNIRVGATEWAY manages this).
func (a *DHTClientAdapter) IsNetworkPaused() bool { return false }

// AnnounceSkill announces a skill via KNIRVGATEWAY DHT.
func (a *DHTClientAdapter) AnnounceSkill(skillID, name, description, category string, metadata map[string]string) error {
	multiaddr := defaultMultiaddr
	if ma, ok := metadata["multiaddress"]; ok {
		multiaddr = ma
	}
	return a.client.AnnounceSkill(context.Background(), skillID, multiaddr)
}

// AnnounceCapability announces a capability via KNIRVGATEWAY DHT.
func (a *DHTClientAdapter) AnnounceCapability(capabilityID, name, description string, schema interface{}, metadata map[string]string) error {
	multiaddr := defaultMultiaddr
	if ma, ok := metadata["multiaddress"]; ok {
		multiaddr = ma
	}
	return a.client.AnnounceCapability(context.Background(), capabilityID, multiaddr)
}

// AnnounceProperty announces a property via KNIRVGATEWAY DHT.
func (a *DHTClientAdapter) AnnounceProperty(propertyID, name, propertyType string, value interface{}, metadata map[string]string) error {
	multiaddr := defaultMultiaddr
	if ma, ok := metadata["multiaddress"]; ok {
		multiaddr = ma
	}
	return a.client.AnnounceProperty(context.Background(), propertyID, multiaddr)
}

// defaultMultiaddr is KNIRVGRAPH's local API address, used when the caller
// supplies no more specific address for the announced resource.
const defaultMultiaddr = "/ip4/127.0.0.1/tcp/1317"

// graphServiceResourceType is the gateway resource type under which graph
// service addresses are cached and looked up.
const graphServiceResourceType = "graph-service"

// FindGraphServices returns the known KNIRVGRAPH peer addresses.
//
// This previously returned (nil, nil) unconditionally, which callers could not
// distinguish from "no peers found" and which silently disabled every
// graph-service discovery path.
//
// It is implemented against KNIRVGATEWAY's real /dht/find route
// (internal/server/server.go), reached through Client.FindResource. The lookup
// is done by well-known resource id so it does not depend on a specific peer
// having announced a particular skill first; when nothing is cached yet the
// result is a genuinely empty list, not a placeholder.
func (a *DHTClientAdapter) FindGraphServices() ([]peer.AddrInfo, error) {
	if a == nil || a.client == nil {
		return nil, errors.New("DHT client is not configured")
	}

	addresses, err := a.client.FindResource(context.Background(), graphServiceResourceType, graphServiceResourceType)
	if err != nil {
		return nil, fmt.Errorf("find graph services via KNIRVGATEWAY: %w", err)
	}

	// Deterministic ordering so callers and tests are not exposed to map
	// iteration order.
	sort.Slice(addresses, func(i, j int) bool {
		return addresses[i].ID.String() < addresses[j].ID.String()
	})
	return addresses, nil
}

// Publish publishes to a topic.
//
// KNIRVGATEWAY exposes no pub/sub publish route — its DHT surface is
// /dht/announce, /dht/find, /dht/peers, /dht/bootstrap and the resource-cache
// routes, and SetupCRDTPubSub is only reachable from the internal KNIRVBASE
// callback handler. There is therefore no honest implementation of this method
// over the HTTP client: the previous `return nil` claimed a successful publish
// that never happened, so any caller that relied on it (DRQ's Q-table gossip)
// silently never exchanged state.
//
// It fails loudly instead. Wiring this for real needs either a gateway
// pub/sub route or a direct libp2p pubsub node in this process.
func (a *DHTClientAdapter) Publish(topic string, data []byte) error {
	return fmt.Errorf("%w: KNIRVGATEWAY exposes no pub/sub publish route, so topic %q cannot be published",
		ErrPubSubUnavailable, topic)
}

// Subscribe subscribes to a topic.
//
// Unimplementable over this transport for the same reason as Publish, and
// additionally because the return type is a libp2p *pubsub.Subscription, which
// only a real libp2p pubsub node can construct. Returning nil, nil previously
// handed callers a nil subscription that would panic on use; it now fails
// loudly at the call site instead.
func (a *DHTClientAdapter) Subscribe(topic string) (*pubsub.Subscription, error) {
	return nil, fmt.Errorf("%w: KNIRVGATEWAY exposes no pub/sub subscribe route, so topic %q cannot be subscribed",
		ErrPubSubUnavailable, topic)
}

// ErrPubSubUnavailable marks a pub/sub operation that this transport cannot
// serve, so callers can distinguish "not implemented here" from a transport
// failure and pick an alternative transport deliberately.
var ErrPubSubUnavailable = errors.New("dht: pub/sub unavailable over the KNIRVGATEWAY HTTP client")
