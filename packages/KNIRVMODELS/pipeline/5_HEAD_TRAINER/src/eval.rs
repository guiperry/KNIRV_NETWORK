use crate::batch;
use crate::data::Example;
use candle_core::{Device, Result, D};
use candle_nn::ops;
use knirv_head::tokenizer::PAD;
use knirv_head::{DecisionHead, HeadKind, Head, Mode};
use serde::Serialize;
use std::collections::{BTreeMap, BTreeSet};

#[derive(Clone, Debug, Default, Serialize)]
pub struct HeadMetrics {
    pub n: usize,
    /// choice: top-1 vs argmax of the (possibly soft) target
    #[serde(skip_serializing_if = "Option::is_none")]
    pub accuracy: Option<f32>,
    /// choice: multi-class Brier against the target distribution
    #[serde(skip_serializing_if = "Option::is_none")]
    pub brier: Option<f32>,
    /// choice: expected calibration error, 10 bins on max(p)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ece: Option<f32>,
    /// score: mean squared error
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mse: Option<f32>,
    /// choice: share of decisions answered (max p >= tau) and their accuracy (H6).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub coverage: Option<f32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub selective_accuracy: Option<f32>,
}

#[derive(Clone, Debug, Default, Serialize)]
pub struct Metrics {
    pub examples: usize,
    pub plan_token_accuracy: Option<f32>,
    pub plan_exact_match: Option<f32>,
    pub heads: BTreeMap<String, HeadMetrics>,
    /// Share of non-`allow` labels the `action` head answered `allow` to AND did
    /// not abstain on (max p >= tau): what a deployed advisor would let through.
    /// The supervisor scorecard requires 0.
    pub false_allow_rate: Option<f32>,
    /// The same without the abstention, for comparison with older reports.
    pub false_allow_rate_raw: Option<f32>,
    /// Mean cycles per example, run at batch 1 like serving (H6).
    pub mean_halting_steps: f32,
    /// The tau these numbers were computed with.
    pub tau: f32,
    /// Choice decisions answered at `tau`, and their accuracy.
    pub coverage: Option<f32>,
    pub selective_accuracy: Option<f32>,
    /// Mean of plan accuracy, each choice head's accuracy and each score head's
    /// 1-mse over what the set supervises. Higher is better.
    pub score: f32,
}

/// (max probability, top-1 correct) for one head, used to fit tau.
pub type Confidences = BTreeMap<String, Vec<(f32, bool)>>;

/// Which heads and which label options a comparison may use (H9). A head absent
/// from the map is skipped; for a choice head only rows whose top label is in
/// the set are scored.
pub type Restrict = BTreeMap<String, BTreeSet<String>>;

/// One example's score: mean over what it supervises. The unit of the paired
/// bootstrap in the promotion gate (H7).
#[derive(Clone, Debug)]
pub struct RowScore {
    pub id: String,
    pub score: f32,
}

pub struct Evaluation {
    pub metrics: Metrics,
    pub confs: Confidences,
    pub rows: Vec<RowScore>,
}

/// Evaluates with tau = 0 and no restriction. Kept for callers that only need
/// the metrics and the confidences to fit tau.
pub fn evaluate(model: &Head, ex: &[&Example], width: usize, dev: &Device) -> Result<(Metrics, Confidences)> {
    let e = evaluate_with(model, ex, width, 0.0, None, dev)?;
    Ok((e.metrics, e.confs))
}

