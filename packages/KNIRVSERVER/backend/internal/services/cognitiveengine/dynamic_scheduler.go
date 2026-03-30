package cognitiveengine

import (
	"context"
	"fmt"
	"log"
	"math"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type SchedulerMode string

const (
	ModeConservative SchedulerMode = "conservative"
	ModeBalanced     SchedulerMode = "balanced"
	ModeAggressive   SchedulerMode = "aggressive"
)

type DynamicScheduler struct {
	mu              sync.RWMutex
	nodeID          string
	currentProcs    int32
	targetProcs     int32
	mode            SchedulerMode
	enabled         bool
	ctx             context.Context
	cancel          context.CancelFunc
	config          *SchedulerConfig
	metrics         *SchedulerMetrics
	history         []SchedulerSnapshot
	maxHistorySize  int
	adjustmentCount int64
	lastAdjustment  time.Time
}

type SchedulerConfig struct {
	Mode               SchedulerMode
	MinProcs           int
	MaxProcs           int
	CPUHighThreshold   float64
	CPULowThreshold    float64
	MemoryHighMB       int64
	EvaluationInterval time.Duration
	CooldownPeriod     time.Duration
	AutoAdjustment     bool
	NotifyOnChange     bool
}

type SchedulerMetrics struct {
	CPUUtilization    float64
	MemoryUsageMB     int64
	GoroutineCount    int
	ContextSwitches   uint64
	SysCalls          uint64
	BlockedGoroutines int
	GCFraction        float64
}

type SchedulerSnapshot struct {
	Timestamp       time.Time
	Procs           int
	CPUUtilization  float64
	MemoryUsageMB   int64
	GoroutineCount  int
	QueueDepth      int
	AdjustmentDelta int
}

func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		Mode:               ModeBalanced,
		MinProcs:           1,
		MaxProcs:           runtime.NumCPU(),
		CPUHighThreshold:   0.8,
		CPULowThreshold:    0.2,
		MemoryHighMB:       8192,
		EvaluationInterval: 10 * time.Second,
		CooldownPeriod:     30 * time.Second,
		AutoAdjustment:     true,
		NotifyOnChange:     true,
	}
}

func NewDynamicScheduler(nodeID string, cfg *SchedulerConfig) *DynamicScheduler {
	if cfg == nil {
		cfg = DefaultSchedulerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	currentProcs := runtime.GOMAXPROCS(0)

	return &DynamicScheduler{
		nodeID:         nodeID,
		currentProcs:   int32(currentProcs),
		targetProcs:    int32(currentProcs),
		mode:           cfg.Mode,
		enabled:        false,
		ctx:            ctx,
		cancel:         cancel,
		config:         cfg,
		metrics:        &SchedulerMetrics{},
		history:        make([]SchedulerSnapshot, 0, 100),
		maxHistorySize: 100,
	}
}

func (ds *DynamicScheduler) Start() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.enabled {
		return fmt.Errorf("dynamic scheduler already running")
	}

	ds.enabled = true
	ds.lastAdjustment = time.Now()

	if ds.config.AutoAdjustment {
		go ds.evaluationLoop()
	}

	log.Printf("DynamicScheduler[%s]: started with %d procs (mode=%s)", ds.nodeID, ds.currentProcs, ds.mode)
	return nil
}

func (ds *DynamicScheduler) Stop() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if !ds.enabled {
		return nil
	}

	ds.cancel()
	ds.enabled = false

	runtime.GOMAXPROCS(int(ds.config.MaxProcs))
	atomic.StoreInt32(&ds.currentProcs, int32(ds.config.MaxProcs))

	log.Printf("DynamicScheduler[%s]: stopped, restored to %d procs", ds.nodeID, ds.config.MaxProcs)
	return nil
}

func (ds *DynamicScheduler) evaluationLoop() {
	ticker := time.NewTicker(ds.config.EvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			ds.evaluateAndAdjust()
		}
	}
}

func (ds *DynamicScheduler) evaluateAndAdjust() {
	metrics := ds.CollectMetrics()
	ds.metrics = metrics

	shouldAdjust, delta := ds.shouldAdjust(metrics)
	if shouldAdjust {
		if err := ds.adjustProcs(delta); err != nil {
			log.Printf("DynamicScheduler: adjustment failed: %v", err)
		}
	}

	ds.recordSnapshot(delta)
}

