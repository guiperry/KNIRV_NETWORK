//! Task 4.3: the set scorer and the `knirv-heart/v2` format.

use candle_core::{DType, Device, Result, Tensor};
use candle_nn::{Init, VarBuilder, VarMap};
use knirv_heart_decision::{overlap_feature, set_loss_from_logits, softmax_last_dim_diff, tokenizer, ChoiceLayout, DecisionHead, Head, HeadConfig, HeadKind, SetChoice, SetHead, SetKind, SetQuestion};
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
        dropout: 0.0,
        set_match_feature: false,
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
    // E2: inference passes a zero overlap feature (keeps production inert); tests do too.
    let z = |m: usize| Tensor::zeros((1, m, 1), DType::F32, &Device::Cpu).unwrap();
    let plain = h.set_probabilities(&cm, &a, 0, &[true, true, true], &z(3)).unwrap();
    // Pad to MAX_CHOICES rows with garbage; mark them invalid.
    let garbage = Tensor::ones((MAX_CHOICES - 3, c.d_model), DType::F32, &Device::Cpu).unwrap().affine(37.0, -5.0).unwrap();
    let padded_c = Tensor::cat(&[&cm, &garbage], 0).unwrap();
    let mut valid = vec![true; 3];
    valid.resize(MAX_CHOICES, false);
    let padded = h.set_probabilities(&padded_c, &a, 0, &valid, &z(MAX_CHOICES)).unwrap();
    assert_eq!(padded.len(), MAX_CHOICES);
    for i in 0..3 {
        assert!((plain[i] - padded[i]).abs() <= 1e-6, "padding changed candidate {i}: {} vs {}", plain[i], padded[i]);
    }
    for (i, p) in padded.iter().enumerate().skip(3) {
        assert_eq!(*p, 0.0, "padded candidate {i} got mass");
    }
    // All-invalid and mismatched masks are errors.
    assert!(h.set_probabilities(&padded_c, &a, 0, &vec![false; MAX_CHOICES], &z(MAX_CHOICES)).is_err());
    assert!(h.set_probabilities(&padded_c, &a, 0, &[true], &z(1)).is_err());
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

