# Clef verification harness

Verifies Cloudflare's Clef decision-model claims and exercises the KNIRV schema
translation described in [`../Clef_KNIRV_Design.md`](../Clef_KNIRV_Design.md).

```
python3 run_banking77.py --preflight  # probe Workers AI readiness, exit 0 = ready
python3 test_contract.py              # 67 tests, no network
python3 run_banking77.py              # baseline arm + Clef arm
python3 run_bfcl.py                   # BFCL v3 function-selection arm
```

## Measured live results (partial — budget exhausted)

Live clef-flash, BANKING77, 20 uniformly-sampled items:

| Metric | Measured | Cloudflare claim |
|---|---|---|
| Accuracy | **1.0000** (20/20) | — |
| ECE | **0.0570** | not reported |
| MCE | 0.1663 | not reported |
| Brier (sum) | 0.00645 | not reported |
| Input tokens / call | 1,853 median | — |
| **Output tokens / call** | **0** | — |
| Latency p50 | **0.774 s** | 38.8 ms |
| Latency p95 | 5.836 s | — |
| Cost | $0.000167 / call | $0.09 / Mtok |

Three findings:

1. **`output_tokens` is 0 on every single live call** (20/20, plus 3 fixture calls). This is the ELM result confirmed directly on the vendor's own endpoint: the output length is constant-zero regardless of task, so decode cost does not scale with anything. It is why cost is purely `O(input_tokens)` and why latency should track prefill alone.
2. **Latency does not reproduce.** p50 0.774 s against a published 38.8 ms is a **20× gap**; p95 5.836 s. Either the published figure is measured from inside Cloudflare's network, on different hardware, or on a much smaller input — 1,853 input tokens is ~48× a typical chat prompt, so the 38.8 ms figure is unlikely to be at comparable input size. Treat 0.774 s as the realistic figure from a normal client, and re-measure before any latency budget is written.
3. **Calibration looks genuinely good** — ECE 0.057, and Brier 0.0065 across 77 classes. This is the number that matters most for KNIRV, and Cloudflare does not publish it. But **n=20 cannot establish it**; §Status below explains why a real number needs the paid plan.

### Macro-F1 caveat

The reported `macro_f1` of 0.2597 is **not** comparable to Cloudflare's 90.93. A 20-item uniform sample touches ~20 distinct classes, so ~57 of the 77 classes have zero support and contribute F1 = 0 to the average. The metric is correct; the sample cannot support it. A directly comparable macro-F1 needs the full 3,080-item test set.

## Blocked: daily free allocation exhausted

The account is on Workers AI's free tier: **10,000 neurons/day**. That was
consumed during the first testing pass:

```
HTTP 429: you have used up your daily free allocation of 10,000 neurons,
please upgrade to Cloudflare's Workers Paid plan if you would like to continue usage
```

At least 437 successful calls were made before exhaustion (20 live-smoke, 12
concurrency smoke, 5 preflight/fixtures, and ~400 in a run killed by a tool
timeout), so the **free tier affords roughly 22 neurons per clef-flash call**,
giving:

| Run | Neurons | vs daily budget |
|---|---|---|
| `--limit 400` | ~9,200 | 0.9× — fits in one day |
| `--limit 1000` | ~22,900 | 2.3× — needs 2–3 days |
| Full 3,080 test set | ~70,500 | 7× — needs the paid plan |

The neuron allowance resets daily. Anything larger than ~400 calls needs
**Workers Paid**. To finish verification, either enable the paid plan or run
`--limit 400` once per day for eight days.

This budget constraint independently reinforces the design doc's §6 conclusion:
a supervisor calling Clef continuously would exhaust a free allocation
**every few hours**, so hosted Clef was never viable as a production runtime.
It is viable as an offline teacher — which is what the design recommends.

## What is verified today

| Layer | State |
|---|---|
| Request/response contract | Validated against Cloudflare's **published JSON schemas**, fetched into `fixtures/` |
| KNIRV ↔ Clef translation | 61 tests, including the three rules in design doc §4.2 |
| BANKING77 data | 10,003 train / 3,080 test / 77 categories, official mirror |
| BFCL v3 data | 1,480 examples across 9 splits, canonical `gorilla-llm` dataset |
| Baseline arm | TF-IDF centroid classifier over all 3,080 test items |
| Calibration metrics | ECE / MCE / Brier / reliability, unit-tested |

## Quick reference

Credentials work and preflight passes. Live measurement is **partially complete**
— see "Measured live results" for what was measured and "BLOCKED — daily free
allocation exhausted" for what remains.

Offline runs use a deterministic stub that emits schema-valid but deliberately
uncalibrated answers, so stub results carry `"stub": true` and no stub number may
be quoted as a Clef measurement.

```bash
python3 run_banking77.py --preflight        # exit 0 = ready
python3 run_banking77.py --limit 400        # ~0.9x the daily free allocation
python3 run_bfcl.py --limit 100             # keeps inside the daily budget
```

Credentials load from `.env` automatically; existing environment variables are
never overwritten, so CI secrets win. `.env` is gitignored and its contents are
never logged.

## Baseline arm results (local, no credentials)

Full 3,080-item BANKING77 test set, TF-IDF cosine nearest-centroid:

| Metric | Value |
|---|---|
| Accuracy | 0.6883 |
| Macro-F1 | 0.6682 |
| **ECE** | **0.6320** |
| MCE | 0.7706 |
| Brier (sum) | 0.9150 |
| Runtime | 2.44 s total |

Two things worth reading off this table:

