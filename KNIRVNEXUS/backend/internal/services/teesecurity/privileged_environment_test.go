package teesecurity

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestPrivilegedEnvironmentSetup verifies that the test environment has all required privileges
// This test should run FIRST and FAIL if privileges are missing in a privileged test environment
func TestPrivilegedEnvironmentSetup(t *testing.T) {
	if !isPrivilegedTestEnvironment() {
		t.Skip("Not in privileged test environment")
		return
	}

	t.Log("========================================")
	t.Log("Privileged Environment Verification")
	t.Log("========================================")

	allPassed := true

	// Test 1: Running as root
	t.Run("RootAccess", func(t *testing.T) {
		uid := os.Geteuid()
		if uid != 0 {
			t.Errorf("FAIL: Not running as root (uid=%d)", uid)
			t.Error("Container must be started with root user")
			allPassed = false
		} else {
			t.Logf("✓ Running as root (uid=0)")
		}
	})

	// Test 2: Cgroup filesystem writable
	t.Run("CgroupWritable", func(t *testing.T) {
		testPath := "/sys/fs/cgroup/knirv-privilege-test"

		if err := os.MkdirAll(testPath, 0755); err != nil {
			t.Errorf("FAIL: Cannot create cgroup directory: %v", err)
			t.Error("Container must be started with:")
			t.Error("  --privileged")
			t.Error("  --cgroupns=host")
			t.Error("  -v /sys/fs/cgroup:/sys/fs/cgroup:rw")
			allPassed = false
		} else {
			t.Logf("✓ Cgroup filesystem is writable")
			os.RemoveAll(testPath)
		}
	})

	// Test 3: Check cgroup mount options
	t.Run("CgroupMountOptions", func(t *testing.T) {
		cmd := exec.Command("mount")
		output, err := cmd.Output()
		if err != nil {
			t.Logf("Warning: Cannot check mount options: %v", err)
			return
		}

		mountInfo := string(output)
		foundCgroup := false

		for _, line := range strings.Split(mountInfo, "\n") {
			if strings.Contains(line, "cgroup") && strings.Contains(line, "/sys/fs/cgroup") {
				foundCgroup = true
				if strings.Contains(line, "(rw,") || strings.Contains(line, "(rw)") {
					t.Logf("✓ Cgroup mounted as read-write: %s", line)
				} else {
					t.Errorf("FAIL: Cgroup mounted as read-only: %s", line)
					t.Error("Container must be started with: -v /sys/fs/cgroup:/sys/fs/cgroup:rw")
					allPassed = false
				}
				break
			}
		}

		if !foundCgroup {
			t.Error("Warning: Could not find cgroup mount in mount table")
		}
	})

	// Test 4: CAP_SYS_ADMIN capability
	t.Run("Capabilities", func(t *testing.T) {
		// Try to create a namespace (requires CAP_SYS_ADMIN)
		cmd := exec.Command("unshare", "--help")
		if err := cmd.Run(); err != nil {
			t.Logf("Warning: unshare command not available: %v", err)
			return
		}

		// Test actual namespace creation
		cmd = exec.Command("unshare", "-p", "-f", "echo", "test")
		if err := cmd.Run(); err != nil {
			t.Errorf("FAIL: Cannot create namespace: %v", err)
			t.Error("Container must be started with: --privileged or --cap-add=SYS_ADMIN")
			allPassed = false
		} else {
			t.Logf("✓ CAP_SYS_ADMIN available (namespace creation works)")
		}
	})

	// Test 5: Network capabilities
	t.Run("NetworkCapabilities", func(t *testing.T) {
		// Try to create a network namespace
		cmd := exec.Command("ip", "netns", "list")
		if err := cmd.Run(); err != nil {
			t.Logf("Warning: ip netns command not available: %v", err)
			return
		}

		t.Logf("✓ Network namespace tools available")
	})

	// Final summary
	if !allPassed {
		t.Error("")
		t.Error("========================================")
		t.Error("PRIVILEGED ENVIRONMENT SETUP FAILED")
		t.Error("========================================")
		t.Error("")
		t.Error("The test container does not have required privileges.")
		t.Error("")
		t.Error("To fix, ensure container is deployed with:")
		t.Error("")
		t.Error("Method 1 - Use container_deployer:")
		t.Error("  cd backend/cmd/container_deployer")
		t.Error("  go run main.go --env development --action 1")
		t.Error("")
		t.Error("Method 2 - Use privileged script:")
		t.Error("  ./scripts/run-docker-privileged.sh --build --test")
		t.Error("")
		t.Error("Method 3 - Manual Docker:")
		t.Error("  docker run --privileged --cgroupns=host \\")
		t.Error("    -v /sys/fs/cgroup:/sys/fs/cgroup:rw \\")
		t.Error("    --cap-add=ALL \\")
		t.Error("    --security-opt apparmor=unconfined \\")
		t.Error("    --security-opt seccomp=unconfined \\")
		t.Error("    knirvnexus-test:latest")
		t.Error("")
		t.Fatal("Cannot proceed with tests without required privileges")
	}

	t.Log("")
	t.Log("========================================")
	t.Log("✓ ALL PRIVILEGE CHECKS PASSED")
	t.Log("========================================")
}
