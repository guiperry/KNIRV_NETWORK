// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"golang.org/x/sys/unix"
)

type TelemetryCollector struct {
	mu            sync.RWMutex
	eventReader   *perf.Reader
	telemetryMap  *ebpf.Map
	stats         TelemetryStats
	stopChan      chan struct{}
	pidFilter     map[uint32]bool
	enabled       bool
	totalSyscalls uint64
	totalMemBytes uint64
	byPID         map[uint32]ProcessTelemetry
}

type TelemetryStats struct {
	TotalSyscalls    uint64
	TotalCPUCycles   uint64
	TotalMemoryBytes uint64
	NetTxBytes       uint64
	NetRxBytes       uint64
	ContextSwitches  uint32
	PageFaults       uint32
	LastUpdate       time.Time
	ByPID            map[uint32]ProcessTelemetry
}

type ProcessTelemetry struct {
	SyscallCount    uint64
	CPUTimeNs       uint64
	MemoryBytes     uint64
	NetTxBytes      uint64
	NetRxBytes      uint64
	ContextSwitches uint32
	PageFaults      uint32
	Command         string
}

func NewTelemetryCollector() *TelemetryCollector {
	return &TelemetryCollector{
		pidFilter: make(map[uint32]bool),
		stopChan:  make(chan struct{}),
		byPID:     make(map[uint32]ProcessTelemetry),
	}
}

func (tc *TelemetryCollector) Initialize() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	spec, err := tc.createTelemetrySpec()
	if err != nil {
		return fmt.Errorf("create telemetry spec: %w", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		log.Printf("eBPF collection creation failed (expected if no kernel headers): %v", err)
		tc.enabled = false
		return nil
	}

	tc.telemetryMap = coll.Maps["process_telemetry"]
	tc.enabled = true

	log.Println("TelemetryCollector: Initialized successfully")
	return nil
}

func (tc *TelemetryCollector) createTelemetrySpec() (*ebpf.CollectionSpec, error) {
	return &ebpf.CollectionSpec{
		Programs: map[string]*ebpf.ProgramSpec{
			"telemetryCollector": {
				Type:    ebpf.TracePoint,
				License: "GPL",
			},
		},
		Maps: map[string]*ebpf.MapSpec{
			"process_telemetry": {
				Type:       ebpf.Hash,
				KeySize:    4,
				ValueSize:  64,
				MaxEntries: 65536,
			},
			"syscall_counts": {
				Type:       ebpf.Array,
				KeySize:    4,
				ValueSize:  8,
				MaxEntries: 512,
			},
		},
	}, nil
}

func (tc *TelemetryCollector) Start(ctx context.Context) error {
	if !tc.enabled {
		return tc.startFallbackCollector(ctx)
	}

	go tc.collectTelemetry(ctx)
	return nil
}

func (tc *TelemetryCollector) startFallbackCollector(ctx context.Context) error {
	log.Println("TelemetryCollector: Using fallback procfs-based collection")

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-tc.stopChan:
				ticker.Stop()
				return
			case <-ticker.C:
				tc.collectFromProc()
			}
		}
	}()

	return nil
}

func (tc *TelemetryCollector) collectFromProc() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	procs, err := os.ReadDir("/proc")
	if err != nil {
		return
	}

	for _, proc := range procs {
		var pid uint32
		if _, err := fmt.Sscanf(proc.Name(), "%d", &pid); err != nil {
			continue
		}

		statPath := fmt.Sprintf("/proc/%d/stat", pid)
		var stat unix.Stat_t
		if err := unix.Stat(statPath, &stat); err != nil {
			continue
		}

		commPath := fmt.Sprintf("/proc/%d/cmdline", pid)
		cmdline, _ := os.ReadFile(commPath)
		comm := string(cmdline)
		if len(comm) > 15 {
			comm = comm[:15]
		}

		existing, exists := tc.byPID[pid]
		if !exists {
			existing = ProcessTelemetry{}
		}

		existing.CPUTimeNs = uint64(stat.Atim.Sec*1e9 + int64(stat.Atim.Nsec))
		existing.MemoryBytes = uint64(stat.Size)

		tc.byPID[pid] = existing
		tc.totalSyscalls++
	}

	tc.stats.TotalSyscalls = tc.totalSyscalls
	tc.stats.TotalMemoryBytes = tc.totalMemBytes
	tc.stats.LastUpdate = time.Now()
}

func (tc *TelemetryCollector) collectTelemetry(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tc.stopChan:
			return
		case <-ticker.C:
			tc.updateFromMap()
		}
	}
}

func (tc *TelemetryCollector) updateFromMap() {
	if tc.telemetryMap == nil {
		return
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	var pid uint32
	var value ProcessTelemetryData
	iter := tc.telemetryMap.Iterate()
	for iter.Next(&pid, &value) {
		existing, exists := tc.byPID[pid]
		if !exists {
			existing = ProcessTelemetry{}
		}

		existing.SyscallCount = value.SyscallCount
		existing.CPUTimeNs = value.CPUTimeNs
		existing.MemoryBytes = value.MemoryBytes
		existing.NetTxBytes = value.NetTxBytes
		existing.NetRxBytes = value.NetRxBytes
		existing.ContextSwitches = value.ContextSwitches
		existing.PageFaults = value.PageFaults

		tc.byPID[pid] = existing
	}

	tc.stats.ByPID = tc.byPID
	tc.stats.LastUpdate = time.Now()
}

type ProcessTelemetryData struct {
	SyscallCount    uint64
	CPUTimeNs       uint64
	MemoryBytes     uint64
	NetTxBytes      uint64
	NetRxBytes      uint64
	ContextSwitches uint32
	PageFaults      uint32
}

func (tc *TelemetryCollector) AddPID(pid uint32) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.pidFilter[pid] = true
}

func (tc *TelemetryCollector) RemovePID(pid uint32) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	delete(tc.pidFilter, pid)
}

func (tc *TelemetryCollector) GetStats() TelemetryStats {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	stats := tc.stats
	stats.ByPID = make(map[uint32]ProcessTelemetry)
	for k, v := range tc.byPID {
		if len(tc.pidFilter) == 0 || tc.pidFilter[k] {
			stats.ByPID[k] = v
		}
	}
	return stats
}

func (tc *TelemetryCollector) GetProcessStats(pid uint32) (ProcessTelemetry, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	stats, exists := tc.byPID[pid]
	return stats, exists
}

func (tc *TelemetryCollector) Stop() error {
	close(tc.stopChan)
	if tc.eventReader != nil {
		tc.eventReader.Close()
	}
	return nil
}

func (tc *TelemetryCollector) IsEnabled() bool {
	return tc.enabled
}

func (tc *TelemetryCollector) ReadSyscallCounts() map[string]uint64 {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	counts := make(map[string]uint64)
	for pid, tel := range tc.byPID {
		counts[fmt.Sprintf("pid_%d", pid)] = tel.SyscallCount
	}
	return counts
}

func (tc *TelemetryCollector) GetAggregatedStats() (cpuNs uint64, memBytes uint64, netTx uint64, netRx uint64) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	for _, tel := range tc.byPID {
		cpuNs += tel.CPUTimeNs
		memBytes += tel.MemoryBytes
		netTx += tel.NetTxBytes
		netRx += tel.NetRxBytes
	}
	return
}

func ReadUint64LE(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}
