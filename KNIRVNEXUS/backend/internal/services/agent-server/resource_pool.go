package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// NewResourcePool creates a new resource pool
func NewResourcePool() (*ResourcePool, error) {
	rp := &ResourcePool{}
	
	// Initialize system resources
	if err := rp.initializeSystemResources(); err != nil {
		return nil, fmt.Errorf("failed to initialize system resources: %w", err)
	}
	
	return rp, nil
}

// initializeSystemResources discovers and initializes system resources
func (rp *ResourcePool) initializeSystemResources() error {
	// Get CPU information
	rp.TotalCPU = float64(runtime.NumCPU())
	rp.AvailableCPU = rp.TotalCPU
	
	// Get memory information
	memInfo, err := rp.getMemoryInfo()
	if err != nil {
		return fmt.Errorf("failed to get memory info: %w", err)
	}
	rp.TotalMemory = memInfo
	rp.AvailableMemory = memInfo
	
	// Get disk information (simplified - use root filesystem)
	diskInfo, err := rp.getDiskInfo("/")
	if err != nil {
		return fmt.Errorf("failed to get disk info: %w", err)
	}
	rp.TotalDisk = diskInfo
	rp.AvailableDisk = diskInfo
	
	return nil
}

// getMemoryInfo gets total system memory
func (rp *ResourcePool) getMemoryInfo() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				memKB, err := strconv.ParseUint(fields[1], 10, 64)
				if err != nil {
					return 0, err
				}
				return memKB * 1024, nil // Convert KB to bytes
			}
		}
	}
	
	return 0, fmt.Errorf("could not find MemTotal in /proc/meminfo")
}

// getDiskInfo gets disk space information for a path
func (rp *ResourcePool) getDiskInfo(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	
	// Total space = block size * total blocks
	totalSpace := uint64(stat.Bsize) * stat.Blocks
	return totalSpace, nil
}

// AllocateResources allocates resources for an agent
func (rp *ResourcePool) AllocateResources(requested *ResourceAllocation) (*ResourceAllocation, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	
	// Check if resources are available
	if requested.CPUCores > rp.AvailableCPU {
		return nil, fmt.Errorf("insufficient CPU: requested %.2f, available %.2f", 
			requested.CPUCores, rp.AvailableCPU)
	}
	
	if requested.MemoryBytes > rp.AvailableMemory {
		return nil, fmt.Errorf("insufficient memory: requested %d, available %d", 
			requested.MemoryBytes, rp.AvailableMemory)
	}
	
	if requested.DiskBytes > rp.AvailableDisk {
		return nil, fmt.Errorf("insufficient disk: requested %d, available %d", 
			requested.DiskBytes, rp.AvailableDisk)
	}
	
	// Allocate resources
	allocation := &ResourceAllocation{
		CPUCores:    requested.CPUCores,
		MemoryBytes: requested.MemoryBytes,
		DiskBytes:   requested.DiskBytes,
		
		// Set limits (same as allocation for now)
		CPULimit:    requested.CPUCores,
		MemoryLimit: requested.MemoryBytes,
		DiskLimit:   requested.DiskBytes,
	}
	
	// Update available resources
	rp.AvailableCPU -= requested.CPUCores
	rp.AvailableMemory -= requested.MemoryBytes
	rp.AvailableDisk -= requested.DiskBytes
	
	// Update allocated resources
	rp.AllocatedCPU += requested.CPUCores
	rp.AllocatedMemory += requested.MemoryBytes
	rp.AllocatedDisk += requested.DiskBytes
	
	return allocation, nil
}

