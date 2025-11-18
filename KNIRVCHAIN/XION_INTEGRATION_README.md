# XION Payment Gateway Integration

## Overview

This implementation provides a comprehensive XION payment gateway integration that enables seamless USDC to NRN token conversions through the KNIRV network ecosystem. The integration includes Meta Accounts support, gasless transactions, and complete end-to-end payment flows.

## Architecture

### Components

1. **XION Payment Gateway** (`xion_payment_gateway.go`)
   - Handles USDC to NRN conversions
   - Supports Meta Accounts (email, social, wallet, passkey)
   - Implements gasless transactions via treasury contracts
   - Provides comprehensive payment tracking and monitoring

2. **XION Integration Service** (`xion_integration_service.go`)
   - Orchestrates complete payment flows
   - Integrates with KNIRVROUTER for NRV minting
   - Manages treasury operations via KNIRVCHAIN
   - Provides real-time monitoring and status tracking

3. **KNIRVCONTROLLER Integration** (`KNIRVCONTROLLER/src/services/AbstraxionWalletService.ts`)
   - Enhanced wallet service with XION Meta Accounts
   - React hooks for seamless UI integration
   - Direct integration with KNIRVCHAIN payment gateway
   - Support for gasless transactions and payment history

4. **Enhanced KNIRVROUTER** (`KNIRVROUTER/connectivity/proof_engine.go`)
   - Advanced NRV minting with quality assessment
   - Route quality grading (A-F) with bonuses
   - Comprehensive metadata generation
   - Integration with KNIRVCHAIN treasury

5. **Enhanced KNIRVCHAIN Treasury** (`KNIRVCHAIN/economics/api.go`)
   - Quality-based NRN minting with multipliers
   - Enhanced validation and processing
   - Integration with XION payment gateway
   - Comprehensive economic bonuses system

## Features

### ✅ Implemented Features

- **Meta Accounts Support**: Email, social, wallet, and passkey authentication
- **Gasless Transactions**: Treasury-sponsored transactions with no gas fees
- **USDC to NRN Conversion**: Seamless conversion with configurable rates
- **NRV Minting Integration**: Automatic NRV minting from KNIRVROUTER
- **Treasury Management**: Automated NRN minting via KNIRVCHAIN treasury
- **Payment Flow Monitoring**: Real-time tracking of payment status and steps
- **Quality-based Bonuses**: Route quality assessment with economic bonuses
- **Multi-component Integration**: Seamless integration across KNIRV ecosystem

### Payment Flow

1. **User Initiates Payment** (KNIRVCONTROLLER)
   - User connects XION wallet via Meta Accounts
   - Selects USDC amount for conversion
   - Initiates gasless transaction

2. **USDC Processing** (XION Payment Gateway)
   - Validates user balance and transaction limits
   - Processes USDC payment on XION network
   - Calculates corresponding NRN amount

3. **NRV Minting** (KNIRVROUTER)
   - Generates route metadata with quality assessment
   - Mints NRV tokens with enhanced metadata
   - Applies quality and certification bonuses

4. **Treasury Processing** (KNIRVCHAIN)
   - Receives NRV metadata from KNIRVROUTER
   - Validates route quality and metadata
   - Mints NRN tokens with bonuses
   - Distributes to user wallet

5. **Completion & Monitoring**
   - Real-time status updates across all components
   - Payment history tracking
   - Comprehensive logging and monitoring

## Configuration

### XION Payment Gateway Config (`config/xion_payment_config.json`)

```json
{
  "xion_payment_gateway": {
    "enabled": true,
    "chain_config": {
      "chain_id": "xion-testnet-1",
      "rpc_endpoint": "https://rpc.xion-testnet-1.burnt.com:443",
      "rest_endpoint": "https://api.xion-testnet-1.burnt.com"
    },
    "contracts": {
      "usdc_contract": "xion1usdc_contract_address",
      "nrn_contract": "xion1nrn_contract_address",
      "treasury_contract": "xion1treasury_address"
    },
    "conversion": {
      "rate": "10",
      "min_transaction_amount": "1000000",
      "max_transaction_amount": "10000000000"
    },
    "gasless": {
      "enabled": true,
      "treasury_sponsored": true
    }
  }
}
```

## API Endpoints

### Payment Gateway Endpoints

- `POST /api/payment/usdc-to-nrn` - Initiate USDC to NRN conversion
- `GET /api/payment/status/{payment_id}` - Check payment status
- `GET /api/payment/history/{user_address}` - Get payment history
- `GET /api/payment/config` - Get gateway configuration
- `GET /api/payment/rates` - Get current conversion rates
- `POST /api/payment/meta-account/connect` - Connect Meta Account
- `GET /api/payment/meta-account/balance/{address}` - Get account balance

### Example Usage

```bash
# Initiate USDC to NRN conversion
curl -X POST http://localhost:8080/api/payment/usdc-to-nrn \
  -H "Content-Type: application/json" \
  -d '{
    "user_address": "xion1user...",
    "usdc_amount": "100000000",
    "meta_account_type": "email",
    "gasless": true
  }'

# Check payment status
curl http://localhost:8080/api/payment/status/pay_123456789

# Get payment history
curl http://localhost:8080/api/payment/history/xion1user...
```

