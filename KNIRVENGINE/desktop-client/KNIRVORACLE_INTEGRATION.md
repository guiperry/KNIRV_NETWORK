# KNIRVORACLE Integration Documentation

## Overview

This document describes the comprehensive integration between KNIRVENGINE/desktop-client and KNIRVORACLE, providing seamless connectivity for agent minting, capability registration, wallet operations, and network interactions.

## Architecture

The integration consists of several key components:

### 1. Service Layer
- **Go Backend**: `services/knirvoracleService.go` - Core KNIRVORACLE service implementation
- **TypeScript Frontend**: `gui/src/services/knirvoracleService.ts` - Frontend service wrapper
- **Type Definitions**: `gui/src/types/knirvoracle.ts` - Comprehensive TypeScript types

### 2. Integration Points
- **Agent Builder**: Automatic agent minting during successful builds
- **MCP Capability Manager**: Capability registration with KNIRVORACLE
- **Wallet Service**: Real balance queries, transactions, and faucet operations
- **API Handlers**: Backend wallet handlers with KNIRVORACLE fallback

## Features

### Agent Minting
- Automatic NFT minting when agents are successfully built
- Comprehensive metadata storage including agent type, model, and configuration
- Asynchronous processing to avoid blocking the build process
- Transaction tracking and NFT ID storage

### Capability Registration
- MCP server to capability transformation
- Real-time registration with KNIRVORACLE during transformation
- Schema validation and gas fee calculation
- Status tracking (registering, available, registration_failed)

### Wallet Operations
- Real-time balance queries from KNIRVORACLE
- Faucet requests for testnet tokens
- Transaction sending through KNIRVORACLE
- Fallback to local API when KNIRVORACLE is unavailable

### Type Safety
- Comprehensive TypeScript interfaces for all API interactions
- Compile-time type checking for requests and responses
- Error handling with custom KNIRVOracleError types
- Validation helpers for addresses and amounts

## Configuration

### Environment Variables

```bash
# KNIRVORACLE service URL
KNIRVORACLE_URL=http://localhost:8080

# API key for authenticated requests (optional)
KNIRVORACLE_API_KEY=your-api-key-here
```

### Frontend Configuration

```typescript
// In your React app
const knirvoracleService = new KNIRVOracleService({
  baseURL: process.env.REACT_APP_KNIRVORACLE_URL || 'http://localhost:8080',
  apiKey: process.env.REACT_APP_KNIRVORACLE_API_KEY,
  timeout: 30000,
});
```

### Backend Configuration

```go
// In your Go application
config := services.KNIRVOracleConfig{
    BaseURL: os.Getenv("KNIRVORACLE_URL"),
    APIKey:  os.Getenv("KNIRVORACLE_API_KEY"),
    Timeout: 30,
}
service := services.NewKNIRVOracleService(config)
```

## API Endpoints

### Agent Operations
- `POST /agent/mint` - Mint new agent as NFT
- `GET /agent/capabilities/{id}` - Get agent capabilities

### Capability Operations
- `POST /wallet/mcp/create_register_capability` - Register capability
- `POST /wallet/mcp/create_invoke_capability` - Invoke capability

### Wallet Operations
- `GET /balance/{address}` - Get wallet balance
- `POST /api/mint/nrv` - Request faucet tokens
- `POST /transactions` - Send transaction

### System Operations
- `GET /health` - Health check
- `GET /api/treasury/status` - Treasury status

## Usage Examples

### Agent Minting

```typescript
const mintRequest: AgentMintRequest = {
  agent_id: 'agent-123',
  name: 'My AI Agent',
  description: 'An intelligent assistant agent',
  owner: 'user-wallet-address',
  metadata: {
    agent_type: 'assistant',
    model: 'gpt-4',
    capabilities: ['search', 'code_execution']
  }
};

const response = await knirvoracleService.mintAgent(mintRequest);
console.log('Agent minted:', response.agent_nft_id);
```

### Capability Registration

