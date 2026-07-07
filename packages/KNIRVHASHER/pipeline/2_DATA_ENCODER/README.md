# Data Encoder

A high-throughput Go utility that transforms raw embeddings from the Data Miner into hardware-ready "Neural Frames" for the Antminer S3 ASIC training pipeline.

## Overview

The Data Encoder serves as Stage 2 in the ML training pipeline, bridging the gap between the Data Miner (Stage 1) and the Data Trainer (Stage 3). It performs the critical translation between human-readable text and the Antminer S3's specific SHA-256 requirements.

### Key Features

- **Dual Format Support**: Processes both JSON and Parquet input files with automatic format detection
- **Smart Fallback**: Gracefully falls back to JSON backup if Parquet file is corrupted or missing
- **Tokenization**: Converts text into integer token IDs using tiktoken (cl100k_base encoding)
- **Feature Mapping**: Projects 768-dimensional embeddings into 12 hardware-ready slots
- **Real Variance Analysis**: Selects signal dimensions from actual record embeddings or freshly generated embeddings
- **Parquet Output**: Generates compressed Parquet files optimized for training
- **Streaming Processing**: Memory-efficient JSON streaming for large datasets
- **Deterministic**: Reproducible results with configurable random seed

## Architecture

### Data Flow

```
Input: ai_knowledge_base.parquet (Primary) or ai_knowledge_base.json (Backup)
  ↓
[Automatic Format Detection]
  ↓
Tokenizer: Text → []int (Token IDs)
  ↓
Mapper: Embedding [768]float32 → [12]uint32 (ASIC Frames)
  ↓
Output: training_frames.parquet (for Data Trainer)
```

### Input Format Support

The Data Encoder supports **two input formats** with automatic detection and fallback:

#### 1. Parquet Input (Primary Format - Recommended)

Efficient columnar storage format with streaming read support:

```go
type DocumentRecord struct {
    FileName  string    `parquet:"name=file_name, type=BYTE_ARRAY, convertedtype=UTF8"`
    ChunkID   int32     `parquet:"name=chunk_id, type=INT32"`
    Content   string    `parquet:"name=content, type=BYTE_ARRAY, convertedtype=UTF8"`
    Embedding []float32 `parquet:"name=embedding, type=LIST, valuetype=FLOAT"`
}
```

**Advantages:**
- Columnar storage for efficient reads
- Streaming processing without loading entire file
- Better compression for large datasets
- Native binary format compatibility

#### 2. JSON Input (Backup Format)

Traditional JSON format maintained for backward compatibility:

```go
type MinedRecord struct {
    FileName  string    `json:"file_name"`
    ChunkID   int       `json:"chunk_id"`
    Content   string    `json:"content"`
    Embedding []float32 `json:"embedding"` // 768-dim vector
}
```

**Note:** The encoder can process both JSON arrays (`[ {...}, {...} ]`) and JSONL format (`{...}\n{...}`).

#### Input File Priority

The encoder automatically detects the best available input file:

1. **Primary**: `~/.local/share/data-miner/ai_knowledge_base.parquet`
2. **Backup**: `~/.local/share/data-miner/backup/json/ai_knowledge_base.json`
3. **Legacy**: `~/.local/share/data-miner/json/ai_knowledge_base.json`

### Output Schema (Parquet)

