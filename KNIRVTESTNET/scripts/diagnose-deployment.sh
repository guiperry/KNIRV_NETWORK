#!/bin/bash
# Deployment Diagnostic Script
# This script provides comprehensive information about the deployment environment

echo "🔍 KNIRV TESTNET DEPLOYMENT DIAGNOSTICS"
echo "======================================="
echo "Diagnostic started at: $(date)"
echo "Process ID: $$"
echo "User: $(whoami)"
echo "Working directory: $(pwd)"
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_section() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

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

# 1. Environment Detection
print_section "ENVIRONMENT DETECTION"
print_info "Checking if running on Render..."
if [ "$RENDER" = "true" ] || [ -n "$RENDER_SERVICE_ID" ]; then
    print_success "✓ Render environment detected"
    print_info "Service ID: ${RENDER_SERVICE_ID:-'not set'}"
    print_info "Service Name: ${RENDER_SERVICE_NAME:-'not set'}"
    print_info "Service Type: ${RENDER_SERVICE_TYPE:-'not set'}"
    print_info "External URL: ${RENDER_EXTERNAL_URL:-'not set'}"
    print_info "Git Commit: ${RENDER_GIT_COMMIT:-'not set'}"
    print_info "Git Branch: ${RENDER_GIT_BRANCH:-'not set'}"
else
    print_warning "Not running on Render"
fi

# 2. Container Detection
print_section "CONTAINER DETECTION"
if [ -f /.dockerenv ]; then
    print_success "✓ Running inside Docker container"
    print_info "Container ID: $(hostname)"
    print_info "Container should handle service startup via Dockerfile"
    CONTAINER_MODE=true
else
    print_warning "Not running in Docker container"
    print_info "Running on host system"
    CONTAINER_MODE=false
fi

# 3. Environment Variables
print_section "ENVIRONMENT VARIABLES"
print_info "All Render-related variables:"
env | grep -E "^RENDER" | sort | while read line; do
    print_info "  $line"
done

print_info "Port and service variables:"
env | grep -E "^(PORT|KNIRV|NODE)" | sort | while read line; do
    print_info "  $line"
done

# 4. File System Analysis
print_section "FILE SYSTEM ANALYSIS"
print_info "Current directory: $(pwd)"
print_info "Directory contents:"
ls -la | head -15

print_info "Checking for key files and directories:"
KEY_ITEMS=(
    "package.json"
    "config/render.yml"
    "testnet-gateway.Dockerfile"
    "data/testnet-gateway"
    "data/testnet-gateway/index.html"
    "scripts/start-render.sh"
)

for item in "${KEY_ITEMS[@]}"; do
    if [ -e "$item" ]; then
        print_success "✓ $item exists"
    else
        print_error "✗ $item missing"
    fi
done

# 5. Network Configuration
print_section "NETWORK CONFIGURATION"
print_info "Hostname: $(hostname)"
print_info "IP addresses:"
ip addr show 2>/dev/null | grep "inet " | head -5 || ifconfig 2>/dev/null | grep "inet " | head -5 || print_warning "Cannot determine IP addresses"

print_info "Port configuration:"
print_info "  PORT environment variable: ${PORT:-'not set'}"
print_info "  Default port would be: 80 (nginx)"

# 6. Process Analysis
print_section "PROCESS ANALYSIS"
print_info "Current processes:"
ps aux | head -10 2>/dev/null || print_warning "Cannot list processes"

print_info "Listening ports:"
netstat -tlnp 2>/dev/null | head -10 || ss -tlnp 2>/dev/null | head -10 || print_warning "Cannot check listening ports"

