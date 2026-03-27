// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package fintech

import (
	"testing"
	"time"
)

func TestNewExecutionTrajectory(t *testing.T) {
	traj := NewExecutionTrajectory("agent-123", "Test Agent", "validation-456")

	if traj.ID == "" {
		t.Error("Expected trajectory ID to be set")
	}

	if traj.AgentID != "agent-123" {
		t.Errorf("Expected AgentID to be 'agent-123', got %s", traj.AgentID)
	}

	if traj.ValidationID != "validation-456" {
		t.Errorf("Expected ValidationID to be 'validation-456', got %s", traj.ValidationID)
	}

	if traj.Status != TrajectoryStatusCapturing {
		t.Errorf("Expected status to be CAPTURING, got %s", traj.Status)
	}

	if len(traj.Events) != 0 {
		t.Error("Expected events to be empty")
	}
}

func TestExecutionTrajectoryAddEvent(t *testing.T) {
	traj := NewExecutionTrajectory("agent-123", "Test Agent", "validation-456")

	event := &ExecutionPoint{
		Timestamp:   time.Now(),
		SequenceNum: 1,
		PID:         1234,
		EventType:   "syscall",
		Syscall: &TrajectorySyscallEvent{
			Timestamp:   time.Now(),
			SyscallID:   0, // read
			SyscallName: "read",
			ReturnValue: 100,
		},
	}

	traj.AddEvent(event)

	if traj.EventCount != 1 {
		t.Errorf("Expected EventCount to be 1, got %d", traj.EventCount)
	}

	if len(traj.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(traj.Events))
	}

	// Check metrics updated
	if traj.Metrics.TotalSyscalls != 1 {
		t.Errorf("Expected TotalSyscalls to be 1, got %d", traj.Metrics.TotalSyscalls)
	}

	if traj.Metrics.SyscallFrequency["read"] != 1 {
		t.Errorf("Expected SyscallFrequency['read'] to be 1, got %d", traj.Metrics.SyscallFrequency["read"])
	}
}

func TestExecutionTrajectoryFinalize(t *testing.T) {
	traj := NewExecutionTrajectory("agent-123", "Test Agent", "validation-456")

	// Add some events
	for i := 0; i < 5; i++ {
		traj.AddEvent(&ExecutionPoint{
			Timestamp:   time.Now(),
			SequenceNum: uint64(i + 1),
			PID:         1234,
			EventType:   "syscall",
			Syscall: &TrajectorySyscallEvent{
				Timestamp:   time.Now(),
				SyscallID:   0,
				SyscallName: "read",
			},
		})
	}

	time.Sleep(10 * time.Millisecond) // Ensure some duration
	traj.Finalize()

	if traj.Status != TrajectoryStatusCaptured {
		t.Errorf("Expected status to be CAPTURED, got %s", traj.Status)
	}

	if traj.EndedAt.IsZero() {
		t.Error("Expected EndedAt to be set")
	}

	if traj.Metrics.DurationMs == 0 {
		t.Error("Expected DurationMs to be greater than 0")
	}

	if traj.Metrics.UniqueSyscalls != 1 {
		t.Errorf("Expected UniqueSyscalls to be 1, got %d", traj.Metrics.UniqueSyscalls)
	}
}

func TestExecutionTrajectoryCalculateDeterminismHash(t *testing.T) {
	traj := NewExecutionTrajectory("agent-123", "Test Agent", "validation-456")

	// Add events with consistent return values
	traj.AddEvent(&ExecutionPoint{
		Timestamp:   time.Now(),
		SequenceNum: 1,
		PID:         1234,
		EventType:   "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallID:   0,
			SyscallName: "read",
			ReturnValue: 100,
		},
	})

	traj.AddEvent(&ExecutionPoint{
		Timestamp:   time.Now(),
		SequenceNum: 2,
		PID:         1234,
		EventType:   "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallID:   1,
			SyscallName: "write",
			ReturnValue: 50,
		},
	})

	hash := traj.CalculateDeterminismHash()

	if hash == "" {
		t.Error("Expected determinism hash to be set")
	}

	if traj.DeterminismHash != hash {
		t.Error("Expected trajectory DeterminismHash to match returned hash")
	}
}

