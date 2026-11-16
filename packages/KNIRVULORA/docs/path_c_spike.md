# Task 5.2 spike: Path C (context distillation → native LoRA) — results 2026-11-02

Engine: `internal/compiler/engines/path_c_distill.py` (`Engine.RunPathC` in `engine.go`). Harness: `engines/tests/spike_path_c.py`. Raw reports: `engines/tests/results/spike_5_2_run{1,2}.json`.
Host: RTX 3060 12 GB, torch 2.7.1+cu128, transformers 4.53.3, peft 0.16.0, base `Qwen/Qwen2.5-0.5B-Instruct` (bf16 base, fp32 LoRA r=16 α=32, all of q/k/v/o/gate/up/down). A 48–80-step run takes ~1–3 min.

## What was tested
Invented, checkable skills (a command + flag value + status code that the base cannot know). Clusters of 3, 4 and 5 skills. Training prompts and held-out prompts use **disjoint phrasings**. Arms: base; base + the cluster's skill text in the prompt; cluster adapter (no skill text). Single-skill adapters give the I5 baseline.

**Stand-in teacher:** completions come from a deterministic template oracle, **not** an external provider. This measures whether gradient distillation can install the facts, not teacher quality or realistic skill text. The real-teacher variant (5.4) is still required before this gate is final.

## Results
| run | recipe | cluster 3 | cluster 4 | cluster 5 | general QA (base 1.00) |
|---|---|---|---|---|---|
| 1 | 4 epochs, no retention/negatives, 4 distinct held-out prompts | members 0.75 each (single 1.0) | members 1.0 | members 0.75–1.0 | 0.70–0.80 |
| 2 | 3 epochs, +26 self-distilled retention, +12 negatives, 10 distinct held-out prompts | 0.8–0.9 (single 0.2–0.6) | 0.5–0.8 (single 0.5–0.7) | 0.4–0.8 (single 0.6–1.0) | 0.80–0.90 |

Base scored **0.0** for every member in every cluster, and so did base + skill text in the prompt (the 0.5B model does not apply it), so every adapter score is a real gradient-installed capability.

Gate (≥2 of 3 clusters: every member beats base by ≥0.5 and I5 interference ≤ η=0.2): **met in run 2 (clusters of 3 and 4); cluster of 5 fails** (member s2 at 0.4, interference 0.5). Run 1 passed margin everywhere but failed η on 2 clusters by one prompt, which was a resolution problem of the 4-prompt eval, fixed in run 2.

**llama.cpp reproduction** (cluster of 4, 40 held-out prompts, f16 base GGUF + f16 LoRA GGUF via `convert_lora_to_gguf.py`, CUDA): PEFT score 0.625, llama.cpp 0.625, **92.5 % exact string match**; adapter scale 0 → score 0.0.

## Findings that are not passes
1. **Regression (I4) is not clean.** General QA fell from 1.00 to 0.70–0.90 in every adapter, even with retention data. A real I4 threshold has to be set, and retention data needs tuning before 5.5.
2. **Over-generalisation.** On an unknown error id the adapter emits a member's command/code (hallucinated fix). Adding negatives made some *known* errors answer "no verified resolution" (3 of the first 40 llama.cpp outputs). Negative share must be tuned; the leakage and abstention behaviour needs its own test in 5.5.
3. **Cluster-size ceiling at this recipe: 4 skills** (rank 16, 3–4 epochs, 32 examples/skill, 0.5B). Size 5 degrades. Per D-K1 this is the data point for the cap; a higher rank or more data per skill may raise it and has not been tried.
4. **Seed variance not measured** (one seed, 10 prompts/skill). Margins near 0.5 (cluster 4: s1 0.5, s0 0.6) are inside noise.

## Verdict
Kill switch **not triggered**: gradient SFT installs skill facts that survive PEFT→GGUF→llama.cpp. Do **not** treat 5.2 as closed: the teacher is a stand-in, I4/leakage are unresolved, and results are single-seed. Remaining before 5.3: real-teacher run, 3 seeds, tuned retention/negatives, rank sweep for the ceiling.


