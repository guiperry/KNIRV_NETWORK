# 6_HANDS_TRAINER

Batch trainer for the HANDS inner GPT, the stage that follows `5_HEAD_TRAINER` in the
models pipeline. One binary, two inputs:

* a signed `.nrv` corpus (NRV3 or canonical `0x4E525621`), or
* a legacy token-window JSONL file.

Both paths funnel through `knirvmodels/pkg/hands.(*Engine).TrainWindows`, which is the
same entry point `POST /models/hands/train` uses. Only frames whose Z3 assertion is
`VALID` are trained on; invalid and unknown frames are counted and skipped.

## Usage

```bash
go run ./cmd/hands-trainer --input /var/lib/knirvserver/knirvmodels/data/supervision/gold.nrv \
    --checkpoint-dir /var/lib/knirvserver/knirvmodels/data/hands \
    --context-len 64 --epochs 1 --lr 3e-4 --max-steps 512
```

The pass writes `<checkpoint-dir>/model-latest` atomically (`--out` overrides the
path) and prints a JSON summary: frames, valid/invalid split, windows and steps.

Resume an existing checkpoint with `--resume <path>`; the model architecture flags
must match the checkpoint.

### Legacy JSONL

```jsonl
[1,2,3,4,5]
{"token_ids":[1,2,3,4]}
{"tokens":[9,8,7,6]}
{"window":[4,5,6,7]}
```

Every line is one token window of at least two tokens, all inside `--vocab-size`.
Blank lines are ignored.

## Model flags

`--vocab-size`, `--embed-dim`, `--heads`, `--layers`, `--model-context`,
`--ffn-dim`, `--dropout` default to `gpt.DefaultConfig()`. `--context-len`
(the training window) must not exceed `--model-context`.

## Bounds

| flag | default | note |
| --- | --- | --- |
| `--epochs` | 1 | must be `> 0` |
| `--lr` | 3e-4 | must be in `(0,1]` |
| `--max-steps` | 512 | hard cap on optimizer steps; `0` derives from the dataset |
| `--context-len` | 64 | token window length |
| `--stride` | context length | window stride |
| `--max-frames` | 100000 | frame ceiling, matching the dispatcher default |

## `.nrv` requirements and signature status

The reader accepts the NRV3 (`"NRV3"`) and canonical (`0x4E525621`) containers. Payloads
are read from the registry frame `payload` member or an `<asset>.payload.json` sidecar
(object, `{ "frames": ... }`, or array form). Bracket geometry is validated against the
bracket section.

The current KNIRVBASE does not expose a producer/consumer for the canonical binary
`Payload ModalityIndex` section, nor an ML-DSA-65 verifier for `PQCManifest`. This
trainer therefore checks only that a manifest is present and names an algorithm; it does
**not** verify the signature and reports that via the dispatcher's
`verification_skipped` field. Do not treat an accepted `.nrv` as signature-verified until
that KNIRVBASE gap is closed (see the F report owner question).

## Build target

There is no `make build-hands-trainer` target yet: `KNIRVMODELS/Makefile` is protected
while the owner's uncommitted model changes are in flight. Build with
`go build ./cmd/hands-trainer` from this directory, or wire it into
`build-pipeline-all` once the Makefile is free.
