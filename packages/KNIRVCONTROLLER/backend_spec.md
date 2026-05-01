# KNIRVWALLET Backend Specification

## 1. Database Schema (PostgreSQL/Supabase Compatible)

### 1.1 Users Table
Stores user account information and authentication details.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_address TEXT UNIQUE NOT NULL,
    public_key TEXT NOT NULL,
    email TEXT UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_wallet_address ON users(wallet_address);
```

### 1.2 User Delegation Certificates (UDC)
Stores user authorization certificates for network access.

```sql
CREATE TABLE user_delegation_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('valid', 'expired', 'revoked', 'pending')),
    issued_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    permissions TEXT[] NOT NULL,
    signature TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_udc_user_id ON user_delegation_certificates(user_id);
CREATE INDEX idx_udc_status ON user_delegation_certificates(status);
```

### 1.3 NRN Transactions
Stores all NRN token transactions (consumption, rewards, transfers).

```sql
CREATE TABLE nrn_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    agent_id TEXT,
    amount NUMERIC(10, 2) NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('consumption', 'reward', 'transfer')),
    description TEXT,
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_transactions_user_id ON nrn_transactions(user_id);
CREATE INDEX idx_transactions_timestamp ON nrn_transactions(timestamp DESC);
CREATE INDEX idx_transactions_type ON nrn_transactions(type);
```

### 1.4 Agents
Stores AI agent configuration and status information.

```sql
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('CodeT5', 'SEAL', 'LoRA', 'NRN')),
    status TEXT NOT NULL CHECK (status IN ('active', 'idle', 'error', 'deploying')),
    tasks INTEGER NOT NULL DEFAULT 0,
    performance INTEGER NOT NULL DEFAULT 0 CHECK (performance BETWEEN 0 AND 100),
    last_active TIMESTAMPTZ,
    config JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_agents_user_id ON agents(user_id);
CREATE INDEX idx_agents_status ON agents(status);
```

### 1.5 Skills
Stores available AI agent skills and their configurations.

```sql
CREATE TABLE skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    category TEXT NOT NULL CHECK (category IN ('automation', 'analysis', 'communication', 'computation')),
    complexity INTEGER NOT NULL CHECK (complexity BETWEEN 1 AND 10),
    nrn_cost NUMERIC(10, 2) NOT NULL,
    requirements TEXT[],
    is_active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_skills_category ON skills(category);
CREATE INDEX idx_skills_active ON skills(is_active);
```

### 1.6 Agent Skills
Stores the skills activated for each agent.

```sql
CREATE TABLE agent_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    skill_id UUID REFERENCES skills(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(agent_id, skill_id)
);

CREATE INDEX idx_agent_skills_agent ON agent_skills(agent_id);
CREATE INDEX idx_agent_skills_skill ON agent_skills(skill_id);
```

### 1.7 SEAL Loop Status
Stores the status of the SEAL (Self-Evolving Autonomous Loop) optimization process.

```sql
CREATE TABLE seal_loop_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT false,
    current_cycle INTEGER NOT NULL DEFAULT 0,
    next_cycle_at TIMESTAMPTZ,
    optimizations INTEGER NOT NULL DEFAULT 0,
    failure_detections INTEGER NOT NULL DEFAULT 0,
    solutions_proposed INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_seal_loop_user ON seal_loop_status(user_id);
