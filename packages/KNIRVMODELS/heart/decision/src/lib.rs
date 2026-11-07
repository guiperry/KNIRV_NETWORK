//! knirv-heart — Hierarchical Reasoning Model. One implementation, three builds:
//! native (KNIRVMODELS trainer), `wasm32-unknown-unknown` + wasm-bindgen (arena),
//! and `wasm32-wasip1` (HEART via wazero).

pub mod adapter;
pub mod config;
pub mod layout;
pub mod model;
pub mod tokenizer;

pub use config::{DecisionHead, HeadConfig, HeadKind, SetHead, SetKind};
pub use layout::{assert_additive, ChoiceLayout};
pub use model::{Candidate, CycleOut, Decision, Head, Mode, Reasoning, SetChoice, SetQuestion, Trace};
pub use model::{log_softmax_last_dim_diff, set_loss_from_logits, softmax_last_dim_diff};
/// Content-overlap feature for the set scorer (Task 4.4): `LCS(candidate, question)/len(candidate)`
/// per candidate. Shared by the trainer and every runtime so the model needs no external feature input.
pub use model::overlap_feature;

use candle_core::Device;
use wasm_bindgen::prelude::*;

/// Browser entry point. Deliberately thin: all logic lives in [`Head`] and is
/// covered by native tests. No fabricated output — until `load_model`
/// succeeds, `reason` is an error and the caller must fall back to the server.
#[wasm_bindgen]
pub struct HEADCognitive {
    model: Option<Head>,
}

#[wasm_bindgen]
impl HEADCognitive {
    #[wasm_bindgen(constructor)]
    pub fn new() -> HEADCognitive {
        HEADCognitive { model: None }
    }

    /// `config_json` is `head_config.json`; `weights` is the safetensors file.
    pub fn load_model(&mut self, config_json: &str, weights: &[u8]) -> Result<(), JsError> {
        let cfg = HeadConfig::from_json(config_json).map_err(|e| JsError::new(&e))?;
        let m = Head::from_safetensors(cfg, weights, &Device::Cpu).map_err(|e| JsError::new(&e.to_string()))?;
        self.model = Some(m);
        Ok(())
    }

    /// As `load_model`, merging a KNIRVULORA `bind` response (`adapter_json`) into
    /// the weights. Returns the JSON `Applied` summary. On error nothing changes.
    pub fn load_model_with_adapter(&mut self, config_json: &str, weights: &[u8], adapter_json: &str) -> Result<String, JsError> {
        let cfg = HeadConfig::from_json(config_json).map_err(|e| JsError::new(&e))?;
        let ad: adapter::Adapter = serde_json::from_str(adapter_json).map_err(|e| JsError::new(&format!("adapter: {e}")))?;
        let (m, applied) =
            Head::from_safetensors_adapted(cfg, weights, &Device::Cpu, Some(&ad)).map_err(|e| JsError::new(&e.to_string()))?;
        self.model = Some(m);
        serde_json::to_string(&applied).map_err(|e| JsError::new(&e.to_string()))
    }

    /// Names of the weights an adapter can change, for the mapping table.
    pub fn adaptable_tensors(&self) -> Vec<String> {
        self.model.as_ref().map(|m| adapter::adaptable_tensors(&m.cfg)).unwrap_or_default()
    }

    pub fn is_loaded(&self) -> bool {
        self.model.is_some()
    }

    /// Reason over token ids; returns the JSON-serialised [`Reasoning`].
    pub fn reason(&self, tokens: Vec<u32>) -> Result<String, JsError> {
        let m = self.model.as_ref().ok_or_else(|| JsError::new("HEAD model not loaded"))?;
        let r = m.reason(&tokens).map_err(|e| JsError::new(&e.to_string()))?;
        serde_json::to_string(&r).map_err(|e| JsError::new(&e.to_string()))
    }

    /// Answer choice-set questions (a v2 model): `questions_json` is an array of
    /// `{name, question?, state?, choices: [{id, text}]}`; returns the JSON [`Reasoning`] whose
    /// decisions carry `candidates` in the request's order.
    pub fn reason_sets(&self, questions_json: &str) -> Result<String, JsError> {
        let m = self.model.as_ref().ok_or_else(|| JsError::new("HEAD model not loaded"))?;
        let qs: Vec<SetQuestion> = serde_json::from_str(questions_json).map_err(|e| JsError::new(&format!("questions: {e}")))?;
        let r = m.reason_sets(&qs).map_err(|e| JsError::new(&e.to_string()))?;
        serde_json::to_string(&r).map_err(|e| JsError::new(&e.to_string()))
    }

    /// `{loaded, total_parameters, l_modules, h_modules, max_input_len, ...}`.
    pub fn get_model_info(&self) -> String {
        match &self.model {
            None => r#"{"loaded":false,"total_parameters":0,"l_modules":0,"h_modules":0}"#.to_string(),
            Some(m) => serde_json::json!({
                "loaded": true,
                "total_parameters": m.param_count(),
                "l_modules": m.cfg.l_layers,
                "h_modules": m.cfg.h_layers,
                "h_cycles": m.cfg.h_cycles,
                "l_steps": m.cfg.l_steps,
                "max_input_len": m.max_input_len(),
                "vocab_size": m.cfg.vocab_size,
                "tokenizer": m.cfg.tokenizer,
                "model_version": m.cfg.model_version,
                "decision_heads": m.cfg.decision_heads,
                "choice_set_heads": m.cfg.choice_set_heads,
                "tau": m.cfg.tau,
            })
            .to_string(),
        }
    }
}

impl Default for HEADCognitive {
    fn default() -> Self {
        Self::new()
    }
}
