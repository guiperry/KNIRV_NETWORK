use crate::batch::{self, Batch};
use crate::data::Example;
use candle_core::{DType, Device, Result, Tensor, D};
use candle_nn::{ops, optim::AdamW, Optimizer, ParamsAdamW, VarMap};
use knirv_head::{CycleOut, DecisionHead, HeadKind, Head, Mode};

#[derive(Clone, Debug)]
pub struct TrainOpts {
    pub epochs: usize,
    pub batch: usize,
    pub lr: f64,
    pub warmup_steps: usize,
    pub max_steps: Option<usize>,
    pub seed: u64,
}

/// Weight of the halting loss relative to plan/decision losses.
const HALT_WEIGHT: f64 = 0.1;

/// A score head counts as "already right" for halting within this distance.
const SCORE_RIGHT: f32 = 0.15;

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

impl Xorshift {
    fn unit(&mut self) -> f64 {
        ((self.next() >> 11) as f64 + 0.5) / (1u64 << 53) as f64
    }
    fn normal(&mut self) -> f32 {
        let (u, v) = (self.unit(), self.unit());
        ((-2.0 * u.ln()).sqrt() * (std::f64::consts::TAU * v).cos()) as f32
    }
}

/// candle's CPU RNG cannot be seeded, so initial weights would differ run to run.
/// Overwrites every variable with seeded values: embeddings, positions, summary and
/// state inits N(0, 0.02); linear weights N(0, 1/sqrt(fan_in)); biases 0; norm gains 1.
pub fn seed_init(vm: &VarMap, seed: u64) -> Result<()> {
    let data = vm.data().lock().unwrap();
    let mut names: Vec<_> = data.keys().cloned().collect();
    names.sort();
    for name in names {
        let var = &data[&name];
        let dims = var.dims().to_vec();
        let n = var.elem_count();
        let h = name.bytes().fold(0xcbf29ce484222325u64, |a, b| (a ^ b as u64).wrapping_mul(0x100000001b3));
        let mut rng = Xorshift((seed ^ h) | 1);
        let vals: Vec<f32> = if name.ends_with(".bias") {
            vec![0.0; n]
        } else if name.ends_with("n1.weight") || name.ends_with("n2.weight") || name == "norm.weight" {
            vec![1.0; n]
        } else if dims.len() == 2 && !(name.starts_with("tok.") || name == "pos" || name == "summary") {
            let std = 1.0 / (dims[1] as f32).sqrt();
            (0..n).map(|_| rng.normal() * std).collect()
        } else {
            (0..n).map(|_| rng.normal() * 0.02).collect()
        };
        var.set(&Tensor::from_vec(vals, dims, var.device())?)?;
    }
    Ok(())
}

fn masked_mean(per: &Tensor, mask: &Tensor) -> Result<Tensor> {
    (per * mask)?.sum_all()? / (mask.sum_all()?.to_scalar::<f32>()? as f64 + 1e-6)
}

/// Loss for one batch under deep supervision: the mean over high-level cycles
/// of each cycle's own loss (each cycle uses a one-step gradient).
///
/// * plan: token cross-entropy, PAD positions down-weighted, skipped without a plan
/// * choice heads: cross-entropy against the (possibly soft) target = KL to the teacher
/// * score heads: Brier (squared error on the sigmoid)
/// * halt: the ACT head learns "stop" (class 0) when this cycle is already right,
///   "continue" otherwise: a supervised simplification of the Q-learning target.
///   "Right" = the plan matches (when the row has one) AND every supervised
///   decision head already predicts its label, so the halt loss has a signal on
///   pipeline data that carries no plan (H1). Because every cycle is supervised,
///   stopping early is a trained state.
pub fn loss(model: &Head, b: &Batch, heads: &[DecisionHead]) -> Result<Tensor> {
    let tr = model.forward(&b.x, Mode::Train)?;
    let mut total: Option<Tensor> = None;
    for c in &tr.cycle_outputs {
        let l = cycle_loss(model, b, heads, c)?;
        total = Some(match total {
            Some(t) => (t + l)?,
            None => l,
        });
    }
    total.expect("h_cycles >= 1") / tr.cycle_outputs.len() as f64
}

