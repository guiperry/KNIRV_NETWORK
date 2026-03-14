# KNIRVBASE: Implementation Plan for NEXUS Support & BuntDB Replacement

## Objective
Upgrade `KNIRVBASE` to become the primary, PQC-secured persistence and distribution layer for the `KNIRVSERVER` platform. This update will deprecate the use of `BuntDB` in favor of a sovereign, encrypted, and optionally distributed data engine.

## 1. Core Architecture Updates

### 1.1 Unified Storage Interface
Implement a new storage manager in `KNIRVBASE` that supports both structured (JSON) and unstructured (Markdown) data.
- **Key-Value API**: Support `Put(key, value)`, `Get(key)`, `Delete(key)`.
- **JSON Support**: Native `StoreObject(key, obj)` and `GetObject(key, dest)` with validation.
- **Markdown Projection**: Ability to "project" structured data into `.md` files for the Memory Fabric.

### 1.2 PQC Security Layer (MANDATORY)
- **Encryption at Rest**: Integrate `Kyber-768` for all data written to disk. Each tenant/user should have a unique encryption key derived from their session/identity.
- **Integrity Signatures**: Use `Dilithium-3` to sign every write operation. Verify signatures on every read to prevent "phantom" or "hallucinated" data injections.
- **Hardware TEE Integration**: Store private keys within the TEE enclave; decryption happens only within the enclave memory space.

## 2. Distributed Persistence & Session Management

### 2.1 P2P Synchronization
- **libp2p Integration**: Use the `KNIRV-NEXUS` P2P transport to synchronize critical data (like sessions and node discovery) across the network.
- **DHT Indexing**: Implement a DHT-based indexing system to replace `BuntDB` local indexes, allowing global lookups of DVE nodes and active sessions.

### 2.2 Session-Scoped Access Control
- **Session Tokens**: `KNIRVBASE` must recognize the hybrid tokens (JWT + Session Tokens) implemented in `KNIRVSERVER`.
- **Decryption-on-Demand**: Data should only be decrypted if the requesting DVE provides a valid, session-scoped access token.

## 3. Replacement of BuntDB Indexes
Implement a "Tagging" system in `KNIRVBASE` to replace `BuntDB` indexes:
- **Searchable Tags**: Allow attaching metadata tags to objects (e.g., `status:online`, `role:admin`).
- **Query Engine**: Support multi-tag queries (e.g., `Query("type:dve_node", "status:online")`).

## 4. KNIRVSERVER Refactor Steps

### Phase 1: Bridge Mode
1. Introduce `KNIRVBASEManager` alongside `BuntDBManager`.
2. Implement a "Dual-Write" strategy where critical data (Sessions, Auth) is written to both.

### Phase 2: Full Migration
1. Update `backend/internal/database/buntdb.go` to wrap `KNIRVBASE` instead of `BuntDB`.
2. Update all services (`AuthService`, `DVEManager`, `ValidationCore`) to use the new `KNIRVBASE` interface.
3. Migrate existing `BuntDB` files into the encrypted `KNIRVBASE` format.

## 5. Status & Visibility
- **Health Checks**: Implement submodule health reporting for `KNIRVBASE` (Online, Syncing, Storage Pressure).
- **Log Streaming**: Expose low-level PQC handshake and sync logs to the `KNIRVSERVER` dashboard.

---
**Prepared By**: Gemini CLI Agent
**Date**: February 18, 2026
