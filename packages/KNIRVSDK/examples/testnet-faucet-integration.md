# Testnet Faucet Integration Examples

This document provides comprehensive examples of how to use the new KNIRVTESTNET NRV Faucet integration across all KNIRV SDK languages and CLI tools.

## Overview

The testnet faucet integration provides:
- **Token Requests**: Request NRV tokens for testing
- **Status Monitoring**: Check faucet availability and limits
- **History Tracking**: View request history for addresses
- **Health Checks**: Monitor faucet service health
- **Automatic Retries**: Built-in retry logic with exponential backoff
- **Rate Limit Handling**: Intelligent handling of rate limits

## CLI Usage

### Basic Faucet Request

```bash
# Request 1000 NRV tokens (default amount)
./knirv economics faucet --wallet my-wallet

# Request specific amount
./knirv economics faucet 5000 --wallet my-wallet

# Request with reason and custom retry settings
./knirv economics faucet 2000 --wallet my-wallet \
  --reason "Testing new feature" \
  --max-retries 5
```

### Faucet Status and Monitoring

```bash
# Check faucet status
./knirv economics faucet-status

# Check faucet health
./knirv economics faucet-health

# View request history
./knirv economics faucet-history --wallet my-wallet --limit 20

# View history for specific address
./knirv economics faucet-history knirv1abc123... --limit 10
```

### Example CLI Output

```
$ ./knirv economics faucet 1000 --wallet test-wallet --reason "Integration testing"

Requesting 1000 NRV tokens from testnet faucet...
Address: knirv1abc123def456...
Reason: Integration testing
Max retries: 3

✅ Testnet faucet request successful!
Transaction Hash: 0x789abc123def456...
Request ID: req_1234567890
Address: knirv1abc123def456...
Amount: 1000 NRV
Status: confirmed
Timestamp: 2024-01-15T10:30:45Z
```

## Go SDK Usage

### Basic Integration

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/guiperry/KNIRVCHAIN-CLI/config"
    "github.com/guiperry/KNIRVCHAIN-CLI/core"
    "github.com/sirupsen/logrus"
)

func main() {
    // Load configuration
    cfg, err := config.LoadConfig("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    logger := logrus.New()
    
    // Create NRN token manager with faucet integration
    knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, logger)
    nrnManager := core.NewNRNTokenManager(&cfg.KNIRV.Wallet, knirvRootClient, logger)

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    // Connect to services
    if err := knirvRootClient.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer knirvRootClient.Disconnect()

    // Request tokens from faucet
    address := "knirv1abc123def456..."
    tx, err := nrnManager.RequestFromFaucet(ctx, address, "1000")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Faucet request successful!\n")
    fmt.Printf("TX Hash: %s\n", tx.Hash)
    fmt.Printf("Amount: %s NRV\n", tx.Amount.String())
}
```

### Advanced Faucet Operations

```go
// Request with retry logic
tx, err := nrnManager.RequestFromFaucetWithRetry(
    ctx, 
    address, 
    "2000", 
    "Testing advanced features", 
    5, // max retries
)

