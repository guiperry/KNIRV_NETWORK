package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var graphchainCmd = &cobra.Command{
	Use:   "graphchain",
	Short: "GraphChain CLI integration",
	Long: `GraphChain CLI integration provides access to the KNIRV GraphChain functionality.
This command opens a new terminal session with the GraphChain CLI tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Open interactive GraphChain CLI terminal
			openGraphChainTerminal()
		} else {
			// Execute GraphChain CLI command directly
			executeGraphChainCommand(args)
		}
	},
}

var graphchainTerminalCmd = &cobra.Command{
	Use:   "terminal",
	Short: "Open GraphChain CLI in a new terminal",
	Long:  `Opens the GraphChain CLI in a new terminal window for interactive use.`,
	Run: func(cmd *cobra.Command, args []string) {
		openGraphChainTerminal()
	},
}

var graphchainExecCmd = &cobra.Command{
	Use:   "exec [command]",
	Short: "Execute GraphChain CLI command directly",
	Long:  `Execute a GraphChain CLI command directly without opening a new terminal.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		executeGraphChainCommand(args)
	},
}

var graphchainStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show GraphChain CLI status and version",
	Long:  `Display the status and version information of the GraphChain CLI binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		showGraphChainStatus()
	},
}

func init() {
	// Add subcommands
	graphchainCmd.AddCommand(graphchainTerminalCmd)
	graphchainCmd.AddCommand(graphchainExecCmd)
	graphchainCmd.AddCommand(graphchainStatusCmd)
	
	// Add to root command
	rootCmd.AddCommand(graphchainCmd)
}

func getGraphChainBinaryPath() string {
	// Get the directory where the KNIRVCLI binary is located
	execPath, err := os.Executable()
	if err != nil {
		log.Warnf("Could not determine executable path: %v", err)
		return "graphchain-cli" // Fallback to PATH lookup
	}
	
	execDir := filepath.Dir(execPath)
	binaryPath := filepath.Join(execDir, "bin", "graphchain-cli")
	
	// Check if the binary exists
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath
	}
	
	// Fallback: check in the same directory as KNIRVCLI
	binaryPath = filepath.Join(execDir, "graphchain-cli")
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath
	}
	
	// Final fallback: assume it's in PATH
	log.Warnf("GraphChain CLI binary not found in expected locations, using PATH lookup")
	return "graphchain-cli"
}

func openGraphChainTerminal() {
	binaryPath := getGraphChainBinaryPath()
	
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "windows":
		// Windows: open in new Command Prompt window
		cmd = exec.Command("cmd", "/c", "start", "cmd", "/k", binaryPath)
	case "darwin":
		// macOS: open in new Terminal window
		script := fmt.Sprintf(`tell application "Terminal" to do script "%s"`, binaryPath)
		cmd = exec.Command("osascript", "-e", script)
	case "linux":
		// Linux: try different terminal emulators
		terminals := []string{"gnome-terminal", "konsole", "xterm", "alacritty", "kitty", "terminator"}
		
		var terminalCmd string
		var terminalArgs []string
		
		for _, terminal := range terminals {
			if _, err := exec.LookPath(terminal); err == nil {
				terminalCmd = terminal
				switch terminal {
				case "gnome-terminal":
					terminalArgs = []string{"--", binaryPath}
				case "konsole":
					terminalArgs = []string{"-e", binaryPath}
				case "xterm":
					terminalArgs = []string{"-e", binaryPath}
				case "alacritty":
					terminalArgs = []string{"-e", binaryPath}
				case "kitty":
					terminalArgs = []string{binaryPath}
				case "terminator":
					terminalArgs = []string{"-e", binaryPath}
				}
				break
			}
		}
		
		if terminalCmd == "" {
			fmt.Println("No supported terminal emulator found. Please install one of: gnome-terminal, konsole, xterm, alacritty, kitty, or terminator")
			fmt.Printf("Alternatively, run the GraphChain CLI directly: %s\n", binaryPath)
			return
		}
		
		cmd = exec.Command(terminalCmd, terminalArgs...)
	default:
		fmt.Printf("Unsupported operating system: %s\n", runtime.GOOS)
		fmt.Printf("Please run the GraphChain CLI directly: %s\n", binaryPath)
		return
	}
	
	fmt.Printf("Opening GraphChain CLI in new terminal...\n")
	fmt.Printf("Binary path: %s\n", binaryPath)
	
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error opening GraphChain CLI terminal: %v\n", err)
		fmt.Printf("Try running the GraphChain CLI directly: %s\n", binaryPath)
		return
	}
	
	fmt.Println("GraphChain CLI terminal opened successfully!")
}

func executeGraphChainCommand(args []string) {
	binaryPath := getGraphChainBinaryPath()
	
	fmt.Printf("Executing GraphChain CLI: %s %v\n", binaryPath, args)
	
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing GraphChain CLI command: %v\n", err)
	}
}

func showGraphChainStatus() {
	binaryPath := getGraphChainBinaryPath()
	
	fmt.Println("GraphChain CLI Status")
	fmt.Println("====================")
	fmt.Printf("Binary Path: %s\n", binaryPath)
	
	// Check if binary exists and is executable
	if info, err := os.Stat(binaryPath); err != nil {
		fmt.Printf("Status: ❌ Not found (%v)\n", err)
		fmt.Println("\nTo install the GraphChain CLI binary:")
		fmt.Println("1. Run 'make sync-binaries' from the KNIRV_NETWORK root")
		fmt.Println("2. Or build manually: cd KNIRVGRAPH && go build -o ../KNIRVCLI/bin/graphchain-cli ./cmd/cli/main.go")
		return
	} else {
		fmt.Printf("Status: ✅ Found (size: %d bytes)\n", info.Size())
		fmt.Printf("Permissions: %s\n", info.Mode())
		fmt.Printf("Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	}
	
	// Try to get version information
	fmt.Println("\nVersion Information:")
	cmd := exec.Command(binaryPath, "--version")
	if output, err := cmd.Output(); err != nil {
		fmt.Printf("Version: ❌ Could not retrieve (%v)\n", err)
	} else {
		fmt.Printf("Version: %s", string(output))
	}
	
	// Try to get help information
	fmt.Println("\nAvailable Commands:")
	cmd = exec.Command(binaryPath, "--help")
	if output, err := cmd.Output(); err != nil {
		fmt.Printf("Help: ❌ Could not retrieve (%v)\n", err)
	} else {
		fmt.Printf("%s", string(output))
	}
}
