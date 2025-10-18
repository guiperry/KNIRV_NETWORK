#!/bin/bash

# KNIRV Integration Test Runner Script
# This script runs the complete integration test suite

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DIR="$PROJECT_ROOT/integration-tests"

# Default values
RUN_SETUP=true
RUN_TEARDOWN=true
TEST_PATTERN=".*"
VERBOSE=false
PARALLEL=false
TIMEOUT="600s"
GENERATE_REPORT=true

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to start real services
start_real_services() {
    print_status "Starting real KNIRV services..."

    # Kill any existing services first
    print_status "Cleaning up any existing services..."
    bash "$PROJECT_ROOT/scripts/kill_knirv.sh" || true

    # Wait a moment for cleanup
    sleep 2

    # Start KNIRVORACLE (Core Network)
    print_status "Starting KNIRVORACLE on port 1317..."
    cd "$PROJECT_ROOT/KNIRVORACLE"
    nohup go run . --role=root --port=1317 --skip-install > "$TEST_DIR/logs/KNIRVORACLE.log" 2>&1 &
    echo $! > "$TEST_DIR/logs/knirvoracle.pid"

    # Start KNIRVGRAPH (Graph Database)
    print_status "Starting KNIRVGRAPH on port 8082..."
    cd "$PROJECT_ROOT/KNIRVGRAPH"
    nohup go run cmd/node/main.go --port=8082 --testnet --memory > "$TEST_DIR/logs/knirvgraph.log" 2>&1 &
    echo $! > "$TEST_DIR/logs/knirvgraph.pid"

    # Start KNIRVCHAIN (Blockchain)
    print_status "Starting KNIRVCHAIN on port 8000..."
    cd "$PROJECT_ROOT/KNIRVCHAIN"
    nohup cargo run > "$TEST_DIR/logs/knirvchain.log" 2>&1 &
    echo $! > "$TEST_DIR/logs/knirvchain.pid"

    # Start KNIRVNEXUS (Management Portal)
    print_status "Starting KNIRVNEXUS on port 8083..."
    cd "$PROJECT_ROOT/KNIRVNEXUS"
    nohup npm run dev > "$TEST_DIR/logs/knirvnexus.log" 2>&1 &
    echo $! > "$TEST_DIR/logs/knirvnexus.pid"

    # Start KNIRVROUTER (Router)
    print_status "Starting KNIRVROUTER on port 5001..."
    cd "$PROJECT_ROOT/KNIRVROUTER"
    nohup go run . --port=5001 > "$TEST_DIR/logs/knirvrouter.log" 2>&1 &
    echo $! > "$TEST_DIR/logs/knirvrouter.pid"

    # Start KNIRVGATEWAY (Gateway) - Node.js server with Netlify Functions
    print_status "Starting KNIRVGATEWAY on port 8888..."
    cd "$PROJECT_ROOT/KNIRVGATEWAY"
    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        npm install
    fi
    nohup npm start > "$TEST_DIR/logs/knirvgateway.log" 2>&1 &
    echo $! > "$TEST_DIR/logs/knirvgateway.pid"

    print_status "Waiting for services to start..."
    print_status "This may take several minutes for Rust compilation and Node.js setup..."

    # Show startup progress by monitoring logs
    show_startup_progress() {
        local wait_time=60
        local check_interval=10
        local elapsed=0

        while [ $elapsed -lt $wait_time ]; do
            echo "=== Startup Progress (${elapsed}s/${wait_time}s) ==="

            # Check KNIRVCHAIN compilation progress
            if [ -f "$TEST_DIR/logs/knirvchain.log" ]; then
                echo "KNIRVCHAIN (Rust compilation):"
                tail -3 "$TEST_DIR/logs/knirvchain.log" 2>/dev/null | sed 's/^/  /' || echo "  No output yet..."
            fi

            # Check KNIRVGATEWAY Node.js startup
            if [ -f "$TEST_DIR/logs/knirvgateway.log" ]; then
                echo "KNIRVGATEWAY (Node.js/Netlify):"
                tail -3 "$TEST_DIR/logs/knirvgateway.log" 2>/dev/null | sed 's/^/  /' || echo "  No output yet..."
            fi

            # Check KNIRVNEXUS Node.js startup
            if [ -f "$TEST_DIR/logs/knirvnexus.log" ]; then
                echo "KNIRVNEXUS (Node.js/TypeScript):"
                tail -3 "$TEST_DIR/logs/knirvnexus.log" 2>/dev/null | sed 's/^/  /' || echo "  No output yet..."
            fi

            # Check KNIRVORACLE startup
            if [ -f "$TEST_DIR/logs/KNIRVORACLE.log" ]; then
                echo "KNIRVORACLE (Go application):"
                tail -3 "$TEST_DIR/logs/KNIRVORACLE.log" 2>/dev/null | sed 's/^/  /' || echo "  No output yet..."
            fi

            echo ""
            sleep $check_interval
            elapsed=$((elapsed + check_interval))
        done
    }

    show_startup_progress

    # Verify services are running
    verify_services_running
}

