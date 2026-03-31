#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KNIRVSERVER_DIR="$(dirname "$SCRIPT_DIR")"
KNIRVCHAIN_DIR="$(dirname "$KNIRVSERVER_DIR")/KNIRVCHAIN"
DEST_DIR="$KNIRVSERVER_DIR/pkg/knirvchain"

echo "=========================================="
echo "KNIRVCHAIN Sync Script"
echo "=========================================="
echo "Source: $KNIRVCHAIN_DIR"
echo "Destination: $DEST_DIR"
echo ""

FORCE_FLAG=""
if [[ "$1" == "--force" || "$1" == "-f" ]]; then
    FORCE_FLAG="--delete"
    echo "Force flag set - will delete files not in source"
fi

echo "Creating directory structure..."
mkdir -p "$DEST_DIR"/{internal,cmd/knirvchain}

echo "Copying internal packages..."
rsync -av $FORCE_FLAG "$KNIRVCHAIN_DIR/internal/" "$DEST_DIR/internal/"

echo "Copying cmd..."
rsync -av $FORCE_FLAG "$KNIRVCHAIN_DIR/cmd/" "$DEST_DIR/cmd/"

echo "Copying go.mod and go.sum..."
cp "$KNIRVCHAIN_DIR/go.mod" "$DEST_DIR/go.mod"
if [ -f "$KNIRVCHAIN_DIR/go.sum" ]; then
    cp "$KNIRVCHAIN_DIR/go.sum" "$DEST_DIR/go.sum"
fi

echo ""
echo "Sync complete!"
echo ""
echo "Next steps:"
echo "1. Run 'cd $DEST_DIR && go mod tidy' to update dependencies"
echo "2. Add the replace directive to backend/go.mod if not present:"
echo "   replace github.com/KNIRV/KNIRV_NETWORK/KNIRVSERVER/pkg/knirvchain => ../pkg/knirvchain"
echo "3. Build the chain binary: cd $DEST_DIR/cmd/knirvchain && go build -o knirvchain"
