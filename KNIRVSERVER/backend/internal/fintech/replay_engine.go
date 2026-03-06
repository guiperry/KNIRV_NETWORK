// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package fintech

import (
	"context"
	"fmt"
	"log"
	"math"
	"reflect"
	"sync"
	"time"
)

// ReplayEngine provides deterministic replay capabilities for execution trajectories
type ReplayEngine struct {
	activeReplays map[string]*ReplaySession
	mu            sync.RWMutex
	comparator    *TrajectoryComparator
}

// ReplaySession represents an active replay operation
type ReplaySession struct {
	ID              string
	BaseTrajectory  *ExecutionTrajectory
	Config          *TrajectoryReplayConfig
	Result          *ReplayResult
	Status          string // preparing, running, paused, completed, failed
	StartTime       time.Time
	EndTime         *time.Time
	CurrentEventIdx uint64
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewReplayEngine creates a new replay engine
func NewReplayEngine() *ReplayEngine {
	return &ReplayEngine{
		activeReplays: make(map[string]*ReplaySession),
		comparator:    NewTrajectoryComparator(),
	}
}

// StartReplay begins replaying a trajectory
func (re *ReplayEngine) StartReplay(ctx context.Context, trajectory *ExecutionTrajectory, config *TrajectoryReplayConfig) (*ReplaySession, error) {
	if trajectory == nil {
		return nil, fmt.Errorf("trajectory cannot be nil")
	}

	if config == nil {
		config = DefaultTrajectoryReplayConfig()
	}

	if trajectory.Status != TrajectoryStatusCaptured {
		return nil, fmt.Errorf("trajectory must be in CAPTURED state, current: %s", trajectory.Status)
	}

	sessionCtx, cancel := context.WithCancel(ctx)
	session := &ReplaySession{
		ID:              fmt.Sprintf("replay-%d", time.Now().UnixNano()),
		BaseTrajectory:  trajectory,
		Config:          config,
		Status:          "preparing",
		StartTime:       time.Now(),
		CurrentEventIdx: 0,
		ctx:             sessionCtx,
		cancel:          cancel,
		Result: &ReplayResult{
			TrajectoryID: trajectory.ID,
			ReplayID:     fmt.Sprintf("replay-%d", time.Now().UnixNano()),
			StartedAt:    time.Now(),
			Status:       "running",
			Mismatches:   make([]*EventMismatch, 0),
		},
	}

	// Store session
	re.mu.Lock()
	re.activeReplays[session.ID] = session
	re.mu.Unlock()

	// Start replay goroutine
	go re.executeReplay(session)

	log.Printf("ReplayEngine: Started replay session %s for trajectory %s",
		session.ID, trajectory.ID)

	return session, nil
}

// executeReplay executes the replay
func (re *ReplayEngine) executeReplay(session *ReplaySession) {
	session.Status = "running"
	trajectory := session.BaseTrajectory

	// Create a simulated execution trace
	simulatedEvents := make([]*ExecutionPoint, 0, len(trajectory.Events))

	for i, expectedEvent := range trajectory.Events {
		select {
		case <-session.ctx.Done():
			session.Result.Status = "cancelled"
			return
		default:
		}

		session.CurrentEventIdx = uint64(i)

		// Simulate the event (in real implementation, this would run the actual agent)
		simulatedEvent := re.simulateEvent(expectedEvent, session.Config)
		simulatedEvents = append(simulatedEvents, simulatedEvent)

		// Compare expected vs actual
		if session.Config.VerifyDeterminism {
			mismatch := re.compareEvents(expectedEvent, simulatedEvent, uint64(i))
			if mismatch != nil {
				session.Result.EventsMismatched++
				session.Result.Mismatches = append(session.Result.Mismatches, mismatch)

				if session.Config.StopOnMismatch {
					session.Result.Status = "failed"
					session.Result.ErrorMessage = fmt.Sprintf("Mismatch at event %d: %s", i, mismatch.Difference)
					re.finalizeReplay(session)
					return
				}
			} else {
				session.Result.EventsMatched++
			}
		}

		session.Result.EventsReplayed++

		// Apply replay speed delay
		if !session.Config.SkipDelays && i < len(trajectory.Events)-1 {
			nextEvent := trajectory.Events[i+1]
			delay := time.Duration(float64(nextEvent.Timestamp.Sub(expectedEvent.Timestamp)) / session.Config.ReplaySpeed)
			if delay > 0 {
				time.Sleep(delay)
			}
		}
	}

	// Calculate determinism score
	if session.Result.EventsReplayed > 0 {
		session.Result.DeterminismScore = float64(session.Result.EventsMatched) / float64(session.Result.EventsReplayed)
		session.Result.IsDeterministic = session.Result.DeterminismScore >= 0.95 // 95% threshold
	}

	// Calculate performance delta
	session.Result.PerformanceDelta = re.calculatePerformanceDelta(trajectory, simulatedEvents)

	session.Result.Status = "success"
	re.finalizeReplay(session)

	log.Printf("ReplayEngine: Completed replay session %s - Events: %d, Matched: %d, Mismatched: %d, Score: %.2f%%",
		session.ID, session.Result.EventsReplayed, session.Result.EventsMatched,
		session.Result.EventsMismatched, session.Result.DeterminismScore*100)
}

// simulateEvent simulates an event (placeholder for actual agent execution)
func (re *ReplayEngine) simulateEvent(expected *ExecutionPoint, config *TrajectoryReplayConfig) *ExecutionPoint {
	simulated := &ExecutionPoint{
		Timestamp:   time.Now(),
		SequenceNum: expected.SequenceNum,
		PID:         expected.PID,
		EventType:   expected.EventType,
	}

	// Copy event data
	switch expected.EventType {
	case "syscall":
		if expected.Syscall != nil {
			simulated.Syscall = &TrajectorySyscallEvent{
				Timestamp:   time.Now(),
				SequenceNum: expected.Syscall.SequenceNum,
				PID:         expected.Syscall.PID,
				SyscallID:   expected.Syscall.SyscallID,
				SyscallName: expected.Syscall.SyscallName,
				ProcessName: expected.Syscall.ProcessName,
			}

			// Fault injection for testing
			if config.InjectFaults && config.FaultInjectionRate > 0 {
				if randFloat() < config.FaultInjectionRate {
					simulated.Syscall.ReturnValue = -1 // Simulate error
				} else {
					simulated.Syscall.ReturnValue = expected.Syscall.ReturnValue
				}
			} else {
				simulated.Syscall.ReturnValue = expected.Syscall.ReturnValue
			}
		}
	case "file_access":
		if expected.FileAccess != nil {
			simulated.FileAccess = &TrajectoryFileAccessEvent{
				Timestamp:   time.Now(),
				SequenceNum: expected.FileAccess.SequenceNum,
				PID:         expected.FileAccess.PID,
				Operation:   expected.FileAccess.Operation,
				Path:        expected.FileAccess.Path,
				ReturnValue: expected.FileAccess.ReturnValue,
			}
		}
	case "network":
		if expected.Network != nil {
			simulated.Network = &TrajectoryNetworkEvent{
				Timestamp:   time.Now(),
				SequenceNum: expected.Network.SequenceNum,
				PID:         expected.Network.PID,
				Operation:   expected.Network.Operation,
				SourceIP:    expected.Network.SourceIP,
				DestIP:      expected.Network.DestIP,
				ReturnValue: expected.Network.ReturnValue,
			}
		}
	case "memory":
		if expected.Memory != nil {
			simulated.Memory = &TrajectoryMemoryEvent{
				Timestamp:   time.Now(),
				SequenceNum: expected.Memory.SequenceNum,
				PID:         expected.Memory.PID,
				Operation:   expected.Memory.Operation,
				Size:        expected.Memory.Size,
				ReturnValue: expected.Memory.ReturnValue,
			}
		}
	}

	return simulated
}

// compareEvents compares expected and actual events
func (re *ReplayEngine) compareEvents(expected, actual *ExecutionPoint, sequenceNum uint64) *EventMismatch {
	// Check event type match
	if expected.EventType != actual.EventType {
		return &EventMismatch{
			SequenceNum:  sequenceNum,
			ExpectedType: expected.EventType,
			ActualType:   actual.EventType,
			MismatchType: "type",
			Difference:   fmt.Sprintf("Event type mismatch: expected %s, got %s", expected.EventType, actual.EventType),
		}
	}

	// Compare based on event type
	switch expected.EventType {
	case "syscall":
		return re.compareSyscallEvents(expected, actual, sequenceNum)
	case "file_access":
		return re.compareFileAccessEvents(expected, actual, sequenceNum)
	case "network":
		return re.compareNetworkEvents(expected, actual, sequenceNum)
	case "memory":
		return re.compareMemoryEvents(expected, actual, sequenceNum)
	}

	return nil
}

// compareSyscallEvents compares syscall events
func (re *ReplayEngine) compareSyscallEvents(expected, actual *ExecutionPoint, sequenceNum uint64) *EventMismatch {
	if expected.Syscall == nil || actual.Syscall == nil {
		return &EventMismatch{
			SequenceNum:  sequenceNum,
			MismatchType: "nil_event",
			Difference:   "Syscall event is nil",
		}
	}

	es := expected.Syscall
	as := actual.Syscall

	// Check syscall name
	if es.SyscallName != as.SyscallName {
		return &EventMismatch{
			SequenceNum:  sequenceNum,
			ExpectedType: "syscall",
			ActualType:   "syscall",
			MismatchType: "syscall_name",
			Difference:   fmt.Sprintf("Syscall name mismatch: expected %s, got %s", es.SyscallName, as.SyscallName),
		}
	}

	// Check return value (allowing for some variance in non-deterministic syscalls)
	if es.ReturnValue != as.ReturnValue {
		// Some syscalls are inherently non-deterministic (e.g., gettimeofday)
		if !isNonDeterministicSyscall(es.SyscallName) {
			return &EventMismatch{
				SequenceNum:  sequenceNum,
				ExpectedType: "syscall",
				ActualType:   "syscall",
				MismatchType: "return_value",
				Difference:   fmt.Sprintf("Return value mismatch for %s: expected %d, got %d", es.SyscallName, es.ReturnValue, as.ReturnValue),
			}
		}
	}

	return nil
}

// compareFileAccessEvents compares file access events
func (re *ReplayEngine) compareFileAccessEvents(expected, actual *ExecutionPoint, sequenceNum uint64) *EventMismatch {
	if expected.FileAccess == nil || actual.FileAccess == nil {
		return &EventMismatch{
			SequenceNum:  sequenceNum,
			MismatchType: "nil_event",
			Difference:   "File access event is nil",
		}
	}

	ef := expected.FileAccess
	af := actual.FileAccess

	if ef.Operation != af.Operation {
		return &EventMismatch{
			SequenceNum:  sequenceNum,
			ExpectedType: "file_access",
			ActualType:   "file_access",
			MismatchType: "operation",
			Difference:   fmt.Sprintf("Operation mismatch: expected %s, got %s", ef.Operation, af.Operation),
		}
	}

	if ef.Path != af.Path {
		return &EventMismatch{
			SequenceNum:  sequenceNum,
			ExpectedType: "file_access",
			ActualType:   "file_access",
			MismatchType: "path",
			Difference:   fmt.Sprintf("Path mismatch: expected %s, got %s", ef.Path, af.Path),
		}
	}

	return nil
}

// compareNetworkEvents compares network events
func (re *ReplayEngine) compareNetworkEvents(expected, actual *ExecutionPoint, sequenceNum uint64) *EventMismatch {
	if expected.Network == nil || actual.Network == nil {
		return &EventMismatch{
			SequenceNum:  sequenceNum,
			MismatchType: "nil_event",
			Difference:   "Network event is nil",
		}
	}

	en := expected.Network
	an := actual.Network

	if en.Operation != an.Operation {
		return &EventMismatch{
			SequenceNum:  sequenceNum,
			ExpectedType: "network",
			ActualType:   "network",
			MismatchType: "operation",
			Difference:   fmt.Sprintf("Operation mismatch: expected %s, got %s", en.Operation, an.Operation),
		}
	}

	if en.DestIP != an.DestIP {
		return &EventMismatch{
			SequenceNum:  sequenceNum,
			ExpectedType: "network",
			ActualType:   "network",
			MismatchType: "dest_ip",
			Difference:   fmt.Sprintf("Destination IP mismatch: expected %s, got %s", en.DestIP, an.DestIP),
		}
	}

	return nil
}

// compareMemoryEvents compares memory events
func (re *ReplayEngine) compareMemoryEvents(expected, actual *ExecutionPoint, sequenceNum uint64) *EventMismatch {
	if expected.Memory == nil || actual.Memory == nil {
		return &EventMismatch{
			SequenceNum:  sequenceNum,
			MismatchType: "nil_event",
			Difference:   "Memory event is nil",
		}
	}

	em := expected.Memory
	am := actual.Memory

	if em.Operation != am.Operation {
		return &EventMismatch{
			SequenceNum:  sequenceNum,
			ExpectedType: "memory",
			ActualType:   "memory",
			MismatchType: "operation",
			Difference:   fmt.Sprintf("Operation mismatch: expected %s, got %s", em.Operation, am.Operation),
		}
	}

	// Allow some variance in memory allocation sizes
	if em.Operation == "mmap" || em.Operation == "brk" {
		sizeDiff := math.Abs(float64(em.Size) - float64(am.Size))
		if sizeDiff > float64(em.Size)*0.1 { // 10% tolerance
			return &EventMismatch{
				SequenceNum:  sequenceNum,
				ExpectedType: "memory",
				ActualType:   "memory",
				MismatchType: "size",
				Difference:   fmt.Sprintf("Size mismatch for %s: expected %d, got %d", em.Operation, em.Size, am.Size),
			}
		}
	}

	return nil
}

// calculatePerformanceDelta compares performance metrics
func (re *ReplayEngine) calculatePerformanceDelta(original *ExecutionTrajectory, replayed []*ExecutionPoint) *PerformanceDelta {
	if original.Metrics == nil || len(replayed) == 0 {
		return nil
	}

	delta := &PerformanceDelta{}

	// Calculate duration delta
	originalDuration := original.EndedAt.Sub(original.StartedAt).Milliseconds()
	replayedDuration := time.Since(original.StartedAt).Milliseconds()
	delta.DurationDeltaMs = replayedDuration - originalDuration

	// Calculate syscall count delta
	originalSyscalls := original.Metrics.TotalSyscalls
	var replayedSyscalls uint64
	for _, event := range replayed {
		if event.EventType == "syscall" {
			replayedSyscalls++
		}
	}
	delta.SyscallCountDelta = int64(replayedSyscalls) - int64(originalSyscalls)

	// Calculate performance change percentage
	if originalDuration > 0 {
		delta.PerformanceChangePct = float64(delta.DurationDeltaMs) / float64(originalDuration) * 100
	}

	return delta
}

// finalizeReplay finalizes a replay session
func (re *ReplayEngine) finalizeReplay(session *ReplaySession) {
	now := time.Now()
	session.EndTime = &now
	session.Status = session.Result.Status
	session.Result.EndedAt = now
}

// StopReplay stops an active replay
func (re *ReplayEngine) StopReplay(sessionID string) (*ReplayResult, error) {
	re.mu.Lock()
	session, exists := re.activeReplays[sessionID]
	re.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("replay session not found: %s", sessionID)
	}

	session.cancel()

	// Wait for replay to finish
	for session.Status == "running" || session.Status == "preparing" {
		time.Sleep(10 * time.Millisecond)
	}

	return session.Result, nil
}

