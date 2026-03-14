package cognitiveengine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/ebpf"
)

// SecurityEventFeedback represents a kernel-level security event received from
// the eBPF subsystem (LSM denials, XDP drops, syscall anomalies).
type SecurityEventFeedback struct {
	NodeID    string
	EventType string // "syscall_deny", "lsm_block", "xdp_drop", "high_page_faults"
	SyscallNr uint32
	ProcessPID uint32
	Details   string
	Timestamp time.Time
	Severity  string // "info", "warning", "critical"
}

// EBPFBridge connects the Cognitive Engine to the eBPF subsystem.
// It polls the process_telemetry eBPF map for resource data, analyses the
// data for anomalies, publishes events to the engine's EventBus, and can
// trigger kernel-level panic isolation for compromised nodes.
type EBPFBridge struct {
	manager    ebpf.ManagerInterface
	eventBus   *EventBus
	telemetry  *ResourceTelemetryCollector
	panicNodes map[string]time.Time // nodeID → isolation timestamp
	mu         sync.RWMutex
	running    bool
	cancel     context.CancelFunc
}

// NewEBPFBridge creates a bridge.  Pass nil for manager to run in no-op mode
// (telemetry will fall back to Go runtime stats only).
func NewEBPFBridge(mgr ebpf.ManagerInterface, bus *EventBus) *EBPFBridge {
	return &EBPFBridge{
		manager:    mgr,
		eventBus:   bus,
		telemetry:  NewResourceTelemetryCollector(mgr),
		panicNodes: make(map[string]time.Time),
	}
}

// Start launches the periodic telemetry collection loop.
// Calling Start on an already-running bridge is a no-op.
func (b *EBPFBridge) Start(ctx context.Context, interval time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running {
		return
	}
	bridgeCtx, cancel := context.WithCancel(ctx)
	b.cancel = cancel
	b.running = true
	go b.telemetryLoop(bridgeCtx, interval)
	log.Println("EBPFBridge: telemetry polling started")
}

// Stop halts the polling loop and waits for it to exit.
func (b *EBPFBridge) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.running {
		return
	}
	b.cancel()
	b.running = false
	log.Println("EBPFBridge: stopped")
}

func (b *EBPFBridge) telemetryLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap := b.telemetry.Collect()
			b.analyzeSecurityTelemetry(snap)
		}
	}
}

// analyzeSecurityTelemetry inspects a resource snapshot for anomalous patterns
// and publishes appropriate events to the EventBus.
func (b *EBPFBridge) analyzeSecurityTelemetry(snap *SystemResourceSnapshot) {
	if snap == nil {
		return
	}

	// Resource pressure warning/critical events
	if snap.CPUPressure > 0.85 || snap.MemoryPressure > 0.85 {
		b.eventBus.Publish(EngineEvent{
			Type:   EventResourcePressure,
			Source: "ebpf_bridge",
			Payload: map[string]float64{
				"cpu_pressure": snap.CPUPressure,
				"mem_pressure": snap.MemoryPressure,
			},
			Timestamp: time.Now(),
		})
		log.Printf("EBPFBridge: resource pressure (cpu=%.2f mem=%.2f)", snap.CPUPressure, snap.MemoryPressure)
	}

	// High page-fault rate is a strong signal of memory thrashing or exploit activity
	if snap.PageFaults > 10_000 {
		b.eventBus.Publish(EngineEvent{
			Type:   EventEBPFSecurityAlert,
			Source: "ebpf_bridge",
			Payload: SecurityEventFeedback{
				EventType: "high_page_faults",
				Details:   fmt.Sprintf("page_faults=%d", snap.PageFaults),
				Timestamp: time.Now(),
				Severity:  "warning",
			},
			Timestamp: time.Now(),
		})
	}

	// Extremely high context-switch rate may indicate a scheduling anomaly / thrash
	if snap.ContextSwitches > 100_000 {
		b.eventBus.Publish(EngineEvent{
			Type:   EventEBPFSecurityAlert,
			Source: "ebpf_bridge",
			Payload: SecurityEventFeedback{
				EventType: "context_switch_storm",
				Details:   fmt.Sprintf("context_switches=%d", snap.ContextSwitches),
				Timestamp: time.Now(),
				Severity:  "warning",
			},
			Timestamp: time.Now(),
		})
	}
}

// InjectSecurityFeedback feeds a pre-parsed eBPF security event (e.g., from an
// LSM audit-log watcher) into the Cognitive Engine learning loop via the bus.
func (b *EBPFBridge) InjectSecurityFeedback(event SecurityEventFeedback) {
	b.eventBus.Publish(EngineEvent{
		Type:      EventEBPFSecurityAlert,
		Source:    "ebpf_bridge",
		Payload:   event,
		Timestamp: time.Now(),
	})
	if event.Severity == "critical" {
		log.Printf("EBPFBridge: critical security event from node %s: %s – %s",
			event.NodeID, event.EventType, event.Details)
	}
}

// TriggerPanicIsolation records a kernel-level isolation request for a node.
// The calling code (e.g., the guardrail "kernel_isolation" remediator) should
// also invoke the eBPF VirtualContainerManager to enforce namespace teardown.
// Returns an error if the node is already isolated.
func (b *EBPFBridge) TriggerPanicIsolation(nodeID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, already := b.panicNodes[nodeID]; already {
		return fmt.Errorf("node %s is already under panic isolation", nodeID)
	}

	b.panicNodes[nodeID] = time.Now()
	log.Printf("EBPFBridge: PANIC ISOLATION recorded for node %s at %s", nodeID, time.Now())

	b.eventBus.Publish(EngineEvent{
		Type:   EventGuardrailViolation,
		Source: "ebpf_bridge",
		Payload: map[string]string{
			"action":  "kernel_isolation",
			"node_id": nodeID,
		},
		Timestamp: time.Now(),
	})
	return nil
}

// IsNodeIsolated reports whether a node is currently under panic isolation.
func (b *EBPFBridge) IsNodeIsolated(nodeID string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, isolated := b.panicNodes[nodeID]
	return isolated
}

// LatestTelemetry returns the most recently collected resource snapshot.
func (b *EBPFBridge) LatestTelemetry() *SystemResourceSnapshot {
	return b.telemetry.Latest()
}
