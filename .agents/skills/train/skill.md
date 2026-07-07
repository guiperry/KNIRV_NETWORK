# SKILL.md — KNIRVHASHER Training Pipeline Initializer

## Skill Name

`train` — Initialize and run the KNIRVHASHER training pipeline

## Voice Triggers

- "initiate hasher training"
- "start hasher pipeline"
- "run hasher training"
- "train the hasher"
- "start KNIRVHASHER"
- "initialize hasher pipeline"
- "run the train command"
- "start training"
- "train"

## Overview

This skill initializes and runs the KNIRVHASHER training pipeline, which transforms raw data (PDFs, HuggingFace datasets, or local text) into trained ASIC neural frames through 4 pipeline stages:

1. **0_DATA_CONNECTOR** — Data ingestion from HuggingFace or local sources
2. **1_DATA_MINER** — Generates deterministic embeddings using `github.com/guiperry/text-embedder`
3. **2_DATA_ENCODER** — Encodes embeddings to 12-slot ASIC Brackets (80-byte .nrv format)
4. **3_DATA_TRAINER** — Evolutionary GRPO training to find optimal GoldenSeed per bracket

> **IMPORTANT**: The primary embedding method is `github.com/guiperry/text-embedder v0.1.0` (deterministic). Cloudflare and Ollama are **fallbacks only** and should NOT be initialized by this skill.

The pipeline outputs:
- `.nrv` bracket files (80-byte binary records)
- Trained weights in CSV format

## Pipeline Root

```
packages/KNIRVHASHER/pipeline/
├── 0_DATA_CONNECTOR/     # HuggingFace / local data ingestion
├── 1_DATA_MINER/       # PDF/arXiv → deterministic embeddings
├── 2_DATA_ENCODER/     # embeddings → 12-slot ASIC brackets
├── 3_DATA_TRAINER/     # Evolutionary training
├── scripts/            # Bootstrap and utility scripts
└── KNIRVHASHER_pipeline_upgrade.md
```

## Prerequisites

Before running, ensure:
- Go 1.21+ installed
- `github.com/guiperry/text-embedder` v0.1.0 installed or available via `go mod tidy`
- `pdftotext` from poppler-utils (for PDF processing)
- At least 8GB RAM available

## Usage

### Quick Start — Full Pipeline

```bash
# Navigate to pipeline root
cd packages/KNIRVHASHER/pipeline

# Bootstrap with local data (recommended for first run)
./scripts/bootstrap_alpaca.sh
```

### Stage-by-Stage (Preferred: Deterministic Embeddings)

#### Stage 1: Data Miner (Deterministic Primary)

```bash
cd packages/KNIRVHASHER/pipeline/1_DATA_MINER

# Install dependencies (includes text-embedder)
go mod tidy

# Build
go build -o data-miner .

# Run with deterministic embeddings (PRIMARY - no external services required)
# Set EMBEDDING_BACKEND=deterministic to ensure deterministic mode
export EMBEDDING_BACKEND=deterministic
./data-miner -input ./documents -output training_data.json

# Run with arXiv mining
export EMBEDDING_BACKEND=deterministic
./data-miner -arxiv-enable -arxiv-max-papers 50
```

> **IMPORTANT**: Always set `EMBEDDING_BACKEND=deterministic` to use the primary deterministic text-embedder. Cloudflare and Ollama should only be used as fallbacks if the deterministic provider fails.

#### Fallback: Hybrid Embeddings (Cloudflare/Ollama)

Only use if deterministic text-embedder fails:

```bash
# Cloudflare primary with Ollama fallback
export CLOUDFLARE_EMBEDDINGS_URL="your-worker-endpoint"
export CLOUDFLARE_DAILY_LIMIT=5000
export EMBEDDING_BACKEND=cloudflare

# Or Ollama only
export OLLAMA_HOST=http://localhost:11434
export EMBEDDING_BACKEND=ollama
```

#### Stage 2: Data Encoder

