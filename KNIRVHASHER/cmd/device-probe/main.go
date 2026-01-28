package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

const (
	DevicePath = "/dev/bitmain-asic"
)

func main() {
	fmt.Println("🔬 Hasher Device Probe Tool")
	fmt.Println("===================================")
	fmt.Println()

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("⚠️  Warning: Not running as root. Device access may fail.")
		fmt.Println("   Run with: sudo /tmp/device-probe")
		fmt.Println()
	}

	// Phase 1: Stop cgminer
	if !stopCGMiner() {
		fmt.Println("⚠️  Warning: Could not confirm cgminer stopped")
		fmt.Println("   Continuing anyway...")
		fmt.Println()
	}

	// Phase 2: Check device availability
	if !checkDevice() {
		fmt.Println("❌ Device not accessible. Exiting.")
		return
	}

	// Phase 3: Open device
	device, err := openDevice()
	if err != nil {
		fmt.Printf("❌ Failed to open device: %v\n", err)
		return
	}
	defer device.Close()

	fmt.Println("✅ Device opened successfully!")
	fmt.Println()

	// Phase 4: Device Information
	probeDeviceInfo(device)

	// Phase 5: Read Operations
	probeReadOperations(device)

	// Phase 6: Write Operations
	probeWriteOperations(device)

	// Phase 7: IOCTL Discovery
	probeIOCTLs(device)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("✅ Probe Complete!")
	fmt.Println()
	fmt.Println("💡 Next: Analyze the output to understand the protocol")
}

func stopCGMiner() bool {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 1: Stopping CGMiner")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Try init script first
	fmt.Println("Attempting: /etc/init.d/cgminer stop")
	cmd := exec.Command("/etc/init.d/cgminer", "stop")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  Init script failed: %v\n", err)
		fmt.Println("  Trying killall...")

		cmd = exec.Command("killall", "cgminer")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("  Killall failed: %v\n", err)
			return false
		}
	}

	fmt.Printf("  Output: %s\n", string(output))

	// Wait a bit for process to die
	time.Sleep(2 * time.Second)

	// Verify it's stopped
	cmd = exec.Command("pgrep", "cgminer")
	err = cmd.Run()
	if err != nil {
		fmt.Println("✅ CGMiner stopped")
		fmt.Println()
		return true
	}

	fmt.Println("⚠️  CGMiner may still be running")
	fmt.Println()
	return false
}

func checkDevice() bool {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 2: Device Availability Check")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	info, err := os.Stat(DevicePath)
	if err != nil {
		fmt.Printf("❌ Device not found: %v\n", err)
		return false
	}

	fmt.Printf("📁 Device: %s\n", DevicePath)
	fmt.Printf("   Mode: %s\n", info.Mode())
	fmt.Printf("   Size: %d\n", info.Size())

	// Get device numbers
	stat := info.Sys().(*syscall.Stat_t)
	major := uint64(stat.Rdev / 256)
	minor := uint64(stat.Rdev % 256)

	fmt.Printf("   Major: %d\n", major)
	fmt.Printf("   Minor: %d\n", minor)
	fmt.Println()

	return true
}

func openDevice() (*os.File, error) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 3: Opening Device")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	fmt.Println("Attempting to open with O_RDWR...")
	file, err := os.OpenFile(DevicePath, os.O_RDWR, 0)
	if err != nil {
		fmt.Printf("  O_RDWR failed: %v\n", err)

		fmt.Println("Attempting to open with O_RDONLY...")
		file, err = os.OpenFile(DevicePath, os.O_RDONLY, 0)
		if err != nil {
			return nil, err
		}
		fmt.Println("  ✅ Opened as read-only")
	} else {
		fmt.Println("  ✅ Opened as read-write")
	}
	fmt.Println()

	return file, nil
}

func probeDeviceInfo(device *os.File) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 4: Device Information")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Get file descriptor info
	fd := device.Fd()
	fmt.Printf("📊 File Descriptor: %d\n", fd)

	// Try to get flags using raw syscall (MIPS compatible)
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFL, 0)
	if errno == 0 {
		fmt.Printf("   Flags: 0x%x\n", flags)
		fmt.Printf("   Read: %v\n", (flags&syscall.O_RDONLY) != 0 || (flags&syscall.O_RDWR) != 0)
		fmt.Printf("   Write: %v\n", (flags&syscall.O_WRONLY) != 0 || (flags&syscall.O_RDWR) != 0)
	} else {
		fmt.Printf("   Could not get flags: %v\n", errno)
	}
	fmt.Println()
}

