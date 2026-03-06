// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

// Note: IntelliSense errors are expected in VSCode as kernel headers
// are not available in the development environment. The programs
// compile correctly using the Makefile with actual kernel headers.

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <linux/errno.h>

struct virtual_container {
    __u64 container_id;
    __u32 root_pid;
    char rootfs[256];
    __u32 network_allowed;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u32);  // PID
    __type(value, __u64);  // Container ID
} pid_to_container SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1000);
    __type(key, __u64);  // Container ID
    __type(value, struct virtual_container);
} containers SEC(".maps");

// Helper to get container ID from PID
static __always_inline __u64 get_container_id_from_pid(__u32 pid)
{
    __u64 *container_id = bpf_map_lookup_elem(&pid_to_container, &pid);
    return container_id ? *container_id : 0;
}

SEC("lsm/file_open")
int BPF_PROG(virtual_ns_file_open, struct file *file, int ret)
{
    if (ret != 0)
        return ret;

    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    __u64 container_id = get_container_id_from_pid(pid);

    if (container_id == 0)
        return 0;  // Not in virtual container

    struct virtual_container *container;
    container = bpf_map_lookup_elem(&containers, &container_id);
    if (!container)
        return 0;

    // NOTE: Full file path checking requires BPF CO-RE with vmlinux.h
    // For now, we allow all file operations within tracked containers
    // TODO: Implement proper rootfs path validation using BPF_CORE_READ macros
    // when BPF CO-RE is set up with vmlinux.h

    // Placeholder: rootfs boundary checking would go here
    // In production, this would validate file->f_path is within container->rootfs
    // using bpf_d_path() and BPF_CORE_READ() helpers

    return 0;  // ALLOW for now
}

SEC("lsm/socket_connect")
int BPF_PROG(virtual_ns_network, struct socket *sock,
              struct sockaddr *address, int addrlen, int ret)
{
    if (ret != 0)
        return ret;

    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    __u64 container_id = get_container_id_from_pid(pid);

    if (container_id == 0)
        return 0;  // Not in virtual container

    struct virtual_container *container;
    container = bpf_map_lookup_elem(&containers, &container_id);
    if (!container)
        return 0;

    if (!container->network_allowed)
        return -EPERM;  // DENY network access

    return 0;
}

char LICENSE[] SEC("license") = "GPL";