// ReleaseResources releases allocated resources
func (rp *ResourcePool) ReleaseResources(allocation *ResourceAllocation) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	
	// Return resources to available pool
	rp.AvailableCPU += allocation.CPUCores
	rp.AvailableMemory += allocation.MemoryBytes
	rp.AvailableDisk += allocation.DiskBytes
	
	// Update allocated resources
	rp.AllocatedCPU -= allocation.CPUCores
	rp.AllocatedMemory -= allocation.MemoryBytes
	rp.AllocatedDisk -= allocation.DiskBytes
	
	// Ensure we don't go negative
	if rp.AllocatedCPU < 0 {
		rp.AllocatedCPU = 0
	}
	if rp.AllocatedMemory < 0 {
		rp.AllocatedMemory = 0
	}
	if rp.AllocatedDisk < 0 {
		rp.AllocatedDisk = 0
	}
	
	// Ensure available doesn't exceed total
	if rp.AvailableCPU > rp.TotalCPU {
		rp.AvailableCPU = rp.TotalCPU
	}
	if rp.AvailableMemory > rp.TotalMemory {
		rp.AvailableMemory = rp.TotalMemory
	}
	if rp.AvailableDisk > rp.TotalDisk {
		rp.AvailableDisk = rp.TotalDisk
	}
}

// UpdateAvailableResources updates available resources based on current system state
func (rp *ResourcePool) UpdateAvailableResources() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	
	// Get current memory usage
	currentMemInfo, err := rp.getCurrentMemoryUsage()
	if err == nil {
		// Update available memory based on actual system usage
		usedMemory := rp.TotalMemory - currentMemInfo
		rp.AvailableMemory = rp.TotalMemory - rp.AllocatedMemory - usedMemory
		if rp.AvailableMemory < 0 {
			rp.AvailableMemory = 0
		}
	}
	
	// CPU and disk updates would be more complex and are simplified here
	// In a real implementation, these would monitor actual system usage
}

// getCurrentMemoryUsage gets current system memory usage
func (rp *ResourcePool) getCurrentMemoryUsage() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	
	lines := strings.Split(string(data), "\n")
	var memAvailable uint64
	
	for _, line := range lines {
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				memKB, err := strconv.ParseUint(fields[1], 10, 64)
				if err != nil {
					return 0, err
				}
				memAvailable = memKB * 1024 // Convert KB to bytes
				break
			}
		}
	}
	
	return memAvailable, nil
}

// GetResourceUsage returns current resource usage statistics
func (rp *ResourcePool) GetResourceUsage() map[string]interface{} {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	
	return map[string]interface{}{
		"cpu": map[string]interface{}{
			"total":     rp.TotalCPU,
			"allocated": rp.AllocatedCPU,
			"available": rp.AvailableCPU,
			"usage_percent": (rp.AllocatedCPU / rp.TotalCPU) * 100,
		},
		"memory": map[string]interface{}{
			"total":     rp.TotalMemory,
			"allocated": rp.AllocatedMemory,
			"available": rp.AvailableMemory,
			"usage_percent": (float64(rp.AllocatedMemory) / float64(rp.TotalMemory)) * 100,
		},
		"disk": map[string]interface{}{
			"total":     rp.TotalDisk,
			"allocated": rp.AllocatedDisk,
			"available": rp.AvailableDisk,
			"usage_percent": (float64(rp.AllocatedDisk) / float64(rp.TotalDisk)) * 100,
		},
	}
}

// CanAllocate checks if the requested resources can be allocated
func (rp *ResourcePool) CanAllocate(requested *ResourceAllocation) bool {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	
	return requested.CPUCores <= rp.AvailableCPU &&
		   requested.MemoryBytes <= rp.AvailableMemory &&
		   requested.DiskBytes <= rp.AvailableDisk
}

// GetAvailableResources returns currently available resources
func (rp *ResourcePool) GetAvailableResources() *ResourceAllocation {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	
	return &ResourceAllocation{
		CPUCores:    rp.AvailableCPU,
		MemoryBytes: rp.AvailableMemory,
		DiskBytes:   rp.AvailableDisk,
	}
}

// GetTotalResources returns total system resources
func (rp *ResourcePool) GetTotalResources() *ResourceAllocation {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	
	return &ResourceAllocation{
		CPUCores:    rp.TotalCPU,
		MemoryBytes: rp.TotalMemory,
		DiskBytes:   rp.TotalDisk,
	}
}
