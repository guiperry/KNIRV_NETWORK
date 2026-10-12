use candle_core::{DType, Device, Tensor};
use candle_nn::{loss, optim::AdamW, Optimizer, ParamsAdamW, VarBuilder, VarMap};
use knirv_cortex_wasm::{DecisionHead, HeadKind, Hrm, HrmConfig, Mode};
use std::collections::HashMap;

fn cfg() -> HrmConfig {
    HrmConfig {
        format: "knirv-hrm/v1".into(),
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
        decision_heads: vec![
            DecisionHead {
                name: "action".into(),
                kind: HeadKind::Choice,
                options: ["allow", "request_review", "ask_operator"].map(String::from).to_vec(),
            },
            DecisionHead { name: "risk".into(), kind: HeadKind::Score, options: vec![] },
        ],
        tau: 0.0,
        model_version: "test-1".into(),
        dataset_manifest_hash: "m".into(),
        policy_bundle_hash: "p".into(),
    }
}

/// Deterministic weights from a formula (no RNG), serialised as safetensors.
fn formula_weights(cfg: &HrmConfig, halt_bias: Option<[f32; 2]>) -> Vec<u8> {
    let vm = VarMap::new();
    Hrm::new(cfg.clone(), VarBuilder::from_varmap(&vm, DType::F32, &Device::Cpu)).unwrap();
    let mut out: HashMap<String, Tensor> = HashMap::new();
    let mut names: Vec<_> = vm.data().lock().unwrap().iter().map(|(k, v)| (k.clone(), v.clone())).collect();
    names.sort_by(|a, b| a.0.cmp(&b.0));
    for (name, var) in names {
        let n = var.elem_count();
        let salt = name.bytes().fold(0u32, |a, b| a.wrapping_mul(31).wrapping_add(b as u32)) % 11;
        let mut v: Vec<f32> = if name.ends_with("n1.weight") || name.ends_with("n2.weight") || name == "norm.weight" {
            vec![1.0; n]
        } else {
            (0..n).map(|i| 0.15 * ((0.37 * i as f32) + salt as f32).sin()).collect()
        };
        if let (Some(b), "halt.bias") = (halt_bias, name.as_str()) {
            v = b.to_vec();
        }
        out.insert(name, Tensor::from_vec(v, var.shape().clone(), &Device::Cpu).unwrap());
    }
    // safetensors buffer via candle
    static N: std::sync::atomic::AtomicUsize = std::sync::atomic::AtomicUsize::new(0);
    let dir = std::env::temp_dir().join(format!("hrm-{}-{}.safetensors", std::process::id(), N.fetch_add(1, std::sync::atomic::Ordering::SeqCst)));
    candle_core::safetensors::save(&out, &dir).unwrap();
    let bytes = std::fs::read(&dir).unwrap();
    let _ = std::fs::remove_file(dir);
    bytes
}

fn load(cfg: &HrmConfig, w: &[u8]) -> Hrm {
    Hrm::from_safetensors(cfg.clone(), w, &Device::Cpu).unwrap()
}

#[test]
fn param_count_matches_architecture() {
    let c = cfg();
    let vm = VarMap::new();
    Hrm::new(c.clone(), VarBuilder::from_varmap(&vm, DType::F32, &Device::Cpu)).unwrap();
    let total: usize = vm.all_vars().iter().map(|v| v.elem_count()).sum();
    assert_eq!(total, c.param_count());
}

#[test]
fn config_validation() {
    let mut c = cfg();
    c.d_model = 15;
    assert!(c.validate().is_err());
    let mut c = cfg();
    c.min_cycles = 4;
    assert!(c.validate().is_err());
    let mut c = cfg();
    c.format = "other".into();
    assert!(c.validate().is_err());
    let mut c = cfg();
    c.decision_heads[0].options.truncate(1);
    assert!(c.validate().is_err());
    let mut c = cfg();
    c.tau = 1.5;
    assert!(c.validate().is_err());
    assert!(HrmConfig::from_json(&serde_json::to_string(&cfg()).unwrap()).is_ok());
}

#[test]
fn rejects_wrong_weights() {
    let c = cfg();
    let w = formula_weights(&c, None);
    let mut bigger = c.clone();
    bigger.d_model = 32;
    assert!(Hrm::from_safetensors(bigger, &w, &Device::Cpu).is_err());
    assert!(Hrm::from_safetensors(c, &w[..w.len() / 2], &Device::Cpu).is_err());
}

