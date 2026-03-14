# eBPF Quick Start Guide

## Prerequisites

```bash
# Install build dependencies (Ubuntu/Debian)
sudo apt-get update
sudo apt-get install -y \
    clang \
    llvm \
    libbpf-dev \
    linux-headers-$(uname -r) \
    pkg-config

# Install bpf2go
go install github.com/cilium/ebpf/cmd/bpf2go@latest
```

## Generate eBPF Code

```bash
# From KNIRVSERVER root
cd backend/internal/ebpf
go generate

# This creates:
# - syscalltrace_bpfel.go/.o (syscall monitoring)
# - sandboxlsm_bpfel.go/.o (LSM enforcement)
# - xdpfilter_bpfel.go/.o (XDP rate limiting)
# - virtualns_bpfel.go/.o (virtual containers)
# Plus big-endian variants (*_bpfeb.go/.o)
```

## Add to Makefile

Add this target to `KNIRVSERVER/Makefile`:

```makefile
.PHONY: ebpf-generate
ebpf-generate: ## Generate eBPF Go code from C programs
	@echo "Generating eBPF programs..."
	cd backend/internal/ebpf && go generate
	@echo "eBPF generation completed"

# Update existing binary target
.PHONY: binary
binary: ebpf-generate backend frontend ## Build unified binary with eBPF
	@echo "Creating unified binary with embedded eBPF..."
	go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/knirv-server .
	@echo "Binary created with embedded eBPF programs"
```

## Update Loaders

After generating, update `loader.go` and `xdp_loader.go` to use generated code:

### loader.go Example

```go
func (l *Loader) LoadSyscallTrace() error {
    // Load generated eBPF objects
    objs := &syscallTraceObjects{}
    if err := loadSyscallTraceObjects(objs, nil); err != nil {
        return fmt.Errorf("load syscall trace objects: %w", err)
    }

    // Attach to tracepoint
    tp, err := link.Tracepoint("raw_syscalls", "sys_enter",
        objs.TraceSysEnter, nil)
    if err != nil {
        objs.Close()
        return fmt.Errorf("attach tracepoint: %w", err)
    }

    l.links = append(l.links, tp)

    // Store collection for event reading
    // Note: You'll need to expose the collection from generated code
    // or access maps directly via objs.Events

    log.Println("Syscall trace program loaded and attached")
    return nil
}
```

### xdp_loader.go Example

```go
func (l *XDPLoader) LoadXDPFilter() error {
    // Load generated XDP objects
    objs := &xdpFilterObjects{}
    if err := loadXdpFilterObjects(objs, nil); err != nil {
        return fmt.Errorf("load XDP filter objects: %w", err)
    }

    l.collection = &ebpf.Collection{
        Programs: map[string]*ebpf.Program{
            "xdp_rate_limit": objs.XdpRateLimit,
        },
        Maps: map[string]*ebpf.Map{
            "rate_limits": objs.RateLimits,
            "whitelist":   objs.Whitelist,
        },
    }

    l.whitelistMap = objs.Whitelist

    log.Println("XDP Loader: XDP program loaded successfully")
    return nil
}
```

## Build and Test

```bash
# Generate eBPF code
make ebpf-generate

# Build binary with embedded eBPF
make binary

# Test in privileged environment
make test-privileged
```

## Verify eBPF Programs Load

```bash
# Run KNIRVSERVER
./dist/knirv-server

# In another terminal, verify programs are loaded
sudo bpftool prog list | grep -E "trace_sys_enter|xdp_rate_limit"
sudo bpftool map list
```

## Common Issues

### "permission denied" loading program
**Solution**: Ensure CAP_BPF or CAP_SYS_ADMIN capability:
```bash
# Check capabilities
capsh --print | grep cap_bpf

# Run with privileged Docker (development)
./scripts/run-docker-privileged.sh

# Or with sudo (testing)
sudo ./dist/knirv-server
```

### "BPF LSM not supported"
**Solution**: Enable BPF LSM in kernel:
```bash
# Check if enabled
cat /sys/kernel/security/lsm | grep bpf

# If not, add to grub config and reboot
# /etc/default/grub: GRUB_CMDLINE_LINUX="lsm=...,bpf"
sudo update-grub
sudo reboot
```

### "vmlinux BTF not found"
**Solution**: Install kernel with BTF support:
```bash
# Check BTF availability
ls /sys/kernel/btf/vmlinux

# Install headers (Ubuntu)
sudo apt-get install linux-headers-$(uname -r)
```

### "clang: command not found" during go generate
**Solution**: Install LLVM/Clang:
```bash
sudo apt-get install clang llvm
```

## Next Steps

See `README.md` for comprehensive documentation, including:
- Detailed architecture overview
- All four eBPF program descriptions
- Go package structure and usage
- Integration with KNIRVSERVER TEE security
- Performance characteristics
- Production deployment guide
