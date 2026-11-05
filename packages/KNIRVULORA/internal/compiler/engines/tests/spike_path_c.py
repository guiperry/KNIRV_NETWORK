#!/usr/bin/env python3
"""
Task 5.2 spike harness for Path C (see PendingImplementations.md).

Builds hand-specified skill clusters whose resolution is an invented, checkable
fact the base model cannot know (command + flag value + code). For each cluster
it trains one cluster adapter and one single-skill adapter per member through
path_c_distill.py, then scores held-out paraphrased tasks in three arms:

    base            : no adapter, no skill text
    base+skill      : no adapter, the cluster's skill text in the prompt (control)
    adapter         : cluster adapter, no skill text

A member "passes" when its adapter-arm score beats base by --margin. I5
interference is single-skill score minus cluster score per member (must be <= eta).
A regression probe compares base and adapter on general questions.

STAND-IN TEACHER: completions are produced by a deterministic template oracle,
not an external provider. That is the same signal an ideal teacher with the skill
in context would give; it measures whether gradient distillation can install the
facts, not teacher quality. Recorded in the report.

Usage: spike_path_c.py --base DIR --out DIR [--clusters 3,4,5] [--epochs N]
"""
import argparse
import hashlib
import json
import os
import random
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ENGINE = os.path.join(HERE, "..", "path_c_distill.py")

SYL = ["vor", "tex", "zan", "qui", "mel", "dra", "kor", "bel", "nis", "pho", "ruk", "sal", "tum", "wex", "yol", "gar"]


def word(rng, n=3):
    return "".join(rng.choice(SYL) for _ in range(n))


SYMPTOMS_TRAIN = [
    "Our {svc} keeps failing with {err}. What should we do?",
    "How do I fix {err} in the {svc}?",
    "{err} just appeared in the {svc} logs. Steps to resolve?",
    "The {svc} is throwing {err} after a restart. Fix?",
    "Help: {err} on {svc}, nothing else changed.",
    "What is the resolution for {err} ({svc})?",
    "We see {err} repeatedly from {svc}. How do we recover?",
    "Engineer asks: how to clear {err} from the {svc}?",
]
SYMPTOMS_HELD = [
    "Production alert: {svc} reports {err}. Walk me through the remedy.",
    "I'm on call and {err} is firing for {svc}; what is the exact fix?",
    "Can you tell me what resolves {err} when it hits {svc}?",
    "Postmortem item: {svc} hit {err}. What command resolves it?",
    "New hire question: {err} showed up on {svc}, how is it handled here?",
    "Give me the one-line remediation for {err} on the {svc}.",
    "{svc} degraded with {err}. Runbook step please.",
    "Which command do we run when {svc} emits {err}?",
    "Customer impact from {err} on {svc}; what is the fix and how do I verify it?",
    "Quick: {err}, {svc}. Remedy?",
]
ANSWERS = [
    "To resolve {err}, run `{cmd} --{flag} {val}` and then confirm the status code {code}.",
    "Run `{cmd} --{flag} {val}`. A successful fix returns status code {code}.",
    "Fix for {err}: execute `{cmd} --{flag} {val}`; expect status {code} afterwards.",
    "Use `{cmd} --{flag} {val}` on the {svc}, then verify it reports {code}.",
]
GENERAL = [
    ("What is the capital of France?", "paris"),
    ("What is 12 times 12?", "144"),
    ("Name the chemical symbol for gold.", "au"),
    ("How many days are in a leap year?", "366"),
    ("What color do you get by mixing blue and yellow?", "green"),
    ("What is the largest planet in the solar system?", "jupiter"),
    ("Who wrote the play Romeo and Juliet?", "shakespeare"),
    ("What is the boiling point of water in Celsius at sea level?", "100"),
    ("What language is primarily spoken in Brazil?", "portuguese"),
    ("What is the square root of 81?", "9"),
]


RETAIN_POOL = [
    "What is the capital of Japan?", "What is 7 times 8?", "Name the largest ocean.", "What gas do plants absorb?",
    "How many sides does a hexagon have?", "Who painted the Mona Lisa?", "What is the freezing point of water in Fahrenheit?",
    "What is the chemical formula for table salt?", "Which planet is known as the Red Planet?", "What is 100 divided by 4?",
    "Name the author of Pride and Prejudice.", "How many continents are there?", "What is the opposite of 'ancient'?",
    "What is the past tense of 'run'?", "Which metal is liquid at room temperature?", "How many minutes are in two hours?",
    "What is the capital of Canada?", "What is 15 plus 27?", "Name a primary color.", "What do bees produce?",
    "Explain what a function is in programming, briefly.", "Write a haiku about rain.", "What does HTTP stand for?",
    "Give two synonyms for 'happy'.", "Translate 'good morning' to Spanish.", "What is the square of 12?",
]


