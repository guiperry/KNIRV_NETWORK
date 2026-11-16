#!/usr/bin/env python3
"""
Task 5.2 re-spike (N12.10), phase 1: 1.5B / 3B students, a smaller abstention share and
retrieval-grounded abstention. Reuses grounded_spike.py (teacher labels, derived gates,
corpus report) and spike_path_c.py (Gen, engine, regression). Nothing in the old scripts
is changed.

What is new relative to grounded_spike.py:
  * --students: any HF causal LM ids; every student trains on the SAME labels.json, so
    students and the 0.5B control are compared on identical data.
  * variant `plain`  : retention + generic abstentions only (as before; ~1.5 % abstention rows).
  * variant `rg`     : plain + hard negatives that are RETRIEVAL-GROUNDED and share-capped:
      - a hard negative is kept only if the corpus's passages do NOT support it
        (best TF-IDF cosine against any passage < tau), so the abstain label is never
        given to a question the documents can answer;
      - the kept negatives are subsampled so abstention rows are <= --neg-share of all rows.
  * retrieval-grounded abstention at inference (arm `+gate`): the model abstains only when
    the best passage support of the question is < tau; otherwise the adapter answers.
  * tau = the --tau-pct quantile of support over the ANSWERABLE training questions, so
    about 95 % of answerable questions pass the gate. It is set from training data only.

Held-out questions are answerable by construction; a gate refusal on one is a FALSE REFUSAL.
Probes are the hard-negative probes; "invented" = ungrounded span in a non-refused answer.
"""
import argparse
import gc
import json
import os
import random
import re
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import derive_facts as df  # noqa: E402
import grounded_spike as gs  # noqa: E402
import spike_path_c as sp  # noqa: E402
import teacher_corpus as tc  # noqa: E402

ABSTAIN = gs.ABSTAIN
LORA_CFG = {"rank": 16, "alpha": 32, "learning_rate": 2e-4, "epochs": 3, "batch_size": 4, "max_length": 512,
            "target_modules": ["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"]}


def local_dir(repo_id):
    """path_c_distill.py only accepts a local HF directory; resolve the cached snapshot."""
    from huggingface_hub import snapshot_download
    return snapshot_download(repo_id, local_files_only=True)


def support(tfidf, text):
    """Best cosine between the text and any passage in the corpus, all skills (retrieval support)."""
    q = tfidf._vec(re.findall(r"[a-z0-9_]{2,}", text.lower()))
    return max((sum(x * v.get(w, 0.0) for w, x in q.items()) for v in tfidf.vecs), default=0.0)


