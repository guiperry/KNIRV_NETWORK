#!/bin/bash

# start-knirvverifier.sh - Start KNIRV Formal Verification Service
# Integrates modp verification capabilities with running testnet

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_ROOT="$(dirname "$SCRIPT_DIR")"
MODP_DIR="$(dirname "$TESTNET_ROOT")/modp"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if modp directory exists
if [ ! -d "$MODP_DIR" ]; then
    print_error "modp directory not found at $MODP_DIR"
    exit 1
fi

cd "$MODP_DIR"

# Create data directory for verifier
mkdir -p data logs

# Check if verification server exists
VERIFIER_SERVER="api/verification-server.js"
if [ ! -f "$VERIFIER_SERVER" ]; then
    print_warning "Verification server not found - creating minimal bridge..."
    mkdir -p api
    
    # Create minimal verification server
    cat > api/verification-server.js << 'EOF'
const express = require('express');
const cors = require('cors');
const path = require('path');
const { spawn } = require('child_process');

const app = express();
const PORT = 9000;

app.use(cors());
app.use(express.json());

// Health check endpoint
app.get('/verification/status', (req, res) => {
  res.json({
    status: 'running',
    mode: 'bridge',
    timestamp: new Date().toISOString(),
    capabilities: [
      'model_compilation',
      'invariant_monitoring',
      'compositional_testing'
    ]
  });
});

// Trigger verification tests
app.post('/verification/run-tests', async (req, res) => {
  const { testSuite = 'all', schedules = 100 } = req.body;
  
  console.log(`Running verification tests: ${testSuite}`);
  
  try {
    const result = await runVerificationTests(testSuite, schedules);
    res.json(result);
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

// Get latest test results
app.get('/verification/results', (req, res) => {
  try {
    const results = getLatestResults();
    res.json(results);
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

// Verification test runner
function runVerificationTests(testSuite, schedules) {
  return new Promise((resolve, reject) => {
    const testScript = path.join(__dirname, '..', 'scripts', 'run-tests.sh');
    
    const child = spawn('bash', [testScript, testSuite], {
      cwd: path.join(__dirname, '..'),
      stdio: 'pipe'
    });
    
    let stdout = '';
    let stderr = '';
    
    child.stdout.on('data', (data) => {
      stdout += data.toString();
    });
    
    child.stderr.on('data', (data) => {
      stderr += data.toString();
    });
    
    child.on('close', (code) => {
      resolve({
        success: code === 0,
        exitCode: code,
        stdout,
        stderr,
        timestamp: new Date().toISOString()
      });
    });
    
    child.on('error', (error) => {
      reject(error);
    });
  });
}

// Get latest test results
function getLatestResults() {
  const fs = require('fs');
  const resultsDir = path.join(__dirname, '..', 'results');
  
  if (!fs.existsSync(resultsDir)) {
    return { message: 'No results directory found' };
  }
  
  const files = fs.readdirSync(resultsDir)
    .filter(file => file.startsWith('test-') && file.endsWith('.log'))
    .sort()
    .reverse();
  
  if (files.length === 0) {
    return { message: 'No test results found' };
  }
  
  const latestFile = files[0];
  const filePath = path.join(resultsDir, latestFile);
  
  try {
    const content = fs.readFileSync(filePath, 'utf8');
    const lines = content.split('\n');
    
    const passed = lines.filter(line => line.includes('passed')).length;
    const failed = lines.filter(line => line.includes('failed')).length;
    
    return {
      filename: latestFile,
      timestamp: latestFile.match(/test-(\d+-\d+)/)?.[1] || 'unknown',
      summary: {
        passed,
        failed,
        total: passed + failed
      },
      lastLines: lines.slice(-10)
    };
  } catch (error) {
    throw new Error(`Failed to read results: ${error.message}`);
  }
}

// Event bridge endpoint - receive testnet events
app.post('/verification/event', (req, res) => {
  const { eventType, component, data } = req.body;
  
  console.log(`Received event: ${eventType} from ${component}`);
  
  // TODO: Map real events to P model events
  // For now, just acknowledge receipt
  res.json({
    received: true,
    timestamp: new Date().toISOString()
  });
});

app.listen(PORT, () => {
  console.log(`KNIRV Verification Service running on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/verification/status`);
});

// Save PID
const fs = require('fs');
fs.writeFileSync(path.join(__dirname, '..', 'data', 'knirvverifier.pid'), process.pid.toString());
EOF

    print_success "Created verification server"
fi

# Check Node.js dependencies
print_status "Checking Node.js dependencies..."
if [ ! -d "node_modules" ]; then
    print_status "Installing Node.js dependencies for verification service..."
    npm init -y >/dev/null 2>&1
    npm install express cors >/dev/null 2>&1 || {
        print_warning "Failed to install dependencies, using system Node.js"
    }
fi

# Start verification service
print_status "Starting KNIRV Formal Verification Service..."

# Check if already running
PID_FILE="data/knirvverifier.pid"
if [ -f "$PID_FILE" ]; then
    EXISTING_PID=$(cat "$PID_FILE" 2>/dev/null)
    if [ -n "$EXISTING_PID" ] && kill -0 "$EXISTING_PID" 2>/dev/null; then
        print_warning "Verification service already running (PID: $EXISTING_PID)"
        exit 0
    else
        rm -f "$PID_FILE"
    fi
fi

# Start in background
cd "$MODP_DIR"
nohup node api/verification-server.js > logs/verification-server.log 2>&1 &
VERIFIER_PID=$!

echo $VERIFIER_PID > "$PID_FILE"

# Wait a moment and check if it started
sleep 2

if kill -0 "$VERIFIER_PID" 2>/dev/null; then
    print_success "KNIRV Verification Service started (PID: $VERIFIER_PID)"
    print_status "Service endpoints:"
    print_status "  Health: http://localhost:9000/verification/status"
    print_status "  Tests: http://localhost:9000/verification/run-tests"
    print_status "  Results: http://localhost:9000/verification/results"
    
    # Quick health check
    if curl -s http://localhost:9000/verification/status >/dev/null 2>&1; then
        print_success "Verification service is responding"
    else
        print_warning "Verification service started but not yet responding"
    fi
else
    print_error "Failed to start verification service"
    rm -f "$PID_FILE"
    exit 1
fi