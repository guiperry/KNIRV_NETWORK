package vector

import (
	"KNIRVGRAPH/internal/storage"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"sync"
)

// Metric controls how vectors are compared. Search scores are always ordered
// with the best match first (cosine/dot are similarities, euclidean is
// converted to 1/(1+distance)).
type Metric string

const (
	MetricCosine    Metric = "cosine"
	MetricDot       Metric = "dot"
	MetricEuclidean Metric = "euclidean"
)

const defaultPersistenceKey = "vector_hnsw_v1"

type Options struct {
	Metric         Metric `json:"metric"`
	M              int    `json:"m"`
	EFConstruction int    `json:"ef_construction"`
	EFSearch       int    `json:"ef_search"`
	PersistenceKey string `json:"persistence_key"`
}

func DefaultOptions() Options {
	return Options{Metric: MetricCosine, M: 16, EFConstruction: 100, EFSearch: 64, PersistenceKey: defaultPersistenceKey}
}

type hnswNode struct {
	ID        string           `json:"id"`
	Vector    []float32        `json:"vector"`
	Level     int              `json:"level"`
	Neighbors map[int][]string `json:"neighbors"`
}

type persistedIndex struct {
	Dimension  int                  `json:"dimension"`
	Options    Options              `json:"options"`
	EntryPoint string               `json:"entry_point"`
	MaxLevel   int                  `json:"max_level"`
	Nodes      map[string]*hnswNode `json:"nodes"`
}

// VectorIndex is a concurrent HNSW approximate nearest-neighbor index. When a
// Storage is supplied every completed mutation is persisted atomically.
type VectorIndex struct {
	dimension  int
	options    Options
	nodes      map[string]*hnswNode
	entryPoint string
	maxLevel   int
	store      storage.Storage
	mu         sync.RWMutex
}

func NewVectorIndex(dimension int) *VectorIndex {
	v, _ := NewPersistentVectorIndex(dimension, DefaultOptions(), nil)
	return v
}

func NewPersistentVectorIndex(dimension int, options Options, store storage.Storage) (*VectorIndex, error) {
	if dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}
	options = normalizeOptions(options)
	v := &VectorIndex{dimension: dimension, options: options, nodes: make(map[string]*hnswNode), maxLevel: -1, store: store}
	if store != nil {
		if err := v.load(); err != nil && err != storage.ErrNotFound {
			return nil, fmt.Errorf("load vector index: %w", err)
		}
	}
	return v, nil
}

func normalizeOptions(o Options) Options {
	d := DefaultOptions()
	if o.M <= 1 {
		o.M = d.M
	}
	if o.EFConstruction < o.M {
		o.EFConstruction = d.EFConstruction
	}
	if o.EFSearch < 1 {
		o.EFSearch = d.EFSearch
	}
	if o.PersistenceKey == "" {
		o.PersistenceKey = d.PersistenceKey
	}
	switch o.Metric {
	case MetricCosine, MetricDot, MetricEuclidean:
	default:
		o.Metric = MetricCosine
	}
	return o
}

