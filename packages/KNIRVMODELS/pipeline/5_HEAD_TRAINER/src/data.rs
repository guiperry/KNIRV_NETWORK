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
//! leak into evaluation. A record without `split` is placed by hashing its group.

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
}

#[derive(Clone, Debug)]
pub enum Label {
    /// Dense probabilities over the head's options (hard labels are one-hot).
    Choice(Vec<f32>),
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
}

pub struct Dataset {
    pub heads: Vec<DecisionHead>,
    pub examples: Vec<Example>,
    pub files: Vec<FileDigest>,
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

fn split_of(r: &RawRecord) -> Result<Split, String> {
    match r.split.as_deref() {
        Some("train") => Ok(Split::Train),
        Some("val") => Ok(Split::Val),
        Some("test") => Ok(Split::Test),
        Some(other) => Err(format!("record {}: unknown split {other:?}", r.id)),
        None => {
            let key = r.group.as_deref().unwrap_or(&r.id);
            let h = Sha256::digest(key.as_bytes());
            Ok(match u16::from_be_bytes([h[0], h[1]]) % 100 {
                0..=79 => Split::Train,
                80..=89 => Split::Val,
                _ => Split::Test,
            })
        }
    }
}

/// Infers each head from the labels present. Strings and objects are `choice`,
/// numbers are `score`; mixing kinds for one head is an error.
fn infer_heads(records: &[RawRecord]) -> Result<Vec<DecisionHead>, String> {
    let mut kinds: BTreeMap<&str, HeadKind> = BTreeMap::new();
    let mut options: BTreeMap<&str, BTreeSet<String>> = BTreeMap::new();
    for r in records {
        for (name, v) in &r.labels {
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
                _ => {}
            }
        }
    }
    kinds
        .into_iter()
        .filter_map(|(name, kind)| match kind {
            HeadKind::Score => Some(Ok(DecisionHead { name: name.into(), kind, options: vec![] })),
            HeadKind::Choice => {
                let opts: Vec<String> = match known_options(name) {
                    Some(known) => {
                        let known: Vec<String> = known.into_iter().map(String::from).collect();
                        if let Some(bad) = options[name].iter().find(|o| !known.contains(o)) {
                            return Some(Err(format!("head {name:?}: option {bad:?} not in the fixed vocabulary {known:?}")));
                        }
                        known
                    }
                    None => options[name].iter().cloned().collect(),
                };
                if opts.len() < 2 {
                    // Nothing to learn from a one-valued head; say so instead of failing the run.
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
            let mut p = vec![0f32; head.options.len()];
            match v {
                Value::String(s) => p[head.options.iter().position(|o| o == s).expect("inferred")] = 1.0,
                Value::Object(m) => {
                    for (k, x) in m {
                        let i = head.options.iter().position(|o| o == k).expect("inferred");
                        p[i] = x.as_f64().filter(|x| *x >= 0.0).ok_or_else(|| format!("record {rec}: bad probability for {k}"))? as f32;
                    }
                }
                Value::Array(a) => {
                    if a.len() != p.len() {
                        return Err(format!("record {rec}: {} array has {} entries, head has {} options", head.name, a.len(), p.len()));
                    }
                    for (i, x) in a.iter().enumerate() {
                        p[i] = x.as_f64().filter(|x| *x >= 0.0).ok_or_else(|| format!("record {rec}: bad probability"))? as f32;
                    }
                }
                _ => unreachable!(),
            }
            let sum: f32 = p.iter().sum();
            if sum <= 0.0 {
                return Err(format!("record {rec}: {} probabilities sum to 0", head.name));
            }
            p.iter_mut().for_each(|x| *x /= sum);
            Ok(Label::Choice(p))
        }
    }
}

/// Reads and encodes JSONL files. `width` = model `max_seq_len - summary_slots`.
pub fn load(paths: &[String], width: usize) -> Result<Dataset, String> {
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
    let heads = infer_heads(&records)?;
    let mut examples = Vec::with_capacity(records.len());
    for r in &records {
        let mut labels = BTreeMap::new();
        for (name, v) in &r.labels {
            let Some(head) = heads.iter().find(|h| &h.name == name) else { continue }; // untrainable head
            labels.insert(name.clone(), encode_label(&r.id, head, v)?);
        }
        examples.push(Example {
            id: r.id.clone(),
            split: split_of(r)?,
            input: pad_to(encode_text(&r.input, width), width),
            plan: r.plan.as_deref().map(|p| encode_plan(p, width)),
            labels,
        });
    }
    Ok(Dataset { heads, examples, files })
}
