package dht

import (
	"context"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ResourceEntry represents a resource cached for DHT announcement.
type ResourceEntry struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Multiaddress string    `json:"multiaddress,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	Source       string    `json:"source"` // "knirvgraph" or "knirvchain"
	ExpiresAt    time.Time `json:"expires_at,omitempty"` // zero value means no expiration
}

// DHTManagerInterface defines the interface for DHT operations.
type DHTManagerInterface interface {
	Start(ctx context.Context) error
	Stop() error
	Provide(ctx context.Context, cid cid.Cid) error
	FindProviders(ctx context.Context, cid cid.Cid) ([]peer.AddrInfo, error)
	AnnounceService(ctx context.Context, serviceID string) error
	FindServices(ctx context.Context, serviceID string) ([]peer.AddrInfo, error)
	PublishAnnouncement(ctx context.Context, topic string, data []byte) error
	SubscribeAnnouncements(ctx context.Context, topic string) (<-chan []byte, error)
	AnnounceAllCached(ctx context.Context) (int, error)
	GetResourceCache() ResourceCacheInterface
}

// ResourceCacheInterface manages resources to announce to DHT.
type ResourceCacheInterface interface {
	AddResource(resource ResourceEntry)
	GetAllResources() []ResourceEntry
	ClearCache()
	GetResourceCount() int
}
