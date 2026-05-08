// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package wasm

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// ResolutionResult captures the outcome of a resolution.wasm invocation.
type ResolutionResult struct {
	BadgeID      string    `json:"badge_id"`
	DVEID        string    `json:"dve_id"`
	NodeID       string    `json:"node_id"`
	Tag          string    `json:"tag"`
	WASMPath     string    `json:"wasm_path"`
	Resolved     bool      `json:"resolved"`
	ErrorClassID uint32    `json:"error_class_id"`
	Error        string    `json:"error,omitempty"`
	Duration     time.Duration `json:"duration_ms"`
	Timestamp    time.Time `json:"timestamp"`
}

// RemediationSink receives resolution results and dispatches them
// into the GuardrailEngine remediation pipeline.
type RemediationSink interface {
	// DispatchRemediation is called when a resolution.wasm returns
	// resolved=true.  The sink should trigger the appropriate
	// remediation action (e.g. restart_service, apply_patch).
	DispatchRemediation(ctx context.Context, result *ResolutionResult) error
}

// ResolutionService implements Option B of the WASM Integration
// Investigation: eBPF-triggered error resolution via resolution.wasm.
//
// It listens for "Resolution Signals" forwarded by the eBPF EventCollector,
// loads the appropriate resolution.wasm for the DVE's Badge, executes
// resolveError() and ErrorClass(), and pipes results into the
// GuardrailEngine's remediation pipeline.
//
// The service maintains a pool of WazeroGate instances per resolution.wasm
// module to avoid re-compilation overhead under high event rates.
type ResolutionService struct {
	mapper       *BadgeWASMMapper
	sink         RemediationSink
	pools        map[string]*WazeroPool // wasmPath → pool
	poolsMu      sync.RWMutex
	poolSize     int
	signalChan   chan *ResolutionSignal
	stopChan     chan struct{}
	workerWg     sync.WaitGroup
	stats        *ResolutionStats
	statsMu      sync.RWMutex
}

// ResolutionSignal is produced by the eBPF EventCollector when a
// kernel-level anomaly matching a Badge's resolution ontology is detected.
type ResolutionSignal struct {
	DVEID     string            `json:"dve_id"`
	NodeID    string            `json:"node_id"`
	BadgeID   string            `json:"badge_id"`
	Tag       string            `json:"tag"` // ontology tag that triggered
	ErrorType string            `json:"error_type"`
	SyscallID uint32            `json:"syscall_id"`
	PID       uint32            `json:"pid"`
	Context   map[string]string `json:"context,omitempty"`
}

// ResolutionStats tracks resolution service metrics.
type ResolutionStats struct {
	TotalSignals      int64 `json:"total_signals"`
	TotalResolutions  int64 `json:"total_resolutions"`
	SuccessfulResolve int64 `json:"successful_resolve"`
	FailedResolve     int64 `json:"failed_resolve"`
	WASMErrors        int64 `json:"wasm_errors"`
	DroppedSignals    int64 `json:"dropped_signals"`
}

// NewResolutionService creates a resolution worker pool.
// poolSize controls how many concurrent WazeroGate instances per module.
// signalBuffer controls the capacity of the incoming signal channel.
func NewResolutionService(
	mapper *BadgeWASMMapper,
	sink RemediationSink,
	poolSize int,
	signalBuffer int,
) *ResolutionService {
	if poolSize < 1 {
		poolSize = 4
	}
	if signalBuffer < 1 {
		signalBuffer = 256
	}
	return &ResolutionService{
		mapper:     mapper,
		sink:       sink,
		pools:      make(map[string]*WazeroPool),
		poolSize:   poolSize,
		signalChan: make(chan *ResolutionSignal, signalBuffer),
		stopChan:   make(chan struct{}),
		stats:      &ResolutionStats{},
	}
}

// Start launches the background resolution workers.
func (rs *ResolutionService) Start(ctx context.Context, numWorkers int) {
	if numWorkers < 1 {
		numWorkers = 2
	}
	for i := 0; i < numWorkers; i++ {
		rs.workerWg.Add(1)
		go rs.worker(ctx, i)
	}
	log.Printf("[resolution-service] started %d workers", numWorkers)
}

// Stop gracefully shuts down the resolution workers.
func (rs *ResolutionService) Stop() {
	close(rs.stopChan)
	rs.workerWg.Wait()

	rs.poolsMu.Lock()
	defer rs.poolsMu.Unlock()
	for _, pool := range rs.pools {
		pool.CloseAll()
	}
	rs.pools = make(map[string]*WazeroPool)
	log.Println("[resolution-service] stopped")
}

