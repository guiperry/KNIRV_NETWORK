# Network Configuration Troubleshooting Guide

## Overview
This guide helps troubleshoot network configuration issues in KNIRVCONTROLLER.

## Supported Networks

### XION Testnet
- **Chain ID**: `xion-testnet-1`
- **RPC URL**: `https://rpc.xion-testnet-1.burnt.com`
- **REST URL**: `https://api.xion-testnet-1.burnt.com`
- **Explorer**: `https://explorer.xion-testnet-1.burnt.com`

### XION Mainnet
- **Chain ID**: `xion-mainnet-1`
- **RPC URL**: `https://rpc.xion-mainnet-1.burnt.com`
- **REST URL**: `https://api.xion-mainnet-1.burnt.com`
- **Explorer**: `https://explorer.xion-mainnet-1.burnt.com`

### Local Development
- **Chain ID**: `local-dev`
- **RPC URL**: `http://localhost:26657`
- **REST URL**: `http://localhost:1317`

## Common Issues

### 1. Network Connection Failed

**Symptoms:**
- "Network connection failed" error
- Unable to load wallet balance
- Transaction failures

**Solutions:**
```typescript
// Check network connectivity
const testConnection = async (rpcUrl: string) => {
  try {
    const response = await fetch(`${rpcUrl}/status`);
    return response.ok;
  } catch (error) {
    console.error('Network test failed:', error);
    return false;
  }
};

// Test current network
const isConnected = await testConnection('https://rpc.xion-testnet-1.burnt.com');
```

**Troubleshooting Steps:**
1. Check internet connection
2. Verify RPC URL is correct
3. Try alternative RPC endpoints
4. Check firewall/proxy settings
5. Clear browser cache

### 2. Wrong Network Selected

**Symptoms:**
- Wallet shows different network
- Transactions fail with "wrong chain" error
- Balance shows as zero

**Solutions:**
```typescript
// Switch to correct network
const switchNetwork = async (chainId: string) => {
  if (window.keplr) {
    await window.keplr.enable(chainId);
  } else if (window.leap) {
    await window.leap.enable(chainId);
  }
};

// Switch to XION testnet
await switchNetwork('xion-testnet-1');
```

### 3. RPC Endpoint Issues

**Symptoms:**
- Slow transaction processing
- Intermittent connection failures
- Outdated blockchain data

**Solutions:**
```typescript
// Test multiple RPC endpoints
const rpcEndpoints = [
  'https://rpc.xion-testnet-1.burnt.com',
  'https://rpc-xion-testnet.cosmos.directory',
  'https://xion-testnet-rpc.polkachu.com'
];

const findBestRPC = async () => {
  for (const rpc of rpcEndpoints) {
    try {
      const start = Date.now();
      const response = await fetch(`${rpc}/status`);
      const latency = Date.now() - start;
      
      if (response.ok && latency < 2000) {
        return rpc;
      }
    } catch (error) {
      continue;
    }
  }
  throw new Error('No working RPC found');
};
```

### 4. Wallet Connection Issues

**Symptoms:**
- Wallet not detected
- Connection rejected
- Permission denied errors

**Solutions:**
```typescript
// Check wallet availability
const checkWalletSupport = () => {
  const wallets = {
    keplr: !!window.keplr,
    leap: !!window.leap,
    cosmostation: !!window.cosmostation
  };
  
  console.log('Available wallets:', wallets);
  return wallets;
};

// Request wallet permissions
const requestPermissions = async () => {
  try {
    if (window.keplr) {
      await window.keplr.enable('xion-testnet-1');
      await window.keplr.experimentalSuggestChain({
        chainId: 'xion-testnet-1',
        chainName: 'XION Testnet',
        rpc: 'https://rpc.xion-testnet-1.burnt.com',
        rest: 'https://api.xion-testnet-1.burnt.com',
        bip44: { coinType: 118 },
        bech32Config: {
          bech32PrefixAccAddr: 'xion',
          bech32PrefixAccPub: 'xionpub',
          bech32PrefixValAddr: 'xionvaloper',
          bech32PrefixValPub: 'xionvaloperpub',
          bech32PrefixConsAddr: 'xionvalcons',
          bech32PrefixConsPub: 'xionvalconspub'
        },
        currencies: [{
          coinDenom: 'XION',
          coinMinimalDenom: 'uxion',
          coinDecimals: 6
        }],
        feeCurrencies: [{
          coinDenom: 'XION',
          coinMinimalDenom: 'uxion',
          coinDecimals: 6
        }],
        stakeCurrency: {
          coinDenom: 'XION',
          coinMinimalDenom: 'uxion',
          coinDecimals: 6
        }
      });
    }
  } catch (error) {
    console.error('Wallet permission error:', error);
  }
};
```

## Configuration Files

