# Live lookup ("flash") and the SearXNG metasearch engine

Status: **provisional, unverified against a live instance.** The owner chose SearXNG "for
now" (O5, 2026-11-03). Nothing here installs or starts anything; this is what an operator
does by hand. The JSON shape the provider reads comes from SearXNG's documented search API,
not from a run.

## Switches

Live lookup is **on by default**. Turn it off with either:

| Setting | Effect |
|---|---|
| `KNIRV_FLASH=0` (also `false`, `off`, `no`) | live lookup off |
| `KNIRV_NO_FLASH=1` (also `true`, `on`, `yes`) | live lookup off; wins over `KNIRV_FLASH` |
| `factuality.flash.enabled: false` in the SERVER yaml | same intent; the launcher maps it to `KNIRV_FLASH` (and `metasearch_url`/`metasearch_engines`/`allow_domains`/`deny_domains` to `KNIRV_METASEARCH_*`, see `KNIRVSERVER/internal/launcher/factualityenv`); the direct environment variables above remain available as explicit overrides |

With it off, every `flash` or `hybrid` request is answered from the evidence index and the
response says so (`mode: "index"`, a note). A caller can also forbid it per request with the
harness constraint `NoFlash` (the backend's factuality path always does, see Task 3.6).

Without `KNIRV_METASEARCH_URL`, live lookup uses only the built-in Wikipedia and arXiv
providers. Setting it adds the metasearch provider:

| Variable | Meaning |
|---|---|
| `KNIRV_METASEARCH_URL` | `http://127.0.0.1:PORT` or `unix:///path/to.sock`. Loopback or a socket only: anything else is refused at start. |
| `KNIRV_METASEARCH_ENGINES` | comma-separated engine allow-list passed as `engines=` |
| `KNIRV_METASEARCH_ALLOW` | comma-separated registered domains; if set, only these are read |
| `KNIRV_METASEARCH_DENY` | comma-separated registered domains that are never read |
| `KNIRV_USER_AGENT` | identifies the fetcher to sites |

## What leaves the machine

Only a claim's minimized search words (at most 8 words, 100 characters, secrets and
identifiers redacted) go to the engine, and the engine queries upstream engines from this
host's IP. Result pages are then fetched directly: https only, public addresses only (every
DNS answer is checked and the connection goes to a checked address), HTML or plain text,
1 MiB, no cookies or referrer. A verdict resting only on pages fetched this way needs two
independent registered domains.

## Running SearXNG (operator steps, not automated)

1. Install SearXNG by its own instructions (container or Python). It is AGPL-3.0 licensed;
   KNIRV talks to it only over HTTP, as a separate process.
2. In its `settings.yml`: enable the JSON output (`search: formats: [html, json]`), bind to
   loopback (`server: bind_address: "127.0.0.1"`, `port: 8888`) or a Unix socket, set a
   `secret_key`, and turn the bot limiter off for a loopback-only instance
   (`server: limiter: false`). Choose the engines you trust; keep the list short.
3. Check it: `curl 'http://127.0.0.1:8888/search?q=eiffel+tower&format=json'` should return
   `{"results":[{"url":...,"title":...,"content":...}]}`.
4. Start the MODELS daemon with `KNIRV_METASEARCH_URL=http://127.0.0.1:8888`.

Supervising the engine like llama.cpp (Task 3.8's "supervised process") is not built; it
needs a decision on where the process manager lives and on shipping a Python service.
