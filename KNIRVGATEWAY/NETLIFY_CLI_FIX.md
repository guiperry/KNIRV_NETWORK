# Netlify CLI Dependency Fix for Render Deployment

This document explains the netlify-cli dependency issues and fixes applied to KNIRVGATEWAY for Render deployment.

## Problem

The KNIRVGATEWAY application uses netlify/functions routes for serverless functionality, even when deployed in persistent mode on Render. However, netlify-cli was frequently becoming corrupted or missing during builds, causing deployment failures with errors like:

```
❌ Found 2 critical issues:
❌   - netlify-cli not found in node_modules
❌   - netlify-cli binary not found in node_modules
❌ Health check failed - automatic fix may be needed
```

## Root Cause

1. **Build Process Issues**: The build process wasn't consistently installing devDependencies on Render
2. **Corruption**: netlify-cli installation was prone to corruption during npm operations
3. **Version Conflicts**: Different versions of netlify-cli had compatibility issues
4. **Render Environment**: Render's build environment sometimes skipped devDependencies

## Solution

### 1. Automated Netlify CLI Ensurer

Created `scripts/ensure-netlify-cli.js` that:
- Checks if netlify-cli is properly installed
- Tests if the netlify command works
- Automatically installs or reinstalls netlify-cli if needed
- Uses a specific stable version (21.6.0)
- Handles timeouts and corruption gracefully

### 2. Updated Build Process

Modified package.json scripts:
- `ensure-netlify-cli`: Runs the automated ensurer
- `build`: Now includes netlify-cli check before building
- `build:persistent`: Optimized for Render deployment with netlify-cli
- `auto-fix`: Uses the new ensurer instead of shell script

### 3. Render Configuration

Updated `render.yaml`:
- Uses `npm install --include=dev` to ensure devDependencies are installed
- Uses `build:persistent` script optimized for Render

## Why Netlify CLI is Required on Render

Even in persistent mode on Render, KNIRVGATEWAY needs netlify-cli because:

1. **Netlify Functions**: The application uses `netlify/functions/` routes for serverless functionality
2. **SSE Support**: Server-Sent Events are implemented using netlify-cli for compatibility
3. **API Gateway**: Many API endpoints are routed through netlify functions
4. **Development Consistency**: Maintains compatibility between local development and deployment

## Usage

### Manual Fix
```bash
npm run ensure-netlify-cli
```

### Automatic Fix (during build)
The build process now automatically ensures netlify-cli is available:
```bash
npm run build
# or for Render deployment
npm run build:persistent
```

### Health Check
The health check script validates netlify-cli installation:
```bash
npm run check-health
```

## Troubleshooting

If netlify-cli issues persist on Render:

1. **Check Installation**:
   ```bash
   npm run ensure-netlify-cli
   ```

2. **Manual Clean Install**:
   ```bash
   npm uninstall netlify-cli
   npm cache clean --force
   npm install netlify-cli@21.6.0 --save-dev
   ```

3. **Verify Working**:
   ```bash
   npx netlify --version
   ```

4. **Check Health**:
   ```bash
   npm run check-health
   ```

## Files Modified

- `scripts/ensure-netlify-cli.js` - New automated ensurer script
- `scripts/check-health.js` - Always checks netlify-cli (no skipping in persistent mode)
- `package.json` - Updated build scripts and dependencies
- `render.yaml` - Updated build command for Render deployment

## Version History

- **v1.0**: Initial netlify-cli dependency management
- **v1.1**: Added automated ensurer script
- **v1.2**: Integrated with build process and Render deployment
- **v1.3**: Fixed persistent mode to still require netlify-cli (2025-01-09)
