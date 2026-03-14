package cognitiveengine

import (
	"runtime"
	"sync"
	"time"

	"backend_server/internal/ebpf"
)

// SystemResourceSnapshot holds a point-in-time view of server resources.
// It merges data from the eBPF process_telemetry map with Go runtime stats.
type SystemResourceSnapshot struct {
	Timestamp        time.Time
	TotalCPUTimeNs   uint64
	TotalMemoryBytes uint64
	TotalNetTxBytes  uint64
	TotalNetRxBytes  uint64
	ContextSwitches  uint64
	PageFaults       uint64
	GoRoutines       int
	HeapAllocBytes   uint64
	NumGC            uint32
	// Derived / normalised fields (populated by collector after first snapshot)
	CPUPressure    float64
	MemoryPressure float64
}

// ResourceTelemetryCollector reads real resource data from the eBPF subsystem
// (if available) and from the Go runtime.  It is safe for concurrent use.
type ResourceTelemetryCollector struct {
	ebpfManager ebpf.ManagerInterface
	latest      *SystemResourceSnapshot
	mu          sync.RWMutex
}

// NewResourceTelemetryCollector creates a collector.  Pass nil for ebpfManager
// to run in Go-runtime-only mode.
func NewResourceTelemetryCollector(ebpfMgr ebpf.ManagerInterface) *ResourceTelemetryCollector {
	return &ResourceTelemetryCollector{
		ebpfManager: ebpfMgr,
	}
}

// Collect gathers a fresh snapshot and caches it.
func (r *ResourceTelemetryCollector) Collect() *SystemResourceSnapshot {
	snap := &SystemResourceSnapshot{
		Timestamp:  time.Now(),
		GoRoutines: runtime.NumGoroutine(),
	}

	// Go runtime memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	snap.HeapAllocBytes = memStats.HeapAlloc
	snap.NumGC = memStats.NumGC

	// eBPF process_telemetry map
	if r.ebpfManager != nil {
		if pm, err := r.ebpfManager.GetProcessMetrics(); err == nil {
			for _, ps := range pm {
				snap.TotalCPUTimeNs += ps.CPUTimeNs
				snap.TotalMemoryBytes += ps.MemoryBytes
				snap.TotalNetTxBytes += ps.NetTxBytes
				snap.TotalNetRxBytes += ps.NetRxBytes
				snap.ContextSwitches += uint64(ps.ContextSwitches)
				snap.PageFaults += uint64(ps.PageFaults)
			}
		}
	}

	// Derive pressure scores
	snap.CPUPressure = r.cpuPressure(snap)
	snap.MemoryPressure = r.memoryPressure(snap)

	r.mu.Lock()
	r.latest = snap
	r.mu.Unlock()

	return snap
}

// Latest returns the cached snapshot without collecting new data.
// Returns a zero-value snapshot if Collect has never been called.
func (r *ResourceTelemetryCollector) Latest() *SystemResourceSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.latest == nil {
		return &SystemResourceSnapshot{Timestamp: time.Now()}
	}
	return r.latest
}

// cpuPressure estimates CPU load as a 0.0–1.0 fraction.
// When eBPF data is available it uses accumulated CPU time;
// otherwise it falls back to a goroutine-count heuristic.
func (r *ResourceTelemetryCollector) cpuPressure(snap *SystemResourceSnapshot) float64 {
	if snap.TotalCPUTimeNs > 0 {
		// EBPFTelemetryInterval worth of CPU time across all monitored processes
		// normalised against GOMAXPROCS * interval nanoseconds.
		const intervalNs = float64(15 * time.Second)
		maxCPUNs := float64(runtime.GOMAXPROCS(0)) * intervalNs
		p := float64(snap.TotalCPUTimeNs) / maxCPUNs
		if p > 1.0 {
			p = 1.0
		}
		return p
	}
	// Goroutine-count heuristic: 500 goroutines ≈ 100% pressure
	p := float64(snap.GoRoutines) / 500.0
	if p > 1.0 {
		p = 1.0
	}
	return p
}

// memoryPressure returns heap + eBPF tracked memory as a fraction of an 8 GB ceiling.
func (r *ResourceTelemetryCollector) memoryPressure(snap *SystemResourceSnapshot) float64 {
	const ceiling = float64(8 * 1024 * 1024 * 1024)
	combined := float64(snap.TotalMemoryBytes + snap.HeapAllocBytes)
	p := combined / ceiling
	if p > 1.0 {
		p = 1.0
	}
	return p
}

// ResourceUtilizationForNode derives a 0.0–1.0 utilisation score for a given
// node by combining the node's task-processing rate with live memory pressure
// from the telemetry snapshot.
func (r *ResourceTelemetryCollector) ResourceUtilizationForNode(taskCount int64, successRate float64) float64 {
	snap := r.Latest()

	// Base utilisation: tasks processed relative to a 100-task-per-snapshot ceiling
	base := min64(1.0, float64(taskCount)/100.0) * successRate

	// Weight by observed memory pressure so that a RAM-starved host scores higher
	weighted := base*(1-snap.MemoryPressure*0.3) + snap.MemoryPressure*0.3
	if weighted > 1.0 {
		weighted = 1.0
	}
	return weighted
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
