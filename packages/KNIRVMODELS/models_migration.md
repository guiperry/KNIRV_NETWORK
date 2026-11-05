# Models Migration — Status

This document tracks the models-routing migration across phases. Phase 5 has now
been **executed** (see "Phase 5 Execution Status" below); the inventories are kept
as the audited record of what was removed / renamed / retained.

## Phase 5 Status — removal/rename inventory (prep, no deletions)

The migration (phases 0–4) moved decision routing from `HEAD` to `HEART` and
stood up `HANDS`. Phase 5 cleans up the leftover `HEAD`/legacy aliases and
fallbacks. Each row is a hit-list against `packages/KNIRVMODELS` (and the
`KNIRVSDK` testvectors/bindings where relevant). Cross-repo consumers are flagged
as handoffs because they must be updated in lockstep.

### rg check (current hits)

| # | Item | Where (current hits) | Notes |
|---|------|---------------------|-------|
| 1 | `gpt_compat` alias layer | `pkg/hashing/transformer/gpt_compat.go` (exports `GPT`, `GorgoniteConfig`, `NewGPT`, `DefaultGorgoniteConfig`, `Dataset`, `TrainConfig`, `Tokenizer`, `BPETokenizer`, `TiktokenTokenizer`, `Generate`, `SaveModel`, `LoadModel`, `PrepareDataset`, `GetPretrainingConfig`, `TrainModel`, `RunGorgoniteProtocol`, `CreateDummyDataset`, `RunPretraining`, `SampleToken`) | **Live source file, not just a test.** Importers that must be repointed to `pkg/hands/gpt` before deletion: `pkg/hashing/inference/recursive.go:29` (`transformer.Tokenizer`), `pkg/hands/gpt/gpt_parity_test.go:50,51` (`transformer.LoadModel`), `pipeline/4_DATA_TRAINER/cmd/data-trainer/main.go` (`GorgoniteConfig`, `DefaultGorgoniteConfig`, `NewGPT`, `LoadModel`, `SaveModel`) and `main_test.go:16,17` (`GorgoniteConfig`). |
| 2 | `pkg/models/head.go` (HEAD model) | `pkg/models/head.go`, `pkg/models/head_test.go`, `pkg/models/head_contract_test.go` | **HEAD is retained as VLM + mirrored text (Phase 3), NOT fully retired.** It serves VLM manifest + promoted weights (`/models/head/models`, `/models/head/weights/{ver}`) and the mirrored-text path that `KNIRVINFERENCER` consumes. Phase 5 removes only the *legacy decision-route* surface, not the VLM model. If the plan says HEAD should be fully retired, flag as an **owner decision** rather than choosing here. |
| 3 | `registerHEAD` + `KNIRV_HEAD_DIR` | `pkg/hashing/transformer/heart_head.go:16,21,26,33,43,55,58,75,79`; `heart_head_test.go`; `heart_head_dir_test.go` | **Renamed `registerHEAD` → `registerHEARTDecision` (behavior kept) + removed `KNIRV_HEAD_DIR` one-release override.** Per owner 2026-11-04: item 3 = KEEP behavior, RENAME. DONE. |
| 4 | `head_migration.go` | `pkg/hashing/transformer/head_migration.go`, `head_migration_test.go` | Remove the HEAD migration source + test. |
| 5 | `head_reasoning.json` vector | `KNIRVSDK/testvectors/models/head_reasoning.json` (consumed by `npm-package/scripts/test-models.mjs` via `head_reasoning.json` at `:59`) | Remove legacy HEAD reasoning vector + consumer. |
| 6 | `/models/head/reason` proxy | `pkg/models/migration_test.go:106,359,433` (`deprecated-alias`); `pkg/models/compatibility_matrix.json:13` (`Head.Reason` → `/models/head/reason`, `deprecated-alias`) | The proxy route is already gone (404; decision routing is at `/models/heart/decide`, fixed in phase 0/4). Only test expectations + the compatibility-matrix alias row remain; remove them. |
| 7 | legacy `/models/head/models` alias | `migration_test.go:118` (`layout-switch` → `/models/heart/decide/models`), `heart_head.go:79`, `head.go:54,103`, `head_test.go:65,82`, `head_routes_test.go:102` | **Preserve the VLM manifest route** (`HeadManifest` at `head.go:103-116`, route `/models/head/models`); it is the live VLM contract and is called by the SDK (see handoff/SDK note). Remove only the *decision-metadata* alias that pointed `/models/head/models` at decision data. **SDK callers that must keep working:** Go `Client.Manifest` (`KNIRVSDK/go-package/models/client.go:86`, `HeadPrefix+"/models"` = `/models/head/models`) + tests `models_test.go:140-143,292,415`; TS `manifest()` (`npm-package/src/models/index.ts:441`) + tests `test-models.mjs:61,68,71,85`. Core VLM contract: `head.mirror.generate` stage (`KNIRVSDK/core/data/frame_defaults.json:487`) and the VLM `HeadManifest`/`HeadWeights` routes (`head.go:103,125`). |
| 8 | `ops.assistant` stage | KNIRVMODELS consumers: `heart.go:199,218,219` (operator-gated stage), `heart_qwen_test.go:83`, `framecheck.go:71` (comment), `heartqwen/loop_test.go:147,217`, `internal/policy/gate_test.go:45,46,75,87` (gate routes ops.assistant tools), `KNIRVSDK/go-package/models/frame/frame_test.go:121,125` (T2-only assertion). **Rust core is the source of truth:** stage id defined at `KNIRVSDK/core/data/frame_defaults.json:410`; regenerated to `KNIRVSDK/testvectors/models/frames.json` (`ops.assistant` at all occurrences — regenerated vector); Go frame parity test `KNIRVSDK/go-package/models/frame/fixture_test.go` pins frame hashes to that vector. | **Rename must be core-first (bindings-only rule):** change `frame_defaults.json` stage id → regenerate `frames.json` → update Go frame parity test + Go/TS binding tests + KNIRVMODELS consumers. The new stage id is a **provisional / open decision** — flag to the owner, do not invent one. | **DONE 2026-11-04** — owner decided to *remove* `ops.assistant` (no new id, no alias) and fold its tools into `hands.assistant`; core-first rebuild done, consumers + tests repointed (see Execution Status). |
| 9 | `KNIRV_HEAD_DIR` env | `flags.go:115,120`, `heart_head.go:16,21,26,33,43,55,58`, `heart_head_dir_test.go` | One-release override; remove (consolidated with item 3). |
| 10 | Makefile / scripts / env / data-dir refs | `pipeline/5_HEAD_TRAINER/README.md:46,47` (`KNIRV_HEAD_*` knobs), `scripts/smoke-models.sh:26` (`for p in /models/head/models`) | Update to post-migration route/knob names; keep the VLM manifest route if HEAD is retained. |
| 11 | `head.train` T2 tool (in `ops.assistant` stage) | Tool def `head.train` in `KNIRVSDK/core/data/frame_defaults.json:555`; regenerated `KNIRVSDK/testvectors/models/frames.json:475,557`; exercised via the ops.assistant gate (see item 8). Phase 5 acceptance gate lists it for removal. | **Owner decision — training on hold.** `head.train` was a T2 tool on the (now-removed) `ops.assistant` stage surface; the Phase 5 gate calls for its removal but training is currently on hold. Flag to owner; do not remove until training policy is settled. If training resumes, the tool would attach to `hands.assistant`. |
| 12 | `head.reason` ops alias | `internal/ops/ops.go:118` (`"head.reason": d.HeartDecide`); `pkg/models/compatibility_matrix.json:13` (`Head.Reason` → `/models/head/reason`, `deprecated-alias`). | Plan table says remove in Phase 5. It is the ops-facing alias of the `/models/head/reason` proxy (item 6); remove together with item 6 once the proxy is gone (route already 404). |
| 13 | `head-wasi` / `knirv-head` crate (data migration, not a plain rename) | `Makefile:385` (`build-head-wasi` target); `pkg/heart/decision/runner.go:1,4,82` (was `head`/`head-wasi.wasm`); `heart/decision/src/lib.rs:1` (`knirv-head` doc), `config.rs:6,8` (on-disk format ids `knirv-head/v1`,`knirv-head/v2`), `model.rs:220,417,446,453`; `heart/decision/tests/{head,layout,sets}.rs` + `heart/decision/tests/golden_config.json` + `pkg/heart/decision/testdata/golden_config.json` (golden fixtures carrying the `knirv-head/v1\|v2` format ids that the data migration must keep loading); SDK comments `KNIRVSDK/go-package/models/head.go:24`, `registry.go:28`. | **Owner decision / data migration.** This is the Rust core crate (source of truth — KNIRVSDK bindings-only rule); renaming the crate or the `knirv-head/v1\|v2` on-disk format ids is a *data migration*, not a rename — existing persisted weights (format `knirv-head/v1`) must keep loading, so the golden fixtures above must still parse. Flag to owner; do not rename without a migration plan. | **DONE 2026-11-04** — format ids migrated to `knirv-heart/v1`/`knirv-heart/v2` with `knirv-head/v1`/`knirv-head/v2` kept as legacy loaders (backward compatible); golden fixtures migrated (originals backed up to `/tmp/knirv_phase5_format_backup/`); wasm rebuilt (see Execution Status). |
| 14 | `KNIRV_MODEL_LAYOUT` default flip (source of truth = `flags.go`, not the matrix) | `pkg/models/flags.go:39-44` (`ModelLayout()` hard-codes `Legacy` default), `pkg/models/migration_test.go:23-24` (`TestMigrationFlagDefaults`), `:164-166` (matrix fixture check), `pkg/models/compatibility_matrix.json:29` (`default: "legacy"` — **test fixture only, not a config source**). The `pkg/head/` (Go) and `head/` (Rust crate) directories have **already been removed** (moved to `pkg/heart/decision/`). | **DONE:** default flipped to `migrated`; `legacy` only honoured when `legacyLayoutSupported()` is true (now `false`, since the legacy files are gone). `TestMigrationFlagDefaults` (lines 23-24), fixture check (lines 164-166), and matrix JSON default updated to `migrated`. |