# Function to verify services are running
verify_services_running() {
    print_status "Verifying services are running..."

    local services_ok=true

    # Helper function to extract port from service logs
    get_service_port() {
        local service_name="$1"
        local log_file="$TEST_DIR/logs/$(echo $service_name | tr '[:upper:]' '[:lower:]').log"

        if [ ! -f "$log_file" ]; then
            return 1
        fi

        case "$service_name" in
            "KNIRVCHAIN")
                # Look for "Starting server at http://127.0.0.1:PORT" or similar
                grep -o "Starting server at http://[^:]*:\([0-9]*\)" "$log_file" | tail -1 | grep -o "[0-9]*$"
                ;;
            "KNIRVORACLE")
                # Look for "Starting Server on port PORT" or similar
                grep -o "Starting.*[Ss]erver.*port \([0-9]*\)" "$log_file" | tail -1 | grep -o "[0-9]*$"
                ;;
            "KNIRVGRAPH")
                # Look for port in startup messages
                grep -o "port \([0-9]*\)" "$log_file" | tail -1 | grep -o "[0-9]*$"
                ;;
            "KNIRVNEXUS")
                # Look for "Server running on port PORT" or similar
                grep -o "[Ss]erver.*port \([0-9]*\)" "$log_file" | tail -1 | grep -o "[0-9]*$"
                ;;
            "KNIRVGATEWAY")
                # Look for Netlify dev server port
                grep -o "Local:.*localhost:\([0-9]*\)" "$log_file" | tail -1 | grep -o "[0-9]*$"
                ;;
            "KNIRVROUTER")
                # Look for router startup port
                grep -o "[Ll]istening.*port \([0-9]*\)" "$log_file" | tail -1 | grep -o "[0-9]*$"
                ;;
        esac
    }

    # Helper function to get service health endpoint
    get_service_health_endpoint() {
        local service_name="$1"
        local port="$2"

        case "$service_name" in
            "KNIRVCHAIN")
                echo "http://localhost:$port/health"
                ;;
            "KNIRVORACLE")
                echo "http://localhost:$port/health"
                ;;
            "KNIRVGRAPH")
                echo "http://localhost:$port/height"
                ;;
            "KNIRVNEXUS")
                echo "http://localhost:$port/health"
                ;;
            "KNIRVGATEWAY")
                echo "http://localhost:$port/gateway/health"
                ;;
            "KNIRVROUTER")
                echo "http://localhost:$port/status"
                ;;
        esac
    }

    # Helper function to check service with retries and dynamic port discovery
    check_service_with_retries() {
        local service_name="$1"
        local fallback_url="$2"
        local max_retries=5
        local retry_delay=10

        for i in $(seq 1 $max_retries); do
            # Try to discover the actual port from logs
            local discovered_port=$(get_service_port "$service_name")
            local url="$fallback_url"

            if [ -n "$discovered_port" ]; then
                url=$(get_service_health_endpoint "$service_name" "$discovered_port")
                echo "  Discovered $service_name running on port $discovered_port"
            fi

            if curl -s "$url" > /dev/null 2>&1; then
                print_success "$service_name is running on $url"
                return 0
            fi

            if [ $i -lt $max_retries ]; then
                print_status "Waiting for $service_name (attempt $i/$max_retries)..."
                # Show recent log output for debugging
                local log_file="$TEST_DIR/logs/$(echo $service_name | tr '[:upper:]' '[:lower:]').log"
                if [ -f "$log_file" ]; then
                    echo "  Recent log output:"
                    tail -2 "$log_file" 2>/dev/null | sed 's/^/    /' || echo "    No recent output..."
                fi
                sleep $retry_delay
            fi
        done
        print_warning "$service_name health check failed after $max_retries attempts"
        # Show final log output for debugging
        local log_file="$TEST_DIR/logs/$(echo $service_name | tr '[:upper:]' '[:lower:]').log"
        if [ -f "$log_file" ]; then
            echo "  Final log output:"
            tail -5 "$log_file" 2>/dev/null | sed 's/^/    /' || echo "    No log output available..."
        fi
        return 1
    }

    # Check KNIRVORACLE health (fallback to expected port)
    if ! check_service_with_retries "KNIRVORACLE" "http://localhost:1317/health"; then
        services_ok=false
    fi

    # Check KNIRVGRAPH height endpoint (fallback to expected port)
    if ! check_service_with_retries "KNIRVGRAPH" "http://localhost:8082/height"; then
        services_ok=false
    fi

    # Check KNIRVCHAIN health (fallback to expected port)
    if ! check_service_with_retries "KNIRVCHAIN" "http://localhost:8000/health"; then
        services_ok=false
    fi

    # Check KNIRVNEXUS health (fallback to expected port)
    if ! check_service_with_retries "KNIRVNEXUS" "http://localhost:8083/health"; then
        services_ok=false
    fi

    # Check KNIRVROUTER status (fallback to expected port)
    if ! check_service_with_retries "KNIRVROUTER" "http://localhost:5001/status"; then
        services_ok=false
    fi

    # Check KNIRVGATEWAY health (fallback to expected port)
    if ! check_service_with_retries "KNIRVGATEWAY" "http://localhost:8888/gateway/health"; then
        services_ok=false
    fi

    if [ "$services_ok" = false ]; then
        print_warning "Some services failed health checks, but continuing with tests..."
    else
        print_success "All services are running and healthy!"
    fi
}