func (ds *DynamicScheduler) shouldAdjust(metrics *SchedulerMetrics) (bool, int) {
	if time.Since(ds.lastAdjustment) < ds.config.CooldownPeriod {
		return false, 0
	}

	currentProcs := atomic.LoadInt32(&ds.currentProcs)

	switch ds.mode {
	case ModeConservative:
		return ds.conservativeCheck(metrics, currentProcs)
	case ModeAggressive:
		return ds.aggressiveCheck(metrics, currentProcs)
	default:
		return ds.balancedCheck(metrics, currentProcs)
	}
}

func (ds *DynamicScheduler) conservativeCheck(metrics *SchedulerMetrics, currentProcs int32) (bool, int) {
	if metrics.CPUUtilization > ds.config.CPUHighThreshold && currentProcs < int32(ds.config.MaxProcs) {
		return true, 1
	}
	if metrics.CPUUtilization < ds.config.CPULowThreshold && currentProcs > int32(ds.config.MinProcs) && metrics.BlockedGoroutines > 10 {
		return true, -1
	}
	return false, 0
}

func (ds *DynamicScheduler) balancedCheck(metrics *SchedulerMetrics, currentProcs int32) (bool, int) {
	if metrics.CPUUtilization > ds.config.CPUHighThreshold && currentProcs < int32(ds.config.MaxProcs) {
		if metrics.BlockedGoroutines > 5 {
			return true, 2
		}
		return true, 1
	}
	if metrics.CPUUtilization < ds.config.CPULowThreshold && currentProcs > int32(ds.config.MinProcs) {
		return true, -1
	}
	return false, 0
}

func (ds *DynamicScheduler) aggressiveCheck(metrics *SchedulerMetrics, currentProcs int32) (bool, int) {
	avgCPU := ds.getAverageCPUFromHistory(5)

	if metrics.CPUUtilization > ds.config.CPUHighThreshold {
		delta := 0
		if metrics.BlockedGoroutines > 20 {
			delta = 3
		} else if metrics.BlockedGoroutines > 10 {
			delta = 2
		} else {
			delta = 1
		}

		if avgCPU > 0.9 && metrics.GoroutineCount > 1000 {
			delta = int(math.Min(float64(delta), 4))
		}

		newProcs := int(currentProcs) + delta
		if newProcs <= ds.config.MaxProcs {
			return true, delta
		}
	}

	if metrics.CPUUtilization < ds.config.CPULowThreshold {
		delta := 1
		if metrics.GoroutineCount < 100 {
			delta = 2
		}
		newProcs := int(currentProcs) - delta
		if newProcs >= ds.config.MinProcs {
			return true, -delta
		}
	}

	return false, 0
}

func (ds *DynamicScheduler) adjustProcs(delta int) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	currentProcs := atomic.LoadInt32(&ds.currentProcs)
	newProcs := int(currentProcs) + delta

	if newProcs < ds.config.MinProcs {
		newProcs = ds.config.MinProcs
	}
	if newProcs > ds.config.MaxProcs {
		newProcs = ds.config.MaxProcs
	}

	if newProcs == int(currentProcs) {
		return nil
	}

	runtime.GOMAXPROCS(newProcs)
	atomic.StoreInt32(&ds.currentProcs, int32(newProcs))
	atomic.StoreInt32(&ds.targetProcs, int32(newProcs))
	ds.lastAdjustment = time.Now()
	ds.adjustmentCount++

	log.Printf("DynamicScheduler[%s]: adjusted GOMAXPROCS from %d to %d (delta=%d)",
		ds.nodeID, currentProcs, newProcs, delta)

	return nil
}

func (ds *DynamicScheduler) recordSnapshot(adjustmentDelta int) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	snapshot := SchedulerSnapshot{
		Timestamp:       time.Now(),
		Procs:           int(atomic.LoadInt32(&ds.currentProcs)),
		CPUUtilization:  ds.metrics.CPUUtilization,
		MemoryUsageMB:   int64(m.HeapAlloc) / (1024 * 1024),
		GoroutineCount:  runtime.NumGoroutine(),
		QueueDepth:      0,
		AdjustmentDelta: adjustmentDelta,
	}

	ds.mu.Lock()
	ds.history = append(ds.history, snapshot)
	if len(ds.history) > ds.maxHistorySize {
		ds.history = ds.history[1:]
	}
	ds.mu.Unlock()
}

