//! Task 4.3: the set scorer and the `knirv-heart/v2` format.

use candle_core::{DType, Device, Tensor};
use candle_nn::{VarBuilder, VarMap};
use knirv_heart_decision::{tokenizer, ChoiceLayout, DecisionHead, Head, HeadConfig, HeadKind, SetChoice, SetHead, SetKind, SetQuestion};
use std::collections::HashMap;

const MAX_CHOICES: usize = 7; // distinct from every other dimension, so a per-slot weight shows
const WIDTH: usize = 96;

fn cfg() -> HeadConfig {
    HeadConfig {
        format: "knirv-heart/v2".into(),
        tokenizer: "byte".into(),
        vocab_size: tokenizer::VOCAB,
        max_seq_len: WIDTH + 1,
        d_model: 16,
        n_heads: 2,
        ffn_mult: 2,
        l_layers: 1,
        h_layers: 1,
        h_cycles: 2,
        l_steps: 2,
        min_cycles: 1,
        summary_slots: 1,
        decision_heads: vec![DecisionHead { name: "action".into(), kind: HeadKind::Choice, options: ["a", "b", "c"].map(String::from).to_vec() }],
        tau: 0.0,
        model_version: "sets-test".into(),
        dataset_manifest_hash: "m".into(),
        policy_bundle_hash: "p".into(),
        choice_set_heads: vec![
            SetHead { name: "route".into(), kind: SetKind::Choice, max_choices: MAX_CHOICES },
            SetHead { name: "risk_level".into(), kind: SetKind::Score, max_choices: MAX_CHOICES },
            SetHead { name: "safe".into(), kind: SetKind::Null, max_choices: 2 },
        ],
    }
}

fn v1_cfg() -> HeadConfig {
    HeadConfig { format: "knirv-heart/v1".into(), choice_set_heads: vec![], ..cfg() }
}

/// Deterministic weights from a formula; also returns every tensor's name and shape.
fn weights(c: &HeadConfig) -> (HashMap<String, Tensor>, Vec<(String, Vec<usize>)>) {
    let vm = VarMap::new();
    Head::new(c.clone(), VarBuilder::from_varmap(&vm, DType::F32, &Device::Cpu)).unwrap();
    let mut out = HashMap::new();
    let mut shapes = Vec::new();
    for (name, var) in vm.data().lock().unwrap().iter() {
        let n = var.elem_count();
        let salt = name.bytes().fold(0u32, |a, b| a.wrapping_mul(31).wrapping_add(b as u32)) % 13;
        let v: Vec<f32> = if name.ends_with("n1.weight") || name.ends_with("n2.weight") || name.ends_with("norm.weight") {
            vec![1.0; n]
        } else {
            (0..n).map(|i| 0.4 * ((0.37 * i as f32) + salt as f32).sin()).collect()
        };
        shapes.push((name.clone(), var.dims().to_vec()));
        out.insert(name.clone(), Tensor::from_vec(v, var.shape().clone(), &Device::Cpu).unwrap());
    }
    (out, shapes)
}

fn head(c: &HeadConfig) -> Head {
    Head::new(c.clone(), VarBuilder::from_tensors(weights(c).0, DType::F32, &Device::Cpu)).unwrap()
}

fn q(name: &str, state: &str, choices: &[(&str, &str)]) -> SetQuestion {
    SetQuestion {
        name: name.into(),
        question: Some("which one?".into()),
        state: Some(state.into()),
        choices: choices.iter().map(|(i, t)| SetChoice { id: (*i).into(), text: (*t).into() }).collect(),
    }
}

fn probs(h: &Head, question: &SetQuestion) -> Vec<(String, f32)> {
    let r = h.reason_sets(std::slice::from_ref(question)).unwrap();
    r.decisions[0].candidates.iter().map(|c| (c.id.clone(), c.probability)).collect()
}

const POOL: [(&str, &str); 6] = [("c1", "wait"), ("c2", "restart"), ("c3", "escalate"), ("c4", "rollback"), ("c5", "page the on-call"), ("c6", "ignore")];

/// Tiny deterministic generator for the permutation property.
struct Lcg(u64);
impl Lcg {
    fn next(&mut self) -> usize {
        self.0 = self.0.wrapping_mul(6364136223846793005).wrapping_add(1442695040888963407);
        (self.0 >> 33) as usize
    }
}

