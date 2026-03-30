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
	NodeID     string
	EventType  string // "syscall_deny", "lsm_block", "xdp_drop", "high_page_faults"
	SyscallNr  uint32
	ProcessPID uint32
	Details    string
	Timestamp  time.Time
	Severity   string // "info", "warning", "critical"
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

type XDPOffloadManager struct {
	mu       sync.RWMutex
	filters  map[string]*XDPFilter
	eventBus *EventBus
	running  bool
	cancel   context.CancelFunc
}

type XDPFilter struct {
	ID          string
	NodeID      string
	Priority    int
	Action      string // "drop", "pass", "redirect"
	MatchPolicy map[string]interface{}
	PacketCount uint64
	ByteCount   uint64
	Enabled     bool
	CreatedAt   time.Time
}

func NewXDPOffloadManager(eventBus *EventBus) *XDPOffloadManager {
	return &XDPOffloadManager{
		filters:  make(map[string]*XDPFilter),
		eventBus: eventBus,
	}
}

func (xom *XDPOffloadManager) AddFilter(filter *XDPFilter) error {
	xom.mu.Lock()
	defer xom.mu.Unlock()

	if filter.ID == "" {
		filter.ID = fmt.Sprintf("xdp_%d", time.Now().UnixNano())
	}
	filter.CreatedAt = time.Now()
	filter.Enabled = true

	xom.filters[filter.ID] = filter

	if xom.eventBus != nil {
		xom.eventBus.Publish(EngineEvent{
			Type:    EventEBPFSecurityAlert,
			Source:  "xdp_offload_manager",
			Payload: map[string]interface{}{"action": "filter_added", "filter_id": filter.ID},
		})
	}

	log.Printf("XDPOffloadManager: added filter %s for node %s with action %s",
		filter.ID, filter.NodeID, filter.Action)
	return nil
}

func (xom *XDPOffloadManager) RemoveFilter(filterID string) error {
	xom.mu.Lock()
	defer xom.mu.Unlock()

	if _, exists := xom.filters[filterID]; !exists {
		return fmt.Errorf("filter %s not found", filterID)
	}

	delete(xom.filters, filterID)

	if xom.eventBus != nil {
		xom.eventBus.Publish(EngineEvent{
			Type:    EventEBPFSecurityAlert,
			Source:  "xdp_offload_manager",
			Payload: map[string]interface{}{"action": "filter_removed", "filter_id": filterID},
		})
	}

	return nil
}

func (xom *XDPOffloadManager) GetFilter(filterID string) (*XDPFilter, bool) {
	xom.mu.RLock()
	defer xom.mu.RUnlock()
	f, exists := xom.filters[filterID]
	return f, exists
}

func (xom *XDPOffloadManager) ListFilters() []*XDPFilter {
	xom.mu.RLock()
	defer xom.mu.RUnlock()

	filters := make([]*XDPFilter, 0, len(xom.filters))
	for _, f := range xom.filters {
		filters = append(filters, f)
	}
	return filters
}

func (xom *XDPOffloadManager) UpdateFilterStats(filterID string, packetCount, byteCount uint64) {
	xom.mu.Lock()
	defer xom.mu.Unlock()

	if filter, exists := xom.filters[filterID]; exists {
		filter.PacketCount = packetCount
		filter.ByteCount = byteCount
	}
}

func (xom *XDPOffloadManager) EnableFilter(filterID string) error {
	xom.mu.Lock()
	defer xom.mu.Unlock()

	if filter, exists := xom.filters[filterID]; exists {
		filter.Enabled = true
		return nil
	}
	return fmt.Errorf("filter %s not found", filterID)
}

func (xom *XDPOffloadManager) DisableFilter(filterID string) error {
	xom.mu.Lock()
	defer xom.mu.Unlock()

	if filter, exists := xom.filters[filterID]; exists {
		filter.Enabled = false
		return nil
	}
	return fmt.Errorf("filter %s not found", filterID)
}

type ResourceQuotaManager struct {
	mu          sync.RWMutex
	quotas      map[string]*ResourceQuota
	eventBus    *EventBus
	enforcement bool
}

type ResourceQuota struct {
	ID         string
	NodeID     string
	CPUQuota   float64 // CPU cores
	MemoryMB   uint64  // Memory in MB
	NetBPS     uint64  // Network bytes per second
	DiskIOPS   uint64  // Disk IO operations per second
	Active     bool
	CreatedAt  time.Time
	ModifiedAt time.Time
}

func NewResourceQuotaManager(eventBus *EventBus) *ResourceQuotaManager {
	return &ResourceQuotaManager{
		quotas:      make(map[string]*ResourceQuota),
		eventBus:    eventBus,
		enforcement: true,
	}
}

func (rqm *ResourceQuotaManager) SetQuota(quota *ResourceQuota) error {
	rqm.mu.Lock()
	defer rqm.mu.Unlock()

	if quota.ID == "" {
		quota.ID = fmt.Sprintf("quota_%s_%d", quota.NodeID, time.Now().UnixNano())
	}
	quota.Active = true
	quota.ModifiedAt = time.Now()

	if quota.CreatedAt.IsZero() {
		quota.CreatedAt = time.Now()
	}

	rqm.quotas[quota.ID] = quota

	if rqm.eventBus != nil {
		rqm.eventBus.Publish(EngineEvent{
			Type:    EventResourcePressure,
			Source:  "resource_quota_manager",
			Payload: map[string]interface{}{"action": "quota_set", "node_id": quota.NodeID},
		})
	}

	log.Printf("ResourceQuotaManager: set quota %s for node %s (CPU=%.2f, Mem=%dMB)",
		quota.ID, quota.NodeID, quota.CPUQuota, quota.MemoryMB)
	return nil
}

