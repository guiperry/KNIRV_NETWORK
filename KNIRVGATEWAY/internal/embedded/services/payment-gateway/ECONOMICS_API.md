# KNIRV Economics API Reference

## Base URL

All economics endpoints are available at `/api/economics/`

## Authentication

Currently, public economics endpoints are open. Admin endpoints at `/api/admin/economics/` should be protected with authentication middleware (to be implemented).

## Response Format

All responses follow this format:

```json
{
  "success": true,
  "data": { ... }
}
```

Error responses:

```json
{
  "success": false,
  "error": "Error message"
}
```

## Public Economics Endpoints

### Skill Invocation

Process a skill invocation and burn tokens.

**Endpoint:** `POST /api/economics/skill/invoke`

**Request Body:**
```json
{
  "userId": "user123",
  "skillId": "skill456",
  "amount": "100000"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "transactionId": "skill_skill456_1234567890_abc123",
    "status": "pending",
    "amount": "100000",
    "timestamp": "2025-12-07T..."
  }
}
```

### LLM Registration

Process LLM registration fee.

**Endpoint:** `POST /api/economics/llm/register`

**Request Body:**
```json
{
  "userId": "user123",
  "llmId": "llm789",
  "registrationFee": "1000000"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "transactionId": "llm_reg_llm789_1234567890_abc123",
    "status": "pending",
    "fee": "1000000",
    "timestamp": "2025-12-07T..."
  }
}
```

### Validation Reward

Process validation reward for validators.

**Endpoint:** `POST /api/economics/validation/reward`

**Request Body:**
```json
{
  "validatorId": "validator123",
  "targetId": "target456",
  "validationResult": true
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "transactionId": "validation_target456_1234567890_abc123",
    "status": "pending",
    "reward": "75000",
    "timestamp": "2025-12-07T..."
  }
}
```

### Calculate Network Fees

Calculate network transaction fees.

**Endpoint:** `POST /api/economics/fees/calculate`

**Request Body:**
```json
{
  "gasUsed": 21000,
  "priority": "medium"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "gasUsed": 21000,
    "priority": "medium",
    "totalFee": "31500000",
    "gasPrice": "1500"
  }
}
```

**Priority Levels:**
- `high`: 2.0x multiplier
- `medium`: 1.5x multiplier
- `low`: 1.0x multiplier

### Get Economic Metrics

Get current network economic metrics.

**Endpoint:** `GET /api/economics/metrics`

**Response:**
```json
{
  "success": true,
  "data": {
    "totalSupply": "0",
    "circulatingSupply": "0",
    "totalBurned": "0",
    "totalStaked": "0",
    "activeValidators": 0,
    "transactionVolume": "0",
    "averageGasPrice": "1000",
    "networkUtilization": 0.75,
    "tokenVelocity": 2.5,
    "lastUpdated": "2025-12-07T...",
    "serviceMetrics": {
      "knirvchain": { ... },
      "knirvnexus": { ... },
      "knirvoracle": { ... },
      "knirvgraph": { ... }
    }
  }
}
```

### Get Transaction

Get transaction by ID.