func (v *VectorIndex) Add(id string, vector []float32) error {
	if id == "" {
		return fmt.Errorf("vector id is required")
	}
	if len(vector) != v.dimension {
		return fmt.Errorf("dimension mismatch: expected %d, got %d", v.dimension, len(vector))
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, exists := v.nodes[id]; exists {
		v.removeLocked(id)
	}
	vec := append([]float32(nil), vector...)
	level := deterministicLevel(id, v.options.M)
	node := &hnswNode{ID: id, Vector: vec, Level: level, Neighbors: make(map[int][]string)}
	if len(v.nodes) == 0 {
		v.nodes[id], v.entryPoint, v.maxLevel = node, id, level
		return v.persistLocked()
	}

	entry := v.entryPoint
	for layer := v.maxLevel; layer > level; layer-- {
		entry = v.greedyClosest(vec, entry, layer)
	}
	upper := level
	if v.maxLevel < upper {
		upper = v.maxLevel
	}
	for layer := upper; layer >= 0; layer-- {
		candidates := v.searchLayer(vec, []string{entry}, v.options.EFConstruction, layer)
		selected := v.closestIDs(vec, candidates, v.options.M)
		node.Neighbors[layer] = selected
		for _, neighborID := range selected {
			v.connectLocked(neighborID, id, layer, vec)
		}
		if len(candidates) > 0 {
			entry = candidates[0]
		}
	}
	v.nodes[id] = node
	// connectLocked needs the new node to exist for pruning on later updates.
	for layer, neighbors := range node.Neighbors {
		for _, neighborID := range neighbors {
			v.pruneLocked(neighborID, layer)
		}
	}
	if level > v.maxLevel {
		v.entryPoint, v.maxLevel = id, level
	}
	return v.persistLocked()
}

func (v *VectorIndex) connectLocked(existingID, newID string, layer int, newVector []float32) {
	n := v.nodes[existingID]
	if n == nil {
		return
	}
	n.Neighbors[layer] = appendUnique(n.Neighbors[layer], newID)
}

func (v *VectorIndex) pruneLocked(id string, layer int) {
	n := v.nodes[id]
	if n == nil || len(n.Neighbors[layer]) <= v.options.M {
		return
	}
	n.Neighbors[layer] = v.closestIDs(n.Vector, n.Neighbors[layer], v.options.M)
}

func (v *VectorIndex) Delete(id string) { _ = v.DeleteWithError(id) }

func (v *VectorIndex) DeleteWithError(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.removeLocked(id)
	return v.persistLocked()
}

func (v *VectorIndex) removeLocked(id string) {
	if _, ok := v.nodes[id]; !ok {
		return
	}
	delete(v.nodes, id)
	for _, n := range v.nodes {
		for layer, ids := range n.Neighbors {
			n.Neighbors[layer] = removeID(ids, id)
		}
	}
	if v.entryPoint == id {
		v.reselectEntryLocked()
	}
}

func (v *VectorIndex) reselectEntryLocked() {
	v.entryPoint, v.maxLevel = "", -1
	for id, n := range v.nodes {
		if n.Level > v.maxLevel || (n.Level == v.maxLevel && (v.entryPoint == "" || id < v.entryPoint)) {
			v.entryPoint, v.maxLevel = id, n.Level
		}
	}
}

func (v *VectorIndex) Search(query []float32, topK int) ([]string, []float64, error) {
	if len(query) != v.dimension {
		return nil, nil, fmt.Errorf("dimension mismatch: expected %d, got %d", v.dimension, len(query))
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if len(v.nodes) == 0 {
		return []string{}, []float64{}, nil
	}
	if topK <= 0 || topK > len(v.nodes) {
		topK = len(v.nodes)
	}
	entry := v.entryPoint
	for layer := v.maxLevel; layer > 0; layer-- {
		entry = v.greedyClosest(query, entry, layer)
	}
	ids := v.searchLayer(query, []string{entry}, max(topK, v.options.EFSearch), 0)
	ids = v.closestIDs(query, ids, topK)
	scores := make([]float64, len(ids))
	for i, id := range ids {
		scores[i] = v.score(query, v.nodes[id].Vector)
	}
	return ids, scores, nil
}

func (v *VectorIndex) greedyClosest(query []float32, entry string, layer int) string {
	best, bestScore := entry, v.score(query, v.nodes[entry].Vector)
	changed := true
	for changed {
		changed = false
		for _, id := range v.nodes[best].Neighbors[layer] {
			if n := v.nodes[id]; n != nil {
				s := v.score(query, n.Vector)
				if s > bestScore {
					best, bestScore, changed = id, s, true
				}
			}
		}
	}
	return best
}

func (v *VectorIndex) searchLayer(query []float32, entries []string, ef, layer int) []string {
	visited := make(map[string]bool)
	queue := append([]string(nil), entries...)
	result := make([]string, 0, ef)
	for len(queue) > 0 {
		bestAt := 0
		for i := 1; i < len(queue); i++ {
			if v.score(query, v.nodes[queue[i]].Vector) > v.score(query, v.nodes[queue[bestAt]].Vector) {
				bestAt = i
			}
		}
		id := queue[bestAt]
		queue = append(queue[:bestAt], queue[bestAt+1:]...)
		if visited[id] {
			continue
		}
		visited[id] = true
		result = append(result, id)
		if len(result) >= ef {
			break
		}
		if n := v.nodes[id]; n != nil {
			for _, next := range n.Neighbors[layer] {
				if !visited[next] && v.nodes[next] != nil {
					queue = append(queue, next)
				}
			}
		}
	}
	return result
}

func (v *VectorIndex) closestIDs(query []float32, ids []string, limit int) []string {
	ids = append([]string(nil), ids...)
	sort.Slice(ids, func(i, j int) bool {
		si, sj := v.score(query, v.nodes[ids[i]].Vector), v.score(query, v.nodes[ids[j]].Vector)
		if si == sj {
			return ids[i] < ids[j]
		}
		return si > sj
	})
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids
}

func (v *VectorIndex) score(a, b []float32) float64 {
	var dot, aa, bb, sq float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		aa += x * x
		bb += y * y
		d := x - y
		sq += d * d
	}
	switch v.options.Metric {
	case MetricDot:
		return dot
	case MetricEuclidean:
		return 1 / (1 + math.Sqrt(sq))
	default:
		if aa == 0 || bb == 0 {
			return 0
		}
		return dot / (math.Sqrt(aa) * math.Sqrt(bb))
	}
}

func (v *VectorIndex) Len() int       { v.mu.RLock(); defer v.mu.RUnlock(); return len(v.nodes) }
func (v *VectorIndex) Dimension() int { return v.dimension }
func (v *VectorIndex) Metric() Metric { return v.options.Metric }

// Optimize rebuilds the graph deterministically, removing stale connections.
func (v *VectorIndex) Optimize() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	vectors := make(map[string][]float32, len(v.nodes))
	ids := make([]string, 0, len(v.nodes))
	for id, n := range v.nodes {
		ids = append(ids, id)
		vectors[id] = append([]float32(nil), n.Vector...)
	}
	sort.Strings(ids)
	v.nodes = make(map[string]*hnswNode)
	v.entryPoint, v.maxLevel = "", -1
	store := v.store
	v.store = nil
	for _, id := range ids {
		if err := v.addLockedNoPersist(id, vectors[id]); err != nil {
			v.store = store
			return err
		}
	}
	v.store = store
	return v.persistLocked()
}

