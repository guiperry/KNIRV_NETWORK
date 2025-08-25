# Phase 5 Test Suite: Synchronization and Optimization

This directory contains comprehensive tests for Phase 5 of the KNIRV Network Major Refactor Implementation Plan, covering **Synchronization and Optimization (Weeks 17-20)**.

## Overview

Phase 5 focuses on two main areas:
- **5.1 Synchronization Strategy Refactor**: Testing synchronization between KNIRVTESTNET and Production Network
- **5.2 KNIRVCORTEX Agent-Builder Updates**: Testing TypeScript WASM compilation pipeline and LoRA adapter training

## Test Structure

```
phase5/
├── README.md                           # This file
├── package.json                        # Node.js dependencies and scripts
├── go.mod                             # Go module dependencies
├── run-phase5-tests.sh                # Main test runner script
├── synchronization-strategy.test.go   # Phase 5.1 tests
├── agent-builder-updates.test.go      # Phase 5.2 tests
├── typescript-wasm-compiler.test.ts   # TypeScript compiler tests
├── reports/                           # Test reports directory
├── logs/                             # Test logs directory
└── coverage/                         # Coverage reports directory
```

## Quick Start

### Run All Phase 5 Tests
```bash
./run-phase5-tests.sh
```

### Run Specific Test Suites
```bash
# Synchronization tests only
./run-phase5-tests.sh sync

# Agent builder tests only
./run-phase5-tests.sh agent-builder

# Integration tests only
./run-phase5-tests.sh integration

# Generate report only
./run-phase5-tests.sh report
```

### Run Individual Test Files
```bash
# Go tests
go test -v synchronization-strategy.test.go
go test -v agent-builder-updates.test.go

# TypeScript tests
npm test typescript-wasm-compiler.test.ts
```

## Test Categories

### 5.1 Synchronization Strategy Refactor Tests

#### 5.1.1 Synchronization Accuracy Tests
- **Purpose**: Verify that script and test patterns are synchronized correctly between production and testnet
- **Coverage**: File matching, content transformation, directory structure preservation
- **Key Validations**:
  - Build scripts are synchronized with testnet transformations
  - Test files maintain proper structure and content
  - Excluded files are properly filtered

#### 5.1.2 Cross-Environment Consistency Tests
- **Purpose**: Ensure synchronization produces consistent results across different environments
- **Coverage**: Multiple target environments (staging, development, testing)
- **Key Validations**:
  - Identical synchronization results across environments
  - Configuration consistency
  - File integrity verification

#### 5.1.3 Automated Sync Mechanism Tests
- **Purpose**: Validate automated synchronization triggers and retry mechanisms
- **Coverage**: Automation configuration, retry logic, health checks
- **Key Validations**:
  - Automated sync execution
  - Retry mechanism functionality
  - Health check integration
  - Failure handling

#### 5.1.4 Monitoring System Validation Tests
- **Purpose**: Test synchronization monitoring and alerting systems
- **Coverage**: Metrics collection, dashboard data, alerting thresholds
- **Key Validations**:
  - Required metrics collection (sync_duration, files_processed, success_rate, error_count)
  - Alerting system functionality
  - Monitoring dashboard data generation

#### 5.1.5 Rollback and Recovery Tests
- **Purpose**: Verify rollback and recovery mechanisms for failed synchronizations
- **Coverage**: Backup creation, rollback execution, recovery procedures
- **Key Validations**:
  - Backup creation before sync
  - Rollback to previous state
  - Recovery mechanism execution
  - State verification after rollback

### 5.2 KNIRVCORTEX Agent-Builder Updates Tests

#### 5.2.1 TypeScript Pipeline Integration Tests
- **Purpose**: Test the TypeScript WASM compilation pipeline
- **Coverage**: Template loading, TypeScript compilation, WASM generation
- **Key Validations**:
  - Pipeline initialization
  - Template processing
  - TypeScript code generation
  - WASM compilation with proper magic numbers
  - Error handling

#### 5.2.2 Pre-training Functionality Tests
- **Purpose**: Validate Tiny LLM core model pre-training
- **Coverage**: Model initialization, training process, validation
- **Key Validations**:
  - Model initialization
  - Pre-training execution
  - Model validation
  - Performance metrics

