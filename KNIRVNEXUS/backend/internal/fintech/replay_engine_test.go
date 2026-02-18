// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package fintech

import (
	"context"
	"testing"
	"time"
)

func TestNewReplayEngine(t *testing.T) {
	engine := NewReplayEngine()

	if engine == nil {
		t.Fatal("Expected replay engine to be created")
	}

	if engine.activeReplays == nil {
		t.Error("Expected activeReplays to be initialized")
	}

	if engine.comparator == nil {
		t.Error("Expected comparator to be initialized")
	}
}

func TestReplayEngineStartReplay(t *testing.T) {
	engine := NewReplayEngine()
	ctx := context.Background()

	// Create a test trajectory
	traj := NewExecutionTrajectory("agent-123", "Test Agent", "validation-456")
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
	traj.Finalize()

	config := DefaultTrajectoryReplayConfig()

	session, err := engine.StartReplay(ctx, traj, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be created")
	}

	if session.ID == "" {
		t.Error("Expected session ID to be set")
	}

	if session.BaseTrajectory != traj {
		t.Error("Expected session to reference the base trajectory")
	}

	// Wait for replay to complete
	time.Sleep(100 * time.Millisecond)

	// Get updated session
	updatedSession, err := engine.GetReplaySession(session.ID)
	if err != nil {
		t.Fatalf("Expected no error getting session, got %v", err)
	}

	if updatedSession.Result == nil {
		t.Fatal("Expected result to be set")
	}

	if updatedSession.Result.Status != "success" {
		t.Errorf("Expected status 'success', got %s", updatedSession.Result.Status)
	}
}

func TestReplayEngineStartReplayNilTrajectory(t *testing.T) {
	engine := NewReplayEngine()
	ctx := context.Background()

	_, err := engine.StartReplay(ctx, nil, DefaultTrajectoryReplayConfig())
	if err == nil {
		t.Error("Expected error for nil trajectory")
	}
}

func TestReplayEngineStartReplayInvalidStatus(t *testing.T) {
	engine := NewReplayEngine()
	ctx := context.Background()

	// Create trajectory with wrong status
	traj := NewExecutionTrajectory("agent-123", "Test Agent", "validation-456")
	// Status is CAPTURING, not CAPTURED

	_, err := engine.StartReplay(ctx, traj, DefaultTrajectoryReplayConfig())
	if err == nil {
		t.Error("Expected error for trajectory with wrong status")
	}
}

func TestReplayEngineGetReplaySession(t *testing.T) {
	engine := NewReplayEngine()
	ctx := context.Background()

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

	session, _ := engine.StartReplay(ctx, traj, DefaultTrajectoryReplayConfig())

	retrieved, err := engine.GetReplaySession(session.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if retrieved.ID != session.ID {
		t.Error("Expected retrieved session to match original")
	}

	// Test non-existent session
	_, err = engine.GetReplaySession("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestReplayEngineListActiveReplays(t *testing.T) {
	engine := NewReplayEngine()
	ctx := context.Background()

	// Should return empty initially
	sessions := engine.ListActiveReplays()
	if len(sessions) != 0 {
		t.Errorf("Expected 0 active replays, got %d", len(sessions))
	}

	// Start a replay
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

	engine.StartReplay(ctx, traj, DefaultTrajectoryReplayConfig())

	// Should have active replay
	sessions = engine.ListActiveReplays()
	if len(sessions) != 1 {
		t.Errorf("Expected 1 active replay, got %d", len(sessions))
	}
}

func TestTrajectoryComparatorEventsMatch(t *testing.T) {
	comparator := NewTrajectoryComparator()

	e1 := &ExecutionPoint{
		EventType: "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallName: "read",
			ReturnValue: 100,
		},
	}

	e2 := &ExecutionPoint{
		EventType: "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallName: "read",
			ReturnValue: 100,
		},
	}

	if !comparator.eventsMatch(e1, e2) {
		t.Error("Expected events to match")
	}

	// Different syscall name
	e3 := &ExecutionPoint{
		EventType: "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallName: "write",
			ReturnValue: 100,
		},
	}

	if comparator.eventsMatch(e1, e3) {
		t.Error("Expected events to not match (different syscall)")
	}

	// Different return value
	e4 := &ExecutionPoint{
		EventType: "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallName: "read",
			ReturnValue: 200,
		},
	}

	if comparator.eventsMatch(e1, e4) {
		t.Error("Expected events to not match (different return value)")
	}
}

func TestTrajectoryComparatorCompareTrajectories(t *testing.T) {
	comparator := NewTrajectoryComparator()

	base := NewExecutionTrajectory("agent-123", "Test Agent", "validation-456")
	base.AddEvent(&ExecutionPoint{
		EventType: "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallName: "read",
		},
	})
	base.AddEvent(&ExecutionPoint{
		EventType: "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallName: "write",
		},
	})

	compare := NewExecutionTrajectory("agent-123", "Test Agent", "validation-457")
	compare.AddEvent(&ExecutionPoint{
		EventType: "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallName: "read",
		},
	})
	compare.AddEvent(&ExecutionPoint{
		EventType: "syscall",
		Syscall: &TrajectorySyscallEvent{
			SyscallName: "write",
		},
	})

	result := comparator.CompareTrajectories(base, compare)

	if result.BaseTrajectoryID != base.ID {
		t.Error("Expected BaseTrajectoryID to match base.ID")
	}

	if result.CompareTrajectoryID != compare.ID {
		t.Error("Expected CompareTrajectoryID to match compare.ID")
	}

	if result.EventCountDiff != 0 {
		t.Errorf("Expected EventCountDiff to be 0, got %d", result.EventCountDiff)
	}

	if !result.AreEquivalent {
		t.Error("Expected trajectories to be equivalent")
	}
}

func TestIsNonDeterministicSyscall(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"gettimeofday", true},
		{"clock_gettime", true},
		{"time", true},
		{"getpid", true},
		{"gettid", true},
		{"getrandom", true},
		{"read", false},
		{"write", false},
		{"open", false},
	}

	for _, tt := range tests {
		result := isNonDeterministicSyscall(tt.name)
		if result != tt.expected {
			t.Errorf("isNonDeterministicSyscall(%s) = %v, expected %v", tt.name, result, tt.expected)
		}
	}
}

func TestRandFloat(t *testing.T) {
	// Test that randFloat returns values between 0 and 1
	for i := 0; i < 100; i++ {
		val := randFloat()
		if val < 0 || val >= 1 {
			t.Errorf("randFloat() returned %f, expected [0, 1)", val)
		}
	}
}

func BenchmarkReplayEngineStartReplay(b *testing.B) {
	engine := NewReplayEngine()
	ctx := context.Background()

	traj := NewExecutionTrajectory("agent-123", "Test Agent", "validation-456")
	for i := 0; i < 100; i++ {
		traj.AddEvent(&ExecutionPoint{
			Timestamp:   time.Now(),
			SequenceNum: uint64(i + 1),
			PID:         1234,
			EventType:   "syscall",
			Syscall: &TrajectorySyscallEvent{
				SyscallID:   0,
				SyscallName: "read",
				ReturnValue: 100,
			},
		})
	}
	traj.Finalize()

	config := DefaultTrajectoryReplayConfig()
	config.SkipDelays = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session, _ := engine.StartReplay(ctx, traj, config)
		time.Sleep(10 * time.Millisecond) // Allow replay to process
		engine.CleanupReplay(session.ID)
	}
}
