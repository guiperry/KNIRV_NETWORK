#!/usr/bin/env python3
"""
I_52 phase 2 prototype: small LoRA adapters per skill, several loaded at once in llama.cpp, routed per request.
Four variants share the same eval sets (held / probe from labels.json), passages, router (top-1 TF-IDF
passage) and span verifier (spans must appear in the retrieved passage(s)). Each variant has its own
--stages names; the default for each is listed below. Earlier variants stay reproducible.

unmatched (default, earlier 3-adapter recipe, unchanged):
  train    : HF/PEFT, 3B student, one adapter per skill (debug, check, document). Rows = that skill's
             teacher-labelled passages + the 26 retention rows (base model's own answers), no abstention rows.
             Same LORA_CFG and seed 0 as the phase-1 runs. GPU.
  convert  : base 3B -> f16 GGUF; each adapter -> GGUF (convert_lora_to_gguf.py).
  eval     : one llama-server (localhost only) with the base GGUF and 4 adapters preloaded
             (--lora-init-without-apply): ids 0-2 = per-skill adapters, id 3 = the phase-1 single plain adapter
             (3B, seed 0, all four skills). Each request selects adapters with the per-request "lora" list.
             Arms: base (no adapter), single (id 3), routed (id of the router's skill), oracle (gold skill).
             Scope has no adapter in this variant; its questions go to the base model.
             Gate: span verifier against the top-1 passage.

matched (I_52 phase 2, owner 24a): four per-skill adapters (debug, check, document, scope), each with the
  phase-1 single adapter's abstention share (1.6 %), the same retention rows and the six generic refusals.
  Row logic: phase2_rows.py (CPU-tested). Stages: train_matched (default train,convert,eval_matched).
  eval_matched: 8 adapters in one server; arms base, single, matched routed/oracle, unmatched routed.
  eval_topk (same variant): top-k verifier. Generations are cached to generations_topk.json once per arm,
  then scored under k=1 and k=3 (the gate passes an answer if each span is in ANY of the top-k passages).
  Retention rows are written to retention_rows.json in the run dir; --retention-rows reuses a saved file.

stronger (I_52 owner decision 20a, seed 0): per-skill adapters with more epochs (STRONGER_EPOCHS) and
  skill-specific abstention rows (the phase-1 labelled hard negatives of that skill, not only the six generic
  refusals; STRONGER_SHARE of rows). Row logic: phase2_rows.build_stronger_rows. Stages: train_stronger,
  convert_stronger, eval_stronger (default all three). The eval reuses base/single/matched generations from
  --reuse-generations (the top-k cache) and generates only the stronger arms (routed, oracle), then scores
  them under k=1 and k=3. Outputs: stronger_rows.json, stronger_rows_<skill>.json, generations_stronger.json,
  phase2_stronger_report.json.

Scope of every variant: the 3 adapted skills in unmatched; all 4 skills in matched and stronger.
"""
import argparse
import json
import os
import shutil
import subprocess
import sys
import time
import urllib.request
from concurrent.futures import ThreadPoolExecutor

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import derive_facts as df  # noqa: E402
import grounded_spike as gs  # noqa: E402
import grounded_spike_r2 as r2  # noqa: E402
import phase2_rows as pr  # noqa: E402
import span_verifier as sv  # noqa: E402
import spike_path_c as sp  # noqa: E402

ABSTAIN = gs.ABSTAIN
LLAMA = "/home/gperry/models/llama.cpp"
# CUDA build of the same llama.cpp tree (the default build in LLAMA is CPU-only: GGML_CUDA=OFF)
SERVER_BIN = "/home/gperry/ori_knirv/scratch_I/llama-cuda/bin/llama-server"
ADAPTER_SKILLS = ["debug", "check", "document"]
PORT = 18110
N_ADAPTERS = 4  # ids 0-2 per-skill, 3 = phase-1 single plain adapter
TARGETS = ["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"]


def write_adapter_dir(safetensors_path, base_dir, out_dir):
    os.makedirs(out_dir, exist_ok=True)
    shutil.copy(safetensors_path, os.path.join(out_dir, "adapter_model.safetensors"))
    cfg = {"peft_type": "LORA", "task_type": "CAUSAL_LM", "r": 16, "lora_alpha": 32, "lora_dropout": 0.0,
           "target_modules": TARGETS, "bias": "none", "fan_in_fan_out": False, "use_rslora": False,
           "modules_to_save": None, "base_model_name_or_path": base_dir, "inference_mode": True}
    json.dump(cfg, open(os.path.join(out_dir, "adapter_config.json"), "w"), indent=1)
    return out_dir


def stage_train(a, L, docs_names):
    import torch  # noqa: F401
    base = r2.local_dir(a.student)
    gen = sp.Gen(a.student)
    rp = sp.RETAIN_POOL[: a.retain]
    retain = [{"prompt": q, "completion": o} for q, o in zip(rp, gen.gen(rp, max_new=60))]
    del gen
    import gc
    gc.collect()
    import torch as _t
    _t.cuda.empty_cache()
    cfg = dict(r2.LORA_CFG, seed=0)
    info = {}
    for skill in ADAPTER_SKILLS:
        out = os.path.join(a.out, f"adapter_{skill}.safetensors")
        rows = [{"prompt": r["prompt"], "completion": r["completion"]} for r in L["train"] if r["skill"] == skill]
        rows = rows + retain
        if os.path.exists(out):
            print("exists, not retrained:", out, flush=True)
            info[skill] = {"rows": len(rows), "reused": True}
            continue
        t0 = time.time()
        res = sp.run_engine(base, rows, cfg, out)
        info[skill] = {"rows": len(rows), "final_loss": res["final_loss"], "train_wall_s": round(time.time() - t0, 1)}
        print("TRAINED", skill, json.dumps(info[skill]), flush=True)
    return info