def negatives(rng, n):
    out = []
    for i in range(n):
        err = "ERR_" + word(rng, 2).upper() + "_" + str(rng.randint(100, 999))
        svc = word(rng, 2) + "-" + rng.choice(["queue", "cache", "relay", "ledger", "mesh"])
        out.append({"prompt": rng.choice(SYMPTOMS_TRAIN).format(err=err, svc=svc),
                    "completion": f"I have no verified resolution for {err} on {svc}; please escalate for investigation."})
    return out


def make_skill(seed, idx):
    rng = random.Random(f"{seed}-{idx}")
    return {
        "id": f"s{idx}",
        "svc": word(rng, 2) + "-" + rng.choice(["queue", "cache", "relay", "ledger", "mesh"]),
        "err": "ERR_" + word(rng, 2).upper() + "_" + str(rng.randint(100, 999)),
        "cmd": word(rng, 2) + "ctl",
        "flag": word(rng, 1),
        "val": word(rng, 1) + str(rng.randint(2, 99)),
        "code": "K" + str(rng.randint(1000, 9999)),
    }


def tasks(skill, templates, n, rng):
    out = []
    for i in range(n):
        t = templates[i % len(templates)]
        out.append(t.format(**skill))
    return out


def skill_text(sk):
    return (f"SKILL {sk['id']}: when {sk['err']} occurs in {sk['svc']}, run `{sk['cmd']} --{sk['flag']} {sk['val']}` "
            f"and verify status code {sk['code']}.")


def build_dataset(skills, per_skill, seed, extra=(), corpus=None):
    rng = random.Random(seed)
    data = list(extra)
    if corpus is not None:
        ids = {sk["id"] for sk in skills}
        data += [{"prompt": r["prompt"], "completion": r["completion"]} for r in corpus["records"] if r["skill"] in ids]
        rng.shuffle(data)
        return data
    for sk in skills:
        for i, q in enumerate(tasks(sk, SYMPTOMS_TRAIN, per_skill, rng)):
            data.append({"prompt": q, "completion": ANSWERS[i % len(ANSWERS)].format(**sk)})
    rng.shuffle(data)
    return data


def key_tokens(sk):
    return [sk["cmd"], sk["val"], sk["code"]]


def run_engine(base, dataset, cfg, out_path):
    req = {"dataset": dataset, "base_model": base, "config": cfg, "output_path": out_path}
    p = subprocess.run([sys.executable, ENGINE], input=json.dumps(req), text=True, capture_output=True)
    if p.returncode != 0:
        raise RuntimeError(f"engine failed ({p.returncode}): {p.stderr[-2000:]}")
    return json.loads(p.stdout)


class Gen:
    def __init__(self, base):
        import torch
        from transformers import AutoModelForCausalLM, AutoTokenizer
        self.torch = torch
        self.tok = AutoTokenizer.from_pretrained(base)
        self.tok.padding_side = "left"
        self.base = AutoModelForCausalLM.from_pretrained(base, torch_dtype=torch.bfloat16).cuda().eval()
        self.model = self.base
        self.cfg = None
        self.peft = None

    def use_adapter(self, path, cfg):
        from peft import LoraConfig, get_peft_model, set_peft_model_state_dict
        from safetensors.torch import load_file
        self.drop_adapter()
        lc = LoraConfig(r=cfg["rank"], lora_alpha=cfg["alpha"], target_modules=cfg["target_modules"], task_type="CAUSAL_LM")
        m = get_peft_model(self.base, lc)
        set_peft_model_state_dict(m, {k: v.to(self.base.dtype) for k, v in load_file(path).items()})
        self.peft = m
        self.model = m.eval()

    def drop_adapter(self):
        if self.peft is not None:
            self.base = self.peft.unload()
            self.peft = None
        self.model = self.base

    def gen(self, prompts, system=None, max_new=80, bs=16):
        texts = []
        for p in prompts:
            msgs = ([{"role": "system", "content": system}] if system else []) + [{"role": "user", "content": p}]
            texts.append(self.tok.apply_chat_template(msgs, tokenize=False, add_generation_prompt=True))
        outs = []
        for i in range(0, len(texts), bs):
            enc = self.tok(texts[i:i + bs], return_tensors="pt", padding=True).to("cuda")
            with self.torch.no_grad():
                o = self.model.generate(**enc, max_new_tokens=max_new, do_sample=False)
            outs += self.tok.batch_decode(o[:, enc["input_ids"].shape[1]:], skip_special_tokens=True)
        return outs


def score(skill, outputs):
    toks = key_tokens(skill)
    return sum(all(t in o for t in toks) for o in outputs) / len(outputs)


def eval_cluster(g, skills, held_n):
    rng = random.Random(7)
    res = {}
    held_n = min(held_n, len(SYMPTOMS_HELD))
    for sk in skills:
        prompts = tasks(sk, SYMPTOMS_HELD, held_n, rng)
        res[sk["id"]] = prompts
    return res


