# Entirely New CLI: Supervisor Agent + Proof-Ledger Evidence Commits

## 0. Purpose

This document merges four inputs into one build plan:

1. **`samples/ralph-workflow`** — an autonomous coding-agent orchestration tool ("Ralph"). We clone its **supervisor** pattern: a declarative phase graph, an evidence-not-self-report verification philosophy, and a structured tool surface the wrapped agent reports through.
2. **`samples/entire-cli`** — a Go CLI ("Entire") that wraps `git commit` behavior for AI-agent-authored work. We clone its **commit/attribution/checkpoint** pattern: git hooks instead of a commit wrapper, trailers as the linkage mechanism, a sharded checkpoint branch, and line-level provenance.
3. **`DVE_Alignment_Plan.md`** — KNIRV's own target architecture for the Deterministic Validation Environment. This plan is the reconciliation layer: it explains where Ralph's and Entire's patterns slot into DVE's phases.
4. **`KNIRVSERVER_Commit_Strategy.md`** — the resolved, authoritative design for the server side of this whole plan. **Supersedes this document's original framing of a KNIRVSERVER-hosted private Forgejo instance.** There is no private Forgejo deployment. The user's real source repository stays wherever it already lives (GitHub, GitLab, self-hosted, anywhere) — KNIRV never becomes a source host. KNIRVSERVER instead runs a slim, purpose-built **Proof Ledger**: bare git repositories whose only content is signed DVE evidence bundles, built by cloning specific Forgejo Go packages directly (now legally straightforward — `KNIRV_NETWORK`, which contains KNIRVSERVER, is GPLv3, confirmed and public-release-bound) rather than deploying Forgejo itself. Every mention of "KNIRVSERVER's Forgejo" or "the same Forgejo instance" in the sections below that predate this resolution has been corrected in place; where a design decision now lives entirely in `KNIRVSERVER_Commit_Strategy.md`, this document points there rather than duplicating it.

Everything below assumes familiarity with `DVE_Alignment_Plan.md` and `KNIRVSERVER_Commit_Strategy.md`; neither is repeated here except where this plan changes or supersedes them.

## 1. Baseline Audit: What KNIRV CLI Already Has

This matters because a naive read of `DVE_Alignment_Plan.md`'s phases would have you re-build things that already exist. Confirmed by reading the actual source:

| Area | File(s) | State |
|---|---|---|
| Evidence bundle schema | `dve/types.go` | **Done.** `Bundle`, `Event`, `PermissionDecision`, `ArtifactRef`, `MemvidRef` match the plan's JSON shape exactly, including the `memvid_refs` field (Phase 5 stub already present). |
| Local session lifecycle | `dve/workspace.go`, `cmd/dve.go` | **Done.** `knirv dve session start/status/commit`, `.knirv/sessions/<id>/` layout, workspace file-hash manifest, append-only hash-chained event log. This is Phase 0/1 of the alignment plan, essentially complete. |
| Hash chain / Merkle / signing | `dve/hashchain.go`, `dve/merkle.go`, `dve/sign.go`, `dve/keys.go` | **Done.** Ed25519 local signing (`KeyStore`), artifact Merkle root, event hash chain. Phase 6 is implemented. |
| Bundle verification & policy replay | `dve/verify.go`, `dve/policy.go` | **Done, locally.** `VerifyBundle`, `ReplayPolicy` — this is the Phase 7 verification *logic*, but it currently only runs where you point `knirv dve verify` at a bundle file. It is not wired to any server ingest trigger yet. |
| Local policy rules | `dve/policy.go` | **Partial.** `Policy` struct has `AllowedCommandPrefixes`/`DeniedCommandPrefixes`/roots/network/secrets exactly as Phase 2 describes, and `ReplayPolicy` can check decisions against it — but nothing in the CLI evaluates *live* actions against this local policy. Live evaluation is delegated entirely to the server (next row). |
| Live permission enforcement, Layer 1 (hook API) | `cmd/hook.go`, `internal/hookguard/`, `internal/policyguard/` | **Done, for Claude Code only.** `knirv hook claude-code` answers Claude Code's native `PreToolUse`/`PostToolUse` hooks by calling `POST /api/guardrail/policies/{id}/evaluate` on KNIRVSERVER and anchoring evidence via `POST /api/anchoring/evidence/create`. No Codex/Hermes/OpenCode hook adapters exist — only `ClaudeCodeAdapter`. |
| Live permission enforcement, Layer 2 (PTY fallback) | `cmd/pty.go`, `internal/ptyproxy/` | **Done, prompt-pattern only.** `knirv pty run --tool claude\|codex\|opencode` recognizes confirmation-prompt text in the child's PTY output and answers it, optionally via the same server policy evaluation. This is regex-level prompt detection, not a structured tool-call log. |
| **Gap** — evidence path is split in two | — | `hookguard`/`ptyproxy` post evidence straight to `POST /api/anchoring/evidence/create` on KNIRVSERVER. They **never call into `dve.Workspace`**. Meanwhile `dve.Workspace.RecordDecision`/`RecordEvent` build the local signed bundle. **Confirmed by direct inspection of KNIRVSERVER: `/api/anchoring/evidence/create` does not exist server-side at all — zero handlers, repo-wide.** Every `knirv hook`/`knirv pty` anchor call today hits a route that isn't implemented; it isn't a working parallel channel that merely goes unreconciled, it's a call to nothing. A session run through `knirv hook`/`knirv pty` today produces **no evidence anywhere**, and a session run through `knirv dve session` produces a local bundle with **no permission decisions in it**, because nothing feeds it any. These two systems must be unified (Phase A below), and per `KNIRVSERVER_Commit_Strategy.md` §3.6, the resolution is to fold the real-time anchoring concern *into* `dve.Workspace`'s own event log rather than build the missing anchoring endpoint at all — see Phase A. |
| Generic agent process supervision | `cmd/run.go`, `cmd/register.go`, `internal/runner/`, `internal/registry/`, `internal/watchtower/`, `internal/incident/` | **Exists, but it's a different concept.** This runs arbitrary long-lived registered processes (local/docker/ssh) under log-signature-based incident detection, writing incidents to an Obsidian vault. It is not a coding-agent loop, has no phase graph, no checkpointing, no artifact submission surface. Reusable primitives: `RingBuffer`, `SignatureSet`, the stdout/stderr scanning loop — but it is not the supervisor this plan builds. |
| Portable sandboxed DVE — **the "Pod Evidence / Dock Path"** | `cmd/dve_pod.go`, `internal/dvepod/`, `dvepod/` | **Resolved: in scope, as a deliberately parallel track, not orthogonal.** Self-contained WASM environment (embedded KNIRVAGENT, TEE simulation, BusyBox) that bundles to a single HTML file and "docks" to KNIRVSERVER via `knirv dve pod dock`. Per `KNIRVSERVER_Commit_Strategy.md` §3.6: this is **not** a mistake or accidental duplication of `dve/` — `dvepod`'s long-term viability is genuinely uncertain, so both are kept, `dve/` as the primary "Bundle Sign Path" and `dvepod` as the fallback/experimental "Pod Evidence / Dock Path." The two-track pairing already half-exists in code: `dvePodDockCmd` → `mgr.Dock(...)` already targets KNIRVSERVER's existing `internal/dveevidence`-backed `/api/dve/:dve/sessions/ingest` HTTP-upload route — it just wasn't previously documented as a deliberate architectural pairing. **Not touched by this plan** — `dvepod`'s dock flow and `internal/dveevidence` continue exactly as they are; only naming/doc-comment clarity is owed here (see the new note at the end of this section). |
| Browser DVE identity | `cmd/browser_dve.go` | **Exists, orthogonal.** Registers a wallet-backed browser-extension DVE node with capabilities/badge NFTs. A separate DVE *identity* mechanism from the CLI session/commit path this plan builds. |
| Git / proof-ledger integration | — | **Does not exist.** No `go-git` dependency, no `exec.Command("git", ...)`, no Forgejo/Gitea client anywhere in the module. This is greenfield — and per `KNIRVSERVER_Commit_Strategy.md`, it targets a slim KNIRVSERVER-hosted Proof Ledger, not a Forgejo deployment (see §4, revised). |
| "MCP" naming collision | `cmd/mcp.go`, `cmd/mcp/*.go` | **Important disambiguation.** KNIRV's existing `knirv mcp` is *Multi-Capability Protocol* — its own capability/procedure/NRV system for blockchain agents. It is **not** the Anthropic Model Context Protocol that Claude Code, Codex, etc. use for tool-calling. This plan needs a *new*, differently-named surface for agent tool-calling (see Phase D) — do not attempt to overload `knirv mcp` for this. |