```go
type TrainingFrame struct {
    // Metadata (Dictionary encoding for compression)
    SourceFile    string `parquet:"name=source_file, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
    ChunkID       int32  `parquet:"name=chunk_id, type=INT32"`

    // Window Metadata
    WindowStart   int32 `parquet:"name=window_start, type=INT32"`
    WindowEnd     int32 `parquet:"name=window_end, type=INT32"`
    ContextLength int32 `parquet:"name=context_length, type=INT32"`

    // The Input (Individually named for easy debugging)
    AsicSlots0  int32 `parquet:"name=asic_slot_0, type=INT32"`
    AsicSlots1  int32 `parquet:"name=asic_slot_1, type=INT32"`
    AsicSlots2  int32 `parquet:"name=asic_slot_2, type=INT32"`
    AsicSlots3  int32 `parquet:"name=asic_slot_3, type=INT32"`
    AsicSlots4  int32 `parquet:"name=asic_slot_4, type=INT32"`
    AsicSlots5  int32 `parquet:"name=asic_slot_5, type=INT32"`
    AsicSlots6  int32 `parquet:"name=asic_slot_6, type=INT32"`
    AsicSlots7  int32 `parquet:"name=asic_slot_7, type=INT32"`
    AsicSlots8  int32 `parquet:"name=asic_slot_8, type=INT32"`
    AsicSlots9  int32 `parquet:"name=asic_slot_9, type=INT32"`
    AsicSlots10 int32 `parquet:"name=asic_slot_10, type=INT32"`
    AsicSlots11 int32 `parquet:"name=asic_slot_11, type=INT32"`

    // The Target (Golden Nonce)
    TargetTokenID int32 `parquet:"name=target_token_id, type=INT32"`

    // Seed Persistence (Placeholder for Stage 3)
    BestSeed string `parquet:"name=best_seed, type=BYTE_ARRAY"`
}
```

**Hardware Compatibility Notes:**
- Individual `AsicSlots0-11` fields allow easy debugging with Parquet viewers
- Each slot is `int32` for Parquet compatibility (bit-casting preserves raw values)
- Negative numbers in Parquet viewers represent maximum unsigned values (bit-compatible with ASIC)
- `BestSeed` uses `BYTE_ARRAY` for 256-bit SHA-256 seed storage

## Installation

### Prerequisites

- Go 1.21 or later
- Input data from Data Miner (JSON or Parquet format)

### Build

```bash
git clone <repository>
cd DATA_ENCODER
go build -o data-encoder .
```

### Dependencies

The following dependencies are automatically downloaded:

- `github.com/pkoukk/tiktoken-go` - Tokenization library
- `github.com/xitongsys/parquet-go` - Parquet file format
- `github.com/xitongsys/parquet-go-source` - File source adapters

## Usage

### Basic Usage

```bash
./data-encoder
```

The encoder automatically detects the best available input file in priority order:

1. `~/.local/share/data-miner/ai_knowledge_base.parquet` (Primary)
2. `~/.local/share/data-miner/backup/json/ai_knowledge_base.json` (Backup)
3. `~/.local/share/data-miner/json/ai_knowledge_base.json` (Legacy)

### Explicit Input Path

```bash
./data-encoder -input ~/.local/share/data-miner/ai_knowledge_base.parquet
```

This will create `training_frames.parquet` in the application data directory (`~/.local/share/data-encoder/`).

Explicit input paths are strict: if the requested file is missing, the encoder fails fast instead of falling back to another dataset. Automatic input discovery is only used when the default placeholder input path is left unchanged.

### Command Line Options

```
-input string
    Input file path - JSON or Parquet format (auto-detected if not specified)
    Default: ~/.local/share/data-miner/ai_knowledge_base.parquet (with fallback)

-output string
    Output Parquet file path (default "~/.local/share/data-encoder/training_frames.parquet")

-seed int
    Random seed for mapper reproducibility (default 1337)

-workers int
    Number of concurrent workers for Parquet writing (default 4)

-window-size int
    Sliding window size in tokens (default 128)

-window-stride int
    Stride between sliding windows (default 1)

-batch-size int
    Batch size for embedding API calls (default 32)
```

**Note:** By default, output files are saved to the application data directory (`~/.local/share/data-encoder/`). This directory is automatically created if it doesn't exist.

### Examples

#### Processing Parquet Input (Recommended)

```bash
./data-encoder -input ~/.local/share/data-miner/ai_knowledge_base.parquet
```

#### Using JSON Backup

```bash
./data-encoder -input ~/.local/share/data-miner/backup/json/ai_knowledge_base.json
```

#### Custom Output Path

```bash
./data-encoder \
  -input ~/.local/share/data-miner/ai_knowledge_base.parquet \
  -output ./output/training_data.parquet
