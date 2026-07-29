# KNIRVBASE

KNIRVBASE is KNIRV's local-first database and synchronization layer. It provides document collections, CRDT conflict resolution, vector clocks, local persistence, optional peer synchronization, and the NRV streaming data model.

This directory contains three implementations:

| Implementation | Role | Best fit |
| --- | --- | --- |
| [`go/`](go/) | **Flagship and protocol reference** | KNIRV services, nodes, streaming, and new core work |
| [`rust/`](rust/) | Alternative native implementation | Rust integrations and performance experiments |
| [`ts/`](ts/) | Alternative client implementation | Browser, Node.js, and TypeScript integrations |

## Which implementation should I use?

Use Go unless you have a specific requirement for Rust or TypeScript. The Go implementation is the canonical implementation for this repository: its public behavior, storage formats, synchronization semantics, network integration, and NRV support define the compatibility target for the other implementations.

The Rust and TypeScript directories are maintained as separate implementations, not language bindings generated from Go. They have their own APIs, storage backends, transports, manifests, tests, and release cadence. Their READMEs describe intended capabilities; verify compatibility with the Go implementation before using them for cross-language replication or production data interchange.

## Common model

All implementations are organized around the same broad concepts:

- local-first collections and background synchronization;
- CRDT operations resolved with vector clocks and deterministic tie-breaking;
- field-level encryption and post-quantum cryptography support;
- local blobs referenced by synchronized metadata rather than copied between peers;
- the `KNIRVQL` query language and vector-oriented data access.

The shared concepts do not imply identical feature parity. Transport, persistence, query coverage, authentication, and operational APIs are implementation-specific unless explicitly tested as compatible.

## Go quickstart

Requirements: Go 1.24 or a compatible toolchain (see [`go/go.mod`](go/go.mod)).

```bash
cd go
go test ./...
go build -o ./bin/knirvbase ./cmd/node
./bin/knirvbase --help
```

The public Go API is in [`go/pkg/knirvbase`](go/pkg/knirvbase). A minimal embedded database looks like this:

```go
db, err := knirvbase.New(ctx, knirvbase.Options{DataDir: "./data"})
if err != nil {
    return err
}
defer db.Shutdown()

docs := db.Collection("docs")
_, err = docs.Insert(ctx, map[string]any{"id": "one", "value": "hello"})
```

The node executable in [`go/cmd/node`](go/cmd/node) exposes the current HTTP and Arrow Flight adapters and supports data-directory, network, bootstrap-peer, and Flight-address flags. For NRV bracket streaming, use `knirvbase.NewNRV` and the `Dataset` API.

## Go architecture

- `go/pkg/knirvbase` — public database, collection, and NRV APIs.
- `go/pkg/nrv` — NRV bracket, frame, codec, and layout types.
- `go/internal/database` and `go/internal/collection` — database and collection orchestration.
- `go/internal/storage` — file and NRV storage, WAL, indexing, and compaction.
- `go/internal/resolver` and `go/internal/clock` — CRDT merge logic and vector clocks.
- `go/internal/p2pconsensus` — supported gateway/standalone synchronization path.
- `go/internal/network` — legacy custom TCP transport; it is disabled by default and should not be used for new work.
- `go/internal/query` — `KNIRVQL` parsing and execution.
- `go/internal/crypto/pqc` and `go/internal/security` — cryptographic and security primitives.

For distributed operation, configure a network secret in the consensus configuration for non-local deployments. An empty secret is open/legacy behavior and is suitable only for local demonstrations. See the [Go README](go/README.md) for the gateway callback channel, consensus modes, storage details, and benchmark commands.

## Other implementations

### Rust

[`rust/`](rust/) is an independent Rust implementation with async APIs, WAL-oriented storage, QUIC/DHT networking, and an HNSW vector index in its current design. Build and test it with:

```bash
cd rust
cargo check
cargo test
cargo build --release
```

Read [`rust/README.md`](rust/README.md) and [`rust/Distributed_Database_Implementation_rust.md`](rust/Distributed_Database_Implementation_rust.md) for its API and implementation-specific behavior. Rust is not the canonical source for protocol or storage changes.

### TypeScript

[`ts/`](ts/) is an independent TypeScript implementation targeting Node.js and browser environments. It includes filesystem and IndexedDB storage abstractions and TypeScript-native client APIs.

```bash
cd ts
npm install
npm run build
npm test
```

Read [`ts/README.md`](ts/README.md) for package usage, browser setup, and its current API. TypeScript is not the canonical source for protocol or storage changes.

## Compatibility policy

When changing KNIRVBASE behavior:

1. Define and test the behavior in Go first.
2. Update the Go specifications and tests when the change affects persistence, CRDT resolution, `KNIRVQL`, NRV encoding, or synchronization.
3. Port the change to Rust and TypeScript only when that implementation is intended to support it.
4. Document deliberate differences in the language-specific README rather than claiming parity.

Do not assume that a database directory, encrypted field, serialized operation, query, or peer transport produced by one implementation can be consumed by another without an explicit compatibility test.

## Documentation

- [Go implementation README](go/README.md)
- [Go database specification](go/docs/Database_Specification.md)
- [Go distributed database design](go/docs/Distributed_Database_Implementation_go.md)
- [KNIRVQL specification](go/docs/KNIRVQL_SPECIFICATION.md)
- [Rust implementation README](rust/README.md)
- [TypeScript implementation README](ts/README.md)

## License

See [`go/LICENSE`](go/LICENSE).
