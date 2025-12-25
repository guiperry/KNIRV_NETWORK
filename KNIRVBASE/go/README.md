# KNIRVBASE (Go)

✅ **Overview**

KNIRVBASE is a lightweight, local-first distributed database prototype implemented in Go. It demonstrates a minimal, self-contained implementation of:

- **PQC Encryption Layer**: Post-quantum cryptography using Kyber-768 (encryption) and Dilithium-3 (signatures) for secure data storage
- CRDT-based conflict resolution using **vector clocks**,
- Local file-backed **storage** for metadata and blobs (blobs are stored locally),
- **Real P2P networking** with DHT-like peer discovery and TCP-based communication,
- A small, human-friendly query language **KNIRVQL** for simple GET/SET/DELETE and basic vector similarity queries.

This package is intended as an architectural prototype and reference implementation for distributed collections and synchronization logic used across the KNIRV ecosystem.

---

## 🔍 Features

- **PQC Encryption at Rest**: Field-level encryption for sensitive data using Kyber-768 KEM + AES-256-GCM, with Dilithium-3 signatures for integrity
- Local-first operations and background sync
- CRDT resolve using vector clocks (merge rules + LWW tie-breakers)
- File-based storage for documents and local blob files
- **Real P2P networking** with TCP connections and DHT-like peer discovery
- `KNIRVQL` parser and executor for quick interactive operations
- Convenience command-line demo in `cmd/main.go`

---

## 🚀 Quickstart

### Prerequisites

- Go 1.24 (see `go.mod`) — ensure your toolchain matches or is compatible: `go version`

### Build

```bash
make build
# or
go build -o ./bin/knirvbase ./cmd
```

### Run

The binary stores data in your OS application data directory by default. On Linux it uses `$XDG_DATA_HOME` or falls back to `~/.local/share/knirvbase`.

```bash
./bin/knirvbase
```

The included `cmd/main.go` file demonstrates:

- Creating a distributed database instance
- Creating/joining a network
- Creating two collections: `auth` and `memory`
- Example `KNIRVQL` usage to `SET` and `GET` authentication keys
- Inserting a `MEMORY` entry with a vector payload and querying via similarity

Press Ctrl+C to exit the running demo.

---

## 📁 Storage Layout

- Data directory (default): `$XDG_DATA_HOME` or `~/.local/share/knirvbase`
- Per-collection JSON files: `<datadir>/<collection>/<id>.json`
- Blobs (for `MEMORY` entries) are saved under `<datadir>/<collection>/blobs/<id>` and are not automatically synchronized across peers — only metadata and vectors are synchronized.

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

- Insert a memory item (demonstrated programmatically in `cmd/main.go`)

- Get similar memory entries (vector search):

```knirvql
GET MEMORY WHERE source = "web-scrape" SIMILAR TO [0.45, 0.12] LIMIT 10
```

The language is intentionally minimal and aimed at quick integration and demos — for production you will likely expose richer query primitives or integrate with a dedicated vector index.

---

## 📦 Package Overview (what's inside)

- `cmd/` — small demo CLI (`cmd/main.go`) showing initialization and sample operations.
- `internal/crypto/pqc` — Post-quantum cryptography: Kyber-768 encryption, Dilithium-3 signatures, key management.
- `internal/database` — `DistributedDatabase`: high-level database orchestration and collection factory.
- `internal/collection` — `DistributedCollection` + `LocalCollection`: local storage, CRDT operation emission, sync logic.
- `internal/storage` — `FileStorage`: file-based persistence and blob handling with PQC encryption.
- `internal/network` — `Network` interface + `NetworkManager`: real P2P networking with TCP connections and DHT-like peer discovery.
- `internal/resolver` — CRDT resolver logic and helpers for converting to/from distributed documents.
- `internal/clock` — vector clock implementation and comparison utilities.
- `internal/query` — `KNIRVQL` parser and query execution.
- `internal/types` — core types (Document, CRDTOperation, NetworkConfig, ProtocolMessage, etc.)

---

## 🛠 Development & Testing

- Run tests (where present):

```bash
go test ./...
```

- Format & vet:

```bash
gofmt -w .
go vet ./...
```

- Manage modules:

```bash
go mod tidy
```

Note: The codebase includes a full P2P networking implementation with TCP connections and DHT-like peer discovery for distributed operation.

---

## ⚠️ Limitations & Security Notes

- **PQC Encryption at Rest:** Field-level encryption for all sensitive data across collections (`credentials`, `pqc_keys`, `sessions`, `audit_log`, `threat_events`, `access_control`). Encrypts specific fields like `hash`, `salt`, `token_hash`, `details`, `indicators`, `permissions`, etc. Uses Kyber-768 + AES-256-GCM for confidentiality and Dilithium-3 for integrity. Master key must be configured for encryption to be active.
- **P2P Networking:** Real TCP-based P2P networking with DHT-like peer discovery enables true distributed operation across multiple nodes.
- **Blob handling:** Blobs are stored locally and only referenced in synchronized metadata; no blob distribution is implemented here.
- **No authentication for network messages:** This prototype does not implement cryptographic signing or authenticated transports — real deployments must use TLS/Noise and authenticated peer identities.

---

## 📚 Design & Reference

The repository includes `Distributed_Database_Implementation_go.md` which documents architecture, rationale, and design decisions in depth — consult it for more detail on synchronization heuristics, CRDT rules, and future extensions (IPFS integration, libp2p transport, vector indexes).

---

## 💡 Contributing

- Open an issue for feature requests or bug reports
- Create a PR with tests and descriptions of changes
- Keep code and docs consistent with Go idioms and the repository's architecture

---

## 📜 License

See the repository `LICENSE`.

---

If you'd like, I can also:
- Add usage examples as runnable scripts
- Add unit tests for untested modules (network, resolver)
- Add performance benchmarks for P2P operations

🔧 **Next step:** tell me which of the above you'd like me to implement next (tests, benchmarks, or example scripts).