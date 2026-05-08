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

	"github.com/cilium/ebpf/ringbuf"
)

// EventHandler is a function that processes eBPF events
type EventHandler func(event *SyscallEvent) error

// ResolutionSignalHandler processes eBPF events that have been classified
// as resolution triggers.  These are forwarded to the ResolutionService
// for resolution.wasm execution.
//
// The handler receives the raw syscall event plus a badgeID and ontology
// tag derived from the DVE's active Badge mapping.
type ResolutionSignalHandler func(event *SyscallEvent, badgeID string, tag string)

// EventCollector manages event collection from eBPF programs
type EventCollector struct {
	manager           *Manager
	handlers          []EventHandler
	resolutionHandler ResolutionSignalHandler // optional WASM resolution handler
	// badgeMapping maps (containerID, syscallID) → badgeID for resolution routing.
	// Populated by the eBPF manager when containers are registered with badges.
	badgeMapping map[string]string // key: "containerID:syscallID"
	mu           sync.Mutex
	stopChan     chan struct{}
	workerWg     sync.WaitGroup
}

// NewEventCollector creates a new EventCollector
func NewEventCollector(manager *Manager) *EventCollector {
	return &EventCollector{
		manager:      manager,
		handlers:     make([]EventHandler, 0),
		badgeMapping: make(map[string]string),
		stopChan:     make(chan struct{}),
	}
}

// SubscribeResolution registers a handler for resolution-classified events.
// When an eBPF event matches a resolution badge mapping, this handler is
// invoked with the event, badge ID, and ontology tag for WASM resolution.
func (ec *EventCollector) SubscribeResolution(handler ResolutionSignalHandler) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.resolutionHandler = handler
}

// SetBadgeMapping associates a container/syscall combination with a badge ID.
// The key format is "containerID:syscallID".
func (ec *EventCollector) SetBadgeMapping(containerID string, syscallID uint32, badgeID string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	key := fmt.Sprintf("%s:%d", containerID, syscallID)
	ec.badgeMapping[key] = badgeID
}

// RemoveBadgeMapping clears badge mappings for a container.
func (ec *EventCollector) RemoveBadgeMapping(containerID string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	prefix := containerID + ":"
	for key := range ec.badgeMapping {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			delete(ec.badgeMapping, key)
		}
	}
}

// Subscribe adds an event handler
func (ec *EventCollector) Subscribe(handler EventHandler) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.handlers = append(ec.handlers, handler)
}

// Start begins event collection
func (ec *EventCollector) Start(ctx context.Context) error {
	if ec.manager.collection == nil {
		return fmt.Errorf("eBPF collection not initialized")
	}

	eventsMap := ec.manager.collection.Maps["events"]
	if eventsMap == nil {
		return fmt.Errorf("events map not found in collection")
	}

	rd, err := ringbuf.NewReader(eventsMap)
	if err != nil {
		return fmt.Errorf("create ringbuf reader: %w", err)
	}

	ec.workerWg.Add(1)
	go func() {
		defer ec.workerWg.Done()
		
		log.Println("eBPF Event Collector started")
		defer log.Println("eBPF Event Collector stopped")

		for {
			select {
			case <-ec.stopChan:
				return
			case <-ctx.Done():
				return
			default:
				record, err := rd.Read()
				if err != nil {
					if err == ringbuf.ErrClosed {
						return
					}
					log.Printf("read event: %v", err)
					continue
				}

				var event SyscallEvent
				if err := binary.Read(
					bytes.NewReader(record.RawSample),
					binary.LittleEndian,
					&event,
				); err != nil {
					log.Printf("parse event: %v", err)
					continue
				}

				ec.mu.Lock()
				// Forward to general event handlers.
				for _, handler := range ec.handlers {
					if err := handler(&event); err != nil {
						log.Printf("handle event: %v", err)
					}
				}

				// Check for resolution-classified events and forward to the
				// resolution handler if a badge mapping exists.
				if ec.resolutionHandler != nil {
					// Look up badge by syscall ID (container context is implicit
					// from the event's PID mapping in VirtualContainerManager).
					// We use a fallback key "any:syscallID" when no specific
					// container mapping exists.
					for _, badgeID := range ec.badgeMapping {
						if badgeID != "" {
							// Derive an ontology tag from the key.
							tag := fmt.Sprintf("resolution:syscall:%d", event.SyscallID)
							ec.resolutionHandler(&event, badgeID, tag)
							break // one resolution per event
						}
					}
				}
				ec.mu.Unlock()
			}
		}
	}()

	return nil
}

// Stop halts event collection
func (ec *EventCollector) Stop() {
	close(ec.stopChan)
	ec.workerWg.Wait()
}