/// Runs every example at batch 1, as serving does, so halting depth is per
/// example and the metrics honour `tau` (H6).
pub fn evaluate_with(model: &Head, ex: &[&Example], width: usize, tau: f32, restrict: Option<&Restrict>, dev: &Device) -> Result<Evaluation> {
    if ex.is_empty() {
        candle_core::bail!("evaluation set is empty");
    }
    let heads: Vec<DecisionHead> = model.cfg.decision_heads.clone();
    let mut m = Metrics { examples: ex.len(), tau, ..Default::default() };
    let mut confs = Confidences::new();
    let (mut tok_ok, mut tok_n, mut exact_ok, mut exact_n) = (0usize, 0usize, 0usize, 0usize);
    let mut halt_sum = 0usize;
    // per head: correct, brier, mse, kept-correct, conf samples
    let mut acc: BTreeMap<String, (Vec<f32>, Vec<f32>, Vec<f32>, Vec<f32>, Vec<(f32, bool)>)> = BTreeMap::new();
    let (mut fa, mut fa_raw, mut non_allow) = (0usize, 0usize, 0usize);
    let mut rows = Vec::with_capacity(ex.len());

    for e in ex {
        let chunk = [*e];
        let b = batch::make(&chunk, &heads, width, dev)?;
        let tr = model.forward(&b.x, Mode::Infer)?;
        halt_sum += tr.cycles;
        let mut row_parts: Vec<f32> = Vec::new();

        if let Some(t) = &e.plan {
            let p = &tr.logits.argmax(D::Minus1)?.to_vec2::<u32>()?[0];
            let (mut all, mut any, mut ok, mut n) = (true, false, 0usize, 0usize);
            for (a, &g) in p.iter().zip(t) {
                if g != PAD {
                    any = true;
                    n += 1;
                    if *a == g {
                        ok += 1;
                    } else {
                        all = false;
                    }
                }
            }
            tok_ok += ok;
            tok_n += n;
            exact_n += 1;
            exact_ok += (all && any) as usize;
            if n > 0 {
                row_parts.push(ok as f32 / n as f32);
            }
        }

        for ((h, hb), out) in heads.iter().zip(&b.heads).zip(model.head_outputs(&tr.summary)?) {
            let allowed = match restrict {
                Some(r) => match r.get(&h.name) {
                    Some(a) => Some(a),
                    None => continue,
                },
                None => None,
            };
            let mask = hb.mask.to_vec1::<f32>()?;
            if mask[0] < 0.5 {
                continue;
            }
            let entry = acc.entry(h.name.clone()).or_default();
            match h.kind {
                HeadKind::Choice => {
                    let p = ops::softmax(&out, D::Minus1)?.to_vec2::<f32>()?.remove(0);
                    let t = hb.target.to_vec2::<f32>()?.remove(0);
                    let (pi, ti) = (argmax(&p), argmax(&t));
                    if let Some(a) = allowed {
                        if !a.contains(&h.options[ti]) {
                            continue;
                        }
                    }
                    let correct = pi == ti;
                    let kept = p[pi] >= tau;
                    entry.0.push(correct as u8 as f32);
                    entry.1.push(p.iter().zip(&t).map(|(a, b)| (a - b) * (a - b)).sum());
                    if kept {
                        entry.3.push(correct as u8 as f32);
                    }
                    entry.4.push((p[pi], correct));
                    row_parts.push(correct as u8 as f32);
                    if h.name == "action" {
                        if let Some(allow) = h.options.iter().position(|o| o == "allow") {
                            if ti != allow {
                                non_allow += 1;
                                fa_raw += (pi == allow) as usize;
                                fa += (pi == allow && kept) as usize;
                            }
                        }
                    }
                }
                HeadKind::Score => {
                    let p = ops::sigmoid(&out)?.squeeze(1)?.to_vec1::<f32>()?[0];
                    let t = hb.target.to_vec1::<f32>()?[0];
                    entry.2.push((p - t).powi(2));
                    row_parts.push(1.0 - (p - t).powi(2));
                }
            }
        }
        if !row_parts.is_empty() {
            rows.push(RowScore { id: e.id.clone(), score: row_parts.iter().sum::<f32>() / row_parts.len() as f32 });
        }
    }

    let mean = |v: &[f32]| v.iter().sum::<f32>() / v.len() as f32;
    let mut parts = Vec::new();
    if tok_n > 0 {
        m.plan_token_accuracy = Some(tok_ok as f32 / tok_n as f32);
        m.plan_exact_match = Some(exact_ok as f32 / exact_n.max(1) as f32);
        parts.push(tok_ok as f32 / tok_n as f32);
    }
    let (mut kept_total, mut kept_ok, mut dec_total) = (0usize, 0f32, 0usize);
    for h in &heads {
        let Some((correct, brier, mse, kept, conf)) = acc.get(&h.name) else { continue };
        let mut hm = HeadMetrics::default();
        match h.kind {
            HeadKind::Choice if !correct.is_empty() => {
                hm.n = correct.len();
                hm.accuracy = Some(mean(correct));
                hm.brier = Some(mean(brier));
                hm.ece = Some(ece(conf));
                hm.coverage = Some(kept.len() as f32 / correct.len() as f32);
                hm.selective_accuracy = if kept.is_empty() { None } else { Some(mean(kept)) };
                kept_total += kept.len();
                kept_ok += kept.iter().sum::<f32>();
                dec_total += correct.len();
                parts.push(mean(correct));
                confs.insert(h.name.clone(), conf.clone());
            }
            HeadKind::Score if !mse.is_empty() => {
                hm.n = mse.len();
                hm.mse = Some(mean(mse));
                parts.push(1.0 - mean(mse));
            }
            _ => continue,
        }
        m.heads.insert(h.name.clone(), hm);
    }
    if dec_total > 0 {
        m.coverage = Some(kept_total as f32 / dec_total as f32);
        m.selective_accuracy = if kept_total > 0 { Some(kept_ok / kept_total as f32) } else { None };
    }
    if non_allow > 0 {
        m.false_allow_rate = Some(fa as f32 / non_allow as f32);
        m.false_allow_rate_raw = Some(fa_raw as f32 / non_allow as f32);
    }
    m.mean_halting_steps = halt_sum as f32 / ex.len() as f32;
    m.score = if parts.is_empty() { 0.0 } else { mean(&parts) };
    Ok(Evaluation { metrics: m, confs, rows })
}

