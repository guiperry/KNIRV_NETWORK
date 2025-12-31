# Kata Containers Local Deployment Issue

**Status**: UNRESOLVED
**Date**: 2025-12-31
**Priority**: Medium - Blocks local development with Kata runtime
**Affects**: Local development environment only (cloud deployment untested)

## Summary

Local deployment using Kata Containers with custom Kali Linux kernel and rootfs fails with QEMU startup error. The container_deployer infrastructure is complete and functional, but QEMU consistently fails to launch the Kata VM with the error: `exiting QMP loop, command cancelled`.

## Environment

### Hardware Constraints
- **CPU**: Intel with 36-bit physical address space
- **CPU Flags**: VMX enabled (virtualization supported)
- **Memory**: Sufficient RAM available
- **Virtualization**: KVM modules loaded

### Software Versions
- **Kata Runtime**: 3.21.0
- **Docker**: 29.1.3
- **QEMU**: /opt/kata/bin/qemu-system-x86_64
- **Containerd**: b98a3aace656320842a23f4a392a33f46af97866
- **OS**: Linux 5.15.0-164-generic

## Root Cause Analysis

### Primary Issue: Hardware Address Space Limitation

The host CPU only supports **36-bit physical address space** (max 64GB addressable). QEMU attempts to allocate a 92GB virtual address space (`0x16ffffffff`), which exceeds the CPU's 36-bit limit (`0xfffffffff`), causing QEMU to fail at startup.

**Error**: `qemu-system-x86_64: Address space limit 0xfffffffff < 0x16ffffffff phys-bits too low (36)`

### Secondary Issues Discovered and Resolved

1. **Compressed Rootfs Image**
   - **Problem**: os_builder created gzip-compressed rootfs image
   - **Solution**: Decompress to raw cpio archive (initrd format)
   - **Status**: ✅ FIXED - Script now auto-decompresses

2. **Incorrect Image Format Configuration**
   - **Problem**: Kata config used `image=` for compressed file
   - **Solution**: Use `initrd=` for uncompressed cpio archive
   - **Status**: ✅ FIXED - Config updated

3. **Rootfs Size Alignment**
   - **Problem**: Rootfs was 25KB too small for QEMU's alignment requirements
   - **Solution**: Resized with `truncate -s 76070912`
   - **Status**: ✅ FIXED - Size corrected

4. **Docker Runtime Configuration**
   - **Problem**: Multiple incorrect runtime configurations
   - **Issues Fixed**:
     - Runtime name: `kata-runtime` → `kata`
     - Configuration key: `runtime-type` → `runtimeType` (camelCase)
     - Runtime type: Direct path → `io.containerd.kata.v2`
   - **Status**: ✅ FIXED - Docker daemon.json corrected

5. **Docker Bridge Network Deletion**
   - **Problem**: docker0 bridge interface repeatedly deleted by system
   - **Solution**: Created custom bridge network `knirvnexus-local`
   - **Status**: ✅ FIXED - Playbook uses custom network

## Attempted Solutions

### Configuration Adjustments
- [x] Reduced VM memory from 2GB → 1GB → 512MB
- [x] Changed machine type: q35 → pc → microvm
- [x] Attempted phys-bits override (blocked by hardware)
- [x] Disabled memory preallocation
- [x] Switched from image to initrd format
- [x] Decompressed rootfs image
- [x] Corrected rootfs size alignment

### System-Level Attempts
- [x] Restarted Docker daemon multiple times
- [x] Verified KVM modules loaded
- [x] Checked ulimits (all unlimited)
- [x] Verified cgroup configuration
- [x] Created custom Docker bridge network

## Current Configuration

### Kata Configuration (`/opt/kata/share/defaults/kata-containers/configuration-qemu.toml`)

```toml
[hypervisor.qemu]
path = "/opt/kata/bin/qemu-system-x86_64"
kernel = "/home/gperry/.local/share/knirvnexus/os_builder/artifacts/output-kata-guest/vmlinuz-kali-clean-tee"
initrd = "/home/gperry/.local/share/knirvnexus/os_builder/artifacts/output-kata-guest/kata-rootfs-kali-clean-tee-raw.img"
machine_type = "microvm"
default_vcpus = 1
default_memory = 1024
```

### Docker Runtime Configuration (`/etc/docker/daemon.json`)

```json
{
  "runtimes": {
    "kata": {
      "runtimeType": "io.containerd.kata.v2"
    }
  }
}
```

## Error Logs

```
time="2025-12-31T04:08:15Z" level=error
msg="qemu-system-x86_64: Address space limit 0xfffffffff < 0x16ffffffff phys-bits too low (36)"
name=containerd-shim-v2
source=virtcontainers/hypervisor
subsystem=qemu

time="2025-12-31T04:08:15Z" level=error
msg="Failed to negotiate QMP Capabilities"
error="exiting QMP loop, command cancelled"
name=containerd-shim-v2
source=virtcontainers/hypervisor
subsystem=qemu

time="2025-12-31T04:08:15Z" level=error
msg="Cannot start VM"
error="exiting QMP loop, command cancelled"
name=containerd-shim-v2
source=virtcontainers
subsystem=sandbox
```

