package data_engine

import (
	"encoding/json"
	"sync"

	"backend_server/internal/ebpf"
)

// LLMFingerprint holds baseline stats for a model
type LLMFingerprint struct {
	ModelName           string
	AvgCPUTimeNs        uint64
	AvgMemoryBytes      uint64
	SyscallDistribution map[int]float64
}

// FingerprintStore stores fingerprints (in-memory)
type FingerprintStore struct {
	mu    sync.RWMutex
	store map[string]*LLMFingerprint
}

func NewFingerprintStore() *FingerprintStore {
	return &FingerprintStore{store: make(map[string]*LLMFingerprint)}
}

func (fs *FingerprintStore) Save(fp *LLMFingerprint) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.store[fp.ModelName] = fp
}

func (fs *FingerprintStore) Get(name string) *LLMFingerprint {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.store[name]
}

// BuildFingerprint builds an LLMFingerprint from historical samples
func BuildFingerprint(modelName string, samples []ebpf.ProcessStats) (*LLMFingerprint, error) {
	if len(samples) == 0 {
		return nil, nil
	}

	fp := &LLMFingerprint{
		ModelName:           modelName,
		SyscallDistribution: make(map[int]float64),
	}

	var totalCPU uint64
	var totalMem uint64

	for _, s := range samples {
		totalCPU += s.CPUTimeNs
		totalMem += s.MemoryBytes

		var sampleTotal uint64
		for _, c := range s.SyscallCount {
			sampleTotal += c
		}
		if sampleTotal == 0 {
			continue
		}
		for id, c := range s.SyscallCount {
			fp.SyscallDistribution[id] += float64(c) / float64(sampleTotal)
		}
	}

	n := float64(len(samples))
	fp.AvgCPUTimeNs = totalCPU / uint64(n)
	fp.AvgMemoryBytes = totalMem / uint64(n)

	for id := range fp.SyscallDistribution {
		fp.SyscallDistribution[id] = fp.SyscallDistribution[id] / n
	}

	return fp, nil
}

// helper: JSON serialize
func (fp *LLMFingerprint) Marshal() string {
	b, _ := json.Marshal(fp)
	return string(b)
}
