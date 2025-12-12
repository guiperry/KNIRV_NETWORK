# Quick Start Guide

## Get Running in 60 Seconds

### 1. Navigate to Service Directory

```bash
cd services/payment-oracle
```

### 2. Install Dependencies (if needed)

```bash
npm install
```

### 3. Configure Environment (optional)

```bash
# Copy example env file
cp .env.example .env

# Edit if needed (or use defaults)
# nano .env
```

### 4. Start the Service

```bash
npm start
```

You should see:

```
Initializing KNIRV Economics Engine...
Economics API routes mounted at /api/economics
Admin API routes mounted at /api/admin/economics
=====================================
KNIRV Payment Gateway & Economics Engine
=====================================
Server running on port 3000
Open http://localhost:3000 in your browser

Available endpoints:
  - Payment Gateway: /api/faucet/*, /api/create-*
  - Economics API: /api/economics/*
  - Admin API: /api/admin/economics/*
  - Health Check: /health
=====================================
```

### 5. Test the Service

```bash
# In a new terminal window:

# Health check
curl http://localhost:3000/health

# Economics health
curl http://localhost:3000/api/economics/health

# Get economic metrics
curl http://localhost:3000/api/economics/metrics
```

## Common Tasks

### Process a Skill Invocation

```bash
curl -X POST http://localhost:3000/api/economics/skill/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "user123",
    "skillId": "skill456",
    "amount": "100000"
  }'
```

### Get Burn History

```bash
curl http://localhost:3000/api/economics/burn/history?limit=10
```

### Request Testnet Tokens

```bash
curl -X POST http://localhost:3000/api/faucet/request \
  -H "Content-Type: application/json" \
  -d '{
    "address": "knirv1abc...",
    "network": "public-testnet"
  }'
```

### Get Admin System Status

```bash
curl http://localhost:3000/api/admin/economics/system/status
```

### Update Economic Rules

```bash
curl -X PUT http://localhost:3000/api/admin/economics/rules \
  -H "Content-Type: application/json" \
  -d '{
    "skillInvocationCost": "150000",
    "llmRegistrationFee": "1500000",
    "validationReward": "60000"
  }'
```

## What's Running

When you start the service, it automatically:

1. ✅ Initializes the Economics Engine
2. ✅ Starts 4 background processors:
   - Transaction processor (every 5s)
   - Metrics updater (every 1m)
   - Reward distributor (every 1h)
   - Burn processor (every 10s)
3. ✅ Starts component integration:
   - KNIRVCHAIN sync (every 10s)
   - KNIRVNEXUS sync (every 15s)
   - KNIRVORACLE sync (every 20s)
   - KNIRVGRAPH sync (every 25s)
4. ✅ Mounts all API routes
5. ✅ Starts Express server on port 3000

## Stopping the Service

```bash
Ctrl+C
```

The service will gracefully shutdown:
- Stop all background processors
- Close component connections
- Save state
- Exit cleanly

## Troubleshooting

### Port 3000 Already in Use

```bash
# Use a different port
PORT=3001 npm start
```

### Can't Find Module Error

```bash
# Reinstall dependencies
rm -rf node_modules
npm install
```

### Environment Variables Not Loading

```bash
# Make sure .env file exists
ls -la .env

# Or set variables directly
NRN_CONTRACT=xxx XION_RPC=yyy npm start
```

## Next Steps

- Read [README.md](./README.md) for full documentation
- See [ECONOMICS_API.md](./ECONOMICS_API.md) for API reference
- Check [ECONOMICS_MIGRATION.md](./ECONOMICS_MIGRATION.md) for migration details

## That's It!

Your payment oracle with integrated economics engine is now running!

Visit `http://localhost:3000/health` to verify.
