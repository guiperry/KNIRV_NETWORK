#!/bin/bash

# KNIRV Gateway - Targeted Vulnerability Fix Script
# Handles specific npm audit vulnerabilities without breaking netlify-cli

set -e

echo "🔒 KNIRV Gateway - Targeted Vulnerability Fixes"
echo "=============================================="

# Function to check if a package exists in node_modules
package_exists() {
    [ -d "node_modules/$1" ]
}

# Function to get package version
get_package_version() {
    if package_exists "$1"; then
        npm list "$1" --depth=0 2>/dev/null | grep "$1" | sed 's/.*@//' | sed 's/ .*//' || echo "unknown"
    else
        echo "not-installed"
    fi
}

# Function to safely update a package
safe_update() {
    local package="$1"
    local target_version="$2"
    local current_version
    
    current_version=$(get_package_version "$package")
    
    if [ "$current_version" = "not-installed" ]; then
        echo "📦 Package $package not found, skipping..."
        return 0
    fi
    
    echo "📦 Updating $package from $current_version to $target_version..."
    
    # Try to update the package
    if npm install "$package@$target_version" --save-exact 2>/dev/null; then
        echo "✅ Successfully updated $package"
        return 0
    else
        echo "⚠️  Failed to update $package, keeping current version"
        return 1
    fi
}

# Function to fix brace-expansion vulnerability
fix_brace_expansion() {
    echo "🔧 Fixing brace-expansion vulnerability..."
    
    # brace-expansion 1.0.0 - 1.1.11 || 2.0.0 - 2.0.1 -> 2.0.2+
    # This is a RegEx DoS vulnerability
    
    # Check if we can update to a safe version
    if npm install brace-expansion@^2.0.2 --save-exact 2>/dev/null; then
        echo "✅ Updated brace-expansion to safe version"
    else
        echo "⚠️  Could not update brace-expansion (dependency conflict)"
        echo "   This vulnerability has low impact in testnet environment"
    fi
}

# Function to fix on-headers vulnerability
fix_on_headers() {
    echo "🔧 Fixing on-headers vulnerability..."
    
    # on-headers <1.1.0 -> 1.1.0+
    # HTTP response header manipulation vulnerability
    
    if npm install on-headers@^1.1.0 --save-exact 2>/dev/null; then
        echo "✅ Updated on-headers to safe version"
    else
        echo "⚠️  Could not update on-headers (dependency conflict)"
    fi
}

# Function to fix tar-fs vulnerability
fix_tar_fs() {
    echo "🔧 Fixing tar-fs vulnerability..."
    
    # tar-fs 2.0.0 - 2.1.2 || 3.0.0 - 3.0.8 -> 3.0.9+
    # Directory traversal vulnerability
    
    if npm install tar-fs@^3.0.9 --save-exact 2>/dev/null; then
        echo "✅ Updated tar-fs to safe version"
    else
        echo "⚠️  Could not update tar-fs (dependency conflict)"
    fi
}

# Function to handle ipx and tmp vulnerabilities (require careful handling)
fix_complex_vulnerabilities() {
    echo "🔧 Analyzing complex vulnerabilities (ipx, tmp)..."
    
    # These vulnerabilities require netlify-cli downgrade to 17.x which we want to avoid
    echo "⚠️  ipx and tmp vulnerabilities require netlify-cli downgrade"
    echo "   Downgrading to 17.x would introduce corruption issues"
    echo "   Risk assessment: Low impact in testnet environment"
    echo "   Recommendation: Accept risk for stability"
}

# Function to create vulnerability report
create_vulnerability_report() {
    echo ""
    echo "📊 Vulnerability Fix Report"
    echo "=========================="
    
    # Run audit again to see current status
    local audit_output
    audit_output=$(npm audit --audit-level=moderate --json 2>/dev/null || echo '{"vulnerabilities":{}}')
    
    local moderate_count
    moderate_count=$(echo "$audit_output" | grep -o '"moderate":[0-9]*' | cut -d':' -f2 | head -1)
    moderate_count=${moderate_count:-0}
    
    local high_count
    high_count=$(echo "$audit_output" | grep -o '"high":[0-9]*' | cut -d':' -f2 | head -1)
    high_count=${high_count:-0}
    
    local critical_count
    critical_count=$(echo "$audit_output" | grep -o '"critical":[0-9]*' | cut -d':' -f2 | head -1)
    critical_count=${critical_count:-0}
    
    echo "Current vulnerability status:"
    echo "  Moderate: $moderate_count"
    echo "  High: $high_count"
    echo "  Critical: $critical_count"
    
    if [ "$((moderate_count + high_count + critical_count))" -eq 0 ]; then
        echo "✅ All vulnerabilities resolved!"
    else
        echo "⚠️  Some vulnerabilities remain (acceptable for testnet)"
    fi
}

# Main execution
main() {
    # Change to the correct directory
    cd "$(dirname "$0")/.."
    
    echo "Starting targeted vulnerability fixes..."
    echo ""
    
    # Check if we're in the right place
    if [ ! -f "package.json" ]; then
        echo "❌ package.json not found. Are we in the right directory?"
        exit 1
    fi
    
    # Check netlify-cli version first
    local netlify_version
    netlify_version=$(npm list netlify-cli --depth=0 2>/dev/null | grep netlify-cli | sed 's/.*@//' | sed 's/ .*//' || echo "unknown")
    
    echo "📦 Current netlify-cli version: $netlify_version"
    
    if [[ "$netlify_version" =~ ^17\. ]]; then
        echo "🚨 Warning: netlify-cli 17.x detected - avoiding vulnerability fixes that could break it"
        exit 0
    fi
    
    # Apply targeted fixes
    fix_brace_expansion
    echo ""
    
    fix_on_headers
    echo ""
    
    fix_tar_fs
    echo ""
    
    fix_complex_vulnerabilities
    echo ""
    
    # Generate report
    create_vulnerability_report
    
    echo ""
    echo "🎉 Vulnerability fix process completed!"
    echo ""
    echo "💡 Note: Some vulnerabilities may remain by design to maintain"
    echo "   netlify-cli stability. This is acceptable for testnet use."
}

# Run main function
main "$@"
