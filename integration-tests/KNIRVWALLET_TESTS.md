# KNIRVWALLET Integration Tests

This document describes the comprehensive integration test suite for KNIRVWALLET functionality.

## Overview

The KNIRVWALLET integration tests validate the complete wallet functionality including wallet creation, transaction operations, NRN token management, and multi-keyring support.

## Test Categories

### 1. HD Wallet Creation and Management
- **CreateHDWalletFromMnemonic**: Tests wallet restoration from existing mnemonic
- **GenerateNewHDWallet**: Tests generation of new HD wallets with random mnemonics
- Validates proper address generation and mnemonic handling

### 2. Private Key Wallet Management
- **CreatePrivateKeyWallet**: Tests wallet creation from private key import
- Validates address derivation from private keys
- Tests keyring type assignment

### 3. Web3Auth Wallet Integration
- **CreateWeb3AuthWallet**: Tests Web3Auth wallet creation
- Validates integration with Web3Auth authentication flow
- Tests private key handling for Web3Auth wallets

### 4. Wallet Operations
- **CheckWalletBalance**: Tests balance retrieval functionality
- **GetWalletInfo**: Tests wallet information retrieval
- **FundWalletFromFaucet**: Tests wallet funding from test faucet
- Validates basic wallet operational capabilities

### 5. NRN Token Operations
- **SendNRNTokens**: Tests NRN token transfers between wallets
- **BurnNRNTokens**: Tests NRN token burning functionality
- Validates transaction confirmation and balance updates
- Tests memo and metadata handling

### 6. Transaction Signing and Broadcasting
- **SignTransaction**: Tests transaction signing capabilities
- **SignMessage**: Tests arbitrary message signing
- Validates signature generation and transaction formatting
- Tests gas limit and fee handling

### 7. Wallet Serialization and Recovery
- **SerializeWallet**: Tests wallet encryption and serialization
- **DeserializeWallet**: Tests wallet decryption and restoration
- Validates password protection and data integrity
- Tests wallet recovery from encrypted data

### 8. Multi-Keyring Support
- **CreateMultipleKeyrings**: Tests creation of different keyring types
- **ListKeyrings**: Tests keyring enumeration and management
- Validates support for HD, Private Key, Web3Auth, Ledger, and Address keyrings

## Running the Tests

### Prerequisites
- KNIRV Network services running (gateway, wallet service)
- Test environment configured with proper endpoints
- Authentication credentials available

### Execution Commands

```bash
# Run all wallet tests
./config/run-tests.sh wallet

# Run specific wallet test
go test -v -run TestKNIRVWalletIntegration

# Run with verbose output
go test -v -run TestKNIRVWalletIntegration -args -verbose
```

### Test Configuration

The tests use the following default endpoints:
- Gateway URL: `http://localhost:8000`
- Wallet URL: `http://localhost:8083`
- Test timeout: 30 seconds per operation

## Test Data

### Test Mnemonics
- Standard test mnemonic: `"abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"`
- Generated mnemonics: Created dynamically during test execution

### Test Private Keys
- Test private key: `"ea97b9fddb7e6bf6867090a7a819657047949fbb9466d617f940538efd888605"`
- Used for private key and Web3Auth wallet testing

### Test Amounts
- Faucet funding: 5-10 NRN tokens
- Transfer amounts: 1 NRN token
- Burn amounts: 0.5 NRN token

## Expected Outcomes

### Success Criteria
- All wallet types create successfully
- Address generation is consistent and valid
- Token transfers complete with proper confirmations
- Signatures are generated correctly
- Serialization/deserialization preserves wallet data
- Multi-keyring operations work seamlessly

### Failure Scenarios
- Service unavailability (handled with timeouts)
- Invalid credentials (authentication failures)
- Insufficient funds (transaction failures)
- Network connectivity issues

## Integration with Existing Tests

The KNIRVWALLET tests integrate with the existing integration test framework:
- Uses shared `TestWallet` struct (extended with `Type` field)
- Follows existing test patterns and conventions
- Integrates with test runner configuration
- Uses common authentication and setup patterns

## Cleanup

The test suite automatically cleans up created wallets and test data after execution to prevent test pollution and resource leaks.

## Troubleshooting

### Common Issues
1. **Service Connection Failures**: Ensure all KNIRV Network services are running
2. **Authentication Errors**: Verify test credentials are configured correctly
3. **Timeout Issues**: Check network connectivity and service responsiveness
4. **Balance Errors**: Ensure faucet is funded and accessible

### Debug Mode
Enable verbose logging by setting environment variables:
```bash
export KNIRV_TEST_DEBUG=true
export KNIRV_TEST_VERBOSE=true
```

## Future Enhancements

Planned additions to the test suite:
- Ledger wallet integration tests
- Cross-chain transaction tests
- Performance benchmarking
- Stress testing with multiple concurrent operations
- Advanced error scenario testing
