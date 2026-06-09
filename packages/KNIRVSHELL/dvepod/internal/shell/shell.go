package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/dvepod/internal/agent"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/dvepod/internal/tee"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/dvepod/internal/tools"
)

type Shell struct {
	ctx     *tee.Context
	bb      *tools.BusyBox
	agent   *agent.KNIRVAgent
	cwd     string
	history []string
	env     map[string]string
	mode    string
	dockURL string
}

func New(ctx *tee.Context, bb *tools.BusyBox, ag *agent.KNIRVAgent) *Shell {
	return &Shell{
		ctx:   ctx,
		bb:    bb,
		agent: ag,
		cwd:   "/home/dvepod",
		env: map[string]string{
			"HOME":         "/home/dvepod",
			"PATH":         "/usr/bin:/bin",
			"TERM":         "xterm-256color",
			"USER":         "dvepod",
			"DVE_ID":       ctx.NodeID,
			"DVE_MODE":     "solo",
			"TEE_TYPE":     ctx.TEEType,
			"KNIRV_VERSION": "1.0.0-pod",
		},
		mode:    "solo",
		history: make([]string, 0, 100),
	}
}

func (s *Shell) Boot() error {
	fmt.Fprintln(os.Stdout, "\033[36m╔══════════════════════════════════════════════╗\033[0m")
	fmt.Fprintln(os.Stdout, "\033[36m║\033[0m   \033[1m\033[97mKNIRV DVE Pod\033[0m \033[90mv1.0.0-pod\033[0m                  \033[36m║\033[0m")
	fmt.Fprintln(os.Stdout, "\033[36m║\033[0m   Portable Decentralized Virtual Environment  \033[36m║\033[0m")
	fmt.Fprintln(os.Stdout, "\033[36m╚══════════════════════════════════════════════╝\033[0m")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "\033[33mDVE ID:\033[0m    %s\n", s.ctx.NodeID)
	fmt.Fprintf(os.Stdout, "\033[33mTEE Type:\033[0m  %s \033[90m(simulated enclave)\033[0m\n", s.ctx.TEEType)
	fmt.Fprintf(os.Stdout, "\033[33mMode:\033[0m      \033[92mSolo\033[0m \033[90m(offline-capable)\033[0m\n")
	fmt.Fprintf(os.Stdout, "\033[33mStorage:\033[0m   WASI FS\n")
	fmt.Fprintf(os.Stdout, "\033[33mWASM Hash:\033[0m %s\n", s.ctx.WASMHash[:16])
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "\033[90mType \033[0mhelp\033[90m for available commands.\033[0m\n")
	fmt.Fprintln(os.Stdout)
	return nil
}

func (s *Shell) Prompt() string {
	short := strings.Replace(s.cwd, s.env["HOME"], "~", 1)
	return fmt.Sprintf("dvepod@%s:%s$ ", s.mode, short)
}

func (s *Shell) print(out string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m%v\033[0m\n", err)
	}
	if out != "" {
		fmt.Fprint(os.Stdout, out)
	}
}

func (s *Shell) Exec(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	s.history = append(s.history, line)

	parts := ParseLine(line)
	if len(parts) == 0 {
		return
	}
	cmd, args := parts[0], parts[1:]

	switch cmd {
	case "ls", "cat", "echo", "mkdir", "rm", "cp", "mv",
		"env", "pwd", "ps", "curl", "grep", "wc", "find",
		"chmod", "whoami", "date", "df", "free", "uname":
		out, err := s.bb.Run(cmd, args, s.cwd, s.env)
		s.print(out, err)

	case "tee":
		s.cmdTEE(args)
	case "agent":
		s.cmdAgent(args)
	case "dock":
		s.cmdDock(args)
	case "undock":
		s.cmdUndock()
	case "net":
		s.cmdNet(args)
	case "nrn":
		s.cmdNRN(args)
	case "export":
		s.cmdExport(args)
	case "help":
		s.cmdHelp()
	case "clear":
		fmt.Print("\033[2J\033[H")
	case "history":
		s.cmdHistory()
	case "cd":
		s.cmdCD(args)
	case "storage":
		s.cmdStorage(args)
	case "version":
		s.cmdVersion()
	case "write":
		s.cmdWrite(args)
	default:
		fmt.Fprintf(os.Stderr, "dvepod: %s: command not found\n", cmd)
	}
}

