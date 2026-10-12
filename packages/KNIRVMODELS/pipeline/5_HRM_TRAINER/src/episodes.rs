//! `SupervisionEpisode v1` (supervisor_model.md) -> `hrm-train/v1`.
//!
//! Only episodes whose decision was *verified* become action/phase labels:
//! `outcome.verified && !outcome.human_override`, with a non-abstaining decision.
//! A shadow advisor's own advice is never a label. The prompt text must equal the
//! server's `buildAdvisePrompt` (backend_server/internal/web/hrm_advise.go).

use regex::Regex;
use serde::Deserialize;
use serde_json::{json, Map, Value};
use std::collections::{BTreeMap, BTreeSet};

/// Shared with `TestPromptSharedVector` in hrm_advise_test.go.
pub const PROMPT_VECTOR: &str = "phase:develop intent:review risk:medium budget:a=1,b=2 agents:claude>codex evidence:diff,report\nrequest:fix the build";

#[derive(Deserialize, Default)]
struct Task {
    #[serde(default)]
    request: String,
    #[serde(default)]
    intent: String,
    #[serde(default)]
    risk: String,
}
#[derive(Deserialize, Default)]
struct State {
    #[serde(default)]
    phase: String,
    #[serde(default)]
    budget: BTreeMap<String, i64>,
    #[serde(default)]
    agent_chain: Vec<String>,
}
#[derive(Deserialize, Default)]
struct Evidence {
    #[serde(default)]
    class: String,
}
#[derive(Deserialize, Default)]
struct Trace {
    #[serde(default)]
    agent: String,
}
#[derive(Deserialize, Default)]
struct Decision {
    #[serde(default)]
    recommended_action: String,
    #[serde(default)]
    next_phase: String,
    #[serde(default)]
    abstain: bool,
}
#[derive(Deserialize, Default)]
struct Outcome {
    #[serde(default)]
    verified: bool,
    #[serde(default)]
    human_override: bool,
}
#[derive(Deserialize, Default)]
struct Workspace {
    #[serde(default)]
    repo_family: String,
}
#[derive(Deserialize)]
pub struct Episode {
    #[serde(default, alias = "event_id")]
    episode_id: String,
    #[serde(default)]
    workspace: Workspace,
    #[serde(default)]
    task: Task,
    #[serde(default, alias = "state_before")]
    state: State,
    #[serde(default)]
    evidence: Vec<Evidence>,
    #[serde(default)]
    trace: Trace,
    decision: Option<Decision>,
    outcome: Option<Outcome>,
}

/// Canonical model input; identical to the server's prompt builder.
pub fn prompt(e: &Episode) -> String {
    let patterns = [
        r"AKIA[0-9A-Z]{16}",
        r"gh[pousr]_[A-Za-z0-9]{36}",
        r"(?i)(?:password|passwd|token|secret|api[_-]?key)\s*[=:]\s*[^\s]{8,}",
    ];
    let mut req = e.task.request.clone();
    for p in patterns {
        req = Regex::new(p).expect("static pattern").replace_all(&req, "[REDACTED]").into_owned();
    }
    let budget = e.state.budget.iter().map(|(k, v)| format!("{k}={v}")).collect::<Vec<_>>().join(","); // BTreeMap: sorted
    let mut classes: Vec<&str> = e.evidence.iter().map(|x| x.class.as_str()).collect();
    classes.sort_unstable();
    format!(
        "phase:{} intent:{} risk:{} budget:{} agents:{} evidence:{}\nrequest:{}",
        e.state.phase, e.task.intent, e.task.risk, budget, e.state.agent_chain.join(">"), classes.join(","), req
    )
}

/// Risk levels are ordinal; the score head regresses their bin centres.
fn risk_score(level: &str) -> Option<f64> {
    match level {
        "low" => Some(0.125),
        "medium" => Some(0.375),
        "high" => Some(0.625),
        "critical" => Some(0.875),
        _ => None,
    }
}

