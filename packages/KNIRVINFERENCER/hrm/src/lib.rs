//! knirv-hrm — Hierarchical Reasoning Model. One implementation, three builds:
//! native (KNIRVMODELS trainer), `wasm32-unknown-unknown` + wasm-bindgen (arena),
//! and `wasm32-wasip1` (HEART via wazero).

pub mod adapter;
pub mod config;
pub mod model;
pub mod tokenizer;

pub use config::{DecisionHead, HeadKind, HrmConfig};
pub use model::{CycleOut, Decision, Hrm, Mode, Reasoning, Trace};

use candle_core::Device;
use wasm_bindgen::prelude::*;

/// Browser entry point. Deliberately thin: all logic lives in [`Hrm`] and is
/// covered by native tests. No fabricated output — until `load_model`
/// succeeds, `reason` is an error and the caller must fall back to the server.
#[wasm_bindgen]
pub struct HRMCognitive {
    model: Option<Hrm>,
}

#[wasm_bindgen]
impl HRMCognitive {
    #[wasm_bindgen(constructor)]
    pub fn new() -> HRMCognitive {
        HRMCognitive { model: None }
    }

    /// `config_json` is `hrm_config.json`; `weights` is the safetensors file.
    pub fn load_model(&mut self, config_json: &str, weights: &[u8]) -> Result<(), JsError> {
        let cfg = HrmConfig::from_json(config_json).map_err(|e| JsError::new(&e))?;
        let m = Hrm::from_safetensors(cfg, weights, &Device::Cpu).map_err(|e| JsError::new(&e.to_string()))?;
        self.model = Some(m);
        Ok(())
    }

    /// As `load_model`, merging a KNIRVULORA `bind` response (`adapter_json`) into
    /// the weights. Returns the JSON `Applied` summary. On error nothing changes.
    pub fn load_model_with_adapter(&mut self, config_json: &str, weights: &[u8], adapter_json: &str) -> Result<String, JsError> {
        let cfg = HrmConfig::from_json(config_json).map_err(|e| JsError::new(&e))?;
        let ad: adapter::Adapter = serde_json::from_str(adapter_json).map_err(|e| JsError::new(&format!("adapter: {e}")))?;
        let (m, applied) =
            Hrm::from_safetensors_adapted(cfg, weights, &Device::Cpu, Some(&ad)).map_err(|e| JsError::new(&e.to_string()))?;
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
        let m = self.model.as_ref().ok_or_else(|| JsError::new("HRM model not loaded"))?;
        let r = m.reason(&tokens).map_err(|e| JsError::new(&e.to_string()))?;
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
                "tau": m.cfg.tau,
            })
            .to_string(),
        }
    }
}

impl Default for HRMCognitive {
    fn default() -> Self {
        Self::new()
    }
}
