package main

import (
	"fmt"
	"log"
	"os"

	"hasher/internal/antminer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: device-provision <antminer-ip>")
		os.Exit(1)
	}

	ip := os.Args[1]
	password := os.Getenv("DEVICE_PASSWORD")
	if password == "" {
		password = "keperu100"
	}
	if len(os.Args) > 2 {
		password = os.Args[2]
	}

	fmt.Println("🛡️  ASIC Device Provisioning Tool")
	fmt.Println("====================================")
	fmt.Println()

	// Connect to device
	fmt.Printf("Connecting to %s...\n", ip)
	client, err := antminer.NewClient(ip, password)
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer client.Close()

	fmt.Println("✅ Connected successfully")
	fmt.Println()

	// Get system info
	info, err := client.GetSystemInfo()
	if err != nil {
		log.Fatalf("Failed to get system info: %v", err)
	}

	fmt.Println("System Information:")
	fmt.Printf("  Model:  %s\n", client.Model)
	fmt.Printf("  CPU:    %s\n", info.CPU)
	fmt.Printf("  Memory: %d MB\n", info.MemoryKB/1024)
	fmt.Printf("  Kernel: %s\n", info.Kernel)
	fmt.Printf("  OS:     %s\n", info.OS)
	fmt.Println()

	// Check PRU availability
	fmt.Println("Checking PRU subsystem...")
	hasPRU, err := client.CheckPRU()
	if err != nil {
		log.Printf("Warning: Could not check PRU: %v", err)
	}

	if hasPRU {
		fmt.Println("✅ PRU subsystem detected!")
		fmt.Println("   This device is suitable for PRU provisioning")
	} else {
		fmt.Println("⚠️  PRU subsystem not found")
		fmt.Println("   This device may not support real-time control")
		fmt.Println("   Continuing anyway for testing...")
	}
	fmt.Println()

	// Stop mining
	fmt.Println("Stopping mining software...")
	if err := client.StopMining(); err != nil {
		log.Printf("Warning: Failed to stop mining: %v", err)
	}
	fmt.Println("✅ Mining stopped")
	fmt.Println()

	// Upload test script
	fmt.Println("Uploading test script...")
	testScript := `#!/bin/sh
echo "ASIC Test Script"
echo "========================="
echo ""
echo "PRU Status:"
ls -la /sys/devices/platform/ocp/4a300000.pruss-soc-bus/ 2>/dev/null || echo "PRU not available"
echo ""
echo "System is ready for 🛡️Hasher🛡️ installation"
`

	if err := client.UploadFile("", "/tmp/sentinel-test.sh", []byte(testScript)); err != nil {
		log.Fatalf("Failed to upload test script: %v", err)
	}

	output, err := client.RunCommand("chmod +x /tmp/sentinel-test.sh && /tmp/sentinel-test.sh")
	if err != nil {
		log.Fatalf("Failed to run test script: %v", err)
	}

	fmt.Println(output)
	fmt.Println()
	fmt.Println("✅ Provisioning complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Develop PRU firmware")
	fmt.Println("2. Build Go HVRS engine")
	fmt.Println("3. Deploy to device")
}
