package checkpoint

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// QuotaLimit <= 0 means the quota is disabled (unlimited). This used to cap
// Cloudflare/web-derived embeddings on a daily limit; with deterministic
// embeddings there is no per-run API quota, so unlimited is now the default.
func (c *Checkpoint) isUnlimited() bool {
	return c.QuotaLimit <= 0
}

// Checkpoint tracks processing progress and quota usage
type Checkpoint struct {
	mu sync.RWMutex

	// Quota tracking
	QuotaLimit    int  `json:"quota_limit"`
	QuotaUsed     int  `json:"quota_used"`
	QuotaExceeded bool `json:"quota_exceeded"`

	// Progress tracking
	LastProcessedFile   string `json:"last_processed_file"`
	LastProcessedChunk  int32  `json:"last_processed_chunk"`
	LastProcessedWindow int32  `json:"last_processed_window"`

	// Statistics
	RecordsProcessed int64 `json:"records_processed"`
	FramesGenerated  int64 `json:"frames_generated"`
	LastUpdated      int64 `json:"last_updated"`

	// ProcessedKeys is an exact-match set of "file#chunk#window" keys that have
	// already been fully processed. Some sources (e.g. Hugging Face dataset rows)
	// assign each record a random, non-sequential FileName, so ordinal comparison
	// against LastProcessedFile/Chunk/Window cannot be used to decide whether a
	// given record is new work.
	ProcessedKeys map[string]bool `json:"processed_keys,omitempty"`
}

// Manager handles checkpoint operations
type Manager struct {
	checkpointPath string
	checkpoint     *Checkpoint
}

// NewManager creates a new checkpoint manager
func NewManager(outputFile string) (*Manager, error) {
	// Store checkpoint next to output file
	checkpointPath := outputFile + ".checkpoint.json"

	mgr := &Manager{
		checkpointPath: checkpointPath,
		checkpoint: &Checkpoint{
			QuotaLimit:    5000, // Default: 5000 embeddings
			QuotaUsed:     0,
			LastUpdated:   time.Now().Unix(),
			ProcessedKeys: make(map[string]bool),
		},
	}

	// Try to load existing checkpoint
	if err := mgr.Load(); err != nil {
		// No existing checkpoint or error loading - start fresh
		fmt.Printf("📋 Starting fresh checkpoint (quota: %d embeddings)\n", mgr.checkpoint.QuotaLimit)
	} else {
		fmt.Printf("📋 Resumed checkpoint: %d/%d embeddings used, %d records processed\n",
			mgr.checkpoint.QuotaUsed, mgr.checkpoint.QuotaLimit, mgr.checkpoint.RecordsProcessed)
	}

	return mgr, nil
}

// Load reads checkpoint from disk
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.checkpointPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no checkpoint file found")
		}
		return fmt.Errorf("failed to read checkpoint: %w", err)
	}

	m.checkpoint.mu.Lock()
	defer m.checkpoint.mu.Unlock()

	if err := json.Unmarshal(data, m.checkpoint); err != nil {
		return fmt.Errorf("failed to parse checkpoint: %w", err)
	}

	if m.checkpoint.ProcessedKeys == nil {
		m.checkpoint.ProcessedKeys = make(map[string]bool)
	}

	return nil
}

// Save writes checkpoint to disk atomically
func (m *Manager) Save() error {
	m.checkpoint.mu.RLock()
	m.checkpoint.LastUpdated = time.Now().Unix()
	data, err := json.MarshalIndent(m.checkpoint, "", "  ")
	m.checkpoint.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	// Write to temp file then rename for atomicity
	tempPath := m.checkpointPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	if err := os.Rename(tempPath, m.checkpointPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename checkpoint: %w", err)
	}

	return nil
}

// UseQuota checks if quota is available and increments usage. When the quota
// is disabled (QuotaLimit <= 0, i.e. unlimited) it always succeeds.
func (m *Manager) UseQuota(amount int) bool {
	m.checkpoint.mu.Lock()
	defer m.checkpoint.mu.Unlock()

	if m.checkpoint.isUnlimited() {
		m.checkpoint.QuotaUsed += amount
		return true
	}

	if m.checkpoint.QuotaUsed+amount > m.checkpoint.QuotaLimit {
		m.checkpoint.QuotaExceeded = true
		return false
	}

	m.checkpoint.QuotaUsed += amount
	return true
}

