//! F1 (Codex audit, mig_phase5_items8_13): legacy on-disk format compatibility.
//!
//! The migration moved the decision-config format id from `knirv-head/v1|v2` to
//! `knirv-heart/v1|v2` and kept `knirv-head/v1|v2` as load-only backward
//! compatibility. These tests prove that already-persisted `knirv-head` weights
//! keep loading, that legacy v2 (with `choice_set_heads`) still loads and scores,
//! and that freshly-created/persisted configs use the new `knirv-heart` ids while
//! legacy ids remain load-only inputs (never rewritten on parse).

use candle_core::{DType, Device};
use candle_nn::{VarBuilder, VarMap};
use knirv_heart_decision::config::{FORMAT, FORMAT_V2};
use knirv_heart_decision::tokenizer;
use knirv_heart_decision::{
    DecisionHead, Head, HeadConfig, HeadKind, SetChoice, SetHead, SetKind, SetQuestion,
};
use std::path::Path;

fn read(rel: &str) -> Vec<u8> {
    std::fs::read(Path::new(env!("CARGO_MANIFEST_DIR")).join(rel)).unwrap()
}

/// Deterministic (default) weights for a config, serialised as safetensors — enough
/// to prove a config + weights pair loads and reasons. `Head::new` only builds the
/// parameter set, so the saved tensor set matches the architecture exactly.
fn weights_for(c: &HeadConfig) -> Vec<u8> {
    let vm = VarMap::new();
    Head::new(c.clone(), VarBuilder::from_varmap(&vm, DType::F32, &Device::Cpu)).unwrap();
    let path = std::env::temp_dir().join(format!("legacy-rt-{}.safetensors", std::process::id()));
    vm.save(&path).unwrap();
    let bytes = std::fs::read(&path).unwrap();
    let _ = std::fs::remove_file(&path);
    bytes
}

fn set_heads() -> Vec<SetHead> {
    vec![
        SetHead { name: "route".into(), kind: SetKind::Choice, max_choices: 7 },
        SetHead { name: "risk_level".into(), kind: SetKind::Score, max_choices: 7 },
        SetHead { name: "safe".into(), kind: SetKind::Null, max_choices: 2 },
    ]
}

fn decision_heads() -> Vec<DecisionHead> {
    vec![DecisionHead {
        name: "action".into(),
        kind: HeadKind::Choice,
        options: ["a", "b", "c"].map(String::from).to_vec(),
    }]
}

/// A fresh v1 config (knirv-heart/v1, no set heads).
fn cfg_v1() -> HeadConfig {
    HeadConfig {
        format: FORMAT.into(),
        tokenizer: "byte".into(),
        vocab_size: 32,
        max_seq_len: 24,
        d_model: 16,
        n_heads: 2,
        ffn_mult: 2,
        l_layers: 1,
        h_layers: 1,
        h_cycles: 3,
        l_steps: 2,
        min_cycles: 1,
        summary_slots: 1,
        decision_heads: decision_heads(),
        tau: 0.0,
        model_version: "legacy-compat".into(),
        dataset_manifest_hash: "m".into(),
        policy_bundle_hash: "p".into(),
        choice_set_heads: vec![],
        dropout: 0.0,
        set_match_feature: false,
    }
}

/// A fresh v2 config (knirv-heart/v2, with set heads). The set scorer tokenizes
/// candidate text into the `n-v1` space (ids up to `tokenizer::VOCAB`), so the
/// model vocab must be the full tokenizer vocab, not a tiny test vocab.
fn cfg_v2() -> HeadConfig {
    HeadConfig {
        format: FORMAT_V2.into(),
        tokenizer: "byte".into(),
        vocab_size: tokenizer::VOCAB,
        max_seq_len: 97,
        d_model: 16,
        n_heads: 2,
        ffn_mult: 2,
        l_layers: 1,
        h_layers: 1,
        h_cycles: 2,
        l_steps: 2,
        min_cycles: 1,
        summary_slots: 1,
        decision_heads: decision_heads(),
        tau: 0.0,
        model_version: "legacy-compat".into(),
        dataset_manifest_hash: "m".into(),
        policy_bundle_hash: "p".into(),
        choice_set_heads: set_heads(),
        dropout: 0.0,
        set_match_feature: false,
    }
}

