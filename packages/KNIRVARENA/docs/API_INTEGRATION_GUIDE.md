# KNIRVARENA API Integration Guide

## Overview
This guide provides comprehensive information for integrating with the KNIRVARENA application APIs and services.

## Core Services

### 1. KnirvanaBridgeService
The main service for KNIRVANA game integration and NRV economy management.

```typescript
import { KnirvanaBridgeService } from '../services/KnirvanaBridgeService';

// Initialize the service
const bridgeService = new KnirvanaBridgeService();

// Start game session
await bridgeService.startGame();

// Deploy an agent (costs NRVs)
const deploymentResult = await bridgeService.deployAgent();

// Check NRV balance
const balance = bridgeService.getNRVBalance();

// Earn NRVs by solving errors
bridgeService.earnNRVs(50);

// Merge to collective network
await bridgeService.mergeToCollectiveNetwork();
```

### 2. RxDBService
Database management with encryption and migration support.

```typescript
import { RxDBService } from '../services/RxDBService';

// Initialize database
const dbService = new RxDBService();
await dbService.initialize();

// Store agent data
await dbService.storeAgent({
  id: 'agent-001',
  name: 'Analyzer Agent',
  type: 'analyze',
  capabilities: ['error-detection', 'code-analysis']
});

// Query agents
const agents = await dbService.getAgents();

// Update agent status
await dbService.updateAgentStatus('agent-001', 'deployed');
```

### 3. WalletIntegrationService
XION wallet and blockchain integration.

```typescript
import { WalletIntegrationService } from '../services/WalletIntegrationService';

// Initialize wallet service
const walletService = new WalletIntegrationService();

// Connect wallet
await walletService.connectWallet();

// Get wallet balance
const balance = await walletService.getBalance();

// Send transaction
const txResult = await walletService.sendTransaction({
  to: 'xion1...',
  amount: '100',
  denom: 'uxion'
});
```

## API Endpoints

### Agent Management
- `POST /api/agents` - Create new agent
- `GET /api/agents` - List all agents
- `GET /api/agents/:id` - Get agent details
- `PUT /api/agents/:id` - Update agent
- `DELETE /api/agents/:id` - Remove agent
- `POST /api/agents/:id/deploy` - Deploy agent

### Game Integration
- `POST /api/game/start` - Start game session
- `GET /api/game/status` - Get game status
- `POST /api/game/deploy-agent` - Deploy agent in game
- `GET /api/game/nrv-balance` - Get NRV balance
- `POST /api/game/merge-collective` - Merge to collective

### Network Management
- `GET /api/networks` - List available networks
- `POST /api/networks/switch` - Switch network
- `GET /api/networks/status` - Get network status
- `POST /api/networks/custom` - Add custom network

## Error Handling

All API responses follow this format:

```typescript
interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: any;
  };
}
```

### Common Error Codes
- `INSUFFICIENT_NRVS` - Not enough NRVs for operation
- `AGENT_NOT_FOUND` - Agent ID not found
- `NETWORK_ERROR` - Network connectivity issue
- `WALLET_NOT_CONNECTED` - Wallet not connected
- `INVALID_PARAMETERS` - Invalid request parameters

## Authentication

### Wallet-based Authentication
```typescript
// Connect wallet for authentication
await walletService.connectWallet();

// Get authentication token
const authToken = await walletService.getAuthToken();

// Use token in API calls
const response = await fetch('/api/agents', {
  headers: {
    'Authorization': `Bearer ${authToken}`,
    'Content-Type': 'application/json'
  }
});
```

## Rate Limiting

API endpoints are rate limited:
- Standard endpoints: 100 requests/minute
- Game operations: 10 requests/minute
- Wallet operations: 5 requests/minute

## WebSocket Events

Real-time updates via WebSocket:

```typescript
import { WebSocketService } from '../services/WebSocketService';

const wsService = new WebSocketService();

// Connect to WebSocket
await wsService.connect();

// Listen for events
wsService.on('agent-deployed', (data) => {
  console.log('Agent deployed:', data);
});

wsService.on('nrv-earned', (data) => {
  console.log('NRVs earned:', data.amount);
});

wsService.on('error-solved', (data) => {
  console.log('Error solved:', data);
});
```

## Testing

### Unit Testing
```typescript
import { KnirvanaBridgeService } from '../services/KnirvanaBridgeService';

describe('KnirvanaBridgeService', () => {
  let service: KnirvanaBridgeService;

  beforeEach(() => {
    service = new KnirvanaBridgeService();
  });

  test('should start game successfully', async () => {
    await service.startGame();
    expect(service.isGameActive()).toBe(true);
  });

  test('should deploy agent with sufficient NRVs', async () => {
    service.earnNRVs(100);
    const result = await service.deployAgent();
    expect(result).toBe(true);
  });
});
```

### Integration Testing
```typescript
import { test, expect } from '@playwright/test';

test('should complete agent deployment flow', async ({ page }) => {
  await page.goto('/');
  
  // Start game
  await page.click('[data-testid="start-game"]');
  
  // Deploy agent
  await page.click('[data-testid="deploy-agent"]');
  
  // Verify deployment
  await expect(page.locator('[data-testid="agent-deployed"]')).toBeVisible();
});
```

## Performance Optimization

### Caching
- Agent data cached for 5 minutes
- Network status cached for 30 seconds
- Game state cached in memory

### Lazy Loading
```typescript
// Lazy load heavy components
const GameVisualization = lazy(() => import('../components/KNIRVANAGameVisualization'));

// Use with Suspense
<Suspense fallback={<LoadingSpinner />}>
  <GameVisualization />
</Suspense>
```

### Memory Management
```typescript
// Clean up resources
useEffect(() => {
  return () => {
    bridgeService.cleanup();
    wsService.disconnect();
  };
}, []);
```

## Security Best Practices

1. **Never store private keys in localStorage**
2. **Validate all user inputs**
3. **Use HTTPS for all API calls**
4. **Implement proper CORS policies**
5. **Rate limit API endpoints**
6. **Sanitize error messages**

## Troubleshooting

### Common Issues

**Agent deployment fails**
- Check NRV balance
- Verify network connection
- Ensure wallet is connected

**Game not starting**
- Clear browser cache
- Check WebAssembly support
- Verify network selection

**Wallet connection issues**
- Install XION wallet extension
- Switch to correct network
- Clear wallet cache

### Debug Mode
```typescript
// Enable debug logging
localStorage.setItem('debug', 'true');

// View debug information
console.log(bridgeService.getDebugInfo());
```

## Support

For additional support:
- Check the troubleshooting guide
- Review error logs in browser console
- Contact development team with error details
