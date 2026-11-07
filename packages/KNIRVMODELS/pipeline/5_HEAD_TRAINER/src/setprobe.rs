//! Set-head training path for the Task 4.4 kill-gate probe (Training 4.4).
//!
//! The decision-head trainer learns fixed option vocabularies; a choice-set head instead
//! scores a request's *own* candidates, so its supervision is different and needs its own
//! path:
//!
//! * **per-batch choice shuffle with target remap** — within every step the candidate list
//!   is reordered and the target index is remapped, so position is never a cue (H4/H9);
//! * **variable N with masked logits** — each question carries 4..=8 candidates; the model is
//!   padded to `max_choices` and the padding rows get `-inf` logits, so they can never win;
//! * **cross-entropy per cycle** — like the decision heads, every high-level cycle is
//!   deep-supervised on its own one-step gradient, then the cycles' losses are averaged.
//!
//! Training processes one question per forward (the variable N makes padded batching awkward);
//! the probe is small, so this is fast enough on a single GPU.

use crate::data::Split;
use candle_core::{DType, Device, Result, Tensor};
use std::collections::HashMap;
use candle_nn::{optim::AdamW, Optimizer, ParamsAdamW, VarMap};
use knirv_heart_decision::tokenizer::encode_set_prompt;
use knirv_heart_decision::{overlap_feature, softmax_last_dim_diff, ChoiceLayout, Head};

#[derive(Clone, Debug)]
pub struct SetExample {
    pub id: String,
    pub group: String,
    pub split: Split,
    pub question: String,
    pub state: String,
    /// Canonical candidate order; `target` indexes this. The training loop shuffles a copy.
    pub candidates: Vec<String>,
    pub target: usize,
}

#[derive(Clone, Debug)]
pub struct SetTrainOpts {
    pub epochs: usize,
    pub lr: f64,
    pub warmup_steps: usize,
    pub max_steps: Option<usize>,
    pub seed: u64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn load_set_examples_parses_required_and_optional_fields() {
        let path = std::env::temp_dir().join("knirv_set_load_test.jsonl");
        let body = concat!(
            "{\"state\":\"q: route\",\"candidates\":[\"a\",\"b\",\"c\"],\"target\":1}\n",
            "{\"question\":\"pick\",\"state\":\"s\",\"candidates\":[\"x\",\"y\"],\"target\":0,\"split\":\"test\",\"group\":\"g1\"}\n",
        );
        std::fs::write(&path, body).unwrap();
        let ex = load_set_examples(path.to_str().unwrap()).unwrap();
        assert_eq!(ex.len(), 2);
        assert_eq!(ex[0].candidates, vec!["a", "b", "c"]);
        assert_eq!(ex[0].target, 1);
        assert_eq!(ex[0].question, "");
        assert_eq!(ex[0].split, Split::Train);
        assert_eq!(ex[1].question, "pick");
        assert_eq!(ex[1].split, Split::Test);
        assert_eq!(ex[1].group, "g1");
        assert_eq!(ex[1].target, 0);
        let _ = std::fs::remove_file(&path);
    }

    #[test]
    fn load_set_examples_rejects_too_few_candidates_and_bad_target() {
        let dir = std::env::temp_dir();
        let bad1 = dir.join("knirv_set_bad1.jsonl");
        std::fs::write(&bad1, "{\"state\":\"s\",\"candidates\":[\"only\"],\"target\":0}\n").unwrap();
        assert!(load_set_examples(bad1.to_str().unwrap()).is_err());
        let bad2 = dir.join("knirv_set_bad2.jsonl");
        std::fs::write(&bad2, "{\"state\":\"s\",\"candidates\":[\"a\",\"b\"],\"target\":5}\n").unwrap();
        assert!(load_set_examples(bad2.to_str().unwrap()).is_err());
        let _ = std::fs::remove_file(&bad1);
        let _ = std::fs::remove_file(&bad2);
    }
}

