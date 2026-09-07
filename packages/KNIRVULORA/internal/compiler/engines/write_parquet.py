#!/usr/bin/env python3
"""
Write calibration_anchors.parquet from a JSON list of dataset records.

Input JSON (stdin):
{
  "records": [{"context": "...", "corrected_completion": "...", "target_model": "..."}],
  "output_path": "/path/to/calibration_anchors.parquet"
}

Output: a parquet file with columns: context, corrected_completion, target_model.
"""

import sys
import json
import pyarrow as pa
import pyarrow.parquet as pq


def main():
    raw = sys.stdin.read()
    payload = json.loads(raw)

    records = payload["records"]
    output_path = payload["output_path"]

    table = pa.table({
        "context": [r.get("context", "") for r in records],
        "corrected_completion": [r.get("corrected_completion", "") for r in records],
        "target_model": [r.get("target_model", "") for r in records],
    })

    pq.write_table(table, output_path)
    print(json.dumps({"status": "ok", "rows": len(records), "output": output_path}))


if __name__ == "__main__":
    main()
