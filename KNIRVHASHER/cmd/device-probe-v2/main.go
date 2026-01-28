package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	DevicePath = "/dev/bitmain-asic"
)

func main() {
	fmt.Println("🔬 Hasher Device Probe Tool v2")
	fmt.Println("======================================")
	fmt.Println()

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("⚠️  Warning: Not running as root. Device access may fail.")
		fmt.Println("   Run with: sudo /tmp/device-probe-v2")
		fmt.Println()
	}

	// Phase 1: Enhanced CGMiner stop with verification
	stopCGMinerEnhanced()

	// Phase 2: Check what processes have the device open
	checkDeviceUsers()

	// Phase 3: Check kernel modules
	checkKernelModules()

	// Phase 4: Check device availability
	if !checkDevice() {
		fmt.Println("❌ Device not accessible. Exiting.")
		return
	}

	// Phase 5: Try to forcefully close any lingering handles
	tryForceClose()

	// Phase 6: Attempt device open with different strategies
	device, err := openDeviceEnhanced()
	if err != nil {
		fmt.Printf("❌ All device open attempts failed: %v\n", err)
		fmt.Println()
		fmt.Println("💡 Suggestions:")
		fmt.Println("   1. Try: rmmod bitmain_asic (if module exists)")
		fmt.Println("   2. Try: killall -9 cgminer")
		fmt.Println("   3. Reboot the device")
		fmt.Println("   4. Check dmesg for kernel errors")
		return
	}
	defer device.Close()

	fmt.Println("✅ Device opened successfully!")
	fmt.Println()

	// Phase 7: Device Information
	probeDeviceInfo(device)

	// Phase 8: Write Operations
	probeWriteOperations(device)

	// Phase 9: IOCTL Discovery
	probeIOCTLs(device)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("✅ Probe Complete!")
	fmt.Println()
	fmt.Println("💡 Next: Analyze the output to understand the protocol")
}

func stopCGMinerEnhanced() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 1: Stopping CGMiner (Enhanced)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Check if cgminer is running first
	cmd := exec.Command("pgrep", "cgminer")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("✅ CGMiner is not running")
		fmt.Println()
		return
	}

	pids := strings.TrimSpace(string(output))
	fmt.Printf("📌 CGMiner PIDs found: %s\n", pids)
	fmt.Println()

	// Try init script first
	fmt.Println("Attempting: /etc/init.d/cgminer stop")
	cmd = exec.Command("/etc/init.d/cgminer", "stop")
	output, err = cmd.CombinedOutput()
	if err == nil {
		fmt.Printf("  Output: %s\n", string(output))
	} else {
		fmt.Printf("  Init script failed: %v\n", err)
	}

	// Wait longer for graceful shutdown
	fmt.Println("Waiting 5 seconds for graceful shutdown...")
	time.Sleep(5 * time.Second)

	// Check if still running
	cmd = exec.Command("pgrep", "cgminer")
	err = cmd.Run()
	if err != nil {
		fmt.Println("✅ CGMiner stopped gracefully")
		fmt.Println()
		return
	}

	// Force kill if still running
	fmt.Println("⚠️  CGMiner still running, using SIGKILL...")
	cmd = exec.Command("killall", "-9", "cgminer")
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  Killall failed: %v\n", err)
	} else {
		fmt.Printf("  Output: %s\n", string(output))
	}

	// Wait for processes to die
	time.Sleep(2 * time.Second)

	// Final verification
	cmd = exec.Command("pgrep", "cgminer")
	err = cmd.Run()
	if err != nil {
		fmt.Println("✅ CGMiner stopped")
	} else {
		fmt.Println("❌ CGMiner still running!")
	}
	fmt.Println()
}

func checkDeviceUsers() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 2: Check Device Users")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Use lsof to check what has the device open
	cmd := exec.Command("lsof", DevicePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Exit code 1 means no processes found (which is good)
		if strings.Contains(err.Error(), "exit status 1") {
			fmt.Println("✅ No processes have the device open")
		} else {
			fmt.Printf("⚠️  lsof command failed: %v\n", err)
		}
	} else {
		fmt.Println("📋 Processes with device open:")
		fmt.Println(string(output))
	}
	fmt.Println()

	// Also check /proc/*/fd/ for any open file descriptors
	cmd = exec.Command("sh", "-c", fmt.Sprintf("find /proc/*/fd -ls 2>/dev/null | grep %s || true", DevicePath))
	output, err = cmd.CombinedOutput()
	if err == nil && len(output) > 0 {
		fmt.Println("📋 File descriptors pointing to device:")
		fmt.Println(string(output))
	}
	fmt.Println()
}