**Note on naming (carried over from `KNIRVSERVER_Commit_Strategy.md` §3.6, decision point 8, still open):** that document proposes "Bundle Sign Path" (`dve/` ↔ KNIRVSERVER's new Proof Ledger) and "Pod Evidence / Dock Path" (`dvepod` ↔ existing `internal/dveevidence`) as working names for the two parallel evidence tracks. Confirm before they propagate further; if kept, both `dve/` and `dvepod`/`internal/dvepod` should gain package-level doc comments (and possibly light renames) stating explicitly which track each belongs to and that neither supersedes the other.

**Conclusion of the audit:** KNIRV is much further along than a plan written purely from `DVE_Alignment_Plan.md` would suggest. The real work is (a) unifying the two evidence paths within the Bundle Sign Path specifically (b) building the actual supervisor *loop* (nothing today loops — every current command is single-shot), (c) building git integration against a KNIRVSERVER-hosted Proof Ledger from zero, and (d) reconciling DVE's "server ingest" concept with "evidence lives in Proof Ledger commits," while leaving the separate, already-working Pod Evidence / Dock Path untouched.

## 2. What We're Cloning From `ralph-workflow`, and What We're Not

Ralph is a declarative finite-state-machine pipeline (`ralph/pipeline/reducer.py`), not a `while(!done)` loop. Phases are data (`ralph/policy/defaults/pipeline.toml`), compiled into a graph, advanced by a reducer that turns `state → Effect → Event → new state`. The mechanisms worth cloning, mapped onto KNIRV:

| Ralph mechanism | Clone into KNIRV as |
|---|---|
| Declarative phase graph + loop/budget counters (`iteration`, `development_analysis_iteration`, `recovery_cycle_cap`) | A `supervisor.Policy` (new package, sibling to `dve.Policy`) compiled from TOML/JSON, driving a phase-graph reducer in a new `internal/supervisor` package. |
| "Done" is evidence-based, never agent self-report or exit code (`subprocess_executor.go:56-60` equivalent) | KNIRV already believes this — `dve.PermissionDecision`/`ArtifactRef` are the evidence primitives. The supervisor's phase-advancement gate must check *artifacts recorded in the active `dve.Workspace`*, never a subprocess exit code. |
| MCP (Model Context Protocol) tool surface as the agent's report channel, not stdout scraping | New evidence tool server (Phase D) — the wrapped agent calls `submit_artifact`/`submit_plan`/`exec` tools instead of KNIRV parsing free text. |
| `BuiltinAgentSpec` triad: transport (PTY/subprocess) + parser + execution-strategy, one per agent | Extend `internal/ptyproxy` adapters from "regex prompt matcher" into a full triad, add Hermes, add a headless-subprocess transport alongside PTY (Phase C). |
| Agent fallback chains per phase (`development = ["claude", "opencode"]`, bounded retries) | `supervisor.Policy.AgentChains[phase] = []ToolName` with per-chain retry/backoff — direct port. |
| `WorkflowInstanceTracker`/`WorkflowInstanceView` — immutable external supervision view (`instance_id`, `lifecycle_status`, `current_stage`, `recent_activity`) | New `knirv supervisor status` command backed by a view type built the same way — this is the single most directly portable piece. |
| Idle/stall watchdog, two-state invariant (backoff-to-next-agent XOR same-agent retry), "channel freshness gate" | `internal/watchtower` already has the raw ingredients (`RingBuffer`, signature matching, incident cooldown); add a `StuckClassifier` and the two-state invariant on top rather than rebuilding from scratch. |
| Anti-fabrication artifact-proof binding (`plan_items_proven`/`analysis_items_addressed` must reference canonical prior IDs) | Extend `dve.PermissionDecision`/new `dve.ArtifactClaim` validation: a `development_result`-class artifact must cite artifact/decision IDs already present in the session's event log, checked at submission time (Phase E). |
| Capability-gated exec tool, static local deny-list (`sudo`, `dd`, `nc`, `docker`, VCS commands under the unsafe variant) as defense-in-depth *before* the network call to the policy engine | Add a static local deny-list check in front of the existing `policyguard.Evaluate` call — cheap, offline, catches the obvious cases even if KNIRVSERVER is unreachable and `--fail-open` was (mis)configured. |
| Checkpoint file (`checkpoint.json`) with phase/loop/budget/recovery state, resumable via `--resume` | Extend `dve.Workspace`'s `session.json` with the same shape (phase, loop_iterations, budget_caps, recovery_cycle_count, last_agent_session_id) — KNIRV already has the file-backed session directory Ralph would need to build from scratch. |
| Sanctioned-commit-only rule (`AGENTS.md`: never run raw `git commit`; only `ralph --generate-commit`) | Directly informs Phase H: `knirv commit` becomes the only sanctioned path once a supervised session is active. |

**What we are explicitly not cloning:**
- Ralph's Python/Typer stack — irrelevant, KNIRV is Go.
- Ralph's Docker packaging of *itself* — it only isolates the `ralph` process, not the wrapped agent (their own admitted gap). Not worth cloning as-is; if KNIRV wants real sandboxing, `dve_pod`'s WASM/TEE approach is already a stronger primitive than what Ralph does.
- The `.claude/skills`-style engineering-discipline skill bundles Ralph installs into target projects — orthogonal to supervision, a documentation/product decision, not architecture.

## 3. What We're Cloning From `entire-cli`, and What We're Not

The critical discovery: **Entire does not wrap `git commit`.** It hooks git's own lifecycle (`prepare-commit-msg`, `post-commit`, `pre-push`, `post-rewrite`) and uses **trailers**, not a wrapper command, as the linkage mechanism between a normal commit and its evidence.

| Entire mechanism | Clone into KNIRV as |
|---|---|
| `prepare-commit-msg` hook appends a trailer (`Entire-Checkpoint: <id>`) to the user's real commit message — the only modification to the commit itself | `prepare-commit-msg` hook appends `KNIRV-DVE-Session: <session_id>` and `KNIRV-Evidence-Ref: sha256:<bundle_hash>` trailers. Standard git trailer format, survives `amend`/rebase per Entire's own finding. |
| Ephemeral shadow branch per session (`entire/<head-hash>-<worktree-hash>`), in-memory via go-git, holds intra-session checkpoint commits | KNIRV's `.knirv/sessions/<id>/` directory is the pre-git-native equivalent of this. We do **not** need a shadow branch for the *local* working state — the existing directory-based session is fine and simpler. We adopt the shadow-branch idea only at the point evidence needs to leave the machine (see next row). |
| `post-commit` condenses session data into a permanent, sharded commit on `entire/checkpoints/v1` (`<id[:2]>/<id[2:]>/metadata.json`, `full.jsonl`, `transcript.jsonl`) | On `knirv dve session commit`, write the finalized bundle + redacted event log + artifacts as a commit on a dedicated evidence ref: `refs/knirv/dve/<session_id>`, sharded the same way — but pushed to KNIRVSERVER's **Proof Ledger** (a bare repo KNIRV controls, per `KNIRVSERVER_Commit_Strategy.md`), not to a ref inside the same repo as the source code (see Phase I, revised). |
| Line-level attribution (`Attribution{AgentLines, HumanAdded, AgentPercentage}`) computed by diffing the pre-agent-edit tree against the committed tree | Directly portable: KNIRV already has `HashWorkspace`'s file manifest before/after; add a real tree diff (Phase H) to compute the same stats and store them in the bundle. |
| `entire blame <file>` / `entire why <file>:<line>` — join `git blame --line-porcelain` output to checkpoint trailers to checkpoint metadata | `knirv blame <file>` / `knirv why <file>:<line>` (Phase N) — same join, fetching from KNIRVSERVER's Proof Ledger instead of Entire's checkpoint branch. |
| Checkpoint remote — evidence branch can push to a **separate repo** from the code (`checkpoint_remote: {provider, repo}`, an *optional* config knob in Entire) | **Not an optional knob for KNIRV — it's the default and only shape.** There is no KNIRVSERVER-hosted source repo to co-locate evidence with; the Proof Ledger is, by construction, always a separate system from wherever the user's real repo lives. Entire's "same repo, different ref" default doesn't apply here at all — see §4, revised, and Phase I. |
| Best-effort git-native commit signing (GPG/SSH, reuses the user's existing `commit.gpgsign` config), **never blocks the commit on failure** | Adopt as a *secondary*, optional signature layer (Phase K). KNIRV's Ed25519 `dve.Signer` bundle signature remains the primary, required proof — git signing is belt-and-suspenders for reviewers who verify commits with plain `git log --show-signature` and don't have KNIRV's verifier. |
| Fail-**closed** pre-push redaction gate (OPF): divergence, ref-move races, or redaction-model failure abort the push entirely rather than shipping under-redacted content | This is the pattern for DVE Plan Phase 4/10's redaction requirement — adopt verbatim as Phase J. This is the one place Entire is *stricter* than best-effort, and it should be, because it's the last gate before secrets leave the machine. |
| `git-remote-entire` — a full git-remote-helper implementing a custom `entire://` transport with pkt-line agent-string injection, RFC 8693 token exchange, and jurisdiction/replica routing | **Still not cloning, for the same reason, now doubly true.** KNIRVSERVER's Proof Ledger is reached over standard `https://` smart-HTTP git protocol — the server side is built by porting Forgejo's `githttp.go` receive-pack handler directly (now legally straightforward under GPLv3, see `KNIRVSERVER_Commit_Strategy.md` §1/§3.2), which itself just shells out to real `git`, not a custom wire protocol. A remote-helper is warranted for non-standard transports; KNIRV needs none. Building and fuzz-testing a pkt-line parser from scratch (Entire's own test plan calls the divergence/recovery matrix "the most re-broken area") remains a large, ongoing maintenance cost for zero benefit here. |
| MCP stdio server (`entire mcp`) exposing `agent_help`/`entire_status` tools | Loosely informs Phase D's evidence tool server, but Entire's is informational (help/status), not an evidence-submission surface like Ralph's. Ralph's `ralph-mcp` is the closer model for what KNIRV needs. |

## 4. Reconciling With `DVE_Alignment_Plan.md`: Evidence Proof Packages on the Proof Ledger (Revised)

**This section originally assumed KNIRVSERVER runs a private Forgejo instance hosting both project source and evidence refs. That assumption is retracted — see `KNIRVSERVER_Commit_Strategy.md` for the full reasoning (no Forgejo deployment; the user's real source repo stays wherever it already lives; KNIRVSERVER instead runs a slim, purpose-built git evidence store, the "Proof Ledger," built by cloning specific Forgejo Go packages under GPLv3 rather than deploying Forgejo).** The redesign below reflects the resolved architecture; everything server-side is specified authoritatively in that document, referenced here rather than duplicated.

`DVE_Alignment_Plan.md` Phase 7 currently specifies:

```
POST /api/dve/{dve_id}/sessions/ingest
```

as a raw multipart/HTTP upload of the evidence bundle to KNIRVSERVER. (Confirmed by direct inspection: this route doesn't exist under that exact name, but a near-equivalent, `internal/dveevidence`'s `/api/dve/:dve/sessions/ingest`, does — see the Pod Evidence / Dock Path note in §1. That existing route stays exactly as-is and is **not** what this section redesigns; it continues serving `dvepod`, untouched.) The redesign here is for a *second*, new, git-native path — the Bundle Sign Path — dedicated to `dve/`'s bundles:

- **Local, unchanged:** `.knirv/sessions/<id>/` remains the working area exactly as today (`dve.Workspace`).
- **The user's real repo, untouched beyond a trailer:** `knirv supervisor init` installs a client-side `prepare-commit-msg` hook (Phase H) that appends `KNIRV-DVE-Session`/`KNIRV-Evidence-Ref` trailers to the user's normal commit message, wherever that repo is hosted. KNIRV never pushes anything else there and never needs write access to arbitrary refs on a third-party host.
- **New, at commit time:** `knirv dve session commit` (or its new alias `knirv commit`, see Phase H) does two things instead of one:
  1. What it does today: finalize, sign (Ed25519), write `bundle.json` locally.
  2. **New:** write the bundle + redacted event log + a small artifact manifest (content hashes only, never the artifact bytes — see Phase L, revised) as a git commit on an evidence ref (`refs/knirv/dve/<session_id>`, sharded the way Entire shards checkpoint IDs), and `git push` it to KNIRVSERVER's Proof Ledger — a bare repo KNIRV itself hosts and controls, one per project, entirely separate infrastructure from wherever the source repo lives (`KNIRVSERVER_Commit_Strategy.md` §3.1). Authenticated via a KNIRV-issued bearer token through a git credential helper (Phase I, revised), routed through KNIRVGATEWAY like every other public-facing endpoint.
- **Server ingest, redesigned:** not a webhook (there's no Forgejo webhook system) and not a poller. KNIRVSERVER's Proof Ledger uses the **hook-dispatch pattern cloned from Forgejo**: a shared `core.hooksPath` directory, a thin dispatcher script that re-execs the KNIRVSERVER binary as `knirvserverd hook post-receive`, which reads git's hook stdin protocol and makes one authenticated call back into the already-running KNIRVSERVER process — over a Unix domain socket with its own boot-minted token, deliberately *not* routed through KNIRVGATEWAY (a tracked, backlogged exception to "everything goes through the gateway," not a silent one — full reasoning in `KNIRVSERVER_Commit_Strategy.md` §3.3). That callback fetches the just-pushed bundle locally (`git show`/`git cat-file`, no network fetch needed) and hands it to a **new, dedicated verify/merkle/sign-check pipeline** built specifically for the Bundle Sign Path — **not** a direct reuse of the CLI's `dve.VerifyBundle`, and **not** `internal/dveevidence` either (confirmed to be a separate, independent server-side reimplementation already serving the Pod Evidence / Dock Path — see §1). The two tracks share the `Bundle` JSON shape as their wire contract but are otherwise independent implementations by design.
- **What this buys, that a raw-upload design wouldn't:** an append-only, git-native audit log of every evidence submission (`git log` over the evidence ref, free); standard tooling for diffing evidence bundles across sessions; the code commit and its evidence are cryptographically linked via the trailer, and the evidence side lives in a system KNIRV fully controls rather than depending on a third-party host's webhook/ref-permission model.
- **Large artifacts no longer route through git at all.** Memvid streams, screen recordings, etc. upload directly to KNIRVSERVER's existing evidence/object storage (`internal/dveevidence`'s `FileStore`, local-disk today — `KNIRVSERVER_Commit_Strategy.md` §3.5) by content hash, matching `ArtifactRef.Hash` unchanged. The evidence commit in the Proof Ledger carries only a small pointer manifest. Git LFS is dropped from the plan entirely — see Phase L, revised.
- **Validation status model** from the alignment plan (`unsupervised → recorded → signed → policy_verified → source_verified → certified`, or `rejected`/`quarantined`) is unchanged; the trigger for entering `recorded` is now "evidence ref pushed to the Proof Ledger and picked up by the hook-dispatch callback."

This reconciliation does not touch Phases 0, 1, 2 (local), 5 (memvid), 6 (signing) of the alignment plan — it specifically redefines Phase 7 (ingest transport, now a KNIRV-controlled git push + hook callback rather than an HTTP upload or a Forgejo webhook) and simplifies part of Phase 8 (commit — "compare workspace_base_hash with current server DVE state" now means comparing against the Proof Ledger's own ref history, which KNIRVSERVER already owns).

## 5. Target Architecture (Revised — Proof Ledger, Not Private Forgejo)

```text
User's real source repo (GitHub / GitLab / self-hosted / anywhere — KNIRV
never hosts this, never needs write access beyond the user's own commit)
          ^
          | client-side prepare-commit-msg hook, installed by
          | `knirv supervisor init`: appends KNIRV-DVE-Session /
          | KNIRV-Evidence-Ref trailers to the user's normal commit
          |
User Host
  Claude / Codex / Hermes / OpenCode  (wrapped agents, unmodified)
          |  (PTY transport, or hook API where available, or headless subprocess)
          v
  KNIRV CLI Supervisor  (new: internal/supervisor)
    - compiles a declarative phase graph from supervisor.Policy
    - drives phases: plan -> plan_review -> develop -> commit_cleanup ->
      commit -> develop_review -> [loop or complete], budget + recovery capped
    - agent adapter triad per tool (transport/parser/strategy)
    - agent fallback chains per phase, bounded retries
    - idle/stall watchdog (two-state invariant)
    - evidence tool server (new protocol surface, NOT `knirv mcp`)
      -> every submit_artifact / exec / git_read call is a dve.Workspace event
    - static local exec deny-list, THEN policyguard.Evaluate (server)
    - anti-fabrication artifact-proof cross-referencing
          |
          v
  dve.Workspace  (existing — unified evidence sink, see Phase A; this IS
    the Bundle Sign Path's local half — hookguard/ptyproxy decisions land
    here directly, no separate anchoring API involved, see 1/4)
    - event log (hash-chained), permission decisions, artifacts, tool runs
    - checkpoint state (phase/loop/budget/recovery — extended, see Phase G)
          |
          v  knirv commit  (new, Phase H)
    - appends KNIRV-DVE-Session trailer to the user's real repo's commit
    - finalize + Ed25519-sign the bundle (existing dve.Signer)
    - fail-closed redaction gate (Phase J) before anything leaves the machine
    - optional git-native commit signing (Phase K, belt-and-suspenders)
    - write evidence commit to refs/knirv/dve/<session_id>
    - large artifacts (memvid etc.) uploaded separately, by hash, to
      KNIRVSERVER's existing evidence/object storage — NOT through git
      (Phase L, revised)
    - `git push` the evidence ref to KNIRVSERVER's Proof Ledger via a
      KNIRV-issued bearer token / git credential helper (Phase I, revised),
      routed through KNIRVGATEWAY
          |
          v
KNIRVSERVER Proof Ledger  (new, slim — full design in
  KNIRVSERVER_Commit_Strategy.md, not duplicated here)
  - one bare repo per project, hosts ONLY evidence refs, never source
  - receives the push over standard git smart-HTTP (ported from Forgejo's
    githttp.go, now legally cloneable — GPLv3, see KNIRVSERVER_Commit_
    Strategy.md §1/§3.2)
  - shared hooks/ dir + core.hooksPath dispatcher re-execs
    `knirvserverd hook post-receive`, which makes one authenticated
    Unix-socket loopback call into the already-running KNIRVSERVER
    process (deliberately, trackedly bypassing KNIRVGATEWAY for this one
    same-host callback — see KNIRVSERVER_Commit_Strategy.md §3.3)
          |
          v
KNIRVSERVER Bundle Sign Path ingest  (new, dedicated pipeline — distinct
  from both the CLI's dve/ package AND from internal/dveevidence, which
  remains the separate Pod Evidence / Dock Path serving dvepod, untouched)
  - verifies signature, hash chain, Merkle root, policy replay
  - links this evidence commit's git hash into KNIRV's validation chain
  - emits validation certificate, updates validation status
  - (Phase 9 of the alignment plan, UI/reporting — unchanged, out of CLI scope)
```

## 6. Phased Implementation Plan

Phases are ordered by dependency, not by priority alone — several early phases are prerequisites the rest of the plan silently assumes.

### Phase A — Unify the Evidence Path (prerequisite for everything else)

**Goal:** every decision recorded through `knirv hook` or `knirv pty` also lands in the active `dve.Workspace` session, so a supervised session run today through either enforcement layer produces one coherent, signable bundle instead of two disconnected evidence trails.

Steps:
1. Give `hookguard.RunnerOptions` and `ptyproxy.Options` an optional `Workspace *dve.Workspace` (or a narrow interface `EvidenceSink{RecordDecision, RecordEvent}` so `dve` doesn't become an import-cycle risk for those packages).
2. In `hookguard.Run`, after evaluating, call `sink.RecordDecision(...)` with the same `action_type`/`context`/`allowed`/`reason` fields, mapped onto `dve.PermissionDecision` — this becomes the **sole** evidence-recording step, not one of two.
3. In `ptyproxy`, do the same at the point a recognized prompt is answered.
4. `cmd/hook.go` / `cmd/pty.go` gain a `--session` (or auto-discover via `dve.LatestSession(".")`) so they attach to the currently active session.
5. **Revised: `AnchorEvidenceAsync`'s call to `POST /api/anchoring/evidence/create` is not a real parallel channel to preserve — that route does not exist server-side (confirmed by direct inspection of KNIRVSERVER, zero handlers repo-wide).** Per `KNIRVSERVER_Commit_Strategy.md` §3.6's resolution, this concern is dropped, not deferred: don't build the missing endpoint, and don't design around `AnchorEvidenceAsync` as if it does anything today. Either remove the call site or leave it behind a feature flag defaulted off with a comment explaining why, so it doesn't silently look like working telemetry. The Bundle Sign Path's own event log (step 2/3 above, flowing to the Proof Ledger at commit time) is now the entire real-time-and-final evidence story for `knirv hook`/`knirv pty` sessions.

Deliverables: `EvidenceSink` interface, wiring in both enforcement layers, updated tests, one integration test proving a `knirv hook claude-code` run followed by `knirv dve session commit` produces a bundle containing that decision, and removal (or explicit disabling) of the dead `AnchorEvidenceAsync` call path.

### Phase B — Supervisor Policy & Phase Graph

**Goal:** a declarative, data-driven phase graph replacing the implicit "run once" behavior of every current command.

Steps:
1. New `internal/supervisor/policy` package: `Policy` struct with `Phases []Phase`, each `Phase{Name, Kind (plan|develop|review|commit|terminal), Verification GateSpec, OnSuccess/OnFailure/OnRequestChanges Route, RetryPolicy}`.
2. `LoopCounter`/`BudgetCounter` types mirroring Ralph's (`max`, `tracks_budget`), with `--counter NAME=VALUE` CLI override validated against policy-declared counters only.
3. Default policy shipped as embedded JSON with the same phase shape as Ralph's default: plan → plan_review → develop → commit_cleanup → commit → develop_review → loop-or-complete.
4. `internal/supervisor/reducer.go`: `Step(state, event) (newState, effect)` pure function, unit-testable without any process spawning.
5. Absolute `recovery_cycle_cap` independent of the budget counter — a circuit breaker even if budget math is wrong.
6. `knirv supervisor run <phase-graph-flags>` command (new file `cmd/supervisor.go`), analogous to `ralph`'s bare invocation.
7. `knirv supervisor status` — the `WorkflowInstanceView` port: `{instance_id, session_id, lifecycle_status, current_phase, recent_activity[5], budget_remaining}`.

Deliverables: `internal/supervisor/{policy,reducer,status}.go`, `cmd/supervisor.go`, default policy file, unit tests on the reducer covering loopback, budget exhaustion, and recovery-cap trip.

### Phase C — Agent Adapter Triad & Fallback Chains

**Goal:** extend today's regex-prompt-matcher adapters into full transport/parser/strategy triads, and add Hermes.

Steps:
1. `internal/supervisor/agents/spec.go`: `AgentSpec{Name, Transport (PTY|Subprocess), Parser, ExecutionStrategy, YoloFlag, SessionResumeFlag, CanCommit bool}`.
2. Port existing `ptyproxy` claude/codex/opencode adapters into this shape; add `hermes`.
3. Add a headless-subprocess transport (reuse `internal/runner/local.go`'s process-spawn primitives) for agents/modes that don't need PTY.
4. `AgentChain` per phase: ordered fallback list + bounded retries + backoff, config-driven (`Policy.AgentChains[phase] = []string`).
5. `knirv supervisor list-agents` — enumerate configured specs and their resolved binary paths (mirrors Ralph's `--list-agents`/`--diagnose`).

Deliverables: `internal/supervisor/agents/{spec,claude,codex,opencode,hermes,strategy}.go`, chain-exhaustion test, `knirv supervisor list-agents`.

### Phase D — Structured Evidence Tool Surface

**Goal:** the wrapped agent reports work through structured tool calls, not stdout scraping — the single highest-leverage anti-fabrication mechanism in Ralph's design.

Steps:
1. Stand up a real Model Context Protocol (MCP, the Anthropic tool-calling standard) stdio/HTTP server — **explicitly namespaced away from `knirv mcp`** (which is KNIRV's own unrelated Multi-Capability Protocol). Suggested command: `knirv supervisor evidence-server` (internal, spawned by `knirv supervisor run`, not meant for direct user invocation).
2. Tools exposed: `submit_artifact` (writes a `dve.ArtifactRef` + finalizes the underlying `dve.Event`), `submit_plan`, `exec` (bounded, capability-gated — see Phase E's deny-list), `git_read` (read-only git plumbing, explicitly excludes `commit`/`push`).
3. Every tool call is, at minimum, a `dve.Event`; artifact-class submissions also become `dve.ArtifactRef` entries.
4. Bind to `127.0.0.1` only, constant-time bearer-token auth (`hmac.compare_digest`-equivalent in Go, `crypto/subtle.ConstantTimeCompare`) — directly port Ralph's `_trust_boundary.py` pattern.
5. Wire each `AgentSpec` (Phase C) with this server's address/token via env var or config file, the same way Claude Code/Codex are told about their native tool servers today.

Deliverables: `internal/supervisor/evidence_server/{server,tools}.go`, auth/trust-boundary tests, one adapter (Claude Code, since it already has hook-based tool visibility) wired end-to-end.

### Phase E — Anti-Fabrication / Artifact-Proof Binding

**Goal:** a `development_result`-class artifact can't claim work it can't back up with references to real, prior artifacts/decisions in the same session.

Steps:
1. Extend `dve.ArtifactRef` (or add a parallel `dve.Claim` type) with optional `ProvesArtifactIDs []string`/`ProvesDecisionIDs []string`.
2. Validation at `submit_artifact` time (Phase D's tool): every referenced ID must already exist in `Workspace.builder.session` — reject the submission (return a tool error to the agent, not a silent drop) otherwise.
3. Static local exec deny-list (`sudo`, `su`, `doas`, `shutdown`, `nc`/`ncat`/`socat`, `docker`/`podman`/`chroot`, and — critically — `git commit`/`git push` so the wrapped agent cannot route around Phase H's sanctioned commit path) checked in the `exec` tool *before* the network call to `policyguard.Evaluate`, matching Ralph's defense-in-depth ordering.

Deliverables: claim-reference validation in the evidence server, deny-list constant + tests, integration test proving a fabricated `plan_items_proven` reference is rejected.

### Phase F — Idle/Stall Watchdog & Recovery Controller

**Goal:** bounded, auditable handling of a stuck or silent agent — no silent hangs, no infinite loops.

Steps:
1. `internal/supervisor/watchdog.go`: reuse `internal/watchtower.RingBuffer`/`SignatureSet` as the "is there fresh activity" signal.
2. Two-state invariant: every watchdog fire is either (a) exponential-backoff-to-next-agent-in-chain, or (b) same-agent retry — never a third state, never a silent `os.Exit`.
3. `SESSION_CEILING_EXCEEDED` — an absolute, operator-configured wall-clock cap that bypasses the stuck-classifier gate entirely (a hard stop distinct from soft stall detection).
4. `RecoveryController` enforcing the phase graph's `recovery_cycle_cap` (Phase B) independent of budget-counter math.

Deliverables: `internal/supervisor/watchdog.go`, `internal/supervisor/recovery.go`, tests for both invariant states and the ceiling escape hatch.

### Phase G — Checkpoint & Resume

**Goal:** extend the existing `dve.Workspace` session file with the supervisor's loop/phase/recovery state, and make it resumable.

Steps:
1. Extend `session.json`'s schema (additive, versioned) with `phase`, `previous_phase`, `loop_iterations`, `budget_caps`, `recovery_cycle_count`, `interrupted_by_user`, `last_agent_session_id` (for PTY session resume where the underlying agent supports it).
2. Save on every reducer step (Phase B) and specifically on `SIGINT`/`SIGTERM` before teardown (KNIRV's `runAgent` in `cmd/run.go` already has the signal-handling skeleton to extend).
3. `knirv supervisor run --resume` / `--no-resume` / `--inspect-checkpoint` flags, mirroring Ralph's.

Deliverables: schema migration for `session.json`, resume path tests, `--inspect-checkpoint` output.

### Phase H — Sanctioned Commit Path (`knirv commit`)

**Goal:** the only sanctioned way to turn supervised work into a git commit, producing both the code commit and its evidence linkage.

Steps:
1. New `cmd/commit.go`: `knirv commit [-m msg]` — requires an active `dve.Workspace` session (mirrors Ralph's `AGENTS.md` ban on raw `git commit` during a supervised session; enforce this by having Phase E's exec deny-list block `git commit`/`git push` for the *agent*, while `knirv commit` is a human/orchestrator-invoked command, not an agent tool).
2. Compute the real code diff, then compute line-level attribution (agent vs. human) by diffing the pre-session workspace manifest (`dve.HashWorkspace`'s baseline, already computed at session start) against the current tree — port Entire's `manual_commit_attribution.go` diff logic.
3. Perform the actual `git commit` in the user's real repo (via `os/exec` calling the user's real `git`, not a reimplementation — keep this simple and let git's own config/hooks/gpg apply), then append `KNIRV-DVE-Session: <id>` and `KNIRV-Evidence-Ref: sha256:<bundle_hash>` trailers to the message before the commit is finalized (standard trailer append, same technique as Entire's `prepare-commit-msg` hook — implement as an actual git hook installed by `knirv supervisor init`, not as logic embedded only in `knirv commit`, so it also fires if the user commits manually mid-session). This is the *only* thing KNIRV writes into the user's real repo — no evidence ref, no second remote, nothing else.
4. `knirv dve session commit` (existing) still finalizes/signs the bundle; `knirv commit` becomes the composed operation: attribution → code commit with trailer (in the user's real repo) → `knirv dve session commit` → Phase I's push (to KNIRVSERVER's Proof Ledger, a wholly separate system).

Deliverables: `cmd/commit.go`, `internal/gitcommit/{trailers,attribution,hooks}.go`, hook-install step in `knirv supervisor init`, end-to-end test (init repo → supervised session → agent edits files → `knirv commit` → assert trailer present and attribution stats populated).

### Phase I — Proof Ledger Transport (Revised — Not Forgejo)

**Goal:** get the evidence ref onto KNIRVSERVER's Proof Ledger, authenticated via a KNIRV-issued token routed through KNIRVGATEWAY, without reimplementing git's wire protocol. The server side of this phase (bare-repo lifecycle, the smart-HTTP receive-pack handler ported from Forgejo's `githttp.go`, the hook-dispatch ingest callback) is specified authoritatively in `KNIRVSERVER_Commit_Strategy.md` §3.1-3.4 and is not repeated here — this phase covers only the CLI-side push mechanics.

**Decision, unchanged in substance, retargeted:** build a **git credential helper** (`knirv git-credential`), not a git-remote-helper. A credential helper is a tiny, well-specified protocol (`get`/`store`/`erase` over `key=value` lines on stdin/stdout, see `git-credential(1)`) that lets `knirv` mint/attach a short-lived KNIRVSERVER-issued bearer token as the `password` for the Proof Ledger's standard `https://` remote (per `KNIRVSERVER_Commit_Strategy.md` §3.4: simple tokens for now, formal CLI login later). This is the same "don't build a remote-helper" reasoning as before, now reinforced by the server side also being built by porting a real git-backed handler rather than a custom protocol.

Steps:
1. `cmd/git_credential.go`: implements `get`/`store`/`erase` against KNIRVSERVER's token-issuing auth (reuse whatever wallet/JWT flow `internal/core` already has for other KNIRVSERVER calls; per the strategy doc this is a stopgap bearer token today, not a full login flow yet).
2. `knirv supervisor init` configures the Proof Ledger's `.git/config` remote (a KNIRV-managed local clone/working reference to `https://<knirvserver-gateway-host>/proof-ledger/<project_id>.git`, **not** the user's real repo's `.git/config` — the two are entirely separate git working trees) with `credential.https://<knirvserver-gateway-host>.helper = knirv git-credential`.
3. Evidence ref strategy: `refs/knirv/dve/<session_id>`, sharded write path `<id[:2]>/<id[2:]>/bundle.json` + `events.jsonl` + `artifact-manifest.json` inside that ref's tree, mirroring Entire's `entire/checkpoints/v1` sharding (avoids one giant flat directory in the evidence tree) and matching `KNIRVSERVER_Commit_Strategy.md` §3.1's server-side layout exactly.
4. **The "separate evidence remote" config knob from the prior draft is removed — it's no longer a choice.** Every evidence push targets the Proof Ledger; there is no "same repo as source" option because KNIRV never has a repo that also holds source. Delete this as a decision point, not just resolve it.
5. Push the evidence ref at the end of `knirv commit` (Phase H), after the code commit (with its trailer) has already landed in the user's real repo via their own normal `git push` — the two pushes target entirely different remotes and are not coupled beyond the trailer's hash reference.

Deliverables: `cmd/git_credential.go`, `internal/gitremote/{refs,config}.go`, `knirv supervisor init` wiring, integration test against a local Proof Ledger test instance (built per `KNIRVSERVER_Commit_Strategy.md`, not a Forgejo docker-compose fixture) proving the evidence ref lands correctly and the code repo is never touched beyond its own commit.

### Phase J — Fail-Closed Redaction Gate

**Goal:** nothing leaves the machine under-redacted. This directly implements `DVE_Alignment_Plan.md` Phase 4's redaction requirement, using Entire's fail-closed pre-push pattern as the enforcement shape.

Steps:
1. `internal/redact` package: entropy scoring + known-secret-pattern matching (reuse/extend anything already in `internal/promptguard` if it overlaps — check before duplicating) as an always-on baseline layer, run against the event log and any artifact bound for the evidence ref.
2. Gate placement: inside `knirv commit`'s push step (Phase H/I), **before** the evidence ref is written — not after.
3. Fail-closed: redaction-pass failure, or a detected secret pattern that can't be auto-redacted, **aborts the commit/push** with a clear error, not a warning. Record a `redaction.aborted` event locally (still useful audit signal even though nothing was pushed).
4. Every actual redaction performed is itself recorded as a `dve.Event` (`redaction.applied`, referencing what was redacted by class, not by value) so the audit trail shows *that* something was removed, per the alignment plan's Phase 4 requirement.

Deliverables: `internal/redact/{scan,redact}.go`, wiring into `knirv commit`, tests for both the fail-closed abort path and the successful-redaction-with-event path.

### Phase K — Git-Native Commit Signing (secondary layer)

**Goal:** let reviewers who only have plain git tooling verify commit authenticity, on top of (not instead of) KNIRV's required Ed25519 bundle signature.

Steps:
1. Port Entire's `objectsigner.go` approach: reuse the user's existing `commit.gpgsign`/`user.signingkey`/`gpg.format` config, support both GPG and SSH-agent signing.
2. Best-effort: signing failure is logged and the commit proceeds unsigned — this layer is optional polish, `dve.Signer`'s Ed25519 signature over the bundle remains the actual trust anchor and is never optional.

Deliverables: `internal/gitcommit/objectsigner.go`, config docs, opt-out flag.

### Phase L — Large-Artifact Storage (Revised — No Git LFS)

**Goal:** get memvid streams / screen recordings from a local session to KNIRVSERVER without putting them in git at all. The prior draft recommended Git LFS on the assumption KNIRVSERVER ran Forgejo (which supports LFS natively); with no Forgejo, LFS is dropped from the plan entirely, not replaced with a different git-based mechanism — full reasoning in `KNIRVSERVER_Commit_Strategy.md` §3.5.

Steps:
1. Large artifacts upload directly to KNIRVSERVER's existing evidence/object storage — confirmed to be `internal/dveevidence`'s `FileStore` (content-addressed local-disk JSON/blob storage today, no S3/minio backend yet) — addressed by the same `sha256:` hash already computed client-side for `ArtifactRef.Hash`. No new hashing scheme, no LFS pointer format to learn.
2. The evidence commit in the Proof Ledger carries only a small `artifact-manifest.json` pointer file per artifact (name, class, hash, size) — never the artifact bytes.
3. No `.gitattributes`/LFS wiring needed anywhere, in `knirv supervisor init` or otherwise.

Deliverables: artifact-upload client code pointed at KNIRVSERVER's evidence/object storage endpoint, manifest generation in `knirv commit`'s push step, size-quota check before upload (ties into `DVE_Alignment_Plan.md` Phase 10's "bundle size quotas" — now scoped to the object-storage upload, not a git push).

### Phase M — Server-Side Ingest via Hook Dispatch (Revised — Not a Forgejo Webhook)

**Goal:** replace the alignment plan's raw-upload Phase 7 with the git-native trigger KNIRVSERVER now implements: not a webhook (no Forgejo), not a poller, but the hook-dispatch pattern cloned from Forgejo's own architecture (shared `core.hooksPath`, dispatcher script, `knirvserverd hook post-receive`, authenticated Unix-socket loopback callback) — fully specified in `KNIRVSERVER_Commit_Strategy.md` §3.3. That document also corrects an assumption this phase originally made: **the server-side ingest pipeline is a new, dedicated implementation for the Bundle Sign Path, not a direct import/reuse of the CLI's `dve.VerifyBundle`/`ReplayPolicy`** (KNIRVSERVER already independently reimplements similar logic once, in `internal/dveevidence`, for the separate Pod Evidence / Dock Path — a second independent reimplementation for the Bundle Sign Path was the deliberate choice, not an oversight to fix). This phase is mostly server-side (KNIRVSERVER, a different repo) but the CLI has a role: emitting a well-formed, documented wire contract.

Steps (CLI-side scope only; server-side implementation tracked in `KNIRVSERVER_Commit_Strategy.md` §4):
1. Document the `Bundle` JSON shape (`dve/types.go`) as the wire contract KNIRVSERVER's new ingest pipeline must parse — the two implementations are independent but must agree on this schema, since it's effectively a versioned API between two separately-maintained codebases now (worth a schema version bump discipline if either side changes shape).
2. Document the ref-naming/sharding contract (`refs/knirv/dve/<session_id>`, `<id[:2]>/<id[2:]>/bundle.json` + `events.jsonl` + `artifact-manifest.json`, Phase I) as the tree layout KNIRVSERVER's hook-dispatch callback must parse when it fetches the just-pushed commit locally.
3. Add a `knirv dve verify --from-ref <ref>` mode that fetches and verifies directly from a Proof Ledger ref against the CLI's own `dve.VerifyBundle` (useful for local debugging of what the server *should* see, and for CI) — explicitly a client-side sanity check, not a claim that it's running the same code the server runs.

Deliverables: `Bundle` schema contract doc (with a versioning note), ref-naming contract doc, `--from-ref` verify mode.

### Phase N — Line-Level Attribution Queries

**Goal:** `knirv blame`/`knirv why`, ported from Entire's `entire blame`/`entire why`.

Steps:
1. `knirv blame <file>`: run `git blame --line-porcelain`, resolve each commit's `KNIRV-DVE-Session` trailer, join to that session's bundle (fetched from the evidence ref if not local) for per-line agent/model/session tagging.
2. `knirv why <file>:<line>`: prose explanation of a single line's provenance — same join, single-line output.

Deliverables: `cmd/blame.go`, `cmd/why.go`, `internal/attribution/query.go`.

### Phase O — Hardening & Governance

**Goal:** carry forward `DVE_Alignment_Plan.md` Phase 10 items that are now concretely actionable given Phases A–N exist.

Steps: bundle size quotas (ties to Phase L's object-storage upload), artifact retention policy on the evidence refs, quarantine flow for suspicious evidence pushes (reject/flag evidence refs that fail `ReplayPolicy` — there's no Forgejo branch-protection to lean on since we're not running Forgejo the application, just its ported git-handling code, so this is new KNIRVSERVER logic, not a config toggle), role-based access to evidence refs (leverage KNIRV's own existing project/org permission model — per `KNIRVSERVER_Commit_Strategy.md` §3.1, the Proof Ledger's one-bare-repo-per-project layout was chosen specifically so this maps directly onto it, rather than needing a parallel access-control system), audit logging for evidence-ref reads, per-organization validation profiles (multiple `supervisor.Policy` + `dve.Policy` pairs selectable per project).

## 7. Command Surface Summary

| Command | Status | Notes |
|---|---|---|
| `knirv dve session start/status/commit` | exists | unchanged |
| `knirv dve verify` | exists | Phase M adds `--from-ref` |
| `knirv hook claude-code` / `install` | exists | Phase A wires it into the active session |
| `knirv pty run` / `adapters` | exists | Phase A wires it in; Phase C extends adapters, adds `hermes` |
| `knirv supervisor run` | **new**, Phase B | the actual loop |
| `knirv supervisor status` | **new**, Phase B | `WorkflowInstanceView` port |
| `knirv supervisor list-agents` | **new**, Phase C | |
| `knirv supervisor init` | **new**, Phase H/I | installs the client-side `prepare-commit-msg` hook in the user's real repo, configures the Proof Ledger's credential helper (no LFS setup — Phase L dropped it) |
| `knirv supervisor evidence-server` | **new**, Phase D | internal, spawned not user-invoked |
| `knirv commit` | **new**, Phase H | the sanctioned commit path |
| `knirv git-credential` | **new**, Phase I | git credential helper protocol |
| `knirv blame` / `knirv why` | **new**, Phase N | |

## 8. Open Questions / Decisions Needed

Several items from the original list are now resolved by `KNIRVSERVER_Commit_Strategy.md` and removed here rather than re-litigated: same-repo-vs-separate-evidence-repo (moot — always separate now, §4/Phase I), LFS availability (moot — dropped, Phase L), and the server-side ingest target (resolved — dedicated Bundle Sign Path pipeline, independent of `internal/dveevidence`, §1/Phase M).

Still open:

1. **Policy config format** — JSON (consistent with rest of KNIRV config) vs. TOML (matches Ralph, arguably more human-writable for a phase graph). Recommend JSON; flag for confirmation.
2. **Evidence ref vs. evidence branch** — this plan recommends `refs/knirv/dve/<session_id>` (one ref per session, like Entire's `git-refs` backend option) over a single growing `knirv/dve/v1` branch (like Entire's `git-branch` backend/default). Recommend starting with per-session refs since they parallelize better and avoid a single branch becoming a merge/rebase bottleneck; confirm KNIRVSERVER's Proof Ledger hook-dispatch callback (which watches individual pushed refs directly via git's hook stdin protocol, not a ref-pattern subscription) has no issue with this — it should not, since Forgejo's own equivalent mechanism works the same way, but worth a sanity check once Phase I/M's server side is actually running.
3. **Which agent gets the first full adapter-triad + evidence-server integration** — recommend Claude Code first (it already has the most infrastructure: `hookguard.ClaudeCodeAdapter`, native hook API), then Codex, then OpenCode, then Hermes — matching the alignment plan's own "Add one tool adapter first, preferably Codex or OpenCode" guidance loosely, adjusted for what already exists.
4. **Naming for the two parallel evidence tracks** — "Bundle Sign Path" (`dve/` ↔ Proof Ledger) and "Pod Evidence / Dock Path" (`dvepod` ↔ `internal/dveevidence`), proposed in `KNIRVSERVER_Commit_Strategy.md` §3.6. Confirm before they're baked into package doc comments / possible renames (§1).
5. **Bundle schema versioning discipline** — now that `dve/types.go`'s `Bundle` shape is a wire contract between two independently-maintained implementations (CLI's `dve/` and KNIRVSERVER's new Bundle Sign Path ingest pipeline, per Phase M), decide how schema changes get coordinated — a shared schema doc, a version bump convention, contract tests, or something else. Not urgent before Phase M starts, but worth deciding before it does.

## 9. Risks

- **Scope.** This is a large plan across ~15 phases, now spanning two repos (`packages/cli` and `packages/KNIRVSERVER`). Phase A is a hard prerequisite for the evidence story to be coherent at all; recommend treating A–D as a single milestone before anything else is scheduled.
- **Two evidence systems drifting further apart if Phase A slips.** `AnchorEvidenceAsync` calling a route that doesn't exist server-side was discovered mid-plan, not designed in — a reminder that assumptions about "what the other side of an integration already does" need verifying against real source, not just the aspirational client code, before being load-bearing.
- **Proof Ledger transport is real, ongoing work, not a config change** — porting Forgejo's `githttp.go` under the new GPLv3 clearance (`KNIRVSERVER_Commit_Strategy.md` §3.2) substantially de-risks this relative to hand-rolling it, but it's still a small git server that needs building, testing, and operating; don't let "we're cloning Forgejo's code now" read as "this got easy."
- **Two fully parallel evidence tracks (Bundle Sign Path / Pod Evidence Dock Path) is a deliberate, accepted maintenance cost, not a free resolution** — verification logic, redaction, retention policy, and future feature work now potentially need doing twice, once per track, until `dvepod`'s long-term viability question is settled one way or the other.
- **The gateway-bypass exception for the hook-dispatch callback (`KNIRVSERVER_Commit_Strategy.md` §3.3) needs its backlog item actually filed**, scoped to cover `AgentControlServer`'s existing bypass, this new one, and the future CLI-Supervisor ↔ server-side "Expert Advisor" channel together — don't let a third instance of the same debt accumulate before the structural fix happens.
- **Redaction is the highest-consequence single component in this plan** — it's the last thing standing between a supervised session and a secret landing in a Proof Ledger commit history that is, by git's nature, very hard to truly delete afterward. Phase J should get proportionally more review/test investment than its line count suggests.
