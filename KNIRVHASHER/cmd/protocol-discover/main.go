package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const (
	DevicePath = "/dev/bitmain-asic"
)

func main() {
	fmt.Println("🔍 Hasher Protocol Discovery Tool")
	fmt.Println("========================================")
	fmt.Println()

	// Stop CGMiner first
	fmt.Println("Stopping CGMiner...")
	stopCGMiner()
	fmt.Println()

	// Open device
	fmt.Println("Opening device...")
	device, err := os.OpenFile(DevicePath, os.O_RDWR, 0)
	if err != nil {
		fmt.Printf("❌ Failed to open device: %v\n", err)
		return
	}
	defer device.Close()
	fmt.Println("✅ Device opened")
	fmt.Println()

	// Test various command patterns
	testPatterns := []struct {
		name string
		data []byte
		desc string
	}{
		{
			name: "NULL_COMMAND",
			data: []byte{0x00},
			desc: "Single null byte",
		},
		{
			name: "MAGIC_HEADER",
			data: []byte{0x55, 0xAA},
			desc: "Common UART magic bytes",
		},
		{
			name: "INIT_SEQUENCE",
			data: []byte{0x55, 0xAA, 0x00, 0x00},
			desc: "Possible initialization",
		},
		{
			name: "STATUS_REQUEST",
			data: []byte{0x01, 0x00, 0x00, 0x00},
			desc: "Status query",
		},
		{
			name: "VERSION_REQUEST",
			data: []byte{0x02, 0x00, 0x00, 0x00},
			desc: "Version query",
		},
		{
			name: "NONCE_PATTERN",
			data: []byte{0x21, 0x36, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00},
			desc: "Possible nonce command (from cgminer code)",
		},
		{
			name: "FREQ_PATTERN",
			data: []byte{0x48, 0x09, 0x00, 0xFA, 0x00, 0x00, 0x00, 0x00},
			desc: "Possible frequency set (250MHz = 0x0FA)",
		},
		{
			name: "RESET_PATTERN",
			data: []byte{0x55, 0xAA, 0x51, 0x09, 0x00, 0x00},
			desc: "Possible reset command",
		},
		{
			name: "WORK_PATTERN",
			data: []byte{
				0x21, 0x36, // Possible work command
				0x00, 0x01, 0x00, 0x00, 0x00, 0x00, // work ID?
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // midstate start
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // midstate end
				0x00, 0x00, 0x00, 0x00, // data start
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, // data end
			},
			desc: "Possible work submission (52 bytes)",
		},
		{
			name: "CHAIN_SELECT_0",
			data: []byte{0x41, 0x00},
			desc: "Select chain 0",
		},
		{
			name: "CHAIN_SELECT_1",
			data: []byte{0x41, 0x01},
			desc: "Select chain 1",
		},
		{
			name: "READ_RESULT",
			data: []byte{0x81, 0x00, 0x00, 0x00},
			desc: "Read result command",
		},
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Testing Command Patterns")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	for i, test := range testPatterns {
		fmt.Printf("[%d/%d] %s\n", i+1, len(testPatterns), test.name)
		fmt.Printf("  Description: %s\n", test.desc)
		fmt.Printf("  Data (%d bytes): %s\n", len(test.data), hex.EncodeToString(test.data))

		// Write command
		n, err := device.Write(test.data)
		if err != nil {
			fmt.Printf("  ❌ Write failed: %v\n", err)
			fmt.Println()
			continue
		}
		fmt.Printf("  ✅ Wrote %d bytes\n", n)

		// Try to read response (with timeout)
		response := make([]byte, 1024)
		device.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

		n, err = device.Read(response)
		if err != nil {
			if os.IsTimeout(err) {
				fmt.Println("  ⏱️  No response (timeout)")
			} else {
				fmt.Printf("  ⚠️  Read error: %v\n", err)
			}
		} else if n > 0 {
			fmt.Printf("  📖 Response (%d bytes):\n", n)
			fmt.Println(hex.Dump(response[:n]))
		} else {
			fmt.Println("  ⚠️  Read 0 bytes")
		}

		fmt.Println()

		// Small delay between tests
		time.Sleep(100 * time.Millisecond)
	}

	// Test: Send multiple writes and then try to read
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Testing Burst Write + Read")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Send a series of commands
	burstCommands := [][]byte{
		{0x55, 0xAA},             // Magic
		{0x01, 0x00},             // Command 1
		{0x02, 0x00},             // Command 2
		{0x81, 0x00, 0x00, 0x00}, // Read request
	}

	for i, cmd := range burstCommands {
		fmt.Printf("  Writing burst command %d: %s\n", i+1, hex.EncodeToString(cmd))
		device.Write(cmd)
		time.Sleep(50 * time.Millisecond)
	}

	// Try to read any accumulated response
	fmt.Println()
	fmt.Println("  Attempting to read response...")
	response := make([]byte, 1024)
	device.SetReadDeadline(time.Now().Add(1 * time.Second))

	n, err := device.Read(response)
	if err != nil {
		if os.IsTimeout(err) {
			fmt.Println("  ⏱️  No response (timeout)")
		} else {
			fmt.Printf("  ⚠️  Read error: %v\n", err)
		}
	} else if n > 0 {
		fmt.Printf("  📖 Response (%d bytes):\n", n)
		fmt.Println(hex.Dump(response[:n]))
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Protocol Discovery Complete!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("💡 Next Steps:")
	fmt.Println("   1. Analyze any responses received")
	fmt.Println("   2. Look for patterns in successful commands")
	fmt.Println("   3. Check cgminer source code for command format")
	fmt.Println("   4. Build protocol specification from findings")
	fmt.Println()
}

func stopCGMiner() {
	// Check if cgminer is running first
	cmd := exec.Command("pgrep", "cgminer")
	err := cmd.Run()
	if err != nil {
		fmt.Println("✅ CGMiner is not running")
		return
	}

	// Try init script first
	fmt.Println("Stopping CGMiner gracefully...")
	cmd = exec.Command("/etc/init.d/cgminer", "stop")
	cmd.Run()

	// Wait longer for graceful shutdown
	fmt.Println("Waiting 5 seconds for graceful shutdown...")
	time.Sleep(5 * time.Second)

	// Check if still running
	cmd = exec.Command("pgrep", "cgminer")
	err = cmd.Run()
	if err != nil {
		fmt.Println("✅ CGMiner stopped gracefully")
		return
	}

	// Force kill if still running
	fmt.Println("Force killing CGMiner...")
	cmd = exec.Command("killall", "-9", "cgminer")
	cmd.Run()
	time.Sleep(2 * time.Second)

	// Final check
	cmd = exec.Command("pgrep", "cgminer")
	err = cmd.Run()
	if err != nil {
		fmt.Println("✅ CGMiner stopped")
	} else {
		fmt.Println("⚠️  CGMiner may still be running")
	}
}
