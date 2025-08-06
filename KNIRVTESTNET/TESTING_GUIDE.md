# KNIRV Testnet Testing Guide

🧪 **Comprehensive Testing Documentation for KNIRV Testnet**

This guide provides detailed testing procedures, expected outcomes, and troubleshooting steps for the KNIRV testnet environment.

## 🚀 Quick Testing Workflow

### 1. Pre-Flight Checks
```bash
# Validate system dependencies
./validate-config.sh

# Check port availability
./validate-config.sh --verbose
```

### 2. Start and Validate
```bash
# Start all services
./start-testnet.sh

# Wait for services to be ready (automatic in start script)
# Verify health
./health-check.sh
```

### 3. Run Tests
```bash
# Run comprehensive integration tests
./test-integration.sh

# Monitor services during testing
./health-check.sh --watch
```

## 📋 Test Categories

### 1. System Validation Tests

#### Dependency Check
**Purpose**: Verify all required dependencies are installed
**Command**: `./validate-config.sh`
**Expected Results**:
- ✅ Go 1.19+ detected
- ✅ Rust/Cargo detected  
- ✅ Node.js 18+ detected
- ✅ Python 3.8+ detected
- ✅ All ports available (1317, 8080, 8081, 8082, 5001, 8888)

#### Configuration Validation
**Purpose**: Ensure all services are properly configured for testnet
**Command**: `./validate-config.sh --verbose`
**Expected Results**:
- ✅ All binary files exist in `bin/` directory
- ✅ All configuration files exist with correct testnet settings
- ✅ Directory structure is complete
- ✅ Environment variables are properly set

### 2. Service Health Tests

#### Individual Service Health
**Purpose**: Verify each service starts and responds to health checks
**Command**: `./health-check.sh`
**Expected Results**:
```
Service Status:
  ✓ KNIRV-ROOT      HEALTHY    PID:12345  45ms http://localhost:1317/health
  ✓ KNIRVCHAIN      HEALTHY    PID:12346  32ms http://localhost:8080/health
  ✓ KNIRVGRAPH      HEALTHY    PID:12347  28ms http://localhost:8081/health
  ✓ KNIRV-NEXUS     HEALTHY    PID:12348  41ms http://localhost:8082/health
  ✓ KNIRV-ROUTER    HEALTHY    PID:12349  35ms http://localhost:5001/health
  ✓ KNIRV-GATEWAY   HEALTHY    PID:12350  22ms http://localhost:8888/gateway/health

Overall Status: All services are healthy (6/6)
```

#### Continuous Health Monitoring
**Purpose**: Monitor service stability over time
**Command**: `./health-check.sh --watch --detailed`
**Expected Results**:
- Services remain healthy over extended periods
- Response times stay consistent
- No service crashes or restarts

### 3. Integration Tests

#### Service Discovery
**Purpose**: Verify gateway can discover all services
**Command**: `./test-integration.sh`
**Test**: Gateway service discovery
**Expected Results**:
- ✅ Gateway discovers knirvroot
- ✅ Gateway discovers knirvchain
- ✅ Gateway discovers knirvgraph
- ✅ Gateway discovers knirvnexus
- ✅ Gateway discovers knirvrouter

#### Authentication System
**Purpose**: Test simplified testnet authentication
**Command**: `./test-integration.sh`
**Test**: Authentication system
**Expected Results**:
- ✅ Testnet authentication tokens available
- ✅ Token validation works correctly

#### Cross-Service Communication
**Purpose**: Verify services can communicate with each other
**Command**: `./test-integration.sh`
**Test**: Cross-service communication
**Expected Results**:
- ✅ KNIRVCHAIN mock LLM validation works
- ✅ KNIRVCHAIN mock skill validation works
- ✅ KNIRV-NEXUS TEE simulation works

### 4. Testnet-Specific Feature Tests

#### Mock LLM Validation
**Purpose**: Test KNIRVCHAIN mock LLM validation endpoint
**Manual Test**:
```bash
curl -X POST http://localhost:8080/testnet/llm/validate \
  -H "Content-Type: application/json" \
  -d '{"model_id":"test-model"}'
```
**Expected Response**:
```json
{
  "success": true,
  "model_id": "test-model",
  "accuracy": 0.92,
  "latency_ms": 45,
  "throughput_tokens_per_sec": 120,
  "validation_result": "Mock validation completed"
}
```

#### Mock Skill Validation
**Purpose**: Test KNIRVCHAIN mock skill validation endpoint
**Manual Test**:
```bash
curl -X POST http://localhost:8080/testnet/skill/validate \
  -H "Content-Type: application/json" \
  -d '{"skill_id":"test-skill","skill_code":"console.log(\"test\")"}'
```
**Expected Response**:
```json
{
  "success": true,
  "skill_id": "test-skill",
  "validation_passed": true,
  "execution_time_ms": 150,
  "test_results": {
    "passed": 9,
    "failed": 1,
    "total": 10
  }
}
```

