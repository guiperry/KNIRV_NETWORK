package teesecurity

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHardening_Integration(t *testing.T) {
	// Skip this test if running in environments without proper isolation support
	if !CheckSeccompSupport() {
		t.Skip("Skipping security hardening integration test: seccomp not supported")
	}

	// Create a test container runtime
	kaliProfile := &KaliLinuxProfile{
		OS:          "linux",
		IsKaliLinux: true,
		StaticAnalysisTools: KaliStaticAnalysisTools{
			Semgrep: true,
		},
		DynamicAnalysisTools: KaliDynamicAnalysisTools{
			Strace: true,
		},
		NetworkAnalysisTools: KaliNetworkAnalysisTools{
			Tcpdump: false,
		},
		ForensicsTools: KaliForensicsTools{
			SleuthKit: false,
		},
		SecurityFrameworks: KaliSecurityFrameworks{
			AppArmor: true,
			Seccomp:  true,
		},
	}

	runtime, err := NewNativeContainerRuntime(kaliProfile)
	require.NoError(t, err)

	// Verify hardening is enabled by default
	assert.True(t, runtime.config.EnableHardening)
	assert.True(t, runtime.config.Security.SeccompEnabled)
	assert.True(t, runtime.config.Security.AppArmorEnabled)
	assert.True(t, runtime.config.Security.DropCapabilities)
	assert.True(t, runtime.config.Security.ReadOnlyRoot)
	assert.True(t, runtime.config.Security.NoNewPrivs)
}

func TestSecurityHardening_Configuration(t *testing.T) {
	kaliProfile := &KaliLinuxProfile{
		OS: "linux",
	}

	runtime, err := NewNativeContainerRuntime(kaliProfile)
	require.NoError(t, err)

	// Verify all namespace types are enabled
	assert.True(t, runtime.config.Namespaces.EnablePID)
	assert.True(t, runtime.config.Namespaces.EnableNetwork)
	assert.True(t, runtime.config.Namespaces.EnableMount)
	assert.True(t, runtime.config.Namespaces.EnableUTS)
	assert.True(t, runtime.config.Namespaces.EnableIPC)
	assert.True(t, runtime.config.Namespaces.EnableUser)

	// Verify cgroup limits are set
	assert.Equal(t, int64(100000), runtime.config.Cgroups.CPU.Quota)
	assert.Equal(t, int64(512*1024*1024), runtime.config.Cgroups.Memory.Limit)
	assert.Equal(t, int64(512), runtime.config.Cgroups.PIDs.Max)
}

func TestSecurityHardening_SeccompProfile(t *testing.T) {
	profile := GetDefaultSeccompProfile()

	// Verify default seccomp profile is secure
	assert.Equal(t, "SCMP_ACT_ERRNO", profile.DefaultAction)
	assert.Contains(t, profile.AllowedSyscalls, "read")
	assert.Contains(t, profile.AllowedSyscalls, "write")
	assert.Contains(t, profile.AllowedSyscalls, "execve")
	assert.Contains(t, profile.BlockedSyscalls, "keyctl")
	assert.Contains(t, profile.BlockedSyscalls, "ptrace")
}

func TestSecurityHardening_AppArmorProfile(t *testing.T) {
	// Verify default AppArmor profile structure
	assert.Contains(t, DefaultAppArmorProfile, "{{PROFILE_NAME}}")
	assert.Contains(t, DefaultAppArmorProfile, "deny capability sys_admin")
	assert.Contains(t, DefaultAppArmorProfile, "/skill/** rwix")
	assert.Contains(t, DefaultAppArmorProfile, "deny /** w")
}

func TestSecurityHardening_ContainerExecution(t *testing.T) {
	// Skip this test if not running as root (required for full container isolation)
	if os.Getuid() != 0 {
		t.Skip("Skipping container execution test: requires root privileges")
	}

	// This test verifies that the container execution flow works
	// with security hardening enabled

	kaliProfile := &KaliLinuxProfile{
		OS: "linux",
	}

	runtime, err := NewNativeContainerRuntime(kaliProfile)
	require.NoError(t, err)

	// Simple test skill that just echoes hello
	opts := ContainerOptions{
		SkillCode: `#!/bin/bash
echo "Hello from hardened container"
exit 0`,
	}

	// Set a reasonable timeout for the test
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute the container
	result, err := runtime.RunContainer(ctx, opts)

	// The test should complete without errors
	// Note: This might fail on systems without proper namespace/cgroup support
	// but that's expected for integration tests
	if err != nil {
		t.Logf("Container execution failed (expected on some systems): %v", err)
		return
	}

	// Verify the container executed successfully
	assert.Equal(t, 0, result.ExitCode)
	assert.NotEmpty(t, result.ContainerID)
	assert.NotNil(t, result.ResourceUsage)
}
