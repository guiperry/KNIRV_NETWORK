package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/knirvcorp/knirvbase/go/pkg/knirvbase"
)

// Storage defines the interface for persistent key-value storage
type Storage interface {
	// Put stores a key-value pair
	Put(key, value []byte) error

	// Get retrieves a value by key
	Get(key []byte) ([]byte, error)

	// Delete removes a key-value pair
	Delete(key []byte) error

	// Has checks if a key exists
	Has(key []byte) (bool, error)

	// Batch creates a new batch for atomic writes
	Batch() Batch

	// NewIterator creates an iterator with optional prefix
	NewIterator(prefix []byte) Iterator

	// Close closes the storage
	Close() error
}

// Batch defines the interface for batch operations
type Batch interface {
	// Put adds a put operation to the batch
	Put(key, value []byte) error

	// Delete adds a delete operation to the batch
	Delete(key []byte) error

	// Write commits all operations atomically
	Write() error

	// Reset clears the batch
	Reset()
}

// Iterator defines the interface for iterating over key-value pairs
type Iterator interface {
	// Next moves to the next key-value pair
	Next() bool

	// Key returns the current key
	Key() []byte

	// Value returns the current value
	Value() []byte

	// Error returns any iteration error
	Error() error

	// Release releases the iterator
	Release()
}

// KNIRVBASEStorage implements Storage using KNIRVBASE
type KNIRVBASEStorage struct {
	db         *knirvbase.DB
	collection knirvbase.Collection
	ctx        context.Context
	mu         sync.RWMutex
}

// NewKNIRVBASEStorage creates a new KNIRVBASE storage instance
func NewKNIRVBASEStorage(dataDir string) (*KNIRVBASEStorage, error) {
	ctx := context.Background()

	// Create KNIRVBASE instance
	db, err := knirvbase.New(ctx, knirvbase.Options{
		DataDir:            dataDir,
		DistributedEnabled: false, // Single-node for KNIRVCHAIN
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create KNIRVBASE: %w", err)
	}

	// Create/get the "blockchain" collection
	collection := db.Collection("blockchain")

	return &KNIRVBASEStorage{
		db:         db,
		collection: collection,
		ctx:        ctx,
	}, nil
}

// keyToID converts a byte key to a string ID for KNIRVBASE
func keyToID(key []byte) string {
	return string(key)
}

// Put stores a key-value pair
func (s *KNIRVBASEStorage) Put(key, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store value as raw bytes (will be JSON encoded by KNIRVBASE)
	doc := map[string]interface{}{
		"id":    keyToID(key),
		"key":   string(key),
		"value": string(value), // Store as string to avoid base64 encoding
	}

	// Check if exists and update or insert
	existing, err := s.collection.Find(keyToID(key))
	if err == nil && existing != nil {
		// Update existing
		_, err = s.collection.Update(keyToID(key), map[string]interface{}{
			"value": string(value),
		})
		return err
	}

	// Insert new
	_, err = s.collection.Insert(s.ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to insert key: %w", err)
	}
	return nil
}

// Get retrieves a value by key
func (s *KNIRVBASEStorage) Get(key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, err := s.collection.Find(keyToID(key))
	if err != nil || doc == nil {
		return nil, fmt.Errorf("key not found")
	}

	// Extract value
	valueInterface, ok := doc["value"]
	if !ok {
		return nil, fmt.Errorf("value field not found")
	}

	// Convert to string and then to bytes
	if str, ok := valueInterface.(string); ok {
		return []byte(str), nil
	}

	// Fallback: try to JSON marshal if it's something else
	data, err := json.Marshal(valueInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value: %w", err)
	}
	return data, nil
}

// Delete removes a key-value pair
func (s *KNIRVBASEStorage) Delete(key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.collection.Delete(keyToID(key))
	return err
}

// Has checks if a key exists
func (s *KNIRVBASEStorage) Has(key []byte) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, err := s.collection.Find(keyToID(key))
	if err != nil || doc == nil {
		return false, nil
	}
	return true, nil
}

