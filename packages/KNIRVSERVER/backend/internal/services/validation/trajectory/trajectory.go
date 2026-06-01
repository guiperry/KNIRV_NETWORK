// Package trajectory provides canonical types for agent execution trajectory
// capture and replay. These types are promoted from the fintech plugin package
// to a platform-level package so they can be shared across validators.
package trajectory

import (
	"backend_server/internal/services/plugins/fintech"
)

// Re-export canonical types from the fintech package.

// ExecutionTrajectory is an alias for fintech.ExecutionTrajectory.
type ExecutionTrajectory = fintech.ExecutionTrajectory

// ExecutionPoint is an alias for fintech.ExecutionPoint.
type ExecutionPoint = fintech.ExecutionPoint

// TrajectoryCaptureConfig is an alias for fintech.TrajectoryCaptureConfig.
type TrajectoryCaptureConfig = fintech.TrajectoryCaptureConfig

// TrajectoryReplayConfig is an alias for fintech.TrajectoryReplayConfig.
type TrajectoryReplayConfig = fintech.TrajectoryReplayConfig

// TrajectoryFilter is an alias for fintech.TrajectoryFilter.
type TrajectoryFilter = fintech.TrajectoryFilter

// TrajectoryComparisonResult is an alias for fintech.TrajectoryComparisonResult.
type TrajectoryComparisonResult = fintech.TrajectoryComparisonResult

// TrajectoryStatus is an alias for fintech.TrajectoryStatus.
type TrajectoryStatus = fintech.TrajectoryStatus

// Constants re-exported for convenience.
const (
	TrajectoryStatusCapturing = fintech.TrajectoryStatusCapturing
	TrajectoryStatusCaptured  = fintech.TrajectoryStatusCaptured
	TrajectoryStatusReplaying = fintech.TrajectoryStatusReplaying
	TrajectoryStatusReplayed  = fintech.TrajectoryStatusReplayed
	TrajectoryStatusFailed    = fintech.TrajectoryStatusFailed
)

// DefaultCaptureConfig returns the default trajectory capture configuration.
func DefaultCaptureConfig() *TrajectoryCaptureConfig {
	return fintech.DefaultTrajectoryCaptureConfig()
}

// DefaultReplayConfig returns the default trajectory replay configuration.
func DefaultReplayConfig() *TrajectoryReplayConfig {
	return fintech.DefaultTrajectoryReplayConfig()
}

// ReplayEngine is an alias for fintech.ReplayEngine.
type ReplayEngine = fintech.ReplayEngine

// ReplaySession is an alias for fintech.ReplaySession.
type ReplaySession = fintech.ReplaySession

// ReplayResult is an alias for fintech.ReplayResult.
type ReplayResult = fintech.ReplayResult

// TrajectoryComparator is an alias for fintech.TrajectoryComparator.
type TrajectoryComparator = fintech.TrajectoryComparator

// NewReplayEngine creates a new ReplayEngine.
func NewReplayEngine() *ReplayEngine {
	return fintech.NewReplayEngine()
}

// NewTrajectoryComparator creates a new TrajectoryComparator.
func NewTrajectoryComparator() *TrajectoryComparator {
	return fintech.NewTrajectoryComparator()
}