#[test]
fn halting_respects_bounds() {
    let c = cfg();
    // Halt bias strongly favours q_halt: stops at min_cycles.
    let halt = load(&c, &formula_weights(&c, Some([50.0, -50.0])));
    assert_eq!(halt.reason(&[1, 2, 3]).unwrap().halting_steps, c.min_cycles);
    // Strongly favours continue: runs to the h_cycles cap.
    let cont = load(&c, &formula_weights(&c, Some([-50.0, 50.0])));
    assert_eq!(cont.reason(&[1, 2, 3]).unwrap().halting_steps, c.h_cycles);
    // min_cycles floor holds even when the head wants to halt immediately.
    let mut c2 = c.clone();
    c2.min_cycles = 2;
    let floor = load(&c2, &formula_weights(&c2, Some([50.0, -50.0])));
    assert_eq!(floor.reason(&[1, 2, 3]).unwrap().halting_steps, 2);
}

#[test]
fn reason_outputs_are_well_formed() {
    let c = cfg();
    let m = load(&c, &formula_weights(&c, None));
    let r = m.reason(&[3, 1, 4, 1, 5]).unwrap();
    assert_eq!(r.plan_tokens.len(), 5);
    assert!(r.plan_tokens.iter().all(|&t| (t as usize) < c.vocab_size));
    assert!((1..=c.h_cycles).contains(&r.halting_steps));
    assert_eq!(r.l_module_activations.len(), r.halting_steps);
    assert!(r.l_module_activations.iter().chain(&r.h_module_activations).all(|x| x.is_finite() && *x > 0.0));
    assert!((0.0..=1.0).contains(&r.confidence));
    let a = &r.decisions[0];
    assert!((a.probabilities.iter().sum::<f32>() - 1.0).abs() < 1e-5);
    assert_eq!(a.selected.is_some(), !a.abstain);
    let s = &r.decisions[1];
    let v = s.value.unwrap();
    assert!(v > 0.0 && v < 1.0, "score stays continuous, got {v}");
    assert_eq!(r.model_version, "test-1");
}

#[test]
fn abstains_below_tau_and_never_on_score() {
    let mut c = cfg();
    c.tau = 1.0; // max(p) < 1 always
    let m = load(&c, &formula_weights(&c, None));
    let r = m.reason(&[1, 2]).unwrap();
    assert!(r.decisions[0].abstain && r.decisions[0].selected.is_none());
    assert!(!r.decisions[1].abstain && r.decisions[1].value.is_some());
}

#[test]
fn input_validation() {
    let c = cfg();
    let m = load(&c, &formula_weights(&c, None));
    assert!(m.reason(&[]).is_err());
    assert!(m.reason(&[c.vocab_size as u32]).is_err());
    let too_long = vec![0u32; m.max_input_len() + 1];
    assert!(m.reason(&too_long).is_err());
    assert!(m.reason(&vec![0u32; m.max_input_len()]).is_ok());
}

#[test]
fn deterministic() {
    let c = cfg();
    let w = formula_weights(&c, None);
    let a = serde_json::to_string(&load(&c, &w).reason(&[7, 8, 9]).unwrap()).unwrap();
    let b = serde_json::to_string(&load(&c, &w).reason(&[7, 8, 9]).unwrap()).unwrap();
    assert_eq!(a, b);
}

/// Fixed weights + input give fixed outputs. `tests/golden.json` is the
/// contract that the WASM (Node/browser) and wazero builds are compared against.
#[test]
fn golden() {
    let c = cfg();
    let w = formula_weights(&c, None);
    let r = load(&c, &w).reason(&[3, 1, 4, 1, 5, 9, 2, 6]).unwrap();
    let path = concat!(env!("CARGO_MANIFEST_DIR"), "/tests/golden.json");
    if std::env::var("UPDATE_GOLDEN").is_ok() {
        // Fixtures for the WASM (Node) and wazero builds: same weights + config + input.
        std::fs::write(concat!(env!("CARGO_MANIFEST_DIR"), "/tests/golden.safetensors"), &w).unwrap();
        std::fs::write(concat!(env!("CARGO_MANIFEST_DIR"), "/tests/golden_config.json"), serde_json::to_string_pretty(&c).unwrap()).unwrap();
        std::fs::write(path, serde_json::to_string_pretty(&r).unwrap()).unwrap();
    }
    let want: knirv_cortex_wasm::Reasoning = serde_json::from_str(&std::fs::read_to_string(path).expect("run with UPDATE_GOLDEN=1")).unwrap();
    assert_eq!(r.plan_tokens, want.plan_tokens);
    assert_eq!(r.halting_steps, want.halting_steps);
    let close = |a: &[f32], b: &[f32]| {
        assert_eq!(a.len(), b.len());
        // relative: activations are O(10) with these formula weights
        a.iter().zip(b).for_each(|(x, y)| assert!((x - y).abs() <= 1e-5 * y.abs().max(1.0), "{x} vs {y}"));
    };
    close(&r.l_module_activations, &want.l_module_activations);
    close(&r.h_module_activations, &want.h_module_activations);
    close(&[r.confidence, r.halt_probability], &[want.confidence, want.halt_probability]);
    for (a, b) in r.decisions.iter().zip(&want.decisions) {
        close(&a.probabilities, &b.probabilities);
        close(&[a.value.unwrap_or(0.0)], &[b.value.unwrap_or(0.0)]);
    }
}

