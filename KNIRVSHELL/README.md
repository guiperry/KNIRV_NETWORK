# AGENTCHAIN CLI

A comprehensive command-line interface for interacting with the AGENTCHAIN distributed ledger system.

## Features

- Wallet management (create, import, export, list)
- MCP capability management
- MCP server registration
- Operational procedure management
- AI-powered plugin generation
- Interactive terminal UI
- Interactive shell (REPL) with command history and tab completion
- Configuration management

## Installation

### Option 1: Download Pre-built Binaries (Recommended)

1. **Download the appropriate binary** for your operating system and architecture from the [GitHub Releases page](https://github.com/guiperry/AGENTCHAIN-CLI/releases).

   ```
   # Example filenames
   knirv_VERSION_linux_amd64.tar.gz   # For 64-bit Linux
   knirv_VERSION_darwin_amd64.tar.gz  # For 64-bit macOS
   knirv_VERSION_windows_amd64.zip    # For 64-bit Windows
   ```

2. **Extract the archive**:

   ```sh
   # Linux/macOS
   tar -xzf knirv_VERSION_YOUR_OS_YOUR_ARCH.tar.gz
   
   # Windows
   # Use Windows Explorer to extract the zip file
   ```

3. **Move the binary to a directory in your PATH**:

   **Linux/macOS**:
   ```sh
   # System-wide installation (requires sudo)
   sudo mv knirv /usr/local/bin/
   
   # OR User-specific installation
   mkdir -p ~/.local/bin
   mv knirv ~/.local/bin/
   
   # If ~/.local/bin is not in your PATH, add it to your shell configuration file:
   echo 'export PATH=$PATH:~/.local/bin' >> ~/.bashrc
   # OR for Zsh
   echo 'export PATH=$PATH:~/.local/bin' >> ~/.zshrc
   
   # Then reload your shell configuration
   source ~/.bashrc  # OR source ~/.zshrc
   ```

   **Windows**:
   ```
   # Create a directory for the executable
   mkdir "C:\Program Files\KnirvCLI"
   
   # Move the executable to this directory
   move knirv.exe "C:\Program Files\KnirvCLI"
   
   # Add to PATH (run in PowerShell as Administrator)
   [Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Program Files\KnirvCLI", [EnvironmentVariableTarget]::Machine)
   ```

4. **Make the binary executable** (Linux/macOS only):

   ```sh
   chmod +x knirv
   ```

5. **Verify installation**:

   ```sh
   knirv version
   ```

### Option 2: Using Installation Scripts

The release packages include installation scripts that automate the process:

**Linux/macOS**:
```sh
# Extract the archive
tar -xzf knirv_VERSION_YOUR_OS_YOUR_ARCH.tar.gz

# Run the installation script
./install.sh

# The script will:
# - Ask where to install the binary (default: ~/.local/bin for regular users, /usr/local/bin if run with sudo)
# - Copy the binary to the chosen location
# - Make it executable
# - Check if the location is in your PATH and provide instructions if it's not
```

**Windows**:
```powershell
# Extract the zip file
# Open PowerShell as Administrator and navigate to the extracted directory

# Run the installation script
.\install.ps1

# The script will:
# - Ask where to install the binary (default: C:\Program Files\KnirvCLI)
# - Copy the binary to the chosen location
# - Add the location to your PATH if it's not already included
```

### Option 3: Package Managers

For a smoother installation experience, AGENTCHAIN CLI is also available through popular package managers:

**macOS (Homebrew)**:
```sh
brew tap guiperry/knirv
brew install knirv
```

**Linux (Debian/Ubuntu)**:
```sh
# Add the repository
curl -fsSL https://guiperry.github.io/apt-repo/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/guiperry-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/guiperry-archive-keyring.gpg] https://guiperry.github.io/apt-repo stable main" | sudo tee /etc/apt/sources.list.d/guiperry.list

# Update and install
sudo apt update
sudo apt install knirv
```

**Linux (Fedora/RHEL)**:
```sh
# Add the repository
sudo dnf config-manager --add-repo https://guiperry.github.io/rpm-repo/guiperry.repo

# Install
sudo dnf install knirv
```

**Windows (Chocolatey)**:
```powershell
choco install knirv
```

**Windows (Scoop)**:
```powershell
scoop bucket add guiperry https://github.com/guiperry/scoop-bucket.git
scoop install knirv
```

### Option 4: Build from Source

If you prefer to build from source or are contributing to the project:

```sh
# Clone the repository
git clone https://github.com/guiperry/AGENTCHAIN-CLI.git
cd AGENTCHAIN-CLI

# Build the binary
go build -o knirv

# Optionally, install to $GOPATH/bin
go install
```

## Usage

AGENTCHAIN CLI can be used in two modes:

### Command Mode

Run specific commands directly:

```sh
knirv [command]

Available Commands:
  help        Help about any command
  init        Initialize AGENTCHAIN CLI configuration
  mcp         Manage MCP capabilities and servers
  version     Display version information
  wallet      Manage AGENTCHAIN wallets

Flags:
      --ai-model string       AI model to use for generation
      --ai-provider string    AI provider to use (openai, anthropic) (default "openai")
      --color-mode string     set color mode (16, 256, truecolor) (default "256")
      --config string         config file (default is $HOME/.knirv.yaml)
  -h, --help                  help for knirv
      --log-format string     set logging format (text, json) (default "text")
      --log-level string      set logging level (debug, info, warn, error) (default "info")
      --no-animations         disable UI animations
      --node-url string       URL of the AGENTCHAIN node
      --theme string          set UI theme (default, dark, light, high-contrast) (default "default")
      --tui                   enable terminal UI mode with bubbletea
  -v, --verbose               enable verbose logging
```

### Interactive Shell Mode

Launch an interactive REPL (Read-Eval-Print Loop) shell:

```sh
knirv
```

This opens an interactive shell with command history, tab completion, and more:

- Use `<tab>` for command completion
- Commands history is saved between sessions
- Type `help` to see available commands
- Type `exit` or `quit` to exit the shell
- Type `clear` or `cls` to clear the screen

Example session:
```
╔════════════════════════════════════════════════════════════╗
║                AGENTCHAIN CLI Interactive Mode             ║
╠════════════════════════════════════════════════════════════╣
║ • Type 'help' for a list of available commands             ║
║ • Use <tab> for command completion                         ║
║ • Type 'exit' or 'quit' to leave                           ║
║ • Type 'clear' or 'cls' to clear the screen                ║
╚════════════════════════════════════════════════════════════╝

knirv> wallet list
[wallet list output]

knirv> mcp capability list
[capability list output]

knirv> exit
Exiting AGENTCHAIN CLI.
```

### Initialize Configuration

```sh
knirv init [flags]

Flags:
      --config string       Configuration file path
      --log-level string    Logging level (debug, info, warn, error)
      --node-url string     AGENTCHAIN node URL
      --overwrite           Overwrite existing configuration
      --wallet-dir string   Wallet directory path
```

### Wallet Management

```sh
knirv wallet [command]

Available Commands:
  export      Export wallet private key
  import      Import an existing private key
  list        List available wallets
  new         Generate a new wallet
```

### MCP Management

```sh
knirv mcp [command]

Available Commands:
  capability  Manage direct plugin capability registration
  generate    AI-powered plugin generation
  invoke      Invoke a capability
  procedure   Manage operational procedure registration (interpolation)
  server      Manage MCP server registration (extrapolation)
```

## Development

```sh
# Run tests
go test ./...

# Build and run in command mode
go run main.go version

# Run in interactive shell mode
go run main.go

# Initialize configuration
go run main.go init --overwrite

# Generate a new wallet (Phase 2)
go run main.go wallet new
```

## Project Structure

- `cmd/`: Command implementations
  - `mcp/`: MCP-related commands
- `config/`: Configuration management
- `core/`: Core functionality
- `internal/`: Internal packages
- `pkg/`: Public packages
  - `ai/`: AI client and generator
  - `tui/`: Terminal UI components

## Contributing

1. Fork the repository
2. Create a feature branch
3. Submit a pull request

## License

Copyright (c) 2025 G. Perry

Permission is hereby granted, Inc. All rights reserved.