def stage_convert(a):
    base = r2.local_dir(a.student)
    base_gguf = os.path.join(a.out, "base-f16.gguf")
    if not os.path.exists(base_gguf):
        subprocess.run([sys.executable, os.path.join(LLAMA, "convert_hf_to_gguf.py"), base, "--outtype", "f16",
                        "--outfile", base_gguf], check=True)
    gguf = {}
    for name, src in [("debug", "adapter_debug"), ("check", "adapter_check"), ("document", "adapter_document"),
                      ("single_plain_all", None)]:
        if src is None:
            st = a.single_adapter
        else:
            st = os.path.join(a.out, f"{src}.safetensors")
        d = write_adapter_dir(st, base, os.path.join(a.out, f"dir_{name}"))
        g = os.path.join(a.out, f"lora_{name}.gguf")
        if not os.path.exists(g):
            subprocess.run([sys.executable, os.path.join(LLAMA, "convert_lora_to_gguf.py"), "--base", base,
                            "--outfile", g, d], check=True)
        gguf[name] = g
    return base_gguf, gguf


def serve(base_gguf, gguf_ids, log_path=None):
    cmd = [SERVER_BIN, "-ngl", "99", "-fa", "on", "-m", base_gguf, "--host", "127.0.0.1", "--port", str(PORT),
           "-ngl", "99", "-c", "4096", "-np", "8", "--lora-init-without-apply",
           "--lora", ",".join(gguf_ids)]
    log_path = log_path or os.path.join(os.path.dirname(gguf_ids[0]), "llama-server.log")
    p = subprocess.Popen(cmd, stdout=open(log_path, "w"),
                         stderr=subprocess.STDOUT)
    for _ in range(240):
        try:
            urllib.request.urlopen(f"http://127.0.0.1:{PORT}/health", timeout=2)
            return p
        except Exception:
            time.sleep(1)
    p.kill()
    raise RuntimeError("llama-server did not become healthy")


def complete(tok, prompt, lora_scales, n_pred=100, n_ids=N_ADAPTERS):
    text = tok.apply_chat_template([{"role": "user", "content": prompt}], tokenize=False, add_generation_prompt=True)
    body = json.dumps({"prompt": text, "n_predict": n_pred, "temperature": 0, "top_k": 1, "cache_prompt": False,
                       # ALWAYS send every loaded adapter id with an explicit scale: an omitted id (or an
                       # empty list) leaves the server on whatever adapter state it has, not on the base model
                       "lora": [{"id": i, "scale": float(lora_scales.get(i, 0.0))} for i in range(n_ids)]}).encode()
    req = urllib.request.Request(f"http://127.0.0.1:{PORT}/completion", data=body,
                                 headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=900) as r:
        return json.loads(r.read())["content"]


def stage_eval(a, L, passages, by_id, corpus_lc, tfidf, gguf, base_gguf):
    from transformers import AutoTokenizer
    tok = AutoTokenizer.from_pretrained(r2.local_dir(a.student))
    ids = {"debug": 0, "check": 1, "document": 2, "single": 3}
    keep = set(ADAPTER_SKILLS)
    held = [h for h in L["held"] if h["skill"] in keep]
    probes = [p for p in L["probes"] if p["skill"] in keep]
    p = serve(base_gguf, [gguf["debug"], gguf["check"], gguf["document"], gguf["single_plain_all"]])
    try:
        def route(q):
            return passages[sv.best_passage_index(tfidf, q)]["skill"]

        results = {}
        for name, qs in [("held", held), ("probe", probes)]:
            prompts = [x["prompt"] for x in qs]
            top = [passages[sv.best_passage_index(tfidf, q)] for q in prompts]
            routed = [route(q) for q in prompts]
            gold = [x["skill"] for x in qs]
            results[name] = {"router_acc": round(sum(r == g for r, g in zip(routed, gold)) / len(gold), 4),
                             "n": len(qs)}
            arms = {
                "base": [{} for _ in prompts],
                "single": [{3: 1.0} for _ in prompts],
                "routed": [({ids[r]: 1.0} if r in ids else {}) for r in routed],  # scope has no adapter -> base
                "oracle": [{ids[g]: 1.0} for g in gold],
            }
            for arm, scales in arms.items():
                t0 = time.time()
                # group requests by adapter set: concurrent requests with different adapter scales
                # force llama.cpp to re-apply the LoRA weights, so each group runs on its own
                groups = {}
                for i, sc in enumerate(scales):
                    groups.setdefault(tuple(sorted(sc.items())), []).append(i)
                outs = [None] * len(prompts)
                for key, idx in groups.items():
                    with ThreadPoolExecutor(max_workers=8) as ex:
                        res = list(ex.map(lambda i: complete(tok, prompts[i], dict(key)), idx))
                    for i, o in zip(idx, res):
                        outs[i] = o
                wall = time.time() - t0
                ptxt = [t["text"] for t in top]
                refuse = [not sv.supported(o, t) for o, t in zip(outs, ptxt)]
                if name == "held":
                    # r2.metrics also scores probes; pass one dummy probe and ignore its fields
                    dummy = [ABSTAIN], [False], [{"passage": None, "prompt": "", "skill": ""}]
                    m_ad = r2.metrics(outs, qs, [False] * len(qs), by_id, corpus_lc, *dummy)
                    m_gt = r2.metrics(outs, qs, refuse, by_id, corpus_lc, *dummy)
                    m_ad = {k: v for k, v in m_ad.items() if not k.startswith("probe")}
                    m_gt = {k: v for k, v in m_gt.items() if not k.startswith("probe")}
                    results[name][arm] = {"adapter_arm": m_ad, "gate_arm": m_gt, "wall_s": round(wall, 1),
                                          "passage_invented_gate": round(
                                              sum(bool(sv.unsupported_spans(o, t)) for o, rf, t in zip(outs, refuse, ptxt) if not rf)
                                              / max(1, sum(1 for rf in refuse if not rf)), 4)}
                else:
                    # probes: a probe is "invented" if the answer is not an abstention and writes an ungrounded span
                    p_ung = sum((not rf) and bool(df.ungrounded(o, corpus_lc)) for o, rf in zip(outs, refuse))
                    p_abs = sum(rf for rf in refuse)
                    results[name][arm] = {"probe_invented_gate": round(p_ung / len(outs), 4),
                                          "probe_refused_gate": round(p_abs / len(outs), 4), "wall_s": round(wall, 1)}
                print(name, arm, json.dumps(results[name][arm], default=str)[:300], flush=True)
        return results
    finally:
        p.terminate()
        p.wait(timeout=60)


