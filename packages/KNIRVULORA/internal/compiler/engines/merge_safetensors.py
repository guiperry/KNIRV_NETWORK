#!/usr/bin/env python3
"""
Merge multiple safetensors files into one, namespacing tensors by their source.

Input JSON (stdin):
{
  "inputs": [
    {"path": "/path/to/source.safetensors", "prefix": "source"},
    {"path": "/path/to/target1.safetensors", "prefix": "target-llama-3-70b"},
    ...
  ],
  "output_path": "/path/to/core_weights.safetensors"
}

Output: a single safetensors file with all tensors, prefixed by their origin.
"""

import sys
import json
import numpy as np
from safetensors.numpy import save_file, load_file


def main():
    raw = sys.stdin.read()
    payload = json.loads(raw)

    inputs = payload["inputs"]
    output_path = payload["output_path"]

    all_tensors = {}
    all_metadata = {}

    for inp in inputs:
        path = inp["path"]
        prefix = inp.get("prefix", "unknown")
        tensors = load_file(path)
        for name, arr in tensors.items():
            all_tensors[f"{prefix}/{name}"] = arr
        for key, val in tensors.metadata or {} if hasattr(tensors, 'metadata') else {}:
            all_metadata[key] = val

    save_file(all_tensors, output_path, metadata=all_metadata)
    print(json.dumps({"status": "ok", "tensors": len(all_tensors), "output": output_path}))


if __name__ == "__main__":
    main()