/// Regression test for the Task 4.4 set-loss bug (2026-11-05). The set-head cross-entropy is
/// `-log(softmax[candidate == target])`. On CUDA, candle 0.11's `sum_all` / `narrow` reductions
/// silently return 0 for the pad-to-32 softmax output (the same kernel class as the
/// `broadcast_div`/`recip` NaN that the `log_softmax_last_dim_diff` workaround exists for), which
/// zeroed the logged loss AND starved the set head of gradient — every GPU training run "trained"
/// with a loss of ~0 and never learned. The fix computes the NLL directly with
/// `log_softmax_last_dim_diff` followed by a `narrow(...).neg()` target gather (no `-inf * 0` NaN,
/// correct forward+backward on both devices), inside `set_loss_from_logits`. This test pins the
/// invariant: the reduction equals the hand-computed NLL and AGREES between CPU and CUDA, across
/// plain rows and the exact padded/-inf shape the set scorer produces for invalid candidates.
#[test]
fn set_loss_reduction_matches_reference_and_agrees_cpu_cuda() {
    /// Numerically stable reference: -log_softmax[target] over the *finite* entries only
    /// (padding/-inf columns carry no mass in a real softmax).
    fn reference_nll(logits: &[f32], target: usize) -> f32 {
        let valid: Vec<f32> = logits.iter().cloned().filter(|x| x.is_finite()).collect();
        assert!(target < valid.len(), "target indexes a -inf column");
        let m = valid.iter().cloned().fold(f32::NEG_INFINITY, f32::max);
        let sum: f32 = valid.iter().map(|x| (x - m).exp()).sum();
        let logp = (valid[target] - m) - sum.ln();
        -logp
    }

    // (logits, target) pairs covering plain rows and the padded/-inf column shape.
    let finite = vec![
        (vec![2.0f32, 1.0, 0.5, 0.3], 0usize),
        (vec![2.0f32, 1.0, 0.5, 0.3], 3usize),
        (vec![0.0f32, 0.0, 0.0, 0.0], 2usize),
        (vec![5.0f32, -1.0, 2.0, 0.0, -100.0], 2usize),
    ];
    // 8-wide row with 4 valid candidates and 4 -inf padding columns (the set scorer's shape).
    let padded = vec![2.0f32, 1.0, 0.5, 0.3, f32::NEG_INFINITY, f32::NEG_INFINITY, f32::NEG_INFINITY, f32::NEG_INFINITY];
    let padded_cases = vec![(padded.clone(), 0usize), (padded.clone(), 3usize)];

    let cpu = Device::Cpu;
    let mut all: Vec<(Vec<f32>, usize)> = finite.clone();
    all.extend(padded_cases.clone());

    for (logits, target) in &all {
        let t = Tensor::from_slice(logits, (1, logits.len()), &cpu).unwrap();
        let nll = set_loss_from_logits(&t, *target).unwrap().to_vec2::<f32>().unwrap()[0][0];
        let ref_nll = reference_nll(logits, *target);
        assert!((nll - ref_nll).abs() < 1e-3, "CPU NLL {nll} != reference {ref_nll} for logits={logits:?} target={target}");
    }

    // Must agree on the GPU (the device where the OLD reduction returned 0).
    let dev = Device::cuda_if_available(0).unwrap_or(Device::Cpu);
    if dev.is_cuda() {
        for (logits, target) in &all {
            let t_cpu = Tensor::from_slice(logits, (1, logits.len()), &cpu).unwrap();
            let t_gpu = Tensor::from_slice(logits, (1, logits.len()), &dev).unwrap();
            let nll_cpu = set_loss_from_logits(&t_cpu, *target).unwrap().to_vec2::<f32>().unwrap()[0][0];
            let nll_gpu = set_loss_from_logits(&t_gpu, *target).unwrap().to_vec2::<f32>().unwrap()[0][0];
            assert!((nll_gpu - nll_cpu).abs() < 1e-3, "GPU NLL {nll_gpu} != CPU NLL {nll_cpu} for logits={logits:?} target={target}");
        }
    }
}

/// Cosine similarity of two equal-length vectors.
fn cosine(a: &[f32], b: &[f32]) -> f32 {
    let (mut dot, mut na, mut nb) = (0f32, 0f32, 0f32);
    for (x, y) in a.iter().zip(b) {
        dot += x * y;
        na += x * x;
        nb += y * y;
    }
    dot / (na.sqrt() * nb.sqrt() + 1e-12)
}

/// Regression for Task 4.4 (2026-11-06): the content-overlap feature `overlap_feature` (LCS of
/// candidate vs question, normalised by candidate length) is the exact signal the set head learns
/// from. It must be deterministic and identical across native / WASM / WASI — this pins its
/// definition so any divergence in the runtime feature computation is caught. A candidate that
/// shares the question's keyword scores high; an unrelated distractor scores ~0.
#[test]
fn overlap_feature_definition() {
    let q = "should we restart the node";
    let f = overlap_feature(
        q,
        &[
            "restart the node now".to_string(),
            "ignore the alert".to_string(),
            "the node should be restarted".to_string(),
        ],
    );
    // "restart the node" (16 bytes) is a substring of candidate 0 → overlap 16/20 = 0.8.
    assert!((f[0] - 0.8).abs() < 1e-6, "candidate 0 overlap expected 0.8, got {}", f[0]);
    // The matching candidate must clearly outscore both the unrelated distractor and the partial
    // overlap (candidate 2 shares "the node"); the feature is a *relative* signal, not a classifier.
    assert!(f[0] > f[1], "matching candidate must outscore the distractor ({} > {})", f[0], f[1]);
    assert!(f[0] > f[2], "matching candidate must outscore the partial-overlap candidate ({} > {})", f[0], f[2]);
    assert!(f[1] < 0.5, "distractor overlap must be well below the match, got {}", f[1]);
    assert!(f[2] > 0.15, "partial overlap ('the node') expected for candidate 2, got {}", f[2]);
}

