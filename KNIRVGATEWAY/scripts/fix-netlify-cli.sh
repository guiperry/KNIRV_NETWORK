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
    
    # Build nexus-portal
    echo "🏗️  Building nexus-portal..."
    npm run build:nexus
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

# Function to run audit and fix vulnerabilities
fix_vulnerabilities() {
    echo "🔒 Checking for security vulnerabilities..."
    if npm audit --audit-level=moderate >/dev/null 2>&1; then
        echo "✅ No moderate or high vulnerabilities found"
    else
        echo "⚠️  Vulnerabilities detected, attempting to fix..."
        npm audit fix --audit-level=moderate
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
        
        # Fix vulnerabilities
        echo ""
        fix_vulnerabilities
        
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
