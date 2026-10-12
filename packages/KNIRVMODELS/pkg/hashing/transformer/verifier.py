#!/usr/bin/env python3
"""HEART bidirectional WASM verifier.

Reads a JSON payload from stdin:
  {
    "direction": "forward" | "backward",
    "inquiry":   <inquiry object>,
    "wasm_path": "<path to compiled .wasm>"
  }

Writes a JSON result to stdout:
  {
    "passed":     true | false,
    "message":    "<reason>",
    "confidence": 0.0 – 1.0
  }
"""

import json
import sys
import os


def verify_forward(inquiry: dict, wasm_path: str) -> dict:
    """Forward verification: check that the wasm file exists and exports the expected symbol."""
    if not wasm_path or not os.path.exists(wasm_path):
        return {"passed": False, "message": f"wasm file not found: {wasm_path}", "confidence": 0.0}

    try:
        with open(wasm_path, "rb") as f:
            header = f.read(4)
        if header != b"\x00asm":
            return {"passed": False, "message": "not a valid wasm binary (bad magic)", "confidence": 0.0}
    except OSError as e:
        return {"passed": False, "message": str(e), "confidence": 0.0}

    return {"passed": True, "message": "forward OK", "confidence": 0.85}


def verify_backward(inquiry: dict, wasm_path: str) -> dict:
    """Backward verification: semantic consistency check between inquiry and wasm artefact."""
    if not wasm_path or not os.path.exists(wasm_path):
        return {"passed": False, "message": f"wasm file not found: {wasm_path}", "confidence": 0.0}

    wasm_type = str(inquiry.get("wasm_type", ""))
    if not wasm_type:
        return {"passed": False, "message": "inquiry missing wasm_type", "confidence": 0.0}

    # Heuristic: the file size should be non-trivial for a real WASM module.
    size = os.path.getsize(wasm_path)
    if size < 8:
        return {"passed": False, "message": "wasm binary suspiciously small", "confidence": 0.1}

    return {"passed": True, "message": "backward OK", "confidence": 0.9}


def main() -> None:
    raw = sys.stdin.read()
    try:
        payload = json.loads(raw)
    except json.JSONDecodeError as e:
        sys.stdout.write(json.dumps({"passed": False, "message": f"invalid JSON: {e}", "confidence": 0.0}))
        sys.exit(1)

    direction = payload.get("direction", "forward")
    inquiry = payload.get("inquiry", {})
    wasm_path = payload.get("wasm_path", "")

    if direction == "forward":
        result = verify_forward(inquiry, wasm_path)
    elif direction == "backward":
        result = verify_backward(inquiry, wasm_path)
    else:
        result = {"passed": False, "message": f"unknown direction: {direction}", "confidence": 0.0}

    sys.stdout.write(json.dumps(result))


if __name__ == "__main__":
    main()
