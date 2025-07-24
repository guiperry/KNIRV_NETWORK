# Scripts Directory

This directory contains various utility scripts for the Agentic-Engine project.

## Go Utilities

### Monitor Cerebras Usage
- **Location**: `scripts/monitor-cerebras/main.go`
- **Purpose**: Monitor Cerebras API usage, check API health, and provide optimization recommendations
- **Usage**: 
  ```bash
  go run ./scripts/monitor-cerebras
  go run ./scripts/monitor-cerebras --health-check
  ```

### Update Agent Images
- **Location**: `scripts/update-agent-images/main.go`
- **Purpose**: Update all existing agents to use the new Agentify logo as default image
- **Usage**: 
  ```bash
  go run ./scripts/update-agent-images
  # Or use the shell script wrapper:
  ./scripts/update_agent_images.sh
  ```

## Shell Scripts

### Build Scripts
- `build-desktop.sh` - Build desktop application
- `cross-compile.sh` - Cross-platform compilation
- `release.sh` - Release preparation
- `run_production.sh` - Run production build and start the application

### Database Scripts
- `seed_db.sh` - Seed database with initial data

### Agent Management
- `update_agent_images.sh` - Shell wrapper for the Go agent image updater

## Notes

- All Go utilities are organized in separate subdirectories to avoid package conflicts
- Each Go utility can be run independently using `go run ./scripts/<utility-name>`
- Shell scripts should be run from the project root directory
