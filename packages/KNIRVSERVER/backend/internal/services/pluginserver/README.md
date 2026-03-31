# Tool Server

A standalone HTTP server for serving, uploading, and managing WASM Plugin Tool files in the KNIRVSERVER ecosystem.

## Overview

The Plugin Tool Server provides a simple HTTP API for managing compiled WASM plugin tools. It supports:

- **File Serving**: Download plugin tool files via HTTP
- **File Upload**: Upload new plugin tool files
- **File Listing**: List all available plugin tools
- **File Management**: Delete plugin tool files
- **Server Info**: Get server status and configuration

## Quick Start

### Building

```bash
# Build the server
make build

# Or build for specific platforms
make build-linux
make build-windows
make build-darwin
make build-all
```

### Running

```bash
# Run with default settings (port 8082, ./tools directory)
./plugin-server

# Run with custom settings
./plugin-server --port 8081 --tools ./my-tools --name "My Server"

# Run with all options
./plugin-server \
  --port 8082 \
  --tools ./tools \
  --name "Production Server" \
  --register \
  --api http://localhost:3000 \
  --cors
```

### Using Make

```bash
# Run development server
make run-dev

# Set up directories
make setup

# Clean build artifacts
make clean
```

## Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8082` | Port to listen on |
| `--tools` | `./tools` | Directory containing Plugin Tools |
| `--name` | `"KNIRVSERVER Plugin Tool Server"` | Name of this server instance |
| `--register` | `false` | Register this server with the KNIRVSERVER system |
| `--api` | `http://localhost:3000` | URL of the KNIRVSERVER API |
| `--cors` | `true` | Enable CORS headers |

## API Endpoints

### GET /info
Get server information and status.

**Response:**
```json
{
  "name": "KNIRVSERVER Plugin Tool Server",
  "port": 8082,
  "tool_dir": "./tools",
  "start_time": "2024-01-01T12:00:00Z",
  "version": "1.0.0"
}
```

### GET /list
List all available plugin tools.

**Response:**
```json
{
  "tools": [
    {
      "name": "my-tool.wasm",
      "size": 1024,
      "last_modified": "2024-01-01T12:00:00Z"
    }
  ],
  "count": 1
}
```

### GET /tools/{name}
Download a specific plugin tool file.

**Parameters:**
- `name`: Name of the plugin tool file

**Response:** Binary file download

### POST /upload
Upload a new plugin tool file.

**Request:** Multipart form with `plugin-tool` file field

**Response:**
```json
{
  "success": true,
  "filename": "my-tool.wasm",
  "size": 1024,
  "message": "Plugin tool uploaded successfully"
}
```

### DELETE /delete/{name}
Delete a plugin tool file.

**Parameters:**
- `name`: Name of the plugin tool file to delete

**Response:**
```json
{
  "success": true,
  "message": "Tool deleted successfully"
}
```

## Usage Examples

### Upload a Plugin Tool

```bash
curl -X POST \
  -F "plugin-tool=@my-tool.wasm" \
  http://localhost:8082/upload
```

### Download a Plugin Tool

```bash
curl -O http://localhost:8082/tools/my-tool.wasm
```

### List Available Tools

```bash
curl http://localhost:8082/list
```

### Delete an Tool

```bash
curl -X DELETE http://localhost:8082/delete/my-tool.wasm
```

## Security Considerations

- File names are sanitized to prevent directory traversal attacks
- Upload size is limited to 100MB
- Only regular files are served (no directories or special files)
- CORS can be disabled for production environments

## Integration with KNIRVSERVER

The server can be registered with the KNIRVSERVER system using the `--register` flag. This allows the blockchain to track server instances and their available plugin tools.

When registered, the server provides a reliable endpoint for:
- Plugin tool distribution
- Version management
- Load balancing across multiple server instances

## Development

### Project Structure

```
cmd/plugin-server/
├── main.go          # Main server implementation
├── Makefile         # Build and development tasks
└── README.md        # This file
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/guiperry/toolchain.git
cd toolchain/cmd/plugin-server

# Install dependencies
make deps

# Build the server
make build

# Run tests
make test
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make fmt` and `make lint`
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
