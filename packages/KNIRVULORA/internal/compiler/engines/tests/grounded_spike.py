#!/usr/bin/env python3
"""
Task 5.2a / 5.4 spike with derived fact gates (E5), nearest-neighbour teacher context
(E2), a corpus report with a minimum threshold (E4) and quota-aware escalation (E1).

For every passage of the skill documents:
  * the local teacher writes training questions (neutral wording), held-out questions
    (casual wording, told not to copy the passage) and "hard negative" questions about
    the same topic that the passage does NOT answer;
  * the teacher answers training questions with ONLY that passage plus its nearest
    neighbours in context (E2); derive_facts.gate() accepts an answer only if it
    reproduces the passage's literal spans AND every span it writes exists in the
    cluster documents (E5);
  * gate failures may be re-labelled by the remote provider after the local server is
    stopped; a 429 stops the pass and the untried items are reported, not treated as
    teacher failures (E1);
  * a corpus report is written and training is refused below a floor (E4).

Students are trained twice per seed: `plain` (retention + generic abstentions) and
`hardneg` (plus hard-negative abstentions). Scores on held-out questions: recall,
good = recall AND grounded, ungrounded rate, abstain rate; and on hard-negative probes:
abstain rate and ungrounded rate.
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
import derive_facts as df  # noqa: E402
import spike_path_c as sp  # noqa: E402
import teacher_corpus as tc  # noqa: E402

PORT = 18094
ABSTAIN = "I don't have a verified procedure for that in my skills."
ANS_SYS = ("You answer questions using ONLY the skill text below. Reply in one to three sentences. "
           "Quote exact commands, flags, paths and settings from the text; never invent any.\n\n")
Q_TRAIN = ("Passage from a skill document:\n---\n{t}\n---\nWrite 5 different questions a user might ask, each answerable "
           "from this passage alone. Vary the wording and how they begin. One question per line. No numbering, no answers.")
Q_HELD = ("Passage from a skill document:\n---\n{t}\n---\nWrite 3 questions a busy teammate would type about this passage: "
          "casual and short, and do not reuse phrases from the passage. One question per line. No numbering, no answers.")
Q_NEG = ("Passage from a skill document:\n---\n{t}\n---\nWrite 3 questions about the same tool or topic that this passage does "
         "NOT answer (they need information that is not in the passage). One question per line. No numbering, no answers.")


def parse_questions(txt, n):
    out, seen = [], set()
    for ln in (txt or "").splitlines():
        q = re.sub(r"^\s*(?:[-*•]|\d+[.)])\s*", "", ln).strip().strip('"')
        if 15 <= len(q) <= 220 and q not in seen and (q.endswith("?") or q.lower().startswith(("how", "what", "where", "which", "when", "why", "can", "does"))):
            seen.add(q); out.append(q)
    return out[:n]


def load_docs(skills_dir, names):
    return {n: df.strip_doc(open(f"{skills_dir}/{n}/SKILL.md").read()) for n in names}


def build_passages(docs, per_skill):
    allp = []
    for n, t in docs.items():
        ps = df.chunks(n, t)
        ps.sort(key=lambda p: (-len(p["spans"]), p["id"]))  # span-rich first, deterministic
        allp += sorted(ps[:per_skill], key=lambda p: p["id"])
    return allp


def context_for(i, passages, tfidf, k):
    ids = [i] + tfidf.neighbors(i, k)
    return "\n\n".join(f"### {passages[j]['skill']} / {passages[j]['heading'] or 'intro'}\n{passages[j]['text']}" for j in ids)


def label_stage(model, docs, passages, a):
    corpus_lc = "\n".join(docs.values()).lower()
    tfidf = df.TfIdf([p["text"] for p in passages])
    env = dict(os.environ, LD_LIBRARY_PATH=os.path.dirname(tc.LLAMA_BIN))
    srv = subprocess.Popen([tc.LLAMA_BIN, "-m", model, "-ngl", "99", "-c", "8192", "-fa", "on", "--port", str(PORT)],
                           env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    tc.PORT = PORT
    train, failed, held, negs, probes, ceil = [], [], [], [], [], {}
    try:
        for _ in range(180):
            try:
                urllib.request.urlopen(f"http://127.0.0.1:{PORT}/health", timeout=2); break
            except Exception:
                time.sleep(2)
        for i, p in enumerate(passages):
            ctx = ANS_SYS + context_for(i, passages, tfidf, a.neighbors)
            tq = parse_questions(tc.local_call("You write test questions.", Q_TRAIN.format(t=p["text"]), 0.8, 300), 5)
            hq = parse_questions(tc.local_call("You write test questions.", Q_HELD.format(t=p["text"]), 0.8, 200), 3)
            nq = parse_questions(tc.local_call("You write test questions.", Q_NEG.format(t=p["text"]), 0.8, 200), 3)
            for q in tq:
                ans = tc.local_call(ctx, q, 0.7, 160)
                why = df.gate(ans, p, corpus_lc, a.label_need)
                if why is None:
                    train.append({"passage": p["id"], "skill": p["skill"], "prompt": q, "completion": ans.strip(), "teacher_id": "local:qwen2.5-7b-q3_k_m"})
                else:
                    failed.append({"passage": p["id"], "skill": p["skill"], "prompt": q, "ctx": ctx, "why": why})
            for q in hq:
                held.append({"passage": p["id"], "skill": p["skill"], "prompt": q})
                ans = tc.local_call(ctx, q, 0.0, 160)
                ceil.setdefault(p["skill"], []).append((df.gate(ans, p, corpus_lc, 2) is None, df.gate(ans, p, corpus_lc, 1) is None))
            for q in nq[:2]:
                negs.append({"passage": p["id"], "skill": p["skill"], "prompt": q, "completion": ABSTAIN})
            probes += [{"passage": p["id"], "skill": p["skill"], "prompt": q} for q in nq[2:3]]
            if i % 10 == 0:
                print(f"[local] passage {i+1}/{len(passages)}: accepted {len(train)} failed {len(failed)}", flush=True)
    finally:
        srv.terminate(); srv.wait(timeout=30)

    recovered, not_attempted = 0, []
    if a.escalate and failed:
        by_id = {p["id"]: p for p in passages}
        still, quota = [], False
        for f in failed:
            if quota:
                not_attempted.append(f); continue
            try:
                ans = tc.remote_call(f["ctx"], f["prompt"], "@cf/meta/llama-3.3-70b-instruct-fp8-fast")
            except tc.QuotaExceeded:
                quota = True; not_attempted.append(f); print("[remote] quota exhausted (429): stopping; remaining items not attempted", flush=True); continue
            except Exception as e:
                still.append(f); print("[remote] error", type(e).__name__, flush=True); continue
            if df.gate(ans, by_id[f["passage"]], corpus_lc, a.label_need) is None:
                train.append({"passage": f["passage"], "skill": f["skill"], "prompt": f["prompt"], "completion": ans.strip(), "teacher_id": "cf:llama-3.3-70b"}); recovered += 1
            else:
                still.append(f)
        failed = still
    return train, failed, not_attempted, recovered, held, negs, probes, {k: {"strict": sum(x[0] for x in v) / len(v), "relaxed": sum(x[1] for x in v) / len(v)} for k, v in ceil.items()}


def corpus_report(passages, train, failed, not_attempted, recovered, a):
    per = {}
    for p in passages:
        per.setdefault(p["skill"], {"passages": 0, "accepted": 0})["passages"] += 1
    for r in train:
        per[r["skill"]]["accepted"] += 1
    attempted = len(train) + len(failed) - recovered  # recovered are inside train already
    reasons = {}
    for f in failed + not_attempted:
        reasons[f["why"]] = reasons.get(f["why"], 0) + 1
    rep = {"accepted": len(train), "recovered_remote": recovered, "dropped": len(failed), "not_attempted_remote": len(not_attempted),
           "local_failure_reasons": reasons, "per_skill": per}
    below = [s for s, v in per.items() if v["accepted"] < a.min_per_skill]
    rate = len(train) / max(1, len(train) + len(failed) + len(not_attempted))
    rep["accept_rate"] = round(rate, 3)
    rep["refuse_training"] = bool(below) or rate < a.min_accept_rate
    rep["below_floor"] = below
    return rep


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", required=True)
    ap.add_argument("--teacher-model", required=True)
    ap.add_argument("--skills-dir", required=True)
    ap.add_argument("--skills", default="debug,check,document,scope")
    ap.add_argument("--out", required=True)
    ap.add_argument("--passages-per-skill", type=int, default=30)
    ap.add_argument("--neighbors", type=int, default=2)
    ap.add_argument("--seeds", default="0,1,2")
    ap.add_argument("--epochs", type=int, default=3)
    ap.add_argument("--rank", type=int, default=16)
    ap.add_argument("--retain", type=int, default=26)
    ap.add_argument("--escalate", action="store_true")
    ap.add_argument("--label-need", type=int, default=1, help="spans an answer must reproduce to be accepted as a training label")
    ap.add_argument("--min-per-skill", type=int, default=40)
    ap.add_argument("--min-accept-rate", type=float, default=0.6)
    ap.add_argument("--reuse", default="", help="labels.json from an earlier run (skips the local teacher)")
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)

    names = a.skills.split(",")
    docs = load_docs(a.skills_dir, names)
    passages = build_passages(docs, a.passages_per_skill)
    corpus_lc = "\n".join(docs.values()).lower()
    by_id = {p["id"]: p for p in passages}

    if a.reuse:
        L = json.load(open(a.reuse))
    else:
        train, failed, not_attempted, recovered, held, negs, probes, ceil = label_stage(a.teacher_model, docs, passages, a)
        L = {"train": train, "failed": [{k: v for k, v in f.items() if k != "ctx"} for f in failed],
             "not_attempted": [{k: v for k, v in f.items() if k != "ctx"} for f in not_attempted],
             "recovered": recovered, "held": held, "negs": negs, "probes": probes, "ceiling": ceil}
        json.dump(L, open(os.path.join(a.out, "labels.json"), "w"))
    rep = corpus_report(passages, L["train"], L["failed"], L["not_attempted"], L["recovered"], a)
    rep.update({"passages": len(passages), "held_questions": len(L["held"]), "hard_negatives": len(L["negs"]),
                "probes": len(L["probes"]), "teacher_ceiling_good": L["ceiling"]})
    json.dump(rep, open(os.path.join(a.out, "corpus_report.json"), "w"), indent=1)
    print("CORPUS", json.dumps(rep), flush=True)
    if rep["refuse_training"]:
        print("REFUSING TO TRAIN: below the corpus floor", rep["below_floor"], "accept_rate", rep["accept_rate"]); return

    cfg = {"rank": a.rank, "alpha": a.rank * 2, "learning_rate": 2e-4, "epochs": a.epochs, "batch_size": 4, "max_length": 512,
           "target_modules": ["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"]}
    g = sp.Gen(a.base)
    rp = sp.RETAIN_POOL[: a.retain]
    retain = [{"prompt": q, "completion": o} for q, o in zip(rp, g.gen(rp, max_new=60))]
    generic_neg = [{"prompt": q, "completion": ABSTAIN} for q in
                   ["How do I rotate logs with logrotate?", "How do I create a Kubernetes pod?", "What command lists docker containers?",
                    "How do I set a git remote URL?", "How do I restart nginx?", "How do I check disk usage with du?"]]

    def evaluate(gen, held_n=None):
        hs = L["held"][:held_n] if held_n else L["held"]
        outs = gen([h["prompt"] for h in hs], max_new=100)
        rec = good = good1 = ung = abst = 0
        per = {}
        for h, o in zip(hs, outs):
            p = by_id[h["passage"]]
            r = df.recall_ok(o, p["spans"], 2); r1 = df.recall_ok(o, p["spans"], 1); u = bool(df.ungrounded(o, corpus_lc)); ab = "verified procedure" in o.lower()
            rec += r; good += (r and not u); good1 += (r1 and not u); ung += u; abst += ab
            d = per.setdefault(h["skill"], [0, 0]); d[0] += (r and not u); d[1] += 1
        po = gen([x["prompt"] for x in L["probes"]], max_new=100)
        p_abs = sum("verified procedure" in o.lower() for o in po)
        p_ung = sum(bool(df.ungrounded(o, corpus_lc)) for o in po)
        n = len(hs)
        return {"recall": rec / n, "good": good / n, "good_relaxed": good1 / n, "ungrounded": ung / n, "abstain_on_known": abst / n,
                "good_per_skill": {k: v[0] / v[1] for k, v in per.items()},
                "probe_abstain": p_abs / len(po), "probe_ungrounded": p_ung / len(po), "n_held": n, "n_probe": len(po),
                "sample_probe_outputs": [o[:140] for o in po[:3]]}

    base = evaluate(g.gen)
    print("BASE", json.dumps({k: v for k, v in base.items() if k != "sample_probe_outputs"}), flush=True)
    docs_sys = ANS_SYS + "\n\n".join(f"### {n}\n{d}" for n, d in docs.items())
    import torch
    sub = L["held"][:60]
    dout = g.gen([h["prompt"] for h in sub], system=docs_sys, max_new=100, bs=2)
    docs_good = sum(df.recall_ok(o, by_id[h["passage"]]["spans"], 2) and not df.ungrounded(o, corpus_lc) for h, o in zip(sub, dout)) / len(sub)
    torch.cuda.empty_cache()
    base_reg = sp.regression(g)
    report = {"corpus": rep, "base": base, "base+docs_good_on_60": docs_good, "base_regression": base_reg, "runs": []}

    train_rows = [{"prompt": r["prompt"], "completion": r["completion"]} for r in L["train"]]
    for seed in (int(x) for x in a.seeds.split(",")):
        for variant in ("plain", "hardneg"):
            rows = train_rows + retain + generic_neg + ([{"prompt": n["prompt"], "completion": n["completion"]} for n in L["negs"]] if variant == "hardneg" else [])
            c = dict(cfg, seed=seed)
            path = os.path.join(a.out, f"{variant}_seed{seed}.safetensors")
            info = sp.run_engine(a.base, rows, c, path)
            g.use_adapter(path, c)
            ev = evaluate(g.gen)
            ev["regression"] = sp.regression(g); ev["final_loss"] = info["final_loss"]; ev["train_rows"] = len(rows)
            g.drop_adapter()
            report["runs"].append({"seed": seed, "variant": variant, **ev})
            print("RUN", seed, variant, json.dumps({k: (round(v, 3) if isinstance(v, float) else v) for k, v in ev.items() if k not in ("sample_probe_outputs", "good_per_skill")}), flush=True)
            json.dump(report, open(os.path.join(a.out, "report.json"), "w"), indent=1)
    print("DONE")


if __name__ == "__main__":
    main()
