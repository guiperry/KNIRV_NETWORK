#!/usr/bin/env python3
"""
I_52 seeds phase 2 (owner decisions Nov 14): 3-seed evaluation of the 3B student with the
SPAN VERIFIER replacing the TF-IDF support gate for retrieval-grounded abstention.

Reuses grounded_spike_r2.py (metrics, support, local_dir, LORA_CFG) and spike_path_c.py
(Gen, run_engine, regression) unchanged, and the labels of the seed-0 run (run_A/labels.json).

Arms (all 3B, all share the same labels and the same training recipe as run_A):
  plain : retention + generic abstentions (1.6 % abstention rows).   Seeds 0 (reused), 1, 2.
  rg    : plain + retrieval-grounded hard negatives, capped at 3.5 %  (same rows as run_A, same
          TF-IDF filter for WHICH negatives are kept during training).  Seeds 0 (reused), 1, 2.

Inference gate:
  adapter arm : no gate.
  gate arm    : the answer is refused unless span_verifier.supported(answer, passage), where
                passage = the top-1 TF-IDF passage over the whole corpus (all 87 passages across
                every skill, not a per-skill cluster) for that question. This is the
                owner-specified replacement for the TF-IDF cosine tau gate used in run_A.

Seed 0 of BOTH arms is NOT retrained: the run_A adapters are reused (the gate is inference-only
and the training rows and seed are identical), so the verifier evaluation is comparable.
"""
import argparse
import gc
import json
import os
import random
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import derive_facts as df  # noqa: E402
import grounded_spike as gs  # noqa: E402
import grounded_spike_r2 as r2  # noqa: E402
import span_verifier as sv  # noqa: E402
import spike_path_c as sp  # noqa: E402

ABSTAIN = gs.ABSTAIN
LORA_CFG = r2.LORA_CFG


def gate_refuse(outs, prompts, passage_texts):
    """Span-verifier refusal per question. Answers are checked against their retrieved passage."""
    return [not sv.supported(o, t) for o, t in zip(outs, passage_texts)]


