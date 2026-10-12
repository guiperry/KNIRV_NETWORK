# 5_HEAD_TRAINER

Trains the `knirv-head` Hierarchical Reasoning Model (arena_fixes.md §2.6). It links the
same crate the arena runs in the browser (`KNIRVINFERENCER/head`), so training and
inference cannot drift.

```bash
cargo build --release
./target/release/head-trainer --data episodes.jsonl --out /var/lib/knirvserver/knirvmodels/data \
    --preset small --epochs 20 --lr 3e-4 --policy-bundle-hash <sha256>
```

Exit status: `0` promoted, `3` trained but refused by the gate, `1` error.

## Input: `head-train/v1` JSONL

```json
{"id":"ep-1","group":"repo-family","input":"redacted task text","plan":"optional plan",
 "labels":{"action":"request_review","intent":{"review":0.8,"validate":0.2},"risk":0.62}}
```

* Heads are inferred from the labels: string/object/array → `choice`, number in [0,1] → `score`.
  `action`, `intent` and `agent` use the supervisor's fixed vocabularies; other choice heads learn
  their options from the data. A soft label (object/array) is a teacher distribution (Clef), trained with KL.
* `split` is `train|val|test`; if absent it is derived from a hash of `group` — **splits are by group,
  never by row**, so near-duplicates cannot leak into evaluation.
* Tokens are `byte-v1` (`knirv_head::tokenizer`): BOS + bytes, PAD-filled to the preset width.
  `plan` is supervised position-by-position (plan bytes, EOS, PAD); records without one skip the plan loss.
* The input must be redacted before it reaches this stage. The trainer does not redact.

## As a pipeline stage

`KNIRVMODELS/pkg/hashing/head/cycle` runs this trainer automatically after pipeline batches (and on demand:
`curl --unix-socket /var/run/knirvmodels.sock -X POST -d '{"force":true}' http://x/api/v1/head/train`, status at
`GET /api/v1/head/status`). It reads Bronze from `<appdata>/knirvmodels/supervision/bronze/*.jsonl`
(`KNIRV_HEAD_BRONZE_DIR`), and uses `<appdata>/knirvmodels/head-work/teacher.jsonl` as `--soft` if present.
Knobs: `KNIRV_HEAD_AUTOTRAIN`, `_PRESET`, `_EPOCHS`, `_MIN_NEW`, `_MIN_GAP_HOURS`, `_POLICY_HASH`.
`make build-head-trainer build-head-wasi` (part of `build-pipeline-all`, so of KNIRVSERVER `make binary`) embeds it.

## Converting supervisor episodes

```bash
./target/release/head-trainer convert --episodes gold.jsonl --out train.jsonl [--soft clef-teacher.jsonl] [--soft-heads intent,next_phase,risk,agent]
```

Only episodes with a non-abstaining decision and `outcome.verified && !human_override` become labels; a
shadow advisor's own advice is never a label. The prompt (`phase intent risk budget agents evidence request`,
secrets re-redacted) is byte-identical to the server's `buildAdvisePrompt`; both sides assert the same
vector. A head with a single observed value (e.g. `next_phase` in a small set) is skipped with a warning.

## What training does

Deep supervision, one-step gradient per cycle: every high-level cycle is supervised with the carried
state detached, so each halting point is a trained state. Loss = plan cross-entropy + per-head
cross-entropy/Brier + a halt loss (stop when this cycle's plan is already right). Initial weights are seeded
from `--seed` (candle's CPU RNG cannot be seeded). Use `--lr ≤ 1e-3`; 3e-3 diverged in testing.

## Gate and outputs

After training, `tau` is fitted on the validation split: the smallest threshold at which every choice head
reaches `--target-acc` selective accuracy (abstain ⇔ max(p) < tau). The candidate is then scored on the
held-out set (`--heldout`, else the test split) **alongside the current `model_latest` on the same data**
and is promoted only if it scores strictly higher and does not raise the false-allow rate.

```
<out>/head/head_config.json  model-<version>.safetensors  model_latest -> …  metrics.json
<out>/head/versions/<version>/{head_config.json,metrics.json}      (kept even when refused)
```

`metrics.json` carries plan accuracy, per-head accuracy/Brier/ECE/MSE, false-allow rate, mean halting
steps, the tau fit, dataset digests and the gate decision.

## Not done yet

* The held-out set is graded locally against labels. Grading through KNIRVGRAPH
  (`/nrv/errors/{id}/tests/grade`) and the full `supervisor_model.md` scorecard are not wired.
* Converting error-node graded runs into `head-train/v1` (episodes are covered by `convert`; `--soft` takes teacher distributions but nothing produces them until the Clef gate, M7-0, is run live).
* Plan generation from scratch learns slowly on tiny models; decisions train much faster.
* The halt loss is a supervised simplification of the ACT Q-learning target.

## Teacher mask
`--soft` teacher distributions may override only `intent,next_phase,risk,agent` by default. `action` (which carries `abstain`) stays outcome-labelled: the M7-0 gate found clef-flash calling a tool on 25% of BFCL irrelevance items. Widen with `--soft-heads` only after `sudo scripts/clef-gate.sh --irrelevance` shows an acceptable false-call rate (see `none_curve` in `clef-harness/results/bfcl_irrelevance.json`).
