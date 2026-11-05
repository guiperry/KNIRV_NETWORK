use candle_core::{DType, Device};
use candle_nn::{VarBuilder, VarMap};
use heart_decision_trainer::data::{self, LoadOpts, Split};
use heart_decision_trainer::eval::{evaluate_with, fit_tau, Restrict, TauFit};
use heart_decision_trainer::export::{decide, version_for, Comparison, GateOpts, Report, Store};
use heart_decision_trainer::preset;
use heart_decision_trainer::preset_v2;
use heart_decision_trainer::setprobe::{evaluate_set, train_set, SetExample, SetTrainOpts};
use heart_decision_trainer::synth;
use heart_decision_trainer::train::seed_init;
use heart_decision_trainer::train::{train, TrainOpts};
use knirv_heart_decision::{Head, SetHead, SetKind};
use std::path::PathBuf;
use std::process::ExitCode;

const USAGE: &str = "heart-decision-trainer --data <train.jsonl>... --out <dir> [options]

  --data FILE         head-train/v1 JSONL (repeatable); split by `split` or by hash of `group`
  --heldout FILE      held-out evaluation set (repeatable); default: the data's test split
  --out DIR           writes DIR/head/ (config, safetensors, model_latest, metrics.json)
  --preset NAME       micro | tiny | small | base (default small)
  --epochs N          (default 20)        --batch N   (default 16)
  --lr X              peak learning rate (default 3e-4; diverges above ~3e-3)
  --warmup N          warmup steps (default 100)      --max-steps N
  --target-acc X      selective accuracy tau must reach on each choice head (default 0.95)
  --policy-bundle-hash H   recorded in the config
  --seed N            (default 1; logged in metrics.json)
  --init-from auto|none   fine-tune from the promoted model when its architecture and heads
                      match (default auto); `none` trains from random init (sanity run)
  --ft-lr-scale X     learning-rate multiplier when fine-tuning (default 0.3)
  --min-groups N      fewer distinct groups than this -> split by time (default 5)
  --max-truncated X   fail if more than this share of inputs is cut to fit the preset (default 0.05)
  --min-heldout N     rows needed before anything is promoted (default 30)
  --min-margin X      smallest mean per-row gain worth replacing the current model (default 0.005)
  --allow-leaky-heads train even if a head's answer is spelled out in its input
Exit status: 0 promoted, 3 trained but refused by the gate, 1 error.";

struct Args {
    data: Vec<String>,
    heldout: Vec<String>,
    out: PathBuf,
    preset: String,
    epochs: usize,
    batch: usize,
    lr: f64,
    warmup: usize,
    max_steps: Option<usize>,
    target_acc: f32,
    policy: String,
    seed: u64,
    init_from: String,
    ft_lr_scale: f64,
    min_groups: usize,
    max_truncated: f32,
    min_heldout: usize,
    min_margin: f32,
    allow_leaky: bool,
}

fn parse() -> Result<Args, String> {
    let mut a = Args {
        data: vec![], heldout: vec![], out: PathBuf::new(), preset: "small".into(), epochs: 20, batch: 16,
        lr: 3e-4, warmup: 100, max_steps: None, target_acc: 0.95, policy: String::new(), seed: 1,
        init_from: "auto".into(), ft_lr_scale: 0.3, min_groups: 5, max_truncated: 0.05, min_heldout: 30, min_margin: 0.005,
        allow_leaky: false,
    };
    let mut it = std::env::args().skip(1);
    while let Some(k) = it.next() {
        let mut v = || it.next().ok_or_else(|| format!("{k} needs a value"));
        let num = |s: String| s.parse::<f64>().map_err(|e| format!("{k}: {e}"));
        match k.as_str() {
            "--data" => a.data.push(v()?),
            "--heldout" => a.heldout.push(v()?),
            "--out" => a.out = v()?.into(),
            "--preset" => a.preset = v()?,
            "--epochs" => a.epochs = num(v()?)? as usize,
            "--batch" => a.batch = num(v()?)? as usize,
            "--lr" => a.lr = num(v()?)?,
            "--warmup" => a.warmup = num(v()?)? as usize,
            "--max-steps" => a.max_steps = Some(num(v()?)? as usize),
            "--target-acc" => a.target_acc = num(v()?)? as f32,
            "--policy-bundle-hash" => a.policy = v()?,
            "--seed" => a.seed = num(v()?)? as u64,
            "--init-from" => a.init_from = v()?,
            "--ft-lr-scale" => a.ft_lr_scale = num(v()?)?,
            "--min-groups" => a.min_groups = num(v()?)? as usize,
            "--max-truncated" => a.max_truncated = num(v()?)? as f32,
            "--min-heldout" => a.min_heldout = num(v()?)? as usize,
            "--min-margin" => a.min_margin = num(v()?)? as f32,
            "--allow-leaky-heads" => a.allow_leaky = true,
            "-h" | "--help" => return Err(USAGE.into()),
            other => return Err(format!("unknown argument {other}\n{USAGE}")),
        }
    }
    if a.data.is_empty() || a.out.as_os_str().is_empty() {
        return Err(USAGE.into());
    }
    if a.batch == 0 || a.epochs == 0 || !(a.lr > 0.0) {
        return Err("--batch, --epochs and --lr must be positive".into());
    }
    if a.init_from != "auto" && a.init_from != "none" {
        return Err("--init-from must be auto or none".into());
    }
    Ok(a)
}