# ---------------------------------------------------------------------------------------------
# MATCHED variant (I_52 phase 2, owner 24a): four per-skill adapters (debug, check, document,
# scope), each with the phase-1 single adapter's abstention share (~1.6 %), same retention rows,
# same recipe. Row logic lives in phase2_rows.py (CPU-tested). The unmatched stages above are
# unchanged and still run with --variant unmatched (the default).
# Adapter ids in the matched eval server: 0-3 matched per-skill, 4 = phase-1 single plain adapter,
# 5-7 = the unmatched per-skill adapters from the earlier run (re-scored under the current verifier).
MATCHED_IDS = {"debug": 0, "check": 1, "document": 2, "scope": 3}
SINGLE_ID = 4
UNMATCHED_IDS = {"debug": 5, "check": 6, "document": 7}
N_IDS_MATCHED = 8


def stage_train_matched(a, L):
    base = r2.local_dir(a.student)
    retain, rsrc = pr.retention_rows(a.retention_rows, os.path.join(a.out, "retention_rows.json"), make_retention(a))
    n_single = len(L["train"]) + len(retain) + len(pr.GENERIC_PROMPTS)
    share = pr.single_share(n_single, len(pr.GENERIC_PROMPTS))
    cfg = dict(r2.LORA_CFG, seed=0)
    info = {"share_target": round(share, 5), "single_total_rows": n_single,
            "single_abstain_rows": len(pr.GENERIC_PROMPTS), "retention_source": rsrc, "adapters": {}}
    for skill in pr.MATCHED_SKILLS:
        out = os.path.join(a.out, f"matched_{skill}.safetensors")
        rows, rinfo = pr.build_matched_rows(skill, L["train"], retain, pr.GENERIC_PROMPTS, share, 0, ABSTAIN)
        if os.path.exists(out):
            print("exists, not retrained:", out, flush=True)
            rinfo["reused"] = True
        else:
            t0 = time.time()
            res = sp.run_engine(base, rows, cfg, out)
            rinfo.update(final_loss=res["final_loss"], train_wall_s=round(time.time() - t0, 1))
            print("TRAINED", skill, json.dumps(rinfo), flush=True)
        info["adapters"][skill] = rinfo
    json.dump(info, open(os.path.join(a.out, "matched_rows.json"), "w"), indent=1)
    return info


def stage_convert_matched(a):
    base = r2.local_dir(a.student)
    base_gguf = os.path.join(a.out, "base-f16.gguf")
    if not os.path.exists(base_gguf):
        src = os.path.join(a.reuse_dir, "base-f16.gguf") if a.reuse_dir else ""
        if src and os.path.exists(src):
            # same base model and converter as the earlier run: link, do not re-convert 6 GB
            os.symlink(src, base_gguf)
        else:
            subprocess.run([sys.executable, os.path.join(LLAMA, "convert_hf_to_gguf.py"), base, "--outtype", "f16",
                            "--outfile", base_gguf], check=True)
    gguf = {}
    todo = [(f"matched_{s}", os.path.join(a.out, f"matched_{s}.safetensors")) for s in pr.MATCHED_SKILLS]
    todo.append(("single_plain_all", a.single_adapter))
    for name, st in todo:
        d = write_adapter_dir(st, base, os.path.join(a.out, f"dir_{name}"))
        g = os.path.join(a.out, f"lora_{name}.gguf")
        if not os.path.exists(g):
            subprocess.run([sys.executable, os.path.join(LLAMA, "convert_lora_to_gguf.py"), "--base", base,
                            "--outfile", g, d], check=True)
        gguf[name] = g
    for s in ADAPTER_SKILLS:  # unmatched per-skill GGUFs from the earlier run, linked (same files)
        src = os.path.join(a.reuse_dir, f"lora_{s}.gguf")
        g = os.path.join(a.out, f"lora_unmatched_{s}.gguf")
        if not os.path.exists(g):
            os.symlink(src, g)
        gguf[f"unmatched_{s}"] = g
    return base_gguf, gguf


def generate_grouped(tok, prompts, scales, n_ids=N_IDS_MATCHED):
    groups = {}
    for i, sc in enumerate(scales):
        groups.setdefault(tuple(sorted(sc.items())), []).append(i)
    outs = [None] * len(prompts)
    for key, idx in groups.items():
        with ThreadPoolExecutor(max_workers=8) as ex:
            res = list(ex.map(lambda i: complete(tok, prompts[i], dict(key), n_ids=n_ids), idx))
        for i, o in zip(idx, res):
            outs[i] = o
    return outs