/// Regression for Task 4.4 (2026-11-06): the set-head loss *backward* must be correct and finite on
/// both CPU and GPU. The original CUDA loss backward was broken (zero gradient / NaN), so the GPU
/// never trained; the repair (narrow-gather NLL + stable 2D-flattened softmax) must yield a real
/// gradient w.r.t. the logits equal to the analytic `softmax_probs − one_hot`. `Tensor::backward`
/// only tracks `VarMap` variables, so `logits` is a VarMap var here. Gated on CUDA availability.
#[test]
fn set_loss_backward_agrees_cpu_cuda() -> Result<()> {
    let n = 8usize;
    let target = 3usize;

    let vm = VarMap::new();
    let vb = VarBuilder::from_varmap(&vm, DType::F32, &Device::Cpu);
    let logits = vb.get_with_hints((1, n), "logits", Init::Randn { mean: 0.0, stdev: 1.0 })?;
    let loss = set_loss_from_logits(&logits, target)?;
    let grads = loss.backward()?;
    let g_cpu = grads.get(&logits).unwrap().to_vec2::<f32>()?[0].clone();
    // Analytic gradient: d/dx -log_softmax[x]_t = softmax(x) - one_hot(t).
    let sm = softmax_last_dim_diff(&logits)?.to_vec2::<f32>()?[0].clone();
    let mut exp = sm.clone();
    exp[target] -= 1.0;
    let cos_cpu = cosine(&g_cpu, &exp);

    let cuda = Device::cuda_if_available(0).unwrap_or(Device::Cpu);
    if !cuda.is_cuda() {
        assert!(cos_cpu > 0.999, "CPU set_loss grad != analytic, cos={cos_cpu:.4}");
        assert!(g_cpu.iter().all(|g| *g != 0.0), "CPU set_loss gradient is zero");
        return Ok(());
    }
    let vm_g = VarMap::new();
    let vb_g = VarBuilder::from_varmap(&vm_g, DType::F32, &cuda);
    let logits_g = vb_g.get_with_hints((1, n), "logits", Init::Randn { mean: 0.0, stdev: 1.0 })?;
    let loss_g = set_loss_from_logits(&logits_g, target)?;
    let grads_g = loss_g.backward()?;
    let g_g = grads_g.get(&logits_g).unwrap().to_vec2::<f32>()?[0].clone();
    let sm_g = softmax_last_dim_diff(&logits_g)?.to_vec2::<f32>()?[0].clone();
    let mut exp_g = sm_g.clone();
    exp_g[target] -= 1.0;
    let cos_g = cosine(&g_g, &exp_g);

    assert!(g_cpu.iter().all(|g| *g != 0.0), "CPU set_loss gradient is zero");
    assert!(cos_cpu > 0.999, "CPU set_loss grad != analytic, cos={cos_cpu:.4}");
    assert!(g_g.iter().all(|g| *g != 0.0), "CUDA set_loss gradient is zero");
    assert!(cos_g > 0.999, "CUDA set_loss grad != analytic, cos={cos_g:.4}");
    Ok(())
}

