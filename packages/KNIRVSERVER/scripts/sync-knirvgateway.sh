#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KNIRVSERVER_DIR="$(dirname "$SCRIPT_DIR")"
KNIRVGATEWAY_DIR="$(dirname "$KNIRVSERVER_DIR")/KNIRVGATEWAY"
DEST_DIR="$KNIRVSERVER_DIR/pkg/knirvgateway"

echo "=========================================="
echo "KNIRVGATEWAY Sync Script"
echo "=========================================="
echo "Source: $KNIRVGATEWAY_DIR"
echo "Destination: $DEST_DIR"
echo ""

FORCE_FLAG=""
if [[ "$1" == "--force" || "$1" == "-f" ]]; then
    FORCE_FLAG="--delete"
    echo "Force flag set - will delete files not in source"
fi

echo "Creating directory structure..."
mkdir -p "$DEST_DIR"/{internal,cmd/gateway}

echo "Copying internal packages..."
rsync -av $FORCE_FLAG "$KNIRVGATEWAY_DIR/internal/" "$DEST_DIR/internal/"

echo "Copying cmd..."
rsync -av $FORCE_FLAG "$KNIRVGATEWAY_DIR/cmd/" "$DEST_DIR/cmd/"

echo "Copying go.mod and go.sum..."
cp "$KNIRVGATEWAY_DIR/go.mod" "$DEST_DIR/go.mod"
if [ -f "$KNIRVGATEWAY_DIR/go.sum" ]; then
    cp "$KNIRVGATEWAY_DIR/go.sum" "$DEST_DIR/go.sum"
fi

echo "Copying package.json if exists..."
if [ -f "$KNIRVGATEWAY_DIR/package.json" ]; then
    cp "$KNIRVGATEWAY_DIR/package.json" "$DEST_DIR/"
fi

echo "Keeping original module path (github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY)..."
# No change to go.mod

echo "Keeping original import paths..."
# No change to import paths

echo ""
echo "Sync complete!"
echo ""
echo "Next steps:"
echo "1. Run 'cd $DEST_DIR && go mod tidy' to update dependencies"
echo "2. Add the replace directive to backend_server/go.mod if not present:"
echo "   replace github.com/KNIRV/KNIRV_NETWORK/KNIRVSERVER/pkg/knirvgateway => ../pkg/knirvgateway"
echo "3. Build the gateway binary: cd $DEST_DIR/cmd/gateway && go build -o knirvgateway"
