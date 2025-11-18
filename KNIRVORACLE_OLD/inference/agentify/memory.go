// memory.go
package agentify

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
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
	m.mutex.Lock()
	defer m.mutex.Unlock()

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

// PersistentMemoryManager implements MemoryManager using chromem-go
// This is a placeholder implementation until chromem-go is integrated
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
