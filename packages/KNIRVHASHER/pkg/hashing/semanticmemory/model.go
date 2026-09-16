// Package semanticmemory implements a bounded, streaming semantic model.
//
// It is deliberately much smaller than a transformer: every frequent target
// token has one centroid of the embeddings that preceded it. Training updates
// a centroid in O(dimensions) and retains at most MaxPrototypes centroids.
package semanticmemory

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

const (
	FormatVersion        = 1
	DefaultMaxPrototypes = 4096
)

// Prototype is the online mean embedding for one next-token target. Count is
// a Space-Saving frequency estimate; Samples is the number of vectors that
// contributed to the centroid after its most recent allocation.
type Prototype struct {
	TargetTokenID int32     `json:"target_token_id"`
	Count         uint64    `json:"count"`
	Samples       uint64    `json:"samples"`
	Centroid      []float32 `json:"centroid"`
}

// Model is a compact semantic next-token memory. The exported fields are the
// on-disk format; lookup and heap state are rebuilt after loading.
type Model struct {
	Version       int          `json:"version"`
	Embedder      string       `json:"embedder"`
	Dimensions    int          `json:"dimensions"`
	MaxPrototypes int          `json:"max_prototypes"`
	FramesSeen    uint64       `json:"frames_seen"`
	FramesIndexed uint64       `json:"frames_indexed"`
	Evictions     uint64       `json:"evictions"`
	Prototypes    []*Prototype `json:"prototypes"`

	byTarget map[int32]int
	minHeap  prototypeHeap
}

// New creates an empty bounded model. maxPrototypes must be positive.
func New(dimensions, maxPrototypes int, embedder string) (*Model, error) {
	if dimensions <= 0 {
		return nil, fmt.Errorf("dimensions must be positive")
	}
	if maxPrototypes <= 0 {
		return nil, fmt.Errorf("max prototypes must be positive")
	}
	m := &Model{
		Version:       FormatVersion,
		Embedder:      embedder,
		Dimensions:    dimensions,
		MaxPrototypes: maxPrototypes,
		Prototypes:    make([]*Prototype, 0, maxPrototypes),
	}
	m.rebuildIndex()
	return m, nil
}

// Update folds vector into the centroid for targetTokenID. When full, the
// least-frequent prototype is replaced using the Space-Saving heavy-hitter
// rule, keeping memory bounded while retaining common targets.
func (m *Model) Update(vector []float32, targetTokenID int32) error {
	if len(vector) != m.Dimensions {
		return fmt.Errorf("embedding dimensions = %d, want %d", len(vector), m.Dimensions)
	}
	m.FramesSeen++
	if isZero(vector) {
		return nil
	}
	if m.byTarget == nil {
		m.rebuildIndex()
	}

	idx, ok := m.byTarget[targetTokenID]
	if !ok {
		if len(m.Prototypes) < m.MaxPrototypes {
			idx = len(m.Prototypes)
			m.Prototypes = append(m.Prototypes, &Prototype{TargetTokenID: targetTokenID, Centroid: append([]float32(nil), vector...)})
			m.byTarget[targetTokenID] = idx
			heap.Push(&m.minHeap, idx)
		} else {
			idx = m.minHeap.entries[0]
			victim := m.Prototypes[idx]
			delete(m.byTarget, victim.TargetTokenID)
			victim.TargetTokenID = targetTokenID
			victim.Count++ // Space-Saving estimate: old minimum count + 1.
			victim.Samples = 0
			copy(victim.Centroid, vector)
			m.byTarget[targetTokenID] = idx
			m.Evictions++
			heap.Fix(&m.minHeap, m.minHeap.position[idx])
		}
	}

	p := m.Prototypes[idx]
	if p.Samples == 0 {
		p.Samples = 1
		if p.Count == 0 {
			p.Count = 1
		}
		copy(p.Centroid, vector)
	} else {
		p.Count++
		p.Samples++
		weight := float32(1.0 / float64(p.Samples))
		for i, v := range vector {
			p.Centroid[i] += (v - p.Centroid[i]) * weight
		}
	}
	heap.Fix(&m.minHeap, m.minHeap.position[idx])
	m.FramesIndexed++
	return nil
}

// Query returns the target represented by the nearest centroid and its cosine
// similarity. It performs no allocation proportional to the training corpus.
func (m *Model) Query(vector []float32) (targetTokenID int32, similarity float32, ok bool) {
	if len(vector) != m.Dimensions || len(m.Prototypes) == 0 || isZero(vector) {
		return 0, 0, false
	}
	best := float32(-2)
	for _, p := range m.Prototypes {
		score := cosine(vector, p.Centroid)
		if score > best {
			best, targetTokenID, ok = score, p.TargetTokenID, true
		}
	}
	return targetTokenID, best, ok
}

func (m *Model) Save(path string) error {
	if m.Version == 0 {
		m.Version = FormatVersion
	}
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal semantic memory: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write semantic memory: %w", err)
	}
	return nil
}

func Load(path string) (*Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read semantic memory: %w", err)
	}
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decode semantic memory: %w", err)
	}
	if m.Version != FormatVersion || m.Dimensions <= 0 || m.MaxPrototypes <= 0 {
		return nil, fmt.Errorf("unsupported semantic memory format")
	}
	if len(m.Prototypes) > m.MaxPrototypes {
		return nil, fmt.Errorf("semantic memory has %d prototypes, maximum is %d", len(m.Prototypes), m.MaxPrototypes)
	}
	for i, p := range m.Prototypes {
		if p == nil || len(p.Centroid) != m.Dimensions {
			return nil, fmt.Errorf("invalid prototype %d", i)
		}
	}
	m.rebuildIndex()
	return &m, nil
}

func (m *Model) rebuildIndex() {
	m.byTarget = make(map[int32]int, len(m.Prototypes))
	m.minHeap = prototypeHeap{
		model:    m,
		entries:  make([]int, 0, len(m.Prototypes)),
		position: make(map[int]int, len(m.Prototypes)),
	}
	for i, p := range m.Prototypes {
		m.byTarget[p.TargetTokenID] = i
		m.minHeap.entries = append(m.minHeap.entries, i)
	}
	heap.Init(&m.minHeap)
}

type prototypeHeap struct {
	model    *Model
	entries  []int
	position map[int]int // prototype index -> heap position
}

func (h prototypeHeap) Len() int { return len(h.entries) }
func (h prototypeHeap) Less(i, j int) bool {
	return h.model.Prototypes[h.entries[i]].Count < h.model.Prototypes[h.entries[j]].Count
}
func (h prototypeHeap) Swap(i, j int) {
	h.entries[i], h.entries[j] = h.entries[j], h.entries[i]
	h.position[h.entries[i]], h.position[h.entries[j]] = i, j
}
func (h *prototypeHeap) Push(x interface{}) {
	idx := x.(int)
	h.entries = append(h.entries, idx)
	h.position[idx] = len(h.entries) - 1
}
func (h *prototypeHeap) Pop() interface{} {
	idx := h.entries[len(h.entries)-1]
	h.entries = h.entries[:len(h.entries)-1]
	delete(h.position, idx)
	return idx
}

func isZero(v []float32) bool {
	for _, x := range v {
		if x != 0 {
			return false
		}
	}
	return true
}

func cosine(a, b []float32) float32 {
	var dot, normA, normB float64
	for i, x := range a {
		dot += float64(x) * float64(b[i])
		normA += float64(x) * float64(x)
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / math.Sqrt(normA*normB))
}
