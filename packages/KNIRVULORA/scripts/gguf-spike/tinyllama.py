#!/usr/bin/env python3
"""Writes a tiny random llama-architecture GGUF (GQA: 4 heads, 2 kv heads) that
llama.cpp can load, plus a GGUF LoRA adapter for it. Used by the P5 spike and the
GGUF writer's round-trip test; it needs only numpy and llama.cpp's gguf-py.

  PYTHONPATH=<llama.cpp>/gguf-py python3 tinyllama.py <llama.cpp> <outdir> [lora-rank]
"""
import sys, os
import numpy as np
from gguf import GGUFReader, GGUFWriter

NE, NH, NKV, NFF, NL = 64, 4, 2, 128, 2
HD = NE // NH
KVD = NKV * HD


def vocab_fields(llama_cpp):
    r = GGUFReader(os.path.join(llama_cpp, "models", "ggml-vocab-llama-spm.gguf"))
    f = {k: v.contents() for k, v in r.fields.items()}
    return f


def main():
    llama_cpp, out = sys.argv[1], sys.argv[2]
    rank = int(sys.argv[3]) if len(sys.argv) > 3 else 4
    os.makedirs(out, exist_ok=True)
    rng = np.random.default_rng(7)
    v = vocab_fields(llama_cpp)
    n_vocab = len(v["tokenizer.ggml.tokens"])

    w = GGUFWriter(os.path.join(out, "tiny.gguf"), "llama")
    w.add_name("tiny-gqa")
    w.add_block_count(NL); w.add_context_length(256); w.add_embedding_length(NE)
    w.add_feed_forward_length(NFF); w.add_head_count(NH); w.add_head_count_kv(NKV)
    w.add_layer_norm_rms_eps(1e-5); w.add_rope_dimension_count(HD); w.add_vocab_size(n_vocab)
    w.add_tokenizer_model("llama"); w.add_tokenizer_pre("default")
    w.add_token_list(v["tokenizer.ggml.tokens"]); w.add_token_scores(v["tokenizer.ggml.scores"])
    w.add_token_types(v["tokenizer.ggml.token_type"])
    w.add_bos_token_id(1); w.add_eos_token_id(2); w.add_unk_token_id(0)
    w.add_add_bos_token(True); w.add_add_eos_token(False)

    def t(name, *shape, scale=0.05):
        w.add_tensor(name, (rng.standard_normal(shape) * scale).astype(np.float32))

    t("token_embd.weight", n_vocab, NE)
    w.add_tensor("output_norm.weight", np.ones(NE, np.float32))
    t("output.weight", n_vocab, NE)
    for i in range(NL):
        b = f"blk.{i}."
        w.add_tensor(b + "attn_norm.weight", np.ones(NE, np.float32))
        t(b + "attn_q.weight", NE, NE); t(b + "attn_k.weight", KVD, NE)
        t(b + "attn_v.weight", KVD, NE); t(b + "attn_output.weight", NE, NE)
        w.add_tensor(b + "ffn_norm.weight", np.ones(NE, np.float32))
        t(b + "ffn_gate.weight", NFF, NE); t(b + "ffn_up.weight", NFF, NE); t(b + "ffn_down.weight", NE, NFF)
    w.write_header_to_file(); w.write_kv_data_to_file(); w.write_tensors_to_file(); w.close()

    # adapter targeting Q and O (the P5 first cut), strong enough to change output
    a = GGUFWriter(os.path.join(out, "adapter.gguf"), "llama")
    a.add_string("general.type", "adapter"); a.add_string("adapter.type", "lora")
    a.add_float32("adapter.lora.alpha", float(rank) * 8)
    for i in range(NL):
        for nm, (o, n) in {"attn_q": (NE, NE), "attn_output": (NE, NE)}.items():
            a.add_tensor(f"blk.{i}.{nm}.weight.lora_a", (rng.standard_normal((rank, n)) * 0.5).astype(np.float32))
            a.add_tensor(f"blk.{i}.{nm}.weight.lora_b", (rng.standard_normal((o, rank)) * 0.5).astype(np.float32))
    a.write_header_to_file(); a.write_kv_data_to_file(); a.write_tensors_to_file(); a.close()
    print("wrote", out)


main()
