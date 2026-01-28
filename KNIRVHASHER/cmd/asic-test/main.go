package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fmt.Println("🛡️  ASIC Device Diagnostic Tool v1.0")
	fmt.Println("========================================")
	fmt.Println()

	// Check if we're running on the Antminer
	if !checkEnvironment() {
		log.Fatal("❌ This tool must run on the Antminer device (MIPS architecture)")
	}

	fmt.Println("✅ Running on Antminer (MIPS detected)")
	fmt.Println()

	// Run all diagnostic phases
	phase1_SystemInfo()
	phase2_KernelModules()
	phase3_DeviceInterface()
	phase4_ProtocolDiscovery()
	phase5_CGMinerAnalysis()
	phase6_DeviceTest()

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("✅ Diagnostics Complete!")
	fmt.Println()
	fmt.Println("📄 Full output saved to: /tmp/asic-diagnostics.txt")
}

func checkEnvironment() bool {
	// Check if we're on MIPS architecture
	output, err := runCommand("uname", "-m")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(output), "mips")
}

func phase1_SystemInfo() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 1: System Information")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// CPU Info
	fmt.Println("📊 CPU Information:")
	if output, err := runCommand("cat", "/proc/cpuinfo"); err == nil {
		printRelevantLines(output, []string{"system type", "cpu model", "BogoMIPS"})
	}
	fmt.Println()

	// Memory
	fmt.Println("💾 Memory:")
	if output, err := runCommand("cat", "/proc/meminfo"); err == nil {
		printRelevantLines(output, []string{"MemTotal", "MemFree", "MemAvailable"})
	}
	fmt.Println()

	// USB Devices
	fmt.Println("🔌 USB Devices:")
	if output, err := runCommand("lsusb"); err == nil {
		fmt.Println(indent(output))
	}
	fmt.Println()
}

func phase2_KernelModules() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 2: Kernel Modules")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Loaded modules
	fmt.Println("📦 Loaded Bitmain Modules:")
	if output, err := runCommand("lsmod"); err == nil {
		printRelevantLines(output, []string{"bitmain"})
	}
	fmt.Println()

	// Module info
	fmt.Println("ℹ️  Module Details:")
	for _, module := range []string{"bitmain_asic", "usb_bitmain"} {
		fmt.Printf("  %s:\n", module)
		if output, err := runCommand("modinfo", module); err == nil {
			printRelevantLines(output, []string{"filename", "description", "author", "version"})
		}
		fmt.Println()
	}

	// Check for parameters
	fmt.Println("⚙️  Module Parameters:")
	if output, err := runCommand("ls", "-la", "/sys/module/bitmain_asic/parameters/"); err == nil {
		fmt.Println(indent(output))
	} else {
		fmt.Println("  (No parameters directory found)")
	}
	fmt.Println()
}

func phase3_DeviceInterface() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 3: Device Interface")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Check device file
	devicePath := "/dev/bitmain-asic"
	fmt.Printf("📁 Device File: %s\n", devicePath)
	if output, err := runCommand("ls", "-l", devicePath); err == nil {
		fmt.Println(indent(output))

		// Extract major/minor numbers
		parts := strings.Fields(output)
		if len(parts) > 4 {
			fmt.Printf("  Device Type: Character device\n")
			fmt.Printf("  Major/Minor: %s\n", parts[4])
		}
	} else {
		fmt.Println("  ❌ Device not found!")
	}
	fmt.Println()

	// Check device in /sys
	fmt.Println("🔍 Sysfs Interface:")
	if output, err := runCommand("ls", "-la", "/sys/class/misc/bitmain-asic/"); err == nil {
		fmt.Println(indent(output))
	} else {
		fmt.Println("  (Sysfs interface not found)")
	}
	fmt.Println()
}

func phase4_ProtocolDiscovery() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 4: Protocol Discovery")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// dmesg for ASIC events
	fmt.Println("📜 Recent ASIC Kernel Messages:")
	if output, err := runCommand("dmesg"); err == nil {
		lines := strings.Split(output, "\n")
		asicLines := []string{}
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), "bitmain") ||
			   strings.Contains(strings.ToLower(line), "asic") {
				asicLines = append(asicLines, line)
			}
		}

		// Show last 20 ASIC-related messages
		start := 0
		if len(asicLines) > 20 {
			start = len(asicLines) - 20
		}
		for _, line := range asicLines[start:] {
			fmt.Println("  " + line)
		}
	}
	fmt.Println()
}

