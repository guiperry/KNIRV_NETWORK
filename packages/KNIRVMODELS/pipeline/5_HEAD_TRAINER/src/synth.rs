//! Synthetic generators for the Task 4.4 kill-gate learnability probe (no gold data).
//!
//! Each task is a choice-set problem whose rule transfers to *unseen choice strings*:
//! the training and test candidate pools are disjoint surface forms sharing only the
//! latent signal the model must learn to read. A model that merely memorises training
//! strings (or leans on position) stays at the base rate; a model that learns the rule
//! clears it.
//!
//! * Task A — routing: the question names a *topic*; the topic maps to a *keyword*; the
//!   correct candidate is the one containing that keyword. Train/test topics are disjoint,
//!   and the candidate strings that carry a keyword are different between splits.
//! * Task B — evidence reranker: the question names an *entity*; the correct evidence is
//!   the sentence that mentions that entity. Train/test entities are disjoint and the
//!   evidence phrasings differ between splits.

use crate::data::Split;
use crate::setprobe::SetExample;

struct Xs(u64);
impl Xs {
    fn next(&mut self) -> u64 {
        self.0 ^= self.0 << 13;
        self.0 ^= self.0 >> 7;
        self.0 ^= self.0 << 17;
        self.0
    }
    fn below(&mut self, n: usize) -> usize {
        (self.next() % n as u64) as usize
    }
    fn pick<T: Clone>(&mut self, v: &[T]) -> T {
        v[self.below(v.len())].clone()
    }
}

/// `topic -> keyword`; train and test use disjoint topics for the same keywords.
fn routing_topics() -> (Vec<(&'static str, &'static str)>, Vec<(&'static str, &'static str)>) {
    let all = [
        ("checkout", "payment"),
        ("cart", "payment"),
        ("invoice", "billing"),
        ("refund", "billing"),
        ("login", "auth"),
        ("session", "auth"),
        ("query", "search"),
        ("index", "search"),
        ("alert", "notify"),
        ("push", "notify"),
        ("blob", "storage"),
        ("volume", "storage"),
        ("ledger", "payment"),
        ("charge", "billing"),
        ("token", "auth"),
        ("lookup", "search"),
        ("signal", "notify"),
        ("archive", "storage"),
    ];
    let (tr, te) = all.split_at(all.len() / 2);
    (tr.to_vec(), te.to_vec())
}

fn routing_pool(key: &str, test: bool) -> Vec<String> {
    if test {
        vec![format!("kn-{key}"), format!("{key}_svc"), format!("tier-{key}")]
    } else {
        vec![format!("svc-{key}"), format!("{key}-api"), format!("{key}-worker")]
    }
}

fn evidence_entities() -> (Vec<&'static str>, Vec<&'static str>) {
    let all = ["atlas", "beacon", "cipher", "delta", "ember", "falcon", "gaia", "helix", "iris", "juno", "kraken", "lyra"];
    let (a, b) = all.split_at(6);
    (a.to_vec(), b.to_vec())
}

fn evidence_pool(entity: &str, test: bool) -> Vec<String> {
    if test {
        vec![
            format!("{entity} served the request"),
            format!("{entity} held steady under load"),
            format!("{entity} answered the probe"),
        ]
    } else {
        vec![
            format!("{entity} handled the packet"),
            format!("{entity} completed the task"),
            format!("{entity} passed the check"),
        ]
    }
}

/// Generates `n_train` training and `n_test` test examples for `task` (`"a"` or `"b"`).
/// The returned examples carry their `split` so the caller can filter.
pub fn generate(task: &str, n_train: usize, n_test: usize, seed: u64) -> Vec<SetExample> {
    match task {
        "a" => routing(n_train, n_test, seed),
        "b" => evidence(n_train, n_test, seed),
        other => panic!("unknown synth task {other:?} (a|b)"),
    }
}

fn routing(n_train: usize, n_test: usize, seed: u64) -> Vec<SetExample> {
    let (tr, te) = routing_topics();
    let mut rng = Xs(seed | 1);
    let mut out = Vec::new();
    let mut make = |topics: &[(&str, &str)], split: Split, start: usize, n: usize, out: &mut Vec<SetExample>| {
        for i in 0..n {
            let (topic, key) = rng.pick(topics);
            let test = split == Split::Test;
            let correct = rng.pick(&routing_pool(key, test));
            // Distractors: other keys' pools (different surface forms, never contain `key`).
            let mut cand = vec![correct.clone()];
            let n_total = 4 + rng.below(5); // 4..=8
            let mut guards = 0;
            while cand.len() < n_total && guards < 50 {
                guards += 1;
                let dk = rng.pick(&["payment", "billing", "auth", "search", "notify", "storage"]);
                if dk == key {
                    continue;
                }
                let d = rng.pick(&routing_pool(dk, test));
                if !cand.contains(&d) {
                    cand.push(d);
                }
            }
            out.push(SetExample {
                id: format!("{}-r{}-{}", if test { "test" } else { "train" }, start + i, topic),
                group: topic.to_string(),
                split,
                question: format!("Triage the {topic} incident and route it to the {key} owner"),
                state: "host=idle".to_string(),
                candidates: cand,
                target: 0,
            });
        }
    };
    make(&tr, Split::Train, 0, n_train, &mut out);
    make(&te, Split::Test, n_train, n_test, &mut out);
    out
}

fn evidence(n_train: usize, n_test: usize, seed: u64) -> Vec<SetExample> {
    let (tr, te) = evidence_entities();
    let mut rng = Xs(seed | 1);
    let mut out = Vec::new();
    let mut make = |entities: &[&str], split: Split, start: usize, n: usize, out: &mut Vec<SetExample>| {
        for i in 0..n {
            let entity = rng.pick(entities);
            let test = split == Split::Test;
            let correct = rng.pick(&evidence_pool(entity, test));
            let mut cand = vec![correct.clone()];
            let n_total = 4 + rng.below(5);
            let mut guards = 0;
            while cand.len() < n_total && guards < 50 {
                guards += 1;
                let dk = rng.pick(entities);
                if dk == entity {
                    continue;
                }
                let d = rng.pick(&evidence_pool(dk, test));
                if !cand.contains(&d) {
                    cand.push(d);
                }
            }
            out.push(SetExample {
                id: format!("{}-e{}-{}", if test { "test" } else { "train" }, start + i, entity),
                group: entity.to_string(),
                split,
                question: format!("Does the evidence support {entity}?"),
                state: String::new(),
                candidates: cand,
                target: 0,
            });
        }
    };
    make(&tr, Split::Train, 0, n_train, &mut out);
    make(&te, Split::Test, n_train, n_test, &mut out);
    out
}
