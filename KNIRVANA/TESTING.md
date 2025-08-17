# KNIRVANA Testing Guide

Comprehensive testing framework for both Rust (on-device) and TypeScript (web portal) implementations.

## Quick Start

```bash
# Setup the test environment
make setup

# Run all tests
make test

# Run tests for specific implementation
make test-rust    # Rust tests only
make test-ts      # TypeScript tests only
```

## Test Structure

### Rust Client Tests (`rust-client/tests/`)
- **Unit Tests** (`unit_tests.rs`): Component and system testing
- **Integration Tests** (`integration_test.rs`): Game engine integration  
- **Performance Tests** (`performance_tests.rs`): Benchmarking and optimization
- **Mobile Tests** (`mobile_tests.rs`): Mobile-specific functionality
- **Benchmarks** (`benches/`): Criterion-based performance benchmarks

### TypeScript Client Tests (`ts-client/src/test/`)
- **Unit Tests** (`unit/`): Component and utility testing
- **Integration Tests** (`integration/`): API and service integration
- **Performance Tests** (`performance/`): Rendering and memory optimization
- **E2E Tests** (`e2e/`): Full user journey testing with Playwright

## Available Commands

### Main Test Commands
```bash
make test              # Run all tests (both implementations)
make test-unit         # Unit tests for both implementations
make test-integration  # Integration tests for both implementations
make test-performance  # Performance tests for both implementations
make test-e2e         # End-to-end tests (TypeScript only)
```

### Rust-Specific Commands
```bash
make rust-unit         # Run Rust unit tests
make rust-integration  # Run Rust integration tests
make rust-performance  # Run Rust performance tests
make rust-mobile       # Run Rust mobile-specific tests
```

### TypeScript-Specific Commands
```bash
make ts-unit           # Run TypeScript unit tests
make ts-integration    # Run TypeScript integration tests
make ts-e2e           # Run TypeScript E2E tests
make ts-performance    # Run TypeScript performance tests
```

### Code Quality Commands
```bash
make coverage          # Generate coverage reports for both implementations
make coverage-rust     # Generate Rust coverage report
make coverage-ts       # Generate TypeScript coverage report
make lint              # Run linters for both implementations
make lint-rust         # Run Rust linter (clippy)
make lint-ts           # Run TypeScript linter (eslint)
make format            # Format code for both implementations
make format-rust       # Format Rust code (rustfmt)
make format-ts         # Format TypeScript code (prettier)
```

### Development Commands
```bash
make build             # Build both implementations
make build-rust        # Build Rust client
make build-ts          # Build TypeScript client
make dev-rust          # Start Rust development build
make dev-ts            # Start TypeScript development server
make clean             # Clean build artifacts
make docs              # Generate documentation
```

## Test Categories

### Unit Tests
Test individual components, functions, and modules in isolation.

**Coverage includes:**
- Game components (Player, Agent, ErrorNode, SkillNode)
- Game systems (movement, combat, economics)
- Utility functions and helpers
- State management
- Network message handling

### Integration Tests
Test interactions between different parts of the system.

**Coverage includes:**
- Game engine initialization
- Component system interactions
- Network communication
- API integration
- Database operations
- Real-time multiplayer functionality

### Performance Tests
Measure and validate performance characteristics.

**Rust Performance Targets:**
- Game loop: <100ms for 60 frames
- Entity spawning: <10ms for 1000 entities
- Component queries: <50ms for 100 queries
- Network serialization: <20ms for 1000 messages

**TypeScript Performance Targets:**
- Component rendering: <100ms for small scenes
- State updates: <200ms for UI changes
- Memory usage: <50MB increase during normal gameplay
- Frame rate: >30 FPS sustained

### End-to-End Tests (TypeScript)
Test complete user workflows in a real browser environment.

**Coverage includes:**
- Game loading and initialization
- User interactions (keyboard, mouse, touch)
- Multiplayer game sessions
- Error handling and recovery
- Cross-browser compatibility
- Mobile responsiveness

