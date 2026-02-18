// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package fintech

import (
	"encoding/json"
	"fmt"
	"time"
)

// TrajectoryStatus represents the status of a captured trajectory
type TrajectoryStatus string

const (
	TrajectoryStatusCapturing TrajectoryStatus = "CAPTURING"
	TrajectoryStatusCaptured  TrajectoryStatus = "CAPTURED"
	TrajectoryStatusReplaying TrajectoryStatus = "REPLAYING"
	TrajectoryStatusReplayed  TrajectoryStatus = "REPLAYED"
	TrajectoryStatusFailed    TrajectoryStatus = "FAILED"
	TrajectoryStatusCorrupted TrajectoryStatus = "CORRUPTED"
)

// TrajectorySyscallEvent represents a system call captured during agent execution
type TrajectorySyscallEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	SequenceNum uint64                 `json:"sequence_num"`
	PID         uint32                 `json:"pid"`
	TID         uint32                 `json:"tid"`
	SyscallID   uint32                 `json:"syscall_id"`
	SyscallName string                 `json:"syscall_name"`
	Arguments   []TrajectorySyscallArg `json:"arguments"`
	ReturnValue int64                  `json:"return_value"`
	DurationNs  uint64                 `json:"duration_ns"`
	ProcessName string                 `json:"process_name"`
	CPU         uint32                 `json:"cpu"`
}

// TrajectorySyscallArg represents a single syscall argument
type TrajectorySyscallArg struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
	Value string `json:"value"`
	Size  uint64 `json:"size"`
}

// TrajectoryFileAccessEvent represents file system access during execution
type TrajectoryFileAccessEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	SequenceNum  uint64    `json:"sequence_num"`
	PID          uint32    `json:"pid"`
	Operation    string    `json:"operation"` // open, read, write, close
	Path         string    `json:"path"`
	Flags        uint64    `json:"flags"`
	Mode         uint32    `json:"mode"`
	BytesRead    uint64    `json:"bytes_read,omitempty"`
	BytesWritten uint64    `json:"bytes_written,omitempty"`
	ReturnValue  int64     `json:"return_value"`
}

// TrajectoryNetworkEvent represents network activity during execution
type TrajectoryNetworkEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	SequenceNum uint64    `json:"sequence_num"`
	PID         uint32    `json:"pid"`
	Operation   string    `json:"operation"` // connect, send, recv, bind, listen
	Protocol    string    `json:"protocol"`  // tcp, udp, unix
	SourceIP    string    `json:"source_ip,omitempty"`
	SourcePort  uint16    `json:"source_port,omitempty"`
	DestIP      string    `json:"dest_ip,omitempty"`
	DestPort    uint16    `json:"dest_port,omitempty"`
	BytesSent   uint64    `json:"bytes_sent,omitempty"`
	BytesRecv   uint64    `json:"bytes_recv,omitempty"`
	ReturnValue int64     `json:"return_value"`
}

// TrajectoryMemoryEvent represents memory allocation/deallocation
type TrajectoryMemoryEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	SequenceNum uint64    `json:"sequence_num"`
	PID         uint32    `json:"pid"`
	Operation   string    `json:"operation"` // mmap, munmap, brk, malloc, free
	Address     uint64    `json:"address"`
	Size        uint64    `json:"size"`
	Flags       uint32    `json:"flags,omitempty"`
	ReturnValue int64     `json:"return_value"`
}

// ExecutionPoint represents a point in the execution timeline
type ExecutionPoint struct {
	Timestamp   time.Time                  `json:"timestamp"`
	SequenceNum uint64                     `json:"sequence_num"`
	PID         uint32                     `json:"pid"`
	EventType   string                     `json:"event_type"`
	Syscall     *TrajectorySyscallEvent    `json:"syscall,omitempty"`
	FileAccess  *TrajectoryFileAccessEvent `json:"file_access,omitempty"`
	Network     *TrajectoryNetworkEvent    `json:"network,omitempty"`
	Memory      *TrajectoryMemoryEvent     `json:"memory,omitempty"`
	CustomData  map[string]interface{}     `json:"custom_data,omitempty"`
}