## Artifacts Status

### Built and Working
- ✅ Custom Kali kernel: `vmlinuz-kali-clean-tee` (13MB)
- ✅ Custom Kali rootfs: `kata-rootfs-kali-clean-tee-raw.img` (175MB, decompressed)
- ✅ Container image: `knirvnexus-go-app:latest` (built with FROM scratch)
- ✅ Ansible playbooks: local-deploy and cloud-deploy
- ✅ Configuration scripts: `local-dev-config.sh`

### Validated
- ✅ Kata can read configuration: `kata-runtime kata-env` succeeds
- ✅ Docker recognizes kata runtime: `docker info | grep kata` shows runtime
- ✅ Kernel and rootfs files exist and are accessible
- ✅ QEMU binary exists at expected path

## Infrastructure Completed

Despite the runtime issue, all deployment infrastructure is complete:

1. **container_deployer** binary builds successfully
2. **Ansible playbooks** for both local and cloud deployment
3. **Custom Kata configuration** with Kali kernel/rootfs paths
4. **Automated setup script** (`local-dev-config.sh`) with:
   - Docker/Kata runtime verification
   - Automatic rootfs decompression
   - Configuration generation
   - Docker daemon.json updates
5. **Comprehensive README** with usage instructions
6. **Docker custom network** to work around bridge issues

## Hypothesis: Hardware/Software Compatibility

The issue appears to be a fundamental incompatibility between:
- CPU with 36-bit address space limitation
- QEMU memory mapping requirements (92GB virtual address space)
- Kata/QEMU version combination
- Custom kernel/rootfs configuration

Even with minimal memory (1GB), QEMU's internal address space allocation exceeds hardware capabilities.

## Recommended Next Steps

### Immediate Workarounds

1. **Test on Different Hardware**
   - Try deployment on CPU with ≥40-bit address space
   - Use newer Intel/AMD processor with extended address capabilities
   - Test in cloud environment (AWS, Azure, GCP VMs)

2. **Alternative Local Development Approach** (Temporary)
   - Use standard Docker with Kali base image for local development
   - Reserve Kata+custom-kernel for cloud/production deployments
   - This preserves access to Kali tools while avoiding Kata compatibility issues

### Long-Term Investigation

1. **Version Compatibility Matrix**
   - Test different Kata runtime versions (try 3.0.x, 3.10.x)
   - Try different QEMU versions
   - Check Kata release notes for 36-bit CPU compatibility

2. **Kernel/Rootfs Optimization**
   - Reduce initrd size further (current: 175MB)
   - Investigate if minimal Kali tools subset reduces memory footprint
   - Try using ext4 image instead of cpio initrd

3. **QEMU Parameter Tuning**
   - Investigate QEMU memory mapping options
   - Try different block device drivers (virtio-blk vs virtio-scsi)
   - Experiment with memory backend options

4. **System-Level Debugging**
   - Enable QEMU debug logging: `QEMU_LOG=trace`
   - Capture strace of QEMU process startup
   - Check for cgroup memory limits affecting QEMU
   - Investigate if SELinux/AppArmor blocking QEMU

### Future Testing Checklist

When testing on new hardware or with new configurations:

```bash
# 1. Verify CPU capabilities
lscpu | grep "Address sizes"  # Should show >36 bits physical

# 2. Run automated setup
./scripts/local-dev-config.sh --auto-fix

# 3. Verify Kata can read config
kata-runtime kata-env | grep -E "Kernel|Initrd"

# 4. Test manual QEMU launch (bypass Kata)
/opt/kata/bin/qemu-system-x86_64 \
  -kernel /path/to/vmlinuz-kali-clean-tee \
  -initrd /path/to/kata-rootfs-kali-clean-tee-raw.img \
  -m 1024 \
  -machine microvm \
  -nographic

# 5. If manual QEMU works, issue is in Kata/Docker integration
# 6. If manual QEMU fails, issue is in kernel/rootfs or QEMU config
```

## Known Limitations

### Won't Fix (Hardware Constraints)
- Cannot override CPU phys-bits beyond hardware capability (36-bit)
- Cannot increase address space beyond 2^36 = 64GB
- QEMU memory mapping requires more than available address space

### Technical Debt
- Need to investigate why QEMU allocates 92GB virtual space for 1GB VM
- Kata might not properly support CPUs with <40-bit address space
- Docker bridge (docker0) instability - root cause unknown

## Related Documentation

- [Kata Configuration](./README.md#configuration)
- [Local Development Setup](./README.md#quick-start)
- [Troubleshooting Guide](./README.md#troubleshooting)
- [Setup Automation](./scripts/local-dev-config.sh)

## Contact & Support

For questions or updates on this issue:
- Check Kata Containers GitHub issues: https://github.com/kata-containers/kata-containers/issues
- KNIRV Network team discussions
- Update this document with any progress or findings

---

**Last Updated**: 2025-12-31
**Contributors**: Development team working on container_deployer local Kata integration
