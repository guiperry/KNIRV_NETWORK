# KNIRVCHAIN Environment Variable Migration Guide

## Overview

This guide documents the migration from legacy environment variable patterns to the standardized `KNIRV_` prefix system. All KNIRVCHAIN components now use consistent environment variable naming for improved maintainability and deployment consistency.

## Migration Summary

### Old Prefix → New Prefix
- `agent_` → `KNIRV_`
- `ECONOMICS_` → `KNIRV_ECONOMICS_`
- `HTTP_API_PORT` → `KNIRV_NODEJS_PORT`
- `NODE_ENV` → `KNIRV_NODEJS_ENV`
- Various service-specific prefixes → `KNIRV_[SERVICE]_`

## Core Configuration Variables

### HTTP and Network Ports
```bash
# Old
agent_HTTPPORT=5001
agent_P2PPORT=9090
agent_WALLETPORT=8081

# New
KNIRV_HTTP_PORT=5001
KNIRV_P2P_PORT=9090
KNIRV_WALLET_PORT=8081
```

### Chain and Network Configuration
```bash
# Old
agent_CHAINID=KNIRVCHAIN
agent_ROLES_CREATOR_HTTPPORT=5001

# New
KNIRV_CHAIN_ID=KNIRVCHAIN
KNIRV_HTTP_PORT=5001
```

### Data and Database Paths
```bash
# Old
agent_BLOCKCHAIN_DATABASE_PATH=/path/to/blockchain
agent_SEARCHABLE_DATABASE_PATH=/path/to/searchable

# New
KNIRV_BLOCKCHAIN_DATABASE_PATH=/path/to/blockchain
KNIRV_SEARCHABLE_DATABASE_PATH=/path/to/searchable
KNIRV_DATA_DIR=/path/to/data
```

## AI and Inference Configuration

### Cerebras Configuration
```bash
# Old
DEFAULT_CEREBRAS_API_KEY=your_api_key
DEFAULT_CEREBRAS_BASE_URL=https://api.cerebras.ai/v1

# New
KNIRV_CEREBRAS_API_KEY=your_api_key
KNIRV_CEREBRAS_BASE_URL=https://api.cerebras.ai/v1
```

### Multi-Model Inference
```bash
# New standardized inference variables
KNIRV_INFERENCE_ENABLED=true
KNIRV_INFERENCE_PROVIDER=cerebras
KNIRV_INFERENCE_DEFAULT_MODEL=llama3.1-8b
KNIRV_INFERENCE_MAX_TOKENS=4096
KNIRV_INFERENCE_TEMPERATURE=0.7

# Deepseek configuration
KNIRV_DEEPSEEK_API_KEY=your_deepseek_key
KNIRV_DEEPSEEK_BASE_URL=https://api.deepseek.com/v1
KNIRV_DEEPSEEK_MODEL=deepseek-chat

# Gemini configuration
KNIRV_GEMINI_API_KEY=your_gemini_key
KNIRV_GEMINI_PROJECT_ID=your_project_id
KNIRV_GEMINI_MODEL=gemini-pro
```

## Service-Specific Configuration

### Economics Service
```bash
# Old
ECONOMICS_PORT=8090
NRN_CONTRACT=0x123...
XION_RPC=https://rpc.xion.network
KNIRVCHAIN_URL=http://localhost:8080
KNIRVNEXUS_URL=http://localhost:8081
KNIRVORACLE_URL=http://localhost:8082

# New
KNIRV_ECONOMICS_PORT=8090
KNIRV_ECONOMICS_NRN_CONTRACT=0x123...
KNIRV_ECONOMICS_XION_RPC=https://rpc.xion.network
KNIRV_ECONOMICS_KNIRVCHAIN_URL=http://localhost:8080
KNIRV_ECONOMICS_KNIRVNEXUS_URL=http://localhost:8081
KNIRV_ECONOMICS_KNIRVORACLE_URL=http://localhost:8082
```

### Network Monitor
```bash
# Old
NETWORK_MONITOR_PORT=8091
NETWORK_MONITOR_INTERVAL=30

# New
KNIRV_NETWORK_MONITOR_PORT=8091
KNIRV_NETWORK_MONITOR_INTERVAL=30
```

### Node.js Services (Operator, Tunnel, Payment)
```bash
# Old
HTTP_API_PORT=3001
PORT=3001
NODE_ENV=production

# New
KNIRV_NODEJS_PORT=3001
KNIRV_NODEJS_ENV=production

# Backward compatibility maintained
PORT=3001  # Still supported
HTTP_API_PORT=3001  # Still supported
NODE_ENV=production  # Still supported
```

