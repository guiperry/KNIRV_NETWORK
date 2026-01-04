// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"context"
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

// NewManager creates a new eBPF Manager instance
func NewManager() *Manager {
	return &Manager{}
}

// Initialize sets up the eBPF Manager with the given configuration
func (m *Manager) Initialize(ctx context.Context, config *Config) error {
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
