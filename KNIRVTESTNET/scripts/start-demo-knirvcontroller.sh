#!/bin/bash

# KNIRV TESTNET - Demo KNIRVCONTROLLER Service
# Provides a lightweight demo KNIRVCONTROLLER for testnet development

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_ROOT="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$TESTNET_ROOT")"

# Configuration
DEMO_CONTROLLER_PORT=8089
DEMO_CONTROLLER_PID_FILE="$TESTNET_ROOT/data/demo-knirvcontroller.pid"
DEMO_CONTROLLER_LOG_FILE="$TESTNET_ROOT/logs/demo-knirvcontroller.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
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

# Check if real KNIRVCONTROLLER is already running
check_real_controller() {
    local real_controller_ports=("3000" "5173" "8088")
    
    for port in "${real_controller_ports[@]}"; do
        if curl -s --max-time 2 "http://localhost:$port/health" > /dev/null 2>&1; then
            print_info "Real KNIRVCONTROLLER detected on port $port"
            return 0
        fi
    done
    
    return 1
}

# Create demo KNIRVCONTROLLER service
create_demo_service() {
    print_info "Creating demo KNIRVCONTROLLER service..."
    
    # Create demo service directory
    mkdir -p "$TESTNET_ROOT/data/demo-knirvcontroller"
    
    # Create simple Node.js demo service
    cat > "$TESTNET_ROOT/data/demo-knirvcontroller/demo-controller.js" << 'EOF'
const http = require('http');
const url = require('url');
const fs = require('fs');
const path = require('path');

const PORT = process.env.PORT || 8089;

// Demo data
const demoAgents = [
    { id: 'agent-001', name: 'Demo Agent 1', status: 'active', skills: ['data-analysis', 'reporting'] },
    { id: 'agent-002', name: 'Demo Agent 2', status: 'idle', skills: ['image-processing', 'classification'] },
    { id: 'agent-003', name: 'Demo Agent 3', status: 'training', skills: ['nlp', 'sentiment-analysis'] }
];

const demoSkills = [
    { id: 'skill-001', name: 'Data Analysis', category: 'analytics', status: 'available' },
    { id: 'skill-002', name: 'Image Processing', category: 'vision', status: 'available' },
    { id: 'skill-003', name: 'NLP Processing', category: 'language', status: 'training' }
];

const demoMetrics = {
    totalAgents: 3,
    activeAgents: 1,
    totalSkills: 3,
    availableSkills: 2,
    totalInvocations: 127,
    successRate: 94.5,
    avgResponseTime: 245
};

// Simple HTML interface
const htmlInterface = `
<!DOCTYPE html>
<html>
<head>
    <title>Demo KNIRVCONTROLLER</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; }
        .metric { background: #f8f9fa; padding: 15px; border-radius: 5px; text-align: center; }
        .metric-value { font-size: 24px; font-weight: bold; color: #667eea; }
        .metric-label { color: #666; margin-top: 5px; }
        .list-item { padding: 10px; border-bottom: 1px solid #eee; display: flex; justify-content: space-between; }
        .status-active { color: #28a745; }
        .status-idle { color: #ffc107; }
        .status-training { color: #17a2b8; }
        .demo-badge { background: #ff6b6b; color: white; padding: 4px 8px; border-radius: 4px; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎮 Demo KNIRVCONTROLLER</h1>
            <p>Lightweight demo service for KNIRV testnet development</p>
            <span class="demo-badge">DEMO MODE</span>
        </div>
        
        <div class="section">
            <h2>📊 System Metrics</h2>
            <div class="metrics">
                <div class="metric">
                    <div class="metric-value">${demoMetrics.totalAgents}</div>
                    <div class="metric-label">Total Agents</div>
                </div>
                <div class="metric">
                    <div class="metric-value">${demoMetrics.activeAgents}</div>
                    <div class="metric-label">Active Agents</div>
                </div>
                <div class="metric">
                    <div class="metric-value">${demoMetrics.totalSkills}</div>
                    <div class="metric-label">Total Skills</div>
                </div>
                <div class="metric">
                    <div class="metric-value">${demoMetrics.successRate}%</div>
                    <div class="metric-label">Success Rate</div>
                </div>
            </div>
        </div>
        
        <div class="section">
            <h2>🤖 Demo Agents</h2>
            ${demoAgents.map(agent => `
                <div class="list-item">
                    <div>
                        <strong>${agent.name}</strong> (${agent.id})<br>
                        <small>Skills: ${agent.skills.join(', ')}</small>
                    </div>
                    <div class="status-${agent.status}">${agent.status.toUpperCase()}</div>
                </div>
            `).join('')}
        </div>
        
        <div class="section">
            <h2>🛠️ Demo Skills</h2>
            ${demoSkills.map(skill => `
                <div class="list-item">
                    <div>
                        <strong>${skill.name}</strong> (${skill.id})<br>
                        <small>Category: ${skill.category}</small>
                    </div>
                    <div class="status-${skill.status}">${skill.status.toUpperCase()}</div>
                </div>
            `).join('')}
        </div>
        
        <div class="section">
            <h2>🔗 API Endpoints</h2>
            <ul>
                <li><code>GET /health</code> - Health check</li>
                <li><code>GET /api/agents</code> - List agents</li>
                <li><code>GET /api/skills</code> - List skills</li>
                <li><code>GET /api/metrics</code> - System metrics</li>
                <li><code>POST /api/agents/{id}/invoke</code> - Invoke agent skill</li>
            </ul>
        </div>
    </div>
</body>
</html>
`;

const server = http.createServer((req, res) => {
    const parsedUrl = url.parse(req.url, true);
    const pathname = parsedUrl.pathname;
    
    // CORS headers
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
    
    if (req.method === 'OPTIONS') {
        res.writeHead(200);
        res.end();
        return;
    }
    
    // Routes
    if (pathname === '/' || pathname === '/dashboard') {
        res.writeHead(200, { 'Content-Type': 'text/html' });
        res.end(htmlInterface);
    } else if (pathname === '/health') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ 
            status: 'healthy', 
            service: 'demo-knirvcontroller',
            mode: 'demo',
            timestamp: new Date().toISOString(),
            port: PORT
        }));
    } else if (pathname === '/api/agents') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ agents: demoAgents }));
    } else if (pathname === '/api/skills') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ skills: demoSkills }));
    } else if (pathname === '/api/metrics') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify(demoMetrics));
    } else if (pathname.startsWith('/api/agents/') && pathname.endsWith('/invoke')) {
        const agentId = pathname.split('/')[3];
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ 
            result: 'success', 
            agentId: agentId,
            message: 'Demo skill invocation completed',
            timestamp: new Date().toISOString()
        }));
    } else {
        res.writeHead(404, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'Not found', service: 'demo-knirvcontroller' }));
    }
});

