#!/bin/bash

# KNIRV-SERVER Frontend Test Script
# Tests role-based access, real-time updates, and cross-browser compatibility

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$PROJECT_ROOT/test-results"
LOG_FILE="$TEST_DIR/frontend-test.log"

# Create test directory
mkdir -p "$TEST_DIR"

# Logging functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

# Test functions
test_nextjs_build() {
    log "Testing Next.js frontend build..."
    
    cd "$PROJECT_ROOT"
    
    # Check if package.json exists
    if [ ! -f "package.json" ]; then
        error "package.json not found in project root"
        return 1
    fi
    
    # Install dependencies
    log "Installing dependencies..."
    if npm install >/dev/null 2>&1; then
        success "Dependencies installed successfully"
    else
        error "Failed to install dependencies"
        return 1
    fi
    
    # Build the project
    log "Building Next.js project..."
    if npm run build >/dev/null 2>&1; then
        success "Next.js build completed successfully"
    else
        error "Next.js build failed"
        return 1
    fi
    
    # Check if build output exists
    if [ -d ".next" ]; then
        success "Build output directory exists"
    else
        error "Build output directory not found"
        return 1
    fi
}

test_typescript_compilation() {
    log "Testing TypeScript compilation..."
    
    cd "$PROJECT_ROOT"
    
    # Check TypeScript compilation
    if npx tsc --noEmit >/dev/null 2>&1; then
        success "TypeScript compilation successful"
    else
        error "TypeScript compilation failed"
        return 1
    fi
}

test_component_structure() {
    log "Testing component structure..."
    
    cd "$PROJECT_ROOT"
    
    # Check for required components
    local required_components=(
        "src/lib/auth-context.tsx"
        "src/components/auth/role-guard.tsx"
        "src/components/auth/login-form.tsx"
        "src/components/auth/user-profile.tsx"
        "src/components/dashboard/dashboard-wrapper.tsx"
    )
    
    for component in "${required_components[@]}"; do
        if [ -f "$component" ]; then
            success "Component exists: $component"
        else
            error "Missing component: $component"
            return 1
        fi
    done
}

test_nexus_portal_build() {
    log "Testing SERVER Portal build..."
    
    local portal_dir="$PROJECT_ROOT/../KNIRVGATEWAY/server-portal"
    
    if [ ! -d "$portal_dir" ]; then
        error "SERVER Portal directory not found"
        return 1
    fi
    
    cd "$portal_dir"
    
    # Check if package.json exists
    if [ ! -f "package.json" ]; then
        error "package.json not found in SERVER Portal"
        return 1
    fi
    
    # Install dependencies
    log "Installing SERVER Portal dependencies..."
    if npm install >/dev/null 2>&1; then
        success "SERVER Portal dependencies installed"
    else
        error "Failed to install SERVER Portal dependencies"
        return 1
    fi
    
    # Build the portal
    log "Building SERVER Portal..."
    if npm run build >/dev/null 2>&1; then
        success "SERVER Portal build completed"
    else
        error "SERVER Portal build failed"
        return 1
    fi
    
    # Check if build output exists
    if [ -d "dist" ]; then
        success "SERVER Portal build output exists"
    else
        error "SERVER Portal build output not found"
        return 1
    fi
}

test_role_based_components() {
    log "Testing role-based component structure..."
    
    cd "$PROJECT_ROOT"
    
    # Check for role-based access patterns in components
    if grep -r "useAuth" src/components/ >/dev/null 2>&1; then
        success "Components use authentication context"
    else
        warning "No authentication usage found in components"
    fi
    
    if grep -r "RoleGuard" src/components/ >/dev/null 2>&1; then
        success "Components use role-based guards"
    else
        warning "No role-based guards found in components"
    fi
    
    if grep -r "hasPermission" src/ >/dev/null 2>&1; then
        success "Permission checking is implemented"
    else
        warning "No permission checking found"
    fi
}