#[derive(Default, Debug, PartialEq)]
pub struct Report {
    pub read: usize,
    pub written: usize,
    pub skipped_malformed: usize,
    pub skipped_unverified: usize,
    pub skipped_abstain: usize,
    pub soft_overridden: usize,
    pub soft_masked: usize,
}

/// Heads the Clef teacher may label by default. `action` is excluded because it
/// carries the abstain decision: the M7-0 gate measured clef-flash calling a tool
/// on 25% of BFCL irrelevance items (should be 0), so abstention is learned only
/// from verified outcomes.
pub const DEFAULT_SOFT_HEADS: &[&str] = &["intent", "next_phase", "risk", "agent"];

/// `soft`: episode_id -> {head -> {option -> probability}} from the Clef teacher,
/// replacing the hard label for that head. Only [`DEFAULT_SOFT_HEADS`] are taken.
pub fn convert(input: &str, soft: &BTreeMap<String, Map<String, Value>>) -> (Vec<Value>, Report) {
    convert_masked(input, soft, DEFAULT_SOFT_HEADS)
}

/// As [`convert`], with an explicit set of heads the teacher may override.
/// Teacher distributions for other heads are dropped and counted in `soft_masked`.
pub fn convert_masked(input: &str, soft: &BTreeMap<String, Map<String, Value>>, allowed: &[&str]) -> (Vec<Value>, Report) {
    let mut rep = Report::default();
    let mut out = Vec::new();
    let mut seen = BTreeSet::new();
    for line in input.lines().filter(|l| !l.trim().is_empty()) {
        rep.read += 1;
        let Ok(e) = serde_json::from_str::<Episode>(line) else {
            rep.skipped_malformed += 1;
            continue;
        };
        let (Some(d), Some(o)) = (&e.decision, &e.outcome) else {
            rep.skipped_unverified += 1;
            continue;
        };
        if !o.verified || o.human_override {
            rep.skipped_unverified += 1;
            continue;
        }
        if d.abstain || d.recommended_action.is_empty() || d.recommended_action == "abstain" {
            rep.skipped_abstain += 1;
            continue;
        }
        if e.episode_id.is_empty() || !seen.insert(e.episode_id.clone()) {
            rep.skipped_malformed += 1;
            continue;
        }
        let mut labels = Map::new();
        labels.insert("action".into(), json!(d.recommended_action));
        if !d.next_phase.is_empty() {
            labels.insert("next_phase".into(), json!(d.next_phase));
        }
        if !e.task.intent.is_empty() {
            labels.insert("intent".into(), json!(e.task.intent));
        }
        if let Some(r) = risk_score(&e.task.risk) {
            labels.insert("risk".into(), json!(r));
        }
        if matches!(e.trace.agent.as_str(), "claude" | "codex" | "opencode" | "hermes") {
            labels.insert("agent".into(), json!(e.trace.agent));
        }
        if let Some(s) = soft.get(&e.episode_id) {
            for (head, dist) in s {
                if !allowed.contains(&head.as_str()) {
                    rep.soft_masked += 1;
                    continue;
                }
                labels.insert(head.clone(), dist.clone());
                rep.soft_overridden += 1;
            }
        }
        let group = if e.workspace.repo_family.is_empty() { "unknown".to_string() } else { e.workspace.repo_family.clone() };
        out.push(json!({"id": e.episode_id, "group": group, "input": prompt(&e), "labels": labels}));
        rep.written += 1;
    }
    (out, rep)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn ep(id: &str, extra: Value) -> String {
        let mut v = json!({
            "episode_id": id, "workspace": {"repo_family": "fam"},
            "task": {"request": "fix the build", "intent": "review", "risk": "medium"},
            "state_before": {"phase": "develop", "budget": {"b": 2, "a": 1}, "agent_chain": ["claude", "codex"]},
            "evidence": [{"class": "report"}, {"class": "diff"}],
            "trace": {"agent": "codex"},
            "decision": {"recommended_action": "request_review", "next_phase": "develop_review", "abstain": false},
            "outcome": {"verified": true, "human_override": false}
        });
        v.as_object_mut().unwrap().extend(extra.as_object().unwrap().clone());
        v.to_string()
    }

    #[test]
    fn prompt_matches_the_servers_vector() {
        let e: Episode = serde_json::from_str(&ep("x", json!({}))).unwrap();
        assert_eq!(prompt(&e), PROMPT_VECTOR);
    }

    #[test]
    fn redacts_secrets_like_the_server() {
        let e: Episode = serde_json::from_str(&ep("x", json!({"task": {"request": "use AKIAABCDEFGHIJKLMNOP and password=hunter2hunter2", "intent": "review", "risk": "low"}}))).unwrap();
        let p = prompt(&e);
        assert!(!p.contains("AKIA") && !p.contains("hunter2"), "{p}");
        assert!(p.contains("[REDACTED]"));
    }

    #[test]
    fn only_verified_non_overridden_decisions_become_labels() {
        let lines = [
            ep("ok", json!({})),
            ep("unverified", json!({"outcome": {"verified": false}})),
            ep("overridden", json!({"outcome": {"verified": true, "human_override": true}})),
            ep("abstained", json!({"decision": {"recommended_action": "allow", "abstain": true}})),
            ep("no-outcome", json!({"outcome": null})),
            ep("ok", json!({})), // duplicate id
            "not json".into(),
        ]
        .join("\n");
        let (out, rep) = convert(&lines, &BTreeMap::new());
        assert_eq!(out.len(), 1);
        assert_eq!(rep, Report { read: 7, written: 1, skipped_malformed: 2, skipped_unverified: 3, skipped_abstain: 1, soft_overridden: 0, soft_masked: 0 });
        let l = &out[0]["labels"];
        assert_eq!((l["action"].as_str(), l["next_phase"].as_str(), l["intent"].as_str(), l["agent"].as_str()),
                   (Some("request_review"), Some("develop_review"), Some("review"), Some("codex")));
        assert_eq!(l["risk"], 0.375);
        assert_eq!(out[0]["group"], "fam");
        assert_eq!(out[0]["input"], PROMPT_VECTOR.replace("evidence:diff,report", "evidence:diff,report"));
    }

    #[test]
    fn teacher_distributions_replace_hard_labels() {
        let mut soft = BTreeMap::new();
        let mut m = Map::new();
        m.insert("intent".into(), json!({"review": 0.7, "validate": 0.3}));
        soft.insert("ok".to_string(), m);
        let (out, rep) = convert(&ep("ok", json!({})), &soft);
        assert_eq!((rep.soft_overridden, rep.soft_masked), (1, 0));
        assert_eq!(out[0]["labels"]["intent"]["validate"], 0.3);
    }

    #[test]
    fn teacher_cannot_label_action_by_default() {
        let mut soft = BTreeMap::new();
        let mut m = Map::new();
        m.insert("action".into(), json!({"abstain": 0.1, "ask_operator": 0.9}));
        soft.insert("ok".to_string(), m.clone());
        let (out, rep) = convert(&ep("ok", json!({})), &soft);
        assert_eq!((rep.soft_overridden, rep.soft_masked), (0, 1));
        assert_eq!(out[0]["labels"]["action"], "request_review"); // hard, outcome-derived label kept
        // Explicitly widening the mask is possible, never implicit.
        let (out, rep) = convert_masked(&ep("ok", json!({})), &soft, &["action"]);
        assert_eq!(rep.soft_overridden, 1);
        assert_eq!(out[0]["labels"]["action"]["ask_operator"], 0.9);
    }

    #[test]
    fn unknown_agents_and_risks_are_not_invented() {
        let (out, _) = convert(&ep("x", json!({"trace": {"agent": "mystery"}, "task": {"request": "r", "intent": "", "risk": "wat"}})), &BTreeMap::new());
        let l = out[0]["labels"].as_object().unwrap();
        assert!(!l.contains_key("agent") && !l.contains_key("risk") && !l.contains_key("intent"));
    }
}
