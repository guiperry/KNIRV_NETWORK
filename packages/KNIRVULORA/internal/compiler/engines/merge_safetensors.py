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
from safetensors import safe_open
from safetensors.numpy import save_file, load_file


def read_metadata(path):
    """
    Read a safetensors file's metadata.

    safe_open is required because load_file returns a plain dict of tensors with
    no metadata attribute — the previous code reached for `.metadata` on that
    dict, so `hasattr` was always False and every input's provenance was silently
    dropped while looking like it was being carried through.
    """
    try:
        with safe_open(path, framework="np") as handle:
            return dict(handle.metadata() or {})
    except (OSError, ValueError) as exc:
        print(f"merge_safetensors: cannot read metadata from {path}: {exc}", file=sys.stderr)
        sys.exit(1)


def main():
    raw = sys.stdin.read()
    payload = json.loads(raw)

    inputs = payload["inputs"]
    output_path = payload["output_path"]

    all_tensors = {}
    # Provenance is collected per source rather than merged flat: the engines now
    # write a single "ulora" key each, so a flat merge would just overwrite and
    # keep whichever input happened to be last.
    provenance = {}

    for index, inp in enumerate(inputs):
        path = inp["path"]
        # An absent or empty prefix means the tensors are already namespaced by
        # whoever wrote them. The Path A engine emits a canonical core keyed by
        # semantic module ("self_attn.q_proj/lora_A") and a connector keyed by its
        # tensor_prefix ("llama-8b/P_in"), so re-prefixing would produce
        # "llama-8b/self_attn.q_proj/lora_A" and "llama-8b/llama-8b/P_in" — names
        # no binder looks for.
        prefix = inp.get("prefix", "")
        key = prefix or f"input_{index}"
        prefix = f"{prefix}/" if prefix else ""

        tensors = load_file(path)
        for name, arr in tensors.items():
            merged_name = f"{prefix}{name}"
            if merged_name in all_tensors:
                # Two inputs claiming the same tensor name would make the result
                # depend on input order, so the bundle would assemble differently
                # depending on how the caller listed its parts.
                print(
                    f"merge_safetensors: duplicate tensor {merged_name!r} from {path}",
                    file=sys.stderr,
                )
                sys.exit(1)
            all_tensors[merged_name] = arr

        metadata = read_metadata(path)
        if metadata:
            provenance[key] = metadata

    # One key holding sorted JSON, for the same reason the engines do it: the
    # metadata mapping is serialised by the Rust implementation from a HashMap,
    # so multiple keys serialise in a random order per process and the same
    # bundle would hash differently on every build.
    all_metadata = {"ulora": json.dumps({"sources": provenance}, sort_keys=True)}

    save_file(all_tensors, output_path, metadata=all_metadata)
    print(json.dumps({"status": "ok", "tensors": len(all_tensors), "output": output_path}))


if __name__ == "__main__":
    main()