# Function to run setup
run_setup() {
    if [ "$RUN_SETUP" = true ]; then
        print_status "Running test setup..."

        # Create necessary directories
        mkdir -p "$TEST_DIR/logs"
        mkdir -p "$TEST_DIR/reports"

        # Start real services instead of using setup.sh
        start_real_services

        print_success "Setup completed"
    else
        print_status "Skipping setup"
    fi
}

# Function to stop services gracefully
stop_services_gracefully() {
    print_status "Stopping services gracefully..."

    # Stop services by PID files
    for service in knirvoracle knirvgraph knirvchain knirvnexus knirvrouter knirvgateway; do
        local pid_file="$TEST_DIR/logs/${service}.pid"
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                print_status "Stopping $service (PID: $pid)..."
                kill -TERM "$pid" 2>/dev/null || true
            fi
            rm -f "$pid_file"
        fi
    done

    # Wait for graceful shutdown
    print_status "Waiting for graceful shutdown..."
    sleep 5
}

# Function to check if services are stopped
check_services_stopped() {
    print_status "Checking if all services are stopped..."

    local services_stopped=true

    # Check each service port
    local ports=(1317 8082 8090 8083 5001 8888)
    local service_names=("KNIRVORACLE" "KNIRVGRAPH" "KNIRVCHAIN" "KNIRVNEXUS" "KNIRVROUTER" "KNIRVGATEWAY")

    for i in "${!ports[@]}"; do
        local port=${ports[$i]}
        local service=${service_names[$i]}

        if lsof -i ":$port" > /dev/null 2>&1; then
            print_warning "$service is still running on port $port"
            services_stopped=false
        else
            print_success "$service stopped successfully"
        fi
    done

    return $([ "$services_stopped" = true ] && echo 0 || echo 1)
}

