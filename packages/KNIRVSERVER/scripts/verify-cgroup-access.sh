#!/bin/bash

# Cgroup Access Verification Script
# This script verifies that the system has proper cgroup v2 access
# for KNIRVSERVER container isolation features

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

CGROUP_BASE="/sys/fs/cgroup"
TEST_CGROUP="knirv-test-$$"
ERRORS=0
WARNINGS=0

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
    ((WARNINGS++))
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
    ((ERRORS++))
}

echo "========================================="
echo "  KNIRVSERVER Cgroup Access Verification"
echo "========================================="
echo ""

# 1. Check if cgroup v2 is available
log_info "Checking cgroup v2 support..."
if [ -d "$CGROUP_BASE" ]; then
    log_success "Cgroup directory exists: $CGROUP_BASE"
else
    log_error "Cgroup directory not found: $CGROUP_BASE"
    log_error "Ensure cgroup v2 is enabled in your kernel"
    exit 1
fi

# 2. Check if it's cgroup v2 unified hierarchy
log_info "Checking cgroup version..."
if [ -f "$CGROUP_BASE/cgroup.controllers" ]; then
    log_success "Cgroup v2 unified hierarchy detected"
else
    log_warning "Cgroup v2 unified hierarchy not detected"
    log_info "You may be using cgroup v1 or hybrid mode"
fi

# 3. Check mount options
log_info "Checking cgroup mount options..."
MOUNT_INFO=$(mount | grep "cgroup" | grep "$CGROUP_BASE" || true)
if [ -z "$MOUNT_INFO" ]; then
    log_warning "Cgroup not found in mount table"
else
    echo "$MOUNT_INFO"
    if echo "$MOUNT_INFO" | grep -q "rw"; then
        log_success "Cgroup filesystem mounted as READ-WRITE"
    else
        log_error "Cgroup filesystem mounted as READ-ONLY"
        log_error "Cannot create cgroups without write access"
    fi
fi

# 4. Test write access
log_info "Testing cgroup write access..."
TEST_PATH="$CGROUP_BASE/$TEST_CGROUP"

if mkdir -p "$TEST_PATH" 2>/dev/null; then
    log_success "Successfully created test cgroup: $TEST_CGROUP"

    # Clean up
    if rmdir "$TEST_PATH" 2>/dev/null; then
        log_success "Successfully removed test cgroup"
    else
        log_warning "Could not remove test cgroup (may need to kill processes first)"
    fi
else
    log_error "Cannot create cgroup directory"
    log_error "Possible causes:"
    echo "  - Running without sufficient privileges (need root or CAP_SYS_ADMIN)"
    echo "  - Cgroup filesystem mounted as read-only"
    echo "  - SELinux/AppArmor blocking access"
fi

# 5. Check available controllers
log_info "Checking available cgroup controllers..."
if [ -f "$CGROUP_BASE/cgroup.controllers" ]; then
    CONTROLLERS=$(cat "$CGROUP_BASE/cgroup.controllers")
    echo "Available controllers: $CONTROLLERS"

    # Check for required controllers
    REQUIRED_CONTROLLERS=("cpu" "memory" "pids" "io")
    for ctrl in "${REQUIRED_CONTROLLERS[@]}"; do
        if echo "$CONTROLLERS" | grep -qw "$ctrl"; then
            log_success "Controller '$ctrl' available"
        else
            log_warning "Controller '$ctrl' not available"
        fi
    done
else
    log_warning "Cannot read cgroup.controllers file"
fi

# 6. Check delegated controllers
log_info "Checking delegated controllers..."
if [ -f "$CGROUP_BASE/cgroup.subtree_control" ]; then
    SUBTREE_CTRL=$(cat "$CGROUP_BASE/cgroup.subtree_control" || echo "")
    if [ -z "$SUBTREE_CTRL" ]; then
        log_warning "No controllers delegated to subtree"
        log_info "You may need to delegate controllers for nested cgroups"
    else
        echo "Delegated controllers: $SUBTREE_CTRL"
        log_success "Controllers are delegated to subtree"
    fi
fi

# 7. Check user capabilities (if not root)
log_info "Checking user capabilities..."
if [ "$(id -u)" -eq 0 ]; then
    log_success "Running as root (uid=0)"
else
    log_warning "Not running as root (uid=$(id -u))"
    log_info "Cgroup operations may require:"
    echo "  - CAP_SYS_ADMIN capability"
    echo "  - CAP_DAC_OVERRIDE for bypassing file permissions"
    echo "  - Cgroup delegation to user namespaces"

    # Check if we have the required capabilities
    if command -v capsh &> /dev/null; then
        CAPS=$(capsh --print | grep "Current:" || echo "")
        if echo "$CAPS" | grep -q "cap_sys_admin"; then
            log_success "CAP_SYS_ADMIN capability present"
        else
            log_warning "CAP_SYS_ADMIN capability not found"
        fi
    fi
fi

# 8. Check for KNIRV-specific cgroup paths
log_info "Checking KNIRV cgroup paths..."
KNIRV_PATHS=("/sys/fs/cgroup/knirv-server" "/sys/fs/cgroup/knirv-objects")
for path in "${KNIRV_PATHS[@]}"; do
    if [ -d "$path" ]; then
        log_success "KNIRV cgroup path exists: $path"
    else
        log_info "KNIRV cgroup path not yet created: $path"
        log_info "Will be created on first use"
    fi
done

# 9. System information
echo ""
log_info "System Information:"
echo "Kernel version: $(uname -r)"
echo "Kernel cmdline: $(cat /proc/cmdline 2>/dev/null | grep -o 'cgroup[^ ]*' || echo 'No cgroup parameters')"

# 10. Summary
echo ""
echo "========================================="
echo "  Verification Summary"
echo "========================================="
if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    log_success "All checks passed! Cgroup access is properly configured."
    echo ""
    echo "You can now run KNIRVSERVER with full cgroup isolation."
    exit 0
elif [ $ERRORS -eq 0 ]; then
    log_warning "Verification completed with $WARNINGS warning(s)"
    echo ""
    echo "Cgroup access is available but with some limitations."
    echo "Review warnings above for potential issues."
    exit 0
else
    log_error "Verification failed with $ERRORS error(s) and $WARNINGS warning(s)"
    echo ""
    echo "Cgroup access is not properly configured."
    echo "Please address the errors above before running KNIRVSERVER."
    echo ""
    echo "Common fixes:"
    echo "1. For Docker containers:"
    echo "   docker run --privileged --cgroupns=host -v /sys/fs/cgroup:/sys/fs/cgroup:rw ..."
    echo ""
    echo "2. For Kata containers:"
    echo "   Ensure 'privileged_without_host_devices = true' in Kata configuration"
    echo ""
    echo "3. For bare metal:"
    echo "   - Run as root or with CAP_SYS_ADMIN capability"
    echo "   - Ensure cgroup v2 is enabled"
    echo "   - Check SELinux/AppArmor policies"
    exit 1
fi
