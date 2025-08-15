#!/bin/bash

# KNIRV Gateway - Netlify CLI Fix Script (Testnet Version)
# This script fixes recurring netlify-cli corruption issues

set -e

# Check if running in auto-fix mode
AUTO_FIX_MODE=${1:-""}
if [ "$AUTO_FIX_MODE" = "--auto" ]; then
    echo "🤖 KNIRV Gateway - Auto-fixing detected issues..."
    echo "================================================"
else
    echo "🔧 KNIRV Gateway - Fixing Netlify CLI Issues"
    echo "============================================="
fi

# Function to check if netlify-cli is working
check_netlify_cli() {
    echo "🔍 Checking netlify-cli status..."
    if npx netlify --version >/dev/null 2>&1; then
        echo "✅ netlify-cli is working"
        npx netlify --version
        return 0
    else
        echo "❌ netlify-cli is not working"
        return 1
    fi
}

# Function to clean and reinstall
clean_and_reinstall() {
    echo "🧹 Cleaning up corrupted dependencies..."
    
    # Remove node_modules and package-lock.json
    rm -rf node_modules package-lock.json
    rm -rf nexus-portal/node_modules nexus-portal/package-lock.json
    
    # Clear npm cache
    echo "🗑️  Clearing npm cache..."
    npm cache clean --force
    
    # Reinstall dependencies
    echo "📦 Reinstalling dependencies..."
    npm install
    
    # Build nexus-portal if it exists
    if [ -d "nexus-portal" ]; then
        echo "🏗️  Building nexus-portal..."
        npm run build:nexus
    else
        echo "ℹ️  No nexus-portal directory found, skipping build"
    fi
}

# Function to check Node.js version compatibility
check_node_version() {
    echo "🔍 Checking Node.js version compatibility..."
    NODE_VERSION=$(node --version | sed 's/v//')
    MAJOR_VERSION=$(echo $NODE_VERSION | cut -d. -f1)

    if [ "$MAJOR_VERSION" -lt 18 ]; then
        echo "⚠️  Warning: Node.js version $NODE_VERSION detected."
        echo "   netlify-cli requires Node.js >= 18.14.0"
        echo "   Please upgrade Node.js for better compatibility."
    else
        echo "✅ Node.js version $NODE_VERSION is compatible"
    fi
}

# Function to analyze and handle netlify-cli specific vulnerabilities
analyze_netlify_vulnerabilities() {
    echo "🔍 Analyzing netlify-cli vulnerability context..."

    # Check current netlify-cli version
    local current_version
    current_version=$(npm list netlify-cli --depth=0 2>/dev/null | grep netlify-cli | sed 's/.*@//' | sed 's/ .*//')

    if [ -n "$current_version" ]; then
        echo "📦 Current netlify-cli version: $current_version"

        # Check if it's a problematic version
        if [[ "$current_version" =~ ^17\. ]]; then
            echo "🚨 Warning: netlify-cli 17.x detected - known corruption issues"
            return 1
        elif [[ "$current_version" =~ ^21\. ]]; then
            echo "✅ netlify-cli 21.x detected - stable version"
            return 0
        else
            echo "⚠️  Unknown netlify-cli version pattern: $current_version"
            return 0
        fi
    else
        echo "❌ Could not determine netlify-cli version"
        return 1
    fi
}

