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
/// The returned examples carry their `split` so the caller can filter. When `canonical` is
/// true, candidates are emitted as short canonical labels (the keyword/entity) plus their
/// positional id — the Task 4.4 fallback (a) — instead of verbose surface-form strings.
pub fn generate(task: &str, n_train: usize, n_test: usize, seed: u64, canonical: bool) -> Vec<SetExample> {
    match task {
        "a" => routing(n_train, n_test, seed, canonical),
        "b" => evidence(n_train, n_test, seed, canonical),
        other => panic!("unknown synth task {other:?} (a|b)"),
    }
}

fn routing(n_train: usize, n_test: usize, seed: u64, canonical: bool) -> Vec<SetExample> {
    let (tr, te) = routing_topics();
    let mut rng = Xs(seed.wrapping_mul(2).wrapping_add(1));
    let mut out = Vec::new();
    let mut make = |topics: &[(&str, &str)], split: Split, start: usize, n: usize, out: &mut Vec<SetExample>| {
        for i in 0..n {
            let (topic, key) = rng.pick(topics);
            let test = split == Split::Test;
            // Fallback (a): the candidate is the bare keyword — a short canonical label — so the
            // rule "pick the candidate whose text equals the question's keyword" is low-entropy.
            let correct = if canonical { key.to_string() } else { rng.pick(&routing_pool(key, test)) };
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
                let d = if canonical { dk.to_string() } else { rng.pick(&routing_pool(dk, test)) };
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

fn evidence(n_train: usize, n_test: usize, seed: u64, canonical: bool) -> Vec<SetExample> {
    let (tr, te) = evidence_entities();
    let mut rng = Xs(seed.wrapping_mul(2).wrapping_add(1));
    let mut out = Vec::new();
    let mut make = |entities: &[&str], split: Split, start: usize, n: usize, out: &mut Vec<SetExample>| {
        for i in 0..n {
            let entity = rng.pick(entities);
            let test = split == Split::Test;
            // Fallback (a): the candidate is the bare entity — a short canonical label.
            let correct = if canonical { entity.to_string() } else { rng.pick(&evidence_pool(entity, test)) };
            let mut cand = vec![correct.clone()];
            let n_total = 4 + rng.below(5);
            let mut guards = 0;
            while cand.len() < n_total && guards < 50 {
                guards += 1;
                let dk = rng.pick(entities);
                if dk == entity {
                    continue;
                }
                let d = if canonical { dk.to_string() } else { rng.pick(&evidence_pool(dk, test)) };
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

// ─────────────────────────────────────────────────────────────────────────────
// Task 4.4 — E1 (fresh-data) vocabulary and generators.
//
// E1 trains on a *stream* of freshly generated examples (no fixed training set) over a
// large keyword/topic vocabulary, so the model cannot memorise. The test set is built
// fresh too and split into a SEEN-key part (topics/phrasings disjoint from training but
// the keywords are from the training vocabulary) and an UNSEEN-key part (some keywords
// are held out of training entirely) — the strictest generalisation check.
// ─────────────────────────────────────────────────────────────────────────────

/// Keyword/topic vocabulary and phrasings for the E1 stream. `train_keys` and `heldout_keys`
/// partition the keyword vocabulary; `train_topics`/`test_topics` partition the topics so the
/// two splits never share a topic. Candidate templates for train vs test are disjoint surface
/// forms, so the candidate strings are also disjoint between splits.
pub struct Pool {
    pub train_keys: Vec<&'static str>,
    pub heldout_keys: Vec<&'static str>,
    pub train_topics: Vec<&'static str>,
    pub test_topics: Vec<&'static str>,
    pub routing_q: Vec<&'static str>,
    pub routing_train_cand: Vec<&'static str>,
    pub routing_test_cand: Vec<&'static str>,
    pub evidence_q: Vec<&'static str>,
    pub evidence_train_cand: Vec<&'static str>,
    pub evidence_test_cand: Vec<&'static str>,
}

pub fn default_pool() -> Pool {
    let keys: Vec<&str> = vec![
        "payment", "billing", "auth", "search", "notify", "storage", "invoice", "refund", "login",
        "session", "query", "index", "alert", "push", "blob", "volume", "ledger", "charge", "token",
        "lookup", "signal", "archive", "cache", "queue", "retry", "batch", "stream", "shard", "proxy",
        "socket", "broker", "tenant", "cluster", "region", "policy", "rule", "metric", "trace", "span",
        "log", "event", "job", "task", "worker", "gateway", "router", "resolver", "catalog", "registry",
    ];
    let (train_keys, heldout_keys) = keys.split_at(40);
    let topics: Vec<&str> = vec![
        "checkout", "cart", "dashboard", "report", "profile", "order", "shipment", "refund", "login",
        "session", "search", "alert", "notification", "file", "image", "video", "message", "build",
        "deploy", "migrate", "backup", "restore", "scan", "audit", "export", "import", "sync", "latency",
        "throughput", "error", "timeout", "crash", "leak", "spike", "drift", "capacity", "quota", "config",
        "secret", "cert", "dns", "tls", "http", "grpc", "kafka", "redis", "postgres", "s3",
    ];
    let (train_topics, test_topics) = topics.split_at(40);
    Pool {
        train_keys: train_keys.to_vec(),
        heldout_keys: heldout_keys.to_vec(),
        train_topics: train_topics.to_vec(),
        test_topics: test_topics.to_vec(),
        routing_q: vec![
            "Triage the {t} incident and route it to the {k} owner",
            "Assign the {t} alert to the {k} team",
            "Send the {t} report to the {k} lead",
            "Handle the {t} request via the {k} queue",
            "Escalate the {t} issue to the {k} on-call",
        ],
        routing_train_cand: vec!["svc-{k}", "{k}-api", "{k}-worker", "{k}-team", "{k}-queue"],
        routing_test_cand: vec!["kn-{k}", "{k}_svc", "tier-{k}", "{k}-prod", "{k}-svc2"],
        evidence_q: vec![
            "Does the evidence support {k}?",
            "Was {k} the root cause?",
            "Did {k} recover after the fix?",
            "Is {k} flagged by the monitor?",
            "Should we page {k}?",
        ],
        evidence_train_cand: vec!["{k}-log", "evt-{k}", "trace-{k}", "span-{k}", "node-{k}"],
        evidence_test_cand: vec!["kn-{k}-e", "{k}-rec", "sig-{k}", "out-{k}", "rep-{k}"],
    }
}

/// Generates ONE fresh example. `use_heldout` selects the unseen-key vocabulary (test strict
/// check); otherwise training/seen-key vocabulary. `seed` makes the example deterministic per
/// step. The correct candidate is the only one containing the question's keyword, so the match
/// rule is both necessary and sufficient.
pub fn gen_one(task: &str, seed: u64, pool: &Pool, use_heldout: bool, canonical: bool) -> SetExample {
    let mut rng = Xs(seed.wrapping_mul(2).wrapping_add(1));
    let keys = if use_heldout { &pool.heldout_keys } else { &pool.train_keys };
    if task == "a" {
        let topic = rng.pick(if use_heldout { &pool.test_topics } else { &pool.train_topics });
        let key = rng.pick(keys);
        let q = rng.pick(&pool.routing_q);
        let question = q.replace("{t}", topic).replace("{k}", key);
        let tmpls = if use_heldout { &pool.routing_test_cand } else { &pool.routing_train_cand };
        let correct = if canonical { key.to_string() } else { rng.pick(tmpls).replace("{k}", key) };
        let mut cand = vec![correct.clone()];
        let n_total = 4 + rng.below(5);
        let mut guards = 0;
        while cand.len() < n_total && guards < 100 {
            guards += 1;
            let dk = rng.pick(keys);
            if dk == key {
                continue;
            }
            let d = if canonical { dk.to_string() } else { rng.pick(tmpls).replace("{k}", dk) };
            if !cand.contains(&d) {
                cand.push(d);
            }
        }
        SetExample {
            id: format!("e1a-{}-{}-{}", if use_heldout { "uh" } else { "tr" }, topic, key),
            group: format!("{topic}/{key}"),
            split: Split::Train,
            question,
            state: "host=idle".to_string(),
            candidates: cand,
            target: 0,
        }
    } else {
        let entity = rng.pick(keys);
        let q = rng.pick(&pool.evidence_q);
        let question = q.replace("{k}", entity);
        let tmpls = if use_heldout { &pool.evidence_test_cand } else { &pool.evidence_train_cand };
        let correct = if canonical { entity.to_string() } else { rng.pick(tmpls).replace("{k}", entity) };
        let mut cand = vec![correct.clone()];
        let n_total = 4 + rng.below(5);
        let mut guards = 0;
        while cand.len() < n_total && guards < 100 {
            guards += 1;
            let dk = rng.pick(keys);
            if dk == entity {
                continue;
            }
            let d = if canonical { dk.to_string() } else { rng.pick(tmpls).replace("{k}", dk) };
            if !cand.contains(&d) {
                cand.push(d);
            }
        }
        SetExample {
            id: format!("e1b-{}-{}-{}", if use_heldout { "uh" } else { "tr" }, entity, rng.below(100000)),
            group: entity.to_string(),
            split: Split::Train,
            question,
            state: String::new(),
            candidates: cand,
            target: 0,
        }
    }
}

/// Builds `n` fresh (deterministic) examples for the test set. `use_heldout` selects the
/// unseen-key vocabulary; otherwise the seen-key vocabulary (topics/phrasings still disjoint
/// from training).
pub fn gen_test(task: &str, n: usize, seed: u64, pool: &Pool, use_heldout: bool, canonical: bool) -> Vec<SetExample> {
    (0..n).map(|i| gen_one(task, seed ^ (i as u64).wrapping_mul(2654435761), pool, use_heldout, canonical)).collect()
}
