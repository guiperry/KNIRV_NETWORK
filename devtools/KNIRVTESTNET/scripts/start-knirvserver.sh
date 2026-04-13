#!/bin/bash
set -e

echo "🚀 Starting KNIRV-SERVER unified testnet node with new architecture..."

# Create necessary directories
mkdir -p logs data config

# Check if unified binary exists
if [ ! -f "./bin/knirvserver" ]; then
    echo "❌ Error: KNIRV-SERVER unified binary not found."
    echo "   Please run: npm run build:nexus"
    echo "   Or: bash scripts/build-knirvserver.sh"
    exit 1
fi

# Get the correct base directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"

# Verify binary is executable
if [ ! -x "./bin/knirvserver" ]; then
    echo "🔧 Making binary executable..."
    chmod +x ./bin/knirvserver
fi

# Setup testnet configuration
echo "⚙️ Setting up testnet configuration..."
if [ -f "config/nexus-testnet.yaml" ]; then
    echo "  ✅ Using existing testnet configuration"
elif [ -f "../packages/KNIRVSERVER/config/nexus-testnet.yaml" ]; then
    echo "  📋 Copying testnet config from KNIRVSERVER..."
    cp ../packages/KNIRVSERVER/config/nexus-testnet.yaml config/nexus-testnet.yaml
else
    echo "  🔧 Creating default testnet configuration..."
    cat > config/nexus-testnet.yaml << 'EOF'
# KNIRV-SERVER Testnet Configuration
host: "0.0.0.0"
port: 8084
backend_port: 8080
log_level: "info"
testnet: true
EOF
fi

# Create data directory for NEXUS
mkdir -p data/knirvserver

# Set environment variables for testnet mode
export NEXUS_HOST="0.0.0.0"
export NEXUS_PORT="8084"
export NEXUS_BACKEND_PORT="8080"
export NEXUS_LOG_LEVEL="info"
export NEXUS_TESTNET="true"

# Start unified KNIRV-SERVER binary with new architecture
echo "🌐 Starting KNIRV-SERVER unified binary with embedded frontend and backend..."
echo "   Frontend: Embedded Next.js application"
echo "   Backend: Embedded Go service"
echo "   Config: config/nexus-testnet.yaml"
echo "   Mode: Testnet with simplified security"

cd $BASE_DIR && (
    # Start with proper testnet configuration
    exec ./bin/knirvserver \
        --testnet \
        --port 8084 \
        --host 0.0.0.0 \
        --config config/nexus-testnet.yaml
) > ./logs/knirvserver.log 2>&1 &

NEXUS_PID=$!
echo $NEXUS_PID > data/knirvserver.pid

echo "📋 Process started with PID: $NEXUS_PID"

# Wait for initial startup - extended wait for full initialization
echo "⏳ Waiting for KNIRV-SERVER to initialize (this may take 15-30 seconds)..."
sleep 15

# Check if the process is still running
if ! kill -0 $(cat ./data/knirvserver.pid) 2>/dev/null; then
    echo "❌ Error: KNIRV-SERVER failed to start. Check logs:"
    echo ""
    echo "=== Last 30 lines of log ==="
    tail -30 ./logs/knirvserver.log
    echo "=========================="
    echo ""
    echo "💡 Troubleshooting tips:"
    echo "   1. Check if port 8084 is already in use: lsof -i :8084"
    echo "   2. Verify binary was built correctly: ls -la bin/knirvserver"
    echo "   3. Check configuration: cat config/nexus-testnet.yaml"
    exit 1
fi

# Test if the service is responding
echo "🔍 Testing service endpoints..."
sleep 2

# Test health endpoint
if curl -s --max-time 5 "http://localhost:8084/health" >/dev/null 2>&1; then
    echo "✅ Health endpoint responding"
else
    echo "⚠️  Health endpoint not yet responding (may still be initializing)"
fi

# Test frontend
if curl -s --max-time 5 "http://localhost:8084/" >/dev/null 2>&1; then
    echo "✅ Frontend responding"
else
    echo "⚠️  Frontend not yet responding (may still be initializing)"
fi

echo ""
echo "🎉 KNIRV-SERVER unified service is running successfully!"
echo "=================================="
echo ""
echo "🌐 Access Points:"
echo "   Frontend (Next.js):     http://localhost:8084"
echo "   API (Go Backend):       http://localhost:8084/api"
echo "   Health Check:           http://localhost:8084/health"
echo "   Version Info:           http://localhost:8084/version"
echo ""
echo "📋 Management:"
echo "   Logs:                   ./logs/knirvserver.log"
echo "   PID file:               ./data/knirvserver.pid"
echo "   Configuration:          ./config/nexus-testnet.yaml"
echo "   Data directory:         ./data/knirvserver"
echo ""
echo "🧪 Testnet Features Enabled:"
echo "   ✅ Unified binary with embedded frontend and backend"
echo "   ✅ TEE simulation enabled"
echo "   ✅ Mock validation responses"
echo "   ✅ Simplified validation proofs"
echo "   ✅ Testnet security mode"
echo "   ✅ In-memory optimizations"
echo ""
echo "🔧 Architecture:"
echo "   • Go wrapper serves embedded Next.js frontend"
echo "   • Embedded Go backend handles API requests"
echo "   • Single binary deployment"
echo "   • Socket.IO support for real-time features"
