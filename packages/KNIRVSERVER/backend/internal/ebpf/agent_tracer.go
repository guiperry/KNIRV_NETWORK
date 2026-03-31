// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type ExecutionTrajectory struct {
	ID           string
	AgentID      string
	ValidationID string
	ProcessID    uint32
	StartTime    time.Time
	EndTime      time.Time
	Events       []*ExecutionPoint
}

func NewExecutionTrajectory(agentID, validationID string) *ExecutionTrajectory {
	return &ExecutionTrajectory{
		AgentID:      agentID,
		ValidationID: validationID,
		StartTime:    time.Now(),
		Events:       make([]*ExecutionPoint, 0),
	}
}

func (t *ExecutionTrajectory) AddEvent(event *ExecutionPoint) {
	t.Events = append(t.Events, event)
}

func (t *ExecutionTrajectory) Finalize() {
	t.EndTime = time.Now()
}

func (t *ExecutionTrajectory) CalculateDeterminismHash() string {
	return fmt.Sprintf("hash-%s", t.ID)
}

type ExecutionPoint struct {
	SequenceNum uint64
	Timestamp   time.Time
	EventType   string
	PID         uint32
	Syscall     *TrajectorySyscallEvent
	Memory      *TrajectoryMemoryEvent
	Network     *TrajectoryNetworkEvent
	FileAccess  *TrajectoryFileAccessEvent
}

type TrajectoryCaptureConfig struct {
	CaptureSyscalls   bool
	CaptureMemory     bool
	CaptureNetwork    bool
	CaptureFileAccess bool
	SamplingRate      int
	MaxPoints         int
	MaxEvents         int
	MaxDurationMs     int
}

func DefaultTrajectoryCaptureConfig() *TrajectoryCaptureConfig {
	return &TrajectoryCaptureConfig{
		CaptureSyscalls:   true,
		CaptureMemory:     false,
		CaptureNetwork:    false,
		CaptureFileAccess: false,
		SamplingRate:      1,
		MaxPoints:         10000,
		MaxEvents:         10000,
		MaxDurationMs:     300000,
	}
}

type TrajectorySyscallEvent struct {
	SyscallID   int
	SyscallName string
	Timestamp   time.Time
	Args        []string
	Result      string
	PID         uint32
	Duration    time.Duration
}

type TrajectoryMemoryEvent struct {
	Timestamp time.Time
	Address   uint64
	Size      uint64
	Operation string
	PID       uint32
}

type TrajectoryNetworkEvent struct {
	Timestamp  time.Time
	Protocol   string
	RemoteAddr string
	RemotePort uint16
	Direction  string
	Operation  string
	SourceIP   string
	DestIP     string
	Bytes      uint64
	PID        uint32
}

type TrajectoryFileAccessEvent struct {
	Timestamp   time.Time
	Path        string
	Operation   string
	Permissions string
	PID         uint32
}

type TrajectoryMetrics struct {
	TotalSyscalls       int
	UniqueSyscalls      int
	NetworkConnections  int
	FileAccessCount     int
	MemoryHighWaterMark uint64
	Duration            time.Duration
}

type EBPFTraceEvidence struct {
	TraceID       string
	AgentID       string
	ValidationID  string
	StartTime     time.Time
	EndTime       time.Time
	Syscalls      []TrajectorySyscallEvent
	NetworkEvents []TrajectoryNetworkEvent
	FileAccesses  []TrajectoryFileAccessEvent
}

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
	Trajectory      *ExecutionTrajectory
	Config          *TrajectoryCaptureConfig
	StartTime       time.Time
	EventsCollected uint64
	Status          string
	EventChan       chan *ExecutionPoint
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
func (at *AgentTracer) StartCapture(ctx context.Context, agentID, validationID string, pid uint32, config *TrajectoryCaptureConfig) (*CaptureSession, error) {
	if config == nil {
		config = DefaultTrajectoryCaptureConfig()
	}

	if at.manager == nil {
		return nil, fmt.Errorf("eBPF manager not initialized")
	}

	sessionCtx, cancel := context.WithCancel(ctx)
	session := &CaptureSession{
		ID:           fmt.Sprintf("capture-%d", time.Now().UnixNano()),
		AgentID:      agentID,
		ValidationID: validationID,
		ProcessID:    pid,
		Trajectory:   NewExecutionTrajectory(agentID, validationID),
		Config:       config,
		StartTime:    time.Now(),
		Status:       "capturing",
		EventChan:    make(chan *ExecutionPoint, at.eventBufferSize),
		ErrorChan:    make(chan error, 100),
		StopChan:     make(chan struct{}),
		ctx:          sessionCtx,
		cancel:       cancel,
	}

	session.Trajectory.AgentID = agentID
	session.Trajectory.ValidationID = validationID
	session.Trajectory.ProcessID = pid

	at.mu.Lock()
	at.activeCaptures[session.ID] = session
	at.mu.Unlock()

	go at.captureEvents(session)
	go at.processEvents(session)

	log.Printf("AgentTracer: Started capture session %s for agent %s (PID: %d)", session.ID, agentID, pid)

	return session, nil
}

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
			events, err := at.pollEvents(session)
			if err != nil {
				select {
				case session.ErrorChan <- err:
				default:
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

				if session.Config.MaxEvents > 0 && int(session.EventsCollected) >= session.Config.MaxEvents {
					log.Printf("AgentTracer: Max events reached for session %s", session.ID)
					return
				}
			}
		}
	}
}

