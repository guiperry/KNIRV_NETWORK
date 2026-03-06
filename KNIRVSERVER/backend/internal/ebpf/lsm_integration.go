// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/cilium/ebpf/rlimit"
	"golang.org/x/sys/unix"
)

type LSMIntegration struct {
	mu            sync.RWMutex
	enforcer      *LSMEnforcer
	policyManager *SecurityProfileManager
	telemetry     *TelemetryCollector
	enabled       bool
	allowedPaths  map[string]bool
	blockedPaths  map[string]bool
	rootMount     string
}

type LSMEnforcer struct {
	mu         sync.RWMutex
	enabled    bool
	rules      map[string]*LSMRule
	auditLog   []LSMAuditEntry
	maxLogSize int
}

type LSMRule struct {
	PathPattern string
	Action      LSMAction
	Priority    int
	CreatedAt   int64
}

type LSMAuditEntry struct {
	Timestamp   int64
	Path        string
	Action      LSMAction
	Decision    string
	ProcessName string
	PID         uint32
	ContainerID uint64
}

type LSMAction string

const (
	LSMActionAllow LSMAction = "ALLOW"
	LSMActionDeny  LSMAction = "DENY"
	LSMActionAudit LSMAction = "AUDIT"
)

func NewLSMIntegration(rootMount string) *LSMIntegration {
	return &LSMIntegration{
		allowedPaths:  make(map[string]bool),
		blockedPaths:  make(map[string]bool),
		rootMount:     rootMount,
		policyManager: NewSecurityProfileManager(),
	}
}

func (li *LSMIntegration) Initialize(ctx context.Context) error {
	li.mu.Lock()
	defer li.mu.Unlock()

	if err := rlimit.RemoveMemlock(); err != nil {
		log.Printf("LSMIntegration: Warning - could not remove memlock limit: %v", err)
	}

	li.enforcer = &LSMEnforcer{
		rules:      make(map[string]*LSMRule),
		auditLog:   make([]LSMAuditEntry, 0),
		maxLogSize: 10000,
		enabled:    true,
	}

	li.telemetry = NewTelemetryCollector()
	if err := li.telemetry.Initialize(); err != nil {
		log.Printf("LSMIntegration: Failed to initialize telemetry: %v", err)
		li.enabled = false
		return nil
	}

	if err := li.telemetry.Start(ctx); err != nil {
		log.Printf("LSMIntegration: Failed to start telemetry: %v", err)
		li.enabled = false
		return nil
	}

	li.setupDefaultRules()

	li.enabled = true
	log.Printf("LSMIntegration: Initialized with root mount %s", li.rootMount)
	return nil
}

func (li *LSMIntegration) setupDefaultRules() {
	if li.enforcer == nil {
		return
	}

	defaultRules := []struct {
		path     string
		action   LSMAction
		priority int
	}{
		{"/proc/self", LSMActionAllow, 100},
		{"/proc/1/ns", LSMActionDeny, 90},
		{"/sys/kernel", LSMActionDeny, 90},
		{"/sys/fs/cgroup", LSMActionDeny, 85},
		{"/dev/mem", LSMActionDeny, 100},
		{"/dev/kmem", LSMActionDeny, 100},
		{"/dev/port", LSMActionDeny, 100},
		{"/sys/firmware", LSMActionDeny, 95},
		{"/boot", LSMActionDeny, 95},
	}

	li.enforcer.mu.Lock()
	defer li.enforcer.mu.Unlock()

	for _, r := range defaultRules {
		li.enforcer.rules[r.path] = &LSMRule{
			PathPattern: r.path,
			Action:      r.action,
			Priority:    r.priority,
			CreatedAt:   nowUnixSec(),
		}
	}
}

func (li *LSMIntegration) IsEnabled() bool {
	li.mu.RLock()
	defer li.mu.RUnlock()
	return li.enabled
}

func (li *LSMIntegration) CheckPathAccess(path string, pid uint32, containerID uint64) (bool, error) {
	li.mu.RLock()
	defer li.mu.RUnlock()

	if !li.enabled || li.enforcer == nil {
		return true, nil
	}

	normalizedPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("normalize path: %w", err)
	}

	// First, check the container-specific security profile if it exists
	if containerID > 0 && li.policyManager != nil {
		if !li.policyManager.IsPathAllowed(containerID, normalizedPath) {
			li.logAccess(normalizedPath, LSMActionDeny, pid, containerID)
			return false, fmt.Errorf("access denied by container security profile: %s", normalizedPath)
		}
	}

	li.enforcer.mu.RLock()
	defer li.enforcer.mu.RUnlock()

	var matchedRule *LSMRule
	var highestPriority int

	for pattern, rule := range li.enforcer.rules {
		if matchesPathPattern(normalizedPath, pattern) {
			if rule.Priority > highestPriority {
				highestPriority = rule.Priority
				matchedRule = rule
			}
		}
	}

	if matchedRule != nil {
		li.logAccess(normalizedPath, matchedRule.Action, pid, containerID)

		if matchedRule.Action == LSMActionDeny {
			return false, fmt.Errorf("access denied by global LSM policy: %s", matchedRule.PathPattern)
		}
		return matchedRule.Action == LSMActionAllow, nil
	}

	return li.enforcer.enabled, nil
}

