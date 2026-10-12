//! The Hierarchical Reasoning Model.
//!
//! Two coupled recurrent transformer stacks: a fast low-level module (L) and a
//! slow high-level module (H). For each of up to `N` high-level cycles the
//! L-module runs `T` steps conditioned on the current H state, then the
//! H-module updates once from the converged L state. A halting head (ACT)
//! decides at inference how many cycles to spend. Training uses the one-step
//! gradient approximation with deep supervision: in each cycle only the last L
//! step and the H update carry gradient and every cycle's output is supervised
//! (carried state detached), so memory is O(1) in depth and every halting point
//! is trained.

use crate::config::{HeadKind, HeadConfig};
use candle_core::{DType, Device, IndexOp, Result, Tensor, D};
use candle_nn::{
    linear, linear_no_bias, ops, rms_norm, Embedding, Init, Linear, Module, RmsNorm, VarBuilder,
};
use serde::{Deserialize, Serialize};

const EPS: f64 = 1e-6;

struct Block {
    n1: RmsNorm,
    q: Linear,
    k: Linear,
    v: Linear,
    o: Linear,
    n2: RmsNorm,
    gate: Linear,
    up: Linear,
    down: Linear,
    heads: usize,
}

impl Block {
    fn new(cfg: &HeadConfig, vb: VarBuilder) -> Result<Self> {
        let d = cfg.d_model;
        let f = cfg.ffn_mult * d;
        Ok(Self {
            n1: rms_norm(d, EPS, vb.pp("n1"))?,
            q: linear_no_bias(d, d, vb.pp("attn.q"))?,
            k: linear_no_bias(d, d, vb.pp("attn.k"))?,
            v: linear_no_bias(d, d, vb.pp("attn.v"))?,
            o: linear_no_bias(d, d, vb.pp("attn.o"))?,
            n2: rms_norm(d, EPS, vb.pp("n2"))?,
            gate: linear_no_bias(d, f, vb.pp("mlp.gate"))?,
            up: linear_no_bias(d, f, vb.pp("mlp.up"))?,
            down: linear_no_bias(f, d, vb.pp("mlp.down"))?,
            heads: cfg.n_heads,
        })
    }

    fn forward(&self, x: &Tensor) -> Result<Tensor> {
        let (b, s, d) = x.dims3()?;
        let hd = d / self.heads;
        let h = self.n1.forward(x)?;
        let split = |t: Tensor| -> Result<Tensor> {
            t.reshape((b, s, self.heads, hd))?.transpose(1, 2)?.contiguous()
        };
        let (q, k, v) = (split(self.q.forward(&h)?)?, split(self.k.forward(&h)?)?, split(self.v.forward(&h)?)?);
        let att = (q.matmul(&k.transpose(2, 3)?.contiguous()?)? / (hd as f64).sqrt())?;
        let att = ops::softmax_last_dim(&att)?;
        let ctx = att.matmul(&v)?.transpose(1, 2)?.contiguous()?.reshape((b, s, d))?;
        let x = (x + self.o.forward(&ctx)?)?;
        let h = self.n2.forward(&x)?;
        let m = (ops::silu(&self.gate.forward(&h)?)? * self.up.forward(&h)?)?;
        x + self.down.forward(&m)?
    }
}

struct Stack(Vec<Block>);

impl Stack {
    fn new(cfg: &HeadConfig, n: usize, vb: VarBuilder) -> Result<Self> {
        (0..n).map(|i| Block::new(cfg, vb.pp(i))).collect::<Result<Vec<_>>>().map(Self)
    }
    fn forward(&self, x: &Tensor) -> Result<Tensor> {
        self.0.iter().try_fold(x.clone(), |x, b| b.forward(&x))
    }
}

enum HeadLayer {
    Choice(Linear),
    Score(Linear),
}

pub struct Head {
    pub cfg: HeadConfig,
    emb: Embedding,
    emb_w: Tensor,
    pos: Tensor,
    summary: Tensor,
    z_l0: Tensor,
    z_h0: Tensor,
    l: Stack,
    h: Stack,
    norm: RmsNorm,
    halt: Linear,
    heads: Vec<HeadLayer>,
    device: Device,
}

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum Mode {
    /// Adaptive depth via the halting head (batch of 1 recommended).
    Infer,
    /// Fixed `h_cycles`; deep supervision with a one-step gradient per cycle.
    Train,
}