// Batch creates a new batch for atomic writes
func (s *KNIRVBASEStorage) Batch() Batch {
	return &KNIRVBASEBatch{
		storage:    s,
		operations: make([]batchOp, 0),
	}
}

// NewIterator creates an iterator with optional prefix
func (s *KNIRVBASEStorage) NewIterator(prefix []byte) Iterator {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get all documents
	docs, err := s.collection.FindAll()
	if err != nil {
		return &KNIRVBASEIterator{err: err}
	}

	// Filter by prefix if specified
	var filtered []map[string]interface{}
	prefixStr := string(prefix)
	for _, doc := range docs {
		if keyVal, ok := doc["key"].(string); ok {
			if prefix == nil || strings.HasPrefix(keyVal, prefixStr) {
				filtered = append(filtered, doc)
			}
		}
	}

	return &KNIRVBASEIterator{
		docs:  filtered,
		index: -1,
	}
}

// Close closes the storage
func (s *KNIRVBASEStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Shutdown()
}

// batchOp represents a batch operation
type batchOp struct {
	opType string // "put" or "delete"
	key    []byte
	value  []byte
}

// KNIRVBASEBatch implements Batch using KNIRVBASE
type KNIRVBASEBatch struct {
	storage    *KNIRVBASEStorage
	operations []batchOp
}

// Put adds a put operation to the batch
func (b *KNIRVBASEBatch) Put(key, value []byte) error {
	b.operations = append(b.operations, batchOp{
		opType: "put",
		key:    key,
		value:  value,
	})
	return nil
}

// Delete adds a delete operation to the batch
func (b *KNIRVBASEBatch) Delete(key []byte) error {
	b.operations = append(b.operations, batchOp{
		opType: "delete",
		key:    key,
	})
	return nil
}

// Write commits all operations atomically
func (b *KNIRVBASEBatch) Write() error {
	// Note: KNIRVBASE doesn't have native batch operations,
	// so we execute them sequentially. For true atomicity,
	// we'd need to add transaction support to KNIRVBASE.
	for _, op := range b.operations {
		switch op.opType {
		case "put":
			if err := b.storage.Put(op.key, op.value); err != nil {
				return fmt.Errorf("batch put failed: %w", err)
			}
		case "delete":
			if err := b.storage.Delete(op.key); err != nil {
				return fmt.Errorf("batch delete failed: %w", err)
			}
		}
	}
	return nil
}

// Reset clears the batch
func (b *KNIRVBASEBatch) Reset() {
	b.operations = nil
}

// KNIRVBASEIterator implements Iterator using KNIRVBASE
type KNIRVBASEIterator struct {
	docs  []map[string]interface{}
	index int
	err   error
}

// Next moves to the next key-value pair
func (i *KNIRVBASEIterator) Next() bool {
	if i.err != nil {
		return false
	}
	i.index++
	return i.index < len(i.docs)
}

// Key returns the current key
func (i *KNIRVBASEIterator) Key() []byte {
	if i.index < 0 || i.index >= len(i.docs) {
		return nil
	}
	if key, ok := i.docs[i.index]["key"].(string); ok {
		return []byte(key)
	}
	return nil
}

// Value returns the current value
func (i *KNIRVBASEIterator) Value() []byte {
	if i.index < 0 || i.index >= len(i.docs) {
		return nil
	}
	valueInterface := i.docs[i.index]["value"]

	// Convert string to bytes
	if str, ok := valueInterface.(string); ok {
		return []byte(str)
	}

	// Fallback: try to JSON marshal
	data, _ := json.Marshal(valueInterface)
	return data
}

// Error returns any iteration error
func (i *KNIRVBASEIterator) Error() error {
	return i.err
}

// Release releases the iterator
func (i *KNIRVBASEIterator) Release() {
	i.docs = nil
}
