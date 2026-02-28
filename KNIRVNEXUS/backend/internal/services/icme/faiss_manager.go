package icme

import (
	"encoding/json"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"backend_server/internal/database"
)

const EmbedDim = 768

type FAISSIndexManager struct {
	mu       sync.Mutex
	vectors  [][]float32
	metadata map[int64]VectorMeta
	nextID   int64
	db       *database.BuntDBManager
	logger   *zap.Logger
}

func NewFAISSIndexManager(db *database.BuntDBManager, logger *zap.Logger) (*FAISSIndexManager, error) {
	return &FAISSIndexManager{
		vectors:  make([][]float32, 0),
		metadata: make(map[int64]VectorMeta),
		db:       db,
		logger:   logger,
	}, nil
}

func (m *FAISSIndexManager) Add(signalID string, vector []float32, meta VectorMeta) (int64, error) {
	if len(vector) != EmbedDim {
		return -1, fmt.Errorf("vector dim mismatch: got %d, want %d", len(vector), EmbedDim)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID
	m.nextID++
	meta.VectorID = id

	m.vectors = append(m.vectors, vector)
	m.metadata[id] = meta

	data, _ := json.Marshal(meta)
	key := fmt.Sprintf("icme:vectors:%d", id)
	if err := m.db.StoreJSON(key, data); err != nil {
		m.logger.Warn("icme faiss meta persist failed", zap.Error(err))
	}

	return id, nil
}

func (m *FAISSIndexManager) Search(query []float32, k int) ([]VectorMeta, []float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.vectors) == 0 {
		return nil, nil, nil
	}

	type scoredResult struct {
		index int
		score float32
	}

	results := make([]scoredResult, 0, len(m.vectors))
	for i, vec := range m.vectors {
		score := cosineDistance(query, vec)
		results = append(results, scoredResult{index: i, score: score})
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score < results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if k > len(results) {
		k = len(results)
	}

	metas := make([]VectorMeta, 0, k)
	scores := make([]float32, 0, k)
	for i := 0; i < k; i++ {
		idx := results[i].index
		realID := int64(idx)
		if meta, ok := m.metadata[realID]; ok {
			metas = append(metas, meta)
			scores = append(scores, results[i].score)
		}
	}

	return metas, scores, nil
}

func cosineDistance(a, b []float32) float32 {
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 1.0
	}
	return 1.0 - dot/(normA*normB)
}