pub struct Xorshift(pub u64);
impl Xorshift {
    fn next(&mut self) -> u64 {
        self.0 ^= self.0 << 13;
        self.0 ^= self.0 >> 7;
        self.0 ^= self.0 << 17;
        self.0
    }
    fn shuffle<T>(&mut self, v: &mut [T]) {
        for i in (1..v.len()).rev() {
            v.swap(i, (self.next() % (i as u64 + 1)) as usize);
        }
    }
}

/// Shuffles a question's candidates (per-batch shuffle + target remap) and encodes the prompt.
/// Returns the token row, the validity mask up to `max_choices`, and the remapped target index.
pub fn encode_shuffled(ex: &SetExample, max_choices: usize, width: usize, rng: &mut Xorshift, dev: &Device) -> Result<(Vec<u32>, Vec<bool>, usize, Tensor)> {
    let mut order: Vec<usize> = (0..ex.candidates.len()).collect();
    rng.shuffle(&mut order);
    let target = order.iter().position(|&i| i == ex.target).expect("target present");
    let cand: Vec<String> = order.iter().map(|&i| ex.candidates[i].clone()).collect();
    let tokens = encode_set_prompt(&ex.question, &ex.state, &cand, width);
    let n = cand.len();
    // One question per forward (variable N): the prompt carries exactly `n` candidates and
    // they are all valid. (Batched training with padded candidates would extend this mask.)
    let valid = vec![true; n];
    let feat = Tensor::from_vec(overlap_feature(&ex.question, &cand), (1, cand.len(), 1), dev)?;
    Ok((tokens, valid, target, feat))
}

/// Loads a JSONL file of set questions into [`SetExample`]s for real (non-synthetic) set-head
/// training. Each line:
/// `{"question":"...","state":"...","candidates":["opt a","opt b"],"target":N[, "split":"train"|"test"|"val","group":"..."]}`.
/// `question` defaults to ""; `split` defaults to Train; `group` defaults to "".
pub fn load_set_examples(path: &str) -> Result<Vec<SetExample>> {
    let txt = std::fs::read_to_string(path).map_err(|x| candle_core::Error::Msg(format!("{path}: {x}").into()))?;
    let mut out = Vec::new();
    for (i, line) in txt.lines().enumerate().filter(|(_, l)| !l.trim().is_empty()) {
        let v: serde_json::Value = serde_json::from_str(line).map_err(|e| candle_core::Error::Msg(format!("{path}:{i}: {e}").into()))?;
        let question = v["question"].as_str().unwrap_or("").to_string();
        let state = v["state"].as_str().ok_or_else(|| candle_core::Error::Msg(format!("{path}:{i}: missing state").into()))?.to_string();
        let candidates: Vec<String> = v["candidates"]
            .as_array()
            .ok_or_else(|| candle_core::Error::Msg(format!("{path}:{i}: missing candidates").into()))?
            .iter()
            .map(|c| c.as_str().unwrap_or("").to_string())
            .collect();
        if candidates.len() < 2 {
            return Err(candle_core::Error::Msg(format!("{path}:{i}: need >=2 candidates").into()));
        }
        let target = v["target"].as_u64().ok_or_else(|| candle_core::Error::Msg(format!("{path}:{i}: missing target").into()))? as usize;
        if target >= candidates.len() {
            return Err(candle_core::Error::Msg(format!("{path}:{i}: target out of range").into()));
        }
        let split = match v["split"].as_str().unwrap_or("train") {
            "test" => Split::Test,
            "val" | "validation" => Split::Val,
            _ => Split::Train,
        };
        let group = v["group"].as_str().unwrap_or("").to_string();
        out.push(SetExample { id: format!("{i}"), group, split, question, state, candidates, target });
    }
    Ok(out)
}