#[test]
fn probabilities_permute_with_the_candidates() {
    let h = head(&cfg());
    let mut rng = Lcg(42);
    for trial in 0..24 {
        let n = 2 + rng.next() % 5; // 2..=6
        let mut chosen: Vec<(&str, &str)> = POOL.to_vec();
        for i in (1..chosen.len()).rev() {
            chosen.swap(i, rng.next() % (i + 1));
        }
        chosen.truncate(n);
        let state = format!("host=h{trial} load={}", rng.next() % 100);
        let base = probs(&h, &q("route", &state, &chosen));
        let mut perm = chosen.clone();
        for i in (1..perm.len()).rev() {
            perm.swap(i, rng.next() % (i + 1));
        }
        let shuffled = probs(&h, &q("route", &state, &perm));
        assert_eq!(base.len(), n);
        assert!((base.iter().map(|(_, p)| p).sum::<f32>() - 1.0).abs() < 1e-5);
        for (id, p) in &base {
            let (_, p2) = shuffled.iter().find(|(i, _)| i == id).unwrap();
            assert!((p - p2).abs() <= 1e-5, "trial {trial}: {id} moved from {p} to {p2} under a permutation");
        }
        // The response is aligned to the request's own order.
        let order: Vec<&str> = shuffled.iter().map(|(i, _)| i.as_str()).collect();
        assert_eq!(order, perm.iter().map(|(i, _)| *i).collect::<Vec<_>>());
    }
}

#[test]
fn duplicate_candidate_text_gets_equal_probability() {
    let h = head(&cfg());
    let p = probs(&h, &q("route", "host=idle", &[("x", "wait"), ("y", "restart"), ("z", "wait")]));
    assert!((p[0].1 - p[2].1).abs() <= 1e-6, "{p:?}");
    assert!((p[0].1 - p[1].1).abs() > 1e-6, "distinct candidates should differ: {p:?}");
    // Even when the duplicates are adjacent or the only candidates.
    let p = probs(&h, &q("route", "host=idle", &[("x", "wait"), ("y", "wait")]));
    assert!((p[0].1 - 0.5).abs() <= 1e-6 && (p[1].1 - 0.5).abs() <= 1e-6, "{p:?}");
}

#[test]
fn one_two_and_max_candidates() {
    let h = head(&cfg());
    let one = probs(&h, &q("route", "s", &[("only", "wait")]));
    assert_eq!(one.len(), 1);
    assert!((one[0].1 - 1.0).abs() <= 1e-6);
    assert_eq!(probs(&h, &q("route", "s", &POOL[..2])).len(), 2);
    let many: Vec<(String, String)> = (0..MAX_CHOICES).map(|i| (format!("c{i}"), format!("option {i}"))).collect();
    let refs: Vec<(&str, &str)> = many.iter().map(|(a, b)| (a.as_str(), b.as_str())).collect();
    let p = probs(&h, &q("route", "s", &refs));
    assert_eq!(p.len(), MAX_CHOICES);
    assert!((p.iter().map(|(_, p)| p).sum::<f32>() - 1.0).abs() <= 1e-5);
    // More than max_choices, none at all and an unknown head are errors.
    let too_many: Vec<(String, String)> = (0..=MAX_CHOICES).map(|i| (format!("c{i}"), format!("o{i}"))).collect();
    let refs: Vec<(&str, &str)> = too_many.iter().map(|(a, b)| (a.as_str(), b.as_str())).collect();
    assert!(h.reason_sets(&[q("route", "s", &refs)]).is_err());
    assert!(h.reason_sets(&[q("route", "s", &[])]).is_err());
    assert!(h.reason_sets(&[q("nope", "s", &POOL[..2])]).is_err());
    assert!(h.reason_sets(&[]).is_err());
}

