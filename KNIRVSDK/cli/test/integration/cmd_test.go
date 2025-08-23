package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionCommand tests the version command
func TestVersionCommand(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build the CLI binary
	tempDir, err := os.MkdirTemp("", "knirvchain-cli-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	binaryPath := filepath.Join(tempDir, "knirv")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	err = buildCmd.Run()
	require.NoError(t, err, "Failed to build CLI binary")

	// Run the version command
	cmd := exec.Command(binaryPath, "version")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	// Check results
	require.NoError(t, err, "Version command failed: %s", stderr.String())
	output := stdout.String()
	assert.Contains(t, output, "KNIRVCHAIN CLI")
	assert.Contains(t, output, "Version")
	assert.Contains(t, output, "Go version")
}

// TestInitCommand tests the init command
func TestInitCommand(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build the CLI binary
	tempDir, err := os.MkdirTemp("", "knirvchain-cli-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	binaryPath := filepath.Join(tempDir, "knirv")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	err = buildCmd.Run()
	require.NoError(t, err, "Failed to build CLI binary")

	// Create a config directory
	configDir := filepath.Join(tempDir, "config")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Run the init command with custom config path
	configPath := filepath.Join(configDir, "config.yaml")
	cmd := exec.Command(binaryPath, "init", "--config", configPath, "--overwrite")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	// Check results
	require.NoError(t, err, "Init command failed: %s", stderr.String())
	output := stdout.String()
	assert.Contains(t, output, "Configuration initialized")

	// Verify config file was created
	_, err = os.Stat(configPath)
	assert.NoError(t, err, "Config file was not created")
}

// TestWalletCommands tests the wallet commands
func TestWalletCommands(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build the CLI binary
	tempDir, err := os.MkdirTemp("", "knirvchain-cli-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	binaryPath := filepath.Join(tempDir, "knirv")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	err = buildCmd.Run()
	require.NoError(t, err, "Failed to build CLI binary")

	// Create a wallet directory
	walletDir := filepath.Join(tempDir, "wallets")
	err = os.MkdirAll(walletDir, 0755)
	require.NoError(t, err)

	// Run the wallet new command
	walletPath := filepath.Join(walletDir, "test-wallet.json")
	cmd := exec.Command(binaryPath, "wallet", "new", "--file", walletPath, "--no-password")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	// Check results
	require.NoError(t, err, "Wallet new command failed: %s", stderr.String())
	output := stdout.String()
	assert.Contains(t, output, "Wallet created successfully")

	// Extract the address from the output
	var address string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "address:") {
			parts := strings.Split(line, "address:")
			if len(parts) > 1 {
				address = strings.TrimSpace(parts[1])
				break
			}
		}
	}
	require.NotEmpty(t, address, "Could not extract wallet address from output")

	// Run the wallet list command
	cmd = exec.Command(binaryPath, "wallet", "list", "--directory", walletDir, "--show-paths")
	stdout.Reset()
	stderr.Reset()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	// Check results
	require.NoError(t, err, "Wallet list command failed: %s", stderr.String())
	output = stdout.String()
	assert.Contains(t, output, address)
	assert.Contains(t, output, walletPath)
}