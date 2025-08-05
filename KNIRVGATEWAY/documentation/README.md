# KNIRV Network Documentation

This directory contains the generated documentation for the KNIRV Network project.

## Structure

- `docsify/` - Docsify-based documentation site with integrated whitepapers
- `whitepapers/` - Standalone whitepaper directory for direct access

## Serving the Documentation Locally

To serve the documentation locally for testing:

```bash
# Install dependencies (if not already installed)
npm install

# Serve the documentation
npm run serve
```

The documentation will be available at `http://localhost:3000`

## Features

- ✅ **Whitepaper Integration**: All whitepapers are accessible through the Docsify navigation
- ✅ **Mermaid Diagrams**: Full support for Mermaid diagram rendering
- ✅ **Search**: Built-in search functionality
- ✅ **Syntax Highlighting**: Support for multiple programming languages
- ✅ **Dark Theme**: Optimized for the KNIRV Network branding

## Regenerating Documentation

To regenerate the documentation from source files, run the doc generator script from the project root:

```bash
node doc_generator.js
```

This will:
1. Process all markdown files from the `docs/` directory
2. Copy whitepapers to both standalone and Docsify directories
3. Generate navigation and index files
4. Update only changed files (hash-based detection)