/// Padded candidates get zero mass and change nothing about the real ones.
#[test]
fn padded_candidates_get_zero_mass() {
    let c = cfg();
    let h = head(&c);
    let cs: Vec<String> = ["wait", "restart", "escalate"].map(String::from).to_vec();
    let toks = tokenizer::encode_set_prompt("which one?", "host=idle", &cs, WIDTH);
    let layout = ChoiceLayout::from_tokens(&toks).unwrap();
    let x = Tensor::from_slice(&toks, (1, toks.len()), &Device::Cpu).unwrap();
    let (cm, a) = h.set_inputs(&x, &layout).unwrap();
    let plain = h.set_probabilities(&cm, &a, 0, &[true, true, true]).unwrap();
    // Pad to MAX_CHOICES rows with garbage; mark them invalid.
    let garbage = Tensor::ones((MAX_CHOICES - 3, c.d_model), DType::F32, &Device::Cpu).unwrap().affine(37.0, -5.0).unwrap();
    let padded_c = Tensor::cat(&[&cm, &garbage], 0).unwrap();
    let mut valid = vec![true; 3];
    valid.resize(MAX_CHOICES, false);
    let padded = h.set_probabilities(&padded_c, &a, 0, &valid).unwrap();
    assert_eq!(padded.len(), MAX_CHOICES);
    for i in 0..3 {
        assert!((plain[i] - padded[i]).abs() <= 1e-6, "padding changed candidate {i}: {} vs {}", plain[i], padded[i]);
    }
    for (i, p) in padded.iter().enumerate().skip(3) {
        assert_eq!(*p, 0.0, "padded candidate {i} got mass");
    }
    // All-invalid and mismatched masks are errors.
    assert!(h.set_probabilities(&padded_c, &a, 0, &vec![false; MAX_CHOICES]).is_err());
    assert!(h.set_probabilities(&padded_c, &a, 0, &[true]).is_err());
}

/// No scorer weight has a per-slot dimension: the same weights serve any N up to max_choices.
#[test]
fn no_set_weight_has_a_per_slot_dimension() {
    let c = cfg();
    let (_, shapes) = weights(&c);
    let set: Vec<_> = shapes.iter().filter(|(n, _)| n.starts_with("set.")).collect();
    assert!(set.iter().any(|(n, _)| n == "set.w") && set.iter().any(|(n, _)| n == "set.role"), "{set:?}");
    for (name, dims) in &set {
        assert!(!dims.contains(&MAX_CHOICES), "{name} {dims:?} has a dimension equal to max_choices: a per-slot weight (Linear(d, num_choices))");
        assert!(!dims.contains(&c.choice_set_heads.len()) || dims == &[c.d_model], "{name} {dims:?}");
    }
    // The count the config states is what the architecture really has.
    let total: usize = shapes.iter().map(|(_, d)| d.iter().product::<usize>()).sum();
    assert_eq!(total, c.param_count());
    assert_eq!(c.param_count(), v1_cfg().param_count() + c.set_param_count());
    // And the extra parameters do not depend on max_choices.
    let mut bigger = c.clone();
    bigger.choice_set_heads[0].max_choices = 30;
    assert_eq!(bigger.param_count(), c.param_count());
    // A model built with a different max_choices accepts the same weights.
    let h = Head::new(bigger.clone(), VarBuilder::from_tensors(weights(&c).0, DType::F32, &Device::Cpu)).unwrap();
    assert!(!probs(&h, &q("route", "s", &POOL[..4])).is_empty());
}

/// v1 configs and weights load unchanged and behave as before.
#[test]
fn v1_weights_and_configs_are_unchanged() {
    let v1 = v1_cfg();
    v1.validate().unwrap();
    assert_eq!(v1.set_param_count(), 0);
    // A v1 JSON round trip does not grow a new field.
    let json = serde_json::to_string(&v1).unwrap();
    assert!(!json.contains("choice_set_heads"), "{json}");
    assert_eq!(HeadConfig::from_json(&json).unwrap(), v1);
    // v1 weights load into the v2-capable code; set calls are refused with a clear error.
    let h = head(&v1);
    let err = h.reason_sets(&[q("route", "s", &POOL[..2])]).err().expect("v1 has no set heads").to_string();
    assert!(err.contains("no choice-set heads"), "{err}");
    // v1 reasoning is as before: no candidates in the output.
    let r = h.reason(&[3, 1, 4, 1, 5]).unwrap();
    assert!(r.decisions.iter().all(|d| d.candidates.is_empty()));
    assert!(!serde_json::to_string(&r).unwrap().contains("candidates"));
    // Format rules.
    let mut bad = v1.clone();
    bad.choice_set_heads = cfg().choice_set_heads;
    assert!(bad.validate().is_err(), "set heads in a v1 file");
    let mut empty = cfg();
    empty.choice_set_heads.clear();
    assert!(empty.validate().is_err(), "a v2 file with no set heads");
    let mut dup = cfg();
    dup.choice_set_heads[0].name = "action".into();
    assert!(dup.validate().is_err(), "a set head named like a decision head");
    let mut tiny = cfg();
    tiny.choice_set_heads[0].max_choices = 1;
    assert!(tiny.validate().is_err());
    let mut other = cfg();
    other.format = "knirv-head/v9".into();
    assert!(other.validate().is_err());
}

