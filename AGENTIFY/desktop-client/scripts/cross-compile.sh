#!/bin/bash
# Cross-compilation script for KNIRVENGINE
# This script builds binaries for Windows, macOS, and Linux platforms

set -e  # Exit immediately if a command exits with a non-zero status

# Configuration variables
# Use BINARY_NAME from environment variable if set, otherwise default to knirv-engine
BINARY_NAME=${BINARY_NAME:-knirv-engine}
# Use VERSION from environment variable if set, otherwise get from git
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
VERSION=$(echo "$VERSION" | xargs)  # Trim whitespace
MAIN_PACKAGE_PATH="./"
OUTPUT_DIR="./dist"
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Check if we should build in production mode (with embedded GUI)
PRODUCTION_MODE=${PRODUCTION_MODE:-1}
echo "PRODUCTION_MODE flag received as: $PRODUCTION_MODE"

# Check if we should build plugins
BUILD_PLUGINS=${BUILD_PLUGINS:-1}
echo "BUILD_PLUGINS flag received as: $BUILD_PLUGINS"

# Define target platforms (sh-compatible)
PLATFORMS="windows/amd64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64"

# Build frontend if in production mode
if [ "$PRODUCTION_MODE" = "1" ]; then
  echo "Building React frontend for production..."
  if [ -d "./gui" ]; then
    cd gui
    if [ ! -d "node_modules" ]; then
      echo "Installing frontend dependencies..."
      npm install
    fi
    echo "Building frontend..."
    npm run build
    cd ..
    echo "Frontend build completed."
  else
    echo "Warning: GUI directory not found. Skipping frontend build."
  fi
fi

# Create output directories
mkdir -p "$OUTPUT_DIR/$VERSION"
mkdir -p "$OUTPUT_DIR/release_assets"
echo "Building version: $VERSION"
echo "Binary name: $BINARY_NAME"
echo "Output directory: $OUTPUT_DIR/$VERSION"
echo "Release assets directory: $OUTPUT_DIR/release_assets"
echo "Production mode: $PRODUCTION_MODE"

# Build for each platform
for platform in $PLATFORMS; do
  os=$(echo "$platform" | cut -d'/' -f1)
  arch=$(echo "$platform" | cut -d'/' -f2)
  
  # Create versioned output directory for build process
  output_dir="$OUTPUT_DIR/$VERSION/${os}_${arch}"
  mkdir -p "$output_dir"
  
  # Set binary name with extension for Windows
  binary_name="$BINARY_NAME"
  if [ "$os" = "windows" ]; then
    binary_name="${binary_name}.exe"
  fi
  
  echo "Building for $os/$arch -> $output_dir/$binary_name"

  # Build the binary
  # Check if we're cross-compiling
  is_cross_compile=0
  if [ "$os" != "$(go env GOOS)" ] || [ "$arch" != "$(go env GOARCH)" ]; then
    is_cross_compile=1
  fi

  # Set build tags based on production mode
  build_tags=""
  if [ "$PRODUCTION_MODE" = "1" ]; then
    build_tags="embed"
    echo "Building with embedded frontend assets for $os/$arch"
  else
    echo "Building development version for $os/$arch"
  fi

  # Build the binary with appropriate settings
  if [ $is_cross_compile -eq 1 ]; then
    # Cross-compilation - use CGO_ENABLED=0 for simplicity and compatibility
    echo "Cross-compiling for $os ($arch)..."
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -v -tags="$build_tags" \
      -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" \
      -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH
  else
    # Native build - can use CGO if needed
    echo "Building natively for $os ($arch)..."
    GOOS=$os GOARCH=$arch CGO_ENABLED=1 go build -v -tags="$build_tags" \
      -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" \
      -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH || \
    (echo "CGO build failed, falling back to non-CGO build..." && \
     GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -v -tags="$build_tags" \
     -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" \
     -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH)
  fi
  
  if [ $? -ne 0 ]; then
    echo "ERROR: Build failed for $os/$arch"
    
    # Ask if we should continue with other platforms
    echo "Do you want to continue building for other platforms? (y/n)"
    read -r continue_build
    
    if [[ "$continue_build" != "y" && "$continue_build" != "Y" ]]; then
      echo "Build process aborted."
      exit 1
    else
      echo "Continuing with other platforms..."
      # Create a placeholder file to indicate this platform failed
      echo "Build failed on $(date)" > "$output_dir/BUILD_FAILED.txt"
    fi
  fi
done

# Build plugins if requested
if [ "$BUILD_PLUGINS" = "1" ]; then
  echo "Building plugins..."
  if [ -d "./plugins" ]; then
    # Find all .go files in plugins directory that are not test files
    plugin_files=$(find ./plugins -name "*.go" -not -name "*_test.go" 2>/dev/null || true)
    if [ -n "$plugin_files" ]; then
      for plugin_file in $plugin_files; do
        plugin_name=$(basename "$plugin_file" .go)
        echo "Building plugin: $plugin_name"
        go build -buildmode=plugin -o "./plugins/${plugin_name}.so" "$plugin_file" || \
        echo "Warning: Failed to build plugin $plugin_name"
      done
    else
      echo "No plugin source files found in ./plugins directory"
    fi
  else
    echo "Plugins directory not found. Skipping plugin build."
  fi
fi

echo "All builds completed successfully in $OUTPUT_DIR/$VERSION"
echo "Build summary:"
find "$OUTPUT_DIR/$VERSION" -type f -name "$BINARY_NAME*" | sort

# Copy binaries to release_assets with consistent names
echo ""
echo "Copying binaries to release_assets with consistent names..."
for platform in $PLATFORMS; do
  os=$(echo "$platform" | cut -d'/' -f1)
  arch=$(echo "$platform" | cut -d'/' -f2)
  
  source_dir="$OUTPUT_DIR/$VERSION/${os}_${arch}"
  
  # Set binary name with extension for Windows
  binary_name="$BINARY_NAME"
  if [ "$os" = "windows" ]; then
    binary_name="${binary_name}.exe"
    archive_ext="zip"
  else
    archive_ext="tar.gz"
  fi
  
  # Check if binary exists
  if [ -f "$source_dir/$binary_name" ]; then
    # Create consistent archive name without version
    consistent_archive_name="${BINARY_NAME}_${os}_${arch}.${archive_ext}"
    
    echo "Creating $consistent_archive_name..."
    if [ "$os" = "windows" ]; then
      (cd "$source_dir" && zip -jrq "$OUTPUT_DIR/release_assets/$consistent_archive_name" "$binary_name")
    else
      tar -czvf "$OUTPUT_DIR/release_assets/$consistent_archive_name" -C "$source_dir" "$binary_name"
    fi
    
    echo "Created $OUTPUT_DIR/release_assets/$consistent_archive_name"
  else
    echo "Warning: Binary not found at $source_dir/$binary_name"
  fi
done

echo ""
echo "KNIRVENGINE cross-compilation completed!"
echo "Binaries are available in: $OUTPUT_DIR/$VERSION"
echo "Release assets are available in: $OUTPUT_DIR/release_assets"
if [ "$PRODUCTION_MODE" = "1" ]; then
  echo "Built with embedded React frontend for production deployment"
fi
if [ "$BUILD_PLUGINS" = "1" ]; then
  echo "Plugins built and available in: ./plugins/"
fi