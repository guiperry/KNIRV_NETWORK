# NRV batch storage contract

Each successful data-encoder run creates one immutable directory:

```
<KNIRV_APP_DATA_DIR>/frames/
  batches/<batch-id>/
    training_frames.json
    training_frames.arrow
    training_frames.nrv
  latest.json
  seed_writes.jsonl
  nrv_submissions.json
```

`latest.json` is atomically replaced only after all three artifacts exist. It
contains the batch ID, creation time, frame count, relative artifact names,
and SHA-256 digests. Consumers resolve and verify the JSON artifact through
this manifest; no consumer should treat `training_frames.json` as a rolling
source of truth.

The `.nrv` container begins with `NRV3`, a little-endian metadata length, JSON
registry metadata, then fixed 80-byte brackets. Each bracket uses the public
KNIRVBASE map: projections `0..31`, microsecond field `32..35`, syntactic byte
`36`, signed dependency head `37`, intent flags `38`, little-endian domain
signature `39..40`, golden seed `41..44`, memory `45..58`, LSH salt `59..62`,
and 17 reserved bytes `63..79`. The syntactic byte packs POS (4 low bits),
tense (next 2), and plurality (top 2).

Older `NRV2` files use the former incompatible layout. They remain ingestible:
the data seeder translates each bracket to canonical layout before sending it
to KNIRVBASE. New writers must only create `NRV3`.

Semantic trainer checkpoints are cumulative and atomic. They record applied
batch IDs together with learned prototype state, so a completed immutable
batch is skipped on retry. The seeder similarly records successful NRV
artifact digests in `nrv_submissions.json`, preventing duplicate `/append`
requests after a stage retry.

Runtime storage resolution is consistent across pipeline and HASHER/HEART:
`FRAMES_DIR` takes precedence, then `KNIRV_APP_DATA_DIR/frames`, then the
production fallback `/var/lib/knirvserver/knirvhasher/data/frames`.
