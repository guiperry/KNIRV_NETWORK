# KNIRVSHELL

A comprehensive command-line interface that provides full integration with the entire KNIRV Network ecosystem. This sophisticated, fully-integrated interface manages and interacts with all KNIRV Network services.

## 🚀 Key Features

### 1. Service Discovery and Configuration
- **Dynamic Service Registry**: Automatically discovers and registers KNIRV Network services
- **Health Monitoring**: Real-time health checks and status monitoring for all services
- **Circuit Breaker Pattern**: Robust error handling and automatic retry mechanisms
- **Hot Configuration Reloading**: Update configuration without restarting

### 2. Complete KNIRV Network Integration
- **KNIRVORACLE Client**: Blockchain operations, agent management, economics integration
- **KNIRVGATEWAY Client**: Unified API gateway access, health monitoring, authentication
- **KNIRVSERVER Client**: DVE rental, KNIRVENGINE, inference API integration
- **KNIRVGRAPH Client**: NRV system, ErrorNode/SkillNode operations, graph queries

### 3. Wallet and Economics Integration
- **XION Meta Account Support**: Gasless transactions and meta account management
- **NRN Token Manager**: Complete NRN token operations with auto-refill capabilities
- **Economics Module Integration**: Skill registration, fee calculation, transaction management

### 4. Real-time Communication
- **WebSocket Manager**: Real-time bidirectional communication with KNIRV services
- **Server-Sent Events (SSE)**: Event streaming for live updates and notifications
- **Event Bus System**: Centralized event handling and distribution

### 5. Advanced Features
- **Network Resolution Vector (NRV)**: Submit ErrorNodes and SkillNodes to the graph
- **MCP capability management**: AI integration with multiple providers
- **MCP server registration**: Operational procedure management
- **AI-powered plugin generation**: Inference Engine Integration with direct access to KNIRV inference capabilities
- **Interactive terminal UI**: Interactive shell (REPL) with command history and tab completion
- **Configuration management**: Environment-specific overrides and service-specific settings

## 📋 Installation and Setup

### Prerequisites
- Go 1.21 or later
- Access to KNIRV Network services
- Valid API keys for enabled services

### Option 1: Download Pre-built Binaries (Recommended)

