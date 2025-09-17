# KNIRVORACLE Viper Configuration Audit

## Executive Summary

This audit identifies multiple Viper configuration instances across the KNIRVORACLE codebase, revealing inconsistent configuration management patterns that need standardization. The analysis shows fragmented configuration approaches with different environment variable prefixes, multiple Viper instances, and inconsistent configuration loading strategies.

## Current Configuration Issues

### 1. Multiple Viper Instances
- **Primary Instance**: `config/viper_loader.go` - Main Oracle configuration
- **Secondary Instance**: `config_integration.go` - ConfigManager wrapper
- **External Instance**: `KNIRVCLI/config/config.go` - CLI-specific configuration
- **Issue**: No centralized configuration management, potential conflicts

### 2. Inconsistent Environment Variable Prefixes
- **KNIRVORACLE**: Uses `agent_` prefix (e.g., `agent_HTTPPORT=5001`)
- **KNIRVCLI**: Uses default Viper behavior
- **Economics Service**: Uses custom prefixes (`ECONOMICS_PORT`, `NRN_CONTRACT`, etc.)
- **Embedded Services**: Uses service-specific variables (`HTTP_API_PORT`, `PORT`)
- **Target**: Standardize to `KNIRV_` prefix across all components

### 3. Configuration File Strategy Inconsistencies
- **KNIRVORACLE**: Uses `default_config.json` + `[role]_config.json` strategy
- **KNIRVCLI**: Uses `.knirv` config file in home directory
- **ConfigManager**: Uses single config file approach
- **Issue**: Different components expect different file formats and locations

## Detailed Audit Findings

### Primary Configuration System (KNIRVORACLE)

**File**: `config/viper_loader.go`
- **Viper Instance**: Creates new instance with `viper.New()`
- **Environment Prefix**: `agent` (line 96)
- **Environment Binding**: 
  - `v.SetEnvPrefix("agent")`
  - `v.AutomaticEnv()`
  - `v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))`
  - Explicit bindings for Cerebras config
- **Config Files**: 
  - `default_config.json` (base configuration)
  - `[role]_config.json` (role-specific overrides)
- **Search Paths**: Current dir, user config dir, `/etc/KNIRVORACLE`, role data dir

**Environment Variables Currently Used**:
```
agent_HTTPPORT=5001
agent_ROLES_CREATOR_HTTPPORT
DEFAULT_CEREBRAS_API_KEY
DEFAULT_CEREBRAS_BASE_URL
```

### Secondary Configuration System (ConfigManager)

**File**: `config_integration.go`
- **Viper Instance**: Creates wrapper around Viper
- **Purpose**: Provides additional configuration management layer
- **Issue**: Duplicates functionality from primary system
- **Environment Variables**: Uses default Viper behavior

### CLI Configuration System

**File**: `KNIRVCLI/config/config.go`
- **Viper Instance**: Separate instance for CLI operations
- **Config File**: `.knirv` in user home directory
- **Environment Variables**: No explicit prefix set
- **Issue**: Completely separate from main Oracle configuration

### Service-Specific Configuration

**Economics Service** (`economics/service.go`):
```go
ECONOMICS_PORT
NRN_CONTRACT
XION_RPC
KNIRVCHAIN_URL
KNIRVNEXUS_URL
KNIRVORACLE_URL
```

**Embedded Node.js Services**:
```go
HTTP_API_PORT
PORT
NODE_ENV=production
```

## Configuration Loading Patterns

### 1. File-Based Configuration
- **Primary**: JSON files with role-based overrides
- **CLI**: YAML/JSON files in user directory
- **ConfigManager**: Single file approach

### 2. Environment Variable Patterns
- **Inconsistent Prefixes**: `agent_`, `ECONOMICS_`, `KNIRV_`, none
- **Mixed Naming**: Some use underscores, others use mixed case
- **No Standardization**: Each component uses different patterns