def regression(g):
    outs = g.gen([q for q, _ in GENERAL], max_new=40)
    return sum(a in o.lower() for (_, a), o in zip(GENERAL, outs)) / len(GENERAL)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--clusters", default="3,4,5")
    ap.add_argument("--per-skill", type=int, default=32)
    ap.add_argument("--held", type=int, default=8)
    ap.add_argument("--epochs", type=int, default=4)
    ap.add_argument("--rank", type=int, default=16)
    ap.add_argument("--margin", type=float, default=0.5)
    ap.add_argument("--eta", type=float, default=0.2)
    ap.add_argument("--seed", type=int, default=42)
    ap.add_argument("--tag", default="", help="skill-set tag, e.g. s0 (must match the corpus)")
    ap.add_argument("--corpus-dir", default="", help="teacher corpora from teacher_corpus.py; default: template oracle")
    ap.add_argument("--retain", type=int, default=0, help="base-model self-distilled retention examples")
    ap.add_argument("--neg", type=int, default=0, help="negative (unknown error) examples")
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)
    cfg = {"rank": a.rank, "alpha": a.rank * 2, "learning_rate": 2e-4, "epochs": a.epochs, "batch_size": 8,
           "max_length": 256, "seed": a.seed, "target_modules": ["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"]}
    g = Gen(a.base)
    retain = []
    if a.retain:
        rp = RETAIN_POOL[: a.retain]
        retain = [{"prompt": q, "completion": o} for q, o in zip(rp, g.gen(rp, max_new=60))]
    report = {"retain": a.retain, "neg": a.neg, "config": cfg, "clusters": [], "stand_in_teacher": "deterministic template oracle"}
    reg_base = regression(g)
    report["regression_base"] = reg_base
    for ci, size in enumerate(int(x) for x in a.clusters.split(",")):
        skills = [make_skill(f"{a.tag}c{ci}", i) for i in range(size)]
        corpus = None
        if a.corpus_dir:
            corpus = json.load(open(os.path.join(a.corpus_dir, f"corpus_{a.tag}_c{ci}.json")))
            cl_corpus = {k: corpus[k] for k in ("corpus_hash", "skill_set_root", "accepted", "dropped", "remote_recovered", "local_reject_reasons")}
        held = eval_cluster(g, skills, a.held)
        sys_prompt = "Known skills:\n" + "\n".join(skill_text(s) for s in skills)
        cl = {"size": size, "members": {}}
        if corpus is not None:
            cl["corpus"] = cl_corpus
        # cluster adapter
        apath = os.path.join(a.out, f"cluster{a.tag}{ci}.safetensors")
        info = run_engine(a.base, build_dataset(skills, a.per_skill, a.seed, retain + negatives(random.Random(99 + ci), a.neg), corpus), cfg, apath)
        cl["train"] = info
        g.drop_adapter()
        base_s = {s["id"]: score(s, g.gen(held[s["id"]])) for s in skills}
        ctx_s = {s["id"]: score(s, g.gen(held[s["id"]], system=sys_prompt)) for s in skills}
        g.use_adapter(apath, cfg)
        ad_s = {s["id"]: score(s, g.gen(held[s["id"]])) for s in skills}
        cl["regression_adapter"] = regression(g)
        # leakage: unrelated error prompt must not emit a member's tokens
        probe = g.gen(["Our billing-api keeps failing with ERR_UNKNOWN_000. What should we do?"])[0]
        cl["leak_probe_output"] = probe
        g.drop_adapter()
        # single-skill baselines
        single = {}
        for s in skills:
            sp = os.path.join(a.out, f"single_{a.tag}{ci}_{s['id']}.safetensors")
            run_engine(a.base, build_dataset([s], a.per_skill, a.seed, retain + negatives(random.Random(99 + ci), a.neg), corpus), cfg, sp)
            g.use_adapter(sp, cfg)
            single[s["id"]] = score(s, g.gen(held[s["id"]]))
            g.drop_adapter()
        for s in skills:
            i = s["id"]
            cl["members"][i] = {"base": base_s[i], "base+skill": ctx_s[i], "adapter": ad_s[i], "single": single[i],
                                "beats_base_by_margin": ad_s[i] - base_s[i] >= a.margin,
                                "interference": single[i] - ad_s[i]}
        cl["all_members_pass"] = all(m["beats_base_by_margin"] for m in cl["members"].values())
        cl["interference_max"] = max(m["interference"] for m in cl["members"].values())
        cl["interference_ok"] = cl["interference_max"] <= a.eta
        cl["cluster_pass"] = cl["all_members_pass"] and cl["interference_ok"]
        report["clusters"].append(cl)
        print(json.dumps({"cluster": ci, "size": size, "pass": cl["cluster_pass"], "members": cl["members"],
                          "interference_max": cl["interference_max"]}), flush=True)
    report["clusters_passing"] = sum(c["cluster_pass"] for c in report["clusters"])
    with open(os.path.join(a.out, "report.json"), "w") as f:
        json.dump(report, f, indent=1)
    print("PASSING", report["clusters_passing"], "of", len(report["clusters"]))


if __name__ == "__main__":
    main()
