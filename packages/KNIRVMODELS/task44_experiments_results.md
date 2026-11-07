# Task 4.4 experiments — choice-set head learns the matching rule

## Status: KILL GATE PASSED (verified through the production feature path)

The choice-set head (Task 4.4) learns the matching rule and *generalises to UNSEEN*
choice strings, clearly above the base rate, on both tasks and across seeds. The result
transfers to the production path: the gate now sets `HeadConfig.set_match_feature = true`
in the trained config (previously the flag defaulted to `false`, so production `reason_sets`
passed zeros and the win did not transfer). A feature-ON model is persisted and the native
golden pins the production path.

## E2 — feature ON, learnability gate (production path)

Generator: 40 training keys vs 10 held-out keys; disjoint test topics/templates. 3000 train
steps, `tiny` preset, both tasks. Gate: `unseen_acc > unseen_base*1.5 && unseen_acc >
unseen_base + 0.2`.

| seed | task | seen acc | seen base | unseen acc | unseen base | unseen pass |
|------|------|----------|-----------|------------|-------------|-------------|
| 1    | a    | 0.952    | 0.177     | 0.972      | 0.175       | true        |
| 1    | b    | 0.995    | 0.175     | 0.997      | 0.173       | true        |
| 2    | a    | 0.955    | 0.176     | 0.975      | 0.177       | true        |
| 2    | b    | 0.995    | 0.178     | 0.998      | 0.177       | true        |
| 3    | a    | 0.955    | 0.176     | 0.975      | 0.177       | true        |
| 3    | b    | 0.995    | 0.178     | 0.998      | 0.177       | true        |

Summary (mean ± std over 3 seeds):
- task a: seen 0.954 ± 0.002 / base 0.176 ; unseen **0.974 ± 0.002** / base 0.176
- task b: seen 0.995 ± 0.000 / base 0.177 ; unseen **0.998 ± 0.001** / base 0.177

Unseen accuracy is ~5.5× the base rate on task a and ~5.6× on task b, with margin
> 0.2 on every row. All 3 seeds pass on both tasks. Re-run log: `/home/gperry/ori_knirv/e2_seeds_rerun.log`.

## E1 — negative control (feature OFF)

Without the explicit match feature the raw-text model cannot learn the rule:
unseen accuracy sits at the base rate (≈ 0.17 / 0.21). This is the honest control that
motivates the `set_match_feature` signal — the model needs the overlap feature, it does not
infer matching from raw bytes alone.

## Production-transfer verification

- `run_task_e1` sets `cfg.set_match_feature = use_feature` and trains on the feature; the flag
  is persisted in the saved config (`/home/gperry/ori_knirv/feature_on_model.safetensors.json`
  → `"set_match_feature": true`).
- Native golden `reason_sets_feature_on_golden_native` (heart/decision/tests/sets.rs) pins the
  feature-ON `reason_sets` path: finite, normalised, decisive probabilities on CPU. WASI runs the
  same Rust core, so native == WASI by construction (verified by `go test ./pkg/heart/decision/`).
- `cargo test --release --test sets` (heart/decision): 18/18 pass.

## Regression tests (Task 4.4)

heart/decision/tests/sets.rs (18 tests): `set_loss_reduction_matches_reference_and_agrees_cpu_cuda`
(`set_loss` == `-log softmax[target]`, incl. padded `-inf` rows, CPU==GPU), `set_loss_backward_agrees_cpu_cuda`
(analytic `softmax − one_hot` parity), `set_scores_agrees_cpu_cuda` (set-head forward CPU==GPU),
`softmax_handles_all_masked_rows_and_widths` (widths 1/2/5/31/32/33 + fully-`-inf` row → 0, no NaN),
`overlap_feature_definition`, `overlap_feature_handles_huge_inputs` (200k-char strings < 200 ms),
`reason_sets_golden_native`, `reason_sets_feature_on_golden_native`, plus structural tests.

trainer/tests/gpu_sanity.rs (2 tests): `gpu_backward_parity_matches_cpu` (cos > 0.95, no zero-grad
parameter), `gpu_overfit_converges` (acc > 0.9, loss < 0.05).

## Audit 4.4_prod — REQUEST CHANGES resolved

Ori's independent audit returned 8 findings; all resolved:

1. **(HIGH) production feature path** — gate now sets + persists `set_match_feature=true`; feature-ON
   native/WASI vector test added; 3-seed gate re-run through the production path (all PASS).
2. **(HIGH) test harness flagged feature** — added `set_match_feature: false` to the 4 struct-literal
   harness files so `cargo test` compiles without forcing the feature.
3. **(HIGH) stale wasm** — rebuilt `heart-decision-wasi.wasm` from final source; the embedded binary was
   stale (different SHA). Re-embedded; a fresh clean rebuild now produces the identical SHA256
   (`db5f9201…e5016`). `go test ./pkg/heart/decision/ -count=1` passes.
4. **(HIGH) doc/code mismatch** — corrected the `set_loss_from_logits` doc comment to the real
   narrow-gather NLL; padded `-inf` + CPU==GPU already covered by regression tests.
5. **(HIGH) dead set attention** — removed `SetScorer.attn` (declared, never consumed) and the unused
   padding bias in `set_scores`; dropped `2*block` from `set_param_count`. Clarified
   `HEAD_Decision_Model_Plan.md` P4 (released scorer is bilinear-only).
6. **(MEDIUM) diagnostics in prod binary** — gated `probe-check` / `probe-overfit` / `probe-gradparity` /
   `cuda-diag` + their helpers behind a new non-default `diag` cargo feature. The default build no longer
   contains them (verified: the strings are absent from the release binary); `--features diag` restores them.
7. **(MEDIUM) softmax NaN on masked rows** — `softmax_last_dim_diff` zeroes all-`-inf` rows instead of
   emitting NaN (uses `x.gt(-1e30)` + `where_cond`); covered by `softmax_handles_all_masked_rows_and_widths`.
8. **(MEDIUM) unbounded overlap input** — `overlap_feature` caps text to 256 chars before LCS so training
   and serving share the bound (O(256²)); covered by `overlap_feature_handles_huge_inputs`.

## Reproduce

```bash
# decision crate (x86, no CUDA needed — wasi bin gated to `--features wasi`)
cd packages/KNIRVMODELS/heart/decision
cargo test --release --test sets

# trainer gate through the production path (GPU; persists feature-ON model with --save)
cd packages/KNIRVMODELS/pipeline/5_HEAD_TRAINER
cargo build --release --features cuda
./target/release/heart-decision-trainer probe-e1 --task both --preset tiny --steps 3000 --seed 1
# optional, to persist the flag in the config:  ... --save model.safetensors

# wasm host contract
cd packages/KNIRVMODELS && go test ./pkg/heart/decision/ -count=1
```
