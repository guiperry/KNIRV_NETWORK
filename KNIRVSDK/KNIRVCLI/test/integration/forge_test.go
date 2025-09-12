package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgeCommands(t *testing.T) {
	// Build the CLI binary for testing
	cliPath := buildCLIBinary(t)
	defer os.Remove(cliPath)

	t.Run("ForgeHelp", func(t *testing.T) {
		cmd := exec.Command(cliPath, "forge", "--help")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "KNIRV Model Forge")
		assert.Contains(t, outputStr, "huggingface")
		assert.Contains(t, outputStr, "local")
		assert.Contains(t, outputStr, "url")
		assert.Contains(t, outputStr, "list")
		assert.Contains(t, outputStr, "validate")
	})

	t.Run("ForgeList", func(t *testing.T) {
		cmd := exec.Command(cliPath, "forge", "list")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Supported Models:")
		assert.Contains(t, outputStr, "Hugging Face Models:")
		assert.Contains(t, outputStr, "microsoft/phi-3-mini-4k-instruct")
		assert.Contains(t, outputStr, "Supported Runtimes:")
		assert.Contains(t, outputStr, "tract-onnx")
		assert.Contains(t, outputStr, "candle")
	})

	t.Run("ForgeHuggingFaceHelp", func(t *testing.T) {
		cmd := exec.Command(cliPath, "forge", "huggingface", "--help")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Forge a model from Hugging Face")
		assert.Contains(t, outputStr, "--revision")
		assert.Contains(t, outputStr, "--output")
		assert.Contains(t, outputStr, "--runtime")
	})

	t.Run("ForgeLocalHelp", func(t *testing.T) {
		cmd := exec.Command(cliPath, "forge", "local", "--help")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Forge a model from local path")
		assert.Contains(t, outputStr, "--output")
		assert.Contains(t, outputStr, "--runtime")
	})

	t.Run("ForgeUrlHelp", func(t *testing.T) {
		cmd := exec.Command(cliPath, "forge", "url", "--help")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Forge a model from URL")
		assert.Contains(t, outputStr, "--format")
		assert.Contains(t, outputStr, "--output")
		assert.Contains(t, outputStr, "--runtime")
	})

	t.Run("ForgeValidateHelp", func(t *testing.T) {
		cmd := exec.Command(cliPath, "forge", "validate", "--help")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Validate a forged model")
		assert.Contains(t, outputStr, "cortex.wasm")
	})
}

func TestForgeIntegration(t *testing.T) {
	// Skip if Model Forge binary is not available
	if !isModelForgeBinaryAvailable() {
		t.Skip("Model Forge binary not available, skipping integration tests")
	}

	cliPath := buildCLIBinary(t)
	defer os.Remove(cliPath)

	// Create temporary directory for test outputs
	tempDir, err := os.MkdirTemp("", "forge-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("ForgeHuggingFaceModel", func(t *testing.T) {
		// Test with a small model (this would normally download and compile)
		cmd := exec.Command(cliPath, "forge", "huggingface",
			"TinyLlama/TinyLlama-1.1B-Chat-v1.0",
			"--output", tempDir,
			"--runtime", "tract-onnx")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// The command should execute without error (even if it's a placeholder)
		assert.NoError(t, err)
		assert.Contains(t, outputStr, "Forging Hugging Face model")
		assert.Contains(t, outputStr, "TinyLlama/TinyLlama-1.1B-Chat-v1.0")
	})

	t.Run("ForgeLocalModel", func(t *testing.T) {
		// Create a dummy model file
		modelDir := filepath.Join(tempDir, "dummy-model")
		err := os.MkdirAll(modelDir, 0755)
		require.NoError(t, err)

		dummyFile := filepath.Join(modelDir, "model.onnx")
		err = os.WriteFile(dummyFile, []byte("dummy model content"), 0644)
		require.NoError(t, err)

		cmd := exec.Command(cliPath, "forge", "local",
			modelDir,
			"--output", tempDir,
			"--runtime", "tract-onnx")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		assert.NoError(t, err)
		assert.Contains(t, outputStr, "Forging local model")
		assert.Contains(t, outputStr, modelDir)
	})

	t.Run("ForgeUrlModel", func(t *testing.T) {
		cmd := exec.Command(cliPath, "forge", "url",
			"https://example.com/model.onnx",
			"--format", "onnx",
			"--output", tempDir,
			"--runtime", "tract-onnx")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		assert.NoError(t, err)
		assert.Contains(t, outputStr, "Forging model from URL")
		assert.Contains(t, outputStr, "https://example.com/model.onnx")
	})

	t.Run("ForgeValidateModel", func(t *testing.T) {
		// Create a dummy WASM file
		dummyWasm := filepath.Join(tempDir, "cortex.wasm")
		err := os.WriteFile(dummyWasm, []byte("dummy wasm content"), 0644)
		require.NoError(t, err)

		cmd := exec.Command(cliPath, "forge", "validate", dummyWasm)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		assert.NoError(t, err)
		assert.Contains(t, outputStr, "Validating model")
		assert.Contains(t, outputStr, "cortex.wasm")
	})
}

