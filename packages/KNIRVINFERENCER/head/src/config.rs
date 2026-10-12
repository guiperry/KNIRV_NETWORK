//! `head_config.json` (format `knirv-head/v1`). The config is the single source of
//! truth for architecture: parameter counts come from it, never from a constant.

use serde::{Deserialize, Serialize};

pub const FORMAT: &str = "knirv-head/v1";

#[derive(Clone, Copy, Debug, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum HeadKind {
    /// Softmax over `options`.
    Choice,
    /// Continuous value in [0, 1] (sigmoid). Never rounded by the model.
    Score,
}

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
pub struct DecisionHead {
    pub name: String,
    pub kind: HeadKind,
    /// Option labels for `choice`; ignored (may be empty) for `score`.
    #[serde(default)]
    pub options: Vec<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
pub struct HeadConfig {
    pub format: String,
    /// Identifies the token id space (e.g. `tiktoken-cl100k`, `byte`).
    pub tokenizer: String,
    pub vocab_size: usize,
    pub max_seq_len: usize,
    pub d_model: usize,
    pub n_heads: usize,
    pub ffn_mult: usize,
    pub l_layers: usize,
    pub h_layers: usize,
    /// Maximum high-level cycles N (ACT upper bound).
    pub h_cycles: usize,
    /// Low-level steps T per high-level cycle.
    pub l_steps: usize,
    /// Minimum cycles before the halt head may stop reasoning.
    #[serde(default = "one")]
    pub min_cycles: usize,
    /// Number of leading positions reserved as summary slots; slot 0 feeds
    /// the halting and decision heads.
    #[serde(default = "one")]
    pub summary_slots: usize,
    pub decision_heads: Vec<DecisionHead>,
    /// Abstention threshold: a `choice` head abstains when max(p) < tau.
    /// Fitted on the Gold manifest by the trainer; 0 disables abstention.
    #[serde(default)]
    pub tau: f32,
    #[serde(default)]
    pub model_version: String,
    #[serde(default)]
    pub dataset_manifest_hash: String,
    #[serde(default)]
    pub policy_bundle_hash: String,
}

fn one() -> usize {
    1
}

impl HeadConfig {
    pub fn from_json(s: &str) -> Result<Self, String> {
        let c: HeadConfig = serde_json::from_str(s).map_err(|e| format!("invalid head_config.json: {e}"))?;
        c.validate()?;
        Ok(c)
    }

    pub fn validate(&self) -> Result<(), String> {
        if self.format != FORMAT {
            return Err(format!("unsupported format {:?}, want {FORMAT:?}", self.format));
        }
        let pos = |name: &str, v: usize| if v == 0 { Err(format!("{name} must be > 0")) } else { Ok(()) };
        pos("vocab_size", self.vocab_size)?;
        pos("max_seq_len", self.max_seq_len)?;
        pos("d_model", self.d_model)?;
        pos("n_heads", self.n_heads)?;
        pos("ffn_mult", self.ffn_mult)?;
        pos("l_layers", self.l_layers)?;
        pos("h_layers", self.h_layers)?;
        pos("h_cycles", self.h_cycles)?;
        pos("l_steps", self.l_steps)?;
        pos("summary_slots", self.summary_slots)?;
        if self.d_model % self.n_heads != 0 {
            return Err("d_model must be divisible by n_heads".into());
        }
        if self.min_cycles == 0 || self.min_cycles > self.h_cycles {
            return Err("min_cycles must be in 1..=h_cycles".into());
        }
        if self.summary_slots >= self.max_seq_len {
            return Err("summary_slots must be < max_seq_len".into());
        }
        if !(0.0..=1.0).contains(&self.tau) {
            return Err("tau must be in [0, 1]".into());
        }
        let mut seen = std::collections::HashSet::new();
        for h in &self.decision_heads {
            if h.name.is_empty() || !seen.insert(h.name.as_str()) {
                return Err(format!("decision head name {:?} empty or duplicated", h.name));
            }
            if h.kind == HeadKind::Choice && h.options.len() < 2 {
                return Err(format!("choice head {:?} needs >= 2 options", h.name));
            }
        }
        Ok(())
    }

    pub fn head_dim(&self) -> usize {
        self.d_model / self.n_heads
    }

    /// Parameter count implied by the architecture (checked against the weights at load).
    pub fn param_count(&self) -> usize {
        let d = self.d_model;
        let f = self.ffn_mult * d;
        // per block: attn q,k,v,o (4 d^2) + SwiGLU gate/up/down (3 d f) + 2 rms norms (2 d)
        let block = 4 * d * d + 3 * d * f + 2 * d;
        let mut n = self.vocab_size * d; // token embedding (tied output head)
        n += self.max_seq_len * d; // learned positions
        n += self.summary_slots * d; // learned summary slots
        n += 2 * d; // z_l / z_h init
        n += (self.l_layers + self.h_layers) * block;
        n += d; // final norm
        n += d * 2 + 2; // halt head (q_halt, q_continue)
        for h in &self.decision_heads {
            let out = match h.kind {
                HeadKind::Choice => h.options.len(),
                HeadKind::Score => 1,
            };
            n += d * out + out;
        }
        n
    }
}