## Testing

### Run Test Suite

```bash
# Run comprehensive test suite
./test_suite.sh

# Run integration test
go run test_xion_integration.go

# Run network monitor integration demo
go run integrate_xion_with_network_monitor.go

# Run individual component tests
go test ./...
```

### Network Monitor Integration Demo

```bash
# Start the existing KNIRV Network Monitor first
cd network-monitor
./scripts/start-testnet-monitoring.sh

# In another terminal, run the XION integration demo
cd ..
go run integrate_xion_with_network_monitor.go
```

This will:
1. **Register XION services** with the existing network monitor
2. **Start collecting metrics** and sending them to Prometheus
3. **Begin health monitoring** and status reporting
4. **Demonstrate payment flows** with real-time monitoring
5. **Show integration** with existing Grafana dashboards

### Test Coverage

- Configuration validation
- Dependencies check
- Build process verification
- XION Payment Gateway functionality
- Integration service operations
- Payment flow simulation
- KNIRVCONTROLLER integration
- API endpoint validation
- Security features testing
- Monitoring capabilities

## Deployment

### Prerequisites

- Go 1.19+
- Node.js 16+ (for KNIRVCONTROLLER)
- XION testnet access
- KNIRV network components

### Setup

1. **Configure Environment**
   ```bash
   cp config/xion_payment_config.json.example config/xion_payment_config.json
   # Edit configuration with actual contract addresses and endpoints
   ```

2. **Build Components**
   ```bash
   # Build KNIRVCHAIN with XION integration
   go build -o knirvoracle .
   
   # Build KNIRVCONTROLLER
   cd ../KNIRVCONTROLLER
   npm install
   npm run build
   ```

3. **Start Services**
   ```bash
   # Start KNIRVCHAIN with XION integration
   ./knirvoracle
   
   # Start KNIRVCONTROLLER
   cd ../KNIRVCONTROLLER
   npm start
   ```

## Integration Points

### KNIRVROUTER Integration
- Enhanced NRV minting with quality assessment
- Route metadata generation with bonuses
- Integration with KNIRVCHAIN treasury

### KNIRVCHAIN Integration
- Treasury management with quality bonuses
- Enhanced economics API
- Payment gateway integration

### KNIRVCONTROLLER Integration
- XION wallet service with Meta Accounts
- React hooks for UI integration
- Payment history and monitoring

## Security Features

- **Rate Limiting**: Configurable request limits
- **Address Verification**: Wallet address validation
- **Amount Limits**: Transaction amount restrictions
- **Signature Verification**: Transaction signature validation
- **Encryption**: AES-256-GCM encryption for sensitive data

## Network Monitor Integration

### 🎯 **INTEGRATED WITH EXISTING KNIRV NETWORK MONITOR**

The XION payment gateway is **fully integrated** with the existing KNIRV Network Monitor located at `KNIRVCHAIN/network-monitor`. This provides:

#### **Unified Monitoring Dashboard**
- **Custom Go/Fyne GUI**: XION services appear in the existing network monitor GUI
- **Grafana Dashboards**: XION metrics integrated with existing Grafana setup
- **ELK Stack Integration**: XION logs flow into existing Elasticsearch/Logstash/Kibana
- **Prometheus Metrics**: XION metrics collected by existing Prometheus instance

#### **Comprehensive Metrics Collection**
- **Payment Metrics**: `xion_payments_total`, `xion_payments_successful_total`, `xion_payment_duration_seconds`
- **Flow Metrics**: `xion_payment_flows_active`, `xion_flow_step_duration_seconds`
- **NRV Metrics**: `xion_nrv_minting_total`, `xion_nrv_quality_grades_total`
- **Treasury Metrics**: `xion_treasury_mints_total`, `xion_nrn_tokens_minted_total`
- **System Metrics**: `xion_gateway_uptime_seconds`, `xion_active_connections`

#### **Automated Service Registration**
- **Auto-Discovery**: XION services automatically register with network monitor
- **Health Checks**: Integrated with existing health check system
- **Status Reporting**: Real-time status updates every 30 seconds
- **Alert Integration**: XION alerts flow through existing AlertManager

#### **Monitoring Features**
- **Payment Tracking**: Real-time payment status monitoring
- **NRV Tracking**: Route validation and quality assessment
- **Treasury Tracking**: Mint validation and balance monitoring
- **Health Checks**: Automated system health monitoring
- **Comprehensive Logging**: Detailed logs for all operations

## Troubleshooting

### Common Issues

1. **Configuration Errors**
   - Verify JSON syntax in config files
   - Check contract addresses and endpoints
   - Ensure all required sections are present

2. **Connection Issues**
   - Verify XION network connectivity
   - Check RPC endpoint availability
   - Validate contract addresses

3. **Payment Failures**
   - Check user USDC balance
   - Verify transaction limits
   - Review payment gateway logs

### Debug Mode

Enable debug logging by setting `debug_logging: true` in the configuration file.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Implement changes with tests
4. Run the test suite
5. Submit a pull request

## License

This implementation is part of the KNIRV network ecosystem and follows the project's licensing terms.
