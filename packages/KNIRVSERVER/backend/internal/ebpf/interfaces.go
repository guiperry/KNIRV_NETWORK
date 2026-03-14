// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import "context"

// ManagerInterface defines the interface for eBPF Manager operations
type ManagerInterface interface {
	Initialize(ctx context.Context, config *Config) error
	Shutdown() error
	GetMetrics() *Metrics
	GetProcessMetrics() (map[uint32]*ProcessStats, error)
}
