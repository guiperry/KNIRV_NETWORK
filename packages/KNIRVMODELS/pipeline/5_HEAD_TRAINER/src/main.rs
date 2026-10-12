use candle_core::{DType, Device};
use candle_nn::{VarBuilder, VarMap};
use head_trainer::data::{self, Split};
use head_trainer::eval::{evaluate, fit_tau};
use head_trainer::export::{decide, version_for, Report, Store};
use head_trainer::train::{train, TrainOpts};
use head_trainer::preset;
use knirv_head::Head;
use std::path::PathBuf;
use std::process::ExitCode;

const USAGE: &str = "head-trainer --data <train.jsonl>... --out <dir> [options]

  --data FILE         head-train/v1 JSONL (repeatable); split by `split` or by hash of `group`
  --heldout FILE      held-out evaluation set (repeatable); default: the data's test split
  --out DIR           writes DIR/head/ (config, safetensors, model_latest, metrics.json)
  --preset NAME       micro | tiny | small | base (default small)
  --epochs N          (default 20)        --batch N   (default 16)
  --lr X              peak learning rate (default 3e-4; diverges above ~3e-3)
  --warmup N          warmup steps (default 100)      --max-steps N
  --target-acc X      selective accuracy tau must reach on each choice head (default 0.95)
  --policy-bundle-hash H   recorded in the config
  --seed N
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
}

fn parse() -> Result<Args, String> {
    let mut a = Args {
        data: vec![], heldout: vec![], out: PathBuf::new(), preset: "small".into(), epochs: 20, batch: 16,
        lr: 3e-4, warmup: 100, max_steps: None, target_acc: 0.95, policy: String::new(), seed: 1,
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
    Ok(a)
}

fn run(a: Args) -> Result<bool, String> {
    let e = |x: candle_core::Error| x.to_string();
    let dev = Device::Cpu;

    // The data decides the heads; the preset decides the architecture.
    let probe = preset(&a.preset, vec![])
        .map(|c| c.max_seq_len - c.summary_slots)?;
    let ds = data::load(&a.data, probe)?;
    let mut cfg = preset(&a.preset, ds.heads.clone())?;
    let width = cfg.max_seq_len - cfg.summary_slots;

    let train_set = ds.split(Split::Train);
    let (val_set, val_src) = match ds.split(Split::Val) {
        v if !v.is_empty() => (v, "val split"),
        _ => (train_set.clone(), "TRAIN split (no validation data: tau is not honestly fitted)"),
    };
    let heldout_ds = if a.heldout.is_empty() { None } else { Some(data::load(&a.heldout, width)?) };
    let (held, held_src): (Vec<_>, String) = match &heldout_ds {
        Some(h) => (h.examples.iter().collect(), format!("--heldout {:?}", a.heldout)),
        None => match ds.split(Split::Test) {
            t if !t.is_empty() => (t, "test split".into()),
            _ => (val_set.clone(), "validation split (no test data)".into()),
        },
    };
    eprintln!(
        "records: {} train / {} val / {} test; heads: {}; held-out: {held_src}; validation: {val_src}",
        train_set.len(), ds.split(Split::Val).len(), ds.split(Split::Test).len(),
        ds.heads.iter().map(|h| h.name.as_str()).collect::<Vec<_>>().join(",")
    );

    let vm = VarMap::new();
    let model = Head::new(cfg.clone(), VarBuilder::from_varmap(&vm, DType::F32, &dev)).map_err(e)?;
    head_trainer::train::seed_init(&vm, a.seed).map_err(e)?;
    eprintln!("preset {}: {} parameters", a.preset, model.param_count());
    let opts = TrainOpts { epochs: a.epochs, batch: a.batch, lr: a.lr, warmup_steps: a.warmup, max_steps: a.max_steps, seed: a.seed };
    let history = train(&model, &vm, &train_set, &ds.heads, width, &opts, &dev, |i, l| eprintln!("epoch {:>3}  loss {l:.4}", i + 1)).map_err(e)?;

    let mut model = model;
    let (val_metrics, confs) = evaluate(&model, &val_set, width, &dev).map_err(e)?;
    let fit = fit_tau(&confs, a.target_acc);
    model.cfg.tau = fit.tau;
    cfg.tau = fit.tau;
    eprintln!("tau {:.4} (met={}, coverage {:.2}, selective accuracy {:.3})", fit.tau, fit.met, fit.coverage, fit.selective_accuracy);

    // Persist weights first: the version is derived from their bytes.
    let store = Store::new(&a.out).map_err(|x| x.to_string())?;
    let tmp = store.dir.join(".candidate.safetensors");
    vm.save(&tmp).map_err(e)?;
    let bytes = std::fs::read(&tmp).map_err(|x| x.to_string())?;
    let version = version_for(&bytes);
    cfg.model_version = version.clone();
    cfg.dataset_manifest_hash = ds.manifest_hash();
    cfg.policy_bundle_hash = a.policy.clone();
    model.cfg = cfg.clone();

    let (held_metrics, _) = evaluate(&model, &held, width, &dev).map_err(e)?;

    // Same held-out data, re-encoded at the current model's own width.
    let current = store.current();
    let cur_metrics = match &current {
        Some(c) => {
            let cm = Head::from_safetensors(c.config.clone(), &c.weights, &dev).map_err(e)?;
            let w = cm.max_input_len();
            let paths = if a.heldout.is_empty() { &a.data } else { &a.heldout };
            let cds = data::load(paths, w)?;
            let set: Vec<_> = if a.heldout.is_empty() {
                let t = cds.split(Split::Test);
                if t.is_empty() { cds.split(Split::Val) } else { t }
            } else {
                cds.examples.iter().collect()
            };
            Some(evaluate(&cm, &set, w, &dev).map_err(e)?.0)
        }
        None => None,
    };
    let gate = decide(&held_metrics, current.as_ref().zip(cur_metrics.as_ref()).map(|(c, m)| (c.version.as_str(), m)));

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

const CONVERT_USAGE: &str = "head-trainer convert --episodes <episodes.jsonl> --out <train.jsonl> [--soft <teacher.jsonl>] [--soft-heads a,b]

  Converts verified SupervisionEpisode v1 records to head-train/v1. --soft lines are
  {\"episode_id\":..., \"labels\":{\"action\":{\"request_review\":0.7,...}}} (Clef teacher distributions).
  --soft-heads: heads the teacher may label (default intent,next_phase,risk,agent). `action`
  (which carries abstain) stays outcome-labelled unless listed here explicitly.";

fn convert_cmd() -> Result<(), String> {
    let (mut episodes, mut out, mut soft_path, mut heads) = (None, None, None, None::<String>);
    let mut it = std::env::args().skip(2);
    while let Some(k) = it.next() {
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
    let (records, rep) = match &heads {
        Some(h) => {
            let h: Vec<&str> = h.split(',').map(str::trim).filter(|x| !x.is_empty()).collect();
            head_trainer::episodes::convert_masked(&input, &soft, &h)
        }
        None => head_trainer::episodes::convert(&input, &soft),
    };
    let body: Vec<String> = records.iter().map(|r| r.to_string()).collect();
    std::fs::write(&out, body.join("\n") + "\n").map_err(|e| format!("{out}: {e}"))?;
    eprintln!("{rep:?}");
    if rep.written == 0 {
        return Err("no verified episodes: nothing to train on".into());
    }
    Ok(())
}

fn main() -> ExitCode {
    if std::env::args().nth(1).as_deref() == Some("convert") {
        return match convert_cmd() {
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
