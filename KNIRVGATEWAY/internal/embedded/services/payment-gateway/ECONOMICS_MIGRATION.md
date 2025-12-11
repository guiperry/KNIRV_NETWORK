# Economics Engine Migration Guide

## Overview

The KNIRV Economics Engine has been successfully migrated from Go to JavaScript/Node.js and integrated directly into the payment-gateway service. This eliminates the need for a separate Go economics service and provides a unified API for all payment and economics operations.

## What Changed

### Before Migration
- **Separate Services**: Economics engine ran as a standalone Go service on port 8090
- **Separate Deployment**: Required running both `payment-gateway` and `economics_engine` services
- **Cross-Service Communication**: HTTP calls between services
- **Language Barrier**: Go backend with Node.js frontend

### After Migration
- **Unified Service**: Economics engine integrated into payment-gateway
- **Single Deployment**: Only need to run `payment-gateway` service
- **Direct Integration**: In-process function calls, no HTTP overhead
- **Consistent Stack**: Pure JavaScript/Node.js implementation

## Architecture

```
payment-gateway/
├── server.js                          # Main server with economics integration
├── lib/
│   └── economics/
│       ├── index.js                   # Main economics module export
│       ├── TokenEconomics.js          # Core economics orchestrator
│       ├── EconomicRules.js           # Economic rules configuration
│       ├── TransactionPool.js         # Transaction management
│       ├── RewardCalculator.js        # Reward calculations
│       ├── BurnTracker.js             # Token burn tracking
│       ├── EconomicMetrics.js         # Network metrics
│       ├── EconomicsIntegration.js    # Component integration
│       ├── EconomicsAPI.js            # Public API routes
│       └── AdminAPI.js                # Admin routes for webgui
└── package.json                       # Updated dependencies
```

## API Endpoints

### Public Economics API (`/api/economics/`)

All economics endpoints are now available under `/api/economics/`:

#### Economic Operations
- `POST /api/economics/skill/invoke` - Process skill invocation
- `POST /api/economics/llm/register` - Process LLM registration
- `POST /api/economics/validation/reward` - Process validation reward
- `POST /api/economics/fees/calculate` - Calculate network fees

#### Data Retrieval
- `GET /api/economics/metrics` - Get economic metrics
- `GET /api/economics/transaction/:id` - Get transaction details
- `GET /api/economics/transactions` - Get transaction list
- `GET /api/economics/burn/history` - Get burn event history
- `GET /api/economics/burn/total` - Get total burned amount

#### Configuration
- `GET /api/economics/rules` - Get economic rules
- `PUT /api/economics/rules` - Update economic rules
- `GET /api/economics/service/:service/metrics` - Get service-specific metrics

#### System
- `GET /api/economics/health` - Health check
- `GET /api/economics/info` - Service information
- `GET /api/economics/integration/status` - Integration status

### Admin API (`/api/admin/economics/`)

Admin endpoints for webgui control (should be protected with authentication):

#### Rules Management
- `GET /api/admin/economics/rules` - Get economic rules
- `PUT /api/admin/economics/rules` - Update economic rules
- `POST /api/admin/economics/rules/reset` - Reset to default rules

#### System Control
- `POST /api/admin/economics/system/start` - Start economics system
- `POST /api/admin/economics/system/stop` - Stop economics system
- `GET /api/admin/economics/system/status` - Get system status

#### Integration Management
- `GET /api/admin/economics/integration/status` - Get integration status
- `PUT /api/admin/economics/integration/component/:component` - Update component URL
- `POST /api/admin/economics/integration/sync` - Force sync with all components

#### Metrics Management
- `GET /api/admin/economics/metrics/summary` - Get metrics summary
- `GET /api/admin/economics/metrics/service/:service` - Get service metrics
- `POST /api/admin/economics/metrics/reset` - Reset metrics

#### Transaction Management
- `GET /api/admin/economics/transactions/pool` - Get transaction pool stats
- `DELETE /api/admin/economics/transactions/:id` - Delete transaction
- `POST /api/admin/economics/transactions/cleanup` - Cleanup old transactions

#### Burn Management
- `GET /api/admin/economics/burn/stats` - Get burn statistics
- `GET /api/admin/economics/burn/total` - Get total burned

#### Reward Management
- `GET /api/admin/economics/rewards/leaderboard` - Get leaderboard
- `GET /api/admin/economics/rewards/performance/:userId` - Get user performance
- `PUT /api/admin/economics/rewards/multiplier/:name` - Update multiplier

#### Configuration
- `GET /api/admin/economics/config` - Get system configuration
- `PUT /api/admin/economics/config` - Update system configuration

#### Export/Import
- `GET /api/admin/economics/export/state` - Export system state
- `POST /api/admin/economics/import/state` - Import system state

## Environment Variables

Update your `.env` file with the following variables:

