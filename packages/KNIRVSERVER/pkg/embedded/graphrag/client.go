package graphrag

// Client is consumed exclusively by backend_server (KNIRV_CORP) — a separate
// OS process from KNIRVSERVER, which owns and initializes the actual
// graphrag-rs engine (see pkg/embedded/manager.go). Every method here dials
// the Unix domain socket KNIRVSERVER exposes via StartServer (server.go)
// instead of calling the package-level CGo wrappers (graphrag.go) directly —
// calling those directly from backend_server's own process would operate on
// backend_server's own separately-linked, never-initialized copy of the
// Rust static library, not the one KNIRVSERVER actually initializes.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
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
	mu         sync.RWMutex
	indexes    map[string]*Index
	logger     *slog.Logger
	Metrics    *Metrics
	socketPath string
	httpClient *http.Client
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

// unixHTTPClient builds an http.Client that dials a Unix domain socket
// regardless of the URL host given to it — callers use the "http://unix"
// base URL by convention (matching pkg/embedded/validationchain's pattern).
func unixHTTPClient(socketPath string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		},
	}
}

// NewClient builds a Client that reaches the embedded graphrag engine over
// the Unix domain socket at socketPath (see StartServer / server.go, owned
// and started by KNIRVSERVER's embedded manager).
func NewClient(logger *slog.Logger, socketPath string) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		indexes:    make(map[string]*Index),
		logger:     logger,
		Metrics:    &Metrics{},
		socketPath: socketPath,
		httpClient: unixHTTPClient(socketPath, 30*time.Second),
	}
}

// post sends a JSON-encoded body to path over the Unix socket and returns
// the raw response body on a 200. A non-2xx response is turned into an
// error carrying the response body as its message.
func (c *Client) post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix"+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to build graphrag socket request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphrag socket request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read graphrag socket response from %s: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphrag socket %s returned %d: %s", path, resp.StatusCode, string(raw))
	}
	return raw, nil
}

func (c *Client) Query(ctx context.Context, kbID string, q *GraphQuery) (*GraphResult, error) {
	if q == nil {
		return nil, fmt.Errorf("query cannot be nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
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
	raw, err := c.post(ctx, "/query", socketQueryRequest{Query: string(queryJSON), Limit: q.Limit})
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

// BuildIndex marks kbID's index "ready". Documents are indexed through
// IndexDocument/IndexDocumentWithResult as they enter the system (see
// memory_store.go's/service.go's syncToGraphRAG and
// knowledge_base/service.go's IndexKnowledgeBase, which reads and indexes
// the KB's content before calling this); `strategy` describes the caller's
// rebuild policy and is never sent to the indexer as document content.
//
// The node/edge/chunk counts reported come from the shared "default" index
// bucket that IndexDocument/IndexDocumentWithResult populate — kbID-scoped
// counts aren't tracked separately since a single graphrag engine instance
// backs every knowledge base.
func (c *Client) BuildIndex(ctx context.Context, kbID string, strategy string) (*IndexStatus, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if kbID == "" {
		return nil, fmt.Errorf("knowledge base id cannot be empty")
	}
	c.mu.Lock()
	c.indexes[kbID] = &Index{KBID: kbID, Status: "building"}
	c.mu.Unlock()

	idx, err := c.buildIndexSync(ctx, kbID)
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

func (c *Client) buildIndexSync(ctx context.Context, kbID string) (*Index, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ch := make(chan *Index, 1)
	go func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		idx := c.indexes[kbID]
		if idx == nil {
			ch <- &Index{KBID: kbID, Status: "error", Err: fmt.Errorf("index not found for kb: %s", kbID)}
			return
		}

		// Reflect the real extraction counts already recorded by
		// IndexDocument/IndexDocumentWithResult (tracked under the shared
		// "default" bucket) instead of reporting zero.
		if shared, ok := c.indexes["default"]; ok {
			idx.NodesCount = shared.NodesCount
			idx.EdgesCount = shared.EdgesCount
			idx.ChunksCount = shared.ChunksCount
		}
		idx.Status = "ready"
		idx.LastUpdated = time.Now()
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

// IndexDocument indexes content under docID and folds its extraction counts
// into the shared "default" bucket (the bucket Query always checks — see
// UnifiedMemorySystem.Query in KNIRV_CORP, which always queries kbID
// "default"). The extraction itself is discarded; callers that need it
// should use IndexDocumentWithResult.
func (c *Client) IndexDocument(ctx context.Context, docID string, content []byte) error {
	_, err := c.indexDocument(ctx, docID, content)
	return err
}

func (c *Client) IndexDocumentWithResult(ctx context.Context, docID string, content []byte) ([]byte, error) {
	return c.indexDocument(ctx, docID, content)
}

func (c *Client) indexDocument(ctx context.Context, docID string, content []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("content cannot be empty")
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
	raw, err := c.post(ctx, "/index-with-result", socketIndexRequest{DocID: docID, Content: string(content)})
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
		return nil, fmt.Errorf("graphrag index document failed: %w", err)
	}

	idx.Status = "ready"
	idx.LastUpdated = time.Now()

	if raw != nil {
		var extraction struct {
			Entities      []json.RawMessage `json:"entities"`
			Relationships []json.RawMessage `json:"relationships"`
			EntityCount   int               `json:"entity_count"`
			RelCount      int               `json:"relationship_count"`
		}
		if err := json.Unmarshal(raw, &extraction); err == nil {
			idx.NodesCount += extraction.EntityCount
			idx.EdgesCount += extraction.RelCount
			idx.ChunksCount += len(extraction.Entities) + len(extraction.Relationships)
		}
	}

	return raw, nil
}

func (c *Client) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	c.Metrics.EmbedCount.Add(1)
	start := time.Now()
	raw, err := c.post(ctx, "/embed", socketEmbedRequest{Texts: texts})
	c.logger.Debug("graphrag embed texts",
		"text_count", len(texts),
		"duration", time.Since(start).String(),
		"error", err,
	)
	if err != nil {
		return nil, fmt.Errorf("graphrag embed texts failed: %w", err)
	}

	var embeddings [][]float32
	if err := json.Unmarshal(raw, &embeddings); err != nil {
		return nil, fmt.Errorf("failed to parse embeddings: %w", err)
	}

	return embeddings, nil
}

// Close releases this client's local state. It intentionally does not shut
// down the graphrag engine itself — KNIRVSERVER's embedded manager is the
// sole owner of that lifecycle (matching Validation Chain/Transaction
// Chain); backend_server merely holds a socket client to it.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.httpClient.CloseIdleConnections()
	c.indexes = make(map[string]*Index)
	return nil
}
