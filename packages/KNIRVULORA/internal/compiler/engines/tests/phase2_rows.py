"""
I_52 phase 2 MATCHED variant: training-row construction for the per-skill adapters.

Pure Python (no torch, no network) so it can be unit-tested on CPU (test_phase2_rows.py).

Why this exists: the unmatched per-skill adapters (phase2_multilora.py, variant "unmatched")
had no abstention rows and no scope adapter, while the phase-1 single adapter had ~1.6 %
abstention rows and covered all four skills. The owner (24a) asked for one retrain of the
per-skill adapters with the SAME abstention share as the single adapter and matched scope.

Recipe (per skill s):
  rows(s) = labelled train rows of skill s  +  the same retention rows as phase 1
            +  k(s) generic abstention rows drawn from the same six generic refusals phase 1 used
  k(s)    = round(share * n_pos / (1 - share)),  n_pos = labelled rows of s + retention rows,
            share = phase-1 single adapter's abstention share (6 / 369 = 1.63 %).
Integer counts cannot hit the share exactly for small adapters, so each adapter's realised
share is reported (k / total) in build_matched_rows' info dict.
"""
import json
import random

# Same six generic refusal prompts as grounded_spike_r2.main / grounded_spike_r3.main.
# tests/test_phase2_rows.py checks these strings still appear in grounded_spike_r2.py.
GENERIC_PROMPTS = [
    "How do I rotate logs with logrotate?",
    "How do I create a Kubernetes pod?",
    "What command lists docker containers?",
    "How do I set a git remote URL?",
    "How do I restart nginx?",
    "How do I check disk usage with du?",
]

MATCHED_SKILLS = ["debug", "check", "document", "scope"]


def abstention_count(n_pos, share):
    """Abstention rows A with A / (n_pos + A) closest to `share` (round-half-up on A)."""
    if share <= 0:
        return 0
    return int(share * n_pos / (1.0 - share) + 0.5)


def single_share(n_single_total, n_single_abstain):
    """Abstention share of the phase-1 single adapter, the target for the matched variant."""
    return n_single_abstain / n_single_total


def build_matched_rows(skill, train_rows, retain_rows, abstain_prompts, share, seed, abstain_completion):
    """Rows for one matched per-skill adapter.

    train_rows: labelled rows as dicts with keys skill, prompt, completion (all skills).
    retain_rows: [{"prompt","completion"}] retention rows (same list for every adapter).
    abstain_prompts: the generic refusal prompts (GENERIC_PROMPTS).
    Returns (rows, info). Deterministic for a given seed.
    """
    own = [{"prompt": r["prompt"], "completion": r["completion"]} for r in train_rows if r["skill"] == skill]
    base_rows = own + [dict(r) for r in retain_rows]
    n_pos = len(base_rows)
    k = min(abstention_count(n_pos, share), len(abstain_prompts))
    rng = random.Random(1000 + seed)
    picked = rng.sample(list(abstain_prompts), k)
    abst = [{"prompt": q, "completion": abstain_completion} for q in picked]
    rows = base_rows + abst
    info = {"skill": skill, "skill_rows": len(own), "retain_rows": len(retain_rows), "abstain_rows": k,
            "total_rows": len(rows), "abstain_share": round(k / len(rows), 4) if rows else 0.0}
    return rows, info


# ---------------------------------------------------------------------------------------------
# STRONGER per-skill recipe (I_52 owner decision 20a, one seed). Same per-skill labelled rows and
# retention rows as the matched variant, plus:
#   * skill-specific abstention rows: the phase-1 labelled hard negatives of THAT skill (questions
#     the skill's own passages do not answer, labelled with the same teacher method as phase 1),
#     instead of only the six generic refusals;
#   * a larger abstention share (STRONGER_SHARE) and more epochs (STRONGER_EPOCHS).
# No synthetic or augmented rows: every labelled positive is already used, and the held/probe
# questions cover all 87 passages, so there is no passage that a new synthetic row could come
# from without overlapping the held/probe passages.
STRONGER_SHARE = 0.10   # target abstention share per adapter (phase 1: 1.6 %)
STRONGER_EPOCHS = 6     # phase 1 and matched: 3 (check adapter ended at final loss 1.05 after 3)


def norm_prompt(s):
    return " ".join(s.lower().split())


