// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

// This file contains go:generate directives for compiling eBPF C programs
// into Go code with embedded bytecode using cilium/ebpf's bpf2go tool.
//
// Usage:
//   cd backend/internal/ebpf
//   go generate
//
// This will create the following generated files (do not commit to git):
//   - syscalltrace_bpfel.go/.o    - Syscall tracing (little-endian)
//   - syscalltrace_bpfeb.go/.o    - Syscall tracing (big-endian)
//   - sandboxlsm_bpfel.go/.o      - LSM sandbox enforcement (little-endian)
//   - sandboxlsm_bpfeb.go/.o      - LSM sandbox enforcement (big-endian)
//   - xdpfilter_bpfel.go/.o       - XDP rate limiting (little-endian)
//   - xdpfilter_bpfeb.go/.o       - XDP rate limiting (big-endian)
//   - virtualns_bpfel.go/.o       - Virtual namespace isolation (little-endian)
//   - virtualns_bpfeb.go/.o       - Virtual namespace isolation (big-endian)
//
// Requirements:
//   - clang/llvm (for compiling eBPF)
//   - bpf2go tool: go install github.com/cilium/ebpf/cmd/bpf2go@latest
//   - Linux kernel headers: apt-get install linux-headers-$(uname -r)
//   - libbpf headers: apt-get install libbpf-dev

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target amd64,arm64 -type syscall_event syscallTrace ./programs/syscall_trace.c

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target amd64,arm64 -type sandbox_policy sandboxLsm ./programs/sandbox_lsm.c

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target amd64,arm64 -type rate_limit xdpFilter ./programs/xdp_filter.c

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target amd64,arm64 -type virtual_container virtualNs ./programs/virtual_ns.c