/// Cross-entropy against the (remapped) target, averaged over the high-level cycles'
/// pre-softmax set logits. Invalid (padding) rows carry `-inf` logits and contribute nothing.
pub fn set_loss(model: &Head, tokens: &Tensor, head: usize, valid: &[bool], target: usize, match_feat: &Tensor) -> Result<Tensor> {
    let layout = ChoiceLayout::from_tokens(&tokens.to_vec2::<u32>()?.remove(0))?;
    let cycles = model.set_cycle_logits(tokens, &layout, head, valid, match_feat)?;
    // Per-cycle deep supervision: each cycle's logits yield the NLL of the target via the
    // lib's `set_loss_from_logits` (the narrow(...).neg() gather that is correct on CPU and CUDA),
    // then the per-cycle NLLs are averaged — same target deep-supervision as the decision heads.
    let mut total: Option<Tensor> = None;
    for logits in &cycles {
        let nll = knirv_heart_decision::set_loss_from_logits(logits, target)?; // owned [1, 1] Tensor
        total = Some(match total {
            Some(x) => (x + nll)?, // fallible add; keeps the autograd graph
            None => nll,
        });
    }
    let sum = total.expect("h_cycles >= 1"); // owned Tensor
    Ok((sum * (1.0 / cycles.len() as f64))?) // fallible mul
}

/// Trains one choice-set head from the synthetic `examples` (Task 4.4). Returns the mean
/// loss per epoch. Mirrors the decision-head [`crate::train::train`] optimiser split.
pub fn train_set(
    model: &Head,
    vm: &VarMap,
    examples: &[SetExample],
    head_index: usize,
    max_choices: usize,
    width: usize,
    o: &SetTrainOpts,
    dev: &Device,
    mut on_epoch: impl FnMut(usize, f32),
) -> Result<Vec<f32>> {
    if examples.is_empty() {
        candle_core::bail!("no set examples");
    }
    let steps_per_epoch = examples.len();
    let total_steps = steps_per_epoch * o.epochs.min(o.max_steps.unwrap_or(usize::MAX));
    let (decayed, plain) = {
        let data = vm.data().lock().unwrap();
        let mut names: Vec<_> = data.keys().cloned().collect();
        names.sort();
        let (mut d, mut n) = (Vec::new(), Vec::new());
        for name in names {
            let v = data[&name].clone();
            if crate::train::decays(&name) {
                d.push(v);
            } else {
                n.push(v);
            }
        }
        (d, n)
    };
    let mut opt = AdamW::new(decayed, ParamsAdamW { lr: o.lr, weight_decay: 0.01, ..Default::default() })?;
    let mut opt_plain = AdamW::new(plain, ParamsAdamW { lr: o.lr, weight_decay: 0.0, ..Default::default() })?;
    let mut rng = Xorshift(o.seed.wrapping_mul(2).wrapping_add(1));
    let mut order: Vec<usize> = (0..examples.len()).collect();
    let (mut step, mut history) = (0usize, Vec::new());
    for epoch in 0..o.epochs {
        rng.shuffle(&mut order);
        let (mut sum, mut n) = (0f32, 0usize);
        for &idx in &order {
            if step >= total_steps {
                break;
            }
            let lr = lr_at(step, total_steps, o);
            opt.set_learning_rate(lr);
            opt_plain.set_learning_rate(lr);
            let (tokens, valid, target, feat) = encode_shuffled(&examples[idx], max_choices, width, &mut rng, dev)?;
            let input = Tensor::from_slice(&tokens, (1, tokens.len()), dev)?;
            let l = set_loss(model, &input, head_index, &valid, target, &feat)?;
            let v = l.to_dtype(DType::F32)?.to_vec2::<f32>()?.remove(0).remove(0);
            if !v.is_finite() {
                candle_core::bail!("set loss became {v} at step {step}; lower --lr (one-step-gradient HEAD diverges above ~3e-3)");
            }
            let grads = l.backward()?;
            if step == 0 {
                let data = vm.data().lock().unwrap();
                for (name, t) in data.iter() {
                    match grads.get(t) {
                        Some(g) => {
                            let gn = g.sqr()?.sum_all()?.to_scalar::<f32>()?.sqrt();
                            eprintln!("  [diag] grad {name} norm {gn:.5}");
                        }
                        None => eprintln!("  [diag] NO GRAD {name}"),
                    }
                }
            }
            opt.step(&grads)?;
            opt_plain.step(&grads)?;
            sum += v;
            n += 1;
            step += 1;
        }
        let mean = sum / n.max(1) as f32;
        on_epoch(epoch, mean);
        history.push(mean);
    }
    Ok(history)
}

