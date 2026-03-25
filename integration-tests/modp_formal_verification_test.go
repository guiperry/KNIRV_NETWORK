package integration_tests

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// resolveModpDir returns the path to the modp directory.
// Priority: MODP_DIR env var > repo-relative default (../modp from integration-tests/).
func resolveModpDir() string {
	if dir := os.Getenv("MODP_DIR"); dir != "" {
		return dir
	}
	// integration-tests/ is one level below the repo root
	_, filename, _, _ := runtime.Caller(0)
	integrationTestsDir := filepath.Dir(filename)
	repoRoot := filepath.Dir(integrationTestsDir)
	return filepath.Join(repoRoot, "modp")
}

// TestKNIRVModPFormalVerification is the top-level entry point.
// It is skipped when:
//   - MODP_ENABLED=false (CI opt-out)
//   - modp directory does not exist
//
// Each P test case maps to a Go subtest so `go test -run TestKNIRVModP/...` works.
func TestKNIRVModPFormalVerification(t *testing.T) {
	if os.Getenv("MODP_ENABLED") == "false" {
		t.Skip("MODP_ENABLED=false - skipping formal verification")
	}

	modpDir := resolveModpDir()
	if _, err := os.Stat(modpDir); os.IsNotExist(err) {
		t.Skipf("modp directory not found at %s (set MODP_DIR to enable)", modpDir)
	}

	runScript := filepath.Join(modpDir, "scripts", "run-tests.sh")
	if _, err := os.Stat(runScript); os.IsNotExist(err) {
		t.Skipf("modp run-tests.sh not found at %s", runScript)
	}

	t.Logf("modp directory: %s", modpDir)

	// ── Subtest: Compile ──────────────────────────────────────────────────────
	t.Run("Compile", func(t *testing.T) {
		runModpSubtest(t, modpDir, runScript, "compile")
	})

	// ── Subtest: NetworkWide ──────────────────────────────────────────────────
	t.Run("NetworkWide", func(t *testing.T) {
		runModpSubtest(t, modpDir, runScript, "network")
	})

	// ── Subtest: ComponentTests ───────────────────────────────────────────────
	t.Run("ComponentTests", func(t *testing.T) {
		runModpSubtest(t, modpDir, runScript, "component")
	})

	// ── Individual named P tests ──────────────────────────────────────────────
	// These mirror the NETWORK_TESTS and COMPONENT_TESTS arrays in modp/scripts/run-tests.sh.
	namedTests := []struct {
		name    string // Go subtest name
		pTest   string // name passed to `run-tests.sh test <name>`
		network bool   // true = network-wide, false = component
	}{
		{"TokenTransferConsistency", "TokenTransferConsistency", false},
		{"GovernanceLifecycle", "GovernanceLifecycle", false},
		{"SkillRegistration", "SkillRegistration", false},
		{"NodeTransformation", "NodeTransformation", false},
		{"KnowledgeGraphConsistency", "KnowledgeGraphConsistency", false},
		{"P2PConnectivity", "P2PConnectivity", false},
		{"ValidationExecution", "ValidationExecution", false},
		{"BaseLayerStorage", "BaseLayerStorage", false},
		{"FullNetworkComposition", "FullNetworkComposition", true},
		{"ErrorToSkillToCrossChain", "ErrorToSkillToCrossChain", true},
		{"ValidationPipeline", "ValidationPipeline", true},
		{"NetworkMaliciousBehaviorRejection", "NetworkMaliciousBehaviorRejection", true},
	}

	for _, tc := range namedTests {
		tc := tc // capture range var
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runModpSubtest(t, modpDir, runScript, "test", tc.pTest)
		})
	}
}

// runModpSubtest executes modp/scripts/run-tests.sh with the given arguments as
// a Go subtest. Output is captured and replayed line-by-line to t.Log so it
// appears in verbose mode. On failure the subtest is marked as failed.
func runModpSubtest(t *testing.T, modpDir, runScript string, args ...string) {
	t.Helper()

	cmd := exec.Command("bash", append([]string{runScript}, args...)...)
	cmd.Dir = modpDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("MODP_DIR=%s", modpDir))

	out, runErr := cmd.CombinedOutput()

	// Always log the output so it's visible under `go test -v`
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		t.Log(scanner.Text())
	}

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			t.Logf("modp command exited with code %d", exitErr.ExitCode())
		}
		// In simulation mode (no P compiler) the script still exits 0,
		// so a non-zero exit always means a real failure.
		t.Errorf("modp test failed: bash %s %s", runScript, strings.Join(args, " "))
	}
}
