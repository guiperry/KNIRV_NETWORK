# VSCode IntelliSense Configuration for eBPF Programs

## Issue
The eBPF C files (`sandbox_lsm.c`, `syscall_trace.c`) show IntelliSense errors in VSCode because:
1. Missing `linux/security.h` header
2. Incomplete type `struct trace_event_raw_sys_enter`

## Root Cause
These are expected IntelliSense errors in the development environment because:
- Kernel headers are not in the default include path
- The `vmlinux.h` file is generated at compile time from BTF
- eBPF programs use kernel-specific types and macros

## Solution Implemented

### 1. C/C++ Configuration
Created `.vscode/c_cpp_properties.json` with proper include paths:
- Linux kernel headers (`/usr/src/linux-headers-5.15.0-164*/`)
- System headers (`/usr/include`, `/usr/include/x86_64-linux-gnu`)
- Clang headers
- Workspace directory for generated `vmlinux.h`

### 2. Important Notes
- **The programs compile correctly** using the Makefile with actual kernel headers
- IntelliSense errors are cosmetic and don't affect compilation
- Run `make` in this directory to generate `vmlinux.h` before expecting full IntelliSense

### 3. If Errors Persist
1. Ensure kernel headers are installed:
   ```bash
   sudo apt install linux-headers-$(uname -r)
   ```

2. Generate vmlinux.h manually:
   ```bash
   cd KNIRVSERVER/backend/internal/ebpf/programs
   make vmlinux.h
   ```

3. Reload VSCode window (Ctrl+Shift+P → "Developer: Reload Window")

### 4. Alternative Approach
For better IntelliSense, you can:
- Use `bear` to generate `compile_commands.json`:
  ```bash
  bear -- make
  ```
- Configure VSCode to use `compileCommands` in `c_cpp_properties.json`

## Verification
To verify the configuration works:
1. Open `sandbox_lsm.c` or `syscall_trace.c`
2. Check if red squiggles are reduced
3. Hover over types like `struct file` or `bpf_get_current_task()` should show definitions

## References
- [VSCode C/C++ Extension Documentation](https://code.visualstudio.com/docs/cpp/c-cpp-properties-schema-reference)
- [eBPF Development Guide](https://docs.kernel.org/bpf/)
- [BPF and XDP Reference Guide](https://docs.cilium.io/en/stable/bpf/)