// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package knowledge_base

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// KnowledgeBase represents a GraphRAG-powered knowledge base
type KnowledgeBase struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Version         string                 `json:"version"`
	Author          string                 `json:"author"`
	Type            string                 `json:"type"` // GraphRAG, Vector, Semantic, Hybrid
	Status          string                 `json:"status"`
	FilePath        string                 `json:"file_path"`
	FileSize        int64                  `json:"file_size"`
	FileHash        string                 `json:"file_hash"`
	Capabilities    []string               `json:"capabilities"`
	Configuration   map[string]interface{} `json:"configuration"`
	Metadata        map[string]interface{} `json:"metadata"`
	Tags            []string               `json:"tags"`
	EmbeddingModel  string                 `json:"embedding_model"`
	GraphIndex      string                 `json:"graph_index"`
	UploadedAt      time.Time              `json:"uploaded_at"`
	DeployedAt      *time.Time             `json:"deployed_at,omitempty"`
	LastModified    time.Time              `json:"last_modified"`
	LastActivity    *time.Time             `json:"last_activity,omitempty"`
	UploadedBy      string                 `json:"uploaded_by"`
	DeployedBy      *string                `json:"deployed_by,omitempty"`
	RuntimeInstance *RuntimeInstance       `json:"runtime_instance,omitempty"`
}

type RuntimeInstance struct {
	InstanceID      string            `json:"instance_id"`
	ProcessID       *int              `json:"process_id,omitempty"`
	StartedAt       time.Time         `json:"started_at"`
	Status          string            `json:"status"` // starting, running, stopping, stopped, crashed
	ResourceUsage   *ResourceUsage    `json:"resource_usage,omitempty"`
	Environment     map[string]string `json:"environment"`
	HealthCheckURL  string            `json:"health_check_url,omitempty"`
	HealthStatus    string            `json:"health_status,omitempty"`
	LastHealthCheck *time.Time        `json:"last_health_check,omitempty"`
	RestartCount    int               `json:"restart_count"`
	UptimeSeconds   int64             `json:"uptime_seconds"`
}

type ResourceUsage struct {
	CPUPercent float64   `json:"cpu_percent"`
	MemoryMB   float64   `json:"memory_mb"`
	Timestamp  time.Time `json:"timestamp"`
}

type Summary struct {
	TotalModels    int `json:"total_models"`
	RunningModels  int `json:"running_models"`
	StoppedModels  int `json:"stopped_models"`
	ErrorModels    int `json:"error_models"`
	DeployedModels int `json:"deployed_models"`
	UploadedModels int `json:"uploaded_models"`
}

// GraphRAGQuery represents a query to the GraphRAG engine
type GraphRAGQuery struct {
	Query  string                 `json:"query"`
	Mode   string                 `json:"mode"` // local, global, hybrid
	Limit  int                    `json:"limit"`
	Params map[string]interface{} `json:"params"`
}

// GraphRAGResult represents the result of a GraphRAG query
type GraphRAGResult struct {
	ID        string      `json:"id"`
	Query     string      `json:"query"`
	Mode      string      `json:"mode"` // local, global, hybrid
	Nodes     []GraphNode `json:"nodes"`
	Edges     []GraphEdge `json:"edges"`
	Chunks    []TextChunk `json:"chunks"`
	Summary   string      `json:"summary"`
	Score     float64     `json:"score"`
	Timestamp time.Time   `json:"timestamp"`
}

type GraphNode struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"`
	Data  map[string]interface{} `json:"data"`
	Score float64                `json:"score"`
}

type GraphEdge struct {
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       string                 `json:"type"`
	Weight     float64                `json:"weight"`
	Attributes map[string]interface{} `json:"attributes"`
}

type TextChunk struct {
	ID        string  `json:"id"`
	Content   string  `json:"content"`
	Relevance float64 `json:"relevance"`
	Source    string  `json:"source,omitempty"`
}