#[test]
fn the_weights_check_uses_the_authoritative_count() {
    let c = cfg();
    // v2 weights into a v2 config: fine. v1 weights into a v2 config: wrong count, refused.
    let (w, _) = weights(&c);
    let path = std::env::temp_dir().join(format!("sets-{}.safetensors", std::process::id()));
    candle_core::safetensors::save(&w, &path).unwrap();
    let bytes = std::fs::read(&path).unwrap();
    Head::from_safetensors(c.clone(), &bytes, &Device::Cpu).unwrap();
    let (w1, _) = weights(&v1_cfg());
    candle_core::safetensors::save(&w1, &path).unwrap();
    let v1_bytes = std::fs::read(&path).unwrap();
    assert!(Head::from_safetensors(c, &v1_bytes, &Device::Cpu).is_err());
    Head::from_safetensors(v1_cfg(), &v1_bytes, &Device::Cpu).unwrap();
    let _ = std::fs::remove_file(path);
}

#[test]
fn score_null_and_task_embeddings() {
    let h = head(&cfg());
    // Score: ordered levels, value = sum p_i * i / (N-1), in [0, 1].
    let r = h.reason_sets(&[q("risk_level", "disk=95%", &[("l", "low"), ("m", "medium"), ("h", "high")])]).unwrap();
    let d = &r.decisions[0];
    assert_eq!(d.kind, HeadKind::Score);
    let p: Vec<f32> = d.candidates.iter().map(|c| c.probability).collect();
    let want = p[1] * 0.5 + p[2];
    assert!((d.value.unwrap() - want).abs() <= 1e-6 && (0.0..=1.0).contains(&d.value.unwrap()), "{p:?} {:?}", d.value);
    assert!(h.reason_sets(&[q("risk_level", "s", &[("l", "low")])]).is_err(), "a score head needs two levels");
    // Null: always yes/no, whatever the request says.
    let r = h.reason_sets(&[q("safe", "rm -rf /tmp/x", &[("junk", "ignored")])]).unwrap();
    let ids: Vec<&str> = r.decisions[0].candidates.iter().map(|c| c.id.as_str()).collect();
    assert_eq!(ids, ["yes", "no"]);
    assert!((r.decisions[0].probabilities.iter().sum::<f32>() - 1.0).abs() <= 1e-5);
    // The task embedding matters: the same candidates under another head score differently.
    let a = probs(&h, &q("route", "host=idle", &POOL[..3]));
    let b = h.reason_sets(&[q("risk_level", "host=idle", &POOL[..3])]).unwrap();
    let b: Vec<f32> = b.decisions[0].candidates.iter().map(|c| c.probability).collect();
    assert!(a.iter().zip(&b).any(|((_, x), y)| (x - y).abs() > 1e-4), "{a:?} vs {b:?}");
    // Several questions in one request, each its own decision, in order.
    let r = h.reason_sets(&[q("route", "s", &POOL[..3]), q("safe", "s", &[])]).unwrap();
    assert_eq!(r.decisions.len(), 2);
    assert_eq!(r.decisions[0].name, "route");
    assert_eq!(r.decisions[1].name, "safe");
    assert_eq!(r.max_steps, 2);
}

#[test]
fn an_oversized_prompt_is_an_error_not_a_silent_truncation() {
    let h = head(&cfg());
    let long = "x".repeat(WIDTH);
    let err = h.reason_sets(&[q("route", &long, &POOL[..3])]).err().expect("too long").to_string();
    assert!(err.contains("exceeds") || err.contains("token input"), "{err}");
}

#[test]
fn the_json_shape_is_candidates_in_request_order() {
    let h = head(&cfg());
    let qs: Vec<SetQuestion> = serde_json::from_str(r#"[{"name":"route","state":"host=idle","choices":[{"id":"b","text":"restart"},{"id":"a","text":"wait"}]}]"#).unwrap();
    let r = h.reason_sets(&qs).unwrap();
    let json = serde_json::to_value(&r).unwrap();
    let cands = json["decisions"][0]["candidates"].as_array().unwrap();
    assert_eq!(cands[0]["id"], "b");
    assert_eq!(cands[1]["id"], "a");
    assert!(cands[0]["probability"].is_number() && cands[0]["text"] == "restart");
}
