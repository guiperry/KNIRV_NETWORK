# KNIRV-NEXUS Kali Linux Foundation

## Overview

KNIRV-NEXUS is built on a Kali Linux foundation, providing a secure and robust platform for TEE-enabled agent operations. This document outlines the Kali Linux implementation details and integration with the Go-based KNIRV-NEXUS application.

Here is the Kali LInux link: https://www.kali.org/docs/development/recompiling-the-kali-linux-kernel/ Recompiling the Kali Linux Kernel | Kali Linux Documentation Recompiling the Kali Linux Kernel | Kali Linux Documentation 

## 🎯 Deployment Guide for the KNIRV-NEXUS DVE

### Executive Summary
> **Use the latest 2024.3 **rolling** Kali Linux **net-installer** image as the canonical upstream for the CLEAN fork.**  
> Strip it down to a **server-only, CLI-only, hardened profile** and **lock the kernel to the 6.6.x LTS tree** that Kali 2024.3 currently ships.  
> Re-compile that kernel once with the minimal set of hardening patches shown in the attached documentation.

---

### 🔍 Rationale (mapped to white-paper requirements)

| Requirement in White-paper | Why 2024.3 Rolling + 6.6.x LTS is the best fit |
|----------------------------|--------------------------------------------------|
| **Proactive security posture** (§4.2) | Rolling branch receives security patches daily; 6.6.x is a **Long-Term Support** kernel, minimizing future re-base effort. |
| **Minimalist, server-only profile** (§4.2) | Net-installer lets us cherry-pick **exactly** openssh-server, ca-certificates, containerd, nftables, and nothing else—no GUI, no kali-linux-* metapackages. |
| **Continuous self-auditing** (§4.2) | All offensive/defensive tools are still in the repos (`apt install kali-linux-headless`) but can be **added on demand** inside the TEE without bloating the base image. |
| **Deterministic builds & remote attestation** (§5) | The 6.6.x kernel config (`/boot/config-6.6.*-kali1-amd64`) is already reproducible; we copy it verbatim, add our 4 extra hardening flags, and freeze the `.deb` in our internal APT repo. |
| **Golang static binary inside TEE** (§4.2) | Kernel 6.6.x offers **fully working SGX/SEV-SNP** patches out-of-the-box, required for Intel TDX & AMD SEV-SNP remote attestation of our Go CLEAN enclave. |

---

### 🛠️ Step-by-Step Fork Recipe

1. **Download**
   ```bash
   wget https://cdimage.kali.org/kali-2024.3/kali-linux-2024.3-installer-netinst-amd64.iso
   ```

2. **Install (unattended)**
   ```bash
   # preseed.cfg -> only openssh-server, standard, no desktop
   virt-install --name kali-clean-base --memory 4096 --disk size=20 \
     --cdrom kali-linux-2024.3-installer-netinst-amd64.iso \
     --initrd-inject=preseed.cfg
   ```

3. **Lock kernel & harden**
   ```bash
   sudo apt-mark hold linux-image-amd64 linux-headers-amd64
   sudo apt install build-essential libncurses5-dev fakeroot xz-utils
   cp /boot/config-$(uname -r) ~/kernel/.config
   make menuconfig
   # enable: CONFIG_SECURITY_LOCKDOWN_LSM, CONFIG_INTEL_TXT, CONFIG_AMD_MEM_ENCRYPT
   make -j$(nproc) bindeb-pkg LOCALVERSION=-clean
   sudo dpkg -i ../linux-image-*-clean*.deb
   ```

4. **Shrink image**
   ```bash
   apt autoremove --purge -y
   apt clean
   rm -rf /usr/share/doc /usr/share/man
   ```

5. **Freeze & sign**
   - Push the resulting `.deb` kernel + rootfs tarball into your **internal APT repo**.  
   - Cosign the hash so every CLEAN node can verify the exact same image.

---

### ⚠️ What **NOT** to do
- ❌ Do **not** base on the 2023.x quarterly ISOs—those kernels (5.x) lack modern TEE patches.  
- ❌ Do **not** enable the full `kali-linux-everything` metapackage; it pulls 20 GB of offensive tools that break reproducibility.  
- ❌ Do **not** chase bleeding-edge 6.9-rc kernels; 6.6.x LTS gives us a **stable CVE feed** for the next 6 years.

---

### ✅ TL;DR

> **Base Image:** `kali-linux-2024.3-netinst-amd64`  
> **Kernel:** `6.6.x LTS` (recompiled once with hardening)  
> **Profile:** CLI-only, 2 GB compressed, reproducible, SGX/SEV-SNP ready.





## Kali Linux Specifications

### Base System
- **Distribution**: Kali Linux 2024.3 Rolling Release
- **Kernel**: Linux 6.6.x LTS (Long Term Support)
- **Architecture**: x86_64 (with ARM64 support planned)
- **Security**: Enhanced with TEE-specific security modules

### Key Features
- **Rolling Release Model**: Continuous updates with latest security tools
- **Penetration Testing Tools**: Pre-installed security analysis capabilities
- **Hardened Kernel**: Enhanced security features for TEE operations
- **Container Support**: Docker and Podman for agent isolation
- **Network Security**: Advanced firewall and network monitoring tools

## TEE Integration

### Supported TEE Technologies
1. **Intel SGX (Software Guard Extensions)**
   - SGX SDK integration
   - Enclave development tools
   - Remote attestation support

2. **AMD SEV-SNP (Secure Encrypted Virtualization)**
   - SEV-SNP kernel modules
   - Memory encryption support
   - Secure VM management

3. **Intel TDX (Trust Domain Extensions)**
   - TDX guest support
   - Trust domain management
   - Hardware-based isolation

