//! `head-train/v1` JSONL records and their encoding.
//!
//! ```json
//! {"id":"ep-1","group":"repo-family","split":"train",
//!  "input":"redacted task text","plan":"optional plan text",
//!  "labels":{"action":"request_review",                       // hard
//!            "intent":{"review":0.8,"validate":0.2},           // soft (Clef teacher)
//!            "risk":0.62}}                                     // score in [0,1]
//! ```
//! Splits are by `group` (repo family), never by row, so near-duplicates cannot
//! leak into evaluation. A record without `split` is placed by hashing its group
//! when there are at least `min_groups` groups, otherwise by time (oldest 80%
//! train, then val, newest test): never all in one split (H3).
//!
//! Ground truth lives in `labels`. Teacher distributions live in `soft` (and
//! `teacher_plan`) and are used for the TRAIN split only (H2).

use knirv_head::tokenizer::{encode_plan, encode_text, pad_to};
use knirv_head::{DecisionHead, HeadKind};
use serde::Deserialize;
use serde_json::Value;
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet};

/// Fixed vocabularies the supervisor already enforces (advisor `allowedActions`,
/// SupervisionEpisode label families, live sub-agents). Unlisted heads learn
/// their options from the data.
pub fn known_options(head: &str) -> Option<Vec<&'static str>> {
    match head {
        "action" => Some(vec!["allow", "request_review", "request_evidence", "ask_operator", "recover"]),
        "intent" => Some(vec!["plan", "investigate", "develop", "review", "validate", "recover", "ask_operator"]),
        "agent" => Some(vec!["claude", "codex", "opencode", "hermes"]),
        _ => None,
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum Split {
    Train,
    Val,
    Test,
}

#[derive(Deserialize, Debug)]
pub struct RawRecord {
    pub id: String,
    #[serde(default)]
    pub group: Option<String>,
    #[serde(default)]
    pub split: Option<String>,
    pub input: String,
    #[serde(default)]
    pub plan: Option<String>,
    #[serde(default)]
    pub labels: BTreeMap<String, Value>,
    /// Teacher distributions per head. Train split only.
    #[serde(default)]
    pub soft: BTreeMap<String, Value>,
    /// Distilled plan text. Train split only.
    #[serde(default)]
    pub teacher_plan: Option<String>,
    /// RFC 3339 time of the episode; orders the time split.
    #[serde(default)]
    pub ts: Option<String>,
}

#[derive(Clone, Debug)]
pub enum Label {
    /// Probabilities keyed by option NAME (hard labels are one-hot), so a label
    /// means the same thing against any model's option order (H9).
    Choice(BTreeMap<String, f32>),
    Score(f32),
}

#[derive(Clone, Debug)]
pub struct Example {
    pub id: String,
    pub split: Split,
    pub input: Vec<u32>,
    /// Plan target at the same fixed width; `None` = no plan supervision.
    pub plan: Option<Vec<u32>>,
    pub labels: BTreeMap<String, Label>,
    /// The input did not fit the model width and was cut (H5).
    pub truncated: bool,
}

#[derive(Clone, Debug, Default, serde::Serialize)]
pub struct LoadStats {
    /// `soft` / `teacher_plan` entries ignored because the record is val/test.
    pub soft_ignored: usize,
    pub truncated: usize,
    pub groups: usize,
    /// `explicit`, `group-hash` or `time`.
    pub split_mode: String,
}

#[derive(Clone, Debug)]
pub struct LoadOpts {
    /// Fewer distinct groups than this -> split by time instead of by group.
    pub min_groups: usize,
}

impl Default for LoadOpts {
    fn default() -> Self {
        Self { min_groups: 5 }
    }
}

pub struct Dataset {
    pub heads: Vec<DecisionHead>,
    pub examples: Vec<Example>,
    pub files: Vec<FileDigest>,
    pub stats: LoadStats,
}

#[derive(Clone, Debug, serde::Serialize)]
pub struct FileDigest {
    pub path: String,
    pub sha256: String,
    pub records: usize,
}

impl Dataset {
    pub fn split(&self, s: Split) -> Vec<&Example> {
        self.examples.iter().filter(|e| e.split == s).collect()
    }

    /// sha256 over the sorted per-file digests: changes iff any input byte changes.
    pub fn manifest_hash(&self) -> String {
        let mut digests: Vec<&str> = self.files.iter().map(|f| f.sha256.as_str()).collect();
        digests.sort_unstable();
        hex(&Sha256::digest(digests.join("\n").as_bytes()))
    }
}

pub fn hex(bytes: &[u8]) -> String {
    bytes.iter().map(|b| format!("{b:02x}")).collect()
}

fn explicit_split(r: &RawRecord) -> Result<Option<Split>, String> {
    match r.split.as_deref() {
        Some("train") => Ok(Some(Split::Train)),
        Some("val") => Ok(Some(Split::Val)),
        Some("test") => Ok(Some(Split::Test)),
        Some(other) => Err(format!("record {}: unknown split {other:?}", r.id)),
        None => Ok(None),
    }
}

fn hash_split(key: &str) -> Split {
    let h = Sha256::digest(key.as_bytes());
    match u16::from_be_bytes([h[0], h[1]]) % 100 {
        0..=79 => Split::Train,
        80..=89 => Split::Val,
        _ => Split::Test,
    }
}

/// Assigns a split to every record. Explicit `split` wins. The rest are split
/// by group hash when there are enough groups for that to mean anything, else
/// by time order.
fn assign_splits(records: &[RawRecord], min_groups: usize) -> Result<(Vec<Split>, usize, String), String> {
    let mut out = vec![Split::Train; records.len()];
    let mut implicit: Vec<usize> = Vec::new();
    for (i, r) in records.iter().enumerate() {
        match explicit_split(r)? {
            Some(sp) => out[i] = sp,
            None => implicit.push(i),
        }
    }
    let groups: BTreeSet<&str> = implicit.iter().map(|&i| records[i].group.as_deref().unwrap_or(&records[i].id)).collect();
    let n_groups = groups.len();
    if implicit.is_empty() {
        return Ok((out, n_groups, "explicit".into()));
    }
    if n_groups >= min_groups {
        for &i in &implicit {
            out[i] = hash_split(records[i].group.as_deref().unwrap_or(&records[i].id));
        }
        return Ok((out, n_groups, "group-hash".into()));
    }
    // Too few groups for a group split: order by time (then file order).
    let mut order = implicit.clone();
    order.sort_by(|&a, &b| records[a].ts.cmp(&records[b].ts).then(a.cmp(&b)));
    let n = order.len();
    let (nval, ntest) = if n >= 3 { ((n / 10).max(1), (n / 10).max(1)) } else { (0, 0) };
    let ntrain = n - nval - ntest;
    for (k, &i) in order.iter().enumerate() {
        out[i] = if k < ntrain { Split::Train } else if k < ntrain + nval { Split::Val } else { Split::Test };
    }
    Ok((out, n_groups, "time".into()))
}

/// Infers each head from the labels present. Strings and objects are `choice`,
/// numbers are `score`; mixing kinds for one head is an error.
fn infer_heads(records: &[RawRecord], train_ids: &BTreeSet<&str>) -> Result<Vec<DecisionHead>, String> {
    let mut kinds: BTreeMap<&str, HeadKind> = BTreeMap::new();
    let mut options: BTreeMap<&str, BTreeSet<String>> = BTreeMap::new();
    let mut array_len: BTreeMap<&str, usize> = BTreeMap::new();
    for r in records {
        // Teacher distributions only count for rows that will train on them.
        let soft = r.soft.iter().filter(|_| train_ids.contains(r.id.as_str()));
        for (name, v) in r.labels.iter().chain(soft) {
            let kind = match v {
                Value::Number(_) => HeadKind::Score,
                Value::String(_) | Value::Object(_) | Value::Array(_) => HeadKind::Choice,
                _ => return Err(format!("record {}: label {name:?} must be string, object, array or number", r.id)),
            };
            if let Some(prev) = kinds.insert(name, kind) {
                if prev != kind {
                    return Err(format!("label {name:?} mixes choice and score values (record {})", r.id));
                }
            }
            let set = options.entry(name).or_default();
            match v {
                Value::String(s) => {
                    set.insert(s.clone());
                }
                Value::Object(m) => set.extend(m.keys().cloned()),
                Value::Array(a) => {
                    // H12: an array-only head has no option names; use positions.
                    if let Some(prev) = array_len.insert(name, a.len()) {
                        if prev != a.len() {
                            return Err(format!("head {name:?}: array labels of length {prev} and {} (record {})", a.len(), r.id));
                        }
                    }
                }
                _ => {}
            }
        }
    }
    kinds
        .into_iter()
        .filter_map(|(name, kind)| match kind {
            HeadKind::Score => Some(Ok(DecisionHead { name: name.into(), kind, options: vec![] })),
            HeadKind::Choice => {
                let mut opts: Vec<String> = match known_options(name) {
                    Some(known) => {
                        let known: Vec<String> = known.into_iter().map(String::from).collect();
                        if let Some(bad) = options[name].iter().find(|o| !known.contains(o)) {
                            return Some(Err(format!("head {name:?}: option {bad:?} not in the fixed vocabulary {known:?}")));
                        }
                        known
                    }
                    None => options[name].iter().cloned().collect(),
                };
                if opts.is_empty() {
                    if let Some(&n) = array_len.get(name) {
                        opts = (0..n).map(|i| i.to_string()).collect();
                    }
                }
                if opts.len() < 2 {
                    eprintln!("warning: head {name:?} has a single value {opts:?}; not trained");
                    return None;
                }
                Some(Ok(DecisionHead { name: name.into(), kind, options: opts }))
            }
        })
        .collect()
}

fn encode_label(rec: &str, head: &DecisionHead, v: &Value) -> Result<Label, String> {
    match head.kind {
        HeadKind::Score => {
            let x = v.as_f64().ok_or_else(|| format!("record {rec}: {} must be a number", head.name))? as f32;
            if !(0.0..=1.0).contains(&x) {
                return Err(format!("record {rec}: score {} = {x} outside [0,1]", head.name));
            }
            Ok(Label::Score(x))
        }
        HeadKind::Choice => {
            let mut p: BTreeMap<String, f32> = BTreeMap::new();
            let prob = |x: &Value, what: &str| x.as_f64().filter(|x| *x >= 0.0).map(|x| x as f32).ok_or_else(|| format!("record {rec}: bad probability for {what}"));
            match v {
                Value::String(s) => {
                    p.insert(s.clone(), 1.0);
                }
                Value::Object(m) => {
                    for (k, x) in m {
                        p.insert(k.clone(), prob(x, k)?);
                    }
                }
                Value::Array(a) => {
                    if a.len() != head.options.len() {
                        return Err(format!("record {rec}: {} array has {} entries, head has {} options", head.name, a.len(), head.options.len()));
                    }
                    for (name, x) in head.options.iter().zip(a) {
                        p.insert(name.clone(), prob(x, name)?);
                    }
                }
                _ => unreachable!(),
            }
            let sum: f32 = p.values().sum();
            if sum <= 0.0 {
                return Err(format!("record {rec}: {} probabilities sum to 0", head.name));
            }
            p.values_mut().for_each(|x| *x /= sum);
            Ok(Label::Choice(p))
        }
    }
}

/// Heads whose label is spelled out in the input as `<head>:<option>` for at
/// least `threshold` of the rows that carry it. Such a head can copy its
/// answer instead of predicting it (H4).
pub fn leaky_heads(examples: &[Example], heads: &[DecisionHead], inputs: &BTreeMap<String, String>, threshold: f32) -> Vec<(String, f32)> {
    let mut out = Vec::new();
    for h in heads.iter().filter(|h| h.kind == HeadKind::Choice) {
        let (mut n, mut hit) = (0usize, 0usize);
        for e in examples {
            let Some(Label::Choice(p)) = e.labels.get(&h.name) else { continue };
            let Some(top) = p.iter().max_by(|a, b| a.1.total_cmp(b.1)).map(|x| x.0) else { continue };
            let Some(text) = inputs.get(&e.id) else { continue };
            n += 1;
            hit += text.contains(&format!("{}:{}", h.name, top)) as usize;
        }
        if n > 0 && hit as f32 / n as f32 >= threshold {
            out.push((h.name.clone(), hit as f32 / n as f32));
        }
    }
    out
}

/// Reads and encodes JSONL files. `width` = model `max_seq_len - summary_slots`.
pub fn load(paths: &[String], width: usize) -> Result<Dataset, String> {
    load_with(paths, width, &LoadOpts::default())
}

pub fn load_with(paths: &[String], width: usize, opts: &LoadOpts) -> Result<Dataset, String> {
    let mut records = Vec::new();
    let mut files = Vec::new();
    for path in paths {
        let bytes = std::fs::read(path).map_err(|e| format!("{path}: {e}"))?;
        let mut n = 0;
        for (i, line) in String::from_utf8_lossy(&bytes).lines().enumerate() {
            if line.trim().is_empty() {
                continue;
            }
            records.push(serde_json::from_str::<RawRecord>(line).map_err(|e| format!("{path}:{}: {e}", i + 1))?);
            n += 1;
        }
        files.push(FileDigest { path: path.clone(), sha256: hex(&Sha256::digest(&bytes)), records: n });
    }
    if records.is_empty() {
        return Err("no records".into());
    }
    let mut seen = BTreeSet::new();
    if let Some(r) = records.iter().find(|r| !seen.insert(r.id.clone())) {
        return Err(format!("duplicate record id {:?}", r.id));
    }
    let (splits, groups, split_mode) = assign_splits(&records, opts.min_groups)?;
    let train_ids: BTreeSet<&str> = records.iter().zip(&splits).filter(|(_, s)| **s == Split::Train).map(|(r, _)| r.id.as_str()).collect();
    let heads = infer_heads(&records, &train_ids)?;

    let mut stats = LoadStats { groups, split_mode, ..Default::default() };
    let mut examples = Vec::with_capacity(records.len());
    for (r, &split) in records.iter().zip(&splits) {
        let mut labels = BTreeMap::new();
        for (name, v) in &r.labels {
            let Some(head) = heads.iter().find(|h| &h.name == name) else { continue }; // untrainable head
            labels.insert(name.clone(), encode_label(&r.id, head, v)?);
        }
        let mut plan_text = r.plan.as_deref();
        if split == Split::Train {
            for (name, v) in &r.soft {
                if let Some(head) = heads.iter().find(|h| &h.name == name) {
                    labels.insert(name.clone(), encode_label(&r.id, head, v)?);
                }
            }
            if plan_text.is_none() {
                plan_text = r.teacher_plan.as_deref();
            }
        } else {
            stats.soft_ignored += r.soft.len() + r.teacher_plan.is_some() as usize;
        }
        let truncated = 1 + r.input.len() > width;
        stats.truncated += truncated as usize;
        examples.push(Example {
            id: r.id.clone(),
            split,
            input: pad_to(encode_text(&r.input, width), width),
            plan: plan_text.map(|p| encode_plan(p, width)),
            labels,
            truncated,
        });
    }
    if stats.soft_ignored > 0 {
        eprintln!("note: ignored {} teacher label(s) on val/test rows (teacher labels train only)", stats.soft_ignored);
    }
    Ok(Dataset { heads, examples, files, stats })
}

/// Raw input text per record id, for [`leaky_heads`].
pub fn input_texts(paths: &[String]) -> Result<BTreeMap<String, String>, String> {
    let mut m = BTreeMap::new();
    for path in paths {
        let text = std::fs::read_to_string(path).map_err(|e| format!("{path}: {e}"))?;
        for (i, line) in text.lines().enumerate().filter(|(_, l)| !l.trim().is_empty()) {
            let r: RawRecord = serde_json::from_str(line).map_err(|e| format!("{path}:{}: {e}", i + 1))?;
            m.insert(r.id, r.input);
        }
    }
    Ok(m)
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    fn write(name: &str, rows: Vec<Value>) -> String {
        let dir = std::env::temp_dir().join(format!("head-data-{}-{name}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let p = dir.join("t.jsonl");
        std::fs::write(&p, rows.iter().map(|r| r.to_string()).collect::<Vec<_>>().join("\n")).unwrap();
        p.to_str().unwrap().to_string()
    }

    fn rows(n: usize, groups: usize) -> Vec<Value> {
        (0..n)
            .map(|i| json!({"id": format!("e{i}"), "group": format!("g{}", i % groups), "ts": format!("2026-01-01T00:{:02}:{:02}Z", i / 60, i % 60),
                "input": format!("request {i}"), "labels": {"action": if i % 2 == 0 {"allow"} else {"ask_operator"}}}))
            .collect()
    }

    fn counts(ds: &Dataset) -> (usize, usize, usize) {
        (ds.split(Split::Train).len(), ds.split(Split::Val).len(), ds.split(Split::Test).len())
    }

    #[test]
    fn teacher_labels_never_reach_val_or_test() {
        let mut r = rows(40, 40);
        for x in r.iter_mut() {
            x["soft"] = json!({"action": {"allow": 0.1, "ask_operator": 0.9}});
        }
        let ds = load(&[write("soft", r)], 64).unwrap();
        let (mut trained_soft, mut eval_soft) = (0, 0);
        for e in &ds.examples {
            let Some(Label::Choice(p)) = e.labels.get("action") else { continue };
            let soft = p.values().any(|&v| (v - 0.9).abs() < 1e-6);
            match e.split {
                Split::Train => trained_soft += soft as usize,
                _ => eval_soft += soft as usize,
            }
        }
        assert!(trained_soft > 0, "train rows take the teacher distribution");
        assert_eq!(eval_soft, 0, "H2: val/test rows keep the hard ground truth");
        assert!(ds.stats.soft_ignored > 0);
    }

    #[test]
    fn many_groups_split_by_group_hash() {
        let ds = load(&[write("many", rows(200, 40))], 64).unwrap();
        assert_eq!(ds.stats.split_mode, "group-hash");
        let (tr, va, te) = counts(&ds);
        assert!(tr > 0 && va > 0 && te > 0, "{:?}", (tr, va, te));
        // a group never straddles splits
        let mut by_group: BTreeMap<String, Split> = BTreeMap::new();
        for (i, e) in ds.examples.iter().enumerate() {
            let g = format!("g{}", i % 40);
            assert_eq!(*by_group.entry(g).or_insert(e.split), e.split);
        }
    }

    #[test]
    fn one_group_splits_by_time_never_all_in_one_split() {
        let ds = load(&[write("one", rows(100, 1))], 64).unwrap();
        assert_eq!(ds.stats.split_mode, "time");
        assert_eq!(counts(&ds), (80, 10, 10));
        // oldest train, newest test
        assert_eq!(ds.examples[0].split, Split::Train);
        assert_eq!(ds.examples[99].split, Split::Test);
        assert_eq!(ds.examples[85].split, Split::Val);
    }

    #[test]
    fn time_split_uses_ts_not_file_order() {
        let mut r = rows(20, 1);
        r.reverse(); // file order is newest first
        let ds = load(&[write("rev", r)], 64).unwrap();
        let newest = ds.examples.iter().find(|e| e.id == "e19").unwrap();
        let oldest = ds.examples.iter().find(|e| e.id == "e0").unwrap();
        assert_eq!((newest.split, oldest.split), (Split::Test, Split::Train));
    }

    #[test]
    fn explicit_splits_are_untouched() {
        let mut r = rows(10, 1);
        for x in r.iter_mut() {
            x["split"] = json!("test");
        }
        let ds = load(&[write("explicit", r)], 64).unwrap();
        assert_eq!(counts(&ds), (0, 0, 10));
        assert_eq!(ds.stats.split_mode, "explicit");
    }

    #[test]
    fn truncation_is_counted() {
        let mut r = rows(4, 4);
        r[0]["input"] = json!("x".repeat(500));
        let ds = load(&[write("trunc", r)], 64).unwrap();
        assert_eq!(ds.stats.truncated, 1);
        assert!(ds.examples[0].truncated && !ds.examples[1].truncated);
    }

    #[test]
    fn labels_are_keyed_by_option_name_not_position() {
        let ds = load(&[write("names", rows(10, 10))], 64).unwrap();
        let Some(Label::Choice(p)) = ds.examples[1].labels.get("action") else { panic!() };
        assert_eq!(p.get("ask_operator"), Some(&1.0));
        // A head with a different option order and one extra option still reads the same label.
        let head = DecisionHead { name: "action".into(), kind: HeadKind::Choice, options: vec!["recover".into(), "ask_operator".into(), "allow".into()] };
        let b = crate::batch::make(&[&ds.examples[1]], &[head], 64, &candle_core::Device::Cpu).unwrap();
        assert_eq!(b.heads[0].target.to_vec2::<f32>().unwrap(), vec![vec![0.0, 1.0, 0.0]]);
        assert_eq!(b.heads[0].mask.to_vec1::<f32>().unwrap(), vec![1.0]);
        // A head that lacks the labelled option cannot score the row.
        let head = DecisionHead { name: "action".into(), kind: HeadKind::Choice, options: vec!["allow".into(), "recover".into()] };
        let b = crate::batch::make(&[&ds.examples[1]], &[head], 64, &candle_core::Device::Cpu).unwrap();
        assert_eq!(b.heads[0].mask.to_vec1::<f32>().unwrap(), vec![0.0]);
    }

    #[test]
    fn array_only_heads_get_positional_options() {
        let r: Vec<Value> = (0..10).map(|i| json!({"id": format!("e{i}"), "group": format!("g{i}"), "input": "x", "labels": {"colour": [0.7, 0.3, 0.0]}})).collect();
        let ds = load(&[write("array", r)], 64).unwrap();
        let h = ds.heads.iter().find(|h| h.name == "colour").expect("H12: head is no longer skipped");
        assert_eq!(h.options, vec!["0", "1", "2"]);
    }

    #[test]
    fn leakage_detector_flags_a_head_spelled_out_in_its_input() {
        let r: Vec<Value> = (0..20)
            .map(|i| {
                let intent = if i % 2 == 0 { "review" } else { "plan" };
                json!({"id": format!("e{i}"), "group": format!("g{i}"), "input": format!("phase:x intent:{intent} request:{i}"), "labels": {"intent": intent, "action": "allow"}})
            })
            .collect();
        let p = write("leak", r);
        let ds = load(&[p.clone()], 128).unwrap();
        let leaks = leaky_heads(&ds.examples, &ds.heads, &input_texts(&[p]).unwrap(), 0.9);
        assert_eq!(leaks.iter().map(|l| l.0.as_str()).collect::<Vec<_>>(), vec!["intent"]);
    }
}
