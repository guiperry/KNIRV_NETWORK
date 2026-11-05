#!/usr/bin/env python3
"""
Path C: context distillation into a NATIVE LoRA, by gradient descent.

Unlike Path B (a linear fit over feature-hash embeddings, fit_kind
"linear_embedding_fit"), this engine loads a real base model, attaches a PEFT
LoRA to the transformer's linear projections and minimises token-level
cross-entropy on teacher completions. The adapter is therefore trained against
the actual forward pass, and is stamped fit_kind "gradient_sft".

The output is a *native* adapter for the training base model (PEFT tensor
naming, loadable by PEFT and convertible to GGUF for llama.cpp). Lifting it into
the canonical space is Task 5.3, not this file.

Input JSON (stdin):
{
  "dataset": [{"prompt": "...", "completion": "...", "system": "optional"}],
  "base_model": "<HF directory containing config.json + *.safetensors>",
  "config": {"rank": 16, "alpha": 32, "learning_rate": 2e-4, "epochs": 3,
             "batch_size": 8, "max_length": 512, "seed": 42,
             "target_modules": ["q_proj","k_proj","v_proj","o_proj",
                                "gate_proj","up_proj","down_proj"],
             "dropout": 0.0},
  "output_path": "<adapter .safetensors>"
}

Stdout: one JSON object {"fit_kind", "base_sha256", "recipe_hash", "seed",
"loss_curve_digest", "final_loss", "steps"}. The same values are written into
the safetensors metadata.
"""

import hashlib
import json
import os
import random
import sys
import traceback

DEFAULT_TARGETS = ["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"]
FIT_KIND = "gradient_sft"


class PathCError(Exception):
    """A refusal: the request cannot be honoured as stated."""


def sha256_files(paths):
    h = hashlib.sha256()
    for p in sorted(paths):
        with open(p, "rb") as f:
            for chunk in iter(lambda: f.read(1 << 20), b""):
                h.update(chunk)
    return h.hexdigest()


def base_weight_files(base_model):
    files = [os.path.join(base_model, n) for n in os.listdir(base_model) if n.endswith(".safetensors")]
    if not files:
        raise PathCError(f"no *.safetensors in base_model {base_model!r}")
    return files


