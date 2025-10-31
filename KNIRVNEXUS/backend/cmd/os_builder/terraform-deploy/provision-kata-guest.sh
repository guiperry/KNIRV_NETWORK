#!/bin/bash
# provision-kata-guest.sh
# Terraform-based provisioning for Kata guest kernel and rootfs
# This script is called by Terraform and logs all output to stdout/stderr

set -euo pipefail

echo "[PROVISIONING] Starting Kata guest provisioning..."
START_TIME=$(date +%s)

# Variables passed from Terraform
SSH_USERNAME="${ssh_username}"
SSH_PASSWORD="${ssh_password}"
SSH_HOST="${ssh_host}"
SSH_PORT="${ssh_port}"
OUTPUT_DIRECTORY="${output_directory}"

# Create output directory
mkdir -p "$OUTPUT_DIRECTORY"
chmod 755 "$OUTPUT_DIRECTORY"

echo "[PROVISIONING] Output directory: $OUTPUT_DIRECTORY"
echo "[PROVISIONING] SSH Target: $SSH_USERNAME@$SSH_HOST:$SSH_PORT"

# SSH connection helper function
ssh_cmd() {
  sshpass -e ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -o ConnectTimeout=120 -o ServerAliveInterval=60 -o ServerAliveCountMax=10 \
    "$SSH_USERNAME@$SSH_HOST" -p "$SSH_PORT" "$@"
}

# SCP helper function
scp_from_vm() {
  sshpass -e scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -o ConnectTimeout=120 -o ServerAliveInterval=60 -o ServerAliveCountMax=10 \
    -P "$SSH_PORT" "$SSH_USERNAME@$SSH_HOST:$1" "$2"
}

# Test basic connectivity
echo "[PROVISIONING] Testing basic connectivity..."
ssh_cmd "echo 'SSH connection working'"

# Configure APT with retries
echo "[PROVISIONING] Configuring APT..."
ssh_cmd "sudo bash -c 'mkdir -p /etc/apt/apt.conf.d && chmod 0755 /etc/apt/apt.conf.d && \
  cat > /etc/apt/apt.conf.d/99-terraform-fixes << \"EOF\"
APT::Acquire::Retries \"5\";
APT::Acquire::http::Timeout \"120\";
APT::Acquire::https::Timeout \"120\";
EOF'"

# Install build prerequisites
echo "[PROVISIONING] Installing build prerequisites..."
ssh_cmd "export DEBIAN_FRONTEND=noninteractive && \
  sudo apt-get update -qq && sudo apt-get install -y \
  build-essential libncurses5-dev fakeroot xz-utils git debootstrap cpio wget \
  flex bison bc pahole libelf-dev libssl-dev" || \
  (sleep 10 && ssh_cmd "export DEBIAN_FRONTEND=noninteractive && \
  sudo apt-get update -qq && sudo apt-get install -y \
  build-essential libncurses5-dev fakeroot xz-utils git debootstrap cpio wget \
  flex bison bc pahole libelf-dev libssl-dev")

# Get kernel version
echo "[PROVISIONING] Getting kernel version..."
KERNEL_VERSION=$(ssh_cmd "uname -r" | tr -d '\r\n')
echo "[PROVISIONING] Detected kernel version: $KERNEL_VERSION"

# Extract major.minor version for Kali linux-source package (e.g., 6.16 from 6.16.8+kali-amd64)
KERNEL_SOURCE_VERSION=$(echo "$KERNEL_VERSION" | sed -E 's/^([0-9]+\.[0-9]+).*/\1/' | tr -d '\n')
echo "[PROVISIONING] Kernel source version for package: $KERNEL_SOURCE_VERSION"

# Install kernel sources
echo "[PROVISIONING] Installing kernel sources (this may take 10+ minutes)..."
ssh_cmd "export DEBIAN_FRONTEND=noninteractive && \
  sudo mkdir -p /usr/src && sudo chmod 0755 /usr/src && \
  sudo apt-get update -qq && \
  (sudo apt-get install -y linux-source-$KERNEL_SOURCE_VERSION || \
   (sleep 10 && sudo apt-get update -qq && sudo apt-get install -y linux-source-$KERNEL_SOURCE_VERSION))"

# Extract kernel sources
echo "[PROVISIONING] Extracting kernel sources..."
ssh_cmd "cd /usr/src && \
  (sudo tar -xf linux-source-$KERNEL_SOURCE_VERSION.tar.xz 2>/dev/null || \
   sudo tar -xJf linux-source-$KERNEL_SOURCE_VERSION.tar.xz) && \
  sudo rm -f /usr/src/linux && \
  sudo ln -s /usr/src/linux-source-$KERNEL_SOURCE_VERSION /usr/src/linux"

# Copy kernel config
echo "[PROVISIONING] Copying kernel config..."
ssh_cmd "sudo cp /boot/config-$KERNEL_VERSION /usr/src/linux/.config && \
  sudo chmod 644 /usr/src/linux/.config"

