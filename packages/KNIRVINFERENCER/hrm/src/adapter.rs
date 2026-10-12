//! Applying a KNIRVULORA adapter to the HRM.
//!
//! KNIRVULORA compiles a portable canonical core and binds it to a model with
//! that model's connector (`POST /ulora/v1/bind`), giving per-role low-rank
//! factors in the model's own frame: `A` is r × d_in, `B` is d_out × r. This
//! module merges them into the base weights, `W' = W + (alpha / r) · strength · B·A`,
//! once at load time, so inference cost and the model surface are unchanged.
//!
//! Roles are semantic ("self_attn.q_proj"), never layer indices; a role applies
//! to the matching projection in every block of the selected stacks. The HRM's
//! attention projections are d × d, which is what the binder produces. MLP roles
//! do not fit (the binder projects through hidden_size only), so they are
//! reported as skipped rather than guessed at.

use crate::config::HrmConfig;
use candle_core::{DType, Device, Result, Tensor};
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, HashMap};

#[derive(Debug, Clone, Deserialize)]
pub struct AdapterLayer {
    pub a: Vec<Vec<f32>>,
    pub b: Vec<Vec<f32>>,
}

/// The `bind` response (extra fields are ignored) plus optional local controls.
#[derive(Debug, Clone, Deserialize)]
pub struct Adapter {
    pub adapter_rank: usize,
    pub scaling_alpha: f64,
    pub layers: BTreeMap<String, AdapterLayer>,
    #[serde(default)]
    pub content_hash: String,
    /// Which stacks to adapt: any of "l", "h". Default both.
    #[serde(default = "both")]
    pub stacks: Vec<String>,
    /// Multiplier on the merged delta; 0 disables, 1 is the compiled strength.
    #[serde(default = "one")]
    pub strength: f32,
}

fn both() -> Vec<String> {
    vec!["l".into(), "h".into()]
}
fn one() -> f32 {
    1.0
}

#[derive(Debug, Clone, Serialize, Default)]
pub struct Applied {
    pub content_hash: String,
    pub tensors_changed: usize,
    pub roles_applied: Vec<String>,
    pub roles_skipped: Vec<String>,
}

/// Role → the per-block tensor it adapts. The layer name inside a block.
pub fn role_target(role: &str) -> Option<&'static str> {
    let r = role.strip_prefix("self_attn.").unwrap_or(role);
    match r {
        "q_proj" | "attn.q" => Some("attn.q"),
        "k_proj" | "attn.k" => Some("attn.k"),
        "v_proj" | "attn.v" => Some("attn.v"),
        "o_proj" | "attn.o" => Some("attn.o"),
        _ => None,
    }
}

/// Every adaptable weight name in this model, for the bridge's mapping table.
pub fn adaptable_tensors(cfg: &HrmConfig) -> Vec<String> {
    let mut v = Vec::new();
    for (stack, n) in [("l", cfg.l_layers), ("h", cfg.h_layers)] {
        for i in 0..n {
            for t in ["attn.q", "attn.k", "attn.v", "attn.o"] {
                v.push(format!("{stack}.{i}.{t}.weight"));
            }
        }
    }
    v
}

fn matrix(rows: &[Vec<f32>], r: usize, c: usize, what: &str, dev: &Device) -> Result<Tensor> {
    if rows.len() != r || rows.iter().any(|x| x.len() != c) {
        candle_core::bail!("{what} must be {r}x{c}, got {}x{}", rows.len(), rows.first().map_or(0, |x| x.len()));
    }
    let flat: Vec<f32> = rows.iter().flatten().copied().collect();
    if flat.iter().any(|x| !x.is_finite()) {
        candle_core::bail!("{what} contains a non-finite value");
    }
    Tensor::from_vec(flat, (r, c), dev)
}

/// Merge `ad` into `tensors` (named as in the safetensors file). Nothing is
/// modified unless every applicable role validates.
pub fn merge(cfg: &HrmConfig, tensors: &mut HashMap<String, Tensor>, ad: &Adapter) -> Result<Applied> {
    let d = cfg.d_model;
    let r = ad.adapter_rank;
    if r == 0 || ad.scaling_alpha <= 0.0 || !ad.strength.is_finite() {
        candle_core::bail!("adapter needs a positive rank and alpha");
    }
    if let Some(s) = ad.stacks.iter().find(|s| *s != "l" && *s != "h") {
        candle_core::bail!("unknown stack {s:?} (use \"l\" or \"h\")");
    }
    let scale = (ad.scaling_alpha / r as f64) as f32 * ad.strength;
    let dev = tensors.values().next().map(|t| t.device().clone()).unwrap_or(Device::Cpu);

    let mut applied = Applied { content_hash: ad.content_hash.clone(), ..Default::default() };
    let mut updates: Vec<(String, Tensor)> = Vec::new();
    for (role, layer) in &ad.layers {
        let Some(target) = role_target(role) else {
            applied.roles_skipped.push(role.clone());
            continue;
        };
        let a = matrix(&layer.a, r, d, &format!("{role}.a"), &dev)?;
        let b = matrix(&layer.b, d, r, &format!("{role}.b"), &dev)?;
        let delta = (b.matmul(&a)? * scale as f64)?; // d_out × d_in, same orientation as the stored weight
        for (stack, n) in [("l", cfg.l_layers), ("h", cfg.h_layers)] {
            if !ad.stacks.iter().any(|s| s == stack) {
                continue;
            }
            for i in 0..n {
                let name = format!("{stack}.{i}.{target}.weight");
                let w = tensors.get(&name).ok_or_else(|| candle_core::Error::Msg(format!("weights have no tensor {name}")))?;
                if w.dims() != [d, d] {
                    candle_core::bail!("{name} is {:?}, expected [{d}, {d}]", w.dims());
                }
                updates.push((name, (w.to_dtype(DType::F32)? + &delta)?));
            }
        }
        applied.roles_applied.push(role.clone());
    }
    if applied.roles_applied.is_empty() {
        candle_core::bail!(
            "adapter has no role this model can take (attention q/k/v/o projections); skipped: {:?}",
            applied.roles_skipped
        );
    }
    applied.tensors_changed = updates.len();
    tensors.extend(updates);
    Ok(applied)
}
