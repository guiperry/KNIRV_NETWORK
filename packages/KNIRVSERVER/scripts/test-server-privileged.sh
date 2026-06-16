#!/bin/bash

# KNIRVSERVER Privileged Tests with Full Docker Instance Deployment
# This script deploys a complete KNIRVSERVER Docker instance with Debian + Kali tools
# and runs all tests that require root privileges (cgroups, namespaces, etc.)
#
# Architecture:
# - Docker container: Uses Podman for nested container orchestration
# - Bare metal: Uses native Go runtime for containers
# - Kata container: Uses native Go runtime (VM-level isolation)

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration - tolerate either the public package layout or the private
# KNIRV_CORP checkout passed in from the Makefile.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

resolve_workspace_root() {
    if [[ -n "${KNIRVSERVER_WORKSPACE_DIR:-}" && -d "${KNIRVSERVER_WORKSPACE_DIR}" ]]; then
        printf '%s\n' "${KNIRVSERVER_WORKSPACE_DIR}"
        return
    fi

    if [[ -n "${KNIRVSERVER_BACKEND_DIR:-}" && -d "${KNIRVSERVER_BACKEND_DIR}" ]]; then
        local parent
        parent="$(cd "$(dirname "${KNIRVSERVER_BACKEND_DIR}")" && pwd)"
        if [[ -d "${parent}/scripts" || -d "${parent}/backend" || -f "${parent}/go.mod" ]]; then
            printf '%s\n' "${parent}"
            return
        fi
        printf '%s\n' "${KNIRVSERVER_BACKEND_DIR}"
        return
    fi

    local package_root
    package_root="$(cd "${SCRIPT_DIR}/.." && pwd)"
    if [[ -d "${package_root}/backend" || -f "${package_root}/go.mod" ]]; then
        printf '%s\n' "${package_root}"
        return
    fi

    if [[ -d "/home/gperry/Documents/GitHub/KNIRV/KNIRV_CORP/server" ]]; then
        printf '%s\n' "/home/gperry/Documents/GitHub/KNIRV/KNIRV_CORP/server"
        return
    fi

    if [[ -d "/home/gperry/Documents/GitHub/KNIRV/KNIRV_CORP/backend" ]]; then
        printf '%s\n' "/home/gperry/Documents/GitHub/KNIRV/KNIRV_CORP/backend"
        return
    fi

    printf '%s\n' "${package_root}"
}

resolve_backend_dir() {
    if [[ -n "${KNIRVSERVER_BACKEND_DIR:-}" && -d "${KNIRVSERVER_BACKEND_DIR}" ]]; then
        printf '%s\n' "${KNIRVSERVER_BACKEND_DIR}"
        return
    fi

    local workspace_root
    workspace_root="$(resolve_workspace_root)"
    if [[ -d "${workspace_root}/backend" ]]; then
        printf '%s\n' "${workspace_root}/backend"
        return
    fi

    printf '%s\n' "${workspace_root}"
}

KNIRVNEXUS_ROOT="$(resolve_workspace_root)"
BACKEND_DIR="$(resolve_backend_dir)"
PROJECT_ROOT="$(cd "${KNIRVNEXUS_ROOT}/.." && pwd)"
CONTAINER_NAME="knirvserver-kali-local"
CONTAINER_DEPLOYER_DIR="${BACKEND_DIR}/cmd/container_deployer"
TEST_RESULTS_DIR="${KNIRVNEXUS_ROOT}/test-results/privileged-tests"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
LOG_FILE="${TEST_RESULTS_DIR}/privileged-tests_${TIMESTAMP}.log"

