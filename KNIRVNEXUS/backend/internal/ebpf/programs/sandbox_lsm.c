// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

// Note: IntelliSense errors are expected in VSCode as kernel headers
// are not available in the development environment. The programs
// compile correctly using the Makefile with actual kernel headers.

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <linux/errno.h>

struct sandbox_policy {
    char allowed_prefix[256];
    __u32 network_allowed;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u64);  // Container ID
    __type(value, struct sandbox_policy);
} policies SEC(".maps");

// Helper to get container ID from task
static __always_inline __u64 get_container_id(void)
{
    // Use PID as container identifier for now
    // In production, integrate with your container tracking system
    // to map PIDs/cgroups to container IDs
    return bpf_get_current_pid_tgid() >> 32;
}

SEC("lsm/file_open")
int BPF_PROG(restrict_file_open, struct file *file, int ret)
{
    if (ret != 0)
        return ret;

    __u64 container_id = get_container_id();
    struct sandbox_policy *policy;

    policy = bpf_map_lookup_elem(&policies, &container_id);
    if (!policy)
        return 0;  // No policy = allow

    // NOTE: Full file path checking requires BPF CO-RE with vmlinux.h
    // For now, we allow all file operations if a policy exists
    // TODO: Implement proper path checking using BPF_CORE_READ macros
    // when BPF CO-RE is set up with vmlinux.h

    // Placeholder: basic policy enforcement would go here
    // In production, this would check file->f_path against allowed_prefix
    // using bpf_d_path() and BPF_CORE_READ() helpers

    return 0;  // ALLOW for now
}

SEC("lsm/socket_connect")
int BPF_PROG(restrict_network, struct socket *sock,
              struct sockaddr *address, int addrlen, int ret)
{
    if (ret != 0)
        return ret;

    __u64 container_id = get_container_id();
    struct sandbox_policy *policy;

    policy = bpf_map_lookup_elem(&policies, &container_id);
    if (!policy)
        return 0;

    if (!policy->network_allowed)
        return -EPERM;  // DENY network access

    return 0;
}

char LICENSE[] SEC("license") = "GPL";