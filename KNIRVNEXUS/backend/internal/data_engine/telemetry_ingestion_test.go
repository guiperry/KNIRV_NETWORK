package data_engine

import (
	"context"
	"sync"
	"testing"
	"time"

	"backend_server/internal/ebpf"

	"github.com/stretchr/testify/require"
)

type fakeMgr struct {
	mu      sync.Mutex
	metrics map[uint32]*ebpf.ProcessStats
}

func (f *fakeMgr) GetProcessMetrics() (map[uint32]*ebpf.ProcessStats, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	copiedMetrics := make(map[uint32]*ebpf.ProcessStats)
	for k, v := range f.metrics {
		copiedMetrics[k] = v
	}
	return copiedMetrics, nil
}

func TestTelemetryCollectorStart(t *testing.T) {
	mgr := &fakeMgr{metrics: make(map[uint32]*ebpf.ProcessStats)}
	ad := NewAnomalyDetector()
	store := NewFingerprintStore()
	collector := NewTelemetryCollector(mgr, store, ad)

	// feed a spike metric
	mgr.mu.Lock()
	mgr.metrics[100] = &ebpf.ProcessStats{CPUTimeNs: 100000}
	mgr.metrics[100].ModelName = "gpt-test"
	mgr.mu.Unlock()

	// add fingerprint so anomaly can be detected
	fp := &LLMFingerprint{ModelName: "gpt-test", AvgCPUTimeNs: 10}
	ad.AddFingerprint(fp)

	// no panic on start
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	err := collector.Start(ctx)
	require.NoError(t, err)

	// wait for collector tick to run at least once
	time.Sleep(1100 * time.Millisecond)
}