# Function to force kill services
force_kill_services() {
    print_status "Force killing remaining services using kill_knirv.sh..."

    cd "$PROJECT_ROOT"
    bash "scripts/kill_knirv.sh"

    # Wait a moment and check again
    sleep 3

    if check_services_stopped; then
        print_success "All services stopped successfully"
    else
        print_error "Some services may still be running"
    fi
}

# Function to run teardown
run_teardown() {
    if [ "$RUN_TEARDOWN" = true ]; then
        print_status "Running test teardown..."

        # Step 1: Try graceful shutdown
        stop_services_gracefully

        # Step 2: Check if services are stopped
        if check_services_stopped; then
            print_success "All services stopped gracefully"
        else
            # Step 3: Force kill if needed
            print_warning "Some services didn't stop gracefully, using kill_knirv.sh..."
            force_kill_services
        fi

        print_success "Teardown completed"
    else
        print_status "Skipping teardown"
    fi
}

# Function to run JavaScript tests
run_javascript_tests() {
    print_status "Running JavaScript integration tests..."

    cd "$TEST_DIR"

    # Set test environment variables
    export KNIRV_TEST_MODE=true
    export KNIRV_TEST_CONFIG="$SCRIPT_DIR/test-config.yaml"
    export KNIRV_TEST_DATA_DIR="$TEST_DIR/data"
    export KNIRV_TEST_LOGS_DIR="$TEST_DIR/logs"
    export GATEWAY_SERVICE_URL=${GATEWAY_SERVICE_URL:-"http://localhost:8888"}

    local js_test_result=0

    # Run KNIRV GraphChain Explorer tests
    if [ -f "knirv-graphchain-explorer.test.js" ]; then
        print_status "Running KNIRV GraphChain Explorer tests..."
        if node knirv-graphchain-explorer.test.js; then
            print_success "KNIRV GraphChain Explorer tests passed"
        else
            print_error "KNIRV GraphChain Explorer tests failed"
            js_test_result=1
        fi
    fi

    # Run Portal Integration tests
    if [ -f "portal-integration.test.js" ]; then
        print_status "Running Portal Integration tests..."
        if node portal-integration.test.js; then
            print_success "Portal Integration tests passed"
        else
            print_error "Portal Integration tests failed"
            js_test_result=1
        fi
    fi

    return $js_test_result
}

# Function to run integration tests
run_integration_tests() {
    print_status "Running integration tests..."

    cd "$TEST_DIR"

    # Set test environment variables
    export KNIRV_TEST_MODE=true
    export KNIRV_TEST_CONFIG="$SCRIPT_DIR/test-config.yaml"
    export KNIRV_TEST_DATA_DIR="$TEST_DIR/data"
    export KNIRV_TEST_LOGS_DIR="$TEST_DIR/logs"
    export ECONOMICS_SERVICE_URL=${ECONOMICS_SERVICE_URL:-"http://localhost:8090"}
    export GATEWAY_SERVICE_URL=${GATEWAY_SERVICE_URL:-"http://localhost:8000"}
    
    # Prepare test command
    local test_cmd="go test"
    
    if [ "$VERBOSE" = true ]; then
        test_cmd="$test_cmd -v"
    fi
    
    if [ "$PARALLEL" = true ]; then
        test_cmd="$test_cmd -parallel 4"
    fi
    
    test_cmd="$test_cmd -timeout $TIMEOUT"
    test_cmd="$test_cmd -run $TEST_PATTERN"
    
    # Add output formatting
    if [ "$GENERATE_REPORT" = true ]; then
        mkdir -p "$TEST_DIR/reports"
        local report_file="$TEST_DIR/reports/test-results-$(date +%Y%m%d-%H%M%S).json"
        test_cmd="$test_cmd -json | tee $report_file"
    fi
    
    test_cmd="$test_cmd ./..."
    
    print_status "Executing: $test_cmd"
    
    # Run Go tests
    local go_test_result=0
    if eval "$test_cmd"; then
        print_success "Go integration tests passed!"
    else
        print_error "Some Go integration tests failed"
        go_test_result=1
    fi

    # Run JavaScript tests
    local js_test_result=0
    run_javascript_tests
    js_test_result=$?

    # Combine results
    if [ $go_test_result -eq 0 ] && [ $js_test_result -eq 0 ]; then
        print_success "All integration tests passed!"
        return 0
    else
        print_error "Some integration tests failed"
        return 1
    fi
}

