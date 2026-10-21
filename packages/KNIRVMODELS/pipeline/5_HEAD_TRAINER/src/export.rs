//! Output layout under `<out>/head/` (what KNIRVSERVER serves):
//!
//! ```text
//! head_config.json            config of the promoted model
//! model-<version>.safetensors
//! model_latest -> model-<version>.safetensors     (atomic symlink swap)
//! metrics.json               metrics + gate decision of the last run
//! versions/<version>/{head_config.json,metrics.json}   every trained candidate
//! ```

use crate::data::hex;
use crate::eval::{Evaluation, Metrics, TauFit};
use knirv_head::HeadConfig;
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::fs;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

pub fn version_for(weights: &[u8]) -> String {
    let secs = SystemTime::now().duration_since(UNIX_EPOCH).map(|d| d.as_secs()).unwrap_or(0);
    format!("v{secs}-{}", &hex(&Sha256::digest(weights))[..8])
}

#[derive(Serialize)]
pub struct Gate {
    pub promoted: bool,
    pub reason: String,
    pub candidate_score: f32,
    pub current_version: Option<String>,
    pub current_score: Option<f32>,
    pub candidate_false_allow_rate: Option<f32>,
    pub current_false_allow_rate: Option<f32>,
    /// Held-out rows scored for the candidate.
    pub heldout_n: usize,
    /// Rows scored by both models on the heads/options they share (H9).
    pub paired_n: usize,
    /// Mean per-row score gain of the candidate over the current model, and the
    /// lower end of its 95% paired-bootstrap interval (H7).
    pub delta_mean: Option<f32>,
    pub delta_lo95: Option<f32>,
    /// Heads the two models were compared on.
    pub compared_heads: Vec<String>,
}

#[derive(Serialize)]
pub struct Report<'a> {
    pub version: &'a str,
    pub sha256: String,
    pub size: usize,
    pub parameters: usize,
    pub epochs_loss: &'a [f32],
    pub validation: &'a Metrics,
    pub heldout: &'a Metrics,
    pub heldout_source: &'a str,
    pub tau_fit: &'a TauFit,
    pub gate: &'a Gate,
    pub seed: u64,
    pub init_from: Option<&'a str>,
    pub load_stats: &'a crate::data::LoadStats,
    pub dataset_manifest_hash: &'a str,
    pub datasets: &'a [crate::data::FileDigest],
}

pub struct Store {
    pub dir: PathBuf,
}

pub struct Current {
    pub version: String,
    pub config: HeadConfig,
    pub weights: Vec<u8>,
}

impl Store {
    pub fn new(out: &Path) -> std::io::Result<Self> {
        let dir = out.join("head");
        fs::create_dir_all(dir.join("versions"))?;
        Ok(Self { dir })
    }

    /// The promoted model, if any.
    pub fn current(&self) -> Option<Current> {
        let link = fs::read_link(self.dir.join("model_latest")).ok()?;
        let name = link.file_name()?.to_str()?.to_string();
        let version = name.strip_prefix("model-")?.strip_suffix(".safetensors")?.to_string();
        let config = HeadConfig::from_json(&fs::read_to_string(self.dir.join("head_config.json")).ok()?).ok()?;
        let weights = fs::read(self.dir.join(&name)).ok()?;
        Some(Current { version, config, weights })
    }

    /// Records a candidate under `versions/`; promotion additionally publishes it.
    pub fn write(&self, cfg: &HeadConfig, weights_path: &Path, report: &Report, promote: bool) -> std::io::Result<()> {
        let v = &cfg.model_version;
        let vdir = self.dir.join("versions").join(v);
        fs::create_dir_all(&vdir)?;
        let cfg_json = serde_json::to_string_pretty(cfg).expect("config serialises");
        let report_json = serde_json::to_string_pretty(report).expect("report serialises");
        fs::write(vdir.join("head_config.json"), &cfg_json)?;
        fs::write(vdir.join("metrics.json"), &report_json)?;
        if !promote {
            fs::write(self.dir.join("metrics.json"), &report_json)?;
            return Ok(());
        }
        let file = format!("model-{v}.safetensors");
        fs::copy(weights_path, self.dir.join(&file))?;
        atomic_write(&self.dir.join("head_config.json"), cfg_json.as_bytes())?;
        atomic_write(&self.dir.join("metrics.json"), report_json.as_bytes())?;
        let tmp = self.dir.join(".model_latest.tmp");
        let _ = fs::remove_file(&tmp);
        #[cfg(unix)]
        std::os::unix::fs::symlink(&file, &tmp)?;
        #[cfg(not(unix))]
        fs::copy(self.dir.join(&file), &tmp)?;
        fs::rename(tmp, self.dir.join("model_latest"))
    }
}

fn atomic_write(path: &Path, data: &[u8]) -> std::io::Result<()> {
    let tmp = path.with_extension("tmp");
    fs::write(&tmp, data)?;
    fs::rename(tmp, path)
}

/// Gate thresholds. All are CLI flags of the trainer.
#[derive(Clone, Debug)]
pub struct GateOpts {
    /// Held-out rows needed before anything is promoted.
    pub min_heldout: usize,
    /// Smallest mean per-row score gain worth a swap.
    pub min_margin: f32,
    pub bootstrap: usize,
    pub seed: u64,
}

impl Default for GateOpts {
    fn default() -> Self {
        Self { min_heldout: 30, min_margin: 0.005, bootstrap: 2000, seed: 1 }
    }
}

/// The candidate and the current model scored on the same rows, restricted to
/// the heads and options both have.
pub struct Comparison<'a> {
    pub current_version: &'a str,
    pub heads: Vec<String>,
    pub candidate: &'a Evaluation,
    pub current: &'a Evaluation,
}