/// Output of one high-level cycle.
#[derive(Clone)]
pub struct CycleOut {
    pub logits: Tensor,
    pub summary: Tensor,
    pub halt: Tensor,
}

pub struct Trace {
    /// `[B, S, vocab]` logits over the input positions (summary slots excluded).
    pub logits: Tensor,
    /// `[B, D]` summary vector feeding halting and decision heads.
    pub summary: Tensor,
    /// `[B, 2]` (q_halt, q_continue) after the last executed cycle.
    pub halt: Tensor,
    pub cycles: usize,
    pub l_act: Vec<f32>,
    pub h_act: Vec<f32>,
    /// `Mode::Train` only: every cycle's output, each with its own one-step gradient.
    pub cycle_outputs: Vec<CycleOut>,
}

impl Head {
    pub fn new(cfg: HeadConfig, vb: VarBuilder) -> Result<Self> {
        cfg.validate().map_err(candle_core::Error::msg)?;
        let d = cfg.d_model;
        let small = Init::Randn { mean: 0.0, stdev: 0.02 };
        let emb_w = vb.pp("tok").get_with_hints((cfg.vocab_size, d), "weight", small)?;
        let heads = cfg
            .decision_heads
            .iter()
            .map(|h| {
                let hvb = vb.pp(format!("head.{}", h.name));
                Ok(match h.kind {
                    HeadKind::Choice => HeadLayer::Choice(linear(d, h.options.len(), hvb)?),
                    HeadKind::Score => HeadLayer::Score(linear(d, 1, hvb)?),
                })
            })
            .collect::<Result<Vec<_>>>()?;
        Ok(Self {
            emb: Embedding::new(emb_w.clone(), d),
            emb_w,
            pos: vb.get_with_hints((cfg.max_seq_len, d), "pos", small)?,
            summary: vb.get_with_hints((cfg.summary_slots, d), "summary", small)?,
            z_l0: vb.get_with_hints(d, "z_l_init", small)?,
            z_h0: vb.get_with_hints(d, "z_h_init", small)?,
            l: Stack::new(&cfg, cfg.l_layers, vb.pp("l"))?,
            h: Stack::new(&cfg, cfg.h_layers, vb.pp("h"))?,
            norm: rms_norm(d, EPS, vb.pp("norm"))?,
            halt: linear(d, 2, vb.pp("halt"))?,
            heads,
            device: vb.device().clone(),
            cfg,
        })
    }

    /// Load from a safetensors buffer, verifying the tensor set matches the config.
    pub fn from_safetensors(cfg: HeadConfig, bytes: &[u8], device: &Device) -> Result<Self> {
        Self::from_safetensors_adapted(cfg, bytes, device, None).map(|(m, _)| m)
    }

    /// As [`Self::from_safetensors`], merging a KNIRVULORA adapter into the weights first.
    pub fn from_safetensors_adapted(
        cfg: HeadConfig,
        bytes: &[u8],
        device: &Device,
        adapter: Option<&crate::adapter::Adapter>,
    ) -> Result<(Self, Option<crate::adapter::Applied>)> {
        let mut tensors = candle_core::safetensors::load_buffer(bytes, device)?;
        let total: usize = tensors.values().map(|t| t.elem_count()).sum();
        let want = cfg.param_count();
        if total != want {
            candle_core::bail!("weights have {total} parameters, config implies {want}");
        }
        if tensors.values().any(|t| t.dtype() != DType::F32) {
            candle_core::bail!("knirv-head/v1 weights must be f32");
        }
        let applied = match adapter {
            Some(ad) => Some(crate::adapter::merge(&cfg, &mut tensors, ad)?),
            None => None,
        };
        Ok((Self::new(cfg, VarBuilder::from_tensors(tensors, DType::F32, device))?, applied))
    }

    pub fn param_count(&self) -> usize {
        self.cfg.param_count()
    }