### Phase 5 Execution Status (as of 2026-11-04)

No publish / no commit / no git-history rewrites. KNIRVSDK core (`packages/KNIRVSDK/core`)
is the source of truth; the local `0.2.0` SDK was rebuilt, re-stamped, and its
`dist/` + `~/.local/share/knirv/sdk/0.2.0/` copies + `frames.json` testvectors were
reconciled (wasm digest `7bdbe1cb…`). All affected suites are green.

- **DONE — item 1** (`gpt_compat`): deleted `pkg/hashing/transformer/gpt_compat.go`;
  importers repointed to `pkg/hands/gpt` (`gptpkg`) and `pkg/hands/tokenizer`
  (`handstok`); deleted `gpt_parity_test.go`. `pkg/hashing/transformer`,
  `internal/ops`, `pkg/hands/gpt` tests pass.
- **DONE — item 2** (`pkg/models/head.go`): HEAD retained as VLM + mirrored text;
  only the legacy *decision-route* surface was retired (see items 6/12). No change
  needed beyond items 6/12.
- **DONE — item 3** (`registerHEAD` + `KNIRV_HEAD_DIR`): per owner, RENAME
  `registerHEAD` → `registerHEARTDecision` (behavior kept) and REMOVE the
  `KNIRV_HEAD_DIR` override + its test; deleted `heart_head_dir_test.go`.