func (li *LSMIntegration) AddRule(pathPattern string, action LSMAction, priority int) error {
	li.mu.RLock()
	defer li.mu.RUnlock()

	if li.enforcer == nil {
		return fmt.Errorf("LSM enforcer not initialized")
	}

	li.enforcer.mu.Lock()
	defer li.enforcer.mu.Unlock()

	li.enforcer.rules[pathPattern] = &LSMRule{
		PathPattern: pathPattern,
		Action:      action,
		Priority:    priority,
		CreatedAt:   nowUnixSec(),
	}

	log.Printf("LSMIntegration: Added rule %s -> %s (priority %d)", pathPattern, action, priority)
	return nil
}

func (li *LSMIntegration) RemoveRule(pathPattern string) error {
	li.mu.RLock()
	defer li.mu.RUnlock()

	if li.enforcer == nil {
		return fmt.Errorf("LSM enforcer not initialized")
	}

	li.enforcer.mu.Lock()
	defer li.enforcer.mu.Unlock()

	delete(li.enforcer.rules, pathPattern)
	log.Printf("LSMIntegration: Removed rule %s", pathPattern)
	return nil
}

func (li *LSMIntegration) GetAuditLog() []LSMAuditEntry {
	li.mu.RLock()
	defer li.mu.RUnlock()

	if li.enforcer == nil {
		return []LSMAuditEntry{}
	}

	li.enforcer.mu.RLock()
	defer li.enforcer.mu.RUnlock()

	result := make([]LSMAuditEntry, len(li.enforcer.auditLog))
	copy(result, li.enforcer.auditLog)
	return result
}

func (li *LSMIntegration) logAccess(path string, action LSMAction, pid uint32, containerID uint64) {
	if li.enforcer == nil {
		return
	}

	li.enforcer.mu.Lock()
	defer li.enforcer.mu.Unlock()

	entry := LSMAuditEntry{
		Timestamp:   nowUnixSec(),
		Path:        path,
		Action:      action,
		Decision:    string(action),
		ProcessName: getProcessName(pid),
		PID:         pid,
		ContainerID: containerID,
	}

	li.enforcer.auditLog = append(li.enforcer.auditLog, entry)

	if len(li.enforcer.auditLog) > li.enforcer.maxLogSize {
		li.enforcer.auditLog = li.enforcer.auditLog[len(li.enforcer.auditLog)-li.enforcer.maxLogSize:]
	}
}

func (li *LSMIntegration) EnforcePathRestrictions(rootDir string) error {
	if _, err := os.Stat(rootDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(rootDir, 0755); err != nil {
				return fmt.Errorf("create root dir: %w", err)
			}
		} else {
			return fmt.Errorf("stat root dir: %w", err)
		}
	}

	li.mu.Lock()
	li.rootMount = rootDir
	li.mu.Unlock()

	log.Printf("LSMIntegration: Enforcing path restrictions for %s", rootDir)
	return nil
}

func (li *LSMIntegration) GetTelemetryStats() TelemetryStats {
	li.mu.RLock()
	defer li.mu.RUnlock()

	if li.telemetry != nil {
		return li.telemetry.GetStats()
	}
	return TelemetryStats{}
}

func (li *LSMIntegration) GetAggregatedStats() (cpuNs, memBytes, netTx, netRx uint64) {
	li.mu.RLock()
	defer li.mu.RUnlock()

	if li.telemetry != nil {
		return li.telemetry.GetAggregatedStats()
	}
	return
}

func (li *LSMIntegration) Stop() error {
	li.mu.Lock()
	defer li.mu.Unlock()

	if li.telemetry != nil {
		li.telemetry.Stop()
	}

	li.enabled = false
	log.Println("LSMIntegration: Stopped")
	return nil
}

func matchesPathNormalize(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

func matchesPathPattern(path, pattern string) bool {
	path = filepath.Clean(path)
	pattern = filepath.Clean(pattern)

	if pattern == path {
		return true
	}

	if len(path) > len(pattern) {
		return path[:len(pattern)] == pattern
	}

	return false
}

func getProcessName(pid uint32) string {
	if pid == 0 {
		return "kernel"
	}

	commPath := fmt.Sprintf("/proc/%d/comm", pid)
	data, err := os.ReadFile(commPath)
	if err != nil {
		return fmt.Sprintf("pid_%d", pid)
	}

	name := string(data)
	if len(name) > 0 && name[len(name)-1] == '\n' {
		name = name[:len(name)-1]
	}
	return name
}

func nowUnixSec() int64 {
	var ts unix.Timespec
	unix.ClockGettime(unix.CLOCK_REALTIME, &ts)
	return ts.Sec
}
