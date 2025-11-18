#!/bin/bash
# Cross-compilation script for Go applications
# This script builds binaries for Windows, macOS, and Linux platforms

set -e  # Exit immediately if a command exits with a non-zero status

# Configuration variables
# Use BINARY_NAME from environment variable if set, otherwise default to KNIRVCHAIN
BINARY_NAME=${BINARY_NAME:-KNIRVCHAIN}
# Use VERSION from environment variable if set, otherwise get from git
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
VERSION=$(echo "$VERSION" | xargs)  # Trim whitespace
MAIN_PACKAGE_PATH="./"
OUTPUT_DIR="./dist"
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
# Check if we should build without WebView
NO_WEBVIEW=${NO_WEBVIEW:-0}
echo "NO_WEBVIEW flag received as: $NO_WEBVIEW"

# Check if we should build for a specific role
NODE_ROLE=${NODE_ROLE:-""}
echo "NODE_ROLE flag received as: $NODE_ROLE"

# Validate NODE_ROLE
if [ "$NODE_ROLE" != "" ] && [ "$NODE_ROLE" != "root" ] && [ "$NODE_ROLE" != "bootnode" ] && [ "$NODE_ROLE" != "developer" ] && [ "$NODE_ROLE" != "client" ]; then
  echo "Invalid NODE_ROLE: $NODE_ROLE. Must be one of: root, bootnode, developer, client, or empty for default."
  exit 1
fi

# Define target platforms (sh-compatible)
PLATFORMS="windows/amd64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64"

# Create output directory
mkdir -p "$OUTPUT_DIR/$VERSION"
echo "Building version: $VERSION"
echo "Binary name: $BINARY_NAME"
echo "Output directory: $OUTPUT_DIR/$VERSION"