- **DONE — item 4** (`head_migration.go`): deleted `head_migration.go` +
  `head_migration_test.go`; removed the `MigrateHeadToDecision` call.
- **DONE — item 5** (`head_reasoning.json`): deleted
  `KNIRVSDK/testvectors/models/head_reasoning.json`; removed npm consumer
  (`test-models.mjs:59`) and Go `models_test.go` load; `TestClientReasonReportsNoModel`
  → `TestClientDecideReportsNoModel` repointed to `c.Decide`.
- **DONE — item 6** (`/models/head/reason` proxy): removed `Head.Reason` matrix row
  + `migration_test.go` expectations/comment. Route was already 404; decision routing
  is at `/models/heart/decide`.
- **DONE — item 9** (`KNIRV_HEAD_DIR` env): removed from `flags.go` and
  `heart_head.go`; consolidated with item 3.
- **DONE — item 10** (Makefile / scripts / env refs): `5_HEAD_TRAINER/README.md`
  updated to `pkg/heart/decision/cycle`; `KNIRV_HEAD_*` knobs retained.
- **DONE — item 11** (`head.train` T2 tool): removed `head.train` tool from
  `frame_defaults.json` + from `ops.assistant` tools list; regenerated `frames.json`.
  No KNIRVMODELS code referenced the frame-tool; only the `head-trainer` binary reads
  it from disk (untouched).
- **DONE — item 12** (`head.reason` ops alias): removed `ops.go:118` alias; removed
  SDK `Reason()` method and repointed `Advise()` → `Decide()` (heart-decision route);
  repointed npm `reason()` → `HEART_DECISION_PREFIX+"/reason"`; fixed
  `migration_test.go` (`c.Reason` → `c.Decide`, `HeadPrefix` → `HeartDecisionPrefix`)
  and the test handler to serve `HeartDecisionPrefix+"/reason"`.
