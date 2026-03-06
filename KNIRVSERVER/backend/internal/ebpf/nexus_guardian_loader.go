// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
)

// NexusGuardianManager manages the Nexus Guardian eBPF program
type NexusGuardianManager struct {
	objs      NexusGuardianObjects
	links     []link.Link
	ringReader *ringbuf.Reader
}

// NewNexusGuardianManager creates a new NexusGuardianManager
func NewNexusGuardianManager() *NexusGuardianManager {
	return &NexusGuardianManager{}
}

// Start loads and attaches the Nexus Guardian eBPF programs
func (m *NexusGuardianManager) Start() (<-chan NexusGuardianGuardianEvent, error) {
	// Load pre-compiled programs and maps into the kernel.
	if err := LoadNexusGuardianObjects(&m.objs, nil); err != nil {
		return nil, fmt.Errorf("loading objects: %w", err)
	}

	// Attach the tracepoint
	l, err := link.Tracepoint("raw_syscalls", "sys_enter", m.objs.TraceConnect, nil)
	if err != nil {
		m.Close()
		return nil, fmt.Errorf("opening tracepoint: %w", err)
	}
	m.links = append(m.links, l)

	// Open a ringbuf reader from userspace
	rd, err := ringbuf.NewReader(m.objs.Events)
	if err != nil {
		m.Close()
		return nil, fmt.Errorf("opening ringbuf reader: %w", err)
	}
	m.ringReader = rd

	eventsChan := make(chan NexusGuardianGuardianEvent, 100)

	go func() {
		defer close(eventsChan)
		for {
			record, err := m.ringReader.Read()
			if err != nil {
				if err == ringbuf.ErrClosed {
					return
				}
				log.Printf("reading from ringbuf: %v", err)
				continue
			}

			var event NexusGuardianGuardianEvent
			if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event); err != nil {
				log.Printf("parsing ringbuf event: %v", err)
				continue
			}

			eventsChan <- event
		}
	}()

	return eventsChan, nil
}

// Close cleans up the NexusGuardianManager resources
func (m *NexusGuardianManager) Close() error {
	if m.ringReader != nil {
		m.ringReader.Close()
	}
	for _, l := range m.links {
		l.Close()
	}
	return m.objs.Close()
}
