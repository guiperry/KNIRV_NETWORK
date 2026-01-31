#!/bin/bash
# Run ASIC monitor tool and capture output

set -e

# Load environment variables
if [ -f .env ]; then
  source .env
fi

ANTMINER_IP="${DEVICE_IP:-192.168.12.151}"
ANTMINER_USER="root"
ANTMINER_PASSWORD="${DEVICE_PASSWORD:-keperu100}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

SSH_OPTS="-o KexAlgorithms=+diffie-hellman-group14-sha1 -o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no"

echo "🛡️  Hasher Monitor Tool Runner"
echo "======================================"
echo ""

# Clean up old binary on Antminer
echo "🧹 Cleaning up old binary..."
sshpass -p "$ANTMINER_PASSWORD" ssh $SSH_OPTS ${ANTMINER_USER}@${ANTMINER_IP} 'killall -9 asic-monitor 2>/dev/null; rm -f /tmp/monitor; echo "Ready"' || true
echo ""

# Set device permissions
echo "🔧 Setting device permissions..."
sshpass -p "$ANTMINER_PASSWORD" ssh $SSH_OPTS ${ANTMINER_USER}@${ANTMINER_IP} 'chmod 666 /dev/bitmain-asic 2>/dev/null && echo "Permissions set" || echo "Device not found (will be created on first access)"'
echo ""

# Build and deploy
echo "🔨 Building ASIC monitor tool..."
cd "$PROJECT_ROOT"
make build-monitor

echo ""
echo "🚀 Deploying to Antminer..."
make deploy-monitor

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Running ASIC monitor on Antminer..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

mkdir -p "$PROJECT_ROOT/logs"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
OUTPUT_FILE="$PROJECT_ROOT/logs/monitor_${TIMESTAMP}.log"

# Run monitor in dump-status mode and capture output
sshpass -p "$ANTMINER_PASSWORD" ssh $SSH_OPTS ${ANTMINER_USER}@${ANTMINER_IP} '/tmp/monitor --dump-status --dump-interval 2' 2>&1 | tee "$OUTPUT_FILE"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Monitor run complete!"
echo ""
echo "📄 Output saved to: $OUTPUT_FILE"
echo ""
echo "🔍 Analyzing results..."
echo ""

# Quick analysis
if grep -q "Device opened successfully" "$OUTPUT_FILE"; then
    echo "✅ Device access: SUCCESS"

    if grep -q "Wrote.*bytes successfully" "$OUTPUT_FILE"; then
        echo "✅ TxConfig packet: Sent successfully"
    fi

    if grep -q "TxTask packet.*Wrote.*bytes" "$OUTPUT_FILE"; then
        echo "✅ TxTask packet: Sent successfully"
    fi

    # Look for parsed status or JSON entries from dump mode
    if grep -q "Parsed RxStatus" "$OUTPUT_FILE" || grep -q '"crc_valid":true' "$OUTPUT_FILE"; then
        echo "✅ RxStatus parsed and logged"
        echo "📄 Dump logs created in logs/ (see asic-monitor-status_*.log on device and host)"
    else
        echo ""
        echo "📊 Check kernel logs for ASIC responses:"
        echo "   ssh antminer 'dmesg | tail -30'"
    fi

elif grep -q "device or resource busy" "$OUTPUT_FILE"; then
    echo "❌ Device access: BUSY"
    echo ""
    echo "💡 The device is still locked. Try:"
    echo "   1. ssh antminer 'lsmod | grep bitmain'"
    echo "   2. ssh antminer 'rmmod bitmain_asic'"
    echo "   3. Reboot the Antminer"

else
    echo "❌ Device access: FAILED"

    if grep -q "CGMiner is not running" "$OUTPUT_FILE"; then
        echo "ℹ️  CGMiner was already stopped"
    fi

    if grep -q "Could not reload kernel module" "$OUTPUT_FILE"; then
        echo "⚠️  Unable to reload kernel module (may need root)"
    fi
fi

echo ""
echo "View full output: cat $OUTPUT_FILE"