```

## 2. API Endpoints (REST)

### 2.1 Authentication Endpoints

#### 2.1.1 Web3 Login (Xion Wallet)
**POST /api/auth/web3/login**

Request:
```json
{
  "walletAddress": "0x742d35Cc6aa34567...8B9fA2e1C4D",
  "signature": "0xabc123...",
  "message": "Login to KNIRV Wallet at 2024-08-06T10:30:00Z"
}
```

Response:
```json
{
  "success": true,
  "data": {
    "userId": "uuid-123",
    "accessToken": "jwt-token-here",
    "refreshToken": "refresh-token-here",
    "walletAddress": "0x742d35Cc6aa34567...8B9fA2e1C4D"
  }
}
```

#### 2.1.2 Refresh Token
**POST /api/auth/refresh**

Request:
```json
{
  "refreshToken": "refresh-token-here"
}
```

Response:
```json
{
  "success": true,
  "data": {
    "accessToken": "new-jwt-token-here"
  }
}
```

### 2.2 Wallet Endpoints

#### 2.2.1 Get Wallet Balance
**GET /api/wallet/balance**

Headers:
- Authorization: Bearer {accessToken}

Response:
```json
{
  "success": true,
  "data": {
    "nrnBalance": 1247,
    "usdValue": 312.75,
    "change24h": 5.2,
    "walletAddress": "0x742d35Cc6aa34567...8B9fA2e1C4D"
  }
}
```

#### 2.2.2 Get Transaction History
**GET /api/wallet/transactions?limit=10&offset=0**

Headers:
- Authorization: Bearer {accessToken}

Response:
```json
{
  "success": true,
  "data": {
    "transactions": [
      {
        "id": "uuid-123",
        "type": "consumption",
        "amount": -25,
        "description": "Code Analysis Skill",
        "timestamp": "2024-08-06T01:15:00Z",
        "agentName": "CodeT5-Alpha"
      },
      {
        "id": "uuid-456",
        "type": "reward",
        "amount": 50,
        "description": "Task completion bonus",
        "timestamp": "2024-08-06T00:45:00Z",
        "agentName": "SEAL-Beta"
      }
    ],
    "total": 15
  }
}
```

#### 2.2.3 Send NRN Tokens
**POST /api/wallet/send**

Headers:
- Authorization: Bearer {accessToken}

Request:
```json
{
  "toAddress": "0x9876543210...",
  "amount": 100,
  "description": "Payment for services"
}
```

Response:
```json
{
  "success": true,
  "data": {
    "transactionId": "uuid-789",
    "status": "pending",
    "txHash": "0xabc123..."
  }
}
```

### 2.3 UDC Endpoints

#### 2.3.1 Get Current UDC
**GET /api/udc/current**

Headers:
- Authorization: Bearer {accessToken}

Response:
```json
{
  "success": true,
  "data": {
    "id": "UDC-7A8B9C2D",
    "status": "valid",
    "issuedAt": "2024-08-01T10:30:00Z",
    "expiresAt": "2024-08-08T10:30:00Z",
    "permissions": [
      "agent.deploy",
      "skill.activate",
      "nrn.transfer",
      "dten.access",
      "wallet.connect"
    ],
    "signature": "0xabc123..."
  }
}
```

#### 2.3.2 Renew UDC
**POST /api/udc/renew**

Headers:
- Authorization: Bearer {accessToken}

Request:
```json
{
  "durationDays": 30
}
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "UDC-7A8B9C2D",
    "status": "valid",
    "issuedAt": "2024-08-01T10:30:00Z",
    "expiresAt": "2024-09-01T10:30:00Z",
    "permissions": [
      "agent.deploy",
      "skill.activate",
      "nrn.transfer",
      "dten.access",
      "wallet.connect"
    ],
    "signature": "0xdef456..."
  }
}
```

### 2.4 Agent Endpoints

#### 2.4.1 Get All Agents
**GET /api/agents**

Headers:
- Authorization: Bearer {accessToken}

Response:
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid-123",
      "name": "CodeT5-Alpha",
      "type": "CodeT5",
      "status": "active",
      "tasks": 12,
      "performance": 94,
      "lastActive": "2024-08-06T10:30:00Z",
      "config": {}
    }
  ]
}
```

#### 2.4.2 Deploy Agent
**POST /api/agents/deploy**

Headers:
- Authorization: Bearer {accessToken}

Request:
```json
{
  "name": "New Agent",
  "type": "CodeT5",
  "config": {}
}
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "uuid-456",
    "name": "New Agent",
    "type": "CodeT5",
    "status": "deploying",
    "tasks": 0,
    "performance": 0,
    "lastActive": null,
    "config": {}
  }
}
```

### 2.5 Skill Endpoints