#[test]
fn safetensors_round_trip() {
    let c = cfg();
    let vm = VarMap::new();
    let m = Hrm::new(c.clone(), VarBuilder::from_varmap(&vm, DType::F32, &Device::Cpu)).unwrap();
    let path = std::env::temp_dir().join(format!("hrm-rt-{}.safetensors", std::process::id()));
    vm.save(&path).unwrap();
    let loaded = load(&c, &std::fs::read(&path).unwrap());
    let _ = std::fs::remove_file(path);
    let (a, b) = (m.reason(&[2, 4, 6]).unwrap(), loaded.reason(&[2, 4, 6]).unwrap());
    assert_eq!(a.plan_tokens, b.plan_tokens);
    assert_eq!(a.halting_steps, b.halting_steps);
    assert!((a.confidence - b.confidence).abs() < 1e-6);
}

/// One-step-gradient training lowers loss on a micro task (reverse a sequence)
/// and trains the decision head too.
#[test]
fn one_step_gradient_training_reduces_loss() {
    let c = cfg();
    let dev = Device::Cpu;
    let vm = VarMap::new();
    let m = Hrm::new(c.clone(), VarBuilder::from_varmap(&vm, DType::F32, &dev)).unwrap();
    let mut opt = AdamW::new(vm.all_vars(), ParamsAdamW { lr: 1e-3, ..Default::default() }).unwrap();

    let seqs: Vec<Vec<u32>> = (0..8u32).map(|i| (0..6).map(|j| (i * 3 + j * 5) % 16 + 1).collect()).collect();
    let x: Vec<u32> = seqs.iter().flatten().copied().collect();
    let y: Vec<u32> = seqs.iter().flat_map(|s| s.iter().rev().copied()).collect();
    let x = Tensor::from_vec(x, (8, 6), &dev).unwrap();
    let y = Tensor::from_vec(y, (8 * 6,), &dev).unwrap();
    let action = Tensor::from_vec((0..8u32).map(|i| i % 3).collect::<Vec<_>>(), (8,), &dev).unwrap();

    let step = |opt: &mut AdamW| -> f32 {
        let tr = m.forward(&x, Mode::Train).unwrap();
        let plan = loss::cross_entropy(&tr.logits.reshape((8 * 6, c.vocab_size)).unwrap(), &y).unwrap();
        let dec = loss::cross_entropy(&m.head_outputs(&tr.summary).unwrap()[0], &action).unwrap();
        let total = (plan + dec).unwrap();
        opt.backward_step(&total).unwrap();
        total.to_scalar::<f32>().unwrap()
    };
    let first = step(&mut opt);
    let mut last = first;
    for _ in 0..120 {
        last = step(&mut opt);
    }
    assert!(first.is_finite() && last.is_finite());
    assert!(last < first * 0.9, "loss {first} -> {last}");
}

mod adapters {
    use super::*;
    use knirv_cortex_wasm::adapter::{self, Adapter, AdapterLayer};
    use std::collections::BTreeMap;

    fn mat(r: usize, c: usize, seed: f32) -> Vec<Vec<f32>> {
        (0..r).map(|i| (0..c).map(|j| 0.3 * ((i * 7 + j * 3) as f32 * 0.21 + seed).sin()).collect()).collect()
    }

    fn adapter(d: usize, r: usize, roles: &[&str]) -> Adapter {
        let layers: BTreeMap<_, _> = roles
            .iter()
            .enumerate()
            .map(|(k, role)| (role.to_string(), AdapterLayer { a: mat(r, d, k as f32), b: mat(d, r, 9.0 + k as f32) }))
            .collect();
        Adapter { adapter_rank: r, scaling_alpha: 8.0, layers, content_hash: "h1".into(), stacks: vec!["l".into(), "h".into()], strength: 1.0 }
    }

    fn tensors(c: &HrmConfig) -> HashMap<String, Tensor> {
        candle_core::safetensors::load_buffer(&formula_weights(c, None), &Device::Cpu).unwrap()
    }

    fn flat(t: &Tensor) -> Vec<f32> {
        t.flatten_all().unwrap().to_vec1().unwrap()
    }

