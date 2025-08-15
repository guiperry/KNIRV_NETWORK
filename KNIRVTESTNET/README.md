# KNIRV Testnet

## Overview
This directory contains the KNIRV testnet implementation, which provides a standalone testing environment that mirrors production functionality.

## Integration with Main Test Suite

The testnet is now fully integrated with the main KNIRV integration test suite:

1. **Test Execution**:
   - Can be run standalone via `./test-integration.sh`
   - Automatically executed as part of the full test suite via `../scripts/run-integration-tests.sh`

2. **Test Reports**:
   - Generates JSON reports in `test-report.json` when run from main suite
   - Includes test counts and success/failure status
   - Reports are stored in the testnet directory

3. **Test Coverage**:
   - Service health endpoints
   - Testnet-specific functionality  
   - Authentication system
   - Cross-service communication
   - Gateway proxy functionality
   - Service discovery

## Running Tests

### Standalone Mode
```bash
./test-integration.sh [options]
```

Options:
- `-v/--verbose`: Show detailed output
- `-t/--timeout SEC`: Set HTTP timeout (default: 10s)

### Integrated Mode
The testnet tests run automatically after the main integration tests complete successfully.

## Configuration

### Environment Variables
- `TESTNET_MODE`: Set to "true" for testnet-specific behavior
- `TESTNET_TIMEOUT`: Override default timeout (seconds)

### Port Configuration
Edit `ports.config` to change service ports.

## Troubleshooting

1. **Services not responding**:
   - Run `./health-check.sh` to verify services
   - Check logs in `./logs/` directory
   - Restart services: `./stop-testnet.sh && ./start-testnet.sh`

2. **Test failures**:
   - Increase timeout with `-t` option
   - Run with `-v` for verbose output
   - Validate config with `./validate-config.sh`

## Next Steps

- Implement Gemini/Cerebras API comparison
- Add performance benchmarking
- Expand test coverage for edge cases
