# KNIRVSERVER Commit Strategy: A Slim Evidence Ledger, Not a Forgejo Deployment

## 0. Why This Document Exists

`entirely_new_cli.md` (Phases H/I/L/M) assumed KNIRVSERVER runs a full private Forgejo instance that hosts both project source and DVE evidence, and that the CLI pushes an evidence ref alongside the code ref to that same instance. That assumption is retracted. The operator constraint driving this document:

- No private Forgejo deployment. Not "not yet" — actively discouraged given current backlog, the learning curve of adopting an unfamiliar system under time pressure, and an already-overloaded KNIRVSERVER that doesn't need another full service to operate, patch, and reason about.
- The user's actual source repository stays exactly where it already is — GitHub, GitLab, a personal Forgejo/Gitea, wherever. KNIRV does not become a source-hosting product.
- What KNIRVSERVER needs is much narrower than "host git repos": a place to durably, verifiably store **proof bundles** (the signed `dve.Bundle` evidence already produced locally today) as git commits, linked by hash to KNIRV's validation chain and to whatever the user's real repo already recorded.

This document analyzes the cloned Forgejo source (`/home/gperry/Documents/GitHub/forgejo`) for reusable parts, finds that the code itself is not usable, and proposes a slim, purpose-built alternative informed by Forgejo's architecture rather than built from its source. It stands alone — `entirely_new_cli.md` is not edited yet, pending deliberation on how the two reconcile.

## 1. Forgejo Analysis: What's There, and the License Finding That Changes the Plan

Forgejo is a hard fork of Gitea, **~all first-party Go code under GPLv3** (root `LICENSE`; some files retain inherited MIT SPDX headers from the pre-fork Gitea codebase, but the project's own governing license for the distributed work is GPLv3 — GPLv3 has no linking exception, so this covers the whole binary, not just the files that still say MIT).

**Update, resolved:** `KNIRV_NETWORK` (the monorepo containing KNIRVSERVER) has been relicensed **GPLv3**, confirmed at its root `LICENSE`, with a public release planned. GPLv3 explicitly permits incorporating GPLv3 code into your own project provided the combined work is also distributed under GPLv3 — which this now is. **The "No — GPLv3" verdicts below are superseded: direct cloning of the packages listed is now endorsed, not just architectural study.** Two housekeeping obligations that come with it, not blockers: preserve copyright/license headers on any file carried over from Forgejo, and keep a rough record of what was copied verbatim vs. modified. GPLv3's source-disclosure duty triggers on *conveying* (distributing) the software, not on running it as a network service — moot here anyway since the whole repo is going public.

Subsystem-by-subsystem findings (full detail from the research pass, condensed here):

| Subsystem | What it is | Reusable as code? |
|---|---|---|
| `modules/git` | Thin `os/exec`-based wrapper: repo init, commit-tree, hash-object, ls-tree, ref management, hook file management. Genuinely low-dependency — no `models/*`, no DB, no user model. | **Yes — clone directly**, preserving license headers. It's already proven low-dependency (§ above), so porting it in whole or in part is low-risk relative to hand-rolling an equivalent. |
| Hook dispatch (`modules/git/hook.go`, `modules/git/hook_generate.go`, `modules/private/hook.go`, `cmd/hook.go`, `routers/private/internal.go`) | **The single most valuable finding.** Forgejo does not write a hook script into every repo. It sets one global `core.hooksPath` git config pointing at one shared directory containing a dispatcher script, which re-execs the Forgejo binary as `forgejo hook pre-receive`. That subcommand reads git's hook stdin protocol (`old-oid new-oid ref-name` lines) plus pusher-identity environment variables, then makes **one authenticated HTTP call back to the always-running server process** — bearer token compared with `crypto/subtle.ConstantTimeCompare`, over a Unix domain socket when configured, so an out-of-process CLI invocation can synchronously ask the live server "is this push allowed / what do I do with it." | **Yes — clone the pattern and, where convenient, the code.** The dispatcher-script/re-exec/loopback-callback shape maps directly onto §3.3's design; KNIRVSERVER's version will differ in payload shape and auth (its own token, not Forgejo's), so treat this as "port the mechanism," lifting literal code where it saves time. |
| `modules/lfs` / `services/lfs` | Content-addressed blob storage over a pluggable backend, plus the LFS batch-API protocol. The storage-layer half is clean; the protocol half is entangled with Forgejo's DB/user/repo models. | **Available, but still not needed** — see §3.5. Large artifacts don't go through git here regardless of licensing. |
| `modules/storage` | A tiny, clean `ObjectStorage` interface (`Open/Save/Stat/Delete/URL/IterateObjects`) over local-disk or S3/minio backends. Decoupled from git entirely. | **Yes — clone directly** if KNIRVSERVER's existing `internal/dveevidence.FileStore` (local-disk-only today, confirmed by direct inspection — see §3.6) ever needs an S3/minio backend; this is a ready-made abstraction for that. |
| `models/asymkey` (GPG/SSH commit signature verification) | Verifies a commit's signature against a *database user's* registered keys, computes a UI trust badge. Fundamentally tied to Forgejo's user/DB model, not a standalone "verify signature against public key" primitive. | **Licensing is no longer the blocker, but there's still no incremental value.** KNIRV already has its own Ed25519 bundle-signing scheme; this is UI/trust-model plumbing tied to Forgejo's DB, not a cryptographic primitive worth porting. |
| Bare-repo creation (`services/repository/create.go`, `modules/repository/init.go`) | Once you strip away Forgejo's DB bookkeeping, the git-side operation is `mkdir -p && git init --bare --template ""`. No per-repo hook install call — hooks work automatically via the global `core.hooksPath` from the row above. | **Yes — clone directly**, it's a handful of lines once isolated from `services/repository`'s DB transaction wrapper. |
| Smart-HTTP receive-pack handler (`routers/web/repo/githttp.go`) | Confirmed by direct inspection: Forgejo does **not** use go-git or a hand-rolled pkt-line implementation to serve pushes. It shells out to the real `git receive-pack --stateless-rpc` binary (`githttp.go:319,399`, via its `modules/git` command wrapper) and pipes the HTTP request/response through it. | **Yes — clone directly. This is now the primary recommendation for §3.2**, superseding the earlier go-git-vs-CGI framing (see §3.2, revised). |

