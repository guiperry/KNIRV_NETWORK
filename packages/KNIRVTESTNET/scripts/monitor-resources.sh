#!/bin/bash

echo "=========================================="
echo "KNIRV-TESTNET: Resource Monitor"
echo "=========================================="

# Already in KNIRVTESTNET directory

echo "📊 Starting resource monitoring..."
echo "Press Ctrl+C to stop monitoring"
echo ""

while true; do
    echo "$(date): CPU: $(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1)% | RAM: $(free | grep Mem | awk '{printf "%.1f%%", $3/$2 * 100.0}') | Disk: $(df -h . | awk 'NR==2{print $5}')"
    
    # Check service memory usage
    echo "  Service Memory Usage:"
    for service in knirvoracle knirvchain knirvgraph knirvnexus knirvrouter; do
        if pgrep $service > /dev/null; then
            MEM=$(ps -o pid,vsz,comm -C $service 2>/dev/null | tail -n +2 | awk '{sum+=$2} END {print sum/1024 "MB"}' || echo "0MB")
            echo "    $service: $MEM"
        else
            echo "    $service: Not running"
        fi
    done
    
    # Check disk usage for data directories
    echo "  Data Directory Usage:"
    for dir in data/knirvoracle data/knirvchain data/knirvgraph data/knirvnexus data/ipfs; do
        if [ -d "$dir" ]; then
            SIZE=$(du -sh "$dir" 2>/dev/null | cut -f1 || echo "0B")
            echo "    $dir: $SIZE"
        fi
    done
    
    echo ""
    sleep 30
done
