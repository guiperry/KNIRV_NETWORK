# LLM Consolidation & KNIRVHASHER Software-Only Fix

Integrating the llama cognitive engine with KNIRVHASHER, and fixing KNIRVHASHER
to run software-only while keeping nonce and Merkle tree parameters under
KNIRV's own control.

## Current state: four disconnected LLM/embedding touchpoints

Before integration, it's worth naming the actual problem: KNIRV_NETWORK
already has **four separate, non-communicating LLM integrations**:

- KNIRVSERVER's embedded `llama.cpp` server (`knirvllama`) — local,
  generative, OpenAI-compatible API.
- KNIRVHASHER's pipeline — Cloudflare BGE-Base embeddings only.
- KNIRVGRAPH — Ollama embeddings only.
- KNIRVCHAIN — Cerebras cloud, generative, via a vendored `gollm_cerebras`
  file.

None of these reference each other. That fragmentation is the real
integration problem — "plug llama into KNIRVHASHER" is really "pick a
canonical local model runtime and make everything else a client of it."

The good news: KNIRVHASHER's own docs (`docs/hasher_validation_patch.md`,
"Assertion-Layer & LM Split") already independently arrived at almost
exactly the architecture proposed below, and most of it is marked
implemented. The task is less "invent a new design" and more "finish and
wire together a design that's already ~80% built."

## The software-only fix (nonce + Merkle control)

**Why it's needed:** The BM1382 ASIC KNIRVHASHER was built around turns out
to be hard-wired for Bitcoin's mining loop only — it can find a nonce
satisfying `SHA256(SHA256(header+nonce)) < target`, but can't do arbitrary
`SHA256(input)`. So the entire "hash as neuron activation" model had to be
smuggled through Bitcoin's 80-byte header shape (LSH projections packed into
the `Merkle Root` and `Previous Hash` fields, ASIC-found nonce read back as a
signature). That's ASIC-shaped by necessity, not by choice.

A pure software path already exists
(`pkg/hashing/methods/software/software.go`) implementing the same
`Execute21PassLoop`/jitter interfaces with plain `crypto/sha256` — but it's
flagged `ProductionReady: false`, and `core/sha256_canonical.go` still
hard-codes Bitcoin's exact difficulty target and 80-byte header layout even
in software mode. That's the actual bug: going software removes the reason
to stay Bitcoin-shaped, but the code never dropped the shape.

Three concrete fixes finish the job, and they line up with decisions
`hasher_validation_patch.md` already made:

1. **Decouple "production" from "hardware."** The patch already redefines
   what KNIRVHASHER attests: not hash-derived neural weights (avalanche
   effect breaks gradient descent for that — correctly abandoned), but
   PoW-witnessed `(context → fact)` assertions. Once that's the job, there's
   no reason software-mode PoW can't be production-legitimate — legitimacy
   should come from a KNIRV-defined difficulty/target you control, not from
   ASIC possession. Drop the Bitcoin target constant in
   `sha256_canonical.go`, replace with a configurable KNIRV difficulty, and
   re-flag `SoftwareMethod` as production-capable under that definition.
2. **Make nonce a first-class, KNIRV-controlled search parameter, not a
   Bitcoin brute-force counter.** `3_DATA_SEEDER`'s `EvolutionaryHarness`
   (population/mutate/fitness/select over nonce candidates) already does
   this — it's the real nonce-control mechanism going forward. The fix is
   to route production assertion-mining through the harness instead of the
   ASIC's Bitcoin-nonce-field convention, so nonce semantics (search space,
   mutation operators, acceptance mask) are entirely yours.
3. **Stop borrowing Bitcoin's Merkle Root field as a data-smuggling
   channel — use a real, KNIRV-owned Merkle tree.** KNIRVCHAIN already has
   one: `internal/blockchain/merkle.go`'s `TxMerkleRoot` with
   domain-separated leaf/parent hashing and a PLONK zk-circuit mirror.
   That's a better foundation than anything KNIRVHASHER would build from
   scratch, and it's already parameterized the way you want (your
   domain-separation bytes, your tree shape, no 32-byte Bitcoin-field
   constraint). Point KNIRVHASHER's assertion ledger at that Merkle
   implementation instead of the ASIC header's Merkle Root field.

