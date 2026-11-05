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

use crate::config::{HeadConfig, HeadKind, SetKind};
use crate::layout::{assert_additive, ChoiceLayout};
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

    /// `bias` is an optional additive `[S, S]` attention bias (0 allowed, -inf forbidden);
    /// `None` is exactly the v1 computation.
    fn forward(&self, x: &Tensor, bias: Option<&Tensor>) -> Result<Tensor> {
        let (b, s, d) = x.dims3()?;
        let hd = d / self.heads;
        let h = self.n1.forward(x)?;
        let split = |t: Tensor| -> Result<Tensor> {
            t.reshape((b, s, self.heads, hd))?.transpose(1, 2)?.contiguous()
        };
        let (q, k, v) = (split(self.q.forward(&h)?)?, split(self.k.forward(&h)?)?, split(self.v.forward(&h)?)?);
        let att = (q.matmul(&k.transpose(2, 3)?.contiguous()?)? / (hd as f64).sqrt())?;
        let att = match bias {
            Some(m) => att.broadcast_add(m)?,
            None => att,
        };
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
    fn forward(&self, x: &Tensor, bias: Option<&Tensor>) -> Result<Tensor> {
        self.0.iter().try_fold(x.clone(), |x, b| b.forward(&x, bias))
    }
}

enum HeadLayer {
    Choice(Linear),
    Score(Linear),
}

/// The set scorer's weights (format v2 only). None of them has a per-slot dimension: the
/// task and role embeddings are `d` vectors, the two blocks and `W` are `d`-sized, so a
/// request of any size up to a head's `max_choices` uses the same weights.
struct SetScorer {
    task: Vec<Tensor>, // one [d] per set head
    role: Tensor,      // [2, d]: row 0 answer, row 1 choice
    attn: Stack,       // two blocks, no position input
    norm: RmsNorm,
    w: Tensor, // [d, d]
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
    set: Option<SetScorer>,
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
    /// The post-norm hidden state `[B, slots+s, D]` of this cycle: pools of it feed the
    /// set scorer during training (Task 4.4). Present in both modes; in `Infer` it is the
    /// last cycle's hidden, used for halting diagnostics.
    pub hidden: Tensor,
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
        let set = if cfg.choice_set_heads.is_empty() {
            None
        } else {
            let task = (0..cfg.choice_set_heads.len())
                .map(|k| vb.get_with_hints(d, &format!("set.task.{k}"), small))
                .collect::<Result<Vec<_>>>()?;
            Some(SetScorer {
                task,
                role: vb.get_with_hints((2, d), "set.role", small)?,
                attn: Stack::new(&cfg, 2, vb.pp("set.attn"))?,
                norm: rms_norm(d, EPS, vb.pp("set.norm"))?,
                w: vb.get_with_hints((d, d), "set.w", small)?,
            })
        };
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
            set,
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
            candle_core::bail!("knirv-head weights must be f32");
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

    /// The fixed-head forward: no layout, v1 positions, no mask. Identical to
    /// `forward_layout(tokens, mode, None)`.
    pub fn forward(&self, tokens: &Tensor, mode: Mode) -> Result<Trace> {
        self.forward_layout(tokens, mode, None)
    }

    /// Forward with an optional choice-set layout (Task 4.2): positions are gathered by
    /// `ChoiceLayout::pos_ids` and one additive bias, built once, is used by every block in
    /// every L step and H update of every cycle. `None` is the v1 computation, bit for bit.
    pub fn forward_layout(&self, tokens: &Tensor, mode: Mode, layout: Option<&ChoiceLayout>) -> Result<Trace> {
        self.run(tokens, mode, layout, None, false).map(|(t, _)| t)
    }

    /// As [`Self::forward`] with a caller-built additive `[S, S]` bias over `slots + len`
    /// positions and the v1 positions. The bias must be additive (0 or -inf); a 1/0 mask is an
    /// error, not a silent leak.
    pub fn forward_with_bias(&self, tokens: &Tensor, mode: Mode, bias: &Tensor) -> Result<Trace> {
        assert_additive(bias)?;
        self.run(tokens, mode, None, Some(bias), false).map(|(t, _)| t)
    }