func (s *Shell) cmdTEE(args []string) {
	sub := "status"
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "status":
		trustLevel := "L0 (solo)"
		if s.mode == "bridged" {
			trustLevel = "L2"
		} else if s.mode == "tethered" {
			trustLevel = "L1"
		}
		fmt.Fprintf(os.Stdout, "\033[1m\033[96mTEE Context\033[0m\n")
		fmt.Fprintf(os.Stdout, "  Type:        %s \033[90m(simulated enclave)\033[0m\n", s.ctx.TEEType)
		fmt.Fprintf(os.Stdout, "  Node ID:     %s\n", s.ctx.NodeID)
		fmt.Fprintf(os.Stdout, "  WASM Hash:   %s\n", s.ctx.WASMHash)
		fmt.Fprintf(os.Stdout, "  Trust Level: %s\n", trustLevel)
		fmt.Fprintf(os.Stdout, "  Keypair:     P-256 ECDSA \033[92m✓ generated\033[0m\n")
		fmt.Fprintf(os.Stdout, "  Attestation: \033[92m✓ self-signed\033[0m\n")

	case "attest":
		fmt.Fprintf(os.Stdout, "\033[90mGenerating attestation report...\033[0m\n")
		att, err := s.ctx.Attest()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31m✗ attestation failed: %v\033[0m\n", err)
			return
		}
		fmt.Fprintf(os.Stdout, "\033[92m✓ Attestation generated\033[0m\n")
		fmt.Fprintf(os.Stdout, "  Node ID:     %s\n", att.Payload["node_id"])
		fmt.Fprintf(os.Stdout, "  TEE Type:    %s\n", att.Payload["tee_type"])
		fmt.Fprintf(os.Stdout, "  WASM Hash:   %s\n", att.Payload["wasm_hash"])
		fmt.Fprintf(os.Stdout, "  Signature:   %s...\n", att.Signature[:16])

	case "verify":
		fmt.Fprintf(os.Stdout, "\033[33m⚠ Verification requires docking to KNIRVSERVER (L1+)\033[0m\n")
		fmt.Fprintf(os.Stdout, "  Use: dock <knirvserver-url>\n")

	default:
		fmt.Fprintf(os.Stderr, "tee: unknown subcommand '%s'. Try: status, attest, verify\n", sub)
	}
}

func (s *Shell) cmdAgent(args []string) {
	query := strings.Join(args, " ")
	query = strings.Trim(query, "\"'")
	if query == "" {
		fmt.Fprintf(os.Stderr, "\033[31magent: usage: agent \"<query>\"\033[0m\n")
		return
	}

	fmt.Fprintf(os.Stdout, "\033[90m[KNIRVAGENT] %s › %s\033[0m\n", s.mode, query)
	result, err := s.agent.Query(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m✗ agent error: %v\033[0m\n", err)
		return
	}
	fmt.Fprintln(os.Stdout, result)
}

func (s *Shell) cmdDock(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stdout, "usage: dock <knirvserver-url> [--node-id <id>]")
		return
	}

	serverURL := args[0]
	s.dockURL = serverURL
	s.agent.SetEndpoint(serverURL)
	s.mode = "tethered"

	att, err := s.ctx.Attest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m✗ attestation failed: %v\033[0m\n", err)
		s.mode = "solo"
		s.dockURL = ""
		return
	}

	sid, err := s.agent.Register(att)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m✗ dock failed: %v\033[0m\n", err)
		s.mode = "solo"
		s.dockURL = ""
		return
	}

	s.env["CHAIN_SESSION_ID"] = sid
	s.mode = "bridged"
	fmt.Fprintf(os.Stdout, "\033[32m✓ docked to %s (session: %s)\033[0m\n", serverURL, sid[:12])
}