fn q(name: &str, choices: &[(&str, &str)]) -> SetQuestion {
    SetQuestion {
        name: name.into(),
        question: Some("which one?".into()),
        state: Some("s".into()),
        choices: choices.iter().map(|(i, t)| SetChoice { id: (*i).into(), text: (*t).into() }).collect(),
    }
}

/// A real persisted `knirv-head/v1` config (the pre-migration format id, copied
/// from the Phase-5 backup) plus its golden safetensors must still load and reason.
#[test]
fn legacy_knirv_head_v1_config_and_weights_load() {
    let c = HeadConfig::from_json(&String::from_utf8(read("tests/legacy_v1_config.json")).unwrap()).unwrap();
    assert_eq!(c.format, "knirv-head/v1", "fixture must keep the legacy id");
    c.validate().unwrap();
    let w = read("tests/golden.safetensors");
    let h = Head::from_safetensors(c.clone(), &w, &Device::Cpu).unwrap();
    let r = h.reason(&[3, 1, 4, 1, 5, 9, 2, 6]).unwrap();
    assert!(!r.decisions.is_empty(), "legacy v1 config produced no decisions");
    assert!(r.decisions.iter().all(|d| d.candidates.is_empty()), "v1 has no set heads");
}

/// A valid `knirv-head/v2` config (the legacy set-scorer format, with
/// `choice_set_heads`) must load and run the set scorer.
#[test]
fn legacy_knirv_head_v2_config_loads_with_set_heads() {
    let mut c = cfg_v2();
    c.format = "knirv-head/v2".into();
    c.validate().unwrap();
    let w = weights_for(&c);
    let h = Head::from_safetensors(c.clone(), &w, &Device::Cpu).unwrap();
    let r = h.reason_sets(std::slice::from_ref(&q("route", &[("a", "alpha"), ("b", "beta"), ("c", "gamma"), ("d", "delta")]))).unwrap();
    assert!(!r.decisions.is_empty(), "legacy v2 set scorer produced no decisions");
    assert!(!r.decisions[0].candidates.is_empty(), "legacy v2 set head returned no candidates");
}

/// Newly created/persisted configs use the `knirv-heart` ids; legacy ids are
/// load-only compatibility inputs (parsed as-is, never rewritten by the crate).
#[test]
fn new_configs_serialize_to_heart_format_legacy_is_load_only() {
    assert_eq!(FORMAT, "knirv-heart/v1");
    assert_eq!(FORMAT_V2, "knirv-heart/v2");

    // Fresh v1/v2 configs persist with the heart ids.
    let v1_json = serde_json::to_string(&cfg_v1()).unwrap();
    assert!(v1_json.contains("\"format\":\"knirv-heart/v1\""), "{v1_json}");
    let v2_json = serde_json::to_string(&cfg_v2()).unwrap();
    assert!(v2_json.contains("\"format\":\"knirv-heart/v2\""), "{v2_json}");

    // Legacy ids survive a parse unchanged and still validate: load-only inputs.
    let legacy_v1 = HeadConfig::from_json(&String::from_utf8(read("tests/legacy_v1_config.json")).unwrap()).unwrap();
    assert_eq!(legacy_v1.format, "knirv-head/v1");
    legacy_v1.validate().unwrap();

    let legacy_v2 = HeadConfig::from_json(&serde_json::to_string(&{
        let mut c = cfg_v2();
        c.format = "knirv-head/v2".into();
        c
    }).unwrap()).unwrap();
    assert_eq!(legacy_v2.format, "knirv-head/v2");
    legacy_v2.validate().unwrap();
}