func (v *VectorIndex) addLockedNoPersist(id string, vec []float32) error {
	// Reuse public insertion safely by temporarily releasing the lock is not
	// possible; build a fresh temporary index and copy its state instead.
	tmp := &VectorIndex{dimension: v.dimension, options: v.options, nodes: v.nodes, entryPoint: v.entryPoint, maxLevel: v.maxLevel}
	// Inline insertion through a small unlocked helper.
	return tmp.addUnlockedInto(v, id, vec)
}

func (tmp *VectorIndex) addUnlockedInto(dst *VectorIndex, id string, vec []float32) error {
	level := deterministicLevel(id, dst.options.M)
	node := &hnswNode{ID: id, Vector: append([]float32(nil), vec...), Level: level, Neighbors: map[int][]string{}}
	if len(dst.nodes) == 0 {
		dst.nodes[id] = node
		dst.entryPoint = id
		dst.maxLevel = level
		return nil
	}
	entry := dst.entryPoint
	for l := dst.maxLevel; l > level; l-- {
		entry = dst.greedyClosest(vec, entry, l)
	}
	upper := level
	if dst.maxLevel < upper {
		upper = dst.maxLevel
	}
	for l := upper; l >= 0; l-- {
		c := dst.searchLayer(vec, []string{entry}, dst.options.EFConstruction, l)
		node.Neighbors[l] = dst.closestIDs(vec, c, dst.options.M)
		if len(c) > 0 {
			entry = c[0]
		}
	}
	dst.nodes[id] = node
	for l, ns := range node.Neighbors {
		for _, nid := range ns {
			dst.connectLocked(nid, id, l, vec)
			dst.pruneLocked(nid, l)
		}
	}
	if level > dst.maxLevel {
		dst.entryPoint = id
		dst.maxLevel = level
	}
	return nil
}

func (v *VectorIndex) persistLocked() error {
	if v.store == nil {
		return nil
	}
	raw, err := json.Marshal(persistedIndex{Dimension: v.dimension, Options: v.options, EntryPoint: v.entryPoint, MaxLevel: v.maxLevel, Nodes: v.nodes})
	if err != nil {
		return err
	}
	return v.store.Put([]byte(v.options.PersistenceKey), raw)
}

func (v *VectorIndex) load() error {
	raw, err := v.store.Get([]byte(v.options.PersistenceKey))
	if err != nil {
		return err
	}
	var state persistedIndex
	if err := json.Unmarshal(raw, &state); err != nil {
		return err
	}
	if state.Dimension != v.dimension {
		return fmt.Errorf("persisted dimension %d does not match configured %d", state.Dimension, v.dimension)
	}
	v.options = normalizeOptions(state.Options)
	v.entryPoint = state.EntryPoint
	v.maxLevel = state.MaxLevel
	v.nodes = state.Nodes
	if v.nodes == nil {
		v.nodes = make(map[string]*hnswNode)
	}
	return nil
}

func deterministicLevel(id string, m int) int {
	h := fnv.New64a()
	_, _ = h.Write([]byte(id))
	x := h.Sum64()
	level := 0
	for level < 32 && x%uint64(m) == 0 {
		level++
		x /= uint64(m)
	}
	return level
}
func appendUnique(ids []string, id string) []string {
	for _, v := range ids {
		if v == id {
			return ids
		}
	}
	return append(ids, id)
}
func removeID(ids []string, id string) []string {
	out := ids[:0]
	for _, v := range ids {
		if v != id {
			out = append(out, v)
		}
	}
	return out
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func cosineSimilarityFloat32(a, b []float32) float64 {
	var dot, aa, bb float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		aa += x * x
		bb += y * y
	}
	if aa == 0 || bb == 0 {
		return 0
	}
	return dot / (math.Sqrt(aa) * math.Sqrt(bb))
}
