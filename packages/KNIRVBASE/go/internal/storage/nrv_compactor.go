package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/knirvcorp/knirvbase/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

const compactionThreshold = 0.20

type Compactor struct {
	datasetPath string
	keyPair     *pqc.PQCKeyPair
	once        sync.Once
	running     bool
	mu          sync.Mutex
	stopCh      chan struct{}
}

func NewCompactor(path string, keyPair *pqc.PQCKeyPair) *Compactor {
	return &Compactor{
		datasetPath: path,
		keyPair:     keyPair,
		stopCh:      make(chan struct{}),
	}
}

func (c *Compactor) MaybeCompact(registry *nrv.Registry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return
	}

	if registry.FrameCount == 0 {
		return
	}

	deadFrames := registry.TombstoneCount + registry.GlobalMetrics.InvalidFrameCount
	ratio := float64(deadFrames) / float64(registry.FrameCount)
	if ratio >= compactionThreshold {
		c.running = true
		go func() {
			defer func() {
				c.mu.Lock()
				c.running = false
				c.mu.Unlock()
			}()
			if err := c.compact(); err != nil {
				return
			}
		}()
	}
}

func (c *Compactor) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return
	}
	c.running = true

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
			case <-c.stopCh:
				return
			}
		}
	}()
}

func (c *Compactor) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.running = false
	close(c.stopCh)
}

func (c *Compactor) compact() error {
	reader, err := NewNRVReader(c.datasetPath)
	if err != nil {
		return fmt.Errorf("nrv: open reader for compaction: %w", err)
	}
	defer reader.Close()

	tmpPath := c.datasetPath + ".tmp"

	writer, err := NewNRVWriter(tmpPath, c.keyPair)
	if err != nil {
		return fmt.Errorf("nrv: create writer for compaction: %w", err)
	}

	for _, entry := range reader.registry.Frames {
		if entry.Tombstone != nil {
			continue
		}
		if entry.Z3.Status == "INVALID" {
			continue
		}

		brackets := reader.decodeBrackets(entry)
		if len(brackets) == 0 {
			continue
		}

		buf := make([]byte, len(brackets)*nrv.BracketSize)
		for i, b := range brackets {
			encoded := nrv.EncodeBracket(b)
			copy(buf[i*nrv.BracketSize:], encoded[:])
		}

		if err := writer.AppendFrame(
			entry.ID,
			buf,
			entry.BracketIndex,
			entry.Thermo,
			entry.Linguistic,
		); err != nil {
			writer.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("nrv: append frame during compaction: %w", err)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	writer.registry.GlobalMetrics.CompactedAt = &now

	if err := writer.saveRegistry(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("nrv: save registry after compaction: %w", err)
	}

	if err := writer.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("nrv: close writer after compaction: %w", err)
	}

	if err := os.Rename(tmpPath, c.datasetPath); err != nil {
		return fmt.Errorf("nrv: rename compacted file: %w", err)
	}

	walPath := c.datasetPath + ".wal"
	os.Remove(walPath)

	return nil
}

func (c *Compactor) GetDatasetPath() string {
	return c.datasetPath
}

func init() {
	_ = filepath.Join
}