def arm_metrics(outs_h, outs_p, idx_h, idx_p, ptxt_h, ptxt_p, held_all, probe_all, by_id, corpus_lc):
    """Gated and ungated metrics on a subset of held/probe indices (same r2.metrics as before)."""
    hs = [held_all[i] for i in idx_h]
    oh = [outs_h[i] for i in idx_h]
    th = [ptxt_h[i] for i in idx_h]
    ps = [probe_all[i] for i in idx_p]
    op = [outs_p[i] for i in idx_p]
    tp = [ptxt_p[i] for i in idx_p]
    rf_h = [not sv.supported(o, t) for o, t in zip(oh, th)]
    rf_p = [not sv.supported(o, t) for o, t in zip(op, tp)]
    no_h, no_p = [False] * len(hs), [False] * len(ps)
    ungated = r2.metrics(oh, hs, no_h, by_id, corpus_lc, op, no_p, ps)
    gated = r2.metrics(oh, hs, rf_h, by_id, corpus_lc, op, rf_p, ps)
    kept = [(o, t) for o, rf, t in zip(oh, rf_h, th) if not rf]
    inv_pass = sum(bool(sv.unsupported_spans(o, t)) for o, t in kept) / max(1, len(kept))
    return {"n_held": len(hs), "n_probe": len(ps), "ungated": ungated, "gated": gated,
            "invented_passage_gate_held": round(inv_pass, 4)}


def stage_eval_matched(a, L, passages, by_id, corpus_lc, tfidf, gguf, base_gguf):
    from transformers import AutoTokenizer
    tok = AutoTokenizer.from_pretrained(r2.local_dir(a.student))
    order = ([gguf[f"matched_{s}"] for s in pr.MATCHED_SKILLS] + [gguf["single_plain_all"]]
             + [gguf[f"unmatched_{s}"] for s in ADAPTER_SKILLS])
    assert len(order) == N_IDS_MATCHED
    held_all, probe_all = L["held"], L["probes"]
    three = {"debug", "check", "document"}
    sets = {"full": None, "three_skill": three}
    p = serve(base_gguf, order)
    try:
        ptxt = {}
        routed = {}
        outs = {}
        for name, qs in [("held", held_all), ("probe", probe_all)]:
            prompts = [x["prompt"] for x in qs]
            gold = [x["skill"] for x in qs]
            tops = [passages[sv.best_passage_index(tfidf, q)] for q in prompts]
            ptxt[name] = [t["text"] for t in tops]
            routed[name] = [t["skill"] for t in tops]
            arms = {
                "base": [{} for _ in prompts],
                "single": [{SINGLE_ID: 1.0} for _ in prompts],
                "matched_routed": [{MATCHED_IDS[r]: 1.0} for r in routed[name]],
                "matched_oracle": [{MATCHED_IDS[g]: 1.0} for g in gold],
                # unmatched: no scope adapter, scope questions go to base (as in the earlier run)
                "unmatched_routed": [({UNMATCHED_IDS[r]: 1.0} if r in UNMATCHED_IDS else {}) for r in routed[name]],
            }
            outs[name] = {}
            for arm, scales in arms.items():
                t0 = time.time()
                outs[name][arm] = generate_grouped(tok, prompts, scales)
                print("GEN", name, arm, round(time.time() - t0, 1), "s", flush=True)
        results = {"router_acc": {}, "arms": {}}
        for sname, keep in sets.items():
            for name, qs in [("held", held_all), ("probe", probe_all)]:
                idx = [i for i, x in enumerate(qs) if keep is None or x["skill"] in keep]
                correct = sum(routed[name][i] == qs[i]["skill"] for i in idx)
                results["router_acc"].setdefault(sname, {})[name] = {"n": len(idx), "acc": round(correct / len(idx), 4)}
        for arm in outs["held"]:
            results["arms"][arm] = {}
            for sname, keep in sets.items():
                idx_h = [i for i, x in enumerate(held_all) if keep is None or x["skill"] in keep]
                idx_p = [i for i, x in enumerate(probe_all) if keep is None or x["skill"] in keep]
                results["arms"][arm][sname] = arm_metrics(
                    outs["held"][arm], outs["probe"][arm], idx_h, idx_p, ptxt["held"], ptxt["probe"],
                    held_all, probe_all, by_id, corpus_lc)
            print("SCORED", arm, json.dumps(results["arms"][arm]["full"]["gated"]["good"]), flush=True)
        return results
    finally:
        p.terminate()
        p.wait(timeout=60)


# ---------------------------------------------------------------------------------------------
# TOP-K VERIFIER (I_52 19a, owner): the gate passes an answer if each span it writes appears in ANY of
# the top-k TF-IDF passages. Generation does not depend on the passage (the prompt is the question
# only), so each arm is generated ONCE, cached to generations_topk.json, and scored under k=1 and k=3.
# Server ids here: 0-3 = matched per-skill (debug, check, document, scope), 4 = phase-1 single adapter.
TOPK_N_IDS = 5
TOPK_SINGLE_ID = 4
TOPK_KS = (1, 3)


