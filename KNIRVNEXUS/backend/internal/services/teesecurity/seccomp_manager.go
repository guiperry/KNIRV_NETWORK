package teesecurity

import (
	"fmt"
	"log"
	"os"

	seccomp "github.com/seccomp/libseccomp-golang"
)

// SeccompManager handles seccomp filter creation and application
type SeccompManager struct {
	config SeccompConfig
}

// NewSeccompManager creates a new SeccompManager instance
func NewSeccompManager(config SeccompConfig) *SeccompManager {
	return &SeccompManager{config: config}
}

// ApplyFilter applies the seccomp filter to the current process
func (sm *SeccompManager) ApplyFilter() error {
	// Create filter with default action
	dfltAction := seccomp.ActErrno
	if sm.config.DefaultAction == "SCMP_ACT_ALLOW" {
		dfltAction = seccomp.ActAllow
	}

	filter, err := seccomp.NewFilter(dfltAction)
	if err != nil {
		return fmt.Errorf("failed to create seccomp filter: %w", err)
	}
	defer filter.Release()

	// Add allowed syscalls
	for _, syscallName := range sm.config.AllowedSyscalls {
		syscallID, err := seccomp.GetSyscallFromName(syscallName)
		if err != nil {
			log.Printf("Warning: unknown syscall %s: %v", syscallName, err)
			continue
		}

		if err := filter.AddRule(syscallID, seccomp.ActAllow); err != nil {
			return fmt.Errorf("failed to add rule for %s: %w", syscallName, err)
		}
	}

	// Explicitly block dangerous syscalls
	for _, syscallName := range sm.config.BlockedSyscalls {
		syscallID, err := seccomp.GetSyscallFromName(syscallName)
		if err != nil {
			continue
		}

		if err := filter.AddRule(syscallID, seccomp.ActKill); err != nil {
			return fmt.Errorf("failed to add block rule for %s: %w", syscallName, err)
		}
	}

	// Load filter into kernel
	if err := filter.Load(); err != nil {
		return fmt.Errorf("failed to load seccomp filter: %w", err)
	}

	log.Println("Seccomp filter applied successfully")
	return nil
}

// GetDefaultSeccompProfile returns a secure default seccomp profile
func GetDefaultSeccompProfile() SeccompConfig {
	return SeccompConfig{
		DefaultAction:     "SCMP_ACT_ERRNO",
		AllowedSyscalls:   DefaultAllowedSyscalls,
		BlockedSyscalls:   BlockedSyscalls,
		UseDefaultProfile: true,
	}
}

// CheckSeccompSupport checks if seccomp is supported on the system
func CheckSeccompSupport() bool {
	// Check if seccomp is available by trying to create a simple filter
	filter, err := seccomp.NewFilter(seccomp.ActAllow)
	if err != nil {
		return false
	}
	defer filter.Release()

	// Try to load a trivial filter
	if err := filter.Load(); err != nil {
		return false
	}

	return true
}

// GetCurrentSeccompMode returns the current seccomp mode
func GetCurrentSeccompMode() (string, error) {
	// Check if seccomp is active by reading /proc/self/status
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/self/status: %w", err)
	}

	// Look for Seccomp line
	for _, line := range string(data) {
		if line == 'S' && string(data) == "Seccomp" {
			return "enabled", nil
		}
	}

	return "disabled", nil
}
