//! WASI entry point for HEART (wazero): one JSON request per stdin line, one JSON
//! response per stdout line. Same `Head` as the browser build, so outputs agree.
//!
//! {"op":"load","config_path":"/m/head_config.json","weights_path":"/m/model.safetensors"}
//! {"op":"load",...,"adapter":{...}} or {"op":"load",...,"adapter_path":"/m/adapter.json"}
//! {"op":"reason","tokens":[257,104,105,256]}
//! {"op":"reason_sets","questions":[{...}]}
//! {"op":"info"}
//! Responses: {"ok":true,"result":...} or {"ok":false,"error":"..."}.

use candle_core::Device;
use knirv_heart_decision::adapter::Adapter;
use knirv_heart_decision::{Head, HeadConfig, SetQuestion};
use serde_json::{json, Value};
use std::io::{BufRead, Write};

fn handle(model: &mut Option<Head>, line: &str) -> Result<Value, String> {
    let req: Value = serde_json::from_str(line).map_err(|e| format!("bad request: {e}"))?;
    match req["op"].as_str().unwrap_or("") {
        "load" => {
            let read = |k: &str| -> Result<Vec<u8>, String> {
                let p = req[k].as_str().ok_or_else(|| format!("{k} required"))?;
                std::fs::read(p).map_err(|e| format!("{p}: {e}"))
            };
            let cfg = HeadConfig::from_json(&String::from_utf8_lossy(&read("config_path")?))?;
            let weights = read("weights_path")?;
            let adapter = match req.get("adapter") {
                Some(v) if !v.is_null() => Some(
                    serde_json::from_value::<Adapter>(v.clone())
                        .map_err(|e| format!("adapter: {e}"))?,
                ),
                _ => match req.get("adapter_path").and_then(Value::as_str) {
                    Some(p) => {
                        let bytes = std::fs::read(p).map_err(|e| format!("{p}: {e}"))?;
                        Some(
                            serde_json::from_slice::<Adapter>(&bytes)
                                .map_err(|e| format!("adapter: {e}"))?,
                        )
                    }
                    None => None,
                },
            };
            // Nothing is swapped until the adapter has merged cleanly; a rejected
            // adapter leaves the previous model serving.
            let (m, applied) =
                Head::from_safetensors_adapted(cfg, &weights, &Device::Cpu, adapter.as_ref())
                    .map_err(|e| e.to_string())?;
            let mut info = info(&m);
            if let Some(applied) = applied {
                info["adapter"] = serde_json::to_value(applied).map_err(|e| e.to_string())?;
            }
            *model = Some(m);
            Ok(info)
        }
        "reason" => {
            let m = model.as_ref().ok_or("no model loaded")?;
            let tokens: Vec<u32> = serde_json::from_value(req["tokens"].clone()).map_err(|e| format!("tokens: {e}"))?;
            let r = m.reason(&tokens).map_err(|e| e.to_string())?;
            serde_json::to_value(r).map_err(|e| e.to_string())
        }
        "reason_sets" => {
            let m = model.as_ref().ok_or("no model loaded")?;
            let qs: Vec<SetQuestion> =
                serde_json::from_value(req["questions"].clone()).map_err(|e| format!("questions: {e}"))?;
            let r = m.reason_sets(&qs).map_err(|e| e.to_string())?;
            serde_json::to_value(r).map_err(|e| e.to_string())
        }
        "info" => Ok(model.as_ref().map(info).unwrap_or_else(|| json!({"loaded": false}))),
        other => Err(format!("unknown op {other:?}")),
    }
}

fn info(m: &Head) -> Value {
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
