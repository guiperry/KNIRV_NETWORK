package knirvoracle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type RollupStatus string

const (
	RollupStatusSubmitted RollupStatus = "submitted"
	RollupStatusFinalized RollupStatus = "finalized"
	RollupStatusDisputed  RollupStatus = "disputed"
)

type RollupRecord struct {
	ID          string                 `json:"id"`
	BatchRoot   string                 `json:"batch_root"`
	ChainID     string                 `json:"chain_id"`
	StartHeight uint64                 `json:"start_height"`
	EndHeight   uint64                 `json:"end_height"`
	BlockCount  int                    `json:"block_count"`
	TxCount     int                    `json:"tx_count"`
	Status      RollupStatus           `json:"status"`
	SubmittedAt time.Time              `json:"submitted_at"`
	FinalizedAt *time.Time             `json:"finalized_at,omitempty"`
	DisputedAt  *time.Time             `json:"disputed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Dispute     string                 `json:"dispute,omitempty"`
}

type RollupCache struct {
	mu      sync.RWMutex
	rollups map[string]*RollupRecord
	path    string
}

func NewRollupCache(path string) (*RollupCache, error) {
	cache := &RollupCache{
		rollups: make(map[string]*RollupRecord),
		path:    path,
	}
	if err := cache.Load(); err != nil {
		return nil, err
	}
	return cache, nil
}

func (c *RollupCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.path == "" {
		return nil
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read oracle rollup state: %w", err)
	}

	var records []*RollupRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("failed to decode oracle rollup state: %w", err)
	}

	c.rollups = make(map[string]*RollupRecord, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		c.rollups[record.ID] = record
	}

	return nil
}

func (c *RollupCache) Upsert(record *RollupRecord) error {
	if record == nil {
		return fmt.Errorf("rollup record is required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.rollups[record.ID] = record
	return c.persistLocked()
}

func (c *RollupCache) Get(id string) (*RollupRecord, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	record, ok := c.rollups[id]
	return record, ok
}

func (c *RollupCache) Finalize(id string, finalizedAt time.Time) (*RollupRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	record, ok := c.rollups[id]
	if !ok {
		return nil, fmt.Errorf("rollup not found: %s", id)
	}

	ts := finalizedAt.UTC()
	record.Status = RollupStatusFinalized
	record.FinalizedAt = &ts
	return record, c.persistLocked()
}

func (c *RollupCache) Dispute(id string, reason string, disputedAt time.Time) (*RollupRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	record, ok := c.rollups[id]
	if !ok {
		return nil, fmt.Errorf("rollup not found: %s", id)
	}

	ts := disputedAt.UTC()
	record.Status = RollupStatusDisputed
	record.Dispute = reason
	record.DisputedAt = &ts
	return record, c.persistLocked()
}

func (c *RollupCache) persistLocked() error {
	if c.path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return fmt.Errorf("failed to create oracle rollup directory: %w", err)
	}

	records := make([]*RollupRecord, 0, len(c.rollups))
	for _, record := range c.rollups {
		records = append(records, record)
	}

	payload, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode oracle rollup state: %w", err)
	}

	tempPath := c.path + ".tmp"
	if err := os.WriteFile(tempPath, payload, 0644); err != nil {
		return fmt.Errorf("failed to write oracle rollup temp file: %w", err)
	}

	if err := os.Rename(tempPath, c.path); err != nil {
		return fmt.Errorf("failed to move oracle rollup state into place: %w", err)
	}

	return nil
}
