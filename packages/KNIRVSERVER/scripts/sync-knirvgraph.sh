#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KNIRVSERVER_DIR="$(dirname "$SCRIPT_DIR")"
KNIRVGRAPH_DIR="$(dirname "$KNIRVSERVER_DIR")/KNIRVGRAPH"
DEST_DIR="$KNIRVSERVER_DIR/pkg/knirvgraph"

echo "=========================================="
echo "KNIRVGRAPH Sync Script"
echo "=========================================="
echo "Source: $KNIRVGRAPH_DIR"
echo "Destination: $DEST_DIR"
echo ""

FORCE_FLAG=""
if [[ "$1" == "--force" || "$1" == "-f" ]]; then
    FORCE_FLAG="--delete"
    echo "Force flag set - will delete files not in source"
fi

echo "Creating directory structure..."
mkdir -p "$DEST_DIR"/{internal,cmd/node,cmd/cli,pkg,dataengine,examples}

echo "Copying internal packages..."
rsync -av $FORCE_FLAG "$KNIRVGRAPH_DIR/internal/" "$DEST_DIR/internal/"

echo "Copying cmd..."
rsync -av $FORCE_FLAG "$KNIRVGRAPH_DIR/cmd/" "$DEST_DIR/cmd/"

echo "Copying pkg..."
rsync -av $FORCE_FLAG "$KNIRVGRAPH_DIR/pkg/" "$DEST_DIR/pkg/"

echo "Copying dataengine..."
rsync -av $FORCE_FLAG "$KNIRVGRAPH_DIR/dataengine/" "$DEST_DIR/dataengine/"

echo "Copying examples..."
rsync -av $FORCE_FLAG "$KNIRVGRAPH_DIR/examples/" "$DEST_DIR/examples/"

echo "Copying go.mod and go.sum..."
cp "$KNIRVGRAPH_DIR/go.mod" "$DEST_DIR/go.mod"
[ -f "$KNIRVGRAPH_DIR/go.sum" ] && cp "$KNIRVGRAPH_DIR/go.sum" "$DEST_DIR/go.sum"

echo ""
echo "Sync complete!"
echo ""
echo "Next steps:"
echo "1. Run 'cd $DEST_DIR && go mod tidy'"
echo "2. Build with: make graph-build"
