# 4_DATA_TRAINER Specification

## Purpose

`4_DATA_TRAINER` is the gradient-descent training stage of the KNIRVHASHER pipeline. It consumes `training_frames.json` produced by `2_DATA_ENCODER` and trains the Gorgonite GPT (`pkg/hashing/transformer/gpt.go`) to produce model checkpoints for the HEART inference service.

## Position in Pipeline

```
2_DATA_ENCODER → 4_DATA_TRAINER  (base LM weights, backprop-trained)
               → 3_DATA_SEEDER   (PoW-witnessed assertions, ES-mined nonces)
```

The CLI executes the trainer's bounded encoded batch before the independent seeder, so model training is observable even when seed mining is long-running. The seeder does not consume trainer output.

## Data Path

- **Input:** `training_frames.json` (JSON array of `schema.TrainingFrame`)
- **Output:** `<checkpoint_dir>/model_latest.bin` + numbered epoch checkpoints
- **Skip:** Does not use `PrepareDataset`/`Dataset` — reads `TokenSequence []int32` / `TargetTokenID int32` directly off each frame and calls `GPT.TrainStep` per frame

## Training Loop

For each epoch, for each frame:
1. Convert `TokenSequence` ([]int32) to `[]int`
2. Build `targetIDs` by shifting input by one position, last target = `TargetTokenID`
3. Call `gpt.TrainStep(inputIDs, targetIDs, learningRate)`
4. Skip frames that produce non-finite loss (NaN/Inf)
5. Log average epoch loss

## Checkpoint Format

Reuses `SaveModel`/`LoadModel` (`gpt.go:681,701`). Writes:
- `checkpoint_epoch_N.bin` — numbered epoch checkpoints for rollback
- `model_latest.bin` — always-updated symlink/copy for inference-side loading

## Inference Wiring

`HEARTConfig.ModelCheckpointPath` tells `NewHEARTServiceWithConfig` where to look for a trained checkpoint. If the file exists and loads successfully, the service uses it; otherwise it falls back to a randomly-initialized model.

## CLI

```
data-trainer \
  -input training_frames.json \
  -checkpoint-dir checkpoints \
  -epochs 3 \
  -lr 0.0003 \
  -save-freq 1 \
  -resume path/to/checkpoint.bin
```

## Makefile Targets

- `make build-data-trainer` — build the binary
- `make train` — run training with defaults from the KNIRVHASHER root
