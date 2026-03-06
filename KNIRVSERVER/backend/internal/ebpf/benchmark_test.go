package ebpf

import (
	"context"
	"os"
	"testing"
)

func setupManager(tb testing.TB) *Manager {
	mgr := NewManager()
	if err := mgr.Initialize(context.TODO(), &Config{}); err != nil {
		// in benchmarks, fail fast
		tb.Fatalf("init manager: %v", err)
	}
	return mgr
}

func startTestProcess() int {
	f, _ := os.CreateTemp("/tmp", "ebpf-test-*")
	defer f.Close()
	return os.Getpid() // reuse current PID for speed
}

func BenchmarkSyscallMonitoring(b *testing.B) {
	mgr := setupManager(b)
	defer mgr.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Perform syscall-heavy operation
		os.Stat("/tmp/test")
	}

	// Report minimal overhead (placeholder measurement)
}

func BenchmarkVirtualContainerCreation(b *testing.B) {
	mgr := setupManager(b)
	defer mgr.Shutdown()

	vcm := NewVirtualContainerManager(mgr)
	if err := vcm.InitializeVirtualContainers(); err != nil {
		b.Fatalf("init virtual container manager: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pid := startTestProcess()
		vcm.CreateVirtualContainer(uint32(pid), "/tmp/test")
	}
}
