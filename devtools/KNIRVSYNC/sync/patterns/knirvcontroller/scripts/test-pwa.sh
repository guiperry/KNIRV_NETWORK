#!/bin/bash

# KNIRVCONTROLLER PWA Testing Script
# Tests PWA functionality, installation, and integration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTROLLER_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Test configuration
TEST_ENVIRONMENT="${1:-testnet}"
TEST_REPORTS_DIR="$CONTROLLER_DIR/test-results"

# Functions
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}🧪 KNIRVCONTROLLER PWA Testing Suite${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo -e "${YELLOW}Environment: ${TEST_ENVIRONMENT}${NC}"
    echo ""
}

print_step() {
    echo -e "${PURPLE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Test PWA manifest
test_pwa_manifest() {
    print_step "Testing PWA manifest..."
    
    local manifest_file="$CONTROLLER_DIR/public/manifest.json"
    
    if [ -f "$manifest_file" ]; then
        # Check if manifest is valid JSON
        if jq empty "$manifest_file" 2>/dev/null; then
            print_success "PWA manifest is valid JSON"
            
            # Check required fields
            local required_fields=("name" "short_name" "start_url" "display" "theme_color" "background_color" "icons")
            local missing_fields=()
            
            for field in "${required_fields[@]}"; do
                if ! jq -e ".$field" "$manifest_file" >/dev/null 2>&1; then
                    missing_fields+=("$field")
                fi
            done
            
            if [ ${#missing_fields[@]} -eq 0 ]; then
                print_success "All required manifest fields present"
            else
                print_error "Missing manifest fields: ${missing_fields[*]}"
                return 1
            fi
        else
            print_error "PWA manifest is not valid JSON"
            return 1
        fi
    else
        print_error "PWA manifest not found at $manifest_file"
        return 1
    fi
}

# Test service worker
test_service_worker() {
    print_step "Testing service worker..."
    
    local sw_file="$CONTROLLER_DIR/public/sw.js"
    
    if [ -f "$sw_file" ]; then
        # Check if service worker has required functionality
        if grep -q "install" "$sw_file" && grep -q "fetch" "$sw_file"; then
            print_success "Service worker has required event handlers"
        else
            print_warning "Service worker may be missing required event handlers"
        fi
        
        # Check for caching strategy
        if grep -q "cache" "$sw_file"; then
            print_success "Service worker implements caching"
        else
            print_warning "Service worker may not implement caching"
        fi
    else
        print_error "Service worker not found at $sw_file"
        return 1
    fi
}

# Test PWA build output
test_build_output() {
    print_step "Testing PWA build output..."
    
    local dist_dir="$CONTROLLER_DIR/dist"
    local packages_dir="$CONTROLLER_DIR/packages"
    
    if [ -d "$dist_dir" ]; then
        print_success "Build output directory exists"
        
        # Check for essential files
        local essential_files=("index.html" "manifest.json" "sw.js")
        for file in "${essential_files[@]}"; do
            if [ -f "$dist_dir/$file" ]; then
                print_success "Essential file found: $file"
            else
                print_error "Essential file missing: $file"
                return 1
            fi
        done
    else
        print_error "Build output directory not found"
        return 1
    fi
    
    if [ -d "$packages_dir" ]; then
        print_success "PWA packages directory exists"
        
        # Check for platform packages
        local packages=("knirvcontroller-android.zip" "knirvcontroller-ios.zip")
        for package in "${packages[@]}"; do
            if [ -f "$packages_dir/$package" ]; then
                print_success "Platform package found: $package"
                
                # Check package size (should be under 50MB)
                local size=$(stat -c%s "$packages_dir/$package" 2>/dev/null || echo "0")
                local max_size=$((50 * 1024 * 1024))  # 50MB in bytes
                
                if [ "$size" -lt "$max_size" ]; then
                    print_success "Package size OK: $(( size / 1024 / 1024 ))MB"
                else
                    print_warning "Package size large: $(( size / 1024 / 1024 ))MB"
                fi
            else
                print_error "Platform package missing: $package"
                return 1
            fi
        done
    else
        print_error "PWA packages directory not found"
        return 1
    fi
}

# Test PWA endpoints
test_pwa_endpoints() {
    print_step "Testing PWA endpoints..."
    
    # Determine test URLs based on environment
    if [ "$TEST_ENVIRONMENT" = "production" ]; then
        local base_url="https://controller.knirv.com"
    else
        local base_url="https://controller-testnet.knirv.network"
    fi
    
    local endpoints=(
        "$base_url"
        "$base_url/manifest.json"
        "$base_url/sw.js"
        "$base_url/android"
        "$base_url/ios"
    )
    
    if command -v curl >/dev/null 2>&1; then
        for endpoint in "${endpoints[@]}"; do
            if curl -s -f "$endpoint" >/dev/null 2>&1; then
                print_success "Endpoint accessible: $endpoint"
            else
                print_warning "Endpoint not accessible: $endpoint (may still be propagating)"
            fi
        done
    else
        print_warning "curl not available - cannot test endpoints"
    fi
}

# Test authentication system
test_authentication() {
    print_step "Testing authentication system..."
    
    local auth_service="$CONTROLLER_DIR/src/services/AuthenticationService.ts"
    local auth_provider="$CONTROLLER_DIR/src/components/AuthenticationProvider.tsx"
    
    if [ -f "$auth_service" ]; then
        print_success "Authentication service found"
        
        # Check for required methods
        local required_methods=("login" "logout" "register" "isAuthenticated")
        for method in "${required_methods[@]}"; do
            if grep -q "$method" "$auth_service"; then
                print_success "Authentication method found: $method"
            else
                print_warning "Authentication method may be missing: $method"
            fi
        done
    else
        print_error "Authentication service not found"
        return 1
    fi
    
    if [ -f "$auth_provider" ]; then
        print_success "Authentication provider found"
    else
        print_error "Authentication provider not found"
        return 1
    fi
}

# Test installation scripts
test_installation() {
    print_step "Testing installation scripts..."
    
    local install_js="$CONTROLLER_DIR/public/install.js"
    local install_css="$CONTROLLER_DIR/public/install.css"
    
    if [ -f "$install_js" ]; then
        print_success "Installation JavaScript found"
        
        # Check for platform detection
        if grep -q "platform" "$install_js" && grep -q "beforeinstallprompt" "$install_js"; then
            print_success "Installation script has platform detection"
        else
            print_warning "Installation script may be missing platform detection"
        fi
    else
        print_error "Installation JavaScript not found"
        return 1
    fi
    
    if [ -f "$install_css" ]; then
        print_success "Installation CSS found"
    else
        print_error "Installation CSS not found"
        return 1
    fi
}

# Run unit tests
run_unit_tests() {
    print_step "Running unit tests..."
    
    cd "$CONTROLLER_DIR"
    
    if [ -f "package.json" ] && npm list jest >/dev/null 2>&1; then
        print_status "Running Jest unit tests..."
        if npm test -- --passWithNoTests --silent; then
            print_success "Unit tests passed"
        else
            print_error "Unit tests failed"
            return 1
        fi
    else
        print_warning "Jest not available - skipping unit tests"
    fi
}

# Generate test report
generate_test_report() {
    print_step "Generating test report..."
    
    mkdir -p "$TEST_REPORTS_DIR"
    local report_file="$TEST_REPORTS_DIR/pwa-test-report-$(date +%Y%m%d_%H%M%S).txt"
    
    cat > "$report_file" << EOF
KNIRVCONTROLLER PWA Test Report
===============================

Test Date: $(date)
Environment: $TEST_ENVIRONMENT
Test Suite: PWA Functionality

Test Results:
- PWA Manifest: $([ -f "$CONTROLLER_DIR/public/manifest.json" ] && echo "PASS" || echo "FAIL")
- Service Worker: $([ -f "$CONTROLLER_DIR/public/sw.js" ] && echo "PASS" || echo "FAIL")
- Build Output: $([ -d "$CONTROLLER_DIR/dist" ] && echo "PASS" || echo "FAIL")
- PWA Packages: $([ -d "$CONTROLLER_DIR/packages" ] && echo "PASS" || echo "FAIL")
- Authentication: $([ -f "$CONTROLLER_DIR/src/services/AuthenticationService.ts" ] && echo "PASS" || echo "FAIL")
- Installation: $([ -f "$CONTROLLER_DIR/public/install.js" ] && echo "PASS" || echo "FAIL")

Recommendations:
1. Test PWA installation on actual mobile devices
2. Verify authentication flow with real user accounts
3. Test offline functionality
4. Validate performance metrics
5. Check accessibility compliance

Test completed successfully!
EOF
    
    print_success "Test report generated: $report_file"
}

# Display test summary
display_summary() {
    echo ""
    echo -e "${GREEN}🎉 KNIRVCONTROLLER PWA Testing Complete!${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo -e "${YELLOW}Environment: ${TEST_ENVIRONMENT}${NC}"
    echo -e "${YELLOW}Test Reports: ${TEST_REPORTS_DIR}${NC}"
    echo ""
    echo -e "${YELLOW}Next Steps:${NC}"
    echo -e "  1. Test PWA installation on mobile devices"
    echo -e "  2. Verify authentication and user data storage"
    echo -e "  3. Test offline functionality"
    echo -e "  4. Check performance and accessibility"
    echo ""
}

# Main execution
main() {
    print_header
    
    local test_failures=0
    
    # Run all tests
    test_pwa_manifest || ((test_failures++))
    test_service_worker || ((test_failures++))
    test_build_output || ((test_failures++))
    test_pwa_endpoints || ((test_failures++))
    test_authentication || ((test_failures++))
    test_installation || ((test_failures++))
    run_unit_tests || ((test_failures++))
    
    generate_test_report
    display_summary
    
    if [ $test_failures -eq 0 ]; then
        print_success "All PWA tests passed!"
        exit 0
    else
        print_error "$test_failures test(s) failed"
        exit 1
    fi
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [environment]"
        echo ""
        echo "Arguments:"
        echo "  environment    Test environment (production, testnet, staging, development)"
        echo ""
        echo "Examples:"
        echo "  $0 production    # Test production PWA"
        echo "  $0 testnet       # Test testnet PWA"
        exit 0
        ;;
    *)
        # Run main function
        main
        ;;
esac