```typescript
const capabilityRequest: CapabilityRegistrationRequest = {
  name: 'Web Search Capability',
  type: 'mcp_capability',
  description: 'Provides web search functionality',
  schema: {
    type: 'object',
    properties: {
      query: { type: 'string', description: 'Search query' },
      limit: { type: 'number', description: 'Result limit' }
    },
    required: ['query']
  },
  owner: 'user-wallet-address',
  gas_fee_nrn: 1000
};

const response = await knirvoracleService.registerCapability(capabilityRequest);
console.log('Capability registered:', response.capability_id);
```

### Wallet Operations

```typescript
// Get balance
const balance = await knirvoracleService.getWalletBalance('0x...');
console.log('NRN Balance:', balance.nrn_balance);

// Request faucet
const faucetResponse = await knirvoracleService.requestFaucet({
  address: '0x...',
  amount: '100',
  reason: 'Testing'
});

// Send transaction
const txResponse = await knirvoracleService.sendTransaction({
  from: '0x...',
  to: '0x...',
  amount: '50',
  token: 'NRN'
});
```

## Error Handling

The integration includes comprehensive error handling:

```typescript
try {
  const response = await knirvoracleService.mintAgent(request);
} catch (error) {
  if (isKNIRVOracleError(error)) {
    console.error('KNIRVORACLE Error:', error.code, error.message);
    console.error('Response:', error.response);
  } else {
    console.error('Network Error:', error.message);
  }
}
```

## Testing

### Running Integration Tests

```bash
# Go tests
cd KNIRVENGINE/desktop-client
export KNIRVORACLE_URL=http://localhost:8080
go test ./tests/... -v

# TypeScript tests
cd gui
npm test knirvoracleIntegration.test.ts
```

### Test Coverage
- Agent minting functionality
- Capability registration
- Wallet operations (balance, faucet, transactions)
- Error handling scenarios
- Type safety validation
- Service configuration

## Monitoring and Debugging

### Health Checks
The integration includes automatic health checking:

```typescript
const isHealthy = await knirvoracleService.healthCheck();
if (!isHealthy) {
  console.warn('KNIRVORACLE service is unavailable');
}
```

### Logging
All KNIRVORACLE operations are logged with appropriate detail levels:
- Info: Successful operations
- Warn: Fallback to local API
- Error: Failed operations with full error details

## Security Considerations

1. **API Key Management**: Store API keys securely in environment variables
2. **Address Validation**: All wallet addresses are validated before API calls
3. **Amount Validation**: Transaction amounts are validated for format and range
4. **Error Information**: Sensitive information is not exposed in error messages
5. **Timeout Configuration**: All requests have appropriate timeouts to prevent hanging

## Performance Optimization

1. **Asynchronous Operations**: Agent minting runs asynchronously to avoid blocking
2. **Fallback Mechanisms**: Local API fallback when KNIRVORACLE is unavailable
3. **Connection Pooling**: HTTP client reuse for efficient connections
4. **Timeout Management**: Appropriate timeouts for different operation types
5. **Error Recovery**: Graceful degradation when services are unavailable

## Troubleshooting

### Common Issues

1. **Connection Refused**: Check KNIRVORACLE_URL environment variable
2. **Authentication Failed**: Verify KNIRVORACLE_API_KEY is correct
3. **Timeout Errors**: Increase timeout configuration for slow networks
4. **Invalid Address**: Ensure wallet addresses are properly formatted
5. **Insufficient Balance**: Check wallet has sufficient NRN for gas fees

### Debug Mode

Enable debug logging by setting:
```bash
export DEBUG=knirvoracle:*
```

## Future Enhancements

1. **Real-time Events**: WebSocket integration for real-time updates
2. **Batch Operations**: Support for batch agent minting and capability registration
3. **Advanced Analytics**: Integration with KNIRVORACLE analytics APIs
4. **Caching Layer**: Local caching for frequently accessed data
5. **Retry Logic**: Automatic retry with exponential backoff for failed operations

## Contributing

When contributing to the KNIRVORACLE integration:

1. Ensure all new functionality includes comprehensive tests
2. Update TypeScript types for any new API endpoints
3. Add appropriate error handling and logging
4. Update this documentation for any new features
5. Follow the existing code style and patterns