#### 5.2.3 Deployment Sequence Tests
- **Purpose**: Test optional KNIRVNEXUS deployment sequence
- **Coverage**: Deployment preparation, execution, validation
- **Key Validations**:
  - Deployment configuration
  - Deployment execution
  - Health validation
  - Optional features

#### 5.2.4 LoRA Adapter Training Tests
- **Purpose**: Validate LoRA adapter training capabilities
- **Coverage**: Training data preparation, adapter training, validation
- **Key Validations**:
  - Training dataset preparation
  - LoRA adapter configuration
  - Training execution
  - Adapter validation and integration

#### 5.2.5 End-to-End Workflow Tests
- **Purpose**: Test complete agent building workflow
- **Coverage**: Full pipeline from initialization to deployment
- **Key Validations**:
  - Complete workflow execution
  - Component integration
  - Performance validation
  - Final deployment verification

## Prerequisites

### System Requirements
- **Go**: Version 1.21 or higher
- **Node.js**: Version 18.0.0 or higher
- **npm**: Version 8.0.0 or higher

### Dependencies
The test suite will automatically install required dependencies:

**Go Dependencies:**
- `github.com/stretchr/testify` - Testing framework

**Node.js Dependencies:**
- `@jest/globals` - Jest testing framework
- `typescript` - TypeScript compiler
- `ts-jest` - TypeScript Jest preset

### Installation
```bash
# Install Go dependencies
go mod tidy

# Install Node.js dependencies
npm install
```

## Configuration

### Environment Variables
```bash
# Optional: Set custom test timeout
export PHASE5_TEST_TIMEOUT=60s

# Optional: Set custom log level
export PHASE5_LOG_LEVEL=debug

# Optional: Set custom report format
export PHASE5_REPORT_FORMAT=json
```

### Test Configuration Files
- `sync-config.json` - Synchronization patterns and components
- `jest.config.js` - Jest testing configuration
- `tsconfig.json` - TypeScript compilation settings

## Test Reports

### Report Generation
Test reports are automatically generated in the `reports/` directory:
- **Markdown Reports**: `phase5-test-report-YYYYMMDD-HHMMSS.md`
- **JSON Reports**: `phase5-test-results-YYYYMMDD-HHMMSS.json`
- **Coverage Reports**: `coverage/lcov-report/index.html`

### Report Contents
- Test execution summary
- Individual test results
- Performance metrics
- Error details and recommendations
- Coverage analysis

## Troubleshooting

### Common Issues

#### Go Module Issues
```bash
# Reset Go modules
rm go.sum
go mod tidy
```

#### Node.js Issues
```bash
# Clear npm cache
npm cache clean --force
rm -rf node_modules package-lock.json
npm install
```

#### Permission Issues
```bash
# Make test runner executable
chmod +x run-phase5-tests.sh
```

### Debug Mode
```bash
# Run with verbose output
./run-phase5-tests.sh --verbose

# Run with debug logging
PHASE5_LOG_LEVEL=debug ./run-phase5-tests.sh
```

## Integration with CI/CD

### GitHub Actions
```yaml
- name: Run Phase 5 Tests
  run: |
    cd KNIRVTESTNET/tests/phase5
    ./run-phase5-tests.sh
```

### Test Results
The test suite returns appropriate exit codes:
- `0`: All tests passed (≥80% pass rate)
- `1`: Tests failed or incomplete (<80% pass rate)

## Contributing

### Adding New Tests
1. Follow existing test patterns
2. Use descriptive test names
3. Include proper error handling
4. Add documentation for new test categories

### Test Naming Convention
- Go tests: `Test<Component><Functionality>`
- TypeScript tests: `should <expected behavior>`
- Test files: `<component>-<functionality>.test.<ext>`

## Related Documentation

- [MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md](../../../MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md) - Overall implementation plan
- [Phase 5 Requirements](../../../MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md#phase-5-synchronization-and-optimization-weeks-17-20) - Detailed Phase 5 requirements
- [Synchronization Documentation](../../sync/README.md) - Sync manager documentation
- [Agent Builder Documentation](../../../KNIRVCORTEX/agent-core/README.md) - Agent builder documentation

## Support

For issues or questions regarding Phase 5 tests:
1. Check the troubleshooting section above
2. Review test logs in the `logs/` directory
3. Examine test reports in the `reports/` directory
4. Consult the main implementation plan documentation
