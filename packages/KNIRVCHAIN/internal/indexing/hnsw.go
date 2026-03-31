package indexing

import (
	"container/heap"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrDimensionMismatch = &IndexError{"dimension mismatch"}

type IndexError struct {
	msg string
}

func (e *IndexError) Error() string {
	return e.msg
}

type HNSWNode struct {
	ID          uuid.UUID
	Vector      []float32
	Connections map[int][]uuid.UUID
	Level       int
}

type HNSWIndex struct {
	dimension      int
	M              int
	Mmax           int
	Mmax0          int
	efConstruction int
	ef             int
	ml             float64
	nodes          map[uuid.UUID]*HNSWNode
	entryPoint     *HNSWNode
	mu             sync.RWMutex
	randSource     *rand.Rand
}

func NewHNSWIndex(dimension, m, efConstruction int) *HNSWIndex {
	return &HNSWIndex{
		dimension:      dimension,
		M:              m,
		Mmax:           m,
		Mmax0:          m * 2,
		efConstruction: efConstruction,
		ef:             efConstruction,
		ml:             1.0 / math.Log(2.0),
		nodes:          make(map[uuid.UUID]*HNSWNode),
		entryPoint:     nil,
		randSource:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (h *HNSWIndex) SetEf(ef int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ef = ef
}

func (h *HNSWIndex) Add(id uuid.UUID, vector []float32) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(vector) != h.dimension {
		return ErrDimensionMismatch
	}

	level := h.randomLevel()

	node := &HNSWNode{
		ID:          id,
		Vector:      vector,
		Connections: make(map[int][]uuid.UUID),
		Level:       level,
	}

	for l := 0; l <= level; l++ {
		node.Connections[l] = make([]uuid.UUID, 0)
	}

	if h.entryPoint == nil {
		h.entryPoint = node
		h.nodes[id] = node
		return nil
	}

	ep := []uuid.UUID{h.entryPoint.ID}

	for lc := h.entryPoint.Level; lc > level; lc-- {
		ep = h.searchLayer(vector, ep, 1, lc)
	}

	for lc := level; lc >= 0; lc-- {
		candidates := h.searchLayer(vector, ep, h.efConstruction, lc)
		ep = h.selectNeighbors(vector, candidates, h.efConstruction, lc)

		for _, candidateID := range ep {
			if candidateID == id {
				continue
			}
			node.Connections[lc] = append(node.Connections[lc], candidateID)
			if candidate, ok := h.nodes[candidateID]; ok {
				candidate.Connections[lc] = append(candidate.Connections[lc], id)
			}
		}

		if lc == 0 {
			break
		}
	}

	h.nodes[id] = node
	if level > h.entryPoint.Level {
		h.entryPoint = node
	}

	return nil
}

func (h *HNSWIndex) randomLevel() int {
	l := -int(math.Log(h.randSource.Float64()) * h.ml)
	if l < 0 {
		l = 0
	}
	return l
}

func (h *HNSWIndex) searchLayer(query []float32, entryPoints []uuid.UUID, ef, layer int) []uuid.UUID {
	var visited = make(map[uuid.UUID]bool)
	for _, ep := range entryPoints {
		visited[ep] = true
	}

	candidates := &priorityQueue{}
	heap.Init(candidates)

	for _, epID := range entryPoints {
		if node, ok := h.nodes[epID]; ok {
			dist := h.distance(query, node.Vector)
			heap.Push(candidates, &pqItem{id: epID, dist: dist})
		}
	}

	result := make([]uuid.UUID, 0)
	resultDists := make(map[uuid.UUID]float32)

	for candidates.Len() > 0 {
		current := heap.Pop(candidates).(*pqItem)
		result = append(result, current.id)
		resultDists[current.id] = current.dist

		if len(result) > ef {
			break
		}

		if node, ok := h.nodes[current.id]; ok {
			for _, neighborID := range node.Connections[layer] {
				if !visited[neighborID] {
					visited[neighborID] = true
					if neighbor, ok := h.nodes[neighborID]; ok {
						dist := h.distance(query, neighbor.Vector)
						if len(result) <= ef || dist <= resultDists[result[len(result)-1]] {
							heap.Push(candidates, &pqItem{id: neighborID, dist: dist})
						}
					}
				}
			}
		}
	}

	return result
}

func (h *HNSWIndex) selectNeighbors(query []float32, candidates []uuid.UUID, m, layer int) []uuid.UUID {
	if len(candidates) == 0 {
		return nil
	}

	neighbors := make([]uuid.UUID, 0)

	if layer == 0 {
		pq := &priorityQueue{}
		heap.Init(pq)

		for _, candidateID := range candidates {
			if node, ok := h.nodes[candidateID]; ok {
				dist := h.distance(query, node.Vector)
				heap.Push(pq, &pqItem{id: candidateID, dist: dist})
			}
		}

		for pq.Len() > 0 && len(neighbors) < h.Mmax0 {
			item := heap.Pop(pq).(*pqItem)
			neighbors = append(neighbors, item.id)
		}
	} else {
		pq := &priorityQueue{}
		heap.Init(pq)

		for _, candidateID := range candidates {
			if node, ok := h.nodes[candidateID]; ok {
				dist := h.distance(query, node.Vector)
				heap.Push(pq, &pqItem{id: candidateID, dist: dist})
			}
		}

		for pq.Len() > 0 && len(neighbors) < h.Mmax {
			item := heap.Pop(pq).(*pqItem)
			neighbors = append(neighbors, item.id)
		}
	}

	return neighbors
}

func (h *HNSWIndex) Search(query []float32, k int) ([]uuid.UUID, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(query) != h.dimension {
		return nil, ErrDimensionMismatch
	}

	if h.entryPoint == nil {
		return []uuid.UUID{}, nil
	}

	ep := []uuid.UUID{h.entryPoint.ID}

	for lc := h.entryPoint.Level; lc > 0; lc-- {
		ep = h.searchLayer(query, ep, 1, lc)
	}

	candidates := h.searchLayer(query, ep, h.ef, 0)
	results := h.selectNeighbors(query, candidates, k, 0)

	if len(results) > k {
		results = results[:k]
	}

	return results, nil
}

func (h *HNSWIndex) Remove(id uuid.UUID) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	node, ok := h.nodes[id]
	if !ok {
		return nil
	}

	for layer := 0; layer <= node.Level; layer++ {
		for _, neighborID := range node.Connections[layer] {
			if neighbor, ok := h.nodes[neighborID]; ok {
				newConns := make([]uuid.UUID, 0)
				for _, connID := range neighbor.Connections[layer] {
					if connID != id {
						newConns = append(newConns, connID)
					}
				}
				neighbor.Connections[layer] = newConns
			}
		}
	}

	delete(h.nodes, id)

	if h.entryPoint != nil && h.entryPoint.ID == id {
		h.entryPoint = nil
		for _, n := range h.nodes {
			if h.entryPoint == nil || n.Level > h.entryPoint.Level {
				h.entryPoint = n
			}
		}
	}

	return nil
}

func (h *HNSWIndex) distance(a, b []float32) float32 {
	var sum float32
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return sum
}

type pqItem struct {
	id   uuid.UUID
	dist float32
}

type priorityQueue []*pqItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].dist < pq[j].dist }
func (pq priorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }
func (pq *priorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*pqItem))
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

func timeNow() interface{ UnixNano() int64 } {
	return &mockTime{}
}

type mockTime struct{}

func (m *mockTime) UnixNano() int64 {
	return rand.Int63()
}
