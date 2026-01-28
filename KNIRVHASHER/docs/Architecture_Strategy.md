# Hasher Architecture Strategy
**Version:** 1.0
**Date:** December 23, 2024
**Status:** Strategic Planning Document

---

## Executive Summary

This document outlines the recommended architectural strategy for implementing Hasher v2.0 based on the current proof-of-concept status and the requirements detailed in SDD_v2.md. The strategy addresses component placement, technology choices, deployment models, and a phased development roadmap.

**Key Decisions:**
- **Backend Architecture:** Monolithic Go application for PoC, evolving to microservices for production
- **Component Placement:** All core logic (ASIC Controller, KDF Engine, Credential Manager) on host server
- **Client Architecture:** Multiple client options - CLI (Go), Web UI (TypeScript/React), Desktop (Rust/Tauri)
- **Deployment Model:** Docker-based containerization with USB device passthrough
- **Communication:** REST API initially, gRPC for production performance

---

## 1. Component Placement Strategy

### 1.1 System Topology

```
┌─────────────────────────────────────────────────────────────┐
│                    CLIENT TIER                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   CLI Tool   │  │   Web UI     │  │  Desktop App │       │
│  │   (Go)       │  │ (React/TS)   │  │ (Rust/Tauri) │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
└─────────┼──────────────────┼──────────────────┼─────────────┘
          │                  │                  │
          └──────────────────┼──────────────────┘
                             │ REST/gRPC API (HTTPS)
┌────────────────────────────┼─────────────────────────────────┐
│                    SERVER TIER (Host System)                 │
│  ┌──────────────────────────────────────────────────────┐    │
│  │         Hasher Backend Service (Go)             │    │
│  │  ┌────────────────────────────────────────────────┐  │    │
│  │  │         API Layer (REST/gRPC)                  │  │    │
│  │  ├────────────────────────────────────────────────┤  │    │
│  │  │         Credential Manager                     │  │    │
│  │  │  • Store/Verify credentials                    │  │    │
│  │  │  • Salt generation                             │  │    │
│  │  │  • Access control                              │  │    │
│  │  ├────────────────────────────────────────────────┤  │    │
│  │  │         KDF Engine                             │  │    │
│  │  │  • Iteration management                        │  │    │
│  │  │  • Work distribution                           │  │    │
│  │  │  • CPU fallback                                │  │    │
│  │  ├────────────────────────────────────────────────┤  │    │
│  │  │         ASIC Controller                        │  │    │
│  │  │  • USB device management                       │  │    │
│  │  │  • Protocol implementation                     │  │    │
│  │  │  • Packet construction/parsing                 │  │    │
│  │  └────────────────────────────────────────────────┘  │    │
│  │                        ↓                             │    │
│  │  ┌────────────────────────────────────────────────┐  │    │
│  │  │         USB Protocol Layer (libusb/gousb)      │  │    │
│  │  └────────────────────────────────────────────────┘  │    │
│  └─────────────────────────┬────────────────────────────┘    │
│                            ↓ USB Bulk Transfer               │
│  ┌────────────────────────────────────────────────────────┐  │
│  │         Antminer S3 (USB Device)                       │  │
│  │  • Atheros AR9330 (MIPS CPU)                           │  │
│  │  • PIC Microcontroller (USB bridge)                    │  │
│  │  • 32x BM1382 ASIC chips (500 GH/s)                    │  │
│  │  • VID:PID = 0x4254:0x4153                             │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │         Storage Backend (SQLite/PostgreSQL)            │  │
│  │  • Encrypted credential database                       │  │
│  │  • PQC-encrypted at rest                               │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

### 1.2 Critical Architectural Decisions

#### Decision 1: ASIC Controller Location
**Where:** Host server (not on Antminer device)

**Rationale:**
- The Antminer S3 runs embedded OpenWrt firmware with limited resources (64MB RAM, 400MHz MIPS CPU)
- We communicate with it as a USB peripheral device, not as a compute platform
- The ASIC Controller is a software layer that speaks the Bitmain USB protocol
- Host server has resources for complex protocol handling, error recovery, and monitoring

**Implication:** ASIC Controller is a Go package within the backend service, managing USB communication via libusb/gousb.

---

#### Decision 2: KDF Engine Integration
**Where:** Integrated into backend service (same process as Credential Manager)

**Rationale:**
- KDF Engine orchestrates work distribution to ASIC Controller
- Tight coupling with Credential Manager for authentication flow
- Minimal latency overhead from inter-process communication
- Simpler deployment and debugging

**Implication:** KDF Engine is a Go package providing high-level KDF operations, calling ASIC Controller internally.

---

#### Decision 3: Credential Manager Architecture
**Where:** Backend service on host server

**Rationale:**
- Central vault service managing all credential operations
- Needs direct access to encrypted database
- Coordinates KDF Engine and storage backend
- Provides API endpoints for client applications

**Implication:** Credential Manager is the top-level orchestration layer in the backend service.

---

## 2. Technology Stack Recommendations

### 2.1 Backend Service (PoC Phase)

**Language:** Go 1.24+

**Core Packages:**
```
github.com/google/gousb              # USB device communication
github.com/cloudflare/circl          # Post-quantum crypto (Kyber, Dilithium)
github.com/mattn/go-sqlite3          # Storage backend (PoC)
github.com/gorilla/mux               # HTTP routing (or gin-gonic/gin)
github.com/prometheus/client_golang  # Metrics
```

**Architecture:** Monolithic application with clear package boundaries

**Project Structure:**
```
hasher/
├── cmd/
│   ├── hasher-server/     # Main backend service
│   │   └── main.go
│   └── hasher-cli/        # CLI client
│       └── main.go
├── internal/
│   ├── asic/                   # ASIC Controller
│   │   ├── controller.go
│   │   ├── protocol.go
│   │   └── usb.go
│   ├── kdf/                    # KDF Engine
│   │   ├── engine.go
│   │   ├── work_batcher.go
│   │   └── cpu_fallback.go
│   ├── vault/                  # Credential Manager
│   │   ├── manager.go
│   │   ├── credential.go
│   │   └── auth.go
│   ├── pqc/                    # Post-quantum crypto
│   │   ├── kyber.go
│   │   └── dilithium.go
│   ├── storage/                # Database backend
│   │   ├── backend.go
│   │   └── sqlite.go
│   └── api/                    # REST API handlers
│       ├── server.go
│       ├── handlers.go
│       └── middleware.go
├── internal/                   # Private application code
│   ├── config/
│   ├── metrics/
│   └── audit/
├── web/                        # Web UI (optional for PoC)
│   ├── src/
│   └── public/
└── docs/
```

**Why Monolithic for PoC:**
- Faster development velocity
- Simpler deployment (single binary + USB device)
- Easier debugging and troubleshooting
- Lower operational complexity
- Clear package boundaries allow future microservices extraction

---

### 2.2 Client Applications

#### Option A: CLI Client (Recommended for PoC)
**Language:** Go

**Features:**
- Credential storage/retrieval
- Authentication testing
- Device status monitoring
- Configuration management

**Advantages:**
- Fast to develop (same language as backend)
- Easy to script and automate
- No GUI complexity
- Perfect for initial testing and validation

**Build:**
```bash
go build -o hasher cmd/hasher-cli/main.go
```

**Usage:**
```bash
# Store credential
hasher store --username alice@example.com --password "MyP@ssw0rd"

