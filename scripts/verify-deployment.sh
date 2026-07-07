#!/bin/bash

echo "🚀 KNIRV Website Netlify Deployment Verification"
echo "================================================"

# Check if we're in the right directory
if [ ! -f "index.html" ]; then
    echo "❌ Error: Please run this script from the KNIRVGATEWAY directory"
    exit 1
fi

echo "✅ Main site files found"

# Check netlify configuration
if [ -f "netlify.toml" ]; then
    echo "✅ netlify.toml configuration found"
else
    echo "❌ netlify.toml missing"
    exit 1
fi

if [ -f "_redirects" ]; then
    echo "✅ _redirects file found"
else
    echo "❌ _redirects file missing"
    exit 1
fi

# Check documentation setup
if [ -d "documentation" ]; then
    echo "✅ Documentation directory found"
    
    cd documentation
    
    if [ -f "package.json" ]; then
        echo "✅ Documentation package.json found"
        
        # Check if node_modules exists
        if [ -d "node_modules" ]; then
            echo "✅ Documentation dependencies installed"
        else
            echo "⚠️  Installing documentation dependencies..."
            npm install
            if [ $? -eq 0 ]; then
                echo "✅ Documentation dependencies installed successfully"
            else
                echo "❌ Failed to install documentation dependencies"
                exit 1
            fi
        fi
        
        # Test build script
        echo "🔧 Testing documentation build script..."
        npm run build
        if [ $? -eq 0 ]; then
            echo "✅ Documentation build script works"
        else
            echo "❌ Documentation build script failed"
            exit 1
        fi
        
    else
        echo "❌ Documentation package.json missing"
        exit 1
    fi
    
    # Check docsify directory
    if [ -d "docsify" ]; then
        echo "✅ Docsify directory found"
        
        if [ -f "docsify/index.html" ]; then
            echo "✅ Docsify index.html found"
        else
            echo "❌ Docsify index.html missing"
            exit 1
        fi
        
        if [ -f "docsify/README.md" ]; then
            echo "✅ Docsify README.md found"
        else
            echo "❌ Docsify README.md missing"
            exit 1
        fi
        
    else
        echo "❌ Docsify directory missing"
        exit 1
    fi
    
    cd ..
else
    echo "❌ Documentation directory missing"
    exit 1
fi

echo ""
echo "🎉 All verification checks passed!"
echo ""
echo "📋 Deployment Summary:"
echo "   • Main site: Static HTML (no build needed)"
echo "   • Documentation: Docsify with npm dependencies"
echo "   • Netlify config: netlify.toml + _redirects"
echo "   • Node version: 18 (set in netlify.toml)"
echo ""
echo "🚀 Ready for Netlify deployment!"
echo ""
echo "Next steps:"
echo "1. Push changes to your Git repository"
echo "2. Connect repository to Netlify"
echo "3. Set base directory to 'KNIRVGATEWAY'"
echo "4. Deploy!"
