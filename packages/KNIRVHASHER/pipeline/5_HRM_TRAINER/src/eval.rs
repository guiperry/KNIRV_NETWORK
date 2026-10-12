use crate::batch;
use crate::data::Example;
use candle_core::{Device, Result, D};
use candle_nn::ops;
use knirv_cortex_wasm::tokenizer::PAD;
use knirv_cortex_wasm::{DecisionHead, HeadKind, Hrm, Mode};
use serde::Serialize;
use std::collections::BTreeMap;

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
}

#[derive(Clone, Debug, Default, Serialize)]
pub struct Metrics {
    pub examples: usize,
    pub plan_token_accuracy: Option<f32>,
    pub plan_exact_match: Option<f32>,
    pub heads: BTreeMap<String, HeadMetrics>,
    /// Share of non-`allow` labels the `action` head answered `allow` to. The
    /// supervisor scorecard requires 0.
    pub false_allow_rate: Option<f32>,
    pub mean_halting_steps: f32,
    /// Mean of plan accuracy, each choice head's accuracy and each score head's
    /// 1-mse over what the set supervises. Higher is better.
    pub score: f32,
}

/// (max probability, top-1 correct) for one head, used to fit tau.
pub type Confidences = BTreeMap<String, Vec<(f32, bool)>>;

pub fn evaluate(model: &Hrm, ex: &[&Example], width: usize, dev: &Device) -> Result<(Metrics, Confidences)> {
    if ex.is_empty() {
        candle_core::bail!("evaluation set is empty");
    }
    let heads: Vec<DecisionHead> = model.cfg.decision_heads.clone();
    let mut m = Metrics { examples: ex.len(), ..Default::default() };
    let mut confs = Confidences::new();
    let (mut tok_ok, mut tok_n, mut exact_ok, mut exact_n) = (0usize, 0usize, 0usize, 0usize);
    let (mut halt_sum, mut halt_batches) = (0usize, 0usize);
    let mut acc: BTreeMap<String, (Vec<f32>, Vec<f32>, Vec<f32>, Vec<f32>, Vec<(f32, bool)>)> = BTreeMap::new();
    let (mut fa, mut non_allow) = (0usize, 0usize);

    for chunk in ex.chunks(32) {
        let b = batch::make(chunk, &heads, width, dev)?;
        let tr = model.forward(&b.x, Mode::Infer)?;
        halt_sum += tr.cycles;
        halt_batches += 1;

        let pred = tr.logits.argmax(D::Minus1)?.to_vec2::<u32>()?;
        for (e, p) in chunk.iter().zip(&pred) {
            if let Some(t) = &e.plan {
                let (mut all, mut any) = (true, false);
                for (a, &g) in p.iter().zip(t) {
                    if g != PAD {
                        any = true;
                        tok_n += 1;
                        if *a == g {
                            tok_ok += 1;
                        } else {
                            all = false;
                        }
                    }
                }
                exact_n += 1;
                exact_ok += (all && any) as usize;
            }
        }

        for ((h, hb), out) in heads.iter().zip(&b.heads).zip(model.head_outputs(&tr.summary)?) {
            let mask = hb.mask.to_vec1::<f32>()?;
            let entry = acc.entry(h.name.clone()).or_default();
            match h.kind {
                HeadKind::Choice => {
                    let p = ops::softmax(&out, D::Minus1)?.to_vec2::<f32>()?;
                    let t = hb.target.to_vec2::<f32>()?;
                    for i in 0..p.len() {
                        if mask[i] < 0.5 {
                            continue;
                        }
                        let (pi, ti) = (argmax(&p[i]), argmax(&t[i]));
                        let correct = pi == ti;
                        entry.0.push(correct as u8 as f32);
                        entry.1.push(p[i].iter().zip(&t[i]).map(|(a, b)| (a - b) * (a - b)).sum());
                        entry.4.push((p[i][pi], correct));
                        if h.name == "action" {
                            if let Some(allow) = h.options.iter().position(|o| o == "allow") {
                                if ti != allow {
                                    non_allow += 1;
                                    fa += (pi == allow) as usize;
                                }
                            }
                        }
                    }
                }
                HeadKind::Score => {
                    let p = ops::sigmoid(&out)?.squeeze(1)?.to_vec1::<f32>()?;
                    let t = hb.target.to_vec1::<f32>()?;
                    for i in 0..p.len() {
                        if mask[i] >= 0.5 {
                            entry.2.push((p[i] - t[i]).powi(2));
                        }
                    }
                }
            }
        }
    }

    let mean = |v: &[f32]| v.iter().sum::<f32>() / v.len() as f32;
    let mut parts = Vec::new();
    if tok_n > 0 {
        m.plan_token_accuracy = Some(tok_ok as f32 / tok_n as f32);
        m.plan_exact_match = Some(exact_ok as f32 / exact_n.max(1) as f32);
        parts.push(tok_ok as f32 / tok_n as f32);
    }
    for h in &heads {
        let (correct, brier, mse, _, conf) = &acc[&h.name];
        let mut hm = HeadMetrics::default();
        match h.kind {
            HeadKind::Choice if !correct.is_empty() => {
                hm.n = correct.len();
                hm.accuracy = Some(mean(correct));
                hm.brier = Some(mean(brier));
                hm.ece = Some(ece(conf));
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
    if non_allow > 0 {
        m.false_allow_rate = Some(fa as f32 / non_allow as f32);
    }
    m.mean_halting_steps = halt_sum as f32 / halt_batches.max(1) as f32;
    m.score = if parts.is_empty() { 0.0 } else { mean(&parts) };
    Ok((m, confs))
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
