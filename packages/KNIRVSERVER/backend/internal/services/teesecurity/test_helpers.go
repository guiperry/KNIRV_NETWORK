package teesecurity

import (
	"os"
	"testing"
)

// RequiresRoot checks if the test requires root privileges and handles accordingly
// In privileged container environments, tests should FAIL if they can't run
// In development unit test mode, tests can skip
func RequiresRoot(t *testing.T, testName string) {
	if os.Geteuid() != 0 {
		// Check if we're in a privileged test environment
		if isPrivilegedTestEnvironment() {
			t.Fatalf("%s requires root privileges, but running as uid=%d. "+
				"Privileged container should have root access. "+
				"Ensure container is running with --privileged flag.", testName, os.Geteuid())
		} else {
			// In local unit test mode, skip gracefully
			t.Skipf("%s requires root privileges (uid=0), running as uid=%d. "+
				"Run in privileged container for full test coverage.", testName, os.Geteuid())
		}
	}
}

// isPrivilegedTestEnvironment detects if we're running in a privileged test container
func isPrivilegedTestEnvironment() bool {
	// Check for KNIRVNEXUS_TEST_MODE environment variable
	if os.Getenv("KNIRVNEXUS_TEST_MODE") == "true" {
		return true
	}

	// Check if /workspace/KNIRVNEXUS exists (our test container marker)
	if _, err := os.Stat("/workspace/KNIRVNEXUS"); err == nil {
		return true
	}

	return false
}

// RequiresCgroupAccess checks if cgroup filesystem is writable
func RequiresCgroupAccess(t *testing.T, testName string) {
	cgroupBase := "/sys/fs/cgroup"
	testPath := cgroupBase + "/knirv-test-access"

	// Try to create a test cgroup
	if err := os.MkdirAll(testPath, 0755); err != nil {
		if isPrivilegedTestEnvironment() {
			t.Fatalf("%s requires cgroup write access, but got error: %v. "+
				"Ensure container is running with: "+
				"--privileged --cgroupns=host -v /sys/fs/cgroup:/sys/fs/cgroup:rw", testName, err)
		} else {
			t.Skipf("%s requires cgroup write access: %v. "+
				"Run in privileged container for full test coverage.", testName, err)
		}
	}

	// Cleanup test cgroup
	os.RemoveAll(testPath)
}

// VerifyPrivilegedEnvironment verifies all required privileges are available
func VerifyPrivilegedEnvironment(t *testing.T) {
	if !isPrivilegedTestEnvironment() {
		t.Skip("Not in privileged test environment, skipping privilege verification")
		return
	}

	// Should be running as root
	if os.Geteuid() != 0 {
		t.Errorf("Expected to be running as root (uid=0), but running as uid=%d", os.Geteuid())
	}

	// Should have cgroup write access
	testPath := "/sys/fs/cgroup/knirv-verify"
	if err := os.MkdirAll(testPath, 0755); err != nil {
		t.Errorf("Expected cgroup write access, but got error: %v", err)
	} else {
		os.RemoveAll(testPath)
	}

	t.Logf("✓ Privileged environment verified: root=%v, cgroup_writable=%v",
		os.Geteuid() == 0, true)
}
