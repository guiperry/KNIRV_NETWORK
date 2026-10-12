//! KNIRVMODELS stage 5: HEAD trainer. See `main.rs` for the CLI.

pub mod batch;
pub mod data;
pub mod episodes;
pub mod eval;
pub mod export;
pub mod train;

use knirv_head::tokenizer;
use knirv_head::{DecisionHead, HeadConfig};

/// Architecture presets. `base` is the ≈27M-parameter target.
pub fn preset(name: &str, heads: Vec<DecisionHead>) -> Result<HeadConfig, String> {
    let (d, nh, ffn, l, h, n, t, seq) = match name {
        "micro" => (64, 2, 2, 1, 1, 2, 2, 32),
        "tiny" => (64, 2, 2, 1, 1, 2, 2, 64),
        "small" => (128, 4, 2, 2, 2, 3, 2, 160),
        "base" => (512, 8, 4, 3, 3, 4, 3, 256),
        other => return Err(format!("unknown preset {other:?} (micro|tiny|small|base)")),
    };
    let cfg = HeadConfig {
        format: knirv_head::config::FORMAT.into(),
        tokenizer: tokenizer::NAME.into(),
        vocab_size: tokenizer::VOCAB,
        max_seq_len: seq,
        d_model: d,
        n_heads: nh,
        ffn_mult: ffn,
        l_layers: l,
        h_layers: h,
        h_cycles: n,
        l_steps: t,
        min_cycles: 1,
        summary_slots: 1,
        decision_heads: heads,
        tau: 0.0,
        model_version: String::new(),
        dataset_manifest_hash: String::new(),
        policy_bundle_hash: String::new(),
    };
    cfg.validate()?;
    Ok(cfg)
}
