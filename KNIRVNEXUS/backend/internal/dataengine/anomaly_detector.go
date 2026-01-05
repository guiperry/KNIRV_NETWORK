package dataengine

import (
	"math"
	"sync"

	"backend_server/internal/ebpf"
)

// Anomaly represents a detected anomaly
type Anomaly struct {
	Type        string
	Severity    int
	Description string
	Metrics     ebpf.ProcessStats
}

// AnomalyDetector analyzes ProcessStats against fingerprints
type AnomalyDetector struct {
	mu           sync.RWMutex
	fingerprints map[string]*LLMFingerprint
}

func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		fingerprints: make(map[string]*LLMFingerprint),
	}
}

func (ad *AnomalyDetector) AddFingerprint(fp *LLMFingerprint) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	ad.fingerprints[fp.ModelName] = fp
}

func (ad *AnomalyDetector) getFingerprint(modelName string) *LLMFingerprint {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	return ad.fingerprints[modelName]
}

func (ad *AnomalyDetector) Detect(stats *ebpf.ProcessStats) *Anomaly {
	fp := ad.getFingerprint(stats.ModelName)
	if fp == nil {
		return nil
	}

	// CPU spike detection
	if stats.CPUTimeNs > fp.AvgCPUTimeNs*3 {
		return &Anomaly{
			Type:        "cpu_spike",
			Severity:    7,
			Description: "CPU spike detected",
			Metrics:     *stats,
		}
	}

	// Syscall distribution anomaly using chi-squared
	chi := ad.chiSquaredTest(stats, fp)
	if chi > 20.0 {
		return &Anomaly{
			Type:        "syscall_anomaly",
			Severity:    8,
			Description: "Unusual syscall distribution",
			Metrics:     *stats,
		}
	}

	return nil
}

// chiSquaredTest compares the syscall distribution in stats against fingerprint
func (ad *AnomalyDetector) chiSquaredTest(stats *ebpf.ProcessStats, fp *LLMFingerprint) float64 {
	// Compute total counts
	total := uint64(0)
	for _, c := range stats.SyscallCount {
		total += c
	}
	if total == 0 {
		return 0.0
	}

	chi := 0.0
	for i, expectedRatio := range fp.SyscallDistribution {
		observed := float64(stats.SyscallCount[i])
		expected := expectedRatio * float64(total)
		if expected == 0 {
			// If expected zero but observed > 0, add large penalty
			if observed > 0 {
				chi += math.Pow(observed-0.1, 2) / 0.1
			}
			continue
		}
		chi += math.Pow(observed-expected, 2) / expected
	}
	return chi
}