server.listen(PORT, () => {
    console.log(`Demo KNIRVCONTROLLER running on port ${PORT}`);
    console.log(`Dashboard: http://localhost:${PORT}/`);
    console.log(`Health: http://localhost:${PORT}/health`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('Demo KNIRVCONTROLLER shutting down...');
    server.close(() => {
        process.exit(0);
    });
});
EOF

    # Create package.json for demo service
    cat > "$TESTNET_ROOT/data/demo-knirvcontroller/package.json" << 'EOF'
{
  "name": "demo-knirvcontroller",
  "version": "1.0.0",
  "description": "Demo KNIRVCONTROLLER service for testnet",
  "main": "demo-controller.js",
  "scripts": {
    "start": "node demo-controller.js"
  },
  "dependencies": {}
}
EOF

    print_success "Demo KNIRVCONTROLLER service created"
}

# Start demo KNIRVCONTROLLER
start_demo_controller() {
    print_info "🎮 Starting Demo KNIRVCONTROLLER service..."
    
    # Check if real controller is running
    if check_real_controller; then
        print_warning "Real KNIRVCONTROLLER detected - skipping demo service"
        print_info "To use demo service, stop the real KNIRVCONTROLLER first"
        return 0
    fi
    
    # Check if demo is already running
    if [ -f "$DEMO_CONTROLLER_PID_FILE" ]; then
        local pid=$(cat "$DEMO_CONTROLLER_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            print_warning "Demo KNIRVCONTROLLER already running (PID: $pid)"
            return 0
        else
            rm -f "$DEMO_CONTROLLER_PID_FILE"
        fi
    fi
    
    # Create demo service if it doesn't exist
    if [ ! -f "$TESTNET_ROOT/data/demo-knirvcontroller/demo-controller.js" ]; then
        create_demo_service
    fi
    
    # Start demo service
    cd "$TESTNET_ROOT/data/demo-knirvcontroller"
    
    PORT="$DEMO_CONTROLLER_PORT" nohup node demo-controller.js > "$DEMO_CONTROLLER_LOG_FILE" 2>&1 &
    local pid=$!
    echo "$pid" > "$DEMO_CONTROLLER_PID_FILE"
    
    # Wait for service to start
    sleep 2
    
    # Verify service is running
    if curl -s --max-time 5 "http://localhost:$DEMO_CONTROLLER_PORT/health" > /dev/null; then
        print_success "Demo KNIRVCONTROLLER started successfully!"
        print_info "Dashboard: http://localhost:$DEMO_CONTROLLER_PORT/"
        print_info "Health: http://localhost:$DEMO_CONTROLLER_PORT/health"
        print_info "PID: $pid"
        print_info "Log: $DEMO_CONTROLLER_LOG_FILE"
    else
        print_error "Failed to start Demo KNIRVCONTROLLER"
        return 1
    fi
}

# Main execution
case "${1:-start}" in
    "start")
        start_demo_controller
        ;;
    "stop")
        if [ -f "$DEMO_CONTROLLER_PID_FILE" ]; then
            pid=$(cat "$DEMO_CONTROLLER_PID_FILE")
            if kill -0 "$pid" 2>/dev/null; then
                kill "$pid"
                rm -f "$DEMO_CONTROLLER_PID_FILE"
                print_success "Demo KNIRVCONTROLLER stopped"
            else
                print_warning "Demo KNIRVCONTROLLER not running"
                rm -f "$DEMO_CONTROLLER_PID_FILE"
            fi
        else
            print_warning "Demo KNIRVCONTROLLER not running"
        fi
        ;;
    "status")
        if [ -f "$DEMO_CONTROLLER_PID_FILE" ]; then
            pid=$(cat "$DEMO_CONTROLLER_PID_FILE")
            if kill -0 "$pid" 2>/dev/null; then
                print_success "Demo KNIRVCONTROLLER running (PID: $pid)"
                curl -s "http://localhost:$DEMO_CONTROLLER_PORT/health" | jq . 2>/dev/null || echo "Health check failed"
            else
                print_warning "Demo KNIRVCONTROLLER not running (stale PID file)"
                rm -f "$DEMO_CONTROLLER_PID_FILE"
            fi
        else
            print_warning "Demo KNIRVCONTROLLER not running"
        fi
        ;;
    *)
        echo "Usage: $0 {start|stop|status}"
        exit 1
        ;;
esac