func (rqm *ResourceQuotaManager) GetQuota(nodeID string) (*ResourceQuota, bool) {
	rqm.mu.RLock()
	defer rqm.mu.RUnlock()

	for _, quota := range rqm.quotas {
		if quota.NodeID == nodeID && quota.Active {
			return quota, true
		}
	}
	return nil, false
}

func (rqm *ResourceQuotaManager) RemoveQuota(quotaID string) error {
	rqm.mu.Lock()
	defer rqm.mu.Unlock()

	if quota, exists := rqm.quotas[quotaID]; exists {
		quota.Active = false
		return nil
	}
	return fmt.Errorf("quota %s not found", quotaID)
}

func (rqm *ResourceQuotaManager) ListQuotas() []*ResourceQuota {
	rqm.mu.RLock()
	defer rqm.mu.RUnlock()

	quotas := make([]*ResourceQuota, 0, len(rqm.quotas))
	for _, q := range rqm.quotas {
		if q.Active {
			quotas = append(quotas, q)
		}
	}
	return quotas
}

func (rqm *ResourceQuotaManager) SetEnforcement(enabled bool) {
	rqm.mu.Lock()
	defer rqm.mu.Unlock()
	rqm.enforcement = enabled
}

func (rqm *ResourceQuotaManager) IsEnforcementEnabled() bool {
	rqm.mu.RLock()
	defer rqm.mu.RUnlock()
	return rqm.enforcement
}

type ControlPlane struct {
	mu           sync.RWMutex
	commands     chan ControlCommand
	eventBus     *EventBus
	ebpfBridge   *EBPFBridge
	xdpManager   *XDPOffloadManager
	quotaManager *ResourceQuotaManager
	running      bool
}

type ControlCommand struct {
	Type    string // "panic_isolate", "remove_filter", "set_quota", "get_stats"
	Target  string // node ID or filter ID
	Payload interface{}
	Result  chan<- error
}

func NewControlPlane(eventBus *EventBus, ebpfBridge *EBPFBridge) *ControlPlane {
	return &ControlPlane{
		commands:     make(chan ControlCommand, 100),
		eventBus:     eventBus,
		ebpfBridge:   ebpfBridge,
		xdpManager:   NewXDPOffloadManager(eventBus),
		quotaManager: NewResourceQuotaManager(eventBus),
	}
}

func (cp *ControlPlane) Start() {
	cp.mu.Lock()
	if cp.running {
		cp.mu.Unlock()
		return
	}
	cp.running = true
	cp.mu.Unlock()

	go cp.commandLoop()
	log.Println("ControlPlane: started")
}

func (cp *ControlPlane) Stop() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if !cp.running {
		return
	}
	cp.running = false
	close(cp.commands)
	log.Println("ControlPlane: stopped")
}

func (cp *ControlPlane) Execute(cmd ControlCommand) error {
	resultCh := make(chan error, 1)
	cmd.Result = resultCh
	cp.commands <- cmd
	return <-resultCh
}

func (cp *ControlPlane) commandLoop() {
	for cmd := range cp.commands {
		var err error
		switch cmd.Type {
		case "panic_isolate":
			err = cp.handlePanicIsolation(cmd.Target)
		case "remove_filter":
			err = cp.xdpManager.RemoveFilter(cmd.Target)
		case "set_quota":
			if quota, ok := cmd.Payload.(*ResourceQuota); ok {
				err = cp.quotaManager.SetQuota(quota)
			} else {
				err = fmt.Errorf("invalid quota payload")
			}
		case "get_stats":
			err = nil
		default:
			err = fmt.Errorf("unknown command type: %s", cmd.Type)
		}
		if cmd.Result != nil {
			cmd.Result <- err
		}
	}
}

func (cp *ControlPlane) handlePanicIsolation(nodeID string) error {
	if cp.ebpfBridge != nil {
		if err := cp.ebpfBridge.TriggerPanicIsolation(nodeID); err != nil {
			return err
		}
	}

	log.Printf("ControlPlane: panic isolation executed for node %s", nodeID)
	return nil
}

func (cp *ControlPlane) GetXDPManger() *XDPOffloadManager {
	return cp.xdpManager
}

func (cp *ControlPlane) GetQuotaManager() *ResourceQuotaManager {
	return cp.quotaManager
}

func (cp *ControlPlane) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"xdp_filters_active": len(cp.xdpManager.ListFilters()),
		"quotas_active":      len(cp.quotaManager.ListQuotas()),
		"enforcement":        cp.quotaManager.IsEnforcementEnabled(),
	}
}

func (b *EBPFBridge) GetControlPlane() *ControlPlane {
	cp := NewControlPlane(b.eventBus, b)
	cp.Start()
	return cp
}
