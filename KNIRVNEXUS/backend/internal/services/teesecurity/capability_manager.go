package teesecurity

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CapabilityManager handles Linux capability management
type CapabilityManager struct {
	config CapabilityConfig
}

// NewCapabilityManager creates a new CapabilityManager instance
func NewCapabilityManager(config CapabilityConfig) *CapabilityManager {
	return &CapabilityManager{config: config}
}

// DropCapabilities drops all capabilities except those in Keep list
func (cm *CapabilityManager) DropCapabilities() error {
	// Use setcap command to drop capabilities
	// This is a simplified version that uses external commands

	// First, drop all capabilities
	if err := cm.runCommand("setcap", "-r", "0"); err != nil {
		return fmt.Errorf("failed to drop all capabilities: %w", err)
	}

	// Then add back the ones we want to keep
	for _, capName := range cm.config.Keep {
		// Validate capability name
		if _, err := parseCapability(capName); err != nil {
			return fmt.Errorf("invalid capability %s: %w", capName, err)
		}

		// Set capability using setcap
		if err := cm.runCommand("setcap", fmt.Sprintf("cap_%s+ep", strings.ToLower(capName)), "0"); err != nil {
			return fmt.Errorf("failed to set capability %s: %w", capName, err)
		}
	}

	return nil
}

// SetNoNewPrivs prevents gaining new privileges
func (cm *CapabilityManager) SetNoNewPrivs() error {
	// Use prctl command to set no_new_privs
	if err := cm.runCommand("prctl", "--no-new-privs", "1"); err != nil {
		return fmt.Errorf("failed to set no_new_privs: %w", err)
	}

	return nil
}

// parseCapability converts capability name to capability number
func parseCapability(name string) (int, error) {
	// Map capability name to capability number
	capMap := map[string]int{
		"CAP_SETUID":  1,
		"CAP_SETGID":  2,
		"CAP_KILL":    5,
		"CAP_NET_RAW": 13,
		// Add more as needed
	}

	cap, ok := capMap[name]
	if !ok {
		return 0, fmt.Errorf("unknown capability: %s", name)
	}

	return cap, nil
}

// DropAllCapabilities drops all capabilities except basic ones
func (cm *CapabilityManager) DropAllCapabilities() error {
	// Get the list of all capabilities
	allCaps := []string{
		"CAP_SYS_ADMIN", "CAP_SYS_MODULE", "CAP_SYS_RAWIO", "CAP_SYS_PTRACE",
		"CAP_SYS_BOOT", "CAP_NET_ADMIN", "CAP_MAC_ADMIN", "CAP_SYS_NICE",
		"CAP_SYS_RESOURCE", "CAP_SYS_TIME", "CAP_SYS_TTY_CONFIG", "CAP_AUDIT_CONTROL",
		"CAP_AUDIT_READ", "CAP_AUDIT_WRITE", "CAP_BLOCK_SUSPEND", "CAP_DAC_OVERRIDE",
		"CAP_DAC_READ_SEARCH", "CAP_FOWNER", "CAP_FSETID", "CAP_IPC_LOCK",
		"CAP_IPC_OWNER", "CAP_LEASE", "CAP_LINUX_IMMUTABLE", "CAP_NET_ADMIN",
		"CAP_NET_BIND_SERVICE", "CAP_NET_BROADCAST", "CAP_NET_RAW", "CAP_SETGID",
		"CAP_SETFCAP", "CAP_SETPCAP", "CAP_SETUID", "CAP_SYSLOG",
		"CAP_WAKE_ALARM", "CAP_CHOWN", "CAP_MKNOD", "CAP_NET_ADMIN",
	}

	// Drop all capabilities
	for _, capName := range allCaps {
		if err := cm.runCommand("setcap", "-r", "0"); err != nil {
			// Log but don't fail - some capabilities might not be available
			fmt.Fprintf(os.Stderr, "Warning: failed to drop capability %s: %v\n", capName, err)
		}
	}

	// Keep only the capabilities specified in config.Keep
	for _, capName := range cm.config.Keep {
		if err := cm.runCommand("setcap", fmt.Sprintf("cap_%s+ep", strings.ToLower(capName)), "0"); err != nil {
			return fmt.Errorf("failed to set capability %s: %w", capName, err)
		}
	}

	return nil
}

// runCommand executes a command with given arguments
func (cm *CapabilityManager) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %s: %w", name, strings.Join(args, " "), string(output), err)
	}
	return nil
}
