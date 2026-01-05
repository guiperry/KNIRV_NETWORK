package dataengine

import (
	"context"
	"testing"
	"time"

	"backend_server/internal/ebpf"

	"github.com/stretchr/testify/require"
)

type fakeMgr struct {
	metrics map[uint32]*ebpf.ProcessStats
}

func (f *fakeMgr) GetProcessMetrics() (map[uint32]*ebpf.ProcessStats, error) {
	return f.metrics, nil
}

func TestTelemetryCollectorStart(t *testing.T) {
	mgr := &fakeMgr{metrics: make(map[uint32]*ebpf.ProcessStats)}
	ad := NewAnomalyDetector()
	store := NewFingerprintStore()
	collector := NewTelemetryCollector(mgr, store, ad)

	// no panic on start
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	err := collector.Start(ctx)
	require.NoError(t, err)

	// feed a spike metric
	mgr.metrics[100] = &ebpf.ProcessStats{CPUTimeNs: 100000}
	mgr.metrics[100].ModelName = "gpt-test"

	// add fingerprint so anomaly can be detected
	fp := &LLMFingerprint{ModelName: "gpt-test", AvgCPUTimeNs: 10}
	ad.AddFingerprint(fp)

	// wait for collector tick to run at least once
	time.Sleep(1100 * time.Millisecond)
}