// ExecutionTrajectory represents a complete captured execution trace
type ExecutionTrajectory struct {
	// Metadata
	ID        string    `json:"id"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Subject (what was traced)
	AgentID      string `json:"agent_id"`
	AgentName    string `json:"agent_name"`
	AgentVersion string `json:"agent_version"`
	ProcessID    uint32 `json:"process_id"`
	ContainerID  string `json:"container_id,omitempty"`

	// Execution context
	ValidationID string           `json:"validation_id"`
	ScenarioID   string           `json:"scenario_id,omitempty"`
	StartedAt    time.Time        `json:"started_at"`
	EndedAt      time.Time        `json:"ended_at,omitempty"`
	Status       TrajectoryStatus `json:"status"`

	// Captured events
	Events     []*ExecutionPoint `json:"events"`
	EventCount uint64            `json:"event_count"`

	// Execution metrics
	Metrics *TrajectoryMetrics `json:"metrics"`

	// Determinism verification
	DeterminismHash string `json:"determinism_hash,omitempty"`
	IsDeterministic bool   `json:"is_deterministic"`

	// References
	EvidencePackID string `json:"evidence_pack_id,omitempty"`
	CertificateID  string `json:"certificate_id,omitempty"`

	// PQC Signature
	Signature *PQCSignature `json:"signature,omitempty"`
}

// TrajectoryMetrics holds performance and behavioral metrics
type TrajectoryMetrics struct {
	TotalSyscalls    uint64            `json:"total_syscalls"`
	UniqueSyscalls   uint32            `json:"unique_syscalls"`
	FileAccesses     uint64            `json:"file_accesses"`
	NetworkCalls     uint64            `json:"network_calls"`
	MemoryOperations uint64            `json:"memory_operations"`
	DurationMs       uint64            `json:"duration_ms"`
	CPUTimeMs        uint64            `json:"cpu_time_ms"`
	MemoryPeakBytes  uint64            `json:"memory_peak_bytes"`
	NetworkBytesSent uint64            `json:"network_bytes_sent"`
	NetworkBytesRecv uint64            `json:"network_bytes_recv"`
	SyscallFrequency map[string]uint64 `json:"syscall_frequency"`
}

// TrajectoryCaptureConfig holds configuration for trajectory capture
type TrajectoryCaptureConfig struct {
	CaptureSyscalls     bool     `json:"capture_syscalls"`
	CaptureFiles        bool     `json:"capture_files"`
	CaptureNetwork      bool     `json:"capture_network"`
	CaptureMemory       bool     `json:"capture_memory"`
	MaxEvents           uint64   `json:"max_events"`
	MaxDurationMs       uint64   `json:"max_duration_ms"`
	FilterSyscalls      []string `json:"filter_syscalls,omitempty"`
	ExcludeSyscalls     []string `json:"exclude_syscalls,omitempty"`
	CaptureArguments    bool     `json:"capture_arguments"`
	CaptureReturnValues bool     `json:"capture_return_values"`
}

// DefaultTrajectoryCaptureConfig returns default capture configuration
func DefaultTrajectoryCaptureConfig() *TrajectoryCaptureConfig {
	return &TrajectoryCaptureConfig{
		CaptureSyscalls:     true,
		CaptureFiles:        true,
		CaptureNetwork:      true,
		CaptureMemory:       false,   // Disabled by default for performance
		MaxEvents:           1000000, // 1M events max
		MaxDurationMs:       300000,  // 5 minutes max
		CaptureArguments:    true,
		CaptureReturnValues: true,
		ExcludeSyscalls: []string{
			"gettimeofday",
			"clock_gettime",
			"time",
		},
	}
}

// TrajectoryReplayConfig holds configuration for replay
type TrajectoryReplayConfig struct {
	ReplaySpeed        float64 `json:"replay_speed"` // 1.0 = real-time, 2.0 = 2x speed
	SkipDelays         bool    `json:"skip_delays"`
	VerifyDeterminism  bool    `json:"verify_determinism"`
	StopOnMismatch     bool    `json:"stop_on_mismatch"`
	InjectFaults       bool    `json:"inject_faults"`
	FaultInjectionRate float64 `json:"fault_injection_rate"` // 0.0 - 1.0
}

// DefaultTrajectoryReplayConfig returns default replay configuration
func DefaultTrajectoryReplayConfig() *TrajectoryReplayConfig {
	return &TrajectoryReplayConfig{
		ReplaySpeed:        1.0,
		SkipDelays:         false,
		VerifyDeterminism:  true,
		StopOnMismatch:     true,
		InjectFaults:       false,
		FaultInjectionRate: 0.0,
	}
}

// ReplayResult represents the outcome of a trajectory replay
type ReplayResult struct {
	TrajectoryID     string            `json:"trajectory_id"`
	ReplayID         string            `json:"replay_id"`
	StartedAt        time.Time         `json:"started_at"`
	EndedAt          time.Time         `json:"ended_at"`
	Status           string            `json:"status"` // success, failed, partial
	EventsReplayed   uint64            `json:"events_replayed"`
	EventsMatched    uint64            `json:"events_matched"`
	EventsMismatched uint64            `json:"events_mismatched"`
	IsDeterministic  bool              `json:"is_deterministic"`
	DeterminismScore float64           `json:"determinism_score"` // 0.0 - 1.0
	Mismatches       []*EventMismatch  `json:"mismatches,omitempty"`
	PerformanceDelta *PerformanceDelta `json:"performance_delta,omitempty"`
	ErrorMessage     string            `json:"error_message,omitempty"`
}

// EventMismatch represents a difference between expected and actual events
type EventMismatch struct {
	SequenceNum   uint64                 `json:"sequence_num"`
	ExpectedType  string                 `json:"expected_type"`
	ActualType    string                 `json:"actual_type"`
	ExpectedEvent map[string]interface{} `json:"expected_event"`
	ActualEvent   map[string]interface{} `json:"actual_event"`
	MismatchType  string                 `json:"mismatch_type"` // type, value, timing
	Difference    string                 `json:"difference"`
}

// PerformanceDelta compares performance metrics between original and replay
type PerformanceDelta struct {
	DurationDeltaMs       int64   `json:"duration_delta_ms"`
	SyscallCountDelta     int64   `json:"syscall_count_delta"`
	MemoryUsageDeltaBytes int64   `json:"memory_usage_delta_bytes"`
	NetworkBytesDelta     int64   `json:"network_bytes_delta"`
	PerformanceChangePct  float64 `json:"performance_change_pct"`
}

// TrajectoryFilter provides filtering options for trajectory queries
type TrajectoryFilter struct {
	AgentID      string
	ValidationID string
	Status       TrajectoryStatus
	StartTime    *time.Time
	EndTime      *time.Time
	HasSignature *bool
	Limit        int
	Offset       int
}

// NewExecutionTrajectory creates a new execution trajectory
func NewExecutionTrajectory(agentID, agentName, validationID string) *ExecutionTrajectory {
	now := time.Now()
	return &ExecutionTrajectory{
		ID:           fmt.Sprintf("traj-%d", now.UnixNano()),
		Version:      "1.0",
		CreatedAt:    now,
		UpdatedAt:    now,
		AgentID:      agentID,
		AgentName:    agentName,
		ValidationID: validationID,
		StartedAt:    now,
		Status:       TrajectoryStatusCapturing,
		Events:       make([]*ExecutionPoint, 0),
		Metrics: &TrajectoryMetrics{
			SyscallFrequency: make(map[string]uint64),
		},
		IsDeterministic: true,
	}
}

// AddEvent adds an execution point to the trajectory
func (t *ExecutionTrajectory) AddEvent(event *ExecutionPoint) {
	t.Events = append(t.Events, event)
	t.EventCount++
	t.UpdatedAt = time.Now()

	// Update metrics
	t.updateMetrics(event)
}

// updateMetrics updates trajectory metrics based on event
func (t *ExecutionTrajectory) updateMetrics(event *ExecutionPoint) {
	if t.Metrics == nil {
		t.Metrics = &TrajectoryMetrics{
			SyscallFrequency: make(map[string]uint64),
		}
	}

	switch event.EventType {
	case "syscall":
		t.Metrics.TotalSyscalls++
		if event.Syscall != nil {
			t.Metrics.SyscallFrequency[event.Syscall.SyscallName]++
		}
	case "file_access":
		t.Metrics.FileAccesses++
	case "network":
		t.Metrics.NetworkCalls++
	case "memory":
		t.Metrics.MemoryOperations++
	}
}

// Finalize marks the trajectory as complete
func (t *ExecutionTrajectory) Finalize() {
	t.EndedAt = time.Now()
	t.Status = TrajectoryStatusCaptured
	t.UpdatedAt = time.Now()

	// Calculate metrics
	if t.Metrics != nil {
		t.Metrics.DurationMs = uint64(t.EndedAt.Sub(t.StartedAt).Milliseconds())
		t.Metrics.UniqueSyscalls = uint32(len(t.Metrics.SyscallFrequency))
	}
}

// CalculateDeterminismHash computes a hash representing the deterministic behavior
func (t *ExecutionTrajectory) CalculateDeterminismHash() string {
	// Create a deterministic representation of the trajectory
	// Only include non-time-sensitive data
	hashData := struct {
		AgentID      string   `json:"agent_id"`
		EventTypes   []string `json:"event_types"`
		SyscallNames []string `json:"syscall_names"`
		SyscallOrder []uint32 `json:"syscall_order"`
		ReturnValues []int64  `json:"return_values"`
	}{
		AgentID:      t.AgentID,
		EventTypes:   make([]string, 0, len(t.Events)),
		SyscallNames: make([]string, 0),
		SyscallOrder: make([]uint32, 0),
		ReturnValues: make([]int64, 0),
	}

	for _, event := range t.Events {
		hashData.EventTypes = append(hashData.EventTypes, event.EventType)

		if event.Syscall != nil {
			hashData.SyscallNames = append(hashData.SyscallNames, event.Syscall.SyscallName)
			hashData.SyscallOrder = append(hashData.SyscallOrder, event.Syscall.SyscallID)
			hashData.ReturnValues = append(hashData.ReturnValues, event.Syscall.ReturnValue)
		}
	}

	// Serialize and hash
	data, _ := json.Marshal(hashData)

	// Simple hash (in production, use proper cryptographic hash)
	hash := fmt.Sprintf("%x", data)

	t.DeterminismHash = hash
	return hash
}

// ToMarkdown exports the trajectory as Markdown for storage
func (t *ExecutionTrajectory) ToMarkdown() ([]byte, error) {
	content := fmt.Sprintf(`# Execution Trajectory: %s

## Metadata

- **ID**: %s
- **Agent ID**: %s
- **Agent Name**: %s
- **Validation ID**: %s
- **Status**: %s
- **Created**: %s
- **Duration**: %d ms

## Execution Metrics

| Metric | Value |
|--------|-------|
| Total Syscalls | %d |
| Unique Syscalls | %d |
| File Accesses | %d |
| Network Calls | %d |
| Memory Operations | %d |
| CPU Time | %d ms |
| Peak Memory | %d bytes |

## Determinism

- **Is Deterministic**: %v
- **Determinism Hash**: %s

## Event Summary

Total Events: %d

### Syscall Frequency

`,
		t.ID,
		t.ID,
		t.AgentID,
		t.AgentName,
		t.ValidationID,
		t.Status,
		t.CreatedAt.Format(time.RFC3339),
		t.Metrics.DurationMs,
		t.Metrics.TotalSyscalls,
		t.Metrics.UniqueSyscalls,
		t.Metrics.FileAccesses,
		t.Metrics.NetworkCalls,
		t.Metrics.MemoryOperations,
		t.Metrics.CPUTimeMs,
		t.Metrics.MemoryPeakBytes,
		t.IsDeterministic,
		t.DeterminismHash,
		t.EventCount,
	)

	// Add syscall frequency table
	content += "| Syscall | Count |\n|---------|-------|\n"
	for syscall, count := range t.Metrics.SyscallFrequency {
		content += fmt.Sprintf("| %s | %d |\n", syscall, count)
	}

	// Add signature if present
	if t.Signature != nil {
		content += fmt.Sprintf(`
## Signature

- **Algorithm**: %s
- **Public Key ID**: %s
- **Signed At**: %s
`,
			t.Signature.Algorithm,
			t.Signature.PublicKeyID,
			t.Signature.SignedAt.Format(time.RFC3339),
		)
	}

	content += "\n---\n*Trajectory captured by KNIRVNEXUS FinTech Validator*\n"

	return []byte(content), nil
}

// TrajectoryComparisonResult holds the result of comparing two trajectories
type TrajectoryComparisonResult struct {
	BaseTrajectoryID    string            `json:"base_trajectory_id"`
	CompareTrajectoryID string            `json:"compare_trajectory_id"`
	SimilarityScore     float64           `json:"similarity_score"` // 0.0 - 1.0
	AreEquivalent       bool              `json:"are_equivalent"`
	EventCountDiff      int64             `json:"event_count_diff"`
	SyscallDiffs        []*SyscallDiff    `json:"syscall_diffs,omitempty"`
	BehavioralDiffs     []*BehavioralDiff `json:"behavioral_diffs,omitempty"`
}

// SyscallDiff represents a difference in syscall behavior
type SyscallDiff struct {
	SyscallName     string `json:"syscall_name"`
	BaseCount       uint64 `json:"base_count"`
	CompareCount    uint64 `json:"compare_count"`
	CountDiff       int64  `json:"count_diff"`
	ReturnValueDiff bool   `json:"return_value_diff"`
}

// BehavioralDiff represents a high-level behavioral difference
type BehavioralDiff struct {
	Category    string `json:"category"` // file_access, network, memory
	Description string `json:"description"`
	Severity    string `json:"severity"` // minor, moderate, major
}
