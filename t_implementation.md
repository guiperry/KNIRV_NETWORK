# Implementing Zero Trust Architecture in KNIRV_NETWORK

## 1. Introduction

This document outlines a plan for implementing a Zero Trust Architecture (ZTA) within the KNIRV_NETWORK, with a particular focus on the `KNIRVGATEWAY` component. The recommendations are based on the principles outlined in NIST SP 800-207 and a thorough analysis of the existing codebase.

## 2. Current State Analysis

The `KNIRVGATEWAY` is a central component in the KNIRV_NETWORK, acting as a reverse proxy and a single point of entry for many services. The current implementation has several security weaknesses that need to be addressed:

*   **Insecure Mock Authentication:** The authentication mechanism in `internal/auth/handler.go` is a mock implementation and is not suitable for production use. It uses a non-standard, insecure token generation and verification mechanism.
*   **Lack of Authorization:** There is no authorization logic to control access to different resources.
*   **Permissive CORS Policy:** The Cross-Origin Resource Sharing (CORS) policy is overly permissive, which could expose the network to cross-site request forgery (CSRF) and other attacks.
*   **No Centralized Policy Enforcement:** While the gateway acts as a proxy, it does not enforce any security policies on the incoming traffic.

## 3. Proposed Zero Trust Architecture: API Key System Implementation

Based on Zero Trust principles of "never trust, always verify," here's a comprehensive architecture for implementing API keys with continuous validation and least-privilege access.

### 3.1. Core Zero Trust Principles for API Keys

*   **Never Implicitly Trust Keys**: Treat every API key as potentially compromised. No key receives permanent trust status, regardless of its origin (internal, partner, or public).
*   **Verify Every Request**: Each API call requires independent authentication, authorization, and validation—no session-based trust.
*   **Least Privilege Access**: Keys grant minimal necessary permissions, scoped to specific endpoints and data.
*   **Continuous Validation**: Monitor key usage patterns in real-time and revoke access at the first sign of compromise.

### 3.2. API Key Design & Generation

#### Key Format
```go
// Zero Trust Key Structure: prefix_keyId.secret
// Example: sk_prod_f8k2m9n4.xQ9zR7tV2wY5uP1aS3dF6gH8jK0l
```

- **Prefix**: Identifies environment and use case (`sk_prod_`, `sk_test_`)
- **Key ID**: Database lookup identifier (stored in plaintext)
- **Secret**: Cryptographically random 32-byte string (NEVER stored, only hashed)

#### Generation Process
The following Go-like pseudocode illustrates the key generation process. This logic would replace the mock implementation in `internal/auth/handler.go`.

```go
import (
    "crypto/rand"
    "database/sql"
    "encoding/base64"
    "golang.org/x/crypto/argon2"
    "time"
)

// generateZeroTrustKey creates a new API key with Zero Trust metadata.
func generateZeroTrustKey(db *sql.DB, userID string, scopes []string, metadata map[string]interface{}) (string, error) {
    // 1. Generate cryptographically secure secret
    secretBytes := make([]byte, 32)
    if _, err := rand.Read(secretBytes); err != nil {
        return "", err
    }
    secret := base64.URLEncoding.EncodeToString(secretBytes)

    // 2. Create key ID for database lookup
    keyIDBytes := make([]byte, 6)
    if _, err := rand.Read(keyIDBytes); err != nil {
        return "", err
    }
    keyID := hex.EncodeToString(keyIDBytes)

    // 3. Hash secret for storage (Argon2id)
    // Use strong, recommended parameters for Argon2id
    salt := make([]byte, 16)
    if _, err := rand.Read(salt); err != nil {
        return "", err
    }
    secretHash := argon2.IDKey([]byte(secret), salt, 1, 64*1024, 4, 32)

    // 4. Store with Zero Trust metadata
    expiresAt := time.Now().Add(90 * 24 * time.Hour) // Mandatory expiration
    _, err := db.Exec(`
        INSERT INTO api_keys (key_id, user_id, secret_hash, salt, scopes, is_active, created_at, expires_at, max_requests_per_minute, allowed_ips, last_rotated, risk_score)
        VALUES ($1, $2, $3, $4, $5, true, NOW(), $6, 100, NULL, NOW(), 0)
    `, keyID, userID, secretHash, salt, scopes)
    if err != nil {
        return "", err
    }

    return "sk_prod_" + keyID + "." + secret, nil // Return once only
}
```

### 3.3. Zero Trust Database Schema

The following SQL schema should be implemented to support the API key system.