def arm_scores_topk(outs, held_all, probe_all, tops, idx_h, idx_p, k, by_id, corpus_lc):
    """Gated/ungated metrics for one arm and one k. tops[name][i] = list of passage dicts, best first."""
    hs = [held_all[i] for i in idx_h]
    ps = [probe_all[i] for i in idx_p]
    oh = [outs["held"][i] for i in idx_h]
    op = [outs["probe"][i] for i in idx_p]
    th = [[p["text"] for p in tops["held"][i][:k]] for i in idx_h]
    tp = [[p["text"] for p in tops["probe"][i][:k]] for i in idx_p]
    t1h = [tops["held"][i][0]["text"] for i in idx_h]
    t1p = [tops["probe"][i][0]["text"] for i in idx_p]
    rf_h = [not sv.supported_topk(o, t) for o, t in zip(oh, th)]
    rf_p = [not sv.supported_topk(o, t) for o, t in zip(op, tp)]
    no_h, no_p = [False] * len(hs), [False] * len(ps)
    ungated = r2.metrics(oh, hs, no_h, by_id, corpus_lc, op, no_p, ps)
    gated = r2.metrics(oh, hs, rf_h, by_id, corpus_lc, op, rf_p, ps)
    kept_h = [(o, t1, tk) for o, rf, t1, tk in zip(oh, rf_h, t1h, th) if not rf]
    # kept answers with a span the passage set does not contain (must be 0 by construction)
    inv_k = sum(bool(sv.unsupported_spans_topk(o, tk)) for o, _, tk in kept_h) / max(1, len(kept_h))
    # kept answers with a span the TOP-1 passage does not contain: what top-k lets through beyond top-1
    beyond_h = sum(bool(sv.unsupported_spans(o, t1)) for o, t1, _ in kept_h) / max(1, len(kept_h))
    kept_p = [(o, t1, rf) for o, rf, t1 in zip(op, rf_p, t1p) if not rf]
    p_n = max(1, len(op))
    probe = {
        "refused_gated": round(sum(rf_p) / p_n, 4),
        # probe invented = answer not refused and writes a span not in the corpus (corpus-level, as before)
        "invented_gated": round(sum(bool(df.ungrounded(o, corpus_lc)) for o, _, _ in kept_p) / p_n, 4),
        "invented_ungated": round(sum(bool(df.ungrounded(o, corpus_lc)) for o in op) / p_n, 4),
        # kept answers with a span outside the top-1 passage (what top-k admits beyond top-1)
        "kept_outside_top1_gated": round(sum(bool(sv.unsupported_spans(o, t1)) for o, t1, _ in kept_p) / p_n, 4),
    }
    return {"n_held": len(hs), "n_probe": len(ps), "ungated": ungated, "gated": gated,
            "held_invented_passage_kept": round(inv_k, 4), "held_kept_outside_top1": round(beyond_h, 4),
            "probe": probe}


def stage_eval_topk(a, L, passages, by_id, corpus_lc, tfidf, gguf, base_gguf):
    from transformers import AutoTokenizer
    tok = AutoTokenizer.from_pretrained(r2.local_dir(a.student))
    gen_path = os.path.join(a.out, "generations_topk.json")
    cache = json.load(open(gen_path)) if os.path.exists(gen_path) else {}
    held_all, probe_all = L["held"], L["probes"]
    order = [gguf[f"matched_{s}"] for s in pr.MATCHED_SKILLS] + [gguf["single_plain_all"]]
    assert len(order) == TOPK_N_IDS
    tops, routed = {}, {}
    for name, qs in [("held", held_all), ("probe", probe_all)]:
        tops[name] = [[passages[i] for i in sv.best_passage_indices(tfidf, x["prompt"], max(TOPK_KS))] for x in qs]
        routed[name] = [t[0]["skill"] for t in tops[name]]
    p = None
    try:
        for name, qs in [("held", held_all), ("probe", probe_all)]:
            prompts = [x["prompt"] for x in qs]
            gold = [x["skill"] for x in qs]
            arms = {
                "base": [{} for _ in prompts],
                "single": [{TOPK_SINGLE_ID: 1.0} for _ in prompts],
                "matched_routed": [{MATCHED_IDS[r]: 1.0} for r in routed[name]],
                "matched_oracle": [{MATCHED_IDS[g]: 1.0} for g in gold],
            }
            for arm, scales in arms.items():
                key = f"{name}/{arm}"
                if key in cache:
                    print("REUSED generations", key, flush=True)
                    continue
                if p is None:
                    p = serve(base_gguf, order, log_path=os.path.join(a.out, "topk_llama-server.log"))
                t0 = time.time()
                outs = generate_grouped(tok, prompts, scales, n_ids=TOPK_N_IDS)
                cache[key] = {"outs": outs, "wall_s": round(time.time() - t0, 1)}
                json.dump(cache, open(gen_path, "w"), indent=1)
                print("GEN", key, cache[key]["wall_s"], "s", flush=True)
    finally:
        if p is not None:
            p.terminate()
            p.wait(timeout=60)
    outs = {name: {arm: cache[f"{name}/{arm}"]["outs"] for arm in ["base", "single", "matched_routed", "matched_oracle"]}
            for name in ["held", "probe"]}
    three = {"debug", "check", "document"}
    sets = {"full": None, "three_skill": three}
    res = {"router_acc": {}, "arms": {}, "wall_s": {k: v["wall_s"] for k, v in cache.items()}}
    for sname, keep in sets.items():
        for name, qs in [("held", held_all), ("probe", probe_all)]:
            idx = [i for i, x in enumerate(qs) if keep is None or x["skill"] in keep]
            correct = sum(routed[name][i] == qs[i]["skill"] for i in idx)
            res["router_acc"].setdefault(sname, {})[name] = {"n": len(idx), "acc": round(correct / len(idx), 4)}
    for arm in outs["held"]:
        res["arms"][arm] = {}
        for sname, keep in sets.items():
            idx_h = [i for i, x in enumerate(held_all) if keep is None or x["skill"] in keep]
            idx_p = [i for i, x in enumerate(probe_all) if keep is None or x["skill"] in keep]
            for k in TOPK_KS:
                sc = arm_scores_topk({"held": outs["held"][arm], "probe": outs["probe"][arm]},
                                     held_all, probe_all, tops, idx_h, idx_p, k, by_id, corpus_lc)
                res["arms"][arm].setdefault(sname, {})[f"k{k}"] = sc
        print("SCORED", arm, "full k1 gated good", round(res["arms"][arm]["full"]["k1"]["gated"]["good"], 4),
              "k3", round(res["arms"][arm]["full"]["k3"]["gated"]["good"], 4), flush=True)
    return res


