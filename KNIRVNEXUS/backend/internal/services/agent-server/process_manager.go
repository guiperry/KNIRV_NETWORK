package main

import (
	"fmt"
	"os"
	"sync"
)

// NewNativeProcessManager creates a new native process manager
func NewNativeProcessManager() (*NativeProcessManager, error) {
	npm := &NativeProcessManager{
		processes: make(map[int]*ProcessInfo),
	}
	
	// Initialize cgroup manager
	cgroupMgr, err := NewCgroupManager()
	if err != nil {
		// Cgroups might not be available, continue without them
		fmt.Printf("Warning: cgroup manager initialization failed: %v\n", err)
		cgroupMgr = &CgroupManager{enabled: false}
	}
	npm.cgroupManager = cgroupMgr
	
	return npm, nil
}

// Start starts the process manager
func (npm *NativeProcessManager) Start() error {
	npm.mu.Lock()
	defer npm.mu.Unlock()
	
	// Initialize cgroup manager if available
	if npm.cgroupManager.enabled {
		if err := npm.cgroupManager.Initialize(); err != nil {
			fmt.Printf("Warning: cgroup manager start failed: %v\n", err)
			npm.cgroupManager.enabled = false
		}
	}
	
	return nil
}

// Stop stops the process manager
func (npm *NativeProcessManager) Stop() error {
	npm.mu.Lock()
	defer npm.mu.Unlock()
	
	// Clean up any remaining processes
	for pid, processInfo := range npm.processes {
		fmt.Printf("Cleaning up process %d (%s)\n", pid, processInfo.AgentID)
		
		// Clean up cgroup if it exists
		if npm.cgroupManager.enabled && processInfo.CgroupPath != "" {
			npm.cgroupManager.RemoveCgroup(processInfo.CgroupPath)
		}
	}
	
	// Clear process map
	npm.processes = make(map[int]*ProcessInfo)
	
	return nil
}

// RegisterProcess registers a new process
func (npm *NativeProcessManager) RegisterProcess(processInfo *ProcessInfo) error {
	npm.mu.Lock()
	defer npm.mu.Unlock()
	
	npm.processes[processInfo.PID] = processInfo
	
	// Setup cgroup if available
	if npm.cgroupManager.enabled {
		cgroupPath := fmt.Sprintf("knirv-agents/%s", processInfo.AgentID)
		if err := npm.cgroupManager.CreateCgroup(cgroupPath); err != nil {
			fmt.Printf("Warning: failed to create cgroup for process %d: %v\n", processInfo.PID, err)
		} else {
			processInfo.CgroupPath = cgroupPath
			
			// Add process to cgroup
			if err := npm.cgroupManager.AddProcessToCgroup(cgroupPath, processInfo.PID); err != nil {
				fmt.Printf("Warning: failed to add process %d to cgroup: %v\n", processInfo.PID, err)
			}
		}
	}
	
	return nil
}

// UnregisterProcess unregisters a process
func (npm *NativeProcessManager) UnregisterProcess(pid int) error {
	npm.mu.Lock()
	defer npm.mu.Unlock()
	
	processInfo, exists := npm.processes[pid]
	if !exists {
		return fmt.Errorf("process %d not found", pid)
	}
	
	// Clean up cgroup
	if npm.cgroupManager.enabled && processInfo.CgroupPath != "" {
		if err := npm.cgroupManager.RemoveCgroup(processInfo.CgroupPath); err != nil {
			fmt.Printf("Warning: failed to remove cgroup for process %d: %v\n", pid, err)
		}
	}
	
	delete(npm.processes, pid)
	return nil
}

// GetProcessInfo returns information about a process
func (npm *NativeProcessManager) GetProcessInfo(pid int) (*ProcessInfo, error) {
	npm.mu.RLock()
	defer npm.mu.RUnlock()
	
	processInfo, exists := npm.processes[pid]
	if !exists {
		return nil, fmt.Errorf("process %d not found", pid)
	}
	
	// Return a copy
	info := *processInfo
	return &info, nil
}

// GetAllProcesses returns all managed processes
func (npm *NativeProcessManager) GetAllProcesses() []*ProcessInfo {
	npm.mu.RLock()
	defer npm.mu.RUnlock()
	
	var processes []*ProcessInfo
	for _, processInfo := range npm.processes {
		// Return copies
		info := *processInfo
		processes = append(processes, &info)
	}
	
	return processes
}

// GetProcessesByAgent returns processes for a specific agent
func (npm *NativeProcessManager) GetProcessesByAgent(agentID string) []*ProcessInfo {
	npm.mu.RLock()
	defer npm.mu.RUnlock()
	
	var processes []*ProcessInfo
	for _, processInfo := range npm.processes {
		if processInfo.AgentID == agentID {
			// Return copy
			info := *processInfo
			processes = append(processes, &info)
		}
	}
	
	return processes
}

