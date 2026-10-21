# 5_HEAD_TRAINER

Trains the `knirv-head` Hierarchical Reasoning Model (arena_fixes.md §2.6). It links the
same crate the arena runs in the browser (`KNIRVMODELS/head`), so training and
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
  never by row**, so near-duplicates cannot leak into evaluation. With fewer than `--min-groups` (5) distinct
  groups the split is by time instead (`ts`, then file order): oldest 80% train, then val, newest test.
  Validation and held-out data are never the training set: with none, the run trains but is refused
  (`no_disjoint_heldout`) and tau is not fitted.
* `soft` (teacher distributions per head) and `teacher_plan` are used on the **train split only**; the
  ground truth in `labels` is what validation, tau and the gate see.
* Labels are keyed by option **name**, so a model with a different option order still reads them correctly.
* Inputs longer than the preset width are cut; the run fails if more than `--max-truncated` (5%) are.
* A head whose answer is spelled out in its input (`<head>:<option>`, >= 90% of rows) is refused as label
  leakage (`--allow-leaky-heads` overrides). `convert` therefore does not emit `intent` and `risk`
  (fields of the prompt) unless `--emit-input-heads`.
* Tokens are `byte-v1` (`knirv_head::tokenizer`): BOS + bytes, PAD-filled to the preset width.
  `plan` is supervised position-by-position (plan bytes, EOS, PAD); records without one skip the plan loss.
* The input must be redacted before it reaches this stage. The trainer does not redact.

## As a pipeline stage

`KNIRVMODELS/pkg/head/cycle` runs this trainer automatically after pipeline batches (and on demand:
`curl --unix-socket /var/run/knirvmodels.sock -X POST -d '{"force":true}' http://x/api/v1/head/train`, status at
`GET /api/v1/head/status`). It reads Bronze from `<appdata>/knirvmodels/supervision/bronze/*.jsonl`
(`KNIRV_HEAD_BRONZE_DIR`), and uses `<appdata>/knirvmodels/head-work/teacher.jsonl` as `--soft` if present.
Knobs: `KNIRV_HEAD_AUTOTRAIN`, `_PRESET`, `_EPOCHS`, `_MIN_NEW`, `_MIN_GAP_HOURS`, `_POLICY_HASH`.
`make build-head-trainer build-head-wasi` (part of `build-pipeline-all`, so of KNIRVSERVER `make binary`) embeds it.

## Converting supervisor episodes

```bash
./target/release/head-trainer convert --episodes gold.jsonl --out train.jsonl [--soft clef-teacher.jsonl] [--soft-heads intent,next_phase,risk,agent]
```

Only episodes with a non-abstaining decision and `outcome.verified && !human_override` become decision
labels (`action`, `next_phase`, `agent`); a shadow advisor's own advice is never a label. Unverified and
overridden episodes are kept with their recorded decision masked, and a human's `outcome.override_action`
(when recorded) becomes the `action` label. Teacher distributions go under `soft`, never over `labels`.
An empty `repo_family` becomes a group of its own per episode. The prompt (`phase intent risk budget agents evidence request`,
secrets re-redacted) is byte-identical to the server's `buildAdvisePrompt`; both sides assert the same
vector. A head with a single observed value (e.g. `next_phase` in a small set) is skipped with a warning.

## What training does

Deep supervision, one-step gradient per cycle: every high-level cycle is supervised with the carried
state detached, so each halting point is a trained state. Loss = plan cross-entropy + per-head
cross-entropy/Brier + a halt loss (stop when this cycle's plan is already right). Initial weights are seeded
from `--seed` (candle's CPU RNG cannot be seeded). Use `--lr ≤ 1e-3`; 3e-3 diverged in testing.

## Gate and outputs

Training fine-tunes the promoted model when its architecture and heads match (`--init-from auto`, lr scaled
by `--ft-lr-scale`); `--init-from none` trains from random init. The halt head is trained from decision
correctness (every supervised head already right at this cycle), so it learns without any plan.
Norm gains, biases and embeddings are not weight-decayed; warmup is clamped to a tenth of the run.

After training, `tau` is fitted on the validation split: the smallest threshold at which every choice head
reaches `--target-acc` selective accuracy (abstain ⇔ max(p) < tau). Held-out metrics are computed at batch 1
(per-example halting, like serving) with that tau: coverage, selective accuracy and an abstain-aware
false-allow rate. Promotion:

* needs at least `--min-heldout` (30) held-out rows;
* first model: only if the tau target was met;
* later models: scored against the current `model_latest` row by row on the heads and options both have;
  promoted only if the lower 95% bound of a paired-bootstrap score gain is above 0, the mean gain is at least
  `--min-margin`, and the false-allow rate does not rise.

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
