#!/bin/bash

# KNIRV Gateway - Netlify CLI Fix Script
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

    # Try with timeout to avoid hanging
    if timeout 30s npx netlify --version >/dev/null 2>&1; then
        echo "✅ netlify-cli is working"
        timeout 30s npx netlify --version
        return 0
    else
        echo "❌ netlify-cli is not working or timed out"

        # Check if netlify-cli binary exists
        if [ -f "node_modules/.bin/netlify" ]; then
            echo "ℹ️  netlify-cli binary found but not responding"
        else
            echo "ℹ️  netlify-cli binary not found"
        fi
        return 1
    fi
}

# Function to clean and reinstall
clean_and_reinstall() {
    echo "🧹 Cleaning up corrupted dependencies..."

    # Remove node_modules and package-lock.json
    echo "🗑️  Removing corrupted node_modules..."
    rm -rf node_modules package-lock.json

    # Clear npm cache aggressively
    echo "🗑️  Clearing npm cache..."
    npm cache clean --force
    npm cache verify

    # Remove any global netlify-cli that might be interfering
    echo "🧹 Cleaning global netlify-cli..."
    npm uninstall -g netlify-cli 2>/dev/null || true

    # Reinstall dependencies with specific netlify-cli version
    echo "📦 Reinstalling dependencies..."
    npm install --include=dev

    # Explicitly install netlify-cli if not found
    echo "🔧 Ensuring netlify-cli is installed..."
    npm install netlify-cli@21.6.0 --save-dev

    # Verify netlify-cli installation
    echo "🔍 Verifying netlify-cli installation..."
    if [ -f "node_modules/.bin/netlify" ]; then
        echo "✅ netlify-cli binary installed successfully"
    else
        echo "❌ netlify-cli binary not found after installation"
        return 1
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

# Function to run audit and fix vulnerabilities (DISABLED to prevent netlify-cli corruption)
fix_vulnerabilities() {
    echo "🔒 Checking for security vulnerabilities..."
    if npm audit --audit-level=moderate >/dev/null 2>&1; then
        echo "✅ No moderate or high vulnerabilities found"
    else
        echo "⚠️  Vulnerabilities detected, but skipping fixes to prevent netlify-cli corruption"
        echo "   Testnet environment has reduced security requirements"
        echo "   Manual vulnerability review recommended for production"
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
        
        # Skip vulnerability fixes to prevent netlify-cli corruption
        echo ""
        echo "⚠️  Skipping vulnerability fixes to maintain netlify-cli stability"
        echo "   Vulnerability fixes can corrupt netlify-cli installation"
        echo "   Testnet environment has reduced security requirements"
        
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