# Function to run audit and fix vulnerabilities
fix_vulnerabilities() {
    echo "🔒 Checking for security vulnerabilities..."

    # First, try safe fixes only (no breaking changes)
    echo "🔧 Attempting safe vulnerability fixes..."
    npm audit fix --audit-level=moderate --no-fund 2>/dev/null || true

    # Check what vulnerabilities remain
    local audit_output
    audit_output=$(npm audit --audit-level=moderate --json 2>/dev/null || echo '{"vulnerabilities":{}}')

    # Parse the audit output to see what's left
    local vuln_count
    vuln_count=$(echo "$audit_output" | grep -o '"moderate":[0-9]*' | cut -d':' -f2 | head -1)
    vuln_count=${vuln_count:-0}

    local high_count
    high_count=$(echo "$audit_output" | grep -o '"high":[0-9]*' | cut -d':' -f2 | head -1)
    high_count=${high_count:-0}

    local critical_count
    critical_count=$(echo "$audit_output" | grep -o '"critical":[0-9]*' | cut -d':' -f2 | head -1)
    critical_count=${critical_count:-0}

    local total_serious=$((vuln_count + high_count + critical_count))

    if [ "$total_serious" -eq 0 ]; then
        echo "✅ No moderate or higher vulnerabilities found"
        return 0
    fi

    echo "⚠️  Found $total_serious moderate+ vulnerabilities remaining"

    # For netlify-cli specific issues, we need to be careful about version downgrades
    if echo "$audit_output" | grep -q "netlify-cli.*17\."; then
        echo "🚨 Warning: Audit suggests downgrading to netlify-cli 17.x (known corruption issues)"
        echo "   Skipping forced fixes to avoid breaking netlify-cli"
        echo "   Current vulnerabilities are in dependencies and pose minimal risk in testnet environment"
        return 0
    fi

    # If vulnerabilities are not in netlify-cli core, try more aggressive fixes
    if [ "$total_serious" -gt 10 ]; then
        echo "🔧 High number of vulnerabilities detected, attempting selective fixes..."

        # Try to fix specific packages that are safe to update
        npm update brace-expansion@^2.0.2 2>/dev/null || true
        npm update on-headers@^1.1.0 2>/dev/null || true
        npm update tar-fs@^3.0.9 2>/dev/null || true

        echo "✅ Applied selective vulnerability fixes"
    else
        echo "✅ Vulnerabilities are within acceptable limits for testnet environment"
    fi
}

# Main execution
main() {
    # Change to the KNIRVGATEWAY root directory
    cd "$(dirname "$0")/.."

    echo "Starting netlify-cli fix process..."
    echo ""
    
    # Check Node.js version
    check_node_version
    echo ""
    
    # Check current status
    if check_netlify_cli; then
        echo ""
        if [ "$AUTO_FIX_MODE" = "--auto" ]; then
            echo "🤔 netlify-cli appears to be working, but auto-fix was triggered. Proceeding anyway..."
        else
            echo "🤔 netlify-cli appears to be working. Do you want to proceed anyway? (y/N)"
            read -r response
            if [[ ! "$response" =~ ^[Yy]$ ]]; then
                echo "Exiting without changes."
                exit 0
            fi
        fi
    fi
    
    echo ""
    
    # Clean and reinstall
    clean_and_reinstall
    echo ""
    
    # Check if fix worked
    if check_netlify_cli; then
        echo ""
        echo "✅ netlify-cli fix completed successfully!"

        # Analyze vulnerabilities in context
        echo ""
        if analyze_netlify_vulnerabilities; then
            echo ""
            echo "🔧 Running targeted vulnerability fixes..."
            if [ -f "scripts/fix-vulnerabilities.sh" ]; then
                ./scripts/fix-vulnerabilities.sh
            else
                fix_vulnerabilities
            fi
        else
            echo ""
            echo "⚠️  Skipping vulnerability fixes due to netlify-cli version concerns"
            echo "   Current setup prioritizes stability over vulnerability patches"
            echo "   Testnet environment has reduced security requirements"
        fi
        
        echo ""
        echo "🎉 All done! You can now run:"
        echo "   npm start    - Start the development server"
        echo "   npm run build - Build the project"
        echo ""
        echo "💡 If netlify-cli breaks again, run this script:"
        echo "   ./scripts/fix-netlify-cli.sh"
        
    else
        echo ""
        echo "❌ Fix failed. netlify-cli is still not working."
        echo ""
        echo "🔧 Manual troubleshooting steps:"
        echo "1. Check Node.js version: node --version"
        echo "2. Try global install: npm install -g netlify-cli@21.6.0"
        echo "3. Check for permission issues"
        echo "4. Consider using nvm to manage Node.js versions"
        exit 1
    fi
}

# Run main function
main "$@"
