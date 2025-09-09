# KNIRVGATEWAY Build System

## Overview

This document describes the automated build system for KNIRVGATEWAY that ensures Netlify functions work correctly by automatically patching missing dependencies during the build process.

## Problem Solved

During Netlify deployment, certain npm packages have missing `dist` files that cause build failures:

1. **bcryptjs** - Missing `dist/bcrypt.js` file that the main index.js requires
2. **formidable** - Missing `dist/*` files that the package.json exports point to

These issues cause Netlify's esbuild bundler to fail with "Could not resolve" errors.

## Solution Architecture

### 1. Automated Patch System

The build system includes an automated patcher that:
- Detects missing dist files in problematic packages
- Copies source files to the expected dist locations
- Verifies patches are applied correctly
- Prevents infinite loops during postinstall hooks

### 2. Build Integration

The patches are automatically applied during:
- **Netlify deployment** via `netlify.toml` build command
- **Local development** via npm scripts
- **CI/CD pipelines** via package.json build scripts

## Files and Scripts

### Core Patcher Script
- **`scripts/patch-netlify-deps.js`** - Main patcher that fixes missing dist files
  - Patches bcryptjs by copying `src/bcrypt.js` to `dist/bcrypt.js`
  - Patches formidable by copying entire `src/` directory to `dist/`
  - Includes infinite loop prevention for postinstall contexts
  - Provides detailed logging and error reporting

### Test and Verification
- **`scripts/test-netlify-patches.js`** - Verification script that tests patches
  - Verifies all required files exist
  - Tests that require paths will work
  - Simulates Netlify build process
  - Provides comprehensive test reporting

### Package Configuration
- **`netlify/functions/package.json`** - Dependencies for Netlify functions
  - Includes postinstall hook that runs the patcher
  - Contains all required dependencies (bcryptjs, formidable, axios, node-cache)
  - Prevents infinite loops with lifecycle event detection

### Build Scripts (package.json)
- **`build:netlify-functions`** - Builds and patches Netlify functions
- **`patch-netlify-deps`** - Runs patcher manually
- **`build`** - Main build script that includes function patching

### Netlify Configuration
- **`netlify.toml`** - Updated to include function patching in build process

## Usage

### Automatic (Recommended)

The patches are applied automatically during:

```bash
# Netlify deployment (automatic)
npm run build

# Local development
npm run dev
npx netlify dev

# Manual function build
npm run build:netlify-functions
```

### Manual Patching

If you need to apply patches manually:

```bash
# Run patcher directly
npm run patch-netlify-deps

# Or run the script directly
node scripts/patch-netlify-deps.js

# Test patches
node scripts/test-netlify-patches.js
```

### Verification

To verify patches are working:

```bash
# Run test suite
npm run test-netlify-patches

# Check Netlify dev works without errors
npx netlify dev
```

## Build Process Flow

1. **Install Dependencies**
   ```bash
   cd netlify/functions && npm install
   ```

2. **Automatic Patch Application** (via postinstall hook)
   ```bash
   node ../../scripts/patch-netlify-deps.js
   ```

3. **Patch Verification**
   - bcryptjs: `dist/bcrypt.js` exists and has content
   - formidable: `dist/index.js`, `dist/Formidable.js`, etc. exist
   - All require paths resolve correctly

4. **Function Loading**
   - Netlify CLI loads all functions without errors
   - esbuild bundling succeeds
   - Functions are ready for deployment

## Error Prevention

### Infinite Loop Protection
The patcher detects when it's running in a postinstall context and skips npm install to prevent infinite loops.

### Idempotent Operations
The patcher checks if patches are already applied and skips them, making it safe to run multiple times.

### Comprehensive Logging
All operations are logged with timestamps and status indicators for easy debugging.

## Troubleshooting

### Common Issues

1. **"Could not resolve ./dist/bcrypt.js"**
   - Run: `npm run patch-netlify-deps`
   - Verify: `node scripts/test-netlify-patches.js`

2. **"Could not resolve formidable"**
   - Run: `npm run build:netlify-functions`
   - Check: `ls netlify/functions/node_modules/formidable/dist/`

3. **Infinite loop during build**
   - Check if postinstall hook is calling npm install
   - Verify lifecycle event detection in patcher

### Debug Commands

```bash
# Check patch status
node scripts/test-netlify-patches.js

# Manually apply patches
node scripts/patch-netlify-deps.js

# Test Netlify functions
npx netlify dev

# Check function loading
curl http://localhost:8888/.netlify/functions/provision
```

## Production Deployment

### Netlify
The build system is fully integrated with Netlify deployment:

1. Netlify runs the build command from `netlify.toml`
2. Build command includes `npm run build:netlify-functions`
3. Patches are applied automatically during function dependency installation
4. All functions load without errors

### Other Platforms
For deployment to other platforms (Vercel, AWS Lambda, etc.), run:

```bash
npm run build:netlify-functions
```

This ensures all patches are applied before deployment.

## Maintenance

### Adding New Patches
To add patches for additional packages:

1. Add patch logic to `scripts/patch-netlify-deps.js`
2. Add verification to `scripts/test-netlify-patches.js`
3. Test with `npm run build:netlify-functions`

### Updating Dependencies
When updating bcryptjs or formidable versions:

1. Test that patches still work: `npm run test-netlify-patches`
2. Update patch logic if package structure changes
3. Verify Netlify dev works: `npx netlify dev`

## Success Indicators

✅ **All functions load without errors in `npx netlify dev`**
✅ **No "Could not resolve" errors in build output**
✅ **Test script passes: `node scripts/test-netlify-patches.js`**
✅ **Provision endpoint responds: `curl http://localhost:8888/.netlify/functions/provision`**

The build system ensures reliable, automated deployment of KNIRVGATEWAY with full Netlify Functions support.
