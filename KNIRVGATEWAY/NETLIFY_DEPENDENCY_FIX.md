# Netlify Function Dependencies Fix

## Problem

The Netlify deploy was failing with the error:
```
A Netlify Function is using "@supabase/supabase-js" but that dependency has not been installed yet.
```

This occurred because Netlify Functions with their own `package.json` files don't automatically install dependencies during the build process.

## Root Cause

1. **Function-specific dependencies**: The `netlify/functions/package.json` file contains dependencies like `@supabase/supabase-js` that are needed by the Discourse forum functions.

2. **Netlify build process**: By default, Netlify doesn't automatically install dependencies for functions that have their own `package.json` files.

3. **Conditional imports**: Even though the Supabase import was conditional (only when `DB_TYPE=supabase`), Netlify's bundling process still tried to resolve the dependency.

## Solutions Implemented

### 1. Added Netlify Plugin

**File**: `netlify.toml`
```toml
[[plugins]]
  package = "@netlify/plugin-functions-install-core"
```

This plugin automatically installs dependencies for Netlify Functions that have their own `package.json` files.

### 2. Added Dependencies to Main Package.json

**File**: `package.json`
```json
{
  "dependencies": {
    "@supabase/supabase-js": "^2.39.0",
    "formidable": "^3.5.1",
    "jsonwebtoken": "^9.0.2",
    "nodemailer": "^6.9.8",
    "uuid": "^9.0.1",
    "mime-types": "^2.1.35",
    "sharp": "^0.33.0",
    "markdown-it": "^14.0.0",
    "dompurify": "^3.0.7",
    "jsdom": "^23.0.1"
  }
}
```

This ensures that all function dependencies are available at the project root level as a fallback.

### 3. Enhanced Build Process

**File**: `package.json`
```json
{
  "scripts": {
    "build": "npm run check-health && npm install && npm run check-function-deps --fix && npm run build:nexus && echo 'KNIRV Gateway built with Netlify Functions support'",
    "install-function-deps": "cd netlify/functions && npm install",
    "check-function-deps": "node scripts/check-function-deps.js"
  }
}
```

**File**: `netlify.toml`
```toml
[build]
  command = "npm run smart-build-with-apps && cd netlify/functions && npm install"
```

### 4. Improved Error Handling

**File**: `netlify/functions/discourse-utils.js`
```javascript
if (this.dbType === 'supabase') {
    try {
        const { createClient } = require('@supabase/supabase-js');
        this.supabaseClient = createClient(
            config.get('SUPABASE_URL'),
            config.get('SUPABASE_ANON_KEY')
        );
    } catch (error) {
        console.warn('Supabase client not available, falling back to JSON database');
        this.dbType = 'json';
        this.supabaseClient = null;
    }
}
```

This gracefully handles cases where Supabase dependencies aren't available and falls back to JSON database mode.

### 5. Dependency Checker Script

**File**: `scripts/check-function-deps.js`

A comprehensive script that:
- Verifies function `package.json` exists and has required dependencies
- Checks that `node_modules` contains required packages
- Tests that Supabase can be imported successfully
- Provides auto-fix functionality
- Gives detailed error messages and suggestions

## Testing the Fix

### Local Testing
```bash
# Check function dependencies
npm run check-function-deps

# Install function dependencies manually
npm run install-function-deps

# Run full build process
npm run build
```

### Netlify Deploy Testing
The build process now includes:
1. Health checks
2. Main dependency installation
3. Function dependency verification and installation
4. NEXUS portal build
5. Comprehensive error reporting

## Verification Steps

1. **Check Plugin Installation**:
   ```bash
   npm list @netlify/plugin-functions-install-core
   ```

2. **Verify Function Dependencies**:
   ```bash
   cd netlify/functions
   npm list @supabase/supabase-js
   ```

3. **Test Function Import**:
   ```bash
   node -e "const { createClient } = require('./netlify/functions/node_modules/@supabase/supabase-js'); console.log('Success');"
   ```

## Fallback Options

If the primary solutions don't work, these alternatives are available:

### Option A: Remove Function Package.json
Remove `netlify/functions/package.json` and rely entirely on main `package.json`.

### Option B: Manual Installation in Build
Add explicit installation commands to the build process:
```bash
cd netlify/functions && npm install && cd ../..
```

### Option C: Environment Variable Override
Set `DB_TYPE=json` to force JSON database mode and avoid Supabase entirely.

## Monitoring

The build process now includes comprehensive logging:
- Dependency installation status
- Function health checks
- Error reporting with suggested fixes
- Performance metrics

## Future Improvements

1. **Dependency Optimization**: Consider consolidating dependencies to reduce bundle size
2. **Caching**: Implement dependency caching for faster builds
3. **Testing**: Add automated tests for function dependencies
4. **Documentation**: Maintain up-to-date dependency documentation

## Related Files

- `netlify.toml` - Netlify configuration with plugin
- `package.json` - Main project dependencies and scripts
- `netlify/functions/package.json` - Function-specific dependencies
- `netlify/functions/discourse-utils.js` - Database abstraction with error handling
- `scripts/check-function-deps.js` - Dependency verification script

## Support

If you encounter dependency issues:

1. Run `npm run check-function-deps` for diagnostics
2. Check the build logs for specific error messages
3. Verify that all required dependencies are listed in both package.json files
4. Ensure the Netlify plugin is properly configured
5. Test locally with `netlify dev` before deploying