# Function to run specific test suites
run_test_suite() {
    local suite=$1
    
    print_status "Running $suite test suite..."
    
    cd "$TEST_DIR"
    
    case $suite in
        "basic")
            go test -v -run "TestIntegrationSuite" ./...
            ;;
        "cross-component")
            go test -v -run "TestCrossComponentValidation" ./...
            ;;
        "performance")
            go test -v -run "TestPerformanceAndLoad" ./...
            ;;
        "e2e")
            go test -v -run "TestE2EWorkflows" ./...
            ;;
        "economics")
            go test -v -run "TestEconomics" ./...
            ;;
        "gateway")
            go test -v -run "TestGateway" ./...
            ;;
        "wallet")
            go test -v -run "TestKNIRVWalletIntegration" ./...
            ;;
        "knirvbackend-server")
            go test -v -run "TestKNIRVNEXUSBackendIntegration" ./...
            ;;
        "knirvnexus-frontend")
            node knirvnexus_frontend_integration_test.js
            ;;
        "knirvnexus")
            go test -v -run "TestKNIRVNEXUSBackendIntegration" ./...
            node knirvnexus_frontend_integration_test.js
            ;;
        "graphchain-explorer")
            node knirv-graphchain-explorer.test.js
            ;;
        "portal")
            node portal-integration.test.js
            ;;
        "gateway-nexus")
            ./gateway_nexus_integration_test.sh
            ;;
        "javascript")
            run_javascript_tests
            ;;
        *)
            print_error "Unknown test suite: $suite"
            return 1
            ;;
    esac
}

