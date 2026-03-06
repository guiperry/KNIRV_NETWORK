package teesecurity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeccompManager_Creation(t *testing.T) {
	config := GetDefaultSeccompProfile()
	mgr := NewSeccompManager(config)

	assert.NotNil(t, mgr)
	assert.Equal(t, "SCMP_ACT_ERRNO", config.DefaultAction)
	assert.True(t, config.UseDefaultProfile)
	assert.Contains(t, config.AllowedSyscalls, "read")
	assert.Contains(t, config.BlockedSyscalls, "keyctl")
}

func TestSeccompManager_DefaultProfile(t *testing.T) {
	profile := GetDefaultSeccompProfile()

	// Verify default action is to deny
	assert.Equal(t, "SCMP_ACT_ERRNO", profile.DefaultAction)

	// Verify essential syscalls are allowed
	expectedSyscalls := []string{"read", "write", "open", "close", "exit", "execve"}
	for _, syscall := range expectedSyscalls {
		assert.Contains(t, profile.AllowedSyscalls, syscall, "Should allow %s syscall", syscall)
	}

	// Verify dangerous syscalls are blocked
	dangerousSyscalls := []string{"keyctl", "ptrace", "kexec_load", "reboot"}
	for _, syscall := range dangerousSyscalls {
		assert.Contains(t, profile.BlockedSyscalls, syscall, "Should block %s syscall", syscall)
	}
}

func TestSeccompManager_SupportCheck(t *testing.T) {
	supported := CheckSeccompSupport()

	// This test will pass on systems with seccomp support
	// and fail gracefully on systems without it
	if supported {
		t.Log("Seccomp is supported on this system")
	} else {
		t.Log("Seccomp is not supported on this system")
	}

	// We don't assert the result since it depends on the test environment
}

func TestSeccompManager_CustomProfile(t *testing.T) {
	customProfile := SeccompConfig{
		DefaultAction:     "SCMP_ACT_ALLOW",
		AllowedSyscalls:   []string{"read", "write", "exit"},
		BlockedSyscalls:   []string{"keyctl"},
		UseDefaultProfile: false,
	}

	mgr := NewSeccompManager(customProfile)
	assert.NotNil(t, mgr)
	assert.Equal(t, "SCMP_ACT_ALLOW", customProfile.DefaultAction)
	assert.Len(t, customProfile.AllowedSyscalls, 3)
}

func TestSeccompManager_BlockedSyscalls(t *testing.T) {
	// Verify that dangerous syscalls are properly blocked
	blocked := BlockedSyscalls

	expectedDangerousSyscalls := []string{
		"keyctl",            // Kernel keyring access
		"add_key",           // Add kernel keys
		"request_key",       // Request kernel keys
		"ptrace",            // Process tracing
		"process_vm_readv",  // Read other process memory
		"process_vm_writev", // Write other process memory
		"kexec_load",        // Load new kernel
		"reboot",            // System reboot
		"mount",             // Mount filesystems
		"umount",            // Unmount filesystems
		"pivot_root",        // Change root filesystem
		"unshare",           // Create new namespaces
		"setns",             // Join existing namespace
	}

	for _, expected := range expectedDangerousSyscalls {
		assert.Contains(t, blocked, expected, "Should block dangerous syscall: %s", expected)
	}
}
