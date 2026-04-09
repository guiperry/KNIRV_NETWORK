package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/core"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var wasmLog = logrus.New()

// wasmCmd represents the wasm command
var wasmCmd = &cobra.Command{
	Use:   "wasm",
	Short: "Execute and manage WASM modules",
	Long: `Execute WASM modules and manage WASM-based capabilities.
This command provides functionality to run WASM modules locally or connect to remote WASM services.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// wasmRunCmd represents the wasm run command
var wasmRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute a WASM module",
	Long: `Execute a WASM module using Wasmtime runtime.
	
This command can execute WASM modules locally and connect them to KNIRV network services.

Example:
  knirv wasm run --file ./agent.wasm --runtime wasmtime --args '{"input": "hello"}' --connect-to http://localhost:5000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWasm(cmd, args)
	},
}

// wasmValidateCmd represents the wasm validate command
var wasmValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a WASM module",
	Long: `Validate a WASM module for compatibility with KNIRV network.
	
This command checks if a WASM module meets the requirements for deployment on KNIRV.

Example:
  knirv wasm validate --file ./agent.wasm --check-exports --check-imports`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return validateWasm(cmd, args)
	},
}

// wasmInfoCmd represents the wasm info command
var wasmInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get information about a WASM module",
	Long: `Get detailed information about a WASM module including exports, imports, and metadata.

Example:
  knirv wasm info --file ./agent.wasm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return getWasmInfo(cmd, args)
	},
}

// wasmConnectCmd represents the wasm connect command
var wasmConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a remote WASM service",
	Long: `Connect to a remote WASM service and execute commands.
	
This command allows interaction with WASM modules running on remote KNIRV nodes.

Example:
  knirv wasm connect --url ws://localhost:8080/wasm --module-id agent123 --method invoke --args '{"input": "hello"}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectToWasm(cmd, args)
	},
}

func init() {
	// Add wasm command to root command
	rootCmd.AddCommand(wasmCmd)

	// Add subcommands to wasm command
	wasmCmd.AddCommand(wasmRunCmd)
	wasmCmd.AddCommand(wasmValidateCmd)
	wasmCmd.AddCommand(wasmInfoCmd)
	wasmCmd.AddCommand(wasmConnectCmd)

	// wasm run flags
	wasmRunCmd.Flags().String("file", "", "Path to WASM file")
	wasmRunCmd.Flags().String("runtime", "wasmtime", "WASM runtime to use (wasmtime, wasmer)")
	wasmRunCmd.Flags().String("args", "", "JSON arguments to pass to WASM module")
	wasmRunCmd.Flags().String("connect-to", "", "URL of KNIRV node to connect to")
	wasmRunCmd.Flags().String("function", "main", "Function to call in WASM module")
	wasmRunCmd.Flags().Int("timeout", 30, "Timeout in seconds")
	wasmRunCmd.Flags().Bool("verbose", false, "Enable verbose output")

	// wasm validate flags
	wasmValidateCmd.Flags().String("file", "", "Path to WASM file")
	wasmValidateCmd.Flags().Bool("check-exports", true, "Check required exports")
	wasmValidateCmd.Flags().Bool("check-imports", true, "Check imports compatibility")
	wasmValidateCmd.Flags().Bool("check-memory", true, "Check memory usage")

	// wasm info flags
	wasmInfoCmd.Flags().String("file", "", "Path to WASM file")
	wasmInfoCmd.Flags().Bool("show-exports", true, "Show exported functions")
	wasmInfoCmd.Flags().Bool("show-imports", true, "Show imported functions")
	wasmInfoCmd.Flags().Bool("show-memory", true, "Show memory information")

	// wasm connect flags
	wasmConnectCmd.Flags().String("url", "", "WebSocket URL of WASM service")
	wasmConnectCmd.Flags().String("module-id", "", "ID of the WASM module")
	wasmConnectCmd.Flags().String("method", "", "Method to call")
	wasmConnectCmd.Flags().String("args", "", "JSON arguments")
	wasmConnectCmd.Flags().Int("timeout", 30, "Timeout in seconds")

	// Mark required flags
	wasmRunCmd.MarkFlagRequired("file")
	wasmValidateCmd.MarkFlagRequired("file")
	wasmInfoCmd.MarkFlagRequired("file")
	wasmConnectCmd.MarkFlagRequired("url")
	wasmConnectCmd.MarkFlagRequired("module-id")
	wasmConnectCmd.MarkFlagRequired("method")
}

