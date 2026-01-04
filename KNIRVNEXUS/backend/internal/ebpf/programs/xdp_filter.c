// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

// Note: IntelliSense errors are expected in VSCode as kernel headers
// are not available in the development environment. The programs
// compile correctly using the Makefile with actual kernel headers.

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#define RATE_LIMIT_PPS 10000    // 10k packets per second
#define RATE_LIMIT_BPS (100 * 1024 * 1024)  // 100 Mbps
#define WINDOW_SIZE_NS (1000000000ULL)      // 1 second

struct rate_limit {
    __u64 packets;
    __u64 bytes;
    __u64 window_start;
    __u64 dropped;
};

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 100000);
    __type(key, __u32);  // Source IP
    __type(value, struct rate_limit);
} rate_limits SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u32);  // Whitelisted IP
    __type(value, __u8); // Dummy value
} whitelist SEC(".maps");

SEC("xdp")
int xdp_rate_limit(struct xdp_md *ctx)
{
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;

    // Parse Ethernet header
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_DROP;

    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return XDP_PASS;  // Not IPv4

    // Parse IP header
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return XDP_DROP;

    __u32 src_ip = ip->saddr;

    // Check whitelist (known libp2p peers)
    if (bpf_map_lookup_elem(&whitelist, &src_ip))
        return XDP_PASS;

    // Check rate limit
    struct rate_limit *limit = bpf_map_lookup_elem(&rate_limits, &src_ip);
    __u64 now = bpf_ktime_get_ns();
    __u32 pkt_size = data_end - data;

    if (!limit) {
        // First packet from this IP
        struct rate_limit new_limit = {
            .packets = 1,
            .bytes = pkt_size,
            .window_start = now,
            .dropped = 0,
        };
        bpf_map_update_elem(&rate_limits, &src_ip, &new_limit, BPF_ANY);
        return XDP_PASS;
    }

    // Check if window expired
    if (now - limit->window_start > WINDOW_SIZE_NS) {
        limit->packets = 1;
        limit->bytes = pkt_size;
        limit->window_start = now;
        return XDP_PASS;
    }

    // Check limits
    if (limit->packets >= RATE_LIMIT_PPS || limit->bytes >= RATE_LIMIT_BPS) {
        __sync_fetch_and_add(&limit->dropped, 1);
        return XDP_DROP;
    }

    __sync_fetch_and_add(&limit->packets, 1);
    __sync_fetch_and_add(&limit->bytes, pkt_size);

    return XDP_PASS;
}

char LICENSE[] SEC("license") = "GPL";