# 7. Service-Specific Checks
print_section "SERVICE-SPECIFIC ANALYSIS"
if [ "$RENDER_SERVICE_NAME" = "knirv-testnet-gateway" ] || [ -z "$RENDER_SERVICE_NAME" ]; then
    print_info "🌐 TESTNET GATEWAY SERVICE"
    
    if [ "$CONTAINER_MODE" = "true" ]; then
        print_info "Container mode - checking nginx:"
        if command -v nginx >/dev/null 2>&1; then
            print_success "✓ nginx is available"
            print_info "nginx version: $(nginx -v 2>&1)"
            print_info "nginx config test:"
            nginx -t 2>&1 || print_error "nginx config test failed"
        else
            print_error "✗ nginx not found"
        fi
        
        print_info "Checking web content:"
        if [ -d "/usr/share/nginx/html" ]; then
            print_success "✓ nginx html directory exists"
            print_info "Contents:"
            ls -la /usr/share/nginx/html/ | head -10
        else
            print_error "✗ nginx html directory missing"
        fi
    else
        print_warning "Not in container mode - this is unexpected for Docker service"
    fi
else
    print_info "🔧 BACKEND SERVICE: ${RENDER_SERVICE_NAME}"
    print_info "Backend services should be fully containerized"
fi

# 8. Deployment Recommendations
print_section "DEPLOYMENT RECOMMENDATIONS"
if [ "$CONTAINER_MODE" = "true" ]; then
    if [ "$RENDER_SERVICE_NAME" = "knirv-testnet-gateway" ] || [ -z "$RENDER_SERVICE_NAME" ]; then
        print_success "✓ Correct setup: Gateway service in container"
        print_error "❌ But npm start should not run in container!"
        print_info "REQUIRED ACTION:"
        print_info "1. Go to Render dashboard → Service Settings"
        print_info "2. Change Start Command to: nginx -g \"daemon off;\""
        print_info "3. This matches the Dockerfile CMD instruction"
    else
        print_success "✓ Correct setup: Backend service in container"
        print_error "❌ But npm start should not run in container!"
        print_info "REQUIRED ACTION for ${RENDER_SERVICE_NAME}:"
        case "$RENDER_SERVICE_NAME" in
            "knirv-oracle")
                print_info "Start Command should be: ./knirv-oracle"
                ;;
            "knirv-chain")
                print_info "Start Command should be: ./knirv-chain"
                ;;
            "knirv-graph")
                print_info "Start Command should be: ./knirv-graph"
                ;;
            "knirv-nexus")
                print_info "Start Command should be: ./knirv-nexus"
                ;;
            "knirv-router")
                print_info "Start Command should be: ./knirv-router"
                ;;
            *)
                print_info "Check the service's Dockerfile CMD instruction"
                ;;
        esac
    fi
else
    print_error "✗ Incorrect setup: Service should be containerized"
    print_info "Recommendation: Check config/render.yml has runtime: docker"
    print_info "Action: Ensure Render dashboard is not overriding Docker config"
fi

# 9. Next Steps
print_section "NEXT STEPS"
if [ "$CONTAINER_MODE" = "true" ]; then
    print_error "CRITICAL: npm start should not be called in Docker containers"
    print_info ""
    print_info "IMMEDIATE ACTIONS REQUIRED:"
    print_info "1. Go to Render dashboard → Service Settings"
    print_info "2. Update Start Command (see recommendations above)"
    print_info "3. Remove any Build Command if present"
    print_info "4. Redeploy the service"
    print_info ""
    print_info "CORRECT START COMMANDS:"
    print_info "• knirv-testnet-gateway: nginx -g \"daemon off;\""
    print_info "• knirv-oracle: ./knirv-oracle"
    print_info "• knirv-chain: ./knirv-chain"
    print_info "• knirv-graph: ./knirv-graph"
    print_info "• knirv-nexus: ./knirv-nexus"
    print_info "• knirv-router: ./knirv-router"
else
    print_info "1. Verify config/render.yml has runtime: docker for all services"
    print_info "2. Check Render dashboard service configuration"
    print_info "3. Ensure Docker deployment is properly configured"
fi

echo ""
print_section "DIAGNOSTIC COMPLETE"
echo "Diagnostic finished at: $(date)"
echo "Summary: $([ "$CONTAINER_MODE" = "true" ] && echo "Container mode detected" || echo "Host mode detected")"
echo ""

# Exit with appropriate code
if [ "$CONTAINER_MODE" = "true" ]; then
    print_error "Exiting with error - npm scripts should not run in containers"
    exit 1
else
    print_info "Diagnostic complete - check recommendations above"
    exit 0
fi
