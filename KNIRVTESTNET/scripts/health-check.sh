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
total=8

echo "Checking service health..."
echo ""

check_service "http://localhost:1317/health" "KNIRV-ROOT" && ((healthy++)) || true
check_service "http://localhost:8080/health" "KNIRVCHAIN" && ((healthy++)) || true
check_service "http://localhost:8081/health" "KNIRVGRAPH" && ((healthy++)) || true
check_service "http://localhost:8082/status" "KNIRV-NEXUS-1" && ((healthy++)) || true
check_service "http://localhost:8083/status" "KNIRV-NEXUS-2" && ((healthy++)) || true
check_service "http://localhost:8086/status" "KNIRV-ROUTER" && ((healthy++)) || true
check_service "http://localhost:8087/health" "KNIRV-GATEWAY" && ((healthy++)) || true
check_service "http://localhost:5001/api/v0/version" "IPFS" && ((healthy++)) || true

echo ""
echo "=========================================="
echo "📊 Health Summary: $healthy/$total services healthy"
echo "=========================================="

if [ $healthy -eq $total ]; then
    echo "🎉 All services are healthy!"
    echo ""
    echo "🌐 Service Endpoints:"
    echo "  KNIRV-ROOT:     http://localhost:1317"
    echo "  KNIRVCHAIN:     http://localhost:8080"
    echo "  KNIRVGRAPH:     http://localhost:8081"
    echo "  KNIRV-NEXUS-1:  http://localhost:8082"
    echo "  KNIRV-NEXUS-2:  http://localhost:8083"
    echo "  KNIRV-ROUTER:   http://localhost:8086"
    echo "  KNIRV-GATEWAY:  http://localhost:8087"
    echo "  IPFS API:       http://localhost:5001"
    exit 0
else
    echo "⚠️  Some services are unhealthy."
    echo "🔍 Check logs in ./logs/ for details"
    echo "🔧 Try restarting with ./scripts/start-testnet.sh"
    exit 1
fi