fn splitmix(state: &mut u64) -> u64 {
    *state = state.wrapping_add(0x9E3779B97F4A7C15);
    let mut z = *state;
    z = (z ^ (z >> 30)).wrapping_mul(0xBF58476D1CE4E5B9);
    z = (z ^ (z >> 27)).wrapping_mul(0x94D049BB133111EB);
    z ^ (z >> 31)
}

/// Mean and 2.5th percentile of the bootstrap distribution of the mean of `d`.
pub fn bootstrap_lo95(d: &[f32], resamples: usize, seed: u64) -> (f32, f32) {
    let n = d.len();
    let mean = d.iter().sum::<f32>() / n as f32;
    let mut st = seed;
    let mut means: Vec<f32> = (0..resamples)
        .map(|_| (0..n).map(|_| d[(splitmix(&mut st) % n as u64) as usize]).sum::<f32>() / n as f32)
        .collect();
    means.sort_by(|a, b| a.total_cmp(b));
    (mean, means[((resamples as f32) * 0.025) as usize])
}

/// Promotion gate.
///
/// * Nothing is promoted without at least `min_heldout` held-out rows.
/// * First model: only if `tau_met`; a model whose tau target is unreachable
///   abstains on everything, so it must not go live.
/// * Later models: judged on the heads and options both models have, row by
///   row. Promote only if the lower 95% bound of the paired-bootstrap score gain
///   is above zero, the mean gain is at least `min_margin`, and the abstain-aware
///   false-allow rate does not rise.
pub fn decide(cand: &Metrics, tau_met: bool, heldout_n: usize, cmp: Option<Comparison>, o: &GateOpts) -> Gate {
    let mut g = Gate {
        promoted: false,
        reason: String::new(),
        candidate_score: cand.score,
        current_version: cmp.as_ref().map(|c| c.current_version.to_string()),
        current_score: cmp.as_ref().map(|c| c.current.metrics.score),
        candidate_false_allow_rate: cand.false_allow_rate,
        current_false_allow_rate: cmp.as_ref().and_then(|c| c.current.metrics.false_allow_rate),
        heldout_n,
        paired_n: 0,
        delta_mean: None,
        delta_lo95: None,
        compared_heads: vec![],
    };
    if heldout_n == 0 {
        g.reason = "no_disjoint_heldout".into();
        return g;
    }
    if heldout_n < o.min_heldout {
        g.reason = format!("held-out set has {heldout_n} rows (< {})", o.min_heldout);
        return g;
    }
    let Some(c) = cmp else {
        if tau_met {
            g.promoted = true;
            g.reason = "no current model; tau target met".into();
        } else {
            g.reason = "no current model, but the tau target was not met (the model would abstain on everything)".into();
        }
        return g;
    };
    g.compared_heads = c.heads.clone();
    g.candidate_score = c.candidate.metrics.score;
    g.candidate_false_allow_rate = c.candidate.metrics.false_allow_rate;
    if c.heads.is_empty() {
        g.reason = "no heads in common with the current model; refusing to compare".into();
        return g;
    }
    if let (Some(a), Some(b)) = (c.candidate.metrics.false_allow_rate, c.current.metrics.false_allow_rate) {
        if a > b {
            g.reason = "candidate raises the false-allow rate".into();
            return g;
        }
    }
    let cur_by_id: std::collections::BTreeMap<&str, f32> = c.current.rows.iter().map(|r| (r.id.as_str(), r.score)).collect();
    let deltas: Vec<f32> = c.candidate.rows.iter().filter_map(|r| cur_by_id.get(r.id.as_str()).map(|cs| r.score - cs)).collect();
    g.paired_n = deltas.len();
    if deltas.len() < o.min_heldout {
        g.reason = format!("only {} rows scored by both models (< {})", deltas.len(), o.min_heldout);
        return g;
    }
    let (mean, lo) = bootstrap_lo95(&deltas, o.bootstrap, o.seed);
    g.delta_mean = Some(mean);
    g.delta_lo95 = Some(lo);
    if lo > 0.0 && mean >= o.min_margin {
        g.promoted = true;
        g.reason = format!("candidate beats the current model: gain {mean:.4}, 95% lower bound {lo:.4}");
    } else {
        g.reason = format!("candidate does not clearly beat the current model: gain {mean:.4}, 95% lower bound {lo:.4} (margin {})", o.min_margin);
    }
    g
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bootstrap_separates_a_real_gain_from_noise() {
        let real: Vec<f32> = (0..200).map(|i| if i % 4 == 0 { 0.0 } else { 0.2 }).collect();
        assert!(bootstrap_lo95(&real, 1000, 1).1 > 0.0);
        let noise: Vec<f32> = (0..200).map(|i| if i % 2 == 0 { 0.5 } else { -0.5 }).collect();
        assert!(bootstrap_lo95(&noise, 1000, 1).1 < 0.0);
        let tiny: Vec<f32> = vec![0.1, -0.05, 0.2];
        assert!(bootstrap_lo95(&tiny, 1000, 1).1 < 0.1);
    }

    #[test]
    fn first_model_needs_tau_and_enough_rows() {
        let m = Metrics { score: 0.9, ..Default::default() };
        let o = GateOpts::default();
        assert!(!decide(&m, false, 100, None, &o).promoted, "tau not met");
        assert!(!decide(&m, true, 5, None, &o).promoted, "too few rows");
        assert_eq!(decide(&m, true, 0, None, &o).reason, "no_disjoint_heldout");
        assert!(decide(&m, true, 100, None, &o).promoted);
    }
}
