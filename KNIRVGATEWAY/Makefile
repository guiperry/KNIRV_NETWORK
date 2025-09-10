# KNIRVGATEWAY Makefile
# Content synchronization and deployment management

.PHONY: help sync-failover-page update-submodules build deploy-render deploy-netlify deploy-vercel test-provision clean

# Default target
help:
	@echo "KNIRVGATEWAY Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  sync-failover-page    - Sync content from knirvcom-repo to home.html"
	@echo "  update-submodules     - Update all git submodules"
	@echo "  build                 - Build the gateway for deployment"
	@echo "  deploy-render         - Deploy to Render (via Git)"
	@echo "  deploy-netlify        - Deploy to Netlify"
	@echo "  deploy-vercel         - Deploy to Vercel"
	@echo "  test-provision        - Test the /provision endpoint"
	@echo "  install-deps          - Install all dependencies"
	@echo "  clean                 - Clean build artifacts"

# Content synchronization target
sync-failover-page:
	@echo "🔄 Syncing failover page content..."
	@if [ -d "knirvcom-repo" ]; then \
		echo "📁 Found knirvcom-repo submodule"; \
		cd knirvcom-repo && git pull origin main; \
		cd ..; \
		if [ -f "knirvcom-repo/index.html" ]; then \
			echo "📄 Copying index.html to home.html"; \
			cp knirvcom-repo/index.html home.html; \
		else \
			echo "⚠️  knirvcom-repo/index.html not found"; \
		fi; \
		if [ -d "knirvcom-repo/assets" ]; then \
			echo "📁 Syncing assets directory"; \
			rsync -av --delete knirvcom-repo/assets/ assets/; \
		fi; \
		if [ -d "knirvcom-repo/images" ]; then \
			echo "🖼️  Syncing images directory"; \
			rsync -av --delete knirvcom-repo/images/ images/; \
		fi; \
		echo "✅ Failover page sync completed"; \
	else \
		echo "⚠️  knirvcom-repo submodule not found"; \
		echo "💡 Run 'make setup-submodules' to initialize"; \
	fi

# Update git submodules
update-submodules:
	@echo "🔄 Updating git submodules..."
	@git submodule update --init --recursive --remote
	@echo "✅ Submodules updated"

# Setup submodules (for initial setup)
setup-submodules:
	@echo "🔧 Setting up knirvcom-repo submodule..."
	@if [ ! -d "knirvcom-repo" ]; then \
		echo "📥 Adding knirvcom-repo as submodule"; \
		git submodule add https://github.com/KNIRV-NETWORK/knirvcom-repo.git knirvcom-repo; \
	fi
	@git submodule update --init --recursive
	@echo "✅ Submodules setup completed"

# Build target
build: sync-failover-page
	@echo "🔨 Building KNIRVGATEWAY..."
	@npm run build
	@echo "✅ Build completed"

# Install dependencies
install-deps:
	@echo "📦 Installing dependencies..."
	@npm install
	@echo "✅ Dependencies installed"

# Deployment targets
deploy-render:
	@echo "🚀 Deploying to Render..."
	@echo "💡 Render deployment is handled via Git integration"
	@echo "📤 Push your changes to trigger deployment:"
	@echo "   git add ."
	@echo "   git commit -m 'Deploy to Render'"
	@echo "   git push origin main"

deploy-netlify: build
	@echo "🚀 Deploying to Netlify..."
	@npm run deploy:netlify

deploy-vercel: build
	@echo "🚀 Deploying to Vercel..."
	@npm run deploy:vercel

# Test the provision endpoint
test-provision:
	@echo "🧪 Testing /provision endpoint..."
	@echo "📡 Testing local server (if running)..."
	@curl -f http://localhost:8080/provision || echo "❌ Local server not available"
	@echo ""
	@echo "📡 Testing Netlify function..."
	@curl -f http://localhost:8888/.netlify/functions/provision || echo "❌ Netlify dev server not available"

# Test health endpoints
test-health:
	@echo "🏥 Testing health endpoints..."
	@echo "📡 Testing local server health..."
	@curl -f http://localhost:8080/health || echo "❌ Local server not available"
	@echo ""
	@echo "📡 Testing Netlify health..."
	@curl -f http://localhost:8888/.netlify/functions/gateway-sse || echo "❌ Netlify dev server not available"

# Development targets
dev-render:
	@echo "🔧 Starting development server in Render mode..."
	@GATEWAY_MODE=persistent npm run start:render

dev-netlify:
	@echo "🔧 Starting development server in Netlify mode..."
	@npm run dev

dev-vercel:
	@echo "🔧 Starting development server in Vercel mode..."
	@npm run start:vercel

# Generate comprehensive documentation with static HTML
docs:
	@echo "📚 Generating comprehensive documentation..."
	@echo "🔄 Step 1: Running doc_generator.js to consolidate all documentation"
	@node scripts/doc_generator.js
	@echo "🔄 Step 2: Installing documentation dependencies"
	@cd knirv.com/public/documentation && npm install
	@echo "🔄 Step 3: Generating static HTML from docsify"
	@cd knirv.com && node scripts/simple-static-generator.js
	@echo "🔄 Step 4: Updating links to point to static version"
	@cd knirv.com && node scripts/update-docsify-links.js
	@echo "✅ Documentation generation completed"
	@echo "📁 Static documentation available at: knirv.com/public/documentation/static/"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf node_modules/.cache
	@rm -rf .netlify
	@rm -rf .vercel
	@echo "✅ Clean completed"

# Full clean (including node_modules)
clean-all: clean
	@echo "🧹 Full clean (including node_modules)..."
	@rm -rf node_modules
	@rm -f package-lock.json
	@echo "✅ Full clean completed"

# Lint and format
lint:
	@echo "🔍 Linting code..."
	@npm run lint || echo "⚠️  Linting not configured"

# Security audit
audit:
	@echo "🔒 Running security audit..."
	@npm audit

# Complete setup for new environment
setup: install-deps setup-submodules sync-failover-page
	@echo "🎉 KNIRVGATEWAY setup completed!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Configure environment variables"
	@echo "2. Run 'make dev-netlify' for local development"
	@echo "3. Run 'make test-provision' to test endpoints"

# CI/CD target
ci-build: install-deps update-submodules build test-provision
	@echo "✅ CI/CD build completed"

# Show current configuration
status:
	@echo "📊 KNIRVGATEWAY Status"
	@echo "====================="
	@echo "Node version: $(shell node --version 2>/dev/null || echo 'Not installed')"
	@echo "NPM version: $(shell npm --version 2>/dev/null || echo 'Not installed')"
	@echo "Git status: $(shell git status --porcelain | wc -l) uncommitted changes"
	@echo "Submodules:"
	@git submodule status 2>/dev/null || echo "  No submodules configured"
	@echo ""
	@echo "Files:"
	@echo "  index.html: $(shell [ -f index.html ] && echo 'Present' || echo 'Missing')"
	@echo "  home.html: $(shell [ -f home.html ] && echo 'Present' || echo 'Missing')"
	@echo "  server.js: $(shell [ -f server.js ] && echo 'Present' || echo 'Missing')"
	@echo ""
	@echo "Dependencies:"
	@echo "  node_modules: $(shell [ -d node_modules ] && echo 'Installed' || echo 'Not installed')"