def canonical_recipe(config, dataset):
    """Hash of everything that determines the result besides the base weights."""
    body = {
        "config": {k: config[k] for k in sorted(config)},
        "dataset": [[r.get("system", ""), r["prompt"], r["completion"]] for r in dataset],
    }
    return hashlib.sha256(json.dumps(body, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def build_example(tok, rec, max_length):
    msgs = []
    if rec.get("system"):
        msgs.append({"role": "system", "content": rec["system"]})
    msgs.append({"role": "user", "content": rec["prompt"]})
    prompt_ids = tok.apply_chat_template(msgs, tokenize=True, add_generation_prompt=True)
    comp_ids = tok(rec["completion"] + tok.eos_token, add_special_tokens=False)["input_ids"]
    ids = (prompt_ids + comp_ids)[:max_length]
    labels = ([-100] * len(prompt_ids) + comp_ids)[:max_length]
    return ids, labels


def train(req):
    # Imported lazily so a bad request is refused before torch is loaded.
    import torch
    from peft import LoraConfig, get_peft_model, get_peft_model_state_dict
    from safetensors.torch import save_file
    from transformers import AutoModelForCausalLM, AutoTokenizer

    dataset = req.get("dataset") or []
    if not dataset:
        raise PathCError("empty dataset")
    base_model = req.get("base_model")
    if not base_model or not os.path.isdir(base_model):
        raise PathCError("base_model must be a local HF directory")
    out = req.get("output_path")
    if not out:
        raise PathCError("output_path required")

    cfg = dict(req.get("config") or {})
    cfg.setdefault("rank", 16)
    cfg.setdefault("alpha", 32)
    cfg.setdefault("learning_rate", 2e-4)
    cfg.setdefault("epochs", 3)
    cfg.setdefault("batch_size", 8)
    cfg.setdefault("max_length", 512)
    cfg.setdefault("seed", 42)
    cfg.setdefault("target_modules", DEFAULT_TARGETS)
    cfg.setdefault("dropout", 0.0)

    if not torch.cuda.is_available() and not cfg.get("allow_cpu"):
        raise PathCError("no CUDA device; set config.allow_cpu to train on CPU")
    device = "cuda" if torch.cuda.is_available() else "cpu"

    seed = int(cfg["seed"])
    random.seed(seed)
    torch.manual_seed(seed)
    torch.cuda.manual_seed_all(seed)

    tok = AutoTokenizer.from_pretrained(base_model)
    model = AutoModelForCausalLM.from_pretrained(base_model, torch_dtype=torch.bfloat16 if device == "cuda" else torch.float32)
    model.to(device)
    lcfg = LoraConfig(
        r=int(cfg["rank"]),
        lora_alpha=float(cfg["alpha"]),
        lora_dropout=float(cfg["dropout"]),
        target_modules=list(cfg["target_modules"]),
        bias="none",
        task_type="CAUSAL_LM",
    )
    model = get_peft_model(model, lcfg)
    # Keep adapter weights in fp32 for stable optimisation; base stays bf16.
    for p in model.parameters():
        if p.requires_grad:
            p.data = p.data.float()

    examples = [build_example(tok, r, int(cfg["max_length"])) for r in dataset]
    params = [p for p in model.parameters() if p.requires_grad]
    opt = torch.optim.AdamW(params, lr=float(cfg["learning_rate"]), weight_decay=0.0)
    bs = int(cfg["batch_size"])
    steps_total = cfg["epochs"] * ((len(examples) + bs - 1) // bs)
    sched = torch.optim.lr_scheduler.LambdaLR(opt, lambda s: min(1.0, (s + 1) / max(1, steps_total // 10)) * max(0.0, 1 - s / steps_total) ** 0.5 if steps_total else 1.0)

    rng = random.Random(seed)
    losses = []
    model.train()
    for _ in range(int(cfg["epochs"])):
        order = list(range(len(examples)))
        rng.shuffle(order)
        for i in range(0, len(order), bs):
            batch = [examples[j] for j in order[i : i + bs]]
            width = max(len(x[0]) for x in batch)
            ids = torch.full((len(batch), width), tok.pad_token_id, dtype=torch.long)
            lab = torch.full((len(batch), width), -100, dtype=torch.long)
            att = torch.zeros((len(batch), width), dtype=torch.long)
            for k, (a, b) in enumerate(batch):
                ids[k, : len(a)] = torch.tensor(a)
                lab[k, : len(b)] = torch.tensor(b)
                att[k, : len(a)] = 1
            with torch.autocast(device, dtype=torch.bfloat16, enabled=device == "cuda"):
                loss = model(input_ids=ids.to(device), attention_mask=att.to(device), labels=lab.to(device)).loss
            loss.backward()
            torch.nn.utils.clip_grad_norm_(params, 1.0)
            opt.step()
            sched.step()
            opt.zero_grad(set_to_none=True)
            losses.append(round(float(loss.detach().cpu()), 5))

    state = {k: v.detach().cpu().contiguous() for k, v in get_peft_model_state_dict(model).items()}
    meta = {
        "fit_kind": FIT_KIND,
        "method": "context_distillation",
        "base_sha256": sha256_files(base_weight_files(base_model)),
        "recipe_hash": canonical_recipe({k: cfg[k] for k in cfg if k != "allow_cpu"}, dataset),
        "seed": str(seed),
        "loss_curve_digest": hashlib.sha256(json.dumps(losses).encode()).hexdigest(),
        "rank": str(cfg["rank"]),
        "alpha": str(cfg["alpha"]),
        "target_modules": ",".join(cfg["target_modules"]),
        "format": "peft",
    }
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    save_file(state, out, metadata=meta)
    return {**{k: meta[k] for k in ("fit_kind", "base_sha256", "recipe_hash", "loss_curve_digest")},
            "seed": seed, "final_loss": losses[-1], "steps": len(losses)}


def main():
    try:
        print(json.dumps(train(json.load(sys.stdin))))
    except PathCError as e:
        print(f"path_c refused: {e}", file=sys.stderr)
        sys.exit(2)
    except Exception:
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
