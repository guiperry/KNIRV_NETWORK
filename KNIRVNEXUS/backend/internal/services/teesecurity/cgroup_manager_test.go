package teesecurity

import (
	"os"
	"path/filepath"
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

	// Verify the cgroup path ends with knirv-nexus/test-container
	// The actual path depends on the container environment (bare metal vs Docker)
	if !strings.HasSuffix(mgr.cgroupPath, "knirv-nexus/test-container") {
		t.Errorf("Expected cgroup path to end with knirv-nexus/test-container, got %s", mgr.cgroupPath)
	}

	// Verify the cgroup directory was created
	if _, err := os.Stat(mgr.cgroupPath); err != nil {
		t.Errorf("Cgroup directory was not created: %v", err)
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
		// In cgroup v2 with delegation (Docker containers), writing to controller files
		// may fail with permission denied if controllers aren't delegated to our cgroup
		if strings.Contains(err.Error(), "permission denied") {
			t.Skipf("Skipping test: cgroup controller delegation not available: %v", err)
		}
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

// TestDetectCgroupDelegation tests the cgroup delegation detection
func TestDetectCgroupDelegation(t *testing.T) {
	RequiresRoot(t, "TestDetectCgroupDelegation")
	RequiresCgroupAccess(t, "TestDetectCgroupDelegation")

	status, err := DetectCgroupDelegation()
	if err != nil {
		t.Fatalf("DetectCgroupDelegation failed: %v", err)
	}

	if status == nil {
		t.Fatal("DetectCgroupDelegation returned nil")
	}

	// Verify status fields are set
	if status.ControllerPath == "" {
		t.Error("ControllerPath should not be empty")
	}

	// Verify cgroup v2 support detection
	if status.SupportsV2 {
		// Verify controllers file exists
		if _, err := os.Stat(status.ControllerPath); err != nil {
			t.Errorf("Expected cgroup v2 controllers file at %s: %v", status.ControllerPath, err)
		}
	}

	t.Logf("Cgroup Delegation Status: IsDelegated=%v, CgroupParent=%s, SupportsV2=%v",
		status.IsDelegated, status.CgroupParent, status.SupportsV2)
}

// TestDetectCgroupDelegation_CgroupParentPath tests that cgroup parent path is correctly set
func TestDetectCgroupDelegation_CgroupParentPath(t *testing.T) {
	RequiresRoot(t, "TestDetectCgroupDelegation_CgroupParentPath")
	RequiresCgroupAccess(t, "TestDetectCgroupDelegation_CgroupParentPath")

	status, err := DetectCgroupDelegation()
	if err != nil {
		t.Fatalf("DetectCgroupDelegation failed: %v", err)
	}

	// The cgroup parent should be set to a valid path
	if status.CgroupParent == "" {
		t.Error("CgroupParent should not be empty")
	}

	// Verify the parent path exists or is accessible
	parentPath := status.CgroupParent
	if _, err := os.Stat(parentPath); os.IsNotExist(err) {
		// It might not exist yet, which is fine - we'll create it
		t.Logf("CgroupParent path %s does not exist yet (will be created)", parentPath)
	}
}

// TestNewCgroupManagerWithDelegation tests creating a cgroup manager with explicit delegation
func TestNewCgroupManagerWithDelegation(t *testing.T) {
	RequiresRoot(t, "TestNewCgroupManagerWithDelegation")
	RequiresCgroupAccess(t, "TestNewCgroupManagerWithDelegation")

	// Detect the actual writable cgroup parent
	actualDelegation, err := DetectCgroupDelegation()
	if err != nil {
		t.Fatalf("Failed to detect cgroup delegation: %v", err)
	}

	// Clean up existing test cgroup if it exists
	cleanupPath := filepath.Join(actualDelegation.CgroupParent, "knirv-nexus")
	if _, err := os.Stat(cleanupPath); err == nil {
		os.RemoveAll(cleanupPath)
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

	// Use the detected delegation status (uses actual writable cgroup parent)
	delegation := actualDelegation

	mgr, err := NewCgroupManagerWithDelegation("test-delegation", config, delegation)
	if err != nil {
		t.Fatalf("NewCgroupManagerWithDelegation failed: %v", err)
	}
	defer mgr.Cleanup()

	if mgr == nil {
		t.Fatal("NewCgroupManagerWithDelegation returned nil")
	}

	// Verify the delegation status was stored
	if mgr.delegation == nil {
		t.Error("CgroupManager should have delegation status")
	}

	if !mgr.delegation.IsDelegated {
		t.Error("Expected IsDelegated to be true")
	}

	// Verify the cgroup path uses the delegated parent
	expectedPath := filepath.Join(delegation.CgroupParent, "knirv-nexus", "test-delegation")
	if mgr.cgroupPath != expectedPath {
		t.Errorf("Expected cgroup path %s, got %s", expectedPath, mgr.cgroupPath)
	}
}

// TestNewCgroupManagerWithDelegation_CustomParent tests creating a cgroup manager with custom parent
func TestNewCgroupManagerWithDelegation_CustomParent(t *testing.T) {
	RequiresRoot(t, "TestNewCgroupManagerWithDelegation_CustomParent")
	RequiresCgroupAccess(t, "TestNewCgroupManagerWithDelegation_CustomParent")

	// Detect the actual writable cgroup parent
	actualDelegation, err := DetectCgroupDelegation()
	if err != nil {
		t.Fatalf("Failed to detect cgroup delegation: %v", err)
	}

	// Use a custom subdirectory under the writable cgroup parent
	customParent := filepath.Join(actualDelegation.CgroupParent, "knirv-delegation-test")

	// Clean up existing test cgroup if it exists
	if _, err := os.Stat(customParent); err == nil {
		os.RemoveAll(customParent)
	}

	// Create the custom parent directory so it exists before calling NewCgroupManagerWithDelegation
	if err := os.MkdirAll(customParent, 0755); err != nil {
		t.Fatalf("Failed to create custom parent directory: %v", err)
	}
	defer os.RemoveAll(customParent)

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

	// Create delegation status with custom parent (subdirectory under writable parent)
	delegation := &CgroupDelegationStatus{
		IsDelegated:    true,
		CgroupParent:   customParent,
		SupportsV2:     actualDelegation.SupportsV2,
		ControllerPath: actualDelegation.ControllerPath,
	}

	mgr, err := NewCgroupManagerWithDelegation("custom-parent-test", config, delegation)
	if err != nil {
		t.Fatalf("NewCgroupManagerWithDelegation with custom parent failed: %v", err)
	}
	defer mgr.Cleanup()

	// Verify the cgroup path uses the custom parent + knirv-nexus + container ID
	expectedPath := filepath.Join(customParent, "knirv-nexus", "custom-parent-test")
	if mgr.cgroupPath != expectedPath {
		t.Errorf("Expected cgroup path %s, got %s", expectedPath, mgr.cgroupPath)
	}

	// Verify the custom parent was created
	if _, err := os.Stat(filepath.Join(customParent, "knirv-nexus")); err != nil {
		t.Errorf("Custom parent cgroup was not created: %v", err)
	}
}

// TestIsRunningInContainer tests container detection
func TestIsRunningInContainer(t *testing.T) {
	// This test just verifies the function doesn't panic
	// In a container, it should return true; on bare metal, false
	_ = isRunningInContainer()
	t.Logf("isRunningInContainer returned (check manually if expected)")
}

// TestCanCreateSubcgroups tests sub-cgroup creation capability
func TestCanCreateSubcgroups(t *testing.T) {
	RequiresRoot(t, "TestCanCreateSubcgroups")
	RequiresCgroupAccess(t, "TestCanCreateSubcgroups")

	canCreate := canCreateSubcgroups()
	t.Logf("canCreateSubcgroups returned: %v", canCreate)

	// If we're in a privileged environment, we should be able to create sub-cgroups
	if os.Geteuid() == 0 {
		// In a properly configured environment, this should be true
		// But we don't fail the test if it's false - that might be expected in some container setups
		t.Logf("Running as root, canCreateSubcgroups=%v (may be false if not properly delegated)", canCreate)
	}
}

// TestFindDockerCgroupParent tests Docker cgroup parent detection
func TestFindDockerCgroupParent(t *testing.T) {
	RequiresRoot(t, "TestFindDockerCgroupParent")
	RequiresCgroupAccess(t, "TestFindDockerCgroupParent")

	parent := findDockerCgroupParent()
	t.Logf("findDockerCgroupParent returned: %s", parent)

	// The function should return a valid path or empty string
	if parent != "" {
		// Verify it's a reasonable path
		if !strings.HasPrefix(parent, "/sys/fs/cgroup") {
			t.Errorf("Expected path starting with /sys/fs/cgroup, got: %s", parent)
		}
	}
}

// TestGetCurrentCgroupPath tests getting the current cgroup path
func TestGetCurrentCgroupPath(t *testing.T) {
	path := getCurrentCgroupPath()
	t.Logf("getCurrentCgroupPath returned: %s", path)

	// The path should not be empty in a cgroup-aware environment
	if path == "" {
		t.Log("Warning: getCurrentCgroupPath returned empty string (might be expected in some environments)")
	}
}

// TestCgroupDelegationStatus_Struct tests the CgroupDelegationStatus struct fields
func TestCgroupDelegationStatus_Struct(t *testing.T) {
	status := &CgroupDelegationStatus{
		IsDelegated:     true,
		CgroupParent:    "/sys/fs/cgroup/test",
		CgroupNamespace: "",
		SupportsV2:      true,
		ControllerPath:  "/sys/fs/cgroup/cgroup.controllers",
	}

	if !status.IsDelegated {
		t.Error("IsDelegated should be true")
	}
	if status.CgroupParent != "/sys/fs/cgroup/test" {
		t.Errorf("Expected CgroupParent /sys/fs/cgroup/test, got %s", status.CgroupParent)
	}
	if !status.SupportsV2 {
		t.Error("SupportsV2 should be true")
	}
	if status.ControllerPath != "/sys/fs/cgroup/cgroup.controllers" {
		t.Errorf("Expected ControllerPath /sys/fs/cgroup/cgroup.controllers, got %s", status.ControllerPath)
	}
	if status.CgroupNamespace != "" {
		t.Errorf("Expected CgroupNamespace to be empty, got %s", status.CgroupNamespace)
	}
}
