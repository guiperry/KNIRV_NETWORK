# KNIRV Unified SDK for TypeScript/JavaScript

✅ **Production Ready** - Comprehensive SDK providing access to all KNIRV Network services.

## Overview

The KNIRV Unified SDK combines all KNIRV Network functionality into a single, easy-to-use package:

- **Badge System**: Skills, Capabilities, and Properties badges
- **XION Integration**: Meta Accounts, Treasury Contracts, Gasless transactions
- **NRN Token Management**: Minting, treasury operations, faucet integration
- **KNIRVNEXUS DVE**: Development environment management
- **KNIRVORACLE Services**: Treasury management, badge validation
- **KNIRVCONTROLLER**: Agent management, skill invocation, workflows
- **KNIRVROUTER**: Proof-of-connectivity, network routing, NRV management
- **Network Configuration**: Environment switching capabilities
- **Health Monitoring**: Service status and network health
- **Factuality Slice**: Content verification and factual accuracy checking
- **Wallet Integration**: Compatible with existing wallet implementations

## Installation

```bash
npm install @knirv/sdk-unified
```

## Quick Start

```typescript
import { KNIRVClient, createProductionClient } from '@knirv/sdk-unified';

// Create client with default configuration
const client = new KNIRVClient();

// Or use convenience functions
const prodClient = createProductionClient('your-api-key');
const testnetClient = createTestnetClient('your-api-key');
const localClient = createLocalClient();

// Custom configuration
const client = new KNIRVClient({
  network: {
    environment: 'public-production',
    customEndpoints: {
      controller: 'https://custom-controller.example.com'
    }
  },
  auth: {
    apiKey: 'your-api-key',
    xionMetaAccount: {
      address: 'xion1...',
      email: 'user@example.com',
      passkeyEnabled: true
    }
  },
  wallet: {
    provider: 'adena',
    chainId: 'gno-1',
    enableGasless: true
  }
});
```

## Core Features

### Badge System

```typescript
// Get agent badges
const badges = await client.badges.getAgentBadges('agent-123');
const skillBadges = await client.badges.getSkillBadges('agent-123');
const capabilityBadges = await client.badges.getCapabilityBadges('agent-123');

// Validate and issue badges
const validation = await client.badges.validateBadge('badge-456');
const newBadge = await client.badges.issueBadge({
  type: 'skill',
  name: 'JavaScript Expert',
  description: 'Advanced JavaScript programming skills'
});
```

### KNIRVNEXUS DVE (Development Environments)

```typescript
// Manage development environments
const environments = await client.dve.listEnvironments();
const newEnv = await client.dve.createEnvironment({
  name: 'My Dev Environment',
  type: 'development',
  resources: {
    cpu: '2 cores',
    memory: '4GB',
    storage: '20GB'
  }
});

// Start and manage sessions
const session = await client.dve.startSession(newEnv.id);
const sessionInfo = await client.dve.getSession(session.id);
```

### Treasury & NRN Token Management

```typescript
// Get NRN token information
const nrnInfo = await client.treasury.getNRNTokenInfo();
const balance = await client.treasury.getTreasuryBalance();

// Request faucet tokens
const faucetRequest = await client.treasury.requestFaucet(
  'knirv1user123...',
  '100'
);

// View treasury operations
const operations = await client.treasury.getTreasuryOperations(10);
```

### Agent Management

```typescript
// List and manage agents
const agents = await client.agents.listAgents();
const agent = await client.agents.getAgent('agent-123');

// Create new agent
const newAgent = await client.agents.createAgent({
  name: 'My AI Assistant',
  type: 'cognitive',
  capabilities: ['text-processing', 'analysis']
});

// Invoke skills
const skillResult = await client.agents.invokeSkill({
  agentId: 'agent-123',
  skillId: 'text-analysis',
  userId: 'user-456',
  amount: '10',
  parameters: { text: 'Analyze this content' }
});
```

### Network Monitoring & Health

```typescript
// Check network health
const networkHealth = await client.health.getNetworkHealth();
const serviceHealth = await client.health.getServiceHealth('controller');

// Ping specific services
const pingResult = await client.health.pingService('router');

// Get connectivity proofs
const proofs = await client.network.getConnectivityProofs();
const routes = await client.network.getNetworkRoutes();
```

### Factuality Verification

```typescript
// Verify content factuality
const verification = await client.factuality.verifyContent(
  'The Earth is round',
  'scientific'
);

// Get verification history
const history = await client.factuality.getVerificationHistory(20);
const sliceInfo = await client.factuality.getSliceInfo();
```

### Network Configuration

```typescript
// Get current network info
const networkInfo = client.getNetworkInfo();

// Switch networks
await client.switchNetwork('public-testnet');

// Check connection status
const isConnected = await client.isConnected();
```

## Network Environments

The SDK supports multiple network environments:

- **public-production**: Main KNIRV Network
- **public-testnet**: Public testnet for development
- **local-testnet**: Local development environment
- **local-production**: Local production-like environment

## Error Handling

```typescript
try {
  const result = await client.agents.invokeSkill({
    agentId: 'agent-123',
    skillId: 'invalid-skill',
    userId: 'user-456',
    amount: '10'
  });
} catch (error) {
  console.error('Skill invocation failed:', error.message);
}
```

## TypeScript Support

The SDK is written in TypeScript and provides full type definitions:

```typescript
import type {
  Badge,
  SkillBadge,
  Agent,
  DVEEnvironment,
  NetworkHealth,
  FactualityVerification
} from '@knirv/sdk-unified';
```

## Contributing

See the main [KNIRV SDK repository](../../README.md) for contribution guidelines.

## License

MIT License - see [LICENSE](../../LICENSE) for details.

## Support

- Documentation: [https://docs.knirv.com](https://docs.knirv.com)
- Issues: [GitHub Issues](https://github.com/knirv/sdk/issues)
- Discord: [KNIRV Community](https://discord.gg/knirv)