### 3. Configuration Merging
- **KNIRVORACLE**: Uses `v.MergeInConfig()` for role-specific overrides
- **Others**: Simple file loading without merging

## Identified Problems

### 1. Configuration Conflicts
- Multiple Viper instances can lead to conflicting configurations
- Different environment variable prefixes cause confusion
- No centralized validation or consistency checking

### 2. Maintenance Complexity
- Changes require updates across multiple configuration systems
- No single source of truth for configuration schema
- Difficult to track which environment variables are actually used

### 3. Development Experience Issues
- Developers must learn multiple configuration patterns
- Environment setup requires knowledge of multiple prefixes
- Configuration debugging is complex due to multiple sources

### 4. Deployment Challenges
- Different components require different environment variable sets
- No unified configuration validation
- Difficult to ensure consistent configuration across environments

## Recommended Standardization Strategy

### 1. Unified Environment Variable Prefix
**Target**: `KNIRV_` prefix for all components
**Examples**:
```
KNIRV_HTTP_PORT=5001
KNIRV_P2P_PORT=9090
KNIRV_CHAIN_ID=KNIRVORACLE
KNIRV_ECONOMICS_PORT=8090
KNIRV_NETWORK_MONITOR_PORT=8091
KNIRV_CEREBRAS_API_KEY=...
KNIRV_CEREBRAS_BASE_URL=...
```

### 2. Centralized Configuration Management
- Single `UnifiedConfigManager` replacing multiple Viper instances
- Component-based configuration registration
- Unified validation and hot-reload capabilities

### 3. Standardized Configuration Files
- Consistent file format (JSON) across all components
- Unified search path strategy
- Role-based configuration inheritance

### 4. Environment Variable Mapping
- Automatic mapping from `KNIRV_` prefixed variables
- Consistent naming conventions (snake_case)
- Clear documentation of all supported variables

## Migration Requirements

### Phase 1: Audit Completion ✅
- [x] Identify all Viper instances
- [x] Document current environment variable usage
- [x] Map configuration file strategies

### Phase 2: UnifiedConfigManager Implementation
- [ ] Create centralized configuration manager
- [ ] Implement component registration system
- [ ] Add configuration validation framework

### Phase 3: Environment Variable Standardization
- [ ] Migrate all environment variables to `KNIRV_` prefix
- [ ] Update all service configurations
- [ ] Create migration documentation

### Phase 4: Component Integration
- [ ] Integrate all services with unified configuration
- [ ] Remove duplicate configuration systems
- [ ] Implement hot-reload capabilities

### Phase 5: Validation and Testing
- [ ] Comprehensive configuration testing
- [ ] Migration validation
- [ ] Documentation updates

## Impact Assessment

### High Impact Changes
- Environment variable prefix changes (requires deployment updates)
- Configuration file format standardization
- Service configuration integration

### Medium Impact Changes
- Viper instance consolidation
- Configuration validation implementation
- Hot-reload capability addition

### Low Impact Changes
- Documentation updates
- Development tooling improvements
- Configuration debugging enhancements

## Success Metrics

### 1. Configuration Consistency
- Single environment variable prefix across all components
- Unified configuration file format and location strategy
- Centralized configuration validation

### 2. Developer Experience
- Single configuration system to learn and maintain
- Clear documentation of all configuration options
- Simplified environment setup process

### 3. Operational Excellence
- Consistent configuration across all deployment environments
- Automated configuration validation
- Hot-reload capabilities for configuration changes

## Next Steps

1. **Implement UnifiedConfigManager** - Create centralized configuration management
2. **Standardize Environment Variables** - Migrate to `KNIRV_` prefix
3. **Component Integration** - Integrate all services with unified system
4. **Validation Framework** - Implement comprehensive configuration validation
5. **Documentation** - Create unified configuration documentation

This audit provides the foundation for implementing Phase 5 of the KNIRVORACLE refactor plan, ensuring consistent and maintainable configuration management across the entire system.
