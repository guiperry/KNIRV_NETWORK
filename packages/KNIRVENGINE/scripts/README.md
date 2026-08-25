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

Most shell scripts resolve the project root from their own location and can be
invoked from any working directory.

### Build & Deployment Scripts

- `build-desktop.sh`: Build the desktop application.
- `cross-compile.sh`: Cross-platform compilation.
- `deploy.sh`: Create a local distributable desktop package.
- `start-desktop.sh`: Start the Electron desktop application from source.
- `start-browser.sh`: Run the local browser development workflow.
- `start-dev.sh`: Alias for the browser development workflow.
- `test-desktop-api.js`: Smoke-test the local desktop API.
- `test-desktop-system.js`: Smoke-test the local API and sandbox service.
- `release.sh`: Release preparation.
- `run_production.sh`: Build and launch the Electron desktop application.
- `copy-env-to-electron.sh`: Copy `.env` into an unpacked `electron-builder` release for local testing.

### Database Scripts

- `seed_db.sh`: Seed the database with initial data.

### Agent Management

- `update_agent_images.sh`: Shell wrapper for the Go agent image updater.


##  Additional Notes

- Each Go utility is designed to be run independently.
- The migrated shell scripts resolve their paths from their own location, so they
  can be invoked from any working directory.