# Test configuration
DEPLOY_CONTAINER=true
CLEANUP_CONTAINER=false
RUN_TESTS=true
SHOW_BUILD_OUTPUT=false
WAIT_TIMEOUT=120  # seconds to wait for container to be ready

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-deploy)
            DEPLOY_CONTAINER=false
            shift
            ;;
        --cleanup)
            CLEANUP_CONTAINER=true
            shift
            ;;
        --no-tests)
            RUN_TESTS=false
            shift
            ;;
        --show-build)
            SHOW_BUILD_OUTPUT=true
            shift
            ;;
        --wait-timeout)
            WAIT_TIMEOUT="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --no-deploy       Skip container deployment (use existing container)"
            echo "  --cleanup         Clean up container after tests"
            echo "  --no-tests        Only deploy container, don't run tests"
            echo "  --show-build      Show verbose Docker build output"
            echo "  --wait-timeout N  Wait N seconds for container to be ready (default: 120)"
            echo "  --help            Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                           # Deploy and run all tests"
            echo "  $0 --no-deploy               # Run tests on existing container"
            echo "  $0 --cleanup                 # Deploy, test, and cleanup"
            echo "  $0 --show-build              # Show verbose build output"
            echo "  $0 --show-build --cleanup    # Show build output and cleanup after"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Create test results directory
mkdir -p "${TEST_RESULTS_DIR}"

# Logging function
log() {
    echo -e "$1" | tee -a "${LOG_FILE}"
}

log_header() {
    log ""
    log "${BLUE}========================================${NC}"
    log "${BLUE}$1${NC}"
    log "${BLUE}========================================${NC}"
}

log_success() {
    log "${GREEN}✓ $1${NC}"
}

log_warning() {
    log "${YELLOW}⚠ $1${NC}"
}

log_error() {
    log "${RED}✗ $1${NC}"
}

# Initialize log file
log_header "KNIRVSERVER Privileged Tests"
log "Started: $(date)"
log "Container: ${CONTAINER_NAME}"
log "KNIRVSERVER Root: ${KNIRVNEXUS_ROOT}"
log "Project Root: ${PROJECT_ROOT}"
log "Results Directory: ${TEST_RESULTS_DIR}"
log ""

# Check prerequisites
check_prerequisites() {
    log_header "Checking Prerequisites"

    local missing_deps=false

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker not found. Please install Docker."
        missing_deps=true
    else
        log_success "Docker found: $(docker --version)"
    fi

    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go not found. Please install Go 1.21+."
        missing_deps=true
    else
        log_success "Go found: $(go version)"
    fi

    # Check container deployer
    if [ ! -f "${CONTAINER_DEPLOYER_DIR}/main.go" ]; then
        log_error "Container deployer not found at ${CONTAINER_DEPLOYER_DIR}/main.go"
        missing_deps=true
    else
        log_success "Container deployer found"
    fi

    # Check KNIRVSERVER backend tests
    if [ ! -d "${KNIRVNEXUS_ROOT}/backend/tests" ]; then
        log_warning "KNIRVSERVER backend tests directory not found"
    else
        log_success "KNIRVSERVER backend tests found"
    fi

    if [ "$missing_deps" = true ]; then
        log_error "Missing prerequisites. Exiting."
        exit 1
    fi

    log_success "All prerequisites satisfied"
}

# Deploy Docker container
deploy_container() {
    log_header "Deploying KNIRVSERVER Docker Container"

    # The Kali Docker base image is assumed to be built by os_builder previously.
    # If not found, container_deployer will provide guidance.


    # Check if container already exists
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_warning "Container ${CONTAINER_NAME} already exists. Removing and recreating..."
        docker rm -f "${CONTAINER_NAME}" || true
    fi

    log "Deploying container using pre-built Kali tools image..."

    cd "${CONTAINER_DEPLOYER_DIR}"

    # Run container deployer binary non-interactively, skipping image build as it's pre-built
    if ./container_deployer \
        --action 1 \
        --deploy-mode container \
        --deploy-type local \
        --env development \
        --skip-image-build \
        2>&1 | tee -a "${LOG_FILE}"; then
        log_success "Container deployment completed"
    else
        log_error "Container deployment failed"
        return 1
    fi

    cd "${KNIRVNEXUS_ROOT}"
}