fn cycle_loss(model: &Head, b: &Batch, heads: &[DecisionHead], c: &CycleOut) -> Result<Tensor> {
    let lp = ops::log_softmax(&c.logits, D::Minus1)?;
    let nll = lp.gather(&b.plan.unsqueeze(2)?, 2)?.squeeze(2)?.neg()?;
    let denom = b.plan_w.sum_all()?.to_scalar::<f32>()? as f64 + 1e-6;
    let mut total = ((&nll * &b.plan_w)?.sum_all()? / denom)?;

    let rows = b.x.dims2()?.0;
    // Per row: does it carry any supervision, and is any supervised head wrong at this cycle?
    let w = b.plan_w.to_vec2::<f32>()?;
    let mut supervised: Vec<bool> = w.iter().map(|r| r.iter().any(|&x| x > 0.0)).collect();
    let mut wrong: Vec<bool> = {
        let pred = c.logits.argmax(D::Minus1)?.to_vec2::<u32>()?;
        let tgt = b.plan.to_vec2::<u32>()?;
        (0..rows).map(|i| (0..pred[i].len()).any(|j| w[i][j] >= 1.0 && pred[i][j] != tgt[i][j])).collect()
    };

    for ((h, hb), out) in heads.iter().zip(&b.heads).zip(model.head_outputs(&c.summary)?) {
        let per = match h.kind {
            HeadKind::Choice => (&hb.target * ops::log_softmax(&out, D::Minus1)?)?.sum(D::Minus1)?.neg()?,
            HeadKind::Score => {
                let d = (ops::sigmoid(&out)?.squeeze(1)? - &hb.target)?;
                (&d * &d)?
            }
        };
        total = (total + masked_mean(&per, &hb.mask)?)?;

        let mask = hb.mask.to_vec1::<f32>()?;
        match h.kind {
            HeadKind::Choice => {
                let pred = out.argmax(D::Minus1)?.to_vec1::<u32>()?;
                let tgt = hb.target.argmax(D::Minus1)?.to_vec1::<u32>()?;
                for i in 0..rows {
                    if mask[i] > 0.5 {
                        supervised[i] = true;
                        wrong[i] |= pred[i] != tgt[i];
                    }
                }
            }
            HeadKind::Score => {
                let pred = ops::sigmoid(&out)?.squeeze(1)?.to_vec1::<f32>()?;
                let tgt = hb.target.to_vec1::<f32>()?;
                for i in 0..rows {
                    if mask[i] > 0.5 {
                        supervised[i] = true;
                        wrong[i] |= (pred[i] - tgt[i]).abs() > SCORE_RIGHT;
                    }
                }
            }
        }
    }

    let y: Vec<u32> = wrong.iter().map(|&x| x as u32).collect();
    let m: Vec<f32> = supervised.iter().map(|&x| if x { 1f32 } else { 0.0 }).collect();
    let dev = b.x.device();
    let y = Tensor::from_vec(y, (rows, 1), dev)?;
    let m = Tensor::from_vec(m, rows, dev)?;
    let hl = ops::log_softmax(&c.halt, D::Minus1)?.gather(&y, 1)?.squeeze(1)?.neg()?;
    total = (total + (masked_mean(&hl, &m)? * HALT_WEIGHT)?)?;
    Ok(total)
}

/// Warmup never exceeds a tenth of the run, so a small dataset still reaches peak LR (H12).
pub fn effective_warmup(warmup: usize, total: usize) -> usize {
    warmup.min((total / 10).max(1))
}

