# KNIRV Economics Service

The KNIRV Economics Service is a comprehensive token economics system that manages the economic model for the entire KNIRV network. It handles token transactions, burn mechanisms, reward distribution, and economic metrics across all KNIRV components.

## Features

### Core Economics
- **Token Economics Management**: Unified management of NRN token economics
- **Skill Invocation Processing**: Handle token burns for skill executions
- **LLM Registration Fees**: Process registration fees for LLM models
- **Validation Rewards**: Distribute rewards for successful validations
- **Network Fee Calculation**: Dynamic fee calculation based on network conditions

### Advanced Features
- **Performance-Based Rewards**: Reward calculation based on user performance metrics
- **Burn Tracking**: Comprehensive tracking of all token burn events
- **Economic Metrics**: Real-time economic data and analytics
- **Service Integration**: Seamless integration with all KNIRV components
- **Transaction Pool**: Efficient transaction processing and management

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Economics Service                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │ Token Economics │  │ Reward Calculator│  │ Burn Tracker │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │Transaction Pool │  │ Economic Metrics│  │ Integration  │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                      REST API Layer                        │
├─────────────────────────────────────────────────────────────┤
│  KNIRVCHAIN  │  KNIRVNEXUS  │  KNIRVORACLE  │  KNIRVGRAPH    │
└─────────────────────────────────────────────────────────────┘
```

## Installation

### Prerequisites
- Go 1.21 or higher
- Access to KNIRV component services
- XION blockchain connection (optional)

### Build
```bash
cd shared-integration/economics
go build -o bin/economics-service cmd/main.go
```

### Configuration
Set environment variables:
```bash
export ECONOMICS_PORT=8090
export NRN_CONTRACT=your_nrn_contract_address
export XION_RPC=https://rpc.xion-testnet-1.burnt.com:443
export KNIRVCHAIN_URL=http://localhost:8080
export KNIRVNEXUS_URL=http://localhost:8081
export KNIRVORACLE_URL=http://localhost:8082
export KNIRVGRAPH_URL=http://localhost:8083
```

## Usage

### Starting the Service

#### Using the startup script (recommended):
```bash
./start-economics.sh
```

#### With custom configuration:
```bash
./start-economics.sh -p 8090 -d
```

#### Direct execution:
```bash
go run cmd/main.go -port 8090
```

### API Endpoints

#### Economic Operations
- `POST /economics/skill/invoke` - Process skill invocation
- `POST /economics/llm/register` - Process LLM registration
- `POST /economics/validation/reward` - Process validation reward
- `POST /economics/fees/calculate` - Calculate network fees

#### Data Retrieval
- `GET /economics/metrics` - Get economic metrics
- `GET /economics/transaction/{id}` - Get transaction details
- `GET /economics/transactions` - Get transaction list
- `GET /economics/burn/history` - Get burn event history
- `GET /economics/burn/total` - Get total burned amount

#### Configuration
- `GET /economics/rules` - Get economic rules
- `PUT /economics/rules` - Update economic rules
- `GET /economics/service/{service}/metrics` - Get service-specific metrics

#### System
- `GET /economics/health` - Health check
- `GET /economics/info` - Service information
- `GET /economics/integration/status` - Integration status

### Example API Calls

#### Process Skill Invocation
```bash
curl -X POST http://localhost:8090/economics/skill/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "skill_id": "skill456",
    "amount": "100000"
  }'
```

#### Get Economic Metrics
```bash
curl http://localhost:8090/economics/metrics
```

#### Calculate Network Fees
```bash
curl -X POST http://localhost:8090/economics/fees/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "gas_used": 21000,
    "priority": "medium"
  }'
```

## Testing

### Run API Tests
```bash
./test-economics.sh
```

### Manual Testing
1. Start the service: `./start-economics.sh`
2. Run the test suite: `./test-economics.sh`
3. Check the logs for any errors

## Economic Rules

The service uses configurable economic rules:

### Default Configuration
- **Skill Invocation Cost**: 0.1 NRN (100,000 units)
- **LLM Registration Fee**: 1 NRN (1,000,000 units)
- **Validation Reward**: 0.05 NRN (50,000 units)
- **Max Supply**: 1B NRN
- **Inflation Rate**: 5% annual

### Staking Requirements
- **Minimum Validator Stake**: 100K NRN
- **Minimum Developer Stake**: 10K NRN
- **Slashing Penalty**: 5%
- **Unbonding Period**: 21 days

### Governance Thresholds
- **Proposal Deposit**: 1K NRN
- **Voting Threshold**: 50%
- **Quorum Threshold**: 33%
- **Voting Period**: 7 days

## Integration

### Component Integration
The service automatically integrates with:
- **KNIRVCHAIN**: Skill executions and LLM registrations
- **KNIRVNEXUS**: Agent activities and validations
- **KNIRVORACLE**: Blockchain events and wallet activities
- **KNIRVGRAPH**: Network topology and connection events

### Event Processing
- Listens for events from all components
- Processes economic transactions in real-time
- Updates metrics and performance data
- Distributes rewards based on activity

## Monitoring

### Metrics Available
- Total supply and circulating supply
- Total burned tokens
- Transaction volume
- Network utilization
- Token velocity
- Service-specific economics
- Performance metrics

### Health Checks
- Service health endpoint
- Component connectivity status
- Database connectivity
- Transaction processing status

## Development

### Project Structure
```
economics/
├── cmd/
│   └── main.go              # Service entry point
├── token_economics.go       # Core economics logic
├── api.go                   # REST API handlers
├── integration.go           # Component integration
├── service.go               # Service orchestration
├── start-economics.sh       # Startup script
├── test-economics.sh        # Test script
└── README.md               # This file
```

### Adding New Features
1. Implement core logic in `token_economics.go`
2. Add API endpoints in `api.go`
3. Update integration logic in `integration.go`
4. Add tests to `test-economics.sh`

## Troubleshooting

### Common Issues

#### Service Won't Start
- Check if port 8090 is available
- Verify environment variables are set
- Check component dependencies

#### API Calls Failing
- Verify service is running: `curl http://localhost:8090/economics/health`
- Check request format and content-type
- Review service logs

#### Integration Issues
- Ensure all KNIRV components are running
- Check component URLs in configuration
- Verify network connectivity

### Logs
Service logs provide detailed information about:
- Transaction processing
- Component integration status
- Error conditions
- Performance metrics

## Contributing

1. Follow Go coding standards
2. Add tests for new features
3. Update documentation
4. Test integration with all components

## License

Part of the KNIRV Network project.