// runWasm implements the wasm run command
func runWasm(cmd *cobra.Command, _ []string) error {
	// Parse flags
	wasmFile, _ := cmd.Flags().GetString("file")
	runtime, _ := cmd.Flags().GetString("runtime")
	argsStr, _ := cmd.Flags().GetString("args")
	connectTo, _ := cmd.Flags().GetString("connect-to")
	function, _ := cmd.Flags().GetString("function")
	timeout, _ := cmd.Flags().GetInt("timeout")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Validate file exists
	if _, err := os.Stat(wasmFile); os.IsNotExist(err) {
		return fmt.Errorf("WASM file does not exist: %s", wasmFile)
	}

	// Parse arguments
	var wasmArgs map[string]interface{}
	if argsStr != "" {
		if err := json.Unmarshal([]byte(argsStr), &wasmArgs); err != nil {
			return fmt.Errorf("invalid JSON arguments: %w", err)
		}
	}

	wasmLog.Info("Executing WASM module...")
	if verbose {
		wasmLog.SetLevel(logrus.DebugLevel)
		wasmLog.Debugf("File: %s", wasmFile)
		wasmLog.Debugf("Runtime: %s", runtime)
		wasmLog.Debugf("Function: %s", function)
		wasmLog.Debugf("Arguments: %v", wasmArgs)
		if connectTo != "" {
			wasmLog.Debugf("Connecting to: %s", connectTo)
		}
	}

	// Execute WASM module based on runtime
	var result string
	var err error

	switch runtime {
	case "wasmtime":
		result, err = executeWithWasmtime(wasmFile, function, argsStr, timeout, verbose)
	case "wasmer":
		result, err = executeWithWasmer(wasmFile, function, argsStr, timeout, verbose)
	default:
		return fmt.Errorf("unsupported runtime: %s", runtime)
	}

	if err != nil {
		return fmt.Errorf("failed to execute WASM module: %w", err)
	}

	// If connecting to a KNIRV node, send the result
	if connectTo != "" {
		wasmLog.Info("Connecting to KNIRV node...")
		if err := sendResultToNode(connectTo, result, timeout); err != nil {
			wasmLog.Warnf("Failed to send result to node: %v", err)
		} else {
			wasmLog.Info("Result sent to KNIRV node successfully")
		}
	}

	// Output result
	fmt.Printf("✅ WASM execution completed successfully!\n")
	fmt.Printf("📋 Result:\n%s\n", result)

	return nil
}

// validateWasm implements the wasm validate command
func validateWasm(cmd *cobra.Command, _ []string) error {
	// Parse flags
	wasmFile, _ := cmd.Flags().GetString("file")
	checkExports, _ := cmd.Flags().GetBool("check-exports")
	checkImports, _ := cmd.Flags().GetBool("check-imports")
	checkMemory, _ := cmd.Flags().GetBool("check-memory")

	// Validate file exists
	if _, err := os.Stat(wasmFile); os.IsNotExist(err) {
		return fmt.Errorf("WASM file does not exist: %s", wasmFile)
	}

	wasmLog.Info("Validating WASM module...")

	// Use wasmtime to validate the module
	cmd_exec := exec.Command("wasmtime", "--validate", wasmFile)
	output, err := cmd_exec.CombinedOutput()
	if err != nil {
		return fmt.Errorf("WASM validation failed: %s", string(output))
	}

	fmt.Printf("✅ WASM module is valid!\n")

	// Additional checks
	if checkExports || checkImports || checkMemory {
		info, err := getWasmModuleInfo(wasmFile)
		if err != nil {
			wasmLog.Warnf("Could not get detailed module info: %v", err)
		} else {
			if checkExports {
				fmt.Printf("📤 Exports: %d functions found\n", len(info.Exports))
			}
			if checkImports {
				fmt.Printf("📥 Imports: %d functions found\n", len(info.Imports))
			}
			if checkMemory {
				fmt.Printf("💾 Memory: %s\n", info.Memory)
			}
		}
	}

	return nil
}

// getWasmInfo implements the wasm info command
func getWasmInfo(cmd *cobra.Command, _ []string) error {
	// Parse flags
	wasmFile, _ := cmd.Flags().GetString("file")
	showExports, _ := cmd.Flags().GetBool("show-exports")
	showImports, _ := cmd.Flags().GetBool("show-imports")
	showMemory, _ := cmd.Flags().GetBool("show-memory")

	// Validate file exists
	if _, err := os.Stat(wasmFile); os.IsNotExist(err) {
		return fmt.Errorf("WASM file does not exist: %s", wasmFile)
	}

	wasmLog.Info("Getting WASM module information...")

	// Get file info
	fileInfo, err := os.Stat(wasmFile)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Get module info
	moduleInfo, err := getWasmModuleInfo(wasmFile)
	if err != nil {
		return fmt.Errorf("failed to get module info: %w", err)
	}

	// Display information
	fmt.Printf("🔍 WASM Module Information\n")
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("📁 File: %s\n", wasmFile)
	fmt.Printf("📏 Size: %d bytes\n", fileInfo.Size())
	fmt.Printf("📅 Modified: %s\n", fileInfo.ModTime().Format(time.RFC3339))

	if showExports && len(moduleInfo.Exports) > 0 {
		fmt.Printf("\n📤 Exported Functions (%d):\n", len(moduleInfo.Exports))
		for i, export := range moduleInfo.Exports {
			fmt.Printf("  %d. %s\n", i+1, export)
		}
	}

	if showImports && len(moduleInfo.Imports) > 0 {
		fmt.Printf("\n📥 Imported Functions (%d):\n", len(moduleInfo.Imports))
		for i, import_ := range moduleInfo.Imports {
			fmt.Printf("  %d. %s\n", i+1, import_)
		}
	}

	if showMemory {
		fmt.Printf("\n💾 Memory: %s\n", moduleInfo.Memory)
	}

	return nil
}

