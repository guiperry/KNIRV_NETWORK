// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"net"
	"time"
)

// Config holds eBPF configuration
type Config struct {
	Programs []ProgramConfig
}

// ProgramConfig defines configuration for an eBPF program
type ProgramConfig struct {
	Name    string
	Enabled bool
}

// Metrics holds eBPF performance and operational metrics
type Metrics struct {
	ProgramsAttached int
	Initialized      bool
}

// SyscallEvent represents a syscall event from kernel
type SyscallEvent struct {
	Timestamp uint64
	PID       uint32
	SyscallID uint32
}

// SandboxPolicy defines security policy for a container
type SandboxPolicy struct {
	AllowedPathPrefix string
	NetworkAllowed    bool
}

// NetworkMetrics holds network-related eBPF metrics
type NetworkMetrics struct {
	DroppedPackets uint64
	AllowedPackets uint64
	TopAttackers   []IPStats
}

// IPStats holds statistics for a specific IP address
type IPStats struct {
	IP             net.IP
	PacketsDropped uint64
	Timestamp      time.Time
}

// VirtualContainer represents an eBPF-based virtual container
type VirtualContainer struct {
	ID             uint64
	RootPID        uint32
	RootFS         string
	NetworkAllowed bool
}