# Enable TEE configs
echo "[PROVISIONING] Enabling TEE related kernel configs..."
ssh_cmd "cd /usr/src/linux && \
  sudo bash -c 'sed -i \"s/^# CONFIG_SECURITY_LOCKDOWN_LSM.*/CONFIG_SECURITY_LOCKDOWN_LSM=y/\" .config && \
  sed -i \"s/^# CONFIG_INTEL_TXT.*/CONFIG_INTEL_TXT=y/\" .config && \
  sed -i \"s/^# CONFIG_AMD_MEM_ENCRYPT.*/CONFIG_AMD_MEM_ENCRYPT=y/\" .config && \
  grep -q CONFIG_SECURITY_LOCKDOWN_LSM .config || echo CONFIG_SECURITY_LOCKDOWN_LSM=y >> .config && \
  grep -q CONFIG_INTEL_TXT .config || echo CONFIG_INTEL_TXT=y >> .config && \
  grep -q CONFIG_AMD_MEM_ENCRYPT .config || echo CONFIG_AMD_MEM_ENCRYPT=y >> .config'"

# Prepare kernel for build
echo "[PROVISIONING] Preparing kernel (olddefconfig)..."
ssh_cmd "cd /usr/src/linux && sudo make olddefconfig"

# Build kernel (this is long-running)
echo "[PROVISIONING] Building custom Kata kernel (this may take 30+ minutes)..."
ssh_cmd "cd /usr/src/linux && sudo make -j\$(nproc) bzImage modules" || \
  { echo "[ERROR] Kernel build failed"; exit 1; }

# Create rootfs
echo "[PROVISIONING] Creating rootfs directory..."
ssh_cmd "sudo mkdir -p /tmp/kata-rootfs && sudo chmod 0755 /tmp/kata-rootfs"

# Bootstrap Kali rootfs
echo "[PROVISIONING] Bootstrapping minimal Kali rootfs (this may take 10+ minutes)..."
ssh_cmd "(timeout 1800 sudo debootstrap --variant=minbase kali-rolling /tmp/kata-rootfs http://http.kali.org/kali) || \
  (sleep 30 && timeout 1800 sudo debootstrap --variant=minbase kali-rolling /tmp/kata-rootfs http://http.kali.org/kali)" || \
  { echo "[ERROR] Rootfs bootstrap failed"; exit 1; }

# Configure rootfs
echo "[PROVISIONING] Configuring rootfs..."
ssh_cmd "sudo bash -c 'chroot /tmp/kata-rootfs apt-get update -qq 2>/dev/null || true && \
  chroot /tmp/kata-rootfs apt-get install -y systemd-sysv curl gnupg 2>/dev/null || true && \
  chroot /tmp/kata-rootfs apt-get clean 2>/dev/null || true && \
  chroot /tmp/kata-rootfs rm -rf /var/lib/apt/lists/* 2>/dev/null || true'"

# Create initramfs
echo "[PROVISIONING] Creating initramfs from rootfs..."
ssh_cmd "cd /tmp/kata-rootfs && \
  sudo find . -print0 | sudo cpio --null --format=newc -o | sudo gzip -9 > /tmp/kata-rootfs.img" || \
  ssh_cmd "cd /tmp/kata-rootfs && \
  sudo find . -print0 | sudo cpio --null --format=newc -o > /tmp/kata-rootfs.img"

# Copy artifacts back to host
echo "[PROVISIONING] Copying artifacts back to host..."
scp_from_vm "/usr/src/linux/arch/x86_64/boot/bzImage" "$OUTPUT_DIRECTORY/vmlinuz-kali-clean-tee" || \
  { echo "[ERROR] Failed to copy kernel"; exit 1; }

scp_from_vm "/tmp/kata-rootfs.img" "$OUTPUT_DIRECTORY/kata-rootfs-kali-clean-tee.img" || \
  { echo "[ERROR] Failed to copy rootfs"; exit 1; }

# Verify artifacts
echo "[PROVISIONING] Verifying artifacts..."
if [ -f "$OUTPUT_DIRECTORY/vmlinuz-kali-clean-tee" ]; then
  echo "[SUCCESS] Kernel artifact: $(ls -lh $OUTPUT_DIRECTORY/vmlinuz-kali-clean-tee)"
else
  echo "[ERROR] Kernel artifact not found!"
  exit 1
fi

if [ -f "$OUTPUT_DIRECTORY/kata-rootfs-kali-clean-tee.img" ]; then
  echo "[SUCCESS] Rootfs artifact: $(ls -lh $OUTPUT_DIRECTORY/kata-rootfs-kali-clean-tee.img)"
else
  echo "[ERROR] Rootfs artifact not found!"
  exit 1
fi

# Cleanup on VM
echo "[PROVISIONING] Cleaning up temporary files on VM..."
ssh_cmd "sudo bash -c 'rm -rf /usr/src/linux /usr/src/linux-source-* /tmp/kata-rootfs /tmp/kata-rootfs.img && sync'" || true

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
MINUTES=$((DURATION / 60))
SECONDS=$((DURATION % 60))
echo "[PROVISIONING] Provisioning completed in $MINUTES minutes and $SECONDS seconds"
echo "[SUCCESS] Kata guest provisioning finished!"