# Verify credential
hasher verify --username alice@example.com --password "MyP@ssw0rd"

# List credentials
hasher list

# Device status
hasher status
```

---

#### Option B: Web UI (Recommended for Production)
**Language:** TypeScript
**Framework:** React + Vite (or Next.js for SSR)
**Styling:** Tailwind CSS
**State Management:** Zustand or React Query

**Features:**
- Modern, responsive web interface
- Works on any device with a browser
- No installation required
- Easy updates (server-side deployment)

**Architecture:**
```
web/
├── src/
│   ├── components/
│   │   ├── CredentialList.tsx
│   │   ├── CredentialForm.tsx
│   │   ├── DeviceStatus.tsx
│   │   └── Dashboard.tsx
│   ├── hooks/
│   │   ├── useCredentials.ts
│   │   └── useDeviceStatus.ts
│   ├── api/
│   │   └── client.ts
│   ├── App.tsx
│   └── main.tsx
├── package.json
└── vite.config.ts
```

**Advantages:**
- Cross-platform (Windows, Mac, Linux, mobile browsers)
- Rich UI/UX capabilities
- Large ecosystem of components
- Easy to deploy (static files or SSR)

**Disadvantages:**
- Requires web browser
- Larger attack surface (XSS, CSRF)
- Network dependency

---

#### Option C: Desktop App (Alternative for Offline Use)
**Language:** Rust
**Framework:** Tauri 2.0
**UI:** HTML/CSS/TypeScript (React, Vue, or Svelte)

**Features:**
- Native desktop application
- Small binary size (~3-5MB)
- Native OS integration
- Secure by default (Rust backend)
- Can work offline with local database

**Architecture:**
```
src-tauri/
├── src/
│   ├── main.rs            # Tauri backend
│   ├── api.rs             # API client
│   ├── commands.rs        # Tauri commands
│   └── state.rs           # Application state
└── Cargo.toml

