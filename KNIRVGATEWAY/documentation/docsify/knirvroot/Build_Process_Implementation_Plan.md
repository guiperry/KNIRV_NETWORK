

---

**Source**: KNIRVROOT/docs/pending_implementation_plans/Build_Process_Implementation_Plan.md

# KNIRVCHAIN Build Process Implementation Plan

## Overview

This document outlines the implementation plan for automating the build, packaging, and release process for KNIRVCHAIN. The plan addresses the requirements for:

1. Cross-platform binary compilation
2. Proper versioning using Git tags
3. Archive creation with consistent naming for auto-updates
4. Digital signing of release assets
5. Checksum generation
6. GitHub release automation

## 1. Architecture

The build process is split into two main components:

1. **Cross-compilation script** (`scripts/cross-compile.sh`): Handles the compilation of binaries for different platforms
2. **Makefile**: Orchestrates the entire process, including:
   - Calling the cross-compilation script
   - Packaging binaries into archives
   - Signing archives
   - Generating checksums
   - Uploading to GitHub releases
   - Managing versioning

## 2. Auto-Update Configuration

The auto-update mechanism uses the following format for archive naming:

```go
RepositoryURL: fmt.Sprintf("https://github.com/%s/%s", GitHubRepoOwner, GitHubRepoName)
ArchiveName: fmt.Sprintf("{{bin_name}}_{{tag}}_{{os}}_{{arch}}")
```

Where:
- `{{bin_name}}`: The application name (KNIRVCHAIN)
- `{{tag}}`: The Git tag/version (e.g., v1.0.1)
- `{{os}}`: Target operating system (linux, windows, darwin)
- `{{arch}}`: Target architecture (amd64, arm64)

## 3. Implementation Details

### 3.1 Cross-Compilation Script

The `scripts/cross-compile.sh` script:

- Accepts `VERSION` and `BINARY_NAME` environment variables
- Builds binaries for multiple OS/architecture combinations:
  - Windows (amd64) with .exe extension
  - macOS (amd64, arm64)
  - Linux (amd64, arm64)
- Handles platform-specific build configurations:
  - Windows: Uses MinGW compiler if available for CGO support
  - macOS: Builds with CGO_ENABLED=0 and appropriate build tags
  - Linux: Uses CGO_ENABLED=1 for native builds, CGO_ENABLED=0 for cross-compilation
- Injects version information via ldflags: `-X main.AppVersion=$VERSION -X main.BuildDate=$BUILD_DATE`
- Outputs compiled binaries to `./dist/$VERSION/${os}_${arch}/`

### 3.2 Makefile Configuration

The Makefile is organized into several sections:

1. **Configuration Section**:
   - `APP_NAME := KNIRVCHAIN` (ensuring consistent capitalization)
   - `VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev-$(shell git rev-parse --short HEAD)")`
   - `TARGETS := windows/amd64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64`
   - Output directories for compiled binaries (`CROSS_COMPILE_OUTPUT_DIR := ./dist`)
   - Output directories for release assets (`RELEASE_ASSETS_DIR := ./release_assets`)
   - GPG key configuration (`GPG_KEY_ID ?= YOUR_GPG_KEY_ID_HERE`)

2. **Production Build Targets**:
   - `all`: Default target that builds and packages everything
   - `build-binaries`: Runs cross-compilation script
   - `package-release`: Archives, signs, and checksums binaries
   - `upload-release`: Uploads assets to GitHub

3. **Development Targets**:
   - `build`: Builds the application for the host OS
   - `build/all`: Builds for all target platforms
   - `run`: Runs the application
   - `run/live`: Runs with live reloading
   - `tidy`: Tidies modfiles and formats code
   - `test`: Runs tests
   - `test/cover`: Runs tests with coverage
   - `audit`: Runs quality control checks

4. **Version Control Section**:
   - Robust version parsing from Git tags
   - Version calculation for micro, minor, and major releases
   - Tag generation and management
   - Automated release workflow

### 3.3 Build Process Flow

1. **Determine version**: 
   - Extract from Git tags or generate from commit hash
   - Parse into MAJOR.MINOR.MICRO components
   - Calculate next version numbers based on release type

2. **Cross-compile binaries**: 
   - Call `scripts/cross-compile.sh` with appropriate parameters
   - Pass VERSION and BINARY_NAME environment variables
   - Build for all target platforms

3. **Package binaries**: 
   - Process each target platform using the `process_target` macro
   - Create .tar.gz archives for Linux/macOS
   - Create .zip archives for Windows
   - Use consistent naming format: `KNIRVCHAIN_${VERSION}_${OS}_${ARCH}.${ARCHIVE_EXT}`