// SetResourceLimits sets resource limits for a process
func (npm *NativeProcessManager) SetResourceLimits(pid int, limits *ResourceAllocation) error {
	npm.mu.RLock()
	processInfo, exists := npm.processes[pid]
	npm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("process %d not found", pid)
	}
	
	if !npm.cgroupManager.enabled {
		return fmt.Errorf("cgroup manager not available")
	}
	
	if processInfo.CgroupPath == "" {
		return fmt.Errorf("process %d has no cgroup", pid)
	}
	
	// Set CPU limit
	if limits.CPULimit > 0 {
		if err := npm.cgroupManager.SetCPULimit(processInfo.CgroupPath, limits.CPULimit); err != nil {
			return fmt.Errorf("failed to set CPU limit: %w", err)
		}
	}
	
	// Set memory limit
	if limits.MemoryLimit > 0 {
		if err := npm.cgroupManager.SetMemoryLimit(processInfo.CgroupPath, limits.MemoryLimit); err != nil {
			return fmt.Errorf("failed to set memory limit: %w", err)
		}
	}
	
	return nil
}

// GetProcessStats returns resource usage statistics for a process
func (npm *NativeProcessManager) GetProcessStats(pid int) (map[string]interface{}, error) {
	npm.mu.RLock()
	processInfo, exists := npm.processes[pid]
	npm.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("process %d not found", pid)
	}
	
	stats := map[string]interface{}{
		"pid":        processInfo.PID,
		"agent_id":   processInfo.AgentID,
		"command":    processInfo.Command,
		"start_time": processInfo.StartTime,
		"status":     processInfo.Status,
	}
	
	// Get cgroup stats if available
	if npm.cgroupManager.enabled && processInfo.CgroupPath != "" {
		cgroupStats, err := npm.cgroupManager.GetCgroupStats(processInfo.CgroupPath)
		if err == nil {
			stats["cgroup_stats"] = cgroupStats
		}
	}
	
	return stats, nil
}

// NewCgroupManager creates a new cgroup manager
func NewCgroupManager() (*CgroupManager, error) {
	cgm := &CgroupManager{
		cgroupRoot: "/sys/fs/cgroup",
		enabled:    false,
	}
	
	// Check if cgroups are available
	if _, err := os.Stat(cgm.cgroupRoot); os.IsNotExist(err) {
		return cgm, fmt.Errorf("cgroups not available")
	}
	
	// Detect cgroup version
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		cgm.cgroupVersion = 2
	} else {
		cgm.cgroupVersion = 1
	}
	
	cgm.enabled = true
	return cgm, nil
}

// Initialize initializes the cgroup manager
func (cgm *CgroupManager) Initialize() error {
	if !cgm.enabled {
		return fmt.Errorf("cgroup manager not enabled")
	}
	
	// Create base directory for KNIRV agents
	baseDir := fmt.Sprintf("%s/knirv-agents", cgm.cgroupRoot)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base cgroup directory: %w", err)
	}
	
	return nil
}

// CreateCgroup creates a new cgroup
func (cgm *CgroupManager) CreateCgroup(path string) error {
	if !cgm.enabled {
		return fmt.Errorf("cgroup manager not enabled")
	}
	
	fullPath := fmt.Sprintf("%s/%s", cgm.cgroupRoot, path)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup %s: %w", path, err)
	}
	
	return nil
}

// RemoveCgroup removes a cgroup
func (cgm *CgroupManager) RemoveCgroup(path string) error {
	if !cgm.enabled {
		return fmt.Errorf("cgroup manager not enabled")
	}
	
	fullPath := fmt.Sprintf("%s/%s", cgm.cgroupRoot, path)
	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("failed to remove cgroup %s: %w", path, err)
	}
	
	return nil
}

// AddProcessToCgroup adds a process to a cgroup
func (cgm *CgroupManager) AddProcessToCgroup(path string, pid int) error {
	if !cgm.enabled {
		return fmt.Errorf("cgroup manager not enabled")
	}
	
	var procsFile string
	if cgm.cgroupVersion == 2 {
		procsFile = fmt.Sprintf("%s/%s/cgroup.procs", cgm.cgroupRoot, path)
	} else {
		procsFile = fmt.Sprintf("%s/%s/cgroup.procs", cgm.cgroupRoot, path)
	}
	
	file, err := os.OpenFile(procsFile, os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open cgroup.procs: %w", err)
	}
	defer file.Close()
	
	if _, err := fmt.Fprintf(file, "%d", pid); err != nil {
		return fmt.Errorf("failed to write PID to cgroup: %w", err)
	}
	
	return nil
}

// SetCPULimit sets CPU limit for a cgroup
func (cgm *CgroupManager) SetCPULimit(path string, limit float64) error {
	if !cgm.enabled {
		return fmt.Errorf("cgroup manager not enabled")
	}
	
	// This is a simplified implementation
	// Real implementation would set appropriate cgroup CPU limits
	return nil
}

// SetMemoryLimit sets memory limit for a cgroup
func (cgm *CgroupManager) SetMemoryLimit(path string, limit uint64) error {
	if !cgm.enabled {
		return fmt.Errorf("cgroup manager not enabled")
	}
	
	// This is a simplified implementation
	// Real implementation would set appropriate cgroup memory limits
	return nil
}

// GetCgroupStats returns cgroup statistics
func (cgm *CgroupManager) GetCgroupStats(path string) (map[string]interface{}, error) {
	if !cgm.enabled {
		return nil, fmt.Errorf("cgroup manager not enabled")
	}
	
	// This is a simplified implementation
	// Real implementation would read cgroup statistics files
	return map[string]interface{}{
		"path":    path,
		"version": cgm.cgroupVersion,
	}, nil
}