def build_stronger_rows(skill, train_rows, retain_rows, negs, generic_prompts, share, seed,
                        abstain_completion, excluded_prompts=()):
    """Rows for one stronger per-skill adapter.

    negs: labelled hard negatives [{"skill","prompt","completion"|"passage"}] for every skill.
    excluded_prompts: prompts that must not be used as abstention rows (held/probe questions).
    Abstention rows = skill-specific hard negatives first (sampled without replacement, seed-fixed),
    then generic refusals only if the share needs more rows than the skill's negatives provide.
    Returns (rows, info). Deterministic for a given seed.
    """
    excl = {norm_prompt(p) for p in excluded_prompts}
    own_all = [{"prompt": r["prompt"], "completion": r["completion"]} for r in train_rows if r["skill"] == skill]
    # a labelled training row whose prompt is also a held/probe question is dropped (leak guard)
    own = [r for r in own_all if norm_prompt(r["prompt"]) not in excl]
    own_dropped = len(own_all) - len(own)
    base_rows = own + [dict(r) for r in retain_rows]
    n_pos = len(base_rows)
    k = abstention_count(n_pos, share)
    pool, seen, dropped = [], set(), 0
    for n in negs:
        if n["skill"] != skill:
            continue
        key = norm_prompt(n["prompt"])
        if key in seen:
            continue
        seen.add(key)
        if key in excl:
            dropped += 1  # a held/probe question that phase 1 also labelled as a hard negative
            continue
        pool.append(n["prompt"])
    rng = random.Random(2000 + seed)
    take = min(k, len(pool))
    picked = rng.sample(pool, take)
    rem = k - take
    generic = [q for q in generic_prompts if norm_prompt(q) not in excl]
    gen_picked = rng.sample(generic, min(rem, len(generic)))
    abst = [{"prompt": q, "completion": abstain_completion, "source": "skill_negative"} for q in picked]
    abst += [{"prompt": q, "completion": abstain_completion, "source": "generic"} for q in gen_picked]
    rows = base_rows + abst
    info = {"skill": skill, "skill_rows": len(own), "own_dropped_overlap": own_dropped,
            "retain_rows": len(retain_rows), "neg_pool": len(pool),
            "neg_dropped_overlap": dropped, "abstain_skill_negs": len(picked), "abstain_generic": len(gen_picked),
            "abstain_rows": len(abst), "total_rows": len(rows), "share_target": share,
            "abstain_share": round(len(abst) / len(rows), 4) if rows else 0.0}
    return rows, info


def drop_eval_overlap(rows, eval_prompts):
    """Leak-free training rows (I_52 owner decision 22, Q2): drop every row whose prompt is a held or
    probe question (normalised). Returns (kept, dropped) where dropped is a list of the removed prompts.
    Applied to every adapter trained in the seeds 1-2 unit, single and per-skill."""
    excl = {norm_prompt(p) for p in eval_prompts}
    kept, dropped = [], []
    for r in rows:
        if norm_prompt(r["prompt"]) in excl:
            dropped.append(r["prompt"])
        else:
            kept.append(r)
    return kept, dropped


def overlap_prompts(rows, held, probes):
    """Prompts (normalised) that appear both in `rows` and in the held or probe questions. Must be []."""
    evalset = {norm_prompt(x["prompt"]) for x in list(held) + list(probes)}
    return sorted({norm_prompt(r["prompt"]) for r in rows} & evalset)


def retention_rows(reuse_path, out_path, make):
    """Retention rows, pinned: reuse a saved JSON if reuse_path is given, else call make() and save.

    The written file is always out_path, so a later run can pass it back as reuse_path and repeat the
    exact same retention rows. Returns (rows, source) with source "reused" or "generated".
    """
    if reuse_path:
        with open(reuse_path, encoding="utf-8") as f:
            rows = json.load(f)
        if not (isinstance(rows, list) and all({"prompt", "completion"} <= set(r) for r in rows)):
            raise ValueError(f"not a retention-rows file: {reuse_path}")
        source = "reused"
    else:
        rows = make()
        source = "generated"
    if out_path:
        with open(out_path, "w", encoding="utf-8") as f:
            json.dump([{"prompt": r["prompt"], "completion": r["completion"]} for r in rows], f, indent=1,
                      ensure_ascii=False)
    return [{"prompt": r["prompt"], "completion": r["completion"]} for r in rows], source
