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
