#!/bin/bash

# KNIRVTESTNET Frontend Update Script
# Updates KNIRVGATEWAY/knirvtestnet with latest changes and configures Netlify integration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Configuration
TESTNET_DIR="$PROJECT_ROOT/KNIRVTESTNET"
GATEWAY_DIR="$PROJECT_ROOT/KNIRVGATEWAY"
FRONTEND_DIR="$GATEWAY_DIR/knirvtestnet"
ANSIBLE_DIR="$PROJECT_ROOT/deployment/ansible"
TESTNET_IP_FILE="$ANSIBLE_DIR/testnet_ip.txt"

# Functions
print_header() {
    echo -e "${BLUE}=================================${NC}"
    echo -e "${BLUE}🌐 KNIRVTESTNET Frontend Update${NC}"
    echo -e "${BLUE}=================================${NC}"
    echo ""
}

print_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
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

check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check if testnet IP file exists
    if [ ! -f "$TESTNET_IP_FILE" ]; then
        print_error "Testnet IP file not found at $TESTNET_IP_FILE"
        print_warning "Please run 'make deploy-testnet-infrastructure' first"
        exit 1
    fi
    
    # Get testnet IP
    TESTNET_IP=$(cat "$TESTNET_IP_FILE")
    if [ -z "$TESTNET_IP" ]; then
        print_error "Testnet IP is empty"
        exit 1
    fi
    
    # Check if KNIRVGATEWAY directory exists
    if [ ! -d "$GATEWAY_DIR" ]; then
        print_error "KNIRVGATEWAY directory not found at $GATEWAY_DIR"
        exit 1
    fi
    
    # Check if KNIRVTESTNET directory exists
    if [ ! -d "$TESTNET_DIR" ]; then
        print_error "KNIRVTESTNET directory not found at $TESTNET_DIR"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
    print_success "Testnet IP: $TESTNET_IP"
}

sync_testnet_frontend() {
    print_step "Syncing testnet frontend files..."
    
    # Create frontend directory if it doesn't exist
    mkdir -p "$FRONTEND_DIR"
    
    # Copy static files from KNIRVTESTNET (excluding Docker and build files)
    rsync -av \
        --exclude='*.log' \
        --exclude='docker-compose.yml' \
        --exclude='Dockerfile' \
        --exclude='bin/' \
        --exclude='data/' \
        --exclude='logs/' \
        --exclude='backups/' \
        --exclude='kubo_v0.24.0_linux-amd64.tar.gz' \
        --exclude='kubo/' \
        --exclude='scripts/build-*.sh' \
        --exclude='scripts/start-*.sh' \
        --exclude='scripts/stop-*.sh' \
        --exclude='scripts/mock-*.py' \
        "$TESTNET_DIR/" "$FRONTEND_DIR/"
    
    print_success "Testnet frontend files synced"
}

