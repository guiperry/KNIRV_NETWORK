package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI/cmd" // Your existing cmd package
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandCompleter provides command completion for the REPL
type CommandCompleter struct {
	rootCmd *cobra.Command
}

// Do implements the readline.AutoCompleter interface
func (c *CommandCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	lineStr := string(line[:pos])
	args := strings.Fields(lineStr)

	// Handle empty input
	if len(args) == 0 {
		return c.getCompletions(c.rootCmd, "")
	}

	// Special case for exit/quit
	if len(args) == 1 {
		prefix := args[0]
		if strings.HasPrefix("exit", prefix) && "exit" != prefix {
			return [][]rune{[]rune("exit")}, len(prefix)
		}
		if strings.HasPrefix("quit", prefix) && "quit" != prefix {
			return [][]rune{[]rune("quit")}, len(prefix)
		}
		if strings.HasPrefix("clear", prefix) && "clear" != prefix {
			return [][]rune{[]rune("clear")}, len(prefix)
		}
		if strings.HasPrefix("cls", prefix) && "cls" != prefix {
			return [][]rune{[]rune("cls")}, len(prefix)
		}
	}

	// Find the command to complete
	cmd := c.rootCmd
	cmdPath := args[:len(args)-1]
	lastArg := ""
	if len(args) > 0 {
		lastArg = args[len(args)-1]
	}

	// Navigate the command tree
	for _, arg := range cmdPath {
		found := false
		for _, subCmd := range cmd.Commands() {
			if subCmd.Name() == arg || contains(subCmd.Aliases, arg) {
				cmd = subCmd
				found = true
				break
			}
		}
		if !found {
			// If we can't find the command, we can't provide completions
			return [][]rune{}, 0
		}
	}

	return c.getCompletions(cmd, lastArg)
}

// getCompletions returns completion candidates for the given command and prefix
func (c *CommandCompleter) getCompletions(cmd *cobra.Command, prefix string) (newLine [][]rune, length int) {
	var candidates []string

	// Add subcommands
	for _, subCmd := range cmd.Commands() {
		if !subCmd.Hidden && strings.HasPrefix(subCmd.Name(), prefix) {
			candidates = append(candidates, subCmd.Name())
		}
		// Add aliases
		for _, alias := range subCmd.Aliases {
			if strings.HasPrefix(alias, prefix) {
				candidates = append(candidates, alias)
			}
		}
	}

	// Add flags if we're at the beginning of a flag
	if strings.HasPrefix(prefix, "-") {
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if len(prefix) <= 1 { // Just "-"
				candidates = append(candidates, "--"+flag.Name)
			} else if strings.HasPrefix("--"+flag.Name, prefix) {
				candidates = append(candidates, "--"+flag.Name)
			} else if strings.HasPrefix("-"+flag.Name[:1], prefix) {
				candidates = append(candidates, "-"+flag.Name[:1])
			}
		})
	}

	// Convert candidates to the expected return format
	completions := make([][]rune, 0, len(candidates))
	for _, candidate := range candidates {
		completions = append(completions, []rune(candidate))
	}

	return completions, len(prefix)
}

// contains checks if a string is in a slice
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func main() {
	// If arguments are passed (e.g., "knirv wallet list"), execute as a standard CLI command.
	// os.Args[0] is the program name. len(os.Args) > 1 means subcommands/flags were passed.
	if len(os.Args) > 1 {
		cmd.Execute() // This will handle the command and os.Exit if needed.
		return
	}

	// No arguments passed, start interactive REPL mode.
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                KNIRVCHAIN CLI Interactive Mode             ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║ • Type 'help' for a list of available commands             ║")
	fmt.Println("║ • Use <tab> for command completion                         ║")
	fmt.Println("║ • Type 'exit' or 'quit' to leave                           ║")
	fmt.Println("║ • Type 'clear' or 'cls' to clear the screen                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not get user home directory for history file: %v\n", err)
		userHomeDir = "/tmp" // Fallback history location
	}
	historyFile := filepath.Join(userHomeDir, ".knirv_history")

	rootCommand := cmd.GetRootCmd()
	completer := &CommandCompleter{rootCmd: rootCommand}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[1;36mknirv>\033[0m ",  // Cyan bold prompt
		HistoryFile:     historyFile,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    completer,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing readline: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt { // Ctrl+C
			if line == "" { // If buffer was empty, exit on second Ctrl+C
				break
			}
			continue // Clear current line
		} else if err == io.EOF { // Ctrl+D
			break
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			break
		}
		
		// Handle special REPL commands
		if line == "clear" || line == "cls" {
			// Clear screen sequence
			fmt.Print("\033[H\033[2J")
			continue
		}

		args := strings.Fields(line)
		rootCommand.SetArgs(args)

		// Execute the command. ExecuteC returns the executed command and an error.
		if err := rootCommand.Execute(); err != nil {
			// Cobra's Execute() already prints the error to Stderr by default
			// If you set SilenceErrors on rootCmd, you'd print it here:
			// fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}
	fmt.Println("Exiting KNIRVCHAIN CLI.")
}