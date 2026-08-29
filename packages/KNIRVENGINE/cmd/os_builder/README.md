# KNIRVENGINE Alpine OS builder

This is the Alpine-focused clone of the KNIRV server OS builder. In the
container deployment model, the host—not the image—provides the kernel, so the
builder creates the application distribution and records the required runtime
privileges instead of attempting to compile a kernel in Docker.

```bash
# Build the local image.
go run ./cmd/os_builder -action 0

# Build and save an air-gap-transferable Docker archive.
go run ./cmd/os_builder -action 1

# Print the eBPF/cgroup runtime requirements without invoking Docker.
go run ./cmd/os_builder -action validate
```

The image contains `bpftrace`, `bubblewrap`, and `nsenter`. Its `nsenter`
helper has `CAP_SYS_ADMIN`, `CAP_SYS_PTRACE`, and `CAP_SYS_CHROOT`; `bpftrace`
has `CAP_BPF`, `CAP_PERFMON`, and `CAP_SYS_ADMIN`. The Docker runtime must
allow those capabilities and provide a compatible host kernel. Use the
explicitly privileged compose profile in `../container_deployer` when cgroup
namespace control or host eBPF tracing is required.
