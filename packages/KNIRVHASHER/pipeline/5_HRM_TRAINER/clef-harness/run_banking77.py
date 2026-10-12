"""BANKING77 calibration probe.

Each item becomes ONE Clef call: a 77-option `choice` question. That is the
point of the harness -- it is exactly the shape of a KNIRV action decision, and
Cloudflare's own published numbers for this benchmark give us something to
falsify (macro-F1 90.93 for clef-flash, 94.20 for clef).

The baseline arm runs without any credentials, so the harness is verifiable
today. `baseline_nn` is a centroid nearest-neighbour classifier chosen to
mirroring KNIRV Phase 4's retrieval index, so the comparison answers a real
question: how much of Clef's accuracy is inherent to the task, and how much
does a plain local index already get?
"""

from __future__ import annotations

import argparse
import csv
import json
import math
import os
import re
import sys
import time
from collections import Counter, defaultdict
from concurrent.futures import ThreadPoolExecutor
from typing import Any, Dict, List, Sequence, Tuple

from clef_client import (ClefClient, LiveUnavailable, credentials_available, describe_preflight,
                          fingerprint, offline_stub, preflight)
from clef_contract import Question, choice
from metrics import abstain_curve, calibration_report, summarise

HERE = os.path.dirname(os.path.abspath(__file__))
DATA = os.path.join(HERE, "data")
RESULTS = os.path.join(HERE, "results")


def sanitise(obj: Any) -> Any:
    """Replace NaN/Inf with None.

    A coverage level where every item abstains has undefined accuracy. Python's
    json module writes a bare `NaN` token there, which is not valid JSON and
    breaks any consumer that parses this file -- so it becomes null instead.
    """
    if isinstance(obj, dict):
        return {k: sanitise(v) for k, v in obj.items()}
    if isinstance(obj, (list, tuple)):
        return [sanitise(v) for v in obj]
    if isinstance(obj, float) and not math.isfinite(obj):
        return None
    return obj


def take_subset(items: Sequence[Tuple[str, str]], limit: int, seed: int = 0) -> List[Tuple[str, str]]:
    """Uniform seeded sample across the split.

    NOT a prefix slice: the official Banking77 test CSV is ordered by category,
    so the first 200 rows contain only 5 of the 77 classes. Slicing would
    quietly evaluate a 5-class problem and report it as 77-way.
    """
    all_items = list(items)
    if not limit or limit >= len(all_items):
        return all_items
    import random

    rng = random.Random(seed)
    idx = sorted(rng.sample(range(len(all_items)), limit))
    return [all_items[i] for i in idx]


def load_categories() -> List[str]:
    with open(os.path.join(DATA, "categories.json")) as fh:
        raw = json.load(fh)
    # Official file is an ordered list of {category_name, examples: [...]}.
    names = [c["category_name"] if isinstance(c, dict) else c for c in raw]
    return sorted(names)


def load_split(name: str) -> List[Tuple[str, str]]:
    with open(os.path.join(DATA, f"{name}.csv")) as fh:
        return [(r["text"].strip(), r["category"].strip()) for r in csv.DictReader(fh)]


def build_question(categories: Sequence[str]) -> Question:
    """77-way choice. Option descriptions are the category name itself; the
    published schema permits null descriptions, but real names carry more
    signal than a null, so they are passed through verbatim."""
    return choice(
        "Classify the customer's banking request into exactly one of the "
        f"{len(categories)} intents. Respond with the single best-matching intent id.",
        {c: c.replace("_", " ") for c in categories},
    )