#### 2.5.1 Get All Skills
**GET /api/skills**

Headers:
- Authorization: Bearer {accessToken}

Response:
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid-123",
      "name": "Code Analysis",
      "description": "Automated code review and optimization",
      "category": "analysis",
      "complexity": 8,
      "nrnCost": 25,
      "requirements": [],
      "isActive": true
    }
  ]
}
```

#### 2.5.2 Activate Skill
**POST /api/skills/{skillId}/activate**

Headers:
- Authorization: Bearer {accessToken}

Response:
```json
{
  "success": true,
  "data": {
    "id": "uuid-123",
    "name": "Code Analysis",
    "isActive": true
  }
}
```

#### 2.5.3 Deactivate Skill
**POST /api/skills/{skillId}/deactivate**

Headers:
- Authorization: Bearer {accessToken}

Response:
```json
{
  "success": true,
  "data": {
    "id": "uuid-123",
    "name": "Code Analysis",
    "isActive": false
  }
}
```

### 2.6 SEAL Loop Endpoints

#### 2.6.1 Get SEAL Loop Status
**GET /api/seal/status**

Headers:
- Authorization: Bearer {accessToken}

Response:
```json
{
  "success": true,
  "data": {
    "isActive": true,
    "currentCycle": 15,
    "nextCycleAt": "2024-08-06T10:35:00Z",
    "optimizations": 127,
    "failureDetections": 3,
    "solutionsProposed": 89
  }
}
```

#### 2.6.2 Toggle SEAL Loop
**POST /api/seal/toggle**

Headers:
- Authorization: Bearer {accessToken}

Request:
```json
{
  "isActive": true
}
```

Response:
```json
{
  "success": true,
  "data": {
    "isActive": true
  }
}
```

## 3. Authentication Flow

### 3.1 Web3 Wallet Authentication (Xion)

1. **Client requests nonce**: Frontend requests a unique nonce from backend for signature
2. **User signs message**: User signs the message containing nonce and timestamp using Xion wallet
3. **Backend verifies signature**: Backend verifies the signature using wallet address and message
4. **JWT token issued**: If valid, backend issues JWT access token and refresh token
5. **Token storage**: Frontend stores tokens securely (localStorage/sessionStorage)
6. **API authorization**: All API requests include Bearer token in Authorization header
7. **Token refresh**: When access token expires, use refresh token to get new access token

### 3.2 JWT Token Structure

```json
{
  "sub": "user-id-uuid",
  "walletAddress": "0x742d35Cc6aa34567...8B9fA2e1C4D",
  "iat": 1628235600,
  "exp": 1628239200,
  "permissions": ["agent.deploy", "skill.activate", "nrn.transfer"]
}
```

### 3.3 Session Management

- Access token expires after 1 hour
- Refresh token expires after 7 days
- Refresh tokens stored in HTTP-only cookies for security
- Token blacklist for invalidation on logout

## 4. Crypto Logic (Xion Integration)

### 4.1 Xion Network Integration

The backend will interact with Xion blockchain for:
- Transaction signing and verification
- Token balance checks
- Transaction history retrieval
- Smart contract interactions

### 4.2 Transaction Signing Flow

```typescript
// Backend code snippet for Xion transaction signing
import { SigningStargateClient } from "@cosmjs/stargate";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { GasPrice } from "@cosmjs/stargate";

// Initialize Xion client
const xionClient = await SigningStargateClient.connectWithSigner(
  "https://xion-testnet-rpc.example.com",
  wallet,
  { gasPrice: GasPrice.fromString("0.0025uxion") }
);

// Transfer NRN tokens
const transferMsg = {
  typeUrl: "/cosmos.bank.v1beta1.MsgSend",
  value: {
    fromAddress: "xion1abc123...",
    toAddress: "xion1def456...",
    amount: [
      {
        denom: "unrn", // micro-NRN
        amount: "1000000" // 1 NRN = 1e6 unrn
      }
    ]
  }
};

// Sign and broadcast transaction
const result = await xionClient.signAndBroadcast(
  "xion1abc123...",
  [transferMsg],
  "auto",
  "Transfer 1 NRN"
);

