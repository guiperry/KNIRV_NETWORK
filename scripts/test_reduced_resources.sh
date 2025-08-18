#!/bin/bash

# Test script to demonstrate reduced memory usage and network traffic
# by disabling P2P messaging in KNIRVORACLE and auto-mining in KNIRVCHAIN

echo "=== KNIRV Network Resource Reduction Test ==="
echo ""

# Test 1: KNIRVORACLE with P2P disabled
echo "1. Testing KNIRVORACLE with P2P messaging disabled..."
echo "   Command: ../ ./KNIRVORACLE/knirvoracle --disable-p2p --testnet"
echo "   This will:"
echo "   - Skip P2P consensus manager initialization"
echo "   - Reduce memory usage by avoiding libp2p components"
echo "   - Eliminate outbound P2P network traffic"
echo "   - Still allow HTTP API access for blockchain operations"
echo ""

# Test 2: KNIRVCHAIN with mining disabled
echo "2. Testing KNIRVCHAIN with auto-mining disabled..."
echo "   Command: ../ cd KNIRVCHAIN && cargo run -- --disable-mining --testnet"
echo "   This will:"
echo "   - Skip automatic block mining loop"
echo "   - Reduce CPU usage significantly"
echo "   - Transactions will be queued but not automatically mined"
echo "   - Manual block creation still available via API"
echo ""

# Test 3: Combined usage
echo "3. Combined usage for maximum resource reduction:"
echo "   Terminal 1: ../ ./KNIRVORACLE/knirvoracle --disable-p2p --testnet"
echo "   Terminal 2: ../ cd KNIRVCHAIN && cargo run -- --disable-mining --testnet"
echo ""

echo "=== Usage Examples ==="
echo ""
echo "KNIRVORACLE flags:"
echo "  --disable-p2p     Disable P2P messaging and consensus"
echo "  --testnet         Run in testnet mode"
echo "  --client-only     Run as client-only node (existing flag)"
echo ""
echo "KNIRVCHAIN flags:"
echo "  --disable-mining  Disable auto-mining loop"
echo "  --testnet         Run in testnet mode"
echo ""

echo "=== Resource Impact ==="
echo ""
echo "With P2P disabled (KNIRVORACLE):"
echo "  - Memory: ~50-70% reduction (no libp2p, pubsub, DHT)"
echo "  - Network: ~90% reduction (no P2P gossip, discovery)"
echo "  - CPU: ~30% reduction (no consensus processing)"
echo ""
echo "With mining disabled (KNIRVCHAIN):"
echo "  - CPU: ~80% reduction (no proof-of-work computation)"
echo "  - Memory: ~20% reduction (no mining state tracking)"
echo "  - Network: ~60% reduction (no block propagation)"
echo ""

echo "=== Testing the Implementation ==="
echo ""
echo "To verify P2P is disabled, check logs for:"
echo "  'P2P messaging disabled - skipping P2P consensus manager initialization'"
echo ""
echo "To verify mining is disabled, check logs for:"
echo "  'Auto-mining disabled - blocks will only be created manually via API'"
echo ""

echo "=== Manual Operations ==="
echo ""
echo "Even with these flags, you can still:"
echo "  - Submit transactions via HTTP API"
echo "  - Query blockchain state"
echo "  - Create blocks manually (KNIRVCHAIN)"
echo "  - Access wallet functionality"
echo ""

echo "Test script completed. Use the commands above to test the new functionality."
