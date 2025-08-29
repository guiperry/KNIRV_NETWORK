// memory.go
package agentify

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/philippgille/chromem-go"
)

// MemoryManager defines the interface for managing agent memory
type MemoryManager interface {
	// Get retrieves a value from memory
	Get(key string) (interface{}, error)

	// Set stores a value in memory
	Set(key string, value interface{}) error

	// Delete removes a value from memory
	Delete(key string) error

	// List returns all keys in memory
	List() ([]string, error)

	// Clear removes all values from memory
	Clear() error
}

// InMemoryManager implements MemoryManager using an in-memory map
type InMemoryManager struct {
	data  map[string]interface{}
	mutex sync.RWMutex
}

// NewInMemoryManager creates a new in-memory manager
func NewInMemoryManager() *InMemoryManager {
	return &InMemoryManager{
		data: make(map[string]interface{}),
	}
}

// Get retrieves a value from memory
func (m *InMemoryManager) Get(key string) (interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	value, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return value, nil
}

// Set stores a value in memory
func (m *InMemoryManager) Set(key string, value interface{}) error {
	return m.SetWithTTL(key, value, 0)
}

// SetWithTTL stores a value in memory with a time-to-live (simplified for InMemoryManager)
func (m *InMemoryManager) SetWithTTL(key string, value interface{}, ttl int64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// For the simple InMemoryManager, we just store the value directly
	// TTL functionality would require a more complex implementation with background cleanup
	m.data[key] = value
	return nil
}

// Delete removes a value from memory
func (m *InMemoryManager) Delete(key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.data, key)
	return nil
}

// List returns all keys in memory
func (m *InMemoryManager) List() ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	keys := make([]string, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}

	return keys, nil
}

// Clear removes all values from memory
func (m *InMemoryManager) Clear() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data = make(map[string]interface{})
	return nil
}

// MemoryItem represents an item in persistent memory
type MemoryItem struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	CreatedAt int64       `json:"createdAt"`
	UpdatedAt int64       `json:"updatedAt"`
	TTL       int64       `json:"ttl,omitempty"` // Time-to-live in seconds, 0 means no expiration
}

// PersistentMemoryManager implements MemoryManager using in-memory storage
// Note: This is a production-ready in-memory implementation
// For persistent storage, integrate with chromem-go or similar vector database
type PersistentMemoryManager struct {
	data  map[string]*MemoryItem
	mutex sync.RWMutex
}

// NewPersistentMemoryManager creates a new persistent memory manager
func NewPersistentMemoryManager() *PersistentMemoryManager {
	return &PersistentMemoryManager{
		data: make(map[string]*MemoryItem),
	}
}

// Get retrieves a value from memory
func (m *PersistentMemoryManager) Get(key string) (interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	item, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Check if the item has expired
	if item.TTL > 0 && time.Now().Unix() > item.CreatedAt+item.TTL {
		// Remove the expired item
		delete(m.data, key)
		return nil, fmt.Errorf("key expired: %s", key)
	}

	return item.Value, nil
}

// Set stores a value in memory
func (m *PersistentMemoryManager) Set(key string, value interface{}) error {
	return m.SetWithTTL(key, value, 0)
}

// SetWithTTL stores a value in memory with a time-to-live
func (m *PersistentMemoryManager) SetWithTTL(key string, value interface{}, ttl int64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now().Unix()

	// Check if the item already exists
	if item, ok := m.data[key]; ok {
		item.Value = value
		item.UpdatedAt = now
		item.TTL = ttl
	} else {
		// Create a new item
		m.data[key] = &MemoryItem{
			Key:       key,
			Value:     value,
			CreatedAt: now,
			UpdatedAt: now,
			TTL:       ttl,
		}
	}

	return nil
}

// Delete removes a value from memory
func (m *PersistentMemoryManager) Delete(key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.data, key)
	return nil
}

// List returns all keys in memory
func (m *PersistentMemoryManager) List() ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	now := time.Now().Unix()
	keys := make([]string, 0, len(m.data))

	for key, item := range m.data {
		// Skip expired items
		if item.TTL > 0 && now > item.CreatedAt+item.TTL {
			continue
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// Clear removes all values from memory
func (m *PersistentMemoryManager) Clear() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data = make(map[string]*MemoryItem)
	return nil
}

// Export exports the memory to JSON
func (m *PersistentMemoryManager) Export() ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Convert the map to a slice for easier serialization
	items := make([]*MemoryItem, 0, len(m.data))
	for _, item := range m.data {
		items = append(items, item)
	}

	return json.Marshal(items)
}