**Endpoint:** `GET /api/economics/transaction/:id`

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "skill_skill456_1234567890_abc123",
    "type": "skill_invocation",
    "from": "user123",
    "to": "skill_registry",
    "amount": "100000",
    "purpose": "skill_invocation",
    "metadata": { "skillId": "skill456" },
    "status": "confirmed",
    "timestamp": "2025-12-07T...",
    "confirmedAt": "2025-12-07T..."
  }
}
```

### Get Transactions

Get list of transactions.

**Endpoint:** `GET /api/economics/transactions?limit=100&status=confirmed`

**Query Parameters:**
- `limit` (optional): Number of transactions to return (default: 100)
- `status` (optional): Filter by status (`pending`, `confirmed`, `failed`)

**Response:**
```json
{
  "success": true,
  "data": {
    "transactions": [ ... ],
    "count": 50,
    "limit": 100,
    "statusFilter": "confirmed"
  }
}
```

### Get Burn History

Get token burn event history.

**Endpoint:** `GET /api/economics/burn/history?limit=100&user=user123&purpose=skill_invocation`

**Query Parameters:**
- `limit` (optional): Number of events to return (default: 100)
- `user` (optional): Filter by user ID
- `purpose` (optional): Filter by burn purpose

**Response:**
```json
{
  "success": true,
  "data": {
    "burnEvents": [
      {
        "txId": "skill_skill456_1234567890_abc123",
        "user": "user123",
        "amount": "100000",
        "purpose": "skill_invocation",
        "skillId": "skill456",
        "timestamp": "2025-12-07T...",
        "validated": true
      }
    ],
    "count": 1,
    "limit": 100
  }
}
```

### Get Total Burned

Get total amount of tokens burned.

**Endpoint:** `GET /api/economics/burn/total`

**Response:**
```json
{
  "success": true,
  "data": {
    "totalBurned": "100000",
    "timestamp": "2025-12-07T..."
  }
}
```

### Get Economic Rules

Get current economic rules configuration.

**Endpoint:** `GET /api/economics/rules`

**Response:**
```json
{
  "success": true,
  "data": {
    "skillInvocationCost": "100000",
    "llmRegistrationFee": "1000000",
    "validationReward": "50000",
    "burnRates": {
      "skill_invocation": "100000",
      "llm_registration": "500000",
      "validation": "25000"
    },
    "mintingRules": { ... },
    "stakingRequirements": { ... },
    "governanceThresholds": { ... }
  }
}
```

### Update Economic Rules

Update economic rules (admin operation).

**Endpoint:** `PUT /api/economics/rules`

**Request Body:**
```json
{
  "skillInvocationCost": "150000",
  "llmRegistrationFee": "1500000",
  "validationReward": "60000"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Economic rules updated successfully",
    "rules": { ... }
  }
}
```

### Get Service Metrics

Get economics metrics for a specific service.

**Endpoint:** `GET /api/economics/service/:service/metrics`

**Services:** `knirvchain`, `knirvnexus`, `knirvoracle`, `knirvgraph`

**Response:**
```json
{
  "success": true,
  "data": {
    "revenue": "0",
    "costs": "0",
    "profit": "0",
    "tokensEarned": "0",
    "tokensSpent": "0",
    "userCount": 0,
    "transactionCount": 0,
    "lastUpdated": "2025-12-07T..."
  }
}
```

### Integration Status

Get integration status with KNIRV components.

**Endpoint:** `GET /api/economics/integration/status`

**Response:**
```json
{
  "success": true,
  "data": {
    "componentURLs": {
      "knirvchain": "http://localhost:8080",
      "knirvnexus": "http://localhost:8081",
      "knirvoracle": "http://localhost:8082",
      "knirvgraph": "http://localhost:8083"
    },
    "lastSync": "2025-12-07T...",
    "status": "active"
  }
}
```

### Health Check

Check economics engine health.

**Endpoint:** `GET /api/economics/health`

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2025-12-07T...",
    "version": "1.0.0",
    "service": "KNIRV Economics Engine"
  }
}
```

### System Info

Get system information.

**Endpoint:** `GET /api/economics/info`

**Response:**
```json
{
  "success": true,
  "data": {
    "service": "KNIRV Economics Service",
    "version": "1.0.0",
    "isRunning": true,
    "nrnContract": "...",
    "xionRPC": "https://rpc.xion-testnet-1.burnt.com:443",
    "timestamp": "2025-12-07T..."
  }
}
```

## Admin Endpoints

All admin endpoints require authentication (to be implemented).

Base URL: `/api/admin/economics/`

### System Control

**Start System**
```
POST /api/admin/economics/system/start
```

**Stop System**
```
POST /api/admin/economics/system/stop
```

**Get System Status**
```
GET /api/admin/economics/system/status
```

