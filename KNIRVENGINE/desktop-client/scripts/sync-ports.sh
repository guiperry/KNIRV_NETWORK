#!/bin/bash

# Sync port configuration from main ports.config to gui/ports.config
# This ensures both backend and frontend use the same port configuration

if [ -f ports.config ]; then
    # Extract API_PORT from main ports.config file
    API_PORT=$(grep "^API_PORT=" ports.config | cut -d'=' -f2)

    if [ ! -z "$API_PORT" ]; then
        # Update or create gui/ports.config with API_PORT
        cat > gui/ports.config << EOF
# Frontend Port Configuration
# This file is automatically synced from the root ports.config file

# API Server Port (for frontend to connect to backend)
API_PORT=$API_PORT
EOF
        echo "✅ Synced API_PORT=$API_PORT to gui/ports.config"
    else
        echo "⚠️  API_PORT not found in ports.config file"
    fi
else
    echo "⚠️  ports.config file not found"
fi
