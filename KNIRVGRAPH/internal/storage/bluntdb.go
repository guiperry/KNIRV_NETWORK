package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/dgraph-io/badger/v3"
)

type BluntDBStorage struct {
	db *badger.DB
}

func NewBluntDBStorage(path string) (*BluntDBStorage, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable badger logging

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open BluntDB: %w", err)
	}

	return &BluntDBStorage{db: db}, nil
}

func (s *BluntDBStorage) Put(key, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (s *BluntDBStorage) Get(key []byte) ([]byte, error) {
	var value []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			value = append([]byte{}, val...)
			return nil
		})
	})

	return value, err
}

func (s *BluntDBStorage) Delete(key []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (s *BluntDBStorage) Has(key []byte) (bool, error) {
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})

	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *BluntDBStorage) Close() error {
	return s.db.Close()
}

func (s *BluntDBStorage) Batch() Batch {
	return &BluntDBBatch{
		db:      s.db,
		entries: make(map[string][]byte),
		deletes: make(map[string]bool),
	}
}

// Graph-specific storage operations
func (s *BluntDBStorage) GetNode(nodeID string) ([]byte, error) {
	key := fmt.Sprintf("node_%s", nodeID)
	return s.Get([]byte(key))
}

func (s *BluntDBStorage) PutNode(nodeID string, nodeData []byte) error {
	key := fmt.Sprintf("node_%s", nodeID)
	return s.Put([]byte(key), nodeData)
}

func (s *BluntDBStorage) GetEdge(edgeID string) ([]byte, error) {
	key := fmt.Sprintf("edge_%s", edgeID)
	return s.Get([]byte(key))
}

func (s *BluntDBStorage) PutEdge(edgeID string, edgeData []byte) error {
	key := fmt.Sprintf("edge_%s", edgeID)
	return s.Put([]byte(key), edgeData)
}

func (s *BluntDBStorage) GetChildren(nodeID string) ([]string, error) {
	key := fmt.Sprintf("children_%s", nodeID)
	data, err := s.Get([]byte(key))
	if err != nil {
		if err == ErrNotFound {
			return []string{}, nil
		}
		return nil, err
	}

	var children []string
	if err := json.Unmarshal(data, &children); err != nil {
		return nil, err
	}
	return children, nil
}

func (s *BluntDBStorage) PutChildren(nodeID string, children []string) error {
	key := fmt.Sprintf("children_%s", nodeID)
	data, err := json.Marshal(children)
	if err != nil {
		return err
	}
	return s.Put([]byte(key), data)
}

func (s *BluntDBStorage) GetParents(nodeID string) ([]string, error) {
	key := fmt.Sprintf("parents_%s", nodeID)
	data, err := s.Get([]byte(key))
	if err != nil {
		if err == ErrNotFound {
			return []string{}, nil
		}
		return nil, err
	}

	var parents []string
	if err := json.Unmarshal(data, &parents); err != nil {
		return nil, err
	}
	return parents, nil
}

func (s *BluntDBStorage) PutParents(nodeID string, parents []string) error {
	key := fmt.Sprintf("parents_%s", nodeID)
	data, err := json.Marshal(parents)
	if err != nil {
		return err
	}
	return s.Put([]byte(key), data)
}

func (s *BluntDBStorage) GetHeads() ([]string, error) {
	data, err := s.Get([]byte("graph_heads"))
	if err != nil {
		if err == ErrNotFound {
			return []string{}, nil
		}
		return nil, err
	}

	var heads []string
	if err := json.Unmarshal(data, &heads); err != nil {
		return nil, err
	}
	return heads, nil
}

func (s *BluntDBStorage) UpdateHeads(heads []string) error {
	data, err := json.Marshal(heads)
	if err != nil {
		return err
	}
	return s.Put([]byte("graph_heads"), data)
}

func (s *BluntDBStorage) GetAllNodesWithPrefix(prefix string) (map[string][]byte, error) {
	nodes := make(map[string][]byte)

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := string(item.Key())

			err := item.Value(func(val []byte) error {
				nodes[key] = append([]byte{}, val...)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return nodes, err
}

type BluntDBBatch struct {
	db      *badger.DB
	entries map[string][]byte
	deletes map[string]bool
}

func (b *BluntDBBatch) Put(key, value []byte) {
	b.entries[string(key)] = value
	delete(b.deletes, string(key))
}

func (b *BluntDBBatch) Delete(key []byte) {
	b.deletes[string(key)] = true
	delete(b.entries, string(key))
}

func (b *BluntDBBatch) Write() error {
	return b.db.Update(func(txn *badger.Txn) error {
		// Apply deletions
		for key := range b.deletes {
			if err := txn.Delete([]byte(key)); err != nil {
				return err
			}
		}

		// Apply puts
		for key, value := range b.entries {
			if err := txn.Set([]byte(key), value); err != nil {
				return err
			}
		}

		return nil
	})
}

// Cleanup and maintenance operations
func (s *BluntDBStorage) RunGC() error {
	return s.db.RunValueLogGC(0.5)
}

func (s *BluntDBStorage) Backup(path string) error {
	// Validate path to prevent directory traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path: directory traversal not allowed")
	}

	// Create backup file
	file, err := os.Create(path) // #nosec G304 - path is validated above
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing backup file: %v", closeErr)
		}
	}()

	_, err = s.db.Backup(file, 0)
	return err
}
