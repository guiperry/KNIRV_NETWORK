//! Task 4.5 — choice-set evaluation additions.
//!
//! Pure functions over set-head outputs. A `ScoredSet` is built from a
//! [`knirv_heart_decision::model::Reasoning`] / `Decision` / `Candidate` by
//! re-keying each candidate's probability by its stable `id` (so the metrics are
//! permutation/order independent). The functions are unit-tested with synthetic
//! fixtures in `tests/setmetrics_test.rs`.
//!
//! Data-dependent pieces that need a trained 4.4 model and held-out/gold data are
//! deliberately NOT fabricated here:
//!   * `novel_option_accuracy` needs the training-split id set (Task 4.6 / gold).
//!   * per-skill tagging and `max_choices` / bucket-size selection need held-out
//!     rows (Task 4.6). The functions accept those inputs and compute when present,
//!     but no synthetic "gold" is invented and `max_choices` is never auto-chosen.
//!
//! The wiring that turns a live model's `reason_sets` output into `ScoredSet` rows
//! lives in the set-scorer evaluation path (see `setprobe`); it is exercised only
//! once a model with `choice_set_heads` exists (Task 4.4 kill gate). Until then the
//! fields below stay `None` and the report's set block is empty.

use crate::eval::{fit_tau, Confidences, TauFit};
use serde::Serialize;
use std::collections::{BTreeMap, BTreeSet};

/// One evaluated choice set. Probabilities are re-keyed by candidate `id`.
#[derive(Clone, Debug, Default)]
pub struct ScoredSet {
    pub head: String,
    pub n: usize,
    /// candidate id -> probability (order independent)
    pub probs: BTreeMap<String, f32>,
    /// gold/held-out correct candidate id, if known
    pub correct: Option<String>,
    /// held-out skill tag for the row, if known
    pub skill: Option<String>,
}

/// Relative deviation between two probabilities, robust to near-zero values.
fn rel_dev(a: f32, b: f32) -> f32 {
    let denom = a.abs().max(b.abs()).max(1e-12);
    (a - b).abs() / denom
}

/// Order-invariance rate over permutation groups.
///
/// `groups` maps a "set identity" (the same underlying choice set, e.g. the sorted
/// id list) to the several `ScoredSet`s produced by permuting the request's option
/// order. A group is order-invariant iff every candidate `id`'s probability is
/// equal across its permutations within `tol` (relative, the plan's 1e-5).
///
/// Returns `(rate, worst_relative_deviation)`: the fraction of invariant groups and
/// the largest relative deviation observed across all groups.
pub fn order_invariance_rate(groups: &BTreeMap<String, Vec<ScoredSet>>, tol: f32) -> (f32, f32) {
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
        let base = &perms[0].probs;
        let mut group_ok = true;
        let mut group_worst = 0.0f32;
        for (id, &p0) in base {
            for perm in &perms[1..] {
                match perm.probs.get(id) {
                    Some(&p1) => {
                        let dev = rel_dev(p0, p1);
                        if dev > group_worst {
                            group_worst = dev;
                        }
                        if dev > tol {
                            group_ok = false;
                        }
                    }
                    None => group_ok = false, // id missing under a permutation
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
/// Only rows with a known `correct` contribute. Returns `N -> (correct, total)`.
pub fn accuracy_by_n(rows: &[ScoredSet]) -> BTreeMap<usize, (usize, usize)> {
    let mut out: BTreeMap<usize, (usize, usize)> = BTreeMap::new();
    for r in rows {
        let Some(correct) = &r.correct else { continue };
        let top = r.probs.iter().max_by(|a, b| a.1.total_cmp(b.1)).map(|(id, _)| id);
        let entry = out.entry(r.n).or_insert((0, 0));
        entry.1 += 1;
        if top == Some(correct) {
            entry.0 += 1;
        }
    }
    out
}

/// Top-1 accuracy on rows whose correct option `id` was NOT in `training_ids`.
///
/// PROVISIONAL (needs the training-split id set from Task 4.6 / gold data). Returns
/// `None` when `training_ids` is empty so a baseline is never fabricated from
/// nothing.
pub fn novel_option_accuracy(rows: &[ScoredSet], training_ids: &BTreeSet<String>) -> Option<f32> {
    if training_ids.is_empty() {
        return None;
    }
    let mut ok = 0usize;
    let mut n = 0usize;
    for r in rows {
        let Some(correct) = &r.correct else { continue };
        if training_ids.contains(correct) {
            continue;
        }
        n += 1;
        let top = r.probs.iter().max_by(|a, b| a.1.total_cmp(b.1)).map(|(id, _)| id);
        if top == Some(correct) {
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
/// (held-out only).
pub fn accuracy_by_skill(rows: &[ScoredSet]) -> Option<BTreeMap<String, (usize, usize)>> {
    let mut out: BTreeMap<String, (usize, usize)> = BTreeMap::new();
    let mut any = false;
    for r in rows {
        let Some(skill) = &r.skill else { continue };
        let Some(correct) = &r.correct else { continue };
        any = true;
        let top = r.probs.iter().max_by(|a, b| a.1.total_cmp(b.1)).map(|(id, _)| id);
        let entry = out.entry(skill.clone()).or_insert((0, 0));
        entry.1 += 1;
        if top == Some(correct) {
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
/// selective accuracy `>= target`. Reuses [`crate::eval::fit_tau`].
pub fn fit_tau_by_n_bucket(rows: &[ScoredSet], target: f32) -> BTreeMap<String, TauFit> {
    // bucket label -> (head -> Vec<(max_p, correct)>)
    let mut by_bucket: BTreeMap<String, Confidences> = BTreeMap::new();
    for r in rows {
        let Some(correct) = &r.correct else { continue };
        let Some((_, &maxp)) = r.probs.iter().max_by(|a, b| a.1.total_cmp(b.1)) else {
            continue;
        };
        let top = r.probs.iter().max_by(|a, b| a.1.total_cmp(b.1)).map(|(id, _)| id);
        let is_correct = top == Some(correct);
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
    /// number of set rows scored with a known correct answer
    pub n: usize,
    /// order-invariance rate over permutation groups, and worst relative deviation
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order_invariance_rate: Option<f32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order_invariance_worst_dev: Option<f32>,
    /// top-1 accuracy by number of candidates N
    pub accuracy_by_n: BTreeMap<usize, (usize, usize)>,
    /// per-N-bucket tau fit
    pub tau_by_n_bucket: BTreeMap<String, TauFit>,
    /// PROVISIONAL (needs training split): novel-option accuracy
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
/// (no permutation study yet). `training_ids` and skill tags are optional inputs;
/// when absent the corresponding provisional fields stay `None`.
pub fn build_set_metrics(
    rows: &[ScoredSet],
    perm_groups: &BTreeMap<String, Vec<ScoredSet>>,
    training_ids: &BTreeSet<String>,
    target: f32,
) -> SetMetrics {
    let n = rows.iter().filter(|r| r.correct.is_some()).count();
    let (oir, odev) = if perm_groups.is_empty() {
        (None, None)
    } else {
        let (r, d) = order_invariance_rate(perm_groups, 1e-5);
        (Some(r), Some(d))
    };
    SetMetrics {
        n,
        order_invariance_rate: oir,
        order_invariance_worst_dev: odev,
        accuracy_by_n: accuracy_by_n(rows),
        tau_by_n_bucket: fit_tau_by_n_bucket(rows, target),
        novel_option_accuracy: novel_option_accuracy(rows, training_ids),
        accuracy_by_skill: accuracy_by_skill(rows),
        max_choices_provisional: true,
    }
}
