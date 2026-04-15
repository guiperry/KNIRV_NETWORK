package cde

import (
	"testing"
)

func TestNewCDEResourcePool(t *testing.T) {
	pool, err := NewCDEResourcePool()
	if err != nil {
		t.Fatalf("Failed to create resource pool: %v", err)
	}

	if pool == nil {
		t.Fatal("Resource pool is nil")
	}

	// Check that resources are initialized
	if pool.TotalCPU <= 0 {
		t.Error("Total CPU should be greater than 0")
	}

	if pool.TotalMemory <= 0 {
		t.Error("Total memory should be greater than 0")
	}

	if pool.TotalDisk <= 0 {
		t.Error("Total disk should be greater than 0")
	}

	// Initially, available should equal total
	if pool.AvailableCPU != pool.TotalCPU {
		t.Errorf("Available CPU %f should equal total CPU %f", pool.AvailableCPU, pool.TotalCPU)
	}

	if pool.AvailableMemory != pool.TotalMemory {
		t.Errorf("Available memory %d should equal total memory %d", pool.AvailableMemory, pool.TotalMemory)
	}

	if pool.AvailableDisk != pool.TotalDisk {
		t.Errorf("Available disk %d should equal total disk %d", pool.AvailableDisk, pool.TotalDisk)
	}
}

func TestCanAllocate(t *testing.T) {
	pool, _ := NewCDEResourcePool()

	// Test allocation within limits
	request := &CDEResourceAllocation{
		CPUCores:    1.0,
		MemoryBytes: 1024 * 1024 * 1024,      // 1GB
		DiskBytes:   10 * 1024 * 1024 * 1024, // 10GB
	}

	if !pool.CanAllocate(request) {
		t.Error("Should be able to allocate within limits")
	}

	// Test allocation exceeding CPU
	excessiveRequest := &CDEResourceAllocation{
		CPUCores:    pool.TotalCPU + 1,
		MemoryBytes: 1024 * 1024 * 1024,
		DiskBytes:   10 * 1024 * 1024 * 1024,
	}

	if pool.CanAllocate(excessiveRequest) {
		t.Error("Should not be able to allocate more CPU than available")
	}

	// Test allocation exceeding memory
	excessiveRequest = &CDEResourceAllocation{
		CPUCores:    1.0,
		MemoryBytes: pool.TotalMemory + 1,
		DiskBytes:   10 * 1024 * 1024 * 1024,
	}

	if pool.CanAllocate(excessiveRequest) {
		t.Error("Should not be able to allocate more memory than available")
	}

	// Test allocation exceeding disk
	excessiveRequest = &CDEResourceAllocation{
		CPUCores:    1.0,
		MemoryBytes: 1024 * 1024 * 1024,
		DiskBytes:   pool.TotalDisk + 1,
	}

	if pool.CanAllocate(excessiveRequest) {
		t.Error("Should not be able to allocate more disk than available")
	}
}

func TestAllocateResources(t *testing.T) {
	pool, _ := NewCDEResourcePool()

	initialCPU := pool.AvailableCPU
	initialMemory := pool.AvailableMemory
	initialDisk := pool.AvailableDisk

	// Allocate some resources
	request := &CDEResourceAllocation{
		CPUCores:    1.0,
		MemoryBytes: 512 * 1024 * 1024,      // 512MB
		DiskBytes:   5 * 1024 * 1024 * 1024, // 5GB
	}

	err := pool.AllocateResources(request)
	if err != nil {
		t.Fatalf("Failed to allocate resources: %v", err)
	}

	// Check that available resources decreased
	if pool.AvailableCPU != initialCPU-1.0 {
		t.Errorf("Available CPU should be %f, got %f", initialCPU-1.0, pool.AvailableCPU)
	}

	if pool.AvailableMemory != initialMemory-512*1024*1024 {
		t.Errorf("Available memory should be %d, got %d", initialMemory-512*1024*1024, pool.AvailableMemory)
	}

	if pool.AvailableDisk != initialDisk-5*1024*1024*1024 {
		t.Errorf("Available disk should be %d, got %d", initialDisk-5*1024*1024*1024, pool.AvailableDisk)
	}

	// Check that allocated resources increased
	if pool.AllocatedCPU != 1.0 {
		t.Errorf("Allocated CPU should be 1.0, got %f", pool.AllocatedCPU)
	}

	if pool.AllocatedMemory != 512*1024*1024 {
		t.Errorf("Allocated memory should be %d, got %d", 512*1024*1024, pool.AllocatedMemory)
	}

	if pool.AllocatedDisk != 5*1024*1024*1024 {
		t.Errorf("Allocated disk should be %d, got %d", 5*1024*1024*1024, pool.AllocatedDisk)
	}
}