---

# Update 2026-11-02 (later): hybrid teacher, 3 seeds

Builder: `engines/tests/teacher_corpus.py`. Reports: `engines/tests/results/spike_5_2_hybrid_seed{0,1,2}.json`; corpus metadata (hashes, accept/drop counts, no text) in `results/corpus_summaries/`.

## Teacher policy tested (one backend per pass, never two at once)
1. **Local** Qwen2.5-7B Q3_K_M on the RTX 3060 (`llama-server`, `-fa on`, ctx 4096), cluster skills in the system prompt, temperature 0.7.
2. **Deterministic gate** (no model): the answer must contain the target skill's exact command, flag value and status code and none of another member's command/code.
3. **Remote escalation** (optional): only gate failures go to Workers AI `llama-3.3-70b-instruct-fp8-fast`, **after the local server has been shut down**. Every record carries `teacher_id`.

**Outcome:** 1,152 of 1,152 records passed the gate on the local teacher, so the remote pass recovered 0 and sent nothing out. Escalation is implemented and unexercised in this regime (invented, short fixture skills). Clef (`@cf/cloudflare/clef*`) cannot be a text teacher: it takes `state` + typed `questions` and returns option probabilities.

## Results (3 seeds × clusters of 3, 4, 5; r=16, 3 epochs; 10 distinct held-out prompts per member)
| | seed 0 | seed 1 | seed 2 |
|---|---|---|---|
| clusters passing (margin 0.5, η 0.2) | 3/3 | 3/3 | 3/3 |
| cluster-adapter score per member (base = 0.0) | min 0.9–1.0 | min 0.8–1.0 | min 1.0 |
| max interference vs single-skill adapters | ≤ 0.1 | ≤ 0.0 | ≤ 0.0 |
| general QA after training (base 1.00) | 0.8, 0.8, 0.8 | 0.8, 0.8, 0.8 | 0.8, 0.8, 0.9 |

The earlier failure at 5 skills did not recur. **The recipe differs from run 2 in three ways at once** (teacher-written answers with sampling diversity instead of 4 fixed templates, 6 negatives instead of 12, a gated corpus), so the improvement cannot be attributed to any one of them, and the ceiling is now "≥ 5 at this recipe", not measured above 5.

## Still open (not fixed by the better teacher)
- **I4 regression:** general QA is 0.8–0.9 in all nine adapters. Retention data did not close it.
- **Hallucinated fix on an unknown error id:** the leak probe still emits a member's command in all nine clusters. The negatives (6) do not prevent it. This needs its own gate in 5.5 and a different mitigation (more/harder negatives, lower rank, fewer epochs, or an abstention check).
- The task set is still invented, short and single-fact. Real `skill.md` files are untested with this teacher.


---

# Update 2026-11-02 (evening): real skill documents

