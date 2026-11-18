"""
I_52 Q35 (owner decision, Nov 15): held-out PASSAGES per skill.

Pure Python (no torch, no network) so it is unit-tested on CPU in test_phase2_heldout.py.

Why: the phase-1 single adapter and every per-skill variant were trained on labelled rows derived from
the same passages that the held and probe questions come from, so the eval measured passage
memorisation as well as generalisation (I_52 stronger report, issue 5: 252/261 held questions sat on
a trained passage). Q35 holds out whole passages per skill, retrains on the remaining passages only,
and scores on the held-out passages only.

Split (deterministic):
  for each skill s, passages(s) = all passage ids whose prefix is s (e.g. "debug#15" -> "debug");
  sort the ids, shuffle them with random.Random(HELDOUT_SEED_BASE + seed), and hold out
  n_hold(s) = round(frac * n(s)) passages (at least 1). frac defaults to 1/3.
  The held-out passage set H = union over skills; the training passage set = all other passages.

Training rows for the held-out run (plain phase-1 recipe, seed 0):
  rows = labelled train rows whose passage is NOT in H
         + the phase-1 retention rows (base-model answers to fixed generic questions; no passage)
         + the six phase-1 generic refusals (no passage)
  Then every row whose prompt equals ANY held or probe question (normalised) is dropped (leak guard,
  same as the seeds 1-2 leak-free single), so no training prompt is an eval prompt.
  Hard negatives (labels "negs") are not used by the plain recipe, so they cannot leak.

Eval set: held and probe questions whose passage is in H, nothing else.
"""
import random

from phase2_rows import GENERIC_PROMPTS, norm_prompt

HELDOUT_FRAC = 1.0 / 3.0
HELDOUT_SEED_BASE = 3500
SKILLS = ["debug", "check", "document", "scope"]


def skill_of(passage_id):
    return passage_id.split("#", 1)[0]


def all_passage_ids(L):
    """Every passage id that appears anywhere in the labels (train, negs, held, probes)."""
    ids = set()
    for key in ("train", "negs", "held", "probes"):
        for r in L[key]:
            ids.add(r["passage"])
    return ids


def split_passages(passage_ids, frac=HELDOUT_FRAC, seed=0):
    """Deterministic per-skill hold-out. Returns (heldout_ids_by_skill, per_skill_counts).

    heldout_ids_by_skill: {skill: sorted list of held-out passage ids}
    per_skill_counts: {skill: {"passages": n, "held_out": k, "train": n-k}}
    """
    by_skill = {}
    for pid in passage_ids:
        by_skill.setdefault(skill_of(pid), []).append(pid)
    rng = random.Random(HELDOUT_SEED_BASE + seed)
    heldout, counts = {}, {}
    for skill in sorted(by_skill):
        ids = sorted(by_skill[skill])
        k = max(1, int(frac * len(ids) + 0.5))
        k = min(k, len(ids) - 1) if len(ids) > 1 else 0
        shuffled = list(ids)
        rng.shuffle(shuffled)
        heldout[skill] = sorted(shuffled[:k])
        counts[skill] = {"passages": len(ids), "held_out": k, "train": len(ids) - k}
    return heldout, counts


def heldout_set(heldout_by_skill):
    out = set()
    for ids in heldout_by_skill.values():
        out.update(ids)
    return out


def split_labels(L, heldout):
    """Partition the labelled train rows and the eval questions by the held-out passage set H.

    Returns dict with:
      train_rows: labelled train rows with passage not in H (kept for training, before leak guard)
      train_heldout_rows: labelled train rows whose passage is in H (must not be trained on)
      eval_held / eval_probes: held / probe questions whose passage is in H
      other_held / other_probes: the remaining held / probe questions (training passages)
    """
    H = heldout
    return {
        "train_rows": [r for r in L["train"] if r["passage"] not in H],
        "train_heldout_rows": [r for r in L["train"] if r["passage"] in H],
        "eval_held": [r for r in L["held"] if r["passage"] in H],
        "eval_probes": [r for r in L["probes"] if r["passage"] in H],
        "other_held": [r for r in L["held"] if r["passage"] not in H],
        "other_probes": [r for r in L["probes"] if r["passage"] not in H],
    }


def build_heldout_rows(train_rows, retain_rows, generic_prompts, eval_prompts, abstain_completion):
    """Training rows for the held-out single adapter (phase-1 plain recipe on the training passages).

    train_rows: [{"prompt","completion"}] from the training passages only.
    eval_prompts: every held and probe prompt (all passages); a training row with one of these prompts is dropped.
    Returns (rows, info); info lists the dropped prompts and the counts.
    """
    excl = {norm_prompt(p) for p in eval_prompts}
    kept, dropped = [], []
    for r in train_rows:
        if norm_prompt(r["prompt"]) in excl:
            dropped.append(r["prompt"])
        else:
            kept.append({"prompt": r["prompt"], "completion": r["completion"]})
    generic = [{"prompt": q, "completion": abstain_completion} for q in generic_prompts]
    retain = [{"prompt": r["prompt"], "completion": r["completion"]} for r in retain_rows]
    rows = kept + retain + generic
    leak = sorted({norm_prompt(r["prompt"]) for r in rows} & excl)
    info = {"train_rows_in": len(train_rows), "dropped_eval_overlap": dropped, "retain_rows": len(retain),
            "abstain_rows": len(generic), "total_rows": len(rows),
            "abstain_share": round(len(generic) / len(rows), 4) if rows else 0.0,
            "eval_prompt_overlap_after_build": leak}
    return rows, info


def check_split(heldout, train_rows, train_heldout_rows, rows, eval_held, eval_probes):
    """Zero-overlap checks. Returns a list of problems (empty = OK)."""
    problems = []
    H = heldout_set(heldout)
    # per skill: no held-out passage of skill s is a training passage of skill s
    for r in train_rows:
        if r["passage"] in H:
            problems.append(f"training row on held-out passage {r['passage']}")
    for r in train_heldout_rows:
        if r["passage"] not in H:
            problems.append(f"inconsistent split for {r['passage']}")
    # the rows that enter training must not carry any held-out passage id (retention/abstention carry none)
    for r in rows:
        if r.get("passage") in H:
            problems.append(f"built row with held-out passage {r['passage']}")
    # eval questions only from held-out passages
    for r in list(eval_held) + list(eval_probes):
        if r["passage"] not in H:
            problems.append(f"eval question on training passage {r['passage']}")
    return problems
