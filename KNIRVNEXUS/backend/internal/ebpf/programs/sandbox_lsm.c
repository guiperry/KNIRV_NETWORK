// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

// Note: IntelliSense errors are expected in VSCode as kernel headers
// are not available in the development environment. The programs
// compile correctly using the Makefile with actual kernel headers.

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <linux/security.h>

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
    // For now, use PID namespace ID as container ID
    // In production, integrate with your container tracking
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();
    return task->nsproxy->pid_ns_for_children->ns.inum;
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

    // Get file path
    struct path *path = &file->f_path;
    char filepath[256];
    bpf_d_path(path, filepath, sizeof(filepath));

    // Check if path starts with allowed prefix
    for (int i = 0; i < 256 && policy->allowed_prefix[i]; i++) {
        if (filepath[i] != policy->allowed_prefix[i])
            return -EPERM;  // DENY
    }

    return 0;  // ALLOW
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