/// Name of every head both models have, with the options they share. Score heads map to an empty set.
fn shared_heads(a: &knirv_heart_decision::HeadConfig, b: &knirv_heart_decision::HeadConfig) -> Restrict {
    let mut r = Restrict::new();
    for ha in &a.decision_heads {
        let Some(hb) = b.decision_heads.iter().find(|h| h.name == ha.name && h.kind == ha.kind) else { continue };
        let common: std::collections::BTreeSet<String> = ha.options.iter().filter(|o| hb.options.contains(o)).cloned().collect();
        if ha.kind == knirv_heart_decision::HeadKind::Choice && common.len() < 2 {
            continue;
        }
        r.insert(ha.name.clone(), common);
    }
    r
}

/// Same architecture and the same heads in the same order, so weights can carry over.
fn same_shape(a: &knirv_heart_decision::HeadConfig, b: &knirv_heart_decision::HeadConfig) -> bool {
    let norm = |c: &knirv_heart_decision::HeadConfig| {
        let mut c = c.clone();
        c.tau = 0.0;
        c.model_version.clear();
        c.dataset_manifest_hash.clear();
        c.policy_bundle_hash.clear();
        serde_json::to_value(&c).expect("config serialises")
    };
    norm(a) == norm(b)
}

fn run(a: Args) -> Result<bool, String> {
    let e = |x: candle_core::Error| x.to_string();
    let dev = Device::Cpu;

    // The data decides the heads; the preset decides the architecture.
    let probe = preset(&a.preset, vec![])
        .map(|c| c.max_seq_len - c.summary_slots)?;
    let lopts = LoadOpts { min_groups: a.min_groups };
    let ds = data::load_with(&a.data, probe, &lopts)?;
    let mut cfg = preset(&a.preset, ds.heads.clone())?;
    let width = cfg.max_seq_len - cfg.summary_slots;

    // H5: a model that never sees the request cannot learn from it.
    let trunc = ds.stats.truncated as f32 / ds.examples.len() as f32;
    eprintln!("truncated inputs: {} of {} ({:.1}%) at width {width}", ds.stats.truncated, ds.examples.len(), trunc * 100.0);
    if trunc > a.max_truncated {
        return Err(format!(
            "{:.1}% of inputs do not fit preset {:?} (width {width}); limit is {:.1}%. Use a wider preset or raise --max-truncated",
            trunc * 100.0, a.preset, a.max_truncated * 100.0
        ));
    }

    // H4: a head whose answer is written in its own input is copying, not deciding.
    let texts = data::input_texts(&a.data)?;
    let leaky = data::leaky_heads(&ds.examples, &ds.heads, &texts, 0.9);
    if !leaky.is_empty() {
        let msg = leaky.iter().map(|(h, f)| format!("{h} ({:.0}% of rows spell out the label)", f * 100.0)).collect::<Vec<_>>().join(", ");
        if !a.allow_leaky {
            return Err(format!("label leakage: {msg}. Drop the head or take the field out of the prompt; --allow-leaky-heads overrides"));
        }
        eprintln!("warning: label leakage allowed: {msg}");
    }

    let train_set = ds.split(Split::Train);
    // H3: validation and held-out data are disjoint from training or absent. Never the training set.
    let val_set = ds.split(Split::Val);
    let heldout_ds = if a.heldout.is_empty() { None } else { Some(data::load_with(&a.heldout, width, &lopts)?) };
    let (held, held_src): (Vec<_>, String) = match &heldout_ds {
        Some(h) => (h.examples.iter().collect(), format!("--heldout {:?}", a.heldout)),
        None => (ds.split(Split::Test), "test split".into()),
    };
    eprintln!(
        "records: {} train / {} val / {} test ({} split, {} groups); heads: {}; held-out: {held_src}",
        train_set.len(), val_set.len(), ds.split(Split::Test).len(), ds.stats.split_mode, ds.stats.groups,
        ds.heads.iter().map(|h| h.name.as_str()).collect::<Vec<_>>().join(",")
    );

    let store = Store::new(&a.out).map_err(|x| x.to_string())?;
    let current = store.current();

    let vm = VarMap::new();
    let model = Head::new(cfg.clone(), VarBuilder::from_varmap(&vm, DType::F32, &dev)).map_err(e)?;
    heart_decision_trainer::train::seed_init(&vm, a.seed).map_err(e)?;

    // H8: continue from the promoted model when it has the same shape.
    let mut init_from: Option<String> = None;
    if a.init_from == "auto" {
        match &current {
            Some(c) if same_shape(&c.config, &cfg) => {
                let tmp = store.dir.join(".init.safetensors");
                std::fs::write(&tmp, &c.weights).map_err(|x| x.to_string())?;
                let mut vm_mut = vm.clone();
                let loaded = vm_mut.load(&tmp).map_err(e);
                let _ = std::fs::remove_file(&tmp);
                loaded?;
                init_from = Some(c.version.clone());
            }
            Some(_) => eprintln!("init-from: promoted model has a different shape; training from random init"),
            None => {}
        }
    }
    let lr = if init_from.is_some() { a.lr * a.ft_lr_scale } else { a.lr };
    eprintln!(
        "preset {}: {} parameters; seed {}; {}",
        a.preset, model.param_count(), a.seed,
        match &init_from { Some(v) => format!("fine-tuning {v} at lr {lr}"), None => "random init".into() }
    );
    let opts = TrainOpts { epochs: a.epochs, batch: a.batch, lr, warmup_steps: a.warmup, max_steps: a.max_steps, seed: a.seed };
    let history = train(&model, &vm, &train_set, &ds.heads, width, &opts, &dev, |i, l| eprintln!("epoch {:>3}  loss {l:.4}", i + 1)).map_err(e)?;

    let mut model = model;
    let fit = if val_set.is_empty() {
        eprintln!("no validation split: tau is not fitted (abstain below certainty)");
        TauFit { tau: 1.0, met: false, target_accuracy: a.target_acc, coverage: 0.0, selective_accuracy: 0.0 }
    } else {
        let v = evaluate_with(&model, &val_set, width, 0.0, None, &dev).map_err(e)?;
        fit_tau(&v.confs, a.target_acc)
    };
    model.cfg.tau = fit.tau;
    cfg.tau = fit.tau;
    eprintln!("tau {:.4} (met={}, coverage {:.2}, selective accuracy {:.3})", fit.tau, fit.met, fit.coverage, fit.selective_accuracy);
    let val_metrics = if val_set.is_empty() { Default::default() } else { evaluate_with(&model, &val_set, width, fit.tau, None, &dev).map_err(e)?.metrics };

    // Persist weights first: the version is derived from their bytes.
    let tmp = store.dir.join(".candidate.safetensors");
    vm.save(&tmp).map_err(e)?;
    let bytes = std::fs::read(&tmp).map_err(|x| x.to_string())?;
    let version = version_for(&bytes);
    cfg.model_version = version.clone();
    cfg.dataset_manifest_hash = ds.manifest_hash();
    cfg.policy_bundle_hash = a.policy.clone();
    model.cfg = cfg.clone();

    let gopts = GateOpts { min_heldout: a.min_heldout, min_margin: a.min_margin, bootstrap: 2000, seed: a.seed };
    let (held_metrics, gate) = if held.is_empty() {
        eprintln!("no disjoint held-out data: trained but refused");
        (Default::default(), decide(&Default::default(), fit.met, 0, None, &gopts))
    } else {
        let cand = evaluate_with(&model, &held, width, fit.tau, None, &dev).map_err(e)?;
        match &current {
            None => {
                let g = decide(&cand.metrics, fit.met, held.len(), None, &gopts);
                (cand.metrics, g)
            }
            Some(c) => {
                // Same rows, re-encoded at the current model's own width, on the heads and
                // options both models have (H9).
                let cm = Head::from_safetensors(c.config.clone(), &c.weights, &dev).map_err(e)?;
                let w = cm.max_input_len();
                let cds = match &heldout_ds {
                    Some(_) => data::load_with(&a.heldout, w, &lopts)?,
                    None => data::load_with(&a.data, w, &lopts)?,
                };
                let set: Vec<_> = match &heldout_ds {
                    Some(_) => cds.examples.iter().collect(),
                    None => cds.split(Split::Test),
                };
                let restrict = shared_heads(&cfg, &c.config);
                let cur = evaluate_with(&cm, &set, w, c.config.tau, Some(&restrict), &dev).map_err(e)?;
                let cand_r = evaluate_with(&model, &held, width, fit.tau, Some(&restrict), &dev).map_err(e)?;
                let cmp = Comparison { current_version: &c.version, heads: restrict.keys().cloned().collect(), candidate: &cand_r, current: &cur };
                let g = decide(&cand.metrics, fit.met, held.len(), Some(cmp), &gopts);
                (cand.metrics, g)
            }
        }
    };

    let report = Report {
        version: &version,
        sha256: data::hex(&<sha2::Sha256 as sha2::Digest>::digest(&bytes)),
        size: bytes.len(),
        parameters: model.param_count(),
        epochs_loss: &history,
        validation: &val_metrics,
        heldout: &held_metrics,
        heldout_source: &held_src,
        tau_fit: &fit,
        gate: &gate,
        seed: a.seed,
        init_from: init_from.as_deref(),
        load_stats: &ds.stats,
        dataset_manifest_hash: &cfg.dataset_manifest_hash,
        datasets: &ds.files,
    };
    store.write(&cfg, &tmp, &report, gate.promoted).map_err(|x| x.to_string())?;
    let _ = std::fs::remove_file(&tmp);
    eprintln!(
        "{version}: held-out score {:.4} vs current {} -> {} ({})",
        gate.candidate_score,
        gate.current_score.map(|s| format!("{s:.4}")).unwrap_or_else(|| "none".into()),
        if gate.promoted { "PROMOTED" } else { "NOT promoted" },
        gate.reason
    );
    Ok(gate.promoted)
}

