package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type PendingChange struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // "error_node", "context_record", etc.
	Data      map[string]interface{} `json:"data"`
	Message   string                 `json:"message"`
	Author    string                 `json:"author"`
	CreatedAt time.Time              `json:"created_at"`
	Retries   int                    `json:"retries"`
}

type SyncManager struct {
	graphClient  *http.Client
	embeddedURL  string
	interval     time.Duration
	logger       *zap.Logger
	pendingQueue []PendingChange
	mu           sync.RWMutex
	stopCh       chan struct{}
	running      bool
}

type SyncManagerConfig struct {
	EmbeddedURL  string
	Interval     time.Duration
	StartTimeout time.Duration
}

func NewSyncManager(cfg *SyncManagerConfig, logger *zap.Logger) *SyncManager {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &SyncManager{
		graphClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		embeddedURL:  cfg.EmbeddedURL,
		interval:     cfg.Interval,
		logger:       logger,
		pendingQueue: make([]PendingChange, 0),
		stopCh:       make(chan struct{}),
		running:      false,
	}
}

func DefaultConfig() *SyncManagerConfig {
	return &SyncManagerConfig{
		EmbeddedURL:  "http://localhost:7090",
		Interval:     30 * time.Second,
		StartTimeout: 10 * time.Second,
	}
}

func (s *SyncManager) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Starting SyncManager",
		zap.String("embedded_url", s.embeddedURL),
		zap.Duration("interval", s.interval),
	)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("SyncManager stopping due to context cancellation")
			return
		case <-s.stopCh:
			s.logger.Info("SyncManager stopping")
			return
		case <-ticker.C:
			s.processPendingChanges(ctx)
		}
	}
}

func (s *SyncManager) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopCh)
	s.running = false
	s.logger.Info("SyncManager stopped")
}

func (s *SyncManager) QueueChange(change PendingChange) {
	s.mu.Lock()
	defer s.mu.Unlock()

	change.ID = fmt.Sprintf("change_%d", time.Now().UnixNano())
	change.CreatedAt = time.Now()
	change.Retries = 0

	s.pendingQueue = append(s.pendingQueue, change)
	s.logger.Info("Change queued for sync",
		zap.String("id", change.ID),
		zap.String("type", change.Type),
		zap.Int("queue_size", len(s.pendingQueue)),
	)
}

func (s *SyncManager) processPendingChanges(ctx context.Context) {
	s.mu.Lock()
	queue := make([]PendingChange, len(s.pendingQueue))
	copy(queue, s.pendingQueue)
	s.mu.Unlock()

	if len(queue) == 0 {
		return
	}

	s.logger.Info("Processing pending changes",
		zap.Int("count", len(queue)),
	)

	var failed []PendingChange

	for _, change := range queue {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := s.commitChange(ctx, change); err != nil {
			s.logger.Warn("Failed to commit change",
				zap.String("id", change.ID),
				zap.Error(err),
			)
			change.Retries++
			if change.Retries < 3 {
				failed = append(failed, change)
			} else {
				s.logger.Error("Change exceeded max retries, dropping",
					zap.String("id", change.ID),
					zap.Int("retries", change.Retries),
				)
			}
		} else {
			s.logger.Info("Successfully committed change",
				zap.String("id", change.ID),
				zap.String("type", change.Type),
			)
		}
	}

	s.mu.Lock()
	s.pendingQueue = failed
	s.mu.Unlock()
}

func (s *SyncManager) commitChange(ctx context.Context, change PendingChange) error {
	payload := map[string]interface{}{
		"node": map[string]interface{}{
			"node_id":    change.ID,
			"type":       change.Type,
			"data":       change.Data,
			"timestamp":  change.CreatedAt,
			"commit_msg": change.Message,
		},
		"message": change.Message,
		"author":  change.Author,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.embeddedURL+"/commit", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.graphClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *SyncManager) GetQueueSize() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.pendingQueue)
}

func (s *SyncManager) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}