#### TEE Simulation
**Purpose**: Test KNIRV-NEXUS TEE simulation endpoint
**Manual Test**:
```bash
curl -X POST http://localhost:8182/testnet/validate/skill \
  -H "Content-Type: application/json" \
  -d '{
    "skill_code": "test code",
    "test_cases": [
      {"input": "test", "expected": "test", "name": "basic test"}
    ]
  }'
```
**Expected Response**:
```json
{
  "valid": true,
  "proof": "a1b2c3d4...",
  "execution_time": "100ms",
  "test_results": {
    "passed": 1,
    "failed": 0,
    "total": 1
  },
  "timestamp": "2025-08-06T..."
}
```

### 5. Gateway Proxy Tests

#### Service Proxying
**Purpose**: Test gateway's ability to proxy requests to services
**Manual Tests**:
```bash
# Test proxy to KNIRV-ROOT
curl http://localhost:8888/knirvroot/health

# Test gateway endpoints
curl http://localhost:8888/gateway/health
curl http://localhost:8888/gateway/services
curl http://localhost:8888/gateway/testnet/status

# Test authentication endpoints
curl http://localhost:8888/auth/testnet-tokens
curl -H "Authorization: Bearer testnet-token-123" \
     http://localhost:8888/auth/validate
```

## 🔧 Troubleshooting Guide

### Common Test Failures

#### Service Won't Start
**Symptoms**: Health check shows service as STOPPED
**Diagnosis**:
```bash
# Check logs
tail -f logs/servicename.log

# Check if binary exists
ls -la bin/servicename

# Check port conflicts
lsof -i :PORT
```
**Solutions**:
- Rebuild the service: `./scripts/build-servicename.sh`
- Kill conflicting processes: `kill $(lsof -t -i:PORT)`
- Check configuration files

#### Health Check Fails
**Symptoms**: Service shows as UNHEALTHY
**Diagnosis**:
```bash
# Test endpoint manually
curl -v http://localhost:PORT/health

# Check service logs
tail -f logs/servicename.log

# Verify process is running
ps aux | grep servicename
```
**Solutions**:
- Restart the service
- Check service configuration
- Verify dependencies are met

#### Integration Tests Fail
**Symptoms**: `./test-integration.sh` reports failures
**Diagnosis**:
```bash
# Run with verbose output
./test-integration.sh --verbose

# Check individual endpoints
curl http://localhost:8080/testnet/status
curl http://localhost:8888/gateway/services
```
**Solutions**:
- Ensure all services are healthy first
- Check service logs for errors
- Verify testnet-specific endpoints are enabled

### Performance Issues

#### Slow Response Times
**Symptoms**: Health check shows high response times (>1000ms)
**Diagnosis**:
- Check system resources: `top`, `htop`
- Monitor disk I/O: `iotop`
- Check network connectivity: `netstat -i`
**Solutions**:
- Restart services
- Check for resource constraints
- Verify no competing processes

#### Memory Issues
**Symptoms**: Services crash or become unresponsive
**Diagnosis**:
```bash
# Check memory usage
free -h
ps aux --sort=-%mem | head

# Check for memory leaks
valgrind --tool=memcheck ./bin/servicename
```
**Solutions**:
- Restart services
- Increase available memory
- Check for memory leaks in logs

## 📊 Test Metrics and Benchmarks

### Expected Performance Metrics

| Service | Startup Time | Response Time | Memory Usage |
|---------|-------------|---------------|--------------|
| KNIRV-ROOT | <10s | <100ms | <100MB |
| KNIRVCHAIN | <15s | <200ms | <200MB |
| KNIRVGRAPH | <10s | <150ms | <150MB |
| KNIRV-NEXUS | <8s | <100ms | <80MB |
| KNIRV-ROUTER | <5s | <50ms | <50MB |
| KNIRV-GATEWAY | <12s | <100ms | <100MB |

### Test Coverage

- ✅ **System Validation**: 100% (all dependencies and configuration)
- ✅ **Service Health**: 100% (all 6 services)
- ✅ **Integration**: 95% (core communication paths)
- ✅ **Testnet Features**: 90% (mock services and simulation)
- ✅ **Authentication**: 100% (simplified testnet auth)
- ✅ **Gateway Proxy**: 85% (core proxy functionality)

## 🎯 Test Automation

### Continuous Testing
```bash
# Run tests every 5 minutes
watch -n 300 './test-integration.sh'

# Monitor health continuously
./health-check.sh --watch
```

### Automated Test Suite
The testnet includes automated testing that runs:
1. **Pre-startup validation** - Dependency and configuration checks
2. **Health monitoring** - Continuous service health verification
3. **Integration testing** - Cross-service communication validation
4. **Performance monitoring** - Response time and resource usage tracking

## 📝 Test Reporting

### Generate Test Report
```bash
# Run all tests and save results
./test-integration.sh --verbose > test-results.log 2>&1

# Generate health report
./health-check.sh --detailed > health-report.log

# Validate configuration
./validate-config.sh --verbose > config-validation.log
```

### Test Results Analysis
- Review logs in `logs/` directory
- Check test output files
- Monitor resource usage during tests
- Verify all endpoints respond correctly

---

**Remember**: This is a testnet environment designed for development and testing. All services include mock functionality and simplified configurations for ease of use.