// GetReplaySession returns an active replay session
func (re *ReplayEngine) GetReplaySession(sessionID string) (*ReplaySession, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	session, exists := re.activeReplays[sessionID]
	if !exists {
		return nil, fmt.Errorf("replay session not found: %s", sessionID)
	}

	return session, nil
}

// ListActiveReplays returns all active replay sessions
func (re *ReplayEngine) ListActiveReplays() []*ReplaySession {
	re.mu.RLock()
	defer re.mu.RUnlock()

	sessions := make([]*ReplaySession, 0, len(re.activeReplays))
	for _, session := range re.activeReplays {
		sessions = append(sessions, session)
	}

	return sessions
}

// CleanupReplay removes a completed replay session
func (re *ReplayEngine) CleanupReplay(sessionID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	session, exists := re.activeReplays[sessionID]
	if !exists {
		return fmt.Errorf("replay session not found: %s", sessionID)
	}

	if session.Status == "running" || session.Status == "preparing" {
		return fmt.Errorf("cannot cleanup active replay session")
	}

	delete(re.activeReplays, sessionID)
	return nil
}

// isNonDeterministicSyscall returns true if the syscall is inherently non-deterministic
func isNonDeterministicSyscall(name string) bool {
	nonDeterministic := map[string]bool{
		"gettimeofday":  true,
		"clock_gettime": true,
		"time":          true,
		"getpid":        true,
		"gettid":        true,
		"getrandom":     true,
	}
	return nonDeterministic[name]
}