# Function to generate test report
generate_test_report() {
    if [ "$GENERATE_REPORT" = false ]; then
        return
    fi

    print_status "Generating comprehensive test report..."

    local report_dir="$TEST_DIR/reports"
    local html_report="$report_dir/integration-test-report-$(date +%Y%m%d-%H%M%S).html"
    local json_report="$report_dir/integration-test-results-$(date +%Y%m%d-%H%M%S).json"

    # Generate JSON report with service status
    cat > "$json_report" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "test_environment": "integration",
    "command": "$COMMAND",
    "test_pattern": "$TEST_PATTERN",
    "services": {
        "knirvoracle": {
            "port": 1317,
            "health_endpoint": "http://localhost:1317/health",
            "status": "$(curl -s http://localhost:1317/health > /dev/null && echo 'running' || echo 'stopped')"
        },
        "knirvgraph": {
            "port": 8082,
            "health_endpoint": "http://localhost:8082/height",
            "status": "$(curl -s http://localhost:8082/height > /dev/null && echo 'running' || echo 'stopped')"
        },
        "knirvchain": {
            "port": 8090,
            "health_endpoint": "http://localhost:8090/health",
            "status": "$(curl -s http://localhost:8090/health > /dev/null && echo 'running' || echo 'stopped')"
        },
        "knirvnexus": {
            "port": 8083,
            "health_endpoint": "http://localhost:8083/health",
            "status": "$(curl -s http://localhost:8083/health > /dev/null && echo 'running' || echo 'stopped')"
        },
        "knirvrouter": {
            "port": 5001,
            "health_endpoint": "http://localhost:5001/status",
            "status": "$(curl -s http://localhost:5001/status > /dev/null && echo 'running' || echo 'stopped')"
        },
        "knirvgateway": {
            "port": 8888,
            "health_endpoint": "http://localhost:8888/",
            "status": "$(curl -s http://localhost:8888/ > /dev/null && echo 'running' || echo 'stopped')"
        }
    },
    "test_results": {
        "command_executed": "$COMMAND",
        "logs_directory": "$TEST_DIR/logs",
        "reports_directory": "$TEST_DIR/reports"
    }
}
EOF

    # Create enhanced HTML report
    cat > "$html_report" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>KNIRV Integration Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .header { background-color: #2c3e50; color: white; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .success { color: #27ae60; font-weight: bold; }
        .error { color: #e74c3c; font-weight: bold; }
        .warning { color: #f39c12; font-weight: bold; }
        .stopped { color: #95a5a6; font-weight: bold; }
        .running { color: #27ae60; font-weight: bold; }
        .section { margin: 20px 0; padding: 20px; background-color: white; border-radius: 5px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; margin: 20px 0; }
        .metric { padding: 15px; background-color: #ecf0f1; border-radius: 5px; border-left: 4px solid #3498db; }
        .service-status { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 15px; }
        .service { padding: 10px; border-radius: 5px; border: 1px solid #ddd; }
        .service.running { border-left: 4px solid #27ae60; background-color: #d5f4e6; }
        .service.stopped { border-left: 4px solid #e74c3c; background-color: #fdf2f2; }
        .timestamp { font-size: 0.9em; color: #7f8c8d; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🚀 KNIRV Network Integration Test Report</h1>
        <p class="timestamp">Generated: $(date)</p>
        <p>Test Environment: <strong>Real Services Integration</strong></p>
        <p>Command Executed: <strong>$COMMAND</strong></p>
        <p>Test Pattern: <strong>$TEST_PATTERN</strong></p>
    </div>

    <div class="section">
        <h2>🔧 Service Status</h2>
        <div class="service-status">
            <div class="service $(curl -s http://localhost:1317/health > /dev/null && echo 'running' || echo 'stopped')">
                <h4>KNIRVORACLE (Core Network)</h4>
                <p>Port: 1317</p>
                <p>Status: <span class="$(curl -s http://localhost:1317/health > /dev/null && echo 'running' || echo 'stopped')">$(curl -s http://localhost:1317/health > /dev/null && echo 'RUNNING' || echo 'STOPPED')</span></p>
                <p>Health: http://localhost:1317/health</p>
            </div>
            <div class="service $(curl -s http://localhost:8082/height > /dev/null && echo 'running' || echo 'stopped')">
                <h4>KNIRVGRAPH (Graph Database)</h4>
                <p>Port: 8082</p>
                <p>Status: <span class="$(curl -s http://localhost:8082/height > /dev/null && echo 'running' || echo 'stopped')">$(curl -s http://localhost:8082/height > /dev/null && echo 'RUNNING' || echo 'STOPPED')</span></p>
                <p>Health: http://localhost:8082/height</p>
            </div>
            <div class="service $(curl -s http://localhost:8090/health > /dev/null && echo 'running' || echo 'stopped')">
                <h4>KNIRVCHAIN (Blockchain)</h4>
                <p>Port: 8090</p>
                <p>Status: <span class="$(curl -s http://localhost:8090/health > /dev/null && echo 'running' || echo 'stopped')">$(curl -s http://localhost:8090/health > /dev/null && echo 'RUNNING' || echo 'STOPPED')</span></p>
                <p>Health: http://localhost:8090/health</p>
            </div>
            <div class="service $(curl -s http://localhost:8083/health > /dev/null && echo 'running' || echo 'stopped')">
                <h4>KNIRVNEXUS (Management Portal)</h4>
                <p>Port: 8083</p>
                <p>Status: <span class="$(curl -s http://localhost:8083/health > /dev/null && echo 'running' || echo 'stopped')">$(curl -s http://localhost:8083/health > /dev/null && echo 'RUNNING' || echo 'STOPPED')</span></p>
                <p>Health: http://localhost:8083/health</p>
            </div>
            <div class="service $(curl -s http://localhost:5001/status > /dev/null && echo 'running' || echo 'stopped')">
                <h4>KNIRVROUTER (Router)</h4>
                <p>Port: 5001</p>
                <p>Status: <span class="$(curl -s http://localhost:5001/status > /dev/null && echo 'running' || echo 'stopped')">$(curl -s http://localhost:5001/status > /dev/null && echo 'RUNNING' || echo 'STOPPED')</span></p>
                <p>Health: http://localhost:5001/status</p>
            </div>
            <div class="service $(curl -s http://localhost:8888/ > /dev/null && echo 'running' || echo 'stopped')">
                <h4>KNIRVGATEWAY (Gateway)</h4>
                <p>Port: 8888</p>
                <p>Status: <span class="$(curl -s http://localhost:8888/ > /dev/null && echo 'running' || echo 'stopped')">$(curl -s http://localhost:8888/ > /dev/null && echo 'RUNNING' || echo 'STOPPED')</span></p>
                <p>Health: http://localhost:8888/</p>
            </div>
        </div>
    </div>

    <div class="metrics">
        <div class="metric">
            <h3>📊 Test Execution</h3>
            <p><strong>Command:</strong> $COMMAND</p>
            <p><strong>Pattern:</strong> $TEST_PATTERN</p>
            <p><strong>Timeout:</strong> $TIMEOUT</p>
            <p><strong>Parallel:</strong> $PARALLEL</p>
            <p><strong>Verbose:</strong> $VERBOSE</p>
        </div>
        <div class="metric">
            <h3>📁 File Locations</h3>
            <p><strong>Logs:</strong> $TEST_DIR/logs/</p>
            <p><strong>Reports:</strong> $TEST_DIR/reports/</p>
            <p><strong>JSON Report:</strong> $json_report</p>
            <p><strong>Config:</strong> $SCRIPT_DIR/test-config.yaml</p>
        </div>
    </div>

    <div class="section">
        <h2>📋 Test Results Summary</h2>
        <p>Integration tests executed with <strong>real services</strong> (not mocks).</p>
        <p>All services were started, tests executed, and services properly stopped.</p>
        <p>For detailed test results, check the JSON reports and log files.</p>

        <h3>🔄 Service Lifecycle</h3>
        <ol>
            <li>✅ Cleaned up any existing services</li>
            <li>🚀 Started all real KNIRV services</li>
            <li>🔍 Verified service health endpoints</li>
            <li>🧪 Executed integration tests</li>
            <li>🛑 Gracefully stopped services</li>
            <li>✅ Verified all services stopped</li>
            <li>🔨 Used kill_knirv.sh if needed</li>
            <li>📊 Generated comprehensive reports</li>
        </ol>
    </div>

    <div class="section">
        <h2>🔗 Integration Points Tested</h2>
        <ul>
            <li><strong>KNIRVORACLE ↔ KNIRVGRAPH:</strong> Core network to graph database communication</li>
            <li><strong>KNIRVCHAIN ↔ KNIRVORACLE:</strong> Blockchain to core network integration</li>
            <li><strong>KNIRVNEXUS ↔ All Services:</strong> Management portal service discovery</li>
            <li><strong>KNIRVROUTER ↔ Network:</strong> P2P routing and connectivity</li>
            <li><strong>KNIRVGATEWAY ↔ Frontend:</strong> API gateway and web interface</li>
            <li><strong>Cross-Component Validation:</strong> End-to-end workflow testing</li>
        </ul>
    </div>

    <div class="section">
        <h2>📈 Next Steps</h2>
        <p>If any tests failed:</p>
        <ol>
            <li>Check individual service logs in <code>$TEST_DIR/logs/</code></li>
            <li>Verify service health endpoints manually</li>
            <li>Review the JSON report for detailed status</li>
            <li>Run specific test suites with <code>--pattern</code> option</li>
            <li>Use <code>--verbose</code> flag for detailed output</li>
        </ol>
    </div>
</body>
</html>
EOF
    
    print_success "HTML report generated: $html_report"
}

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS] [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  all                  Run all test suites (default)"
    echo "  basic               Run basic integration tests only"
    echo "  cross-component     Run cross-component validation tests only"
    echo "  performance         Run performance and load tests only"
    echo "  e2e                 Run end-to-end workflow tests only"
    echo "  economics           Run economics integration tests only"
    echo "  gateway             Run gateway integration tests only"
    echo "  wallet              Run KNIRVWALLET integration tests only"
    echo "  graphchain-explorer Run KNIRV GraphChain Explorer tests only"
    echo "  portal              Run Portal integration tests only"
    echo "  javascript          Run all JavaScript tests only"
    echo ""
    echo "Options:"
    echo "  --no-setup          Skip test environment setup"
    echo "  --no-teardown       Skip test environment teardown"
    echo "  --pattern PATTERN   Run tests matching pattern (regex)"
    echo "  --timeout DURATION  Test timeout (default: 600s)"
    echo "  --parallel          Run tests in parallel"
    echo "  --verbose           Enable verbose test output"
    echo "  --no-report         Skip generating test report"
    echo "  --help              Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                           # Run all tests with setup and teardown"
    echo "  $0 --no-setup basic         # Run basic tests without setup"
    echo "  $0 --pattern TestLLM        # Run tests matching 'TestLLM'"
    echo "  $0 --parallel --verbose     # Run all tests in parallel with verbose output"
}

# Parse command line arguments
COMMAND="all"

while [[ $# -gt 0 ]]; do
    case $1 in
        --no-setup)
            RUN_SETUP=false
            shift
            ;;
        --no-teardown)
            RUN_TEARDOWN=false
            shift
            ;;
        --pattern)
            TEST_PATTERN="$2"
            shift 2
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --parallel)
            PARALLEL=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --no-report)
            GENERATE_REPORT=false
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        all|basic|cross-component|performance|e2e|economics|gateway|wallet|graphchain-explorer|portal|javascript|gateway-nexus|knirvnexus|knirvbackend-server|knirvnexus-frontend)
            COMMAND="$1"
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            print_error "Available test suites: all, basic, cross-component, performance, e2e, economics, gateway, wallet, graphchain-explorer, portal, javascript, gateway-nexus, knirvnexus, knirvbackend-server, knirvnexus-frontend"
            exit 1
            ;;
    esac
done

# Trap to ensure teardown runs on exit
cleanup() {
    local exit_code=$?
    if [ "$RUN_TEARDOWN" = true ]; then
        print_status "Running cleanup due to script exit (exit code: $exit_code)..."
        run_teardown

        # Generate final report even if tests failed
        if [ "$GENERATE_REPORT" = true ]; then
            print_status "Generating final report..."
            generate_test_report
        fi
    fi
    exit $exit_code
}
trap cleanup EXIT INT TERM

# Main execution
main() {
    print_status "Starting KNIRV Integration Test Runner"
    print_status "Command: $COMMAND"
    print_status "Test pattern: $TEST_PATTERN"
    print_status "Timeout: $TIMEOUT"
    print_status "Parallel: $PARALLEL"
    print_status "Verbose: $VERBOSE"
    
    # Run setup
    run_setup
    
    # Run tests based on command
    local test_result=0
    
    case $COMMAND in
        "all")
            run_integration_tests
            test_result=$?
            ;;
        "basic"|"cross-component"|"performance"|"e2e"|"economics"|"gateway"|"wallet"|"graphchain-explorer"|"portal"|"javascript"|"gateway-nexus"|"knirvnexus"|"knirvbackend-server"|"knirvnexus-frontend")
            run_test_suite "$COMMAND"
            test_result=$?
            ;;
        *)
            print_error "Unknown command: $COMMAND"
            exit 1
            ;;
    esac
    
    # Check results (report will be generated in cleanup)
    if [ $test_result -eq 0 ]; then
        print_success "All tests completed successfully!"
        print_status "Services will be stopped and report generated during cleanup..."
    else
        print_error "Some tests failed!"
        print_status "Services will be stopped and report generated during cleanup..."
        exit 1
    fi
}

# Run main function
main "$@"
