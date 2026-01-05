package dataengine

import (
	"context"
	"log"
	"sync"
	"time"

	"backend_server/internal/ebpf"
)

// TelemetryCollector collects and processes telemetry periodically
// EBPFManager defines the minimal interface used by the TelemetryCollector
type EBPFManager interface {
	GetProcessMetrics() (map[uint32]*ebpf.ProcessStats, error)
}

type TelemetryCollector struct {
	ebpfMgr    EBPFManager
	store      *FingerprintStore
	aiDetector *AnomalyDetector
	mu         sync.Mutex
	running    bool
}

func NewTelemetryCollector(mgr EBPFManager, store *FingerprintStore, ad *AnomalyDetector) *TelemetryCollector {
	return &TelemetryCollector{
		ebpfMgr:    mgr,
		store:      store,
		aiDetector: ad,
	}
}

func (tc *TelemetryCollector) Start(ctx context.Context) error {
	if tc.running {
		return nil
	}
	go func() {
		tc.running = true
		defer func() { tc.running = false }()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics, err := tc.ebpfMgr.GetProcessMetrics()
				if err != nil {
					log.Printf("get metrics: %v", err)
					continue
				}
				for pid, stats := range metrics {
					_ = pid
					// Persist or analyze as needed
					if anomaly := tc.aiDetector.Detect(stats); anomaly != nil {
						log.Printf("Anomaly detected for pid %d: %s", pid, anomaly.Description)
					}
				}
			}
		}
	}()
	return nil
}
