// Copyright 2026 KNIRV-NEXUS
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

// EventCollector manages event collection from eBPF programs
type EventCollector struct {
	manager     *Manager
	handlers    []EventHandler
	mu          sync.Mutex
	stopChan    chan struct{}
	workerWg    sync.WaitGroup
}

// NewEventCollector creates a new EventCollector
func NewEventCollector(manager *Manager) *EventCollector {
	return &EventCollector{
		manager:  manager,
		handlers: make([]EventHandler, 0),
		stopChan: make(chan struct{}),
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
				for _, handler := range ec.handlers {
					if err := handler(&event); err != nil {
						log.Printf("handle event: %v", err)
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