```sql
CREATE TABLE api_keys (
  key_id VARCHAR(12) PRIMARY KEY,          -- For lookup
  user_id UUID NOT NULL,
  secret_hash BYTEA NOT NULL,              -- Argon2id hash
  salt BYTEA NOT NULL,                     -- Salt for Argon2id
  scopes JSONB,                            -- Granular permissions
  is_active BOOLEAN DEFAULT false,         -- Deny by default
  created_at TIMESTAMP WITH TIME ZONE,
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL, -- Mandatory TTL
  last_used_at TIMESTAMP WITH TIME ZONE,
  usage_count INTEGER DEFAULT 0,

  -- Zero Trust monitoring fields
  risk_score INTEGER DEFAULT 0,            -- 0-100
  anomaly_flags JSONB,                     -- Suspicious patterns
  allowed_ips INET[],                      -- Null = no IP trust
  revoked_at TIMESTAMP WITH TIME ZONE,

  -- Rotation metadata
  parent_key_id VARCHAR(12),               -- For audit trail
  rotation_reason VARCHAR(50)
);

CREATE TABLE api_key_events (
  event_id UUID PRIMARY KEY,
  key_id VARCHAR(12),
  timestamp TIMESTAMP WITH TIME ZONE,
  request_method VARCHAR(10),
  endpoint VARCHAR(200),
  status_code INTEGER,
  response_time_ms INTEGER,
  ip_address INET,
  user_agent TEXT,
  risk_factors JSONB                       -- Captured anomalies
);
```

### 3.4. Zero Trust Validation Pipeline

The `KNIRVGATEWAY` will be the primary enforcement layer.

```
Client Request
    ↓
[API Gateway (KNIRVGATEWAY)] ← First line of defense
    ↓
[Auth Service (part of KNIRVGATEWAY)] ← Trust verification
    ↓
[Policy Engine (part of KNIRVGATEWAY)] ← Authorization decision
    ↓
[Backend API] ← Final validation
```

#### Step 1: Key Extraction & Initial Validation (in `internal/server/middleware.go`)

A new middleware in the `KNIRVGATEWAY` will perform the initial validation.