# Build for each platform
for platform in $PLATFORMS; do
  os=$(echo "$platform" | cut -d'/' -f1)
  arch=$(echo "$platform" | cut -d'/' -f2)
  
  output_dir="$OUTPUT_DIR/$VERSION/${os}_${arch}"
  mkdir -p "$output_dir"
  
  # Set binary name with extension for Windows
  binary_name="$BINARY_NAME"
  if [ "$os" = "windows" ]; then
    binary_name="${binary_name}.exe"
    
    # Set C compiler for Windows CGo cross-compilation if MinGW is found
    env_prefix=""
    if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
      env_prefix="CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++"
    elif command -v mingw32-gcc >/dev/null 2>&1; then
      env_prefix="CC=mingw32-gcc CXX=mingw32-g++"
    else
      echo "Warning: MinGW C/C++ compiler for Windows (e.g., x86_64-w64-mingw32-gcc) not found in PATH."
      echo "CGo cross-compilation for Windows may fail. Consider installing MinGW."
    fi
  fi
  
  echo "Building for $os/$arch -> $output_dir/$binary_name"
  
  # Build the binary
  # Check if we're cross-compiling
  is_cross_compile=0
  if [ "$os" != "$(go env GOOS)" ] || [ "$arch" != "$(go env GOARCH)" ]; then
    is_cross_compile=1
  fi

  # Set build tags based on platform, NO_WEBVIEW flag, and NODE_ROLE
  if [ "$NO_WEBVIEW" = "1" ]; then
    # When NO_WEBVIEW is set, use no_webview tag for all platforms
    build_tags="no_webview"
    echo "Building without WebView for $os (NO_WEBVIEW=1)"
  elif [ "$os" = "windows" ]; then
    # For Windows, use fyne_gui tag unless NO_WEBVIEW is set
    build_tags="fyne_gui"
    echo "Using Fyne GUI for Windows build"
  else
    # For non-Windows platforms, don't specify fyne_gui tag to use the default altgui
    build_tags=""
    echo "Using default GUI for $os build"
  fi
  
  # Add role tag if specified
  if [ "$NODE_ROLE" != "" ]; then
    if [ "$build_tags" != "" ]; then
      build_tags="${build_tags},${NODE_ROLE}"
    else
      build_tags="${NODE_ROLE}"
    fi
    echo "Building for $NODE_ROLE role"
    
    # Append role to binary name
    binary_name="${binary_name}_${NODE_ROLE}"
  fi

  if [ "$os" = "windows" ] && [ "$NO_WEBVIEW" != "1" ]; then
    # For Windows (only when not in NO_WEBVIEW mode), check for MinGW compilers
    CC_FOR_TARGET="x86_64-w64-mingw32-gcc"
    CXX_FOR_TARGET="x86_64-w64-mingw32-g++"
    
    # Check if compilers exist
    if command -v $CC_FOR_TARGET &> /dev/null && command -v $CXX_FOR_TARGET &> /dev/null; then
      echo "Building for Windows ($arch) with MinGW compilers and Fyne GUI..."
      
      # Try to build with Fyne GUI
      CC=$CC_FOR_TARGET CXX=$CXX_FOR_TARGET GOOS=$os GOARCH=$arch CGO_ENABLED=1 \
        go build -v -tags="$build_tags" -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" \
        -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH || \
      (echo "Fyne GUI build failed, falling back to default GUI..." && \
       GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -v -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH)
    else
      echo "WARNING: MinGW compilers ($CC_FOR_TARGET, $CXX_FOR_TARGET) not found."
      
      if [ -n "$env_prefix" ]; then
        # Use the env_prefix that was detected earlier
        echo "Using detected MinGW compilers with Fyne GUI..."
        eval "$env_prefix GOOS=$os GOARCH=$arch CGO_ENABLED=1 go build -v -tags=\"$build_tags\" -ldflags=\"-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE\" -o \"$output_dir/$binary_name\" $MAIN_PACKAGE_PATH" || \
        (echo "Fyne GUI build failed, falling back to default GUI..." && \
         GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -v -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH)
      else
        echo "No MinGW compilers found. Building with default GUI..."
        GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -v -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH
      fi
    fi
  elif [ "$NO_WEBVIEW" = "1" ]; then
    # For NO_WEBVIEW mode, always use CGO_ENABLED=0 to avoid WebView dependencies
    echo "Building for $os ($arch) without WebView dependencies..."
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -v -tags="$build_tags" -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH
  # For macOS and other platforms - use the default GUI (altgui.go)
  else
    if [ $is_cross_compile -eq 1 ]; then
      # Cross-compilation - try with CGO first, fallback to no-CGO if it fails
      echo "Cross-compiling for $os ($arch) with default GUI..."
      GOOS=$os GOARCH=$arch CGO_ENABLED=1 go build -v -tags="$build_tags" -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH || \
      (echo "CGO build failed, falling back to non-CGO build..." && \
       GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -v -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH)
    else
      # Native build - can use CGO
      echo "Building for $os ($arch) with native compiler and default GUI..."
      GOOS=$os GOARCH=$arch CGO_ENABLED=1 go build -v -tags="$build_tags" -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH || \
      (echo "CGO build failed, falling back to non-CGO build..." && \
       GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -v -ldflags="-s -w -X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE" -o "$output_dir/$binary_name" $MAIN_PACKAGE_PATH)
    fi
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

echo "All builds completed successfully in $OUTPUT_DIR/$VERSION"
echo "Build summary:"
find "$OUTPUT_DIR/$VERSION" -type f -name "$BINARY_NAME*" | sort

# Clone the dist folder to the external directory
EXTERNAL_DIR="/home/gperry/Documents/GitHub/cloud-equities/KNIRVORACLE_PUBLIC"
echo "Copying dist folder to external directory: $EXTERNAL_DIR"
mkdir -p "$EXTERNAL_DIR"
rsync -av --delete "$OUTPUT_DIR/" "$EXTERNAL_DIR/build/"
if [ $? -eq 0 ]; then
  echo "Successfully copied dist folder to $EXTERNAL_DIR/build"
else
  echo "ERROR: Failed to copy dist folder to external directory"
  exit 1
fi