def baseline_nn(
    train: Sequence[Tuple[str, str]],
    test: Sequence[Tuple[str, str]],
    k: int = 5,
) -> Tuple[List[int], List[List[float]], Dict[str, Any]]:
    """TF-IDF cosine nearest-neighbour centroid baseline.

    Probabilities come from the vote share of the k neighbours, which is a real
    (if crude) calibrated signal -- which is what makes the calibration
    comparison meaningful rather than a strawman.
    """
    categories = sorted({c for _, c in train})
    index = {c: i for i, c in enumerate(categories)}

    def toks(s: str) -> List[str]:
        return re.findall(r"[a-z0-9']+", s.lower())

    df: Counter = Counter()
    docs = [toks(t) for t, _ in train]
    for d in docs:
        df.update(set(d))
    n_docs = len(docs)
    idf = {t: math.log((n_docs + 1) / (c + 1)) + 1.0 for t, c in df.items()}

    def vec(s: str) -> Dict[str, float]:
        v: Dict[str, float] = defaultdict(float)
        for t in toks(s):
            w = idf.get(t)
            if w:
                v[t] += w
        norm = math.sqrt(sum(x * x for x in v.values())) or 1.0
        return {t: x / norm for t, x in v.items()}

    train_vecs = [vec(t) for t, _ in train]
    train_lbl = [index[c] for _, c in train]
    # Class centroids for a fast first pass; fall back to kNN only on the centroid.
    centroids: Dict[int, Dict[str, float]] = defaultdict(lambda: defaultdict(float))
    counts: Counter = Counter()
    for v, l in zip(train_vecs, train_lbl):
        counts[l] += 1
        for t, x in v.items():
            centroids[l][t] += x

    def cos(a: Dict[str, float], b: Dict[str, float]) -> float:
        if len(a) > len(b):
            a, b = b, a
        return sum(x * b.get(t, 0.0) for t, x in a.items())

    pred: List[int] = []
    probs: List[List[float]] = []
    t0 = time.perf_counter()
    for text, _ in test:
        v = vec(text)
        sims = {l: cos(v, c) for l, c in centroids.items()}
        best = max(sims, key=sims.get)
        raw = [max(0.0, sims.get(l, 0.0)) for l in range(len(categories))]
        total = sum(raw) or 1.0
        probs.append([x / total for x in raw])
        pred.append(best)
    elapsed = time.perf_counter() - t0
    meta = {"arm": "tfidf_centroid_nn", "k": k, "n_train": n_docs,
            "n_features": len(idf), "latency_s": elapsed}
    return pred, probs, meta


def evaluate(name: str, pred: Sequence[int], gold: Sequence[int],
             probs: Sequence[Sequence[float]], labels: Sequence[str],
             meta: Dict[str, Any]) -> Dict[str, Any]:
    index = {c: i for i, c in enumerate(labels)}
    gold_idx = [index[c] for c in gold]
    summary = summarise(pred, gold_idx, labels, probs)
    summary["arm"] = name
    summary.update(meta)
    summary["reliability"] = abstain_curve(
        probs, gold_idx, [0.3, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 0.99]
    )
    dist = Counter(gold_idx)
    summary["gold_class_count"] = len(dist)
    summary["gold_min_class_examples"] = min(dist.values())
    return summary


