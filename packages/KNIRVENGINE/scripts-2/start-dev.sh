#!/bin/bash

# Start KNIRVENGINE in development mode with security middleware disabled

echo "🚀 Starting KNIRVENGINE in Development Mode"
echo "🔓 Security middleware will be DISABLED"
echo "📡 WebSocket connections will be allowed from all origins"
echo ""

# Set development environment variables
export DEVELOPMENT_MODE=true
export NODE_ENV=development

# Start the server
./knirv-engine
