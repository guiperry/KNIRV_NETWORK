#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KNIRVSERVER_DIR="$(dirname "$SCRIPT_DIR")"
KNIRVHASHER_DIR="$(dirname "$KNIRVSERVER_DIR")/KNIRVHASHER"
DEST_DIR="$KNIRVSERVER_DIR/pkg/knirvhasher"

echo "=========================================="
echo "KNIRVHASHER Sync Script"
echo "=========================================="
echo "Source: $KNIRVHASHER_DIR"
echo "Destination: $DEST_DIR"
echo ""

FORCE_FLAG=""
if [[ "$1" == "--force" || "$1" == "-f" ]]; then
    FORCE_FLAG="--delete"
    echo "Force flag set - will delete files not in source"
fi

echo "Creating directory structure..."
mkdir -p "$DEST_DIR"/{cmd,internal,pkg,pipeline,toolchain,bin,tests}

echo "Copying cmd..."
rsync -av $FORCE_FLAG "$KNIRVHASHER_DIR/cmd/" "$DEST_DIR/cmd/"

echo "Copying internal packages..."
rsync -av $FORCE_FLAG "$KNIRVHASHER_DIR/internal/" "$DEST_DIR/internal/"

echo "Copying pkg packages..."
rsync -av $FORCE_FLAG "$KNIRVHASHER_DIR/pkg/" "$DEST_DIR/pkg/"

echo "Copying pipeline..."
rsync -av $FORCE_FLAG "$KNIRVHASHER_DIR/pipeline/" "$DEST_DIR/pipeline/"

echo "Copying toolchain..."
rsync -av $FORCE_FLAG "$KNIRVHASHER_DIR/toolchain/" "$DEST_DIR/toolchain/"

echo "Copying tests..."
rsync -av $FORCE_FLAG "$KNIRVHASHER_DIR/tests/" "$DEST_DIR/tests/"

echo "Copying go.mod and go.sum..."
cp "$KNIRVHASHER_DIR/go.mod" "$DEST_DIR/go.mod"
if [ -f "$KNIRVHASHER_DIR/go.sum" ]; then
    cp "$KNIRVHASHER_DIR/go.sum" "$DEST_DIR/go.sum"
fi

echo "Copying Makefile..."
cp "$KNIRVHASHER_DIR/Makefile" "$DEST_DIR/Makefile"

echo "Copying other config files..."
[ -f "$KNIRVHASHER_DIR/.env" ] && cp "$KNIRVHASHER_DIR/.env" "$DEST_DIR/.env"
[ -f "$KNIRVHASHER_DIR/README.md" ] && cp "$KNIRVHASHER_DIR/README.md" "$DEST_DIR/README.md"
[ -f "$KNIRVHASHER_DIR/LICENSE" ] && cp "$KNIRVHASHER_DIR/LICENSE" "$DEST_DIR/LICENSE"
[ -f "$KNIRVHASHER_DIR/docker-compose.yml" ] && cp "$KNIRVHASHER_DIR/docker-compose.yml" "$DEST_DIR/"
[ -f "$KNIRVHASHER_DIR/Dockerfile" ] && cp "$KNIRVHASHER_DIR/Dockerfile" "$DEST_DIR/"

echo "Keeping original module path (hasher)..."
echo "No changes to go.mod module path required"

echo ""
echo "Sync complete!"
echo ""
echo "Next steps:"
echo "1. Run 'cd $DEST_DIR && go mod tidy' to update dependencies"
echo "2. Add the replace directive to KNIRVSERVER/go.mod if not present:"
echo "   replace hasher => ./pkg/knirvhasher"
echo "3. Build hasher pipeline binaries: cd $DEST_DIR && make build-pipeline-all"
