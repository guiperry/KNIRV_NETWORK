package transformer

import (
	"encoding/binary"
	"fmt"
	"sync"

	"knirvhasher/pkg/hashing/core"
)

// FallbackStrategy determines how HardwareRouter handles hardware failures.
type FallbackStrategy int

const (
	FallbackSoftware FallbackStrategy = iota
	FallbackError
	FallbackMixed
)

// HardwareRouter routes projections through hardware (hashMethod.ComputeBatch)
// when available, falling back to software seed-to-float computation when needed.
type HardwareRouter struct {
	hashMethod core.HashMethod
	fallback   FallbackStrategy
	cache      *ProjectionCache
	once       sync.Once
}

// NewHardwareRouter creates a HardwareRouter with the given hash method and fallback strategy.
func NewHardwareRouter(method core.HashMethod, strategy FallbackStrategy) *HardwareRouter {
	r := &HardwareRouter{
		hashMethod: method,
		fallback:   strategy,
		cache:      NewProjectionCache(),
	}
	r.once.Do(func() {})
	return r
}

// Project computes a single projection: input vector projected through seed weights.
// It attempts hardware batch hashing first, then falls back to software.
func (r *HardwareRouter) Project(input []float32, seeds [][32]byte) ([]float32, error) {
	if len(seeds) == 0 {
		return []float32{}, nil
	}
	if r.hashMethod != nil && r.hashMethod.IsAvailable() {
		hashes, err := r.computeBatchHW(input, seeds)
		if err == nil {
			return hashesToFloats(hashes), nil
		}
		if r.fallback == FallbackError {
			return nil, fmt.Errorf("hardware projection failed: %w", err)
		}
	}
	return ProjectSeeds(input, seeds, "hash"), nil
}

// ProjectBatch projects multiple input vectors through their respective seed sets.
func (r *HardwareRouter) ProjectBatch(inputs [][]float32, seedSets [][][32]byte) ([][]float32, error) {
	if len(inputs) != len(seedSets) {
		return nil, fmt.Errorf("ProjectBatch: inputs and seedSets length mismatch (%d vs %d)", len(inputs), len(seedSets))
	}
	results := make([][]float32, len(inputs))
	for i := range inputs {
		out, err := r.Project(inputs[i], seedSets[i])
		if err != nil {
			return nil, fmt.Errorf("ProjectBatch[%d]: %w", i, err)
		}
		results[i] = out
	}
	return results, nil
}

// HashToVocab projects hidden state to vocabulary logits.
// Uses hardware ComputeBatch when available, otherwise software path.
func (r *HardwareRouter) HashToVocab(hidden []float32, outputSeed [32]byte, vocabSize int) []float32 {
	if r.hashMethod != nil && r.hashMethod.IsAvailable() {
		inputs := make([][]byte, vocabSize)
		for i := 0; i < vocabSize; i++ {
			data := make([]byte, 36)
			binary.BigEndian.PutUint32(data[0:4], uint32(i))
			copy(data[4:], outputSeed[:])
			inputs[i] = data
		}
		hashes, err := r.hashMethod.ComputeBatch(inputs)
		if err == nil {
			scores := make([]float32, vocabSize)
			base := float32(0)
			for _, v := range hidden {
				base += v
			}
			for i := 0; i < vocabSize; i++ {
				scores[i] = base * hashesToFloats(hashes)[i]
			}
			return scores
		}
	}
	return HashToVocab(hidden, outputSeed, vocabSize)
}

func (r *HardwareRouter) computeBatchHW(input []float32, seeds [][32]byte) ([][32]byte, error) {
	if r.hashMethod == nil || !r.hashMethod.IsAvailable() {
		return nil, fmt.Errorf("hardware not available")
	}
	inputs := make([][]byte, len(seeds))
	inputBytes := float32SliceToBytes(input)
	for i, seed := range seeds {
		combined := make([]byte, len(inputBytes)+32)
		copy(combined, inputBytes)
		copy(combined[len(inputBytes):], seed[:])
		inputs[i] = combined
	}
	return r.hashMethod.ComputeBatch(inputs)
}

func float32SliceToBytes(vals []float32) []byte {
	bytes := make([]byte, len(vals)*4)
	for i, v := range vals {
		binary.BigEndian.PutUint32(bytes[i*4:], float32ToUint32(v))
	}
	return bytes
}

func float32ToUint32(v float32) uint32 {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return uint32(v * float32(^uint32(0)))
}

func hashesToFloats(hashes [][32]byte) []float32 {
	out := make([]float32, len(hashes))
	for i, h := range hashes {
		out[i] = SeedToFloat(h)
	}
	return out
}

// ProjectionCache caches recent (inputHash, seedHash) -> float32[] mappings.
type ProjectionCache struct {
	mu      sync.RWMutex
	entries map[string][]float32
	maxSize int
}

// NewProjectionCache creates a new ProjectionCache with default max size.
func NewProjectionCache() *ProjectionCache {
	return &ProjectionCache{
		entries: make(map[string][]float32),
		maxSize: 10000,
	}
}

// Get retrieves a cached projection result.
func (c *ProjectionCache) Get(key string) ([]float32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.entries[key]
	return val, ok
}

// Put stores a projection result in the cache.
func (c *ProjectionCache) Put(key string, val []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxSize {
		return
	}
	c.entries[key] = val
}

// Size returns the current number of cached entries.
func (c *ProjectionCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
