package teesecurity

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CgroupManager manages cgroup v2 configuration and resource limits
type CgroupManager struct {
	cgroupPath string  // /sys/fs/cgroup/knirv-nexus/skill-<id>
	config     CgroupConfig
}

// NewCgroupManager creates a new CgroupManager instance
func NewCgroupManager(containerID string, config CgroupConfig) (*CgroupManager, error) {
	// Use cgroups v2 unified hierarchy
	basePath := "/sys/fs/cgroup/knirv-nexus"

	// Check if cgroup filesystem is mounted and writable
	if err := verifyCgroupWritable(); err != nil {
		return nil, fmt.Errorf("cgroup filesystem not writable: %w\n"+
			"Ensure container is running with:\n"+
			"  - Docker: --privileged --cgroupns=host -v /sys/fs/cgroup:/sys/fs/cgroup:rw\n"+
			"  - Kata: privileged_without_host_devices=true with cgroup delegation", err)
	}

	// Create parent cgroup if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base cgroup directory: %w", err)
	}

	// Enable controllers in the parent cgroup (required for cgroup v2)
	if err := enableCgroupControllers(basePath); err != nil {
		return nil, fmt.Errorf("failed to enable cgroup controllers: %w", err)
	}

	// Create container-specific cgroup
	cgroupPath := filepath.Join(basePath, containerID)
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create container cgroup: %w", err)
	}

	return &CgroupManager{
		cgroupPath: cgroupPath,
		config:     config,
	}, nil
}

// verifyCgroupWritable checks if the cgroup filesystem is mounted and writable
func verifyCgroupWritable() error {
	cgroupBase := "/sys/fs/cgroup"

	// Check if cgroup directory exists
	if _, err := os.Stat(cgroupBase); os.IsNotExist(err) {
		return fmt.Errorf("cgroup directory %s does not exist", cgroupBase)
	}

	// Try to create a test directory to verify write permissions
	testPath := filepath.Join(cgroupBase, "knirv-objects")
	if err := os.MkdirAll(testPath, 0755); err != nil {
		return fmt.Errorf("cannot create directory in %s (read-only filesystem): %w", cgroupBase, err)
	}

	// Test passed, directory is writable
	return nil
}

// enableCgroupControllers enables all available controllers in a cgroup
// This is required for cgroup v2 - controllers must be enabled in parent before child can use them
func enableCgroupControllers(cgroupPath string) error {
	// Read available controllers from root cgroup
	rootControllers, err := os.ReadFile("/sys/fs/cgroup/cgroup.controllers")
	if err != nil {
		// If we can't read controllers, it might be cgroup v1 or permission issue
		// Try to continue anyway
		return nil
	}

	controllers := strings.Fields(string(rootControllers))
	if len(controllers) == 0 {
		// No controllers available, nothing to enable
		return nil
	}

	// For the parent cgroup, we need to enable controllers in its parent first
	// Get the parent of our basePath
	parentPath := filepath.Dir(cgroupPath)

	// If parent is root cgroup, read from root subtree_control
	if parentPath == "/sys/fs/cgroup" || parentPath == "." {
		// Enable controllers in root cgroup's subtree_control
		subtreeControlPath := "/sys/fs/cgroup/cgroup.subtree_control"

		// Build controller enable string (e.g., "+cpu +memory +io +pids")
		var enableStr string
		for _, controller := range controllers {
			enableStr += "+" + controller + " "
		}
		enableStr = strings.TrimSpace(enableStr)

		// Write to subtree_control to enable controllers
		if err := os.WriteFile(subtreeControlPath, []byte(enableStr), 0644); err != nil {
			// This might fail if already enabled or not supported - that's okay
			// We'll check if individual operations succeed later
		}
	} else {
		// Enable controllers in the parent's subtree_control
		parentSubtreeControl := filepath.Join(parentPath, "cgroup.subtree_control")

		// Build controller enable string
		var enableStr string
		for _, controller := range controllers {
			enableStr += "+" + controller + " "
		}
		enableStr = strings.TrimSpace(enableStr)

		// Write to parent's subtree_control
		if err := os.WriteFile(parentSubtreeControl, []byte(enableStr), 0644); err != nil {
			// This might fail if already enabled - that's okay
		}
	}

	// Now enable controllers in this cgroup's subtree_control for its children
	subtreeControlPath := filepath.Join(cgroupPath, "cgroup.subtree_control")

	// Build controller enable string
	var enableStr string
	for _, controller := range controllers {
		enableStr += "+" + controller + " "
	}
	enableStr = strings.TrimSpace(enableStr)

	// Write to subtree_control
	if err := os.WriteFile(subtreeControlPath, []byte(enableStr), 0644); err != nil {
		// This might fail if already enabled or if we don't have processes in this cgroup yet
		// That's okay - we'll be able to write to controller files anyway
	}

	return nil
}