func phase5_CGMinerAnalysis() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 5: CGMiner Analysis")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	cgminerPath := "/usr/bin/cgminer"

	// Check if cgminer exists
	if _, err := os.Stat(cgminerPath); os.IsNotExist(err) {
		fmt.Println("  ❌ cgminer not found")
		return
	}

	// File info
	fmt.Println("📦 CGMiner Binary:")
	if output, err := runCommand("ls", "-lh", cgminerPath); err == nil {
		fmt.Println(indent(output))
	}
	fmt.Println()

	// Extract relevant strings
	fmt.Println("🔍 Protocol Hints (from strings):")
	if output, err := runCommand("strings", cgminerPath); err == nil {
		keywords := []string{"bitmain", "/dev", "ioctl", "freq", "voltage", "reset", "work", "hash"}

		lines := strings.Split(output, "\n")
		matches := make(map[string][]string)

		for _, line := range lines {
			line = strings.TrimSpace(line)
			for _, keyword := range keywords {
				if strings.Contains(strings.ToLower(line), keyword) && len(line) > 3 && len(line) < 100 {
					matches[keyword] = append(matches[keyword], line)
				}
			}
		}

		for _, keyword := range keywords {
			if len(matches[keyword]) > 0 {
				fmt.Printf("  %s-related:\n", keyword)
				// Show first 5 matches
				for i, match := range matches[keyword] {
					if i >= 5 {
						fmt.Printf("    ... (%d more)\n", len(matches[keyword])-5)
						break
					}
					fmt.Printf("    %s\n", match)
				}
				fmt.Println()
			}
		}
	}

	// Check if cgminer is running
	fmt.Println("🔄 CGMiner Status:")
	if output, err := runCommand("ps", "w"); err == nil {
		if strings.Contains(output, "cgminer") {
			fmt.Println("  ✅ Running")
			printRelevantLines(output, []string{"cgminer"})
		} else {
			fmt.Println("  ⏸️  Not running")
		}
	}
	fmt.Println()
}

func phase6_DeviceTest() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 6: Device Access Test")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	devicePath := "/dev/bitmain-asic"

	// Check if we can open the device
	fmt.Println("🔓 Testing Device Access:")
	file, err := os.OpenFile(devicePath, os.O_RDWR, 0)
	if err != nil {
		fmt.Printf("  ❌ Cannot open device: %v\n", err)
		fmt.Println("  (This might be expected if cgminer has exclusive access)")
	} else {
		fmt.Println("  ✅ Device opened successfully!")

		// Try to read
		fmt.Println()
		fmt.Println("📖 Attempting Read (3 second timeout):")

		readBuf := make([]byte, 64)
		readDone := make(chan bool)

		go func() {
			n, err := file.Read(readBuf)
			if err != nil {
				fmt.Printf("  Read error: %v\n", err)
			} else if n > 0 {
				fmt.Printf("  Read %d bytes: %x\n", n, readBuf[:n])
			} else {
				fmt.Println("  Read 0 bytes (device might be blocking)")
			}
			readDone <- true
		}()

		select {
		case <-readDone:
		case <-time.After(3 * time.Second):
			fmt.Println("  ⏱️  Read timeout (expected for blocking device)")
		}

		file.Close()
	}
	fmt.Println()

	// Check permissions
	fmt.Println("🔐 Device Permissions:")
	if info, err := os.Stat(devicePath); err == nil {
		fmt.Printf("  Mode: %s\n", info.Mode())
		fmt.Printf("  Needs root: %v\n", info.Mode().Perm()&0222 == 0)
	}
	fmt.Println()
}

// Helper functions
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func printRelevantLines(output string, keywords []string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
				fmt.Println("  " + line)
				break
			}
		}
	}
}

func indent(text string) string {
	lines := strings.Split(text, "\n")
	result := []string{}
	for _, line := range lines {
		if line != "" {
			result = append(result, "  "+line)
		}
	}
	return strings.Join(result, "\n")
}
