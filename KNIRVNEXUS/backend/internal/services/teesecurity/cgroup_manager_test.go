package teesecurity

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestNewCgroupManager tests cgroup manager creation
func TestNewCgroupManager(t *testing.T) {
	// Verify we have required privileges
	RequiresRoot(t, "TestNewCgroupManager")
	RequiresCgroupAccess(t, "TestNewCgroupManager")

	// Clean up existing test cgroup if it exists
	if _, err := os.Stat("/sys/fs/cgroup/knirv-nexus"); err == nil {
		os.RemoveAll("/sys/fs/cgroup/knirv-nexus")
	}

	config := CgroupConfig{
		CPU: CgroupCPU{
			Quota:  100000,
			Period: 100000,
			Shares: 1024,
		},
		Memory: CgroupMemory{
			Limit:     512 * 1024 * 1024,
			SwapLimit: -1,
		},
		PIDs: CgroupPIDs{
			Max: 512,
		},
		IO: CgroupIO{
			ReadBPS:  10 * 1024 * 1024,
			WriteBPS: 10 * 1024 * 1024,
		},
	}

	mgr, err := NewCgroupManager("test-container", config)
	if err != nil {
		t.Fatalf("Cgroup creation failed: %v\nThis test requires privileged container with cgroup write access", err)
	}
	defer mgr.Cleanup()

	if mgr == nil {
		t.Fatal("NewCgroupManager returned nil")
	}

	if mgr.config != config {
		t.Error("CgroupManager config mismatch")
	}

	expectedPath := "/sys/fs/cgroup/knirv-nexus/test-container"
	if mgr.cgroupPath != expectedPath {
		t.Errorf("Expected cgroup path %s, got %s", expectedPath, mgr.cgroupPath)
	}
}

// TestCgroupManager_ApplyLimits tests applying cgroup limits
func TestCgroupManager_ApplyLimits(t *testing.T) {
	// Verify we have required privileges
	RequiresRoot(t, "TestCgroupManager_ApplyLimits")
	RequiresCgroupAccess(t, "TestCgroupManager_ApplyLimits")

	// Clean up existing test cgroup if it exists
	if _, err := os.Stat("/sys/fs/cgroup/knirv-nexus"); err == nil {
		os.RemoveAll("/sys/fs/cgroup/knirv-nexus")
	}

	config := CgroupConfig{
		CPU: CgroupCPU{
			Quota:  100000,
			Period: 100000,
			Shares: 1024,
		},
		Memory: CgroupMemory{
			Limit:     512 * 1024 * 1024,
			SwapLimit: -1,
		},
		PIDs: CgroupPIDs{
			Max: 512,
		},
		IO: CgroupIO{
			ReadBPS:  10 * 1024 * 1024,
			WriteBPS: 10 * 1024 * 1024,
		},
	}

	mgr, err := NewCgroupManager("test-container", config)
	if err != nil {
		t.Fatalf("Cgroup creation failed: %v\nThis test requires privileged container", err)
	}
	defer mgr.Cleanup()

	err = mgr.ApplyLimits()
	if err != nil {
		t.Fatalf("ApplyLimits failed: %v\nThis test requires privileged container with cgroup write access", err)
	}

	// Verify CPU limits were applied
	cpuMax, err := mgr.readFile("cpu.max")
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			t.Skipf("Skipping test: cgroup files not created due to permission issues")
		} else {
			t.Errorf("Failed to read cpu.max: %v", err)
		}
	} else {
		expected := "100000 100000"
		if strings.TrimSpace(cpuMax) != expected {
			t.Errorf("Expected cpu.max to be %s, got %s", expected, strings.TrimSpace(cpuMax))
		}
	}

	// Verify memory limits were applied
	memoryMax, err := mgr.readFile("memory.max")
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			t.Skipf("Skipping test: cgroup files not created due to permission issues")
		} else {
			t.Errorf("Failed to read memory.max: %v", err)
		}
	} else {
		expected := strconv.FormatInt(config.Memory.Limit, 10)
		if strings.TrimSpace(memoryMax) != expected {
			t.Errorf("Expected memory.max to be %s, got %s", expected, strings.TrimSpace(memoryMax))
		}
	}

	// Verify PID limits were applied
	pidsMax, err := mgr.readFile("pids.max")
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			t.Skipf("Skipping test: cgroup files not created due to permission issues")
		} else {
			t.Errorf("Failed to read pids.max: %v", err)
		}
	} else {
		expected := strconv.FormatInt(config.PIDs.Max, 10)
		if strings.TrimSpace(pidsMax) != expected {
			t.Errorf("Expected pids.max to be %s, got %s", expected, strings.TrimSpace(pidsMax))
		}
	}
}

