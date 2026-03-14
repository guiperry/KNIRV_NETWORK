# KNIRV Synchronization Manager

## Table of Contents
- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Configuration](#configuration)
- [Usage](#usage)
- [Testing](#testing)
- [Monitoring](#monitoring)
- [Production Deployment](#production-deployment)
- [Troubleshooting](#troubleshooting)
- [Synchronization Analysis Report](#synchronization-analysis-report)
- [Related Documentation](#related-documentation)
- [Support](#support)


## Overview

The KNIRV Synchronization Manager provides two key synchronization capabilities:

1. **Environment Synchronization**: Automates synchronization between KNIRVTESTNET and Production Network components, focusing on scripts and testing patterns while maintaining environment-specific configurations.

2. **Documentation Synchronization**: Maintains consistency across all KNIRV service documentation, API specifications, and generates unified API documentation for the entire network.

This document details both synchronization systems, their features, configuration, usage, and troubleshooting.


## Features

### Environment Synchronization
- Script Pattern Synchronization: Automated sync of build and test scripts
- Environment-Specific Transformations: Testnet-specific modifications during sync
- Selective Component Sync: Configurable component-based synchronization
- Retry Logic: Robust retry mechanisms with exponential backoff

### Documentation Synchronization (NEW)
- Documentation Gap Detection: Automatically scans for missing or outdated documentation
- README Synchronization: Syncs service READMEs to central documentation locations
- API Specification Sync: Distributes api.yaml files to documentation portals
- Unified API Generation: Creates a single unified OpenAPI specification from all services
- Functional Gap Analysis: Detects discrepancies between code and documentation

### Monitoring and Validation
- Real-time Metrics: Sync duration, files processed, success rates, error counts
- Health Checks: Continuous monitoring of sync service health
- Alerting System: Configurable thresholds with multiple notification channels
- Dashboard Integration: Comprehensive monitoring dashboard data

### Rollback and Recovery
- Automatic Backups: Pre-sync backup creation with timestamp management
- Rollback Mechanisms: Quick rollback to previous stable state
- Recovery Procedures: Automated recovery from failed synchronizations
- State Validation: Post-rollback state verification


## Architecture

```
KNIRVSYNC/
├── cmd/
│   └── doc-sync/           # Documentation sync command
│       └── main.go
├── internal/               # Internal packages
│   ├── sync-manager.go     # Environment sync manager
│   ├── doc-scanner.go      # Documentation gap scanner
│   ├── doc-sync.go         # Documentation sync manager
│   ├── monitor.go          # Monitoring and metrics
│   ├── rollback.go         # Rollback mechanisms
│   └── orchestrator.go     # Sync orchestration
├── config/                 # Configuration files
│   ├── sync-config.json    # Environment sync config
│   └── doc-sync-config.json # Documentation sync config
├── scripts/                # Automation scripts
│   ├── auto-sync.sh        # Automated sync execution
│   ├── rollback.sh         # Rollback script
│   ├── sync-integration.sh # Integration testing
│   ├── validate-sync.sh    # Sync validation
│   └── doc-sync.sh         # Documentation sync wrapper
├── reports/                # Sync reports and logs
├── backups/                # Backup storage
├── bin/                    # Compiled binaries
└── Makefile               # Build and sync commands
```


## Configuration

### Sync Patterns
The synchronization manager supports multiple pattern types:

- Script Patterns: Build scripts, test scripts, deployment scripts
- Test Patterns: Unit tests, integration tests, performance tests
- Component Patterns: KNIRVCONTROLLER, KNIRVCORTEX, KNIRVNEXUS

### Environment Transformations
- Build Transformations: Add testnet build tags
- Test Transformations: Modify test configurations for testnet
- Configuration Transformations: Environment-specific config updates


## Usage

### Quick Start
```bash
# Build sync manager
cd KNIRVTESTNET/sync
go build -o bin/sync-manager .

# Run synchronization
./bin/sync-manager -config sync-config.json

# Validate sync configuration
./bin/sync-manager -config sync-config.json -validate

# Run with monitoring
./bin/sync-manager -config sync-config.json -monitor
```

### Automated Synchronization
```bash
# Enable automated sync (5-minute intervals)
./scripts/auto-sync.sh start

# Stop automated sync
./scripts/auto-sync.sh stop

# Check sync status
./scripts/auto-sync.sh status
```

### Rollback Operations
```bash
# List available backups
./scripts/rollback.sh list

# Rollback to specific backup
./scripts/rollback.sh restore <backup-timestamp>

# Rollback to latest backup
./scripts/rollback.sh latest
```

## Documentation Synchronization

### Overview

The documentation synchronization system ensures that all KNIRV service documentation stays synchronized across the monorepo. It maintains consistency between:

- Individual service README.md files (source of truth for each service)
- Individual service api.yaml files (source of truth for each service API)
- Central documentation at `KNIRVGATEWAY/network-website/public/documentation/`
- KNIRVRAMP documentation at `KNIRVRAMP/src/app/documentation/`
- API documentation at `KNIRVRAMP/src/app/api-docs/`
- Unified OpenAPI specification at `KNIRVGATEWAY/config/unified-openapi.yaml`

### Documentation Sync Quick Start

```bash
cd KNIRVSYNC

# Build the documentation sync tool
make build

# Scan for documentation gaps
make scan

# Sync all documentation
make sync-all

# Sync only README files
make sync-readme

# Sync only API specifications
make sync-api

# Generate unified OpenAPI spec
make sync-unified
```

### Using the Shell Script

```bash
cd KNIRVSYNC

# Build the binary
./scripts/doc-sync.sh build

# Scan for gaps
./scripts/doc-sync.sh scan

# Sync everything
./scripts/doc-sync.sh all

# Sync only READMEs
./scripts/doc-sync.sh readme

# Sync only API specs
./scripts/doc-sync.sh api

# Generate unified API
./scripts/doc-sync.sh unified
```

### Documentation Gap Detection

The scanner detects the following types of gaps:

1. **Missing API Endpoints**: Endpoints defined in code but not documented in api.yaml
2. **Undocumented Functions**: Exported functions not mentioned in README.md
3. **Missing API Specifications**: Services with API endpoints but no api.yaml file
4. **Outdated Documentation**: Discrepancies between code and documentation

Example scan output:

```bash
$ make scan

[DOC-SYNC] Scanning for documentation gaps...
[DOC-SYNC] Scanning service: KNIRVCHAIN
[DOC-SYNC] Found 3 documentation gaps in KNIRVCHAIN
[DOC-SYNC] Scanning service: KNIRVNEXUS
[DOC-SYNC] Found 1 documentation gaps in KNIRVNEXUS
...
[DOC-SYNC] Gap report generated: reports/doc-gaps-report.md
```

### Synchronization Targets

**README Synchronization**:
- Source: `KNIRV*/README.md` (each service)
- Targets:
  - `KNIRVGATEWAY/network-website/public/documentation/[service-name].md`
  - `KNIRVRAMP/src/app/documentation/[service-name].md`

**API Specification Synchronization**:
- Source: `KNIRV*/api.yaml` (each service)
- Targets:
  - `KNIRVRAMP/src/app/api-docs/[service-name]-api.yaml`
  - `KNIRVGATEWAY/network-website/public/api-specs/[service-name]-api.yaml`

**Unified API Generation**:
- Sources: All service `api.yaml` files
- Target: `KNIRVGATEWAY/config/unified-openapi.yaml`
- Consolidates all service APIs into single OpenAPI 3.1.0 specification

### Configuration

Documentation sync is configured via `config/doc-sync-config.json`:

```json
{
  "services": [
    {
      "name": "KNIRVCHAIN",
      "path": "KNIRVCHAIN",
      "has_api": true,
      "api_spec_path": "api.yaml",
      "enabled": true
    }
    // ... more services
  ],
  "central_doc_paths": [
    "KNIRVGATEWAY/network-website/public/documentation",
    "KNIRVRAMP/src/app/documentation"
  ],
  "api_doc_paths": [
    "KNIRVRAMP/src/app/api-docs",
    "KNIRVGATEWAY/network-website/public/api-specs"
  ],
  "unified_api_path": "KNIRVGATEWAY/config/unified-openapi.yaml",
  "sync_readmes": true,
  "sync_api_specs": true,
  "generate_unified_api": true
}
```

### Sync Reports

After each sync operation, detailed reports are generated:

- **Gap Report**: `reports/doc-gaps-report.md` - Markdown report of all documentation gaps
- **Sync Report**: `reports/doc-sync-report.json` - JSON report of sync operations

Example sync report structure:

```json
[
  {
    "service": "KNIRVCHAIN",
    "sync_type": "readme",
    "source_path": "KNIRVCHAIN/README.md",
    "target_paths": [
      "KNIRVGATEWAY/network-website/public/documentation/knirvchain.md",
      "KNIRVRAMP/src/app/documentation/knirvchain.md"
    ],
    "files_updated": 2,
    "errors": [],
    "timestamp": "2025-12-08T...",
    "success": true
  }
]
```

### Automation

Add documentation sync to your CI/CD pipeline:

```bash
# In your CI script
cd KNIRVSYNC
make build
make scan              # Detect gaps
make sync-all          # Sync all documentation

# Check for failures
if [ $? -ne 0 ]; then
  echo "Documentation sync failed"
  exit 1
fi
```

### Best Practices

1. **Source of Truth**: Always update service README.md and api.yaml files directly, not the synced copies
2. **Run Scans Regularly**: Schedule periodic gap scans to catch documentation drift
3. **Review Gap Reports**: Address documentation gaps before they accumulate
4. **Sync After Updates**: Run documentation sync after updating any service documentation
5. **Version Control**: Commit both source documentation and synced copies to git

### Troubleshooting Documentation Sync

**Issue**: Gap scanner reports false positives
- **Solution**: Update exclusion patterns in config or use proper Go export conventions

**Issue**: Unified API generation fails
- **Solution**: Ensure all api.yaml files are valid OpenAPI 3.1.0 specifications

**Issue**: Sync targets don't exist
- **Solution**: Directories are created automatically; check file permissions

**Issue**: README sync overwrites manual edits
- **Solution**: Always edit source README.md files, not synced copies


## Testing

### Phase 5.1 Test Results ✅
All synchronization tests completed successfully:

- Synchronization Accuracy Tests: ✅ PASSED
- Cross-Environment Consistency Tests: ✅ PASSED
- Automated Sync Mechanism Tests: ✅ PASSED
- Monitoring System Validation Tests: ✅ PASSED
- Rollback and Recovery Tests: ✅ PASSED

### Test Execution
```bash
# Run all sync tests
cd ../tests/phase5
go test -v -run "TestSynchronizationStrategyTestSuite" .

# Run integration tests
./scripts/sync-integration.sh

# Validate sync configuration
./scripts/validate-sync.sh
```


## Monitoring

### Metrics Collected
- `sync_duration`: Time taken for synchronization operations
- `files_processed`: Number of files synchronized
- `success_rate`: Percentage of successful sync operations
- `error_count`: Number of errors encountered
- `last_sync_timestamp`: Timestamp of last successful sync

### Health Checks
- Sync service availability
- File system accessibility
- Network connectivity
- Dependency health

### Alerting
- Error rate thresholds
- Sync duration limits
- Failed sync notifications
- Recovery status updates


## Production Deployment

### Prerequisites
- Go 1.21 or higher
- Proper file system permissions
- Network access to production components
- Monitoring system integration

### Deployment Steps
1. Build sync manager binary
2. Configure sync patterns and components
3. Set up monitoring and alerting
4. Enable automated synchronization
5. Validate sync operations

### Security Considerations
- File permission validation
- Secure backup storage
- Access control for sync operations
- Audit logging for all sync activities


## Troubleshooting

### Common Issues
- Permission Denied: Check file system permissions
- Network Timeout: Verify network connectivity
- Config Validation Failed: Review sync-config.json syntax
- Backup Creation Failed: Check backup directory permissions

### Debug Mode
```bash
# Run with debug logging
./bin/sync-manager -config sync-config.json -debug

# Verbose sync operation
./bin/sync-manager -config sync-config.json -verbose
```


## Synchronization Analysis Report

### Executive Summary
This analysis identifies similarities and differences between KNIRVTESTNET and Production Network components to establish a synchronization strategy focusing on scripts and testing patterns.

### Key Findings

#### 1. Build Script Patterns
**KNIRVTESTNET Patterns:** Centralized build system, individual component builders, testnet-specific flags, cross-compilation support, unified binary output.
**Production Network Patterns:** Component-specific build systems, optimizations, and flags.
**Synchronization Opportunities:** Standardize build flag patterns, implement consistent cross-compilation, unify binary output directory structures, synchronize dependency management patterns.

#### 2. Testing Methodologies
**KNIRVTESTNET Testing:** Comprehensive test suite, category-based testing, integration testing, automated orchestration, health monitoring.
**Production Network Testing:** Component-specific test patterns and reporting.
**Synchronization Opportunities:** Standardize test categorization, implement consistent coverage reporting, unify integration test patterns, synchronize performance testing methodologies.

#### 3. Deployment and Lifecycle Management
**KNIRVTESTNET Patterns:** Lifecycle management scripts, process cleanup, deployment preparation, service orchestration with health checks.
**Production Network Patterns:** Component-specific startup scripts, individual service management, Docker/Ansible deployment.
**Synchronization Opportunities:** Standardize service lifecycle management, implement consistent health check patterns, unify deployment preparation workflows, synchronize process management approaches.

#### 4. Automation and Orchestration
**KNIRVTESTNET Automation:** Go-based orchestrator, service management automation, report generation, multi-scenario test execution.
**Production Network Automation:** Component-specific automation scripts, build automation, CI/CD integration patterns.
**Synchronization Opportunities:** Extend orchestrator patterns, standardize automation interfaces, implement consistent reporting mechanisms, unify scenario-based testing approaches.

### Recommended Synchronization Strategy

#### Phase 1: Script Pattern Standardization
1. Create unified script templates for build, test, and deployment.
2. Implement consistent naming conventions.
3. Standardize command-line interfaces and parameters.
4. Establish common utility functions and error handling.

#### Phase 2: Testing Framework Alignment
1. Implement category-based testing in all production components.
2. Standardize test reporting and coverage analysis.
3. Create unified integration test patterns.
4. Establish consistent performance benchmarking.

#### Phase 3: Automation Integration
1. Extend KNIRVTESTNET orchestrator to support production components.
2. Implement automated synchronization mechanisms.
3. Create monitoring and validation systems.
4. Establish rollback and recovery procedures.

### Implementation Priority
1. **High Priority**: Build script standardization and testing framework alignment.
2. **Medium Priority**: Deployment pattern unification and automation integration.
3. **Low Priority**: Advanced orchestration features and monitoring systems.

### Success Metrics
- Consistent script patterns across all components.
- Unified testing methodologies and reporting.
- Automated synchronization with validation.
- Reduced maintenance overhead and improved reliability.


## Related Documentation

- [Phase 5 Test Results](../tests/phase5/PHASE5_TEST_RESULTS.md)
- [Major Refactor Implementation Plan](../../MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md)
- [KNIRVTESTNET Overview](../README.md)


## Support

For issues or questions regarding the synchronization manager:
1. Check the troubleshooting section above.
2. Review sync logs in the `reports/` directory.
3. Consult the Phase 5 test documentation.
4. Examine sync configuration and patterns.

