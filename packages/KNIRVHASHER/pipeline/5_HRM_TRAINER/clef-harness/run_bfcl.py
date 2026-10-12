"""BFCL v3 function-selection probe.

WHAT THIS MEASURES, precisely
-----------------------------
A `choice` question: given a user request and a set of candidate function
signatures, which function should be called?

This is NOT the official BFCL score. Official BFCL is AST-based exact match over
the entire call expression, including argument names, types, and ordering.
Those `possible_answer/` files and the AST matcher are deliberately NOT
implemented here, so every number below is reported under
`function_selection_accuracy_proxy` and must never be quoted as a BFCL score.

Why this proxy is still worth running: selecting among known functions is
exactly the decision a `choice` question expresses, and selecting the right
team member is exactly what `supervisorrouting/router.go:92` does with Claude
Haiku today. It is the same shape of problem, at a scale where ground truth is
unambiguous.

WHY THE IRRELEVANCE SPLIT MATTERS MOST
--------------------------------------
BFCL_v3_irrelevance is 240 examples where the right answer is to call nothing.
That is KNIRV's `abstain` action, i.e. the single most consequential decision in
the supervisor. And Cloudflare's own table shows Clef-flash weakest on
When2Call (65.58 vs Jev 80.97) -- "should a tool be called at all". So this split
is the harness's sharpest test of the merge's riskiest assumption, and it is
weighted accordingly in the report.

Data: gorilla-llm/Berkeley-Function-Calling-Leaderboard (canonical, 147k
downloads), fetched 2026-10-05. Note Cloudflare's published BFCL figure does not
state a version; a v1/v3 split difference would make it non-reproducible here.
"""

from __future__ import annotations

import argparse
import json
import os
import random
import sys
from collections import Counter, defaultdict
from concurrent.futures import ThreadPoolExecutor
from typing import Any, Dict, List, Sequence, Tuple

from clef_client import (ClefClient, LiveUnavailable, credentials_available, describe_preflight,
                          fingerprint, offline_stub, preflight)
from clef_contract import choice
from metrics import abstain_curve, calibration_report, summarise

HERE = os.path.dirname(os.path.abspath(__file__))
BFCL = os.path.join(HERE, "data", "bfcl")
RESULTS = os.path.join(HERE, "results")

# No-function sentinel. Prefixed to avoid colliding with a real function name.
NONE = "__none__"

# AST splits: no ground_truth (BFCL scores these against possible_answer/ with an
# AST matcher). Their labels are still recoverable -- the gold function is the
# one the question is about -- but only for selection, not for full arguments.
AST_SPLITS = ["BFCL_v3_simple", "BFCL_v3_multiple", "BFCL_v3_parallel",
              "BFCL_v3_parallel_multiple", "BFCL_v3_irrelevance"]


def load_split(name: str) -> List[Dict[str, Any]]:
    """BFCL files are JSONL despite the .json extension."""
    path = os.path.join(BFCL, f"{name}.json")
    if not os.path.exists(path):
        return []
    with open(path) as fh:
        return [json.loads(line) for line in fh if line.strip()]


def render_params(f: Dict[str, Any]) -> str:
    """Render a function's parameters as a one-line signature fragment.

    BFCL v3 stores `parameters` as a JSON-schema object
    ({type, properties:{name:{type,description}}, required:[...]}), while v1
    stored a list of {name, type, description}. Both are accepted because the
    Cloudflare published figure does not name a BFCL version.
    """
    params = f.get("parameters", [])
    if isinstance(params, dict):
        props = params.get("properties", {}) or {}
        required = set(params.get("required", []) or [])
        bits = []
        for name, spec in props.items():
            typ = spec.get("type", "any") if isinstance(spec, dict) else "any"
            bits.append(f"{typ} {name}" + ("" if name in required else "?"))
        return ", ".join(bits)
    if isinstance(params, list):
        bits = []
        for p in params:
            if isinstance(p, dict):
                bits.append(f"{p.get('type', 'any')} {p.get('name', '')}".strip())
            else:
                bits.append(str(p))
        return ", ".join(bits)
    return ""


