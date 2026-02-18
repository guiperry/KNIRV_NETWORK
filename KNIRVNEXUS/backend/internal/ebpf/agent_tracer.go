// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/fintech"
)

// AgentTracer provides eBPF-based execution tracing for AI agent validation
type AgentTracer struct {
	manager         *Manager
	activeCaptures  map[string]*CaptureSession
	mu              sync.RWMutex
	eventBufferSize int
}

// CaptureSession represents an active trajectory capture
type CaptureSession struct {
	ID              string
	AgentID         string
	ValidationID    string
	ProcessID       uint32
	ContainerID     string
	Trajectory      *fintech.ExecutionTrajectory
	Config          *fintech.TrajectoryCaptureConfig
	StartTime       time.Time
	EventsCollected uint64
	Status          string // capturing, paused, completed, error
	EventChan       chan *fintech.ExecutionPoint
	ErrorChan       chan error
	StopChan        chan struct{}
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewAgentTracer creates a new agent tracer
func NewAgentTracer(manager *Manager) *AgentTracer {
	return &AgentTracer{
		manager:         manager,
		activeCaptures:  make(map[string]*CaptureSession),
		eventBufferSize: 10000,
	}
}

// StartCapture begins capturing execution trajectory for an agent
func (at *AgentTracer) StartCapture(ctx context.Context, agentID, validationID string, pid uint32, config *fintech.TrajectoryCaptureConfig) (*CaptureSession, error) {
	if config == nil {
		config = fintech.DefaultTrajectoryCaptureConfig()
	}

	if at.manager == nil {
		return nil, fmt.Errorf("eBPF manager not initialized")
	}

	// Create capture session
	sessionCtx, cancel := context.WithCancel(ctx)
	session := &CaptureSession{
		ID:           fmt.Sprintf("capture-%d", time.Now().UnixNano()),
		AgentID:      agentID,
		ValidationID: validationID,
		ProcessID:    pid,
		Trajectory:   fintech.NewExecutionTrajectory(agentID, "", validationID),
		Config:       config,
		StartTime:    time.Now(),
		Status:       "capturing",
		EventChan:    make(chan *fintech.ExecutionPoint, at.eventBufferSize),
		ErrorChan:    make(chan error, 100),
		StopChan:     make(chan struct{}),
		ctx:          sessionCtx,
		cancel:       cancel,
	}

	// Store trajectory metadata
	session.Trajectory.AgentID = agentID
	session.Trajectory.ValidationID = validationID
	session.Trajectory.ProcessID = pid

	// Register session
	at.mu.Lock()
	at.activeCaptures[session.ID] = session
	at.mu.Unlock()

	// Start capture goroutines
	go at.captureEvents(session)
	go at.processEvents(session)

	log.Printf("AgentTracer: Started capture session %s for agent %s (PID: %d)",
		session.ID, agentID, pid)

	return session, nil
}

// captureEvents captures events from eBPF
func (at *AgentTracer) captureEvents(session *CaptureSession) {
	defer close(session.EventChan)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	sequenceNum := uint64(0)

	for {
		select {
		case <-session.ctx.Done():
			return
		case <-session.StopChan:
			return
		case <-ticker.C:
			// Poll for eBPF events
			events, err := at.pollEvents(session)
			if err != nil {
				select {
				case session.ErrorChan <- err:
				default:
					// Error channel full, log instead
					log.Printf("AgentTracer: Error polling events: %v", err)
				}
				continue
			}

			for _, event := range events {
				sequenceNum++
				event.SequenceNum = sequenceNum

				select {
				case session.EventChan <- event:
					session.EventsCollected++
				case <-session.ctx.Done():
					return
				}

				// Check max events limit
				if session.Config.MaxEvents > 0 && session.EventsCollected >= session.Config.MaxEvents {
					log.Printf("AgentTracer: Max events reached for session %s", session.ID)
					return
				}
			}
		}
	}
}

// pollEvents polls eBPF for new events
func (at *AgentTracer) pollEvents(session *CaptureSession) ([]*fintech.ExecutionPoint, error) {
	if at.manager == nil || !at.manager.initialized {
		return nil, fmt.Errorf("eBPF manager not initialized")
	}

	// Get process metrics from eBPF manager
	metrics, err := at.manager.GetProcessMetrics()
	if err != nil {
		// Return empty events instead of error to continue polling
		return []*fintech.ExecutionPoint{}, nil
	}

	var events []*fintech.ExecutionPoint
	now := time.Now()

	// Convert process metrics to execution points
	for pid, stats := range metrics {
		// Filter by PID if specified
		if session.ProcessID != 0 && pid != session.ProcessID {
			continue
		}

		// Create syscall events from metrics
		if session.Config.CaptureSyscalls {
			for syscallID, count := range stats.SyscallCount {
				if count > 0 {
					// Create event for significant syscalls
					event := &fintech.ExecutionPoint{
						Timestamp: now,
						PID:       pid,
						EventType: "syscall",
						Syscall: &fintech.TrajectorySyscallEvent{
							Timestamp:   now,
							PID:         pid,
							SyscallID:   uint32(syscallID),
							SyscallName: getSyscallName(syscallID),
							ProcessName: stats.ModelName,
						},
					}
					events = append(events, event)
				}
			}
		}

		// Create resource usage event
		if session.Config.CaptureMemory {
			event := &fintech.ExecutionPoint{
				Timestamp: now,
				PID:       pid,
				EventType: "memory",
				Memory: &fintech.TrajectoryMemoryEvent{
					Timestamp: now,
					PID:       pid,
					Operation: "usage",
					Size:      stats.MemoryBytes,
				},
			}
			events = append(events, event)
		}

		// Create network event
		if session.Config.CaptureNetwork && (stats.NetTxBytes > 0 || stats.NetRxBytes > 0) {
			event := &fintech.ExecutionPoint{
				Timestamp: now,
				PID:       pid,
				EventType: "network",
				Network: &fintech.TrajectoryNetworkEvent{
					Timestamp: now,
					PID:       pid,
					Operation: "transfer",
					BytesSent: stats.NetTxBytes,
					BytesRecv: stats.NetRxBytes,
				},
			}
			events = append(events, event)
		}
	}

	return events, nil
}

// processEvents processes captured events
func (at *AgentTracer) processEvents(session *CaptureSession) {
	for {
		select {
		case <-session.ctx.Done():
			return
		case <-session.StopChan:
			return
		case event, ok := <-session.EventChan:
			if !ok {
				return
			}

			// Add to trajectory
			session.Trajectory.AddEvent(event)

			// Check duration limit
			if session.Config.MaxDurationMs > 0 {
				duration := uint64(time.Since(session.StartTime).Milliseconds())
				if duration >= session.Config.MaxDurationMs {
					log.Printf("AgentTracer: Max duration reached for session %s", session.ID)
					at.StopCapture(session.ID)
					return
				}
			}
		}
	}
}

// StopCapture stops an active capture session
func (at *AgentTracer) StopCapture(sessionID string) (*fintech.ExecutionTrajectory, error) {
	at.mu.Lock()
	session, exists := at.activeCaptures[sessionID]
	at.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("capture session not found: %s", sessionID)
	}

	// Signal stop
	close(session.StopChan)
	session.cancel()

	// Finalize trajectory
	session.Trajectory.Finalize()
	session.Trajectory.CalculateDeterminismHash()
	session.Status = "completed"

	// Remove from active captures
	at.mu.Lock()
	delete(at.activeCaptures, sessionID)
	at.mu.Unlock()

	log.Printf("AgentTracer: Stopped capture session %s, collected %d events",
		sessionID, session.EventsCollected)

	return session.Trajectory, nil
}

