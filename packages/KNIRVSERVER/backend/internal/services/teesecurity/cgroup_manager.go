package teesecurity

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CgroupDelegationStatus holds information about cgroup delegation status
type CgroupDelegationStatus struct {
	IsDelegated     bool   // True if cgroups are properly delegated
	CgroupParent    string // The parent cgroup path being used
	CgroupNamespace string // The cgroup namespace (if available)
	SupportsV2      bool   // True if cgroups v2 is in use
	ControllerPath  string // Path to controllers file
}

// CgroupManager manages cgroup v2 configuration and resource limits
type CgroupManager struct {
	cgroupPath string // /sys/fs/cgroup/knirv-server/skill-<id>
	config     CgroupConfig
	delegation *CgroupDelegationStatus
}

// NewCgroupManager creates a new CgroupManager instance
func NewCgroupManager(containerID string, config CgroupConfig) (*CgroupManager, error) {
	return NewCgroupManagerWithDelegation(containerID, config, nil)
}

// NewCgroupManagerWithDelegation creates a new CgroupManager with explicit delegation settings
func NewCgroupManagerWithDelegation(containerID string, config CgroupConfig, delegation *CgroupDelegationStatus) (*CgroupManager, error) {
	// Use cgroups v2 unified hierarchy
	basePath := "/sys/fs/cgroup/knirv-server"

	// Detect cgroup delegation status if not provided
	if delegation == nil {
		var err error
		delegation, err = DetectCgroupDelegation()
		if err != nil {
			return nil, fmt.Errorf("failed to detect cgroup delegation: %w", err)
		}
	}

	// Use delegated parent if available, otherwise use default path
	if delegation.IsDelegated && delegation.CgroupParent != "" {
		basePath = filepath.Join(delegation.CgroupParent, "knirv-server")
	}

	// Check if cgroup filesystem is mounted and writable
	if err := verifyCgroupWritable(basePath); err != nil {
		return nil, fmt.Errorf("cgroup filesystem not writable: %w\n"+
			"Ensure container is running with:\n"+
			"  - Docker: --privileged --cgroupns=host -v /sys/fs/cgroup:/sys/fs/cgroup:rw --cgroup-parent=docker\n"+
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
		delegation: delegation,
	}, nil
}

// DetectCgroupDelegation checks if cgroups are properly delegated in the current environment
func DetectCgroupDelegation() (*CgroupDelegationStatus, error) {
	status := &CgroupDelegationStatus{
		IsDelegated:    false,
		CgroupParent:   "",
		SupportsV2:     false,
		ControllerPath: "/sys/fs/cgroup/cgroup.controllers",
	}

	// Check if cgroup v2 is in use (unified hierarchy)
	// In cgroup v2, cgroup.controllers exists at root
	if _, err := os.Stat(status.ControllerPath); err == nil {
		status.SupportsV2 = true
	} else {
		// Check for cgroup v1
		if _, err := os.Stat("/sys/fs/cgroup/cpu"); err == nil {
			status.SupportsV2 = false
			return status, nil // v1 doesn't need the same delegation handling
		}
		return status, fmt.Errorf("cgroup filesystem not mounted at /sys/fs/cgroup")
	}

	// Check for Docker container environment
	if isRunningInContainer() {
		// Try to find the container's writable cgroup path
		dockerCgroupPath := findDockerCgroupParent()
		if dockerCgroupPath != "" {
			status.IsDelegated = true
			status.CgroupParent = dockerCgroupPath
		} else {
			// If we can't find a writable cgroup path, we're not delegated
			status.IsDelegated = false
			status.CgroupParent = "/sys/fs/cgroup"
		}
	} else {
		// Bare metal - full access
		status.IsDelegated = true
		status.CgroupParent = "/sys/fs/cgroup"
	}

	return status, nil
}

// isRunningInContainer checks if we're running inside a container
func isRunningInContainer() bool {
	// Check for Docker-specific files
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}
	// Check cgroup for Docker patterns
	if isDockerCgroup() {
		return true
	}
	return false
}

// getCurrentCgroupPath returns the current process's cgroup path
func getCurrentCgroupPath() string {
	// Read /proc/self/cgroup to get our cgroup path
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "0:") {
			parts := strings.SplitN(line, ":", 3)
			if len(parts) >= 3 {
				return parts[2]
			}
		}
	}
	return ""
}

// canCreateSubcgroups tests if we can create sub-cgroups in the current environment
func canCreateSubcgroups() bool {
	testPath := "/sys/fs/cgroup/knirv-test-subgroup"
	if err := os.MkdirAll(testPath, 0755); err == nil {
		os.RemoveAll(testPath)
		return true
	}
	return false
}

// findDockerCgroupParent attempts to find the Docker cgroup parent path
func findDockerCgroupParent() string {
	// Read /proc/self/cgroup to find our container's cgroup
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return ""
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// In cgroup v2, there's only one line with format: 0::/path/to/cgroup
	// In cgroup v1, there are multiple lines with format: id:controller:/path
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Split by colon
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		cgroupPath := strings.TrimSpace(parts[2])
		if cgroupPath == "" || cgroupPath == "/" {
			// Root cgroup - use /sys/fs/cgroup directly
			continue
		}

		// Remove leading slash for consistency
		cgroupPath = strings.TrimPrefix(cgroupPath, "/")

		// If the path contains "knirv-server", strip it and everything after it
		// This prevents path doubling when tests run inside cgroups they created
		if idx := strings.Index(cgroupPath, "/knirv-server"); idx >= 0 {
			cgroupPath = cgroupPath[:idx]
		}

		// Build full path: /sys/fs/cgroup + cgroup path
		fullPath := filepath.Join("/sys/fs/cgroup", cgroupPath)

		// Test if we can write to this path
		testPath := filepath.Join(fullPath, "knirv-test")
		if err := os.MkdirAll(testPath, 0755); err == nil {
			os.RemoveAll(testPath)
			return fullPath
		}
	}

	// Fallback: try to write to /sys/fs/cgroup root
	testPath := filepath.Join("/sys/fs/cgroup", "knirv-test")
	if err := os.MkdirAll(testPath, 0755); err == nil {
		os.RemoveAll(testPath)
		return "/sys/fs/cgroup"
	}

	return ""
}

// verifyCgroupWritable checks if the cgroup filesystem is mounted and writable
func verifyCgroupWritable(cgroupBase string) error {
	// Get the parent directory to check if it's writable
	// We need to verify the parent is writable so we can create cgroupBase
	parentDir := filepath.Dir(cgroupBase)

	// Check if parent cgroup directory exists (usually /sys/fs/cgroup)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		return fmt.Errorf("parent cgroup directory %s does not exist", parentDir)
	}

	// Try to create a test directory in the parent to verify write permissions
	testPath := filepath.Join(parentDir, "knirv-test")
	if err := os.MkdirAll(testPath, 0755); err != nil {
		return fmt.Errorf("cannot create directory in %s (read-only filesystem): %w", parentDir, err)
	}

	// Clean up test directory
	os.RemoveAll(testPath)

	// Test passed, parent directory is writable
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
	swapLimit := "0" // Disable swap by default
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
