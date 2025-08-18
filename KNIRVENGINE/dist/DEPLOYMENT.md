# KNIRVENGINE Deployment Guide

## Quick Start

1. **Start the complete system**:
   ```bash
   ./start-system.sh
   ```

2. **Start individual components**:
   ```bash
   ./start-desktop-host.sh    # Desktop Host only
   ./start-mobile-controller.sh     # Mobile Tool dev server
   ```

## System URLs

- **Desktop Host API**: http://localhost:8082
- **Mobile Tool**: ./mobile-controller/index.html
- **MCP WebSocket**: ws://localhost:8082/api/mcp/ws
- **Health Check**: http://localhost:8082/api/health

## Configuration

Edit `config.json` to customize system settings.

## Troubleshooting

1. **Port conflicts**: Change port in config.json
2. **Permission errors**: Ensure scripts are executable
3. **Missing dependencies**: Check system requirements
4. **HRM not working**: Ensure weights.safetensors is present

## Logs

- Desktop Host logs: Check console output
- System logs: Check deploy.log in parent directory

## Support

See README.md for detailed documentation and support information.
