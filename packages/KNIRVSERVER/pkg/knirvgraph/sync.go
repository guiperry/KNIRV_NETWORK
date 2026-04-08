package knirvgraph

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Change struct {
	Type    string                 `json:"type"`
	Data    map[string]interface{} `json:"data"`
	Message string                 `json:"message"`
	Author  string                 `json:"author"`
}

type SyncManager struct {
	graphClient  *Client
	interval     time.Duration
	logger       *zap.Logger
	pendingQueue []Change
	mu           sync.RWMutex
	stopCh       chan struct{}
	running      bool
}

type SyncManagerConfig struct {
	GraphURL string
	Interval time.Duration
}

func NewSyncManager(cfg *SyncManagerConfig, logger *zap.Logger) *SyncManager {
	if cfg == nil {
		cfg = DefaultSyncConfig()
	}

	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &SyncManager{
		graphClient:  NewClient(cfg.GraphURL),
		interval:     cfg.Interval,
		logger:       logger,
		pendingQueue: make([]Change, 0),
		stopCh:       make(chan struct{}),
		running:      false,
	}
}

func DefaultSyncConfig() *SyncManagerConfig {
	return &SyncManagerConfig{
		GraphURL: "http://localhost:7090",
		Interval: 30 * time.Second,
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
		zap.String("graph_url", s.graphClient.baseURL),
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

func (s *SyncManager) QueueChange(change interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := change.(map[string]interface{}); ok {
		s.pendingQueue = append(s.pendingQueue, Change{
			Type:    getString(c, "type"),
			Data:    getMap(c, "data"),
			Message: getString(c, "message"),
			Author:  getString(c, "author"),
		})
		s.logger.Info("Change queued for sync",
			zap.String("type", getString(c, "type")),
			zap.Int("queue_size", len(s.pendingQueue)),
		)
	}
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

func (s *SyncManager) processPendingChanges(ctx context.Context) {
	s.mu.Lock()
	queue := make([]Change, len(s.pendingQueue))
	copy(queue, s.pendingQueue)
	s.mu.Unlock()

	if len(queue) == 0 {
		return
	}

	s.logger.Info("Processing pending changes",
		zap.Int("count", len(queue)),
	)

	var failed []Change

	for _, change := range queue {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := s.commitChange(ctx, change); err != nil {
			s.logger.Warn("Failed to commit change",
				zap.String("type", change.Type),
				zap.Error(err),
			)
			failed = append(failed, change)
		} else {
			s.logger.Info("Successfully committed change",
				zap.String("type", change.Type),
			)
		}
	}

	s.mu.Lock()
	s.pendingQueue = failed
	s.mu.Unlock()
}

func (s *SyncManager) commitChange(ctx context.Context, change Change) error {
	return s.graphClient.CommitNode(ctx, GraphNode{
		NodeID:    change.Type + "_" + time.Now().Format("20060102150405"),
		Type:      change.Type,
		Data:      change.Data,
		Timestamp: time.Now(),
	}, change.Message, change.Author)
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
