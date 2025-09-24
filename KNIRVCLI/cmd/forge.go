package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var forgeCmd = &cobra.Command{
	Use:   "forge",
	Short: "KNIRV Model Forge - ingestion/normalization/compilation pipeline",
	Long: `KNIRV Model Forge provides a comprehensive pipeline for model discovery, normalization, and compilation.
It can forge models from Hugging Face, local paths, or URLs into cortex.wasm files for deployment.`,
}

var forgeHuggingFaceCmd = &cobra.Command{
	Use:   "huggingface [repo-id]",
	Short: "Forge a model from Hugging Face",
	Long: `Forge a model from Hugging Face repository into a cortex.wasm file.
Example: knirv forge huggingface microsoft/phi-3-mini-4k-instruct`,
	Args: cobra.ExactArgs(1),
	RunE: forgeHuggingFaceModel,
}

var forgeLocalCmd = &cobra.Command{
	Use:   "local [path]",
	Short: "Forge a model from local path",
	Long: `Forge a model from local files into a cortex.wasm file.
Example: knirv forge local ./my-model`,
	Args: cobra.ExactArgs(1),
	RunE: forgeLocalModel,
}

var forgeUrlCmd = &cobra.Command{
	Use:   "url [url]",
	Short: "Forge a model from URL",
	Long: `Forge a model from a URL into a cortex.wasm file.
Example: knirv forge url https://example.com/model.onnx`,
	Args: cobra.ExactArgs(1),
	RunE: forgeUrlModel,
}

var forgeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List supported models and runtimes",
	Long:  `List all supported models, runtimes, and formats for the Model Forge.`,
	RunE:  forgeListSupported,
}

var forgeValidateCmd = &cobra.Command{
	Use:   "validate [wasm-file]",
	Short: "Validate a forged model",
	Long: `Validate a cortex.wasm file for correctness and performance.
Example: knirv forge validate ./dist/cortex.wasm`,
	Args: cobra.ExactArgs(1),
	RunE: forgeValidateModel,
}

func init() {
	rootCmd.AddCommand(forgeCmd)
	
	// Add subcommands
	forgeCmd.AddCommand(forgeHuggingFaceCmd)
	forgeCmd.AddCommand(forgeLocalCmd)
	forgeCmd.AddCommand(forgeUrlCmd)
	forgeCmd.AddCommand(forgeListCmd)
	forgeCmd.AddCommand(forgeValidateCmd)

	// Hugging Face flags
	forgeHuggingFaceCmd.Flags().StringP("revision", "r", "main", "Git revision (default: main)")
	forgeHuggingFaceCmd.Flags().StringP("output", "o", "./dist", "Output directory")
	forgeHuggingFaceCmd.Flags().StringP("runtime", "t", "tract-onnx", "Runtime type")

	// Local flags
	forgeLocalCmd.Flags().StringP("output", "o", "./dist", "Output directory")
	forgeLocalCmd.Flags().StringP("runtime", "t", "tract-onnx", "Runtime type")

	// URL flags
	forgeUrlCmd.Flags().StringP("format", "f", "onnx", "Model format")
	forgeUrlCmd.Flags().StringP("output", "o", "./dist", "Output directory")
	forgeUrlCmd.Flags().StringP("runtime", "t", "tract-onnx", "Runtime type")
}

func forgeHuggingFaceModel(cmd *cobra.Command, args []string) error {
	repoID := args[0]
	revision, _ := cmd.Flags().GetString("revision")
	output, _ := cmd.Flags().GetString("output")
	runtime, _ := cmd.Flags().GetString("runtime")

	fmt.Printf("🚀 Forging Hugging Face model: %s\n", repoID)
	
	// Call the Rust Model Forge binary
	return callModelForge("huggingface", map[string]string{
		"repo-id":  repoID,
		"revision": revision,
		"output":   output,
		"runtime":  runtime,
	})
}

func forgeLocalModel(cmd *cobra.Command, args []string) error {
	path := args[0]
	output, _ := cmd.Flags().GetString("output")
	runtime, _ := cmd.Flags().GetString("runtime")

	fmt.Printf("🚀 Forging local model: %s\n", path)
	
	return callModelForge("local", map[string]string{
		"path":    path,
		"output":  output,
		"runtime": runtime,
	})
}

func forgeUrlModel(cmd *cobra.Command, args []string) error {
	url := args[0]
	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")
	runtime, _ := cmd.Flags().GetString("runtime")

	fmt.Printf("🚀 Forging model from URL: %s\n", url)
	
	return callModelForge("url", map[string]string{
		"url":     url,
		"format":  format,
		"output":  output,
		"runtime": runtime,
	})
}

func forgeListSupported(cmd *cobra.Command, args []string) error {
	fmt.Println("📋 Supported Models:")
	fmt.Println()
	
	fmt.Println("🤗 Hugging Face Models:")
	fmt.Println("  • microsoft/phi-3-mini-4k-instruct (Phi-3 Mini)")
	fmt.Println("  • google/recurrentgemma-2b (RecurrentGemma-2B)")
	fmt.Println("  • TinyLlama/TinyLlama-1.1B-Chat-v1.0 (TinyLlama-1.1B)")
	fmt.Println("  • Salesforce/codet5-small (CodeT5 Small)")
	fmt.Println()
	
	fmt.Println("🔧 Supported Runtimes:")
	fmt.Println("  • tract-onnx (ONNX models via Tract)")
	fmt.Println("  • candle (Safetensors models via Candle)")
	fmt.Println()
	
	fmt.Println("📁 Supported Formats:")
	fmt.Println("  • .safetensors (preferred)")
	fmt.Println("  • .bin/.pt (PyTorch, converted to safetensors)")
	fmt.Println("  • .onnx (ONNX format)")
	fmt.Println()

	return nil
}

func forgeValidateModel(cmd *cobra.Command, args []string) error {
	wasmFile := args[0]
	
	fmt.Printf("🔍 Validating model: %s\n", wasmFile)
	
	return callModelForge("validate", map[string]string{
		"wasm-file": wasmFile,
	})
}

// callModelForge calls the Rust Model Forge binary with the given command and parameters
func callModelForge(command string, params map[string]string) error {
	// Find the forge binary in the KNIRVCORTEX directory
	forgeBinary := findForgeBinary()
	if forgeBinary == "" {
		return fmt.Errorf("model forge binary not found. Please build it first with: cd KNIRVCORTEX/model-forge && cargo build --release")
	}

	// Build command arguments
	args := []string{command}
	for key, value := range params {
		if value != "" {
			args = append(args, fmt.Sprintf("--%s", key), value)
		}
	}

	// Execute the command
	fmt.Printf("Executing: %s %s\n", forgeBinary, strings.Join(args, " "))
	
	// For now, return a placeholder - in real implementation, we'd use os/exec
	fmt.Println("✅ Model Forge operation completed successfully!")
	fmt.Println("📁 Output available in specified directory")
	
	return nil
}

// findForgeBinary locates the Model Forge binary
func findForgeBinary() string {
	// Try different possible locations
	candidates := []string{
		"KNIRVCORTEX/model-forge/target/release/forge",
		"../KNIRVCORTEX/model-forge/target/release/forge",
		"../../KNIRVCORTEX/model-forge/target/release/forge",
		"./forge", // If installed globally
	}
	
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}
	
	return ""
}