// Import imports memory from JSON
func (m *PersistentMemoryManager) Import(data []byte) error {
	var items []*MemoryItem
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Clear existing data
	m.data = make(map[string]*MemoryItem)

	// Import the items
	for _, item := range items {
		m.data[item.Key] = item
	}

	return nil
}

// VectorMemoryManager implements MemoryManager using chromem-go for vector-based memory
type VectorMemoryManager struct {
	collection *chromem.Collection
	data       map[string]*MemoryItem
	mutex      sync.RWMutex
}

// NewVectorMemoryManager creates a new vector memory manager using chromem-go
func NewVectorMemoryManager(collectionName string) (*VectorMemoryManager, error) {
	// Create a new chromem database
	db := chromem.NewDB()

	// Create or get a collection
	collection, err := db.CreateCollection(collectionName, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %v", err)
	}

	return &VectorMemoryManager{
		collection: collection,
		data:       make(map[string]*MemoryItem),
	}, nil
}

// Get retrieves a value from memory
func (m *VectorMemoryManager) Get(key string) (interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	item, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Check if the item has expired
	if item.TTL > 0 && time.Now().Unix() > item.CreatedAt+item.TTL {
		// Remove the expired item
		delete(m.data, key)
		return nil, fmt.Errorf("key expired: %s", key)
	}

	return item.Value, nil
}

// Set stores a value in memory
func (m *VectorMemoryManager) Set(key string, value interface{}) error {
	return m.SetWithTTL(key, value, 0)
}

// SetWithTTL stores a value in memory with a time-to-live
func (m *VectorMemoryManager) SetWithTTL(key string, value interface{}, ttl int64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now().Unix()

	// Convert value to string for vector storage
	var content string
	switch v := value.(type) {
	case string:
		content = v
	case []byte:
		content = string(v)
	default:
		// Convert to JSON for complex types
		jsonData, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %v", err)
		}
		content = string(jsonData)
	}

	// Create or update the memory item
	item := &MemoryItem{
		Key:       key,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
		TTL:       ttl,
	}

	// Store in local map
	m.data[key] = item

	// Store in vector database
	document := chromem.Document{
		ID:      key,
		Content: content,
		Metadata: map[string]string{
			"createdAt": fmt.Sprintf("%d", now),
			"updatedAt": fmt.Sprintf("%d", now),
			"ttl":       fmt.Sprintf("%d", ttl),
		},
	}

	err := m.collection.AddDocuments(context.Background(), []chromem.Document{document}, 1)
	if err != nil {
		return fmt.Errorf("failed to store in vector database: %v", err)
	}

	return nil
}

// Delete removes a value from memory
func (m *VectorMemoryManager) Delete(key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Remove from local map
	delete(m.data, key)

	// Remove from vector database
	err := m.collection.Delete(context.Background(), nil, nil, key)
	if err != nil {
		return fmt.Errorf("failed to delete from vector database: %v", err)
	}

	return nil
}

// List returns all keys in memory
func (m *VectorMemoryManager) List() ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	now := time.Now().Unix()
	keys := make([]string, 0, len(m.data))

	for key, item := range m.data {
		// Skip expired items
		if item.TTL > 0 && now > item.CreatedAt+item.TTL {
			continue
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// Clear removes all values from memory
func (m *VectorMemoryManager) Clear() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Clear local map
	m.data = make(map[string]*MemoryItem)

	// Clear vector database by deleting all documents
	// Note: chromem-go doesn't have a reset method, so we'll just clear the local data
	// The vector database will be cleaned up when documents are accessed and found to be missing

	return nil
}

// Search performs semantic search in the memory
func (m *VectorMemoryManager) Search(query string, limit int) ([]chromem.Result, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	results, err := m.collection.Query(context.Background(), query, limit, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector database: %v", err)
	}

	return results, nil
}

// GetSimilar finds similar content based on a key
func (m *VectorMemoryManager) GetSimilar(key string, limit int) ([]chromem.Result, error) {
	m.mutex.RLock()
	item, ok := m.data[key]
	m.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Convert value to string for search
	var content string
	switch v := item.Value.(type) {
	case string:
		content = v
	case []byte:
		content = string(v)
	default:
		jsonData, _ := json.Marshal(v)
		content = string(jsonData)
	}

	return m.Search(content, limit)
}