def passage_invented(outs, refuse, passage_texts):
    """Share of non-refused answers that write a span absent from their retrieved passage."""
    kept = [(o, t) for o, rf, t in zip(outs, refuse, passage_texts) if not rf]
    if not kept:
        return 0.0
    return sum(bool(sv.unsupported_spans(o, t)) for o, t in kept) / len(kept)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--student", default="Qwen/Qwen2.5-3B-Instruct")
    ap.add_argument("--skills-dir", required=True)
    ap.add_argument("--skills", default="debug,check,document,scope")
    ap.add_argument("--labels", required=True)
    ap.add_argument("--reuse-dir", required=True, help="run_A dir holding the seed-0 adapters")
    ap.add_argument("--out", required=True)
    ap.add_argument("--passages-per-skill", type=int, default=30)
    ap.add_argument("--retain", type=int, default=26)
    ap.add_argument("--neg-share", type=float, default=0.10)
    ap.add_argument("--tau-pct", type=float, default=0.05)
    ap.add_argument("--runs", default="base,plain:0,rg:0,plain:1,rg:1,plain:2,rg:2",
                    help="ordered list; 'base' evaluates the base model; 'variant:seed' trains or reuses")
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)
    T0 = time.time()

    names = a.skills.split(",")
    docs = gs.load_docs(a.skills_dir, names)
    passages = gs.build_passages(docs, a.passages_per_skill)
    corpus_lc = "\n".join(docs.values()).lower()
    by_id = {p["id"]: p for p in passages}
    tfidf = df.TfIdf([p["text"] for p in passages])
    L = json.load(open(a.labels))

    # identical to grounded_spike_r2: tau from training answerables, negatives filtered by it
    sup_pos = sorted(r2.support(tfidf, r["prompt"]) for r in L["train"])
    tau = sup_pos[int(a.tau_pct * len(sup_pos))]
    neg_kept_all = [n for n in L["negs"] if r2.support(tfidf, n["prompt"]) < tau]

    train_rows = [{"prompt": r["prompt"], "completion": r["completion"]} for r in L["train"]]
    generic_neg = [{"prompt": q, "completion": ABSTAIN} for q in
                   ["How do I rotate logs with logrotate?", "How do I create a Kubernetes pod?", "What command lists docker containers?",
                    "How do I set a git remote URL?", "How do I restart nginx?", "How do I check disk usage with du?"]]
    rp = sp.RETAIN_POOL[: a.retain]

    held_prompts = [h["prompt"] for h in L["held"]]
    probe_prompts = [x["prompt"] for x in L["probes"]]
    held_ptxt = [passages[sv.best_passage_index(tfidf, q)]["text"] for q in held_prompts]
    probe_ptxt = [passages[sv.best_passage_index(tfidf, q)]["text"] for q in probe_prompts]
    no_held = [False] * len(held_prompts)
    no_probe = [False] * len(probe_prompts)

    def arms(outs, pouts):
        rh = gate_refuse(outs, held_prompts, held_ptxt)
        rp_ = gate_refuse(pouts, probe_prompts, probe_ptxt)
        m_ad = r2.metrics(outs, L["held"], no_held, by_id, corpus_lc, pouts, no_probe, L["probes"])
        m_gt = r2.metrics(outs, L["held"], rh, by_id, corpus_lc, pouts, rp_, L["probes"])
        m_gt["invented_passage_held"] = round(passage_invented(outs, rh, held_ptxt), 4)
        m_gt["invented_passage_probe"] = round(passage_invented(pouts, rp_, probe_ptxt), 4)
        m_ad["invented_passage_held"] = round(passage_invented(outs, no_held, held_ptxt), 4)
        return {"adapter_arm": m_ad, "gate_arm": m_gt}

    report_path = os.path.join(a.out, "report.json")
    report = {"config": dict(LORA_CFG, neg_share=a.neg_share, tau_train=round(tau, 4), student=a.student,
                             gate="span_verifier (top-1 TF-IDF passage)", labels=a.labels,
                             passages_per_skill=a.passages_per_skill, retain=a.retain),
              "runs": []}
    import torch

    g = sp.Gen(a.student)
    retain = [{"prompt": q, "completion": o} for q, o in zip(rp, g.gen(rp, max_new=60))]
    for spec in a.runs.split(","):
        t_run = time.time()
        if spec == "base":
            outs = g.gen(held_prompts, max_new=100, bs=48)
            pouts = g.gen(probe_prompts, max_new=100, bs=48)
            run = {"name": "base", "seed": None, "variant": "base", **arms(outs, pouts),
                   "regression": sp.regression(g), "wall_s": round(time.time() - t_run, 1)}
            report["base"] = run
            print("BASE", json.dumps({"good": run["adapter_arm"]["good"], "ung": run["adapter_arm"]["ungrounded"],
                                      "probe_ung": run["adapter_arm"]["probe_ungrounded"]}), flush=True)
        else:
            variant, seed_s = spec.split(":")
            seed = int(seed_s)
            # rows exactly as grounded_spike_r2.main builds them
            rows = list(train_rows) + retain + list(generic_neg)
            n_pos = len(rows)
            if variant == "rg":
                rng = random.Random(1000 + seed)
                cap = int(a.neg_share * n_pos / (1 - a.neg_share))
                k = max(0, cap - len(generic_neg))
                pick = rng.sample(neg_kept_all, min(k, len(neg_kept_all)))
                rows += [{"prompt": n["prompt"], "completion": n["completion"]} for n in pick]
            abst_rows = sum(r["completion"] == ABSTAIN for r in rows)
            cfg = dict(LORA_CFG, seed=seed)
            tag = a.student.split("/")[-1]
            reuse = os.path.join(a.reuse_dir, f"{tag}_{variant}_seed{seed}.safetensors")
            if seed == 0 and os.path.exists(reuse):
                path, train_s, reused = reuse, 0.0, True
            else:
                path = os.path.join(a.out, f"{tag}_{variant}_seed{seed}.safetensors")
                del g; gc.collect(); torch.cuda.empty_cache()
                t0 = time.time()
                info = sp.run_engine(r2.local_dir(a.student), rows, cfg, path)
                train_s, reused = time.time() - t0, False
                g = sp.Gen(a.student)
                final_loss = info["final_loss"]  # noqa
            g.use_adapter(path, cfg)
            t1 = time.time()
            outs = g.gen(held_prompts, max_new=100, bs=48)
            pouts = g.gen(probe_prompts, max_new=100, bs=48)
            eval_s = time.time() - t1
            run = {"name": f"{variant}:{seed}", "seed": seed, "variant": variant, "adapter_reused": reused,
                   "train_rows": len(rows), "abstain_rows": abst_rows, "abstain_share": round(abst_rows / len(rows), 4),
                   **arms(outs, pouts), "regression": sp.regression(g),
                   "train_wall_s": round(train_s, 1), "eval_wall_s": round(eval_s, 1),
                   "wall_s": round(time.time() - t_run, 1)}
            if not reused:
                run["final_loss"] = final_loss
            g.drop_adapter()
            report["runs"].append(run)
            print("RUN", spec, json.dumps({"good_gate": run["gate_arm"]["good"], "invented_held": run["adapter_arm"]["ungrounded"],
                                           "inv_passage_gate": run["gate_arm"]["invented_passage_held"],
                                           "false_ref": run["gate_arm"]["gate_false_refusal"],
                                           "probe_ung_gate": run["gate_arm"]["probe_ungrounded"],
                                           "abst_known_gate": run["gate_arm"]["abstain_on_known"], "reg": run["regression"],
                                           "reused": reused}), flush=True)
        json.dump(report, open(report_path, "w"), indent=1)
    report["total_wall_s"] = round(time.time() - T0, 1)
    json.dump(report, open(report_path, "w"), indent=1)
    print("DONE", report["total_wall_s"], flush=True)


if __name__ == "__main__":
    main()