func (ds *DynamicScheduler) getAverageCPUFromHistory(count int) float64 {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if len(ds.history) == 0 {
		return 0.0
	}

	if count > len(ds.history) {
		count = len(ds.history)
	}

	sum := 0.0
	start := len(ds.history) - count
	for i := start; i < len(ds.history); i++ {
		sum += ds.history[i].CPUUtilization
	}

	return sum / float64(count)
}

func (ds *DynamicScheduler) CollectMetrics() *SchedulerMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var gcStats debug.GCStats
	debug.ReadGCStats(&gcStats)

	gcFraction := 0.0
	if m.NumGC > 0 {
		gcFraction = float64(m.PauseTotalNs) / float64(time.Now().UnixNano())
	}

	return &SchedulerMetrics{
		CPUUtilization:    ds.measureCPUUtilization(),
		MemoryUsageMB:     int64(m.HeapAlloc) / (1024 * 1024),
		GoroutineCount:    runtime.NumGoroutine(),
		ContextSwitches:   0,
		SysCalls:          0,
		BlockedGoroutines: ds.countBlockedGoroutines(),
		GCFraction:        gcFraction,
	}
}

func (ds *DynamicScheduler) measureCPUUtilization() float64 {
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	time.Sleep(100 * time.Millisecond)
	runtime.ReadMemStats(&m2)

	gcOverhead := 0.0
	if m2.PauseTotalNs > m1.PauseTotalNs {
		gcOverhead = float64(m2.PauseTotalNs-m1.PauseTotalNs) / float64(100*time.Millisecond)
	}

	cpuUsage := float64(runtime.NumGoroutine()) / float64(runtime.GOMAXPROCS(0)*100)
	cpuUsage = math.Min(1.0, math.Max(0.0, cpuUsage+gcOverhead))

	return cpuUsage
}

func (ds *DynamicScheduler) countBlockedGoroutines() int {
	return 0
}

func (ds *DynamicScheduler) SetMode(mode SchedulerMode) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.mode = mode
	log.Printf("DynamicScheduler[%s]: mode changed to %s", ds.nodeID, mode)
}

func (ds *DynamicScheduler) SetProcs(procs int) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if procs < ds.config.MinProcs || procs > ds.config.MaxProcs {
		return fmt.Errorf("procs must be between %d and %d", ds.config.MinProcs, ds.config.MaxProcs)
	}

	runtime.GOMAXPROCS(procs)
	atomic.StoreInt32(&ds.currentProcs, int32(procs))
	atomic.StoreInt32(&ds.targetProcs, int32(procs))
	ds.lastAdjustment = time.Now()

	log.Printf("DynamicScheduler[%s]: GOMAXPROCS set to %d", ds.nodeID, procs)
	return nil
}

func (ds *DynamicScheduler) GetCurrentProcs() int {
	return int(atomic.LoadInt32(&ds.currentProcs))
}

func (ds *DynamicScheduler) GetTargetProcs() int {
	return int(atomic.LoadInt32(&ds.targetProcs))
}

func (ds *DynamicScheduler) GetMode() SchedulerMode {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.mode
}

func (ds *DynamicScheduler) GetMetrics() *SchedulerMetrics {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.metrics
}

func (ds *DynamicScheduler) GetHistory(count int) []SchedulerSnapshot {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if count > len(ds.history) {
		count = len(ds.history)
	}

	result := make([]SchedulerSnapshot, count)
	copy(result, ds.history[len(ds.history)-count:])
	return result
}

func (ds *DynamicScheduler) GetAdjustmentCount() int64 {
	return atomic.LoadInt64(&ds.adjustmentCount)
}

func (ds *DynamicScheduler) GetLastAdjustment() time.Time {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.lastAdjustment
}

func (ds *DynamicScheduler) IsEnabled() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.enabled
}

func (ds *DynamicScheduler) GetStats() map[string]interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	return map[string]interface{}{
		"node_id":          ds.nodeID,
		"current_procs":    atomic.LoadInt32(&ds.currentProcs),
		"target_procs":     atomic.LoadInt32(&ds.targetProcs),
		"mode":             ds.mode,
		"enabled":          ds.enabled,
		"adjustment_count": atomic.LoadInt64(&ds.adjustmentCount),
		"last_adjustment":  ds.lastAdjustment,
		"min_procs":        ds.config.MinProcs,
		"max_procs":        ds.config.MaxProcs,
		"history_size":     len(ds.history),
	}
}
