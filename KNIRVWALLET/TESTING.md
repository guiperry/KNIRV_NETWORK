# KNIRVWALLET Testing Documentation

## Overview

This document provides comprehensive information about the testing infrastructure for KNIRVWALLET, including unit tests, integration tests, and end-to-end tests across all platforms and components.

## Test Architecture

### Test Structure
```
KNIRVWALLET/
├── tests/
│   ├── unit/                    # Unit tests
│   │   ├── browser-wallet/      # Browser extension tests
│   │   ├── react-native/        # Mobile app tests
│   │   └── go-backend/          # Backend service tests
│   ├── integration/             # Integration tests
│   │   ├── cross-platform-sync.test.go
│   │   └── xion-integration.test.go
│   └── e2e/                     # End-to-end tests
│       ├── wallet-workflow.test.js
│       ├── playwright.config.js
│       ├── global-setup.js
│       └── global-teardown.js
├── test-utils/                  # Shared test utilities
│   ├── test-data.ts
│   ├── wallet-test-utils.ts
│   ├── crypto-test-utils.ts
│   ├── xion-test-utils.ts
│   ├── mock-services.ts
│   └── test-helpers.ts
├── jest.config.js              # Jest configuration
└── scripts/
    └── run-tests.sh            # Test runner script
```

## Test Categories

### 1. Unit Tests

#### Browser Wallet Module Tests
- **Location**: `tests/unit/browser-wallet/`
- **Framework**: Jest + Testing Library
- **Coverage**: 
  - Wallet core functionality
  - Transaction signing
  - Keyring management
  - Cryptographic operations

#### React Native Mobile Tests
- **Location**: `tests/unit/react-native/`
- **Framework**: Jest + React Native Testing Library
- **Coverage**:
  - XION meta accounts
  - Mobile UI components
  - Wallet screens
  - Cross-platform functionality

#### Go Backend Tests
- **Location**: `tests/unit/go-backend/`
- **Framework**: Go testing + Testify
- **Coverage**:
  - Multichain wallet service
  - XION integration
  - Wallet synchronization
  - API handlers

### 2. Integration Tests

#### Cross-Platform Synchronization
- **File**: `tests/integration/cross-platform-sync.test.go`
- **Coverage**:
  - QR code pairing
  - WebSocket communication
  - Wallet data synchronization
  - Session management

#### XION Blockchain Integration
- **File**: `tests/integration/xion-integration.test.go`
- **Coverage**:
  - Meta account management
  - NRN token operations
  - Skill invocation
  - Gasless transactions
  - Faucet integration

### 3. End-to-End Tests

#### Complete Wallet Workflow
- **File**: `tests/e2e/wallet-workflow.test.js`
- **Framework**: Playwright
- **Coverage**:
  - Browser extension workflow
  - Mobile app workflow
  - Cross-platform synchronization
  - Error handling
  - Performance testing
  - Security validation

## Test Utilities

### Shared Test Data
- **File**: `test-utils/test-data.ts`
- **Contents**:
  - Test mnemonics and private keys
  - Sample addresses and transactions
  - XION configuration data
  - Wallet configuration templates

### Wallet Test Utilities
- **File**: `test-utils/wallet-test-utils.ts`
- **Features**:
  - Wallet factory for test creation
  - Transaction validation helpers
  - Account and keyring utilities
  - Mock wallet services

### Cryptographic Test Utilities
- **File**: `test-utils/crypto-test-utils.ts`
- **Features**:
  - Mnemonic validation
  - Private key testing
  - Encryption/decryption testing
  - Signature verification
  - Key derivation testing

### XION Test Utilities
- **File**: `test-utils/xion-test-utils.ts`
- **Features**:
  - XION meta account mocking
  - Transaction simulation
  - Contract interaction testing
  - Gasless transaction testing

## Running Tests

### Prerequisites
```bash
# Install dependencies
cd KNIRVWALLET
npm install

# Install Go dependencies
cd agentic-wallet/go-backend
go mod download

# Install Playwright browsers
npx playwright install
```

### Running All Tests
```bash
# Run comprehensive test suite
./scripts/run-tests.sh

# Run with specific options
./scripts/run-tests.sh --unit-only
./scripts/run-tests.sh --integration-only
./scripts/run-tests.sh --e2e-only
./scripts/run-tests.sh --all --verbose
```

### Running Specific Test Categories

#### Unit Tests Only
```bash
# Browser wallet tests
cd browser-wallet
npm test

# React Native tests
cd agentic-wallet
npm test

# Go backend tests
cd agentic-wallet/go-backend
go test ./...
```

#### Integration Tests Only
```bash
cd integration-tests
go test -v ./...
```