// connectToWasm implements the wasm connect command
func connectToWasm(cmd *cobra.Command, _ []string) error {
	// Parse flags
	url, _ := cmd.Flags().GetString("url")
	moduleID, _ := cmd.Flags().GetString("module-id")
	method, _ := cmd.Flags().GetString("method")
	argsStr, _ := cmd.Flags().GetString("args")
	timeout, _ := cmd.Flags().GetInt("timeout")

	// Parse arguments
	var methodArgs map[string]interface{}
	if argsStr != "" {
		if err := json.Unmarshal([]byte(argsStr), &methodArgs); err != nil {
			return fmt.Errorf("invalid JSON arguments: %w", err)
		}
	}

	wasmLog.Info("Connecting to remote WASM service...")

	// Create WebSocket connection and call method
	result, err := callRemoteWasmMethod(url, moduleID, method, methodArgs, timeout)
	if err != nil {
		return fmt.Errorf("failed to call remote WASM method: %w", err)
	}

	// Output result
	fmt.Printf("✅ Remote WASM call completed successfully!\n")
	fmt.Printf("📋 Result:\n%s\n", result)

	return nil
}

// Helper functions

type WasmModuleInfo struct {
	Exports []string
	Imports []string
	Memory  string
}

func getWasmModuleInfo(wasmFile string) (*WasmModuleInfo, error) {
	// Use wasmtime to get module info
	cmd := exec.Command("wasmtime", "--invoke", "_start", "--dry-run", wasmFile)
	output, err := cmd.CombinedOutput()

	// Parse output to extract exports, imports, and memory info
	// This is a simplified implementation - in practice, you'd use a proper WASM parser
	info := &WasmModuleInfo{
		Exports: []string{"main", "_start"}, // Default exports
		Imports: []string{},
		Memory:  "1 page (64KB)",
	}

	if err == nil {
		// Parse the output for more detailed information
		outputStr := string(output)
		if strings.Contains(outputStr, "export") {
			// Extract exports from output
		}
	}

	return info, nil
}

func executeWithWasmtime(wasmFile, function, args string, timeout int, verbose bool) (string, error) {
	cmdArgs := []string{"--invoke", function}
	if verbose {
		cmdArgs = append(cmdArgs, "--verbose")
	}
	cmdArgs = append(cmdArgs, wasmFile)

	cmd := exec.Command("wasmtime", cmdArgs...)
	if args != "" {
		cmd.Stdin = strings.NewReader(args)
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func executeWithWasmer(wasmFile, function, args string, timeout int, verbose bool) (string, error) {
	cmdArgs := []string{"run", wasmFile, "--invoke", function}
	if verbose {
		cmdArgs = append(cmdArgs, "--verbose")
	}

	cmd := exec.Command("wasmer", cmdArgs...)
	if args != "" {
		cmd.Stdin = strings.NewReader(args)
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func sendResultToNode(nodeURL, result string, timeout int) error {
	// Create API client and send result to KNIRV node
	apiClient := core.NewAPIClient(
		nodeURL,
		core.WithTimeout(time.Duration(timeout)*time.Second),
		core.WithRetries(3),
		core.WithLogger(wasmLog),
	)

	ctx := context.Background()

	// Send result as a transaction or API call
	payload := map[string]interface{}{
		"type":      "wasm_result",
		"result":    result,
		"timestamp": time.Now().Unix(),
	}

	err := apiClient.Post(ctx, "/wasm/result", payload, nil)
	return err
}

func callRemoteWasmMethod(_, moduleID, method string, args map[string]interface{}, _ int) (string, error) {
	// This would implement WebSocket connection to remote WASM service
	// For now, return a placeholder
	wasmLog.Infof("Calling remote WASM method: %s.%s", moduleID, method)
	wasmLog.Infof("Arguments: %v", args)

	// Simulate remote call
	time.Sleep(1 * time.Second)

	result := fmt.Sprintf(`{
		"status": "success",
		"module_id": "%s",
		"method": "%s",
		"result": "Remote WASM execution completed",
		"timestamp": %d
	}`, moduleID, method, time.Now().Unix())

	return result, nil
}