func probeReadOperations(device *os.File) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 5: Read Operations")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Test 1: Small read with timeout
	fmt.Println("📖 Test 1: Small read (64 bytes, 2 second timeout)")
	readWithTimeout(device, 64, 2*time.Second)

	// Test 2: Larger read
	fmt.Println("📖 Test 2: Larger read (256 bytes, 2 second timeout)")
	readWithTimeout(device, 256, 2*time.Second)

	// Test 3: Read at specific offset
	fmt.Println("📖 Test 3: Seek and read")
	device.Seek(0, 0) // Seek to start
	readWithTimeout(device, 64, 2*time.Second)
}

func readWithTimeout(device *os.File, size int, timeout time.Duration) {
	done := make(chan bool)
	buffer := make([]byte, size)
	var n int
	var err error

	go func() {
		n, err = device.Read(buffer)
		done <- true
	}()

	select {
	case <-done:
		if err != nil {
			fmt.Printf("  ❌ Read error: %v\n", err)
		} else if n == 0 {
			fmt.Println("  ⚠️  Read 0 bytes (no data available)")
		} else {
			fmt.Printf("  ✅ Read %d bytes:\n", n)
			fmt.Println("  Hex dump:")
			fmt.Printf("    %s\n", hex.Dump(buffer[:n]))
		}
	case <-time.After(timeout):
		fmt.Println("  ⏱️  Read timeout (device may be blocking)")
	}
	fmt.Println()
}

func probeWriteOperations(device *os.File) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 6: Write Operations")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Test 1: Write test pattern
	fmt.Println("✍️  Test 1: Write test pattern")
	testData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	n, err := device.Write(testData)
	if err != nil {
		fmt.Printf("  ❌ Write error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Wrote %d bytes\n", n)
		fmt.Printf("  Data: %s\n", hex.EncodeToString(testData))
	}
	fmt.Println()

	// Test 2: Write zero bytes
	fmt.Println("✍️  Test 2: Write zero pattern")
	zeroData := make([]byte, 16)
	n, err = device.Write(zeroData)
	if err != nil {
		fmt.Printf("  ❌ Write error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Wrote %d bytes (all zeros)\n", n)
	}
	fmt.Println()

	// Test 3: Write FF pattern
	fmt.Println("✍️  Test 3: Write 0xFF pattern")
	ffData := make([]byte, 16)
	for i := range ffData {
		ffData[i] = 0xFF
	}
	n, err = device.Write(ffData)
	if err != nil {
		fmt.Printf("  ❌ Write error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Wrote %d bytes (all 0xFF)\n", n)
	}
	fmt.Println()
}

func probeIOCTLs(device *os.File) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 7: IOCTL Discovery")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	fmt.Println("🔍 Testing common IOCTL commands...")
	fmt.Println("   (These are educated guesses based on typical device drivers)")
	fmt.Println()

	// Common IOCTL numbers to try
	ioctls := []struct {
		name string
		cmd  uintptr
	}{
		{"RESET", 0x00},
		{"GET_INFO", 0x01},
		{"SET_FREQ", 0x02},
		{"GET_FREQ", 0x03},
		{"SET_VOLTAGE", 0x04},
		{"GET_VOLTAGE", 0x05},
		{"START_WORK", 0x10},
		{"GET_RESULT", 0x11},
		{"FLUSH", 0x20},
	}

	for _, ioctl := range ioctls {
		// Create a buffer for potential return data
		buffer := make([]byte, 64)

		fmt.Printf("  Testing 0x%02x (%s)... ", ioctl.cmd, ioctl.name)

		_, _, errno := syscall.Syscall(
			syscall.SYS_IOCTL,
			device.Fd(),
			ioctl.cmd,
			uintptr(unsafe.Pointer(&buffer[0])),
		)

		if errno == 0 {
			fmt.Println("✅ Success!")
			// Check if buffer has data
			hasData := false
			for _, b := range buffer {
				if b != 0 {
					hasData = true
					break
				}
			}
			if hasData {
				fmt.Printf("    Returned data: %s\n", hex.EncodeToString(buffer[:16]))
			}
		} else if errno == syscall.EINVAL {
			fmt.Println("⚠️  Invalid (not supported)")
		} else if errno == syscall.ENOTTY {
			fmt.Println("⚠️  Not a typewriter (IOCTL not implemented)")
		} else {
			fmt.Printf("❌ Error: %v\n", errno)
		}
	}
	fmt.Println()

	// Try TCGETS (terminal control - probably won't work but worth trying)
	fmt.Println("🔍 Testing terminal IOCTLs (unlikely to work)...")
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		device.Fd(),
		syscall.TCGETS,
		0,
	)
	if errno == 0 {
		fmt.Println("  ✅ TCGETS worked (unexpected!)")
	} else {
		fmt.Printf("  ❌ TCGETS failed (expected): %v\n", errno)
	}
	fmt.Println()
}
