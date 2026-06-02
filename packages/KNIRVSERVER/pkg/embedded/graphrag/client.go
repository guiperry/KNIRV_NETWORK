package graphrag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics exposes counters so the caller can register them with Prometheus
// via the MetricCollector callback, or simply log them.
type Metrics struct {
	QueryCount      atomic.Int64
	IndexCount      atomic.Int64
	EmbedCount      atomic.Int64
	IndexDurationNs atomic.Int64 // cumulative nanoseconds
}

func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"queries_total":         m.QueryCount.Load(),
		"indexed_docs_total":    m.IndexCount.Load(),
		"embeddings_total":      m.EmbedCount.Load(),
		"index_duration_ns_sum": m.IndexDurationNs.Load(),
	}
}

type Client struct {
	mu      sync.RWMutex
	indexes map[string]*Index
	logger  *slog.Logger
	Metrics *Metrics
}

type Index struct {
	KBID        string
	Status      string
	NodesCount  int
	EdgesCount  int
	ChunksCount int
	LastUpdated time.Time
	Err         error
}

func NewClient(logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		indexes: make(map[string]*Index),
		logger:  logger,
		Metrics: &Metrics{},
	}
}

func (c *Client) Query(ctx context.Context, kbID string, q *GraphQuery) (*GraphResult, error) {
	if q == nil {
		return nil, fmt.Errorf("query cannot be nil")
	}

	c.mu.RLock()
	idx, ok := c.indexes[kbID]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("index not found for knowledge base: %s", kbID)
	}
	if idx.Status != "ready" {
		return nil, fmt.Errorf("index is not ready (status: %s)", idx.Status)
	}

	queryJSON, err := json.Marshal(map[string]interface{}{
		"query": q.Query,
		"mode":  q.Mode,
		"limit": q.Limit,
		"kb_id": kbID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	c.Metrics.QueryCount.Add(1)
	start := time.Now()
	raw, err := Query(string(queryJSON), q.Limit)
	c.logger.Debug("graphrag query",
		"kb_id", kbID,
		"duration", time.Since(start).String(),
		"error", err,
	)
	if err != nil {
		return nil, fmt.Errorf("graphrag query failed: %w", err)
	}

	var result GraphResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("graphrag result parse failed: %w", err)
	}

	result.Timestamp = time.Now()
	return &result, nil
}

func (c *Client) BuildIndex(ctx context.Context, kbID string, strategy string) (*IndexStatus, error) {
	c.mu.Lock()
	c.indexes[kbID] = &Index{KBID: kbID, Status: "building"}
	c.mu.Unlock()

	idx, err := c.buildIndexSync(ctx, kbID, strategy)
	if err != nil {
		return nil, err
	}

	return &IndexStatus{
		KBId:        idx.KBID,
		Status:      idx.Status,
		Progress:    100.0,
		NodesCount:  idx.NodesCount,
		EdgesCount:  idx.EdgesCount,
		ChunksCount: idx.ChunksCount,
		LastUpdated: idx.LastUpdated,
	}, nil
}

func (c *Client) buildIndexSync(ctx context.Context, kbID string, strategy string) (*Index, error) {
	ch := make(chan *Index, 1)
	go func() {
		c.mu.RLock()
		idx := c.indexes[kbID]
		c.mu.RUnlock()

		if idx == nil {
			ch <- &Index{KBID: kbID, Status: "error", Err: fmt.Errorf("index not found for kb: %s", kbID)}
			return
		}

		start := time.Now()
		raw, err := IndexDocumentWithResult(kbID, []byte(strategy))
		dur := time.Since(start)
		c.Metrics.IndexCount.Add(1)
		c.Metrics.IndexDurationNs.Add(dur.Nanoseconds())
		c.logger.Debug("graphrag build index",
			"kb_id", kbID,
			"strategy", strategy,
			"duration", dur.String(),
			"error", err,
		)

		c.mu.Lock()
		defer c.mu.Unlock()
		if err != nil {
			idx.Status = "error"
			idx.Err = err
		} else {
			idx.Status = "ready"
			idx.NodesCount = 0
			idx.EdgesCount = 0
			idx.ChunksCount = 0
			if raw != nil {
				var extraction struct {
					Entities      []json.RawMessage `json:"entities"`
					Relationships []json.RawMessage `json:"relationships"`
					EntityCount   int               `json:"entity_count"`
					RelCount      int               `json:"relationship_count"`
				}
				if err := json.Unmarshal(raw, &extraction); err == nil {
					idx.NodesCount = extraction.EntityCount
					idx.EdgesCount = extraction.RelCount
					idx.ChunksCount = len(extraction.Entities) + len(extraction.Relationships)
				}
			}
			idx.LastUpdated = time.Now()
		}
		ch <- idx
	}()

	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) GetIndexStatus(ctx context.Context, kbID string) (*IndexStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	idx, ok := c.indexes[kbID]
	if !ok {
		return &IndexStatus{KBId: kbID, Status: "not_built"}, nil
	}

	errMsg := ""
	if idx.Err != nil {
		errMsg = idx.Err.Error()
	}

	return &IndexStatus{
		KBId:         idx.KBID,
		Status:       idx.Status,
		Progress:     100.0,
		NodesCount:   idx.NodesCount,
		EdgesCount:   idx.EdgesCount,
		ChunksCount:  idx.ChunksCount,
		LastUpdated:  idx.LastUpdated,
		ErrorMessage: errMsg,
	}, nil
}