### Rules Management

**Get Rules**
```
GET /api/admin/economics/rules
```

**Update Rules**
```
PUT /api/admin/economics/rules
```

**Reset Rules**
```
POST /api/admin/economics/rules/reset
```

### Integration Management

**Get Integration Status**
```
GET /api/admin/economics/integration/status
```

**Update Component URL**
```
PUT /api/admin/economics/integration/component/:component
Body: { "url": "http://new-url:port" }
```

**Force Sync**
```
POST /api/admin/economics/integration/sync
```

### Metrics Management

**Get Metrics Summary**
```
GET /api/admin/economics/metrics/summary
```

**Get Service Metrics**
```
GET /api/admin/economics/metrics/service/:service
```

**Reset Metrics**
```
POST /api/admin/economics/metrics/reset
```

### Transaction Management

**Get Pool Stats**
```
GET /api/admin/economics/transactions/pool
```

**Delete Transaction**
```
DELETE /api/admin/economics/transactions/:id
```

**Cleanup Transactions**
```
POST /api/admin/economics/transactions/cleanup
```

### Burn Management

**Get Burn Stats**
```
GET /api/admin/economics/burn/stats
```

**Get Total Burned**
```
GET /api/admin/economics/burn/total
```

### Reward Management

**Get Leaderboard**
```
GET /api/admin/economics/rewards/leaderboard?limit=10&metric=successRate
```

**Get User Performance**
```
GET /api/admin/economics/rewards/performance/:userId
```

**Update Multiplier**
```
PUT /api/admin/economics/rewards/multiplier/:name
Body: { "value": 1.5 }
```

### Configuration

**Get Config**
```
GET /api/admin/economics/config
```

**Update Config**
```
PUT /api/admin/economics/config
Body: {
  "nrnContract": "new_contract_address",
  "xionRPC": "https://new-rpc-url"
}
```

### Export/Import

**Export State**
```
GET /api/admin/economics/export/state
```

**Import State**
```
POST /api/admin/economics/import/state
```

## Error Codes

- `400` - Bad Request (invalid parameters)
- `404` - Not Found (resource doesn't exist)
- `429` - Too Many Requests (rate limit exceeded)
- `500` - Internal Server Error

## Rate Limiting

Currently, no rate limiting is implemented on economics endpoints. This should be added in production.

## WebSocket Support

WebSocket support for real-time updates is not yet implemented but planned for future releases.

## Code Examples

### JavaScript/Node.js

```javascript
const axios = require('axios');

// Process skill invocation
async function invokeSkill(userId, skillId, amount) {
  const response = await axios.post('http://localhost:3000/api/economics/skill/invoke', {
    userId,
    skillId,
    amount: amount.toString()
  });
  return response.data;
}

// Get metrics
async function getMetrics() {
  const response = await axios.get('http://localhost:3000/api/economics/metrics');
  return response.data;
}
```

### cURL

```bash
# Process skill invocation
curl -X POST http://localhost:3000/api/economics/skill/invoke \
  -H "Content-Type: application/json" \
  -d '{"userId":"user123","skillId":"skill456","amount":"100000"}'

# Get metrics
curl http://localhost:3000/api/economics/metrics

# Get burn history
curl "http://localhost:3000/api/economics/burn/history?limit=10&user=user123"
```

## BigInt Handling

All token amounts are handled as BigInt in JavaScript for precision. When sending/receiving via API, they are converted to strings:

```javascript
// Sending
const amount = BigInt(100000);
fetch('/api/economics/skill/invoke', {
  body: JSON.stringify({ amount: amount.toString() })
});

// Receiving
const response = await fetch('/api/economics/metrics');
const data = await response.json();
const totalBurned = BigInt(data.data.totalBurned);
```

## Version History

- v1.0.0 (2025-12-07) - Initial JavaScript implementation
  - Migrated from Go
  - Full feature parity with Go version
  - Added admin API endpoints
  - Integrated into payment-gateway