def describe_functions(funcs: Sequence[Dict[str, Any]]) -> Dict[str, str]:
    out: Dict[str, str] = {}
    for f in funcs:
        if not isinstance(f, dict) or "name" not in f:
            continue
        desc = str(f.get("description", "") or "").strip()
        sig = render_params(f)
        out[str(f["name"])] = f"{desc} [{sig}]".strip() if sig else desc
    return out


def question_text(item: Dict[str, Any]) -> str:
    """Flatten a BFCL `question` into plain text.

    All nine v3 splits store `question` as list[list[{role, content}]] -- even
    the single-turn ones -- so one level of nesting has to come off before the
    turns can be read. Content may itself be a list of content parts.
    """
    turns = item.get("question", [])
    if isinstance(turns, str):
        return turns
    flat: List[Dict[str, Any]] = []
    for turn in turns if isinstance(turns, list) else []:
        if isinstance(turn, dict):
            flat.append(turn)
        elif isinstance(turn, list):
            flat.extend(m for m in turn if isinstance(m, dict))

    lines: List[str] = []
    for m in flat:
        content = m.get("content", "")
        if isinstance(content, list):
            content = " ".join(
                c.get("text", "") for c in content if isinstance(c, dict)
            )
        role = m.get("role", "user")
        if content:
            lines.append(f"{role}: {content}")
    return "\n".join(lines)


def gold_function(item: Dict[str, Any], split: str) -> str:
    """Recover the gold function name.

    For irrelevance the gold answer is "call nothing" -- derived from the split,
    since BFCL provides no ground_truth for it either.
    """
    if "irrelevance" in split:
        return NONE
    funcs = [f["name"] for f in item.get("function", [])]
    if len(funcs) == 1:
        return funcs[0]
    gt = item.get("ground_truth")
    if isinstance(gt, str):
        for name in funcs:
            if name in gt:
                return name
    if isinstance(gt, list) and gt:
        first = gt[0]
        if isinstance(first, dict) and "name" in first:
            return str(first["name"])
    # Fall back to a recorded answer file when present.
    return funcs[0] if funcs else NONE


def build_question(names: Sequence[str], descriptions: Dict[str, str]) -> Any:
    crit = {n: descriptions.get(n, n) for n in names}
    crit[NONE] = "None of the listed functions applies to this request."
    return choice(
        "Which function, if any, should be called to satisfy this request? "
        "Choose exactly one. If no listed function applies, choose the none option.",
        crit,
    )


def sample(items: Sequence[Dict[str, Any]], limit: int, seed: int) -> List[Dict[str, Any]]:
    if not limit or limit >= len(items):
        return list(items)
    rng = random.Random(seed)
    return [items[i] for i in sorted(rng.sample(range(len(items)), limit))]


