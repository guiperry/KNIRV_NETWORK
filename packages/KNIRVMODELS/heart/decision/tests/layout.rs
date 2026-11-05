//! Task 4.2: positions, the additive mask and choice isolation, through the full recurrent forward.

use candle_core::{DType, Device, Tensor};
use candle_nn::{VarBuilder, VarMap};
use knirv_heart_decision::{assert_additive, tokenizer, ChoiceLayout, DecisionHead, Head, HeadConfig, HeadKind, Mode};
use std::collections::HashMap;

const WIDTH: usize = 64;

fn cfg() -> HeadConfig {
    HeadConfig {
        format: "knirv-heart/v1".into(),
        tokenizer: "byte".into(),
        vocab_size: tokenizer::VOCAB,
        max_seq_len: WIDTH + 2,
        d_model: 16,
        n_heads: 2,
        ffn_mult: 2,
        l_layers: 2,
        h_layers: 1,
        h_cycles: 3,
        l_steps: 2,
        min_cycles: 1,
        summary_slots: 1,
        decision_heads: vec![DecisionHead { name: "action".into(), kind: HeadKind::Choice, options: ["a", "b", "c"].map(String::from).to_vec() }],
        tau: 0.0,
        model_version: "layout-test".into(),
        dataset_manifest_hash: "m".into(),
        policy_bundle_hash: "p".into(),
        choice_set_heads: vec![],
    }
}

/// Deterministic weights from a formula (no RNG).
fn head(c: &HeadConfig) -> Head {
    let vm = VarMap::new();
    Head::new(c.clone(), VarBuilder::from_varmap(&vm, DType::F32, &Device::Cpu)).unwrap();
    let mut out: HashMap<String, Tensor> = HashMap::new();
    for (name, var) in vm.data().lock().unwrap().iter() {
        let n = var.elem_count();
        let salt = name.bytes().fold(0u32, |a, b| a.wrapping_mul(31).wrapping_add(b as u32)) % 11;
        let v: Vec<f32> = if name.ends_with("n1.weight") || name.ends_with("n2.weight") || name == "norm.weight" {
            vec![1.0; n]
        } else {
            (0..n).map(|i| 0.15 * ((0.37 * i as f32) + salt as f32).sin()).collect()
        };
        out.insert(name.clone(), Tensor::from_vec(v, var.shape().clone(), &Device::Cpu).unwrap());
    }
    Head::new(c.clone(), VarBuilder::from_tensors(out, DType::F32, &Device::Cpu)).unwrap()
}

fn prompt(question: &str, choices: &[&str]) -> (Tensor, ChoiceLayout, Vec<u32>) {
    let cs: Vec<String> = choices.iter().map(|c| c.to_string()).collect();
    let toks = tokenizer::encode_set_prompt(question, "host=idle", &cs, WIDTH);
    let layout = ChoiceLayout::from_tokens(&toks).unwrap();
    (Tensor::from_slice(&toks, (1, toks.len()), &Device::Cpu).unwrap(), layout, toks)
}

fn flat(t: &Tensor) -> Vec<f32> {
    t.flatten_all().unwrap().to_vec1::<f32>().unwrap()
}

/// (a) With no layout the new entry point is the v1 computation, bit for bit, and the
/// committed golden contract (which `tests/head.rs::golden` checks against `golden.json`) is
/// untouched: `forward` is now a thin call into the same code.
#[test]
fn no_layout_reproduces_the_v1_forward_bit_for_bit() {
    let c = cfg();
    let m = head(&c);
    let x = Tensor::from_slice(&[3u32, 1, 4, 1, 5, 9, 2, 6], (1, 8), &Device::Cpu).unwrap();
    for mode in [Mode::Infer, Mode::Train] {
        let a = m.forward(&x, mode).unwrap();
        let b = m.forward_layout(&x, mode, None).unwrap();
        assert_eq!(flat(&a.logits), flat(&b.logits));
        assert_eq!(flat(&a.summary), flat(&b.summary));
        assert_eq!(flat(&a.halt), flat(&b.halt));
        assert_eq!(a.cycles, b.cycles);
    }
    // And an all-allowed additive bias (zeros) changes nothing either: the bias path adds exactly 0.
    let total = c.summary_slots + 8;
    let zeros = Tensor::zeros((total, total), DType::F32, &Device::Cpu).unwrap();
    let a = m.forward(&x, Mode::Infer).unwrap();
    let z = m.forward_with_bias(&x, Mode::Infer, &zeros).unwrap();
    assert_eq!(flat(&a.logits), flat(&z.logits));
}