**License-boundary rule, updated:** with `KNIRV_NETWORK` now GPLv3, cloning Forgejo's Go packages listed above is permitted and is now the recommended path where it saves real effort — port `modules/git`, the `core.hooksPath`/dispatcher pattern, and `githttp.go`'s receive-pack handler directly rather than reimplementing them from a blank page. Keep license headers intact on carried-over files.

Running actual Forgejo as a black-box sibling process remains rejected — not for licensing reasons (that was never the issue even under the old premise) but because it reintroduces exactly the operational cost — another full service, another system to learn, more backlog — that this document exists to avoid. The goal is still "clone the useful packages into a slim purpose-built binary," not "run Forgejo."

## 2. What Changes From `entirely_new_cli.md`

For deliberation, stated plainly, not yet applied to that file:

- **Phase I (Forgejo Transport)** is retracted as written. There is no KNIRVSERVER Forgejo to push evidence refs to. Replaced by §3 below: a slim, purpose-built proof-ledger service, and the credential-helper idea survives (§3.4) but points at this new endpoint instead.
- **Phase L (Large-Artifact Storage / Git LFS)** is retracted. With no Forgejo, there's no LFS server to lean on — and it turns out not to be needed (§3.5): large artifacts route through KNIRVSERVER's existing evidence/object storage by content hash, exactly as `ArtifactRef.Hash` already anticipates, never through git at all.
- **Phase M (Server-Side Ingest via Forgejo Webhook)** is retracted as "webhook" — there's no Forgejo webhook system. Replaced by the hook-dispatch pattern in §3.3, which achieves the same "server is notified the instant evidence lands" property via a mechanism KNIRV controls end-to-end.
- **Phases H (Sanctioned Commit Path), J (Redaction Gate), K (Git-Native Signing), N (Attribution Queries)** are essentially unaffected — they operate on the user's real repo (wherever hosted) and on the local `dve.Workspace`, neither of which depended on KNIRVSERVER hosting anything. Phase H's trailer-append step now points the `KNIRV-Evidence-Ref` trailer at KNIRVSERVER's proof ledger instead of a same-host evidence ref.
- **`DVE_Alignment_Plan.md` Phase 7 (Server DVE Ingest)** — the reconciliation this document proposes is actually *closer* to that phase's original raw-upload-endpoint framing than `entirely_new_cli.md`'s Forgejo-webhook redesign was. The git-receive-pack push in §3.2 effectively *is* the ingest endpoint, just speaking git's wire protocol instead of a bespoke multipart upload — which buys the append-only/content-addressed/audit-log properties of git without requiring KNIRVSERVER to run anything as heavy as Forgejo.