# Wait for container to be ready
wait_for_container() {
    log_header "Waiting for Container to be Ready"

    local elapsed=0
    local interval=5

    while [ $elapsed -lt $WAIT_TIMEOUT ]; do
        if docker ps --filter "name=${CONTAINER_NAME}" --filter "status=running" --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
            log_success "Container is running"

            # Wait a bit more for services to start inside the container
            log "Waiting for services to initialize..."
            sleep 10

            # Check if we can execute commands
            if docker exec "${CONTAINER_NAME}" echo "Container ready" &> /dev/null; then
                log_success "Container is ready for test execution"
                return 0
            fi
        fi

        log "Waiting for container... (${elapsed}s / ${WAIT_TIMEOUT}s)"
        sleep $interval
        elapsed=$((elapsed + interval))
    done

    log_error "Container failed to become ready within ${WAIT_TIMEOUT} seconds"
    return 1
}

# Run privileged tests inside container
run_privileged_tests() {
    log_header "Running Privileged Tests"

    # Verify container is running
    if ! docker ps --filter "name=${CONTAINER_NAME}" --filter "status=running" --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_error "Container ${CONTAINER_NAME} is not running"
        return 1
    fi

    log "Executing tests inside Docker container as root..."
    log ""

    # Define test patterns that require root privileges
    local ROOT_TEST_PATTERNS=(
        "TestNewCgroupManager"
        "TestCgroupManager"
        "TestNamespaceManager"
        "TestNamespace"
        "TestCapabilityManager"
        "TestCapability"
        "TestSeccompManager"
        "TestSeccomp"
        "TestAppArmorManager"
        "TestAppArmor"
        "TestNetworkManager"
        "TestNetwork"
        "TestMountManager"
        "TestMount"
        "TestHardenedContainer"
        "TestHardenedContainerExecution"
        "TestContainerRuntime"
        "TestPrivileged"
        "TestPodmanRuntime"
        "TestNativeRuntime"
    )

    local test_pattern=$(IFS='|'; echo "${ROOT_TEST_PATTERNS[*]}")

    # Run tests inside container with root privileges
    log "${CYAN}Test pattern: ${test_pattern}${NC}"
    log ""

    # Copy KNIRVSERVER source to container if needed
    log "Ensuring KNIRVSERVER source is available in container..."
    if ! docker exec "${CONTAINER_NAME}" bash -c "test -d /workspace/KNIRVSERVER/backend" 2>/dev/null; then
        log "Copying KNIRVSERVER source to container..."
        # Create workspace directory first
        docker exec "${CONTAINER_NAME}" mkdir -p /workspace
        # Copy the entire KNIRVSERVER directory
        docker cp "${KNIRVNEXUS_ROOT}/." "${CONTAINER_NAME}:/workspace/KNIRVSERVER/"
        log "Source copied successfully"
    else
        log "Source already present in container"
    fi

    # Verify the path exists
    log "Verifying source path..."
    if ! docker exec "${CONTAINER_NAME}" bash -c "test -d /workspace/KNIRVSERVER/backend"; then
        log_error "Backend directory not found in container"
        log "Listing /workspace contents:"
        docker exec "${CONTAINER_NAME}" ls -la /workspace/
        return 1
    fi

    # Install test dependencies inside container
    log "Installing test dependencies..."
    docker exec "${CONTAINER_NAME}" bash -c "
        export PATH=\"/usr/local/go/bin:\$PATH\" && \
        cd /workspace/KNIRVSERVER/backend && \
        go mod download && \
        go mod tidy
    " | tee -a "${LOG_FILE}"

    # Ensure required tools are available in container (for tests)
    log "Ensuring required tools and permissions are available in container..."
    docker exec "${CONTAINER_NAME}" bash -c "
        # Use /var/tmp instead of /tmp (Docker is less likely to mount it as read-only)
        chmod 1777 /var/tmp 2>/dev/null || true
        mkdir -p /var/tmp/knirv-containers 2>/dev/null || true
        chmod 755 /var/tmp/knirv-containers 2>/dev/null || true
        
        # Also ensure /tmp is writable as fallback
        chmod 1777 /tmp 2>/dev/null || true
        mkdir -p /tmp/knirv-containers 2>/dev/null || true
        chmod 755 /tmp/knirv-containers 2>/dev/null || true
        
        # Check if iproute2 is installed, install if missing
        if ! command -v ip &> /dev/null; then
            echo 'Installing iproute2...'
            apt-get update -qq && apt-get install -y -qq iproute2 2>/dev/null || true
        fi
        # Check if util-linux is installed (provides unshare)
        if ! command -v unshare &> /dev/null; then
            echo 'Installing util-linux...'
            apt-get update -qq && apt-get install -y -qq util-linux 2>/dev/null || true
        fi
        # Check if mount is available
        if ! command -v mount &> /dev/null; then
            echo 'Installing mount utilities...'
            apt-get update -qq && apt-get install -y -qq mount 2>/dev/null || true
        fi
        # Check if Podman is available (for nested container orchestration)
        if command -v podman &> /dev/null; then
            echo 'Podman found: $(podman --version)'
        else
            echo 'Podman not found - will use native runtime'
        fi
        # Ensure PATH includes /usr/sbin
        export PATH=\"/usr/sbin:/sbin:\$PATH\"
        echo 'Tools check:'
        which ip 2>/dev/null || echo 'ip not found'
        which unshare 2>/dev/null || echo 'unshare not found'
        which mount 2>/dev/null || echo 'mount not found'
        which podman 2>/dev/null || echo 'podman not found'
    " | tee -a "${LOG_FILE}"

    # Run privileged tests
    log ""
    log "${BLUE}Executing privileged tests...${NC}"
    log ""

    local test_output="${TEST_RESULTS_DIR}/test-output_${TIMESTAMP}.txt"
    local test_json_raw="${TEST_RESULTS_DIR}/test-results-raw_${TIMESTAMP}.json" # Raw JSON output
    local test_json_summary="${TEST_RESULTS_DIR}/test-results-summary_${TIMESTAMP}.json" # Summarized JSON output

    # Check for jq in container
    if ! docker exec "${CONTAINER_NAME}" bash -c "export PATH=\"/usr/local/go/bin:/usr/sbin:/sbin:\$PATH\"; command -v jq" &> /dev/null; then
        log_warning "jq not found in container. Installing jq..."
        docker exec "${CONTAINER_NAME}" bash -c "
            export PATH=\"/usr/local/go/bin:/usr/sbin:/sbin:\$PATH\" && \
            apt-get update -qq && \
            apt-get install -y -qq jq
        " | tee -a "${LOG_FILE}"
        if [ $? -ne 0 ]; then
            log_error "Failed to install jq in container. Cannot parse JSON test results accurately."
        fi
    fi

    # Execute Go tests inside the container
    if docker exec "${CONTAINER_NAME}" bash -c "
        set -e
        export KNIRVNEXUS_TEST_MODE=true
        export PATH=\"/usr/local/go/bin:/usr/sbin:/sbin:\$PATH\"
        
        # Detect host environment
        if [ -f /.dockerenv ] || [ -f /run/.containerenv ]; then
            export KNIRVNEXUS_HOST_ENV=docker
            echo 'Detected environment: Docker container' >&2
        elif grep -q 'agent.container_manager=kata' /proc/cmdline 2>/dev/null; then
            export KNIRVNEXUS_HOST_ENV=kata-container
            echo 'Detected environment: Kata container' >&2
        else
            export KNIRVNEXUS_HOST_ENV=bare-metal
            echo 'Detected environment: Bare metal' >&2
        fi
        
        # Set preferred runtime based on environment
        if [ \"${KNIRVNEXUS_HOST_ENV}\" = \"docker\" ]; then
            if command -v podman &> /dev/null; then
                export KNIRVNEXUS_PREFERRED_RUNTIME=podman
                echo 'Using Podman for nested containers (Docker environment)' >&2
            else
                export KNIRVNEXUS_PREFERRED_RUNTIME=native-go
                echo 'Podman not available, using native-go runtime' >&2
            fi
        else
            export KNIRVNEXUS_PREFERRED_RUNTIME=native-go
            echo 'Using native-go runtime (bare-metal or Kata)' >&2
        fi
        
        cd /workspace/KNIRVSERVER/backend

        # Check kernel features
        echo '=== Kernel Feature Check ===' >&2
        uname -r >&2
        echo '' >&2
        echo 'Cgroups:' >&2
        mount | grep cgroup >&2 || echo 'No cgroup mounts found' >&2
        echo '' >&2
        echo 'Namespaces:' >&2
        ls -la /proc/self/ns/ >&2 || echo 'Namespace info not available' >&2
        echo '' >&2

        # Run tests with verbose output and JSON results
        go test -v -json -timeout=30m \
            -run '${test_pattern}' \
            ./internal/services/teesecurity/...
    " > "${test_json_raw}" ; then # Capture raw JSON output here
        log_success "Privileged tests completed successfully (initial run)"
    else
        log_error "Privileged tests failed (go test command returned non-zero exit code)"
    fi

    log_header "Raw Go Test Output (for debugging jq parse error)"
    if [ -f "${test_json_raw}" ]; then
        log "Content of ${test_json_raw}:"
        cat "${test_json_raw}" | tee -a "${LOG_FILE}"
    else
        log "WARNING: ${test_json_raw} not found."
    fi
    log_header "End Raw Go Test Output"

    # Now, process results (moved outside the initial if/else for logging raw output)
    if [ -f "${test_json_raw}" ]; then
        # Filter for actual test results (PASS/FAIL/SKIP) and save as summarized JSON
        cat "${test_json_raw}" | jq -s 'map(select(.Action == "pass" or .Action == "fail" or .Action == "skip"))' > "${test_json_summary}" || true # Add || true to prevent immediate script exit on jq error

        # Count results using jq
        local passed=$(jq 'map(select(.Action == "pass")) | length' "${test_json_summary}") || passed=0
        local failed=$(jq 'map(select(.Action == "fail")) | length' "${test_json_summary}") || failed=0
        local skipped=$(jq 'map(select(.Action == "skip")) | length' "${test_json_summary}") || skipped=0

        # Handle potential null/empty outputs from jq if file is malformed or empty
        passed=${passed:-0}
        failed=${failed:-0}
        skipped=${skipped:-0}

        log ""
        log_header "Test Results Summary"
        log "${GREEN}Passed:  ${passed}${NC}"
        log "${RED}Failed:  ${failed}${NC}"
        log "${YELLOW}Skipped: ${skipped}${NC}"
        log ""
        log "Detailed results (raw JSON): ${test_json_raw}"
        log "Summarized results (JSON): ${test_json_summary}" # New entry for summarized JSON

        if [ "${failed}" -gt 0 ]; then
            log_error "Some tests failed"
            return 1
        else
            return 0 # All tests passed or skipped
        fi
    else
        log_error "No raw test results JSON found to parse. Test run likely failed completely."
        return 1
    fi
}

# Collect container logs
collect_container_logs() {
    log_header "Collecting Container Logs"

    local container_logs="${TEST_RESULTS_DIR}/container-logs_${TIMESTAMP}.txt"

    if docker ps -a --filter "name=${CONTAINER_NAME}" --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log "Saving container logs to ${container_logs}"
        docker logs "${CONTAINER_NAME}" > "${container_logs}" 2>&1
        log_success "Container logs collected"
    else
        log_warning "Container not found, cannot collect logs"
    fi
}

# Cleanup container
cleanup_container() {
    log_header "Cleaning Up Container"

    if docker ps -a --filter "name=${CONTAINER_NAME}" --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log "Stopping and removing container ${CONTAINER_NAME}..."
        docker rm -f "${CONTAINER_NAME}" || true
        log_success "Container removed"
    else
        log_warning "Container not found, nothing to clean up"
    fi
}

# Generate test report
generate_test_report() {
    log_header "Generating Test Report"

    local report_file="${TEST_RESULTS_DIR}/test-report_${TIMESTAMP}.md"

    cat > "${report_file}" <<EOF
# KNIRVSERVER Privileged Tests Report

**Generated:** $(date)
**Container:** ${CONTAINER_NAME}
**Timestamp:** ${TIMESTAMP}

## Test Configuration

- **Deploy Container:** ${DEPLOY_CONTAINER}
- **Run Tests:** ${RUN_TESTS}
- **Cleanup After:** ${CLEANUP_CONTAINER}
- **Wait Timeout:** ${WAIT_TIMEOUT}s

## Test Results

EOF

    # Append test summary if available
    if [ -f "${TEST_RESULTS_DIR}/test-results-summary_${TIMESTAMP}.json" ]; then
        echo "### Summary" >> "${report_file}"
        echo "" >> "${report_file}"

        local passed=$(jq 'map(select(.Action == "pass")) | length' "${TEST_RESULTS_DIR}/test-results-summary_${TIMESTAMP}.json") || passed=0
        local failed=$(jq 'map(select(.Action == "fail")) | length' "${TEST_RESULTS_DIR}/test-results-summary_${TIMESTAMP}.json") || failed=0
        local skipped=$(jq 'map(select(.Action == "skip")) | length' "${TEST_RESULTS_DIR}/test-results-summary_${TIMESTAMP}.json") || skipped=0
        
        # Handle empty strings from jq
        passed=${passed:-0}
        failed=${failed:-0}
        skipped=${skipped:-0}

        echo "- ✅ **Passed:** ${passed}" >> "${report_file}"
        echo "- ❌ **Failed:** ${failed}" >> "${report_file}"
        echo "- ⏭️  **Skipped:** ${skipped}" >> "${report_file}"
    fi

    log_success "Test report generated: ${report_file}"
}

# Main function - orchestrates the entire test workflow
main() {
    local exit_code=0

    # Step 1: Check prerequisites
    check_prerequisites || exit 1

    # Step 2: Deploy container if requested
    if [ "$DEPLOY_CONTAINER" = true ]; then
        if ! deploy_container; then
            log_error "Container deployment failed"
            exit 1
        fi

        if ! wait_for_container; then
            log_error "Container failed to become ready"
            exit 1
        fi
    else
        log_warning "Skipping container deployment (--no-deploy)"
        # Still check if container exists
        if ! docker ps --filter "name=${CONTAINER_NAME}" --filter "status=running" --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}\$"; then
            log_error "Container ${CONTAINER_NAME} is not running. Remove --no-deploy flag to deploy."
            exit 1
        fi
    fi

    # --- INSERT DIAGNOSTIC STEPS HERE ---
    log_header "Container Network Diagnostics"
    log "Overwriting /etc/resolv.conf with public DNS (8.8.8.8)..."
    docker exec "${CONTAINER_NAME}" bash -c "echo 'nameserver 8.8.8.8' > /etc/resolv.conf" | tee -a "${LOG_FILE}"
    log "Checking /etc/resolv.conf inside container:"
    docker exec "${CONTAINER_NAME}" cat /etc/resolv.conf | tee -a "${LOG_FILE}"
    log "Attempting to ping google.com from inside container:"
    docker exec "${CONTAINER_NAME}" bash -c "ping -c 3 google.com || echo 'Ping failed'" | tee -a "${LOG_FILE}"
    log_success "Network diagnostics complete."
    # --- END DIAGNOSTIC STEPS ---

    # Step 3: Run tests if requested
    if [ "$RUN_TESTS" = true ]; then
        if ! run_privileged_tests; then
            exit_code=1
        fi
    else
        log_warning "Skipping tests (--no-tests)"
    fi

    # Step 4: Collect logs
    collect_container_logs

    # Step 5: Generate report
    generate_test_report

    # Step 6: Cleanup if requested
    if [ "$CLEANUP_CONTAINER" = true ]; then
        cleanup_container
    else
        log_warning "Container left running (use --cleanup to remove)"
        log "To view container logs: docker logs ${CONTAINER_NAME}"
        log "To enter container: docker exec -it ${CONTAINER_NAME} bash"
        log "To stop container: docker stop ${CONTAINER_NAME}"
        log "To remove container: docker rm -f ${CONTAINER_NAME}"
    fi

    # Final summary
    log ""
    log_header "Test Execution Complete"
    log "Exit code: ${exit_code}"
    log "Results directory: ${TEST_RESULTS_DIR}"
    log "Full log: ${LOG_FILE}"

    if [ ${exit_code} -eq 0 ]; then
        log_success "All tests passed!"
    else
        log_error "Some tests failed. Review logs for details."
    fi

    exit ${exit_code}
}

# Trap Ctrl+C and cleanup
trap 'log_error "Script interrupted"; exit 130' INT

# Run main function
main "\$@"