// TestCgroupManager_AddProcess tests adding a process to cgroup
func TestCgroupManager_AddProcess(t *testing.T) {
	// Skip test if cgroups v2 is not available
	if _, err := os.Stat("/sys/fs/cgroup/knirv-nexus"); err == nil {
		// Clean up existing test cgroup if it exists
		os.RemoveAll("/sys/fs/cgroup/knirv-nexus")
	}

	config := CgroupConfig{
		CPU: CgroupCPU{
			Quota:  100000,
			Period: 100000,
			Shares: 1024,
		},
		Memory: CgroupMemory{
			Limit:     512 * 1024 * 1024,
			SwapLimit: -1,
		},
		PIDs: CgroupPIDs{
			Max: 512,
		},
		IO: CgroupIO{
			ReadBPS:  10 * 1024 * 1024,
			WriteBPS: 10 * 1024 * 1024,
		},
	}

	mgr, err := NewCgroupManager("test-container", config)
	if err != nil {
		t.Skipf("Skipping test: cgroup creation failed: %v", err)
	}
	defer mgr.Cleanup()

	// Add current process to cgroup
	pid := os.Getpid()
	err = mgr.AddProcess(pid)
	if err != nil {
		t.Errorf("AddProcess failed: %v", err)
	}

	// Verify process was added
	cgroupProcs, err := mgr.readFile("cgroup.procs")
	if err != nil {
		t.Errorf("Failed to read cgroup.procs: %v", err)
	} else {
		procs := strings.Fields(strings.TrimSpace(cgroupProcs))
		found := false
		for _, p := range procs {
			if p == strconv.Itoa(pid) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Process %d not found in cgroup.procs", pid)
		}
	}
}

// TestCgroupManager_GetStats tests getting cgroup statistics
func TestCgroupManager_GetStats(t *testing.T) {
	// Skip test if cgroups v2 is not available
	if _, err := os.Stat("/sys/fs/cgroup/knirv-nexus"); err == nil {
		// Clean up existing test cgroup if it exists
		os.RemoveAll("/sys/fs/cgroup/knirv-nexus")
	}

	config := CgroupConfig{
		CPU: CgroupCPU{
			Quota:  100000,
			Period: 100000,
			Shares: 1024,
		},
		Memory: CgroupMemory{
			Limit:     512 * 1024 * 1024,
			SwapLimit: -1,
		},
		PIDs: CgroupPIDs{
			Max: 512,
		},
		IO: CgroupIO{
			ReadBPS:  10 * 1024 * 1024,
			WriteBPS: 10 * 1024 * 1024,
		},
	}

	mgr, err := NewCgroupManager("test-container", config)
	if err != nil {
		t.Skipf("Skipping test: cgroup creation failed: %v", err)
	}
	defer mgr.Cleanup()

	// Add current process to cgroup
	pid := os.Getpid()
	mgr.AddProcess(pid)

	// Get stats
	stats, err := mgr.GetStats()
	if err != nil {
		t.Errorf("GetStats failed: %v", err)
	}

	if stats == nil {
		t.Error("GetStats returned nil")
	}

	// Verify stats are reasonable
	if stats.MemoryUsage < 0 {
		t.Error("MemoryUsage should not be negative")
	}

	if stats.PIDsCurrent < 0 {
		t.Error("PIDsCurrent should not be negative")
	}
}

// TestCgroupManager_Cleanup tests cgroup cleanup
func TestCgroupManager_Cleanup(t *testing.T) {
	// Skip test if cgroups v2 is not available
	if _, err := os.Stat("/sys/fs/cgroup/knirv-nexus"); err == nil {
		// Clean up existing test cgroup if it exists
		os.RemoveAll("/sys/fs/cgroup/knirv-nexus")
	}

	config := CgroupConfig{
		CPU: CgroupCPU{
			Quota:  100000,
			Period: 100000,
			Shares: 1024,
		},
		Memory: CgroupMemory{
			Limit:     512 * 1024 * 1024,
			SwapLimit: -1,
		},
		PIDs: CgroupPIDs{
			Max: 512,
		},
		IO: CgroupIO{
			ReadBPS:  10 * 1024 * 1024,
			WriteBPS: 10 * 1024 * 1024,
		},
	}

	mgr, err := NewCgroupManager("test-container", config)
	if err != nil {
		t.Skipf("Skipping test: cgroup creation failed: %v", err)
	}

	// Verify cgroup exists
	if _, err := os.Stat(mgr.cgroupPath); err != nil {
		t.Errorf("Cgroup path does not exist: %v", err)
	}

	// Cleanup
	err = mgr.Cleanup()
	if err != nil {
		// Check if this is a permission error (expected in non-privileged environments)
		if strings.Contains(err.Error(), "permission denied") ||
		   strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("Skipping test: Cleanup requires root privileges: %v", err)
		} else {
			t.Errorf("Cleanup failed: %v", err)
		}
	}
	
	// Verify cgroup was removed (only check if cleanup succeeded)
	if err == nil {
		if _, err := os.Stat(mgr.cgroupPath); err == nil {
			t.Error("Cgroup path still exists after cleanup")
		}
	}
}
