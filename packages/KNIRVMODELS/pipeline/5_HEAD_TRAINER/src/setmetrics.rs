//! Task 4.5 — choice-set evaluation additions.
//!
//! Pure functions over set-head outputs. A `ScoredSet` is built from a
//! [`knirv_heart_decision::model::Reasoning`] / `Decision` / `Candidate`. To keep the
//! metrics honest it carries three views of the same row:
//!   * `probs` — candidate id -> probability, for cross-permutation lookup (order
//!     independent),
//!   * `candidates` — `(id, prob)` in the SERVED (request) order; top-1 uses
//!     first-maximum on this order, matching the runtime/tensor argmax tie rule,
//!   * `texts` — candidate id -> canonical choice text, so the novel-option gate
//!     (Task 4.4/4.5) keys on unseen *choice strings*, not opaque ids.
//!
//! Every aggregate validates each row first (N>=2, N unique candidates, finite
//! non-negative probabilities, finite near-one sum); invalid rows are excluded and
//! reported, never silently folded into a favourable bucket.
//!
//! Data-dependent pieces that need a trained 4.4 model and held-out/gold data are
//! deliberately NOT fabricated here: `novel_option_accuracy` takes the training
//! option-text set (Task 4.6 / gold) and stays `None` when empty; per-skill tagging
//! and `max_choices`/bucket-size selection stay `None`/provisional without held-out
//! rows. No synthetic "gold" is invented.

use crate::eval::{fit_tau, Confidences, TauFit};
use serde::Serialize;
use std::collections::{BTreeMap, BTreeSet};

/// One evaluated choice set.
#[derive(Clone, Debug, Default)]
pub struct ScoredSet {
    pub head: String,
    pub n: usize,
    /// candidate id -> probability (kept for cross-permutation lookup)
    pub probs: BTreeMap<String, f32>,
    /// candidates in the served (request) order; top-1 uses first-maximum here
    pub candidates: Vec<(String, f32)>,
    /// candidate id -> canonical choice text (lowercased/trimmed)
    pub texts: BTreeMap<String, String>,
    /// gold/held-out correct candidate id, if known
    pub correct: Option<String>,
    /// held-out skill tag for the row, if known
    pub skill: Option<String>,
}

/// Build a scored set from request-ordered choices `(id, canonical_text, prob)`.
pub fn from_choices(
    head: &str,
    choices: &[(&str, &str, f32)],
    correct: Option<&str>,
    skill: Option<&str>,
) -> ScoredSet {
    let mut probs = BTreeMap::new();
    let mut candidates = Vec::with_capacity(choices.len());
    let mut texts = BTreeMap::new();
    for (id, text, p) in choices {
        probs.insert(id.to_string(), *p);
        candidates.push((id.to_string(), *p));
        texts.insert(id.to_string(), canonical_text(text));
    }
    ScoredSet {
        head: head.to_string(),
        n: choices.len(),
        probs,
        candidates,
        texts,
        correct: correct.map(String::from),
        skill: skill.map(String::from),
    }
}

/// Canonicalise a choice string for the novel-option gate (Task 4.4/4.5 gate on
/// unseen *choice strings*; ids may be reused with changed text, or distinct ids
/// for the same text).
pub fn canonical_text(s: &str) -> String {
    s.trim().to_lowercase()
}

/// A row is valid for aggregation iff it has >=2 candidates, exactly `n` UNIQUE ids,
/// finite non-negative probabilities, and (model-output boundary) a finite sum within
/// 1e-3 of 1.0. Invalid rows are excluded from every aggregate and counted once.
pub fn valid_row(r: &ScoredSet) -> bool {
    if r.n < 2 {
        return false;
    }
    if r.candidates.len() != r.n {
        return false;
    }
    let mut ids = BTreeSet::new();
    let mut sum = 0.0f32;
    for (id, p) in &r.candidates {
        if ids.contains(id) {
            return false; // duplicate id
        }
        ids.insert(id.clone());
        if !p.is_finite() || *p < 0.0 {
            return false;
        }
        sum += *p;
    }
    if ids.len() != r.n {
        return false;
    }
    (sum - 1.0).abs() <= 1e-3
}

/// Relative deviation between two probabilities, robust to near-zero values.
/// Non-finite inputs return +infinity so they are never called "invariant".
pub fn rel_dev(a: f32, b: f32) -> f32 {
    if !a.is_finite() || !b.is_finite() {
        return f32::INFINITY;
    }
    let denom = a.abs().max(b.abs()).max(1e-12);
    (a - b).abs() / denom
}

/// First-maximum top-1 in request order. On equal probabilities the earliest request
/// candidate wins — identical to the runtime/tensor argmax, NOT the lexicographically
/// smallest id. This is the single documented tie rule used by every aggregate.
pub fn top1(cands: &[(String, f32)]) -> Option<String> {
    let mut best_idx: Option<usize> = None;
    let mut best_p = f32::NEG_INFINITY;
    for (i, (_, p)) in cands.iter().enumerate() {
        if *p > best_p {
            best_p = *p;
            best_idx = Some(i);
        }
    }
    best_idx.map(|i| cands[i].0.clone())
}