## MCP (Model Context Protocol) Configuration
```bash
# New MCP configuration variables
KNIRV_MCP_ENABLED=false
KNIRV_MCP_PORT=8080
KNIRV_MCP_HOST=localhost
KNIRV_MCP_MAX_CLIENTS=100
KNIRV_MCP_TIMEOUT=30
KNIRV_MCP_LOG_LEVEL=info
KNIRV_MCP_TLS_ENABLED=false
KNIRV_MCP_CERT_FILE=/path/to/cert.pem
KNIRV_MCP_KEY_FILE=/path/to/key.pem
```

## Logging and Debugging
```bash
# New logging configuration
KNIRV_LOG_LEVEL=info
KNIRV_DEBUG_ENABLED=false
KNIRV_VERBOSE_LOGGING=false
```

## Migration Steps

### 1. Update Environment Files
Update your `.env` files, deployment scripts, and configuration management:

```bash
# Example .env file migration
# Before
agent_HTTPPORT=5001
ECONOMICS_PORT=8090
DEFAULT_CEREBRAS_API_KEY=your_key

# After
KNIRV_HTTP_PORT=5001
KNIRV_ECONOMICS_PORT=8090
KNIRV_CEREBRAS_API_KEY=your_key
```

### 2. Update Docker Compose Files
```yaml
# Before
environment:
  - agent_HTTPPORT=5001
  - ECONOMICS_PORT=8090

# After
environment:
  - KNIRV_HTTP_PORT=5001
  - KNIRV_ECONOMICS_PORT=8090
```

### 3. Update Kubernetes Deployments
```yaml
# Before
env:
  - name: agent_HTTPPORT
    value: "5001"

# After
env:
  - name: KNIRV_HTTP_PORT
    value: "5001"
```

### 4. Update CI/CD Pipelines
Update your CI/CD environment variable configurations:

```bash
# GitHub Actions example
# Before
env:
  agent_HTTPPORT: 5001

# After
env:
  KNIRV_HTTP_PORT: 5001
```

## Backward Compatibility

### Transition Period
During the migration period, the following backward compatibility is maintained:

1. **Economics Service**: Old environment variables are checked as fallback
2. **Node.js Services**: Both old and new variables are set
3. **Core Configuration**: Viper configuration supports both prefixes

### Deprecation Timeline
- **Phase 1** (Current): New variables introduced, old variables still supported
- **Phase 2** (Next Release): Deprecation warnings for old variables
- **Phase 3** (Future Release): Old variables removed

## Validation

### Environment Variable Checker Script
```bash
#!/bin/bash
# check_env_migration.sh

echo "Checking environment variable migration..."

# Check for old variables that should be migrated
OLD_VARS=(
    "agent_HTTPPORT"
    "ECONOMICS_PORT"
    "DEFAULT_CEREBRAS_API_KEY"
    "HTTP_API_PORT"
)

NEW_VARS=(
    "KNIRV_HTTP_PORT"
    "KNIRV_ECONOMICS_PORT"
    "KNIRV_CEREBRAS_API_KEY"
    "KNIRV_NODEJS_PORT"
)

for var in "${OLD_VARS[@]}"; do
    if [[ -n "${!var}" ]]; then
        echo "⚠️  Old variable found: $var=${!var}"
        echo "   Consider migrating to new format"
    fi
done

for var in "${NEW_VARS[@]}"; do
    if [[ -n "${!var}" ]]; then
        echo "✅ New variable found: $var=${!var}"
    fi
done
```

## Testing Migration

### 1. Test Configuration Loading
```bash
# Test that new environment variables are loaded correctly
KNIRV_HTTP_PORT=5001 KNIRV_ECONOMICS_PORT=8090 ./knirvoracle --test-config
```

### 2. Verify Service Startup
```bash
# Test that services start with new environment variables
KNIRV_NODEJS_PORT=3001 KNIRV_NODEJS_ENV=production npm start
```

### 3. Check Backward Compatibility
```bash
# Test that old variables still work during transition
ECONOMICS_PORT=8090 ./knirvoracle --test-config
```

## Troubleshooting

### Common Issues

1. **Service Not Starting**: Check that new environment variables are set
2. **Configuration Not Loading**: Verify environment variable names and values
3. **Port Conflicts**: Ensure new port variables are correctly configured

### Debug Commands
```bash
# Check current environment variables
env | grep KNIRV_

# Test configuration loading
./knirvoracle --config-debug

# Validate environment variable format
./knirvoracle --validate-env
```

## Support

For migration assistance or issues:
1. Check the configuration validation output
2. Review service logs for environment variable warnings
3. Use the environment variable checker script
4. Consult the unified configuration documentation

## Next Steps

After completing the environment variable migration:
1. Update all deployment documentation
2. Train team members on new variable names
3. Update monitoring and alerting configurations
4. Plan for removal of backward compatibility in future releases

This migration ensures consistent, maintainable configuration management across all KNIRVCHAIN components.