/// (b) Isolation, through the full recurrent forward (3 cycles x 2 L steps, 2 L layers):
/// changing choice j's bytes leaves every other choice's pooled vector identical, bit for bit.
#[test]
fn changing_one_choice_leaves_the_others_identical() {
    let m = head(&cfg());
    let (xa, la, _) = prompt("why?", &["wait", "restart", "escalate"]);
    let (xb, lb, _) = prompt("why?", &["wait", "RESTART NOW", "escalate"]);
    assert_eq!(la.choice_spans.len(), 3);
    // The second choice differs in length, so the third sits elsewhere; compare by choice.
    let ea = m.embed_choices(&xa, &la).unwrap();
    let eb = m.embed_choices(&xb, &lb).unwrap();
    let row = |t: &Tensor, i: usize| flat(&t.narrow(1, i, 1).unwrap());
    assert_eq!(row(&ea, 0), row(&eb, 0), "choice 0 moved when choice 1 changed");
    assert_eq!(row(&ea, 2), row(&eb, 2), "choice 2 moved when choice 1 changed");
    assert_ne!(row(&ea, 1), row(&eb, 1), "the changed choice should differ");

    // The question and state do reach every choice (that is their job).
    let (xq, lq, _) = prompt("what now?", &["wait", "restart", "escalate"]);
    let eq = m.embed_choices(&xq, &lq).unwrap();
    assert_ne!(row(&ea, 0), row(&eq, 0), "the question must still condition the choice");

    // Same-length edit, in every position.
    for (i, edited) in [["XAIT", "restart", "escalate"], ["wait", "RESTART", "escalate"], ["wait", "restart", "ESCALATE"]].iter().enumerate() {
        let (xe, le, _) = prompt("why?", edited);
        let ee = m.embed_choices(&xe, &le).unwrap();
        for j in 0..3 {
            if j != i {
                assert_eq!(row(&ea, j), row(&ee, j), "editing choice {i} moved choice {j}");
            }
        }
    }
}

/// (c) A 1/0 keep-mask must fail: the mask is additive 0 / -inf.
#[test]
fn a_one_zero_mask_is_refused() {
    let c = cfg();
    let m = head(&c);
    let x = Tensor::from_slice(&[3u32, 1, 4, 1], (1, 4), &Device::Cpu).unwrap();
    let total = c.summary_slots + 4;
    let keep: Vec<f32> = (0..total * total).map(|i| if i % 3 == 0 { 0.0 } else { 1.0 }).collect();
    let keep = Tensor::from_vec(keep, (total, total), &Device::Cpu).unwrap();
    let err = m.forward_with_bias(&x, Mode::Infer, &keep).err().expect("a 1/0 mask must fail").to_string();
    assert!(err.contains("additive"), "{err}");
    assert!(assert_additive(&keep).is_err());
    // A mask of the wrong size is also refused rather than broadcast.
    let wrong = Tensor::zeros((total + 1, total + 1), DType::F32, &Device::Cpu).unwrap();
    assert!(m.forward_with_bias(&x, Mode::Infer, &wrong).is_err());
    // The mask the layout builds is additive.
    let (_, layout, toks) = prompt("why?", &["wait", "restart"]);
    assert_additive(&layout.bias(c.summary_slots, toks.len(), &Device::Cpu).unwrap()).unwrap();
}