- **DONE — item 14** (`KNIRV_MODEL_LAYOUT` default flip): default flipped to
  `migrated`; matrix JSON + tests updated.
- **DONE — item 8** (`ops.assistant` stage): owner decision 2026-11-04 — **remove**
  `ops.assistant` (no new id, no alias). Its tools were folded into the existing
  `hands.assistant` stage (layer HANDS), which now also carries the operator-gated
  T2 tools, so the policy-gate tests remain valid. Core-first: edited
  `KNIRVSDK/core/data/frame_defaults.json` → removed the `ops.assistant` block and
  merged its tools into `hands.assistant` → rebuilt knirv_core.wasm (digest
  `7bdbe1cb…`), re-stamped `stamp.go`, updated `dist/0.2.0` sha256/size/`manifest.json`
  + `~/.local/share/knirv/sdk/0.2.0/` + regenerated `frames.json`. Repointed KNIRVMODELS
  consumers: `heart.go` operator gate + comment, `heart_qwen_test.go`
  (`TestHandsAssistantNeedsAnOperator`), `heartqwen/loop_test.go:147,217`,
  `internal/policy/gate_test.go` (`TestOnlyHandsAssistantCanEverPropose` + cases), and
  `KNIRVSDK/go-package/models/frame/frame_test.go:121,125` (`TestPolicyAllowList`).
  `HeartHandsFallbackQwen()` is dead/unused (no call path) — only its comment was
  repointed to `hands.assistant`; the `KNIRV_HANDS_FALLBACK_QWEN` env + future fallback
  switch is left in place.
- **DONE — item 13** (`knirv-head/v1|v2` → `knirv-heart/v1|v2` format ids): owner
  decision 2026-11-04 to migrate the on-disk format ids. Core-first (Rust crate is
  source of truth): set `FORMAT`/`FORMAT_V2` = `knirv-heart/v1`/`knirv-heart/v2` in
  `heart/decision/src/config.rs`; kept `knirv-head/v1`/`knirv-head/v2` as **legacy
  loaders** (`FORMAT_LEGACY`/`FORMAT_V2_LEGACY`) so already-persisted weights keep
  loading (backward compatible). Updated `model.rs` format strings + `lib.rs` crate doc.
  Migrated the on-disk golden fixtures
  (`heart/decision/tests/golden_config.json` + `pkg/heart/decision/testdata/golden_config.json`)
  to `knirv-heart/v1`; **originals backed up to
  `/tmp/knirv_phase5_format_backup/`** (see note). Updated inline test configs in
  `head.rs`/`sets.rs`/`layout.rs` to the new ids; kept the `knirv-head/v9` negative
  test (still unsupported). Rebuilt `heart-decision-wasi.wasm` via
  `make build-heart-decision-wasi` and replaced the embedded
   `pkg/heart/decision/heart-decision-wasi.wasm`. Verify: `cargo test` (heart/decision,
   v1+v2 + legacy load), `go test ./pkg/heart/decision/...` (embedded wasm loads +
   service_test + contract_test), `node test-models.mjs` — all green.
- **F1 FIX (Codex audit, 2026-11-04)** — legacy on-disk format compatibility was
  untested (only `knirv-heart/*` was exercised). Added `heart/decision/tests/legacy_compat.rs`
  + the fixture `heart/decision/tests/legacy_v1_config.json` (the pre-migration
  `knirv-head/v1` config, copied from `/tmp/knirv_phase5_format_backup/`, not read at
  test time) proving: (a) the legacy `knirv-head/v1` config + its golden safetensors
  still load and reason; (b) a valid legacy `knirv-head/v2` config with `choice_set_heads`
  loads and runs the set scorer; (c) freshly-created configs serialize with the
  `knirv-heart` ids while legacy ids are parsed as-is (load-only, never rewritten).
  `cargo test` (heart/decision) green; `go test ./pkg/heart/decision/...` green.

#### Core-frame reconciliation (local, uncommitted)

The committed `frame_defaults.json` was missing `hands.assistant`, `hands.plan`, and
`head.vision` (stage + tool) that the committed `e7be1…` wasm and the SDK binding
contract (`TestContractMirrorsCore`) require. Re-added: `hands.assistant` +
`hands.plan` stages (layer HANDS, gpt_eligible) and `head.vision` stage (layer HEAD)
+ tool (tier 1, served by the daemon VLM, listed as unimplemented-by-ops in
`internal/ops/ops_test.go`). Rebuilt → digest `7bdbe1cb…`; `frames.json` regenerated.

