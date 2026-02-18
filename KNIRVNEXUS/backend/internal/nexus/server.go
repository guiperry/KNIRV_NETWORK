// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package nexus

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"backend_server/internal/ebpf"

	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/flight"
	"github.com/apache/arrow/go/v14/arrow/ipc"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/gorilla/mux"
)

// Intent represents what an agent said it would do
type Intent struct {
	AgentID   string
	IntentStr string
	Timestamp time.Time
}

// FabricEvent represents a captured event for the dashboard
type FabricEvent struct {
	Timestamp      int64  `json:"timestamp"`
	AgentID        string `json:"agent_id"`
	Intent         string `json:"intent"`
	ObservedAction string `json:"observed_action"`
	Verified       bool   `json:"verified"`
}

type ActiveMemoryProvider interface {
	StreamToArrow(agentID string, fs flight.FlightService_DoGetServer) error
}

// NexusMemoryServer implements the Arrow Flight server for the Memory Fabric
type NexusMemoryServer struct {
	flight.BaseFlightServer
	mem             memory.Allocator
	guardianManager *ebpf.NexusGuardianManager
	eventsChan      <-chan ebpf.NexusGuardianGuardianEvent
	memoryProvider  ActiveMemoryProvider
	
	intentsMu sync.Mutex
	intents   map[uint32]*Intent // Map PID to Intent

	recentEvents   []FabricEvent
	recentEventsMu sync.Mutex
}

// NewNexusMemoryServer creates a new NexusMemoryServer
func NewNexusMemoryServer(mem memory.Allocator) *NexusMemoryServer {
	return &NexusMemoryServer{
		mem:          mem,
		intents:      make(map[uint32]*Intent),
		recentEvents: make([]FabricEvent, 0),
	}
}

// SetMemoryProvider sets the provider for active memory streaming
func (s *NexusMemoryServer) SetMemoryProvider(provider ActiveMemoryProvider) {
	s.memoryProvider = provider
}

// RegisterIntent registers an agent's intent
func (s *NexusMemoryServer) RegisterIntent(pid uint32, agentID, intentStr string) {
	s.intentsMu.Lock()
	defer s.intentsMu.Unlock()
	s.intents[pid] = &Intent{
		AgentID:   agentID,
		IntentStr: intentStr,
		Timestamp: time.Now(),
	}
}

// StartGuardian starts the eBPF guardian
func (s *NexusMemoryServer) StartGuardian() error {
	s.guardianManager = ebpf.NewNexusGuardianManager()
	events, err := s.guardianManager.Start()
	if err != nil {
		return err
	}
	s.eventsChan = events

	// Start a loop to collect recent events for the dashboard
	go func() {
		for event := range s.eventsChan {
			s.intentsMu.Lock()
			intent, ok := s.intents[event.Pid]
			s.intentsMu.Unlock()

			agentID := "unknown"
			intentStr := "unknown"
			if ok {
				agentID = intent.AgentID
				intentStr = intent.IntentStr
			}

			observedAction := fmt.Sprintf("[%s] connect to %s:%d", 
				int8ArrayToString(event.Comm),
				intToIP(event.Addr), 
				ntohs(event.Port))
			
			verified := s.correlate(intentStr, observedAction)

			fe := FabricEvent{
				Timestamp:      int64(event.Timestamp),
				AgentID:        agentID,
				Intent:         intentStr,
				ObservedAction: observedAction,
				Verified:       verified,
			}

			s.recentEventsMu.Lock()
			s.recentEvents = append(s.recentEvents, fe)
			if len(s.recentEvents) > 100 {
				s.recentEvents = s.recentEvents[1:]
			}
			s.recentEventsMu.Unlock()
		}
	}()

	return nil
}

// DoGet streams the "Living Memory" to the agent
func (s *NexusMemoryServer) DoGet(tkt *flight.Ticket, fs flight.FlightService_DoGetServer) error {
	agentID := string(tkt.Ticket)
	log.Printf("Streaming memory fabric for agent: %s", agentID)

	if s.memoryProvider != nil {
		return s.memoryProvider.StreamToArrow(agentID, fs)
	}

	schema := NewAgentMemorySchema()
	writer := flight.NewRecordWriter(fs, ipc.WithSchema(schema))
	defer writer.Close()

	// In a real implementation, we would filter s.eventsChan or use a pub/sub
	// For this prototype, we'll just return some recent events matching the agentID
	s.recentEventsMu.Lock()
	events := make([]FabricEvent, len(s.recentEvents))
	copy(events, s.recentEvents)
	s.recentEventsMu.Unlock()

	for _, fe := range events {
		if fe.AgentID != agentID {
			continue
		}

		builder := array.NewRecordBuilder(s.mem, schema)
		builder.Field(0).(*array.Int64Builder).Append(fe.Timestamp)
		builder.Field(1).(*array.StringBuilder).Append(fe.AgentID)
		builder.Field(2).(*array.StringBuilder).Append(fe.Intent)
		builder.Field(3).(*array.StringBuilder).Append(fe.ObservedAction)
		builder.Field(4).(*array.Int32Builder).Append(0)
		builder.Field(5).(*array.Float64Builder).Append(1.0)
		builder.Field(6).(*array.BooleanBuilder).Append(fe.Verified)

		record := builder.NewRecord()
		if err := writer.Write(record); err != nil {
			record.Release()
			builder.Release()
			return err
		}
		record.Release()
		builder.Release()
	}

	return nil
}

func (s *NexusMemoryServer) correlate(intent, action string) bool {
	return true 
}

// RegisterRoutes registers HTTP routes for the dashboard
func (s *NexusMemoryServer) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/nexus/events", s.HandleGetEvents).Methods("GET", "OPTIONS")
}

// HandleGetEvents returns recent fabric events
func (s *NexusMemoryServer) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	s.recentEventsMu.Lock()
	defer s.recentEventsMu.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.recentEvents)
}

// Helper: convert int8 array to string
func int8ArrayToString(arr [16]int8) string {
	b := make([]byte, 0, len(arr))
	for _, v := range arr {
		if v == 0 {
			break
		}
		b = append(b, byte(v))
	}
	return string(b)
}

// Helper: convert uint32 to IP string
func intToIP(nn uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", 
		byte(nn), byte(nn>>8), byte(nn>>16), byte(nn>>24))
}

// Helper: network to host short
func ntohs(n uint16) uint16 {
	return ((n & 0xff00) >> 8) | ((n & 0x00ff) << 8)
}

// Serve starts the Arrow Flight server
func (s *NexusMemoryServer) Serve(addr string) error {
	server := flight.NewServerWithMiddleware(nil)
	server.RegisterFlightService(s)

	if err := server.Init(addr); err != nil {
		return err
	}

	log.Printf("KNIRV NEXUS: Arrow Flight Memory Fabric active on %s", addr)
	return server.Serve()
}

// Stop stops the NexusMemoryServer and the guardian
func (s *NexusMemoryServer) Stop() error {
	if s.guardianManager != nil {
		return s.guardianManager.Close()
	}
	return nil
}