func TestDefaultTrajectoryCaptureConfig(t *testing.T) {
	config := DefaultTrajectoryCaptureConfig()

	if !config.CaptureSyscalls {
		t.Error("Expected CaptureSyscalls to be true")
	}

	if !config.CaptureFiles {
		t.Error("Expected CaptureFiles to be true")
	}

	if !config.CaptureNetwork {
		t.Error("Expected CaptureNetwork to be true")
	}

	if config.CaptureMemory {
		t.Error("Expected CaptureMemory to be false")
	}

	if config.MaxEvents != 1000000 {
		t.Errorf("Expected MaxEvents to be 1000000, got %d", config.MaxEvents)
	}

	if config.MaxDurationMs != 300000 {
		t.Errorf("Expected MaxDurationMs to be 300000, got %d", config.MaxDurationMs)
	}
}

func TestDefaultTrajectoryReplayConfig(t *testing.T) {
	config := DefaultTrajectoryReplayConfig()

	if config.ReplaySpeed != 1.0 {
		t.Errorf("Expected ReplaySpeed to be 1.0, got %f", config.ReplaySpeed)
	}

	if config.SkipDelays {
		t.Error("Expected SkipDelays to be false")
	}

	if !config.VerifyDeterminism {
		t.Error("Expected VerifyDeterminism to be true")
	}

	if !config.StopOnMismatch {
		t.Error("Expected StopOnMismatch to be true")
	}

	if config.InjectFaults {
		t.Error("Expected InjectFaults to be false")
	}
}

func TestExecutionTrajectoryToMarkdown(t *testing.T) {
	traj := NewExecutionTrajectory("agent-123", "Test Agent", "validation-456")
	traj.AddEvent(&ExecutionPoint{
		Timestamp:   time.Now(),
		SequenceNum: 1,
		PID:         1234,
		EventType:   "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallID:   0,
			SyscallName: "read",
		},
	})
	traj.Finalize()

	md, err := traj.ToMarkdown()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(md) == 0 {
		t.Error("Expected markdown content")
	}

	// Check for expected content
	content := string(md)
	if content == "" {
		t.Error("Expected non-empty markdown content")
	}
}

func TestTrajectoryComparisonResult(t *testing.T) {
	result := &TrajectoryComparisonResult{
		BaseTrajectoryID:    "traj-1",
		CompareTrajectoryID: "traj-2",
		SimilarityScore:     0.95,
		AreEquivalent:       true,
		EventCountDiff:      0,
		SyscallDiffs:        []*SyscallDiff{},
		BehavioralDiffs:     []*BehavioralDiff{},
	}

	if result.BaseTrajectoryID != "traj-1" {
		t.Errorf("Expected BaseTrajectoryID to be 'traj-1', got %s", result.BaseTrajectoryID)
	}

	if result.CompareTrajectoryID != "traj-2" {
		t.Errorf("Expected CompareTrajectoryID to be 'traj-2', got %s", result.CompareTrajectoryID)
	}

	if result.SimilarityScore != 0.95 {
		t.Errorf("Expected SimilarityScore to be 0.95, got %f", result.SimilarityScore)
	}

	if !result.AreEquivalent {
		t.Error("Expected AreEquivalent to be true")
	}

	if result.EventCountDiff != 0 {
		t.Errorf("Expected EventCountDiff to be 0, got %d", result.EventCountDiff)
	}

	if len(result.SyscallDiffs) != 0 {
		t.Errorf("Expected SyscallDiffs to be empty, got length %d", len(result.SyscallDiffs))
	}

	if len(result.BehavioralDiffs) != 0 {
		t.Errorf("Expected BehavioralDiffs to be empty, got length %d", len(result.BehavioralDiffs))
	}
}