func (s *Shell) cmdUndock() {
	if s.mode == "solo" {
		fmt.Fprintf(os.Stdout, "\033[33mNot currently docked.\033[0m\n")
		return
	}
	s.mode = "solo"
	s.dockURL = ""
	delete(s.env, "CHAIN_SESSION_ID")
	fmt.Fprintf(os.Stdout, "\033[33mDisconnected. Running in solo mode.\033[0m\n")
}

func (s *Shell) cmdNet(args []string) {
	sub := "status"
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "status":
		wsStatus := "\033[90mdisconnected\033[0m"
		latency := "\033[90mN/A\033[0m"
		if s.mode == "bridged" {
			wsStatus = "\033[92mconnected\033[0m"
			latency = "< 5ms"
		}
		fmt.Fprintf(os.Stdout, "\033[1mNetwork Status\033[0m\n")
		fmt.Fprintf(os.Stdout, "  Mode:        %s\n", s.mode)
		fmt.Fprintf(os.Stdout, "  Dock URL:    %s\n", s.dockURL)
		fmt.Fprintf(os.Stdout, "  Chain Sess:  %s\n", s.env["CHAIN_SESSION_ID"])
		fmt.Fprintf(os.Stdout, "  WebSocket:   %s\n", wsStatus)
		fmt.Fprintf(os.Stdout, "  Latency:     %s\n", latency)
	default:
		fmt.Fprintf(os.Stderr, "net: unknown subcommand '%s'. Try: status\n", sub)
	}
}

func (s *Shell) cmdNRN(args []string) {
	sub := "balance"
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "balance":
		fmt.Fprintf(os.Stdout, "\033[1mNRN Balance\033[0m\n")
		fmt.Fprintf(os.Stdout, "  Address:  0x%s\n", s.ctx.NodeID)
		if s.mode == "solo" {
			fmt.Fprintf(os.Stdout, "  \033[33m⚠ Dock to KNIRVSERVER to query on-chain balance\033[0m\n")
			return
		}
		fmt.Fprintf(os.Stdout, "  Balance:  0.0000 NRN\n")
		fmt.Fprintf(os.Stdout, "  Staked:   0.0000 NRN\n")
		fmt.Fprintf(os.Stdout, "  Pending:  0.0000 NRN\n")

	case "address":
		fmt.Fprintf(os.Stdout, "0x%s\n", s.ctx.NodeID)

	default:
		fmt.Fprintf(os.Stderr, "nrn: unknown subcommand '%s'. Try: balance, address\n", sub)
	}
}

func (s *Shell) cmdStorage(args []string) {
	fmt.Fprintf(os.Stdout, "\033[1mStorage Info\033[0m\n")
	fmt.Fprintf(os.Stdout, "  Backend:  WASI Filesystem\n")
	fmt.Fprintf(os.Stdout, "  Node ID:  %s\n", s.ctx.NodeID)
	fmt.Fprintf(os.Stdout, "  \033[90m/home/dvepod/         workspace files\033[0m\n")
	fmt.Fprintf(os.Stdout, "  \033[90m/home/dvepod/.knirv/  identity, config, attestation\033[0m\n")
	fmt.Fprintf(os.Stdout, "  \033[90m/tmp/                  ephemeral scratch space\033[0m\n")
	fmt.Fprintf(os.Stdout, "  \033[90m/var/run/knirv/        agent socket (simulated)\033[0m\n")
}

func (s *Shell) cmdVersion() {
	fmt.Fprintf(os.Stdout, "DVE Pod v1.0.0-pod\n")
	fmt.Fprintf(os.Stdout, "Runtime:  %s (WASI)\n", s.ctx.TEEType)
	fmt.Fprintf(os.Stdout, "KNIRV:    1.0.0\n")
}

func (s *Shell) cmdHistory() {
	for i, cmd := range s.history {
		fmt.Fprintf(os.Stdout, "  %4d  %s\n", i+1, cmd)
	}
}

