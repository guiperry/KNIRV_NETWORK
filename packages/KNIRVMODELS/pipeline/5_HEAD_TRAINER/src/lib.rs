//! KNIRVMODELS stage 5: HEAD trainer. See `main.rs` for the CLI.

pub mod batch;
pub mod data;
pub mod episodes;
pub mod eval;
pub mod export;
pub mod setmetrics;
pub mod setprobe;
pub mod synth;
pub mod train;

use knirv_heart_decision::tokenizer;
use knirv_heart_decision::{DecisionHead, HeadConfig, SetHead};

/// Architecture presets. `base` is the ≈27M-parameter target.
pub fn preset(name: &str, heads: Vec<DecisionHead>) -> Result<HeadConfig, String> {
    let (d, nh, ffn, l, h, n, t, seq) = match name {
        "micro" => (64, 2, 2, 1, 1, 2, 2, 32),
        "tiny" => (64, 2, 2, 1, 1, 2, 2, 64),
        "small" => (128, 4, 2, 2, 2, 3, 2, 160),
        "base" => (512, 8, 4, 3, 3, 4, 3, 256),
        // "512" = the Task 4.4 fallback: base dimensions with a 512-token sequence window
        // (the HEAD plan's intended set-head width), giving the set scorer room for 4..=8
        // candidates plus the question/state.
        "512" => (512, 8, 4, 3, 3, 4, 3, 512),
        other => return Err(format!("unknown preset {other:?} (micro|tiny|small|base|512)")),
    };
    let cfg = HeadConfig {
        format: knirv_heart_decision::config::FORMAT.into(),
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
        choice_set_heads: Vec::new(),
        dropout: 0.0,
        set_match_feature: false,
    };
    cfg.validate()?;
    Ok(cfg)
}

/// As [`preset`], but for the `knirv-heart/v2` set-scorer format: `heads` are the decision
/// heads and `sets` the choice-set heads (Task 4.3/4.4).
pub fn preset_v2(name: &str, heads: Vec<DecisionHead>, sets: Vec<SetHead>) -> Result<HeadConfig, String> {
    let mut cfg = preset(name, heads)?;
    cfg.format = knirv_heart_decision::config::FORMAT_V2.into();
    cfg.choice_set_heads = sets;
    cfg.validate()?;
    Ok(cfg)
}
