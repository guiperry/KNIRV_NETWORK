# KNIRVBASE conformance (Go ↔ TypeScript)

Go (`../go`) is the source of truth. The suites live next to the TS code in
`../ts/src/conformance/` and drive the Go helper `../go/cmd/nrvconform`
(jest builds it once per run via `scripts/jest-global-setup.cjs`).

| Suite | What is checked |
|---|---|
| `nrv` | registry JSON schema, byte-length padding + overflow, DepHead packing; Go↔TS `.nrv` files (every bracket byte-equal); signed frames; **NRV3 → KNIRVBASE conversion** incl. sharding and signing (E17/E18) |
| `pqc` | ML-KEM-768 + AES-256-GCM ciphertexts, ML-DSA-65 signatures, signed envelope, key JSON (`knirv-pqc-key/v2`) — both directions |
| `knirvql` | 47 parse cases (incl. errors) and 53 execute cases, each run with and without an index |
| `ticker_wal` | `FrameTicker` I/P deltas, drift, memory XOR, thermo aggregation; WAL format and torn-line handling |
| `flight` | Arrow IPC bracket batches both directions, drift score |
| `p2p` | shared-secret derivation, gateway proxy, Unix-socket server (both directions), manager failover |
| `p2p_interop` | a real libp2p TS node and Go node: gossipsub + direct-stream sync both ways; secret isolation |

```bash
cd ../ts && npm run test:conformance        # fails if Go is missing
NRVCONFORM_BIN=/path/to/nrvconform npx jest src/conformance   # prebuilt helper
```

Without Go on the PATH (and without `NRVCONFORM_REQUIRED=1`) the Go-dependent
cases skip; the pure cases still run. CI must set `NRVCONFORM_REQUIRED=1`
(`make test-knirvbase` does).

Opt-in, against real data: `KNIRV_REAL_NRV3=<hasher .nrv> go test ./internal/storage -run RealNRV3 -v`.
