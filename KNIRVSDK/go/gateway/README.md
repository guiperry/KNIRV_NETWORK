# KNIRV Gateway SDK for Go

The official Go SDK for KNIRV Gateway services, providing access to the Economics Service and API Gateway functionality.

## Features

- **Economics Service**: Complete integration with Month 11 economics implementation
  - Skill invocation processing
  - LLM registration and fees
  - Validation rewards
  - Network fee calculation
  - Economic metrics and analytics
  - Transaction management
  - Token burn tracking
  - Economic rules management

- **Gateway Service**: API Gateway functionality
  - Route management
  - Service status monitoring
  - Health checks

- **Integration Service**: KNIRV component integration
  - Component connectivity testing
  - Integration status monitoring
  - Cross-service communication

- **Health Service**: Comprehensive health monitoring
  - Service health checks
  - System-wide health status
  - Real-time monitoring capabilities

## Installation

```bash
go get github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway
```

## Quick Start

### Basic Setup

```go
package main

import (
    "context"
    "log"

    "github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway"
    "github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway/option"
)

func main() {
    // Create a client with default options
    client := gateway.NewClient()

    // Or create with custom options
    client = gateway.NewClient(
        option.WithEnvironmentDevelopment(),
        option.WithDebug(true),
        option.WithEconomicsURL("http://localhost:8090"),
    )

    // Create economics-specific client
    economicsClient := gateway.NewEconomicsClient(
        option.WithAPIKey("your-api-key"),
    )
}
```

### Economics Operations

```go
ctx := context.Background()

// Process skill invocation
skillResp, err := client.Economics.Skills.Invoke(ctx, gateway.SkillInvocationRequest{
    UserID:  "user123",
    SkillID: "skill456",
    Amount:  "100000", // 0.1 NRN
})
if err != nil {
    log.Fatal(err)
}

// Register LLM
llmResp, err := client.Economics.LLM.Register(ctx, gateway.LLMRegistrationRequest{
    UserID:          "user123",
    LLMID:           "llm789",
    RegistrationFee: "1000000", // 1 NRN
})

// Process validation reward
validationResp, err := client.Economics.Validation.Reward(ctx, gateway.ValidationRewardRequest{
    ValidatorID:      "validator123",
    TargetID:         "target456",
    ValidationResult: true,
})

// Calculate network fees
feesResp, err := client.Economics.Fees.Calculate(ctx, gateway.NetworkFeesRequest{
    GasUsed:  21000,
    Priority: "medium",
})

// Get economic metrics
metrics, err := client.Economics.Metrics.Get(ctx)
```

### Health Monitoring

```go
// Check economics service health
isHealthy, err := client.Health.Check(ctx)

// Get integration status
status, err := client.Integration.GetStatus(ctx)
```

## Configuration

### Environment Variables

```bash
export ECONOMICS_SERVICE_URL="http://localhost:8090"
export GATEWAY_SERVICE_URL="http://localhost:8000"
export KNIRVGATEWAY_API_KEY="your-api-key"
export NRN_CONTRACT="your-nrn-contract-address"
export KNIRVCHAIN_URL="http://localhost:8080"
export KNIRVNEXUS_URL="http://localhost:8081"
export KNIRVROOT_URL="http://localhost:8082"
export KNIRVGRAPH_URL="http://localhost:8083"
```

### Client Options

```go
client := gateway.NewClient(
    // Environment settings
    option.WithEnvironmentProduction(),
    option.WithBaseURL("https://gateway.knirv.network"),
    option.WithEconomicsURL("https://economics.knirv.network"),
    
    // Authentication
    option.WithAPIKey("your-api-key"),
    option.WithNRNContract("your-contract-address"),
    
    // Network settings
    option.WithTimeout(30 * time.Second),
    option.WithRetries(3),
    
    // KNIRV services
    option.WithAllKNIRVServices(
        "https://chain.knirv.network",
        "https://nexus.knirv.network", 
        "https://root.knirv.network",
        "https://graph.knirv.network",
    ),
    
    // Debugging
    option.WithDebug(true),
    option.WithVerbose(true),
)
```

## API Reference

### Economics Service

#### Skills Service
- `Invoke(ctx, SkillInvocationRequest) (*SkillInvocationResponse, error)`

#### LLM Service
- `Register(ctx, LLMRegistrationRequest) (*LLMRegistrationResponse, error)`

#### Validation Service
- `Reward(ctx, ValidationRewardRequest) (*ValidationRewardResponse, error)`

#### Fees Service
- `Calculate(ctx, NetworkFeesRequest) (*NetworkFeesResponse, error)`

#### Metrics Service
- `Get(ctx) (*EconomicMetrics, error)`
- `GetServiceMetrics(ctx, serviceName) (*ServiceEconomics, error)`

#### Transactions Service
- `Get(ctx, transactionID) (*Transaction, error)`
- `List(ctx, limit, status) ([]*Transaction, error)`

#### Burn Service
- `GetHistory(ctx, limit) ([]*BurnEvent, error)`
- `GetTotal(ctx) (string, error)`

#### Rules Service
- `Get(ctx) (*EconomicRules, error)`
- `Update(ctx, *EconomicRules) (*EconomicRules, error)`

### Health Service
- `Check(ctx) (bool, error)`

### Integration Service
- `GetStatus(ctx) (*IntegrationStatus, error)`

## Error Handling

The SDK provides structured error handling:

```go
skillResp, err := client.Economics.Skills.Invoke(ctx, request)
if err != nil {
    // Handle different types of errors
    switch {
    case strings.Contains(err.Error(), "insufficient balance"):
        // Handle insufficient balance
    case strings.Contains(err.Error(), "invalid skill"):
        // Handle invalid skill
    default:
        // Handle general error
    }
}
```

## Examples

See the [examples](examples/) directory for comprehensive usage examples:

- [Basic Usage](examples/main.go) - Basic client setup and operations
- [Advanced Usage](examples/main.go) - Complex workflows and error handling

## Testing

```bash
go test ./...
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- Documentation: [https://docs.knirv.network](https://docs.knirv.network)
- Issues: [GitHub Issues](https://github.com/knirv-network/knirv-sdk/issues)
- Discord: [KNIRV Community](https://discord.gg/knirv)

## Changelog

### v1.0.0
- Initial release
- Complete Economics Service integration
- Gateway Service support
- Health monitoring
- Integration status tracking
- Comprehensive examples and documentation
