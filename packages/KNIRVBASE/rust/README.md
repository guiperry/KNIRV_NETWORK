# KNIRVBASE (Rust)

✅ **Overview**

KNIRVBASE is a high-performance, memory-safe distributed database prototype implemented in Rust. This is the optimized production reference implementation with zero runtime overhead, featuring:

- **PQC Encryption Layer**: Post-quantum cryptography using Kyber-768 (encryption) and Dilithium-3 (signatures) for secure data storage
- CRDT-based conflict resolution using **vector clocks**,
- **WAL (Write Ahead Logging)** for ACID compliance and crash recovery,
- **HNSW vector index** for approximate nearest neighbor search with 1M+ vector support,
- **Real P2P networking** with QUIC transport and DHT peer discovery,
- A small, human-friendly query language **KNIRVQL** with zero-copy parsing and execution,
- Memory safe implementation with zero unsafe code in the public API.

This package is intended as the high-performance server-side reference implementation for distributed collections and synchronization logic used across the KNIRV ecosystem.

---

## 🔍 Features

- **PQC Encryption at Rest**: Field-level encryption for sensitive data using Kyber-768 KEM + AES-256-GCM, with Dilithium-3 signatures for integrity
- Local-first operations and background sync with automatic conflict resolution
- CRDT resolve using vector clocks (merge rules + LWW tie-breakers)
- Write Ahead Logging (WAL) with atomic batch commits
- **HNSW Vector Index** with SIMD optimized distance calculations
- **Real P2P networking** with QUIC connections and DHT peer discovery
- `KNIRVQL` parser and executor with zero-copy parsing
- Async/await support using Tokio runtime
- Embedded HTTP API server with Axum integration
- Prometheus metrics endpoint
- Zero unsafe code in public API surface
- Memory efficient with minimal heap allocations on hot paths

---

## 🚀 Quickstart

### Prerequisites

- Rust 1.70+ (see `rust-toolchain.toml`)

### Build

```bash
cargo build --release
```

### Run

```bash
./target/release/knirvbase
```

### Usage Example

```rust
use knirvbase::DistributedDatabase;
use knirvbase::Config;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Initialize database
    let config = Config {
        data_dir: "~/.local/share/knirvbase".into(),
        enable_p2p: true,
        enable_encryption: true,
        ..Default::default()
    };

    let db = DistributedDatabase::open(config).await?;

    // Create collections
    db.create_collection("auth").await?;
    db.create_collection("memory").await?;

    // KNIRVQL operations
    db.query(r#"SET google_maps_api_key = "AIzaSy...""#).await?;
    let result = db.query(r#"GET AUTH WHERE key = "google_maps_api_key""#).await?;

    // Vector similarity search
    let similar = db.query(r#"
        GET MEMORY WHERE source = "web-scrape" 
        SIMILAR TO [0.45, 0.12, ...] 
        LIMIT 10
    "#).await?;

    Ok(())
}
```

---

## 📁 Storage Layout

- Data directory (default): `$XDG_DATA_HOME` or `~/.local/share/knirvbase`
- Write Ahead Log: `<datadir>/wal/`
- Per-collection SSTables: `<datadir>/<collection>/<generation>.sst`
- Vector index: `<datadir>/<collection>/hnsw.index`
- Blobs are saved under `<datadir>/<collection>/blobs/<id>`

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

The language is intentionally minimal and aimed at quick integration. Query optimizer automatically selects appropriate indexes.

---

## 📦 Package Overview (what's inside)

- `lib.rs` — Main exports and public API entry point
- `main.rs` — CLI binary entry point
- `database.rs` — `DistributedDatabase`: high-level database orchestration
- `collection.rs` — `DistributedCollection` + `LocalCollection`: local storage, CRDT operation emission, sync logic
- `crypto/pqc.rs` — Post-quantum cryptography: Kyber-768 encryption, Dilithium-3 signatures, key management
- `network.rs` — `NetworkManager`: P2P networking with QUIC + DHT peer discovery
- `resolver.rs` — CRDT resolver logic and merge operations
- `query.rs` / `query_parser.rs` — `KNIRVQL` parser and query execution
- `storage.rs` — File persistence with SSTable format
- `wal.rs` — Write Ahead Log implementation for crash safety
- `hnsw.rs` — Hierarchical Navigable Small World vector index
- `index_manager.rs` — Secondary index management
- `clock.rs` — Vector clock implementation and comparison utilities
- `security.rs` — Key derivation, secure memory handling, constant time operations
- `auth.rs` — Authentication and permission system
- `types.rs` — Core types (Document, CRDTOperation, NetworkConfig, ProtocolMessage, etc.)