## 3. Proposed Architecture: The KNIRV Proof Ledger

```text
User's real source repo (GitHub / GitLab / self-hosted / anywhere — unchanged, untouched)
  |
  | client-side git hook, installed by `knirv supervisor init`:
  |   prepare-commit-msg: append KNIRV-DVE-Session / KNIRV-Evidence-Ref trailers
  |   (same mechanism as entirely_new_cli.md Phase H — unaffected by this document)
  v
User's real commit, now carrying a trailer that points at KNIRVSERVER's proof ledger
  |
  |  (separately, at `knirv commit` / `knirv dve session commit` time)
  v
KNIRV CLI
  - finalizes + Ed25519-signs the dve.Bundle (dve/sign.go, unchanged)
  - large artifacts (memvid, recordings) uploaded to KNIRVSERVER's existing
    evidence/object storage by content hash (NOT through git — see 3.5)
  - `git push` the bundle + event log + small artifact manifest to
    knirvserver-proof-ledger, via a KNIRV-issued credential (3.4)
  |
  v
KNIRVSERVER Proof Ledger  (new, slim — this document's actual subject)
  - one bare repo per project: data/proof-ledger/<project_id>.git
  - receives the push over git's smart-HTTP protocol (3.2)
  - a shared hooks/ dir + core.hooksPath dispatcher re-execs
    `knirvserverd hook post-receive` (3.3, patterned on Forgejo's
    dispatch mechanism, containing none of its code)
  - that hook subcommand makes one authenticated loopback call into the
    already-running KNIRVSERVER process
  |
  v
KNIRVSERVER Proof Ledger ingest (new, dedicated to the Bundle Sign Path —
  NOT internal/dveevidence, which remains dvepod's separate, untouched
  Pod Evidence / Dock Path; see 3.6)
  - verifies signature, hash chain, Merkle root, policy replay
  - links this evidence commit's git hash into KNIRV's validation chain
  - emits validation certificate
```

Note: `dvepod`'s evidence never enters this pipeline — it flows through the existing, separate `internal/dveevidence` HTTP-upload path (`/api/dve/:dve/sessions/ingest`), unchanged by this plan. See §3.6 for the full two-track picture.

### 3.1 Repository Topology

One bare repo per project, not one giant repo for the whole server: `data/proof-ledger/<project_id>.git`. Rationale: natural permission boundary (per-project access control maps directly onto KNIRV's existing project/org model rather than needing a new one), no cross-project ref namespace collisions, and a single project's evidence history stays independently cloneable/inspectable for debugging or audit without exposing every other project's ledger.

Inside each bare repo, evidence lands on `refs/knirv/dve/<session_id>` — unchanged from `entirely_new_cli.md`'s ref-naming decision (§8.2 of that document), sharded tree layout `<id[:2]>/<id[2:]>/bundle.json`, `events.jsonl`, `artifact-manifest.json` (small pointer file, not the artifacts themselves — see 3.5).

### 3.2 Transport: How a Push Actually Lands

**Resolved.** With Forgejo's code now cloneable (§1), the "verify go-git first" question mostly evaporates — Forgejo itself, a mature and heavily used product, doesn't trust go-git or a hand-rolled pkt-line implementation for this. Confirmed by direct inspection of `routers/web/repo/githttp.go`: it shells out to the real `git receive-pack --stateless-rpc` binary via its `modules/git` command wrapper (`githttp.go:319`: `git.NewCommand(ctx, "receive-pack", "--stateless-rpc", "--advertise-refs", ".")`, `githttp.go:399` for the actual RPC call) and pipes the HTTP request/response body straight through it. That's strong, production-proven evidence for the approach.

**Recommendation: port `githttp.go`'s receive-pack handler directly**, stripped of everything specific to Forgejo's permission/user model (the auth check gets replaced with §3.4's token validation; the repo-path resolution gets replaced with KNIRVSERVER's `project_id → data/proof-ledger/<project_id>.git` mapping). Concretely: a small Go HTTP handler that (1) validates the request under §3.4's auth, (2) resolves the target bare repo path, (3) for `GET .../info/refs?service=git-receive-pack`, execs `git receive-pack --stateless-rpc --advertise-refs .` in that directory and streams stdout back with the right `Content-Type`; (4) for `POST .../git-receive-pack`, execs `git receive-pack --stateless-rpc .` and pipes the request body in / response out. No pkt-line parsing of our own, no go-git server-transport dependency, no CGI environment-variable plumbing (`git http-backend`'s CGI contract, option (b) in the prior draft, is no longer necessary once the equivalent logic can just be read directly out of `githttp.go`).

