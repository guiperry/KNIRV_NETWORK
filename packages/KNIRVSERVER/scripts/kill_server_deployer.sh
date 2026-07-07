#!/bin/bash

################################################################################
# kill_nexus_deployer.sh
# 
# Emergency cleanup script for KNIRV Nexus Deployer hangs
# Kills all background processes spawned by the continer-deployer Go program
# including Packer, VirtualBox VMs, Ansible, and SSH connections
#
# Usage: ./kill_nexus_deployer.sh [--force] [--vms-only]
# 
# Options:
#   --force      Skip graceful shutdown, force kill immediately
#   --vms-only   Only kill VirtualBox VMs, don't kill deployer processes
#   --verbose    Show detailed process information before killing
################################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script flags
FORCE_KILL=false
VMS_ONLY=false
VERBOSE=false
GRACE_PERIOD=5

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --force)
            FORCE_KILL=true
            GRACE_PERIOD=0
            shift
            ;;
        --vms-only)
            VMS_ONLY=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--force] [--vms-only] [--verbose]"
            exit 1
            ;;
    esac
done

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to kill processes by name
kill_process_by_name() {
    local proc_name=$1
    local signal=${2:-TERM}
    
    log_info "Searching for '$proc_name' processes..."
    
    # Find PIDs matching the process name
    local pids=$(pgrep -f "$proc_name" 2>/dev/null || true)
    
    if [ -z "$pids" ]; then
        log_warning "No processes found matching '$proc_name'"
        return 0
    fi
    
    # Count PIDs
    local count=$(echo "$pids" | wc -l)
    log_warning "Found $count process(es) matching '$proc_name': $pids"
    
    if [ "$VERBOSE" = true ]; then
        echo "$pids" | while read -r pid; do
            if ps -p "$pid" > /dev/null 2>&1; then
                local cmd=$(ps -p "$pid" -o cmd= 2>/dev/null || echo "unknown")
                log_info "  PID $pid: $cmd"
            fi
        done
    fi
    
    # Kill each PID
    echo "$pids" | while read -r pid; do
        if [ -z "$pid" ]; then
            continue
        fi
        
        if ps -p "$pid" > /dev/null 2>&1; then
            log_warning "Killing PID $pid with signal $signal..."
            kill -$signal "$pid" 2>/dev/null || log_error "Failed to send signal $signal to PID $pid"
        fi
    done
    
    # If using TERM, wait a bit then force kill any stragglers
    if [ "$signal" = "TERM" ] && [ "$GRACE_PERIOD" -gt 0 ]; then
        log_info "Waiting ${GRACE_PERIOD}s for graceful shutdown..."
        sleep "$GRACE_PERIOD"
        
        local remaining=$(pgrep -f "$proc_name" 2>/dev/null || true)
        if [ -n "$remaining" ]; then
            log_warning "Some processes still running, force killing..."
            echo "$remaining" | while read -r pid; do
                if [ -z "$pid" ]; then
                    continue
                fi
                if ps -p "$pid" > /dev/null 2>&1; then
                    kill -9 "$pid" 2>/dev/null || true
                fi
            done
        fi
    fi
}

# Function to kill VirtualBox VMs
kill_virtualbox_vms() {
    log_info "Checking for VirtualBox VMs..."
    
    if ! command -v VBoxManage &> /dev/null; then
        log_warning "VBoxManage not found, skipping VM cleanup"
        return 0
    fi
    
    # List all running VMs
    local running_vms=$(VBoxManage list runningvms 2>/dev/null | grep -o '"[^"]*"' | tr -d '"' || true)
    
    if [ -z "$running_vms" ]; then
        log_success "No running VirtualBox VMs found"
        return 0
    fi
    
    log_warning "Found running VirtualBox VMs:"
    echo "$running_vms" | while read -r vm; do
        log_warning "  - $vm"
    done
    
    # Kill each VM
    echo "$running_vms" | while read -r vm; do
        if [ -z "$vm" ]; then
            continue
        fi
        log_warning "Stopping VM: $vm"
        VBoxManage controlvm "$vm" acpipower off 2>/dev/null || \
            VBoxManage controlvm "$vm" poweroff 2>/dev/null || \
            log_error "Failed to stop VM: $vm"
    done
    
    # Wait a moment for VMs to shut down
    sleep 2
    
    # Force kill any remaining VBoxHeadless processes
    log_info "Cleaning up VBoxHeadless processes..."
    local vbox_pids=$(pgrep -f VBoxHeadless 2>/dev/null || true)
    if [ -n "$vbox_pids" ]; then
        echo "$vbox_pids" | while read -r pid; do
            if [ -z "$pid" ]; then
                continue
            fi
            log_warning "Force killing VBoxHeadless PID $pid"
            kill -9 "$pid" 2>/dev/null || true
        done
    fi
}

