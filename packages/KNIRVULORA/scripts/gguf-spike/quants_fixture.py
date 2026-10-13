#!/usr/bin/env python3
"""Writes quants.gguf (one 4x64 tensor per supported type, quantised by llama.cpp's
own gguf-py) and expected.json (what gguf-py dequantises each back to). The Go
dequantisers must reproduce it exactly.

  PYTHONPATH=<llama.cpp>/gguf-py python3 quants_fixture.py <outdir>
"""
import json, os, sys
import numpy as np
from gguf import GGUFWriter, GGMLQuantizationType as Q
from gguf.quants import quantize, dequantize

out = sys.argv[1]
os.makedirs(out, exist_ok=True)
rng = np.random.default_rng(3)
data = (rng.standard_normal((4, 64)) * 2).astype(np.float32)
w = GGUFWriter(os.path.join(out, "quants.gguf"), "llama")
expected = {}
for name, qt in [("q4_0", Q.Q4_0), ("q4_1", Q.Q4_1), ("q5_0", Q.Q5_0), ("q5_1", Q.Q5_1), ("q8_0", Q.Q8_0)]:
    packed = quantize(data, qt)
    w.add_tensor(name, packed, raw_dtype=qt)
    expected[name] = dequantize(packed, qt).reshape(4, 64).tolist()
w.add_tensor("f16", data.astype(np.float16))
expected["f16"] = data.astype(np.float16).astype(np.float32).tolist()
w.add_tensor("f32", data)
expected["f32"] = data.tolist()
bf = quantize(data, Q.BF16)
w.add_tensor("bf16", bf, raw_dtype=Q.BF16)
expected["bf16"] = dequantize(bf, Q.BF16).reshape(4, 64).tolist()
w.write_header_to_file(); w.write_kv_data_to_file(); w.write_tensors_to_file(); w.close()
json.dump(expected, open(os.path.join(out, "expected.json"), "w"))
print("ok")
