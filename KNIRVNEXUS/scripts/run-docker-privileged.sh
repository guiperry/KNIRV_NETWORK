#!/bin/bash

# KNIRVNEXUS Docker Runner with Full Cgroup Access
# This script runs the KNIRVNEXUS container with full filesystem privileges
# required for cgroup and namespace manipulation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KNIRVNEXUS_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONTAINER_NAME="knirvnexus-kali-local"
IMAGE_NAME="knirvnexus-test:latest"
BUILD_DOCKERFILE="${KNIRVNEXUS_ROOT}/Dockerfile.test"

# Parse command line arguments
BUILD_IMAGE=false
RUN_INTERACTIVE=false
RUN_TESTS=false
CLEANUP=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --build)
            BUILD_IMAGE=true
            shift
            ;;
        --interactive|-i)
            RUN_INTERACTIVE=true
            shift
            ;;
        --test)
            RUN_TESTS=true
            shift
            ;;
        --cleanup)
            CLEANUP=true
            shift
            ;;
        --help|-h)
            cat <<EOF
Usage: $0 [OPTIONS]

Options:
    --build              Build the test Docker image
    --interactive, -i    Run container interactively
    --test               Run privileged tests
    --cleanup            Remove container after completion
    --help, -h           Show this help message

Examples:
    # Build and run interactively
    $0 --build --interactive

    # Run tests in privileged container
    $0 --test

    # Build, test, and cleanup
    $0 --build --test --cleanup

EOF
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    log_success "Docker found: $(docker --version)"
}

# Build Docker image
build_image() {
    log_info "Building Docker test image..."

    cd "${KNIRVNEXUS_ROOT}"

    if ! docker build -f "${BUILD_DOCKERFILE}" -t "${IMAGE_NAME}" .; then
        log_error "Failed to build Docker image"
        exit 1
    fi

    log_success "Docker image built: ${IMAGE_NAME}"
}

# Stop and remove existing container
cleanup_existing() {
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_warning "Removing existing container: ${CONTAINER_NAME}"
        docker rm -f "${CONTAINER_NAME}" || true
    fi
}

# Run container with full privileges
run_container() {
    log_info "Running container with full cgroup access..."

    # Docker run flags explanation:
    # --privileged: Give extended privileges to this container
    # --cgroupns=host: Use host's cgroup namespace
    # --pid=host: Use host's PID namespace (optional, for debugging)
    # -v /sys/fs/cgroup:/sys/fs/cgroup:rw: Mount cgroup filesystem as writable
    # --cap-add=ALL: Add all Linux capabilities
    # --security-opt apparmor=unconfined: Disable AppArmor
    # --security-opt seccomp=unconfined: Disable seccomp

    local DOCKER_FLAGS=(
        --name "${CONTAINER_NAME}"
        --privileged
        --cgroupns=host
        -v /sys/fs/cgroup:/sys/fs/cgroup:rw
        --cap-add=ALL
        --security-opt apparmor=unconfined
        --security-opt seccomp=unconfined
        -v "${KNIRVNEXUS_ROOT}":/workspace/KNIRVNEXUS:rw
        -w /workspace/KNIRVNEXUS
    )

    if [ "$RUN_INTERACTIVE" = true ]; then
        log_info "Running in interactive mode..."
        docker run -it --rm "${DOCKER_FLAGS[@]}" "${IMAGE_NAME}" /bin/bash
    elif [ "$RUN_TESTS" = true ]; then
        log_info "Running privileged tests..."
        docker run "${DOCKER_FLAGS[@]}" -d "${IMAGE_NAME}" tail -f /dev/null

        # Wait for container to be ready
        sleep 2

        # Run tests inside container
        log_info "Executing tests inside container..."
        docker exec "${CONTAINER_NAME}" bash -c "
            set -e
            cd /workspace/KNIRVNEXUS/backend

            echo '=== Cgroup Verification ==='
            ls -la /sys/fs/cgroup/ || echo 'Cannot list cgroup directory'
            mount | grep cgroup || echo 'No cgroup mounts found'
            echo ''

            echo '=== Testing cgroup write access ==='
            mkdir -p /sys/fs/cgroup/knirv-test && \
                echo 'Cgroup filesystem is writable' && \
                rmdir /sys/fs/cgroup/knirv-test || \
                echo 'WARNING: Cgroup filesystem is READ-ONLY'
            echo ''

            echo '=== Running Go tests ==='
            go mod download
            go mod tidy

            # Run privileged tests
            go test -v -timeout=30m \
                -run 'TestNewCgroupManager|TestCgroupManager|TestNamespaceManager|TestNamespace' \
                ./internal/services/teesecurity/... \
                2>&1
        "

        if [ "$CLEANUP" = true ]; then
            log_info "Cleaning up container..."
            docker rm -f "${CONTAINER_NAME}"
        else
            log_success "Container left running. To stop: docker stop ${CONTAINER_NAME}"
            log_info "To enter container: docker exec -it ${CONTAINER_NAME} bash"
        fi
    else
        log_info "Starting container in background..."
        docker run -d "${DOCKER_FLAGS[@]}" "${IMAGE_NAME}" tail -f /dev/null
        log_success "Container started: ${CONTAINER_NAME}"
        log_info "To enter container: docker exec -it ${CONTAINER_NAME} bash"
        log_info "To stop container: docker stop ${CONTAINER_NAME}"
    fi
}

# Verify cgroup access
verify_cgroup_access() {
    log_info "Verifying cgroup access in container..."

    if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_error "Container is not running"
        return 1
    fi

    docker exec "${CONTAINER_NAME}" bash -c "
        echo 'Checking cgroup filesystem...'
        if mount | grep -q 'cgroup.*rw'; then
            echo 'Cgroup filesystem is mounted as READ-WRITE ✓'
        else
            echo 'WARNING: Cgroup filesystem may be read-only'
        fi

        echo ''
        echo 'Testing write access...'
        if mkdir -p /sys/fs/cgroup/knirv-test 2>/dev/null; then
            echo 'Successfully created test cgroup directory ✓'
            rmdir /sys/fs/cgroup/knirv-test
        else
            echo 'ERROR: Cannot create cgroup directory (read-only)'
            exit 1
        fi
    "
}

# Main execution
main() {
    log_info "KNIRVNEXUS Docker Runner with Full Privileges"

    check_docker

    if [ "$BUILD_IMAGE" = true ]; then
        build_image
    fi

    cleanup_existing
    run_container

    if [ "$RUN_TESTS" = false ] && [ "$RUN_INTERACTIVE" = false ]; then
        verify_cgroup_access
    fi

    log_success "Done!"
}

main