def run_matched(a, L, passages, by_id, corpus_lc, tfidf, stages):
    rep = {"config": dict(r2.LORA_CFG, student=a.student, variant="matched", skills=pr.MATCHED_SKILLS,
                          retain=a.retain, abstain_prompts=pr.GENERIC_PROMPTS)}
    if "train" in stages:
        rep["train"] = stage_train_matched(a, L)
    if "eval_topk" in stages:
        base_gguf, gguf = stage_convert_matched(a)
        rep["gguf"] = {k: os.path.getsize(v) for k, v in gguf.items()}
        rep["gguf"]["base"] = os.path.getsize(base_gguf)
        rep["eval_topk"] = stage_eval_topk(a, L, passages, by_id, corpus_lc, tfidf, gguf, base_gguf)
        json.dump(rep, open(os.path.join(a.out, "phase2_topk_report.json"), "w"), indent=1, default=str)
        print("DONE", flush=True)
        return
    if "convert" in stages or "eval_matched" in stages:
        base_gguf, gguf = stage_convert_matched(a)
        rep["gguf"] = {k: os.path.getsize(v) for k, v in gguf.items()}
        rep["gguf"]["base"] = os.path.getsize(base_gguf)
    if "eval_matched" in stages:
        rep["eval"] = stage_eval_matched(a, L, passages, by_id, corpus_lc, tfidf, gguf, base_gguf)
    json.dump(rep, open(os.path.join(a.out, "phase2_matched_report.json"), "w"), indent=1, default=str)
    print("DONE", flush=True)


# ---------------------------------------------------------------------------------------------
# STRONGER per-skill recipe (I_52 owner decision 20a, seed 0 only). Row logic in phase2_rows.py
# (build_stronger_rows: skill-specific hard-negative abstentions, STRONGER_SHARE, STRONGER_EPOCHS).
# Eval: the matched and single arms are reused from the cached generations of the top-k run (same
# eval sets, same passages, same verifier); only the stronger arms are generated here. Server ids:
# 0-3 = stronger per-skill (debug, check, document, scope), 4 = phase-1 single adapter.
STRONG_IDS = {"debug": 0, "check": 1, "document": 2, "scope": 3}
STRONG_SINGLE_ID = 4
STRONG_N_IDS = 5


def make_retention(a):
    """Callable that generates the retention rows from the base model (used only when no file is reused)."""
    def make():
        import gc
        import torch
        gen = sp.Gen(a.student)
        rp = sp.RETAIN_POOL[: a.retain]
        out = [{"prompt": q, "completion": o} for q, o in zip(rp, gen.gen(rp, max_new=60))]
        del gen
        gc.collect()
        torch.cuda.empty_cache()
        return out
    return make


def stage_train_single_lf(a, L):
    """Leak-free single adapter (I_52 owner decision 22, Q2): the phase-1 plain recipe (labelled rows +
    retention + the six generic refusals, same as grounded_spike_r2 main) with every row whose prompt is a
    held/probe question dropped. Sets a.single_adapter so the convert/eval stages use this adapter."""
    import torch  # noqa: F401
    base = r2.local_dir(a.student)
    retain, rsrc = pr.retention_rows(a.retention_rows, os.path.join(a.out, "retention_rows.json"), make_retention(a))
    eval_prompts = [x["prompt"] for x in list(L["held"]) + list(L["probes"])]
    train_rows = [{"prompt": r["prompt"], "completion": r["completion"]} for r in L["train"]]
    kept, dropped = pr.drop_eval_overlap(train_rows, eval_prompts)
    generic = [{"prompt": q, "completion": ABSTAIN} for q in pr.GENERIC_PROMPTS]
    rows = kept + [dict(r) for r in retain] + generic
    ov = pr.overlap_prompts(rows, L["held"], L["probes"])
    if ov:
        raise RuntimeError(f"single_lf: training prompts still overlap held/probe questions: {ov[:5]}")
    out = os.path.join(a.out, f"single_lf_seed{a.seed}.safetensors")
    info = {"seed": a.seed, "retention_source": rsrc, "train_rows_in": len(train_rows),
            "dropped_eval_overlap": dropped, "total_rows": len(rows), "abstain_rows": len(generic),
            "abstain_share": round(len(generic) / len(rows), 4)}
    json.dump(rows, open(os.path.join(a.out, f"single_lf_rows_seed{a.seed}.json"), "w"), indent=1, ensure_ascii=False)
    if os.path.exists(out):
        print("exists, not retrained:", out, flush=True)
        info["reused"] = True
    else:
        cfg = dict(r2.LORA_CFG, seed=a.seed)
        t0 = time.time()
        res = sp.run_engine(base, rows, cfg, out)
        info.update(final_loss=res["final_loss"], steps=res.get("steps"), train_wall_s=round(time.time() - t0, 1))
        print("TRAINED single_lf", json.dumps(info), flush=True)
    a.single_adapter = out
    return info


def stage_train_stronger(a, L):
    import torch  # noqa: F401
    base = r2.local_dir(a.student)
    retain, rsrc = pr.retention_rows(a.retention_rows, os.path.join(a.out, "retention_rows.json"), make_retention(a))
    eval_prompts = [x["prompt"] for x in list(L["held"]) + list(L["probes"])]
    cfg = dict(r2.LORA_CFG, seed=a.seed, epochs=pr.STRONGER_EPOCHS)
    info = {"seed": a.seed, "retention_source": rsrc, "retention_rows": len(retain), "epochs": pr.STRONGER_EPOCHS,
            "share_target": pr.STRONGER_SHARE, "adapters": {}}
    for skill in pr.MATCHED_SKILLS:
        out = os.path.join(a.out, f"stronger_{skill}.safetensors")
        rows, rinfo = pr.build_stronger_rows(skill, L["train"], retain, L["negs"], pr.GENERIC_PROMPTS,
                                             pr.STRONGER_SHARE, a.seed, ABSTAIN, excluded_prompts=eval_prompts)
        ov = pr.overlap_prompts(rows, L["held"], L["probes"])
        if ov:
            raise RuntimeError(f"{skill}: training prompts overlap held/probe questions: {ov[:5]}")
        rinfo["held_probe_prompt_overlap"] = len(ov)
        # informational: training passages of this skill that are also held/probe passages (labelled rows
        # and eval questions share passages; a synthetic-row no-overlap rule cannot be met by passage)
        own_pass = {r["passage"] for r in L["train"] if r["skill"] == skill}
        eval_pass = {x["passage"] for x in list(L["held"]) + list(L["probes"])}
        rinfo["train_passages"] = len(own_pass)
        rinfo["train_passages_also_in_eval"] = len(own_pass & eval_pass)
        json.dump(rows, open(os.path.join(a.out, f"stronger_rows_{skill}.json"), "w"), indent=1, ensure_ascii=False)
        if os.path.exists(out):
            print("exists, not retrained:", out, flush=True)
            rinfo["reused"] = True
        else:
            t0 = time.time()
            res = sp.run_engine(base, rows, cfg, out)
            rinfo.update(final_loss=res["final_loss"], steps=res.get("steps"), train_wall_s=round(time.time() - t0, 1))
            print("TRAINED", skill, json.dumps(rinfo), flush=True)
        info["adapters"][skill] = rinfo
    json.dump(info, open(os.path.join(a.out, "stronger_rows.json"), "w"), indent=1)
    return info