src/                       # Frontend (React/Vue/Svelte)
├── components/
├── App.tsx
└── main.tsx
```

**Why Rust + Tauri:**
- **Security:** Memory-safe language, sandboxed architecture
- **Performance:** Native speed, low resource usage
- **Cross-platform:** Single codebase for Windows, Mac, Linux
- **Small footprint:** Smaller than Electron (~3MB vs ~150MB)
- **Native feel:** Uses system webview instead of bundled Chromium

**Advantages:**
- Native app experience
- Offline capability
- Better security than Electron
- Smaller distribution size

**Disadvantages:**
- More complex build process
- Rust learning curve (if team unfamiliar)
- Less mature ecosystem than Electron

---

### 2.3 Recommended Client Strategy

**Phase 1 (PoC):** CLI Client (Go)
- Fast to develop
- Perfect for testing backend
- Easy integration with existing Go codebase

**Phase 2 (Beta):** Web UI (TypeScript/React)
- Accessible to non-technical users
- Cross-platform by nature
- Easy to deploy and update

**Phase 3 (Production):** Desktop App (Rust/Tauri) + Web UI
- Rust/Tauri for power users wanting offline/native experience
- Web UI for quick access and mobile devices
- Both clients talk to same backend API

---

## 3. Deployment Architecture

### 3.1 PoC Deployment (Single Server)

```
Host Server (Linux/macOS)
├── Docker Container: hasher-server
│   ├── Backend service (Go binary)
│   ├── SQLite database (mounted volume)
│   ├── USB device passthrough
│   └── Port 8443 (HTTPS API)
├── Antminer S3 (Network connected)
│   └── VID:PID 0x4254:0x4153
└── Client applications (local or remote)
    ├── CLI (same host or SSH)
    └── Web browser (HTTPS to port 8443)
```

**Docker Compose:**
```yaml
version: '3.8'

services:
  hasher-server:
    build: .
    container_name: hasher
    privileged: true  # Required for USB access
    devices:
      - /dev/bus/usb:/dev/bus/usb  # USB passthrough
    ports:
      - "8443:8443"  # HTTPS API
      - "9090:9090"  # Prometheus metrics
    volumes:
      - ./data:/data              # Database persistence
      - ./config:/config          # Configuration
      - ./logs:/logs              # Logging
    environment:
      - ASIC_DEVICE_VID=0x4254
      - ASIC_DEVICE_PID=0x4153
      - DB_PATH=/data/vault.db
      - KDF_ITERATIONS=100000000
      - LOG_LEVEL=info
      - TLS_CERT=/config/tls/server.crt
      - TLS_KEY=/config/tls/server.key
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "https://localhost:8443/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

---

### 3.2 Production Deployment (High Availability)

```
Load Balancer (nginx/HAProxy)
        ↓
┌───────┴───────┐
│               │
Server 1        Server 2
├── Hasher ├── Hasher
├── Antminer S3 ├── Antminer S3
├── PostgreSQL  └─ PostgreSQL
│   (replicated)    (replicated)
└── Monitoring
```

**Key Considerations:**
- Multiple ASIC devices for redundancy
- PostgreSQL with replication for database
- Shared configuration via etcd or Consul
- Centralized logging (ELK stack or Loki)
- Monitoring (Prometheus + Grafana)

---

## 4. Communication Protocols

### 4.1 Client-Server API