1. **Download the appropriate binary** for your operating system and architecture from the [GitHub Releases page](https://github.com/guiperry/KNIRV_Network/releases).

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
   
   # Add to PATH (run in PowerCLI as Administrator)
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
# Open PowerCLI as Administrator and navigate to the extracted directory

# Run the installation script
.\install.ps1

# The script will:
# - Ask where to install the binary (default: C:\Program Files\KnirvCLI)
# - Copy the binary to the chosen location
# - Add the location to your PATH if it's not already included
```

### Option 3: Package Managers

For a smoother installation experience, KNIRVSHELL is also available through popular package managers:

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
git clone https://github.com/guiperry/KNIRV_Network.git
cd KNIRVSHELL

# Install dependencies
go mod tidy

# Build the binary
go build -o knirv

# Optionally, install to $GOPATH/bin
go install
```

### Configuration

1. Copy the sample configuration:
```bash
cp config/sample.yaml ~/.knirv/config.yaml
```

2. Edit the configuration file to match your environment:
```yaml
knirv:
  network:
    environment: "development"
  services:
    knirvoracle:
      url: "http://localhost:9999"
      api_key: "your-api-key"
    # ... other services
```

3. Set environment variables:
```bash
export KNIRV_ROOT_API_KEY="your-root-api-key"
export KNIRV_GATEWAY_API_KEY="your-gateway-api-key"
```

## 🎯 Quick Start

### 1. Initialize the System
```bash
# Initialize KNIRV Network integration
./knirv system init

# Check system status
./knirv system status

# Test integration
./knirv system test
```

### 2. Network Operations
```bash
# Check network status
./knirv network status

# Discover services
./knirv network discover

# Connect to all services
./knirv network connect
```

### 3. Economics and NRN Tokens
```bash
# Check NRN balance
./knirv economics balance --wallet my-wallet

# Transfer NRN tokens
./knirv economics transfer 0x1234... 1000 --wallet my-wallet

# Request from faucet
./knirv economics faucet 5000 --wallet my-wallet

# View transaction history
./knirv economics history
```

### 4. NRV System Operations
```bash
# Submit an error to the NRV system
./knirv mcp nrv submit-error "connection-timeout" "Database connection timeout" --severity 7

# Submit a skill to resolve errors
./knirv mcp nrv submit-skill "database-troubleshooting" "connection-repair,timeout-handling"

# Query skills for error resolution
./knirv mcp nrv query-skills connection-timeout

# View NRV system statistics
./knirv mcp nrv stats
```

## Usage

KNIRVSHELL can be used in two modes:

### Command Mode

Run specific commands directly:

```sh
knirv [command]

Available Commands:
  help        Help about any command
  init        Initialize KNIRVSHELL configuration
  mcp         Manage MCP capabilities and servers
  version     Display version information
  wallet      Manage KNIRV wallets

Flags:
      --ai-model string       AI model to use for generation
      --ai-provider string    AI provider to use (openai, anthropic) (default "openai")
      --color-mode string     set color mode (16, 256, truecolor) (default "256")
      --config string         config file (default is $HOME/.knirv.yaml)
  -h, --help                  help for knirv
      --log-format string     set logging format (text, json) (default "text")
      --log-level string      set logging level (debug, info, warn, error) (default "info")
      --no-animations         disable UI animations
      --node-url string       URL of the KNIRV node
      --theme string          set UI theme (default, dark, light, high-contrast) (default "default")
      --tui                   enable terminal UI mode with bubbletea
  -v, --verbose               enable verbose logging
```

### Interactive CLI Mode

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
║                KNIRVSHELL Interactive Mode                  ║
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
Exiting KNIRVSHELL.
```

### Initialize Configuration

```sh
knirv init [flags]

Flags:
      --config string       Configuration file path
      --log-level string    Logging level (debug, info, warn, error)
      --node-url string     KNIRV-ORACLE node URL
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
  nrv         Network Resolution Vector operations
```

## 🏗️ Architecture

### Core Components

#### Service Registry (`core/service_registry.go`)
- Manages service discovery and registration
- Maintains service health status
- Provides service lookup and routing

#### Health Monitor (`core/health_monitor.go`)
- Continuous health monitoring of all services
- Circuit breaker implementation
- Health status reporting and alerting

#### KNIRV Service Clients
- **KNIRVRootClient** (`core/knirvoracle_client.go`): Blockchain and economics operations
- **KNIRVGatewayClient** (`core/knirvgateway_client.go`): Gateway and proxy operations
- **KNIRVServerClient** (`core/knirvnexus_client.go`): DVE and inference operations
- **KNIRVGraphClient** (`core/knirvgraph_client.go`): Graph and NRV operations

#### Wallet Management
- **XIONWalletManager** (`core/xion_wallet_manager.go`): XION Meta Account support
- **NRNTokenManager** (`core/nrn_token_manager.go`): NRN token operations

#### Real-time Communication
- **WebSocketManager** (`core/websocket_manager.go`): WebSocket connections
- **SSEClient** (`core/sse_client.go`): Server-Sent Events
- **EventBus** (`core/event_bus.go`): Event distribution

### Configuration Structure

The configuration supports:
- Service-specific endpoints and authentication
- Real-time communication settings
- Wallet and economics configuration
- Environment-specific overrides

## 📚 Command Reference

### System Commands
- `system init` - Initialize KNIRV Network integration
- `system status` - Show comprehensive system status
- `system test` - Test integration functionality
- `system reset` - Reset configuration to defaults

### Network Commands
- `network status` - Show network and service health
- `network discover` - Discover available services
- `network connect` - Connect to all services
- `network disconnect` - Disconnect from all services

### Economics Commands
- `economics balance [address]` - Get NRN token balance
- `economics transfer <to> <amount>` - Transfer NRN tokens
- `economics faucet [amount]` - Request tokens from faucet
- `economics history` - Show transaction history
- `economics stats` - Show token statistics

### NRV System Commands
- `mcp nrv submit-error <type> <description>` - Submit ErrorNode
- `mcp nrv submit-skill <type> <capabilities>` - Submit SkillNode
- `mcp nrv query-skills [error-type]` - Query available skills
- `mcp nrv query-errors [status]` - Query error nodes
- `mcp nrv stats` - Show NRV system statistics

## 🔧 Development

### Adding New Service Clients
1. Create a new client file in `core/`
2. Implement the `KNIRVServiceClient` interface
3. Register the client in the service registry
4. Add configuration support

### Extending Commands
1. Create command files in `cmd/`
2. Follow the existing command structure
3. Add appropriate flags and validation
4. Update help documentation

### Testing
```bash
# Run all tests
go test ./...

# Test specific components
go test ./core/...

# Integration tests
./knirv system test --all

# Build and run in command mode
go run main.go version

# Run in interactive shell mode
go run main.go

# Initialize configuration
go run main.go init --overwrite

# Generate a new wallet (Phase 2)
go run main.go wallet new
```

## 🌐 Integration Examples

### Custom Event Handlers
```go
type MyEventHandler struct{}

func (h *MyEventHandler) HandleEvent(event *core.Event) error {
    log.Printf("Received event: %s from %s", event.Type, event.Source)
    return nil
}

// Register handler
eventBus.Subscribe([]string{"blockchain-update"}, &MyEventHandler{})
```

### Service Health Monitoring
```go
healthMonitor := core.NewHealthMonitor(registry, logger)
healthCh := healthMonitor.Subscribe()

go func() {
    for result := range healthCh {
        if result.Status == core.ServiceStatusUnhealthy {
            log.Warnf("Service %s is unhealthy: %s", result.ServiceName, result.Error)
        }
    }
}()
```

## 🚨 Troubleshooting

### Common Issues

1. **Service Discovery Fails**
   - Check network connectivity
   - Verify service URLs in configuration
   - Ensure services are running

2. **Authentication Errors**
   - Verify API keys are set correctly
   - Check environment variables
   - Ensure keys have proper permissions

3. **WebSocket Connection Issues**
   - Check firewall settings
   - Verify WebSocket endpoints
   - Review connection timeouts

### Debug Mode
```bash
# Enable debug logging
export LOG_LEVEL=debug
./knirv system status
```

## 📈 Performance Considerations

- Service discovery runs every 30 seconds by default
- Health checks have configurable timeouts
- Connection pooling for HTTP clients
- Circuit breaker prevents cascade failures
- Event bus uses goroutines for non-blocking event handling

## 🔐 Security

- API keys stored in environment variables
- TLS/SSL support for all external connections
- Input validation on all commands
- Secure wallet storage with encryption

## Project Structure

- `cmd/`: Command implementations
  - `mcp/`: MCP-related commands
  - `system/`: System management commands
  - `network/`: Network operation commands
  - `economics/`: Economics and NRN token commands
- `config/`: Configuration management
- `core/`: Core functionality
  - Service clients for KNIRV Network integration
  - Health monitoring and service registry
  - Real-time communication components
- `internal/`: Internal packages
- `pkg/`: Public packages
  - `ai/`: AI client and generator
  - `tui/`: Terminal UI components

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

For more information, see the main KNIRV Network documentation.

## 📄 License

Copyright (c) 2025 KNIRV Network

Permission is hereby granted, Inc. All rights reserved.

This project is part of the KNIRV Network ecosystem.