// Get faucet status
status, err := nrnManager.GetFaucetStatus(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Faucet enabled: %v\n", status.FaucetEnabled)
fmt.Printf("Current balance: %d NRV\n", status.CurrentBalance)
fmt.Printf("Daily limit: %d NRV\n", status.DailyLimit)

// Get request history
history, err := nrnManager.GetFaucetHistory(ctx, address, 10)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total requests: %d\n", history.TotalRequests)
for _, entry := range history.History {
    fmt.Printf("- %s: %d NRV (%s)\n", 
        entry.Timestamp, entry.Amount, entry.Status)
}

// Check faucet health
health, err := nrnManager.CheckFaucetHealth(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Faucet health: %v\n", health["status"])
```

## TypeScript SDK Usage

### Basic Integration

```typescript
import { EconomicsService, defaultClientOptions } from '@knirv/gateway-sdk';

async function main() {
    // Initialize economics service with faucet integration
    const config = defaultClientOptions();
    const economics = new EconomicsService(config);

    try {
        // Request tokens from faucet
        const response = await economics.requestFromFaucet(
            'knirv1abc123def456...',
            1000,
            { reason: 'Testing TypeScript integration' }
        );

        console.log('Faucet request successful!');
        console.log('TX Hash:', response.tx_hash);
        console.log('Request ID:', response.request_id);
        console.log('Amount:', response.amount, 'NRV');

    } catch (error) {
        console.error('Faucet request failed:', error.message);
    }
}

main().catch(console.error);
```

### Advanced Operations

```typescript
import { FaucetService, createFaucetService } from '@knirv/gateway-sdk';

async function advancedFaucetOperations() {
    // Create dedicated faucet service
    const faucet = createFaucetService('http://localhost:10000', {
        debug: true,
        maxRetries: 5,
        timeout: 30000
    });

    const address = 'knirv1abc123def456...';

    try {
        // Get faucet status
        const status = await faucet.getStatus();
        console.log('Faucet Status:');
        console.log('- Enabled:', status.faucet_enabled);
        console.log('- Balance:', status.current_balance, 'NRV');
        console.log('- Daily Limit:', status.daily_limit, 'NRV');
        console.log('- Remaining Today:', status.remaining_today, 'NRV');

        // Request tokens with custom options
        const response = await faucet.requestTokens(address, 2000, {
            reason: 'Advanced testing',
            maxRetries: 3,
            useExponentialBackoff: true
        });

        console.log('Request successful:', response.success);

        // Get request history
        const history = await faucet.getHistory(address, 20);
        console.log(`\nRequest History (${history.total_requests} total):`);
        
        history.history.forEach((entry, index) => {
            console.log(`${index + 1}. ${entry.timestamp}: ${entry.amount} NRV (${entry.status})`);
            if (entry.reason) console.log(`   Reason: ${entry.reason}`);
        });

        // Check health
        const health = await faucet.checkHealth();
        console.log('\nFaucet Health:', health.status);

    } catch (error) {
        console.error('Operation failed:', error.message);
    }
}

advancedFaucetOperations().catch(console.error);
```

## Python SDK Usage

### Basic Integration

```python
import asyncio
from knirv_unified_sdk.faucet import create_faucet_client

async def main():
    # Create faucet client
    faucet = create_faucet_client(
        faucet_url='http://localhost:10000',
        debug=True,
        max_retries=3
    )

    address = 'knirv1abc123def456...'

    try:
        # Request tokens
        response = faucet.request_tokens(
            address=address,
            amount=1000,
            reason='Python SDK testing'
        )

        print('Faucet request successful!')
        print(f'TX Hash: {response.tx_hash}')
        print(f'Request ID: {response.request_id}')
        print(f'Amount: {response.amount} NRV')

        # Get status
        status = faucet.get_status()
        print(f'\nFaucet Status:')
        print(f'- Enabled: {status.faucet_enabled}')
        print(f'- Balance: {status.current_balance} NRV')
        print(f'- Daily Limit: {status.daily_limit} NRV')

        # Get history
        history = faucet.get_history(address, limit=10)
        print(f'\nRequest History ({history.total_requests} total):')
        
        for i, entry in enumerate(history.history, 1):
            print(f'{i}. {entry.timestamp}: {entry.amount} NRV ({entry.status})')
            if entry.reason:
                print(f'   Reason: {entry.reason}')

    except Exception as error:
        print(f'Error: {error}')

if __name__ == '__main__':
    asyncio.run(main())
```

### Async Usage

```python
import asyncio
from knirv_unified_sdk.faucet import create_faucet_client

async def async_faucet_operations():
    # Create async faucet client
    faucet = create_faucet_client(
        faucet_url='http://localhost:10000',
        async_client=True,
        debug=True
    )

    address = 'knirv1abc123def456...'

    try:
        # Async operations
        status_task = faucet.get_status()
        health_task = faucet.check_health()

        # Wait for both operations
        status, health = await asyncio.gather(status_task, health_task)

        print('Faucet Status:', status.faucet_enabled)
        print('Faucet Health:', health['status'])

        # Request tokens asynchronously
        response = await faucet.request_tokens(
            address=address,
            amount=1500,
            reason='Async testing'
        )

        print(f'Async request successful: {response.success}')

    except Exception as error:
        print(f'Async operation failed: {error}')

asyncio.run(async_faucet_operations())
```

## Configuration

### Environment Variables

```bash
# Testnet faucet configuration
export TESTNET_FAUCET_URL="http://localhost:10000"
export KNIRV_DEBUG="true"
export KNIRV_VERBOSE="true"

# For production testnet
export TESTNET_FAUCET_URL="https://testnet.knirv.com/api/faucet"
```

### Configuration Files

```yaml
# config.yaml
knirv:
  wallet:
    nrn:
      enabled: true
      faucet_url: "http://localhost:10000"
      max_retries: 3
      request_timeout: 30
      default_reason: "SDK request"
```

## Error Handling

### Common Error Scenarios

```typescript
try {
    const response = await faucet.requestTokens(address, amount);
} catch (error) {
    if (error.code === 'RATE_LIMITED') {
        console.log(`Rate limited. Retry after: ${error.retry_after}s`);
    } else if (error.code === 'INSUFFICIENT_BALANCE') {
        console.log('Faucet has insufficient balance');
    } else if (error.code === 'INVALID_ADDRESS') {
        console.log('Invalid address format');
    } else {
        console.log('Unknown error:', error.message);
    }
}
```

## Best Practices

1. **Use Appropriate Amounts**: Request reasonable amounts (100-5000 NRV)
2. **Provide Reasons**: Always include a reason for tracking
3. **Handle Rate Limits**: Implement proper retry logic
4. **Monitor Status**: Check faucet status before large operations
5. **Validate Addresses**: Ensure addresses are properly formatted
6. **Use Timeouts**: Set appropriate timeouts for requests
7. **Log Operations**: Enable debug logging for troubleshooting

## Integration Testing

```bash
# Test faucet integration
./knirv economics faucet-health
./knirv economics faucet-status
./knirv economics faucet 100 --wallet test-wallet --reason "Integration test"
./knirv economics faucet-history --wallet test-wallet
```

This comprehensive integration provides a robust, production-ready interface to the KNIRVTESTNET NRV Faucet across all KNIRV SDK languages and tools.