    /// One pooled vector per choice, `[B, n_choices, D]`: the mean of the choice's final hidden
    /// states. It runs the full recurrent forward at fixed depth (`h_cycles`, no halting), because
    /// the halting decision reads the summary slot, which sees every choice, and an adaptive depth
    /// would let one choice change another's embedding.
    pub fn embed_choices(&self, tokens: &Tensor, layout: &ChoiceLayout) -> Result<Tensor> {
        let (_, hidden) = self.run(tokens, Mode::Train, Some(layout), None, true)?;
        let hidden = hidden.expect("run returns the hidden state when asked");
        let slots = self.cfg.summary_slots;
        let pooled = layout
            .choice_spans
            .iter()
            .map(|&(start, len)| hidden.narrow(1, slots + start, len)?.mean(1))
            .collect::<Result<Vec<_>>>()?;
        Tensor::stack(&pooled, 1)
    }

    fn run(&self, tokens: &Tensor, mode: Mode, layout: Option<&ChoiceLayout>, bias_in: Option<&Tensor>, want_hidden: bool) -> Result<(Trace, Option<Tensor>)> {
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
        let pos = match layout {
            None => self.pos.narrow(0, 0, total)?,
            Some(l) => {
                l.validate(s)?;
                let ids = Tensor::from_vec(l.pos_ids(slots, s), total, &self.device)?;
                self.pos.index_select(&ids, 0)?
            }
        };
        let x = (x.broadcast_add(&pos.unsqueeze(0)?)? * scale)?;
        // Built once per forward and shared by every block of every step.
        let built;
        let bias: Option<&Tensor> = match (layout, bias_in) {
            (Some(l), _) => {
                built = l.bias(slots, s, &self.device)?;
                assert_additive(&built)?;
                Some(&built)
            }
            (None, Some(m)) => {
                if m.dims2()? != (total, total) {
                    candle_core::bail!("attention bias is {:?}, input needs ({total}, {total})", m.dims2()?);
                }
                Some(m)
            }
            (None, None) => None,
        };

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
                z_l = self.l.forward(&if tracked { inp } else { inp.detach() }, bias)?;
                if !tracked {
                    z_l = z_l.detach();
                }
            }
            z_h = self.h.forward(&(&z_h + &z_l)?, bias)?;
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