// randFloat returns a pseudo-random float between 0 and 1
func randFloat() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

// TrajectoryComparator provides advanced trajectory comparison capabilities
type TrajectoryComparator struct{}

// NewTrajectoryComparator creates a new comparator
func NewTrajectoryComparator() *TrajectoryComparator {
	return &TrajectoryComparator{}
}

// CompareTrajectories compares two trajectories and returns detailed differences
func (tc *TrajectoryComparator) CompareTrajectories(base, compare *ExecutionTrajectory) *TrajectoryComparisonResult {
	result := &TrajectoryComparisonResult{
		BaseTrajectoryID:    base.ID,
		CompareTrajectoryID: compare.ID,
		EventCountDiff:      int64(len(compare.Events)) - int64(len(base.Events)),
		SyscallDiffs:        make([]*SyscallDiff, 0),
		BehavioralDiffs:     make([]*BehavioralDiff, 0),
	}

	// Build syscall frequency maps
	baseSyscalls := make(map[string]uint64)
	compareSyscalls := make(map[string]uint64)

	for _, event := range base.Events {
		if event.Syscall != nil {
			baseSyscalls[event.Syscall.SyscallName]++
		}
	}

	for _, event := range compare.Events {
		if event.Syscall != nil {
			compareSyscalls[event.Syscall.SyscallName]++
		}
	}

	// Compare syscall frequencies
	allSyscalls := make(map[string]bool)
	for name := range baseSyscalls {
		allSyscalls[name] = true
	}
	for name := range compareSyscalls {
		allSyscalls[name] = true
	}

	for name := range allSyscalls {
		baseCount := baseSyscalls[name]
		compareCount := compareSyscalls[name]

		if baseCount != compareCount {
			result.SyscallDiffs = append(result.SyscallDiffs, &SyscallDiff{
				SyscallName:  name,
				BaseCount:    baseCount,
				CompareCount: compareCount,
				CountDiff:    int64(compareCount) - int64(baseCount),
			})
		}
	}

	// Calculate similarity score
	if len(base.Events) > 0 && len(compare.Events) > 0 {
		minEvents := len(base.Events)
		if len(compare.Events) < minEvents {
			minEvents = len(compare.Events)
		}

		matchedEvents := 0
		for i := 0; i < minEvents; i++ {
			if tc.eventsMatch(base.Events[i], compare.Events[i]) {
				matchedEvents++
			}
		}

		result.SimilarityScore = float64(matchedEvents) / float64(minEvents)
		result.AreEquivalent = result.SimilarityScore >= 0.95
	}

	return result
}