    #[test]
    fn merge_adds_scaled_ba_to_the_right_tensors_only() {
        let c = cfg();
        let base = tensors(&c);
        let mut merged = base.clone();
        let ad = adapter(c.d_model, 2, &["self_attn.q_proj", "self_attn.o_proj"]);
        let applied = adapter::merge(&c, &mut merged, &ad).unwrap();
        assert_eq!(applied.tensors_changed, 4); // q,o × (l.0, h.0)
        let scale = 8.0f32 / 2.0;
        let (a, b) = (&ad.layers["self_attn.q_proj"].a, &ad.layers["self_attn.q_proj"].b);
        let w0 = flat(&base["h.0.attn.q.weight"]);
        let w1 = flat(&merged["h.0.attn.q.weight"]);
        let d = c.d_model;
        for i in 0..d {
            for j in 0..d {
                let delta: f32 = (0..2).map(|k| b[i][k] * a[k][j]).sum::<f32>() * scale;
                assert!((w1[i * d + j] - w0[i * d + j] - delta).abs() < 1e-5, "({i},{j})");
            }
        }
        for name in ["l.0.attn.k.weight", "l.0.attn.v.weight", "l.0.mlp.up.weight", "tok.weight"] {
            assert_eq!(flat(&base[name]), flat(&merged[name]), "{name} must be untouched");
        }
    }

    #[test]
    fn adapter_changes_behaviour_and_strength_zero_does_not() {
        let c = cfg();
        let w = formula_weights(&c, None);
        let toks = [3u32, 1, 4, 1, 5];
        let base = Hrm::from_safetensors(c.clone(), &w, &Device::Cpu).unwrap().reason(&toks).unwrap();

        let mut ad = adapter(c.d_model, 4, &["self_attn.q_proj", "self_attn.k_proj", "self_attn.v_proj", "self_attn.o_proj"]);
        ad.scaling_alpha = 64.0;
        let (m, applied) = Hrm::from_safetensors_adapted(c.clone(), &w, &Device::Cpu, Some(&ad)).unwrap();
        assert_eq!(applied.unwrap().roles_applied.len(), 4);
        let tuned = m.reason(&toks).unwrap();
        assert_ne!(base.decisions[0].probabilities, tuned.decisions[0].probabilities);

        ad.strength = 0.0;
        let (m0, _) = Hrm::from_safetensors_adapted(c.clone(), &w, &Device::Cpu, Some(&ad)).unwrap();
        assert_eq!(base.decisions[0].probabilities, m0.reason(&toks).unwrap().decisions[0].probabilities);
    }

    #[test]
    fn stacks_select_where_the_delta_lands() {
        let c = cfg();
        let base = tensors(&c);
        let mut merged = base.clone();
        let mut ad = adapter(c.d_model, 2, &["q_proj"]);
        ad.stacks = vec!["h".into()];
        adapter::merge(&c, &mut merged, &ad).unwrap();
        assert_eq!(flat(&base["l.0.attn.q.weight"]), flat(&merged["l.0.attn.q.weight"]));
        assert_ne!(flat(&base["h.0.attn.q.weight"]), flat(&merged["h.0.attn.q.weight"]));
    }

    #[test]
    fn bad_adapters_are_refused_and_change_nothing() {
        let c = cfg();
        let base = tensors(&c);
        let before = flat(&base["l.0.attn.q.weight"]);
        let d = c.d_model;
        let mut cases: Vec<(&str, Adapter)> = Vec::new();
        cases.push(("wrong a shape", Adapter { layers: [("q_proj".to_string(), AdapterLayer { a: mat(2, d + 1, 0.0), b: mat(d, 2, 1.0) })].into(), ..adapter(d, 2, &[]) }));
        cases.push(("mlp only", adapter(d, 2, &["mlp.up_proj", "mlp.down_proj"])));
        let mut nan = adapter(d, 2, &["q_proj"]);
        nan.layers.get_mut("q_proj").unwrap().b[0][0] = f32::NAN;
        cases.push(("non-finite", nan));
        let mut zero = adapter(d, 2, &["q_proj"]);
        zero.adapter_rank = 0;
        cases.push(("rank 0", zero));
        let mut st = adapter(d, 2, &["q_proj"]);
        st.stacks = vec!["x".into()];
        cases.push(("unknown stack", st));
        // a good role followed by a bad one: nothing may be half-applied
        let mut mixed = adapter(d, 2, &["k_proj"]);
        mixed.layers.insert("v_proj".into(), AdapterLayer { a: mat(1, d, 0.0), b: mat(d, 2, 0.0) });
        cases.push(("partial", mixed));
        for (name, ad) in cases {
            let mut t = base.clone();
            assert!(adapter::merge(&c, &mut t, &ad).is_err(), "{name} accepted");
            assert_eq!(flat(&t["l.0.attn.q.weight"]), before, "{name}");
            assert_eq!(flat(&t["l.0.attn.k.weight"]), flat(&base["l.0.attn.k.weight"]), "{name} left a partial merge");
        }
    }

    #[test]
    fn lists_adaptable_tensors() {
        let names = adapter::adaptable_tensors(&cfg());
        assert_eq!(names.len(), 8);
        assert!(names.contains(&"h.0.attn.o.weight".to_string()));
    }
}