```go
// AuthMiddleware extracts and performs initial validation on the API key.
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            http.Error(w, "Missing credentials", http.StatusUnauthorized)
            return
        }

        fullKey := strings.TrimPrefix(authHeader, "Bearer ")
        parts := strings.Split(fullKey, ".")
        if len(parts) != 2 {
            http.Error(w, "Invalid key format", http.StatusUnauthorized)
            return
        }
        keyID, secret := parts[0], parts[1]

        clientIP := r.RemoteAddr // Simplified; use a proper IP extraction method
        
        ctx := context.WithValue(r.Context(), "keyID", keyID)
        ctx = context.WithValue(ctx, "secret", secret)
        ctx = context.WithValue(ctx, "clientIP", clientIP)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

#### Step 2: Cryptographic Verification (in `internal/auth/handler.go`)

The `authenticateUser` function will be replaced with this logic.

```go
// verifyKeyIntegrity checks the key against the database.
func verifyKeyIntegrity(ctx context.Context, db *sql.DB, keyID, secret, clientIP string) (map[string]interface{}, error) {
    var keyRecord struct {
        secretHash []byte
        salt []byte
        isActive bool
        expiresAt time.Time
        allowedIPs []string
    }

    // Fetch key metadata (use a cache like Redis here for performance)
    err := db.QueryRowContext(ctx, "SELECT secret_hash, salt, is_active, expires_at, allowed_ips FROM api_keys WHERE key_id = $1", keyID).Scan(&keyRecord.secretHash, &keyRecord.salt, &keyRecord.isActive, &keyRecord.expiresAt, &keyRecord.allowedIPs)
    if err != nil {
        return nil, fmt.Errorf("authentication failed")
    }
    
    if !keyRecord.isActive {
        return nil, fmt.Errorf("key invalid or deactivated")
    }

    if time.Now().After(keyRecord.expiresAt) {
        // Implement revokeKey logic
        return nil, fmt.Errorf("key expired")
    }

    // Verify secret hash (never trust the key itself)
    hash := argon2.IDKey([]byte(secret), keyRecord.salt, 1, 64*1024, 4, 32)
    if !bytes.Equal(keyRecord.secretHash, hash) {
        // Implement logSuspiciousEvent
        return nil, fmt.Errorf("authentication failed")
    }

    // Zero Trust: Verify IP even if previously allowed
    if len(keyRecord.allowedIPs) > 0 {
        // IP validation logic here
    }

    // Return key metadata for policy evaluation
    return map[string]interface{}{"key_id": keyID}, nil
}
```

#### Step 3: Continuous Risk Assessment (in a new `internal/risk` package)

A new service for real-time risk scoring.

```go
// calculateRiskScore assesses the risk of a request in real-time.
func calculateRiskScore(keyRecord map[string]interface{}, requestContext map[string]interface{}) (int, error) {
    // Anomaly detection factors
    // - Velocity check
    // - Geographic anomaly
    // - Time-of-day anomaly
    // - User agent mismatch
    // - Endpoint access pattern
    
    // Increment score based on anomalies
    score := 0
    // ... complex logic here ...
    
    if score > 75 {
        // Implement revokeKey
        return score, fmt.Errorf("key suspended due to suspicious activity")
    }
    
    return score, nil
}
```

### 3.5. Zero Trust Authorization Model

Scoped API keys with microsegmentation can be defined in the `scopes` and `anomaly_flags` JSONB columns.

```json
{
  "key_id": "f8k2m9n4",
  "user_id": "usr_12345",
  "scopes": [
    "read:users:profile",
    "write:orders:create",
    "read:products:*"
  ],
  "denied_endpoints": [
    "DELETE /admin/*"
  ],
  "field_level_access": {
    "users": ["id", "name", "email"],
    "orders": ["id", "status", "total"]
  },
  "rate_limits": {
    "/api/v1/orders": "10/minute",
    "/api/v1/users": "100/minute"
  }
}
```

### 3.6. Zero Trust Infrastructure Components

```
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway Layer (KNIRVGATEWAY)       │
│  - Key extraction & rate limiting                           │
│  - DDoS protection (no IP trust)                            │
│  - Request signature validation                             │
└──────────────────────┬──────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────────────┐
│                    Auth Service (in KNIRVGATEWAY)           │
│  - Cryptographic verification                               │
│  - Cache key metadata (5-min TTL)                           │
└──────────────────────┬──────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────────────┐
│          Policy Decision Point (PDP) (in KNIRVGATEWAY)      │
│  - Real-time risk scoring                                   │
│  - Scope & permission evaluation                            │
│  - Anomaly detection ML model                               │
└──────────────────────┬──────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────────────┐
│            Policy Enforcement Point (PEP) (in KNIRVGATEWAY) │
│  - Enforce field-level access                               │
│  - Audit logging                                            │
│  - Circuit breaker for revoked keys                         │
└─────────────────────────────────────────────────────────────┘
```

### 3.7. Key Security Controls

| Control      | Zero Trust Implementation                               |
|--------------|---------------------------------------------------------|
| **Storage**  | Argon2id hash ONLY; no plaintext secrets                |
| **Network**  | Explicit deny; no IP whitelisting by default            |
| **Rotation** | Forced 90-day expiry; automated rotation alerts         |
| **Revocation**| Instant global revocation via distributed cache (e.g., Redis) |
| **Monitoring**| Real-time anomaly detection; risk scoring               |
| **Access**   | Deny-by-default; explicit scope grants                  |
| **Encryption**| TLS 1.3 minimum; mTLS for internal services             |

## 4. Implementation Phases

### Phase 1: Foundation (Weeks 1-2)
- [ ] Deploy API Gateway (`KNIRVGATEWAY`) with key extraction middleware.
- [ ] Implement Argon2id hashing and the `api_keys` database schema.
- [ ] Create "deny by default" policy (all keys inactive initially).
- [ ] Set up `api_key_events` audit logging for all key events.

### Phase 2: Continuous Verification (Weeks 3-4)
- [ ] Build risk scoring engine with anomaly detection.
- [ ] Implement a real-time monitoring dashboard.
- [ ] Deploy IP/user agent tracking per key.
- [ ] Create automated revocation triggers based on risk score.

### Phase 3: Least Privilege (Weeks 5-6)
- [ ] Design and implement the granular scope system in the `scopes` column.
- [ ] Migrate existing keys to scoped permissions.
- [ ] Implement field-level access controls in the policy enforcement point.
- [ ] Deploy microsegmentation between API endpoints.

### Phase 4: Automation & Rotation (Weeks 7-8)
- [ ] Build automated key rotation (90-day max lifecycle).
- [ ] Create a self-service key management portal for users.
- [ ] Implement MFA for key generation/modification.
- [ ] Deploy CI/CD integration for key provisioning.

## 5. Continuous Monitoring Dashboard

Track these Zero Trust metrics:
- **Active key risk scores** (real-time heatmap)
- **Anomaly rate** (% of requests flagged)
- **Mean time to revoke** (for compromised keys)
- **Scope violation attempts** (blocked requests)
- **Key usage entropy** (unusual patterns)

## 6. Critical Zero Trust Pitfalls to Avoid

- **Inconsistent application**: Apply Zero Trust to ALL APIs, including internal ones.
- **Static security**: Update risk models continuously; never rely on yesterday's trust.
- **Ignoring dependencies**: Secure ALL APIs in the ecosystem; a chain is only as strong as its weakest link.
- **Network-based trust**: Never trust IP addresses or internal networks.
- **Shared secrets**: Each key must be unique; no shared credentials.

This architecture ensures that every API key is treated as a potential threat vector, with continuous verification at every layer. The system provides granular control, automated response to anomalies, and maintains a zero-standing-privilege posture.