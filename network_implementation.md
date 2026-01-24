# KNIRV Network Implementation Plan

## Comprehensive Implementation Strategy for D-TEN Ecosystem

**Version:** 1.0
**Date:** January 2026
**Status:** Consolidated Implementation Plan

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Ecosystem Architecture Overview](#2-ecosystem-architecture-overview)
3. [Zero Trust Security Foundation](#3-zero-trust-security-foundation)
4. [gRPC & Protocol Buffer Strategy](#4-grpc--protocol-buffer-strategy)
5. [Component Integration Patterns](#5-component-integration-patterns)
6. [STEM vs CORTEX Architecture](#6-stem-vs-cortex-architecture)
7. [Unified Implementation Roadmap](#7-unified-implementation-roadmap)
8. [Testing & Validation Strategy](#8-testing--validation-strategy)
9. [Migration Strategy](#9-migration-strategy)
10. [Success Metrics](#10-success-metrics)
11. [Risk Mitigation](#11-risk-mitigation)
12. [Appendix](#12-appendix)

---

## 1. Executive Summary

This document consolidates the implementation strategy for the KNIRV D-TEN (Decentralized Trusted Execution Network) ecosystem, merging Zero Trust security architecture, gRPC communication protocols, and component integration into a unified plan.

### Current State
- **Security:** Mock authentication, no authorization logic, permissive CORS, no mTLS
- **Communication:** HTTP/REST, WebSocket, custom P2P protocols with JSON serialization
- **Integration:** 16 loosely coupled applications with inconsistent interfaces

### Target State
- **Security:** Zero Trust Architecture with API keys, mTLS, continuous validation
- **Communication:** Full gRPC with consolidated protobuf definitions (4-10x performance improvement)
- **Integration:** Unified, enterprise-grade decentralized network with event-driven architecture

### Implementation Priority Order

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  PHASE 0: ZERO TRUST FOUNDATION (Weeks 1-2)                                 │
│  - Private PKI & mTLS infrastructure                                        │
│  - API key system with Argon2id hashing                                     │
│  - Deny-by-default security posture                                         │
└────────────────────────────────┬────────────────────────────────────────────┘
                                 ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│  PHASE 1: COMMUNICATION LAYER (Weeks 3-6)                                   │
│  - Consolidated protobuf definitions                                        │
│  - gRPC service implementations                                             │
│  - Health check services across all components                              │
└────────────────────────────────┬────────────────────────────────────────────┘
                                 ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│  PHASE 2: COMPONENT INTEGRATION (Weeks 7-12)                                │
│  - Unified authentication via ORACLE                                        │
│  - Event bus via ROUTER P2P layer                                           │
│  - IBC cross-chain communication                                            │
│  - NEXUS TEE integration                                                    │
└────────────────────────────────┬────────────────────────────────────────────┘
                                 ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│  PHASE 3: ADVANCED FEATURES (Weeks 13-20)                                   │
│  - CORTEX dynamic STEM loading                                              │
│  - HEART integration everywhere                                             │
│  - Unified dashboard & user experience                                      │
│  - Marketplace ecosystem                                                    │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Ecosystem Architecture Overview

### 2.1 Component Classification

#### Open Source Layer
| Component | Purpose | Tech | Integration Priority |
|-----------|---------|------|---------------------|
| **SDK** | Multi-language development toolkits (Go, TypeScript, Python) | Multi | HIGH |
| **WALLET** | Non-custodial wallet with XION Meta Accounts | Node.js | HIGH |
| **TESTNET** | Comprehensive testing environment | Multi | MEDIUM |
| **KNIRVANA** | Gaming gateway (Rust + TypeScript clients) | Rust/TS | LOW |

#### Network Layer
| Component | Purpose | Tech | Integration Priority |
|-----------|---------|------|---------------------|
| **CHAIN** | Memory-optimized blockchain with MCP capabilities | Go | CRITICAL |
| **GRAPH** | Knowledge graph with NRV (Network Resolution Vectors) | Go | CRITICAL |
| **ROUTER** | P2P network with Proof-of-Connectivity | Go | CRITICAL |
| **BASE** | Shared libraries (Go, Rust, TypeScript) | Multi | HIGH |

#### Private Layer
| Component | Purpose | Tech | Integration Priority |
|-----------|---------|------|---------------------|
| **NEXUS** | Distributed Validation Environment (DVE) | Go | CRITICAL |
| **ORACLE** | Cross-chain hub, governance, WebGUI | Go | CRITICAL |
| **HEART** | Heuristic Error Analysis Transformer | Go | HIGH |
| **SYNC** | Documentation and environment synchronization | Multi | MEDIUM |

#### Free Layer
| Component | Purpose | Tech | Integration Priority |
|-----------|---------|------|---------------------|
| **RAMP** | Neural Intelligence Model builder platform | Node.js | HIGH |
| **CONTROLLER** | NIM management and lifecycle platform | Rust/WASM | CRITICAL |
| **CORTEX** | Cognitive shell orchestrator (WASM) | Rust/WASM | CRITICAL |
| **STEM** | Small Language Model runtime (WASM module) | Rust/WASM | CRITICAL |

### 2.2 Communication Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  KNIRVGATEWAY   │    │   KNIRVNEXUS    │    │   KNIRVCHAIN    │
│   (Port 8000)   │◄──►│  (Backend Srv)  │◄──►│ (MCP Registry)  │
│   API Gateway   │    │   gRPC + REST   │    │   gRPC + HTTP   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │              ┌─────────────────┐              │
         └─────────────►│  KNIRVROUTER    │◄─────────────┘
                        │  P2P Network    │
                        │   (libp2p)      │
         ^              └─────────────────┘              ^
         │                       │                       |
         │              ┌─────────────────┐              |
         └─────────────►│    KNIRVGRAPH   │◄─────────────┘
                        │   (Port 7080)   │
                        │    Data Engine  │
                        └─────────────────┘
```

### 2.3 Current Communication Patterns

| Source | Destination | Current Protocol | Target Protocol |
|--------|-------------|------------------|-----------------|
| KNIRVGATEWAY | KNIRVNEXUS | HTTP/REST | gRPC |
| KNIRVGATEWAY | KNIRVGRAPH | HTTP/REST | gRPC |
| KNIRVROUTER | All Components | libp2p | libp2p + gRPC |
| Frontend | KNIRVGATEWAY | HTTP/WebSocket | HTTP/WebSocket + gRPC-Web |

---

## 3. Zero Trust Security Foundation

### 3.1 Core Zero Trust Principles

Zero Trust must be established BEFORE any other integration work proceeds. All communication channels must be secured.

- **Never Implicitly Trust**: Treat every request as potentially malicious
- **Verify Every Request**: Independent authentication, authorization, and validation per call
- **Least Privilege Access**: Minimal necessary permissions, scoped to specific endpoints
- **Continuous Validation**: Real-time monitoring with automatic revocation

### 3.2 Zero Trust Validation Pipeline

```
Client Request (KNIRVCONTROLLER)
    ↓
[Transport Layer (KNIRVROUTER)]
    ├── mTLS Authentication (Private PKI)
    ├── Path Certificate Creation & Validation
    └── zkTLS for Selective Disclosure
    ↓
[API Gateway (KNIRVGATEWAY)] ← First line of defense
    ↓
[Auth Service (part of KNIRVGATEWAY)] ← Trust verification
    ↓
[Policy Engine (part of KNIRVORACLE)] ← Authorization decision
    ↓
[Backend API (KNIRVBASE)] ← Final validation
```

### 3.3 API Key System Design

#### Key Format
```
sk_prod_f8k2m9n4.xQ9zR7tV2wY5uP1aS3dF6gH8jK0l
└──────┘└───────┘ └─────────────────────────────┘
 prefix   key_id              secret
```

- **Prefix**: Identifies environment and use case (`sk_prod_`, `sk_test_`)
- **Key ID**: Database lookup identifier (stored in plaintext)
- **Secret**: Cryptographically random 32-byte string (NEVER stored, only Argon2id hashed)

#### Key Generation Implementation
```go
func generateZeroTrustKey(db *sql.DB, userID string, scopes []string) (string, error) {
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

    // 3. Hash secret for storage (Argon2id - NEVER store plaintext)
    salt := make([]byte, 16)
    rand.Read(salt)
    secretHash := argon2.IDKey([]byte(secret), salt, 1, 64*1024, 4, 32)

    // 4. Store with Zero Trust metadata (90-day mandatory expiration)
    expiresAt := time.Now().Add(90 * 24 * time.Hour)
    _, err := db.Exec(`
        INSERT INTO api_keys (key_id, user_id, secret_hash, salt, scopes,
                              is_active, created_at, expires_at, risk_score)
        VALUES ($1, $2, $3, $4, $5, true, NOW(), $6, 0)
    `, keyID, userID, secretHash, salt, scopes, expiresAt)

    return "sk_prod_" + keyID + "." + secret, nil // Return ONCE only
}
```

### 3.4 Database Schema for Zero Trust

```sql
-- KNIRVBASE Schema (translate to KNIRVQL)
CREATE TABLE api_keys (
  key_id VARCHAR(12) PRIMARY KEY,
  user_id UUID NOT NULL,
  secret_hash BYTEA NOT NULL,              -- Argon2id hash
  salt BYTEA NOT NULL,
  scopes JSONB,                            -- Granular permissions
  is_active BOOLEAN DEFAULT false,         -- Deny by default
  created_at TIMESTAMP WITH TIME ZONE,
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
  last_used_at TIMESTAMP WITH TIME ZONE,
  usage_count INTEGER DEFAULT 0,

  -- Zero Trust monitoring
  risk_score INTEGER DEFAULT 0,            -- 0-100
  anomaly_flags JSONB,
  allowed_ips INET[],
  revoked_at TIMESTAMP WITH TIME ZONE,

  -- Rotation metadata
  parent_key_id VARCHAR(12),
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
  risk_factors JSONB
);
```

### 3.5 mTLS and Private PKI

For consortium networks operating without internet connectivity:

#### Private Certificate Authority Trust Model

1. **Root CA (KNIRVORACLE)**: Single consortium Root CA signs certificates for all nodes
2. **Identity Certificates**: Every node receives:
   - Private Key (kept secret)
   - Certificate (signed by Root CA)
   - Root CA Public Certificate (to verify others)

#### Server-Side mTLS Configuration
```go
// Load consortium's Root CA certificate
certPool := x509.NewCertPool()
ca, _ := ioutil.ReadFile("consortium-ca.crt")
certPool.AppendCertsFromPEM(ca)

// Load server's certificate and private key
serverCert, _ := tls.LoadX509KeyPair("server.crt", "server.key")

// Create TLS credentials requiring client certificate
creds := credentials.NewTLS(&tls.Config{
    Certificates: []tls.Certificate{serverCert},
    ClientAuth:   tls.RequireAndVerifyClientCert,  // mTLS enabled
    ClientCAs:    certPool,
})

s := grpc.NewServer(grpc.Creds(creds))
```

#### Client-Side mTLS Configuration
```go
clientCert, _ := tls.LoadX509KeyPair("client.crt", "client.key")
certPool := x509.NewCertPool()
ca, _ := ioutil.ReadFile("consortium-ca.crt")
certPool.AppendCertsFromPEM(ca)

creds := credentials.NewTLS(&tls.Config{
    Certificates: []tls.Certificate{clientCert},
    RootCAs:      certPool,
    ServerName:   "member-a-saas.consortium.local",
})

conn, _ := grpc.Dial("10.0.0.5:50051", grpc.WithTransportCredentials(creds))
```

### 3.6 Layered Security Model

- **L4 Security (mTLS)**: Ensures encrypted, authenticated connections between consortium nodes
- **L7 Security (URI Path Certificates)**: gRPC requests carry path certificates proving verified network pathways
- **zkTLS Integration**: Zero-Knowledge proofs for privacy-preserving connectivity verification

### 3.7 Certificate Management Without Internet

Use **Short-Lived Certificates** (24-hour validity):
1. KNIRVROUTERS use Proof-of-Connectivity to request new certificates from KNIRVORACLE
2. Compromised routers are "burned" by stopping certificate issuance (effective within 24 hours)

### 3.8 Zero Trust Security Controls Summary

| Control | Implementation |
|---------|---------------|
| **Storage** | Argon2id hash ONLY; no plaintext secrets |
| **Network** | Explicit deny; no IP whitelisting by default |
| **Rotation** | Forced 90-day expiry; automated alerts |
| **Revocation** | Instant global revocation via distributed cache |
| **Monitoring** | Real-time anomaly detection; risk scoring |
| **Access** | Deny-by-default; explicit scope grants |
| **Encryption** | TLS 1.3 minimum; mTLS for internal services |

### 3.9 Zero Trust Authorization Model

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

---

## 4. gRPC & Protocol Buffer Strategy

### 4.1 Performance Benefits

Migrating from HTTP/REST with JSON to gRPC with Protocol Buffers provides:

| Metric | HTTP/REST | gRPC | Improvement |
|--------|-----------|------|-------------|
| Average Latency | ~50ms | ~5-12ms | 4-10x faster |
| Bandwidth/Request | ~2MB | ~200KB | 60-80% reduction |
| CPU (Serialization) | High | Low | 50-70% reduction |
| Runtime Type Errors | Common | Near-zero | 90% reduction |

### 4.2 Consolidated Protobuf Structure

```
shared-proto/
├── common/
│   └── v1/
│       ├── base.proto          # Base messages (Request/Response/Status)
│       ├── authentication.proto # Auth messages
│       └── errors.proto         # Error definitions
├── model/
│   └── v1/
│       ├── model.proto          # Model compilation and runtime
│       └── inference.proto      # Inference requests/responses
├── cortex/
│   └── v1/
│       ├── cortex.proto         # Cognitive capabilities
│       └── capability.proto     # Capability definitions
├── memory/
│   └── v1/
│       ├── memory.proto         # Memory management
│       └── storage.proto        # Storage interfaces
├── lora/
│   └── v1/
│       ├── lora.proto           # LoRA adapter definitions
│       └── training.proto       # Training related messages
├── graph/
│   └── v1/
│       ├── graph.proto          # Graph structure
│       ├── node.proto           # Node definitions
│       └── relationship.proto   # Relationship definitions
├── agent/
│   └── v1/
│       ├── agent.proto          # Agent lifecycle
│       └── task.proto           # Task management
└── network/
    └── v1/
        ├── p2p.proto            # P2P networking
        ├── blockchain.proto     # Blockchain operations
        └── transaction.proto    # Transaction handling
```

### 4.3 Critical Shared Types

#### Base Types (common/v1/base.proto)
```protobuf
syntax = "proto3";
package knirv.common.v1;

import "google/protobuf/timestamp.proto";

message BaseRequest {
  string request_id = 1;
  google.protobuf.Timestamp timestamp = 2;
  string client_id = 3;
  map<string, string> metadata = 4;
}

message BaseResponse {
  string request_id = 1;
  bool success = 2;
  string error_message = 3;
  map<string, string> metadata = 4;
}

message Status {
  enum Code {
    UNKNOWN = 0;
    OK = 200;
    CREATED = 201;
    BAD_REQUEST = 400;
    UNAUTHORIZED = 401;
    FORBIDDEN = 403;
    NOT_FOUND = 404;
    INTERNAL_ERROR = 500;
  }
  Code code = 1;
  string message = 2;
  map<string, string> details = 3;
}

message PaginatedResponse {
  int32 page = 1;
  int32 page_size = 2;
  int64 total_count = 3;
  bool has_next = 4;
}
```

#### Authentication Types (common/v1/authentication.proto)
```protobuf
syntax = "proto3";
package knirv.common.v1;

message AuthenticationHeader {
  string token = 1;
  string token_type = 2;  // "bearer", "api_key"
  google.protobuf.Timestamp expires_at = 3;
}

message UserIdentity {
  string user_id = 1;
  string username = 2;
  string email = 3;
  repeated string roles = 4;
  map<string, string> attributes = 5;
}

message ServiceIdentity {
  string service_id = 1;
  string service_name = 2;
  string version = 3;
  repeated string permissions = 4;
}
```

#### Consolidated LoRA Adapter (lora/v1/lora.proto)
```protobuf
syntax = "proto3";
package knirv.lora.v1;

import "google/protobuf/timestamp.proto";

message LoRAAdapter {
  string id = 1;
  string name = 2;
  string description = 3;
  string base_model_id = 4;
  bytes weights = 5;  // Binary data (NOT repeated float)
  int32 rank = 6;
  float alpha = 7;
  repeated string target_modules = 8;
  LoRAConfig config = 9;
  google.protobuf.Timestamp created_at = 10;
  google.protobuf.Timestamp updated_at = 11;
  map<string, string> metadata = 12;
}

message LoRAConfig {
  int32 r = 1;
  int32 lora_alpha = 2;
  float lora_dropout = 3;
  string bias = 4;
  string task_type = 5;
}
```

#### Unified SemanticTriple (graph/v1/relationship.proto)
```protobuf
syntax = "proto3";
package knirv.graph.v1;

import "google/protobuf/timestamp.proto";

message SemanticTriple {
  string id = 1;
  string subject = 2;
  string predicate = 3;
  string object = 4;
  float confidence = 5;
  string source = 6;
  google.protobuf.Timestamp timestamp = 7;
  map<string, string> metadata = 8;
}
```

### 4.4 gRPC Service Definitions

#### Gateway Services (KNIRVGATEWAY)
```protobuf
service OrchestrationService {
  rpc CreateAgent(CreateAgentRequest) returns (CreateAgentResponse);
  rpc GetAgent(GetAgentRequest) returns (GetAgentResponse);
  rpc UpdateAgent(UpdateAgentRequest) returns (UpdateAgentResponse);
  rpc DeleteAgent(DeleteAgentRequest) returns (DeleteAgentResponse);
  rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse);

  rpc DeployTask(DeployTaskRequest) returns (DeployTaskResponse);
  rpc GetTaskStatus(GetTaskStatusRequest) returns (GetTaskStatusResponse);
  rpc CancelTask(CancelTaskRequest) returns (CancelTaskResponse);
}

service GatewayProxyService {
  rpc ProxyRequest(ProxyRequest) returns (ProxyResponse);
  rpc StreamProxy(stream StreamProxyRequest) returns (stream StreamProxyResponse);
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

#### Nexus Services (KNIRVNEXUS)
```protobuf
service AgentManagementService {
  rpc CreateAgent(CreateAgentRequest) returns (CreateAgentResponse);
  rpc GetAgent(GetAgentRequest) returns (GetAgentResponse);
  rpc UpdateAgent(UpdateAgentRequest) returns (UpdateAgentResponse);
  rpc DeleteAgent(DeleteAgentRequest) returns (DeleteAgentResponse);
  rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse);

  rpc StartAgent(StartAgentRequest) returns (StartAgentResponse);
  rpc StopAgent(StopAgentRequest) returns (StopAgentResponse);
  rpc RestartAgent(RestartAgentRequest) returns (RestartAgentResponse);
}

service TaskExecutionService {
  rpc ExecuteTask(ExecuteTaskRequest) returns (ExecuteTaskResponse);
  rpc StreamTaskExecution(stream StreamTaskRequest) returns (stream StreamTaskResponse);
  rpc GetTaskResult(GetTaskResultRequest) returns (GetTaskResultResponse);
  rpc CancelExecution(CancelExecutionRequest) returns (CancelExecutionResponse);
}

service ControllerCommunicationService {
  rpc RegisterController(RegisterControllerRequest) returns (RegisterControllerResponse);
  rpc UnregisterController(UnregisterControllerRequest) returns (UnregisterControllerResponse);
  rpc SendHeartbeat(SendHeartbeatRequest) returns (SendHeartbeatResponse);
  rpc UpdateControllerStatus(UpdateControllerStatusRequest) returns (UpdateControllerStatusResponse);
}
```

#### Graph Services (KNIRVGRAPH)
```protobuf
service DataQueryService {
  rpc QueryData(QueryDataRequest) returns (QueryDataResponse);
  rpc StreamQuery(stream StreamQueryRequest) returns (stream StreamQueryResponse);
  rpc GetNode(GetNodeRequest) returns (GetNodeResponse);
  rpc GetRelationships(GetRelationshipsRequest) returns (GetRelationshipsResponse);
}

service GraphIngestionService {
  rpc IngestNode(IngestNodeRequest) returns (IngestNodeResponse);
  rpc IngestRelationship(IngestRelationshipRequest) returns (IngestRelationshipResponse);
  rpc BatchIngest(BatchIngestRequest) returns (BatchIngestResponse);
  rpc UpdateNode(UpdateNodeRequest) returns (UpdateNodeResponse);
}

service AnalyticsService {
  rpc GetAnalytics(GetAnalyticsRequest) returns (GetAnalyticsResponse);
  rpc StreamAnalytics(stream StreamAnalyticsRequest) returns (stream StreamAnalyticsResponse);
  rpc GenerateReport(GenerateReportRequest) returns (GenerateReportResponse);
}
```

#### Router Services (KNIRVROUTER)
```protobuf
service P2PNetworkService {
  rpc ConnectToPeer(ConnectToPeerRequest) returns (ConnectToPeerResponse);
  rpc DisconnectFromPeer(DisconnectFromPeerRequest) returns (DisconnectFromPeerResponse);
  rpc ListPeers(ListPeersRequest) returns (ListPeersResponse);
  rpc SendMessage(SendMessageRequest) returns (SendMessageResponse);
  rpc StreamMessages(stream StreamMessagesRequest) returns (stream StreamMessagesResponse);
}

service BlockPropagationService {
  rpc PropagateBlock(PropagateBlockRequest) returns (PropagateBlockResponse);
  rpc RequestBlock(RequestBlockRequest) returns (RequestBlockResponse);
  rpc SyncBlocks(SyncBlocksRequest) returns (stream SyncBlocksResponse);
}

service TransactionRelayService {
  rpc RelayTransaction(RelayTransactionRequest) returns (RelayTransactionResponse);
  rpc GetTransaction(GetTransactionRequest) returns (GetTransactionResponse);
  rpc GetTransactionStatus(GetTransactionStatusRequest) returns (GetTransactionStatusResponse);
}
```

#### Chain Services (KNIRVCHAIN)
```protobuf
service MCPRegistryService {
  rpc RegisterModel(RegisterModelRequest) returns (RegisterModelResponse);
  rpc GetModel(GetModelRequest) returns (GetModelResponse);
  rpc ListModels(ListModelsRequest) returns (ListModelsResponse);
  rpc UpdateModel(UpdateModelRequest) returns (UpdateModelResponse);
  rpc DeleteModel(DeleteModelRequest) returns (DeleteModelResponse);
}

service SkillManagementService {
  rpc CreateSkill(CreateSkillRequest) returns (CreateSkillResponse);
  rpc GetSkill(GetSkillRequest) returns (GetSkillResponse);
  rpc ListSkills(ListSkillsRequest) returns (ListSkillsResponse);
  rpc UpdateSkill(UpdateSkillRequest) returns (UpdateSkillResponse);
  rpc DeleteSkill(DeleteSkillRequest) returns (DeleteSkillResponse);
}

service InferenceService {
  rpc ExecuteInference(ExecuteInferenceRequest) returns (ExecuteInferenceResponse);
  rpc StreamInference(stream StreamInferenceRequest) returns (stream StreamInferenceResponse);
  rpc GetInferenceStatus(GetInferenceStatusRequest) returns (GetInferenceStatusResponse);
  rpc CancelInference(CancelInferenceRequest) returns (CancelInferenceResponse);
}
```

---

## 5. Component Integration Patterns

### 5.1 Event-Driven Architecture

Replace direct HTTP calls with an event bus using ROUTER's P2P layer.

#### Event Schema (ProtoBuf)
```protobuf
message NetworkEvent {
  string event_type = 1;      // "skill_minted", "error_resolved", "model_deployed"
  string source_component = 2; // "CHAIN", "GRAPH", "CONTROLLER"
  bytes payload = 3;           // Component-specific data
  uint64 timestamp = 4;
  string trace_id = 5;         // For distributed tracing
}
```

#### Event Flow Examples

**Error Resolution Flow:**
```
CONTROLLER → ErrorSubmitted → GRAPH
GRAPH → ErrorClusterUpdated → [CHAIN, KNIRVANA]
CORTEX → HEARTQuerySent → HEART
HEART → HeuristicResponseSent → CORTEX
CORTEX → SkillRecommendation → CONTROLLER
```

**Skill Minting Flow:**
```
CONTROLLER → SkillMintRequest → CHAIN
CHAIN → SkillMinted → [GRAPH, NEXUS, CONTROLLER]
GRAPH → SkillIndexed → CONTROLLER
NEXUS → ValidationQueued → CONTROLLER
```

### 5.2 Unified Authentication & Authorization

Centralized JWT-based auth via ORACLE with UDC delegation.

#### Authentication Flow
```
1. User logs into CONTROLLER/WALLET (XION Meta Account)
2. CONTROLLER/WALLET requests JWT from ORACLE /auth/login
3. ORACLE issues JWT with role claims
4. User presents JWT to any component
5. Component validates JWT via ORACLE /auth/verify
6. Component checks UDC for delegated permissions
```

#### UDC (User Delegation Certificate) Format
```protobuf
message UDC {
  string issuer_address = 1;       // WALLET address
  string delegate_address = 2;     // CONTROLLER/NIM address
  repeated string permissions = 3;  // ["skill:invoke", "model:deploy"]
  uint64 expiration = 4;
  bytes signature = 5;             // Signed by issuer
}
```

#### Integration Points
- **WALLET**: Issues UDCs for CONTROLLER NIMs
- **ORACLE**: Validates all JWTs and UDCs
- **CONTROLLER**: Presents UDCs when invoking skills
- **CHAIN**: Verifies UDCs before skill execution
- **NEXUS**: Checks UDCs for validation permissions

### 5.3 Cross-Chain Communication (IBC)

Implement Cosmos IBC for cross-chain token transfers and state synchronization.

#### IBC Channel Configuration
```yaml
channels:
  - source: ORACLE
    destination: CHAIN
    purpose: NRN token transfers for skill invocations

  - source: CHAIN
    destination: GRAPH
    purpose: Skill confirmations and error cluster updates

  - source: ORACLE
    destination: GRAPH
    purpose: NRN rewards for error resolution

  - source: GRAPH
    destination: NEXUS
    purpose: Validation task assignments
```

### 5.4 Standardized API Contracts

Dual protocol support (REST for external, gRPC for internal).

#### API Gateway Routing (ORACLE WebGUI)
```yaml
# External Routes (REST - public)
/api/v1/chain/*      → CHAIN (REST proxy)
/api/v1/graph/*      → GRAPH (REST proxy)
/api/v1/nexus/*      → NEXUS (REST proxy)
/api/v1/controller/* → CONTROLLER (REST proxy)

# Internal Routes (gRPC - service-to-service)
grpc://chain.knirv.internal:9090
grpc://graph.knirv.internal:9091
grpc://nexus.knirv.internal:9092
```

### 5.5 Distributed Tracing & Observability

OpenTelemetry integration across all services.

#### Trace Example
```
Trace: skill_invocation_abc123
  Span 1: CONTROLLER.invoke_skill (10ms)
    ├─ Span 2: WALLET.sign_transaction (5ms)
    ├─ Span 3: CHAIN.submit_transaction (50ms)
    │   └─ Span 4: NEXUS.validate_skill (200ms)
    │       └─ Span 5: GRAPH.query_similar_skills (30ms)
    └─ Span 6: ORACLE.burn_nrn_fees (15ms)

Total Duration: 310ms
```

#### Metrics to Track
- End-to-end latency per operation type
- Component-level error rates
- NRN token flow between chains
- Validation success rates in NEXUS
- P2P network connectivity (ROUTER)

---

## 6. STEM vs CORTEX Architecture

### 6.1 Critical Distinction

**These are NOT separate applications but complementary WASM modules with strict separation of concerns.**

### 6.2 STEM (stem.wasm)

**Purpose:** Pure Small Language Model (SLM) inference runtime

**Responsibilities:**
- Load and execute compiled SLM weights
- Perform forward passes through neural network
- Apply quantization and optimization
- **NO error handling**
- **NO orchestration logic**
- **NO LoRA adapter management**

**Exports:**
```rust
#[no_mangle]
pub extern "C" fn stem_load_weights(ptr: *const u8, len: usize) -> bool;

#[no_mangle]
pub extern "C" fn stem_inference(input_ptr: *const u8, input_len: usize) -> u64;

#[no_mangle]
pub extern "C" fn stem_get_model_info() -> *const u8;
```

**Acquisition Tiers:**
- **Free**: Pre-compiled stem.wasm with 1-7B parameter models
- **Pro**: Custom SLM compilation with user-provided weights
- **Enterprise**: Fully optimized stem.wasm with hardware-specific tuning

### 6.3 CORTEX (cortex.wasm)

**Purpose:** Cognitive shell orchestrator that loads and manages stem.wasm

**Responsibilities:**
- Load stem.wasm as dynamic WASM module
- Apply LoRA adapters to modify stem.wasm behavior
- Handle ALL errors from stem.wasm via HEART integration
- Manage memory policies and context windows
- Orchestrate tool calls and multi-step reasoning
- Provide external inference fallbacks during beta

**Implementation:**
```rust
pub struct Cortex {
    stem_instance: WasmInstance,  // Dynamically loaded stem.wasm
    lora_adapters: Vec<LoRAAdapter>,
    heart_client: HEARTClient,
    memory_policy: MemoryPolicy,
}

impl Cortex {
    pub async fn run_cognitive_task(&mut self, input: InferenceInput) -> Result<InferenceOutput> {
        // 1. Apply LoRA adapters to stem.wasm
        self.apply_lora_to_stem()?;

        // 2. Call stem.wasm for inference
        let result = self.stem_instance.call_inference(input.prompt);

        // 3. If stem.wasm errors, query HEART for analysis
        if let Err(e) = result {
            let heuristic = self.heart_client.analyze_error(e).await?;
            return self.handle_error_with_heuristic(e, heuristic);
        }

        // 4. Return successful result
        Ok(result?)
    }
}
```

**Current Status:** Complete but statically links stem.wasm (should be dynamic)

### 6.4 Technical Comparison

| Aspect | STEM (stem.wasm) | CORTEX (cortex.wasm) |
|--------|------------------|----------------------|
| **Purpose** | Pure ML inference | Cognitive orchestration |
| **Errors** | Surfaces errors to caller | Handles errors with HEART |
| **LoRA** | Receives LoRA modifications | Applies LoRA adapters |
| **Memory** | Model weights only | Context + tools + memory |
| **Size** | 50-200MB (model-dependent) | 290KB base + plugins |
| **Upgrade** | Swap for different SLM | Upgrade orchestration logic |
| **Acquisition** | RAMP marketplace | CONTROLLER downloads |

### 6.5 Why This Matters

**Separation of Concerns:**
1. **stem.wasm** = Pure ML inference (fast, deterministic, replaceable)
2. **cortex.wasm** = Intelligent orchestration (adaptive, error-handling, context-aware)

**Upgrade Paths:**
- Users can swap stem.wasm for different SLM sizes without changing cortex.wasm
- cortex.wasm can be updated for better orchestration without retraining models
- LoRA adapters modify stem.wasm behavior without recompilation

---

## 7. Unified Implementation Roadmap

### Phase 0: Zero Trust Foundation (Weeks 1-2)

**Goal:** Establish security infrastructure before any other work.

| Task | Component | Status |
|------|-----------|--------|
| Establish Private PKI with Root CA | KNIRVORACLE | [ ] |
| Generate identity certificates for all nodes | All | [ ] |
| Configure gRPC servers/clients with mTLS | All | [ ] |
| Implement Argon2id key hashing | KNIRVGATEWAY | [ ] |
| Deploy `api_keys` database schema | KNIRVBASE | [ ] |
| Create deny-by-default policy | KNIRVGATEWAY | [ ] |
| Set up `api_key_events` audit logging | KNIRVGATEWAY | [ ] |
| Deploy key extraction middleware | KNIRVROUTER | [ ] |

**Success Criteria:**
- All inter-service communication uses mTLS
- API keys cannot be used without explicit activation
- All key events logged for audit

### Phase 1: Communication Layer (Weeks 3-6)

**Goal:** Establish high-performance gRPC communication.

#### Weeks 3-4: Protobuf Foundation
| Task | Component | Status |
|------|-----------|--------|
| Create consolidated protobuf structure | shared-proto | [ ] |
| Define base common types | shared-proto | [ ] |
| Implement gRPC server/client utilities | KNIRVBASE | [ ] |
| Set up protobuf compilation pipeline | Build System | [ ] |
| Add health check services | All | [ ] |

#### Weeks 5-6: Core Services
| Task | Component | Status |
|------|-----------|--------|
| Implement Gateway ↔ Nexus gRPC | GATEWAY/NEXUS | [ ] |
| Implement Gateway ↔ Graph gRPC | GATEWAY/GRAPH | [ ] |
| Implement NEXUS Agent Management service | NEXUS | [ ] |
| Implement NEXUS Task Execution service | NEXUS | [ ] |
| Implement GRAPH Data Query service | GRAPH | [ ] |

**Success Criteria:**
- All internal communication uses gRPC
- Latency reduced by 4-10x
- Type-safe communication with zero runtime errors

### Phase 2: Component Integration (Weeks 7-12)

**Goal:** Enable seamless cross-component workflows.

#### Weeks 7-8: Continuous Security Verification
| Task | Component | Status |
|------|-----------|--------|
| Build risk scoring engine | KNIRVORACLE | [ ] |
| Implement real-time monitoring dashboard | KNIRVGATEWAY | [ ] |
| Deploy IP/user agent tracking per key | KNIRVGATEWAY | [ ] |
| Create automated revocation triggers | KNIRVORACLE | [ ] |

#### Weeks 9-10: Unified Authentication & Event Bus
| Task | Component | Status |
|------|-----------|--------|
| Implement JWT authentication in ORACLE | KNIRVORACLE | [ ] |
| Add JWT validation middleware | CHAIN/GRAPH/NEXUS | [ ] |
| Integrate UDC delegation in CONTROLLER | CONTROLLER | [ ] |
| Define ProtoBuf event schemas | shared-proto | [ ] |
| Implement event publishing in ROUTER | ROUTER | [ ] |
| Add event subscribers | CHAIN/GRAPH/CONTROLLER/NEXUS | [ ] |

#### Weeks 11-12: TEE & IBC Integration
| Task | Component | Status |
|------|-----------|--------|
| Complete Intel SGX integration | NEXUS | [ ] |
| Implement attestation verification | CHAIN | [ ] |
| Add TEE-backed validation proofs | GRAPH | [ ] |
| Set up IBC relayer | ORACLE/CHAIN/GRAPH | [ ] |
| Implement NRN token transfers via IBC | All Chains | [ ] |
| Enable cross-chain state queries | All Chains | [ ] |

**Success Criteria:**
- Users can log in once via CONTROLLER and access all services
- Skill minting on CHAIN triggers automatic indexing in GRAPH
- NEXUS produces cryptographically verifiable validation proofs
- NRN flows seamlessly between chains via IBC
- Cross-chain transactions complete in < 5 seconds

### Phase 3: Advanced Features (Weeks 13-20)

**Goal:** Enable enterprise use cases and ecosystem growth.

#### Weeks 13-14: Least Privilege & Automation
| Task | Component | Status |
|------|-----------|--------|
| Design granular scope system | KNIRVGATEWAY | [ ] |
| Migrate existing keys to scoped permissions | All | [ ] |
| Implement field-level access controls | KNIRVGATEWAY | [ ] |
| Deploy microsegmentation | KNIRVGATEWAY | [ ] |
| Build automated key rotation (90-day max) | KNIRVORACLE | [ ] |
| Create self-service key management portal | KNIRVGATEWAY | [ ] |
| Implement MFA for key generation | KNIRVGATEWAY | [ ] |

#### Weeks 15-16: CORTEX & HEART Integration
| Task | Component | Status |
|------|-----------|--------|
| Separate stem.wasm from cortex.wasm build | CORTEX/STEM | [ ] |
| Implement WebAssembly.instantiate() in cortex | CORTEX | [ ] |
| Create stem.wasm marketplace in RAMP | RAMP | [ ] |
| Enable hot-swapping of stem.wasm | CONTROLLER | [ ] |
| Add HEART client to CONTROLLER | CONTROLLER | [ ] |
| Integrate HEART error analysis in GRAPH | GRAPH | [ ] |
| Connect HEART recommendations to CHAIN | CHAIN | [ ] |

#### Weeks 17-18: User Experience
| Task | Component | Status |
|------|-----------|--------|
| Consolidate ORACLE WebGUI as primary interface | ORACLE | [ ] |
| Embed WALLET management into WebGUI | ORACLE | [ ] |
| Add CONTROLLER integration to WebGUI | ORACLE | [ ] |
| Create unified NIM lifecycle view | ORACLE | [ ] |
| RAMP exports directly to CONTROLLER format | RAMP | [ ] |
| CONTROLLER auto-registers models with ORACLE | CONTROLLER | [ ] |

#### Weeks 19-20: Ecosystem & SDK
| Task | Component | Status |
|------|-----------|--------|
| WebSocket connections to GRAPH for KNIRVANA | KNIRVANA | [ ] |
| Unified SDK supports all network operations | SDK | [ ] |
| Code generation from ProtoBuf contracts | SDK | [ ] |
| End-to-end examples for each language | SDK | [ ] |
| RAMP marketplace for stem.wasm modules | RAMP | [ ] |
| CONTROLLER marketplace for cortex.wasm | CONTROLLER | [ ] |
| LoRA adapter marketplace on GRAPH | GRAPH | [ ] |

**Success Criteria:**
- New users can create and deploy a NIM in < 5 minutes
- All network functionality accessible from ORACLE WebGUI
- Users can download and swap stem.wasm modules from RAMP
- All errors automatically analyzed by HEART across network
- Developers can build on KNIRV using any supported language
- Marketplace has 100+ available models and skills

---

## 8. Testing & Validation Strategy

### 8.1 Integration Test Suite

#### Component Integration Tests
```yaml
test_wallet_to_chain_flow:
  steps:
    - Create wallet in WALLET component
    - Request JWT from ORACLE
    - Submit skill to CHAIN using JWT
    - Verify skill appears in GRAPH
    - Check NRN balance decreased
  expected_duration: < 10 seconds

test_error_to_skill_flow:
  steps:
    - Submit ErrorNode to GRAPH from CONTROLLER
    - Wait for cluster analysis
    - Receive skill recommendations from HEART
    - Mint recommended skill on CHAIN
    - Deploy skill to CONTROLLER
  expected_duration: < 30 seconds
```

#### End-to-End Workflow Tests
1. New user onboarding flow
2. NIM creation and deployment flow
3. Skill invocation with NRN payment flow
4. Error resolution with LoRA mining flow
5. Cross-chain token transfer flow

### 8.2 Performance Benchmarks

#### Throughput Targets
- CHAIN: 1000 transactions/second
- GRAPH: 500 error ingestions/second
- NEXUS: 50 validation tasks/second
- ROUTER: 10,000 concurrent P2P connections

#### Latency Targets
- Authentication: < 100ms
- Skill invocation: < 500ms
- Validation proof: < 5 seconds
- Cross-chain transfer: < 3 seconds

### 8.3 Chaos Engineering

#### Failure Scenarios
- NEXUS goes offline during validation
- ROUTER network partition
- ORACLE database failover
- CHAIN consensus stall
- GRAPH memory spike

#### Expected Behaviors
- Graceful degradation (not total failure)
- Automatic retries with exponential backoff
- Event replay from queue
- User-facing error messages
- Automated alerts to operations team

### 8.4 Zero Trust Monitoring Dashboard

Track these metrics:
- Active key risk scores (real-time heatmap)
- Anomaly rate (% of requests flagged)
- Mean time to revoke (for compromised keys)
- Scope violation attempts (blocked requests)
- Key usage entropy (unusual patterns)

---

## 9. Migration Strategy

### 9.1 Incremental Protocol Migration

1. **Parallel Implementation**: Run gRPC alongside existing HTTP services
2. **Feature Flags**: Use feature flags to switch between protocols
3. **Gradual Migration**: Migrate one service endpoint at a time
4. **Backward Compatibility**: Maintain HTTP endpoints during transition
5. **Monitoring**: Compare performance and reliability metrics

### 9.2 API Versioning

```yaml
# CHAIN API versioning
/api/v1/skills/*  # Legacy REST API (deprecated)
/api/v2/skills/*  # New REST API with events
grpc://chain.knirv.internal:9090  # Internal gRPC
```

**Migration Timeline:**
- Weeks 1-6: v2 APIs available, v1 marked deprecated
- Weeks 7-12: All new features only in v2
- Weeks 13-16: v1 APIs give deprecation warnings
- Weeks 17-20: v1 APIs removed

### 9.3 Data Migration

#### CHAIN → GRAPH Skill Synchronization
```bash
# One-time historical sync
knirv-cli sync skills --from-chain --to-graph --since=genesis

# Ongoing event-based sync (automatically handled by event bus)
```

#### CONTROLLER → WALLET UDC Migration
```bash
# Generate UDCs for all existing NIMs
knirv-cli migrate udcs --controller-db=./controller.db --wallet-rpc=https://wallet.knirv.network
```

### 9.4 Configuration Management

#### Environment-Specific Configs
```yaml
# config/production.yaml
services:
  oracle:
    url: https://oracle.knirv.network
    grpc_port: 9090
  chain:
    url: https://chain.knirv.network
    grpc_port: 9090
  graph:
    url: https://graph.knirv.network
    grpc_port: 9091
  router:
    bootstrap_peers:
      - /ip4/54.123.45.67/tcp/4001/p2p/QmExample1
      - /ip4/54.123.45.68/tcp/4001/p2p/QmExample2

# config/development.yaml
services:
  oracle:
    url: http://localhost:1317
    grpc_port: 9090
  # ... local URLs
```

---

## 10. Success Metrics

### 10.1 Technical Metrics

| Metric | Baseline | Target | Timeline |
|--------|----------|--------|----------|
| Cross-component latency | 2-5s | < 500ms | Week 6 |
| API error rate | 5-10% | < 1% | Week 4 |
| Event delivery success | 85% | 99.9% | Week 10 |
| TEE validation coverage | 0% | 80% | Week 12 |
| Integration test coverage | 30% | 90% | Week 16 |
| gRPC migration completion | 0% | 100% | Week 6 |

### 10.2 Security Metrics

| Metric | Baseline | Target | Timeline |
|--------|----------|--------|----------|
| mTLS coverage | 0% | 100% | Week 2 |
| API keys with Argon2id | 0% | 100% | Week 2 |
| Keys with 90-day expiry | 0% | 100% | Week 14 |
| Mean time to revoke | N/A | < 5 min | Week 8 |
| Zero standing privilege | No | Yes | Week 14 |

### 10.3 User Experience Metrics

| Metric | Baseline | Target | Timeline |
|--------|----------|--------|----------|
| Time to first NIM deployment | 30+ min | < 5 min | Week 18 |
| Auth token refresh failures | 20% | < 2% | Week 10 |
| Cross-chain tx confirmation | 15-30s | < 5s | Week 12 |
| Mobile responsiveness score | 60 | 95+ | Week 18 |

### 10.4 Business Metrics

| Metric | Baseline | Target | Timeline |
|--------|----------|--------|----------|
| Active NIMs on network | 100 | 10,000 | Week 20 |
| Daily skill invocations | 500 | 50,000 | Week 20 |
| Marketplace revenue | $0 | $10k/month | Week 20 |
| Developer SDK downloads | 50 | 5,000 | Week 20 |

---

## 11. Risk Mitigation

### 11.1 Technical Risks

**Risk:** NEXUS TEE integration delays block production validation
- **Mitigation:** Implement software-based TEE simulation for Phase 2
- **Fallback:** Manual validation by trusted operators

**Risk:** IBC implementation complexity delays cross-chain features
- **Mitigation:** Start with simple token transfers before complex state sync
- **Fallback:** Custom bridge contracts as interim solution

**Risk:** gRPC migration causes service disruptions
- **Mitigation:** Run parallel HTTP/gRPC services during transition
- **Fallback:** Quick rollback to HTTP if issues arise

**Risk:** Event bus performance bottleneck
- **Mitigation:** Use ROUTER's proven P2P layer instead of custom message queue
- **Fallback:** Direct HTTP calls with eventual consistency

### 11.2 Security Risks

**Risk:** Certificate management without internet
- **Mitigation:** Short-lived 24-hour certificates with automated renewal
- **Fallback:** Manual certificate distribution for critical nodes

**Risk:** API key compromise
- **Mitigation:** Real-time anomaly detection with automatic revocation
- **Fallback:** Manual key revocation with clear escalation path

### 11.3 User Experience Risks

**Risk:** Auth token management confuses users
- **Mitigation:** Auto-refresh tokens in background
- **Fallback:** Clear error messages with re-auth flow

**Risk:** Cross-component workflows too complex
- **Mitigation:** Guided wizards in ORACLE WebGUI
- **Fallback:** Comprehensive documentation and video tutorials

---

## 12. Appendix

### 12.1 Component Dependency Matrix

```yaml
WALLET:
  depends_on: [ORACLE]
  consumed_by: [CONTROLLER, KNIRVANA]

CONTROLLER:
  depends_on: [WALLET, CHAIN, GRAPH, ORACLE]
  consumed_by: [USERS]

CHAIN:
  depends_on: [ORACLE, NEXUS, ROUTER]
  consumed_by: [CONTROLLER, SDK, GRAPH]

GRAPH:
  depends_on: [ORACLE, CHAIN, ROUTER]
  consumed_by: [CONTROLLER, KNIRVANA, HEART]

NEXUS:
  depends_on: [CHAIN, ORACLE]
  consumed_by: [CHAIN]

ORACLE:
  depends_on: [ROUTER]
  consumed_by: [ALL_COMPONENTS]

ROUTER:
  depends_on: []
  consumed_by: [ALL_COMPONENTS]

CORTEX:
  depends_on: [HEART]
  consumed_by: [CONTROLLER]

STEM:
  depends_on: []
  consumed_by: [CORTEX]

HEART:
  depends_on: [GRAPH]
  consumed_by: [CORTEX, CONTROLLER]

RAMP:
  depends_on: [ORACLE]
  consumed_by: [CONTROLLER]

SYNC:
  depends_on: []
  consumed_by: [CI_CD_PIPELINE]

SDK:
  depends_on: [ALL_NETWORK_COMPONENTS]
  consumed_by: [DEVELOPERS]

TESTNET:
  depends_on: [ALL_COMPONENTS]
  consumed_by: [DEVELOPERS, QA]

KNIRVANA:
  depends_on: [GRAPH, CHAIN, WALLET]
  consumed_by: [GAMERS]

BASE:
  depends_on: []
  consumed_by: [ALL_GO_RUST_TS_COMPONENTS]
```

### 12.2 Zero Trust Infrastructure Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Auth Service (in KNIRVROUTER)            │
│  - Cryptographic verification                               │
│  - Cache key metadata (5-min TTL)                           │
└──────────────────────┬──────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway Layer (KNIRVGATEWAY)       │
│  - Key extraction & rate limiting                           │
│  - DDoS protection (no IP trust)                            │
│  - Request signature validation                             │
└──────────────────────┬──────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────────────┐
│          Policy Decision Point (PDP) (in KNIRVORACLE)       │
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

### 12.3 Integration Checklist for New Components

When adding a new component to the KNIRV Network:

- [ ] Implements standardized ProtoBuf API contracts
- [ ] Uses mTLS with consortium CA certificates
- [ ] Publishes events to ROUTER event bus
- [ ] Validates JWTs via ORACLE auth service
- [ ] Exposes both REST and gRPC interfaces
- [ ] Includes OpenTelemetry tracing instrumentation
- [ ] Provides Prometheus metrics endpoint
- [ ] Has comprehensive integration tests
- [ ] Documents API in OpenAPI/ProtoBuf format
- [ ] Registers service discovery with ROUTER
- [ ] Follows KNIRV naming conventions
- [ ] Includes health check endpoint
- [ ] Implements graceful shutdown
- [ ] Has configuration via environment variables
- [ ] Logs structured JSON to stdout
- [ ] Supports multi-network deployment (testnet/mainnet)

### 12.4 Critical Anti-Patterns to Avoid

| Anti-Pattern | Problem | Correct Approach |
|--------------|---------|------------------|
| Direct database access | Bypasses business logic | Use component APIs |
| Synchronous cross-component calls | Blocking, slow | Async events |
| Duplicate state management | Inconsistency | Event subscriptions |
| Custom auth per component | Security gaps | Shared auth library |
| Hardcoded service URLs | Inflexible | Service discovery |
| Network-based trust | Security risk | Zero Trust always |
| Shared API keys | Audit impossible | Unique keys per entity |
| Static security posture | Outdated protection | Continuous validation |

### 12.5 Free Acquisition Steps for WASM Modules

**For STEM (stem.wasm):**
1. Visit RAMP marketplace at https://ramp.knirv.network
2. Browse pre-compiled SLM models (1B, 3B, 7B parameters)
3. Select desired model and click "Download STEM"
4. Receive `stem_1b.wasm` or similar
5. Load into CONTROLLER via "Upload STEM Module"

**For CORTEX (cortex.wasm):**
1. Install CONTROLLER from https://controller.knirv.network
2. Default cortex.wasm included in installation
3. Optional: Build custom cortex.wasm via RAMP "Cortex Builder"
4. Load LoRA adapters from GRAPH to modify behavior
5. cortex.wasm automatically loads stem.wasm at runtime

---

## Conclusion

This consolidated implementation plan transforms the KNIRV D-TEN ecosystem from 16 loosely coupled applications into a unified, secure, enterprise-grade decentralized network. The phased approach ensures:

1. **Security First:** Zero Trust foundation established before any integration work
2. **Performance Next:** gRPC communication layer provides 4-10x improvement
3. **Integration Then:** Event-driven architecture with unified auth and IBC
4. **Experience Last:** User-facing features built on solid infrastructure

**Next Steps:**
1. Review and approve this consolidated implementation plan
2. Assign engineering teams to Phase 0 milestones
3. Set up weekly integration sync meetings
4. Create tracking dashboard for success metrics
5. Begin Phase 0 implementation immediately

---

**Document Version Control:**
- v1.0 (January 2026): Consolidated from zt_implementation.md, integration_plan.md, proto_gRPC_strategy.md
- Sources: Zero Trust Architecture, gRPC Strategy, Integration Plan