/// Order-invariance rate over permutation groups.
///
/// `groups` maps a set identity to the several `ScoredSet`s produced by permuting the
/// request's option order. A group is order-invariant iff every member shares the same
/// id set and `n`, and every candidate id's probability is equal across permutations
/// within `tol`. A member with a different id set/size (e.g. `{a,b}` vs `{a,b,c}`) or
/// any non-finite probability makes the group non-invariant.
pub fn order_invariance_rate(
    groups: &BTreeMap<String, Vec<ScoredSet>>,
    tol: f32,
) -> (f32, f32) {
    if groups.is_empty() {
        return (1.0, 0.0);
    }
    let mut invariant = 0usize;
    let mut worst = 0.0f32;
    for perms in groups.values() {
        if perms.len() < 2 {
            invariant += 1; // nothing to compare against
            continue;
        }
        let base = &perms[0];
        if !valid_row(base) {
            // an invalid base row cannot form a meaningful invariance group
            continue;
        }
        let base_ids: BTreeSet<&String> = base.probs.keys().collect();
        let mut group_ok = true;
        let mut group_worst = 0.0f32;
        for perm in &perms[1..] {
            if !valid_row(perm) {
                group_ok = false;
                continue;
            }
            // F1: identical id set AND same n required; otherwise we are comparing
            // different choice sets, not permutations of one set.
            if perm.probs.keys().collect::<BTreeSet<_>>() != base_ids || perm.n != base.n {
                group_ok = false;
                continue;
            }
            for (id, &p0) in &base.probs {
                let p1 = *perm.probs.get(id).unwrap(); // id guaranteed present
                let dev = rel_dev(p0, p1);
                if dev > group_worst {
                    group_worst = dev;
                }
                if dev > tol {
                    group_ok = false;
                }
            }
        }
        if group_worst > worst {
            worst = group_worst;
        }
        if group_ok {
            invariant += 1;
        }
    }
    (invariant as f32 / groups.len() as f32, worst)
}

/// Top-1 accuracy of set heads, bucketed by number of candidates `N`.
/// Only valid rows with a known `correct` contribute. Returns `N -> (correct, total)`.
pub fn accuracy_by_n(rows: &[ScoredSet]) -> BTreeMap<usize, (usize, usize)> {
    let mut out: BTreeMap<usize, (usize, usize)> = BTreeMap::new();
    for r in rows {
        let Some(correct) = &r.correct else { continue };
        if !valid_row(r) {
            continue;
        }
        let top = top1(&r.candidates);
        let entry = out.entry(r.n).or_insert((0, 0));
        entry.1 += 1;
        if top.as_deref() == Some(correct) {
            entry.0 += 1;
        }
    }
    out
}

/// Top-1 accuracy on rows whose correct option's *canonical text* was NOT in
/// `training_texts` (the plan's unseen-choice-string gate).
///
/// PROVISIONAL (needs the training-split option-text set from Task 4.6 / gold data).
/// Returns `None` when `training_texts` is empty so a baseline is never fabricated.
/// Novelty is decided by canonical text, not id: an id reused with new text is novel,
/// and a new id carrying a seen text is seen.
pub fn novel_option_accuracy(
    rows: &[ScoredSet],
    training_texts: &BTreeSet<String>,
) -> Option<f32> {
    if training_texts.is_empty() {
        return None;
    }
    let mut ok = 0usize;
    let mut n = 0usize;
    for r in rows {
        let Some(correct) = &r.correct else { continue };
        if !valid_row(r) {
            continue;
        }
        let text = match r.texts.get(correct) {
            Some(t) => t,
            None => continue, // cannot resolve the correct option's text -> skip
        };
        if training_texts.contains(text) {
            continue; // seen option
        }
        n += 1;
        let top = top1(&r.candidates);
        if top.as_deref() == Some(correct) {
            ok += 1;
        }
    }
    if n == 0 {
        None
    } else {
        Some(ok as f32 / n as f32)
    }
}

/// Per-skill top-1 accuracy. Returns `None` when no row carries a skill tag
/// (held-out only). Invalid rows are excluded.
pub fn accuracy_by_skill(rows: &[ScoredSet]) -> Option<BTreeMap<String, (usize, usize)>> {
    let mut out: BTreeMap<String, (usize, usize)> = BTreeMap::new();
    let mut any = false;
    for r in rows {
        let Some(skill) = &r.skill else { continue };
        let Some(correct) = &r.correct else { continue };
        if !valid_row(r) {
            continue;
        }
        any = true;
        let top = top1(&r.candidates);
        let entry = out.entry(skill.clone()).or_insert((0, 0));
        entry.1 += 1;
        if top.as_deref() == Some(correct) {
            entry.0 += 1;
        }
    }
    if !any {
        None
    } else {
        Some(out)
    }
}