fn lr_at(step: usize, total: usize, o: &SetTrainOpts) -> f64 {
    let warmup = o.warmup_steps.min((total / 10).max(1));
    if step < warmup {
        return o.lr * (step + 1) as f64 / warmup as f64;
    }
    let p = (step - warmup) as f64 / (total.saturating_sub(warmup)).max(1) as f64;
    o.lr * (0.1 + 0.9 * 0.5 * (1.0 + (std::f64::consts::PI * p.min(1.0)).cos()))
}

/// Streaming variant of [`train_set`] for the E1 experiment: instead of looping over a fixed
/// `examples` vec, it draws a *fresh* example every step from `gen` (which should sample from a
/// large vocabulary so the model cannot memorise), with new per-step seeds and a warm-up + cosine
/// schedule over `o.max_steps`. This is the primary Task 4.4 suspect — whether a set head can
/// learn the matching rule from unlimited fresh data rather than overfitting a tiny fixed set.
pub fn train_set_fresh<F>(
    model: &Head,
    vm: &VarMap,
    head_index: usize,
    max_choices: usize,
    width: usize,
    o: &SetTrainOpts,
    dev: &Device,
    use_feature: bool,
    mut gen: F,
    mut on_epoch: impl FnMut(usize, f32),
) -> Result<Vec<f32>>
where
    F: FnMut(usize) -> SetExample,
{
    let total_steps = o.max_steps.ok_or_else(|| candle_core::Error::msg("train_set_fresh needs max_steps"))?;
    if total_steps == 0 {
        candle_core::bail!("no steps");
    }
    let (decayed, plain) = {
        let data = vm.data().lock().unwrap();
        let mut names: Vec<_> = data.keys().cloned().collect();
        names.sort();
        let (mut d, mut n) = (Vec::new(), Vec::new());
        for name in names {
            let v = data[&name].clone();
            if crate::train::decays(&name) {
                d.push(v);
            } else {
                n.push(v);
            }
        }
        (d, n)
    };
    let mut opt = AdamW::new(decayed, ParamsAdamW { lr: o.lr, weight_decay: 0.01, ..Default::default() })?;
    let mut opt_plain = AdamW::new(plain, ParamsAdamW { lr: o.lr, weight_decay: 0.0, ..Default::default() })?;
    let log_every = (total_steps / 30).max(100).min(2000);
    let mut step = 0usize;
    let mut history = Vec::new();
    let mut loss_sum = 0f32;
    let mut loss_n = 0usize;
    while step < total_steps {
        let lr = lr_at(step, total_steps, o);
        opt.set_learning_rate(lr);
        opt_plain.set_learning_rate(lr);
        let mut rng = Xorshift((o.seed ^ step as u64).wrapping_mul(2).wrapping_add(1));
        let ex = gen(step);
        let (tokens, valid, target, feat) = encode_shuffled(&ex, max_choices, width, &mut rng, dev)?;
        let feat = if use_feature { feat } else { Tensor::zeros((1, valid.len(), 1), DType::F32, dev)? };
        let input = Tensor::from_slice(&tokens, (1, tokens.len()), dev)?;
        // (unused) capture the final loss scalar before it goes out of scope
        let l = set_loss(model, &input, head_index, &valid, target, &feat)?;
        let v = l.to_dtype(DType::F32)?.to_vec2::<f32>()?.remove(0).remove(0);
        if !v.is_finite() {
            candle_core::bail!("set loss became {v} at step {step}; lower --lr");
        }
        let grads = l.backward()?;
        if step == 0 {
            let data = vm.data().lock().unwrap();
            for (name, t) in data.iter() {
                match grads.get(t) {
                    Some(g) => {
                        let gn = g.sqr()?.sum_all()?.to_scalar::<f32>()?.sqrt();
                        eprintln!("  [diag] grad {name} norm {gn:.5}");
                    }
                    None => eprintln!("  [diag] NO GRAD {name}"),
                }
            }
            // DEBUG: what does the *initial* model predict on a fresh example? If pred==target
            // already, the loss is degenerate (not learning); if pred!=target, loss should be ~1.5.
            let ex0 = gen(0);
            let mut r0 = Xorshift((o.seed ^ 0).wrapping_mul(2).wrapping_add(1));
            let (toks0, valid0, tgt0, feat0) = encode_shuffled(&ex0, max_choices, width, &mut r0, dev)?;
            let feat0 = if use_feature { feat0 } else { Tensor::zeros((1, valid0.len(), 1), DType::F32, dev)? };
            let inp0 = Tensor::from_slice(&toks0, (1, toks0.len()), dev)?;
            let lay0 = ChoiceLayout::from_tokens(&toks0)?;
            let cyc0 = model.set_cycle_logits(&inp0, &lay0, head_index, &valid0, &feat0)?;
            for (ci, lg) in cyc0.iter().enumerate() {
                let p = softmax_last_dim_diff(lg)?.to_vec2::<f32>()?.remove(0);
                let pr = argmax(&p);
                eprintln!("  [dbg] init cycle {ci} pred {pr} target {tgt0} top5 {:?}", &p[..5.min(p.len())]);
            }
        }
        opt.step(&grads)?;
        opt_plain.step(&grads)?;
        loss_sum += v;
        loss_n += 1;
        step += 1;
        if step % log_every == 0 {
            let mean = loss_sum / loss_n.max(1) as f32;
            on_epoch(step, mean);
            history.push(mean);
            loss_sum = 0.0;
            loss_n = 0;
        }
    }
    if loss_n > 0 {
        history.push(loss_sum / loss_n as f32);
    }
    // post-training debug: does the model actually fit the *shuffled* training distribution?
    // If pred==target here but canonical eval is random, the model exploited a shuffle-specific
    // artifact (train/eval mismatch) rather than learning the rule.
    {
        let ex = gen(0);
        let mut rr = Xorshift(o.seed.wrapping_mul(2).wrapping_add(1));
        let (toks, valid, tgt, feat) = encode_shuffled(&ex, max_choices, width, &mut rr, dev)?;
        let feat = if use_feature { feat } else { Tensor::zeros((1, valid.len(), 1), DType::F32, dev)? };
        let inp = Tensor::from_slice(&toks, (1, toks.len()), dev)?;
        let lay = ChoiceLayout::from_tokens(&toks)?;
        let cyc = model.set_cycle_logits(&inp, &lay, head_index, &valid, &feat)?;
        let lg = cyc.last().unwrap();
        let p = softmax_last_dim_diff(lg)?.to_vec2::<f32>()?.remove(0);
        let pr = argmax(&p);
        let feat_v = feat.flatten_all()?.to_vec1::<f32>()?;
        eprintln!("  [dbg end] shuffled pred {pr} target {tgt} (fits train dist? {})", pr == tgt);
        eprintln!("  [dbg end] feat (overlap) per candidate {:?}", &feat_v);
    }
    Ok(history)
}

