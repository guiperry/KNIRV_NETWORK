#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PIPELINE_DIR="$(dirname "$SCRIPT_DIR")"
DATA_MAPPER_DIR="$PIPELINE_DIR/1_DATA_MAPPER"
DATA_ENCODER_DIR="$PIPELINE_DIR/2_DATA_ENCODER"
DATA_TRAINER_DIR="$PIPELINE_DIR/3_DATA_SEEDER"
ALPACA_DIR="$PIPELINE_DIR/AlpacaDataCleaned-main"
OUTPUT_DIR="$HOME/.local/share/knirvhasher"

echo "=== KNIRVHASHER Pipeline Bootstrap ==="
echo "Using AlpacaDataCleaned as source data"
echo ""

mkdir -p "$OUTPUT_DIR/miner"
mkdir -p "$OUTPUT_DIR/encoder"

ALPACA_JSON="$ALPACA_DIR/alpaca_data_cleaned.json"
if [ ! -f "$ALPACA_JSON" ]; then
    echo "ERROR: Alpaca data not found at $ALPACA_JSON"
    exit 1
fi

echo "[1/5] Building Data Mapper..."
cd "$DATA_MAPPER_DIR"
go mod tidy 2>/dev/null || true
go build -o data-mapper ./cmd/data-mapper 2>/dev/null || {
    echo "ERROR: Failed to build data-mapper"
    exit 1
}

echo "[2/5] Loading Alpaca data and generating deterministic embeddings..."
export EMBEDDING_BACKEND=deterministic
OUTPUT_JSON="$OUTPUT_DIR/miner/alpaca_mined.json"

python3 -c "
import json

alpaca_file = '$ALPACA_JSON'
output_file = '$OUTPUT_JSON'

print(f'Reading {alpaca_file}...')
with open(alpaca_file, 'r') as f:
    data = json.load(f)

print(f'Processing {len(data)} records...')
records = []
for i, item in enumerate(data):
    instruction = item.get('instruction', '')
    input_text = item.get('input', '')
    output = item.get('output', '')

    if input_text:
        text = f'Instruction: {instruction}\nInput: {input_text}\nOutput: {output}'
    else:
        text = f'Instruction: {instruction}\nOutput: {output}'

    records.append({
        'file_name': f'alpaca/{i}',
        'chunk_id': 0,
        'content': text,
    })

    if (i + 1) % 5000 == 0:
        print(f'  Processed {i + 1}/{len(data)} records...')

print(f'Writing {len(records)} records to {output_file}...')
with open(output_file, 'w') as f:
    json.dump(records, f)

print('Done!')
"

if [ ! -f "$OUTPUT_JSON" ]; then
    echo "ERROR: Failed to generate mined data"
    exit 1
fi

echo "[3/5] Building Data Encoder..."
cd "$DATA_ENCODER_DIR"
go mod tidy 2>/dev/null || true
go build -o data-encoder . 2>/dev/null || {
    echo "ERROR: Failed to build data-encoder"
    exit 1
}

echo "[4/5] Encoding embeddings to 12-slot ASIC brackets..."
OUTPUT_NRV="$OUTPUT_DIR/encoder/alpaca_frames.nrv"
./data-encoder -input "$OUTPUT_JSON" -output "$OUTPUT_NRV" -seed 1337 || {
    echo "ERROR: Failed to encode data"
    echo "Note: If encoder expects Parquet, convert JSON to Parquet first"
    exit 1
}

echo "[5/5] Building Data Trainer..."
cd "$DATA_TRAINER_DIR"
go mod tidy 2>/dev/null || true
go build -o data-seeder . 2>/dev/null || {
    echo "ERROR: Failed to build data-seeder"
    exit 1
}

echo ""
echo "=== Bootstrap Complete ==="
echo "Output files:"
echo "  - Miner:  $OUTPUT_JSON"
echo "  - Encoder: $OUTPUT_NRV"
echo ""
echo "Next steps:"
echo "  1. To run training:"
echo "     cd $DATA_TRAINER_DIR"
echo "     ./data-seeder -epochs 10 -population 64 -data $OUTPUT_NRV"
echo ""
echo "  2. Or run incrementally:"
echo "     cd $DATA_MAPPER_DIR && ./data-mapper -single-batch"