Harness: `engines/tests/real_skill_spike.py` with fact sheets `real_skills.py` (KNIRVAGENT workspace skills) and `jsm_skills.py` (https://github.com/jsmastery-pro/skills, MIT, commit 43b69e4, cloned to scratch and only read). Reports: `results/real_agent_skills_report.json`, `results/real_jsm_skills_report.json`.

**Method.** Facts are copied by hand from each document (exact commands, paths, env vars, ladders); an answer counts when it contains the fact's tokens (case-insensitive). Training questions use two topic phrasings × 8 frames per fact; held-out questions use a third phrasing × 4 different frames. The local 7B reads **all** documents of the cluster in its prompt. Single seed for labeling, 3 training seeds. Arms: base, base + docs in prompt (0.5B), cluster adapter, single-skill adapters, and the teacher itself (ceiling).

| set | skills | records (accepted / dropped) | base | 0.5B + docs in prompt | cluster adapter (3 seeds) | teacher ceiling |
|---|---|---|---|---|---|---|
| agent skills | github, tmux, weather, summarize, hardware | 462 / 18 | 0.0 | 0.08–0.42 | **0.56–0.71** mean (min 0.50) | 0.62–0.92 |
| jsmastery | debug, check, document, scope | 283 / 101 | 0.0–0.04 | 0.12–0.25 | **0.40–0.56** mean (min 0.33) | 0.54–0.75 |

Excluded for size: `skill-creator` (18 KB) from the first set; `architect`, `audit`, `develop`, `sync`, `test` (13–28 KB each) from the second.

## What it shows
1. **Real documents are much harder than the fixtures.** The fixtures reached 0.8–1.0; here the adapter reaches 0.4–0.7, roughly 60–80 % of what the 7B teacher itself achieves with the documents in context. The teacher is the ceiling, and it is only 0.54–0.92.
2. **The adapter still beats pasting the documents into the small model's prompt** by roughly 2–4× (0.5B + docs: 0.08–0.42), which is the point of distilling.
3. **Gate status on real documents is mixed.** With the margin of 0.5 over base: the agent-skill set passes (min 0.50, borderline); the jsmastery set **does not** (debug 0.46, check 0.44, document 0.40 on average). It was trained on only 283 records because 101 were dropped (see next section), so this is partly a data-quantity result.
4. **No harmful interference.** Cluster adapters match or beat single-skill adapters (on the second set the singles are worse: 0.21–0.62), consistent with small per-skill data.
5. **Unresolved, unchanged:** general QA 1.0 → 0.7–0.9; on off-topic questions the adapter writes confident invented commands (e.g. `gh pr merges <n> --rep…`), 1–3 of 6 probes contained a document fact and none abstained.

## Escalation findings (important for the opt-in remote path)
- **The local teacher does fail on real documents:** 78/480 (agent set) and ~125/384 (jsmastery) records failed the fact gate, unlike 0 of 1,152 on fixtures. Escalation is therefore needed, not a corner case.
- **Remote recovery:** agent set 60 of 78 recovered (Workers AI `llama-3.3-70b`, ~3k-token prompts). **The Workers AI free tier then ran out** ("daily free allocation of 10,000 neurons", HTTP 429), so the jsmastery pass (≈8k-token prompts) recovered only 24 and 89 calls failed. Each record resends the whole cluster in the prompt, which is what burns the allocation.
- **Consequences for 5.4:** (a) the escalation must be quota-aware and resumable (record which items were not attempted, do not count 429s as teacher failures); (b) send only the target skill plus its nearest neighbours instead of all documents (the proposal's §4.3 already says this), or use a paid plan / another provider; (c) a corpus with 26 % dropped records should be reported as such and not silently trained on.
- **Process bug fixed:** the base+docs scoring left ~8k-token KV cache allocated and the next training run ran out of GPU memory; the harness now frees the cache and scores that arm in batches of 2.


---

# Update 2026-11-02 (night): derived fact gates, focused context, corpus floor

Code: `engines/tests/{derive_facts.py,test_derive_facts.py,grounded_spike.py}` (unit tests: `python3 -m unittest test_derive_facts`, 5 pass). Reports: `results/grounded_jsm_report.json`, `results/grounded_jsm_corpus_report.json`. Skill set: the four jsmastery workflow skills, 87 passages.

## What was built (E1, E2, E4, E5 from PendingImplementations 5.2a)
- **E5 derived gates.** No hand-written facts. Each passage's facts are its literal spans (inline code, flags, paths, ENV_VARS, file names). An answer is accepted if it reproduces enough of them (**recall**) and every span it writes exists somewhere in the cluster documents (**grounded**). The same two checks score held-out answers, so "invented command" is now a measured rate.
- **E2 focused context.** The local teacher sees only the target passage plus its 2 nearest passages by TF-IDF (about 1k tokens, not 8k). Questions are generated by the teacher: neutral wording for training, casual "don't reuse the passage's words" wording for held-out, and "same topic but NOT answered here" for hard negatives.
- **E4 corpus floor.** The run writes a report and refuses to train below a per-skill floor or an acceptance rate. It did refuse twice (see below).
- **E1 quota handling.** A 429 stops the remote pass and the untried items are reported as `not_attempted`, never as teacher failures. Verified on a real 429 (Workers AI daily allocation still exhausted: 76 items not attempted).

## Corpus outcome
Strict labeling (answer must reproduce 2 spans): 152 of 428 failed, the floor refused (`debug` 31 < 40). Relaxed labeling (≥1 span, still fully grounded): **355 accepted, 76 failed** (66 recall, 10 invented spans), 82 % acceptance; the floor was lowered from 40 to 30 per skill because `debug` has only 13 passages (the floor was never calibrated; this is a judgement call, not a measured threshold). Teacher ceiling on held-out questions (the 7B with the focused context): strict 0.36 (debug) to 0.78 (check), relaxed 0.49 to 0.88.

## Student results (3 seeds, 261 held-out questions, 79 hard-negative probes)
| | base | 0.5B + all docs in prompt (60 q) | adapter `plain` | adapter `hardneg` |
|---|---|---|---|---|
| good (≥2 spans AND grounded) | 0.14 | 0.17 | **0.25** (0.23–0.26) | 0.15 (0.13–0.17) |
| good, relaxed (≥1 span AND grounded) | 0.33 | n/a | **0.41** | 0.24 |
| answers containing an invented span | 0.13 | n/a | **0.23–0.31** | 0.11–0.17 |
| probes: invented span / abstained | 0.06 / 0 | n/a | **0.20–0.27 / 0** | 0.03–0.09 / 0.65–0.71 |
| abstains on questions it should answer | 0 | n/a | 0.01–0.03 | **0.44–0.52** |
| general QA (base 1.0) | | | 0.8–0.9 | 0.7–0.8 |

## Reading
1. **The earlier "0.4–0.7" was largely an artefact of easy, hand-picked tokens.** With derived multi-span, grounded gates the adapter improves on base only modestly (0.14 → 0.25 strict; 0.33 → 0.41 relaxed) and sits well below its teacher (0.36–0.78). It beats the 0.5B-with-all-docs control, but only just.
2. **Training on grounded answers did not stop invented commands; it increased them** (base 13 % → 23–31 % of answers, probes 6 % → 20–27 %). A 0.5B student learns the style of the documents faster than their facts.
3. **Hard-negative abstention trades one failure for another.** It cuts invented spans on probes to 3–9 %, but the model then refuses 44–52 % of questions it should answer, and answer quality falls below the plain adapter. At 166 negatives against 355 positives the mix is too abstention-heavy; a smaller share was not tried.
4. **Metric caveat.** "Good" fails any answer that writes a code span not literally in the documents, so legitimate paraphrased examples also count as ungrounded (base itself scores 13 %). Treat absolute numbers as pessimistic and the base-vs-adapter differences as the signal.
5. **Not tried:** a larger student (1.5B or 3B fits 12 GB for LoRA), a higher rank or more epochs, a smaller abstention share, or a verifier head that refuses only when retrieval finds nothing.

## Verdict for the process-style skills
Path C with a 0.5B student does not turn long prose workflow skills into reliable behaviour. It works for short, factual or procedural skills (the fixtures and the agent skills) and should not be used for the long prose ones without a bigger student and a retrieval-grounded abstention mechanism. The 5.2 kill switch is therefore **not cleared for prose workflow skills**.


---

# Update 2026-11-14: re-spike with 1.5B and 3B students (N12.10, phase 1)

Harness: `engines/tests/grounded_spike_r2.py` (new; imports `grounded_spike.py` for labels/gates/corpus report and `spike_path_c.py` for `Gen`/engine/regression; those files are unchanged). Pinned venv built by `scripts/setup-venv.sh` (torch 2.7.1+cu128, transformers 4.53.3, peft 0.16.0). Students: `Qwen2.5-{0.5B,1.5B,3B}-Instruct` (same family as the 0.5B run). Teacher: local `qwen2.5-7b-instruct-q3_k_m` (llama.cpp, CUDA); no remote calls, escalation off. Corpus: the same four jsmastery workflow skills (`debug, check, document, scope`, commit 43b69e4), 30 passages per skill cap, 87 passages.

## Setup changed from the 2026-11-02 run
| | 2026-11-02 run | this run |
|---|---|---|
| student | 0.5B | 0.5B, 1.5B, 3B |
| labels | 355 accepted, 76 failed (relaxed) | relabelled with the same teacher: 337 accepted, 94 dropped (78.2 %). Labels are shared by all students in this run. |
| abstention rows (share of training rows) | plain: 6 generic (~1.6 %); hardneg: 172/553 (**31 %**) | plain: 6 generic (**1.6 %**); `rg`: retrieval-grounded negatives only (**3.5 %**); `hn10`: all hard negatives capped at **10 %** |
| retrieval-grounded abstention | none | abstain only when best TF-IDF cosine to any passage in the 87-passage corpus < tau = 0.177 (5th percentile of support over answerable training questions). Negatives kept only if below tau. |
| seeds | 0, 1, 2 | **0 only** (scaled down, see below) |
| eval batch | 16 | 48 (new runs; the 0.5B control was also re-run under the new protocol) |

Retrieval-grounded negatives are rare: of 167 hard-negative candidates only **7** fall below tau (median answerable support is 0.31), so `rg` is almost `plain` plus 7 abstentions. The 3.5 % is the arithmetic 13 / 376 (6 generic + 7 grounded abstention rows out of 369 + 7 rows). The 10 % cap would allow 41 abstention rows (35 beyond the generic ones), so the pool of 7 is the binding limit, not the cap. A 10 % share needs the ungrounded negatives, which is why `hn10` exists as a separate arm.

## Results (seed 0; held-out = 261 answerable questions; probes = 80 hard-negative questions; base = no adapter)
Margin = adapter minus base on the same questions. "invented" = answer writes a span not in the documents (held / probe). "gate" = retrieval-grounded abstention applied at inference.

| student | arm | abstention rows | good (strict) | margin vs base | good (relaxed) | invented (held) | invented (probe) | abstains on known | gate false refusal | I4 general QA |
|---|---|---|---|---|---|---|---|---|---|---|
| 0.5B | base | 0 | 0.134 | — | 0.261 | 0.149 | 0.075 | 0 | — | 1.0 |
| 0.5B | plain | 1.6 % | 0.211 | +0.077 | 0.345 | 0.268 | 0.312 | 0.077 | 0.046 | 0.8 |
| 0.5B | rg | 3.5 % | 0.222 | +0.088 | 0.364 | 0.257 | 0.250 | 0.100 | 0.046 | 0.9 |
| 0.5B | hn10 | 10.2 % | 0.172 | +0.038 | 0.314 | 0.157 | 0.175 | 0.241 | 0.046 | 0.7 |
| 1.5B | base | 0 | 0.111 | — | 0.234 | 0.169 | 0.125 | 0 | — | 1.0 |
| 1.5B | plain | 1.6 % | 0.226 | +0.115 | 0.356 | 0.287 | 0.263 | 0.080 | 0.046 | 1.0 |
| 1.5B | rg | 3.5 % | 0.207 | +0.096 | 0.326 | 0.238 | 0.300 | 0.142 | 0.046 | 1.0 |
| 1.5B | hn10 | 10.2 % | 0.195 | +0.084 | 0.318 | 0.234 | 0.300 | 0.188 | 0.046 | 1.0 |
| 3B | base | 0 | 0.126 | — | 0.314 | 0.111 | 0.113 | 0 | — | 1.0 |
| 3B | plain | 1.6 % | 0.276 | +0.150 | 0.398 | 0.230 | 0.212 | 0.065 | 0.046 | 0.9 |
| 3B | rg | 3.5 % | 0.245 | +0.119 | 0.375 | 0.230 | 0.287 | 0.107 | 0.046 | 1.0 |
| 3B | hn10 | 10.2 % | 0.253 | +0.127 | 0.360 | 0.192 | 0.250 | 0.203 | 0.046 | 0.9 |

Gate columns (not in the table, all in `run_A/report.json` under `gate_arm`): with the retrieval gate, strict good falls by 0.01–0.02 in every arm (the 4.6 % false refusals), and **probe-invented rate is unchanged in every arm**. The gate only abstains where retrieval support is below tau, and most hard-negative probes have support above tau, so the gate does not catch the invented spans the adapter writes on them.

Comparison with the 2026-11-02 0.5B run (3 seeds, older labels): plain good 0.25 (0.23–0.26) vs base 0.14, invented 0.23–0.31; hardneg (31 % share) good 0.15, abstains on known 0.44–0.52, probe-invented 0.03–0.09. The new 0.5B plain/hn10 numbers reproduce the same pattern at the same scale: the 10 % share cuts invented spans (0.157 held) but abstention on answerable questions rises (0.24), though less than the 31 % run (0.44–0.52).

Wall time (seed 0, per run, train + eval; recomputed 2026-11-14 from `run_A/report.json` `train_wall_s` / `eval_wall_s`): 0.5B ≈ 3.1–3.4 min train + 2.2–2.5 min eval; 1.5B ≈ 3.4–4.7 min train + 2.7–3.6 min eval; 3B ≈ 5.4–7.4 min train + 4.0–4.9 min eval. The nine runs sum to 71.2 min. The per-student `wall_s` in the JSON are block times overwritten by resumes and are not arm totals. Crashed or resumed segments are not timed. Labelling pass wall time was not recorded separately.

**Correction 2026-11-14 (wall time).** An earlier version of this section gave 0.5B training as ≈ 5 min. The per-run figures above are the correct ones.

## What it shows
1. **Scale helps the margin, not the invention rate.** Strict margin over base grows 0.077 → 0.115 → 0.150 (0.5B → 1.5B → 3B). The invented-span rate on held questions stays 1.5–2× base at every size (0.23–0.29 vs 0.11–0.17). Bigger students learn the facts faster; they do not learn to stop inventing.
2. **Reduced abstention share trades inventions for refusals.** Going from 1.6 % to 10 % abstention rows lowers invented spans on held questions (0.5B: 0.268 → 0.157; 1.5B: 0.287 → 0.234; 3B: 0.230 → 0.192) at a cost of 0.11–0.16 more abstention on answerable questions (0.5B: 0.08 → 0.24; 1.5B: 0.08 → 0.19; 3B: 0.07 → 0.20). The 31 % share from 2026-11-02 was worse on refusals.
3. **Retrieval-grounded abstention does not do what it was meant to.** Grounded negatives are too few to train on (7 of 167), and the inference gate (tau 0.177) refuses 4.6 % of answerable questions while leaving the probe invented-span rate unchanged.
4. **I4 is not clean.** General QA on 10 questions: 0.8–1.0 vs base 1.0 (single seed, 10 items; noisy).

## Recommendation (go/no-go for 5.3–5.8)
**No-go for prose workflow skills as single adapters at this recipe, including at 3B.** The best arm (3B plain, +0.15 strict margin) still invents spans at about twice the base rate, and the arms that reduce inventions (hn10) pay with about 20 % refusals on answerable questions. Retrieval-grounded abstention, as built here, does not catch the failures. Per the owner's order, this points to **phase 2** (one small adapter per skill, several loaded together via llama.cpp multi-LoRA, see `Multi-LoRA-Loading.md`). Phase 2 has not been started here.

Caveat on confidence: single seed for the new arms. The 2026-11-02 0.5B runs suggest a seed-to-seed range of about 0.03 on strict good (plain: 0.23–0.26 over three seeds), so the margins above are larger than that spread, but the invented-span and refusal differences between arms have not been replicated.

## Not done (scaled down)
- Seeds 1 and 2 for the new arms (GPU budget; the remaining 18 runs at the observed 5–12 min each would take roughly 3–4 h). 
- The 2026-11-02 label set was not reused (not saved); relabelling produced 337 accepted vs 355.
- No `base+docs` control for 1.5B/3B (the 0.5B control was in the earlier run only).
- Escalation (remote teacher) not exercised, by design (brief: no remote calls).

## Update 2026-11-14 (seeds phase 2): span verifier, 3 seeds, 3B

Owner decisions of the same day: replace the TF-IDF abstention support with a span verifier for the
retrieval-grounded arm, run seeds 1 and 2 for the 3B plain and rg arms, decide go/no-go from 3 seeds.

**Span verifier (replaces the TF-IDF tau gate at inference).** `tests/span_verifier.py`. The retrieved
passage is the top-1 TF-IDF passage over the whole 87-passage corpus (all four skills, not a per-skill cluster) for the question. An answer is supported only if every
literal span it writes (the `derive_facts.spans` extraction) appears in that passage. Normalisation: NFKC,
casefold, curly quotes and backticks to ASCII, whitespace collapsed; span edges and trailing `.,;:!?)`
stripped; a span matches on token boundaries (no `[0-9a-z_]` on either side when it starts with a word
character). An answer with no spans is vacuously supported; an empty answer is not. Unit tests:
`tests/test_span_verifier.py` (14 tests, all pass). The rg training rows are unchanged (the TF-IDF filter still
chooses which hard negatives are kept for training), so the arms are the same as in the seed-0 run.

**Harness.** `tests/grounded_spike_r3.py` (new; reuses r2 metrics, labels and recipe). Labels: the seed-0
run's `labels.json`. Seed 0 of both arms was not retrained: the run_A adapters were re-evaluated with the
span verifier (inference-only change, identical rows and seed). Seeds 1–2 were trained. First launch hit a CUDA
out-of-memory in the training subprocess at plain seed 1 (the 3B training process alone needs about 10.4 GiB
of the 11.6 GiB card); the remaining runs were relaunched with `PYTORCH_CUDA_ALLOC_CONF=expandable_segments:True`.
Base (no adapter) reproduces the seed-0 run: good 0.126, invented 0.111.

**Results, 3B, 261 held answerable questions, 80 hard-negative probes.** Mean over seeds 0–2; "±" and "spread" are max − min across the three seeds, not a standard deviation.

| arm | good (ungated) | good (span gate) | invented held (ungated) | false refusal (gate) | abstain on known (gate) | probe invented (ungated) | probe invented (gate) | I4 (general QA, 10) | wall s / seed |
|---|---|---|---|---|---|---|---|---|---|
| base | 0.126 | 0.100 | 0.111 | 0.169 | – | 0.113 | – | 1.0 | – |
| plain | 0.272 ± 0.008 | 0.149 ± 0.015 | 0.222 ± 0.038 | 0.563 ± 0.038 | 0.604 ± 0.073 | 0.221 ± 0.050 | 0.004 ± 0.013 | 0.9 / 1.0 / 1.0 | 462 (seed 0: eval only, 290) |
| rg (span gate) | 0.253 ± 0.054 | 0.142 ± 0.042 | 0.217 ± 0.069 | 0.520 ± 0.077 | 0.627 ± 0.054 | 0.242 ± 0.075 | 0.000 | 1.0 / 0.9 / 1.0 | 447 (seed 0: eval only, 258) |

Margins vs base (good, ungated base 0.126): plain +0.146 ungated, +0.023 gated; rg +0.126 ungated, +0.015
gated (gated base is 0.100, so gated margins vs gated base are +0.050 plain, +0.042 rg). Span-gated answers
never write a span outside their retrieved passage (`invented_passage_held` = 0.000 by construction).

**Why the gate refuses so much.** The top-1 TF-IDF passage is the question's source passage for only 60 %
of the 261 held questions. Correct answers whose spans sit in a passage ranked second or lower are refused.
So the verifier's false-refusal rate is mostly a retrieval error, and a top-1 span check cannot separate
"invented" from "retrieved elsewhere". Verifying against top-k passages is the obvious next variant; it was
not run here because the owner specified the span check against the retrieved passage.

**Go/no-go: NO-GO (confirmed on 3 seeds).** The span gate removes almost all probe inventions, but it
refuses about half of the answerable questions (false refusal 0.52–0.56, base gate 0.17), and the
gated good score (0.14–0.15) is only +0.02 over the base. Ungated, the arms reach 0.25–0.27 good but
write invented spans on about twice the base rate (0.22 vs 0.11). Neither setting clears 5.2 for prose workflow
skills. Seed spread on good is 0.01–0.05, so the ungated margin is well above the noise; the gated margin is
at the noise level.

**Phase 2 (multi-LoRA) started on the no-go.** Result in the section "Phase 2 result (2026-11-14)" below.

### Phase 2 result (2026-11-14): one adapter per skill, multi-LoRA in llama.cpp, 3B

Scope: three per-skill adapters (debug, check, document), trained on the 3B student with the same LoRA recipe,
seed 0, rows = that skill's teacher-labelled passages + the 26 retention rows (no abstention rows). `scope` has
no adapter and is excluded from the phase-2 evaluation (the phase-1 single arm was trained on it; see Confound below). Held-out: 171 answerable questions on those three skills; 53 probes.
Served from one llama.cpp server (CUDA build, 37/37 layers on GPU) with four adapters preloaded
(`--lora-init-without-apply`): ids 0–2 per-skill, id 3 the phase-1 all-skill plain adapter (seed 0). Each request
sets every id's scale explicitly. Router: the top-1 TF-IDF passage's skill (accuracy 0.830 on held).

| arm | good (ungated) | invented held (ungated) | good (span gate) | false refusal (gate) |
|---|---|---|---|---|
| base (no adapter) | 0.117 | 0.129 | 0.088 | 0.181 |
| single (phase-1 plain, id 3) | 0.322 | 0.181 | 0.164 | 0.544 |
| routed (3 per-skill adapters) | 0.228 | 0.152 | 0.152 | 0.380 |
| oracle (gold-skill adapter) | 0.246 | 0.135 | 0.152 | 0.404 |

Gated invented-span rate on answered questions is 0.000 for every arm by construction (span verifier).
Probe refusals with the gate: base 0.21, single 0.60, routed 0.40, oracle 0.43; probe invented 0.00 in all arms.
Single seed; no spread estimate for phase 2.

**Confound.** The "single" arm is the phase-1 plain adapter: trained on all four skills (scope included) with 1.6 % abstention rows (6 generic refusals). The per-skill adapters have three skills each and no abstention rows. The single-vs-routed gap therefore mixes routing, the scope skill and abstention share. A clean test would retrain a per-skill arm with the same 1.6 % share and a scope adapter; not run.

*Correction (I_52_report_fixes2, 2026-11-14):* the scope sentence above originally read "excluded from every arm". Scope is excluded from the phase-2 evaluation and has no per-skill adapter, but the phase-1 single arm was trained on it.

Reading: the per-skill adapters invent fewer spans and are refused less than the single adapter, but
they are weaker on good answers. The oracle arm is about the same as routed, so routing error (17 %) is
not the main loss; the per-skill adapters themselves are weaker. Under the span gate, routed does not beat
the single adapter on good (0.152 vs 0.164). **Phase 2 go/no-go: no-go for prose workflow skills with this
recipe.** Caveat: one seed, three small adapters trained on 57–138 rows each.

Measurement notes: a first llama.cpp eval gave a contaminated base arm (requests that omitted adapter ids
were not running the base model; the server kept previous adapter state). Fixed by sending an explicit scale
for every loaded id; the table above is from the corrected run only. The local llama.cpp checkout was built
without CUDA (`GGML_CUDA=OFF`); evaluation used a CUDA build of the same tree in scratch.
