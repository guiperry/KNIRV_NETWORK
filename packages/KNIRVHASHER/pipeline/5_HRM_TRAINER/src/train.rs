use crate::batch::{self, Batch};
use crate::data::Example;
use candle_core::{DType, Device, Result, Tensor, D};
use candle_nn::{ops, optim::AdamW, Optimizer, ParamsAdamW, VarMap};
use knirv_cortex_wasm::{CycleOut, DecisionHead, HeadKind, Hrm, Mode};

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
/// * halt: the ACT head learns "stop" (class 0) when this cycle's plan is already
///   right, "continue" otherwise: a supervised simplification of the Q-learning
///   target. Because every cycle is supervised, stopping early is a trained state.
pub fn loss(model: &Hrm, b: &Batch, heads: &[DecisionHead]) -> Result<Tensor> {
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

fn cycle_loss(model: &Hrm, b: &Batch, heads: &[DecisionHead], c: &CycleOut) -> Result<Tensor> {
    let lp = ops::log_softmax(&c.logits, D::Minus1)?;
    let nll = lp.gather(&b.plan.unsqueeze(2)?, 2)?.squeeze(2)?.neg()?;
    let denom = b.plan_w.sum_all()?.to_scalar::<f32>()? as f64 + 1e-6;
    let mut total = ((&nll * &b.plan_w)?.sum_all()? / denom)?;

    for ((h, hb), out) in heads.iter().zip(&b.heads).zip(model.head_outputs(&c.summary)?) {
        let per = match h.kind {
            HeadKind::Choice => (&hb.target * ops::log_softmax(&out, D::Minus1)?)?.sum(D::Minus1)?.neg()?,
            HeadKind::Score => {
                let d = (ops::sigmoid(&out)?.squeeze(1)? - &hb.target)?;
                (&d * &d)?
            }
        };
        total = (total + masked_mean(&per, &hb.mask)?)?;
    }

    let pred = c.logits.argmax(D::Minus1)?.to_vec2::<u32>()?;
    let tgt = b.plan.to_vec2::<u32>()?;
    let w = b.plan_w.to_vec2::<f32>()?;
    let (mut y, mut m) = (Vec::new(), Vec::new());
    for i in 0..pred.len() {
        let has_plan = w[i].iter().any(|&x| x > 0.0);
        let wrong = (0..pred[i].len()).any(|j| w[i][j] >= 1.0 && pred[i][j] != tgt[i][j]);
        y.push(wrong as u32);
        m.push(if has_plan { 1f32 } else { 0.0 });
    }
    let dev = b.x.device();
    let y = Tensor::from_vec(y, (pred.len(), 1), dev)?;
    let m = Tensor::from_vec(m, pred.len(), dev)?;
    let hl = ops::log_softmax(&c.halt, D::Minus1)?.gather(&y, 1)?.squeeze(1)?.neg()?;
    total = (total + (masked_mean(&hl, &m)? * HALT_WEIGHT)?)?;
    Ok(total)
}

fn lr_at(step: usize, total: usize, o: &TrainOpts) -> f64 {
    if step < o.warmup_steps {
        return o.lr * (step + 1) as f64 / o.warmup_steps as f64;
    }
    let p = (step - o.warmup_steps) as f64 / (total.saturating_sub(o.warmup_steps)).max(1) as f64;
    o.lr * (0.1 + 0.9 * 0.5 * (1.0 + (std::f64::consts::PI * p.min(1.0)).cos()))
}

/// Returns the mean loss of each epoch.
pub fn train(
    model: &Hrm,
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
    let mut opt = AdamW::new(vm.all_vars(), ParamsAdamW { lr: o.lr, weight_decay: 0.01, ..Default::default() })?;
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
            opt.set_learning_rate(lr_at(step, total_steps, o));
            let ex: Vec<&Example> = chunk.iter().map(|&i| data[i]).collect();
            let l = loss(model, &batch::make(&ex, heads, width, dev)?, heads)?;
            let v = l.to_dtype(DType::F32)?.to_scalar::<f32>()?;
            if !v.is_finite() {
                candle_core::bail!("loss became {v} at step {step}; lower --lr (one-step-gradient HRM diverges above ~3e-3)");
            }
            opt.backward_step(&l)?;
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