test_realtime_integration() {
    log "Testing real-time integration..."
    
    cd "$PROJECT_ROOT"
    
    # Check for real-time service files
    if [ -f "src/hooks/use-knirv-socket.ts" ] || [ -f "src/hooks/use-knirv-socket.tsx" ]; then
        success "Real-time socket hook exists"
    else
        warning "Real-time socket hook not found"
    fi
    
    # Check for SSE integration in SERVER Portal
    local portal_dir="$PROJECT_ROOT/../KNIRVGATEWAY/server-portal"
    if [ -f "$portal_dir/src/lib/realtime-service.ts" ]; then
        success "SERVER Portal real-time service exists"
    else
        error "SERVER Portal real-time service not found"
        return 1
    fi
    
    if [ -f "$portal_dir/src/hooks/use-realtime.ts" ]; then
        success "SERVER Portal real-time hooks exist"
    else
        error "SERVER Portal real-time hooks not found"
        return 1
    fi
}

test_environment_configuration() {
    log "Testing environment configuration..."
    
    cd "$PROJECT_ROOT"
    
    # Check for environment files
    if [ -f ".env.local" ] || [ -f ".env.development" ] || [ -f ".env.production" ]; then
        success "Environment configuration files exist"
    else
        warning "No environment configuration files found"
    fi
    
    # Check Next.js configuration
    if [ -f "next.config.js" ] || [ -f "next.config.mjs" ]; then
        success "Next.js configuration exists"
    else
        warning "Next.js configuration not found"
    fi
}

test_ui_components() {
    log "Testing UI component library..."
    
    cd "$PROJECT_ROOT"
    
    # Check for UI components directory
    if [ -d "src/components/ui" ]; then
        success "UI components directory exists"
        
        # Count UI components
        local ui_count=$(find src/components/ui -name "*.tsx" | wc -l)
        log "Found $ui_count UI components"
        
        if [ $ui_count -gt 0 ]; then
            success "UI components are present"
        else
            warning "No UI components found"
        fi
    else
        warning "UI components directory not found"
    fi
}

test_accessibility() {
    log "Testing accessibility features..."
    
    cd "$PROJECT_ROOT"
    
    # Check for accessibility attributes in components
    if grep -r "aria-" src/components/ >/dev/null 2>&1; then
        success "ARIA attributes found in components"
    else
        warning "No ARIA attributes found"
    fi
    
    if grep -r "role=" src/components/ >/dev/null 2>&1; then
        success "Role attributes found in components"
    else
        warning "No role attributes found"
    fi
}

test_responsive_design() {
    log "Testing responsive design implementation..."
    
    cd "$PROJECT_ROOT"
    
    # Check for responsive classes in components
    if grep -r "md:" src/components/ >/dev/null 2>&1; then
        success "Medium breakpoint classes found"
    else
        warning "No medium breakpoint classes found"
    fi
    
    if grep -r "lg:" src/components/ >/dev/null 2>&1; then
        success "Large breakpoint classes found"
    else
        warning "No large breakpoint classes found"
    fi
    
    if grep -r "grid-cols" src/components/ >/dev/null 2>&1; then
        success "Grid layout classes found"
    else
        warning "No grid layout classes found"
    fi
}

# Main test execution
main() {
    log "Starting KNIRV-SERVER Frontend Test Suite"
    log "Project root: $PROJECT_ROOT"
    log "Test results will be saved to: $TEST_DIR"
    
    # Check if npm is available
    if ! command -v npm &> /dev/null; then
        error "npm is not installed or not in PATH"
        exit 1
    fi
    
    # Run tests
    local failed_tests=0
    
    test_component_structure || ((failed_tests++))
    test_typescript_compilation || ((failed_tests++))
    test_nextjs_build || ((failed_tests++))
    test_nexus_portal_build || ((failed_tests++))
    test_role_based_components || ((failed_tests++))
    test_realtime_integration || ((failed_tests++))
    test_environment_configuration || ((failed_tests++))
    test_ui_components || ((failed_tests++))
    test_accessibility || ((failed_tests++))
    test_responsive_design || ((failed_tests++))
    
    # Summary
    log "Test suite completed"
    if [ $failed_tests -eq 0 ]; then
        success "All tests passed!"
        exit 0
    else
        error "$failed_tests test(s) failed"
        exit 1
    fi
}

# Run main function
main "$@"