def run_split(
    split: str,
    limit: int,
    seed: int,
    model: str,
    dry_run: bool,
    stub_accuracy: float,
    concurrency: int = 1,
) -> Dict[str, Any]:
    items = load_split(split)
    if not items:
        return {"arm": f"bfcl-{split}", "n": 0, "note": "split file missing"}

    subset = sample(items, limit, seed)
    use_stub = dry_run or not credentials_available()
    client = ClefClient(model=model) if not use_stub else None

    # Union of function names, used ONLY to pad probability vectors to a common
    # width so the metric helpers can consume them. The question itself is built
    # per item from that item's own candidates -- scoring against a split-wide
    # union would inflate the option count with functions the request never
    # mentions and make results incomparable across --limit values.
    all_names: List[str] = []
    for it in subset:
        for f in it.get("function", []):
            if isinstance(f, dict) and f.get("name") and f["name"] not in all_names:
                all_names.append(f["name"])
    if NONE not in all_names:
        all_names.append(NONE)

    pred: List[int] = []
    gold: List[int] = []
    probs: List[List[float]] = []
    option_counts: List[int] = []

    calls = 0
    errors: List[str] = []
    latencies: List[float] = []
    cost = 0.0
    t0 = __import__("time").perf_counter()

    jobs = []
    for i, it in enumerate(subset):
        g = gold_function(it, split)
        if g not in all_names:
            all_names.append(g)
        available = describe_functions(it.get("function", []))
        item_names = list(available) + [NONE]
        q = build_question(item_names, available)
        state = {
            "request": question_text(it),
            "available_functions": available,
        }
        jobs.append((i, state, q, g, item_names))

    def work(job):
        i, state, q, g, item_names = job
        try:
            if client is not None:
                resp, call = client.ask(state, {"which": q})
                if not call.ok:
                    return None, call.error
                chosen = resp.decisions["which"].chosen
                row = resp.decisions["which"].probabilities
                return (chosen, row, g, item_names, call), None
            resp, call = offline_stub(state, {"which": q}, seed=i + seed * 7919,
                                      accuracy=stub_accuracy, seed_state=g)
            return ((resp.decisions["which"].chosen,
                     resp.decisions["which"].probabilities, g, item_names, call), None)
        except Exception as exc:  # a malformed item must not kill the split
            return None, f"{type(exc).__name__}: {exc}"

    n = len(jobs)
    done = 0
    t_start = __import__("time").perf_counter()
    if concurrency > 1:
        ex = ThreadPoolExecutor(max_workers=concurrency)
        iterator = ex.map(work, jobs)
    else:
        iterator = map(work, jobs)
    for out, err in iterator:
        done += 1
        if out is None:
            errors.append(err or "call failed")
        else:
            chosen, row, g, item_names, call = out
            pred.append(all_names.index(chosen) if chosen in all_names else -1)
            gold.append(all_names.index(g))
            option_counts.append(len(item_names))
            # Pad to the union width. Zero padding cannot win an argmax because
            # the parsed probabilities sum to 1.
            probs.append([row.get(n_, 0.0) if n_ in row else 0.0 for n_ in all_names])
            if not use_stub:
                latencies.append(call.latency_s)
                cost += call.cost_usd
            calls += 1
        if done % 200 == 0:
            el = __import__("time").perf_counter() - t_start
            rate = done / el if el else 0
            print(f"  {done}/{n}  {rate:.1f} req/s  ${cost:.4f}  "
                  f"eta {((n-done)/rate)/60:.1f} min", file=sys.stderr)
    wall_s = __import__("time").perf_counter() - t_start
    latencies.sort()

    if not pred:
        from clef_client import preflight
        pf = preflight(model) if not use_stub else {}
        raise LiveUnavailable(
            f"{split}: no successful {model} calls out of {len(subset)} attempted. "
            f"First error(s): {errors[:3]}. "
            + (f"Blocked: {pf.get('blocker')}" if pf.get("blocker") else "")
        )

    summary = summarise(pred, gold, all_names, probs)
    summary["arm"] = f"bfcl-{split}"
    summary["metric"] = "function_selection_accuracy_proxy"   # NOT the BFCL score
    summary["official_bfcl_metric_implemented"] = False
    summary["stub"] = use_stub
    # With >1 listed function and no ground_truth, gold_function() falls back to
    # funcs[0], which is a guess, not a label. Those splits cannot be scored here
    # (BFCL keeps the answers in possible_answer/, which is not vendored).
    if any(len(it.get("function", [])) > 1 and "ground_truth" not in it for it in subset):
        summary["label_valid"] = False
        summary["label_note"] = ("gold = first listed function (no ground_truth); accuracy/ECE "
                                 "on this split are NOT meaningful")
    summary["n_options_union"] = len(all_names)
    summary["n_options_per_item_median"] = (
        sorted(option_counts)[len(option_counts) // 2] if option_counts else 0
    )
    summary["n_options_per_item_max"] = max(option_counts) if option_counts else 0
    summary["elapsed_s"] = __import__("time").perf_counter() - t0
    summary["wall_s"] = wall_s
    summary["concurrency"] = concurrency
    summary["req_per_s"] = (calls / wall_s) if wall_s else None
    summary["latency_p50_s"] = latencies[len(latencies)//2] if latencies else None
    summary["latency_p95_s"] = latencies[int(len(latencies)*0.95)] if latencies else None
    summary["cost_usd_total"] = cost
    summary["cost_usd_per_item"] = cost / max(calls, 1)
    summary["n_errors"] = len(errors)
    summary["errors"] = errors[:3]
    summary["reliability"] = abstain_curve(probs, gold, [0.3, 0.5, 0.7, 0.9, 0.95])

    # Split the headline number by whether the gold answer was "call nothing".
    # That is the abstention-shaped case and deserves its own number.
    none_idx = all_names.index(NONE) if NONE in all_names else None
    if none_idx is not None:
        is_none = [i == none_idx for i in gold]
        if any(is_none):
            sub_gold = [g for g, f in zip(gold, is_none) if f]
            sub_pred = [p for p, f in zip(pred, is_none) if f]
            summary["no_function_subset"] = {
                "n": len(sub_gold),
                "accuracy": sum(1 for p, g in zip(sub_pred, sub_gold) if p == g) / len(sub_gold),
            }
            # Abstention operating curve: "abstain iff P(none) >= tau". On this
            # split every gold answer is "none", so abstain_rate is recall and
            # (1 - abstain_rate) is the false-call rate at that threshold.
            pn = [row[none_idx] for row, f in zip(probs, is_none) if f]
            summary["none_curve"] = [
                {"tau": t, "abstain_rate": sum(1 for x in pn if x >= t) / len(pn),
                 "false_call_rate": sum(1 for x in pn if x < t) / len(pn)}
                for t in (0.1, 0.2, 0.3, 0.5, 0.7, 0.9)
            ]
            summary["p_none_mean"] = sum(pn) / len(pn)
    return summary


def main(argv: Sequence[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description="BFCL v3 function-selection probe for Clef")
    ap.add_argument("--model", default="clef-flash", choices=["clef", "clef-flash"])
    ap.add_argument("--splits", default=",".join(AST_SPLITS))
    ap.add_argument("--limit", type=int, default=0, help="per-split sample size (0 = all)")
    ap.add_argument("--seed", type=int, default=0)
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--stub-accuracy", type=float, default=0.35)
    ap.add_argument("--concurrency", type=int, default=1,
                    help="parallel requests; >1 inflates measured latency")
    ap.add_argument("--out", default=os.path.join(RESULTS, "bfcl.json"))
    args = ap.parse_args(argv)

    if not os.path.isdir(BFCL):
        print(f"missing {BFCL}; see README for the fetch command", file=sys.stderr)
        return 2

    reports = []
    for split in [s.strip() for s in args.splits.split(",") if s.strip()]:
        print(f"[bfcl] {split}", file=sys.stderr)
        r = run_split(split, args.limit, args.seed, args.model, args.dry_run,
                      args.stub_accuracy, concurrency=max(1, args.concurrency))
        reports.append(r)
        if r.get("n"):
            print(f"  n={r['n']} acc={r['accuracy']:.4f} macroF1={r['macro_f1']:.4f} "
                  f"ECE={r['ece']:.4f} opts/item={r.get('n_options_per_item_median')}",
                  file=sys.stderr)
        else:
            print(f"  {r.get('note')}", file=sys.stderr)

    os.makedirs(RESULTS, exist_ok=True)
    report = {
        "benchmark": "BFCL_v3_function_selection_proxy",
        "official_bfcl_metric_implemented": False,
        "model": args.model,
        "stub": args.dry_run or not credentials_available(),
        "reports": reports,
    }
    with open(args.out, "w") as fh:
        from run_banking77 import sanitise
        json.dump(sanitise(report), fh, indent=2, allow_nan=False)
    print(f"\nwrote {args.out}", file=sys.stderr)
    if report["stub"]:
        print("NOTE: stub arm -- these are NOT Clef measurements.", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())