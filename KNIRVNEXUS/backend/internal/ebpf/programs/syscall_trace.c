// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

// Note: IntelliSense errors are expected in VSCode as kernel headers
// are not available in the development environment. The programs
// compile correctly using the Makefile with actual kernel headers.

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

struct syscall_event {
    __u64 timestamp;
    __u32 pid;
    __u32 syscall_id;
};

// Define the tracepoint context structure for raw_syscalls/sys_enter
// This structure matches the kernel's tracepoint format
struct trace_event_raw_sys_enter {
    __u64 __unused__;      // Common tracepoint fields (not used)
    long id;               // Syscall ID
    unsigned long args[6]; // Syscall arguments
};

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} events SEC(".maps");

// Dummy map to expose syscall_event type to bpf2go for Go code generation
// This map is never used at runtime, but ensures the type is in BTF data
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct syscall_event);
} syscall_event_type SEC(".maps");

SEC("tracepoint/raw_syscalls/sys_enter")
int trace_sys_enter(struct trace_event_raw_sys_enter *ctx)
{
    struct syscall_event *e;

    e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
    if (!e)
        return 0;

    e->timestamp = bpf_ktime_get_ns();
    e->pid = bpf_get_current_pid_tgid() >> 32;
    e->syscall_id = ctx->id;

    bpf_ringbuf_submit(e, 0);
    return 0;
}

char LICENSE[] SEC("license") = "GPL";