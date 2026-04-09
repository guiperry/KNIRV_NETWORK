package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/cmd"
	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type CommandCompleter struct {
	rootCmd *cobra.Command
}

func (c *CommandCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	lineStr := string(line[:pos])
	args := strings.Fields(lineStr)

	if len(args) == 0 {
		return c.getCompletions(c.rootCmd, "")
	}

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

	cmd := c.rootCmd
	cmdPath := args[:len(args)-1]
	lastArg := ""
	if len(args) > 0 {
		lastArg = args[len(args)-1]
	}

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
			return [][]rune{}, 0
		}
	}

	return c.getCompletions(cmd, lastArg)
}

func (c *CommandCompleter) getCompletions(cmd *cobra.Command, prefix string) (newLine [][]rune, length int) {
	var candidates []string

	for _, subCmd := range cmd.Commands() {
		if !subCmd.Hidden && strings.HasPrefix(subCmd.Name(), prefix) {
			candidates = append(candidates, subCmd.Name())
		}
		for _, alias := range subCmd.Aliases {
			if strings.HasPrefix(alias, prefix) {
				candidates = append(candidates, alias)
			}
		}
	}

	if strings.HasPrefix(prefix, "-") {
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if len(prefix) <= 1 {
				candidates = append(candidates, "--"+flag.Name)
			} else if strings.HasPrefix("--"+flag.Name, prefix) {
				candidates = append(candidates, "--"+flag.Name)
			} else if strings.HasPrefix("-"+flag.Name[:1], prefix) {
				candidates = append(candidates, "-"+flag.Name[:1])
			}
		})
	}

	completions := make([][]rune, 0, len(candidates))
	for _, candidate := range candidates {
		completions = append(completions, []rune(candidate))
	}

	return completions, len(prefix)
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func main() {
	if len(os.Args) > 1 {
		cmd.Execute()
		return
	}

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                KNIRVSHELL Interactive Mode                  ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║ • Type 'help' for a list of available commands             ║")
	fmt.Println("║ • Use <tab> for command completion                         ║")
	fmt.Println("║ • Type 'exit' or 'quit' to leave                           ║")
	fmt.Println("║ • Type 'clear' or 'cls' to clear the screen                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not get user home directory for history file: %v\n", err)
		userHomeDir = "/tmp"
	}
	historyFile := filepath.Join(userHomeDir, ".knirv_history")

	rootCommand := cmd.GetRootCmd()
	completer := &CommandCompleter{rootCmd: rootCommand}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[1;36mknirv>\033[0m ",
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
		if err == readline.ErrInterrupt {
			if line == "" {
				break
			}
			continue
		} else if err == io.EOF {
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

		if line == "clear" || line == "cls" {
			fmt.Print("\033[H\033[2J")
			continue
		}

		args := strings.Fields(line)
		rootCommand.SetArgs(args)

		if err := rootCommand.Execute(); err != nil {
		}

		rootCommand.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
		rootCommand.SetArgs(nil)

		if viper.IsSet("config") {
			_ = viper.ReadInConfig()
		}
	}
	fmt.Println("Exiting KNIRVSHELL.")
}
