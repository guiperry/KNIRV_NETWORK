//! WASI entry point for HEART (wazero): one JSON request per stdin line, one JSON
//! response per stdout line. Same `Hrm` as the browser build, so outputs agree.
//!
//! {"op":"load","config_path":"/m/hrm_config.json","weights_path":"/m/model.safetensors"}
//! {"op":"reason","tokens":[257,104,105,256]}
//! {"op":"info"}
//! Responses: {"ok":true,"result":...} or {"ok":false,"error":"..."}.

use candle_core::Device;
use knirv_cortex_wasm::{Hrm, HrmConfig};
use serde_json::{json, Value};
use std::io::{BufRead, Write};

fn handle(model: &mut Option<Hrm>, line: &str) -> Result<Value, String> {
    let req: Value = serde_json::from_str(line).map_err(|e| format!("bad request: {e}"))?;
    match req["op"].as_str().unwrap_or("") {
        "load" => {
            let read = |k: &str| -> Result<Vec<u8>, String> {
                let p = req[k].as_str().ok_or_else(|| format!("{k} required"))?;
                std::fs::read(p).map_err(|e| format!("{p}: {e}"))
            };
            let cfg = HrmConfig::from_json(&String::from_utf8_lossy(&read("config_path")?))?;
            let m = Hrm::from_safetensors(cfg, &read("weights_path")?, &Device::Cpu).map_err(|e| e.to_string())?;
            let info = info(&m);
            *model = Some(m);
            Ok(info)
        }
        "reason" => {
            let m = model.as_ref().ok_or("no model loaded")?;
            let tokens: Vec<u32> = serde_json::from_value(req["tokens"].clone()).map_err(|e| format!("tokens: {e}"))?;
            let r = m.reason(&tokens).map_err(|e| e.to_string())?;
            serde_json::to_value(r).map_err(|e| e.to_string())
        }
        "info" => Ok(model.as_ref().map(info).unwrap_or_else(|| json!({"loaded": false}))),
        other => Err(format!("unknown op {other:?}")),
    }
}

fn info(m: &Hrm) -> Value {
    json!({
        "loaded": true, "total_parameters": m.param_count(),
        "l_modules": m.cfg.l_layers, "h_modules": m.cfg.h_layers,
        "max_input_len": m.max_input_len(), "tokenizer": m.cfg.tokenizer,
        "model_version": m.cfg.model_version, "vocab_size": m.cfg.vocab_size,
    })
}

fn main() {
    let mut model = None;
    let stdout = std::io::stdout();
    for line in std::io::stdin().lock().lines() {
        let Ok(line) = line else { break };
        if line.trim().is_empty() {
            continue;
        }
        let resp = match handle(&mut model, &line) {
            Ok(v) => json!({"ok": true, "result": v}),
            Err(e) => json!({"ok": false, "error": e}),
        };
        let mut out = stdout.lock();
        let _ = writeln!(out, "{resp}");
        let _ = out.flush();
    }
}
