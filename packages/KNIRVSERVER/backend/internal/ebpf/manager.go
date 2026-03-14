// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

// Manager is the main eBPF subsystem controller
type Manager struct {
	collection  *ebpf.Collection
	links       []link.Link
	mu          sync.Mutex
	initialized bool
}

// Ensure Manager implements ManagerInterface
var _ ManagerInterface = (*Manager)(nil)

// NewManager creates a new eBPF Manager instance
func NewManager() *Manager {
	return &Manager{}
}

// Initialize sets up the eBPF Manager with the given configuration
func (m *Manager) Initialize(ctx context.Context, config *Config) error {
	if m == nil {
		return fmt.Errorf("ebpf manager is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return fmt.Errorf("manager already initialized")
	}

	// Load eBPF programs
	loader := NewLoader()
	if err := loader.LoadSyscallTrace(); err != nil {
		return fmt.Errorf("load syscall trace: %w", err)
	}

	m.collection = loader.collection
	m.links = loader.links
	m.initialized = true

	log.Println("eBPF Manager initialized successfully")
	return nil
}

// Shutdown cleans up all eBPF resources
func (m *Manager) Shutdown() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return nil
	}

	// Close all links
	for _, link := range m.links {
		if err := link.Close(); err != nil {
			log.Printf("Error closing link: %v", err)
		}
	}
	m.links = nil

	// Close collection
	if m.collection != nil {
		m.collection.Close()
		m.collection = nil
	}

	m.initialized = false
	log.Println("eBPF Manager shutdown complete")
	return nil
}

// GetMetrics returns current eBPF metrics
func (m *Manager) GetMetrics() *Metrics {
	if m == nil {
		return &Metrics{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return &Metrics{}
	}

	return &Metrics{
		ProgramsAttached: len(m.links),
		Initialized:      m.initialized,
	}
}

// GetProcessMetrics reads the process telemetry map and returns a map[pid]*ProcessStats
func (m *Manager) GetProcessMetrics() (map[uint32]*ProcessStats, error) {
	if m == nil {
		return nil, fmt.Errorf("manager not initialized")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized || m.collection == nil {
		return nil, fmt.Errorf("manager not initialized")
	}

	pm := make(map[uint32]*ProcessStats)
	mmap := m.collection.Maps["process_telemetry"]
	if mmap == nil {
		return pm, nil // no telemetry map available
	}

	var key, nextKey uint32
	var raw = make([]byte, 4096)

	for {
		if err := mmap.NextKey(&key, &nextKey); err != nil {
			break
		}

		// Lookup the value
		if err := mmap.Lookup(nextKey, &raw); err != nil {
			// Try direct byte-slice lookup (some map implementations require pointer)
			var buf [4096]byte
			if err := mmap.Lookup(nextKey, &buf); err != nil {
				// skip entries we can't read
				key = nextKey
				continue
			}
			raw = buf[:]
		}

		// Parse raw into ProcessStats
		ps := &ProcessStats{}
		offset := 0
		// Syscall counts
		for i := 0; i < 400; i++ {
			if offset+8 > len(raw) {
				break
			}
			ps.SyscallCount[i] = uint64(binary.LittleEndian.Uint64(raw[offset : offset+8]))
			offset += 8
		}

		if offset+8 <= len(raw) {
			ps.CPUTimeNs = binary.LittleEndian.Uint64(raw[offset : offset+8])
			offset += 8
		}
		if offset+8 <= len(raw) {
			ps.MemoryBytes = binary.LittleEndian.Uint64(raw[offset : offset+8])
			offset += 8
		}
		if offset+8 <= len(raw) {
			ps.NetTxBytes = binary.LittleEndian.Uint64(raw[offset : offset+8])
			offset += 8
		}
		if offset+8 <= len(raw) {
			ps.NetRxBytes = binary.LittleEndian.Uint64(raw[offset : offset+8])
			offset += 8
		}
		if offset+4 <= len(raw) {
			ps.ContextSwitches = binary.LittleEndian.Uint32(raw[offset : offset+4])
			offset += 4
		}
		if offset+4 <= len(raw) {
			ps.PageFaults = binary.LittleEndian.Uint32(raw[offset : offset+4])
			offset += 4
		}

		// ModelName: remaining bytes until first 0
		if offset < len(raw) {
			n := bytes.IndexByte(raw[offset:], 0)
			if n < 0 {
				n = len(raw) - offset
			}
			ps.ModelName = string(raw[offset : offset+n])
		}

		pm[nextKey] = ps
		key = nextKey
	}

	return pm, nil
}