// Submit enqueues a resolution signal for asynchronous processing.
// Non-blocking: returns false if the signal buffer is full (signal dropped).
func (rs *ResolutionService) Submit(signal *ResolutionSignal) bool {
	rs.statsMu.Lock()
	rs.stats.TotalSignals++
	rs.statsMu.Unlock()

	select {
	case rs.signalChan <- signal:
		return true
	default:
		rs.statsMu.Lock()
		rs.stats.DroppedSignals++
		rs.statsMu.Unlock()
		return false
	}
}

// Resolve synchronously executes the resolution pipeline for a signal.
// Returns the resolution result.
func (rs *ResolutionService) Resolve(ctx context.Context, signal *ResolutionSignal) (*ResolutionResult, error) {
	start := time.Now()
	result := &ResolutionResult{
		BadgeID:   signal.BadgeID,
		DVEID:     signal.DVEID,
		NodeID:    signal.NodeID,
		Tag:       signal.Tag,
		Timestamp: time.Now(),
	}

	// Look up the .wasm for this badge + tag.
	wasmPath, wasmType, err := rs.mapper.LookupWASM(signal.BadgeID, signal.Tag)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		rs.statsMu.Lock()
		rs.stats.WASMErrors++
		rs.statsMu.Unlock()
		return result, err
	}

	if wasmType != WASMTypeResolution {
		result.Error = fmt.Sprintf("tag %s maps to %s, not resolution", signal.Tag, wasmType)
		result.Duration = time.Since(start)
		return result, errors.New(result.Error)
	}

	result.WASMPath = wasmPath

	// Acquire a WazeroGate from the pool for this module.
	pool := rs.getOrCreatePool(wasmPath)
	gate, err := pool.Acquire(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("wazero pool acquire: %v", err)
		result.Duration = time.Since(start)
		rs.statsMu.Lock()
		rs.stats.WASMErrors++
		rs.statsMu.Unlock()
		return result, err
	}

	// Execute resolveError()
	resolved, err := gate.ResolveError(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("resolveError(): %v", err)
		result.Duration = time.Since(start)
		rs.statsMu.Lock()
		rs.stats.WASMErrors++
		rs.stats.TotalResolutions++
		rs.stats.FailedResolve++
		rs.statsMu.Unlock()
		return result, err
	}

	result.Resolved = resolved

	// Execute ErrorClass()
	errorClass, err := gate.ErrorClass(ctx)
	if err != nil {
		// Non-fatal: we still have resolveError's result.
		log.Printf("[resolution-service] ErrorClass() failed: %v", err)
	}
	result.ErrorClassID = errorClass
	result.Duration = time.Since(start)

	rs.statsMu.Lock()
	rs.stats.TotalResolutions++
	if resolved {
		rs.stats.SuccessfulResolve++
	} else {
		rs.stats.FailedResolve++
	}
	rs.statsMu.Unlock()

	// If resolved, dispatch to remediation sink.
	if resolved && rs.sink != nil {
		if err := rs.sink.DispatchRemediation(ctx, result); err != nil {
			log.Printf("[resolution-service] remediation dispatch error: %v", err)
		}
	}

	return result, nil
}

// Stats returns current resolution statistics.
func (rs *ResolutionService) Stats() *ResolutionStats {
	rs.statsMu.RLock()
	defer rs.statsMu.RUnlock()
	s := *rs.stats
	return &s
}

// —— internal ——

func (rs *ResolutionService) worker(ctx context.Context, id int) {
	defer rs.workerWg.Done()
	log.Printf("[resolution-service] worker %d running", id)

	for {
		select {
		case <-rs.stopChan:
			log.Printf("[resolution-service] worker %d stopping", id)
			return
		case <-ctx.Done():
			log.Printf("[resolution-service] worker %d context done", id)
			return
		case signal := <-rs.signalChan:
			result, err := rs.Resolve(ctx, signal)
			if err != nil {
				log.Printf("[resolution-service] worker %d: resolve %s/%s: %v",
					id, signal.DVEID, signal.Tag, err)
			} else {
				log.Printf("[resolution-service] worker %d: resolved=%v class=%d dve=%s badge=%s (%v)",
					id, result.Resolved, result.ErrorClassID, signal.DVEID, signal.BadgeID, result.Duration)
			}
		}
	}
}

func (rs *ResolutionService) getOrCreatePool(wasmPath string) *WazeroPool {
	rs.poolsMu.RLock()
	pool, ok := rs.pools[wasmPath]
	rs.poolsMu.RUnlock()
	if ok {
		return pool
	}

	rs.poolsMu.Lock()
	defer rs.poolsMu.Unlock()
	// Double-check after acquiring write lock.
	if pool, ok = rs.pools[wasmPath]; ok {
		return pool
	}
	pool = NewWazeroPool(wasmPath, rs.poolSize)
	rs.pools[wasmPath] = pool
	return pool
}