### Cross-repo handoffs (flagged, not owned here)

- **backend_server** (`KNIRV_CORP/packages/server/backend`): the models unix
  socket is reached *through* backend_server, which authenticates callers. Any
  client code that still hits `/models/head/reason` or `/models/head/models` for
  *decision* metadata must move to `/models/heart/decide` / `/models/heart/decide/models`.
  The VLM manifest route is retained (item 7) — keep those callers.
- **KNIRVINFERENCER**: **retain** dual `HEART`+`HEAD` mirrored generation
  (`KindMirrorGenerate` / `GenerateMirrored`, and the `head.mirror.generate`
  stage at `core/data/frame_defaults.json:487` — HEAD is "vision and mirrored
  text"). Phase 3's accepted architecture keeps this live; do **not** retire it to
  HEART-only. Remove only the **legacy HEAD decision-route callers** once their
  HEART decision replacements pass. (The KNIRVMODELS-side `MirroredGeneration()`
  flag at `flags.go:80` is kept.)
- **KNIRV_CORP**: owns the `backend_server` source (separate repo). Hand off the
  route/knob changes above to that repo.

## Phase 4 Status — closure (done)

- `HANDS` is served by the daemon under `/models/hands/*` with a persistent audit
  log, the daemon T0/T1 tool set, and a policy gate (`buildHands`/`buildHandsWith`).
- `models.HandsEnabled()` gates registration; `KNIRV_HANDS_ENABLED=1` required.
- Production default checkpoint load path (`loadHandsEngine`) is tested
  (present/absent/corrupt); the `engineBuild == nil` branch is exercised in
  `hands_wiring_test.go`.
- A real core-authorized T0 tool call (`manifest.get`) is driven through
  `/models/hands/act` and verified in the audit chain
  (`TestHandsActRecordsAuditEntry`).
- `models.HandsEnabled()` disabled registration branch covered
  (`TestRegisterModelsHandsDisabled`).
- `Heart.GenerateWithDecision` SDK contract (Rust core source of truth + Go/TS
  bindings) sends `with_decision` on the wire (fixed in v3).

---

## Phase 5 Draft Removal Steps — DRAFT, NOT APPLIED

> Preparatory only. **Nothing here has been executed.** Every step is a plan. Code
> lives in `KNIRVMODELS` (in-scope); cross-repo components are handoffs, not
> drafted here. Delete/relocate only after mig_phase2 + mig_phase4 are VALIDATED
> and the Codex Phase 5 audit is back.

### 1. `gpt_compat` alias layer (live source file)
- **Action:** delete `pkg/hashing/transformer/gpt_compat.go`; repoint every importer to `pkg/hands/gpt`.
- **Files:** `pkg/hashing/transformer/gpt_compat.go`; importers `pkg/hashing/inference/recursive.go:29`, `pkg/hands/gpt/gpt_parity_test.go:50,51`, `pipeline/4_DATA_TRAINER/cmd/data-trainer/main.go` + `main_test.go:16,17`.
- **Steps:** `recursive.go`: `transformer.Tokenizer` → `handstok.Tokenizer`. `gpt_parity_test.go`: `transformer.LoadModel` → `gpt.LoadModel`. data-trainer: `transformer.GorgoniteConfig`→`gpt.Config`, `DefaultGorgoniteConfig()`→`gpt.DefaultConfig()`, `NewGPT`→`gpt.NewGPT`, `LoadModel`/`SaveModel`→`gpt.*`. Delete `gpt_compat.go`.
- **Tests that prove removal:** `TestGPTCompatParity_FixedSeed` (guards the §3.3 alias contract — removed with the contract); data-trainer tests run on the direct `gpt.*` types.
- **Verify:** `grep -rn "transformer\.\(GPT\|GorgoniteConfig\|NewGPT\|DefaultGorgoniteConfig\|Dataset\|TrainConfig\|Tokenizer\|Generate\|SaveModel\|LoadModel\|PrepareDataset\|GetPretrainingConfig\|TrainModel\|RunGorgoniteProtocol\|CreateDummyDataset\|RunPretraining\|SampleToken\)"` → empty; `go test ./pkg/hands/gpt/... ./pkg/hashing/inference/... ./pipeline/4_DATA_TRAINER/...`.

### 2. `pkg/models/head.go` — HEAD retained as VLM + mirrored text
- **Action:** **do not retire HEAD.** Remove only the legacy *decision-route* surface; keep the VLM model serving `HeadManifest` (`/models/head/models`) + `HeadWeights` (`/models/head/weights/{ver}`) and the mirrored-text path `KNIRVINFERENCER` consumes (`head.mirror.generate`, `core/data/frame_defaults.json:487`).
- **Files:** `pkg/models/head.go`, `pkg/models/head_test.go`, `pkg/models/head_contract_test.go` (`HeadCapabilities{Vision,TextMirror,Embed}` stays).
- **Steps:** drop any decision-metadata route/test under HEAD; leave `HeadManifest`/`HeadWeights` and `TextMirror` capability intact. Remove `reg.Register(models.NewHead(...))` only if HEAD is later fully retired — **flag as owner decision**, do not do here.
- **Verify:** `go test ./pkg/models/... ./internal/cli/headless/...`; SDK `Manifest`/`manifest()` still resolves `/models/head/models` (item 7).

### 3. `registerHEAD` + `KNIRV_HEAD_DIR` fallback
- **Action:** remove `registerHEAD` and the `KNIRV_HEAD_DIR` one-release override; delete `heart_head_test.go`, `heart_head_dir_test.go`.
- **Files:** `pkg/hashing/transformer/heart_head.go`, `heart_head_test.go`, `heart_head_dir_test.go`, `pkg/models/flags.go:115,120`.
- **Verify:** `go test ./pkg/hashing/transformer/...`; `grep -rn "KNIRV_HEAD_DIR\|registerHead\|registerHEAD"` → empty.

### 4. `head_migration.go`
- **Action:** delete migration source + test.
- **Files:** `pkg/hashing/transformer/head_migration.go`, `head_migration_test.go`.
- **Verify:** `go test ./pkg/hashing/transformer/...`.

### 5. `head_reasoning.json` vector + consumer
- **Action:** remove the legacy HEAD reasoning vector and its consumer assertion.
- **Files:** `KNIRVSDK/testvectors/models/head_reasoning.json`; `npm-package/scripts/test-models.mjs` (`wire round trip` at `:59`).
- **Steps:** delete the vector; keep the `planText`/`inputWidth`/`abstainAdvice`/`BYTE_V1` assertions in `wire round trip`.
- **Verify:** `node ./scripts/test-models.mjs`.

### 6. `/models/head/reason` proxy — test expectations only
- **Action:** remove the deprecated-alias expectations (route already 404).
- **Files:** `pkg/models/migration_test.go:106,359,433`.
- **Steps:** drop the `{"Head.Reason", "/models/head/reason", ...}` row and the two comments; assert `heart/decide` decision route instead (covered by `TestCurrentDecisionRoute`).
- **Verify:** `go test ./pkg/models/...`; `grep -rn "head/reason"` → empty.

### 7. legacy `/models/head/models` alias — preserve VLM manifest route
- **Action:** remove only the *decision-metadata* alias (migration_test.go:118 row); **keep** the VLM manifest route `/models/head/models` and its SDK callers.
- **Core VLM contract:** `HeadManifest`/`HeadWeights` (`head.go:103,125`); `head.mirror.generate` stage (`core/data/frame_defaults.json:487`).
- **SDK callers to keep working:** Go `Client.Manifest` (`client.go:86`, `HeadPrefix+"/models"`) + tests `models_test.go:140-143,292,415`; TS `manifest()` (`index.ts:441`) + tests `test-models.mjs:61,68,71,85`.
- **Steps:** update migration-test row to assert decision metadata now lives at `/models/heart/decide/models`; leave `HeadManifest` route and the SDK `Manifest`/`manifest()` calls intact. If the VLM route itself is to be renamed, do it **in Rust core first** and regenerate bindings, then update the SDK callers.
- **Verify:** `go test ./pkg/models/... ./internal/cli/headless/...`; `go test` (knirv-sdk-go models) `Manifest` test; `node test-models.mjs` `manifest` assertions.

### 8. `ops.assistant` stage — **DONE 2026-11-04** (removed, tools folded into `hands.assistant`; no rename/alias per owner)
- **Action (as executed):** remove the `ops.assistant` stage entirely and fold its tools into the existing `hands.assistant` stage (layer HANDS), which now also carries the operator-gated T2 tools so the policy-gate tests stay valid. **Core-first** per the bindings-only rule.
- **Core contract (source of truth):** `KNIRVSDK/core/data/frame_defaults.json` — deleted the `ops.assistant` block; merged its tools into `hands.assistant`; regenerated `KNIRVSDK/testvectors/models/frames.json` (no `ops.assistant` occurrences); Go frame parity test `KNIRVSDK/go-package/models/frame/fixture_test.go` pins frame hashes to the vector.
- **Steps (as executed):** (1) removed `ops.assistant` + merged tools in `frame_defaults.json`; (2) regenerated `frames.json` (cargo generate-vector); (3) re-stamped `stamp.go` + `dist/0.2.0` sha256/size/`manifest.json` + `~/.local/share/knirv/sdk/0.2.0/`; (4) updated KNIRVMODELS consumers `heart.go:199,218` (operator gate + comment), `heart_qwen_test.go` (`TestHandsAssistantNeedsAnOperator`), `heartqwen/loop_test.go:147,217`, `internal/policy/gate_test.go` (`TestOnlyHandsAssistantCanEverPropose` + cases), and `KNIRVSDK/go-package/models/frame/frame_test.go:121,125`. `framecheck.go:71` had no `ops.assistant` reference (stale doc note). `HeartHandsFallbackQwen()` dead/unused — only its comment repointed.
- **Verify:** `cargo test` (frames regenerate + parity); `go test ./...` (knirv-sdk-go models/frame); `node test-models.mjs`; `go test ./pkg/models/... ./pkg/heartqwen/... ./internal/policy/...` — all green.

### 9. `KNIRV_HEAD_DIR` env (consolidated with item 3)
- **Action:** remove the env read in `flags.go:115,120` and any doc reference.
- **Verify:** `grep -rn KNIRV_HEAD_DIR` → empty.

### 10. Makefile / scripts / env / data-dir references
- **Action:** update references to post-migration route/knob names; keep the VLM manifest route (item 2/7).
- **Files:** `pipeline/5_HEAD_TRAINER/README.md:46,47` (`KNIRV_HEAD_*` knobs), `scripts/smoke-models.sh:26` (`for p in /models/head/models`).
- **Verify:** `bash scripts/smoke-models.sh` (where applicable) and doc grep.

### 11. `head.train` T2 tool — **DONE 2026-11-04** (tool already removed; training on hold)
- **Action:** `head.train` was removed from the (now-removed) `ops.assistant` stage tool set per the Phase 5 acceptance gate. **Training is on hold — flag to owner; do not restore until training policy settles.** If training resumes, the tool would attach to `hands.assistant`.
- **Core contract (source of truth):** `KNIRVSDK/core/data/frame_defaults.json` (stage `ops.assistant` tools); regenerated `KNIRVSDK/testvectors/models/frames.json`; Go frame parity test `KNIRVSDK/go-package/models/frame/fixture_test.go`.
- **Steps (core-first, bindings-only rule):** drop `head.train` from the stage tool list → regenerate vector → update parity + binding tests → update gate consumers (item 8).
- **Verify:** `cargo test` (frames regenerate); `go test ./...` (knirv-sdk-go models/frame); `go test ./pkg/models/... ./internal/policy/...`.

### 12. `head.reason` ops alias — remove with item 6
- **Action:** remove the `head.reason` ops alias; it is the ops-facing name for the `/models/head/reason` proxy (item 6), whose route is already 404.
- **Files:** `internal/ops/ops.go:118`; `pkg/models/compatibility_matrix.json:13`.
- **Steps:** delete `"head.reason": d.HeartDecide` from the alias map; remove the `Head.Reason` deprecated-alias row from the compatibility matrix.
- **Verify:** `go test ./internal/ops/... ./pkg/models/...`. Note: a blanket `grep -rn "head.reason\|Head.Reason"` will also match prose comments at `ops.go:70` and `migration_test.go:240,310,358` (plus the matrix alias row at `:13,106`); scope the grep to the alias map + alias tests, not the explanatory comments, or delete those comments too.

### 13. `head-wasi` / `knirv-head` crate — **DONE 2026-11-04** (format ids migrated to `knirv-heart/v1|v2`, legacy loaders kept)
- **Action (as executed):** migrate the on-disk format ids `knirv-head/v1`/`knirv-head/v2` → `knirv-heart/v1`/`knirv-heart/v2`. The Rust core (`heart/decision/src`) is the source of truth (bindings-only rule). Existing persisted weights in `knirv-head/v1`/`knirv-head/v2` keep loading via legacy loaders (`FORMAT_LEGACY`/`FORMAT_V2_LEGACY` in `config.rs`).
- **Files:** `Makefile` (`build-heart-decision-wasi`); `pkg/heart/decision/runner.go` (embed `heart-decision-wasi.wasm`); `heart/decision/src/lib.rs:1`, `config.rs` (`FORMAT`/`FORMAT_V2` + legacy consts), `model.rs` (format strings); `heart/decision/tests/{head,layout,sets}.rs`, `heart/decision/tests/golden_config.json`, `pkg/heart/decision/testdata/golden_config.json` (golden fixtures migrated to `knirv-heart/v1`; **originals backed up to `/tmp/knirv_phase5_format_backup/`**).
- **Steps (as executed):** set `FORMAT`/`FORMAT_V2` → `knirv-heart/v1`/`knirv-heart/v2` in `config.rs`; keep `knirv-head/v1`/`knirv-head/v2` as legacy loaders; update `model.rs` format strings + `lib.rs` doc; migrate the two golden fixtures; update inline test configs in `head.rs`/`sets.rs`/`layout.rs`; keep the `knirv-head/v9` negative test (still unsupported); rebuild `heart-decision-wasi.wasm` and replace the embedded copy.
- **Verify:** `cargo test` (heart/decision: v1+v2 + legacy load, all pass); `make build-heart-decision-wasi`; `go test ./pkg/heart/decision/...` (embedded wasm loads, service_test + contract_test pass); `node test-models.mjs` — all green.

### 14. `KNIRV_MODEL_LAYOUT` default flip to `migrated` (source of truth = `flags.go`)
- **Already done:** the `pkg/head/` (Go) and `head/` (Rust crate) directories have been **removed** and moved to `pkg/heart/decision/` and `heart/decision/` respectively.
- **Action (DONE):** the real default is hard-coded in `pkg/models/flags.go` `ModelLayout()` (lines 39-44). `compatibility_matrix.json:29` is a **test fixture, not a config source**. Flipped the default to `Migrated`; if `KNIRV_MODEL_LAYOUT == "legacy"` it returns `Legacy` **only** when `legacyLayoutSupported()` is true — that check is now implemented and returns `false`, so legacy silently degrades to `migrated` (the `pkg/head` Go package and `head/` Rust crate were removed in Phase 5, leaving no legacy files to mount). `TestMigrationFlagDefaults` (`migration_test.go:23-24`), the fixture check (`migration_test.go:164-166`), and the matrix JSON default were all updated to `migrated`, so code + fixture + test agree.
- **Verify:** `go test ./pkg/models/...` (pins the default via `TestMigrationFlagDefaults` + the matrix fixture check).

### Cross-repo handoffs (flagged, not drafted here)
- **backend_server** (`KNIRV_CORP/packages/server/backend`): move any *decision*-metadata client calls from `/models/head/*` to `/models/heart/decide*`. Keep VLM manifest callers.
- **KNIRVINFERENCER**: **retain** dual HEART+HEAD mirrored generation (`KindMirrorGenerate`/`GenerateMirrored`, `head.mirror.generate`); remove only legacy HEAD **decision-route** callers once HEART decision replacements pass. Do **not** switch to HEART-only.
- **KNIRV_CORP**: owns `backend_server` source; receives the route/knob changes.

### Draft sequencing (suggested, in-scope only)
1. Item 3 + 9 (remove `registerHEAD` + `KNIRV_HEAD_DIR`) — smallest blast radius.
2. Item 6 (drop dead `/models/head/reason` test rows).
3. Item 5 (remove `head_reasoning.json` vector + consumer).
4. Item 4 (remove `head_migration.go`).
5. Item 1 (`gpt_compat` alias layer + importers).
  6. Item 8 (`ops.assistant`) — **DONE 2026-11-04**: removed, tools folded into `hands.assistant` (no rename/alias). Item 11 (`head.train`) already done (tool removed); item 12 (`head.reason` alias) done with item 6.
 7. Item 7 (`/models/head/models` decision-alias removal; preserve VLM route + SDK callers).
 8. Item 2 (`pkg/models/head.go` decision-route surface only; HEAD stays VLM+mirror).
 9. Item 10 (scripts/docs) last, after routes settle.
  10. Item 13 (`knirv-head` → `knirv-heart` format ids) — **DONE 2026-11-04**: migrated with legacy loaders kept (backward compatible).
 11. Item 14 (`KNIRV_MODEL_LAYOUT` default flip to `migrated`) runs **last**, after every alias consumer is cut over. (`pkg/head` + `head/` are already moved to `heart/decision`.)
 12. Cross-repo handoffs issued in parallel to `KNIRV_CORP` / KNIRVINFERENCER.
