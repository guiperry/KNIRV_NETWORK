package storage

import (
	"context"
	"errors"
	"io"
)

var (
	ErrNotFound = errors.New("key not found")
)

type Storage interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Has(key []byte) (bool, error)
	Close() error
	Batch() Batch
	ScanPrefix(prefix []byte) (map[string][]byte, error)
}

type GraphStorage interface {
	Storage
	GetNode(nodeID string) ([]byte, error)
	PutNode(nodeID string, nodeData []byte) error
	DeleteNode(nodeID string) error
	GetEdge(edgeID string) ([]byte, error)
	PutEdge(edgeID string, edgeData []byte) error
	DeleteEdge(edgeID string) error
	GetChildren(nodeID string) ([]string, error)
	PutChildren(nodeID string, children []string) error
	GetParents(nodeID string) ([]string, error)
	PutParents(nodeID string, parents []string) error
	GetHeads() ([]string, error)
	UpdateHeads(heads []string) error
	GetAllNodesWithPrefix(prefix string) (map[string][]byte, error)
	RunGC() error
}

type Batch interface {
	Put(key, value []byte)
	Delete(key []byte)
	Write() error
}

type MaintenanceStorage interface {
	BackupTo(context.Context, io.Writer) error
	RestoreFrom(context.Context, io.Reader) error
	Compact(context.Context) error
}