func TestReleaseResources(t *testing.T) {
	pool, _ := NewCDEResourcePool()

	// First allocate some resources
	request := &CDEResourceAllocation{
		CPUCores:    1.0,
		MemoryBytes: 512 * 1024 * 1024,
		DiskBytes:   5 * 1024 * 1024 * 1024,
	}

	err := pool.AllocateResources(request)
	if err != nil {
		t.Fatalf("Failed to allocate resources: %v", err)
	}

	allocatedCPU := pool.AllocatedCPU
	allocatedMemory := pool.AllocatedMemory
	allocatedDisk := pool.AllocatedDisk

	// Now release the resources
	pool.ReleaseResources(request)

	// Check that resources are returned to available pool
	if pool.AvailableCPU != pool.TotalCPU {
		t.Errorf("Available CPU should be back to total %f, got %f", pool.TotalCPU, pool.AvailableCPU)
	}

	if pool.AvailableMemory != pool.TotalMemory {
		t.Errorf("Available memory should be back to total %d, got %d", pool.TotalMemory, pool.AvailableMemory)
	}

	if pool.AvailableDisk != pool.TotalDisk {
		t.Errorf("Available disk should be back to total %d, got %d", pool.TotalDisk, pool.AvailableDisk)
	}

	// Check that allocated resources decreased
	if pool.AllocatedCPU != allocatedCPU-1.0 {
		t.Errorf("Allocated CPU should be %f, got %f", allocatedCPU-1.0, pool.AllocatedCPU)
	}

	if pool.AllocatedMemory != allocatedMemory-512*1024*1024 {
		t.Errorf("Allocated memory should be %d, got %d", allocatedMemory-512*1024*1024, pool.AllocatedMemory)
	}

	if pool.AllocatedDisk != allocatedDisk-5*1024*1024*1024 {
		t.Errorf("Allocated disk should be %d, got %d", allocatedDisk-5*1024*1024*1024, pool.AllocatedDisk)
	}
}

func TestGetResourceUsage(t *testing.T) {
	pool, _ := NewCDEResourcePool()

	usage := pool.GetResourceUsage()

	// Check that usage map contains expected keys
	if _, ok := usage["cpu"]; !ok {
		t.Error("Usage should contain 'cpu' key")
	}

	if _, ok := usage["memory"]; !ok {
		t.Error("Usage should contain 'memory' key")
	}

	if _, ok := usage["disk"]; !ok {
		t.Error("Usage should contain 'disk' key")
	}

	// Check CPU usage structure
	cpuUsage, ok := usage["cpu"].(map[string]interface{})
	if !ok {
		t.Error("CPU usage should be a map")
	} else {
		if _, ok := cpuUsage["total"]; !ok {
			t.Error("CPU usage should contain 'total'")
		}
		if _, ok := cpuUsage["allocated"]; !ok {
			t.Error("CPU usage should contain 'allocated'")
		}
		if _, ok := cpuUsage["available"]; !ok {
			t.Error("CPU usage should contain 'available'")
		}
		if _, ok := cpuUsage["usage_percent"]; !ok {
			t.Error("CPU usage should contain 'usage_percent'")
		}
	}
}

func TestGetAvailableResources(t *testing.T) {
	pool, _ := NewCDEResourcePool()

	available := pool.GetAvailableResources()

	if available.CPUCores != pool.AvailableCPU {
		t.Errorf("Available CPU cores should be %f, got %f", pool.AvailableCPU, available.CPUCores)
	}

	if available.MemoryBytes != pool.AvailableMemory {
		t.Errorf("Available memory should be %d, got %d", pool.AvailableMemory, available.MemoryBytes)
	}

	if available.DiskBytes != pool.AvailableDisk {
		t.Errorf("Available disk should be %d, got %d", pool.AvailableDisk, available.DiskBytes)
	}
}

func TestGetTotalResources(t *testing.T) {
	pool, _ := NewCDEResourcePool()

	total := pool.GetTotalResources()

	if total.CPUCores != pool.TotalCPU {
		t.Errorf("Total CPU cores should be %f, got %f", pool.TotalCPU, total.CPUCores)
	}

	if total.MemoryBytes != pool.TotalMemory {
		t.Errorf("Total memory should be %d, got %d", pool.TotalMemory, total.MemoryBytes)
	}

	if total.DiskBytes != pool.TotalDisk {
		t.Errorf("Total disk should be %d, got %d", pool.TotalDisk, total.DiskBytes)
	}
}

func TestUpdateAvailableResources(t *testing.T) {
	pool, _ := NewCDEResourcePool()

	initialAvailableMemory := pool.AvailableMemory

	// Update available resources
	pool.UpdateAvailableResources()

	// Available memory might change based on system usage, but should not exceed total
	if pool.AvailableMemory > pool.TotalMemory {
		t.Errorf("Available memory %d should not exceed total memory %d", pool.AvailableMemory, pool.TotalMemory)
	}

	// The exact value depends on system state, so we just check it's reasonable
	// (AvailableMemory is uint64, so it can't be negative)

	// Test that allocated resources are considered
	request := &CDEResourceAllocation{
		CPUCores:    0.5,
		MemoryBytes: 100 * 1024 * 1024,      // 100MB
		DiskBytes:   1 * 1024 * 1024 * 1024, // 1GB
	}

	err := pool.AllocateResources(request)
	if err != nil {
		t.Fatalf("Failed to allocate resources: %v", err)
	}

	pool.UpdateAvailableResources()

	// After allocation, available should be less than initial
	if pool.AvailableMemory >= initialAvailableMemory {
		t.Error("Available memory should decrease after allocation")
	}
}
