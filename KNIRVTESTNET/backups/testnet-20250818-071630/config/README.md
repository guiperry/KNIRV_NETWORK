# KNIRVTESTNET Configuration Files

This directory contains all configuration files for the KNIRVTESTNET deployment. Configuration has been consolidated from various locations into this centralized config folder.

## Configuration Files

### Core Configuration
- **`testnet-config.yaml`** - Main testnet configuration parameters
- **`ports-config.yaml`** - Port assignments for all services
- **`test-config.yaml`** - Test environment configuration
- **`endpoints.yaml`** - Production endpoint configuration

### Portal Configuration
- **`portal-config.json`** - Portal-specific settings
- **`portal-links.yaml`** - Navigation links configuration

### Service-Specific Configuration
- **`knirvchain-config.toml`** - KNIRVCHAIN configuration
- **`knirvgraph-config.yaml`** - KNIRVGRAPH configuration
- **`knirvnexus-config.yaml`** - KNIRVNEXUS configuration
- **`knirvrouter-config.yaml`** - KNIRVROUTER configuration
- **`knirvroot-config.yaml`** - KNIRVROOT configuration
- **`knirvsdk-config.yaml`** - KNIRV SDK configuration
- **`knirvwallet-config.json`** - KNIRV Wallet configuration

## Configuration Loading

Configuration is loaded using the `ConfigLoader` class in `scripts/config-loader.js`:

```javascript
const ConfigLoader = require('./scripts/config-loader');
const loader = new ConfigLoader('testnet');

// Load specific configurations
const testnetConfig = loader.getTestnetConfig();
const portsConfig = loader.getPortsConfig();
const endpoints = loader.getServiceEndpoints();
```

## Environment Variables vs Configuration

**Environment Variables** (`.env.testnet`):
- Runtime environment settings (`NODE_ENV`, `PORT`, `HOST`)
- Database connection strings (`DATABASE_URL`, `REDIS_URL`)
- Security secrets (`JWT_SECRET`)
- **NO endpoints, ports, or configuration parameters**

**Configuration Files** (this folder):
- Service endpoints (`config/testnet-config.yaml`)
- Port assignments (`config/ports-config.yaml`)
- Application parameters and feature flags
- Service settings and default values
- **ALL configuration parameters**

## Usage

### Load Configuration
```bash
npm run config:show              # Show all config
npm run config:show:testnet      # Show testnet config
npm run config:show:production   # Show production config
```

### Validate Configuration
```bash
npm run config:validate          # Validate all config files
```

### Load Endpoints
```bash
npm run load-endpoints:testnet   # Load testnet endpoints
npm run load-endpoints:production # Load production endpoints
```

## Configuration Structure

### testnet-config.yaml
```yaml
endpoints:
  knirvchain: "http://localhost:8080"
  knirvgraph: "http://localhost:8081"
  # ... other endpoints

testnet:
  mode: true
  debug_mode: true
  # ... testnet settings

server:
  enable_cors: true
  host: "0.0.0.0"
  # ... server settings
```

### ports-config.yaml
```yaml
core_services:
  knirvchain: 8080
  knirvgraph: 8081
  # ... service ports

gateway:
  main_port: 10000
  api_port: 8084
  # ... gateway ports
```

## Migration Notes

The following files were consolidated into this config folder:

- `ports.config` → `ports-config.yaml`
- `test.env` → `test-config.yaml`
- Configuration parameters from `.env.testnet` → `testnet-config.yaml`
- **All endpoints removed from `.env.testnet`** → `testnet-config.yaml`
- **All ports removed from `.env.testnet`** → `ports-config.yaml`

Environment variables in `.env.testnet` now contain **ONLY**:
- Runtime environment settings
- Database connection strings
- Security secrets

## Validation

All configuration files are validated for:
- YAML/JSON syntax
- Required sections
- Port conflicts
- Endpoint availability
- Configuration completeness

Run `npm run config:validate` to check all configurations.

## Best Practices

1. **Separate Concerns**: Keep environment variables separate from configuration parameters
2. **Use YAML**: Prefer YAML for configuration files (better readability)
3. **Validate**: Always validate configuration after changes
4. **Document**: Update this README when adding new config files
5. **Version Control**: All config files are safe to commit (no secrets)

## Security

- No sensitive information (passwords, keys) in config files
- Secrets are in environment variables only
- Default values are safe for development
- Production secrets must be set via environment variables