// eventsMatch checks if two events are functionally equivalent
func (tc *TrajectoryComparator) eventsMatch(e1, e2 *ExecutionPoint) bool {
	if e1.EventType != e2.EventType {
		return false
	}

	switch e1.EventType {
	case "syscall":
		if e1.Syscall == nil || e2.Syscall == nil {
			return e1.Syscall == e2.Syscall
		}
		return e1.Syscall.SyscallName == e2.Syscall.SyscallName &&
			e1.Syscall.ReturnValue == e2.Syscall.ReturnValue
	case "file_access":
		if e1.FileAccess == nil || e2.FileAccess == nil {
			return e1.FileAccess == e2.FileAccess
		}
		return e1.FileAccess.Operation == e2.FileAccess.Operation &&
			e1.FileAccess.Path == e2.FileAccess.Path
	case "network":
		if e1.Network == nil || e2.Network == nil {
			return e1.Network == e2.Network
		}
		return e1.Network.Operation == e2.Network.Operation &&
			e1.Network.DestIP == e2.Network.DestIP &&
			e1.Network.DestPort == e2.Network.DestPort
	default:
		return reflect.DeepEqual(e1, e2)
	}
}

// GetDeterminismReport generates a comprehensive determinism report
func (re *ReplayEngine) GetDeterminismReport(trajectoryID string) (map[string]interface{}, error) {
	// In a real implementation, this would query stored replay results
	return map[string]interface{}{
		"trajectory_id": trajectoryID,
		"status":        "report_generated",
		"timestamp":     time.Now().Format(time.RFC3339),
	}, nil
}
