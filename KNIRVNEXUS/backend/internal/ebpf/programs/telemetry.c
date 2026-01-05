#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

struct process_stats {
    __u64 syscall_count[400];
    __u64 cpu_time_ns;
    __u64 memory_bytes;
    __u64 net_tx_bytes;
    __u64 net_rx_bytes;
    __u32 context_switches;
    __u32 page_faults;
    char model_name[64];
};

// Define the tracepoint context structure for sched/sched_switch
struct trace_event_raw_sched_switch {
    __u64 __unused__;
    char prev_comm[16];
    __u32 prev_pid;
    __u32 prev_prio;
    long prev_state;
    char next_comm[16];
    __u32 next_pid;
    __u32 next_prio;
};

// Define the tracepoint context structure for kmem/mm_page_alloc
struct trace_event_raw_mm_page_alloc {
    __u64 __unused__;
    unsigned long pfn;
    unsigned int order;
    unsigned int gfp_flags;
    int migratetype;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 100000);
    __type(key, __u32);
    __type(value, struct process_stats);
} process_telemetry SEC(".maps");

SEC("tracepoint/sched/sched_switch")
int trace_context_switch(struct trace_event_raw_sched_switch *ctx)
{
    __u32 pid = ctx->prev_pid;
    struct process_stats *stats;

    stats = bpf_map_lookup_elem(&process_telemetry, &pid);
    if (stats) {
        __sync_fetch_and_add(&stats->context_switches, 1);
    }

    return 0;
}

SEC("tracepoint/kmem/mm_page_alloc")
int trace_page_alloc(struct trace_event_raw_mm_page_alloc *ctx)
{
    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    struct process_stats *stats;

    stats = bpf_map_lookup_elem(&process_telemetry, &pid);
    if (stats) {
        __u32 pages = 1 << ctx->order;
        __sync_fetch_and_add(&stats->memory_bytes, pages * 4096);
    }
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
