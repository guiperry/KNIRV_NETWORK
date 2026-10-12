"""Decision-quality metrics, weighted toward calibration.

Cloudflare's benchmark tables report task metrics (macro-F1, accuracy) and omit
calibration. KNIRV cannot ship on accuracy alone: `supervisor_model.md` gates
safety on thresholds that are only meaningful if the reported confidence is
honest, and the whole premise of a decision model is that it replaces an
unvalidated `MinConfidence = 0.3` constant.

So the headline pair here is Brier + ECE, and accuracy is reported alongside
rather than instead.
"""

from __future__ import annotations

import math
from collections import defaultdict
from dataclasses import dataclass
from typing import Dict, List, Sequence, Tuple


@dataclass
class Calibration:
    """Multiclass calibration, one-hot targets against a probability vector."""

    ece: float          # Expected Calibration Error, equal-width bins
    mce: float          # Max Calibration Error, per-bin worst gap
    brier: float        # Multiclass Brier score (sum of squared errors)
    n: int

    def as_dict(self) -> Dict[str, float]:
        return {"ece": self.ece, "mce": self.mce, "brier": self.brier, "n": self.n}


def multiclass_brier(probs: Sequence[float], true_index: int) -> float:
    """Sum of squared differences from the one-hot target. Range [0, 2].

    Some papers divide by K; this reports the unnormalised sum so that a
    77-way BANKING77 result is comparable with a 6-way KNIRV one only after
    dividing -- see `brier_normalised`.
    """
    return sum((p - (1.0 if i == true_index else 0.0)) ** 2 for i, p in enumerate(probs))


def brier_normalised(probs: Sequence[float], true_index: int) -> float:
    k = len(probs)
    return multiclass_brier(probs, true_index) / k if k else float("nan")


def expected_calibration_error(
    confidences: Sequence[float],
    correct: Sequence[bool],
    n_bins: int = 15,
) -> Tuple[float, float]:
    """ECE and MCE over equal-width confidence bins.

    Confidence is the peak probability -- i.e. "how much the model would claim"
    -- against observed accuracy. Returns (ece, mce).
    """
    if not confidences:
        return float("nan"), float("nan")
    bins: Dict[int, List[Tuple[float, bool]]] = defaultdict(list)
    for c, k in zip(confidences, correct):
        idx = min(int(c * n_bins), n_bins - 1)
        bins[idx].append((c, k))

    ece = 0.0
    mce = 0.0
    total = len(confidences)
    for idx, items in bins.items():
        acc = sum(1 for _, k in items if k) / len(items)
        conf = sum(c for c, _ in items) / len(items)
        gap = abs(acc - conf)
        ece += (len(items) / total) * gap
        mce = max(mce, gap)
    return ece, mce


def calibration_report(
    probs: Sequence[Sequence[float]],
    gold: Sequence[int],
    labels: Sequence[str],
    n_bins: int = 15,
) -> Calibration:
    if len(probs) != len(gold):
        raise ValueError("probs and gold must be the same length")
    if not probs:
        raise CalibrationError("empty input")

    confidences: List[float] = []
    correct: List[bool] = []
    briers: List[float] = []

    for p, g in zip(probs, gold):
        if len(p) != len(labels):
            raise ValueError("probability vector length does not match the label set")
        peak = max(p)
        pick = max(range(len(p)), key=lambda i: p[i])
        confidences.append(peak)
        correct.append(pick == g)
        briers.append(multiclass_brier(p, g))

    ece, mce = expected_calibration_error(confidences, correct, n_bins)
    return Calibration(ece=ece, mce=mce, brier=sum(briers) / len(briers), n=len(gold))


class CalibrationError(ValueError):
    pass


def macro_f1(pred: Sequence[int], gold: Sequence[int], k: int) -> Tuple[float, Dict[str, float]]:
    """Macro-averaged F1 across all k classes, with zeros for unseen classes.

    Banking77's published numbers are macro-F1, which is why Cloudflare reports
    90.93 against it. Accuracy on this dataset is ~5 points higher because the
    class distribution is skewed, so the two must not be compared loosely.
    """
    tp = [0] * k
    fp = [0] * k
    fn = [0] * k
    for p, g in zip(pred, gold):
        if p == g:
            tp[g] += 1
        else:
            fp[p] += 1
            fn[g] += 1

    per_class: Dict[str, float] = {}
    f1s: List[float] = []
    for i in range(k):
        prec = tp[i] / (tp[i] + fp[i]) if (tp[i] + fp[i]) else 0.0
        rec = tp[i] / (tp[i] + fn[i]) if (tp[i] + fn[i]) else 0.0
        f1 = (2 * prec * rec / (prec + rec)) if (prec + rec) else 0.0
        f1s.append(f1)
        per_class[f"{i}"] = f1
    return (sum(f1s) / k if k else float("nan")), per_class


def accuracy(pred: Sequence[int], gold: Sequence[int]) -> float:
    if not gold:
        return float("nan")
    return sum(1 for p, g in zip(pred, gold) if p == g) / len(gold)


def confusion(pred: Sequence[int], gold: Sequence[int], labels: Sequence[str]) -> List[List[int]]:
    k = len(labels)
    m = [[0] * k for _ in range(k)]
    for p, g in zip(pred, gold):
        m[g][p] += 1
    return m


def brier_skill_score(brier: float, baseline_brier: float) -> float:
    """1.0 matches the baseline, negative means worse than it.

    The baseline is a fixed prior over the class distribution -- without this,
    a Brier number is not interpretable on its own.
    """
    if baseline_brier == 0:
        return float("nan")
    return 1.0 - brier / baseline_brier


def abstain_curve(
    probs: Sequence[Sequence[float]],
    gold: Sequence[int],
    taus: Sequence[float],
) -> List[Dict[str, float]]:
    """Coverage / risk / selective-accuracy across abstention thresholds.

    This is the curve that actually justifies a merge: it shows how much risk the
    system takes on for a given coverage, which is precisely the operating
    choice `MinConfidence` was being used to make badly.
    """
    n = len(gold)
    rows: List[Dict[str, float]] = []
    for tau in taus:
        kept = [(p, g) for p, g in zip(probs, gold) if max(p) >= tau]
        if not kept:
            rows.append({"tau": tau, "coverage": 0.0, "accuracy": float("nan"), "n": 0})
            continue
        correct = sum(1 for p, g in kept if max(range(len(p)), key=lambda i: p[i]) == g)
        rows.append(
            {
                "tau": tau,
                "coverage": len(kept) / n if n else 0.0,
                "accuracy": correct / len(kept),
                "n": float(len(kept)),
            }
        )
    return rows


def summarise(pred: Sequence[int], gold: Sequence[int], labels: Sequence[str], probs=None) -> Dict:
    k = len(labels)
    f1, per_class = macro_f1(pred, gold, k)
    out: Dict = {"n": len(gold), "accuracy": accuracy(pred, gold), "macro_f1": f1}
    if probs is not None:
        cal = calibration_report(probs, gold, labels)
        out.update(cal.as_dict())
        out["brier_normalised"] = sum(brier_normalised(p, g) for p, g in zip(probs, gold)) / max(len(gold), 1)
        best = max(range(k), key=lambda i: sum(1 for g in gold if g == i))
        prior = [1.0 / k] * k
        out["brier_uniform_baseline"] = sum(
            brier_normalised(prior, g) for g in gold
        ) / max(len(gold), 1)
    return out