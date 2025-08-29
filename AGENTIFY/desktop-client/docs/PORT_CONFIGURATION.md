# Port Configuration Guide

The Agentic-Engine now supports configurable server and GUI ports through dedicated `ports.config` files that can be safely committed to version control.

## Configuration

### Port Configuration Files

Edit the `ports.config` file in the project root to configure ports:

```config
# Port Configuration for Agentic-Engine
# This file contains non-sensitive port configuration that can be safely committed to version control

# Backend API Server Port
API_PORT=8081

# Frontend GUI Server Port
GUI_PORT=3000
```

### Automatic Synchronization

The project includes a sync script that automatically updates the frontend configuration:

```bash
./sync-env.sh
```

This script:
- Reads `API_PORT` from the main `ports.config` file
- Updates `gui/ports.config` with the API port for the frontend
- Ensures both backend and frontend use the same API port

## Usage Examples

### Default Configuration
```config
API_PORT=8081
GUI_PORT=8080
```
- Backend API: `http://localhost:8081`
- Frontend GUI: `http://localhost:8080`

### Custom Ports
```config
API_PORT=9000
GUI_PORT=4000
```
- Backend API: `http://localhost:9000`
- Frontend GUI: `http://localhost:4000`

## How It Works

### Backend (Go)
The main.go file reads from ports.config:
```go
// Read port configuration from ports.config file
portConfig, err := utils.ReadPortConfig("ports.config")
if err != nil {
    log.Printf("⚠️  Warning: Failed to read ports.config: %v. Using defaults.", err)
    portConfig = &utils.PortConfig{APIPort: 8081, GUIPort: 8080}
}
apiPort := portConfig.APIPort
```

### Frontend (React/Vite)
The frontend configuration automatically adapts:
- `vite.config.ts` reads from `gui/ports.config` for proxy configuration
- `api.ts` uses relative URLs that work with the Vite proxy

## Build and Run

1. **Configure ports** in `ports.config`:
   ```config
   API_PORT=8081
   GUI_PORT=8080
   ```

2. **Sync port configuration**:
   ```bash
   ./sync-env.sh
   ```

3. **Build and run**:
   ```bash
   go build -o knirv-engine
   ./knirv-engine
   ```

The application will automatically use the configured ports.

## Command Line Override

You can still override the GUI port via command line:
```bash
./knirv-engine --gui-port 4000
```

This takes precedence over the ports.config file.

## Troubleshooting

### Port Already in Use
If you get a "port already in use" error:
1. Check what's using the port: `lsof -i :8081`
2. Kill the process or choose a different port
3. Update `ports.config` and run `./sync-env.sh`

### Frontend Can't Connect to Backend
1. Ensure both ports are configured correctly
2. Run `./sync-env.sh` to sync configurations
3. Restart both backend and frontend servers
