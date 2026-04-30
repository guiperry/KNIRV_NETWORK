package dht

import (
	"fmt"
	"sync"
	"time"
)

// DefaultTTL is the default TTL for cached resources (24 hours).
// Zero value means no expiration.
const DefaultTTL = 24 * time.Hour

// ResourceCache stores resources from KNIRVGRAPH/KNIRVCHAIN for DHT announcement.
type ResourceCache struct {
	mu        sync.RWMutex
	resources map[string]ResourceEntry // key: "id:type"
	ttl       time.Duration
}

// NewResourceCache creates a new ResourceCache with optional TTL.
// If ttl is zero, resources never expire.
func NewResourceCache(ttl ...time.Duration) *ResourceCache {
	cacheTTL := time.Duration(0)
	if len(ttl) > 0 {
		cacheTTL = ttl[0]
	}
	return &ResourceCache{
		resources: make(map[string]ResourceEntry),
		ttl:       cacheTTL,
	}
}

// AddResource adds or updates a resource in the cache.
func (rc *ResourceCache) AddResource(resource ResourceEntry) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	resource.Timestamp = time.Now()
	if rc.ttl > 0 {
		resource.ExpiresAt = time.Now().Add(rc.ttl)
	}
	key := fmt.Sprintf("%s:%s", resource.ID, resource.Type)
	rc.resources[key] = resource
}

// GetAllResources returns all non-expired cached resources.
// Expired entries are removed during this call.
func (rc *ResourceCache) GetAllResources() []ResourceEntry {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	result := make([]ResourceEntry, 0, len(rc.resources))

	for key, entry := range rc.resources {
		if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(now) {
			delete(rc.resources, key)
			continue
		}
		result = append(result, entry)
	}
	return result
}

// ClearCache removes all resources from the cache.
func (rc *ResourceCache) ClearCache() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.resources = make(map[string]ResourceEntry)
}

// GetResourceCount returns the number of cached resources (excluding expired).
func (rc *ResourceCache) GetResourceCount() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	count := 0
	for key, entry := range rc.resources {
		if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(now) {
			delete(rc.resources, key)
			continue
		}
		count++
	}
	return count
}

// CleanupExpired removes expired entries from the cache.
func (rc *ResourceCache) CleanupExpired() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	removed := 0
	for key, entry := range rc.resources {
		if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(now) {
			delete(rc.resources, key)
			removed++
		}
	}
	return removed
}
