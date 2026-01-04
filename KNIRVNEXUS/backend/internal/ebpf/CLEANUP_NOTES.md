# eBPF Cleanup Notes

## Duplicate Implementation

**Location**: `backend/internal/ebpf/ebpf/`

This directory contains a supplementary eBPF implementation that was created in error. It duplicates functionality that already exists in the main `backend/internal/ebpf/` directory.

### Contents (Duplicates - Should be Removed)

```
backend/internal/ebpf/ebpf/
├── bpf/
│   ├── headers/
│   │   ├── vmlinux.h
│   │   └── common.h
│   └── syscall_monitor.c  # Duplicate of syscall_trace.c functionality
├── generate.go            # Duplicate of ../generate.go
├── README.md              # Duplicate documentation
├── example_usage.go       # Example code (may be useful reference)
└── syscall_monitor.go     # Duplicate wrapper (functionality in ../events.go)
```

### Recommended Action

**Remove the entire `backend/internal/ebpf/ebpf/` directory**:

```bash
cd backend/internal/ebpf
rm -rf ebpf/
```

### Why Remove?

1. **syscall_monitor.c** duplicates the functionality of `programs/syscall_trace.c`
   - Both monitor syscalls via tracepoint/raw_syscalls/sys_enter
   - Both use ring buffers for event streaming
   - The existing `syscall_trace.c` is simpler and already integrated

2. **syscall_monitor.go** duplicates functionality already in:
   - `events.go` - Ring buffer event collection
   - `manager.go` - eBPF lifecycle management

3. **generate.go** conflicts with `../generate.go`
   - Would generate files with different names for the same purpose
   - Could cause confusion during build

4. **README.md** and documentation are now in:
   - `../README.md` - Comprehensive implementation documentation
   - `../QUICKSTART.md` - Quick start guide
   - `../../docs/eBPF_Integration_Guide.md` - Integration guide

### What to Keep (If Useful)

**example_usage.go** - Contains example code that demonstrates usage patterns. Review and extract useful examples before deletion:

```go
// Example patterns that might be useful:
// - PID filtering approach
// - Event processing patterns
// - Error handling examples
```

Consider extracting any useful code snippets and adding them to:
- `../README.md` (usage examples section)
- Integration tests in `../*_test.go`

## Clean State

After cleanup, the directory structure should be:

```
backend/internal/ebpf/
├── programs/              # eBPF C programs (THE SOURCE OF TRUTH)
│   ├── syscall_trace.c
│   ├── sandbox_lsm.c
│   ├── xdp_filter.c
│   └── virtual_ns.c
├── demo/                  # Demo applications
├── generate.go            # Build directives
├── manager.go             # Main controller
├── loader.go              # Program loader
├── xdp_loader.go          # XDP loader
├── xdp_manager.go         # XDP operations
├── virtual_container_manager.go  # Virtual containers
├── policy.go              # Policy management
├── events.go              # Event collection
├── types.go               # Type definitions
├── README.md              # Comprehensive docs
├── QUICKSTART.md          # Quick start guide
├── CLEANUP_NOTES.md       # This file
├── .gitignore             # Ignore generated files
└── *_test.go              # Tests
```

## Verification After Cleanup

```bash
# Ensure no duplicate implementations
find backend/internal/ebpf -name "syscall_monitor.*" -o -name "ebpf/ebpf"

# Should return nothing
```

## Summary

The `backend/internal/ebpf/ebpf/` directory was created as a supplementary implementation before discovering the existing eBPF infrastructure. Now that proper documentation (`README.md`, `QUICKSTART.md`, updated `eBPF_Integration_Guide.md`) has been created for the existing implementation, the duplicate can be safely removed.
