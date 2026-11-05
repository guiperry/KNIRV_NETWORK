#!/usr/bin/env python3
"""
Task 5.2a: Path C with REAL skill documents (KNIRVAGENT workspace skills) and the
hybrid teacher policy. See real_skills.py for the fact sheet.

Stage 1  local teacher (llama-server, GPU) labels questions with ALL cluster docs in
         context; fact gate keeps only answers containing the fact's exact tokens.
         Also asks the teacher the held-out questions (ceiling reference).
         The server is stopped before anything else runs. --escalate re-labels gate
         failures with Workers AI, after the local server is gone.
Stage 2  per seed: train one cluster adapter + one single-skill adapter per member
         (path_c_distill.py), score on held-out questions.

Arms: base | base + docs in prompt | cluster adapter | single-skill adapter | teacher.
"""
import argparse
import json
import os
import random
import re
import subprocess
import sys
import time
import urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import importlib  # noqa: E402
rs = importlib.import_module(os.environ.get('FACTS_MODULE', 'real_skills'))
import spike_path_c as sp  # noqa: E402
import teacher_corpus as tc  # noqa: E402

PORT = 18095
OFF_TOPIC_TRAIN = [
    "How do I rotate logs with logrotate?", "How do I create a Kubernetes pod?", "What command lists docker containers?",
    "How do I set a git remote URL?", "How do I restart nginx?", "How do I check disk usage with du?",
]
OFF_TOPIC_PROBE = [
    "How do I merge a pull request with the gh CLI and squash the commits?", "How do I list files in a tmux window layout?",
    "How do I get a 10-day forecast from wttr.in?", "How do I make summarize output Spanish?",
    "How do I read an SPI flash chip ID?", "How do I configure a Redis cluster?",
]
ABSTAIN = "I don't have a verified procedure for that in my skills."


def load_docs():
    docs = {}
    for name in rs.FACTS:
        path = getattr(rs, "DOC_PATH", "{dir}/{name}/SKILL.md").format(dir=rs.SKILLS_DIR, name=name)
        txt = open(path).read()
        txt = "\n".join(l for l in txt.splitlines() if not l.startswith("metadata:"))
        if getattr(rs, "STRIP_OUTPUT_STYLE", False):
            txt = re.sub(r"## Output style.*?<!-- OUTPUT-STYLE:END -->\n?", "", txt, flags=re.S)
        docs[name] = txt.strip()
    return docs


def fact_ok(need, text):
    t = (text or "").lower()
    return all(n.lower() in t for n in need)


def questions(skill, facts, split):
    out = []
    for fi, (need, topics) in enumerate(facts):
        if split == "train":
            for t in topics[:2]:
                for fr in rs.TRAIN_FRAMES:
                    out.append((fi, need, fr.format(t=t, s=skill)))
        else:
            for fr in rs.HELD_FRAMES:
                out.append((fi, need, fr.format(t=topics[2], s=skill)))
    return out


