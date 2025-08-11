#!/bin/bash
set -e

echo "=========================================="
echo "KNIRV-TESTNET: Health Check"
echo "=========================================="

# Already in KNIRVTESTNET directory

# Function to check service health
check_service() {
    local url=$1
    local name=$2
    
    if curl -s -f "$url" > /dev/null 2>&1; then
        echo "✅ $name: HEALTHY"
        return 0
    else
        echo "❌ $name: UNHEALTHY"
        return 1
    fi
}

# Check all services
healthy=0
total=7  # Core KNIRV services including GATEWAY

echo "Checking service health..."
echo ""

check_service "http://localhost:1317/health" "KNIRV-ROOT" && ((healthy++)) || true
check_service "http://localhost:8090/health" "KNIRVCHAIN" && ((healthy++)) || true
check_service "http://localhost:8082/height" "KNIRVGRAPH" && ((healthy++)) || true
check_service "http://localhost:8084/health" "KNIRV-NEXUS-DVE" && ((healthy++)) || true
check_service "http://localhost:8085/health" "KNIRV-NEXUS-VAL" && ((healthy++)) || true
check_service "http://localhost:8086/status" "KNIRV-ROUTER" && ((healthy++)) || true
check_service "http://localhost:8087/" "KNIRV-GATEWAY" && ((healthy++)) || true
# Optional services for full deployment:
# check_service "http://localhost:5001/api/v0/version" "IPFS" && ((healthy++)) || true

echo ""
echo "=========================================="
echo "📊 Health Summary: $healthy/$total services healthy"
echo "=========================================="

if [ $healthy -eq $total ]; then
    echo "🎉 All services are healthy!"
    echo ""
    echo "🌐 Service Endpoints:"
    echo "  KNIRV-ROOT:     http://localhost:1317/health"
    echo "  KNIRVCHAIN:     http://localhost:8090/health"
    echo "  KNIRVGRAPH:     http://localhost:8082/height"
    echo "  KNIRV-NEXUS-DVE: http://localhost:8084/health"
    echo "  KNIRV-NEXUS-VAL: http://localhost:8085/health"
    echo "  KNIRV-ROUTER:   http://localhost:8086/status"
    echo "  KNIRV-GATEWAY:  http://localhost:8087/"
    exit 0
else
    echo "⚠️  Some services are unhealthy."
    echo "🔍 Check logs in ./logs/ for details"
    echo "🔧 Try restarting with ./scripts/start-testnet.sh"
    exit 1
fi
