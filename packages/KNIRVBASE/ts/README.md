# KNIRVBASE (TypeScript)

✅ **Overview**

KNIRVBASE is a lightweight, local-first distributed database prototype implemented in TypeScript. This browser and Node.js compatible implementation provides full feature parity with the Go reference implementation, plus additional browser-native capabilities including:

- **PQC Encryption Layer**: Post-quantum cryptography using Kyber-768 (encryption) and Dilithium-3 (signatures) for secure data storage
- CRDT-based conflict resolution using **vector clocks**,
- Local file-backed **storage** for metadata and blobs with IndexedDB browser support,
- **Real P2P networking** with WebRTC peer discovery and WebSocket communication,
- A small, human-friendly query language **KNIRVQL** for simple GET/SET/DELETE and basic vector similarity queries,
- Built-in authentication middleware, structured logging, monitoring metrics, and security hardening.

This package is intended as the client-side reference implementation for distributed collections and synchronization logic used across the KNIRV ecosystem, including browser extensions, desktop applications, and Node.js services.

---

## 🔍 Features

- **PQC Encryption at Rest**: Field-level encryption for sensitive data using Kyber-768 KEM + AES-256-GCM, with Dilithium-3 signatures for integrity
- Local-first operations and background sync with automatic conflict resolution
- CRDT resolve using vector clocks (merge rules + LWW tie-breakers)
- Dual storage backends: Node.js filesystem and browser IndexedDB
- **Real P2P networking** with WebSocket connections and WebRTC peer discovery
- `KNIRVQL` parser and executor with query optimization
- Built-in JWT authentication middleware and permission system
- Structured logging with multiple output formatters
- Prometheus-compatible monitoring metrics collection
- Express.js integration middleware
- Full type safety with zero `any` types throughout the codebase

---

## 🚀 Quickstart

### Prerequisites

- Node.js 18+ / Bun 1.0+ / Deno 1.30+
- TypeScript 5.0+ (see `package.json`)

### Installation

```bash
npm install @knirvcorp/knirvbase-ts
# or
yarn add @knirvcorp/knirvbase-ts
```

### Build from source

```bash
npm install
npm run build
```

### Usage Example

```typescript
import { DistributedDatabase } from '@knirvcorp/knirvbase-ts';

// Initialize database
const db = await DistributedDatabase.create({
  dataDir: '~/.local/share/knirvbase',
  enableP2P: true,
  enableEncryption: true
});

// Create collections
await db.createCollection('auth');
await db.createCollection('memory');

// KNIRVQL operations
await db.query(`SET google_maps_api_key = "AIzaSy..."`);
const result = await db.query(`GET AUTH WHERE key = "google_maps_api_key"`);

// Vector similarity search
const similar = await db.query(`
  GET MEMORY WHERE source = "web-scrape" 
  SIMILAR TO [0.45, 0.12, ...] 
  LIMIT 10
`);
```

### Browser Usage

```typescript
import { BrowserStorage } from '@knirvcorp/knirvbase-ts';

// Automatically uses IndexedDB for persistence
const db = await DistributedDatabase.create({
  storageBackend: new BrowserStorage(),
  enableP2P: true
});
```

---

## 📁 Storage Layout

### Node.js
- Data directory (default): `$XDG_DATA_HOME` or `~/.local/share/knirvbase`
- Per-collection JSON files: `<datadir>/<collection>/<id>.json`
- Blobs are saved under `<datadir>/<collection>/blobs/<id>`

### Browser
- IndexedDB database: `KNIRVBASE`
- Object stores per collection
- Blobs stored as ArrayBuffers with metadata indexing

Why blobs are not synced: to preserve network bandwidth and storage efficiency. The system synchronizes discovery metadata (including blob references) rather than raw blobs.

---

## 🧭 KNIRVQL (Query Language) — Examples

- Set an auth key:
```knirvql
SET google_maps_api_key = "AIzaSy..."
```

- Get an auth key:
```knirvql
GET AUTH WHERE key = "google_maps_api_key"
```

- Insert a memory entry:
```knirvql
INSERT MEMORY { 
  "source": "web-scrape", 
  "content": "...", 
  "vector": [0.45, 0.12, ...] 
}
```

- Get similar memory entries (vector search):
```knirvql
GET MEMORY WHERE source = "web-scrape" SIMILAR TO [0.45, 0.12] LIMIT 10
```