type IndexStatus struct {
	KBId         string    `json:"kb_id"`
	Status       string    `json:"status"` // building, ready, failed
	Progress     float64   `json:"progress"`
	NodesCount   int       `json:"nodes_count"`
	EdgesCount   int       `json:"edges_count"`
	ChunksCount  int       `json:"chunks_count"`
	LastUpdated  time.Time `json:"last_updated"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// KnowledgeBaseService manages knowledge bases
type KnowledgeBaseService struct {
	db       map[string]*KnowledgeBase // In-memory for now; replace with real DB
	graphRAG GraphRAGInterface         // FFI interface to graphrag-rs
	mu       sync.RWMutex
	indexes  map[string]*IndexStatus
}

// GraphRAGInterface defines the FFI boundary with the graphrag-rs library
type GraphRAGInterface interface {
	// Query executes a query against the GraphRAG index
	Query(ctx context.Context, kbID string, query *GraphRAGQuery) (*GraphRAGResult, error)

	// BuildIndex creates or rebuilds the GraphRAG index
	BuildIndex(ctx context.Context, kbID string, strategy string) (*IndexStatus, error)

	// GetIndexStatus returns the current index state
	GetIndexStatus(ctx context.Context, kbID string) (*IndexStatus, error)

	// Close cleans up FFI resources
	Close() error
}

// NewKnowledgeBaseService creates a new knowledge base service
func NewKnowledgeBaseService(graphRAG GraphRAGInterface) *KnowledgeBaseService {
	return &KnowledgeBaseService{
		db:       make(map[string]*KnowledgeBase),
		graphRAG: graphRAG,
		indexes:  make(map[string]*IndexStatus),
	}
}

// ListKnowledgeBases lists all knowledge bases with optional filtering
func (s *KnowledgeBaseService) ListKnowledgeBases(ctx context.Context, status, author, search string) ([]*KnowledgeBase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*KnowledgeBase
	for _, kb := range s.db {
		if status != "" && kb.Status != status {
			continue
		}
		if author != "" && kb.Author != author {
			continue
		}
		// Simple substring search
		if search != "" && kb.Name != search && kb.Description != search {
			continue
		}
		result = append(result, kb)
	}

	return result, nil
}

// GetKnowledgeBase retrieves a specific knowledge base
func (s *KnowledgeBaseService) GetKnowledgeBase(ctx context.Context, kbID string) (*KnowledgeBase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	kb, ok := s.db[kbID]
	if !ok {
		return nil, fmt.Errorf("knowledge base not found: %s", kbID)
	}

	return kb, nil
}

// CreateKnowledgeBase creates a new knowledge base
func (s *KnowledgeBaseService) CreateKnowledgeBase(ctx context.Context, kb *KnowledgeBase) (*KnowledgeBase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if kb.ID == "" {
		return nil, fmt.Errorf("knowledge base ID is required")
	}

	if _, exists := s.db[kb.ID]; exists {
		return nil, fmt.Errorf("knowledge base already exists: %s", kb.ID)
	}

	s.db[kb.ID] = kb
	return kb, nil
}

// UpdateKnowledgeBase updates an existing knowledge base
func (s *KnowledgeBaseService) UpdateKnowledgeBase(ctx context.Context, kbID string, updates map[string]interface{}) (*KnowledgeBase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	kb, ok := s.db[kbID]
	if !ok {
		return nil, fmt.Errorf("knowledge base not found: %s", kbID)
	}

	// Apply updates (simple map-based patching)
	if name, ok := updates["name"].(string); ok {
		kb.Name = name
	}
	if desc, ok := updates["description"].(string); ok {
		kb.Description = desc
	}
	if status, ok := updates["status"].(string); ok {
		kb.Status = status
	}

	kb.LastModified = time.Now()
	return kb, nil
}

// DeleteKnowledgeBase deletes a knowledge base
func (s *KnowledgeBaseService) DeleteKnowledgeBase(ctx context.Context, kbID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.db[kbID]; !ok {
		return fmt.Errorf("knowledge base not found: %s", kbID)
	}

	delete(s.db, kbID)
	delete(s.indexes, kbID)
	return nil
}

// DeployKnowledgeBase marks a knowledge base as deployed
func (s *KnowledgeBaseService) DeployKnowledgeBase(ctx context.Context, kbID string) (*KnowledgeBase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	kb, ok := s.db[kbID]
	if !ok {
		return nil, fmt.Errorf("knowledge base not found: %s", kbID)
	}

	now := time.Now()
	kb.Status = "deployed"
	kb.DeployedAt = &now
	kb.LastModified = now
	kb.LastActivity = &now

	return kb, nil
}

// QueryGraphRAG executes a query against the knowledge base via graphrag-rs
func (s *KnowledgeBaseService) QueryGraphRAG(ctx context.Context, kbID string, query *GraphRAGQuery) (*GraphRAGResult, error) {
	s.mu.RLock()
	kb, ok := s.db[kbID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("knowledge base not found: %s", kbID)
	}

	if kb.Status != "deployed" && kb.Status != "running" {
		return nil, fmt.Errorf("knowledge base is not deployed: %s (status: %s)", kbID, kb.Status)
	}

	if s.graphRAG == nil {
		return nil, fmt.Errorf("graphrag-rs interface not available")
	}

	result, err := s.graphRAG.Query(ctx, kbID, query)
	if err != nil {
		return nil, fmt.Errorf("graphrag query failed: %w", err)
	}

	// Update last activity
	s.mu.Lock()
	now := time.Now()
	kb.LastActivity = &now
	s.mu.Unlock()

	return result, nil
}

// IndexKnowledgeBase builds or rebuilds the GraphRAG index
func (s *KnowledgeBaseService) IndexKnowledgeBase(ctx context.Context, kbID string, strategy string, async bool) (*IndexStatus, error) {
	s.mu.RLock()
	_, ok := s.db[kbID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("knowledge base not found: %s", kbID)
	}

	if s.graphRAG == nil {
		return nil, fmt.Errorf("graphrag-rs interface not available")
	}

	if async {
		// Launch indexing in background
		go func() {
			s.graphRAG.BuildIndex(context.Background(), kbID, strategy)
		}()

		return &IndexStatus{
			KBId:     kbID,
			Status:   "building",
			Progress: 0,
		}, nil
	}

	return s.graphRAG.BuildIndex(ctx, kbID, strategy)
}

// GetIndexStatus returns the current index status for a knowledge base
func (s *KnowledgeBaseService) GetIndexStatus(ctx context.Context, kbID string) (*IndexStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, ok := s.indexes[kbID]
	if !ok {
		return &IndexStatus{
			KBId:   kbID,
			Status: "unknown",
		}, nil
	}

	return status, nil
}

// GetSummary returns a summary of all knowledge bases
func (s *KnowledgeBaseService) GetSummary(ctx context.Context) (*Summary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := &Summary{}

	for _, kb := range s.db {
		summary.TotalModels++

		switch kb.Status {
		case "running":
			summary.RunningModels++
		case "stopped":
			summary.StoppedModels++
		case "error":
			summary.ErrorModels++
		case "deployed":
			summary.DeployedModels++
		case "uploaded":
			summary.UploadedModels++
		}
	}

	return summary, nil
}
