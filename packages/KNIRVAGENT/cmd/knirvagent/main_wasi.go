//go:build wasip1 && wasm

// KnirvAgent WASI entrypoint.

package main

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/knirvcorp/knirvagent/pkg/agent"
	"github.com/knirvcorp/knirvagent/pkg/bus"
	"github.com/knirvcorp/knirvagent/pkg/config"
	"github.com/knirvcorp/knirvagent/pkg/providers"
	"github.com/knirvcorp/knirvagent/pkg/relay"
)

//go:generate cp -r ../../workspace .
//go:embed workspace
var embeddedFiles embed.FS

var (
	version   = "dev"
	gitCommit string
	buildTime string
	goVersion string
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "agent":
		agentCmd()
	case "onboard":
		onboard()
	case "version", "--version", "-v":
		printVersion()
	default:
		fmt.Fprintf(os.Stderr, "Unsupported WASI command: %s\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf("knirvagent - WASI portable agent v%s\n\n", version)
	fmt.Println("Usage: knirvagent <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  onboard     Initialize knirvagent configuration and workspace")
	fmt.Println("  agent       Run the agent in direct or simple interactive mode")
	fmt.Println("  version     Show version information")
	fmt.Println()
	fmt.Println("Native-only commands such as server, gateway, and inner PTY sessions are not available in WASI.")
}

func printVersion() {
	fmt.Printf("knirvagent %s\n", formatVersion())
	if buildTime != "" {
		fmt.Printf("  Build: %s\n", buildTime)
	}
	goVer := goVersion
	if goVer == "" {
		goVer = runtime.Version()
	}
	fmt.Printf("  Go: %s\n", goVer)
	fmt.Printf("  Target: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func formatVersion() string {
	v := version
	if gitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", gitCommit)
	}
	return v
}

func onboard() {
	configPath := getConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at %s\n", configPath)
		return
	}

	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	if err := copyEmbeddedToTarget(cfg.WorkspacePath()); err != nil {
		fmt.Fprintf(os.Stderr, "Error copying workspace templates: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("knirvagent is ready")
	fmt.Println("Config:", configPath)
}

func agentCmd() {
	message := ""
	sessionKey := "cli:default"

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-m", "--message":
			if i+1 < len(args) {
				message = args[i+1]
				i++
			}
		case "-s", "--session":
			if i+1 < len(args) {
				sessionKey = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: knirvagent agent [-m message] [-s session]")
			return
		default:
			if message == "" {
				message = args[i]
			}
		}
	}

	relayCfg := relay.FromEnv()
	if relayCfg.Ready() {
		if message != "" {
			response, err := relayCfg.Execute(context.Background(), message)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Relay error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(response)
			return
		}
		relayInteractiveMode(relayCfg)
		return
	}
	if relayCfg.Enabled {
		fmt.Fprintln(os.Stderr, "Relay mode requires KNIRVAGENT_GATEWAY_URL and KNIRVAGENT_DVE_ID")
		os.Exit(1)
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating provider: %v\n", err)
		os.Exit(1)
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	if message != "" {
		response, err := agentLoop.ProcessDirect(context.Background(), message, sessionKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(response)
		return
	}

	simpleInteractiveMode(agentLoop, sessionKey)
}

func relayInteractiveMode(relayCfg relay.Config) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("knirvagent-relay> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			return
		}

		response, err := relayCfg.Execute(context.Background(), input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Relay error: %v\n", err)
			continue
		}
		fmt.Println(response)
	}
}

func simpleInteractiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("knirvagent> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			return
		}

		response, err := agentLoop.ProcessDirect(context.Background(), input, sessionKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		fmt.Println(response)
	}
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".knirvagent", "config.json")
}

func loadConfig() (*config.Config, error) {
	return config.LoadConfig(getConfigPath())
}

func copyEmbeddedToTarget(targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	return fs.WalkDir(embeddedFiles, "workspace", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded file %s: %w", path, err)
		}

		rel, err := filepath.Rel("workspace", path)
		if err != nil {
			return fmt.Errorf("get relative path for %s: %w", path, err)
		}

		targetPath := filepath.Join(targetDir, rel)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", filepath.Dir(targetPath), err)
		}
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("write file %s: %w", targetPath, err)
		}
		return nil
	})
}
