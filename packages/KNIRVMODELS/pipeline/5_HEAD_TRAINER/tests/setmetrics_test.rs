//! Task 4.5 — choice-set metric unit tests (synthetic fixtures, no model needed).

use heart_decision_trainer::setmetrics::*;
use std::collections::{BTreeMap, BTreeSet};

/// Build a scored set from request-ordered choices `(id, text, prob)`. `n` is the
/// number of choices (must sum to ~1.0 to be a valid, aggregated row).
fn set(head: &str, choices: &[(&str, &str, f32)], correct: Option<&str>) -> ScoredSet {
    from_choices(head, choices, correct, None)
}

#[test]
fn order_invariance_detects_permutation_stable_and_unstable() {
    // g1: same ids get the same probability under a different order -> invariant.
    // g2: one id's probability changes with order -> not invariant.
    let mut groups: BTreeMap<String, Vec<ScoredSet>> = BTreeMap::new();
    groups.insert(
        "g1".to_string(),
        vec![
            set("route", &[("a", "ta", 0.5), ("b", "tb", 0.3), ("c", "tc", 0.2)], None),
            set("route", &[("b", "tb", 0.3), ("a", "ta", 0.5), ("c", "tc", 0.2)], None),
        ],
    );
    groups.insert(
        "g2".to_string(),
        vec![
            set("route", &[("a", "ta", 0.6), ("b", "tb", 0.4)], None),
            set("route", &[("a", "ta", 0.2), ("b", "tb", 0.8)], None),
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

// F1: a group whose members have different id sets / sizes is NOT a permutation,
// so it must not be counted order-invariant.
#[test]
fn order_invariance_rejects_different_id_sets() {
    let mut groups: BTreeMap<String, Vec<ScoredSet>> = BTreeMap::new();
    groups.insert(
        "extra_option".to_string(),
        vec![
            set("route", &[("a", "ta", 0.5), ("b", "tb", 0.5)], None),
            // an extra option, equal a/b values: these are different choice sets
            set("route", &[("a", "ta", 0.5), ("b", "tb", 0.5), ("c", "tc", 0.0)], None),
        ],
    );
    let (rate, _) = order_invariance_rate(&groups, 1e-5);
    assert_eq!(rate, 0.0, "a set with an extra option is not a permutation");
}

// F3: a NaN candidate must never count as invariant.
#[test]
fn order_invariance_rejects_nan_probability() {
    let mut groups: BTreeMap<String, Vec<ScoredSet>> = BTreeMap::new();
    groups.insert(
        "nan".to_string(),
        vec![
            set("route", &[("a", "ta", 0.5), ("b", "tb", 0.5)], None),
            set("route", &[("a", "ta", f32::NAN), ("b", "tb", 0.5)], None),
        ],
    );
    let (rate, _) = order_invariance_rate(&groups, 1e-5);
    assert_eq!(rate, 0.0, "NaN candidate must not be called invariant");
}

#[test]
fn accuracy_by_n_buckets_correctly() {
    let rows = vec![
        set("route", &[("a", "ta", 0.6), ("b", "tb", 0.4)], Some("a")), // correct
        set("route", &[("a", "ta", 0.4), ("b", "tb", 0.6)], Some("a")), // wrong
        set("route", &[("x", "tx", 0.5), ("y", "ty", 0.3), ("z", "tz", 0.2)], Some("x")), // correct
    ];
    let by_n = accuracy_by_n(&rows);
    assert_eq!(by_n[&2], (1, 2));
    assert_eq!(by_n[&3], (1, 1));
    // a row without a known correct answer does not count
    let rows2 = vec![set("route", &[("a", "ta", 0.6), ("b", "tb", 0.4)], None)];
    assert!(accuracy_by_n(&rows2).is_empty());
}

// F4: top-1 uses request order, not lexicographic id. A tied set whose first request
// candidate is lexically later must evaluate exactly as the serving (first-max) argmax,
// including after a permutation.
#[test]
fn top1_uses_request_order_on_ties() {
    // served order [b, a]; b is lexically later but first in the request.
    let tied = set("route", &[("b", "tb", 0.5), ("a", "ta", 0.5)], Some("a"));
    assert_eq!(top1(&tied.candidates).as_deref(), Some("b"));
    let by_n = accuracy_by_n(&[tied.clone()]);
    assert_eq!(by_n[&2], (0, 1), "request-order tie rule selects b, not lexicographic a");

    // permuted request [a, b]: the serving argmax now selects a, matching the metric.
    let perm = set("route", &[("a", "ta", 0.5), ("b", "tb", 0.5)], Some("a"));
    assert_eq!(top1(&perm.candidates).as_deref(), Some("a"));
    assert_eq!(accuracy_by_n(&[perm.clone()])[&2], (1, 1));
}

// F2: novelty gates on canonical choice *text*, not id.
#[test]
fn novel_option_accuracy_gates_on_text_not_id() {
    let mut training = BTreeSet::new();
    training.insert(canonical_text("route-a"));
    training.insert(canonical_text("route-x"));

    // same id, DIFFERENT text -> novel (text gate, not id)
    let same_id_diff_text = set("route", &[("a", "route-z", 0.6), ("b", "route-b", 0.4)], Some("a"));
    // different id, SAME text -> seen (skipped)
    let diff_id_same_text = set("route", &[("q", "route-a", 0.6), ("b", "route-b", 0.4)], Some("q"));

    let rows = vec![
        set("route", &[("b", "route-b", 0.6), ("c", "route-c", 0.4)], Some("b")), // novel, correct
        set("route", &[("a", "route-a", 0.6), ("c", "route-c", 0.4)], Some("a")), // seen -> skip
        set("route", &[("c", "route-c", 0.6), ("b", "route-b", 0.4)], Some("b")), // novel, top=c -> wrong
        same_id_diff_text,
        diff_id_same_text,
    ];
    let acc = novel_option_accuracy(&rows, &training).unwrap();
    // novel counted: rows 1, 3, same_id_diff_text = 3; correct among them = 2 => 2/3
    assert!((acc - 2.0 / 3.0).abs() < 1e-6, "expected 2/3, got {acc}");

    // never fabricate a baseline from an empty training set
    assert!(novel_option_accuracy(&rows, &BTreeSet::new()).is_none());
}

#[test]
fn fit_tau_by_n_bucket_covers_the_four_buckets() {
    let rows = vec![
        set("route", &[("a", "ta", 0.9), ("b", "tb", 0.1)], Some("a")), // n=2
        set(
            "route",
            &[("a", "ta", 0.8), ("b", "tb", 0.1), ("c", "tc", 0.05), ("d", "td", 0.05)],
            Some("a"),
        ), // n=4
        set(
            "route",
            &[
                ("a", "ta", 0.7),
                ("b", "tb", 0.1),
                ("c", "tc", 0.1),
                ("d", "td", 0.05),
                ("e", "te", 0.03),
                ("f", "tf", 0.01),
                ("g", "tg", 0.01),
            ],
            Some("a"),
        ), // n=7
        // n=9 (9+ bucket), all-correct confidences
        set(
            "route",
            &[
                ("a", "ta", 0.85),
                ("b", "tb", 0.01875),
                ("c", "tc", 0.01875),
                ("d", "td", 0.01875),
                ("e", "te", 0.01875),
                ("f", "tf", 0.01875),
                ("g", "tg", 0.01875),
                ("h", "th", 0.01875),
                ("i", "ti", 0.01875),
            ],
            Some("a"),
        ),
    ];
    let fits = fit_tau_by_n_bucket(&rows, 0.95);
    assert_eq!(fits.len(), 4);
    for b in ["2", "3-4", "5-8", "9+"] {
        assert!(fits.contains_key(b), "missing bucket {b}");
        assert!(fits[b].met, "bucket {b} should meet target with all-correct confidences");
    }
}

// F3: invalid rows are excluded and reported, never placed in a 9+ tau bucket.
#[test]
fn invalid_rows_excluded_from_tau_and_reported() {
    let valid = set(
        "route",
        &[
            ("a", "ta", 0.85),
            ("b", "tb", 0.01875),
            ("c", "tc", 0.01875),
            ("d", "td", 0.01875),
            ("e", "te", 0.01875),
            ("f", "tf", 0.01875),
            ("g", "tg", 0.01875),
            ("h", "th", 0.01875),
            ("i", "ti", 0.01875),
        ],
        Some("a"),
    );
    let nan_row = set(
        "route",
        &[
            ("a", "ta", f32::NAN),
            ("b", "tb", f32::NAN),
            ("c", "tc", f32::NAN),
            ("d", "td", f32::NAN),
            ("e", "te", f32::NAN),
            ("f", "tf", f32::NAN),
            ("g", "tg", f32::NAN),
            ("h", "th", f32::NAN),
            ("i", "ti", f32::NAN),
        ],
        Some("a"),
    );
    let empty_row = from_choices("route", &[], None, None);
    let n1_row = set("route", &[("a", "ta", 1.0)], Some("a")); // N=1, invalid
    let mismatched_row = from_choices(
        "route",
        &[("a", "ta", 0.6), ("b", "tb", 0.4)],
        Some("a"),
        None,
    );
    // n is set to 3 but only 2 candidates -> invalid
    let mut mismatched = mismatched_row;
    mismatched.n = 3;

    let rows = vec![valid.clone(), nan_row, empty_row, n1_row, mismatched];
    let fits = fit_tau_by_n_bucket(&rows, 0.95);
    // the only valid 9+ row is `valid`; the NaN/empty/N=1 rows never reach a bucket
    let m = build_set_metrics(&rows, &BTreeMap::new(), &BTreeSet::new(), 0.95);
    assert_eq!(m.invalid_rows, 4, "NaN, empty, N=1 and n!=len rows are invalid");
    assert!(fits.contains_key("9+"), "valid row still lands in 9+");
    assert!(fits["9+"].met);
}

#[test]
fn build_set_metrics_serializes_into_metrics_json_shape() {
    let rows = vec![
        set("route", &[("a", "ta", 0.6), ("b", "tb", 0.4)], Some("a")),
        set("route", &[("x", "tx", 0.5), ("y", "ty", 0.3), ("z", "tz", 0.2)], Some("x")),
    ];
    let mut groups: BTreeMap<String, Vec<ScoredSet>> = BTreeMap::new();
    groups.insert(
        "g1".to_string(),
        vec![
            set("route", &[("x", "tx", 0.5), ("y", "ty", 0.3), ("z", "tz", 0.2)], None),
            set("route", &[("y", "ty", 0.3), ("x", "tx", 0.5), ("z", "tz", 0.2)], None),
        ],
    );
    let m = build_set_metrics(&rows, &groups, &BTreeSet::new(), 0.95);
    // provisional fields are None without training texts / skill tags
    assert!(m.novel_option_accuracy.is_none());
    assert!(m.accuracy_by_skill.is_none());
    assert!(m.max_choices_provisional);
    assert_eq!(m.invalid_rows, 0);

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