func (at *AgentTracer) pollEvents(session *CaptureSession) ([]*ExecutionPoint, error) {
	if at.manager == nil || !at.manager.initialized {
		return nil, fmt.Errorf("eBPF manager not initialized")
	}

	metrics, err := at.manager.GetProcessMetrics()
	if err != nil {
		return []*ExecutionPoint{}, nil
	}

	var events []*ExecutionPoint
	now := time.Now()

	for pid, stats := range metrics {
		if session.ProcessID != 0 && pid != session.ProcessID {
			continue
		}

		if session.Config.CaptureSyscalls {
			for syscallID, count := range stats.SyscallCount {
				if count > 0 {
					event := &ExecutionPoint{
						Timestamp: now,
						PID:       pid,
						EventType: "syscall",
						Syscall: &TrajectorySyscallEvent{
							Timestamp:   now,
							PID:         pid,
							SyscallID:   int(syscallID),
							SyscallName: getSyscallName(syscallID),
						},
					}
					events = append(events, event)
				}
			}
		}

		if session.Config.CaptureMemory {
			event := &ExecutionPoint{
				Timestamp: now,
				PID:       pid,
				EventType: "memory",
				Memory: &TrajectoryMemoryEvent{
					Timestamp: now,
					PID:       pid,
					Operation: "usage",
					Size:      stats.MemoryBytes,
				},
			}
			events = append(events, event)
		}

		if session.Config.CaptureNetwork && (stats.NetTxBytes > 0 || stats.NetRxBytes > 0) {
			event := &ExecutionPoint{
				Timestamp: now,
				PID:       pid,
				EventType: "network",
				Network: &TrajectoryNetworkEvent{
					Timestamp: now,
					PID:       pid,
					Operation: "transfer",
					Bytes:     stats.NetTxBytes + stats.NetRxBytes,
				},
			}
			events = append(events, event)
		}
	}

	return events, nil
}

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

			session.Trajectory.AddEvent(event)

			if session.Config.MaxDurationMs > 0 {
				duration := int(time.Since(session.StartTime).Milliseconds())
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
func (at *AgentTracer) StopCapture(sessionID string) (*ExecutionTrajectory, error) {
	at.mu.Lock()
	session, exists := at.activeCaptures[sessionID]
	at.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("capture session not found: %s", sessionID)
	}

	close(session.StopChan)
	session.cancel()

	session.Trajectory.Finalize()
	session.Trajectory.CalculateDeterminismHash()
	session.Status = "completed"

	at.mu.Lock()
	delete(at.activeCaptures, sessionID)
	at.mu.Unlock()

	log.Printf("AgentTracer: Stopped capture session %s, collected %d events", sessionID, session.EventsCollected)

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

// GetTrajectoryMetrics returns metrics for a capture session
func (at *AgentTracer) GetTrajectoryMetrics(sessionID string) (*TrajectoryMetrics, error) {
	at.mu.RLock()
	session, exists := at.activeCaptures[sessionID]
	at.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("capture session not found: %s", sessionID)
	}

	metrics := &TrajectoryMetrics{
		TotalSyscalls:      len(session.Trajectory.Events),
		UniqueSyscalls:     0,
		NetworkConnections: 0,
		FileAccessCount:    0,
		Duration:           time.Since(session.StartTime),
	}

	return metrics, nil
}

// ConvertToEBPFTrace converts execution trajectory to eBPF trace evidence
func (at *AgentTracer) ConvertToEBPFTrace(trajectory *ExecutionTrajectory) *EBPFTraceEvidence {
	trace := &EBPFTraceEvidence{
		TraceID:       trajectory.ID,
		AgentID:       trajectory.AgentID,
		ValidationID:  trajectory.ValidationID,
		StartTime:     trajectory.StartTime,
		EndTime:       trajectory.EndTime,
		Syscalls:      make([]TrajectorySyscallEvent, 0),
		NetworkEvents: make([]TrajectoryNetworkEvent, 0),
		FileAccesses:  make([]TrajectoryFileAccessEvent, 0),
	}

	for _, event := range trajectory.Events {
		if event.Syscall != nil {
			trace.Syscalls = append(trace.Syscalls, *event.Syscall)
		}
		if event.Network != nil {
			trace.NetworkEvents = append(trace.NetworkEvents, *event.Network)
		}
		if event.FileAccess != nil {
			trace.FileAccesses = append(trace.FileAccesses, *event.FileAccess)
		}
	}

	return trace
}

// getSyscallName returns a human-readable name for a syscall number
func getSyscallName(syscallID int) string {
	syscallNames := map[int]string{
		0:   "read",
		1:   "write",
		2:   "open",
		3:   "close",
		4:   "stat",
		5:   "fstat",
		9:   "mmap",
		10:  "mprotect",
		11:  "munmap",
		12:  "brk",
		21:  "access",
		59:  "execve",
		60:  "exit",
		231: "exit_group",
	}

	if name, ok := syscallNames[syscallID]; ok {
		return name
	}
	return fmt.Sprintf("syscall_%d", syscallID)
}