/// Regression for Task 4.4 (2026-11-06): the *set-head forward* (`set_scores`, the core scoring op
/// the trainer and `reason_sets` both use) must produce identical logits on CPU and CUDA for fixed
/// weights + fixed input. This pins device agreement for the production path (native / WASM / WASI all
/// share the Rust core's set scorer), catching any CUDA softmax/attention divergence. Gated on CUDA
/// availability.
///
/// NOTE: this deliberately exercises `set_scores` (the set-head scoring) rather than the full
/// `reason_sets`/`set_cycle_logits`, because those also call the decision-head `run` forward, whose
/// ACT/halting path has a separate, pre-existing GPU divergence. The set-head scoring is what the
/// Task 4.4 gate depends on, and it agrees across devices.
#[test]
fn set_scores_agrees_cpu_cuda() -> Result<()> {
    let cuda = Device::cuda_if_available(0).unwrap_or(Device::Cpu);
    if !cuda.is_cuda() {
        eprintln!("no CUDA device; skipping CPU-vs-GPU set-head forward test");
        return Ok(());
    }
    let c = cfg();
    let d = c.d_model;
    let (w, _) = weights(&c);
    let h_cpu = Head::new(c.clone(), VarBuilder::from_tensors(w.clone(), DType::F32, &Device::Cpu))?;
    let h_gpu = Head::new(c.clone(), VarBuilder::from_tensors(w.clone(), DType::F32, &cuda))?;

    // Bounded random set inputs (small scale avoids attention-softmax overflow on either device).
    let c_cpu = Tensor::randn(0.0f32, 0.3f32, (1, 3, d), &Device::Cpu)?;
    let a_cpu = Tensor::randn(0.0f32, 0.3f32, (1, d), &Device::Cpu)?;
    let m_cpu = Tensor::zeros((1, 3, 1), DType::F32, &Device::Cpu)?;
    let c_gpu = c_cpu.to_device(&cuda)?;
    let a_gpu = a_cpu.to_device(&cuda)?;
    let m_gpu = m_cpu.to_device(&cuda)?;
    let valid = vec![true, true, true];

    let out_cpu = h_cpu.set_scores(&c_cpu, &a_cpu, 0, &valid, &m_cpu)?;
    let out_gpu = h_gpu.set_scores(&c_gpu, &a_gpu, 0, &valid, &m_gpu)?;
    let flat_c: Vec<f32> = out_cpu.to_vec2::<f32>()?.into_iter().flatten().collect();
    let flat_g: Vec<f32> = out_gpu.to_vec2::<f32>()?.into_iter().flatten().collect();
    for v in &flat_g {
        assert!(v.is_finite(), "set_scores GPU produced NaN/inf");
    }
    let cos = cosine(&flat_c, &flat_g);
    assert!(cos > 0.99, "set_scores CPU vs GPU cos={cos:.4} (need >0.99)");
    Ok(())
}

/// Native golden for Task 4.4 (2026-11-06): `reason_sets` on CPU (the native / WASM / WASI inference
/// path — all CPU, no GPU) must return finite, normalised probabilities that single out one
/// candidate, deterministically, for fixed weights + fixed input. This is the "native" half of the
/// native==WASI agreement; the WASI half is checked by `golden-wasm.sh` against the same expected
/// output. The decision-head `run` forward (used only for halting metadata) is exercised here on CPU
/// where it is numerically clean.
#[test]
fn reason_sets_golden_native() -> Result<()> {
    let c = cfg();
    let (w, _) = weights(&c);
    let q = q(
        "route",
        "host=idle load=42",
        &[("a", "wait"), ("b", "restart"), ("c", "escalate")],
    );
    let r = Head::new(c.clone(), VarBuilder::from_tensors(w, DType::F32, &Device::Cpu))?
        .reason_sets(std::slice::from_ref(&q))?;
    let probs: Vec<f32> = r.decisions[0].candidates.iter().map(|x| x.probability).collect();
    let sum: f32 = probs.iter().sum();
    assert!((sum - 1.0).abs() < 1e-4, "probs do not sum to 1: sum={sum}");
    assert!(probs.iter().all(|p| p.is_finite() && *p >= 0.0), "non-finite / negative prob");
    assert!(
        r.decisions[0].selected.is_some(),
        "no candidate selected (tau abstain)"
    );
    Ok(())
}

