#!/bin/bash

# build-cortex-desktop.sh - Build cross-platform desktop executables with embedded cortex.wasm

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CORTEX_ROOT="$(cd "$PROJECT_ROOT/../../KNIRVCORTEX" && pwd)"
DIST_DIR="$PROJECT_ROOT/dist"
DESKTOP_DIR="$PROJECT_ROOT/desktop"

# Build information
VERSION=${VERSION:-"1.0.0"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo -e "${BLUE}🚀 Building KNIRV-ENGINE Desktop with Embedded Cortex.wasm${NC}"
echo -e "${BLUE}Version: $VERSION${NC}"
echo -e "${BLUE}Build Time: $BUILD_TIME${NC}"
echo -e "${BLUE}Git Commit: $GIT_COMMIT${NC}"
echo ""

# Function to print status
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if cortex.wasm exists
CORTEX_WASM_PATH="$CORTEX_ROOT/dist/cortex.wasm"
if [ ! -f "$CORTEX_WASM_PATH" ]; then
    print_error "cortex.wasm not found at $CORTEX_WASM_PATH"
    echo "Please build cortex.wasm first by running:"
    echo "  cd $CORTEX_ROOT && make build-cortex"
    exit 1
fi

print_status "Found cortex.wasm at $CORTEX_WASM_PATH"

# Create dist directory
mkdir -p "$DIST_DIR"

# Copy cortex.wasm to desktop directory for embedding
print_status "Copying cortex.wasm for embedding..."
cp "$CORTEX_WASM_PATH" "$DESKTOP_DIR/cortex.wasm"

# Build for different platforms
PLATFORMS=(
    "linux/amd64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    echo ""
    echo -e "${BLUE}🔨 Building for $GOOS/$GOARCH...${NC}"
    
    # Set output filename
    OUTPUT_NAME="knirv-engine-cortex-$GOOS-$GOARCH"
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="$OUTPUT_NAME.exe"
    fi
    
    OUTPUT_PATH="$DIST_DIR/$OUTPUT_NAME"
    
    # Build with embedded cortex.wasm
    env GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build \
        -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT -s -w" \
        -tags="cortex_embedded" \
        -o "$OUTPUT_PATH" \
        "$PROJECT_ROOT"
    
    if [ $? -eq 0 ]; then
        # Get file size
        if command -v stat >/dev/null 2>&1; then
            if [[ "$OSTYPE" == "darwin"* ]]; then
                SIZE=$(stat -f%z "$OUTPUT_PATH")
            else
                SIZE=$(stat -c%s "$OUTPUT_PATH")
            fi
            SIZE_MB=$((SIZE / 1024 / 1024))
            print_status "Built $OUTPUT_NAME (${SIZE_MB}MB)"
        else
            print_status "Built $OUTPUT_NAME"
        fi
        
        # Create compressed archive
        ARCHIVE_NAME="knirv-engine-cortex-$GOOS-$GOARCH"
        if [ "$GOOS" = "windows" ]; then
            ARCHIVE_PATH="$DIST_DIR/$ARCHIVE_NAME.zip"
            (cd "$DIST_DIR" && zip -q "$ARCHIVE_NAME.zip" "$OUTPUT_NAME")
        else
            ARCHIVE_PATH="$DIST_DIR/$ARCHIVE_NAME.tar.gz"
            (cd "$DIST_DIR" && tar -czf "$ARCHIVE_NAME.tar.gz" "$OUTPUT_NAME")
        fi
        
        if [ -f "$ARCHIVE_PATH" ]; then
            print_status "Created archive: $(basename "$ARCHIVE_PATH")"
        fi
    else
        print_error "Failed to build for $GOOS/$GOARCH"
    fi
done

# Create a universal build script for development
echo ""
echo -e "${BLUE}📝 Creating development build script...${NC}"

cat > "$DIST_DIR/build-dev.sh" << 'EOF'
#!/bin/bash
# Development build script for KNIRV-ENGINE with embedded cortex.wasm

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🔨 Building development version with embedded cortex.wasm..."

# Build for current platform
go build \
    -ldflags="-X main.Version=dev -X main.BuildTime=$(date -u +"%Y-%m-%dT%H:%M:%SZ") -X main.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")" \
    -tags="cortex_embedded" \
    -o "$SCRIPT_DIR/knirv-engine-dev" \
    "$PROJECT_ROOT"

echo "✅ Development build complete: $SCRIPT_DIR/knirv-engine-dev"
echo ""
echo "To run:"
echo "  $SCRIPT_DIR/knirv-engine-dev"
EOF

chmod +x "$DIST_DIR/build-dev.sh"
print_status "Created development build script: $DIST_DIR/build-dev.sh"

# Create README for the dist directory
echo ""
echo -e "${BLUE}📚 Creating distribution README...${NC}"

cat > "$DIST_DIR/README.md" << EOF
# KNIRV-ENGINE Desktop with Embedded Cortex.wasm

This directory contains cross-platform desktop executables of KNIRV-ENGINE with embedded cortex.wasm.

## Build Information
- Version: $VERSION
- Build Time: $BUILD_TIME
- Git Commit: $GIT_COMMIT
- Cortex.wasm: Embedded

## Available Platforms

### Linux
- \`knirv-engine-cortex-linux-amd64\` - Linux x86_64
- \`knirv-engine-cortex-linux-amd64.tar.gz\` - Compressed archive

### macOS
- \`knirv-engine-cortex-darwin-amd64\` - macOS Intel
- \`knirv-engine-cortex-darwin-arm64\` - macOS Apple Silicon
- \`knirv-engine-cortex-darwin-amd64.tar.gz\` - Compressed archive (Intel)
- \`knirv-engine-cortex-darwin-arm64.tar.gz\` - Compressed archive (Apple Silicon)

### Windows
- \`knirv-engine-cortex-windows-amd64.exe\` - Windows x86_64
- \`knirv-engine-cortex-windows-amd64.zip\` - Compressed archive

## Features

Each executable includes:
- Complete KNIRV-ENGINE desktop client
- Embedded cortex.wasm (no external dependencies)
- LoRA Adapter Engine support
- Cross-platform WASM runtime (wazero)
- Full cognitive task processing
- Agent compilation and management

## Usage

1. Download the appropriate executable for your platform
2. Make it executable (Linux/macOS): \`chmod +x knirv-engine-cortex-*\`
3. Run: \`./knirv-engine-cortex-*\`

The embedded cortex.wasm will be automatically initialized on startup.

## Development

Use \`build-dev.sh\` to create a development build for your current platform:

\`\`\`bash
./build-dev.sh
\`\`\`

## Architecture

The desktop executables use Go's embed directive to include cortex.wasm at compile time:
- No external WASM files needed
- Single executable deployment
- Automatic cortex initialization
- Full LoRA and cognitive task support

For more information, see the KNIRV-CORTEX documentation.
EOF

print_status "Created distribution README: $DIST_DIR/README.md"

# Clean up temporary files
rm -f "$DESKTOP_DIR/cortex.wasm"

echo ""
echo -e "${GREEN}🎉 Build complete!${NC}"
echo -e "${GREEN}Distribution files created in: $DIST_DIR${NC}"
echo ""
echo "Available executables:"
ls -la "$DIST_DIR"/knirv-engine-cortex-* 2>/dev/null || true
echo ""
echo "Available archives:"
ls -la "$DIST_DIR"/*.tar.gz "$DIST_DIR"/*.zip 2>/dev/null || true
echo ""
echo -e "${BLUE}To test a build:${NC}"
echo "  $DIST_DIR/knirv-engine-cortex-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64"