pub fn argmax(v: &[f32]) -> usize {
    v.iter().enumerate().max_by(|a, b| a.1.total_cmp(b.1)).map(|x| x.0).unwrap_or(0)
}

/// Accuracy on `examples` using the SAME shuffled distribution as training (the correct
/// candidate is placed at a random position each time, matching how the model was trained),
/// plus the random base rate `mean(1/N)`. Evaluating in canonical order (correct always at
/// index 0) is unfair: the model learns "avoid position 0" from shuffled data and fails a
/// fixed-order test. Shuffling the eval makes it a faithful generalization check.
pub fn evaluate_set(model: &Head, examples: &[SetExample], head_index: usize, max_choices: usize, width: usize, dev: &Device, use_feature: bool) -> Result<(f32, f32)> {
    if examples.is_empty() {
        candle_core::bail!("no set examples to evaluate");
    }
    let mut correct = 0usize;
    let mut total = 0usize;
    let mut inv = 0f32;
    let mut rng = Xorshift(0x9e37_79b9_1234_5678);
    for ex in examples {
        let n = ex.candidates.len();
        let mut order: Vec<usize> = (0..n).collect();
        rng.shuffle(&mut order);
        let target = order.iter().position(|&i| i == ex.target).expect("target present");
        let cand: Vec<String> = order.iter().map(|&i| ex.candidates[i].clone()).collect();
        let tokens = encode_set_prompt(&ex.question, &ex.state, &cand, width);
        let layout = ChoiceLayout::from_tokens(&tokens)?;
        let valid = vec![true; n];
        let t = Tensor::from_slice(&tokens, (1, tokens.len()), dev)?;
        let feat = if use_feature { Tensor::from_vec(overlap_feature(&ex.question, &cand), (1, n, 1), dev)? } else { Tensor::zeros((1, n, 1), DType::F32, dev)? };
        let cycles = model.set_cycle_logits(&t, &layout, head_index, &valid, &feat)?;
        let logits = cycles.last().expect("h_cycles >= 1");
        let p = softmax_last_dim_diff(logits)?.to_vec2::<f32>()?.remove(0);
        let pred = argmax(&p);
        if pred == target {
            correct += 1;
        }
        total += 1;
        inv += 1.0 / n as f32;
    }
    Ok((correct as f32 / total as f32, inv / total as f32))
}