The dropped alternatives, for the record: go-git's server-side transport was deprioritized because its receive-pack support has historically lagged its client-side maturity and there's no need to gamble on it now that a working reference exists; hand-rolling the smart-HTTP protocol from scratch was already rejected (`entire-cli`'s own test plan calls the pkt-line/divergence surface "the most re-broken area" even in a mature implementation).

Either way, the CLI side is unaffected — it's pushing to a normal `https://` git remote.

### 3.3 Hook Dispatch (the Forgejo-pattern piece)

Reimplemented fresh, containing none of Forgejo's code, following the pattern documented in §1:

1. One shared directory, e.g. `data/proof-ledger/.hooks/`, containing a small dispatcher script and a delegate script that re-execs the KNIRVSERVER binary: `knirvserverd hook post-receive`.
2. Every bare repo under `data/proof-ledger/` gets `core.hooksPath` set to that shared directory at creation time (one `git config` call, not a per-repo file write) — avoids the "write and maintain N copies of a hook script" problem entirely, exactly as Forgejo's design avoids it.
3. `knirvserverd hook post-receive` reads git's hook stdin (`old-oid new-oid ref-name` lines — a stable, documented git protocol, not Forgejo-specific), extracts the pushed ref (expects `refs/knirv/dve/*`; anything else is a no-op or a reject, since this ledger has exactly one legitimate use), and makes one authenticated call back into the running KNIRVSERVER process.
4. **Resolved: that callback is a Unix domain socket loopback, not a TCP/gateway-routed call, and it carries a bearer token.** Confirmed by direct inspection that this matches existing precedent rather than inventing a new one — `AgentControlServer` (`main.go:227-271,422-426`) already binds exclusively to a chmod'd Unix socket for exactly this "only local processes should reach this" reason, and `/api/dve/*` (the nearest existing analog to this new route) already bypasses KNIRVGATEWAY entirely today, served directly off KNIRVSERVER's public port. Unlike `AgentControlServer`, which relies on socket-file-permissions alone with no token, the proof-ledger's callback adds a bearer token minted at KNIRVSERVER boot and handed to the hook subprocess via env var — the same propagation mechanism already used to pass the JWT secret to the spawned KNIRVGATEWAY child process (`manager.go:267`, plain env inheritance). This is new code, so there's no reason to repeat `AgentControlServer`'s no-token gap in it.

   **This is a deliberate, tracked exception to "no endpoint should bypass the gateway," not a quiet violation of it.** The operator's standing policy is that every endpoint routes through KNIRVGATEWAY, full stop — this callback (and `AgentControlServer` before it) are accepted as interim debt because the hook subprocess and KNIRVSERVER are structurally always the same host/container in every deployment topology (the hook is a child of the git-receive-pack process KNIRVSERVER itself hosts), and building gateway-routing for a same-process loopback call right now isn't worth the time against the rest of this plan. **File a backlog item now covering the general fix**, scoped to include: `AgentControlServer`'s existing bypass, this new hook-callback bypass, and — forward-looking — whatever channel eventually connects the CLI-side Supervisor to a server-side counterpart agent (the operator is evaluating a formal split naming the CLI supervisor "Supervisor" and a server-side agent "Expert Advisor"; any Supervisor↔Expert-Advisor channel should be designed gateway-compliant from day one rather than adding a third instance of this same debt). The endpoint itself: `POST /internal/proof-ledger/ingest` on the Unix socket, carrying `{project_id, session_id, ref, new_oid}` plus the bearer token. It fetches the actual bundle from the just-pushed commit (already on disk — a local `git show`/`git cat-file`, not a network fetch) and hands it to the ingest logic in §3.6.

This is a genuinely small amount of new code — a stdin-line reader, an env-var extractor, and one authenticated Unix-socket HTTP call — matching the actual size of Forgejo's own `cmd/hook.go` core logic once you strip out its AGit/proc-receive sideband handling, which KNIRVSERVER's proof ledger has no reason to need.

### 3.4 Auth: Reuse What Already Exists (Public-Facing Push, Distinct From §3.3's Internal Token)

Note this is a **different trust boundary from §3.3 step 4's Unix-socket token** — that one authenticates KNIRVSERVER's own hook subprocess to itself and never leaves the host; this one authenticates the external CLI's `git push` over `https://`, and it does need to go through KNIRVGATEWAY like everything else public-facing.

**Resolved:** the CLI will eventually get a formal login flow against KNIRVSERVER once the product is mature enough to ship — full identity, not built yet. **Until then, use simple bearer tokens.** `knirv git-credential` mints/attaches a KNIRVSERVER-issued token as the `password` half of standard git HTTP Basic auth against the proof-ledger's `https://` endpoint, routed through KNIRVGATEWAY. This is also the moment to close the gap the KNIRVSERVER analysis surfaced independently: `/api/dve/*` currently has **no authentication middleware at all** — confirmed by direct inspection, anyone reaching KNIRVSERVER's port can POST a fabricated bundle to `/api/dve/:dve/sessions/ingest` today. The same lightweight token mechanism should cover that route too rather than leaving it as a second, unrelated gap next to the one this document is actively fixing.

### 3.5 Large Artifacts Don't Go Through Git At All

This is a simplification the retraction of the "same Forgejo, LFS" design (§2) actually enables: since the proof ledger's only content is proof bundles (JSON event logs, small manifests — not project source, not raw video), there is no reason to solve the "large binary in git" problem here. Memvid streams and screen recordings upload directly to whatever object/blob storage KNIRVSERVER's existing DVE evidence system already uses (referenced by the user's own phrase "linked to our validation chain hash and DVE file system" — this document assumes that store already exists server-side and does not propose a new one), addressed by the same `sha256:` hash already computed client-side for `ArtifactRef.Hash`. The evidence commit in the proof ledger carries only a small `artifact-manifest.json` pointer file per artifact (name, class, hash, size) — never the artifact bytes. This removes Git LFS from the picture entirely, not just "Forgejo's LFS."

### 3.6 Two Parallel Evidence Tracks (Resolved)

Earlier drafts of this document assumed the proof ledger's ingest step would call the same `dve.VerifyBundle`/`ReplayPolicy` logic the CLI already has. **That was wrong — direct inspection of KNIRVSERVER found `internal/dveevidence/` (`ingest.go`, `types.go`, `verify.go`, `merkle.go`, `sign.go`, `hashcrypto.go`, `keys.go`, `policy.go` — 1,127 lines), a separate, independently-maintained server-side reimplementation of the same evidence-bundle concepts**, not a shared package. It already backs `/api/dve/:dve/sessions/ingest` (registered on its own dedicated `gin.Engine`, `ingest.go:186-192`, dispatched by prefix match in `main.go:1401-1406`), and stores bundles as content-addressed JSON files on local disk (`FileStore`, `ingest.go:54-108`) — no database, session→bundle mappings in-memory only.

**Resolved: keep both, as two intentionally parallel, end-to-end tracks — not a shared ingest pipeline with one server-side backend serving two CLI producers.** This mirrors the CLI-side decision to keep `dve/` and `dvepod/` as parallel-not-duplicate, extended all the way through to the server:

| Track | CLI producer | Server consumer | Transport | Status |
|---|---|---|---|---|
| **Bundle Sign Path** (primary/canonical) | `dve/` — session start/status/commit, Ed25519-signed bundle, hash-chained event log | **Proof Ledger** (this document, §3.1-3.5) — new, git-native | `git push` to a bare repo, ingested via the hook-dispatch callback (§3.3) | New, being built by this plan |
| **Pod Evidence / Dock Path** (fallback/experimental) | `dvepod`/`internal/dvepod` — portable WASM/TEE-simulated sandbox, `knirv dve pod dock` | `internal/dveevidence` — existing `/api/dve/:dve/sessions/ingest` HTTP-upload API | `POST` over HTTPS through KNIRVGATEWAY | **Already exists and is already wired** — `dvePodDockCmd` → `mgr.Dock(...)` on the CLI side already targets this route conceptually; it just wasn't previously named as a deliberate pairing |

This is a clean split, not a compromise: a session's evidence goes down exactly one track depending on which CLI subsystem produced it, so there is no cross-track reconciliation to design — `internal/dveevidence` keeps serving `dvepod` exactly as it does today, untouched by this plan, while the proof ledger is new infrastructure dedicated entirely to `dve/`'s bundles. Neither is asked to become the other's fallback or to validate against the other.

**Also corrected:** `/api/anchoring/evidence/create`, which `entirely_new_cli.md` Phase A's dual-path-gap framing was written around, **does not exist anywhere server-side** — confirmed by repo-wide search, zero handlers. That gap is scoped entirely within the Bundle Sign Path now: once Phase A wires `hookguard`/`ptyproxy`'s real-time decisions into the active `dve.Workspace` session (as already planned), those decisions ride along naturally when that session's bundle is finalized and pushed to the proof ledger — there's no longer a second, separate "anchoring API" endpoint to build or reconcile against. The live-anchoring concern folds into the Bundle Sign Path's own event log rather than needing its own server-side counterpart.

## 4. Phased Build Order (Ledger-Specific)

Kept intentionally short — this is a narrower, self-contained subsystem compared to the full supervisor plan, and shouldn't be over-decomposed:

1. **Transport:** port `githttp.go`'s receive-pack handler (§3.2) — GET `info/refs?service=git-receive-pack` advertisement + POST `git-receive-pack` proxy, both shelling out to the real `git` binary.
2. **Bare-repo lifecycle:** `knirvserverd proof-ledger create <project_id>` — `git init --bare`, set `core.hooksPath`, register the project→repo-path mapping.
3. **Public auth:** the bearer-token path from §3.4, routed through KNIRVGATEWAY; use the same mechanism to close the existing `/api/dve/*` no-auth gap while touching this code.
4. **Hook dispatch + internal auth:** shared hooks dir, dispatcher script, `knirvserverd hook post-receive` subcommand, Unix-socket loopback callback with its own boot-minted token (§3.3) — and file the gateway-bypass backlog item (§3.3) explicitly rather than letting it go untracked.
5. **Ingest wiring:** build a new, dedicated verify/merkle/sign-check pipeline behind the loopback callback for the Bundle Sign Path (§3.6) — resolved to be independent of `internal/dveevidence`, not a call into it. This can share design (and, now that licensing permits it, potentially code shape) with the CLI's `dve/` package's own `VerifyBundle`/`ReplayPolicy` logic, since both sides of this specific track already agree on the `Bundle` shape.
6. **CLI-side:** point `knirv commit`'s push step (`entirely_new_cli.md` Phase H/I) at the proof-ledger endpoint; artifact manifest generation per §3.5.
7. **Query surface:** `knirv blame`/`knirv why` (`entirely_new_cli.md` Phase N) fetch from the proof ledger instead of a Forgejo evidence ref — mechanically the same change, different remote.

## 5. Decision Points — Status

Resolved this round:

1. ✅ **KNIRV's own license** — `KNIRV_NETWORK` is GPLv3, public release planned. Forgejo packages are directly cloneable (§1).
2. ✅ **go-git server-side maturity** — moot; Forgejo's own `githttp.go` shells out to real `git`, not go-git, and that's now the recommended approach (§3.2). No spike needed.
3. ✅ **KNIRVSERVER's existing evidence/object storage** — confirmed to exist: `internal/dveevidence` with a local-disk `FileStore` (§3.6). Not S3/minio-backed yet; `modules/storage`'s interface (§1) is now available to clone if that becomes necessary.
4. ✅ **Internal-API convention / gateway-bypass question** — resolved as a deliberate, tracked exception: Unix-socket loopback with its own token, matching the existing `AgentControlServer` precedent, filed as backlog debt rather than fixed now (§3.3). Broader "everything through the gateway" fix is deferred, scoped to cover this, `AgentControlServer`, and the future Supervisor↔Expert-Advisor channel together.
5. ✅ **Public-facing auth (adjacent question, resolved alongside #4)** — simple bearer tokens for now, covering both the proof-ledger push and the existing unauthenticated `/api/dve/*` gap; formal CLI login flow deferred until the product is closer to shipping (§3.4).

6. ✅ **Ingest target (§3.6)** — resolved: keep two fully parallel, end-to-end tracks. `dve/` ↔ Proof Ledger (Bundle Sign Path, new, this document's subject) and `dvepod` ↔ `internal/dveevidence` (Pod Evidence / Dock Path, existing, already wired via `knirv dve pod dock`, untouched). No cross-track reconciliation needed — each session's evidence has exactly one producer and one consumer.
7. **Reconciliation semantics** — no longer applicable in the form originally posed (there's nothing to reconcile between tracks now that they're cleanly separated). What remains: ordinary verification failure handling *within* the Bundle Sign Path (signature invalid, hash chain broken, Merkle root mismatch) — reject the push, matching how `dve.VerifyBundle` already treats these locally. Not yet decided: whether a partial/soft failure category exists within one track, or whether verification is simply binary pass/reject. Low-stakes enough to decide during implementation rather than needing sign-off now.

New from this round:

8. **Naming.** This document proposes "Bundle Sign Path" and "Pod Evidence / Dock Path" as working names for the two tracks (§3.6's table). Worth confirming these (or better names) before they propagate into code/docs, and worth carrying into `entirely_new_cli.md`'s eventual update alongside the `dve/`/`dvepod` doc-comment or rename work already noted there.

## 6. Risks

- **Reinventing a small git server is still reinventing a git server.** Even "slim," §3.2/§3.3 is real protocol-and-process-lifecycle work, not a config change. Budget for it accordingly rather than treating "no Forgejo" as "no work" — cloning Forgejo's code (§1) reduces this risk but doesn't eliminate it; it still needs to be adapted, tested, and operated.
- **One bare repo per project (§3.1) at scale** — fine for the near term; if KNIRV ends up with a very large number of low-activity projects, revisit whether per-project bare repos or a sharded multi-tenant layout is the better operational shape before it's load-bearing in production.
- **The gateway-bypass backlog item (§3.3) needs to actually get filed and tracked, not just live in this document.** Two instances of "internal endpoint bypasses the gateway" already exist (`AgentControlServer`, and now this one by design) before the Supervisor↔Expert-Advisor channel even exists; three would be a pattern worth fixing structurally rather than letting each new one restate the same justification.
- **Two fully parallel evidence tracks (§3.6) means two things to maintain, not one.** This was chosen deliberately over forcing a merge while `dvepod`'s long-term viability is still uncertain, but it's worth naming the cost plainly: verification logic, storage format decisions, and future feature work (redaction, retention policy, UI reporting) now potentially need to happen twice, once per track, unless it's actively kept in mind that they should stay in sync where their concerns actually overlap. Revisit whether one track should absorb the other once `dvepod`'s viability question is settled, rather than letting both drift indefinitely by default.
- **Naming needs to actually stick.** "Bundle Sign Path" / "Pod Evidence / Dock Path" (§3.6, decision point 8) are working names invented in this document — if they're good, get them into code comments and `entirely_new_cli.md` promptly; if they're not, replace them before they show up in a dozen places and become expensive to rename.
