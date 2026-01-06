package ebpf

import (
	"context"
	"encoding/binary"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetProcessMetrics(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Skipping eBPF test: requires root privileges")
	}

	mgr := NewManager()
	err := mgr.Initialize(context.Background(), &Config{})
	require.NoError(t, err)
	defer mgr.Shutdown()

	// Ensure process_telemetry map exists
	m := mgr.collection.Maps["process_telemetry"]
	require.NotNil(t, m)

	// Create a test entry for PID 1234
	pid := uint32(1234)
	var buf [4096]byte
	// syscall 1 count = 42
	binary.LittleEndian.PutUint64(buf[0:8], 0)
	binary.LittleEndian.PutUint64(buf[8:16], 42)
	// Set CPU time
	off := 8 * 400
	binary.LittleEndian.PutUint64(buf[off:off+8], 999999)

	// Put into map
	err = m.Put(pid, buf)
	require.NoError(t, err)

	metrics, err := mgr.GetProcessMetrics()
	require.NoError(t, err)
	ps, ok := metrics[pid]
	require.True(t, ok)
	require.Equal(t, uint64(42), ps.SyscallCount[1])
	require.Equal(t, uint64(999999), ps.CPUTimeNs)
}