---

## 🛠 Development & Testing

### Makefile Usage

All common operations are automated via the included Makefile:

```bash
# Show all available commands
make help

# Build targets
make build                # Debug build
make build-release        # Optimized release build
make clean                # Clean all build artifacts

# Development
make check                # Fast compilation check
make fmt                  # Format code
make lint                 # Run Clippy lints (strict mode)
make run                  # Run debug build
make run-release          # Run release build

# Testing
make test                 # Run all tests
make test-debug           # Run tests with debug logging
make test-log             # Run tests with full trace logging
make bench                # Run all benchmarks
make bench-sla            # Run SLA performance validation

# Analysis & Documentation
make doc                  # Build documentation
make doc-open             # Build and open docs in browser
make audit                # Run security audit
make flamegraph           # Generate performance flamegraph
```

### Manual Cargo Commands

- Run all tests:
```bash
cargo test
```

- Run tests with logging:
```bash
RUST_LOG=debug cargo test
```

- Run benchmarks:
```bash
cargo bench
```

- Clippy lints:
```bash
cargo clippy --all-targets --all-features -- -D warnings
```

- Format code:
```bash
cargo fmt
```

- Build release:
```bash
cargo build --release
```

---

## 📊 Performance Benchmarks

KNIRVBASE Rust includes a comprehensive benchmark suite that validates performance against ASIC-Shield SLA requirements:

### SLA Targets (ASIC-Shield Integration)
- **Credential Insert**: p99 < 3ms
- **Credential Query**: p99 < 1ms
- **Authentication Workflow**: p99 < 120ms (including 100M PBKDF2 iterations)
- **PQC Encryption**: < 8ms per operation
- **Vector Search**: < 5ms p99 for 1M vectors
- **Large Scale**: No performance degradation with 100K+ credentials

### Running Benchmarks
```bash
# Run all benchmarks
cargo bench

# Run SLA validation
cargo bench --bench sla_validation

# Generate flamegraphs
cargo flamegraph --bench vector_search
```

---

## ⚠️ Limitations & Security Notes

- **PQC Encryption at Rest:** Field-level encryption for all sensitive data across collections (`credentials`, `pqc_keys`, `sessions`, `audit_log`, `threat_events`, `access_control`). Encrypts specific fields like `hash`, `salt`, `token_hash`, `details`, `indicators`, `permissions`, etc. Uses Kyber-768 + AES-256-GCM for confidentiality and Dilithium-3 for integrity. Master key must be configured for encryption to be active.
- **P2P Networking:** QUIC based P2P networking with DHT peer discovery enables true distributed operation across multiple nodes.
- **Blob handling:** Blobs are stored locally and only referenced in synchronized metadata; no blob distribution is implemented here.
- **Memory Safety:** Zero unsafe code in public API. All cryptographic operations use constant time implementations.
- **Crash Safety:** Write Ahead Logging ensures atomicity and durability. Database automatically recovers on restart without data loss.

---

## 📚 Design & Reference

The repository includes `Distributed_Database_Implementation_rust.md` which documents architecture, rationale, and design decisions in depth — consult it for more detail on synchronization heuristics, CRDT rules, and future extensions.

---

## 💡 Contributing

- Open an issue for feature requests or bug reports
- Create a PR with tests and descriptions of changes
- Keep code and docs consistent with Rust idioms and the repository's architecture
- Maintain zero unsafe code policy for public API
- All changes must pass existing tests and Clippy lints
- Benchmark performance critical changes

---

## 📜 License

See the repository `LICENSE`.

---

Suggested next steps:
- Add libp2p transport backend
- Add RocksDB storage backend
- Add incremental vector index updates
- Add distributed transactions
- Add replication and consensus