    pub fn max_input_len(&self) -> usize {
        self.cfg.max_seq_len - self.cfg.summary_slots
    }

    pub fn forward(&self, tokens: &Tensor, mode: Mode) -> Result<Trace> {
        let (b, s) = tokens.dims2()?;
        let slots = self.cfg.summary_slots;
        if s == 0 || s > self.max_input_len() {
            candle_core::bail!("input length {s} outside 1..={}", self.max_input_len());
        }
        let d = self.cfg.d_model;
        let total = slots + s;
        let x = self.emb.forward(tokens)?;
        let summ = self.summary.unsqueeze(0)?.broadcast_as((b, slots, d))?;
        let x = Tensor::cat(&[&summ, &x], 1)?;
        // Embeddings are initialised small (0.02) so the tied output head starts calibrated;
        // scaling the inputs by sqrt(d) (as in the original Transformer) keeps them from
        // being drowned by block outputs in the pre-norm residual stream.
        let scale = (d as f64).sqrt();
        let x = (x.broadcast_add(&self.pos.narrow(0, 0, total)?.unsqueeze(0)?)? * scale)?;

        let init = |v: &Tensor| (v * scale)?.reshape((1, 1, d))?.broadcast_as((b, total, d))?.contiguous();
        let (mut z_l, mut z_h) = (init(&self.z_l0)?, init(&self.z_h0)?);

        let (n, t) = (self.cfg.h_cycles, self.cfg.l_steps);
        let train = mode == Mode::Train;
        let (mut cycles, mut l_act, mut h_act) = (0usize, Vec::new(), Vec::new());
        let mut cycle_outputs = Vec::new();
        let mut last = None;

        for i in 0..n {
            if train && i > 0 {
                // Deep supervision: each cycle is trained on its own with the carried state detached.
                z_l = z_l.detach();
                z_h = z_h.detach();
            }
            for j in 0..t {
                // One-step gradient: only the final L step of a cycle is tracked.
                let tracked = !train || j + 1 == t;
                let inp = ((&z_l + &z_h)? + &x)?;
                z_l = self.l.forward(&if tracked { inp } else { inp.detach() })?;
                if !tracked {
                    z_l = z_l.detach();
                }
            }
            z_h = self.h.forward(&(&z_h + &z_l)?)?;
            cycles = i + 1;

            let out = self.cycle_output(&z_h, slots, s)?;
            if train {
                cycle_outputs.push(out);
            } else {
                l_act.push(z_l.abs()?.mean_all()?.to_scalar::<f32>()?);
                h_act.push(z_h.abs()?.mean_all()?.to_scalar::<f32>()?);
                let halt_now = cycles >= self.cfg.min_cycles && self.should_halt(&out.halt)?;
                last = Some(out);
                if halt_now {
                    break;
                }
            }
        }

        let CycleOut { logits, summary, halt } = match (train, last) {
            (false, Some(o)) => o,
            _ => cycle_outputs.last().cloned().expect("h_cycles >= 1"),
        };
        Ok(Trace { logits, summary, halt, cycles, l_act, h_act, cycle_outputs })
    }

    fn cycle_output(&self, z_h: &Tensor, slots: usize, s: usize) -> Result<CycleOut> {
        let out = self.norm.forward(z_h)?;
        let summary = out.i((.., 0, ..))?;
        let logits = out.narrow(1, slots, s)?.broadcast_matmul(&self.emb_w.t()?)?;
        let halt = self.halt.forward(&summary)?;
        Ok(CycleOut { logits, summary, halt })
    }

    fn should_halt(&self, halt: &Tensor) -> Result<bool> {
        let q = halt.to_vec2::<f32>()?;
        Ok(q.iter().all(|r| r[0] > r[1]))
    }

    /// Raw head outputs for a `[B, D]` summary: logits for `choice`, pre-sigmoid for `score`.
    pub fn head_outputs(&self, summary: &Tensor) -> Result<Vec<Tensor>> {
        self.heads
            .iter()
            .map(|h| match h {
                HeadLayer::Choice(l) | HeadLayer::Score(l) => l.forward(summary),
            })
            .collect()
    }

