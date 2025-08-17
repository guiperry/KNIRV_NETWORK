#!/bin/bash

# KNIRV TESTNET - CORTEX Demo Execution Script
# Runs all automated CORTEX demonstrations

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_ROOT="$TESTNET_ROOT/tests"
DEMO_SUITE_DIR="$TEST_ROOT/e2e/cortex-demo-suite"
LOG_DIR="$TEST_ROOT/logs"
REPORT_DIR="$TEST_ROOT/reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Demo configuration
AVAILABLE_DEMOS=("skill-development" "multi-agent-collaboration" "learning-adaptation")
DEMOS_TO_RUN=()
CONTINUOUS_MODE=false
INTERVAL="30m"
MAX_ITERATIONS=0
CURRENT_ITERATION=0

# Print functions
print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================${NC}"
}

print_step() {
    echo -e "${YELLOW}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Usage function
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --scenario SCENARIO    Run specific demo scenario"
    echo "                         Available: ${AVAILABLE_DEMOS[*]}"
    echo "  --all                  Run all demo scenarios"
    echo "  --continuous           Run demos continuously"
    echo "  --interval INTERVAL    Interval between continuous runs (default: 30m)"
    echo "  --max-iterations N     Maximum iterations for continuous mode (0 = infinite)"
    echo "  --list                 List available demo scenarios"
    echo "  --help                 Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --scenario skill-development"
    echo "  $0 --all"
    echo "  $0 --continuous --interval 1h --max-iterations 5"
}

# Parse command line arguments
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --scenario)
                if [[ -n "$2" && " ${AVAILABLE_DEMOS[*]} " =~ " $2 " ]]; then
                    DEMOS_TO_RUN+=("$2")
                    shift 2
                else
                    print_error "Invalid or missing scenario: $2"
                    echo "Available scenarios: ${AVAILABLE_DEMOS[*]}"
                    exit 1
                fi
                ;;
            --all)
                DEMOS_TO_RUN=("${AVAILABLE_DEMOS[@]}")
                shift
                ;;
            --continuous)
                CONTINUOUS_MODE=true
                shift
                ;;
            --interval)
                if [[ -n "$2" ]]; then
                    INTERVAL="$2"
                    shift 2
                else
                    print_error "Missing interval value"
                    exit 1
                fi
                ;;
            --max-iterations)
                if [[ -n "$2" && "$2" =~ ^[0-9]+$ ]]; then
                    MAX_ITERATIONS="$2"
                    shift 2
                else
                    print_error "Invalid max-iterations value: $2"
                    exit 1
                fi
                ;;
            --list)
                echo "Available CORTEX demo scenarios:"
                for demo in "${AVAILABLE_DEMOS[@]}"; do
                    echo "  - $demo"
                done
                exit 0
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # Default to all demos if none specified
    if [[ ${#DEMOS_TO_RUN[@]} -eq 0 ]]; then
        DEMOS_TO_RUN=("${AVAILABLE_DEMOS[@]}")
    fi
}

# Initialize demo environment
initialize_demo_environment() {
    print_step "Initializing CORTEX demo environment..."
    
    # Create necessary directories
    mkdir -p "$LOG_DIR" "$REPORT_DIR"
    mkdir -p "$DEMO_SUITE_DIR/logs" "$DEMO_SUITE_DIR/reports"
    
    # Check if demo scripts exist
    for demo in "${DEMOS_TO_RUN[@]}"; do
        local demo_script="$DEMO_SUITE_DIR/${demo}-demo.sh"
        if [[ ! -f "$demo_script" ]]; then
            print_error "Demo script not found: $demo_script"
            return 1
        fi
        
        # Make sure script is executable
        chmod +x "$demo_script"
    done
    
    print_success "Demo environment initialized"
}

# Check prerequisites
check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check required tools
    local required_tools=("curl" "jq")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            print_error "Required tool not found: $tool"
            return 1
        fi
    done
    
    # Check if CORTEX is running
    if ! curl -s "http://localhost:3001/health" &> /dev/null; then
        print_error "CORTEX endpoint not accessible at http://localhost:3001"
        print_info "Please ensure CORTEX is running before executing demos"
        return 1
    fi
    
    # Check if testnet services are running
    local services=("1317" "8090" "8082" "8084" "5001" "8087")
    local service_names=("knirv-oracle" "knirvchain" "knirvgraph" "knirv-nexus" "knirv-router" "knirv-gateway")
    
    for i in "${!services[@]}"; do
        local port="${services[$i]}"
        local name="${service_names[$i]}"
        
        if ! curl -s "http://localhost:$port/health" &> /dev/null; then
            print_error "Service $name not accessible at port $port"
            return 1
        fi
    done
    
    print_success "All prerequisites satisfied"
}

# Run a single demo
run_demo() {
    local demo_name="$1"
    local iteration="$2"
    
    print_step "Running CORTEX demo: $demo_name"
    
    if [[ -n "$iteration" ]]; then
        print_info "Iteration: $iteration"
    fi
    
    local demo_script="$DEMO_SUITE_DIR/${demo_name}-demo.sh"
    local log_file="$LOG_DIR/cortex_demo_${demo_name}_${TIMESTAMP}.log"
    
    # Run the demo
    local start_time=$(date +%s)
    
    if "$demo_script" 2>&1 | tee "$log_file"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        print_success "Demo '$demo_name' completed successfully in ${duration}s"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        print_error "Demo '$demo_name' failed after ${duration}s"
        print_info "Check log file: $log_file"
        return 1
    fi
}

# Run all selected demos
run_demos() {
    local iteration="$1"
    local total_demos=${#DEMOS_TO_RUN[@]}
    local successful_demos=0
    local failed_demos=0
    
    print_header "Running CORTEX Demos"
    
    if [[ -n "$iteration" ]]; then
        print_info "Iteration: $iteration"
    fi
    
    print_info "Demos to run: ${DEMOS_TO_RUN[*]}"
    print_info "Total demos: $total_demos"
    
    for demo in "${DEMOS_TO_RUN[@]}"; do
        if run_demo "$demo" "$iteration"; then
            ((successful_demos++))
        else
            ((failed_demos++))
        fi
        
        # Add delay between demos
        if [[ "$demo" != "${DEMOS_TO_RUN[-1]}" ]]; then
            print_info "Waiting 30 seconds before next demo..."
            sleep 30
        fi
    done
    
    # Summary
    print_header "Demo Execution Summary"
    print_info "Total demos: $total_demos"
    print_success "Successful: $successful_demos"
    
    if [[ $failed_demos -gt 0 ]]; then
        print_error "Failed: $failed_demos"
    else
        print_info "Failed: $failed_demos"
    fi
    
    local success_rate=$(( (successful_demos * 100) / total_demos ))
    print_info "Success rate: ${success_rate}%"
    
    return $failed_demos
}

# Generate demo report
generate_demo_report() {
    print_step "Generating CORTEX demo report..."
    
    local report_file="$REPORT_DIR/cortex_demos_report_${TIMESTAMP}.html"
    
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>CORTEX Demos Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .success { color: green; }
        .error { color: red; }
        .demo { margin: 15px 0; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>CORTEX Automated Demos Report</h1>
        <p>Generated: $(date)</p>
        <p>Execution ID: $TIMESTAMP</p>
    </div>
    
    <div class="demo">
        <h2>Execution Summary</h2>
        <p>Demos executed: ${DEMOS_TO_RUN[*]}</p>
        <p>Continuous mode: $CONTINUOUS_MODE</p>
        $(if [[ "$CONTINUOUS_MODE" == "true" ]]; then
            echo "<p>Interval: $INTERVAL</p>"
            echo "<p>Iterations: $CURRENT_ITERATION</p>"
        fi)
    </div>
    
    <div class="demo">
        <h2>Available Demos</h2>
        <ul>
            <li><strong>skill-development:</strong> Demonstrates CORTEX agent creating and registering a new skill</li>
            <li><strong>multi-agent-collaboration:</strong> Shows multiple CORTEX agents collaborating on complex tasks</li>
            <li><strong>learning-adaptation:</strong> Demonstrates CORTEX learning and improving performance</li>
        </ul>
    </div>
    
    <div class="demo">
        <h2>Log Files</h2>
        <p>Demo logs are available in: $LOG_DIR</p>
        <p>Individual demo reports are available in: $DEMO_SUITE_DIR/reports</p>
    </div>
</body>
</html>
EOF
    
    print_success "Demo report generated: $report_file"
}

# Convert interval to seconds
interval_to_seconds() {
    local interval="$1"
    local number="${interval%[a-zA-Z]*}"
    local unit="${interval#$number}"
    
    case "$unit" in
        s|sec|seconds) echo "$number" ;;
        m|min|minutes) echo $((number * 60)) ;;
        h|hour|hours) echo $((number * 3600)) ;;
        d|day|days) echo $((number * 86400)) ;;
        *) echo 1800 ;; # Default to 30 minutes
    esac
}