### Security Enhancements
- **Secure Boot**: UEFI Secure Boot with custom keys
- **Measured Boot**: TPM-based boot measurement
- **Kernel Hardening**: KASLR, SMEP, SMAP, Control Flow Integrity
- **Memory Protection**: KPTI, KAISER, Spectre/Meltdown mitigations

## KNIRV-NEXUS Application Layer

### Go Application Architecture
```
┌─────────────────────────────────────┐
│         KNIRV-NEXUS Frontend        │
│         (React/TypeScript)          │
├─────────────────────────────────────┤
│         KNIRV-NEXUS Backend         │
│            (Go/Gin)                 │
├─────────────────────────────────────┤
│         Plugin Server               │
│         (Go/WASM Runtime)           │
├─────────────────────────────────────┤
│         TEE Runtime Layer           │
│      (SGX/SEV-SNP/TDX Support)      │
├─────────────────────────────────────┤
│         Kali Linux Foundation       │
│         (Kernel 6.6.x LTS)          │
└─────────────────────────────────────┘
```

### System Services
- **knirv-nexus.service**: Main application service
- **knirv-plugin-server.service**: Plugin server service
- **knirv-tee-monitor.service**: TEE monitoring service
- **knirv-attestation.service**: Attestation verification service

## Installation and Deployment

### Prerequisites
```bash
# Update Kali Linux to latest rolling release
sudo apt update && sudo apt full-upgrade -y

# Install required packages
sudo apt install -y \
    golang-1.21 \
    nodejs \
    npm \
    docker.io \
    podman \
    tpm2-tools \
    intel-sgx-sdk \
    amd-sev-tools
```

### TEE Setup
```bash
# Enable Intel SGX
sudo modprobe intel_sgx
echo 'intel_sgx' | sudo tee -a /etc/modules

# Configure AMD SEV-SNP
sudo modprobe ccp
sudo modprobe sev-guest

# Setup TPM for attestation
sudo systemctl enable tpm2-abrmd
sudo systemctl start tpm2-abrmd
```

### KNIRV-NEXUS Installation
```bash
# Clone and build KNIRV-NEXUS
git clone https://github.com/guiperry/KNIRV_NETWORK.git
cd KNIRV_NETWORK/KNIRVNEXUS

# Build the application
make build

# Install system services
sudo make install-services

# Start services
sudo systemctl enable knirv-nexus
sudo systemctl start knirv-nexus
```

## Security Configuration

### Firewall Rules
```bash
# Configure UFW for KNIRV-NEXUS
sudo ufw allow 8080/tcp  # Main application
sudo ufw allow 8082/tcp  # Plugin server
sudo ufw deny 22/tcp     # Disable SSH by default
sudo ufw enable
```

### SELinux/AppArmor Policies
- Custom AppArmor profiles for KNIRV-NEXUS processes
- Mandatory Access Control for TEE operations
- Container security policies for agent isolation

### Audit Configuration
```bash
# Enable comprehensive auditing
sudo auditctl -w /opt/knirv-nexus -p rwxa -k knirv-access
sudo auditctl -w /var/log/knirv-nexus -p rwxa -k knirv-logs
```

## Monitoring and Logging

### System Monitoring
- **Prometheus**: Metrics collection
- **Grafana**: Visualization dashboards
- **AlertManager**: Alert routing and management
- **Node Exporter**: System metrics

### Log Management
- **rsyslog**: Centralized logging
- **logrotate**: Log rotation and archival
- **auditd**: Security event logging
- **journald**: Systemd service logging

### TEE-Specific Monitoring
- Enclave health monitoring
- Attestation status tracking
- Memory encryption verification
- Performance metrics collection

## Development Environment

### Development Tools
```bash
# Install development dependencies
sudo apt install -y \
    build-essential \
    git \
    vim \
    tmux \
    htop \
    iotop \
    strace \
    gdb \
    valgrind
```

### IDE Integration
- **VS Code**: With Go and TypeScript extensions
- **Vim/Neovim**: With language server support
- **GoLand**: JetBrains IDE for Go development

## Backup and Recovery

### System Backup
- **Automated snapshots**: LVM snapshots for system state
- **Configuration backup**: Git-based configuration management
- **Data backup**: Encrypted backups to secure storage

### Disaster Recovery
- **System imaging**: Full system image creation
- **Recovery procedures**: Documented recovery steps
- **Testing**: Regular disaster recovery testing

## Performance Optimization

### Kernel Tuning
```bash
# Optimize for TEE workloads
echo 'vm.swappiness=1' >> /etc/sysctl.conf
echo 'kernel.sched_migration_cost_ns=5000000' >> /etc/sysctl.conf
echo 'kernel.sched_autogroup_enabled=0' >> /etc/sysctl.conf
```

### Resource Management
- **cgroups v2**: Resource isolation and limits
- **CPU affinity**: TEE-specific CPU allocation
- **Memory management**: Optimized for enclave operations

## Compliance and Certification

### Security Standards
- **Common Criteria**: EAL4+ certification target
- **FIPS 140-2**: Cryptographic module compliance
- **ISO 27001**: Information security management

### Audit Requirements
- **SOC 2 Type II**: Service organization controls
- **PCI DSS**: Payment card industry compliance
- **GDPR**: Data protection regulation compliance

## Future Roadmap

### Planned Enhancements
- **ARM64 Support**: Native ARM TEE support
- **Kubernetes Integration**: Container orchestration
- **Zero-Trust Networking**: Enhanced network security
- **Quantum-Resistant Cryptography**: Post-quantum algorithms

### Research Areas
- **Confidential Computing**: Advanced TEE research
- **Homomorphic Encryption**: Privacy-preserving computation
- **Secure Multi-Party Computation**: Distributed privacy
- **Federated Learning**: Secure ML training