**Phase 1 (PoC):** REST API with JSON

**Endpoints:**
```
POST   /api/v1/credentials          # Store credential
POST   /api/v1/verify               # Verify credential
GET    /api/v1/credentials          # List credentials
DELETE /api/v1/credentials/:id      # Delete credential
GET    /api/v1/health               # Health check
GET    /api/v1/device/status        # ASIC device status
GET    /api/v1/metrics              # Prometheus metrics
```

**Example Request:**
```bash
curl -X POST https://localhost:8443/api/v1/credentials \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "username": "alice@example.com",
    "password": "SecureP@ssw0rd!",
    "metadata": {
      "department": "engineering"
    }
  }'
```

**Phase 2 (Production):** gRPC for performance

**Advantages:**
- Binary protocol (faster, smaller)
- Built-in streaming
- Strong typing with Protocol Buffers
- Better performance for high-throughput

**Proto Definition:**
```protobuf
syntax = "proto3";

package asicshield.v1;

service VaultService {
  rpc StoreCredential(StoreRequest) returns (StoreResponse);
  rpc VerifyCredential(VerifyRequest) returns (VerifyResponse);
  rpc ListCredentials(ListRequest) returns (ListResponse);
  rpc GetDeviceStatus(StatusRequest) returns (StatusResponse);
}

message StoreRequest {
  string username = 1;
  string password = 2;
  map<string, string> metadata = 3;
}

message StoreResponse {
  bool success = 1;
  string message = 2;
}

message VerifyRequest {
  string username = 1;
  string password = 2;
}

message VerifyResponse {
  bool authenticated = 1;
  string message = 2;
}
```

---

### 4.2 Internal Communication

All components run in same process (monolithic architecture):

```go
// Internal communication via Go interfaces
type ASICController interface {
    Initialize() error
    ComputeKDF(input []byte, iterations int) ([]byte, error)
    GetStatus() (*DeviceStatus, error)
    Close() error
}

type KDFEngine interface {
    DeriveKey(password, salt []byte, iterations int) ([]byte, error)
    RecommendIterations(threatLevel ThreatLevel) int
}

type CredentialManager interface {
    Store(username, password string) error
    Verify(username, password string) (bool, error)
    List() ([]*Credential, error)
    Delete(username string) error
}
```

**Advantages:**
- Zero serialization overhead
- Type safety at compile time
- Easy to test with mocks
- Simple to reason about
- Can extract to microservices later

---

## 5. Data Flow Architecture

### 5.1 Credential Storage Flow

```
Client Application
    ↓ HTTPS POST /api/v1/credentials
    ↓ {username, password, metadata}
API Handler
    ↓ Validate request
    ↓ Call CredentialManager.Store()
Credential Manager
    ↓ Generate 256-bit random salt
    ↓ Determine iteration count (100M-500M)
    ↓ Call KDFEngine.DeriveKey()
KDF Engine
    ↓ Check if ASIC available
    ↓ Split work into chunks (10M iterations each)
    ↓ Call ASICController.ComputeKDF()
ASIC Controller
    ↓ Build TxTask packets
    ↓ Send via USB bulk transfer
    ↓ Read RxNonce responses
    ↓ Assemble final hash
    ↓ Return to KDF Engine
KDF Engine
    ↓ Return derived key to Credential Manager
Credential Manager
    ↓ Encrypt hash with PQC (Kyber KEM + AES-GCM)
    ↓ Call StorageBackend.Save()
Storage Backend
    ↓ Insert to database (encrypted)
    ↓ Return success
    ↓ Audit log entry
API Handler
    ↓ Return success response to client
Client Application
```

**Performance Expectations:**
- Salt generation: <1ms
- KDF computation (100M iterations): ~200ms (ASIC) vs ~60 seconds (CPU)
- PQC encryption: ~10ms
- Database write: ~5ms
- **Total: ~220ms for storage operation**

---

### 5.2 Credential Verification Flow

