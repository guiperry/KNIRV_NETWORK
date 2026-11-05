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
use candle_core::{DType, Device, Result, Tensor, D};
use candle_nn::{ops, optim::AdamW, Optimizer, ParamsAdamW, VarMap};
use knirv_heart_decision::tokenizer::encode_set_prompt;
use knirv_heart_decision::{ChoiceLayout, Head};

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

struct Xorshift(u64);
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
fn encode_shuffled(ex: &SetExample, max_choices: usize, width: usize, rng: &mut Xorshift) -> Result<(Vec<u32>, Vec<bool>, usize)> {
    let mut order: Vec<usize> = (0..ex.candidates.len()).collect();
    rng.shuffle(&mut order);
    let target = order.iter().position(|&i| i == ex.target).expect("target present");
    let cand: Vec<String> = order.iter().map(|&i| ex.candidates[i].clone()).collect();
    let tokens = encode_set_prompt(&ex.question, &ex.state, &cand, width);
    let n = cand.len();
    // One question per forward (variable N): the prompt carries exactly `n` candidates and
    // they are all valid. (Batched training with padded candidates would extend this mask.)
    let valid = vec![true; n];
    Ok((tokens, valid, target))
}

/// Cross-entropy against the (remapped) target, averaged over the high-level cycles'
/// pre-softmax set logits. Invalid (padding) rows carry `-inf` logits and contribute nothing.
fn set_loss(model: &Head, tokens: &Tensor, head: usize, valid: &[bool], target: usize) -> Result<Tensor> {
    let layout = ChoiceLayout::from_tokens(&tokens.to_vec2::<u32>()?.remove(0))?;
    let cycles = model.set_cycle_logits(tokens, &layout, head, valid)?;
    let mut total: Option<Tensor> = None;
    for logits in &cycles {
        let ls = ops::log_softmax(logits, D::Minus1)?;
        let nll = ls.narrow(1, target, 1)?.neg()?; // [1, 1]
        total = Some(match total {
            Some(x) => (x + nll)?,
            None => nll,
        });
    }
    total.expect("h_cycles >= 1") / cycles.len() as f64
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
    let mut rng = Xorshift(o.seed | 1);
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
            let (tokens, valid, target) = encode_shuffled(&examples[idx], max_choices, width, &mut rng)?;
            let input = Tensor::from_slice(&tokens, (1, tokens.len()), dev)?;
            let l = set_loss(model, &input, head_index, &valid, target)?;
            let v = l.to_dtype(DType::F32)?.sum_all()?.to_scalar::<f32>()?;
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

fn argmax(v: &[f32]) -> usize {
    v.iter().enumerate().max_by(|a, b| a.1.total_cmp(b.1)).map(|x| x.0).unwrap_or(0)
}

/// Accuracy on `examples` (canonical order, no shuffle) and the random base rate
/// `mean(1/N)`. The kill gate is cleared when accuracy is clearly above the base rate.
pub fn evaluate_set(model: &Head, examples: &[SetExample], head_index: usize, max_choices: usize, width: usize, dev: &Device) -> Result<(f32, f32)> {
    if examples.is_empty() {
        candle_core::bail!("no set examples to evaluate");
    }
    let mut correct = 0usize;
    let mut total = 0usize;
    let mut inv = 0f32;
    for ex in examples {
        let n = ex.candidates.len();
        let tokens = encode_set_prompt(&ex.question, &ex.state, &ex.candidates, width);
        let layout = ChoiceLayout::from_tokens(&tokens)?;
        let valid = vec![true; n];
        let t = Tensor::from_slice(&tokens, (1, tokens.len()), dev)?;
        let layout = ChoiceLayout::from_tokens(&tokens)?;
        let cycles = model.set_cycle_logits(&t, &layout, head_index, &valid)?;
        let logits = cycles.last().expect("h_cycles >= 1");
        let p = ops::softmax(logits, D::Minus1)?.to_vec2::<f32>()?.remove(0);
        let pred = argmax(&p);
        if pred == ex.target {
            correct += 1;
        }
        total += 1;
        inv += 1.0 / n as f32;
    }
    Ok((correct as f32 / total as f32, inv / total as f32))
}