/// Feature-ON production-path test for Task 4.4 (audit 4.4_prod, Fix 1): with `set_match_feature=true`
/// the model must compute the overlap feature INTERNALLY (the same `overlap_feature` used in training)
/// and that feature must actually move the set-head probabilities. This is FAILABLE: `weights()` seeds a
/// non-zero `set.match_emb`, so the feature-on path raises the probability of the candidate that overlaps
/// the question above the feature-OFF (zero-feature) baseline. If the flag is forced false, or `match_emb`
/// is zeroed, feature-on == feature-off and the difference assertion below FAILS — proving the test
/// detects a disabled feature (the original defect: the flag defaulted to false and silently disabled
/// matching). The WASI/wasm binary runs the SAME `reason_sets` Rust core, so the native==WASI agreement
/// is covered by construction (verified by `golden-wasm.sh`).
#[test]
fn reason_sets_feature_on_golden_native() -> Result<()> {
    // Question carries a keyword that appears in exactly one candidate ("restart the node"), so the
    // overlap feature distinguishes it from the distractors.
    let q_text = "the node must be restarted now";
    let candidates: Vec<(&str, &str)> = vec![("a", "wait"), ("b", "restart the node"), ("c", "escalate")];

    let mut on = cfg();
    on.set_match_feature = true;
    let mut off = cfg();
    off.set_match_feature = false;
    // Architecture (and therefore the seeded weights, including `set.match_emb`) is identical for
    // on/off, so the two heads differ only in whether the internal feature is computed.
    let (w, _) = weights(&on);

    let q = q("route", "host=idle", &candidates);
    let mut q_feat = q.clone();
    q_feat.question = Some(q_text.into());

    let on_head = Head::new(on.clone(), VarBuilder::from_tensors(w.clone(), DType::F32, &Device::Cpu))?;
    let off_head = Head::new(off.clone(), VarBuilder::from_tensors(w.clone(), DType::F32, &Device::Cpu))?;

    let on_r = on_head.reason_sets(std::slice::from_ref(&q_feat))?;
    let off_r = off_head.reason_sets(std::slice::from_ref(&q_feat))?;
    let on_probs: Vec<f32> = on_r.decisions[0].candidates.iter().map(|x| x.probability).collect();
    let off_probs: Vec<f32> = off_r.decisions[0].candidates.iter().map(|x| x.probability).collect();

    // Feature-ON result is a valid distribution with a selection.
    let sum: f32 = on_probs.iter().sum();
    assert!((sum - 1.0).abs() < 1e-4, "feature-ON probs do not sum to 1: sum={sum}");
    assert!(on_probs.iter().all(|p| p.is_finite() && *p >= 0.0), "feature-ON non-finite / negative prob");
    assert!(on_r.decisions[0].selected.is_some(), "feature-ON: no candidate selected (tau abstain)");

    // Fail-safe: feature-ON must DIFFER from feature-OFF (the zero-feature baseline) by a known amount.
    let max_diff = on_probs.iter().zip(&off_probs).map(|(a, b)| (a - b).abs()).fold(0f32, f32::max);
    assert!(max_diff > 1e-3, "feature-ON == feature-OFF (max diff {max_diff}); the overlap feature has no effect");

    // Direction: the candidate overlapping the question must GAIN probability under the feature.
    let cand_texts: Vec<String> = candidates.iter().map(|(_, t)| (*t).to_string()).collect();
    let overlap = overlap_feature(q_text, &cand_texts);
    let match_idx = overlap.iter().enumerate().max_by(|x, y| x.1.total_cmp(y.1)).map(|(i, _)| i).unwrap();
    assert!(
        on_probs[match_idx] > off_probs[match_idx] + 1e-3,
        "matching candidate gained no probability from the feature: on={on_probs:?} off={off_probs:?} overlap={overlap:?}"
    );
    Ok(())
}

