package integration_tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestKNIRVTestnetIntegration(t *testing.T) {
	// Get absolute path to test-integration.sh
	testnetDir := filepath.Join("..", "KNIRVGATEWAY", "knirvtestnet")
	scriptPath := filepath.Join(testnetDir, "test-integration.sh")

	// Verify script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatalf("test-integration.sh not found at: %s", scriptPath)
	}

	// Run the test script
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = testnetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Errorf("KNIRVTestnet integration tests failed: %v", err)
	}

	// Parse test report
	reportPath := filepath.Join(testnetDir, "test-report.json")
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Logf("Could not read test report: %v", err)
		return
	}

	var report struct {
		TestSuite    string `json:"testSuite"`
		TotalTests   int    `json:"totalTests"`
		PassedTests  int    `json:"passedTests"`
		FailedTests  int    `json:"failedTests"`
		Success      bool   `json:"success"`
	}
	if err := json.Unmarshal(reportData, &report); err != nil {
		t.Logf("Could not parse test report: %v", err)
		return
	}

	t.Logf("KNIRVTestnet test results - Total: %d, Passed: %d, Failed: %d", 
		report.TotalTests, report.PassedTests, report.FailedTests)

	// Compare testnet vs production issues
	// TODO: Implement Gemini/Cerebras API comparison
}