# Main execution
main() {
    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║         KNIRV Nexus Deployer Emergency Cleanup Script         ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    
    log_info "Starting cleanup process..."
    log_info "Force kill: $FORCE_KILL"
    log_info "VMs only: $VMS_ONLY"
    log_info "Verbose: $VERBOSE"
    echo ""
    
    # Kill VirtualBox VMs (always do this first)
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    kill_virtualbox_vms
    echo ""
    
    # If --vms-only flag is set, stop here
    if [ "$VMS_ONLY" = true ]; then
        log_success "VM cleanup complete"
        exit 0
    fi
    
    # Kill deployer processes
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Killing continer-deployer Go processes..."
    kill_process_by_name "continer-deployer" "$([ "$FORCE_KILL" = true ] && echo '9' || echo 'TERM')"
    echo ""
    
    # Kill Packer
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Killing Packer processes..."
    kill_process_by_name "packer" "$([ "$FORCE_KILL" = true ] && echo '9' || echo 'TERM')"
    echo ""
    
    # Kill Ansible
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Killing Ansible processes..."
    kill_process_by_name "ansible" "$([ "$FORCE_KILL" = true ] && echo '9' || echo 'TERM')"
    echo ""
    
    # Kill SSH connections
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Killing SSH/sshpass processes..."
    kill_process_by_name "sshpass" "9"
    kill_process_by_name "ssh.*packer" "TERM"
    echo ""
    
    # Kill any remaining Python processes from Ansible
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Killing Python/Ansible processes..."
    local python_pids=$(pgrep -f "python.*ansible" 2>/dev/null || true)
    if [ -n "$python_pids" ]; then
        echo "$python_pids" | while read -r pid; do
            if [ -z "$pid" ]; then
                continue
            fi
            if ps -p "$pid" > /dev/null 2>&1; then
                log_warning "Killing Python PID $pid"
                kill -9 "$pid" 2>/dev/null || true
            fi
        done
    else
        log_warning "No Python/Ansible processes found"
    fi
    echo ""
    
    # Final verification
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Verifying cleanup..."
    
    local remaining_deployer=$(pgrep -f "continer-deployer" 2>/dev/null || true)
    local remaining_packer=$(pgrep -f "packer" 2>/dev/null || true)
    local remaining_ansible=$(pgrep -f "ansible" 2>/dev/null || true)
    local remaining_vms=$(VBoxManage list runningvms 2>/dev/null | wc -l || echo "0")
    
    echo ""
    if [ -z "$remaining_deployer" ] && [ -z "$remaining_packer" ] && [ -z "$remaining_ansible" ] && [ "$remaining_vms" -lt 2 ]; then
        log_success "╔════════════════════════════════════════════════════════════════╗"
        log_success "║                  CLEANUP SUCCESSFUL                            ║"
        log_success "║          All processes and VMs have been terminated            ║"
        log_success "╚════════════════════════════════════════════════════════════════╝"
        echo ""
        log_success "You can now restart the deployer with:"
        log_success "  cd /home/gperry/Documents/GitHub/cloud-equities/KNIRV_NETWORK/KNIRVSERVER/backend_server/cmd/continer-deployer"
        log_success "  ./continer-deployer"
        echo ""
        exit 0
    else
        log_error "╔════════════════════════════════════════════════════════════════╗"
        log_error "║          CLEANUP INCOMPLETE - PROCESSES STILL RUNNING          ║"
        log_error "╚════════════════════════════════════════════════════════════════╝"
        echo ""
        
        [ -n "$remaining_deployer" ] && log_error "Deployer still running: $remaining_deployer"
        [ -n "$remaining_packer" ] && log_error "Packer still running: $remaining_packer"
        [ -n "$remaining_ansible" ] && log_error "Ansible still running: $remaining_ansible"
        [ "$remaining_vms" -gt 1 ] && log_error "VirtualBox VMs still running"
        
        echo ""
        log_warning "Try running again with --force flag:"
        log_warning "  $0 --force"
        echo ""
        exit 1
    fi
}

# Run main function
main