func TestReplayResult(t *testing.T) {
	result := &ReplayResult{
		TrajectoryID:     "traj-123",
		ReplayID:         "replay-456",
		Status:           "success",
		EventsReplayed:   100,
		EventsMatched:    95,
		EventsMismatched: 5,
		IsDeterministic:  true,
		DeterminismScore: 0.95,
		Mismatches:       make([]*EventMismatch, 0),
	}

	if result.TrajectoryID != "traj-123" {
		t.Errorf("Expected TrajectoryID to be 'traj-123', got %s", result.TrajectoryID)
	}

	if result.ReplayID != "replay-456" {
		t.Errorf("Expected ReplayID to be 'replay-456', got %s", result.ReplayID)
	}

	if result.Status != "success" {
		t.Errorf("Expected Status to be 'success', got %s", result.Status)
	}

	if result.EventsReplayed != 100 {
		t.Errorf("Expected EventsReplayed to be 100, got %d", result.EventsReplayed)
	}

	if result.EventsMatched != 95 {
		t.Errorf("Expected EventsMatched to be 95, got %d", result.EventsMatched)
	}

	if result.EventsMismatched != 5 {
		t.Errorf("Expected EventsMismatched to be 5, got %d", result.EventsMismatched)
	}

	if !result.IsDeterministic {
		t.Error("Expected IsDeterministic to be true")
	}

	if result.DeterminismScore != 0.95 {
		t.Errorf("Expected DeterminismScore to be 0.95, got %f", result.DeterminismScore)
	}

	if len(result.Mismatches) != 0 {
		t.Errorf("Expected Mismatches to be empty, got length %d", len(result.Mismatches))
	}
}

func TestEventMismatch(t *testing.T) {
	mismatch := &EventMismatch{
		SequenceNum:  5,
		ExpectedType: "syscall",
		ActualType:   "file_access",
		MismatchType: "type",
		Difference:   "Event type mismatch",
	}

	if mismatch.SequenceNum != 5 {
		t.Errorf("Expected SequenceNum to be 5, got %d", mismatch.SequenceNum)
	}

	if mismatch.ExpectedType != "syscall" {
		t.Errorf("Expected ExpectedType to be 'syscall', got %s", mismatch.ExpectedType)
	}

	if mismatch.ActualType != "file_access" {
		t.Errorf("Expected ActualType to be 'file_access', got %s", mismatch.ActualType)
	}

	if mismatch.MismatchType != "type" {
		t.Errorf("Expected MismatchType to be 'type', got %s", mismatch.MismatchType)
	}

	if mismatch.Difference != "Event type mismatch" {
		t.Errorf("Expected Difference to be 'Event type mismatch', got %s", mismatch.Difference)
	}
}

func TestPerformanceDelta(t *testing.T) {
	delta := &PerformanceDelta{
		DurationDeltaMs:       100,
		SyscallCountDelta:     5,
		MemoryUsageDeltaBytes: 1024,
		NetworkBytesDelta:     2048,
		PerformanceChangePct:  2.5,
	}

	if delta.DurationDeltaMs != 100 {
		t.Errorf("Expected DurationDeltaMs to be 100, got %d", delta.DurationDeltaMs)
	}

	if delta.SyscallCountDelta != 5 {
		t.Errorf("Expected SyscallCountDelta to be 5, got %d", delta.SyscallCountDelta)
	}

	if delta.MemoryUsageDeltaBytes != 1024 {
		t.Errorf("Expected MemoryUsageDeltaBytes to be 1024, got %d", delta.MemoryUsageDeltaBytes)
	}

	if delta.NetworkBytesDelta != 2048 {
		t.Errorf("Expected NetworkBytesDelta to be 2048, got %d", delta.NetworkBytesDelta)
	}

	if delta.PerformanceChangePct != 2.5 {
		t.Errorf("Expected PerformanceChangePct to be 2.5, got %f", delta.PerformanceChangePct)
	}
}

func BenchmarkExecutionTrajectoryAddEvent(b *testing.B) {
	traj := NewExecutionTrajectory("agent-123", "Test Agent", "validation-456")

	event := &ExecutionPoint{
		Timestamp:   time.Now(),
		SequenceNum: 1,
		PID:         1234,
		EventType:   "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallID:   0,
			SyscallName: "read",
			ReturnValue: 100,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		traj.AddEvent(event)
	}
}
