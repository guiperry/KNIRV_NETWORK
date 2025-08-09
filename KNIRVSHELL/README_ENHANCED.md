# Enhanced KNIRVSHELL - Complete KNIRV Network Integration

## Overview

The Enhanced KNIRVSHELL is a comprehensive command-line interface that provides full integration with the entire KNIRV Network ecosystem. This implementation transforms the basic CLI tool into a sophisticated, fully-integrated interface for managing and interacting with all KNIRV Network services.

## 🚀 Key Features

### 1. Enhanced Service Discovery and Configuration
- **Dynamic Service Registry**: Automatically discovers and registers KNIRV Network services
- **Health Monitoring**: Real-time health checks and status monitoring for all services
- **Circuit Breaker Pattern**: Robust error handling and automatic retry mechanisms
- **Hot Configuration Reloading**: Update configuration without restarting

### 2. Complete KNIRV Network Integration
- **KNIRVROOT Client**: Blockchain operations, agent management, economics integration
- **KNIRVGATEWAY Client**: Unified API gateway access, health monitoring, authentication
- **KNIRVNEXUS Client**: DVE rental, agentic engine, inference API integration
- **KNIRVGRAPH Client**: NRV system, ErrorNode/SkillNode operations, graph queries

### 3. Enhanced Wallet and Economics Integration
- **XION Meta Account Support**: Gasless transactions and meta account management
- **NRN Token Manager**: Complete NRN token operations with auto-refill capabilities
- **Economics Module Integration**: Skill registration, fee calculation, transaction management

### 4. Real-time Communication
- **WebSocket Manager**: Real-time bidirectional communication with KNIRV services
- **Server-Sent Events (SSE)**: Event streaming for live updates and notifications
- **Event Bus System**: Centralized event handling and distribution

### 5. Advanced Features
- **Network Resolution Vector (NRV)**: Submit ErrorNodes and SkillNodes to the graph
- **Enhanced MCP Management**: AI integration with multiple providers
- **Inference Engine Integration**: Direct access to KNIRV inference capabilities

## 📋 Installation and Setup

### Prerequisites
- Go 1.21 or later
- Access to KNIRV Network services
- Valid API keys for enabled services

### Installation
```bash
# Clone the repository
git clone <repository-url>
cd KNIRVSHELL

# Install dependencies
go mod tidy

# Build the enhanced shell
go build -o knirv .
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
    knirvroot:
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
- **KNIRVRootClient** (`core/knirvroot_client.go`): Blockchain and economics operations
- **KNIRVGatewayClient** (`core/knirvgateway_client.go`): Gateway and proxy operations
- **KNIRVNexusClient** (`core/knirvnexus_client.go`): DVE and inference operations
- **KNIRVGraphClient** (`core/knirvgraph_client.go`): Graph and NRV operations

#### Enhanced Wallet Management
- **XIONWalletManager** (`core/xion_wallet_manager.go`): XION Meta Account support
- **NRNTokenManager** (`core/nrn_token_manager.go`): NRN token operations

#### Real-time Communication
- **WebSocketManager** (`core/websocket_manager.go`): WebSocket connections
- **SSEClient** (`core/sse_client.go`): Server-Sent Events
- **EventBus** (`core/event_bus.go`): Event distribution

### Configuration Structure

The enhanced configuration supports:
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

### Enhanced MCP Commands
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

## 📄 License

This project is part of the KNIRV Network ecosystem. See LICENSE file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

For more information, see the main KNIRV Network documentation.