create_testnet_index() {
    print_step "Creating testnet frontend index..."
    
    cat > "$FRONTEND_DIR/index.html" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>KNIRV Testnet - Blockchain Development Network</title>
    <link rel="icon" href="../images/favicon.png">
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
        }
        
        .header {
            background: rgba(0, 0, 0, 0.2);
            padding: 1rem 2rem;
            backdrop-filter: blur(10px);
        }
        
        .header h1 {
            font-size: 2.5rem;
            margin-bottom: 0.5rem;
        }
        
        .header p {
            opacity: 0.9;
            font-size: 1.1rem;
        }
        
        .container {
            flex: 1;
            padding: 2rem;
            max-width: 1200px;
            margin: 0 auto;
            width: 100%;
        }
        
        .status-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2rem;
        }
        
        .status-card {
            background: rgba(255, 255, 255, 0.1);
            border-radius: 12px;
            padding: 1.5rem;
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255, 255, 255, 0.2);
        }
        
        .status-card h3 {
            margin-bottom: 1rem;
            color: #fff;
        }
        
        .service-list {
            list-style: none;
        }
        
        .service-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 0.5rem 0;
            border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        }
        
        .service-item:last-child {
            border-bottom: none;
        }
        
        .service-name {
            font-weight: 500;
        }
        
        .service-status {
            padding: 0.25rem 0.75rem;
            border-radius: 20px;
            font-size: 0.8rem;
            font-weight: 600;
        }
        
        .status-healthy {
            background: #10b981;
            color: white;
        }
        
        .status-checking {
            background: #f59e0b;
            color: white;
        }
        
        .status-unhealthy {
            background: #ef4444;
            color: white;
        }
        
        .info-section {
            background: rgba(255, 255, 255, 0.1);
            border-radius: 12px;
            padding: 2rem;
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255, 255, 255, 0.2);
            margin-bottom: 2rem;
        }
        
        .endpoint-list {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 1rem;
            margin-top: 1rem;
        }
        
        .endpoint-item {
            background: rgba(0, 0, 0, 0.2);
            padding: 1rem;
            border-radius: 8px;
            text-align: center;
        }
        
        .endpoint-item a {
            color: #60a5fa;
            text-decoration: none;
            font-weight: 500;
        }
        
        .endpoint-item a:hover {
            color: #93c5fd;
            text-decoration: underline;
        }
        
        .footer {
            background: rgba(0, 0, 0, 0.2);
            padding: 1rem 2rem;
            text-align: center;
            backdrop-filter: blur(10px);
        }
        
        @media (max-width: 768px) {
            .header h1 {
                font-size: 2rem;
            }
            
            .container {
                padding: 1rem;
            }
            
            .status-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>🧪 KNIRV Testnet</h1>
        <p>Blockchain Development Network - Live Testing Environment</p>
    </div>
    
    <div class="container">
        <div class="status-grid">
            <div class="status-card">
                <h3>🔗 Core Services</h3>
                <ul class="service-list">
                    <li class="service-item">
                        <span class="service-name">KNIRVORACLE</span>
                        <span class="service-status status-checking" id="status-root">Checking...</span>
                    </li>
                    <li class="service-item">
                        <span class="service-name">KNIRVCHAIN</span>
                        <span class="service-status status-checking" id="status-chain">Checking...</span>
                    </li>
                    <li class="service-item">
                        <span class="service-name">KNIRVGRAPH</span>
                        <span class="service-status status-checking" id="status-graph">Checking...</span>
                    </li>
                    <li class="service-item">
                        <span class="service-name">KNIRVGATEWAY</span>
                        <span class="service-status status-checking" id="status-gateway">Checking...</span>
                    </li>
                </ul>
            </div>
            
            <div class="status-card">
                <h3>🤖 AI Services</h3>
                <ul class="service-list">
                    <li class="service-item">
                        <span class="service-name">KNIRVNEXUS-1</span>
                        <span class="service-status status-checking" id="status-nexus1">Checking...</span>
                    </li>
                    <li class="service-item">
                        <span class="service-name">KNIRVNEXUS-2</span>
                        <span class="service-status status-checking" id="status-nexus2">Checking...</span>
                    </li>
                    <li class="service-item">
                        <span class="service-name">KNIRVROUTER</span>
                        <span class="service-status status-checking" id="status-router">Checking...</span>
                    </li>
                </ul>
            </div>
            
            <div class="status-card">
                <h3>📊 Infrastructure</h3>
                <ul class="service-list">
                    <li class="service-item">
                        <span class="service-name">IPFS Gateway</span>
                        <span class="service-status status-checking" id="status-ipfs">Checking...</span>
                    </li>
                    <li class="service-item">
                        <span class="service-name">Tendermint RPC</span>
                        <span class="service-status status-checking" id="status-tendermint">Checking...</span>
                    </li>
                </ul>
            </div>
        </div>
        
        <div class="info-section">
            <h3>🌐 Testnet Endpoints</h3>
            <div class="endpoint-list">
                <div class="endpoint-item">
                    <h4>KNIRVORACLE API</h4>
                    <a href="http://$TESTNET_IP:1317" target="_blank">http://$TESTNET_IP:1317</a>
                </div>
                <div class="endpoint-item">
                    <h4>KNIRVCHAIN</h4>
                    <a href="http://$TESTNET_IP:8080" target="_blank">http://$TESTNET_IP:8080</a>
                </div>
                <div class="endpoint-item">
                    <h4>KNIRVGRAPH</h4>
                    <a href="http://$TESTNET_IP:8081" target="_blank">http://$TESTNET_IP:8081</a>
                </div>
                <div class="endpoint-item">
                    <h4>KNIRVNEXUS-1</h4>
                    <a href="http://$TESTNET_IP:8082" target="_blank">http://$TESTNET_IP:8082</a>
                </div>
                <div class="endpoint-item">
                    <h4>KNIRVNEXUS-2</h4>
                    <a href="http://$TESTNET_IP:8083" target="_blank">http://$TESTNET_IP:8083</a>
                </div>
                <div class="endpoint-item">
                    <h4>KNIRVROUTER</h4>
                    <a href="http://$TESTNET_IP:8086" target="_blank">http://$TESTNET_IP:8086</a>
                </div>
                <div class="endpoint-item">
                    <h4>KNIRVGATEWAY</h4>
                    <a href="http://$TESTNET_IP:8087" target="_blank">http://$TESTNET_IP:8087</a>
                </div>
                <div class="endpoint-item">
                    <h4>IPFS Gateway</h4>
                    <a href="http://$TESTNET_IP:8080" target="_blank">http://$TESTNET_IP:8080</a>
                </div>
            </div>
        </div>
        
        <div class="info-section">
            <h3>📚 Documentation & Resources</h3>
            <p>Access comprehensive documentation and testing guides for the KNIRV Testnet.</p>
            <div class="endpoint-list">
                <div class="endpoint-item">
                    <h4>Testing Guide</h4>
                    <a href="./TESTING_GUIDE.md" target="_blank">View Testing Guide</a>
                </div>
                <div class="endpoint-item">
                    <h4>Configuration</h4>
                    <a href="./config/" target="_blank">View Configuration</a>
                </div>
                <div class="endpoint-item">
                    <h4>Main Gateway</h4>
                    <a href="../" target="_blank">Return to KNIRV Gateway</a>
                </div>
            </div>
        </div>
    </div>
    
    <div class="footer">
        <p>&copy; 2024 KNIRV Network - Testnet Environment</p>
    </div>
    
    <script>
        // Service health check configuration
        const services = [
            { id: 'status-root', url: 'http://$TESTNET_IP:1317/health', name: 'KNIRVORACLE' },
            { id: 'status-chain', url: 'http://$TESTNET_IP:8080/health', name: 'KNIRVCHAIN' },
            { id: 'status-graph', url: 'http://$TESTNET_IP:8081/health', name: 'KNIRVGRAPH' },
            { id: 'status-gateway', url: 'http://$TESTNET_IP:8087/health', name: 'KNIRVGATEWAY' },
            { id: 'status-nexus1', url: 'http://$TESTNET_IP:8082/health', name: 'KNIRVNEXUS-1' },
            { id: 'status-nexus2', url: 'http://$TESTNET_IP:8083/health', name: 'KNIRVNEXUS-2' },
            { id: 'status-router', url: 'http://$TESTNET_IP:8086/health', name: 'KNIRVROUTER' },
            { id: 'status-ipfs', url: 'http://$TESTNET_IP:5001/api/v0/version', name: 'IPFS' },
            { id: 'status-tendermint', url: 'http://$TESTNET_IP:26657/health', name: 'Tendermint' }
        ];
        
        // Check service health
        async function checkServiceHealth(service) {
            const element = document.getElementById(service.id);
            try {
                const response = await fetch(service.url, { 
                    method: 'GET',
                    mode: 'cors',
                    timeout: 5000
                });
                
                if (response.ok) {
                    element.textContent = 'Healthy';
                    element.className = 'service-status status-healthy';
                } else {
                    element.textContent = 'Unhealthy';
                    element.className = 'service-status status-unhealthy';
                }
            } catch (error) {
                element.textContent = 'Unhealthy';
                element.className = 'service-status status-unhealthy';
            }
        }
        
        // Check all services
        function checkAllServices() {
            services.forEach(service => {
                checkServiceHealth(service);
            });
        }
        
        // Initial check
        checkAllServices();
        
        // Periodic health checks every 30 seconds
        setInterval(checkAllServices, 30000);
        
        // Add refresh button functionality
        document.addEventListener('DOMContentLoaded', function() {
            const header = document.querySelector('.header');
            const refreshButton = document.createElement('button');
            refreshButton.textContent = '🔄 Refresh Status';
            refreshButton.style.cssText = \`
                background: rgba(255, 255, 255, 0.2);
                border: 1px solid rgba(255, 255, 255, 0.3);
                color: white;
                padding: 0.5rem 1rem;
                border-radius: 6px;
                cursor: pointer;
                margin-top: 1rem;
                font-size: 0.9rem;
            \`;
            refreshButton.onclick = checkAllServices;
            header.appendChild(refreshButton);
        });
    </script>
</body>
</html>
EOF
    
    print_success "Testnet frontend index created"
}

update_netlify_config() {
    print_step "Updating Netlify configuration for testnet..."
    
    # Update netlify.toml with testnet-specific configuration
    cat >> "$GATEWAY_DIR/netlify.toml" << EOF

# =============================================================================
# KNIRVTESTNET CONFIGURATION
# =============================================================================

# Testnet build context
[context.testnet]
  base = "knirvtestnet/"
  publish = "knirvtestnet/"
  command = "echo 'KNIRVTESTNET static files - no build needed'"

  [context.testnet.environment]
    NODE_ENV = "testnet"
    VITE_API_BASE_URL = "http://$TESTNET_IP"
    VITE_TESTNET_IP = "$TESTNET_IP"
    VITE_KNIRVORACLE_URL = "http://$TESTNET_IP:1317"
    VITE_KNIRVCHAIN_URL = "http://$TESTNET_IP:8080"
    VITE_KNIRVGRAPH_URL = "http://$TESTNET_IP:8081"
    VITE_KNIRVNEXUS_1_URL = "http://$TESTNET_IP:8082"
    VITE_KNIRVNEXUS_2_URL = "http://$TESTNET_IP:8083"
    VITE_KNIRVROUTER_URL = "http://$TESTNET_IP:8086"
    VITE_KNIRVGATEWAY_URL = "http://$TESTNET_IP:8087"
    VITE_IPFS_API_URL = "http://$TESTNET_IP:5001"
    VITE_IPFS_GATEWAY_URL = "http://$TESTNET_IP:8080"
    VITE_TENDERMINT_RPC_URL = "http://$TESTNET_IP:26657"

# Testnet redirect rules
[[redirects]]
  from = "/testnet"
  to = "/knirvtestnet/"
  status = 200

[[redirects]]
  from = "/testnet/*"
  to = "/knirvtestnet/:splat"
  status = 200

# Testnet API proxy redirects
[[redirects]]
  from = "/api/testnet/root/*"
  to = "http://$TESTNET_IP:1317/:splat"
  status = 200

[[redirects]]
  from = "/api/testnet/chain/*"
  to = "http://$TESTNET_IP:8080/:splat"
  status = 200

[[redirects]]
  from = "/api/testnet/graph/*"
  to = "http://$TESTNET_IP:8081/:splat"
  status = 200

[[redirects]]
  from = "/api/testnet/nexus1/*"
  to = "http://$TESTNET_IP:8082/:splat"
  status = 200

[[redirects]]
  from = "/api/testnet/nexus2/*"
  to = "http://$TESTNET_IP:8083/:splat"
  status = 200

[[redirects]]
  from = "/api/testnet/router/*"
  to = "http://$TESTNET_IP:8086/:splat"
  status = 200

[[redirects]]
  from = "/api/testnet/gateway/*"
  to = "http://$TESTNET_IP:8087/:splat"
  status = 200

[[redirects]]
  from = "/api/testnet/ipfs/*"
  to = "http://$TESTNET_IP:5001/:splat"
  status = 200

# Testnet headers for CORS
[[headers]]
  for = "/api/testnet/*"
  [headers.values]
    Access-Control-Allow-Origin = "*"
    Access-Control-Allow-Methods = "GET, POST, PUT, DELETE, OPTIONS"
    Access-Control-Allow-Headers = "Content-Type, Authorization"
EOF
    
    print_success "Netlify configuration updated"
}

create_testnet_api_config() {
    print_step "Creating testnet API configuration..."
    
    cat > "$FRONTEND_DIR/testnet-config.js" << EOF
// KNIRVTESTNET Configuration
// This file contains the testnet-specific configuration for frontend integration

window.KNIRV_TESTNET_CONFIG = {
    // Testnet identification
    network: 'testnet',
    chainId: 'knirv-testnet-1',
    testnetIP: '$TESTNET_IP',
    
    // Service endpoints
    endpoints: {
        knirvoracle: {
            api: 'http://$TESTNET_IP:1317',
            rpc: 'http://$TESTNET_IP:26657',
            health: 'http://$TESTNET_IP:1317/health'
        },
        knirvchain: {
            api: 'http://$TESTNET_IP:8080',
            health: 'http://$TESTNET_IP:8080/health'
        },
        knirvgraph: {
            api: 'http://$TESTNET_IP:8081',
            health: 'http://$TESTNET_IP:8081/health'
        },
        knirvnexus: {
            node1: {
                api: 'http://$TESTNET_IP:8082',
                health: 'http://$TESTNET_IP:8082/health'
            },
            node2: {
                api: 'http://$TESTNET_IP:8083',
                health: 'http://$TESTNET_IP:8083/health'
            }
        },
        knirvrouter: {
            api: 'http://$TESTNET_IP:8086',
            health: 'http://$TESTNET_IP:8086/health'
        },
        knirvgateway: {
            api: 'http://$TESTNET_IP:8087',
            health: 'http://$TESTNET_IP:8087/health'
        },
        ipfs: {
            api: 'http://$TESTNET_IP:5001',
            gateway: 'http://$TESTNET_IP:8080',
            health: 'http://$TESTNET_IP:5001/api/v0/version'
        }
    },
    
    // Netlify proxy endpoints
    proxyEndpoints: {
        root: '/api/testnet/root',
        chain: '/api/testnet/chain',
        graph: '/api/testnet/graph',
        nexus1: '/api/testnet/nexus1',
        nexus2: '/api/testnet/nexus2',
        router: '/api/testnet/router',
        gateway: '/api/testnet/gateway',
        ipfs: '/api/testnet/ipfs'
    },
    
    // Feature flags
    features: {
        realTimeUpdates: true,
        healthChecks: true,
        mockData: false,
        debugMode: true
    },
    
    // Authentication (testnet simplified)
    auth: {
        required: false,
        testTokens: {
            admin: 'testnet-admin-123',
            developer: 'testnet-dev-456',
            observer: 'testnet-observer-789'
        }
    }
};

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = window.KNIRV_TESTNET_CONFIG;
}
EOF
    
    print_success "Testnet API configuration created"
}

update_existing_frontend_files() {
    print_step "Updating existing frontend integration files..."
    
    # Update any existing JavaScript files to use testnet configuration
    if [ -f "$FRONTEND_DIR/js/testnet.js" ]; then
        # Add testnet IP to existing testnet.js
        sed -i "s/localhost/$TESTNET_IP/g" "$FRONTEND_DIR/js/testnet.js"
        sed -i "s/127.0.0.1/$TESTNET_IP/g" "$FRONTEND_DIR/js/testnet.js"
    fi
    
    # Update any configuration files
    find "$FRONTEND_DIR" -name "*.json" -type f -exec sed -i "s/localhost/$TESTNET_IP/g" {} \;
    find "$FRONTEND_DIR" -name "*.yaml" -type f -exec sed -i "s/localhost/$TESTNET_IP/g" {} \;
    find "$FRONTEND_DIR" -name "*.yml" -type f -exec sed -i "s/localhost/$TESTNET_IP/g" {} \;
    
    print_success "Existing frontend files updated"
}

create_readme() {
    print_step "Creating testnet frontend README..."
    
    cat > "$FRONTEND_DIR/README.md" << EOF
# KNIRV Testnet Frontend

This directory contains the frontend interface for the KNIRV Testnet, integrated with the KNIRVGATEWAY Netlify deployment.

## Overview

The KNIRV Testnet is a live blockchain development environment running on AWS EC2, accessible through this Netlify-hosted frontend.

## Testnet Information

- **Testnet IP**: $TESTNET_IP
- **Network**: knirv-testnet-1
- **Environment**: Development/Testing

## Service Endpoints

| Service | Port | Endpoint | Description |
|---------|------|----------|-------------|
| KNIRVORACLE | 1317 | http://$TESTNET_IP:1317 | Core blockchain API |
| KNIRVCHAIN | 8080 | http://$TESTNET_IP:8080 | Smart contracts & LLM validation |
| KNIRVGRAPH | 8081 | http://$TESTNET_IP:8081 | Graph storage & DHT |
| KNIRVNEXUS-1 | 8082 | http://$TESTNET_IP:8082 | TEE simulation node 1 |
| KNIRVNEXUS-2 | 8083 | http://$TESTNET_IP:8083 | TEE simulation node 2 |
| KNIRVROUTER | 8086 | http://$TESTNET_IP:8086 | Network routing |
| KNIRVGATEWAY | 8087 | http://$TESTNET_IP:8087 | API gateway |
| IPFS API | 5001 | http://$TESTNET_IP:5001 | IPFS API |
| IPFS Gateway | 8080 | http://$TESTNET_IP:8080 | IPFS Gateway |
| Tendermint RPC | 26657 | http://$TESTNET_IP:26657 | Tendermint RPC |

## Frontend Integration

### Direct Access
- **Testnet Frontend**: https://knirv.com/testnet
- **Local Development**: Access via KNIRVGATEWAY

### API Proxy
All testnet services are accessible through Netlify proxy endpoints:
- \`/api/testnet/root/*\` → KNIRVORACLE
- \`/api/testnet/chain/*\` → KNIRVCHAIN  
- \`/api/testnet/graph/*\` → KNIRVGRAPH
- \`/api/testnet/nexus1/*\` → KNIRVNEXUS-1
- \`/api/testnet/nexus2/*\` → KNIRVNEXUS-2
- \`/api/testnet/router/*\` → KNIRVROUTER
- \`/api/testnet/gateway/*\` → KNIRVGATEWAY
- \`/api/testnet/ipfs/*\` → IPFS API

### Configuration
The testnet configuration is available in \`testnet-config.js\` and includes:
- Service endpoints
- Proxy configurations  
- Feature flags
- Authentication settings

## Development

### Local Testing
1. Ensure testnet services are running
2. Update configuration if testnet IP changes
3. Test service connectivity

### Deployment
The frontend is automatically deployed via Netlify when changes are pushed to the main branch.

### Health Monitoring
The frontend includes real-time health monitoring for all testnet services with automatic status updates.

## Files

- \`index.html\` - Main testnet dashboard
- \`testnet-config.js\` - Configuration file
- \`TESTING_GUIDE.md\` - Testing documentation
- \`config/\` - Service configuration files

## Support

For issues with the testnet:
1. Check service status on the dashboard
2. Review testnet logs: \`ssh knirv-testnet 'docker-compose logs'\`
3. Restart services if needed: \`make deploy-testnet-services\`

## Last Updated

Frontend last updated: $(date)
Testnet IP: $TESTNET_IP
EOF
    
    print_success "Testnet frontend README created"
}

display_summary() {
    echo ""
    echo -e "${GREEN}🎉 KNIRVTESTNET Frontend Update Complete!${NC}"
    echo -e "${BLUE}=========================================${NC}"
    echo ""
    echo -e "${YELLOW}Frontend Information:${NC}"
    echo -e "  🌐 Testnet IP: $TESTNET_IP"
    echo -e "  📁 Frontend Directory: $FRONTEND_DIR"
    echo -e "  🔗 Testnet URL: https://knirv.com/testnet"
    echo ""
    echo -e "${YELLOW}Updated Files:${NC}"
    echo -e "  ✅ index.html - Main testnet dashboard"
    echo -e "  ✅ testnet-config.js - API configuration"
    echo -e "  ✅ netlify.toml - Netlify configuration"
    echo -e "  ✅ README.md - Documentation"
    echo ""
    echo -e "${YELLOW}Next Steps:${NC}"
    echo -e "  1. Commit and push changes to deploy via Netlify"
    echo -e "  2. Access testnet at: https://knirv.com/testnet"
    echo -e "  3. Monitor services via the dashboard"
    echo ""
    echo -e "${CYAN}Git Commands:${NC}"
    echo -e "  git add KNIRVGATEWAY/knirvtestnet/"
    echo -e "  git add KNIRVGATEWAY/netlify.toml"
    echo -e "  git commit -m 'Update KNIRVTESTNET frontend with IP $TESTNET_IP'"
    echo -e "  git push origin main"
    echo ""
}

# Main execution
main() {
    print_header
    
    check_prerequisites
    sync_testnet_frontend
    create_testnet_index
    update_netlify_config
    create_testnet_api_config
    update_existing_frontend_files
    create_readme
    display_summary
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h    Show this help message"
        echo "  --force       Skip confirmation prompts"
        echo ""
        echo "Prerequisites:"
        echo "  - Testnet infrastructure must be deployed"
        echo "  - KNIRVGATEWAY directory must exist"
        echo "  - KNIRVTESTNET directory must exist"
        exit 0
        ;;
    --force)
        print_warning "Force mode - skipping confirmations"
        ;;
    *)
        # Confirm update
        echo -e "${YELLOW}This will update the KNIRVTESTNET frontend integration.${NC}"
        echo -e "${YELLOW}This will modify KNIRVGATEWAY/netlify.toml and frontend files.${NC}"
        echo ""
        read -p "Continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Update cancelled."
            exit 0
        fi
        ;;
esac

# Run main function
main
