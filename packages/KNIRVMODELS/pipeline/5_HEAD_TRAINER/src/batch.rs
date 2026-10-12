use crate::data::{Example, Label};
use candle_core::{Device, Result, Tensor};
use knirv_head::tokenizer::PAD;
use knirv_head::{DecisionHead, HeadKind};

/// Weight of PAD positions in the plan loss (most positions are PAD).
const PAD_WEIGHT: f32 = 0.1;

pub struct HeadBatch {
    /// `[B, K]` probabilities (choice) or `[B]` values (score).
    pub target: Tensor,
    /// `[B]` 1 where the example carries this label.
    pub mask: Tensor,
}

pub struct Batch {
    pub x: Tensor,
    pub plan: Tensor,
    /// `[B, W]` loss weights; all zero for examples without a plan.
    pub plan_w: Tensor,
    pub heads: Vec<HeadBatch>,
    pub width: usize,
}

pub fn make(examples: &[&Example], heads: &[DecisionHead], width: usize, dev: &Device) -> Result<Batch> {
    let b = examples.len();
    let mut x = Vec::with_capacity(b * width);
    let mut plan = Vec::with_capacity(b * width);
    let mut plan_w = Vec::with_capacity(b * width);
    for e in examples {
        x.extend_from_slice(&e.input);
        match &e.plan {
            Some(p) => {
                plan.extend_from_slice(p);
                plan_w.extend(p.iter().map(|&t| if t == PAD { PAD_WEIGHT } else { 1.0 }));
            }
            None => {
                plan.extend(std::iter::repeat(PAD).take(width));
                plan_w.extend(std::iter::repeat(0f32).take(width));
            }
        }
    }
    let hb = heads
        .iter()
        .map(|h| {
            let k = if h.kind == HeadKind::Choice { h.options.len() } else { 1 };
            let (mut t, mut m) = (Vec::with_capacity(b * k), Vec::with_capacity(b));
            for e in examples {
                match e.labels.get(&h.name) {
                    Some(Label::Choice(p)) if h.kind == HeadKind::Choice && p.len() == k => {
                        t.extend_from_slice(p);
                        m.push(1f32);
                    }
                    Some(Label::Score(v)) if h.kind == HeadKind::Score => {
                        t.push(*v);
                        m.push(1f32);
                    }
                    // Absent, or a label of another kind/arity than this head: unsupervised.
                    _ => {
                        t.extend(std::iter::repeat(0f32).take(k));
                        m.push(0f32);
                    }
                }
            }
            let target = if h.kind == HeadKind::Choice { Tensor::from_vec(t, (b, k), dev)? } else { Tensor::from_vec(t, b, dev)? };
            Ok(HeadBatch { target, mask: Tensor::from_vec(m, b, dev)? })
        })
        .collect::<Result<Vec<_>>>()?;
    Ok(Batch {
        x: Tensor::from_vec(x, (b, width), dev)?,
        plan: Tensor::from_vec(plan, (b, width), dev)?,
        plan_w: Tensor::from_vec(plan_w, (b, width), dev)?,
        heads: hb,
        width,
    })
}
