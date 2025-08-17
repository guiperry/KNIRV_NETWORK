# KNIRVORACLE Documentation System

This directory contains the organized documentation for the KNIRVORACLE project. The documentation is automatically generated from the original docs in the `docs/` directory using the `scripts/organize_docs.js` script.

## Structure

The documentation is organized in the `docsify/` subdirectory and follows this structure:

- **Getting Started**: Installation guides, configuration, and quick start information
- **Core Concepts**: Fundamental concepts of KNIRVORACLE including blockchain architecture, MCP, capabilities, and URI scheme
- **Protocols**: Detailed documentation of all protocols used in KNIRVORACLE
- **API Reference**: Complete reference for all KNIRVORACLE APIs
- **Components**: Information about the various components in the KNIRVORACLE ecosystem
- **Guides**: Step-by-step guides for common tasks
- **Troubleshooting**: Solutions to common problems
- **SDK**: Documentation for the KNIRVORACLE SDK

## Updating the Documentation

The documentation is automatically updated when you run `make build` or `make docs`. You can also update it manually:

```bash
# From the root directory
make docs

# Or from this directory
npm run build
```

The script is idempotent - running it multiple times will produce the same result as running it once. This means you can safely run the script whenever you update the original documentation without worrying about duplicate content or other side effects.

## Viewing the Documentation

### Option 1: Using the embedded server

The documentation is embedded in the main application and can be accessed at:

```
http://localhost:<port>/docs/
```

When the KNIRVORACLE server is running.

### Option 2: Using the Docsify development server

You can also view the documentation using the Docsify development server:

```bash
# From this directory
npm start
```

This will start a local server at http://localhost:3000 where you can view the documentation.

### Option 3: Using the standalone docs server

For development purposes, you can run the standalone documentation server:

```bash
# From the root directory
go run docs_server.go serve-docs
```

This will start a server at http://localhost:8080/docs/ where you can view the documentation.

## Implementation Details

The documentation system uses:

1. **Docsify**: A documentation site generator that doesn't generate static HTML files
2. **Go embed**: The documentation is embedded in the Go binary using the `embed` package
3. **HTTP FileServer**: The embedded documentation is served using Go's HTTP file server

The embedding is implemented in `docs_server.go` with this directive:

```go
//go:embed documentation/docsify
var docsifyFiles embed.FS
```

This embeds all files from the `documentation/docsify` directory into the binary.