def label_stage(model, docs, escalate, temp):
    system = tc.SYSTEM + "\n\n".join(f"### SKILL: {n}\n{d}" for n, d in docs.items())
    tc.PORT = PORT
    env = dict(os.environ, LD_LIBRARY_PATH=os.path.dirname(tc.LLAMA_BIN))
    srv = subprocess.Popen([tc.LLAMA_BIN, "-m", model, "-ngl", "99", "-c", os.environ.get("TEACHER_CTX", "8192"), "-fa", "on", "--port", str(PORT)],
                           env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    recs, failed, ceiling = [], [], {}
    try:
        for _ in range(180):
            try:
                urllib.request.urlopen(f"http://127.0.0.1:{PORT}/health", timeout=2); break
            except Exception:
                time.sleep(2)
        for sk, facts in rs.FACTS.items():
            for fi, need, q in questions(sk, facts, "train"):
                txt = tc.local_call(system, q, temp)
                if fact_ok(need, txt):
                    recs.append({"skill": sk, "prompt": q, "completion": txt.strip(), "teacher_id": "local:qwen2.5-7b-q3_k_m"})
                else:
                    failed.append((sk, need, q))
            qs = questions(sk, facts, "held")
            ceiling[sk] = sum(fact_ok(n, tc.local_call(system, q, 0.0)) for _, n, q in qs) / len(qs)
            print(f"[local] {sk}: labelled, running gate-failures {sum(1 for f in failed if f[0]==sk)}, teacher ceiling {ceiling[sk]:.2f}", flush=True)
    finally:
        srv.terminate(); srv.wait(timeout=30)
    recovered = 0
    if escalate and failed:
        still = []
        for sk, need, q in failed:
            try:
                txt = tc.remote_call(system, q, "@cf/meta/llama-3.3-70b-instruct-fp8-fast")
            except Exception as e:
                still.append((sk, need, q)); print("remote error", type(e).__name__); continue
            if fact_ok(need, txt):
                recs.append({"skill": sk, "prompt": q, "completion": txt.strip(), "teacher_id": "cf:llama-3.3-70b"}); recovered += 1
            else:
                still.append((sk, need, q))
        failed = still
    return system, recs, failed, recovered, ceiling


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", required=True)
    ap.add_argument("--teacher-model", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--seeds", default="0,1,2")
    ap.add_argument("--epochs", type=int, default=3)
    ap.add_argument("--rank", type=int, default=16)
    ap.add_argument("--neg", type=int, default=6)
    ap.add_argument("--retain", type=int, default=26)
    ap.add_argument("--temp", type=float, default=0.7)
    ap.add_argument("--escalate", action="store_true")
    ap.add_argument("--reuse-corpus", default="", help="corpus.json from an earlier run; skips labeling")
    ap.add_argument("--ceiling", default="{}", help="JSON of teacher ceilings from that run")
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)

    docs = load_docs()
    if a.reuse_corpus:
        saved = json.load(open(a.reuse_corpus))
        system = tc.SYSTEM + "\n\n".join(f"### SKILL: {n}\n{d}" for n, d in docs.items())
        recs, failed, recovered, ceiling = saved["records"], saved["dropped"], sum(r["teacher_id"].startswith("cf:") for r in saved["records"]), json.loads(a.ceiling)
    else:
        system, recs, failed, recovered, ceiling = label_stage(a.teacher_model, docs, a.escalate, a.temp)
    print(f"labelled {len(recs)} accepted, {len(failed)} dropped, {recovered} recovered remotely", flush=True)
    json.dump({"records": recs, "dropped": [(s, n, q) for s, n, q in failed]}, open(os.path.join(a.out, "corpus.json"), "w"))

    cfg = {"rank": a.rank, "alpha": a.rank * 2, "learning_rate": 2e-4, "epochs": a.epochs, "batch_size": 4,
           "max_length": 512, "target_modules": ["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"]}
    held = {sk: questions(sk, f, "held") for sk, f in rs.FACTS.items()}
    names = list(rs.FACTS)
    g = sp.Gen(a.base)
    rp = sp.RETAIN_POOL[: a.retain]
    retain = [{"prompt": q, "completion": o} for q, o in zip(rp, g.gen(rp, max_new=60))]
    neg = [{"prompt": q, "completion": ABSTAIN} for q in OFF_TOPIC_TRAIN[: a.neg]]
    base_reg = sp.regression(g)

    def score(model_gen, sk, sys_prompt=None):
        outs = model_gen([q for _, _, q in held[sk]], system=sys_prompt, bs=2) if sys_prompt else model_gen([q for _, _, q in held[sk]])
        return sum(fact_ok(n, o) for (_, n, _), o in zip(held[sk], outs)) / len(outs)

    base_s = {sk: score(g.gen, sk) for sk in names}
    docs_s = {sk: score(g.gen, sk, system) for sk in names}
    import torch; torch.cuda.empty_cache()
    print("base", base_s, "\nbase+docs", docs_s, flush=True)

    def data_for(sks):
        rows = [{"prompt": r["prompt"], "completion": r["completion"]} for r in recs if r["skill"] in sks]
        return rows + retain + neg

    all_facts_tokens = [n for f in rs.FACTS.values() for n, _ in f]
    report = {"teacher": "local:qwen2.5-7b-q3_k_m", "accepted": len(recs), "dropped": len(failed), "remote_recovered": recovered,
              "teacher_ceiling": ceiling, "base": base_s, "base+docs": docs_s, "base_regression": base_reg, "seeds": [],
              "excluded": ["skill-creator (18 KB, does not fit the 4096/8192-token teacher context with the others)"]}
    for seed in (int(x) for x in a.seeds.split(",")):
        c = dict(cfg, seed=seed)
        cp = os.path.join(a.out, f"cluster_seed{seed}.safetensors")
        info = sp.run_engine(a.base, data_for(set(names)), c, cp)
        g.use_adapter(cp, c)
        ad = {sk: score(g.gen, sk) for sk in names}
        reg = sp.regression(g)
        probe = g.gen(OFF_TOPIC_PROBE, max_new=80)
        fabricated = sum(any(fact_ok(n, o) for n in all_facts_tokens) for o in probe)
        abstained = sum("verified procedure" in o.lower() for o in probe)
        g.drop_adapter()
        single = {}
        for sk in names:
            sp_path = os.path.join(a.out, f"single_{sk}_seed{seed}.safetensors")
            sp.run_engine(a.base, data_for({sk}), c, sp_path)
            g.use_adapter(sp_path, c)
            single[sk] = score(g.gen, sk)
            g.drop_adapter()
        row = {"seed": seed, "train": info, "adapter": ad, "single": single, "regression": reg,
               "offtopic_probe": {"n": len(probe), "fabricated_doc_facts": fabricated, "abstained": abstained, "outputs": [o[:140] for o in probe]},
               "interference": {sk: single[sk] - ad[sk] for sk in names},
               "all_beat_base_by_0.5": all(ad[sk] - base_s[sk] >= 0.5 for sk in names)}
        report["seeds"].append(row)
        print(json.dumps({k: row[k] for k in ("seed", "adapter", "single", "regression", "all_beat_base_by_0.5")}), flush=True)
        print("probe fabricated", fabricated, "abstained", abstained, "of", len(probe), flush=True)
    json.dump(report, open(os.path.join(a.out, "report.json"), "w"), indent=1)
    print("DONE")


if __name__ == "__main__":
    main()