- Delete operation:
```knirvql
DELETE AUTH WHERE key = "expired_key"
```

The language is intentionally minimal and aimed at quick integration. Query optimizer automatically indexes frequently accessed fields.

---

## 📦 Package Overview (what's inside)

- `lib/index.ts` — Main exports and public API entry point
- `components/clock/` — Vector clock implementation and comparison utilities
- `components/collection/` — `DistributedCollection` + `LocalCollection`: local storage, CRDT operation emission, sync logic
- `components/crypto/pqc` — Post-quantum cryptography: Kyber-768 encryption, Dilithium-3 signatures, key management
- `components/network/` — `NetworkManager`: P2P networking with WebSocket + WebRTC peer discovery
- `components/resolver/` — CRDT resolver logic and merge operations
- `components/query/` — `KNIRVQL` parser, optimizer, and query execution
- `components/storage/` — FileStorage (Node.js) + BrowserStorage (IndexedDB) persistence layers
- `components/auth/` — JWT authentication, token management, permission system, Express middleware
- `components/logging/` — Structured logging with JSON and human readable formatters
- `components/monitoring/` — Metrics collection, counters, gauges, histograms
- `components/security/` — Key derivation, secure memory handling, threat detection
- `components/types/` — Core types (Document, CRDTOperation, NetworkConfig, ProtocolMessage, etc.)
- `utils/` — Common utilities and helper functions

---

## 🛠 Development & Testing

- Run all tests:
```bash
npm test
```

- Run tests with coverage:
```bash
npm test -- --coverage
```

- Lint code:
```bash
npm run lint
```

- Build package:
```bash
npm run build
```

- Watch mode development:
```bash
npm run dev
```

- Clean build artifacts:
```bash
npm run clean
```

---

## 📊 Performance Benchmarks

KNIRVBASE TypeScript includes a comprehensive benchmark suite that validates performance against ASIC-Shield SLA requirements:

### SLA Targets (ASIC-Shield Integration)
- **Credential Insert**: p99 < 12ms
- **Credential Query**: p99 < 6ms
- **Authentication Workflow**: p99 < 550ms (including 100M PBKDF2 iterations)
- **PQC Encryption**: < 25ms per operation
- **Large Scale**: No performance degradation with 10K+ credentials

### Running Benchmarks
```bash
# Run all benchmarks
npm run bench

# Run SLA validation
npm run bench:sla

# Generate CPU/memory profiles
npm run bench:profile
```

---

## ⚠️ Limitations & Security Notes

- **PQC Encryption at Rest:** Field-level encryption for all sensitive data across collections (`credentials`, `pqc_keys`, `sessions`, `audit_log`, `threat_events`, `access_control`). Encrypts specific fields like `hash`, `salt`, `token_hash`, `details`, `indicators`, `permissions`, etc. Uses Kyber-768 + AES-256-GCM for confidentiality and Dilithium-3 for integrity. Master key must be configured for encryption to be active.
- **P2P Networking:** WebSocket + WebRTC based P2P networking with peer discovery enables true distributed operation across browser and Node.js nodes.
- **Blob handling:** Blobs are stored locally and only referenced in synchronized metadata; no blob distribution is implemented here.
- **Authentication:** Built-in JWT authentication with role-based access control. All network messages can be optionally signed and encrypted.
- **Browser Security:** Implements secure memory wiping, constant-time comparisons, and protection against XSS and timing attacks.

---

## 📚 Design & Reference

The repository includes `docs/Distributed_Database_Implementation_ts.md` which documents architecture, rationale, and design decisions in depth — consult it for more detail on synchronization heuristics, CRDT rules, and future extensions.

Additional documentation:
- `docs/PRODUCTION_READINESS_ASSESSMENT.md` — Security audit and production readiness report
- `coverage/` — Full test coverage reports

---

## 💡 Contributing

- Open an issue for feature requests or bug reports
- Create a PR with tests and descriptions of changes
- Keep code and docs consistent with TypeScript idioms and the repository's architecture
- Maintain zero `any` types policy
- All changes must pass existing tests and maintain 90%+ test coverage

---

## 📜 License

See the repository `LICENSE`.

---

Suggested next steps:
- Add React hooks for browser integration
- Add WebTransport network backend
- Add persistence encryption for IndexedDB
- Add real-time change event streaming
- Add vector index optimization for large datasets