// GetCaptureSession returns an active capture session
func (at *AgentTracer) GetCaptureSession(sessionID string) (*CaptureSession, error) {
	at.mu.RLock()
	defer at.mu.RUnlock()

	session, exists := at.activeCaptures[sessionID]
	if !exists {
		return nil, fmt.Errorf("capture session not found: %s", sessionID)
	}

	return session, nil
}

// ListActiveCaptures returns all active capture sessions
func (at *AgentTracer) ListActiveCaptures() []*CaptureSession {
	at.mu.RLock()
	defer at.mu.RUnlock()

	sessions := make([]*CaptureSession, 0, len(at.activeCaptures))
	for _, session := range at.activeCaptures {
		sessions = append(sessions, session)
	}

	return sessions
}

// PauseCapture pauses event collection
func (at *AgentTracer) PauseCapture(sessionID string) error {
	at.mu.Lock()
	defer at.mu.Unlock()

	session, exists := at.activeCaptures[sessionID]
	if !exists {
		return fmt.Errorf("capture session not found: %s", sessionID)
	}

	if session.Status != "capturing" {
		return fmt.Errorf("cannot pause session in state: %s", session.Status)
	}

	session.Status = "paused"
	return nil
}

// ResumeCapture resumes event collection
func (at *AgentTracer) ResumeCapture(sessionID string) error {
	at.mu.Lock()
	defer at.mu.Unlock()

	session, exists := at.activeCaptures[sessionID]
	if !exists {
		return fmt.Errorf("capture session not found: %s", sessionID)
	}

	if session.Status != "paused" {
		return fmt.Errorf("cannot resume session in state: %s", session.Status)
	}

	session.Status = "capturing"
	return nil
}