Net effect: after this, "software-only" isn't a fallback mode you apologize
for — it's the primary path, with nonce and Merkle parameters fully under
KNIRV's definition instead of Bitcoin's.

## Making the hardware path work

The software-only fix above doesn't require abandoning the ASIC path — it's
also fixable, and the fixes are grounded in what's already in
`internal/driver/device/controller.go`.

**The one constraint that can't be fixed:** the code already documents this
correctly (controller.go:680-684) — *"a SHA-256 ASIC doesn't compute an
arbitrary caller-specified function on demand — it searches a mining header
for a nonce that makes the resulting hash satisfy a target, and returns that
nonce/hash pair, which is never equal to plain sha256(data)."* No firmware or
driver change adds generic-hash capability to a mining ASIC. Any hardware
path has to be built entirely around nonce-search-for-a-target, never around
evaluating an arbitrary hash.

**What's already there and correct:**

- `computeHashDirectASIC` → `BuildTxTaskFromHeader` → raw USB `SendPacket` →
  `pollForDirectNonce` (2ms poll, 500ms timeout) → substitute the found
  nonce back into the real header and recompute `sha256d` locally. Protocol-
  correct: midstate is a real `computeMidstate(header[0:64])`, not a
  stand-in.
- The TxTask wire format already has a real `nBits`/target field and a real
  starting-nonce field, at fixed offsets in the 45-byte `ASIC_TASK` payload
  — genuine Bitmain-protocol slots, the same ones pool software uses to set
  per-job difficulty.
- `ComputeBatch`/`MaxBatchSize = 256` already exists as a batch submission
  path.
- `internal/discovery` already supports discovering and connecting to
  multiple hasher-host/ASIC servers over the network, and
  `docs/HASHER_SDD.md` (line 1543) already documents an alternative
  "distributed mesh of 21 ASIC devices" design — real multi-chip ensemble
  isn't a new idea here, it's a shelved one.

