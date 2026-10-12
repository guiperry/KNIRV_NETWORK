# `.nrv` container formats

Three on-disk layouts exist. **KNIRVBASE v1 is the canonical stored format.** The
models's `NRV3` is a pipeline interchange format, and `NRV2` is legacy.
All three share the 80-byte bracket layout (`pkg/nrv.EncodeBracket`).

| | Magic | Written by | Read by | Status |
|---|---|---|---|---|
| KNIRVBASE v1 | `0x4E525621` LE (`!VRN`) | KNIRVBASE Go/TS `NRVWriter` | KNIRVBASE Go/TS `NRVReader` | canonical |
| NRV3 | ASCII `NRV3` | KNIRVMODELS `2_DATA_ENCODER/pkg/nrvio` | `pkg/nrv.OpenNRV3`, seeder `knirvbaseclient` | interchange |
| NRV2 | ASCII `NRV2` | older models builds | nothing safe | refused |

## KNIRVBASE v1

```
[ 12-byte header ][ 5 MiB registry region ][ frame 0 brackets ][ frame 1 brackets ] ...
  magic u32 LE, version u32 LE (=1), total_length u32 LE
```

* The registry is JSON (Go schema, snake_case; see `pkg/nrv/frame.go`) padded with NULs to
  exactly `5 * 1024 * 1024` **bytes**. Writing a registry larger than that fails with
  `nrv: registry exceeds padding` — see *Capacity* below.
* A frame is `count × 80` bytes at `brackets.offset` (absolute file offset). `bracket_index`
  has one entry per bracket: `id`, `type` (`I`/`P`), `anchor_id`, `offset` (within the frame),
  `drift_score`.
* **Delta encoding.** An `I` bracket is stored in full. A `P` bracket stores its 32 projection
  bytes XORed with the projections of its anchor (`anchor_id`, an earlier `I` bracket of the
  *same frame*); the other 48 bytes are stored in full. Readers undo this
  (`ApplyProjectionDelta`). The writer policy (`FrameTicker`, `pkg/nrv.IFrameInterval` /
  `DriftThreshold`): the first bracket, every 50th, and any bracket whose
  `EuclideanDrift` from the anchor exceeds 0.25 is an `I`.
* `pqc_manifest.frame_signatures[frame_id]` is the base64 ML-DSA-65 signature over the
  frame's raw bracket bytes, present when the writer had a key (`key_id` names it).
* Capacity: each `bracket_index` entry costs about 200 bytes of indented JSON, so a file holds
  roughly 25k brackets. Converters shard (`storage.MaxBracketsPerFile = 16000`).

## NRV3 (models interchange)

```
"NRV3" | registry_length u32 LE | registry JSON | bracket section
```

Measured on a real models batch (`…/frames/batches/*/training_frames.nrv`, 77 MB):

* `registry`: `{"version":"3.0","frames":[{id, timestamp_unix, count, offset, linguistic{tokens,domain},
  thermo{}, z3{}, bracket_index:[{index, type, xor}]}]}`. `thermo`/`z3` are written empty by the
  models.
* The **bracket section holds full 80-byte brackets** (`count × 80` bytes per frame at `offset`).
* `bracket_index[i].xor` (80 bytes, base64) is the bracket **XOR the previous bracket**
  (index 0: the bracket itself); `type` is `I` for index 0 and `P` after. It is a chain delta kept for
  the seeder — **not** KNIRVBASE's anchor-relative delta. Verified on 2000 consecutive brackets:
  `xor[i] == raw[i] ^ raw[i-1]` for 1999/1999, `== raw[i] ^ raw[0]` for 20/1999.
* A real frame is huge: one batch file is a single frame of 344,167 brackets, whose registry is 49 MB.

## NRV3 → KNIRVBASE conversion

`storage.ConvertNRV3File` / `nrv.ConvertFrame` (CLI: `nrvconform convert-nrv3 <in> <out> [keyfile]`):

1. Parse the NRV3 registry; take the frame's **full** brackets from the bracket section. The `xor`
   blobs are ignored.
2. Split each frame into output frames of at most `MaxBracketsPerFrame` (1024) brackets
   (`<id>#1`, `<id>#2`, …; a frame that fits keeps its id). Each output frame is self-contained.
3. Re-encode every output frame with KNIRVBASE's I/P policy above. **Do not copy NRV3 `P` labels:**
   labelling full brackets `P` makes a KNIRVBASE reader XOR them with an anchor and corrupt every
   projection.
4. Shard the output across files of at most 16,000 brackets: the first file is `<out>`, then
   `<out minus .nrv>.1.nrv`, `.2.nrv`, …. The written paths are printed.
5. With a key file, every frame is signed (ML-DSA-65). Without one the output is unsigned.
6. On any error nothing is left behind.

`z3` is not carried over (the models never fills it); linguistic `domain`→`token` and `tokens`→`unit`;
`thermo.temperature`→`avg_temp_c`. Bracket ids are `<nrv3 frame id>-<bracket index>`.

Verified on the real 77 MB batch above: 344,167 brackets → 23 files, every bracket decodes identically
through the KNIRVBASE reader (`KNIRV_REAL_NRV3=<file> go test ./internal/storage -run RealNRV3`).

## Signing (E18)

New datasets are signed. The key is a post-quantum key pair (`knirv-pqc-key/v2`, ML-KEM-768 +
ML-DSA-65) kept **outside** the data directory it protects. The arena dataset service
(`backend_server/internal/services/arenadatasets`) loads or creates it at
`$KNIRV_DATASET_KEY_FILE`, default `<user config dir>/knirv-server/.key/arena-datasets.pqc.json`
(mode 0600), and refuses to open the store if the key is unusable rather than falling back to
unsigned. `GET /api/error-nodes/{id}/datasets` lists the key (`signing.key_id`, `algorithm`,
`public_key`); `GET …/frames/{frameId}` returns the frame's `signature`. Datasets written before this
change stay unsigned.
