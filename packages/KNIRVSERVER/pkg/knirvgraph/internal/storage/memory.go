package storage

import (
	"strings"
	"sync"
)

// MemoryStorage implements GraphStorage interface using in-memory maps
type MemoryStorage struct {
	data     map[string][]byte
	nodes    map[string][]byte
	edges    map[string][]byte
	children map[string][]string
	parents  map[string][]string
	heads    []string
	mutex    sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage() (GraphStorage, error) {
	return &MemoryStorage{
		data:     make(map[string][]byte),
		nodes:    make(map[string][]byte),
		edges:    make(map[string][]byte),
		children: make(map[string][]string),
		parents:  make(map[string][]string),
		heads:    make([]string, 0),
	}, nil
}

// Put stores a key-value pair
func (m *MemoryStorage) Put(key, value []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data[string(key)] = value
	return nil
}

// Get retrieves a value by key
func (m *MemoryStorage) Get(key []byte) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	value, exists := m.data[string(key)]
	if !exists {
		return nil, ErrNotFound
	}
	return value, nil
}

// Delete removes a key-value pair
func (m *MemoryStorage) Delete(key []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.data, string(key))
	return nil
}

// Has checks if a key exists
func (m *MemoryStorage) Has(key []byte) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	_, exists := m.data[string(key)]
	return exists, nil
}

// Close closes the storage (no-op for memory storage)
func (m *MemoryStorage) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data = make(map[string][]byte)
	m.nodes = make(map[string][]byte)
	m.edges = make(map[string][]byte)
	m.children = make(map[string][]string)
	m.parents = make(map[string][]string)
	m.heads = make([]string, 0)
	return nil
}

// Batch returns a new batch for atomic operations
func (m *MemoryStorage) Batch() Batch {
	return &MemoryBatch{
		storage:    m,
		operations: make([]batchOp, 0),
	}
}

// GetNode retrieves node data
func (m *MemoryStorage) GetNode(nodeID string) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	value, exists := m.nodes[nodeID]
	if !exists {
		return nil, ErrNotFound
	}
	return value, nil
}

// PutNode stores node data
func (m *MemoryStorage) PutNode(nodeID string, nodeData []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.nodes[nodeID] = nodeData
	return nil
}

// GetEdge retrieves edge data
func (m *MemoryStorage) GetEdge(edgeID string) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	value, exists := m.edges[edgeID]
	if !exists {
		return nil, ErrNotFound
	}
	return value, nil
}

// PutEdge stores edge data
func (m *MemoryStorage) PutEdge(edgeID string, edgeData []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.edges[edgeID] = edgeData
	return nil
}

// GetChildren retrieves children of a node
func (m *MemoryStorage) GetChildren(nodeID string) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	children, exists := m.children[nodeID]
	if !exists {
		return []string{}, nil
	}
	return children, nil
}

// PutChildren stores children of a node
func (m *MemoryStorage) PutChildren(nodeID string, children []string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.children[nodeID] = children
	return nil
}

// GetParents retrieves parents of a node
func (m *MemoryStorage) GetParents(nodeID string) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	parents, exists := m.parents[nodeID]
	if !exists {
		return []string{}, nil
	}
	return parents, nil
}

// PutParents stores parents of a node
func (m *MemoryStorage) PutParents(nodeID string, parents []string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.parents[nodeID] = parents
	return nil
}

// GetHeads retrieves head nodes
func (m *MemoryStorage) GetHeads() ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.heads, nil
}

// UpdateHeads updates head nodes
func (m *MemoryStorage) UpdateHeads(heads []string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.heads = heads
	return nil
}

// GetAllNodesWithPrefix retrieves all nodes with a given prefix
func (m *MemoryStorage) GetAllNodesWithPrefix(prefix string) (map[string][]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	result := make(map[string][]byte)
	for nodeID, nodeData := range m.nodes {
		if strings.HasPrefix(nodeID, prefix) {
			result[nodeID] = nodeData
		}
	}
	return result, nil
}

// RunGC runs garbage collection (no-op for memory storage)
func (m *MemoryStorage) RunGC() error {
	return nil
}

// MemoryBatch implements Batch interface for memory storage
type MemoryBatch struct {
	storage    *MemoryStorage
	operations []batchOp
}

type batchOp struct {
	opType string // "put" or "delete"
	key    string
	value  []byte
}

// Put adds a put operation to the batch
func (b *MemoryBatch) Put(key, value []byte) {
	b.operations = append(b.operations, batchOp{
		opType: "put",
		key:    string(key),
		value:  value,
	})
}

// Delete adds a delete operation to the batch
func (b *MemoryBatch) Delete(key []byte) {
	b.operations = append(b.operations, batchOp{
		opType: "delete",
		key:    string(key),
	})
}

// Write executes all operations in the batch atomically
func (b *MemoryBatch) Write() error {
	b.storage.mutex.Lock()
	defer b.storage.mutex.Unlock()

	for _, op := range b.operations {
		switch op.opType {
		case "put":
			b.storage.data[op.key] = op.value
		case "delete":
			delete(b.storage.data, op.key)
		}
	}

	b.operations = b.operations[:0] // Clear operations
	return nil
}