**What's actually broken:** `ComputeBatch`'s inner loop calls a different,
older packet builder — `buildTxTaskPacket` — which hardcodes
`EasyTarget = 0x207FFFFF` ("any hash is valid, so ASIC finds nonce
immediately"). That's not proof-of-work at all; it turns the ASIC into a
fast rubber stamp, using the nonce purely as an entropy source rather than a
witnessed search. The single-hash path (`computeHashDirectASIC`) never uses
this shortcut. So today there are two hardware paths with two different
security properties, and the one used for the 21-pass ensemble (via
`ComputeBatch`) is the weak one.

**Concrete fixes:**

1. **Kill `buildTxTaskPacket`/`EasyTarget`, route `ComputeBatch` through
   `BuildTxTaskFromHeader`.** One packet builder, one set of security
   properties, used everywhere. This alone removes the trivial-target hole.
2. **Make `nBits` a real, KNIRV-chosen difficulty instead of either
   Bitcoin's Difficulty-1 or `EasyTarget`.** The wire field already exists —
   thread a configurable target through `BuildTxTaskFromHeader` the way a
   pool sets per-job difficulty. High enough for real PoW cost, low enough
   to fit the latency budget at ~500 GH/s.
3. **Verify determinism before trusting "first nonce" as a signature.** For
   the same header+target to be a reproducible bucket ID across different
   physical chips, the nonce-space search order has to be fixed and
   chip-independent. Test it: run the same header on two physical BM1382
   units and confirm they return the same lowest valid nonce. If they
   don't, collect and hash the K lowest nonces found within a time window
   instead of trusting a single "first."
4. **Fix the polling tax, not the compute.** At real difficulty the ASIC
   finds a nonce in microseconds; the 2ms fixed poll interval and per-pass
   round trip is the actual latency floor for the 21-pass loop. Pipeline
   pass N+1's submission instead of waiting out pass N's full timeout
   window, and route batched ensemble passes through `ComputeBatch` as one
   submission instead of 21 serial `computeHashDirectASIC` calls.
5. **Stop virtualizing the ensemble on one chip if more boards are
   available.** The single-ASIC temporal ensemble was a compromise for one
   Antminer on hand; the discovery code and the documented mesh design
   already exist to fan the 21 passes out across real independent ASICs
   instead of 21 sequential passes on the same silicon — near-21x wall
   clock, and truer to the ensemble's original robustness rationale.
6. **Pick one canonical backend.** `controller.go` currently supports
   raw-USB direct mode, a CGMiner-backed mode, and an ioctl/kernel-device
   mode. Difficulty and nonce-control fixes need to land once, not three
   times — designate direct-USB (the protocol-correct one) canonical for
   production and keep the others as dev/compat fallbacks only.

Fixes #1, #2, and #6 are the load-bearing ones for preserving KNIRV-controlled
nonce/target parameters in hardware mode — a targeted patch to existing code,
not a redesign. #3-#5 are about making the result trustworthy and fast enough
to matter.

## Wiring in the llama cognitive engine

With the attestation layer decoupled from hardware, `knirvllama` becomes the
natural generative/embedding engine for the whole hasher pipeline, not just
a KNIRVSERVER-local convenience. Five concrete integration points, roughly
in build order:

1. **Hallucination guardrail loop (highest value, mostly already
   scaffolded).** `attestation_bridge.go` already quantizes LM output into
   an LSH bucket and checks it against the assertion ledger without
   blocking generation on a miss. Wire llama's chat/completion output (via
   `knirvllama`'s OpenAI-compatible API) into `VerifyMath` for any claim
   MATHASHER's schema covers — variables, operators, logical derivations.
   This is the `/v1/verify/math` use case the docs already describe; it's
   currently unconnected to an actual LLM caller.
2. **Local embeddings instead of Cloudflare BGE.** `2_DATA_ENCODER`
   currently calls a Cloudflare worker for the Slot 0-3 embeddings. Since
   `knirvllama` and `hasher-host` are already launched as sibling
   subprocesses by the same launcher (`-llama` / `-hasher` flags, currently
   orthogonal with zero cross-wiring), add an embeddings endpoint to the
   local llama server and point the encoder at it. Removes an external
   dependency and puts both "LLM touchpoints" of the hasher under one
   locally-managed runtime.
3. **Cognitive-guided evolutionary search.** Let llama score candidate
   assertions in `EvolutionaryHarness` as an auxiliary fitness signal ("is
   this derivation internally consistent?") alongside the deterministic
   PoW/mask check. Important constraint: llama should only ever influence
   *search direction*, never the *acceptance boundary* — the mask/PoW check
   stays the sole deterministic gate, or you lose the verifiability the
   whole attestation layer exists for.
4. **Agent-level plumbing.** KNIRVAGENT's `HTTPProvider` is already a
   generic OpenAI-compatible client — point it at `knirvllama`'s socket,
   and have KNIRVULORA-compiled skill outputs that produce structured/
   logical claims pass through MATHASHER verification before being acted
   on. This extends the guardrail from "hasher pipeline" to "every agent
   skill invocation."
5. **One Merkle authority, not two.** Once KNIRVHASHER's assertion ledger
   uses KNIRVCHAIN's Merkle primitives (fix #3 above), have assertions land
   as a KNIRVCHAIN transaction type instead of a separate ledger. That
   gives KNIRVGRAPH's synthesizer a single place to query hash-attested
   facts against graph knowledge, closing the loop between the graph RAG
   layer and the attestation layer.

## Sequencing

The dependency order is: finish the software-mode fixes first (drop Bitcoin
target/header assumptions, adopt KNIRVCHAIN's Merkle tree, promote
`SoftwareMethod`) → then wire local embeddings (#2) since it's low-risk and
immediately removes an external dependency → then the verification loop (#1)
since the RPC (`VerifyMath`) and bridge code already exist → then the
evolutionary fitness signal (#3) and agent plumbing (#4) → Merkle/ledger
unification with KNIRVCHAIN (#5) is the largest structural change and should
come last, after the simpler pieces prove the pattern out.