```

#### Different Random Seed

```bash
./data-encoder \
  -input ai_knowledge_base.parquet \
  -seed 42 \
  -workers 8
```

#### Processing Large Files

For large datasets, increase worker count:

```bash
./data-encoder \
  -input large_dataset.parquet \
  -workers 16 \
  -output training_frames.parquet
```

#### Custom Sliding Window Parameters

```bash
./data-encoder \
  -input dataset.parquet \
  -window-size 256 \
  -window-stride 2 \
  -batch-size 64
```

## Feature Mapping Details

The mapper performs the following transformations:

1. **Projection**: Multiplies 768-dim embedding by a 24×768 projection matrix
2. **Activation**: Applies sigmoid to squash values to [0,1] range
3. **Quantization**: Scales to int16 range [-32768, 32767]
4. **Packing**: Combines 2 int16 values into 1 uint32 (bit packing)

This results in 24 features packed into 12 uint32 slots (48 bytes total).

### Variance Signal Selection

Before encoding JSON input, the encoder performs variance analysis over real 768-dimensional samples and writes the selected signal dimensions to `bge_signal_indices.json`. This file is saved next to the input data when possible, or to the encoder application data directory when it is writable.

The analyzer accepts existing `embedding` vectors from JSON/JSONL records. For GOAT/Alpaca-style records that only contain text fields such as `instruction`, `input`, `output`, `content`, or `text`, it generates embeddings through the configured embedding service and analyzes those vectors. Individual sample embedding failures are retried briefly. If no usable vectors can be produced, the encoder warns and continues with cached `bge_signal_indices.json`; if no cache exists, it uses spread fallback dimensions across the 768-dimensional space rather than the old sequential `0..23` placeholder.

## Input File Management

### Setting Up Parquet Input

The Data Miner should output Parquet files for optimal performance:

```go
// DocumentRecord schema for Parquet output
type DocumentRecord struct {
    FileName  string
    ChunkID   int32
    Content   string
    Embedding []float32
}
```

### Creating Backup JSON

For fallback scenarios, maintain a JSON backup:

```bash
# Create backup directory
mkdir -p ~/.local/share/data-miner/backup/json

# Convert Parquet to JSON (example using Python)
python3 -c "
import pyarrow.parquet as pq
df = pq.read_table('ai_knowledge_base.parquet').to_pandas()
df.to_json('~/.local/share/data-miner/backup/json/ai_knowledge_base.json', orient='records', lines=True)
"
```

### File Detection Behavior

The encoder implements intelligent file detection:

1. **Format Detection**: Automatically identifies Parquet vs JSON by:
   - File extension (`.parquet` vs `.json`)
   - Parquet magic bytes (`PAR1`) for extensionless files

2. **Fallback Logic**:
   - Primary Parquet unavailable → Try JSON backup
   - JSON backup unavailable → Try legacy JSON
   - All formats fail → Error with helpful message

3. **Corruption Handling**:
   - Parquet read error → Automatic fallback to JSON
   - JSON processing errors → Record-level skipping with logging

## Testing

Run all tests:

```bash
go test ./... -v
```

Run specific package tests:

```bash
go test ./pkg/tokenizer -v
go test ./pkg/mapper -v
go test ./pkg/schema -v
```

Run benchmarks:

```bash
go test ./pkg/mapper -bench=.
go test ./pkg/tokenizer -bench=.
```

## Project Structure

```
DATA_ENCODER/
├── main.go                 # CLI, orchestration, and file detection logic
├── main_test.go            # Integration tests
├── go.mod, go.sum         # Go module files
├── README.md              # This file
├── pkg/
│   ├── schema/
│   │   ├── input.go       # DocumentRecord (Parquet) and MinedRecord (JSON) schemas
│   │   ├── output.go      # TrainingFrame (Parquet output) schema
│   │   └── schema_test.go # Schema validation tests
│   ├── tokenizer/
│   │   ├── tokenizer.go   # Tiktoken wrapper
│   │   └── tokenizer_test.go
│   ├── mapper/
│   │   ├── mapper.go      # Feature mapping logic (768-dim → 12 slots)
│   │   └── mapper_test.go
│   ├── sliding/
│   │   ├── sliding.go     # Sliding window generation
│   │   └── sliding_test.go
│   └── embeddings/
│       ├── embeddings.go  # Cloudflare embeddings API client
│       └── embeddings_test.go
└── scripts/               # Utility scripts (if any)
```

## Performance Considerations

- **Format Selection**: Parquet provides 2-5x faster processing than JSON
- **Memory Efficiency**: Parquet enables streaming reads without loading entire file
- **Concurrency**: Configurable worker pool for Parquet writing
- **Compression**: Snappy compression for efficient storage
- **Throughput**:
  - Parquet: ~5000 records/second
  - JSON: ~1000 records/second
  - (Varies by text length and embedding size)

## Integration with Pipeline

### Stage 1: Data Miner

Outputs `ai_knowledge_base.parquet` containing:
- File names and chunk IDs for traceability
- Raw text content
- 768-dimensional embeddings from Ollama

### Stage 2: Data Encoder (This Tool)

Transforms data into `training_frames.parquet`:
- 12-slot ASIC-ready frames
- Token IDs for next-token prediction
- Metadata for debugging
- Automatic format detection and fallback

### Stage 3: Data Trainer

Consumes Parquet file to:
- Search for optimal SHA-256 seeds
- Fill in BestSeed field
- Train the ASIC-based model

## Troubleshooting

### "No input file found in any location"

Ensure at least one input file exists:

```bash
# Check for Parquet (primary)
ls -la ~/.local/share/data-miner/ai_knowledge_base.parquet

