

---

**Source**: KNIRVROOT/cmd/plugin-server/README.md

# KNIRVROOT Plugin Agent Server

A standalone HTTP server for serving, uploading, and managing WASM Plugin Agent files in the KNIRVROOT ecosystem.

## Overview

The Plugin Agent Server provides a simple HTTP API for managing compiled WASM plugin agents. It supports:

- **File Serving**: Download plugin agent files via HTTP
- **File Upload**: Upload new plugin agent files
- **File Listing**: List all available plugin agents
- **File Management**: Delete plugin agent files
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
# Run with default settings (port 8080, ./agents directory)
./plugin-server

# Run with custom settings
./plugin-server --port 8081 --agents ./my-agents --name "My Server"

# Run with all options
./plugin-server \
  --port 8080 \
  --agents ./agents \
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
| `--port` | `8080` | Port to listen on |
| `--agents` | `./agents` | Directory containing Plugin Agents |
| `--name` | `"KNIRVROOT Plugin Agent Server"` | Name of this server instance |
| `--register` | `false` | Register this server with the KNIRVROOT system |
| `--api` | `http://localhost:3000` | URL of the KNIRVROOT API |
| `--cors` | `true` | Enable CORS headers |

## API Endpoints

### GET /info
Get server information and status.

**Response:**
```json
{
  "name": "KNIRVROOT Plugin Agent Server",
  "port": 8080,
  "agent_dir": "./agents",
  "start_time": "2024-01-01T12:00:00Z",
  "version": "1.0.0"
}
```

### GET /list
List all available plugin agents.

**Response:**
```json
{
  "agents": [
    {
      "name": "my-agent.wasm",
      "size": 1024,
      "last_modified": "2024-01-01T12:00:00Z"
    }
  ],
  "count": 1
}
```

### GET /agents/{name}
Download a specific plugin agent file.

**Parameters:**
- `name`: Name of the plugin agent file

**Response:** Binary file download

### POST /upload
Upload a new plugin agent file.

**Request:** Multipart form with `plugin-agent` file field

**Response:**
```json
{
  "success": true,
  "filename": "my-agent.wasm",
  "size": 1024,
  "message": "Plugin agent uploaded successfully"
}
```

### DELETE /delete/{name}
Delete a plugin agent file.

**Parameters:**
- `name`: Name of the plugin agent file to delete

**Response:**
```json
{
  "success": true,
  "message": "Agent deleted successfully"
}
```

## Usage Examples

### Upload a Plugin Agent

```bash
curl -X POST \
  -F "plugin-agent=@my-agent.wasm" \
  http://localhost:8080/upload
```

### Download a Plugin Agent

```bash
curl -O http://localhost:8080/agents/my-agent.wasm
```

### List Available Agents

```bash
curl http://localhost:8080/list
```

### Delete an Agent

```bash
curl -X DELETE http://localhost:8080/delete/my-agent.wasm
```

## Security Considerations

- File names are sanitized to prevent directory traversal attacks
- Upload size is limited to 100MB
- Only regular files are served (no directories or special files)
- CORS can be disabled for production environments

## Integration with KNIRVROOT

The server can be registered with the KNIRVROOT system using the `--register` flag. This allows the blockchain to track server instances and their available plugin agents.

When registered, the server provides a reliable endpoint for:
- Plugin agent distribution
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
git clone https://github.com/guiperry/KNIRVROOT.git
cd KNIRVROOT/cmd/plugin-server

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


---

<div class="footer-links">


© 2025 KNIRV Network
</div>
