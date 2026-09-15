#!/usr/bin/env python3
"""
Derive a model's CONNECTOR from its own weight matrix.

A connector is the projection pair mapping between a base model's native frame
and the uLoRA canonical space (see connector_core.py). It is a property of the
MODEL, not of a skill, so it is derived once per model and cached by the runtime
(see internal/connector) rather than embedded in every bundle. That is what lets
one artifact bind to any model the cache can serve, instead of a bundle having to
know at compile time which models it will ever run on.

Derivation needs the model's weight matrices. There is no fallback: without real
weights there is no basis to project against, and an invented one would produce a
connector that binds without aligning anything.

Input JSON (stdin):
{
  "target_model": {"family": "llama", "param_count": "8b", "weight_matrix": [[...]]},
  "config": {"canonical_dim": 1024, "rank": 16},
  "output_path": "<safetensors>"
}

Output tensors (names match internal/connector):
  P_in   (K, d_in)
  P_out  (d_out, K)
"""

import json
import sys

from safetensors.numpy import save_file

from connector_core import (
    ConnectorError,
    build_connector,
    connector_metadata,
    fail,
    fail_unexpected,
    require_weight_matrix,
)


def main():
    payload = json.loads(sys.stdin.read())
    config = payload.get("config", {})
    canonical_dim = int(config.get("canonical_dim", 0))
    rank = config.get("rank")
    rank = int(rank) if rank else None

    w_tgt = require_weight_matrix(payload.get("target_model"), "target")

    p_in, p_out, info = build_connector(w_tgt, canonical_dim, rank=rank)
    info["source"] = "derived_from_target_weights"

    save_file(
        {"P_in": p_in, "P_out": p_out},
        payload["output_path"],
        metadata=connector_metadata("derive_connector", info),
    )
    print(json.dumps({
        "status": "ok",
        "engine": "derive_connector",
        "tensors": ["P_in", "P_out"],
        "p_in_shape": list(p_in.shape),
        "p_out_shape": list(p_out.shape),
        "subspace_rank": info["subspace_rank"],
    }))


if __name__ == "__main__":
    try:
        main()
    except ConnectorError as exc:
        fail(f"derive_connector: {exc}")
    except (KeyError, ValueError, OSError) as exc:
        fail_unexpected(exc)
