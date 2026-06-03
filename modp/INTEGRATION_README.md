# KNIRVTESTNET + modp Integration

This document describes the integration between KNIRVTESTNET and the modp formal verification system.

## Overview

The integration provides formal verification capabilities for the KNIRV Network testnet through multiple approaches:

1. **Pre-deployment Verification Gate** - Verify network model before starting services
2. **Runtime Verification Service** - Continuous monitoring during testnet operation
3. **Health Monitoring Integration** - Include verification status in health checks

## Configuration

### Environment Variables

- `ENABLE_FORMAL_VERIFICATION=true` - Enable pre-deployment verification gate
- `ENABLE_RUNTIME_VERIFICATION=true` - Start verification service with testnet
- `VERIFICATION_LEVEL=standard` - Set verification thoroughness (minimal, standard, thorough)
- `MAX_DURATION=300` - Maximum time for verification tests in seconds

### Quick Start

```bash
# Start testnet with formal verification enabled
npm run testnet:start:verified

# Or manually enable verification
ENABLE_FORMAL_VERIFICATION=true ENABLE_RUNTIME_VERIFICATION=true npm run testnet:start
```

## Available Commands

### Verification Commands

```bash
# Run verification tests on demand
npm run verification:run

# Check verification system health
npm run verification:health

# Start verification service manually
npm run verification:start

# Run pre-deployment verification gate
npm run verification:verify
```

### Integration Points

#### Health Monitoring

The verification health check integrates with the existing health monitoring system:

```bash
# Check overall system health (includes verification status)
npm run check-health

# Check verification-specific health
npm run verification:health
```

#### Service Discovery

When runtime verification is enabled, the verification service appears at:

- **Health Endpoint**: http://localhost:9000/verification/status
- **Test Runner**: http://localhost:9000/verification/run-tests
- **Results**: http://localhost:9000/verification/results

## Architecture

### Components

1. **KNIRVTESTNET** - Testnet orchestration system
2. **modp** - P language formal verification framework
3. **Verification Bridge** - Node.js service connecting testnet to modp
4. **Health Monitor** - Extended to include verification status

### Data Flow

```
KNIRVTESTNET Services → Events → Verification Bridge → P Model → Invariant Checks → Alerts
```

## Verification Levels

### Minimal
- P syntax validation
- Basic component tests (Token economics)
- Duration: ~2 minutes

### Standard (Default)
- P model compilation
- Component tests (Token, Governance, Skills)
- Network composition tests
- Duration: ~5 minutes

### Thorough
- All standard tests
- Additional network-wide tests
- Malicious behavior rejection tests
- Duration: ~10+ minutes

## Integration Examples

### Pre-deployment Gate

```bash
# Block deployment if verification fails
ENABLE_FORMAL_VERIFICATION=true npm run testnet:start
```

### Runtime Monitoring

```bash
# Start verification service with testnet
ENABLE_RUNTIME_VERIFICATION=true npm run testnet:start

# Check verification status
curl http://localhost:9000/verification/status
```

### Health Check Integration

```bash
# Run health check (includes verification status)
npm run check-health

# Expected output includes:
# ✅ Verification service running (PID: 12345)
# ✅ Latest verification results: 8 passed, 0 failed
```

## Troubleshooting

### Common Issues

1. **P compiler not found**
   - System runs in simulation mode
   - Install P compiler for full verification

2. **Verification service fails to start**
   - Check Node.js dependencies
   - Ensure modp directory exists

3. **Verification timeout**
   - Increase MAX_DURATION
   - Use lower verification level

### Debugging

```bash
# Check verification service logs
tail -f ../modp/logs/verification-server.log

# Check verification test results
ls -la ../modp/results/

# Run specific test
cd ../modp && bash scripts/run-tests.sh test TokenTransferConsistency
```

## Development

### Adding New Verification Tests

1. Add P test case to `modp/tests/network_composition_tests.p`
2. Update test runner in `modp/scripts/run-tests.sh`
3. Add integration logic if needed

### Extending Event Bridge

1. Modify `KNIRVTESTNET/scripts/start-knirvverifier.sh`
2. Add event mappings in verification server
3. Update health check endpoints

## Benefits

1. **Formal Correctness** - Mathematical proof of network properties
2. **Early Detection** - Catch issues before deployment
3. **Continuous Monitoring** - Real-time invariant checking
4. **Safety Assurance** - Proof against malicious behavior
5. **Compliance** - Formal verification for regulatory requirements

## Future Enhancements

1. Real-time event bridging from testnet to P model
2. Automated test generation from network specifications
3. Integration with CI/CD pipelines
4. Visual verification dashboards
5. Formal specifications for smart contracts