```bash
cd packages/KNIRVHASHER/pipeline/2_DATA_ENCODER

# Build
go mod tidy
go build -o data-encoder .

# Encode Stage 1 output to 12-slot ASIC brackets
./data-encoder -input ../1_DATA_MINER/training_data.json -output ./brackets.nrv

# With custom LSH seed
./data-encoder -input ../1_DATA_MINER/training_data.json -output ./brackets.nrv -seed 1337
```

#### Stage 3: Data Trainer

```bash
cd packages/KNIRVHASHER/pipeline/3_DATA_TRAINER

# Build
go mod tidy
go build -o data-trainer .

# Run evolutionary training
./data-trainer -epochs 10 -population 64 -data ../2_DATA_ENCODER/brackets.nrv

# Or with config file
./data-trainer -config config.json
```

## Embedding Configuration

### Primary: Deterministic (text-embedder v0.1.0)

The deterministic provider generates reproducible 768-dim embeddings locally without network calls:

```go
// Uses github.com/guiperry/text-embedder
import textembed "github.com/guiperry/text-embedder/pkg/embed"

embedding := textembed.Embed(text) // Returns [768]float32
```

Environment:
```bash
export EMBEDDING_BACKEND=deterministic
```

### Fallbacks (DO NOT initialize - only for failures)

| Fallback | Env | Description |
|----------|-----|-------------|
| Cloudflare | `EMBEDDING_BACKEND=cloudflare` | Workers AI (requires API endpoint) |
| Ollama | `EMBEDDING_BACKEND=ollama` | Local Ollama server |

## 12-Slot ASIC Specification

The encoder maps 768-dim embeddings to 12 slots:

| Slot | Zone | Description |
|------|------|-------------|
| 0-3 | Identity Zone | LSH Projections (semantic compass) |
| 4-5 | Syntactic Registers | POS, Tense, Dependency |
| 6-8 | Memory Zone | History XOR (temporal recurrence) |
| 9 | Intent Flags | Question/Command/Code detection |
| 10 | Domain Signature | Math/Code/Prose |
| 11 | Temporal Lock | Position + Salt |

## Output

### Bracket Format (80 bytes)

```
Offset  Size  Field
0x00   32B  LSH Projections (Slots 0-3)
0x20   4B   SubSecondUS (timestamp)
0x24   1B   POSTag
0x25   1B   Tense
0x26   1B   Plurality
0x27   1B   DepHead
0x28   1B   IntentFlags
0x29   2B   DomainSig
0x2B   4B   GoldenSeed
0x2F   14B  Memory (XOR recursive)
0x3D   4B   LSH Salt
0x41   15B  Reserved
```

### Training Output

- `data/weights/best_seeds.csv` — Optimal seeds per bracket
- `data/checkpoints/` — Training state snapshots

## Troubleshooting

### Common Issues

1. **text-embedder import fails**
   ```bash
   cd packages/KNIRVHASHER/pipeline/1_DATA_MINER
   go get github.com/guiperry/text-embedder@v0.1.0
   go mod tidy
   ```

2. **Out of memory**
   - Reduce batch size: `-batch-size 8`
   - Reduce population: `-population 32`

3. **Checkpoint corruption**
   - Delete `checkpoints.db` and restart

4. **PDF extraction fails**
   ```bash
   sudo apt install poppler-utils
   ```

### Deterministic Embedding Verification

Verify deterministic mode is active:

```bash
export EMBEDDING_BACKEND=deterministic
./data-miner -input ./test.txt -output test.json -verbose
# Should log: "Using deterministic (local) embeddings"
```

## References

- Pipeline upgrade doc: `KNIRVHASHER_pipeline_upgrade.md`
- Data Miner README: `1_DATA_MINER/README.md`
- Data Trainer README: `3_DATA_TRAINER/README.md`
- Embedder provider: `1_DATA_MINER/internal/embedder/provider.go`
- Full specification: `packages/KNIRVBASE/go/KNIRVBASE_Upgrade.md`

## Notes

- **Always prioritize deterministic embeddings** — they are reproducible, require no network, and are faster for development
- The 21-pass temporal loop requires Syntactic Steering (Slots 4-5) for proper convergence
- Stage 3 training uses GRPO-style evolutionary selection
- GoldenSeed is solved per-bracket via evolutionary search