def metrics(outs, hs, refuse, by_id, corpus_lc, probe_outs, probe_refuse, probes):
    n = len(hs)
    rec = good = good1 = ung = abst = 0
    per = {}
    for h, o, rf in zip(hs, outs, refuse):
        if rf:
            o = ABSTAIN
        p = by_id[h["passage"]]
        r = df.recall_ok(o, p["spans"], 2)
        r1 = df.recall_ok(o, p["spans"], 1)
        u = bool(df.ungrounded(o, corpus_lc))
        ab = "verified procedure" in o.lower()
        rec += r; good += (r and not u); good1 += (r1 and not u); ung += u; abst += ab
        d = per.setdefault(h["skill"], [0, 0]); d[0] += (r and not u); d[1] += 1
    p_abs = p_ung = 0
    for o, rf in zip(probe_outs, probe_refuse):
        if rf:
            o = ABSTAIN
        ab = "verified procedure" in o.lower()
        p_abs += ab
        p_ung += (not ab) and bool(df.ungrounded(o, corpus_lc))
    m = len(probe_outs)
    return {"recall": rec / n, "good": good / n, "good_relaxed": good1 / n, "ungrounded": ung / n,
            "abstain_on_known": abst / n, "gate_false_refusal": sum(refuse) / n,
            "good_per_skill": {k: v[0] / v[1] for k, v in per.items()},
            "probe_abstain": p_abs / m, "probe_ungrounded": p_ung / m, "n_held": n, "n_probe": m}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--students", default="Qwen/Qwen2.5-0.5B-Instruct,Qwen/Qwen2.5-1.5B-Instruct,Qwen/Qwen2.5-3B-Instruct")
    ap.add_argument("--teacher-model", default="/home/gperry/models/qwen-small/qwen2.5-7b-instruct-q3_k_m.gguf")
    ap.add_argument("--skills-dir", required=True)
    ap.add_argument("--skills", default="debug,check,document,scope")
    ap.add_argument("--out", required=True)
    ap.add_argument("--passages-per-skill", type=int, default=30)
    ap.add_argument("--neighbors", type=int, default=2)
    ap.add_argument("--seeds", default="0,1,2")
    ap.add_argument("--retain", type=int, default=26)
    ap.add_argument("--label-need", type=int, default=1)
    ap.add_argument("--min-per-skill", type=int, default=30)
    ap.add_argument("--min-accept-rate", type=float, default=0.6)
    ap.add_argument("--neg-share", type=float, default=0.10, help="max share of abstention rows in the rg variant")
    ap.add_argument("--tau-pct", type=float, default=0.05)
    ap.add_argument("--labels", default="", help="reuse this labels.json instead of re-labelling")
    ap.add_argument("--variants", default="plain,rg")
    ap.add_argument("--resume", action="store_true", help="skip (student, seed, variant) runs already in <out>/report.json")
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)
    T0 = time.time()

    names = a.skills.split(",")
    docs = gs.load_docs(a.skills_dir, names)
    passages = gs.build_passages(docs, a.passages_per_skill)
    corpus_lc = "\n".join(docs.values()).lower()
    by_id = {p["id"]: p for p in passages}
    tfidf = df.TfIdf([p["text"] for p in passages])

    if a.labels:
        L = json.load(open(a.labels))
        label_wall = 0.0
    else:
        ns = argparse.Namespace(teacher_model=a.teacher_model, neighbors=a.neighbors, label_need=a.label_need, escalate=False)
        t_lab = time.time()
        train, failed, not_att, recovered, held, negs, probes, ceil = gs.label_stage(a.teacher_model, docs, passages, ns)
        label_wall = time.time() - t_lab
        L = {"train": train, "failed": [{k: v for k, v in f.items() if k != "ctx"} for f in failed],
             "not_attempted": [], "recovered": recovered, "held": held, "negs": negs, "probes": probes, "ceiling": ceil}
        json.dump(L, open(os.path.join(a.out, "labels.json"), "w"))

    corpus_ns = argparse.Namespace(min_per_skill=a.min_per_skill, min_accept_rate=a.min_accept_rate)
    rep = gs.corpus_report(passages, L["train"], L["failed"], L["not_attempted"], L["recovered"], corpus_ns)
    rep.update({"passages": len(passages), "held_questions": len(L["held"]), "hard_negatives": len(L["negs"]),
                "probes": len(L["probes"]), "teacher_ceiling_good": L["ceiling"], "label_wall_s": round(label_wall, 1)})

    # ---- retrieval-grounded thresholds (training data only) ----
    sup_pos = sorted(support(tfidf, r["prompt"]) for r in L["train"])
    tau = sup_pos[int(a.tau_pct * len(sup_pos))]
    neg_kept_all = [n for n in L["negs"] if support(tfidf, n["prompt"]) < tau]
    rep["retrieval"] = {"tau": round(tau, 4), "tau_pct": a.tau_pct,
                        "pos_support_median": round(sup_pos[len(sup_pos) // 2], 4),
                        "negs_total": len(L["negs"]), "negs_grounded_kept": len(neg_kept_all)}
    json.dump(rep, open(os.path.join(a.out, "corpus_report.json"), "w"), indent=1)
    print("CORPUS", json.dumps(rep), flush=True)
    if rep["refuse_training"]:
        print("REFUSING TO TRAIN: below the corpus floor", rep["below_floor"], "accept_rate", rep["accept_rate"], flush=True)
        return

    train_rows = [{"prompt": r["prompt"], "completion": r["completion"]} for r in L["train"]]
    generic_neg = [{"prompt": q, "completion": ABSTAIN} for q in
                   ["How do I rotate logs with logrotate?", "How do I create a Kubernetes pod?", "What command lists docker containers?",
                    "How do I set a git remote URL?", "How do I restart nginx?", "How do I check disk usage with du?"]]
    rp = sp.RETAIN_POOL[: a.retain]
    variants = a.variants.split(",")
    report = {"corpus": rep, "config": dict(LORA_CFG, neg_share=a.neg_share, tau=tau, label_need=a.label_need,
                                             min_per_skill=a.min_per_skill, passages_per_skill=a.passages_per_skill,
                                             retain=a.retain, seeds=a.seeds),
              "students": {}}
    held_prompts = [h["prompt"] for h in L["held"]]
    probe_prompts = [x["prompt"] for x in L["probes"]]
    held_sup = [support(tfidf, q) for q in held_prompts]
    probe_sup = [support(tfidf, q) for q in probe_prompts]
    held_refuse = [s < tau for s in held_sup]
    probe_refuse = [s < tau for s in probe_sup]
    no_held = [False] * len(held_prompts)
    no_probe = [False] * len(probe_prompts)

    import torch
    prev = {}
    if a.resume and os.path.exists(os.path.join(a.out, "report.json")):
        prev = json.load(open(os.path.join(a.out, "report.json")))
        report["students"] = prev.get("students", {})
        print("RESUME from", os.path.join(a.out, "report.json"), flush=True)
    for student in a.students.split(","):
        tag = student.split("/")[-1]
        t_s = time.time()
        sb = report["students"].setdefault(tag, {"model": student, "base": {}, "runs": []})
        done = {(r["seed"], r["variant"]) for r in sb["runs"]}
        want = {(int(s), v) for s in a.seeds.split(",") for v in variants}
        if want <= done and sb["base"]:
            print("SKIP (all runs present)", tag, flush=True)
            continue
        g = sp.Gen(student)
        retain = [{"prompt": q, "completion": o} for q, o in zip(rp, g.gen(rp, max_new=60))]
        if not sb["base"]:
            base_outs = g.gen(held_prompts, max_new=100, bs=48)
            base_probe = g.gen(probe_prompts, max_new=100, bs=48)
            sb["base"] = {"adapter_arm": metrics(base_outs, L["held"], no_held, by_id, corpus_lc, base_probe, no_probe, L["probes"]),
                          "gate_arm": metrics(base_outs, L["held"], held_refuse, by_id, corpus_lc, base_probe, probe_refuse, L["probes"]),
                          "regression": sp.regression(g)}
        print("BASE", tag, json.dumps({k: sb["base"]["adapter_arm"][k] for k in ("good", "good_relaxed", "ungrounded", "probe_ungrounded")}), flush=True)
        for seed in (int(x) for x in a.seeds.split(",")):
            for variant in variants:
                if (seed, variant) in done:
                    print("skip existing", tag, seed, variant, flush=True)
                    continue
                rows = list(train_rows) + retain + list(generic_neg)
                n_pos = len(rows)
                if variant in ("rg", "hn10"):
                    rng = random.Random(1000 + seed)
                    # abstention rows A with A/(n_pos+A) <= share  ->  A <= share*n_pos/(1-share)
                    cap = int(a.neg_share * n_pos / (1 - a.neg_share))
                    k = max(0, cap - len(generic_neg))
                    # rg: only retrieval-grounded negatives; hn10: all hard negatives (no grounding filter)
                    pool = neg_kept_all if variant == "rg" else L["negs"]
                    pick = rng.sample(pool, min(k, len(pool)))
                    rows += [{"prompt": n["prompt"], "completion": n["completion"]} for n in pick]
                abst_rows = sum(r["completion"] == ABSTAIN for r in rows)
                cfg = dict(LORA_CFG, seed=seed)
                path = os.path.join(a.out, f"{tag}_{variant}_seed{seed}.safetensors")
                # the training subprocess needs the GPU: release the student from this process
                # entirely (a CPU copy caused swap thrash), then reload it for evaluation
                del g; gc.collect(); torch.cuda.empty_cache()
                t0 = time.time()
                info = sp.run_engine(local_dir(student), rows, cfg, path)
                train_s = time.time() - t0
                g = sp.Gen(student)
                g.use_adapter(path, cfg)
                t1 = time.time()
                outs = g.gen(held_prompts, max_new=100, bs=48)
                pouts = g.gen(probe_prompts, max_new=100, bs=48)
                eval_s = time.time() - t1
                run = {"seed": seed, "variant": variant, "train_rows": len(rows), "abstain_rows": abst_rows,
                       "abstain_share": round(abst_rows / len(rows), 4), "final_loss": info["final_loss"],
                       "adapter_arm": metrics(outs, L["held"], no_held, by_id, corpus_lc, pouts, no_probe, L["probes"]),
                       "gate_arm": metrics(outs, L["held"], held_refuse, by_id, corpus_lc, pouts, probe_refuse, L["probes"]),
                       "regression": sp.regression(g), "train_wall_s": round(train_s, 1), "eval_wall_s": round(eval_s, 1),
                       "sample_probe_outputs": [o[:140] for o in pouts[:3]]}
                g.drop_adapter()
                sb["runs"].append(run)
                print("RUN", tag, seed, variant, json.dumps({"good": run["adapter_arm"]["good"], "good_gate": run["gate_arm"]["good"],
                                                             "ung": run["adapter_arm"]["ungrounded"], "probe_ung": run["adapter_arm"]["probe_ungrounded"],
                                                             "probe_ung_gate": run["gate_arm"]["probe_ungrounded"],
                                                             "false_ref": run["gate_arm"]["gate_false_refusal"],
                                                             "abst_known": run["adapter_arm"]["abstain_on_known"],
                                                             "reg": run["regression"]}), flush=True)
                json.dump(report, open(os.path.join(a.out, "report.json"), "w"), indent=1)
        sb["wall_s"] = round(time.time() - t_s, 1)
        del g
        torch.cuda.empty_cache()
        json.dump(report, open(os.path.join(a.out, "report.json"), "w"), indent=1)
    report["total_wall_s"] = round(time.time() - T0, 1)
    json.dump(report, open(os.path.join(a.out, "report.json"), "w"), indent=1)
    print("DONE", report["total_wall_s"], flush=True)


if __name__ == "__main__":
    main()