def stage_convert_stronger(a):
    base = r2.local_dir(a.student)
    base_gguf = os.path.join(a.out, "base-f16.gguf")
    if not os.path.exists(base_gguf):
        src = os.path.join(a.reuse_dir, "base-f16.gguf") if a.reuse_dir else ""
        if src and os.path.exists(src):
            os.symlink(src, base_gguf)  # same base model and converter as the earlier runs
        else:
            subprocess.run([sys.executable, os.path.join(LLAMA, "convert_hf_to_gguf.py"), base, "--outtype", "f16",
                            "--outfile", base_gguf], check=True)
    gguf = {}
    todo = [(f"stronger_{s}", os.path.join(a.out, f"stronger_{s}.safetensors")) for s in pr.MATCHED_SKILLS]
    todo.append(("single_plain_all", a.single_adapter))
    for name, st in todo:
        d = write_adapter_dir(st, base, os.path.join(a.out, f"dir_{name}"))
        g = os.path.join(a.out, f"lora_{name}.gguf")
        if not os.path.exists(g):
            subprocess.run([sys.executable, os.path.join(LLAMA, "convert_lora_to_gguf.py"), "--base", base,
                            "--outfile", g, d], check=True)
        gguf[name] = g
    return base_gguf, gguf


def stage_eval_stronger(a, L, passages, by_id, corpus_lc, tfidf, gguf, base_gguf):
    from transformers import AutoTokenizer
    tok = AutoTokenizer.from_pretrained(r2.local_dir(a.student))
    gen_path = os.path.join(a.out, "generations_stronger.json")
    cache, source = {}, {}
    reuse_arms = [x for x in a.reuse_arms.split(",") if x]
    if a.reuse_generations:
        reused = json.load(open(a.reuse_generations))
        for key in [f"{n}/{arm}" for n in ["held", "probe"] for arm in reuse_arms]:
            if key not in reused:
                raise RuntimeError(f"reused generations lack {key}")
            cache[key] = reused[key]
            source[key] = "reused"
    if os.path.exists(gen_path):
        own = json.load(open(gen_path))
        for k, v in own.items():
            cache[k] = v
            source[k] = "generated"
    held_all, probe_all = L["held"], L["probes"]
    order = [gguf[f"stronger_{s}"] for s in pr.MATCHED_SKILLS] + [gguf["single_plain_all"]]
    assert len(order) == STRONG_N_IDS
    tops, routed = {}, {}
    for name, qs in [("held", held_all), ("probe", probe_all)]:
        tops[name] = [[passages[i] for i in sv.best_passage_indices(tfidf, x["prompt"], max(TOPK_KS))] for x in qs]
        routed[name] = [t[0]["skill"] for t in tops[name]]
    p = None
    try:
        for name, qs in [("held", held_all), ("probe", probe_all)]:
            prompts = [x["prompt"] for x in qs]
            gold = [x["skill"] for x in qs]
            arms = {
                # single: cached (seed 0) or generated with the seed's own leak-free single adapter
                "single": [{STRONG_SINGLE_ID: 1.0} for _ in prompts],
                "stronger_routed": [{STRONG_IDS[r]: 1.0} for r in routed[name]],
                "stronger_oracle": [{STRONG_IDS[g]: 1.0} for g in gold],
            }
            for arm, scales in arms.items():
                key = f"{name}/{arm}"
                if key in cache:
                    continue
                if p is None:
                    p = serve(base_gguf, order, log_path=os.path.join(a.out, "stronger_llama-server.log"))
                t0 = time.time()
                outs = generate_grouped(tok, prompts, scales, n_ids=STRONG_N_IDS)
                cache[key] = {"outs": outs, "wall_s": round(time.time() - t0, 1)}
                source[key] = "generated"
                json.dump({k: v for k, v in cache.items() if source.get(k) == "generated"},
                          open(gen_path, "w"), indent=1)
                print("GEN", key, cache[key]["wall_s"], "s", flush=True)
    finally:
        if p is not None:
            p.terminate()
            p.wait(timeout=60)
    all_arms = ["base", "single", "matched_routed", "matched_oracle", "stronger_routed", "stronger_oracle"]
    # arms not in the cache (seed 1-2 runs reuse base only, so no matched arms) are not scored
    arm_names = [x for x in all_arms if f"held/{x}" in cache and f"probe/{x}" in cache]
    outs = {name: {arm: cache[f"{name}/{arm}"]["outs"] for arm in arm_names} for name in ["held", "probe"]}
    three = {"debug", "check", "document"}
    sets = {"full": None, "three_skill": three}
    res = {"router_acc": {}, "arms": {}, "generation_source": source,
           "wall_s": {k: v.get("wall_s") for k, v in cache.items()}}
    for sname, keep in sets.items():
        for name, qs in [("held", held_all), ("probe", probe_all)]:
            idx = [i for i, x in enumerate(qs) if keep is None or x["skill"] in keep]
            correct = sum(routed[name][i] == qs[i]["skill"] for i in idx)
            res["router_acc"].setdefault(sname, {})[name] = {"n": len(idx), "acc": round(correct / len(idx), 4)}
    for arm in arm_names:
        res["arms"][arm] = {}
        for sname, keep in sets.items():
            idx_h = [i for i, x in enumerate(held_all) if keep is None or x["skill"] in keep]
            idx_p = [i for i, x in enumerate(probe_all) if keep is None or x["skill"] in keep]
            for k in TOPK_KS:
                sc = arm_scores_topk({"held": outs["held"][arm], "probe": outs["probe"][arm]},
                                     held_all, probe_all, tops, idx_h, idx_p, k, by_id, corpus_lc)
                res["arms"][arm].setdefault(sname, {})[f"k{k}"] = sc
        print("SCORED", arm, "full k1 gated good", round(res["arms"][arm]["full"]["k1"]["gated"]["good"], 4),
              "k3", round(res["arms"][arm]["full"]["k3"]["gated"]["good"], 4), flush=True)
    return res


