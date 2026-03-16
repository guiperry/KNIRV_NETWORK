package teesecurity

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestBasicIsolation tests basic container isolation functionality
func TestBasicIsolation(t *testing.T) {
	// Skip test if not running as root (user namespaces may not be available)
	if os.Geteuid() != 0 {
		t.Skip("Skipping test: requires root privileges for namespace creation")
	}

	// Create a mock KaliLinuxProfile
	kaliProfile := &KaliLinuxProfile{
		OS: "linux",
		NetworkAnalysisTools: KaliNetworkAnalysisTools{
			Tcpdump: false,
		},
		ForensicsTools: KaliForensicsTools{
			SleuthKit: false,
		},
	}

	// Create container runtime
	runtime, err := NewNativeContainerRuntime(kaliProfile)
	if err != nil {
		t.Fatalf("Failed to create container runtime: %v", err)
	}

	// Test with simple skill code
	opts := ContainerOptions{
		SkillCode: "echo 'Hello from container' && sleep 1",
		Env:       []string{"TEST=integration"},
	}

	// Run container
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := runtime.RunContainer(ctx, opts)
	if err != nil {
		t.Errorf("RunContainer failed: %v", err)
	}

	if result == nil {
		t.Fatal("RunContainer returned nil result")
	}

	// Verify result
	if result.ContainerID == "" {
		t.Error("ContainerID should not be empty")
	}

	if result.ExecutionTime <= 0 {
		t.Error("ExecutionTime should be positive")
	}

	// Verify resource usage is tracked
	if result.ResourceUsage == nil {
		t.Error("ResourceUsage should not be nil")
	} else {
		if result.ResourceUsage.MemoryUsage < 0 {
			t.Error("MemoryUsage should not be negative")
		}
		if result.ResourceUsage.PIDsUsed < 0 {
			t.Error("PIDsUsed should not be negative")
		}
	}
}

// TestHardenedContainerExecution tests hardened container execution
func TestHardenedContainerExecution(t *testing.T) {
	// Skip test if not running as root
	if os.Geteuid() != 0 {
		t.Skip("Skipping test: requires root privileges")
	}

	// Skip test if cgroups v2 is not available
	if _, err := os.Stat("/sys/fs/cgroup/knirv-server"); err == nil {
		// Clean up existing test cgroup if it exists
		os.RemoveAll("/sys/fs/cgroup/knirv-server")
	}

	// Create a mock KaliLinuxProfile
	kaliProfile := &KaliLinuxProfile{
		OS: "linux",
		NetworkAnalysisTools: KaliNetworkAnalysisTools{
			Tcpdump: false,
		},
		ForensicsTools: KaliForensicsTools{
			SleuthKit: false,
		},
	}

	// Create container runtime
	runtime, err := NewNativeContainerRuntime(kaliProfile)
	if err != nil {
		t.Fatalf("Failed to create container runtime: %v", err)
	}

	// Ensure hardening is enabled
	runtime.config.EnableHardening = true
	runtime.config.FallbackToMonitoring = false

	// Test with simple skill code
	opts := ContainerOptions{
		SkillCode: "echo 'Hello from hardened container' && sleep 1",
		Env:       []string{"TEST=hardened"},
	}

	// Run container
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := runtime.RunContainer(ctx, opts)
	if err != nil {
		// Check if this is a permission error (expected in non-privileged environments)
		if strings.Contains(err.Error(), "permission denied") {
			t.Skipf("Skipping test: Hardened container execution requires root privileges: %v", err)
		} else {
			t.Errorf("Hardened RunContainer failed: %v", err)
		}
	}

	if result == nil {
		t.Fatal("Hardened RunContainer returned nil result")
	}

	// Verify result
	if result.ContainerID == "" {
		t.Error("ContainerID should not be empty")
	}

	if result.ExecutionTime <= 0 {
		t.Error("ExecutionTime should be positive")
	}

	// Verify resource usage is tracked
	if result.ResourceUsage == nil {
		t.Error("ResourceUsage should not be nil")
	} else {
		if result.ResourceUsage.MemoryUsage < 0 {
			t.Error("MemoryUsage should not be negative")
		}
		if result.ResourceUsage.PIDsUsed < 0 {
			t.Error("PIDsUsed should not be negative")
		}
	}
}

