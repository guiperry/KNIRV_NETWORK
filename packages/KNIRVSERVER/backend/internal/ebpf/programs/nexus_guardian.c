// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

struct guardian_event {
    __u32 pid;
    __u32 addr;
    __u16 port;
    char comm[16];
    __u64 timestamp;
};

// Define the tracepoint context structure for raw_syscalls/sys_enter
// which is used when we don't have the specific syscall tracepoint struct
struct trace_event_raw_sys_enter {
    __u64 __unused__;
    long id;
    unsigned long args[6];
};

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} events SEC(".maps");

// Dummy map for type info
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct guardian_event);
} guardian_event_type SEC(".maps");

// Minimal sockaddr definitions
struct sockaddr {
    unsigned short sa_family;
    char sa_data[14];
};

struct in_addr {
    unsigned int s_addr;
};

struct sockaddr_in {
    unsigned short sin_family;
    unsigned short sin_port;
    struct in_addr sin_addr;
    unsigned char sin_zero[8];
};

// Syscall number for connect on x86_64 is 42
#define SYS_CONNECT 42

SEC("tracepoint/raw_syscalls/sys_enter")
int trace_connect(struct trace_event_raw_sys_enter *ctx)
{
    if (ctx->id != SYS_CONNECT)
        return 0;

    struct guardian_event *e;
    struct sockaddr *addr;
    struct sockaddr_in *addr_in;

    // args[1] is the sockaddr* addr
    addr = (struct sockaddr *)ctx->args[1];
    if (!addr)
        return 0;

    e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
    if (!e)
        return 0;

    e->timestamp = bpf_ktime_get_ns();
    e->pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&e->comm, sizeof(e->comm));

    // Read sin_family to check if it's AF_INET
    short family;
    if (bpf_probe_read_kernel(&family, sizeof(family), &addr->sa_family) < 0) {
        bpf_ringbuf_discard(e, 0);
        return 0;
    }

    if (family == 2) { // AF_INET
        addr_in = (struct sockaddr_in *)addr;
        bpf_probe_read_kernel(&e->addr, sizeof(e->addr), &addr_in->sin_addr.s_addr);
        bpf_probe_read_kernel(&e->port, sizeof(e->port), &addr_in->sin_port);
    } else {
        e->addr = 0;
        e->port = 0;
    }

    bpf_ringbuf_submit(e, 0);
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