console.log("Transaction hash:", result.transactionHash);
```

### 4.3 Token Conversion

- NRN token symbol: `NRN`
- Decimal places: 6
- Minimum unit: `unrn` (1 NRN = 1,000,000 unrn)
- USD price oracle integration for real-time price updates

### 4.4 Transaction Types

1. **Consumption**: NRN tokens spent on skills/agent usage
2. **Reward**: NRN tokens earned from task completion
3. **Transfer**: Direct token transfers between wallets

### 4.5 Gas Fees

- Transactions require gas fees in XION tokens
- Gas price dynamically calculated based on network congestion
- Estimated gas fee provided before transaction confirmation

## 5. WebSocket Endpoints

### 5.1 Real-time Updates

**GET /api/ws/updates**

Subscriptions:
- `transaction`: New transaction notifications
- `balance`: Balance updates
- `agent-status`: Agent status changes
- `seal-loop`: SEAL loop cycle updates

Example message:
```json
{
  "type": "transaction",
  "data": {
    "id": "uuid-789",
    "type": "reward",
    "amount": 50,
    "description": "Task completion bonus",
    "timestamp": "2024-08-06T00:45:00Z",
    "agentName": "SEAL-Beta"
  }
}
```

## 6. Error Handling

### 6.1 Standard Error Response

```json
{
  "success": false,
  "error": {
    "code": "AUTH_001",
    "message": "Invalid signature",
    "details": "Signature verification failed for wallet address 0xabc123..."
  }
}
```

### 6.2 Error Codes

| Code | Message | Description |
|------|---------|-------------|
| AUTH_001 | Invalid signature | Signature verification failed |
| AUTH_002 | Expired token | Access token has expired |
| AUTH_003 | Invalid token | Token format is invalid |
| WALLET_001 | Insufficient balance | Not enough NRN for transaction |
| WALLET_002 | Invalid address | Recipient address is invalid |
| UDC_001 | Certificate expired | UDC has expired |
| UDC_002 | Insufficient permissions | User lacks required permissions |
| AGENT_001 | Agent not found | Agent with given ID does not exist |
| SKILL_001 | Skill not found | Skill with given ID does not exist |

## 7. Rate Limiting

### 7.1 API Rate Limits

- **Authentication endpoints**: 100 requests/minute per IP
- **Wallet endpoints**: 50 requests/minute per user
- **Transaction endpoints**: 10 requests/minute per user
- **WebSocket connections**: 5 connections per user

### 7.2 Response Headers

```http
X-RateLimit-Limit: 50
X-RateLimit-Remaining: 49
X-RateLimit-Reset: 1628235600
```

## 8. Caching Strategy

### 8.1 Client-side Caching

- Wallet balance: Cache for 30 seconds
- Transaction history: Cache for 1 minute
- Agent status: Cache for 5 seconds
- Skill list: Cache for 5 minutes

### 8.2 Server-side Caching

- UDC status: Cache for 1 hour
- Agent configuration: Cache for 5 minutes
- Skill metadata: Cache for 10 minutes

## 9. Security Measures

### 9.1 Data Encryption

- All sensitive data encrypted at rest (AES-256)
- SSL/TLS for all API endpoints
- Encrypted WebSocket connections (WSS)

### 9.2 Input Validation

- All inputs validated against Zod schemas
- SQL injection protection via parameterized queries
- XSS protection via output encoding

### 9.3 Access Control

- Role-based access control (RBAC)
- Least privilege principle
- Permission checks for all API endpoints

## 10. Monitoring and Logging

### 10.1 Audit Logs

- User login attempts
- Token generation/refresh
- Transaction history
- UDC renewals/revocations
- Agent deployments

### 10.2 Performance Monitoring

- API response times
- Error rates
- Transaction volumes
- Network latency to Xion blockchain

---

This backend specification provides a complete framework for supporting the KNIRVWALLET frontend with secure, scalable, and efficient API endpoints, database schema, authentication mechanisms, and Xion blockchain integration.