# Check for JSON backup
ls -la ~/.local/share/data-miner/backup/json/ai_knowledge_base.json

# Check for legacy JSON
ls -la ~/.local/share/data-miner/json/ai_knowledge_base.json
```

### Parquet Read Errors

If Parquet file is corrupted, the encoder automatically falls back to JSON:

```
2026/02/08 06:13:39 📁 Detected input file type: ai_knowledge_base.parquet
2026/02/08 06:13:39 ⚠️  Parquet read failed (failed to open parquet file: open failed: no such file or directory), attempting JSON fallback...
2026/02/08 06:13:39 📁 Detected input file type: ai_knowledge_base.json (backup)
```

### Out of Memory

Reduce worker count for large files:

```bash
./data-encoder -input large.parquet -workers 2
```

### Invalid JSON Records

The encoder skips malformed records with a warning. Check logs for details:

```
2026/02/08 06:13:39 ⚠️  Skipping unfixable record 5: invalid character...
```

### Parquet Viewer Compatibility

For debugging output files, use individual slot fields:

```python
import pyarrow.parquet as pq

df = pq.read_table('training_frames.parquet').to_pandas()
print(df[['asic_slot_0', 'asic_slot_1', 'target_token_id']].head())
```

Note: Negative numbers in slots represent maximum unsigned values (bit-compatible with ASIC).

## Development

### Adding New Features

1. Schema changes:
   - Input schemas: Update `pkg/schema/input.go`
   - Output schema: Update `pkg/schema/output.go`
2. Format detection: Update `detectInputFile()` in `main.go`
3. Format-specific processing: Update `readParquetFile()` or `processJSONFile()` in `main.go`
4. Mapping logic: Update `pkg/mapper/mapper.go`
5. CLI options: Update `main.go` flag definitions

### Testing New Formats

1. Add test files in appropriate format
2. Update `TestValidateConfig` in `main_test.go`
3. Add format-specific tests if needed
4. Run integration tests: `go test ./... -run Integration`

### Code Style

- Follow standard Go conventions
- Add tests for new functionality
- Use meaningful variable names
- Document public APIs
- Maintain backward compatibility when adding formats

## License

GPLv3

## Contributing

Contributions welcome! Please ensure:
- Tests pass (`go test ./...`)
- Code is formatted (`gofmt -w .`)
- Documentation is updated
- New formats include fallback support

## Contact

For issues and feature requests, please use the GitHub issue tracker.

---

**Part of the HASHER ML Training Pipeline**