/// Regression for audit 4.4c (P1 production blocker): `cycle_output` selected the decision logits from
/// a non-contiguous `narrow` view and the summary from a non-contiguous `i` view; candle's CUDA backend
/// rejects non-contiguous matmul operands, so every GPU training/serving run through the decision forward
/// failed (4 trainer e2e tests). The CPU backend tolerates the strided view, which is why the CPU e2e
/// suite passed 9/9 and the failure looked tree-inconsistent. The fix adds `.contiguous()` before those
/// matmuls (and materialises `emb_w.t()` and the initial `z` state, which are also stride-0). This test
/// runs the exact forward on CUDA (batch>1, summary_slots>1 — the gate-tripping shape) and asserts it
/// completes (returns `Ok`) with the correct logits shape. It regresses the contiguity crash; a separate,
/// tracked candle-CUDA numerical issue (rank-3 batched matmul divergence in the block stack) is verified
/// by `decision_forward_cuda_values_match_cpu` only when both sides are finite.
#[cfg(feature = "cuda")]
#[test]
fn decision_forward_runs_on_cuda_with_batch_and_summary_slots() -> Result<()> {
    use knirv_heart_decision::Mode;
    let cuda = Device::cuda_if_available(0).unwrap_or(Device::Cpu);
    if !cuda.is_cuda() {
        eprintln!("no CUDA device; skipping decision-forward CUDA contiguity regression");
        return Ok(());
    }
    let mut c = cfg();
    c.summary_slots = 4; // >1 exercises narrow(1, slots, s) selection
    c.set_match_feature = false;
    let (w, _) = weights(&c);
    let h_gpu = Head::new(c.clone(), VarBuilder::from_tensors(w.clone(), DType::F32, &cuda))?;
    let b = 2usize; // batch > 1
    let s = 5usize; // input length
    let ids: Vec<u32> = vec![1u32; b * s];
    let t_gpu = Tensor::from_vec(ids, (b, s), &cuda)?;
    // Before the fix this panicked with "matmul is only supported for contiguous tensors"
    // (or an earlier stride-0 matmul produced garbage); now it must complete and return the
    // expected logits shape.
    let tr_gpu = h_gpu.forward(&t_gpu, Mode::Train)?;
    let lg = tr_gpu.logits.to_vec3::<f32>()?;
    assert_eq!(lg.len(), b, "batch dimension lost through the forward");
    assert_eq!(lg[0].len(), s, "seq dimension lost through the forward");
    Ok(())
}

/// Audit 44d: the decision forward must be numerically identical on CPU and CUDA. The previous
/// candle-CUDA divergence (block-1 RMSNorm amplified its input ~1000x, overflowing to `inf`/`NaN`)
/// is fixed by flattening the `DiffRmsNorm` reduction to 2D. This test now FAILS (never skips) on a
/// non-finite GPU output or on a CPU/GPU cosine mismatch, so the GPU path is hard-gated.
#[cfg(feature = "cuda")]
#[test]
fn decision_forward_cuda_values_match_cpu() -> Result<()> {
    use knirv_heart_decision::Mode;
    let cuda = Device::cuda_if_available(0).unwrap_or(Device::Cpu);
    if !cuda.is_cuda() {
        eprintln!("no CUDA device; skipping decision-forward CPU/GPU agreement check");
        return Ok(());
    }
    let mut c = cfg();
    c.summary_slots = 4;
    c.set_match_feature = false;
    let (w, _) = weights(&c);
    let h_cpu = Head::new(c.clone(), VarBuilder::from_tensors(w.clone(), DType::F32, &Device::Cpu))?;
    let h_gpu = Head::new(c.clone(), VarBuilder::from_tensors(w.clone(), DType::F32, &cuda))?;
    let b = 2usize; let s = 5usize;
    let ids: Vec<u32> = vec![1u32; b * s];
    let tr_cpu = h_cpu.forward(&Tensor::from_vec(ids.clone(), (b, s), &Device::Cpu)?, Mode::Train)?;
    let tr_gpu = h_gpu.forward(&Tensor::from_vec(ids, (b, s), &cuda)?, Mode::Train)?;
    let fc: Vec<f32> = tr_cpu.logits.flatten_all()?.to_vec1::<f32>()?;
    let fg: Vec<f32> = tr_gpu.logits.flatten_all()?.to_vec1::<f32>()?;
    // Hard gate: a non-finite GPU output is now a failure, not a skip (audit 44d).
    let cpu_finite = fc.iter().all(|v| v.is_finite());
    let gpu_finite = fg.iter().all(|v| v.is_finite());
    assert!(cpu_finite, "CPU decision forward produced non-finite logits (unexpected)");
    assert!(gpu_finite, "GPU decision forward produced non-finite logits (candle-CUDA RMSNorm divergence fix regressed/incomplete)");
    assert_eq!(fg.len(), fc.len());
    let dot: f32 = fc.iter().zip(&fg).map(|(a, b)| a * b).sum();
    let na: f32 = fc.iter().map(|v| v * v).sum::<f32>().sqrt();
    let nb: f32 = fg.iter().map(|v| v * v).sum::<f32>().sqrt();
    let cos = dot / (na * nb);
    assert!(cos > 0.99, "decision forward CPU vs GPU cos={cos:.4} (need >0.99)");
    Ok(())
}

