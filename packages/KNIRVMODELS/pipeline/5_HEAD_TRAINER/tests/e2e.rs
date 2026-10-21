use candle_core::Device;
use knirv_head::{tokenizer, Head, HeadConfig};
use std::path::{Path, PathBuf};
use std::process::Command;

const KINDS: [(&str, &str, f32); 5] = [
    ("delete files", "ask_operator", 0.9),
    ("review diff", "request_review", 0.4),
    ("run tests", "allow", 0.1),
    ("add feature", "allow", 0.2),
    ("fix crash", "recover", 0.6),
];

fn scratch(name: &str) -> PathBuf {
    let d = std::env::temp_dir().join(format!("head-trainer-{name}-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&d);
    std::fs::create_dir_all(&d).unwrap();
    d
}

fn write_data(dir: &Path, n: usize) -> PathBuf {
    let mut lines = Vec::new();
    for i in 0..n {
        let (text, action, risk) = KINDS[i % KINDS.len()];
        lines.push(serde_json::json!({
            "id": format!("ep-{i}"), "group": format!("fam-{}", i % 37),
            "input": format!("{text} {i}"),
            "labels": {"action": action, "risk": risk}
        }).to_string());
    }
    let p = dir.join("train.jsonl");
    std::fs::write(&p, lines.join("\n")).unwrap();
    p
}

fn run(args: &[&str]) -> (i32, String) {
    let o = Command::new(env!("CARGO_BIN_EXE_head-trainer")).args(args).output().unwrap();
    (o.status.code().unwrap_or(-1), String::from_utf8_lossy(&o.stderr).into_owned())
}

fn metrics(out: &Path) -> serde_json::Value {
    serde_json::from_str(&std::fs::read_to_string(out.join("head/metrics.json")).unwrap()).unwrap()
}

#[test]
fn trains_promotes_and_serves_a_loadable_model_then_the_gate_refuses_a_worse_one() {
    let dir = scratch("e2e");
    let data = write_data(&dir, 250);
    let (d, out) = (data.to_str().unwrap(), dir.join("out"));
    let o = out.to_str().unwrap();

    let (code, log) = run(&["--data", d, "--out", o, "--preset", "micro", "--epochs", "50", "--batch", "16", "--lr", "1e-3", "--warmup", "20", "--seed", "3", "--min-heldout", "10"]);
    assert_eq!(code, 0, "{log}");
    let m = metrics(&out);
    assert_eq!(m["gate"]["promoted"], true);
    let acc = m["heldout"]["heads"]["action"]["accuracy"].as_f64().unwrap();
    // Majority-class baseline is 0.4; a short run on a tiny model need only clearly beat it.
    assert!(acc > 0.65, "held-out action accuracy {acc}\n{log}");
    assert!(m["heldout"]["heads"]["risk"]["mse"].as_f64().unwrap() < 0.08);
    assert!(m["heldout"]["plan_token_accuracy"].is_null(), "no plan supervision in this data");
    // H6: the held-out metrics are tau-aware (batch-1 evaluation at the fitted tau).
    assert!(m["heldout"]["coverage"].as_f64().is_some() && m["heldout"]["tau"].as_f64().is_some(), "H6: tau-aware metrics");
    assert_eq!(m["load_stats"]["split_mode"], "group-hash");
    assert!(m["init_from"].is_null(), "first model trains from random init");

    // Published layout, and the artefact loads in the same crate the browser/HEART run.
    let head = out.join("head");
    let cfg = HeadConfig::from_json(&std::fs::read_to_string(head.join("head_config.json")).unwrap()).unwrap();
    assert!(cfg.model_version.starts_with('v') && cfg.dataset_manifest_hash.len() == 64);
    assert_eq!(cfg.tokenizer, "byte-v1");
    let link = std::fs::read_link(head.join("model_latest")).unwrap();
    let weights = std::fs::read(head.join(&link)).unwrap();
    assert_eq!(m["sha256"].as_str().unwrap().len(), 64);
    assert_eq!(m["size"].as_u64().unwrap() as usize, weights.len());
    let model = Head::from_safetensors(cfg.clone(), &weights, &Device::Cpu).unwrap();
    let w = model.max_input_len();
    let input = tokenizer::pad_to(tokenizer::encode_text("run tests 2", w), w);
    let r = model.reason(&input).unwrap();
    let action = r.decisions.iter().find(|d| d.name == "action").unwrap();
    assert!(action.selected.is_some() == !action.abstain);
    assert!((action.probabilities.iter().sum::<f32>() - 1.0).abs() < 1e-4);
    let risk = r.decisions.iter().find(|d| d.name == "risk").unwrap().value.unwrap();
    assert!(risk > 0.0 && risk < 1.0, "risk {risk}");
    assert_eq!(r.plan_tokens.len(), w);

    // A barely trained candidate must not replace it.
    let first = cfg.model_version.clone();
    let (code, log) = run(&["--data", d, "--out", o, "--preset", "micro", "--epochs", "1", "--max-steps", "1", "--seed", "9"]);
    assert_eq!(code, 3, "{log}");
    let m2 = metrics(&out);
    assert_eq!(m2["gate"]["promoted"], false);
    assert_eq!(m2["init_from"], first.as_str(), "H8: the next cycle fine-tunes the promoted model");
    assert_eq!(m2["gate"]["current_version"], first.as_str());
    let still = HeadConfig::from_json(&std::fs::read_to_string(head.join("head_config.json")).unwrap()).unwrap();
    assert_eq!(still.model_version, first);
    assert_eq!(std::fs::read_link(head.join("model_latest")).unwrap(), link);
    // ...but the rejected candidate is kept for inspection.
    let kept = std::fs::read_dir(head.join("versions")).unwrap().count();
    assert_eq!(kept, 2);
}

#[test]
fn rejects_malformed_data() {
    let dir = scratch("bad");
    let out = dir.join("out");
    let cases = [
        (r#"{"id":"a","input":"x","labels":{"risk":0.5}}
{"id":"b","input":"y","labels":{"risk":"high"}}"#, "mixes choice and score"),
        (r#"{"id":"a","input":"x","labels":{"action":"explode"}}
{"id":"b","input":"y","labels":{"action":"allow"}}"#, "not in the fixed vocabulary"),
        (r#"{"id":"a","input":"x","labels":{"risk":1.5}}"#, "outside [0,1]"),
        (r#"{"id":"a","input":"x"}
{"id":"a","input":"y"}"#, "duplicate record id"),
        (r#"{"id":"a","input":"x","split":"dev"}"#, "unknown split"),
    ];
    for (i, (body, want)) in cases.iter().enumerate() {
        let p = dir.join(format!("bad{i}.jsonl"));
        std::fs::write(&p, body).unwrap();
        let (code, log) = run(&["--data", p.to_str().unwrap(), "--out", out.to_str().unwrap(), "--preset", "micro"]);
        assert_eq!(code, 1, "{want}: {log}");
        assert!(log.contains(want), "want {want:?} in {log}");
    }
}

#[test]
fn splits_by_group_not_by_row() {
    let dir = scratch("split");
    let data = write_data(&dir, 200);
    let ds = head_trainer::data::load(&[data.to_str().unwrap().into()], 31).unwrap();
    use head_trainer::data::Split;
    let mut owner = std::collections::HashMap::new();
    for e in &ds.examples {
        // ids are ep-<i>; group is fam-(i%37)
        let i: usize = e.id.trim_start_matches("ep-").parse().unwrap();
        let prev = owner.insert(i % 37, e.split);
        assert!(prev.is_none() || prev == Some(e.split), "group fam-{} straddles splits", i % 37);
    }
    for s in [Split::Train, Split::Val, Split::Test] {
        assert!(!ds.split(s).is_empty(), "{s:?} empty");
    }
}

#[test]
fn convert_produces_trainable_records() {
    let dir = scratch("convert");
    let mut lines = Vec::new();
    for i in 0..40 {
        let (act, risk) = if i % 2 == 0 { ("allow", "low") } else { ("ask_operator", "high") };
        lines.push(serde_json::json!({
            "episode_id": format!("ep-{i}"), "workspace": {"repo_family": format!("fam-{}", i % 9)},
            "task": {"request": format!("task {i}"), "intent": "develop", "risk": risk},
            "state_before": {"phase": "develop", "budget": {"iteration": i}, "agent_chain": ["claude"]},
            "evidence": [{"class": "diff"}], "trace": {"agent": "claude"},
            "decision": {"recommended_action": act, "next_phase": "commit", "abstain": false},
            "outcome": {"verified": i % 5 != 0, "human_override": false}
        }).to_string());
    }
    let (ep, out) = (dir.join("ep.jsonl"), dir.join("train.jsonl"));
    std::fs::write(&ep, lines.join("\n")).unwrap();
    let o = Command::new(env!("CARGO_BIN_EXE_head-trainer"))
        .args(["convert", "--episodes", ep.to_str().unwrap(), "--out", out.to_str().unwrap()]).output().unwrap();
    assert!(o.status.success(), "{}", String::from_utf8_lossy(&o.stderr));
    assert!(String::from_utf8_lossy(&o.stderr).contains("written: 32"), "{}", String::from_utf8_lossy(&o.stderr));
    let ds = head_trainer::data::load(&[out.to_str().unwrap().into()], 63).unwrap();
    assert_eq!(ds.examples.len(), 32);
    let names: Vec<_> = ds.heads.iter().map(|h| h.name.as_str()).collect();
    // next_phase has one value in this data and is skipped; intent/agent use fixed vocabularies.
    // intent and risk are fields of the prompt, so they are not emitted (H4).
    assert_eq!(names, ["action", "agent"]);
}

fn write_rows(dir: &Path, name: &str, rows: Vec<serde_json::Value>) -> PathBuf {
    let p = dir.join(name);
    std::fs::write(&p, rows.iter().map(|r| r.to_string()).collect::<Vec<_>>().join("\n")).unwrap();
    p
}

fn kind_rows(n: usize, group: impl Fn(usize) -> String, input: impl Fn(usize, &str) -> String) -> Vec<serde_json::Value> {
    (0..n)
        .map(|i| {
            let (text, action, risk) = KINDS[i % KINDS.len()];
            serde_json::json!({"id": format!("ep-{i}"), "group": group(i), "ts": format!("2026-01-01T{:02}:{:02}:00Z", i / 60, i % 60),
                "input": input(i, text), "labels": {"action": action, "risk": risk}})
        })
        .collect()
}

#[test]
fn single_group_data_is_split_by_time_and_never_evaluated_on_train() {
    let dir = scratch("onegroup");
    let p = write_rows(&dir, "one.jsonl", kind_rows(250, |_| "only".into(), |i, t| format!("{t} {i}")));
    let out = dir.join("out");
    let (code, log) = run(&["--data", p.to_str().unwrap(), "--out", out.to_str().unwrap(), "--preset", "micro", "--epochs", "3", "--min-heldout", "10"]);
    assert!(code == 0 || code == 3, "{log}");
    let m = metrics(&out);
    assert_eq!(m["load_stats"]["split_mode"], "time");
    assert_eq!(m["heldout_source"], "test split");
    assert!(m["heldout"]["examples"].as_u64().unwrap() >= 10);
    assert!(log.contains("182 train") || log.contains("200 train"), "{log}");
}

#[test]
fn without_disjoint_heldout_data_the_run_trains_but_is_refused() {
    let dir = scratch("noheld");
    let p = write_rows(&dir, "two.jsonl", kind_rows(2, |i| format!("g{i}"), |i, t| format!("{t} {i}")));
    let out = dir.join("out");
    let (code, log) = run(&["--data", p.to_str().unwrap(), "--out", out.to_str().unwrap(), "--preset", "micro", "--epochs", "1"]);
    assert_eq!(code, 3, "{log}");
    let m = metrics(&out);
    assert_eq!(m["gate"]["promoted"], false);
    assert_eq!(m["gate"]["reason"], "no_disjoint_heldout");
    assert_eq!(m["tau_fit"]["met"], false);
    assert!(!out.join("head/model_latest").exists());
}

#[test]
fn a_head_that_can_read_its_answer_in_the_prompt_is_refused() {
    let dir = scratch("leak");
    let rows: Vec<_> = (0..60)
        .map(|i| {
            let action = if i % 2 == 0 { "allow" } else { "ask_operator" };
            serde_json::json!({"id": format!("e{i}"), "group": format!("g{i}"), "input": format!("action:{action} do {i}"), "labels": {"action": action}})
        })
        .collect();
    let p = write_rows(&dir, "leak.jsonl", rows);
    let (code, log) = run(&["--data", p.to_str().unwrap(), "--out", dir.join("out").to_str().unwrap(), "--preset", "micro", "--epochs", "1"]);
    assert_eq!(code, 1, "{log}");
    assert!(log.contains("label leakage") && log.contains("action"), "{log}");
}

#[test]
fn inputs_cut_off_by_the_preset_fail_the_run() {
    let dir = scratch("trunc");
    let p = write_rows(&dir, "long.jsonl", kind_rows(60, |i| format!("g{i}"), |i, t| format!("{t} {i} {}", "x".repeat(200))));
    let (code, log) = run(&["--data", p.to_str().unwrap(), "--out", dir.join("out").to_str().unwrap(), "--preset", "micro", "--epochs", "1"]);
    assert_eq!(code, 1, "{log}");
    assert!(log.contains("do not fit preset"), "{log}");
}

#[test]
fn the_halt_head_learns_without_any_plan_supervision() {
    // H1: this data carries no plan. The halt target comes from decision-head correctness, so
    // on an easy task (two actions told apart by one word) the model learns to stop at cycle 1.
    let dir = scratch("halt");
    let rows: Vec<_> = (0..300)
        .map(|i| {
            let action = if i % 2 == 0 { "allow" } else { "ask_operator" };
            let word = if i % 2 == 0 { "list" } else { "delete" };
            serde_json::json!({"id": format!("e{i}"), "group": format!("g{}", i % 30), "input": format!("{word} files number {i}"), "labels": {"action": action}})
        })
        .collect();
    let p = write_rows(&dir, "easy.jsonl", rows);
    let out = dir.join("out");
    let (code, log) = run(&["--data", p.to_str().unwrap(), "--out", out.to_str().unwrap(), "--preset", "micro", "--epochs", "40", "--lr", "1e-3", "--warmup", "20", "--seed", "3", "--min-heldout", "10"]);
    assert!(code == 0 || code == 3, "{log}");
    let m = metrics(&out);
    let acc = m["heldout"]["heads"]["action"]["accuracy"].as_f64().unwrap();
    let halting = m["heldout"]["mean_halting_steps"].as_f64().unwrap();
    println!("accuracy {acc}, mean_halting_steps {halting}");
    assert!(acc > 0.95, "task should be easy: {acc}\n{log}");
    assert!(halting < 1.5, "halt head did not learn to stop early: {halting}");
}