// TestContainerWithResourceLimits tests container execution with resource limits
func TestContainerWithResourceLimits(t *testing.T) {
	// Skip test if not running as root
	if os.Geteuid() != 0 {
		t.Skip("Skipping test: requires root privileges")
	}

	// Skip test if cgroups v2 is not available
	if _, err := os.Stat("/sys/fs/cgroup/knirv-server"); err == nil {
		// Clean up existing test cgroup if it exists
		os.RemoveAll("/sys/fs/cgroup/knirv-server")
	}

	// Create a mock KaliLinuxProfile
	kaliProfile := &KaliLinuxProfile{
		OS: "linux",
		NetworkAnalysisTools: KaliNetworkAnalysisTools{
			Tcpdump: false,
		},
		ForensicsTools: KaliForensicsTools{
			SleuthKit: false,
		},
	}

	// Create container runtime
	runtime, err := NewNativeContainerRuntime(kaliProfile)
	if err != nil {
		t.Fatalf("Failed to create container runtime: %v", err)
	}

	// Configure resource limits
	runtime.config.Cgroups.Memory.Limit = 100 * 1024 * 1024 // 100 MB
	runtime.config.Cgroups.CPU.Quota = 50000                // 0.5 CPU
	runtime.config.Cgroups.PIDs.Max = 100                   // Max 100 processes

	// Test with memory-intensive skill code
	opts := ContainerOptions{
		SkillCode: `echo 'Testing resource limits' && 
		          dd if=/dev/zero of=/tmp/testfile bs=1M count=50 2>/dev/null && 
		          rm -f /tmp/testfile`,
		Env: []string{"TEST=limits"},
	}

	// Run container
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := runtime.RunContainer(ctx, opts)
	if err != nil {
		t.Errorf("Resource-limited RunContainer failed: %v", err)
	}

	if result == nil {
		t.Fatal("Resource-limited RunContainer returned nil result")
	}

	// Verify result
	if result.ContainerID == "" {
		t.Error("ContainerID should not be empty")
	}

	if result.ExecutionTime <= 0 {
		t.Error("ExecutionTime should be positive")
	}

	// Verify resource usage is tracked and within limits
	if result.ResourceUsage == nil {
		t.Error("ResourceUsage should not be nil")
	} else {
		if result.ResourceUsage.MemoryUsage < 0 {
			t.Error("MemoryUsage should not be negative")
		}
		if result.ResourceUsage.MemoryUsage > runtime.config.Cgroups.Memory.Limit {
			t.Errorf("MemoryUsage %d exceeds limit %d",
				result.ResourceUsage.MemoryUsage, runtime.config.Cgroups.Memory.Limit)
		}
		if result.ResourceUsage.PIDsUsed < 0 {
			t.Error("PIDsUsed should not be negative")
		}
	}
}

// TestFallbackToMonitoring tests fallback to monitoring when hardening fails
func TestFallbackToMonitoring(t *testing.T) {
	// Create a mock KaliLinuxProfile
	kaliProfile := &KaliLinuxProfile{
		OS: "linux",
		NetworkAnalysisTools: KaliNetworkAnalysisTools{
			Tcpdump: false,
		},
		ForensicsTools: KaliForensicsTools{
			SleuthKit: false,
		},
	}

	// Create container runtime
	runtime, err := NewNativeContainerRuntime(kaliProfile)
	if err != nil {
		t.Fatalf("Failed to create container runtime: %v", err)
	}

	// Enable hardening with fallback
	runtime.config.EnableHardening = true
	runtime.config.FallbackToMonitoring = true

	// Test with simple skill code
	opts := ContainerOptions{
		SkillCode: "echo 'Testing fallback to monitoring' && sleep 1",
		Env:       []string{"TEST=fallback"},
	}

	// Run container
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := runtime.RunContainer(ctx, opts)
	if err != nil {
		// Skip if the environment lacks required privileges (cgroups, /var/tmp write)
		if strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "read-only filesystem") {
			t.Skipf("Skipping: insufficient privileges for container isolation test: %v", err)
		}
		t.Errorf("Fallback RunContainer failed: %v", err)
	}

	if result == nil {
		t.Fatal("Fallback RunContainer returned nil result")
	}

	// Verify result
	if result.ContainerID == "" {
		t.Error("ContainerID should not be empty")
	}

	if result.ExecutionTime <= 0 {
		t.Error("ExecutionTime should be positive")
	}
}