/// (d) The same choice gives the same embedding in a 3-choice and a 2-choice input, and at
/// any position in the list.
#[test]
fn a_choice_embeds_the_same_in_any_company() {
    let m = head(&cfg());
    let row = |t: &Tensor, i: usize| flat(&t.narrow(1, i, 1).unwrap());
    let (x3, l3, _) = prompt("why?", &["wait", "restart", "escalate"]);
    let (x2, l2, _) = prompt("why?", &["wait", "restart"]);
    let (x1, l1, _) = prompt("why?", &["wait"]);
    let (xs, ls, _) = prompt("why?", &["escalate", "wait", "restart"]);
    let (e3, e2, e1, es) = (m.embed_choices(&x3, &l3).unwrap(), m.embed_choices(&x2, &l2).unwrap(), m.embed_choices(&x1, &l1).unwrap(), m.embed_choices(&xs, &ls).unwrap());
    assert_eq!(row(&e3, 0), row(&e2, 0), "'wait' differs between a 3-choice and a 2-choice input");
    assert_eq!(row(&e3, 0), row(&e1, 0));
    assert_eq!(row(&e3, 1), row(&e2, 1));
    // Permuting the list permutes the embeddings and changes nothing else.
    assert_eq!(row(&e3, 0), row(&es, 1));
    assert_eq!(row(&e3, 1), row(&es, 2));
    assert_eq!(row(&e3, 2), row(&es, 0));
}

/// The padding after the last choice is never attended to: more padding, same embeddings.
#[test]
fn padding_width_does_not_change_a_choice() {
    let m = head(&cfg());
    let cs = ["wait".to_string(), "restart".to_string()];
    let embed = |width: usize| {
        let toks = tokenizer::encode_set_prompt("why?", "host=idle", &cs, width);
        let l = ChoiceLayout::from_tokens(&toks).unwrap();
        let x = Tensor::from_slice(&toks, (1, toks.len()), &Device::Cpu).unwrap();
        flat(&m.embed_choices(&x, &l).unwrap())
    };
    assert_eq!(embed(40), embed(60));
}

/// A layout for an input it does not describe is an error, never a silent mis-mask.
#[test]
fn a_mismatched_layout_is_an_error() {
    let m = head(&cfg());
    let (x, _, _) = prompt("why?", &["wait", "restart"]);
    let bad = ChoiceLayout { question_len: 3, choice_spans: vec![(3, 4)], valid_len: None };
    assert!(m.forward_layout(&x, Mode::Infer, Some(&bad)).is_err());
    assert!(m.embed_choices(&x, &bad).is_err());
}

/// The set-head forward still halts and summarises: the layout changes attention, not the interface.
#[test]
fn the_layout_forward_has_the_same_outputs_as_the_fixed_one() {
    let c = cfg();
    let m = head(&c);
    let (x, l, toks) = prompt("why?", &["wait", "restart", "escalate"]);
    for mode in [Mode::Infer, Mode::Train] {
        let t = m.forward_layout(&x, mode, Some(&l)).unwrap();
        assert_eq!(t.logits.dims(), &[1, toks.len(), c.vocab_size]);
        assert_eq!(t.summary.dims(), &[1, c.d_model]);
        assert!(t.cycles >= 1 && t.cycles <= c.h_cycles);
        assert!(flat(&t.logits).iter().all(|v| v.is_finite()), "a masked row produced NaN or inf");
        assert!(flat(&t.summary).iter().all(|v| v.is_finite()));
    }
    // The summary reads the choices: with the layout it differs from the unmasked one.
    let masked = m.forward_layout(&x, Mode::Train, Some(&l)).unwrap();
    let plain = m.forward(&x, Mode::Train).unwrap();
    assert_ne!(flat(&masked.summary), flat(&plain.summary));
}
