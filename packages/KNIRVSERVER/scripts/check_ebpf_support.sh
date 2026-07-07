#!/bin/bash
# scripts/check_ebpf_support.sh

echo "Checking eBPF support..."

MULTIARCH_DIR=""
if command -v dpkg-architecture >/dev/null 2>&1; then
    MULTIARCH_DIR="$(dpkg-architecture -qDEB_HOST_MULTIARCH 2>/dev/null)"
else
    case "$(uname -m)" in
        x86_64) MULTIARCH_DIR="x86_64-linux-gnu" ;;
        aarch64|arm64) MULTIARCH_DIR="aarch64-linux-gnu" ;;
        armv7l|armv7*) MULTIARCH_DIR="arm-linux-gnueabihf" ;;
        riscv64) MULTIARCH_DIR="riscv64-linux-gnu" ;;
    esac
fi

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

if [ -n "$MULTIARCH_DIR" ] && [ -f "/usr/include/$MULTIARCH_DIR/asm/types.h" ]; then
    echo "✓ Arch-specific UAPI headers found at /usr/include/$MULTIARCH_DIR"
else
    echo "✗ Arch-specific UAPI headers missing (/usr/include/*/asm/types.h)"
    echo "  Install linux-libc-dev or equivalent so clang can resolve asm/types.h"
fi