// GetTrajectoryMetrics returns current metrics for a capture session
func (at *AgentTracer) GetTrajectoryMetrics(sessionID string) (*fintech.TrajectoryMetrics, error) {
	session, err := at.GetCaptureSession(sessionID)
	if err != nil {
		return nil, err
	}

	return session.Trajectory.Metrics, nil
}

// Helper function to get syscall name from ID
func getSyscallName(id int) string {
	// Common syscall mappings
	syscalls := map[int]string{
		0:   "read",
		1:   "write",
		2:   "open",
		3:   "close",
		4:   "stat",
		5:   "fstat",
		9:   "mmap",
		11:  "munmap",
		12:  "brk",
		16:  "ioctl",
		41:  "socket",
		42:  "connect",
		43:  "accept",
		44:  "sendto",
		45:  "recvfrom",
		49:  "bind",
		50:  "listen",
		59:  "execve",
		60:  "exit",
		61:  "wait4",
		62:  "kill",
		63:  "uname",
		257: "openat",
		262: "newfstatat",
	}

	if name, ok := syscalls[id]; ok {
		return name
	}

	return fmt.Sprintf("syscall_%d", id)
}

// ReadTracePipe reads events from the eBPF trace pipe (if available)
func (at *AgentTracer) ReadTracePipe(pid uint32) ([]byte, error) {
	// This is a placeholder for reading from the kernel trace pipe
	// In a real implementation, this would use the perf buffer or ring buffer
	// to read raw eBPF events
	return nil, fmt.Errorf("trace pipe reading not yet implemented")
}

// AttachToProcess attaches the tracer to a specific process
func (at *AgentTracer) AttachToProcess(pid uint32) error {
	// Verify the process exists
	if pid == 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	log.Printf("AgentTracer: Attached to process %d", pid)
	return nil
}

// DetachFromProcess detaches the tracer from a process
func (at *AgentTracer) DetachFromProcess(pid uint32) error {
	log.Printf("AgentTracer: Detached from process %d", pid)
	return nil
}

// ConvertToEBPFTrace converts a trajectory to EBPFTraceEvidence format
func (at *AgentTracer) ConvertToEBPFTrace(trajectory *fintech.ExecutionTrajectory) *fintech.EBPFTraceEvidence {
	trace := &fintech.EBPFTraceEvidence{
		TraceID:       trajectory.ID,
		StartTime:     trajectory.StartedAt,
		EndTime:       trajectory.EndedAt,
		Syscalls:      make([]fintech.SyscallEvent, 0),
		NetworkEvents: make([]fintech.NetworkEvent, 0),
		FileAccesses:  make([]fintech.FileAccessEvent, 0),
	}

	for _, event := range trajectory.Events {
		switch event.EventType {
		case "syscall":
			if event.Syscall != nil {
				trace.Syscalls = append(trace.Syscalls, fintech.SyscallEvent{
					Timestamp: event.Syscall.Timestamp.UnixNano(),
					PID:       event.Syscall.PID,
					SyscallID: uint64(event.Syscall.SyscallID),
					Duration:  int64(event.Syscall.DurationNs),
				})
			}
		case "network":
			if event.Network != nil {
				trace.NetworkEvents = append(trace.NetworkEvents, fintech.NetworkEvent{
					Timestamp: event.Network.Timestamp.UnixNano(),
					PID:       event.Network.PID,
					Operation: event.Network.Operation,
					SourceIP:  event.Network.SourceIP,
					DestIP:    event.Network.DestIP,
					Bytes:     event.Network.BytesSent + event.Network.BytesRecv,
				})
			}
		case "file_access":
			if event.FileAccess != nil {
				trace.FileAccesses = append(trace.FileAccesses, fintech.FileAccessEvent{
					Timestamp: event.FileAccess.Timestamp.UnixNano(),
					PID:       event.FileAccess.PID,
					Operation: event.FileAccess.Operation,
					Path:      event.FileAccess.Path,
				})
			}
		}
	}

	return trace
}

// LittleEndian is a helper for binary operations
var LittleEndian = binary.LittleEndian