### Environment Variables
```bash
# .env.local
VITE_XION_TESTNET_RPC=https://rpc.xion-testnet-1.burnt.com
VITE_XION_TESTNET_REST=https://api.xion-testnet-1.burnt.com
VITE_XION_MAINNET_RPC=https://rpc.xion-mainnet-1.burnt.com
VITE_XION_MAINNET_REST=https://api.xion-mainnet-1.burnt.com
VITE_DEFAULT_NETWORK=xion-testnet-1
```

### Network Configuration
```typescript
// src/config/networks.ts
export const networks = {
  'xion-testnet-1': {
    chainId: 'xion-testnet-1',
    chainName: 'XION Testnet',
    rpc: process.env.VITE_XION_TESTNET_RPC,
    rest: process.env.VITE_XION_TESTNET_REST,
    bech32Prefix: 'xion',
    coinType: 118,
    gasPrice: '0.025uxion'
  },
  'xion-mainnet-1': {
    chainId: 'xion-mainnet-1',
    chainName: 'XION Mainnet',
    rpc: process.env.VITE_XION_MAINNET_RPC,
    rest: process.env.VITE_XION_MAINNET_REST,
    bech32Prefix: 'xion',
    coinType: 118,
    gasPrice: '0.025uxion'
  }
};
```

## Testing Network Configuration

### Manual Testing
```typescript
// Test network connectivity
const testNetwork = async (networkConfig) => {
  console.log(`Testing network: ${networkConfig.chainName}`);
  
  // Test RPC
  try {
    const rpcResponse = await fetch(`${networkConfig.rpc}/status`);
    console.log('RPC Status:', rpcResponse.ok ? 'OK' : 'Failed');
  } catch (error) {
    console.error('RPC Error:', error);
  }
  
  // Test REST
  try {
    const restResponse = await fetch(`${networkConfig.rest}/cosmos/base/tendermint/v1beta1/node_info`);
    console.log('REST Status:', restResponse.ok ? 'OK' : 'Failed');
  } catch (error) {
    console.error('REST Error:', error);
  }
};
```

### Automated Testing
```typescript
// tests/network-config.test.ts
import { networks } from '../src/config/networks';

describe('Network Configuration', () => {
  Object.entries(networks).forEach(([chainId, config]) => {
    test(`${chainId} should have valid configuration`, () => {
      expect(config.chainId).toBe(chainId);
      expect(config.rpc).toMatch(/^https?:\/\//);
      expect(config.rest).toMatch(/^https?:\/\//);
      expect(config.bech32Prefix).toBeTruthy();
    });
    
    test(`${chainId} RPC should be accessible`, async () => {
      const response = await fetch(`${config.rpc}/status`);
      expect(response.ok).toBe(true);
    }, 10000);
  });
});
```

## Performance Optimization

### Connection Pooling
```typescript
class NetworkManager {
  private connections = new Map();
  
  async getConnection(chainId: string) {
    if (!this.connections.has(chainId)) {
      const config = networks[chainId];
      const connection = await this.createConnection(config);
      this.connections.set(chainId, connection);
    }
    return this.connections.get(chainId);
  }
  
  private async createConnection(config) {
    // Create optimized connection with retry logic
    return new CosmosClient(config);
  }
}
```

### Caching
```typescript
// Cache network status
const networkStatusCache = new Map();

const getNetworkStatus = async (chainId: string) => {
  const cacheKey = `status-${chainId}`;
  const cached = networkStatusCache.get(cacheKey);
  
  if (cached && Date.now() - cached.timestamp < 30000) {
    return cached.data;
  }
  
  const status = await fetchNetworkStatus(chainId);
  networkStatusCache.set(cacheKey, {
    data: status,
    timestamp: Date.now()
  });
  
  return status;
};
```

## Security Considerations

### RPC Security
- Always use HTTPS endpoints
- Validate SSL certificates
- Implement request timeouts
- Rate limit requests

### Wallet Security
- Never store private keys
- Validate all transactions
- Use secure communication channels
- Implement proper error handling

## Monitoring

### Health Checks
```typescript
const healthCheck = async () => {
  const results = {};
  
  for (const [chainId, config] of Object.entries(networks)) {
    try {
      const start = Date.now();
      const response = await fetch(`${config.rpc}/health`);
      const latency = Date.now() - start;
      
      results[chainId] = {
        status: response.ok ? 'healthy' : 'unhealthy',
        latency,
        timestamp: new Date().toISOString()
      };
    } catch (error) {
      results[chainId] = {
        status: 'error',
        error: error.message,
        timestamp: new Date().toISOString()
      };
    }
  }
  
  return results;
};
```

### Logging
```typescript
// Network operation logging
const logNetworkOperation = (operation: string, chainId: string, result: any) => {
  console.log({
    timestamp: new Date().toISOString(),
    operation,
    chainId,
    success: !result.error,
    latency: result.latency,
    error: result.error
  });
};
```

## Support

For network configuration issues:
1. Check this troubleshooting guide
2. Verify network status on explorer
3. Test with different RPC endpoints
4. Contact network operators if persistent issues
5. Report bugs to development team
