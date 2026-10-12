//! Output layout under `<out>/hrm/` (what KNIRVSERVER serves):
//!
//! ```text
//! hrm_config.json            config of the promoted model
//! model-<version>.safetensors
//! model_latest -> model-<version>.safetensors     (atomic symlink swap)
//! metrics.json               metrics + gate decision of the last run
//! versions/<version>/{hrm_config.json,metrics.json}   every trained candidate
//! ```

use crate::data::hex;
use crate::eval::{Metrics, TauFit};
use knirv_cortex_wasm::HrmConfig;
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
    pub dataset_manifest_hash: &'a str,
    pub datasets: &'a [crate::data::FileDigest],
}

pub struct Store {
    pub dir: PathBuf,
}

pub struct Current {
    pub version: String,
    pub config: HrmConfig,
    pub weights: Vec<u8>,
}

impl Store {
    pub fn new(out: &Path) -> std::io::Result<Self> {
        let dir = out.join("hrm");
        fs::create_dir_all(dir.join("versions"))?;
        Ok(Self { dir })
    }

    /// The promoted model, if any.
    pub fn current(&self) -> Option<Current> {
        let link = fs::read_link(self.dir.join("model_latest")).ok()?;
        let name = link.file_name()?.to_str()?.to_string();
        let version = name.strip_prefix("model-")?.strip_suffix(".safetensors")?.to_string();
        let config = HrmConfig::from_json(&fs::read_to_string(self.dir.join("hrm_config.json")).ok()?).ok()?;
        let weights = fs::read(self.dir.join(&name)).ok()?;
        Some(Current { version, config, weights })
    }

    /// Records a candidate under `versions/`; promotion additionally publishes it.
    pub fn write(&self, cfg: &HrmConfig, weights_path: &Path, report: &Report, promote: bool) -> std::io::Result<()> {
        let v = &cfg.model_version;
        let vdir = self.dir.join("versions").join(v);
        fs::create_dir_all(&vdir)?;
        let cfg_json = serde_json::to_string_pretty(cfg).expect("config serialises");
        let report_json = serde_json::to_string_pretty(report).expect("report serialises");
        fs::write(vdir.join("hrm_config.json"), &cfg_json)?;
        fs::write(vdir.join("metrics.json"), &report_json)?;
        if !promote {
            fs::write(self.dir.join("metrics.json"), &report_json)?;
            return Ok(());
        }
        let file = format!("model-{v}.safetensors");
        fs::copy(weights_path, self.dir.join(&file))?;
        atomic_write(&self.dir.join("hrm_config.json"), cfg_json.as_bytes())?;
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

/// Promotion gate: the candidate must strictly beat the current model on the
/// same held-out set, and must not raise the false-allow rate.
pub fn decide(cand: &Metrics, cur: Option<(&str, &Metrics)>) -> Gate {
    let base = |promoted, reason: &str| Gate {
        promoted,
        reason: reason.into(),
        candidate_score: cand.score,
        current_version: cur.map(|c| c.0.to_string()),
        current_score: cur.map(|c| c.1.score),
        candidate_false_allow_rate: cand.false_allow_rate,
        current_false_allow_rate: cur.and_then(|c| c.1.false_allow_rate),
    };
    let Some((_, cm)) = cur else { return base(true, "no current model") };
    if let (Some(a), Some(b)) = (cand.false_allow_rate, cm.false_allow_rate) {
        if a > b {
            return base(false, "candidate raises the false-allow rate");
        }
    }
    if cand.score > cm.score {
        base(true, "candidate beats the current model on the held-out set")
    } else {
        base(false, "candidate does not beat the current model on the held-out set")
    }
}