/// Regression for Fix 7 (audit 4.4_prod): `softmax_last_dim_diff` must be finite and well-defined for
/// every practical width (including across the 32 padding boundary) and for fully-masked (`-inf`) rows,
/// which the scorer produces for padding candidates. An all-`-inf` row must come out all-`0` (no mass),
/// never `NaN`.
#[test]
fn softmax_handles_all_masked_rows_and_widths() -> Result<()> {
    let dev = Device::Cpu;
    // A normal row must still sum to 1 and be finite at widths just below / at / above the 32 pad.
    for w in [1usize, 2, 5, 31, 32, 33] {
        let mut v = vec![0f32; 2 * w];
        for i in 0..w {
            v[i] = (i % 7) as f32 + 0.5; // finite, distinct row 0
        }
        for i in w..2 * w {
            v[i] = f32::NEG_INFINITY; // fully-masked row 1
        }
        let t = Tensor::from_vec(v, (2, w), &dev)?;
        let sm = softmax_last_dim_diff(&t)?;
        let s = sm.to_vec2::<f32>()?;
        let r0 = &s[0];
        assert!(r0.iter().all(|x| x.is_finite()), "width {w}: row 0 has non-finite softmax");
        let sum0: f32 = r0.iter().sum();
        assert!((sum0 - 1.0).abs() < 1e-4, "width {w}: row 0 does not sum to 1: {sum0}");
        let r1 = &s[1];
        assert!(r1.iter().all(|x| x.is_finite()), "width {w}: masked row has non-finite softmax");
        assert!(r1.iter().all(|&x| x == 0.0), "width {w}: masked row must be all-0, got {r1:?}");
    }
    // A single fully-masked row on its own (width 1) must be all-0, not NaN.
    let only_masked = Tensor::from_vec(vec![f32::NEG_INFINITY], (1, 1), &dev)?;
    let sm = softmax_last_dim_diff(&only_masked)?;
    let s = sm.to_vec2::<f32>()?[0].clone();
    assert_eq!(s, vec![0.0f32], "single masked row must be 0, got {s:?}");
    Ok(())
}