        let CycleOut { logits, summary, halt, .. } = match (train, last) {
            (false, Some(o)) => o,
            _ => cycle_outputs.last().cloned().expect("h_cycles >= 1"),
        };
        let hidden = if want_hidden { Some(self.norm.forward(&z_h)?) } else { None };
        Ok((Trace { logits, summary, halt, cycles, l_act, h_act, cycle_outputs }, hidden))
    }

    fn cycle_output(&self, z_h: &Tensor, slots: usize, s: usize) -> Result<CycleOut> {
        let out = self.norm.forward(z_h)?;
        let summary = out.i((.., 0, ..))?;
        let logits = out.narrow(1, slots, s)?.broadcast_matmul(&self.emb_w.t()?)?;
        let halt = self.halt.forward(&summary)?;
        Ok(CycleOut { logits, summary, halt, hidden: out })
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

    /// The set scorer's pooled inputs for one prompt: `c` is `[N, D]` (per choice: the mean of
    /// its span's final hidden states plus the hidden state at its `OPT_END`) and `a` is `[D]`
    /// (the summary slot). Runs the full recurrent forward at fixed depth under the layout.
    pub fn set_inputs(&self, tokens: &Tensor, layout: &ChoiceLayout) -> Result<(Tensor, Tensor)> {
        let (_, hidden) = self.run(tokens, Mode::Train, Some(layout), None, true)?;
        let hidden = hidden.expect("run returns the hidden state when asked").i(0)?; // [total, D]
        let slots = self.cfg.summary_slots;
        let c = layout
            .choice_spans
            .iter()
            .map(|&(start, len)| {
                let pooled = hidden.narrow(0, slots + start, len)?.mean(0)?;
                pooled + hidden.i(slots + start + len - 1)?
            })
            .collect::<Result<Vec<_>>>()?;
        Ok((Tensor::stack(&c, 0)?, hidden.i(0)?))
    }

    /// Pre-softmax set scores over the `valid` candidates, index-aligned with the rows of `c`
    /// (`[B, M, D]`); `a` is `[B, D]`. Padded rows (where `valid` is false) get `-inf` and are
    /// never a key. The same kernel backs [`Self::set_probabilities`] (inference) and the
    /// training-time logits ([`Self::set_logits_from_hidden`], [`Self::set_cycle_logits`]).
    fn set_scores(&self, c: &Tensor, a: &Tensor, head: usize, valid: &[bool]) -> Result<Tensor> {
        let set = self.set.as_ref().ok_or_else(|| candle_core::Error::msg("this model has no choice-set heads (format knirv-heart/v1)"))?;
        let (b, m, _d) = c.dims3()?;
        if a.dims2()?.0 != b {
            candle_core::bail!("set_scores: answer batch {} != candidate batch {b}", a.dims2()?.0);
        }
        if head >= set.task.len() || valid.len() != m || !valid.iter().any(|&v| v) {
            candle_core::bail!("set scoring needs a known head and at least one valid candidate of {m}");
        }
        let d = self.cfg.d_model;
        let task = set.task[head].reshape((1, 1, d))?;
        let role_ans = set.role.i(0)?.reshape((1, 1, d))?;
        let role_ch = set.role.i(1)?.reshape((1, 1, d))?;
        let answer = (a.unsqueeze(1)? + &task)?.broadcast_add(&role_ans)?; // [B, 1, D]
        let choices = (c.broadcast_add(&task)?).broadcast_add(&role_ch)?; // [B, M, D]
        let x = Tensor::cat(&[&choices, &answer], 1)?.contiguous()?; // [B, M+1, D]
        // Key bias: padded candidates are never keys; the answer row always is.
        let n = m + 1;
        let mut bias = vec![0f32; b * n * n];
        for (bi, ok) in valid.iter().enumerate() {
            if !ok {
                for r in 0..n {
                    bias[bi * n * n + r * n + bi] = f32::NEG_INFINITY;
                }
            }
        }
        let bias = Tensor::from_vec(bias, (b, n, n), &self.device)?;
        let x = set.attn.forward(&x, Some(&bias))?;
        let x = set.norm.forward(&x)?; // [B, M+1, D]
        let a_out = x.narrow(1, m, 1)?.contiguous()?.reshape((b, d, 1))?; // [B, D, 1]
        // scores_i = c'_i . (W a): batched matmul with W broadcast across the batch.
        let w = set.w.broadcast_as((b, d, d))?;
        let wa = a_out.transpose(1, 2)?.matmul(&w)?; // [B, 1, D]
        let scores = x.narrow(1, 0, m)?.matmul(&wa.transpose(1, 2)?)?.squeeze(D::Minus1)?; // [B, M]
        let mask = Tensor::from_vec(
            valid.iter().map(|&v| if v { 0.0f32 } else { f32::NEG_INFINITY }).collect::<Vec<_>>(),
            (1, m),
            &self.device,
        )?;
        scores.broadcast_add(&mask)
    }

    /// Probabilities over the `valid` candidates of one set question, index-aligned with the
    /// rows of `c`; a row marked invalid (padding) gets exactly zero mass and is never a key.
    ///
    /// `a = a + task_k + role_answer`, every `c_i = c_i + task_k + role_choice`, then two
    /// set-attention blocks over `[c_1 .. c_M, a]` with no position input, and
    /// `s_i = c'_i . W a'`, softmaxed over the valid rows. Nothing in it depends on the order
    /// of the rows or on how many there are.
    pub fn set_probabilities(&self, c: &Tensor, a: &Tensor, head: usize, valid: &[bool]) -> Result<Vec<f32>> {
        let scores = self.set_scores(&c.unsqueeze(0)?, &a.unsqueeze(0)?, head, valid)?.to_vec2::<f32>()?.remove(0);
        let max = scores.iter().zip(valid).filter(|(_, &v)| v).map(|(s, _)| *s).fold(f32::NEG_INFINITY, f32::max);
        let exps: Vec<f32> = scores.iter().zip(valid).map(|(s, &v)| if v { (s - max).exp() } else { 0.0 }).collect();
        let sum: f32 = exps.iter().sum();
        Ok(exps.iter().map(|e| e / sum).collect())
    }

    /// Pre-softmax set logits `[B, M]` for one set question from a per-cycle hidden state
    /// `[B, slots+s, D]`. Padded candidates (`valid` false) get `-inf` so softmax ignores them.
    /// Used by the trainer to compute a cross-entropy target per cycle (Task 4.4).
    pub fn set_logits_from_hidden(&self, hidden: &Tensor, layout: &ChoiceLayout, head: usize, valid: &[bool]) -> Result<Tensor> {
        let slots = self.cfg.summary_slots;
        let c: Vec<Tensor> = layout
            .choice_spans
            .iter()
            .map(|&(start, len)| {
                let pooled = hidden.narrow(1, slots + start, len)?.mean(1)?;
                let last = hidden.narrow(1, slots + start + len - 1, 1)?.squeeze(1)?;
                (pooled + last)
            })
            .collect::<Result<Vec<_>>>()?;
        let c = Tensor::stack(&c, 1)?; // [B, M, D]
        let a = hidden.narrow(1, 0, 1)?.squeeze(1)?; // [B, D]
        self.set_scores(&c, &a, head, valid)
    }

    /// Per-cycle set logits for training: runs the recurrent forward at fixed depth under
    /// `layout` and returns one `[B, M]` pre-softmax tensor per high-level cycle. Each cycle's
    /// carried state is detached (see [`Self::run`]), so summing the cycles' losses trains each
    /// on its own one-step gradient (the same deep-supervision scheme as the decision heads).
    pub fn set_cycle_logits(&self, tokens: &Tensor, layout: &ChoiceLayout, head: usize, valid: &[bool]) -> Result<Vec<Tensor>> {
        let tr = self.forward_layout(tokens, Mode::Train, Some(layout))?;
        tr.cycle_outputs
            .iter()
            .map(|c| self.set_logits_from_hidden(&c.hidden, layout, head, valid))
            .collect()
    }

    /// Answers choice-set questions (format `knirv-heart/v2`). Each question is its own forward
    /// under its own layout; its probabilities are index-aligned with the request's candidates.
    /// A `score` head's candidates are ordered levels and its value is `sum p_i i / (N-1)`; a
    /// `null` head's candidates are always `yes` and `no`. The run-level fields (activations,
    /// halting) come from the first question's forward; scoring always uses the full depth.
    pub fn reason_sets(&self, questions: &[SetQuestion]) -> Result<Reasoning> {
        if self.set.is_none() {
            candle_core::bail!("this model has no choice-set heads (format knirv-heart/v1)");
        }
        if questions.is_empty() {
            candle_core::bail!("reason_sets needs at least one question");
        }
        let width = self.max_input_len();
        let mut decisions = Vec::new();
        let (mut first_halt, mut first_acts) = (None, None);
        for q in questions {
            let (k, spec) = self
                .cfg
                .choice_set_heads
                .iter()
                .enumerate()
                .find(|(_, h)| h.name == q.name)
                .ok_or_else(|| candle_core::Error::msg(format!("no choice-set head named {:?}", q.name)))?;
            let choices: Vec<SetChoice> = match spec.kind {
                SetKind::Null => ["yes", "no"].iter().map(|t| SetChoice { id: (*t).into(), text: (*t).into() }).collect(),
                _ => q.choices.clone(),
            };
            if choices.is_empty() || choices.len() > spec.max_choices {
                candle_core::bail!("head {:?} takes 1..={} candidates, got {}", spec.name, spec.max_choices, choices.len());
            }
            if spec.kind == SetKind::Score && choices.len() < 2 {
                candle_core::bail!("a score head needs at least two ordered levels");
            }
            if choices.iter().any(|c| c.text.is_empty() || c.text.bytes().any(|b| b == 0)) {
                candle_core::bail!("candidate text must be non-empty and contain no NUL");
            }
            let texts: Vec<String> = choices.iter().map(|c| c.text.clone()).collect();
            let question = q.question.clone().unwrap_or_else(|| q.name.clone());
            let toks = crate::tokenizer::encode_set_prompt(&question, q.state.as_deref().unwrap_or(""), &texts, width);
            let layout = ChoiceLayout::from_tokens(&toks).map_err(|e| {
                candle_core::Error::msg(format!("{e} (the prompt may exceed the model's {width}-token input)"))
            })?;
            if layout.choice_spans.len() != choices.len() {
                candle_core::bail!("the prompt holds {} of {} candidates: it exceeds the model's {width}-token input", layout.choice_spans.len(), choices.len());
            }
            let input = Tensor::from_slice(&toks, (1, toks.len()), &self.device)?;
            let (c, a) = self.set_inputs(&input, &layout)?;
            let probs = self.set_probabilities(&c, &a, k, &vec![true; choices.len()])?;
            if first_halt.is_none() {
                let tr = self.run(&input, Mode::Train, Some(&layout), None, false)?.0;
                first_halt = Some(ops::softmax(&tr.halt, D::Minus1)?.i((0, 0))?.to_scalar::<f32>()?);
                first_acts = Some(self.cfg.h_cycles);
            }
            let (arg, &max) = probs.iter().enumerate().max_by(|x, y| x.1.total_cmp(y.1)).expect("at least one candidate");
            let candidates: Vec<Candidate> = choices
                .iter()
                .zip(&probs)
                .map(|(c, &p)| Candidate { id: c.id.clone(), text: c.text.clone(), probability: p })
                .collect();
            decisions.push(match spec.kind {
                SetKind::Choice | SetKind::Null => {
                    let abstain = max < self.cfg.tau;
                    Decision {
                        name: spec.name.clone(),
                        kind: HeadKind::Choice,
                        options: texts,
                        probabilities: probs.clone(),
                        selected: if abstain { None } else { Some(choices[arg].text.clone()) },
                        value: None,
                        confidence: Some(max),
                        abstain,
                        candidates,
                    }
                }
                SetKind::Score => {
                    let n = probs.len();
                    let value: f32 = probs.iter().enumerate().map(|(i, p)| p * i as f32 / (n - 1) as f32).sum();
                    Decision {
                        name: spec.name.clone(),
                        kind: HeadKind::Score,
                        options: texts,
                        probabilities: probs.clone(),
                        selected: None,
                        value: Some(value),
                        confidence: Some(value.max(1.0 - value)),
                        abstain: false,
                        candidates,
                    }
                }
            });
        }
        let confidence = decisions.iter().filter_map(|d| d.confidence).sum::<f32>() / decisions.len() as f32;
        Ok(Reasoning {
            plan_tokens: Vec::new(),
            halting_steps: first_acts.unwrap_or(self.cfg.h_cycles),
            max_steps: self.cfg.h_cycles,
            halt_probability: first_halt.unwrap_or(0.0),
            confidence,
            decisions,
            l_module_activations: Vec::new(),
            h_module_activations: Vec::new(),
            model_version: self.cfg.model_version.clone(),
            dataset_manifest_hash: self.cfg.dataset_manifest_hash.clone(),
            policy_bundle_hash: self.cfg.policy_bundle_hash.clone(),
        })
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
                        candidates: Vec::new(),
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
                        candidates: Vec::new(),
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
    /// Set heads only: the request's candidates in the request's order, each with its
    /// probability. Empty (and omitted from the JSON) for a v1 head.
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub candidates: Vec<Candidate>,
}

/// One candidate of a set question with the probability the head gave it.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
pub struct Candidate {
    pub id: String,
    pub text: String,
    pub probability: f32,
}

/// One candidate of a request.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
pub struct SetChoice {
    pub id: String,
    pub text: String,
}

/// One question for a set head. `name` selects the head (and its task embedding); `question`
/// is the text the model reads (the name when absent); `state` is the context; `choices` is
/// ignored for a `null` head, whose candidates are always yes and no.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
pub struct SetQuestion {
    pub name: String,
    #[serde(default)]
    pub question: Option<String>,
    #[serde(default)]
    pub state: Option<String>,
    #[serde(default)]
    pub choices: Vec<SetChoice>,
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
