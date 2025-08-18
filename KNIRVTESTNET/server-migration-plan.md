# KNIRVTESTNET Server Migration Plan

## Overview
The KNIRVTESTNET server at port 10000 is experiencing axios corruption issues and needs to be migrated to Netlify Functions to operate as a unified API proxy through the testnet-gateway (port 8888). This will eliminate the need for a separate Express server and consolidate all routing through Netlify.

## Current Architecture Issues

### 1. Axios Corruption Problem
- **Issue**: `Cannot find module '/home/gperry/Documents/GitHub/cloud-equities/KNIRV_NETWORK/KNIRVTESTNET/node_modules/axios/dist/node/axios.cjs'`
- **Root Cause**: Missing `dist` directory in axios 1.11.0 installation
- **Impact**: Server fails to start, breaking all port 10000 routes

### 2. Dual Server Architecture Problems
- **Port 8888**: Netlify Functions (working)
- **Port 10000**: Express Server (failing)
- **Port 10001**: Health Monitor (working)
- **Issue**: Inconsistent routing, some URLs work, others don't

## Migration Strategy

### Phase 1: Immediate Axios Fix ✅ COMPLETED
**Priority**: Critical
**Timeline**: Immediate

1. **Create Strategic Repair Script** ✅ (Already created: `scripts/fix-axios-corruption.sh`)
   - Remove corrupted axios installation
   - Clear npm cache
   - Reinstall axios with proper dist files
   - Test functionality

2. **Successful Fix Applied** ✅:
   - **Root Cause**: axios@1.11.0 missing `dist/node/axios.cjs` file
   - **Solution**: Downgraded to axios@1.6.8 (known stable version)
   - **Result**: Server can now start without axios corruption errors
   - **Status**: package.json updated, fix script updated for future use

### Phase 2: Server Function Analysis
**Priority**: High
**Timeline**: 1-2 days

#### Current KNIRVTESTNET Server Functions to Migrate:

1. **Health Check Handler** (`server/handlers/health-check.js`)
   - **Function**: Comprehensive health checking for all KNIRV services
   - **Dependencies**: axios, load-endpoints script
   - **Migration Target**: `data/testnet-gateway/netlify/functions/health-check.js`

2. **Config Loader** (`server/handlers/config-loader.js`)
   - **Function**: Load and validate configuration for different environments
   - **Dependencies**: js-yaml, fs operations
   - **Migration Target**: Convert to build-time script or Netlify function

3. **API Proxy Routes** (if any)
   - **Function**: Proxy requests to backend services
   - **Migration Target**: `data/testnet-gateway/netlify/functions/api-proxy.js`

4. **Static File Serving**
   - **Function**: Serve static assets and portal files
   - **Migration Target**: Netlify static hosting + functions for dynamic content

### Phase 3: Netlify Functions Migration
**Priority**: High
**Timeline**: 2-3 days

#### 3.1 Health Check Migration
```javascript
// netlify/functions/health-check.js
exports.handler = async (event, context) => {
  // Migrate health-check.js logic here
  // Use fetch instead of axios
  // Return JSON response
}
```

#### 3.2 Config Loader Migration
**Option A**: Convert to Build Script
- Move config-loader.js to scripts/
- Run during build process
- Generate static config files

**Option B**: Netlify Function
- Create netlify/functions/config.js
- Serve configuration dynamically
- Cache responses for performance

#### 3.3 API Routing Consolidation
- Merge all API routes into existing gateway-sse.js
- Use unified routing pattern
- Maintain backward compatibility

### Phase 4: Dependency Cleanup
**Priority**: Medium
**Timeline**: 1 day

1. **Remove Express Dependencies**
   - Remove express, cors, helmet, morgan from package.json
   - Clean up server/ directory
   - Update scripts to not start Express server

2. **Update Build Scripts**
   - Modify start-testnet.sh to not start port 10000 server
   - Update health checks to use port 8888 routes
   - Remove server-specific scripts

3. **Port Consolidation**
   - All API routes → port 8888 (Netlify)
   - Health monitor → port 10001 (keep separate)
   - Remove port 10000 entirely

## Implementation Steps

### Step 1: Fix Axios (Immediate)
```bash
cd KNIRVTESTNET
chmod +x scripts/fix-axios-corruption.sh
./scripts/fix-axios-corruption.sh
```

### Step 2: Analyze Current Server Routes
```bash
# Audit all routes in server/app.js
grep -r "app\." server/
grep -r "router\." server/
```

### Step 3: Create Netlify Function Templates
```bash
mkdir -p data/testnet-gateway/netlify/functions
# Create function templates based on server routes
```

### Step 4: Migrate Functions One by One
1. Start with health-check (most critical)
2. Move config-loader
3. Migrate any remaining API routes
4. Test each migration thoroughly

### Step 5: Update All References
1. Update all scripts that reference port 10000
2. Change health check URLs to port 8888
3. Update documentation and README files

## Risk Assessment

### High Risk
- **Axios corruption**: Immediate server failure
- **Route migration**: Potential service disruption during migration

### Medium Risk
- **Configuration changes**: May affect other services
- **Dependency removal**: Could break build processes

### Low Risk
- **Port consolidation**: Netlify handles routing well
- **Function migration**: Netlify Functions are reliable

## Testing Strategy

### 1. Pre-Migration Testing
- Test axios fix thoroughly
- Verify all current routes work
- Document expected behavior

### 2. Migration Testing
- Test each migrated function individually
- Verify backward compatibility
- Load test critical endpoints

### 3. Post-Migration Testing
- Full integration testing
- Performance comparison
- Monitor for regressions

## Rollback Plan

### If Axios Fix Fails
1. Use alternative HTTP client (node-fetch)
2. Downgrade axios to known working version
3. Implement custom HTTP wrapper

### If Migration Fails
1. Keep Express server as fallback
2. Gradually migrate functions
3. Use feature flags for routing

## Success Criteria

1. ✅ Axios corruption resolved
2. ✅ All port 10000 routes migrated to port 8888
3. ✅ No service disruption during migration
4. ✅ Improved performance and reliability
5. ✅ Simplified architecture (single port for API)
6. ✅ All tests passing
7. ✅ Documentation updated

## Timeline Summary

- **Day 1**: Fix axios, analyze server routes
- **Day 2-3**: Migrate health-check and config-loader
- **Day 4-5**: Migrate remaining routes, test thoroughly
- **Day 6**: Cleanup, documentation, final testing

## Dependencies

### External
- Netlify Functions runtime
- Node.js 18+ compatibility
- Existing testnet-gateway infrastructure

### Internal
- testnet-gateway Netlify configuration
- Existing gateway-sse.js function
- Health monitor system (port 10001)

## Notes

1. **Nexus Portal Location**: Updated to `data/knirvnexus/portal`
2. **Health Checks**: Need to be updated for new nexus-portal path
3. **NANDA-ANS**: Remove all references as requested
4. **Configuration**: Config-loader can be converted to build-time script
5. **Performance**: Netlify Functions may have cold start latency but better reliability
