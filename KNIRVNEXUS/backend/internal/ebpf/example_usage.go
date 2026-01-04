package ebpf

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"
)

// ExampleUsage demonstrates how to use the eBPF syscall monitor
// This is not a real function - it's for documentation purposes
func ExampleUsage() {
	// 1. Create the syscall monitor
	monitor, err := NewSyscallMonitor()
	if err != nil {
		log.Fatalf("Failed to create syscall monitor: %v", err)
	}
	defer monitor.Close()

	// 2. Start a skill validation process
	cmd := exec.Command("/bin/bash", "-c", "ls -la /tmp")
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start process: %v", err)
	}

	// 3. Add the PID to the filter (only monitor this process)
	pid := uint32(cmd.Process.Pid)
	if err := monitor.AddPID(pid); err != nil {
		log.Fatalf("Failed to add PID to filter: %v", err)
	}

	// 4. Attach the eBPF programs
	if err := monitor.Attach(); err != nil {
		log.Fatalf("Failed to attach eBPF programs: %v", err)
	}

	// 5. Process events
	go func() {
		for event := range monitor.Events() {
			switch event.Type {
			case 1: // Syscall enter
				syscallName := GetSyscallName(event.SyscallNr)
				fmt.Printf("[%d] %s: %s() called with args: %v\n",
					event.PID,
					string(event.Comm[:]),
					syscallName,
					event.Args)
			case 2: // Syscall exit
				syscallName := GetSyscallName(event.SyscallNr)
				fmt.Printf("[%d] %s: %s() returned %d\n",
					event.PID,
					string(event.Comm[:]),
					syscallName,
					event.Ret)
			}
		}
	}()

	// 6. Wait for process to complete
	if err := cmd.Wait(); err != nil {
		log.Printf("Process exited with error: %v", err)
	}

	// 7. Check statistics
	stats, err := monitor.GetStats()
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		fmt.Printf("Total syscalls captured: %d\n", stats)
	}

	// 8. Cleanup happens via defer monitor.Close()
}

// Integration with KNIRVNEXUS native runtime
func IntegrationExample() {
	// This shows how to integrate eBPF monitoring into skill validation

	// Instead of using strace (slow):
	// cmd := exec.Command("strace", "-f", "-e", "trace=all", "./skill")

	// Use eBPF monitoring (fast):
	monitor, err := NewSyscallMonitor()
	if err != nil {
		log.Fatalf("Failed to create monitor: %v", err)
	}
	defer monitor.Close()

	// Start the skill process
	cmd := exec.Command("./skill")
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start skill: %v", err)
	}

	// Monitor this specific PID
	monitor.AddPID(uint32(cmd.Process.Pid))
	monitor.Attach()

	// Collect suspicious activity
	suspiciousSyscalls := []string{"execve", "fork", "clone", "socket"}
	alerts := make([]string, 0)

	// Monitor for violations
	done := make(chan struct{})
	go func() {
		for event := range monitor.Events() {
			syscallName := GetSyscallName(event.SyscallNr)

			// Check for suspicious activity
			for _, suspicious := range suspiciousSyscalls {
				if syscallName == suspicious {
					alerts = append(alerts, fmt.Sprintf(
						"Suspicious syscall: %s at %d",
						syscallName, event.Timestamp))
				}
			}
		}
		close(done)
	}()

	// Wait for process with timeout
	waitChan := make(chan error, 1)
	go func() {
		waitChan <- cmd.Wait()
	}()

	select {
	case err := <-waitChan:
		if err != nil {
			log.Printf("Skill process failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		log.Printf("Skill process timeout - killed")
	}

	// Report results
	fmt.Printf("Alerts: %d\n", len(alerts))
	for _, alert := range alerts {
		fmt.Println(alert)
	}
}

// Performance comparison
func PerformanceExample() {
	// Benchmark: strace vs eBPF

	testCmd := []string{"/bin/bash", "-c", "for i in {1..1000}; do echo $i > /dev/null; done"}

	// Method 1: strace (traditional)
	start := time.Now()
	cmd := exec.Command("strace", append([]string{"-f", "-e", "trace=all"}, testCmd...)...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.Run()
	straceTime := time.Since(start)

	// Method 2: eBPF (modern)
	monitor, _ := NewSyscallMonitor()
	defer monitor.Close()

	start = time.Now()
	cmd = exec.Command(testCmd[0], testCmd[1:]...)
	cmd.Start()
	monitor.AddPID(uint32(cmd.Process.Pid))
	monitor.Attach()
	cmd.Wait()
	ebpfTime := time.Since(start)

	fmt.Printf("strace time: %v\n", straceTime)
	fmt.Printf("eBPF time: %v\n", ebpfTime)
	fmt.Printf("Speedup: %.2fx\n", float64(straceTime)/float64(ebpfTime))
	// Expected output: Speedup: 5-10x or more!
}
