# KNIRV Gateway SDK for TypeScript/JavaScript

The official TypeScript/JavaScript SDK for KNIRV Gateway services, providing access to the Economics Service and API Gateway functionality.

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
npm install @knirv/gateway-sdk
# or
yarn add @knirv/gateway-sdk
# or
pnpm add @knirv/gateway-sdk
```

## Quick Start

### Basic Setup

```typescript
import { KNIRVGatewayClient } from '@knirv/gateway-sdk';

// Create a client with default options
const client = new KNIRVGatewayClient();

// Or create with custom options
const client = new KNIRVGatewayClient({
  environment: 'development',
  debug: true,
  economicsURL: 'http://localhost:8090',
});

// Create economics-specific client
const economicsClient = KNIRVGatewayClient.createEconomicsClient({
  apiKey: 'your-api-key',
});
```

### Economics Operations

```typescript
// Process skill invocation
const skillResp = await client.economics.invokeSkill({
  user_id: 'user123',
  skill_id: 'skill456',
  amount: '100000', // 0.1 NRN
});

// Register LLM
const llmResp = await client.economics.registerLLM({
  user_id: 'user123',
  llm_id: 'llm789',
  registration_fee: '1000000', // 1 NRN
});

// Process validation reward
const validationResp = await client.economics.processValidationReward({
  validator_id: 'validator123',
  target_id: 'target456',
  validation_result: true,
});

// Calculate network fees
const feesResp = await client.economics.calculateNetworkFees({
  gas_used: 21000,
  priority: 'medium',
});

// Get economic metrics
const metrics = await client.economics.getMetrics();
```

### Health Monitoring

```typescript
// Check economics service health
const economicsHealth = await client.health.checkEconomicsHealth();

// Check all services
const allHealth = await client.health.checkAllServices();

// Get system health summary
const systemHealth = await client.health.getSystemHealth();

// Wait for service to become healthy
const isReady = await client.health.waitForService('economics', {
  timeout: 30000,
  interval: 2000,
});
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
export KNIRVORACLE_URL="http://localhost:8082"
export KNIRVGRAPH_URL="http://localhost:8083"
```

### Client Options

```typescript
const client = new KNIRVGatewayClient({
  // Environment settings
  environment: 'production',
  baseURL: 'https://gateway.knirv.network',
  economicsURL: 'https://economics.knirv.network',
  
  // Authentication
  apiKey: 'your-api-key',
  nrnContract: 'your-contract-address',
  
  // Network settings
  timeout: 30000,
  retries: 3,
  
  // KNIRV services
  serviceURLs: {
    knirvchain: 'https://chain.knirv.network',
    knirvnexus: 'https://nexus.knirv.network',
    knirvoracle: 'https://root.knirv.network',
    knirvgraph: 'https://graph.knirv.network',
  },
  
  // Debugging
  debug: true,
  verbose: true,
});
```

## API Reference

### Economics Service

```typescript
// Skills
await client.economics.invokeSkill(request: SkillInvocationRequest): Promise<SkillInvocationResponse>

// LLM
await client.economics.registerLLM(request: LLMRegistrationRequest): Promise<LLMRegistrationResponse>

// Validation
await client.economics.processValidationReward(request: ValidationRewardRequest): Promise<ValidationRewardResponse>

// Fees
await client.economics.calculateNetworkFees(request: NetworkFeesRequest): Promise<NetworkFeesResponse>

// Metrics
await client.economics.getMetrics(): Promise<EconomicMetrics>
await client.economics.getServiceMetrics(serviceName: string): Promise<ServiceEconomics>

// Transactions
await client.economics.getTransaction(transactionId: string): Promise<Transaction>
await client.economics.listTransactions(options?: ListOptions): Promise<PaginatedResponse<Transaction>>

// Burn
await client.economics.getBurnHistory(limit?: number): Promise<PaginatedResponse<BurnEvent>>
await client.economics.getTotalBurned(): Promise<{ total_burned: string; timestamp: string }>

// Rules
await client.economics.getEconomicRules(): Promise<EconomicRules>
await client.economics.updateEconomicRules(rules: EconomicRules): Promise<EconomicRules>

// Convenience methods
await client.economics.checkSkillInvocationBalance(userId: string, skillId: string, amount: string): Promise<boolean>
await client.economics.getUserEconomicsSummary(userId: string): Promise<UserEconomicsSummary>
await client.economics.getNetworkStatistics(): Promise<NetworkStatistics>
```

### Health Service

```typescript
await client.health.checkEconomicsHealth(): Promise<HealthCheckResponse>
await client.health.checkGatewayHealth(): Promise<HealthCheckResponse>
await client.health.checkAllServices(): Promise<Record<string, HealthCheckResponse | null>>
await client.health.getSystemHealth(): Promise<SystemHealthResponse>
await client.health.waitForService(serviceName: string, options?: WaitOptions): Promise<boolean>
await client.health.getDetailedHealth(): Promise<DetailedHealthResponse>
```

### Integration Service

```typescript
await client.integration.getStatus(): Promise<IntegrationStatus>
await client.integration.testConnectivity(): Promise<Record<string, ConnectivityStatus>>
await client.integration.getMetrics(): Promise<IntegrationMetrics>
await client.integration.getComponentDetails(componentName: string): Promise<ComponentDetails>
await client.integration.triggerSync(): Promise<SyncResult>
```

## Error Handling

The SDK provides structured error handling with custom error types:

```typescript
import { EconomicsServiceError, GatewayServiceError } from '@knirv/gateway-sdk';

try {
  const result = await client.economics.invokeSkill(request);
} catch (error) {
  if (error instanceof EconomicsServiceError) {
    console.error('Economics service error:', error.message);
    console.error('Status code:', error.statusCode);
  } else if (error instanceof GatewayServiceError) {
    console.error('Gateway service error:', error.message);
  } else {
    console.error('General error:', error.message);
  }
}
```

## Real-time Monitoring

```typescript
// Monitor system health
const healthMonitor = setInterval(async () => {
  try {
    const systemHealth = await client.health.getSystemHealth();
    console.log(`System Status: ${systemHealth.status}`);
    
    if (systemHealth.status === 'unhealthy') {
      const detailedHealth = await client.health.getDetailedHealth();
      console.log('Detailed health:', detailedHealth);
    }
  } catch (error) {
    console.error('Health monitoring error:', error.message);
  }
}, 30000);

// Stop monitoring
clearInterval(healthMonitor);
```

## Examples

See the [examples](examples/) directory for comprehensive usage examples:

- [Basic Usage](examples/index.ts) - Basic client setup and operations
- [Advanced Usage](examples/index.ts) - Complex workflows and error handling
- [Real-time Monitoring](examples/index.ts) - Health monitoring and system status

## TypeScript Support

The SDK is written in TypeScript and provides full type definitions:

```typescript
import {
  KNIRVGatewayClient,
  SkillInvocationRequest,
  EconomicMetrics,
  HealthCheckResponse,
} from '@knirv/gateway-sdk';

const client = new KNIRVGatewayClient();

// Full type safety
const request: SkillInvocationRequest = {
  user_id: 'user123',
  skill_id: 'skill456',
  amount: '100000',
};

const response = await client.economics.invokeSkill(request);
// response is fully typed as SkillInvocationResponse
```

## Testing

```bash
npm test
# or
yarn test
# or
pnpm test
```

## Building

```bash
npm run build
# or
yarn build
# or
pnpm build
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
- Full TypeScript support
- Comprehensive examples and documentation
