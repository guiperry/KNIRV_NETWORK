package dataengine

import (
	"testing"

	"backend_server/internal/ebpf"

	"github.com/stretchr/testify/require"
)

func TestDetectCPUSpike(t *testing.T) {
	ad := NewAnomalyDetector()
	fp := &LLMFingerprint{
		ModelName:    "gpt-test",
		AvgCPUTimeNs: 1000,
	}
	ad.AddFingerprint(fp)

	stats := &ebpf.ProcessStats{CPUTimeNs: 4000}
	stats.ModelName = "gpt-test"

	a := ad.Detect(stats)
	require.NotNil(t, a)
	require.Equal(t, "cpu_spike", a.Type)
}

func TestDetectSyscallAnomaly(t *testing.T) {
	ad := NewAnomalyDetector()
	// fingerprint: syscall 1 has 100% of calls
	fp := &LLMFingerprint{
		ModelName:           "gpt-a",
		SyscallDistribution: map[int]float64{1: 1.0},
	}
	ad.AddFingerprint(fp)

	var stats ebpf.ProcessStats
	stats.ModelName = "gpt-a"
	stats.SyscallCount[5] = 1000 // Unexpected syscalls

	a := ad.Detect(&stats)
	require.NotNil(t, a)
	require.Equal(t, "syscall_anomaly", a.Type)
}