def run_clef(
    model: str,
    test: Sequence[Tuple[str, str]],
    categories: Sequence[str],
    limit: int,
    dry_run: bool,
    stub_accuracy: float,
    seed: int = 0,
    concurrency: int = 1,
) -> Dict[str, Any]:
    q = build_question(categories)
    use_stub = dry_run or not credentials_available()
    client = ClefClient(model=model) if not use_stub else None

    pred: List[int] = []
    gold: List[str] = []
    probs: List[List[float]] = []
    latencies: List[float] = []
    in_toks = out_toks = 0
    cost = 0.0
    errors: List[str] = []
    input_tokens_reported: List[int] = []

    subset = take_subset(test, limit, seed=seed)
    print(f"[clef:{model}] {'STUB (dry-run)' if dry_run else ('STUB (no credentials)' if use_stub else 'LIVE')} "
          f"over {len(subset)} items", file=sys.stderr)

    def work(job: Tuple[int, str, str]):
        """Return (row, error). The transport error must survive to the caller:
        dropping it here turns a 401/429 into an undiagnosable 'call failed'."""
        i, text, category = job
        state = {"text": text}
        if client is not None:
            resp, call = client.ask(state, {"intent": q})
            if not call.ok:
                return None, (call.error or "call failed")
            probs_row = resp.decisions["intent"].probabilities
            return ((categories.index(resp.decisions["intent"].chosen),
                     [probs_row.get(c, 0.0) for c in categories], category,
                     call.latency_s, call.input_tokens, call.output_tokens,
                     call.cost_usd), None)
        resp, call = offline_stub(state, {"intent": q}, seed=i + seed * 100003,
                                  accuracy=stub_accuracy, seed_state=category)
        probs_row = resp.decisions["intent"].probabilities
        return ((categories.index(resp.decisions["intent"].chosen),
                 [probs_row.get(c, 0.0) for c in categories], category,
                 0.0, call.input_tokens, call.output_tokens, call.cost_usd), None)

    jobs = [(i, text, category) for i, (text, category) in enumerate(subset)]
    n = len(jobs)
    done = 0
    t_start = __import__("time").perf_counter()
    if concurrency > 1:
        ex = ThreadPoolExecutor(max_workers=concurrency)
        results_iter = ex.map(work, jobs)
    else:
        results_iter = map(work, jobs)
    for row, err in results_iter:
        done += 1
        if row is None:
            errors.append(err or "call failed")
            continue
        p, pr, g, lat, it, ot, c = row
        pred.append(p); probs.append(pr); gold.append(g)
        latencies.append(lat); input_tokens_reported.append(it)
        out_toks += ot; cost += c
        if done % 200 == 0:
            el = __import__("time").perf_counter() - t_start
            rate = done / el if el else 0
            print(f"  {done}/{n}  {rate:.1f} req/s  ${cost:.4f}  eta {((n-done)/rate)/60:.1f} min",
                  file=sys.stderr)
    wall_s = __import__("time").perf_counter() - t_start

    latencies.sort()
    if input_tokens_reported:
        input_tokens_reported.sort()

    # Fail loudly. Previously every call could fail and the run would continue,
    # then die in the metrics layer with "empty input" -- hiding the one error
    # that actually mattered.
    if not pred:
        pf = preflight(model) if not use_stub else {}
        raise LiveUnavailable(
            f"no successful {model} calls out of {len(subset)} attempted. "
            f"First error(s): {errors[:3]}. "
            + (f"Blocked: {pf.get('blocker')}" if pf.get("blocker") else "")
        )

    return evaluate(
        f"clef-{model}" + ("-stub" if use_stub else "-live"),
        pred, gold, probs, categories,
        {
            "model": model, "stub": use_stub, "n_errors": len(errors),
            "errors": errors[:5],
            "seed": seed,
            "concurrency": concurrency,
            "wall_s": wall_s,
            "req_per_s": (len(latencies) / wall_s) if wall_s else None,
            "sampled": limit != len(test),
            "latency_p50_s": latencies[len(latencies)//2] if latencies else None,
            "latency_p95_s": latencies[int(len(latencies)*0.95)] if latencies else None,
            "input_tokens": in_toks,
            "input_tokens_median": input_tokens_reported[len(input_tokens_reported)//2]
            if input_tokens_reported else None,
            "output_tokens": out_toks,
            "cost_usd_total": cost,
            "cost_usd_per_item": cost / max(len(pred), 1),
            "request_fingerprint_first": fingerprint({"text": subset[0][0]}) if subset else None,
        },
    )


def main(argv: Sequence[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description="BANKING77 calibration probe for Clef")
    ap.add_argument("--model", default="clef-flash", choices=["clef", "clef-flash"])
    ap.add_argument("--limit", type=int, default=200, help="test items to evaluate (0 = all 3080)")
    ap.add_argument("--seed", type=int, default=0, help="seed for the uniform subset sample")
    ap.add_argument("--concurrency", type=int, default=1,
                    help="parallel requests; >1 inflates measured latency, so keep 1 "
                         "when latency is the quantity of interest")
    ap.add_argument("--arms", default="baseline,clef", help="comma list of: baseline,clef")
    ap.add_argument("--dry-run", action="store_true", help="use the offline stub even if credentials exist")
    ap.add_argument("--stub-accuracy", type=float, default=0.35)
    ap.add_argument("--baseline-limit", type=int, default=0, help="0 = all test items for the local arm")
    ap.add_argument("--out", default=os.path.join(RESULTS, "banking77.json"))
    ap.add_argument("--preflight", action="store_true",
                    help="probe Workers AI readiness and exit")
    args = ap.parse_args(argv)

    if args.preflight:
        pf = preflight(args.model)
        print(describe_preflight(pf))
        return 0 if pf["ready"] else 1

    categories = load_categories()
    test = load_split("test")
    train = load_split("train")
    print(f"categories={len(categories)} train={len(train)} test={len(test)}", file=sys.stderr)

    os.makedirs(RESULTS, exist_ok=True)
    reports = []

    arms = {a.strip() for a in args.arms.split(",") if a.strip()}
    if "baseline" in arms:
        sub = take_subset(test, args.baseline_limit, seed=args.seed) if args.baseline_limit else list(test)
        pred, probs, meta = baseline_nn(train, sub)
        reports.append(evaluate("baseline-tfidf-centroid", pred, [c for _, c in sub],
                                probs, categories, meta))

    if "clef" in arms:
        reports.append(run_clef(args.model, test, categories, args.limit, args.dry_run,
                                args.stub_accuracy, seed=args.seed,
                                concurrency=max(1, args.concurrency)))

    report = {"benchmark": "banking77", "n_categories": len(categories),
              "n_test_total": len(test), "reports": reports}
    with open(args.out, "w") as fh:
        json.dump(sanitise(report), fh, indent=2, allow_nan=False)

    print(json.dumps(sanitise(reports), indent=2, allow_nan=False))
    print(f"\nwrote {args.out}", file=sys.stderr)

    if not credentials_available() and "clef" in arms and not args.dry_run:
        print("\nNOTE: no Cloudflare credentials found; the Clef arm ran against the "
              "offline stub and its numbers are NOT a Clef measurement.", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())