```
Client Application
    ↓ HTTPS POST /api/v1/verify
    ↓ {username, password}
API Handler
    ↓ Validate request
    ↓ Rate limiting check
    ↓ Call CredentialManager.Verify()
Credential Manager
    ↓ Call StorageBackend.Get(username)
Storage Backend
    ↓ Query database
    ↓ Return {hash, salt, iterations, algorithm}
Credential Manager
    ↓ Decrypt hash with PQC
    ↓ Call KDFEngine.DeriveKey(password, salt, iterations)
KDF Engine
    ↓ Call ASICController.ComputeKDF()
ASIC Controller
    ↓ Execute KDF computation (same as storage)
    ↓ Return computed hash
KDF Engine
    ↓ Return to Credential Manager
Credential Manager
    ↓ Constant-time comparison (subtle.ConstantTimeCompare)
    ↓ Update last_used timestamp
    ↓ Audit log entry
API Handler
    ↓ Return authentication result
Client Application
```

**Performance Expectations:**
- Database read: ~2ms
- PQC decryption: ~10ms
- KDF computation (100M iterations): ~200ms (ASIC)
- Comparison + logging: ~2ms
- **Total: ~214ms for verification**

---

## 6. Security Architecture

### 6.1 Multi-Layer Defense

```
Layer 1: TLS Transport Security
├── TLS 1.3 with strong cipher suites
├── Certificate-based authentication (mutual TLS optional)
└── Perfect forward secrecy

Layer 2: API Authentication
├── JWT tokens or API keys
├── Token expiration and refresh
└── Rate limiting (60 requests/minute)

Layer 3: Extreme-Iteration KDF
├── 100M-500M SHA-256 iterations
├── ASIC-accelerated computation
├── Unique 256-bit salt per credential
└── Economic infeasibility for attackers

Layer 4: Post-Quantum Cryptography
├── CRYSTALS-Kyber-768 (key encapsulation)
├── CRYSTALS-Dilithium-3 (signatures)
└── Encrypted database with PQC

Layer 5: Storage Encryption
├── AES-256-GCM for database encryption
├── Separate encryption key (not in database)
├── Key derived from master password + hardware binding
└── No plaintext credential data

Layer 6: Audit Logging
├── All authentication attempts logged
├── Immutable audit trail
├── Anomaly detection
└── Alert on suspicious patterns
```

---

### 6.2 Threat Mitigation

**Offline Attacks:**
- Mitigation: Extreme-iteration KDF (100M-500M iterations)
- Result: 50-100x slower for attackers than defenders
- Quantum resistance: Even with Grover's 2x speedup, still economically infeasible

**Online Attacks:**
- Mitigation: Rate limiting (60 requests/minute)
- Result: Maximum 86,400 attempts per day
- At 500M iterations: would take billions of years

**Database Compromise:**
- Mitigation: PQC encryption at rest
- Result: Stolen database is useless without decryption keys
- Keys stored separately from database

**Man-in-the-Middle:**
- Mitigation: TLS 1.3 with certificate pinning
- Result: Encrypted transport, authenticated endpoints

**Timing Attacks:**
- Mitigation: Constant-time comparison for password verification
- Result: No timing side-channel leakage

---

## 7. Phased Development Roadmap

### Phase 1: Proof of Concept (Weeks 1-8)

**Goal:** Validate core concept with minimal viable product

**Components:**
- ✅ ASIC Controller with USB protocol (COMPLETED - December 2024)
- ⏳ KDF Engine with CPU fallback (IN PROGRESS)
- ⏳ Basic Credential Manager (IN PROGRESS)
- ⏳ CLI client for testing
- ⏳ SQLite storage backend
- ⏳ Simple REST API

**Deliverables:**
- Working prototype demonstrating ASIC-accelerated KDF
- Performance benchmarks (ASIC vs CPU)
- Basic credential storage and verification
- Documentation and test results

**Success Criteria:**
- KDF computation time: <1 second for 100M iterations on ASIC
- Functional credential storage and retrieval
- USB communication stable over extended periods
- 10x+ performance improvement over CPU

---

### Phase 2: Production Readiness (Weeks 9-16)

**Goal:** Harden system for production deployment

**Components:**
- PQC integration (Kyber + Dilithium)
- Enhanced error handling and retry logic
- PostgreSQL backend option
- Comprehensive monitoring (Prometheus + Grafana)
- Web UI (React + TypeScript)
- Security audit
- Load testing

**Deliverables:**
- Production-grade backend service
- Web-based user interface
- Monitoring dashboards
- Security audit report
- Performance test results
- Deployment documentation

