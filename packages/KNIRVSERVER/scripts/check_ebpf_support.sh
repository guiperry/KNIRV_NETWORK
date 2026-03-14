#!/bin/bash
# scripts/check_ebpf_support.sh

echo "Checking eBPF support..."

# Kernel version
KERNEL_VERSION=$(uname -r | cut -d. -f1-2)
echo "Kernel version: $KERNEL_VERSION"

# BTF support (required for CO-RE)
if [ -f /sys/kernel/btf/vmlinux ]; then
    echo "✓ BTF support enabled"
else
    echo "✗ BTF support missing (required for CO-RE)"
fi

# LSM BPF
if grep -q "bpf" /sys/kernel/security/lsm 2>/dev/null; then
    echo "✓ LSM BPF enabled"
else
    echo "✗ LSM BPF not enabled"
    echo "  Add 'lsm=...,bpf' to kernel boot parameters"
fi

# XDP support
if [ -d /sys/class/net/eth0/xdp ]; then
    echo "✓ XDP support available"
else
    echo "⚠ XDP support unclear (check network driver)"
fi