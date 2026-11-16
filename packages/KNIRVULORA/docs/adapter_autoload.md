# Adapter auto-load and base binding

Status: implemented in `internal/daemon` (`autoload.go`, `bind.go`), with tests in
`autoload_test.go` and `bind_test.go`.

## Data directory

Adapters live under the ulorad data dir, the same root the rest of the daemon
uses (see `internal/config`):

1. `ULORA_DATA_DIR`, if set;
2. otherwise `<KNIRV_APP_DATA_DIR>/ulora`, or the launcher app data dir `/ulora`.

Under that root:

| Path | Contents |
|------|----------|
| `adapters/<base12>/<bundle12>.gguf` | Adapter files written by `POST /ulora/v1/adapters/gguf`. `<base12>` is the first 12 hex chars of the base SHA-256. |
| `adapters/hash-cache.json` | Cache of base-model SHA-256 digests keyed by `path|size|mtime`. Safe to delete. |
| `adapters/autoload.json` | The auto-load descriptor (below). Read only at start. |

## Auto-load descriptor (`adapters/autoload.json`, version 1)

```json
{
  "version": 1,
  "adapters": [
    {
      "adapter_path": "adapters/0123456789ab/fedcba987654.gguf",
      "base_model_path": "/models/llama-3.1-8b-q4.gguf"
    }
  ]
}
```

- A missing file means nothing is auto-loaded. This is not an error.
- An unknown `version` loads nothing and logs why.
- A relative `adapter_path` resolves against `<data dir>/adapters`. An absolute
  path is used as given.
- `base_model_path` is read and hashed, never written.

## Verification at start

Each entry is checked independently. Start never aborts on a refusal.

1. The adapter must be a readable GGUF LoRA file (`general.type=adapter`,
   `adapter.type=lora`).
2. The adapter must record a non-empty `base_sha256`. An adapter with no recorded
   base cannot be verified, so it is refused.
3. The SHA-256 of `base_model_path` is computed with the same cached function the
   GGUF and bind routes use. It must equal the recorded digest
   (case-insensitive).

A refused entry is logged as `ulorad autoload: refused adapter <path>: <reason>`.
The reason names the adapter path and the two digests. It never includes key
material. A loaded entry is logged as `ulorad autoload: loaded adapter ...`.

Loaded adapters are listed by the authenticated route
`GET /ulora/v1/adapters/autoloaded`, which returns the adapter path, architecture,
skill, bundle hash and verified base digest for each one.

## Bind: base digest

`POST /ulora/v1/bind` takes an optional `base_model_path`. When it is set, the
daemon computes the base SHA-256 with the same function as the GGUF route and
returns it as `base_sha256` in the response. The digest is never taken from the
request: a caller-supplied digest cannot be checked, so the `base_sha256` request
field was removed. When `base_model_path` is omitted, the response has no
`base_sha256`.

## Design choices

- Owner decision: a base mismatch is refused and logged. Here that means the
  adapter is skipped and start continues.
- Auto-load verifies against the base file named in each descriptor entry. The
  daemon has no single configured base today, so there is no daemon-wide check.
- A refused adapter is not retried and is not replaced by the base model. The
  operator sees the refusal in the log and in the listing route.
