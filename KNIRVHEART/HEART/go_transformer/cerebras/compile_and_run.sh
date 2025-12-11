#!/bin/bash
# Quick-start script for HEART Cerebras implementation
# This script provides an easy way to compile, test, and run the HEART transformer

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print banner
echo -e "${BLUE}"
echo "═══════════════════════════════════════════════════════════"
echo "  HEART Transformer - Cerebras WSE Quick Start"
echo "  Heuristic Error Analysis and Recognition Transformer"
echo "═══════════════════════════════════════════════════════════"
echo -e "${NC}"

# Function to print status
print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[ℹ]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

# Check prerequisites
print_info "Checking prerequisites..."

if ! command -v cslc &> /dev/null; then
    print_error "cslc not found. Please install Cerebras SDK."
    exit 1
fi
print_status "Cerebras SDK (cslc) found"

if ! command -v cs_python &> /dev/null; then
    print_error "cs_python not found. Please install Cerebras SDK."
    exit 1
fi
print_status "Cerebras Python (cs_python) found"

if ! command -v jq &> /dev/null; then
    print_warning "jq not found. Install with: sudo apt-get install jq"
fi

# Parse command line arguments
MODE=${1:-"test"}  # Default to test mode

case $MODE in
    "compile")
        print_info "Compilation mode selected"
        make compile
        print_status "Compilation complete!"
        ;;

    "init")
        print_info "Initialization mode: Setting up weights and test data"
        make init-weights
        make test-data
        print_status "Initialization complete!"
        ;;

    "test")
        print_info "Test mode: Full compilation and simulator test"

        # Clean previous builds
        print_info "Cleaning previous builds..."
        make clean

        # Compile CSL program
        print_info "Compiling CSL program..."
        make compile
        print_status "CSL program compiled"

        # Initialize weights
        print_info "Initializing random weights for testing..."
        make init-weights
        print_status "Weights initialized"

        # Run test
        print_info "Running on fabric simulator..."
        make run-simulator
        print_status "Test passed!"

        # Verify output
        print_info "Verifying output..."
        make verify-output
        ;;

    "device")
        print_info "Device mode: Running on actual Cerebras WSE"

        if [ -z "$CEREBRAS_CMADDR" ]; then
            print_error "CEREBRAS_CMADDR environment variable not set!"
            print_info "Set it with: export CEREBRAS_CMADDR='ip:port'"
            exit 1
        fi

        print_info "Cerebras system: $CEREBRAS_CMADDR"

        # Compile if needed
        if [ ! -d "build/heart_program" ]; then
            print_info "No compiled program found. Compiling..."
            make compile
        fi

        # Check for weights
        if [ ! -f "weights/model_weights.npz" ]; then
            print_warning "No weights found. Using random initialization."
            make init-weights
        fi

        # Run on device
        print_info "Launching on Cerebras WSE..."
        make run-device
        print_status "Execution complete!"
        ;;

    "benchmark")
        print_info "Benchmark mode: Performance testing"

        print_info "Compiling optimized build..."
        make compile

        print_info "Running benchmark suite..."
        for i in {1..5}; do
            echo -e "${BLUE}Iteration $i/5${NC}"
            make run-simulator 2>&1 | grep -E "(Launching|complete|elapsed)"
        done

        print_status "Benchmark complete!"
        ;;

    "clean")
        print_info "Cleaning all build artifacts..."
        make clean-all
        print_status "Clean complete!"
        ;;

    "info")
        print_info "HEART Configuration:"
        make info
        ;;

    "help"|*)
        echo "Usage: $0 [mode]"
        echo ""
        echo "Modes:"
        echo "  compile   - Compile CSL program only"
        echo "  init      - Initialize weights and test data"
        echo "  test      - Full test on simulator (default)"
        echo "  device    - Run on actual Cerebras WSE (requires CEREBRAS_CMADDR)"
        echo "  benchmark - Run performance benchmarks"
        echo "  clean     - Clean all build artifacts"
        echo "  info      - Show configuration information"
        echo "  help      - Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0 test                    # Run full test on simulator"
        echo "  $0 device                  # Run on WSE device"
        echo "  CEREBRAS_CMADDR=10.0.0.1:9000 $0 device"
        echo ""
        exit 0
        ;;
esac

echo ""
print_status "All operations completed successfully!"
echo ""