const CONVERT_USAGE: &str = "heart-decision-trainer convert --episodes <episodes.jsonl> --out <train.jsonl> [--soft <teacher.jsonl>] [--soft-heads a,b]

  Converts verified SupervisionEpisode v1 records to head-train/v1. --soft lines are
  {\"episode_id\":..., \"labels\":{\"action\":{\"request_review\":0.7,...}}} (Clef teacher distributions).
  --soft-heads: heads the teacher may label (default next_phase,agent). `action`
  (which carries abstain) stays outcome-labelled unless listed here explicitly. Teacher
  distributions are written under `soft` (train split only), never over the ground-truth labels.
  --emit-input-heads: also emit `intent` and `risk` labels. They are fields of the prompt, so the
  trainer refuses to train on them (label leakage) unless told otherwise.";

fn convert_cmd() -> Result<(), String> {
    let (mut episodes, mut out, mut soft_path, mut heads) = (None, None, None, None::<String>);
    let mut emit_input = false;
    let mut it = std::env::args().skip(2);
    while let Some(k) = it.next() {
        if k == "--emit-input-heads" {
            emit_input = true;
            continue;
        }
        let v = it.next().ok_or_else(|| format!("{k} needs a value"))?;
        match k.as_str() {
            "--episodes" => episodes = Some(v),
            "--out" => out = Some(v),
            "--soft" => soft_path = Some(v),
            "--soft-heads" => heads = Some(v),
            _ => return Err(format!("unknown argument {k}\n{CONVERT_USAGE}")),
        }
    }
    let (Some(episodes), Some(out)) = (episodes, out) else { return Err(CONVERT_USAGE.into()) };
    let mut soft = std::collections::BTreeMap::new();
    if let Some(p) = soft_path {
        for (i, line) in std::fs::read_to_string(&p).map_err(|e| format!("{p}: {e}"))?.lines().enumerate().filter(|(_, l)| !l.trim().is_empty()) {
            let v: serde_json::Value = serde_json::from_str(line).map_err(|e| format!("{p}:{}: {e}", i + 1))?;
            let (Some(id), Some(labels)) = (v["episode_id"].as_str(), v["labels"].as_object()) else {
                return Err(format!("{p}:{}: need episode_id and labels", i + 1));
            };
            soft.insert(id.to_string(), labels.clone());
        }
    }
    let input = std::fs::read_to_string(&episodes).map_err(|e| format!("{episodes}: {e}"))?;
    let allowed: Vec<&str> = match &heads {
        Some(h) => h.split(',').map(str::trim).filter(|x| !x.is_empty()).collect(),
        None => heart_decision_trainer::episodes::DEFAULT_SOFT_HEADS.to_vec(),
    };
    let (records, rep) = heart_decision_trainer::episodes::convert_opts(&input, &soft, &allowed, emit_input);
    let body: Vec<String> = records.iter().map(|r| r.to_string()).collect();
    std::fs::write(&out, body.join("\n") + "\n").map_err(|e| format!("{out}: {e}"))?;
    eprintln!("{rep:?}");
    if rep.written == 0 {
        return Err("no verified episodes: nothing to train on".into());
    }
    Ok(())
}

fn main() -> ExitCode {
    let first = std::env::args().nth(1);
    let sub = first.as_deref();
    if sub == Some("convert") {
        return match convert_cmd() {
            Ok(()) => ExitCode::SUCCESS,
            Err(m) => {
                eprintln!("{m}");
                ExitCode::FAILURE
            }
        };
    }
    if sub == Some("probe") {
        return match probe_cmd() {
            Ok(()) => ExitCode::SUCCESS,
            Err(m) => {
                eprintln!("{m}");
                ExitCode::FAILURE
            }
        };
    }
    match parse().and_then(run) {
        Ok(true) => ExitCode::SUCCESS,
        Ok(false) => ExitCode::from(3),
        Err(m) => {
            eprintln!("{m}");
            ExitCode::FAILURE
        }
    }
}

const PROBE_USAGE: &str = "heart-decision-trainer probe [--task a|b] [--preset tiny] [--epochs N] [--lr X] [--seed N] [--n-train N] [--n-test N]

  Task 4.4 kill-gate learnability probe. Trains a choice-set head on synthetic data
  (no gold) and checks it generalises to UNSEEN choice strings, clearly above the base
  rate, on both tasks:
    a  routing:    question topic -> keyword -> the candidate containing that keyword
    b  evidence:   question entity -> the evidence sentence mentioning that entity
  Train and test candidate pools are disjoint surface forms, so position/memorisation
  shortcuts stay at the base rate. Default runs both tasks.";

/// Trains and evaluates one synthetic task; returns (train_acc, train_base, test_acc, test_base).
fn run_task(task: &str, preset: &str, epochs: usize, seed: u64, lr: f64, n_train: usize, n_test: usize) -> Result<(f32, f32, f32, f32), String> {
    let head_name = if task == "a" { "route" } else { "evidence" };
    let sets = vec![SetHead { name: head_name.into(), kind: SetKind::Choice, max_choices: 16 }];
    let mut cfg = preset_v2(preset, vec![], sets).map_err(|e| e.to_string())?;
    // "Preset tiny, seq 512": tiny dims, a long enough sequence for several candidates. The
    // probe keeps it shorter so the CPU run finishes quickly; the gate does not depend on it.
    cfg.max_seq_len = 256;
    cfg.validate().map_err(|e| e.to_string())?;
    let width = cfg.max_seq_len - cfg.summary_slots;
    let max_choices = cfg.choice_set_heads[0].max_choices;

    let dev = Device::Cpu;
    let vm = VarMap::new();
    let model = Head::new(cfg.clone(), VarBuilder::from_varmap(&vm, DType::F32, &dev)).map_err(|e| e.to_string())?;
    seed_init(&vm, seed).map_err(|e| e.to_string())?;

    let all = synth::generate(task, n_train, n_test, seed);
    let train: Vec<SetExample> = all.iter().filter(|e| e.split == Split::Train).cloned().collect();
    let test: Vec<SetExample> = all.iter().filter(|e| e.split == Split::Test).cloned().collect();
    eprintln!("task {task}: {n_train} train / {n_test} test set questions; width {width}; head {head_name}");

    let norm_of = |vm: &VarMap, name: &str| -> f32 {
        let d = vm.data().lock().unwrap();
        d.get(name).map(|t| t.sqr().unwrap().sum_all().unwrap().to_scalar::<f32>().unwrap().sqrt()).unwrap_or(0.0)
    };
    let w0 = norm_of(&vm, "set.w");

    let opts = SetTrainOpts { epochs, lr, warmup_steps: 50, max_steps: None, seed };
    let _hist = train_set(&model, &vm, &train, 0, max_choices, width, &opts, &dev, |i, l| {
        eprintln!("  task {task} epoch {:>3}  loss {l:.4}", i + 1);
    })
    .map_err(|e| e.to_string())?;

    let (tr_acc, tr_base) = evaluate_set(&model, &train, 0, max_choices, width, &dev).map_err(|e| e.to_string())?;
    let (te_acc, te_base) = evaluate_set(&model, &test, 0, max_choices, width, &dev).map_err(|e| e.to_string())?;
    let w1 = norm_of(&vm, "set.w");
    eprintln!("  [diag] set.w norm before train {w0:.4}, after {w1:.4}");
    Ok((tr_acc, tr_base, te_acc, te_base))
}

fn probe_cmd() -> Result<(), String> {
    let (mut task, mut preset, mut epochs, mut seed, mut lr) = ("both".to_string(), "tiny".to_string(), 60usize, 1u64, 3e-4);
    let (mut n_train, mut n_test) = (120usize, 60usize);
    let mut it = std::env::args().skip(2);
    while let Some(k) = it.next() {
        let mut v = || it.next().ok_or_else(|| format!("{k} needs a value"));
        let num = |s: String| s.parse::<f64>().map_err(|e| format!("{k}: {e}"));
        match k.as_str() {
            "--task" => task = v()?,
            "--preset" => preset = v()?,
            "--epochs" => epochs = num(v()?)?.round() as usize,
            "--seed" => seed = num(v()?)?.round() as u64,
            "--lr" => lr = num(v()?)?,
            "--n-train" => n_train = num(v()?)?.round() as usize,
            "--n-test" => n_test = num(v()?)?.round() as usize,
            "-h" | "--help" => return Err(PROBE_USAGE.into()),
            other => return Err(format!("unknown argument {other}\n{PROBE_USAGE}")),
        }
    }
    let tasks: Vec<&str> = if task == "both" { vec!["a", "b"] } else { vec![task.as_str()] };
    let mut all_pass = true;
    for t in tasks {
        let (tr_acc, tr_base, te_acc, te_base) = run_task(t, &preset, epochs, seed, lr, n_train, n_test)?;
        let pass = te_acc > te_base * 1.5 && te_acc > te_base + 0.2;
        all_pass &= pass;
        eprintln!(
            "task {t}: train acc {tr_acc:.3} (base {tr_base:.3}) | test acc {te_acc:.3} (base {te_base:.3}) -> {}",
            if pass { "PASSED" } else { "FAILED" }
        );
    }
    if all_pass {
        eprintln!("KILL GATE: PASSED — set heads learn the rule and generalise to unseen choice strings");
        Ok(())
    } else {
        eprintln!("KILL GATE: FAILED — only memorises or leans on position; stop (do not proceed to 4.5+)");
        Err("kill gate failed".into())
    }
}