/// Norm gains, biases and embeddings are not weight-decayed (H12).
pub fn decays(name: &str) -> bool {
    !(name.ends_with(".bias")
        || name.ends_with("n1.weight")
        || name.ends_with("n2.weight")
        || name == "norm.weight"
        || name.starts_with("tok.")
        || name == "pos"
        || name == "summary")
}

fn lr_at(step: usize, total: usize, o: &TrainOpts) -> f64 {
    let warmup = effective_warmup(o.warmup_steps, total);
    if step < warmup {
        return o.lr * (step + 1) as f64 / warmup as f64;
    }
    let p = (step - warmup) as f64 / (total.saturating_sub(warmup)).max(1) as f64;
    o.lr * (0.1 + 0.9 * 0.5 * (1.0 + (std::f64::consts::PI * p.min(1.0)).cos()))
}

/// Returns the mean loss of each epoch.
pub fn train(
    model: &Head,
    vm: &VarMap,
    data: &[&Example],
    heads: &[DecisionHead],
    width: usize,
    o: &TrainOpts,
    dev: &Device,
    mut on_epoch: impl FnMut(usize, f32),
) -> Result<Vec<f32>> {
    if data.is_empty() {
        candle_core::bail!("no training examples");
    }
    let steps_per_epoch = data.len().div_ceil(o.batch);
    let total_steps = (steps_per_epoch * o.epochs).min(o.max_steps.unwrap_or(usize::MAX));
    let (decayed, plain) = {
        let data = vm.data().lock().unwrap();
        let (mut d, mut n) = (Vec::new(), Vec::new());
        let mut names: Vec<_> = data.keys().cloned().collect();
        names.sort();
        for name in names {
            let v = data[&name].clone();
            if decays(&name) { d.push(v) } else { n.push(v) }
        }
        (d, n)
    };
    let mut opt = AdamW::new(decayed, ParamsAdamW { lr: o.lr, weight_decay: 0.01, ..Default::default() })?;
    let mut opt_plain = AdamW::new(plain, ParamsAdamW { lr: o.lr, weight_decay: 0.0, ..Default::default() })?;
    let mut rng = Xorshift(o.seed | 1);
    let mut order: Vec<usize> = (0..data.len()).collect();
    let (mut step, mut history) = (0usize, Vec::new());

    'epochs: for epoch in 0..o.epochs {
        rng.shuffle(&mut order);
        let (mut sum, mut n) = (0f32, 0usize);
        for chunk in order.chunks(o.batch) {
            if step >= total_steps {
                break 'epochs;
            }
            let lr = lr_at(step, total_steps, o);
            opt.set_learning_rate(lr);
            opt_plain.set_learning_rate(lr);
            let ex: Vec<&Example> = chunk.iter().map(|&i| data[i]).collect();
            let l = loss(model, &batch::make(&ex, heads, width, dev)?, heads)?;
            let v = l.to_dtype(DType::F32)?.to_scalar::<f32>()?;
            if !v.is_finite() {
                candle_core::bail!("loss became {v} at step {step}; lower --lr (one-step-gradient HEAD diverges above ~3e-3)");
            }
            let grads = l.backward()?;
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn warmup_is_clamped_so_small_runs_reach_peak_lr() {
        assert_eq!(effective_warmup(100, 50), 5);
        assert_eq!(effective_warmup(100, 10_000), 100);
        let o = TrainOpts { epochs: 1, batch: 1, lr: 1e-3, warmup_steps: 100, max_steps: None, seed: 1 };
        assert!((lr_at(4, 50, &o) - 1e-3).abs() < 1e-9, "peak LR is reached inside a 50-step run");
    }

    #[test]
    fn norms_biases_and_embeddings_are_not_decayed() {
        for n in ["blk.0.n1.weight", "blk.0.n2.weight", "norm.weight", "tok.weight", "pos", "summary", "heads.action.bias"] {
            assert!(!decays(n), "{n}");
        }
        for n in ["blk.0.attn.q.weight", "heads.action.weight"] {
            assert!(decays(n), "{n}");
        }
    }
}
