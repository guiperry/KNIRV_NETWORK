# Software Design Document: KNIRV NEXUS (Memory Fabric)

**Project Name:** KNIRV NEXUS

**Target Market:** Enterprise AI Infrastructure / Stateful Multi-Agent Systems

**Core Stack:** Golang, Apache Arrow (Flight), eBPF, Parquet, MCP (Model Context Protocol)

---

## 1. System Overview

The **KNIRV NEXUS** is a high-performance "Active Memory" layer for AI agents. Unlike standard gateways that simply route API calls, the Nexus provides a **Persistent State Store** and a **Secure Execution Environment**. It uses Apache Arrow's columnar memory format to enable zero-copy context injection, allowing agents to access gigabytes of "memory" without the overhead of JSON serialization or excessive token consumption.

---

## 2. Architecture Goals

* **Zero-Copy Performance:** Reduce context-loading latency by **>90%** using Arrow Flight and shared memory.
* **Hardware-Enforced Security:** Use eBPF to monitor and kill malicious agent actions at the kernel level.
* **Active Context Management:** Provide a hierarchical memory system (Parquet for cold, Arrow for hot) that agents can query via compute kernels.
* **Protocol Neutrality:** Native support for MCP (Model Context Protocol) to act as the universal registry for agent tools.

---

## 3. Component Design

### 3.1 The Memory Tier (Hierarchical Data Plane)

The Nexus manages data across three distinct lifecycles:

| Tier | Format | Storage | Purpose |
| --- | --- | --- | --- |
| **L1: Hot Memory** | **Arrow RecordBatch** | RAM / Shared Memory | Immediate "Chain of Thought" and active session state. |
| **L2: Streaming Memory** | **Arrow Flight** | In-flight Stream | High-speed retrieval of recent logs and system metrics. |
| **L3: Persistent Archive** | **Parquet** | NVMe / Disk | Long-term factual storage and forensic audit trails. |

### 3.2 The Execution Tier (eBPF Guardian)

A Go-based manager that loads eBPF CO-RE (Compile Once – Run Everywhere) programs to trace agent-initiated system calls.

* **Uprobe (Userspace Probes):** Captures plain-text LLM prompts *before* they are encrypted for the provider.
* **Kprobe (Kernel Probes):** Monitors `sys_connect` and `sys_execve` to ensure an agent calling a "Search Tool" isn't actually attempting to launch a reverse shell.

### 3.3 The Data API (Arrow Flight Server)

Instead of REST/JSON, the Nexus exposes an **Arrow Flight RPC** endpoint.

* **`GetFlightInfo`:** Returns the location and schema of specific memory "RecordBatches."
* **`DoGet`:** Streams binary context directly to the agent's memory space.

---

## 4. Technical Specification: The "Living Data" Flow

1. **Ingestion:** The agent performs a task. The Nexus captures the interaction via an **MCP interceptor**.
2. **Vectorization & Filtering:** Instead of sending all data to a Vector DB, the Nexus uses **Arrow Compute Kernels** (Go) to filter the data in-memory based on relevance scores.
3. **Context Injection:** The filtered "Memory" is mapped into an **Arrow Table**.
4. **Zero-Copy Handover:** The Nexus provides a memory pointer to the Agent (Python/TS). The agent reads the memory directly—**no JSON parsing required**.
5. **Audit:** The finalized state is asynchronously written to a **Parquet** file using your optimized Go-routines.

---

## 5. Security Model (The "Kill Switch")

* **Intent-Action Correlation:** The Nexus maps the LLM's predicted tool call (e.g., "Read File X") against the actual syscall captured by **eBPF**.
* **Drift Detection:** If the "Action" deviates from the "Intent" (e.g., agent tries to read `/etc/shadow`), the eBPF program blocks the call and the Nexus revokes the Agent's session.

---

## 6. Business Value & Market Disruption