# Main execution function
main() {
    local overall_start_time=$(date +%s)
    
    print_header "KNIRV TESTNET - CORTEX Demo Suite"
    print_info "Timestamp: $TIMESTAMP"
    
    # Parse arguments
    parse_arguments "$@"
    
    # Initialize environment
    if ! initialize_demo_environment; then
        exit 1
    fi
    
    # Check prerequisites
    if ! check_prerequisites; then
        exit 1
    fi
    
    # Run demos
    if [[ "$CONTINUOUS_MODE" == "true" ]]; then
        print_info "Running in continuous mode with interval: $INTERVAL"
        if [[ $MAX_ITERATIONS -gt 0 ]]; then
            print_info "Maximum iterations: $MAX_ITERATIONS"
        else
            print_info "Running indefinitely (Ctrl+C to stop)"
        fi
        
        local interval_seconds=$(interval_to_seconds "$INTERVAL")
        
        while true; do
            ((CURRENT_ITERATION++))
            
            print_header "Continuous Demo Execution - Iteration $CURRENT_ITERATION"
            
            if ! run_demos "$CURRENT_ITERATION"; then
                print_error "Demo execution failed in iteration $CURRENT_ITERATION"
            fi
            
            # Check if we've reached max iterations
            if [[ $MAX_ITERATIONS -gt 0 && $CURRENT_ITERATION -ge $MAX_ITERATIONS ]]; then
                print_info "Reached maximum iterations ($MAX_ITERATIONS)"
                break
            fi
            
            # Wait for next iteration
            print_info "Waiting $INTERVAL before next iteration..."
            sleep "$interval_seconds"
        done
    else
        # Single execution
        if ! run_demos; then
            print_error "Demo execution failed"
            exit 1
        fi
    fi
    
    # Generate report
    generate_demo_report
    
    # Final summary
    local overall_end_time=$(date +%s)
    local total_duration=$((overall_end_time - overall_start_time))
    
    print_header "CORTEX Demo Suite Completed"
    print_success "Total execution time: ${total_duration}s"
    
    if [[ "$CONTINUOUS_MODE" == "true" ]]; then
        print_success "Completed $CURRENT_ITERATION iterations"
    fi
    
    print_info "Reports available in: $REPORT_DIR"
    print_info "Logs available in: $LOG_DIR"
}

# Handle Ctrl+C gracefully
trap 'print_info "Demo execution interrupted by user"; exit 130' INT

# Run main function
main "$@"