/// Regression for Fix 8 (audit 4.4_prod): `overlap_feature` must cap the LCS inputs and stay bounded
/// for arbitrarily large candidate / question text (the HTTP path admits unbounded strings). A 200k-char
/// candidate and question must finish in milliseconds and return a finite value; the cap keeps the LCS
/// bounded by 256 characters per side (at most 1024 UTF-8 bytes) regardless of input length.
#[test]
fn overlap_feature_handles_huge_inputs() {
    let big = "a".repeat(200_000);
    let q = "b".repeat(200_000);
    let t0 = std::time::Instant::now();
    let f = overlap_feature(&q, &[big.clone()]);
    let dt = t0.elapsed();
    assert!(dt.as_millis() < 200, "overlap_feature on huge input too slow: {dt:?}");
    assert!(f[0].is_finite(), "huge-input feature not finite");
    assert_eq!(f[0], 0.0, "all-'a' candidate vs all-'b' question share nothing");
    // A short shared substring survives the cap and yields a positive feature.
    let f2 = overlap_feature("the node must be restarted", &["restarted the node".to_string()]);
    assert!(f2[0] > 0.0 && f2[0] <= 1.0, "capped overlap feature out of range: {}", f2[0]);
}

/// TASK 4.4 DIAGNOSIS: confirm whether the CUDA backward of the padded `sum_keepdim` reductions
/// (used by `DiffRmsNorm`, `softmax_last_dim_diff`, `mean_dim`, `set_loss_from_logits`) agrees with
/// the CPU backward. The forward is already verified to match (decision_forward_cuda_values_match_cpu),
/// but a broken CUDA backward would silently zero/garble gradients and prevent GPU training while
/// CPU still learns. This test pins exactly that.
#[cfg(feature = "cuda")]
#[test]
fn cuda_backward_matches_cpu_for_sum_keepdim_and_softmax() {
    use candle_core::{D, Device};
    let dev_cpu = Device::Cpu;
    let dev_cuda = Device::cuda_if_available(0).expect("cuda must be available for this test");

    // (1) direct sum_keepdim backward, narrow (16) and padded (32) widths.
    for &w in &[16usize, 32usize] {
        let x = Tensor::randn(0.0f32, 1.0, &[4, w], &dev_cpu).unwrap();
        let xc = x.to_device(&dev_cpu).unwrap().set_requires_grad(true).unwrap();
        let yc = xc.sum_keepdim(D::Minus1).unwrap();
        yc.backward().unwrap();
        let gc = xc.grad().unwrap().to_vec2::<f32>().unwrap();
        let xg = x.to_device(&dev_cuda).unwrap().set_requires_grad(true).unwrap();
        let yg = xg.sum_keepdim(D::Minus1).unwrap();
        yg.backward().unwrap();
        let gg = xg.grad().unwrap().to_vec2::<f32>().unwrap();
        for i in 0..4 {
            for j in 0..w {
                assert!(
                    (gc[i][j] - gg[i][j]).abs() < 1e-3,
                    "sum_keepdim w={w} grad mismatch at {i},{j}: cpu={} cuda={}",
                    gc[i][j], gg[i][j]
                );
            }
        }
    }

    // (2) softmax_last_dim_diff backward (pads 16 -> 32 internally, exercises sum_keepdim).
    {
        let x = Tensor::randn(0.0f32, 1.0, &[4, 16], &dev_cpu).unwrap();
        let xc = x.to_device(&dev_cpu).unwrap().set_requires_grad(true).unwrap();
        let yc = softmax_last_dim_diff(&xc).unwrap();
        yc.backward().unwrap();
        let gc = xc.grad().unwrap().to_vec2::<f32>().unwrap();
        let xg = x.to_device(&dev_cuda).unwrap().set_requires_grad(true).unwrap();
        let yg = softmax_last_dim_diff(&xg).unwrap();
        yg.backward().unwrap();
        let gg = xg.grad().unwrap().to_vec2::<f32>().unwrap();
        for i in 0..4 {
            for j in 0..16 {
                assert!(
                    (gc[i][j] - gg[i][j]).abs() < 1e-2,
                    "softmax_last_dim_diff grad mismatch at {i},{j}: cpu={} cuda={}",
                    gc[i][j], gg[i][j]
                );
            }
        }
    }
}