func (c *Client) IndexDocument(ctx context.Context, docID string, content []byte) error {
	if docID == "" {
		return fmt.Errorf("docID cannot be empty")
	}
	if len(content) == 0 {
		return fmt.Errorf("content cannot be empty")
	}

	c.mu.RLock()
	_, ok := c.indexes["default"]
	c.mu.RUnlock()

	if !ok {
		c.mu.Lock()
		c.indexes["default"] = &Index{KBID: "default", Status: "building"}
		c.mu.Unlock()
	}

	start := time.Now()
	extractionRaw, err := IndexDocumentWithResult(docID, content)
	dur := time.Since(start)
	c.Metrics.IndexCount.Add(1)
	c.Metrics.IndexDurationNs.Add(dur.Nanoseconds())
	c.logger.Debug("graphrag index document",
		"doc_id", docID,
		"content_bytes", len(content),
		"duration", dur.String(),
		"error", err,
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	idx, exists := c.indexes["default"]
	if !exists {
		idx = &Index{KBID: "default"}
		c.indexes["default"] = idx
	}

	if err != nil {
		idx.Err = err
		idx.Status = "error"
		return err
	}

	idx.Status = "ready"
	idx.LastUpdated = time.Now()

	if extractionRaw != nil {
		var extraction struct {
			Entities      []json.RawMessage `json:"entities"`
			Relationships []json.RawMessage `json:"relationships"`
			EntityCount   int               `json:"entity_count"`
			RelCount      int               `json:"relationship_count"`
		}
		if err := json.Unmarshal(extractionRaw, &extraction); err == nil {
			idx.NodesCount += extraction.EntityCount
			idx.EdgesCount += extraction.RelCount
			idx.ChunksCount += len(extraction.Entities) + len(extraction.Relationships)
		}
	}

	return nil
}

func (c *Client) IndexDocumentWithResult(ctx context.Context, docID string, content []byte) ([]byte, error) {
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("content cannot be empty")
	}

	start := time.Now()
	raw, err := IndexDocumentWithResult(docID, content)
	dur := time.Since(start)
	c.Metrics.IndexCount.Add(1)
	c.Metrics.IndexDurationNs.Add(dur.Nanoseconds())
	c.logger.Debug("graphrag index doc with result",
		"doc_id", docID,
		"content_bytes", len(content),
		"duration", dur.String(),
		"result_bytes", len(raw),
		"error", err,
	)
	return raw, err
}

func (c *Client) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	c.Metrics.EmbedCount.Add(1)
	start := time.Now()
	embeddings, err := EmbedTexts(texts)
	c.logger.Debug("graphrag embed texts",
		"text_count", len(texts),
		"duration", time.Since(start).String(),
		"error", err,
	)
	if err != nil {
		return nil, fmt.Errorf("graphrag embed texts failed: %w", err)
	}

	return embeddings, nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	start := time.Now()
	err := Shutdown()
	c.logger.Debug("graphrag shutdown", "duration", time.Since(start).String())
	if err != nil {
		return fmt.Errorf("graphrag shutdown failed: %w", err)
	}

	c.indexes = make(map[string]*Index)
	return nil
}