// ApplyLimits applies all configured resource limits
func (cm *CgroupManager) ApplyLimits() error {
	// CPU limits
	if err := cm.applyCPULimits(); err != nil {
		return err
	}

	// Memory limits
	if err := cm.applyMemoryLimits(); err != nil {
		return err
	}

	// I/O limits
	if err := cm.applyIOLimits(); err != nil {
		return err
	}

	// PID limits
	if err := cm.applyPIDLimits(); err != nil {
		return err
	}

	return nil
}

// applyCPULimits applies CPU resource limits
func (cm *CgroupManager) applyCPULimits() error {
	// Write cpu.max: "quota period"
	cpuMax := fmt.Sprintf("%d %d", cm.config.CPU.Quota, cm.config.CPU.Period)
	if err := cm.writeFile("cpu.max", cpuMax); err != nil {
		return fmt.Errorf("failed to set CPU limit: %w", err)
	}

	// Write cpu.weight (shares)
	if err := cm.writeFile("cpu.weight", strconv.FormatInt(cm.config.CPU.Shares, 10)); err != nil {
		return fmt.Errorf("failed to set CPU shares: %w", err)
	}

	return nil
}

// applyMemoryLimits applies memory resource limits
func (cm *CgroupManager) applyMemoryLimits() error {
	// Write memory.max
	if err := cm.writeFile("memory.max", strconv.FormatInt(cm.config.Memory.Limit, 10)); err != nil {
		return fmt.Errorf("failed to set memory limit: %w", err)
	}

	// Write memory.swap.max
	swapLimit := "0"  // Disable swap by default
	if cm.config.Memory.SwapLimit > 0 {
		swapLimit = strconv.FormatInt(cm.config.Memory.SwapLimit, 10)
	}
	if err := cm.writeFile("memory.swap.max", swapLimit); err != nil {
		return fmt.Errorf("failed to set swap limit: %w", err)
	}

	// Write memory.oom.group if OOM killer should be disabled
	if cm.config.Memory.OOMKillDisable {
		if err := cm.writeFile("memory.oom.group", "1"); err != nil {
			return fmt.Errorf("failed to set OOM group: %w", err)
		}
	}

	return nil
}

// applyPIDLimits applies process limits
func (cm *CgroupManager) applyPIDLimits() error {
	// Write pids.max
	if err := cm.writeFile("pids.max", strconv.FormatInt(cm.config.PIDs.Max, 10)); err != nil {
		return fmt.Errorf("failed to set PID limit: %w", err)
	}

	return nil
}

// applyIOLimits applies I/O resource limits
func (cm *CgroupManager) applyIOLimits() error {
	// Get device major:minor for root device
	// For simplicity, hardcode common values (8:0 for /dev/sda)
	// In production, detect dynamically
	device := "8:0"

	// Write io.max: "major:minor rbps=X wbps=Y"
	ioMax := fmt.Sprintf("%s rbps=%d wbps=%d",
		device,
		cm.config.IO.ReadBPS,
		cm.config.IO.WriteBPS,
	)
	if err := cm.writeFile("io.max", ioMax); err != nil {
		// Non-fatal if device not found
		fmt.Printf("Warning: failed to set I/O limits: %v\n", err)
	}

	return nil
}

// AddProcess adds a PID to this cgroup
func (cm *CgroupManager) AddProcess(pid int) error {
	return cm.writeFile("cgroup.procs", strconv.Itoa(pid))
}

// GetStats retrieves resource usage statistics
func (cm *CgroupManager) GetStats() (*CgroupStats, error) {
	stats := &CgroupStats{}

	// Read CPU stats
	cpuStat, _ := cm.readFile("cpu.stat")
	stats.CPUStat = cpuStat

	// Read memory stats
	memoryCurrent, _ := cm.readFile("memory.current")
	if memoryCurrent != "" {
		stats.MemoryUsage, _ = strconv.ParseInt(strings.TrimSpace(memoryCurrent), 10, 64)
	}

	memoryPeak, _ := cm.readFile("memory.peak")
	if memoryPeak != "" {
		stats.MemoryPeak, _ = strconv.ParseInt(strings.TrimSpace(memoryPeak), 10, 64)
	}

	// Read PID stats
	pidsCurrent, _ := cm.readFile("pids.current")
	if pidsCurrent != "" {
		stats.PIDsCurrent, _ = strconv.Atoi(strings.TrimSpace(pidsCurrent))
	}

	return stats, nil
}

// Cleanup removes the cgroup
func (cm *CgroupManager) Cleanup() error {
	// First, move all processes out of this cgroup
	if err := cm.writeFile("cgroup.procs", "0"); err != nil {
		// Log the error but try to continue cleanup, as some cgroups might not have procs writable
		fmt.Printf("Warning: failed to move processes out of cgroup %s: %v\n", cm.cgroupPath, err)
	}

	// Then, remove the cgroup directory
	return os.RemoveAll(cm.cgroupPath)
}

// writeFile writes a value to a cgroup control file
func (cm *CgroupManager) writeFile(name, value string) error {
	path := filepath.Join(cm.cgroupPath, name)
	return os.WriteFile(path, []byte(value), 0644)
}

// readFile reads a value from a cgroup control file
func (cm *CgroupManager) readFile(name string) (string, error) {
	path := filepath.Join(cm.cgroupPath, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