func checkKernelModules() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 3: Kernel Modules")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Check for bitmain-related modules
	cmd := exec.Command("lsmod")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("⚠️  lsmod failed: %v\n", err)
	} else {
		lines := strings.Split(string(output), "\n")
		foundAny := false
		for _, line := range lines {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "bitmain") || strings.Contains(lower, "asic") {
				if !foundAny {
					fmt.Println("📋 Bitmain/ASIC modules:")
					foundAny = true
				}
				fmt.Println("  ", line)
			}
		}
		if !foundAny {
			fmt.Println("ℹ️  No bitmain/asic modules found in lsmod")
		}
	}
	fmt.Println()

	// Check /proc/devices for the character device
	cmd = exec.Command("grep", "-i", "bitmain", "/proc/devices")
	output, err = cmd.CombinedOutput()
	if err == nil && len(output) > 0 {
		fmt.Println("📋 /proc/devices entry:")
		fmt.Println("  ", strings.TrimSpace(string(output)))
	}
	fmt.Println()
}

func checkDevice() bool {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 4: Device Availability Check")
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

func tryForceClose() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 5: Force Close Lingering Handles")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Try to get exclusive access using flock
	fmt.Println("Attempting to break any advisory locks...")

	// This is just informational - we can't actually force-close from userspace
	fmt.Println("ℹ️  Note: Cannot force-close kernel driver handles from userspace")
	fmt.Println("   If device is still busy, kernel module may need to be reloaded")
	fmt.Println()
}

func openDeviceEnhanced() (*os.File, error) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 6: Opening Device (Enhanced)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Strategy 1: O_RDWR
	fmt.Println("Strategy 1: O_RDWR")
	file, err := os.OpenFile(DevicePath, os.O_RDWR, 0)
	if err != nil {
		fmt.Printf("  ❌ Failed: %v\n", err)
	} else {
		fmt.Println("  ✅ Success!")
		fmt.Println()
		return file, nil
	}

	// Strategy 2: O_RDONLY
	fmt.Println("Strategy 2: O_RDONLY")
	file, err = os.OpenFile(DevicePath, os.O_RDONLY, 0)
	if err != nil {
		fmt.Printf("  ❌ Failed: %v\n", err)
	} else {
		fmt.Println("  ✅ Success!")
		fmt.Println()
		return file, nil
	}

	// Strategy 3: O_RDWR | O_NONBLOCK
	fmt.Println("Strategy 3: O_RDWR | O_NONBLOCK")
	file, err = os.OpenFile(DevicePath, os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		fmt.Printf("  ❌ Failed: %v\n", err)
	} else {
		fmt.Println("  ✅ Success!")
		fmt.Println()
		return file, nil
	}

	// Strategy 4: O_RDONLY | O_NONBLOCK
	fmt.Println("Strategy 4: O_RDONLY | O_NONBLOCK")
	file, err = os.OpenFile(DevicePath, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		fmt.Printf("  ❌ Failed: %v\n", err)
	} else {
		fmt.Println("  ✅ Success!")
		fmt.Println()
		return file, nil
	}

	fmt.Println()
	return nil, fmt.Errorf("all open strategies failed")
}

func probeDeviceInfo(device *os.File) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 7: Device Information")
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
	fmt.Println("Phase 8: Read Operations")
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

// crc16 calculates the CRC-16-IBM checksum.
func crc16(data []byte) uint16 {
	var crc uint16 = 0xFFFF
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if (crc & 1) == 1 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func probeWriteOperations(device *os.File) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 9: Write Operations (Protocol-Aware)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Test 1: Write a valid RXSTATUS request packet
	fmt.Println("✍️  Test 1: Write valid RXSTATUS request")
	// As per PROTOCOL.md: Token 0x53, Version 0x00, Length 2 (for CRC)
	header := []byte{0x53, 0x00, 0x02, 0x00}
	checksum := crc16(header)
	packet := append(header, byte(checksum&0xFF), byte(checksum>>8))

	fmt.Printf("  Constructed Packet: %s\n", hex.EncodeToString(packet))
	fmt.Printf("    - Header: %s\n", hex.EncodeToString(header))
	fmt.Printf("    - CRC16: 0x%04x\n", checksum)
	fmt.Println()

	n, err := device.Write(packet)
	if err != nil {
		fmt.Printf("  ❌ Write error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Wrote %d bytes successfully\n", n)
	}
	fmt.Println()
	fmt.Println("💡 IMPORTANT:")
	fmt.Println("   Check the kernel log on the device now!")
	fmt.Println("   Run: dmesg | tail")
	fmt.Println("   A successful write will NOT produce a 'Tx token err' message.")
	fmt.Println("   A 'Tx token err' means the write was rejected by the driver.")
	fmt.Println()

	// Test 2: Write an invalid packet to confirm driver rejection
	fmt.Println("✍️  Test 2: Write invalid packet (for comparison)")
	invalidPacket := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	fmt.Printf("  Writing invalid packet: %s\n", hex.EncodeToString(invalidPacket))

	n, err = device.Write(invalidPacket)
	if err != nil {
		fmt.Printf("  ❌ Write error: %v\n", err)
		fmt.Println("     This is expected for invalid packets if the driver enforces rules.")
	} else {
		fmt.Printf("  ✅ Wrote %d bytes successfully\n", n)
		fmt.Println("     Now check dmesg. You SHOULD see a 'Tx token err {0xde}' message.")
	}
	fmt.Println()
}

func probeIOCTLs(device *os.File) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Phase 10: IOCTL Discovery")
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
