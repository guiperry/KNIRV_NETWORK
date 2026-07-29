# KNIRVAGENT

KNIRVAGENT is the Go-based AI agent runtime used by KNIRV. It provides a
workspace-aware tool-using agent for interactive CLI sessions, messaging
channels, and managed per-DVE processes. The agent exchanges messages through
an internal bus, keeps session and state data in its workspace, and can call
OpenAI-compatible or native provider integrations.

This package is part of the KNIRV repository and is not the upstream PicoClaw
project. Some embedded workspace templates and source comments retain that
project's historical naming.

## What it does

- Runs an agent loop with conversation history, context construction,
  summarization, token budgets, and configurable tool-iteration limits.
- Uses configurable providers including Gemini, OpenAI, Anthropic, OpenRouter,
  Groq, vLLM, NVIDIA, Moonshot, DeepSeek, ShengSuanYun, and GitHub Copilot.
  OpenAI and Anthropic can also use the built-in OAuth/token authentication
  flows. Gateway/server mode can promote configured fallback providers when the
  primary provider is unavailable.
- Provides workspace-scoped tools for reading, writing, listing, editing, and
  appending files; shell execution; web search and fetching; subagents; direct
  message delivery; cron jobs; and Linux I2C/SPI access.
- Loads skills from the workspace, the global KNIRVAGENT skills directory, and
  built-in skills. Workspace skills take precedence over global and built-in
  skills.
- Connects to Telegram, Discord, Slack, WhatsApp, MaixCam, and KNIRVWallet
  when enabled. When running inside a DVE it also registers DVE, controller,
  and terminal channels as applicable.
- Runs optional heartbeat jobs, scheduled agent jobs, USB/device monitoring,
  and Groq voice transcription for Telegram, Discord, and Slack.
- Builds for native platforms as well as WASI WebAssembly and browser
  JavaScript/WebAssembly. The Makefile copies those WebAssembly artifacts into
  the KNIRV CLI and KNIRVBRIDGE packages.

## Requirements

- Go 1.25.7 or newer (the module declares `go 1.25.7`).
- An API key, local endpoint, OAuth/token credential, or Claude/Codex CLI
  installation matching the selected provider.
- Linux is required for the functional I2C, SPI, and USB-monitoring paths.

## Build and test

From this directory:

```bash
go test ./...
make build       # native binary plus WASI and browser WASM artifacts
make build-all   # Linux amd64/arm64/riscv64, macOS arm64, Windows amd64
make install     # installs to ~/.local/bin by default
```

`make build` runs `go generate` first. Generation embeds the repository's
`workspace/` directory into the executable, so changes to the workspace
templates should be followed by a rebuild.

Useful Make variables include `INSTALL_PREFIX`, `KNIRVAGENT_HOME` (used by
the workspace defaults), `VERSION`, and `GOFLAGS`.

## Quick start

```bash
make build
./build/knirvagent onboard
```

`onboard` creates `~/.knirvagent/config.json` and the initial workspace. Add
credentials to the generated configuration, then run either:

```bash
./build/knirvagent agent -m "List the files in my workspace"
./build/knirvagent agent                 # interactive mode
```

The default configuration selects the `gemini` provider and
`gemini-2.0-flash`. A minimal provider configuration is:

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.knirvagent/workspace",
      "restrict_to_workspace": true,
      "provider": "openrouter",
      "model": "google/gemini-2.0-flash",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "YOUR_OPENROUTER_API_KEY",
      "api_base": "https://openrouter.ai/api/v1"
    }
  }
}
```

The complete checked-in example is at
[`config/config.example.json`](config/config.example.json). Configuration can
also be populated through the `KNIRV_*` environment variables defined on the
configuration structs.

## Commands

```text
knirvagent onboard
knirvagent agent [-m MESSAGE] [-s SESSION] [--debug]
knirvagent gateway [--debug]
knirvagent server --dve-id ID --socket-path PATH
knirvagent status
knirvagent version
knirvagent auth login|logout|status
knirvagent cron list|add|remove|enable|disable
knirvagent skills list|install|remove|install-builtin|list-builtin|search|show
knirvagent migrate [--dry-run] [--refresh] [--config-only|--workspace-only]
```

`agent` processes one message or starts a readline-based interactive session.
`gateway` starts the configured channels, cron service, heartbeat service,
device service, and agent loop. `status` reports configuration, workspace,
provider-key, and stored OAuth/token status. `migrate` imports configuration and
workspace data from OpenClaw.

## Gateway configuration

The gateway listens on `gateway.host` and `gateway.port` (default
`0.0.0.0:18790`). Channel credentials and sender allow-lists are configured
under `channels`; every channel is disabled by default. The currently
implemented externally configured channels are:

| Channel | Main configuration |
| --- | --- |
| Telegram | `enabled`, `token`, optional `proxy`, `allow_from` |
| Discord | `enabled`, `token`, `allow_from` |
| Slack | `enabled`, `bot_token`, `app_token`, `allow_from` |
| WhatsApp | `enabled`, `bridge_url`, `allow_from` |
| MaixCam | `enabled`, `host`, `port`, `allow_from` |
| KNIRVWallet | `enabled`, `token`, `allow_from` |

The heartbeat service is controlled by `heartbeat.enabled` and
`heartbeat.interval` (minutes). Cron jobs are stored in
`<workspace>/cron/jobs.json` and can be created from the CLI or by the agent's
`cron` tool.

## Managed server mode

KNIRVSERVER starts one KNIRVAGENT server process per DVE. The process listens
on the supplied Unix socket and exposes:

- `GET /health` for readiness and provider/model information.
- `POST /api/execute` with `{ "command": "..." }` for a direct agent request.
- `/api/inner/*` for spawning and managing inner agent sessions, including
  session input, resize, streaming output, logs, and termination.

Example:

```bash
knirvagent server --dve-id dve-123 --socket-path /tmp/dve-123.sock
```

The server normalizes the DVE identity into `DVE_ID`, starts DVE/controller
channels when the relevant environment is present, and uses the configured
workspace as the shared root for inner agents.

## Workspace and safety

The default workspace is `~/.knirvagent/workspace`. It contains the agent
identity/instructions, memory, sessions, state, cron data, and workspace
skills. `restrict_to_workspace` is enabled by default; when enabled, file and
shell tools reject paths outside the configured workspace.

The embedded templates are `workspace/AGENT.md`, `SOUL.md`, `IDENTITY.md`, and
`USER.md`. Edit the copied files in the user's workspace to customize behavior
without changing the embedded defaults.

## License

This package is distributed under the MIT License; see [LICENSE](LICENSE).