```bash
# Payment Gateway Port
PORT=3000

# Stripe Configuration
STRIPE_SECRET_KEY=your_stripe_key

# Coinbase Commerce
COINBASE_API_KEY=your_coinbase_key

# NRN Token Contract
NRN_CONTRACT=your_nrn_contract_address

# XION RPC
XION_RPC=https://rpc.xion-testnet-1.burnt.com:443

# KNIRV Component URLs
KNIRVCHAIN_URL=http://localhost:8080
KNIRVNEXUS_URL=http://localhost:8081
KNIRVORACLE_URL=http://localhost:8082
KNIRVGRAPH_URL=http://localhost:8083
```

## Running the Service

### Start the Unified Service

```bash
cd services/payment-gateway
npm install
npm start
```

The service will:
1. Initialize the Economics Engine
2. Start all background processors
3. Begin integration with KNIRV components
4. Mount all API routes
5. Start the Express server

### Stop the Old Go Service

The separate Go economics service is no longer needed:

```bash
# You can now stop the old economics_engine service
# No longer need to run: ./economics_engine/start-economics.sh
```

## Migration Steps

1. **Update Dependencies**
   ```bash
   cd services/payment-gateway
   npm install
   ```

2. **Configure Environment**
   - Copy `.env.example` to `.env`
   - Update environment variables as needed

3. **Start New Service**
   ```bash
   npm start
   ```

4. **Test Economics Endpoints**
   ```bash
   # Health check
   curl http://localhost:3000/health

   # Economics health check
   curl http://localhost:3000/api/economics/health

   # Get metrics
   curl http://localhost:3000/api/economics/metrics
   ```

5. **Stop Old Go Service**
   - The old `economics_engine` Go service can be shut down
   - Remove from startup scripts if applicable

## WebGUI Integration

The webgui service can now access economics admin controls through the admin API:

```javascript
// Example: Get system status
fetch('http://localhost:3000/api/admin/economics/system/status')
  .then(res => res.json())
  .then(data => console.log(data));

// Example: Update economic rules
fetch('http://localhost:3000/api/admin/economics/rules', {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    skillInvocationCost: '150000',
    llmRegistrationFee: '1500000',
    validationReward: '60000'
  })
})
  .then(res => res.json())
  .then(data => console.log(data));
```

## Feature Parity

All functionality from the Go version has been preserved:

✅ **Token Economics**
- Skill invocation processing
- LLM registration fees
- Validation rewards
- Network fee calculation

✅ **Transaction Management**
- Transaction pool with pending/confirmed states
- Automatic transaction processing
- Transaction history and queries

✅ **Burn Tracking**
- Burn event recording
- Total burned calculations
- Burn history with filtering
- Burn statistics by purpose

✅ **Reward Calculations**
- Performance-based multipliers
- User performance metrics
- Leaderboard functionality
- Customizable reward rules

✅ **Economic Metrics**
- Total supply tracking
- Circulating supply
- Token burn metrics
- Service-specific metrics
- Network utilization

✅ **Component Integration**
- KNIRVCHAIN integration
- KNIRVNEXUS integration
- KNIRVORACLE integration
- KNIRVGRAPH integration
- Automatic event syncing

## Performance Improvements

- **Faster Response Times**: No cross-service HTTP calls
- **Lower Memory Usage**: Single Node.js process
- **Simplified Deployment**: One service instead of two
- **Better Error Handling**: Unified error management

## Testing

Test the migration with these commands:

```bash
# 1. Health check
curl http://localhost:3000/health

# 2. Economics health check
curl http://localhost:3000/api/economics/health

# 3. Get metrics
curl http://localhost:3000/api/economics/metrics

# 4. Process skill invocation
curl -X POST http://localhost:3000/api/economics/skill/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "user123",
    "skillId": "skill456",
    "amount": "100000"
  }'

# 5. Get economic rules
curl http://localhost:3000/api/economics/rules

# 6. Get burn history
curl http://localhost:3000/api/economics/burn/history?limit=10

# 7. Get total burned
curl http://localhost:3000/api/economics/burn/total

# 8. Calculate network fees
curl -X POST http://localhost:3000/api/economics/fees/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "gasUsed": 21000,
    "priority": "medium"
  }'
```

## Rollback Plan

If needed, you can rollback to the Go version:

1. Stop the payment-gateway service
2. Restart the old economics_engine Go service
3. Update client code to point back to port 8090

The Go code is still available in `/economics_engine/` for reference.

## Next Steps

1. ✅ Economics engine migrated and integrated
2. ✅ Admin API routes created for webgui
3. ⏳ Add authentication/authorization to admin routes
4. ⏳ Implement database persistence layer
5. ⏳ Add comprehensive test suite
6. ⏳ Remove old Go economics_engine directory

## Support

For issues or questions:
- Check the logs: `npm start` shows detailed startup and runtime logs
- Health endpoint: `http://localhost:3000/health`
- Economics health: `http://localhost:3000/api/economics/health`

## License

Part of the KNIRV Network project.