/// Sanity-ladder overfit / gradient / weight-delta diagnostic (Ori, 2026-11-05).
///
/// Trains on a small FIXED set of `examples` (repeated, batch=1) for `o.max_steps` steps and
/// reports every rung:
///   (1) loss curve — overfitting MUST drive loss well below 0.1 and train-acc → 1.0; if not,
///       training is broken (the set head cannot even memorise 8 examples);
///   (2) weight checksum before/after + the per-tensor delta norms (top changed tensors) — does
///       anything the forward path actually uses change?;
///   (3) grad norms for EVERY parameter at step 0 and step 100 — any forward-path param whose
///       grad is missing/zero means the autograd graph is cut somewhere (one-step detach,
///       carried-state detach, pad-to-32 softmax workaround, or the narrow-gather loss);
///   (4) the lr schedule value at step 0/100/1000 — confirms warmup + cosine actually apply.
/// `const_lr` bypasses the warmup+cosine schedule (pure constant lr) for a clean overfit test.
/// Returns the per-step loss history.
pub fn overfit_diag(
    model: &Head,
    vm: &VarMap,
    examples: &[SetExample],
    head_index: usize,
    max_choices: usize,
    width: usize,
    o: &SetTrainOpts,
    dev: &Device,
    use_feature: bool,
    const_lr: bool,
    on_step: impl Fn(usize, f32),
) -> Result<(Vec<f32>, f32, f32, f32, Option<usize>)> {
    let snap0 = snapshot(vm);
    let (decayed, plain) = {
        let data = vm.data().lock().unwrap();
        let mut names: Vec<_> = data.keys().cloned().collect();
        names.sort();
        let (mut d, mut n) = (Vec::new(), Vec::new());
        for name in names {
            let v = data[&name].clone();
            if crate::train::decays(&name) {
                d.push(v);
            } else {
                n.push(v);
            }
        }
        (d, n)
    };
    eprintln!("  [opt] decayed vars {} plain vars {} (both must be non-empty)", decayed.len(), plain.len());
    let mut opt = AdamW::new(decayed, ParamsAdamW { lr: o.lr, weight_decay: 0.01, ..Default::default() })?;
    let mut opt_plain = AdamW::new(plain, ParamsAdamW { lr: o.lr, weight_decay: 0.0, ..Default::default() })?;
    let mut rng = Xorshift(o.seed.wrapping_mul(2).wrapping_add(1));
    let n_ex = examples.len();
    let total = o.max_steps.unwrap_or(500);
    let mut step = 0usize;
    let mut snap50: Option<HashMap<String, Vec<f32>>> = None;
    let mut history = Vec::new();
    let mut final_loss = f32::INFINITY;
    let mut first_loss_below_005: Option<usize> = None;
    while step < total {
        let lr = if const_lr { o.lr } else { lr_at(step, total, o) };
        if step == 0 || step == 100 || step == 1000 {
            eprintln!("  [lr] step {step} lr {lr:.6}");
        }
        opt.set_learning_rate(lr);
        opt_plain.set_learning_rate(lr);
        let idx = step % n_ex;
        let (tokens, valid, target, feat) = encode_shuffled(&examples[idx], max_choices, width, &mut rng, dev)?;
        let input = Tensor::from_slice(&tokens, (1, tokens.len()), dev)?;
        let l = set_loss(model, &input, head_index, &valid, target, &feat)?;
        let v = l.to_dtype(DType::F32)?.to_vec2::<f32>()?.remove(0).remove(0);
        if !v.is_finite() {
            candle_core::bail!("overfit loss became {v} at step {step}; lower --lr");
        }
        let grads = l.backward()?;
        if step == 0 || step == 100 {
            eprintln!("  [grad@{}] -----", step);
            let data = vm.data().lock().unwrap();
            for (name, t) in data.iter() {
                match grads.get(t) {
                    Some(g) => {
                        let gn = g.sqr()?.sum_all()?.to_scalar::<f32>()?.sqrt();
                        eprintln!("  [grad@{step}] {name} norm {gn:.6}");
                    }
                    None => eprintln!("  [grad@{step}] NO GRAD {name}"),
                }
            }
        }
        opt.step(&grads)?;
        opt_plain.step(&grads)?;
        if step == 50 {
            snap50 = Some(snapshot(vm));
        }
        if step % 50 == 0 {
            on_step(step, v);
        }
        final_loss = v;
        if first_loss_below_005.is_none() && v < 0.05 {
            first_loss_below_005 = Some(step);
        }
        step += 1;
    }
    // (2) Weight-delta diagnostics: checksum + top changed tensors.
    let snapE = snapshot(vm);
    let c0 = checksum_f64(&snap0);
    let cE = checksum_f64(&snapE);
    eprintln!("  [wd] checksum start {c0:.4} end {cE:.4} total delta {:.4}", cE - c0);
    let mut deltas: Vec<(String, f64)> = snap0
        .iter()
        .filter_map(|(name, a)| {
            snapE.get(name).map(|b| {
                let d: f64 = a.iter().zip(b.iter()).map(|(x, y)| { let d = (*y - *x) as f64; d * d }).sum::<f64>().sqrt();
                (name.clone(), d)
            })
        })
        .collect();
    deltas.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap());
    for (name, d) in deltas.iter().take(15) {
        eprintln!("  [wd] delta-norm {} = {:.6}", name, d);
    }
    if let Some(s50) = &snap50 {
        eprintln!("  [wd] checksum @step50 {:.4} (delta from start {:.4})", checksum_f64(s50), checksum_f64(s50) - c0);
    }
    // (1) final train accuracy on the fixed set (shuffled eval, same as evaluate_set).
    let (acc, base) = evaluate_set(model, examples, head_index, max_choices, width, dev, use_feature)?;
    eprintln!("  [acc] overfit train acc {acc:.3} (base {base:.3}) over {} fixed examples", examples.len());
    eprintln!("  [gate] final_loss {final_loss:.4} first_loss<0.05 step {:?}", first_loss_below_005);
    Ok((history, final_loss, acc, base, first_loss_below_005))
}

fn snapshot(vm: &VarMap) -> HashMap<String, Vec<f32>> {
    let data = vm.data().lock().unwrap();
    data.iter()
        .map(|(n, t)| {
            let v = t.to_dtype(DType::F32).unwrap().flatten_all().unwrap().to_vec1::<f32>().unwrap();
            (n.clone(), v)
        })
        .collect()
}

fn checksum_f64(snap: &HashMap<String, Vec<f32>>) -> f64 {
    snap.values().map(|v| v.iter().map(|x| *x as f64).sum::<f64>()).sum()
}