func TestForgeErrorHandling(t *testing.T) {
	cliPath := buildCLIBinary(t)
	defer os.Remove(cliPath)

	t.Run("ForgeHuggingFaceMissingArgs", func(t *testing.T) {
		cmd := exec.Command(cliPath, "forge", "huggingface")
		output, err := cmd.CombinedOutput()

		// Should fail with missing arguments
		assert.Error(t, err)
		outputStr := string(output)
		assert.Contains(t, outputStr, "requires exactly 1 arg")
	})

	t.Run("ForgeLocalMissingArgs", func(t *testing.T) {
		cmd := exec.Command(cliPath, "forge", "local")
		output, err := cmd.CombinedOutput()

		assert.Error(t, err)
		outputStr := string(output)
		assert.Contains(t, outputStr, "requires exactly 1 arg")
	})

	t.Run("ForgeUrlMissingArgs", func(t *testing.T) {
		cmd := exec.Command(cliPath, "forge", "url")
		output, err := cmd.CombinedOutput()

		assert.Error(t, err)
		outputStr := string(output)
		assert.Contains(t, outputStr, "requires exactly 1 arg")
	})

	t.Run("ForgeValidateMissingArgs", func(t *testing.T) {
		cmd := exec.Command(cliPath, "forge", "validate")
		output, err := cmd.CombinedOutput()

		assert.Error(t, err)
		outputStr := string(output)
		assert.Contains(t, outputStr, "requires exactly 1 arg")
	})
}

// Helper function to build the CLI binary for testing
func buildCLIBinary(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "knirv-cli-test-*")
	require.NoError(t, err)

	binaryPath := filepath.Join(tempDir, "knirv-cli")

	// Build the CLI binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "../../main.go")
	cmd.Dir = filepath.Join("..", "..")
	err = cmd.Run()
	require.NoError(t, err)

	return binaryPath
}

// Helper function to check if Model Forge binary is available
func isModelForgeBinaryAvailable() bool {
	candidates := []string{
		"../../../../KNIRVCORTEX/model-forge/target/release/forge",
		"../../../KNIRVCORTEX/model-forge/target/release/forge",
		"../../KNIRVCORTEX/model-forge/target/release/forge",
		"forge", // If installed globally
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}

	return false
}

func TestProtoBufSync(t *testing.T) {
	t.Run("ProtoBufSyncValidation", func(t *testing.T) {
		// Test that protobuf sync validation works
		cmd := exec.Command("make", "sync-protobuf-validate")
		cmd.Dir = "../../../.." // Go to project root
		output, err := cmd.CombinedOutput()

		require.NoError(t, err)
		outputStr := string(output)
		assert.Contains(t, outputStr, "All ProtoBuf files are valid!")
	})

	t.Run("ProtoBufSyncDryRun", func(t *testing.T) {
		// Test that protobuf sync dry run works
		cmd := exec.Command("make", "sync-protobuf-dry-run")
		cmd.Dir = "../../../.." // Go to project root
		output, err := cmd.CombinedOutput()

		require.NoError(t, err)
		outputStr := string(output)
		assert.Contains(t, outputStr, "Dry run complete - no changes made")
	})
}

func TestProtoBufFiles(t *testing.T) {
	projectRoot := "../../../.."

	t.Run("SharedProtoFilesExist", func(t *testing.T) {
		protoFiles := []string{
			"shared-proto/cortex/v1/cortex.proto",
			"shared-proto/lora/v1/lora.proto",
			"shared-proto/agent/v1/agent.proto",
			"shared-proto/memory/v1/memory.proto",
		}

		for _, protoFile := range protoFiles {
			fullPath := filepath.Join(projectRoot, protoFile)
			_, err := os.Stat(fullPath)
			assert.NoError(t, err, "ProtoBuf file should exist: %s", protoFile)
		}
	})

	t.Run("SyncedProtoFilesExist", func(t *testing.T) {
		syncTargets := map[string][]string{
			"KNIRVCORTEX/shared-types/proto":            {"cortex.proto", "lora.proto", "agent.proto", "memory.proto"},
			"KNIRVENGINE/desktop-client/proto":          {"cortex.proto", "lora.proto", "agent.proto", "memory.proto"},
			"KNIRVCONTROLLER/src/core/protobuf/schemas": {"cortex.proto", "lora.proto", "agent.proto", "memory.proto"},
			"KNIRVANA/gaming/cortex-compiler/proto":     {"cortex.proto", "lora.proto", "agent.proto", "memory.proto"},
		}

		for targetDir, files := range syncTargets {
			for _, file := range files {
				fullPath := filepath.Join(projectRoot, targetDir, file)
				_, err := os.Stat(fullPath)
				assert.NoError(t, err, "Synced ProtoBuf file should exist: %s/%s", targetDir, file)
			}
		}
	})
}