**Success Criteria:**
- 99.9% uptime over 1-week test
- Handle 1000+ credentials
- <500ms p99 latency for authentication
- Pass security audit
- Complete documentation

---

### Phase 3: Enterprise Features (Weeks 17-24)

**Goal:** Add enterprise-level features and integrations

**Components:**
- High-availability deployment
- LDAP/Active Directory integration
- SAML/OAuth support
- Desktop application (Rust + Tauri)
- Multi-ASIC support (load balancing)
- Disaster recovery and backup
- Role-based access control (RBAC)
- Compliance certifications (SOC 2, ISO 27001)

**Deliverables:**
- HA deployment guide
- Enterprise integrations
- Desktop application (Windows, Mac, Linux)
- Backup and recovery procedures
- Compliance documentation

**Success Criteria:**
- Multi-region deployment working
- LDAP/SAML authentication functional
- Desktop app published to app stores
- Zero data loss in failover scenarios
- Compliance audit passed

---

### Phase 4: Advanced Features (Weeks 25+)

**Goal:** Research and implement advanced capabilities

**Components:**
- Homomorphic encryption for zero-knowledge verification
- Multi-party computation for distributed key management
- Support for newer ASIC models (S9, S17, S19)
- FPGA acceleration option
- Mobile applications (iOS, Android)
- Hardware security module (HSM) integration
- Quantum key distribution (QKD) preparation

**Deliverables:**
- Research papers on advanced features
- Prototype implementations
- Mobile applications
- HSM integration guide

---

## 8. Technology Decision Matrix

### 8.1 Backend Language Comparison

| Language | Pros | Cons | Verdict |
|----------|------|------|---------|
| **Go** | • USB library support (gousb)<br>• Fast compilation<br>• Easy deployment (single binary)<br>• Great concurrency<br>• Existing PoC codebase | • No generics (before 1.18)<br>• Verbose error handling | ✅ **RECOMMENDED** |
| Rust | • Memory safety<br>• Best performance<br>• Growing ecosystem | • Steep learning curve<br>• Slower compilation<br>• Less mature USB libraries | ⚠️ Consider for Phase 3+ |
| C/C++ | • Maximum performance<br>• Direct hardware access | • Memory safety issues<br>• Complex dependency management<br>• Slower development | ❌ Not recommended |

**Decision:** Go for backend - existing codebase, proven USB support, fast development

---

### 8.2 Client Application Comparison

| Option | Best For | Pros | Cons | Phase |
|--------|----------|------|------|-------|
| **CLI (Go)** | Testing, scripting, automation | • Fast to develop<br>• Same language as backend<br>• Easy to script | • No GUI<br>• Technical users only | ✅ Phase 1 |
| **Web UI (React/TS)** | General users, cross-platform | • Rich UI/UX<br>• Cross-platform<br>• Easy updates | • Requires browser<br>• Network dependency | ✅ Phase 2 |
| **Desktop (Rust/Tauri)** | Power users, offline use | • Native experience<br>• Small binary<br>• Offline capable | • More complex build<br>• Learning curve | ✅ Phase 3 |
| **Mobile (React Native)** | Mobile users | • iOS/Android support<br>• Shared codebase | • Different UX paradigm<br>• App store overhead | ⏳ Phase 4 |

**Decision:**
- Phase 1: CLI (Go)
- Phase 2: Web UI (React/TypeScript)
- Phase 3: Desktop (Rust/Tauri)
- Phase 4: Mobile (React Native or Flutter)

---

### 8.3 Database Comparison

| Database | Best For | Pros | Cons | Phase |
|----------|----------|------|------|-------|
| **SQLite** | PoC, small deployments | • Simple setup<br>• Single file<br>• No server needed | • Not distributed<br>• Limited concurrency | ✅ Phase 1 |
| **PostgreSQL** | Production | • ACID compliance<br>• Rich features<br>• Replication support | • More complex setup<br>• Requires server | ✅ Phase 2 |
| MySQL | Alternative | • Widely used<br>• Good performance | • Less advanced features | ⚠️ Alternative |
| etcd/Consul | Config store | • Distributed<br>• Highly available | • Not for large data | ⏳ Phase 3 (config only) |

