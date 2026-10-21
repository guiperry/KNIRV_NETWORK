#!/usr/bin/env python3
"""Download Wikipedia as JSONL for `evidence submit -jsonl` (plan Phase 2).

Source: the HuggingFace dataset wikimedia/wikipedia, which is the Wikimedia dump already
cleaned to plain text (CC-BY-SA-4.0 / GFDL). Each parquet shard is downloaded, converted
to one JSONL file {id,url,title,text}, and deleted, so peak disk is one shard. Re-running
skips shards that are already converted (resumable), and a partly downloaded shard is
re-fetched.

    fetch_wikipedia.py --config 20231101.simple --out DIR
    fetch_wikipedia.py --config 20231101.en     --out DIR [--shards 0-4]

The snapshot is the dataset's, not the live site's: record its date with the index.
"""
import argparse, json, os, sys, time, urllib.request, urllib.error

BASE = "https://huggingface.co"
DATASET = "wikimedia/wikipedia"


def get_json(url):
    with urllib.request.urlopen(urllib.request.Request(url, headers={"User-Agent": "knirv-evidence/1"}), timeout=60) as r:
        return json.load(r)


def download(url, dest, tries=5):
    for attempt in range(1, tries + 1):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": "knirv-evidence/1"})
            with urllib.request.urlopen(req, timeout=120) as r, open(dest + ".part", "wb") as f:
                total = int(r.headers.get("Content-Length", 0))
                got = 0
                while True:
                    chunk = r.read(1 << 20)
                    if not chunk:
                        break
                    f.write(chunk)
                    got += len(chunk)
                if total and got != total:
                    raise IOError(f"short read {got}/{total}")
            os.replace(dest + ".part", dest)
            return
        except (urllib.error.URLError, IOError, TimeoutError) as e:
            print(f"  download attempt {attempt}/{tries} failed: {e}", file=sys.stderr)
            time.sleep(min(60, 5 * attempt))
    raise SystemExit(f"giving up on {url}")


def convert(parquet, jsonl):
    import pyarrow.parquet as pq
    pf = pq.ParquetFile(parquet)
    n = 0
    with open(jsonl + ".part", "w", encoding="utf-8") as out:
        for batch in pf.iter_batches(batch_size=2000, columns=["id", "url", "title", "text"]):
            cols = batch.to_pydict()
            for i in range(batch.num_rows):
                text = (cols["text"][i] or "").strip()
                if not text:
                    continue
                out.write(json.dumps({"id": cols["id"][i], "url": cols["url"][i], "title": cols["title"][i], "text": text},
                                     ensure_ascii=False) + "\n")
                n += 1
    os.replace(jsonl + ".part", jsonl)
    return n


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", default="20231101.simple")
    ap.add_argument("--out", required=True)
    ap.add_argument("--shards", default="", help="e.g. 0-4 or 7 (default: all)")
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)
    files = sorted(x["path"] for x in get_json(f"{BASE}/api/datasets/{DATASET}/tree/main/{a.config}") if x["path"].endswith(".parquet"))
    if a.shards:
        lo, _, hi = a.shards.partition("-")
        lo, hi = int(lo), int(hi or lo)
        files = files[lo:hi + 1]
    print(f"{len(files)} shard(s) of {a.config} -> {a.out}", flush=True)
    total = 0
    for i, path in enumerate(files):
        name = os.path.basename(path).replace(".parquet", ".jsonl")
        dest = os.path.join(a.out, f"{a.config}-{name}")
        if os.path.exists(dest):
            print(f"[{i + 1}/{len(files)}] {name} already converted", flush=True)
            continue
        t0 = time.time()
        pq_path = os.path.join(a.out, "shard.parquet")
        download(f"{BASE}/datasets/{DATASET}/resolve/main/{path}", pq_path)
        n = convert(pq_path, dest)
        os.remove(pq_path)
        total += n
        print(f"[{i + 1}/{len(files)}] {name}: {n} articles in {time.time() - t0:.0f}s", flush=True)
    print(f"done: {total} new articles", flush=True)


if __name__ == "__main__":
    main()