4. **Sign packages**: 
   - Use GPG to create detached signatures
   - Skip signing if GPG_KEY_ID is not properly configured
   - Store signatures as .sig files alongside archives

5. **Generate checksums**: 
   - Create SHA256 checksums for each archive
   - Store checksums as .sha256 files

6. **Upload to GitHub**: 
   - Check for GitHub CLI installation
   - Create GitHub release if it doesn't exist
   - Upload all assets (archives, signatures, checksums)

### 3.4 Versioning System

The Makefile includes a sophisticated versioning system that:

- Extracts version components (MAJOR.MINOR.MICRO) from Git tags using `git describe`
- Handles cases where no tags exist by defaulting to 0.0.0
- Calculates next version numbers based on semantic versioning rules:
  - Micro (patch): Increment the third number (X.Y.Z → X.Y.Z+1)
  - Minor: Increment the second number, reset third (X.Y.Z → X.Y+1.0)
  - Major: Increment the first number, reset others (X.Y.Z → X+1.0.0)
- Provides targets for viewing and creating new version tags
- Automates the release workflow through the `RELEASE_WORKFLOW` macro

## 4. Using the Makefile

### 4.1 Development Workflow

For day-to-day development:

```bash
# Build for local development
make build

# Run the application
make run

# Run with live reloading
make run/live

# Run tests
make test

# Format code and tidy dependencies
make tidy

# Run quality control checks
make audit
```

### 4.2 Release Workflow

To create a new release:

```bash
# Ensure your Git working directory is clean
git status

# View the next version numbers
make version-micro
make version-minor
make version-major

# Create a new release (choose one)
make release-micro   # For patch releases (bug fixes)
make release-minor   # For minor releases (backward-compatible features)
make release-major   # For major releases (breaking changes)

# Upload to GitHub
make upload-release
```

The release process will:
1. Check for a clean Git working directory
2. Ask for confirmation
3. Create and push a new Git tag
4. Build binaries for all target platforms
5. Package, sign, and checksum the binaries
6. Prepare assets for GitHub release

### 4.3 Manual Build Steps

If you need more control over the build process:

```bash
# Clean previous builds
make clean

# Build binaries only
make build-binaries

# Package binaries into archives (without uploading)
make package-release

# Upload previously packaged assets
make upload-release
```

### 4.4 Configure GPG Signing

To enable GPG signing of release assets:

```bash
# Set GPG key ID as an environment variable
export GPG_KEY_ID="Your Name <your.email@example.com>"

# Or edit the Makefile directly
# GPG_KEY_ID := YOUR_GPG_KEY_ID_HERE
```

## 5. Target Platforms

The build process supports the following platforms:

- Linux (amd64, arm64)
- Windows (amd64)
- macOS (amd64, arm64)

Additional platforms can be added by modifying:
- The `TARGETS` variable in the Makefile
- The `PLATFORMS` variable in the cross-compile.sh script

## 6. Auto-Update Integration

The auto-update mechanism in the application uses the archive naming format:
`KNIRVCHAIN_v1.0.0_linux_amd64.tar.gz`

Key considerations:
- The binary name must be consistent (KNIRVCHAIN)
- The application uses `filepath.Base(os.Args[0])` to determine the binary name
- When running the compiled binary (e.g., `./KNIRVCHAIN`), the update mechanism works correctly
- When developing with `go run .`, the binary name might be unpredictable

A wrapper script (`run_wrapper.sh`) can be generated with:
```bash
make generate/wrapper
```

This script handles automatic restarts after updates.

## 7. Troubleshooting

Common issues and solutions:

- **GPG signing fails**: 
  - Ensure GPG is properly configured: `gpg --list-keys`
  - Set the correct key ID: `export GPG_KEY_ID="Your Key ID"`

- **GitHub upload fails**: 
  - Verify GitHub CLI is installed: `gh --version`
  - Authenticate if needed: `gh auth login`

- **Version calculation issues**: 
  - If no Git tags exist, create an initial tag: `git tag -a v0.1.0 -m "Initial release"`
  - Push the tag: `git push --tags`

- **Cross-compilation errors**:
  - For Windows builds, install MinGW: `sudo apt install mingw-w64`
  - For macOS builds, ensure you're using CGO_ENABLED=0

## 8. Future Improvements

Potential enhancements to consider:

- Add CI/CD integration for automated builds on push
- Implement release notes generation
- Add support for additional platforms
- Integrate code signing for Windows and macOS binaries
- Add Docker container builds

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
