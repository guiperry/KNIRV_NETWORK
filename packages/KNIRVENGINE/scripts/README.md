# KNIRVENGINE Desktop Client Scripts

[TOC]

## Overview

This directory contains utility scripts for the KNIRVENGINE desktop client.  These scripts are categorized into Go utilities and shell scripts, each designed for specific tasks related to building, deploying, and managing the application.

## Go Utilities

All Go utilities are organized in separate subdirectories to avoid package conflicts. Each can be run independently using `go run ./scripts/<utility-name>`.

### Monitor Cerebras Usage
- **Location**: `scripts/monitor-cerebras/main.go`
- **Purpose**: Monitor Cerebras API usage, check API health, and provide optimization recommendations.
- **Usage**:
  ```bash
  go run ./scripts/monitor-cerebras
  go run ./scripts/monitor-cerebras --health-check
  ```

### Update Agent Images
- **Location**: `scripts/update-agent-images/main.go`
- **Purpose**: Update all existing agents to use the new Agentify logo as default image.
- **Usage**:
  ```bash
  go run ./scripts/update-agent-images
  # Or use the shell script wrapper:
  ./scripts/update_agent_images.sh
  ```

## Shell Scripts

These scripts should be run from the project root directory.

### Build & Deployment Scripts

- `build-desktop.sh`: Build the desktop application.
- `cross-compile.sh`: Cross-platform compilation.
- `release.sh`: Release preparation.
- `run_production.sh`: Run the production build and start the application.  This script performs the following steps:
    1. Rebuilds the frontend: `cd gui && npm run build`
    2. Rebuilds the backend (if needed): `cd .. && go build -o knirv-engine .`
    3. Repackages the Electron app: `cd electron && npm run pack:linux`
    4. Runs the application: `./dist/linux-unpacked/knirv-engine-desktop`
    **Note:** Vulkan warnings are disabled via `app.disableHardwareAcceleration()` in `main.js`. If you encounter graphics issues, you can re-enable hardware acceleration and use command line flags (see comments in `main.js`).

### Database Scripts

- `seed_db.sh`: Seed the database with initial data.

### Agent Management

- `update_agent_images.sh`: Shell wrapper for the Go agent image updater.


##  Additional Notes

- Each Go utility is designed to be run independently.
- Shell scripts are intended to be executed from the project's root directory.