**Decision:**
- Phase 1: SQLite (PoC simplicity)
- Phase 2+: PostgreSQL (production reliability)

---

## 9. Deployment Recommendations

### 9.1 Development Environment

```bash
# Developer workstation
Host OS: macOS, Linux, or Windows + WSL2
├── Go 1.24+
├── Docker Desktop
├── Make
├── Git
├── IDE: VSCode or GoLand
└── Antminer S3 (USB connected)

# Development workflow
1. Edit code locally
2. Run tests: make test
3. Build: make build
4. Run locally: ./bin/hasher-server
5. Test with CLI: ./bin/hasher-cli
```

---

### 9.2 Production Environment

**Minimum Requirements:**
```
Hardware:
├── CPU: 4 cores (x86_64 or ARM64)
├── RAM: 8GB
├── Storage: 100GB SSD
├── Network: Gigabit Ethernet
├── USB: Available USB port for Antminer
└── Power: 500W (400W for Antminer + 100W for server)

Software:
├── OS: Ubuntu 22.04 LTS or Debian 12
├── Docker 24+
├── Docker Compose 2.20+
└── systemd (for service management)
```

**Deployment Steps:**
```bash
# 1. Clone repository
git clone https://github.com/guiperry/hasher.git
cd hasher

# 2. Configure environment
cp .env.example .env
vim .env  # Edit configuration

# 3. Generate TLS certificates
./scripts/generate-certs.sh

# 4. Build Docker image
docker build -t hasher:latest .

# 5. Start services
docker-compose up -d

# 6. Verify health
curl -k https://localhost:8443/health

# 7. Initialize first user
./scripts/init-admin.sh
```

---

### 9.3 High Availability Deployment

```
┌─────────────────────────────────────────────┐
│          Load Balancer (nginx)              │
│  • TLS termination                          │
│  • Health checks                            │
│  • Sticky sessions                          │
└──────────┬──────────────────┬───────────────┘
           │                  │
    ┌──────┴───────┐   ┌──────┴───────┐
    │   Server 1   │   │   Server 2   │
    │  hasher │   │  hasher │
    │  Antminer S3 │   │  Antminer S3 │
    └──────┬───────┘   └──────┬───────┘
           │                  │
           └────────┬─────────┘
                    │
         ┌──────────┴──────────┐
         │   PostgreSQL HA     │
         │  • Primary/Replica  │
         │  • Streaming rep.   │
         └─────────────────────┘
```

**Key Features:**
- Load balancer distributes traffic
- Multiple Hasher instances
- Shared PostgreSQL with replication
- Each instance has dedicated Antminer
- Failover handled by load balancer

---

## 10. Cost Analysis

### 10.1 Development Costs

**Team (3 months for Phase 1-2):**
```
1x Backend Engineer (Go):       $40,000
1x Frontend Engineer (React):   $35,000
0.5x DevOps Engineer:           $20,000
0.25x Security Consultant:      $15,000
───────────────────────────────────────
Total Development:              $110,000
```

**Hardware (PoC):**
```
1x Antminer S3 (used):          $50
1x Development Server:          $1,500
Cables/Accessories:             $100
───────────────────────────────────────
Total Hardware:                 $1,650
```

**Total PoC Cost:** ~$112,000

---

### 10.2 Operating Costs (Annual, 10K users)

```
Hardware (10x Antminer S3):     $500 (already owned)
Host Server:                    $3,000 (amortized over 5 years)
Power (3.6 kW @ $0.12/kWh):     $4,541
Personnel (sysadmin):           $19,000
Software/Services:              $1,000
───────────────────────────────────────
Total Annual:                   $27,541
Per User Per Year:              $2.75
```

**vs. Commercial Solutions:**
- Duo MFA: $10/user/year = $100,000/year
- YubiKey: $50/user one-time = $500,000
- AWS KMS: $5/user/year = $50,000/year

**Break-even:** 2-3 years vs commercial solutions

---

## 11. Risk Assessment & Mitigation