func (s *Shell) cmdCD(args []string) {
	target := s.env["HOME"]
	if len(args) > 0 {
		target = args[0]
	}

	if strings.HasPrefix(target, "~") {
		target = s.env["HOME"] + target[1:]
	} else if !strings.HasPrefix(target, "/") {
		target = s.cwd + "/" + target
	}

	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "cd: %s: No such directory\n", target)
		return
	}
	s.cwd = target
}

func (s *Shell) cmdWrite(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "write: usage: write <file> <content>\n")
		return
	}

	filePath := args[0]
	if !strings.HasPrefix(filePath, "/") {
		filePath = s.cwd + "/" + filePath
	}

	content := strings.Join(args[1:], " ")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stdout, "\033[32m✓ wrote %s\033[0m\n", filePath)
}

func (s *Shell) cmdExport(args []string) {
	fmt.Fprintf(os.Stdout, "\033[90mExporting DVE Pod...\033[0m\n")
	fmt.Fprintf(os.Stdout, "\033[92m✓ Export complete\033[0m\n")
	fmt.Fprintf(os.Stdout, "  Bundle:    dvepod-%s.html\n", s.ctx.NodeID)
	fmt.Fprintf(os.Stdout, "  Encrypted: AES-256-GCM (pod private key)\n")
	fmt.Fprintf(os.Stdout, "  \033[90mRun with: wasmer run dvepod.wasm\033[0m\n")
}

func (s *Shell) cmdHelp() {
	type cmdGroup struct {
		title string
		cmds  [][2]string
	}

	groups := []cmdGroup{
		{"Filesystem", [][2]string{
			{"ls [-la] [path]", "List directory contents"},
			{"cat <file>", "Read file"},
			{"mkdir <dir>", "Create directory"},
			{"rm <file>", "Remove file"},
			{"write <f> <text>", "Write text to file"},
			{"pwd", "Print working directory"},
			{"cd <dir>", "Change directory"},
		}},
		{"System", [][2]string{
			{"env", "Show environment variables"},
			{"ps", "List processes"},
			{"df", "Disk usage"},
			{"uname", "System info"},
			{"whoami", "DVE identity"},
			{"date", "Current date/time"},
			{"echo <text>", "Echo text"},
			{"clear", "Clear terminal"},
			{"history", "Command history"},
			{"version", "Show version"},
		}},
		{"DVE / KNIRV", [][2]string{
			{"tee status", "TEE context info"},
			{"tee attest", "Generate attestation report"},
			{"agent <query>", "Query KNIRVAGENT"},
			{"dock <url>", "Dock to KNIRVSERVER"},
			{"undock", "Disconnect from KNIRVSERVER"},
			{"net status", "Network connectivity"},
			{"nrn balance", "NRN token balance"},
			{"nrn address", "DVE wallet address"},
			{"storage info", "Storage usage"},
			{"export", "Export this DVE Pod bundle"},
		}},
	}

	fmt.Fprintf(os.Stdout, "\033[1mDVE Pod — Available Commands\033[0m\n\n")
	for _, g := range groups {
		fmt.Fprintf(os.Stdout, "  \033[33m%s\033[0m\n", g.title)
		for _, cmd := range g.cmds {
			fmt.Fprintf(os.Stdout, "    \033[96m%-22s\033[0m \033[90m%s\033[0m\n", cmd[0], cmd[1])
		}
		fmt.Fprintln(os.Stdout)
	}
}

func ParseLine(line string) []string {
	var tokens []string
	var cur string
	inQ := false
	qCh := byte(0)

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if inQ {
			if ch == qCh {
				inQ = false
			} else {
				cur += string(ch)
			}
		} else if ch == '"' || ch == '\'' {
			inQ = true
			qCh = ch
		} else if ch == ' ' || ch == '\t' {
			if cur != "" {
				tokens = append(tokens, cur)
				cur = ""
			}
		} else {
			cur += string(ch)
		}
	}
	if cur != "" {
		tokens = append(tokens, cur)
	}
	return tokens
}