fn argmax(v: &[f32]) -> usize {
    v.iter().enumerate().max_by(|a, b| a.1.total_cmp(b.1)).map(|x| x.0).unwrap_or(0)
}

pub fn ece(samples: &[(f32, bool)]) -> f32 {
    let mut bins = [(0usize, 0f32, 0usize); 10];
    for &(c, ok) in samples {
        let i = ((c * 10.0) as usize).min(9);
        bins[i].0 += 1;
        bins[i].1 += c;
        bins[i].2 += ok as usize;
    }
    let n = samples.len().max(1) as f32;
    bins.iter().filter(|b| b.0 > 0).map(|&(k, c, ok)| (k as f32 / n) * ((c / k as f32) - (ok as f32 / k as f32)).abs()).sum()
}

#[derive(Clone, Debug, Serialize)]
pub struct TauFit {
    pub tau: f32,
    /// Whether any threshold reached the target selective accuracy on every head.
    pub met: bool,
    pub target_accuracy: f32,
    /// Share of decisions kept (not abstained) at `tau`.
    pub coverage: f32,
    pub selective_accuracy: f32,
}

/// Smallest `tau` such that, on every choice head, the decisions with
/// max(p) >= tau are at least `target` accurate. If none exists, `tau = 1.0`
/// (abstain on everything below certainty) and `met = false`. The model never
/// outputs an `abstain` label; this threshold is the only abstention path.
pub fn fit_tau(confs: &Confidences, target: f32) -> TauFit {
    let total: usize = confs.values().map(Vec::len).sum();
    let mut cands: Vec<f32> = confs.values().flatten().map(|c| c.0).collect();
    cands.push(0.0);
    cands.sort_by(|a, b| a.total_cmp(b));
    cands.dedup();
    for &t in &cands {
        let (mut kept, mut ok, mut feasible) = (0usize, 0usize, true);
        for v in confs.values() {
            let (k, o) = v.iter().filter(|c| c.0 >= t).fold((0, 0), |(k, o), c| (k + 1, o + c.1 as usize));
            if k > 0 && (o as f32 / k as f32) < target {
                feasible = false;
                break;
            }
            kept += k;
            ok += o;
        }
        if feasible && kept > 0 {
            return TauFit { tau: t, met: true, target_accuracy: target, coverage: kept as f32 / total as f32, selective_accuracy: ok as f32 / kept as f32 };
        }
    }
    TauFit { tau: 1.0, met: false, target_accuracy: target, coverage: 0.0, selective_accuracy: 0.0 }
}