// HasQuotaAvailable checks if there's quota remaining. Returns true when the
// quota is disabled (unlimited).
func (m *Manager) HasQuotaAvailable() bool {
	m.checkpoint.mu.RLock()
	defer m.checkpoint.mu.RUnlock()

	if m.checkpoint.isUnlimited() {
		return true
	}

	return m.checkpoint.QuotaUsed < m.checkpoint.QuotaLimit && !m.checkpoint.QuotaExceeded
}

// GetQuotaStatus returns current quota status. When unlimited, limit and
// remaining are reported as 0 and math.MaxInt respectively.
func (m *Manager) GetQuotaStatus() (used, limit int, remaining int) {
	m.checkpoint.mu.RLock()
	defer m.checkpoint.mu.RUnlock()

	used = m.checkpoint.QuotaUsed
	if m.checkpoint.isUnlimited() {
		return used, 0, math.MaxInt
	}

	limit = m.checkpoint.QuotaLimit
	return used, limit, limit - used
}

// UpdateProgress updates the last processed position
func (m *Manager) UpdateProgress(file string, chunkID, windowStart int32) {
	m.checkpoint.mu.Lock()
	defer m.checkpoint.mu.Unlock()

	m.checkpoint.LastProcessedFile = file
	m.checkpoint.LastProcessedChunk = chunkID
	m.checkpoint.LastProcessedWindow = windowStart
	m.checkpoint.LastUpdated = time.Now().Unix()
}

// IncrementStats increments processing statistics
func (m *Manager) IncrementStats(records, frames int64) {
	m.checkpoint.mu.Lock()
	defer m.checkpoint.mu.Unlock()

	m.checkpoint.RecordsProcessed += records
	m.checkpoint.FramesGenerated += frames
	m.checkpoint.LastUpdated = time.Now().Unix()
}

// ShouldSkipRecord checks if a record has already been processed. This is an
// exact-match check (not an ordinal "before the checkpoint" comparison)
// because some sources (e.g. Hugging Face dataset rows) assign each record a
// random, non-sequential FileName - lexicographically comparing such names
// against LastProcessedFile is meaningless and can cause every record in a
// brand-new batch to be misidentified as already processed.
func (m *Manager) ShouldSkipRecord(file string, chunkID, windowStart int32) bool {
	m.checkpoint.mu.RLock()
	defer m.checkpoint.mu.RUnlock()

	return m.checkpoint.ProcessedKeys[recordKey(file, chunkID, windowStart)]
}

// MarkRecordProcessed records that a record has been fully processed so that
// a future resumed run can skip it via ShouldSkipRecord.
func (m *Manager) MarkRecordProcessed(file string, chunkID, windowStart int32) {
	m.checkpoint.mu.Lock()
	defer m.checkpoint.mu.Unlock()

	if m.checkpoint.ProcessedKeys == nil {
		m.checkpoint.ProcessedKeys = make(map[string]bool)
	}
	m.checkpoint.ProcessedKeys[recordKey(file, chunkID, windowStart)] = true
}

func recordKey(file string, chunkID, windowStart int32) string {
	return fmt.Sprintf("%s#%d#%d", file, chunkID, windowStart)
}

// GetCheckpoint returns a copy of the current checkpoint
func (m *Manager) GetCheckpoint() Checkpoint {
	m.checkpoint.mu.RLock()
	defer m.checkpoint.mu.RUnlock()

	return *m.checkpoint
}

// SetQuotaLimit sets the quota limit (useful for testing or overrides)
func (m *Manager) SetQuotaLimit(limit int) {
	m.checkpoint.mu.Lock()
	defer m.checkpoint.mu.Unlock()

	m.checkpoint.QuotaLimit = limit
}

// ResetDailyQuota resets the quota usage if it's a new day
func (m *Manager) ResetDailyQuota() bool {
	m.checkpoint.mu.Lock()
	defer m.checkpoint.mu.Unlock()

	now := time.Now()
	lastUpdate := time.Unix(m.checkpoint.LastUpdated, 0)

	// Check if it's a different day
	if now.Year() != lastUpdate.Year() || now.YearDay() != lastUpdate.YearDay() {
		m.checkpoint.QuotaUsed = 0
		m.checkpoint.QuotaExceeded = false
		m.checkpoint.LastUpdated = now.Unix()
		log.Printf("🌅 New day detected - quota reset to 0/%d", m.checkpoint.QuotaLimit)
		return true
	}

	return false
}

// GetCheckpointPath returns the path to the checkpoint file
func (m *Manager) GetCheckpointPath() string {
	return m.checkpointPath
}

// EnsureDir ensures the checkpoint directory exists
func EnsureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}