    pub fn device(&self) -> &Device {
        &self.device
    }

    /// Full inference on one token sequence.
    pub fn reason(&self, tokens: &[u32]) -> Result<Reasoning> {
        if let Some(bad) = tokens.iter().find(|&&t| t as usize >= self.cfg.vocab_size) {
            candle_core::bail!("token id {bad} outside vocab of {}", self.cfg.vocab_size);
        }
        let input = Tensor::from_slice(tokens, (1, tokens.len()), &self.device)?;
        let tr = self.forward(&input, Mode::Infer)?;

        let probs = ops::softmax(&tr.logits, D::Minus1)?.i(0)?;
        let plan_tokens = tr.logits.argmax(D::Minus1)?.i(0)?.to_vec1::<u32>()?;
        let top = probs.max(D::Minus1)?.to_vec1::<f32>()?;
        let confidence = top.iter().sum::<f32>() / top.len() as f32;

        let q = tr.halt.i(0)?.to_vec1::<f32>()?;
        let halt_prob = ops::softmax(&tr.halt, D::Minus1)?.i((0, 0))?.to_scalar::<f32>()?;
        let _ = q;

        let raw = self.head_outputs(&tr.summary)?;
        let mut decisions = Vec::with_capacity(raw.len());
        for (spec, out) in self.cfg.decision_heads.iter().zip(raw) {
            decisions.push(match spec.kind {
                HeadKind::Choice => {
                    let p = ops::softmax(&out, D::Minus1)?.i(0)?.to_vec1::<f32>()?;
                    let (arg, &max) = p
                        .iter()
                        .enumerate()
                        .max_by(|a, b| a.1.total_cmp(b.1))
                        .expect("validated: >= 2 options");
                    let abstain = max < self.cfg.tau;
                    Decision {
                        name: spec.name.clone(),
                        kind: HeadKind::Choice,
                        options: spec.options.clone(),
                        probabilities: p.clone(),
                        selected: if abstain { None } else { Some(spec.options[arg].clone()) },
                        value: None,
                        confidence: Some(max),
                        abstain,
                    }
                }
                HeadKind::Score => {
                    let v = ops::sigmoid(&out)?.i((0, 0))?.to_scalar::<f32>()?;
                    Decision {
                        name: spec.name.clone(),
                        kind: HeadKind::Score,
                        options: Vec::new(),
                        probabilities: Vec::new(),
                        selected: None,
                        value: Some(v),
                        // Rule 4: yes/no style confidence is max(P, 1-P).
                        confidence: Some(v.max(1.0 - v)),
                        abstain: false,
                    }
                }
            });
        }

        Ok(Reasoning {
            plan_tokens,
            halting_steps: tr.cycles,
            max_steps: self.cfg.h_cycles,
            halt_probability: halt_prob,
            confidence,
            decisions,
            l_module_activations: tr.l_act,
            h_module_activations: tr.h_act,
            model_version: self.cfg.model_version.clone(),
            dataset_manifest_hash: self.cfg.dataset_manifest_hash.clone(),
            policy_bundle_hash: self.cfg.policy_bundle_hash.clone(),
        })
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct Decision {
    pub name: String,
    pub kind: HeadKind,
    pub options: Vec<String>,
    /// Sums to 1 for `choice`; empty for `score`.
    pub probabilities: Vec<f32>,
    /// `None` when the head abstained (max(p) < tau) or for `score`.
    pub selected: Option<String>,
    /// Continuous value in [0, 1] for `score`; never rounded here.
    pub value: Option<f32>,
    pub confidence: Option<f32>,
    pub abstain: bool,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct Reasoning {
    pub plan_tokens: Vec<u32>,
    pub halting_steps: usize,
    pub max_steps: usize,
    pub halt_probability: f32,
    /// Mean top-1 probability over the plan positions.
    pub confidence: f32,
    pub decisions: Vec<Decision>,
    pub l_module_activations: Vec<f32>,
    pub h_module_activations: Vec<f32>,
    pub model_version: String,
    pub dataset_manifest_hash: String,
    pub policy_bundle_hash: String,
}