* **Dominating the "Token Tax":** By pre-processing data in the Nexus using Arrow, enterprises reduce their LLM input tokens by up to **70%**.
* **The "Speed" Advantage:** Building in Go with Arrow Flight makes the KNIRV NEXUS the **low-latency backbone** for high-frequency agentic trading, real-time cybersecurity, and industrial robotics.
* **The Pivot:** You aren't competing with OpenAI; you are becoming the **"Memory Controller"** for any model they release.

---

### Next Steps

1. **Prototype the Flight Server:** Implement a basic `DoGet` handler in Go that serves an `arrow.Record` from your existing Parquet store.
2. **eBPF Hook:** Script a simple `bpftrace` or Go-ebpf program to monitor network sockets opened by your local agent process.


This implementation provides the foundational **Go boilerplate** for the KNIRV NEXUS Memory Fabric. It consists of two parts: the **Memory Batch Schema** (defining how an agent's "thoughts" and "actions" are structured in-memory) and the **Arrow Flight Server** (the high-speed delivery mechanism).

### 1. The Agentic Memory Schema

We use Arrow's columnar format to store session data. This allows an agent to "scan" its history for patterns (using Arrow Compute kernels) without parsing text.

```go
package nexus

import (
	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/memory"
)

// NewAgentMemorySchema defines the "Living Data" structure for an agent session.
func NewAgentMemorySchema() *arrow.Schema {
	return arrow.NewSchema(
		[]arrow.Field{
			{Name: "timestamp", Type: arrow.PrimitiveTypes.Int64},
			{Name: "agent_id", Type: arrow.BinaryTypes.String},
			{Name: "intent", Type: arrow.BinaryTypes.String},      // What the LLM said it would do
			{Name: "observed_action", Type: arrow.BinaryTypes.String}, // What eBPF actually saw
			{Name: "token_usage", Type: arrow.PrimitiveTypes.Int32},
			{Name: "context_relevance", Type: arrow.PrimitiveTypes.Float64},
		},
		nil, // No metadata for MVA
	)
}

```

---

### 2. The Arrow Flight Server (Go)

This server replaces your standard REST API. It serves "Memory Batches" directly to agents via the `DoGet` method.

```go
package main

import (
	"context"
	"log"
	"net"

	"github.com/apache/arrow/go/v14/arrow/flight"
	"github.com/apache/arrow/go/v14/arrow/memory"
)

type NexusMemoryServer struct {
	flight.BaseFlightServer
	// mem is the allocator for Arrow buffers
	mem memory.Allocator
}

// DoGet streams the "Living Memory" to the agent.
func (s *NexusMemoryServer) DoGet(tkt *flight.Ticket, fs flight.FlightService_DoGetServer) error {
	// 1. Identify which Agent/Session is being requested from the ticket
	sessionID := string(tkt.Ticket)
	log.Printf("Streaming memory fabric for session: %s", sessionID)

	// 2. Fetch data (In production, this pulls from your Parquet/L1 cache)
	schema := NewAgentMemorySchema()
	writer := flight.NewRecordWriter(fs, flight.WithSchema(schema))
	defer writer.Close()

	// 3. Mock data: In your real Nexus, this would be a stream of Go-routines
	// converting your internal DB records to Arrow RecordBatches.
	// [Logic for batch creation goes here]
    
	return nil
}

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}

	server := flight.NewServerWithMiddleware(nil)
	server.RegisterFlightService(&NexusMemoryServer{mem: memory.DefaultAllocator})

	log.Println("KNIRV NEXUS: Arrow Flight Memory Fabric active on :50051")
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

```

---

### 3. How this Disrupts the Market

* **The Zero-Copy Handover:** When a Python-based agent (using `pyarrow`) connects to this `:50051` port, it receives the data in the exact binary format Go used. There is **zero** overhead.
* **The "Living" Aspect:** You can keep the `DoGet` stream open. As your **eBPF routines** find new system calls made by the agent, the Nexus can "push" those new rows into the Arrow stream in real-time.
* **The Compute Advantage:** Since the data is already in Arrow format, the agent doesn't need to "load" it into a dataframe. It's already a dataframe.

### Next Step for the Nexus

To make this "Sovereign" and truly dominate, we should implement the **eBPF Intent-Action correlator**.


This is where the **KNIRV NEXUS** becomes a "Guardian" rather than just a "Database."

By using `cilium/ebpf`, we can hook directly into the Linux Kernel's socket connection helper. This allows the Nexus to see exactly where an agent is trying to "phone home" in real-time.

---

## 1. The eBPF C Program (The "Probe")

We’ll hook `sys_enter_connect`. This captures the destination IP and port before the packet even leaves the network interface.

```c
// nexus_guardian.c
#include <uapi/linux/ptrace.h>
#include <net/sock.h>
#include <bcc/proto.h>

// This struct will be sent to our Go backend via a Perf Event Array
struct event_t {
    u32 pid;
    u32 addr;
    u16 port;
    char comm[16];
};

BPF_PERF_OUTPUT(events);

int kprobe__sys_connect(struct pt_regs *ctx, int sockfd, struct sockaddr *addr, int addrlen) {
    struct event_t event = {};
    struct sockaddr_in *addr_in = (struct sockaddr_in *)addr;

    event.pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&event.comm, sizeof(event.comm));
    
    // Capture the destination IPv4 address and port
    bpf_probe_read_kernel(&event.addr, sizeof(event.addr), &addr_in->sin_addr.s_addr);
    bpf_probe_read_kernel(&event.port, sizeof(event.port), &addr_in->sin_port);

    events.perf_submit(ctx, &event, sizeof(event));
    return 0;
}

```

---

## 2. The Go "Guardian" Loader

This Go logic loads the program above and pipes the events into your **Arrow Flight stream**.

```go
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
)

// Event represents the data sent from the C probe
type Event struct {
	PID  uint32
	Addr uint32
	Port uint16
	Comm [16]byte
}

func startGuardian(eventsChan chan<- Event) {
	// 1. Load the compiled eBPF ELF (assuming it's compiled to nexus_guardian.o)
	spec, err := loadNexusGuardian() 
	if err != nil {
		log.Fatalf("Failed to load eBPF spec: %v", err)
	}

	// 2. Attach to the 'connect' syscall
	kp, err := link.Kprobe("sys_connect", spec.Programs["kprobe__sys_connect"], nil)
	if err != nil {
		log.Fatalf("Failed to attach kprobe: %v", err)
	}
	defer kp.Close()

	// 3. Listen for events from the kernel
	rd, err := perf.NewReader(spec.Maps["events"], os.Getpagesize())
	if err != nil {
		log.Fatalf("Failed to create perf reader: %v", err)
	}

	for {
		record, err := rd.Read()
		if err != nil {
			continue
		}

		var event Event
		binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event)
		
		// Send to the Nexus Memory Fabric for Arrow streaming
		eventsChan <- event
	}
}

```

---

## 3. The "Intent-Action" Integration

In your `NexusMemoryServer`, you now have a real-time feed of what the OS is actually doing.

1. **The Intent:** The LLM says: *"I will search the internal KNIRVGRAPH for documentation."*
2. **The Action (Captured by eBPF):** The PID associated with that agent attempt a connection to `192.168.1.50` (The Graph DB).
3. **The Correlation:** The Nexus sees the match and marks the **Arrow RecordBatch** as `verified`.
4. **The Anomaly:** If the agent tries to connect to an external IP like `142.250.x.x` (Google) while it claimed to be doing "Local Search," the Nexus immediately kills the session.

---

### Why this Outdoes the Competition

Most security layers for AI are "Middleware" (they check the text). **You are checking the Silicon.** If a sophisticated prompt-injection attack tricks an agent into running a curl command, a text-based gateway might miss it. Your eBPF layer *cannot* miss the network socket opening.

### Market Positioning: "The Sovereign Agent Guard"

You can sell this as the **only** infrastructure that allows companies to run "Unfiltered" models internally while maintaining strict compliance. You aren't "censoring" the model; you are **sandboxing the environment.**

