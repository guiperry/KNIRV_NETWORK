// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

// This demo shows how the eBPF integration would be used in KNIRV-NEXUS
// Note: This requires root privileges to run actual eBPF programs

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"backend_server/internal/ebpf"
)

func main() {
	fmt.Println("KNIRV-NEXUS eBPF Integration Demo")
	fmt.Println("==================================")

	// Create eBPF Manager
	fmt.Println("1. Creating eBPF Manager...")
	mgr := ebpf.NewManager()

	// Initialize with configuration
	fmt.Println("2. Initializing eBPF Manager...")
	config := &ebpf.Config{
		Programs: []ebpf.ProgramConfig{
			{Name: "syscall_trace", Enabled: true},
			{Name: "sandbox_lsm", Enabled: true},
		},
	}

	err := mgr.Initialize(context.Background(), config)
	if err != nil {
		log.Printf("Initialization failed (expected without root): %v", err)
		fmt.Println("   ⚠️  Initialization failed - this is expected without root privileges")
		fmt.Println("   To run actual eBPF programs, use:")
		fmt.Println("   sudo -E go run ./internal/ebpf/demo/main.go")
		return
	}
	defer mgr.Shutdown()

	fmt.Println("   ✅ eBPF Manager initialized successfully")

	// Create Policy Manager
	fmt.Println("3. Setting up Policy Manager...")
	policyMgr := ebpf.NewPolicyManager(mgr)

	// Set a sandbox policy
	containerID := uint64(1234)
	policy := &ebpf.SandboxPolicy{
		AllowedPathPrefix: "/tmp/nexus-sandbox/test",
		NetworkAllowed:    false,
	}

	err = policyMgr.SetSandboxPolicy(containerID, policy)
	if err != nil {
		log.Printf("Set policy failed: %v", err)
	} else {
		fmt.Printf("   ✅ Set sandbox policy for container %d\n", containerID)
	}

	// Create Event Collector
	fmt.Println("4. Setting up Event Collector...")
	collector := ebpf.NewEventCollector(mgr)

	// Subscribe to events
	collector.Subscribe(func(event *ebpf.SyscallEvent) error {
		fmt.Printf("   📊 Syscall Event: PID=%d, SyscallID=%d, Timestamp=%d\n",
			event.PID, event.SyscallID, event.Timestamp)
		return nil
	})

	// Start event collection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		err := collector.Start(ctx)
		if err != nil {
			log.Printf("Event collection failed: %v", err)
		}
	}()

	fmt.Println("   ✅ Event collector started - listening for syscall events...")

	// Wait for events
	time.Sleep(3 * time.Second)
	collector.Stop()

	// Get metrics
	fmt.Println("5. Getting eBPF Metrics...")
	metrics := mgr.GetMetrics()
	fmt.Printf("   📈 Programs Attached: %d\n", metrics.ProgramsAttached)
	fmt.Printf("   📈 Initialized: %v\n", metrics.Initialized)

	fmt.Println("\n✅ Demo completed successfully!")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("• eBPF Manager initialization")
	fmt.Println("• Sandbox policy management")
	fmt.Println("• Syscall event collection")
	fmt.Println("• Real-time monitoring")
	fmt.Println("\nNext Steps:")
	fmt.Println("1. Run with sudo for actual eBPF functionality")
	fmt.Println("2. Integrate with CDE service")
	fmt.Println("3. Add XDP network filtering")
	fmt.Println("4. Implement AI anomaly detection")
}
