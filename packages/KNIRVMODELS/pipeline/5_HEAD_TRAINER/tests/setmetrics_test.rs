//! Task 4.5 — choice-set metric unit tests (synthetic fixtures, no model needed).

use heart_decision_trainer::setmetrics::*;
use std::collections::{BTreeMap, BTreeSet};

fn set(head: &str, n: usize, probs: &[(&str, f32)], correct: Option<&str>) -> ScoredSet {
    ScoredSet {
        head: head.to_string(),
        n,
        probs: probs.iter().map(|(id, p)| (id.to_string(), *p)).collect(),
        correct: correct.map(String::from),
        skill: None,
    }
}

#[test]
fn order_invariance_detects_permutation_stable_and_unstable() {
    // g1: same ids get the same probability under a different order -> invariant.
    // g2: one id's probability changes with order -> not invariant.
    let mut groups: BTreeMap<String, Vec<ScoredSet>> = BTreeMap::new();
    groups.insert(
        "g1".to_string(),
        vec![
            set("route", 3, &[("a", 0.5), ("b", 0.3), ("c", 0.2)], None),
            set("route", 3, &[("b", 0.3), ("a", 0.5), ("c", 0.2)], None),
        ],
    );
    groups.insert(
        "g2".to_string(),
        vec![
            set("route", 2, &[("a", 0.5), ("b", 0.3)], None),
            set("route", 2, &[("a", 0.5), ("b", 0.9)], None),
        ],
    );

    let (rate, worst) = order_invariance_rate(&groups, 1e-5);
    assert!((rate - 0.5).abs() < 1e-6, "expected rate 0.5, got {rate}");
    assert!(worst > 0.66 && worst < 0.67, "expected worst dev ~0.667, got {worst}");
}

#[test]
fn order_invariance_is_trivially_one_with_no_groups() {
    let groups: BTreeMap<String, Vec<ScoredSet>> = BTreeMap::new();
    let (rate, worst) = order_invariance_rate(&groups, 1e-5);
    assert_eq!(rate, 1.0);
    assert_eq!(worst, 0.0);
}

#[test]
fn accuracy_by_n_buckets_correctly() {
    let rows = vec![
        set("route", 2, &[("a", 0.6), ("b", 0.4)], Some("a")), // correct
        set("route", 2, &[("a", 0.4), ("b", 0.6)], Some("a")), // wrong
        set("route", 3, &[("x", 0.5), ("y", 0.3), ("z", 0.2)], Some("x")), // correct
    ];
    let by_n = accuracy_by_n(&rows);
    assert_eq!(by_n[&2], (1, 2));
    assert_eq!(by_n[&3], (1, 1));
    // a row without a known correct answer does not count
    let rows2 = vec![set("route", 2, &[("a", 0.6), ("b", 0.4)], None)];
    assert!(accuracy_by_n(&rows2).is_empty());
}

#[test]
fn novel_option_accuracy_skips_training_ids_and_returns_none_without_them() {
    let mut training = BTreeSet::new();
    training.insert("a".to_string());
    training.insert("x".to_string());
    let rows = vec![
        set("route", 2, &[("b", 0.6), ("c", 0.4)], Some("b")), // novel, correct
        set("route", 2, &[("a", 0.6), ("c", 0.4)], Some("a")), // in training -> skipped
        set("route", 2, &[("c", 0.6), ("b", 0.4)], Some("b")), // novel, top is c -> wrong
    ];
    let acc = novel_option_accuracy(&rows, &training).unwrap();
    assert!((acc - 0.5).abs() < 1e-6, "expected 0.5, got {acc}");

    // never fabricate a baseline from an empty training set
    assert!(novel_option_accuracy(&rows, &BTreeSet::new()).is_none());
}

#[test]
fn fit_tau_by_n_bucket_covers_the_four_buckets() {
    let rows = vec![
        set("route", 2, &[("a", 0.9), ("b", 0.1)], Some("a")),
        set("route", 4, &[("a", 0.8), ("b", 0.1), ("c", 0.05), ("d", 0.05)], Some("a")),
        set("route", 7, &[("a", 0.7), ("b", 0.1), ("c", 0.1), ("d", 0.05), ("e", 0.03), ("f", 0.01), ("g", 0.01)], Some("a")),
        set("route", 12, &[("a", 0.85), ("b", 0.05)], Some("a")),
    ];
    let fits = fit_tau_by_n_bucket(&rows, 0.95);
    assert_eq!(fits.len(), 4);
    for b in ["2", "3-4", "5-8", "9+"] {
        assert!(fits.contains_key(b), "missing bucket {b}");
        assert!(fits[b].met, "bucket {b} should meet target with all-correct confidences");
    }
}

#[test]
fn build_set_metrics_serializes_into_metrics_json_shape() {
    let rows = vec![
        set("route", 2, &[("a", 0.6), ("b", 0.4)], Some("a")),
        set("route", 3, &[("x", 0.5), ("y", 0.3), ("z", 0.2)], Some("x")),
    ];
    let mut groups: BTreeMap<String, Vec<ScoredSet>> = BTreeMap::new();
    groups.insert(
        "g1".to_string(),
        vec![
            set("route", 3, &[("x", 0.5), ("y", 0.3), ("z", 0.2)], None),
            set("route", 3, &[("y", 0.3), ("x", 0.5), ("z", 0.2)], None),
        ],
    );
    let m = build_set_metrics(&rows, &groups, &BTreeSet::new(), 0.95);
    // provisional fields are None without training ids / skill tags
    assert!(m.novel_option_accuracy.is_none());
    assert!(m.accuracy_by_skill.is_none());
    assert!(m.max_choices_provisional);

    let json = serde_json::to_value(&m).unwrap();
    assert!(json.get("accuracy_by_n").is_some());
    assert!(json.get("tau_by_n_bucket").is_some());
    assert_eq!(json["order_invariance_rate"], 1.0);
    // the whole Metrics report embeds this under `set`
    let mut report = heart_decision_trainer::eval::Metrics::default();
    report.set = Some(m);
    let report_json = serde_json::to_string(&report).unwrap();
    assert!(report_json.contains("\"set\""), "metrics.json must carry the set block: {report_json}");
}