### Mobile Tests (Rust)
Test mobile-specific features and optimizations.

**Coverage includes:**
- Touch input handling
- Battery optimization
- Performance scaling
- Memory management
- Orientation changes
- Platform-specific features

## Test Data and Fixtures

Test data is located in `test-data/`:
- `shared/`: Common test data for both implementations
- `rust/`: Rust-specific configurations
- `typescript/`: TypeScript-specific configurations
- `fixtures/`: Mock data for players, games, skills, errors
- `mocks/`: API, blockchain, and network mocks

### Setting Up Test Data
```bash
./scripts/setup-test-data.sh
```

## Configuration Files

### Rust Configuration
- `Cargo.toml`: Dependencies and test features
- Test features: `mobile`, `test-utils`
- Benchmark configuration for Criterion

### TypeScript Configuration
- `vitest.config.ts`: Unit test configuration
- `vitest.integration.config.ts`: Integration test configuration
- `vitest.performance.config.ts`: Performance test configuration
- `playwright.config.ts`: E2E test configuration
- `.eslintrc.js`: Linting rules
- `.prettierrc`: Code formatting

## Running Specific Tests

### Rust Examples
```bash
# Run specific test file
cd rust-client && cargo test unit_tests

# Run tests with specific features
cd rust-client && cargo test --features mobile

# Run benchmarks
cd rust-client && cargo bench

# Run with output
cd rust-client && cargo test -- --nocapture
```

### TypeScript Examples
```bash
# Run specific test file
cd ts-client && npm run test:unit -- types.test.ts

# Run tests in watch mode
cd ts-client && npm run test:watch

# Run with coverage
cd ts-client && npm run test:coverage

# Run E2E tests in headed mode
cd ts-client && npx playwright test --headed
```

## Continuous Integration

For CI/CD environments:
```bash
make ci-test       # Full CI test suite
```

This runs:
1. Code linting and formatting checks
2. All unit and integration tests
3. Performance validation
4. Coverage report generation

## Troubleshooting

### Common Issues

**Rust Tests:**
- Install required tools: `cargo install cargo-tarpaulin cargo-watch`
- For mobile tests: `cargo test --features mobile`
- Memory issues: Use `--release` flag for performance tests

**TypeScript Tests:**
- Ensure Node.js 18+ is installed
- Install dependencies: `npm install`
- For E2E tests: `npx playwright install`
- Browser issues: Update Playwright browsers

**Performance Tests:**
- Close other applications during testing
- Ensure adequate system resources
- Use consistent test environment

### Environment Variables

Create `test-data/.env.test`:
```bash
NODE_ENV=test
RUST_LOG=debug
RUST_BACKTRACE=1
DATABASE_URL=sqlite://test.db
NETWORK_MODE=test
ENABLE_PERFORMANCE_MONITORING=true
MOCK_BLOCKCHAIN=true
```

## Test Scripts

Located in `scripts/`:
- `setup-test-data.sh`: Initialize test data and fixtures
- `run-all-tests.sh`: Comprehensive test runner with reporting

## Dependencies

### Rust Testing Dependencies
- `tokio-test`: Async testing utilities
- `mockall`: Mocking framework
- `criterion`: Benchmarking
- `proptest`: Property-based testing
- `serial_test`: Sequential test execution

### TypeScript Testing Dependencies
- `vitest`: Test runner and framework
- `@testing-library/react`: React component testing
- `@playwright/test`: E2E testing
- `jsdom`: DOM simulation
- `msw`: API mocking

## Contributing to Tests

When adding new tests:
1. Follow existing test structure and naming conventions
2. Add appropriate test data to `test-data/`
3. Update performance thresholds if needed
4. Ensure tests pass in CI environment
5. Update documentation for new test categories

### Test Naming Conventions
- **Rust**: `test_function_name`
- **TypeScript**: `should describe what it tests`
- **Files**: `*.test.ts` for unit tests, `*.spec.ts` for E2E tests