1. **ECE 0.63 is the finding.** The centroid's confidence is badly calibrated —
   it is roughly as wrong about its own confidence as it is about the task. This
   is the concrete evidence for the design doc's central argument: KNIRV cannot
   ship on accuracy alone, because every threshold in the scorecard assumes the
   reported confidence means something. A plain local index does not provide it.
2. **The reference point.** Cloudflare claims macro-F1 90.93 for clef-flash on
   this dataset. A trivial local baseline reaches 0.668, so a Clef measurement
   landing near 0.69 would mean the model adds nothing over local retrieval, and
   only a number far above that justifies the merge.

`brier_normalised` is also reported (0.0119 vs 0.0128 for a uniform prior) but
is a weak discriminator on 77 classes; the unnormalised Brier is the useful one.

## BFCL: what this harness does and does not measure

**This is NOT the official BFCL metric.** Official BFCL is AST-based exact match
over the whole call expression — argument names, types, and order — evaluated
against `possible_answer/` files. Neither that matcher nor those files are
implemented here, so every BFCL number is reported as
`function_selection_accuracy_proxy` and `"official_bfcl_metric_implemented": false`.

What it measures instead: a `choice` question over each item's **own** candidate
functions plus an explicit no-function option. That is the shape of decision
`supervisorrouting/router.go:92` makes today with Claude Haiku, at a scale with
unambiguous ground truth.

**The irrelevance split is the important one.** `BFCL_v3_irrelevance` (240 items)
is where the correct answer is to call nothing — which is KNIRV's `abstain`
action, the supervisor's most consequential decision. Cloudflare's own table
shows Clef-flash weakest on When2Call (65.58 vs Jev 80.97), i.e. "should a tool
be called at all". The harness reports that subset separately as
`no_function_subset`. **If that number is poor, the merge's central assumption
is wrong**, and design doc §8 already says why the evidence gate must stay
deterministic regardless.

Per-item option sets are 2 (`simple`, `parallel`) or 4 (`multiple`,
`parallel_multiple`). Options are per-item, not a split-wide union — see the
note in `run_bfcl.run_split`.

**Version caveat:** Cloudflare's published BFCL figure does not state a BFCL
version. This is v3. If their number is v1, it is not reproducible here.

## Data provenance

```bash
# BANKING77 — official mirror
B=https://raw.githubusercontent.com/PolyAI-LDN/task-specific-datasets/master/banking_data
for f in categories.json train.csv test.csv; do curl -sSfLO "$B/$f"; done

# BFCL v3 — canonical leaderboard dataset (147k downloads)
H=https://huggingface.co/datasets/gorilla-llm/Berkeley-Function-Calling-Leaderboard/resolve/main
for s in simple multiple parallel parallel_multiple irrelevance; do
  curl -sSfL -o "BFCL_v3_$s.json" "$H/BFCL_v3_$s.json"
done

# Clef's own published schemas
D=https://developers.cloudflare.com/workers-ai/models/clef-flash
curl -sSfLO "$D/schema-input.json"  # fixtures/clef-schema-input.json
curl -sSfLO "$D/schema-output.json" # fixtures/clef-schema-output.json
```

BFCL files are **JSONL despite the `.json` extension**, and their `question`
field is `list[list[{role, content}]]` — double-nested even in single-turn
splits. Both are handled in `run_bfcl.py` and pinned by tests.

BANKING77's `test.csv` is **ordered by category**: the first 200 rows contain only
5 of the 77 classes. `--limit` therefore takes a uniform seeded sample, never a
prefix slice. Slicing would evaluate a 5-class problem and report it as 77-way.

## Files

| File | Purpose |
|---|---|
| `clef_contract.py` | Clef request/response contract, transcribed from the published schemas |
| `knirv_schema.py` | KNIRV ↔ Clef translation; the piece that would ship |
| `clef_client.py` | Workers AI transport, `.env` loader, 4-stage preflight, credential-free stub |
| `metrics.py` | ECE / MCE / Brier / macro-F1 / reliability / abstention curve |
| `run_banking77.py` | 77-way calibration probe + local baseline arm |
| `run_bfcl.py` | BFCL v3 function-selection probe |
| `test_contract.py` | 67 tests: contract, schemas, translation, metrics, loaders, dotenv, preflight |
| `fixtures/` | Cloudflare's published input/output schemas |
| `data/bfcl/` | BFCL v3 splits (JSONL) |
| `results/` | JSON reports (`NaN` converted to `null` so files are valid JSON) |

## Contract notes worth keeping

Transcribing the schemas by hand produced six defects that only the fetched
schemas exposed, so **read them before changing `clef_contract.py`**:

- `choice.criteria` descriptions may be **string, object, array, or `null`** —
  not a required string. Options: **2..255**.
- `score.criteria` allows **2..10** levels, items may be string/object/array.
- `score` answers **require `legend`**; a bare `{score, probabilities,
  confidence}` is invalid.
- `noul` answers carry **only** `noul` (P(yes)) — no `confidence`. KNIRV
  derives `max(P, 1-P)` and records that derivation.
- Top-level `model`, `answers`, **and `usage`** are all required; `usage` is
  what makes the design doc's cost model measured rather than guessed.
- `model` matches `^\s*(clef|clef-flash)\s*$` — surrounding whitespace is legal.

## Adding the official BFCL metric

Not done. It needs `possible_answer/BFCL_v3_*.json` plus BFCL's AST matcher, and
argument extraction — which is `noul`/`score` territory rather than `choice`.
Until that lands, treat the proxy as the only number this harness produces.