def run_stronger(a, L, passages, by_id, corpus_lc, tfidf, stages):
    rep = {"config": dict(r2.LORA_CFG, student=a.student, variant="stronger", epochs=pr.STRONGER_EPOCHS,
                          share_target=pr.STRONGER_SHARE, skills=pr.MATCHED_SKILLS, retain=a.retain, seed=a.seed)}
    if "train_single_lf" in stages:
        rep["single_lf"] = stage_train_single_lf(a, L)
    if "train_stronger" in stages:
        rep["train"] = stage_train_stronger(a, L)
    if "convert_stronger" in stages or "eval_stronger" in stages:
        base_gguf, gguf = stage_convert_stronger(a)
        rep["gguf"] = {k: os.path.getsize(v) for k, v in gguf.items()}
        rep["gguf"]["base"] = os.path.getsize(base_gguf)
    if "eval_stronger" in stages:
        rep["eval"] = stage_eval_stronger(a, L, passages, by_id, corpus_lc, tfidf, gguf, base_gguf)
    json.dump(rep, open(os.path.join(a.out, "phase2_stronger_report.json"), "w"), indent=1, default=str)
    print("DONE", flush=True)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--student", default="Qwen/Qwen2.5-3B-Instruct")
    ap.add_argument("--skills-dir", required=True)
    ap.add_argument("--labels", required=True)
    ap.add_argument("--single-adapter", required=True, help="phase-1 plain adapter (3B seed 0, all skills)")
    ap.add_argument("--out", required=True)
    ap.add_argument("--variant", default="unmatched", choices=["unmatched", "matched", "stronger"],
                    help="unmatched: earlier 3-adapter recipe (default, unchanged). matched: I_52 phase 2 "
                         "4 per-skill adapters with the single adapter's abstention share")
    ap.add_argument("--reuse-dir", default="", help="matched only: earlier phase-2 dir to link base/unmatched GGUFs from")
    ap.add_argument("--stages", default=None,
                    help="default: train,convert,eval (unmatched) or train,convert,eval_matched (matched)")
    ap.add_argument("--retain", type=int, default=26)
    ap.add_argument("--retention-rows", default="",
                    help="matched/stronger: reuse a saved retention_rows.json instead of generating them")
    ap.add_argument("--reuse-generations", default="",
                    help="stronger only: generations_topk.json whose base/single/matched arms are reused")
    ap.add_argument("--reuse-arms", default="base,single,matched_routed,matched_oracle",
                    help="stronger only: which arms to take from --reuse-generations (seed 1-2 runs: base)")
    ap.add_argument("--seed", type=int, default=0,
                    help="stronger only: training seed for row sampling and LoRA init (default 0 = seed-0 run)")
    ap.add_argument("--passages-per-skill", type=int, default=30)
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)
    names = ["debug", "check", "document", "scope"]
    docs = gs.load_docs(a.skills_dir, names)
    passages = gs.build_passages(docs, a.passages_per_skill)
    corpus_lc = "\n".join(docs.values()).lower()
    by_id = {p["id"]: p for p in passages}
    tfidf = df.TfIdf([p["text"] for p in passages])
    L = json.load(open(a.labels))
    if a.variant == "stronger":
        stages = (a.stages or "train_stronger,convert_stronger,eval_stronger").split(",")
        run_stronger(a, L, passages, by_id, corpus_lc, tfidf, stages)
        return
    if a.variant == "matched":
        stages = (a.stages or "train,convert,eval_matched").split(",")
        run_matched(a, L, passages, by_id, corpus_lc, tfidf, stages)
        return
    stages = (a.stages or "train,convert,eval").split(",")
    rep = {"config": dict(r2.LORA_CFG, student=a.student, adapters=ADAPTER_SKILLS, retain=a.retain)}
    if "train" in stages:
        rep["train"] = stage_train(a, L, names)
    if "convert" in stages or "eval" in stages:
        base_gguf, gguf = stage_convert(a)
        rep["gguf"] = {k: os.path.getsize(v) for k, v in gguf.items()}
        rep["gguf"]["base"] = os.path.getsize(base_gguf)
    if "eval" in stages:
        rep["eval"] = stage_eval(a, L, passages, by_id, corpus_lc, tfidf, gguf, base_gguf)
    json.dump(rep, open(os.path.join(a.out, "phase2_report.json"), "w"), indent=1, default=str)
    print("DONE", flush=True)


if __name__ == "__main__":
    main()