### 11.1 Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| ASIC hardware failure | Medium | High | • Multiple ASICs for redundancy<br>• CPU fallback mode<br>• Spare hardware inventory |
| USB communication instability | Low | Medium | • Robust retry logic<br>• Health monitoring<br>• Automatic reconnection |
| Performance degradation | Low | Medium | • Monitoring and alerting<br>• Performance testing<br>• Capacity planning |
| Security vulnerability | Low | Critical | • Security audits<br>• Penetration testing<br>• Bug bounty program |

---

### 11.2 Operational Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Power outage | Medium | Medium | • UPS system<br>• Graceful shutdown<br>• Database persistence |
| Network failure | Low | High | • Redundant network paths<br>• Offline mode (future)<br>• Local caching |
| Data loss | Very Low | Critical | • Regular backups<br>• Replication<br>• Disaster recovery plan |
| Scaling issues | Medium | Medium | • Load testing<br>• Multi-ASIC support<br>• Microservices migration |

---

## 12. Success Metrics

### 12.1 Phase 1 (PoC) Success Criteria

**Performance:**
- ✅ KDF computation time: <1 second for 100M iterations (ASIC)
- ⏳ 10x+ speedup over CPU
- ⏳ <500ms p99 authentication latency

**Reliability:**
- ⏳ 99% uptime over 1-week continuous operation
- ⏳ Zero data corruption
- ⏳ Graceful handling of USB disconnects

**Functionality:**
- ⏳ Store and verify credentials correctly
- ⏳ Support 1000+ credentials
- ⏳ CLI client functional

---

### 12.2 Phase 2 (Production) Success Criteria

**Performance:**
- 99.9% uptime over 1 month
- <500ms p99 authentication latency
- Handle 10,000+ credentials
- Support 100+ concurrent users

**Security:**
- Pass security audit (no critical vulnerabilities)
- PQC integration functional
- Encrypted storage validated
- Audit logging complete

**Usability:**
- Web UI functional
- User documentation complete
- API documentation complete
- Deployment guide validated

---

## 13. Recommendations Summary

### 13.1 Immediate Actions (Phase 1)

1. **Complete KDF Engine Implementation**
   - Integrate ASIC Controller with KDF operations
   - Implement work batching and pipelining
   - Add CPU fallback for development/testing

2. **Build Credential Manager**
   - Implement Store/Verify operations
   - Add salt generation
   - Integrate with KDF Engine

3. **Create CLI Client**
   - Basic credential operations
   - Device status monitoring
   - Configuration management

4. **Setup Testing Infrastructure**
   - Unit tests for all packages
   - Integration tests for full workflow
   - Performance benchmarks

---

### 13.2 Technology Choices

**Backend:** Go (monolithic for PoC, microservices for production)

**Clients:**
- Phase 1: CLI (Go)
- Phase 2: Web UI (React + TypeScript)
- Phase 3: Desktop (Rust + Tauri)

**Database:**
- Phase 1: SQLite
- Phase 2+: PostgreSQL

**API:**
- Phase 1: REST (JSON)
- Phase 2+: gRPC (Protocol Buffers)

---

### 13.3 Deployment Strategy

**Development:** Local Docker containers with USB passthrough

**Production:** Docker Compose on Linux server with:
- TLS certificates
- PostgreSQL database
- Prometheus monitoring
- Log aggregation
- Automated backups

---

## 14. Conclusion

The recommended architecture for Hasher leverages a pragmatic, phased approach:

1. **Monolithic Go backend** for rapid PoC development
2. **Multiple client options** (CLI, Web, Desktop) for different use cases
3. **Clear component boundaries** allowing future microservices extraction
4. **USB-based ASIC communication** from host server (not on Antminer device)
5. **Integrated KDF Engine** tightly coupled with Credential Manager
6. **Layered security** with extreme-iteration KDF + PQC + encrypted storage

This architecture provides:
- ✅ Fast time-to-market for PoC validation
- ✅ Clear path to production hardening
- ✅ Flexibility for future enhancements
- ✅ Cost-effective deployment
- ✅ Strong security guarantees

**Next Steps:**
1. Review and approve this architectural strategy
2. Begin Phase 1 implementation (KDF Engine + Credential Manager)
3. Build CLI client for testing
4. Conduct performance benchmarking
5. Plan Phase 2 production hardening

---

**Document Version:** 1.0
**Last Updated:** December 23, 2024
**Next Review:** After Phase 1 completion
**Status:** Awaiting approval