/// Per-N bucket label for the tau fit (4.5): `2`, `3-4`, `5-8`, `9+`.
pub fn n_bucket(n: usize) -> String {
    match n {
        2 => "2".to_string(),
        3..=4 => "3-4".to_string(),
        5..=8 => "5-8".to_string(),
        _ => "9+".to_string(),
    }
}

/// Per-N-bucket tau fit. For each bucket, fit the smallest `tau` (on that bucket's
/// confidences, scored at batch 1 with each candidate's real `max(p)`) at which
/// selective accuracy `>= target`. Reuses [`crate::eval::fit_tau`]. Invalid rows are
/// excluded so a NaN/empty/N=1 row can never land in a `9+` bucket.
pub fn fit_tau_by_n_bucket(rows: &[ScoredSet], target: f32) -> BTreeMap<String, TauFit> {
    let mut by_bucket: BTreeMap<String, Confidences> = BTreeMap::new();
    for r in rows {
        let Some(correct) = &r.correct else { continue };
        if !valid_row(r) {
            continue;
        }
        // first-maximum: index of the top candidate and its probability
        let mut best_idx: Option<usize> = None;
        let mut best_p = f32::NEG_INFINITY;
        for (i, (_, p)) in r.candidates.iter().enumerate() {
            if *p > best_p {
                best_p = *p;
                best_idx = Some(i);
            }
        }
        let Some(idx) = best_idx else { continue };
        let maxp = r.candidates[idx].1;
        let top_id = &r.candidates[idx].0;
        let is_correct = top_id == correct;
        by_bucket
            .entry(n_bucket(r.n))
            .or_default()
            .entry(r.head.clone())
            .or_default()
            .push((maxp, is_correct));
    }
    by_bucket
        .into_iter()
        .map(|(b, confs)| (b, fit_tau(&confs, target)))
        .collect()
}

/// The choice-set evaluation report. Serialises into the eval `metrics.json` via
/// [`crate::eval::Metrics`].
#[derive(Clone, Debug, Default, Serialize)]
pub struct SetMetrics {
    /// number of valid set rows scored with a known correct answer
    pub n: usize,
    /// number of rows excluded by validation (NaN/empty/N=1/duplicate id/non-unit sum)
    pub invalid_rows: usize,
    /// order-invariance rate over permutation groups, and worst relative deviation
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order_invariance_rate: Option<f32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order_invariance_worst_dev: Option<f32>,
    /// top-1 accuracy by number of candidates N
    pub accuracy_by_n: BTreeMap<usize, (usize, usize)>,
    /// per-N-bucket tau fit
    pub tau_by_n_bucket: BTreeMap<String, TauFit>,
    /// PROVISIONAL (needs training option-text set): novel-option accuracy
    #[serde(skip_serializing_if = "Option::is_none")]
    pub novel_option_accuracy: Option<f32>,
    /// PROVISIONAL (needs held-out skill tags): per-skill accuracy
    #[serde(skip_serializing_if = "Option::is_none")]
    pub accuracy_by_skill: Option<BTreeMap<String, (usize, usize)>>,
    /// PROVISIONAL: `max_choices` / bucket-size selection is derived from data
    /// (Task 4.6); never auto-chosen here so nothing is fabricated.
    pub max_choices_provisional: bool,
}

/// Build the set-head metrics report from scored rows. `perm_groups` may be empty
/// (no permutation study yet). `training_texts` and skill tags are optional inputs;
/// when absent the corresponding provisional fields stay `None`.
pub fn build_set_metrics(
    rows: &[ScoredSet],
    perm_groups: &BTreeMap<String, Vec<ScoredSet>>,
    training_texts: &BTreeSet<String>,
    target: f32,
) -> SetMetrics {
    let mut n = 0usize;
    let mut invalid = 0usize;
    for r in rows {
        if !valid_row(r) {
            invalid += 1;
            continue;
        }
        if r.correct.is_some() {
            n += 1;
        }
    }
    let (oir, odev) = if perm_groups.is_empty() {
        (None, None)
    } else {
        let (r, d) = order_invariance_rate(perm_groups, 1e-5);
        (Some(r), Some(d))
    };
    SetMetrics {
        n,
        invalid_rows: invalid,
        order_invariance_rate: oir,
        order_invariance_worst_dev: odev,
        accuracy_by_n: accuracy_by_n(rows),
        tau_by_n_bucket: fit_tau_by_n_bucket(rows, target),
        novel_option_accuracy: novel_option_accuracy(rows, training_texts),
        accuracy_by_skill: accuracy_by_skill(rows),
        max_choices_provisional: true,
    }
}