#### End-to-End Tests Only
```bash
# All browsers
npx playwright test

# Specific browser
npx playwright test --project=chromium

# Mobile testing
npx playwright test --project="Mobile Chrome"

# Cross-platform tests
npx playwright test cross-platform
```

### Test Configuration

#### Jest Configuration
- **File**: `jest.config.js`
- **Features**:
  - Multi-project setup
  - Coverage reporting
  - Custom matchers
  - Mock configurations
  - Parallel execution

#### Playwright Configuration
- **File**: `tests/e2e/playwright.config.js`
- **Features**:
  - Multi-browser testing
  - Mobile device simulation
  - Cross-platform test projects
  - Video/screenshot capture
  - Test reporting

## Test Data and Mocking

### Mock Services
- **File**: `test-utils/mock-services.ts`
- **Services**:
  - Mock XION client
  - Mock wallet manager
  - Mock transaction service
  - Mock synchronization service

### Test Data Management
- Deterministic test data for consistent results
- Isolated test environments
- Automatic cleanup after tests
- Secure handling of test credentials

## Coverage and Reporting

### Coverage Targets
- **Unit Tests**: 80% line coverage minimum
- **Integration Tests**: Critical path coverage
- **E2E Tests**: User workflow coverage

### Report Generation
```bash
# Generate coverage report
npm run test:coverage

# Generate HTML report
npm run test:report

# View coverage in browser
open coverage/lcov-report/index.html
```

### Test Reports
- **HTML Reports**: Detailed test results with screenshots
- **JUnit XML**: CI/CD integration
- **JSON Reports**: Programmatic analysis
- **Coverage Reports**: Line and branch coverage

## Continuous Integration

### GitHub Actions Integration
```yaml
# Example CI configuration
- name: Run KNIRVWALLET Tests
  run: |
    ./scripts/run-tests.sh --all --bail
    
- name: Upload Test Results
  uses: actions/upload-artifact@v3
  with:
    name: test-results
    path: test-results/
```

### Test Environment Setup
- Automated service startup
- Database seeding
- Network configuration
- Environment variable management

## Best Practices

### Writing Tests
1. **Descriptive Test Names**: Use clear, descriptive test names
2. **Arrange-Act-Assert**: Follow AAA pattern
3. **Test Isolation**: Each test should be independent
4. **Mock External Dependencies**: Use mocks for external services
5. **Test Edge Cases**: Include error conditions and edge cases

### Test Maintenance
1. **Regular Updates**: Keep tests updated with code changes
2. **Flaky Test Management**: Identify and fix unstable tests
3. **Performance Monitoring**: Monitor test execution time
4. **Documentation**: Keep test documentation current

### Security Testing
1. **Sensitive Data**: Never use real private keys or mnemonics
2. **Input Validation**: Test all input validation
3. **Authentication**: Test authentication flows
4. **Authorization**: Verify access controls

## Debugging Tests

### Debug Configuration
```bash
# Run tests in debug mode
npm run test:debug

# Run specific test file
npm test -- wallet-core.test.ts

# Run with verbose output
npm test -- --verbose

# Run in watch mode
npm test -- --watch
```

### Common Issues
1. **Timing Issues**: Use proper waits and timeouts
2. **Mock Configuration**: Ensure mocks are properly configured
3. **Environment Setup**: Verify test environment is correct
4. **Data Cleanup**: Ensure proper test data cleanup

## Performance Testing

### Load Testing
- Transaction throughput testing
- Concurrent user simulation
- Memory usage monitoring
- Response time measurement

### Stress Testing
- High-volume transaction testing
- Resource exhaustion scenarios
- Recovery testing
- Scalability validation

## Security Testing

### Vulnerability Testing
- Input sanitization validation
- Authentication bypass attempts
- Authorization escalation testing
- Data exposure verification

### Cryptographic Testing
- Key generation validation
- Signature verification
- Encryption/decryption testing
- Random number generation testing

## Troubleshooting

### Common Test Failures
1. **Service Unavailable**: Ensure all services are running
2. **Network Timeouts**: Check network connectivity
3. **Mock Failures**: Verify mock configurations
4. **Data Conflicts**: Ensure proper test isolation

### Getting Help
- Check test logs in `test-results/`
- Review error screenshots and videos
- Consult test documentation
- Contact development team

## Contributing

### Adding New Tests
1. Follow existing test patterns
2. Add appropriate test utilities
3. Update documentation
4. Ensure CI/CD integration

### Test Review Process
1. Code review for test quality
2. Coverage impact assessment
3. Performance impact evaluation
4. Documentation updates

---

For more information, see the individual test files and their documentation comments.
