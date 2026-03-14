package host

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SystemInfoCollector collects and monitors system information
type SystemInfoCollector struct {
	ctx      context.Context
	interval time.Duration
	mu       sync.RWMutex

	currentInfo *SystemInfo
	lastUpdate  time.Time
	running     bool
}

// SystemInfo contains comprehensive system information
type SystemInfo struct {
	// Basic system info
	Hostname      string    `json:"hostname"`
	OS            string    `json:"os"`
	Architecture  string    `json:"architecture"`
	KernelVersion string    `json:"kernel_version"`
	Uptime        string    `json:"uptime"`
	LoadAverage   []float64 `json:"load_average"`

	// Kali Linux specific
	KaliVersion   string   `json:"kali_version"`
	KaliCodename  string   `json:"kali_codename"`
	SecurityTools []string `json:"security_tools"`

	// Hardware info
	CPUInfo     *CPUInfo                `json:"cpu_info"`
	MemoryInfo  *MemoryInfo             `json:"memory_info"`
	DiskInfo    []*DiskInfo             `json:"disk_info"`
	NetworkInfo []*NetworkInterfaceInfo `json:"network_info"`

	// TEE Support
	TEESupport []string `json:"tee_support"`
	SGXSupport bool     `json:"sgx_support"`
	SEVSupport bool     `json:"sev_support"`
	TDXSupport bool     `json:"tdx_support"`

	// Timestamps
	CollectedAt time.Time `json:"collected_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CPUInfo contains CPU information
type CPUInfo struct {
	Model       string  `json:"model"`
	Cores       int     `json:"cores"`
	Threads     int     `json:"threads"`
	Frequency   float64 `json:"frequency_mhz"`
	Usage       float64 `json:"usage_percent"`
	Temperature float64 `json:"temperature_celsius"`
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	Total     uint64  `json:"total_bytes"`
	Available uint64  `json:"available_bytes"`
	Used      uint64  `json:"used_bytes"`
	Free      uint64  `json:"free_bytes"`
	Cached    uint64  `json:"cached_bytes"`
	Buffers   uint64  `json:"buffers_bytes"`
	Usage     float64 `json:"usage_percent"`
}

// DiskInfo contains disk information
type DiskInfo struct {
	Device     string  `json:"device"`
	Mountpoint string  `json:"mountpoint"`
	Filesystem string  `json:"filesystem"`
	Total      uint64  `json:"total_bytes"`
	Used       uint64  `json:"used_bytes"`
	Available  uint64  `json:"available_bytes"`
	Usage      float64 `json:"usage_percent"`
}

// NetworkInterfaceInfo contains network interface information
type NetworkInterfaceInfo struct {
	Interface string `json:"interface"`
	IPAddress string `json:"ip_address"`
	MACAddr   string `json:"mac_address"`
	Status    string `json:"status"`
	Speed     string `json:"speed"`
	RxBytes   uint64 `json:"rx_bytes"`
	TxBytes   uint64 `json:"tx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	TxPackets uint64 `json:"tx_packets"`
}

// NewSystemInfoCollector creates a new system information collector
func NewSystemInfoCollector(ctx context.Context, interval time.Duration) (*SystemInfoCollector, error) {
	collector := &SystemInfoCollector{
		ctx:      ctx,
		interval: interval,
	}

	// Collect initial system information
	if err := collector.collectSystemInfo(); err != nil {
		return nil, fmt.Errorf("failed to collect initial system info: %w", err)
	}

	return collector, nil
}

// Start begins periodic system information collection
func (sic *SystemInfoCollector) Start() error {
	sic.mu.Lock()
	defer sic.mu.Unlock()

	if sic.running {
		return fmt.Errorf("system info collector is already running")
	}

	sic.running = true

	go sic.collectLoop()

	return nil
}

// Stop stops the system information collection
func (sic *SystemInfoCollector) Stop() error {
	sic.mu.Lock()
	defer sic.mu.Unlock()

	sic.running = false
	return nil
}

// GetCurrentInfo returns the current system information
func (sic *SystemInfoCollector) GetCurrentInfo() (*SystemInfo, error) {
	sic.mu.RLock()
	defer sic.mu.RUnlock()

	if sic.currentInfo == nil {
		return nil, fmt.Errorf("no system information available")
	}

	// Return a copy to prevent modification
	info := *sic.currentInfo
	return &info, nil
}

// HealthCheck verifies the collector is working properly
func (sic *SystemInfoCollector) HealthCheck() error {
	sic.mu.RLock()
	defer sic.mu.RUnlock()

	if !sic.running {
		return fmt.Errorf("system info collector is not running")
	}

	if sic.currentInfo == nil {
		return fmt.Errorf("no system information collected")
	}

	// Check if data is stale (more than 2x the collection interval)
	if time.Since(sic.lastUpdate) > sic.interval*2 {
		return fmt.Errorf("system information is stale (last update: %v)", sic.lastUpdate)
	}

	return nil
}

// collectLoop runs the periodic collection loop
func (sic *SystemInfoCollector) collectLoop() {
	ticker := time.NewTicker(sic.interval)
	defer ticker.Stop()

	for {
		select {
		case <-sic.ctx.Done():
			return
		case <-ticker.C:
			sic.mu.RLock()
			running := sic.running
			sic.mu.RUnlock()

			if !running {
				return
			}

			if err := sic.collectSystemInfo(); err != nil {
				// Log error but continue collecting
				fmt.Printf("Error collecting system info: %v\n", err)
			}
		}
	}
}

// collectSystemInfo collects current system information
func (sic *SystemInfoCollector) collectSystemInfo() error {
	info := &SystemInfo{
		CollectedAt: time.Now(),
	}

	// Basic system information
	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	info.OS = runtime.GOOS
	info.Architecture = runtime.GOARCH

	// Kernel version
	if kernelVersion, err := sic.getKernelVersion(); err == nil {
		info.KernelVersion = kernelVersion
	}

	// Uptime
	if uptime, err := sic.getUptime(); err == nil {
		info.Uptime = uptime
	}

	// Load average
	if loadAvg, err := sic.getLoadAverage(); err == nil {
		info.LoadAverage = loadAvg
	}

	// Kali Linux specific information
	if kaliVersion, codename, err := sic.getKaliInfo(); err == nil {
		info.KaliVersion = kaliVersion
		info.KaliCodename = codename
	}

	// Security tools
	info.SecurityTools = sic.getSecurityTools()

	// CPU information
	if cpuInfo, err := sic.getCPUInfo(); err == nil {
		info.CPUInfo = cpuInfo
	}

	// Memory information
	if memInfo, err := sic.getMemoryInfo(); err == nil {
		info.MemoryInfo = memInfo
	}

	// Disk information
	if diskInfo, err := sic.getDiskInfo(); err == nil {
		info.DiskInfo = diskInfo
	}

	// TEE support detection
	info.TEESupport, info.SGXSupport, info.SEVSupport, info.TDXSupport = sic.detectTEESupport()

	info.UpdatedAt = time.Now()

	// Update current info
	sic.mu.Lock()
	sic.currentInfo = info
	sic.lastUpdate = time.Now()
	sic.mu.Unlock()

	return nil
}

// getKernelVersion gets the kernel version
func (sic *SystemInfoCollector) getKernelVersion() (string, error) {
	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getUptime gets system uptime
func (sic *SystemInfoCollector) getUptime() (string, error) {
	cmd := exec.Command("uptime", "-p")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getLoadAverage gets system load average
func (sic *SystemInfoCollector) getLoadAverage() ([]float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return nil, fmt.Errorf("invalid loadavg format")
	}

	var loadAvg []float64
	for i := 0; i < 3; i++ {
		val, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return nil, err
		}
		loadAvg = append(loadAvg, val)
	}

	return loadAvg, nil
}

// getKaliInfo gets Kali Linux specific information
func (sic *SystemInfoCollector) getKaliInfo() (string, string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", "", err
	}

	var version, codename string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VERSION=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION="), "\"")
		} else if strings.HasPrefix(line, "VERSION_CODENAME=") {
			codename = strings.Trim(strings.TrimPrefix(line, "VERSION_CODENAME="), "\"")
		}
	}

	return version, codename, nil
}

// getSecurityTools gets list of installed security tools
func (sic *SystemInfoCollector) getSecurityTools() []string {
	tools := []string{
		"nmap", "wireshark", "metasploit-framework", "burpsuite",
		"sqlmap", "nikto", "dirb", "gobuster", "hydra", "john",
		"hashcat", "aircrack-ng", "reaver", "ettercap", "tcpdump",
	}

	var installed []string
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			installed = append(installed, tool)
		}
	}

	return installed
}

// detectTEESupport detects TEE (Trusted Execution Environment) support
func (sic *SystemInfoCollector) detectTEESupport() ([]string, bool, bool, bool) {
	var teeSupport []string
	var sgx, sev, tdx bool

	// Check for Intel SGX
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		if strings.Contains(string(data), "sgx") {
			sgx = true
			teeSupport = append(teeSupport, "SGX")
		}
	}

	// Check for AMD SEV
	if _, err := os.Stat("/dev/sev"); err == nil {
		sev = true
		teeSupport = append(teeSupport, "SEV-SNP")
	}

	// Check for Intel TDX
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		if strings.Contains(string(data), "tdx") {
			tdx = true
			teeSupport = append(teeSupport, "TDX")
		}
	}

	return teeSupport, sgx, sev, tdx
}

// getCPUInfo gets CPU information
func (sic *SystemInfoCollector) getCPUInfo() (*CPUInfo, error) {
	cpuInfo := &CPUInfo{}

	// Read CPU info from /proc/cpuinfo
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "model name") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				cpuInfo.Model = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	// Get CPU count
	cpuInfo.Cores = runtime.NumCPU()
	cpuInfo.Threads = runtime.NumCPU() // Simplified

	// Get CPU usage (simplified)
	if usage, err := sic.getCPUUsage(); err == nil {
		cpuInfo.Usage = usage
	}

	return cpuInfo, nil
}

// getCPUUsage gets current CPU usage percentage
func (sic *SystemInfoCollector) getCPUUsage() (float64, error) {
	// Read /proc/stat for CPU usage calculation
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0, fmt.Errorf("no CPU data found")
	}

	// Parse first line (overall CPU)
	fields := strings.Fields(lines[0])
	if len(fields) < 8 || fields[0] != "cpu" {
		return 0, fmt.Errorf("invalid CPU data format")
	}

	// Calculate usage (simplified)
	var total, idle uint64
	for i := 1; i < len(fields) && i < 8; i++ {
		val, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			continue
		}
		total += val
		if i == 4 { // idle time
			idle = val
		}
	}

	if total == 0 {
		return 0, nil
	}

	usage := float64(total-idle) / float64(total) * 100
	return usage, nil
}

// getMemoryInfo gets memory information
func (sic *SystemInfoCollector) getMemoryInfo() (*MemoryInfo, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}

	memInfo := &MemoryInfo{}
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		// Convert from kB to bytes
		value *= 1024

		switch key {
		case "MemTotal":
			memInfo.Total = value
		case "MemAvailable":
			memInfo.Available = value
		case "MemFree":
			memInfo.Free = value
		case "Cached":
			memInfo.Cached = value
		case "Buffers":
			memInfo.Buffers = value
		}
	}

	memInfo.Used = memInfo.Total - memInfo.Available
	if memInfo.Total > 0 {
		memInfo.Usage = float64(memInfo.Used) / float64(memInfo.Total) * 100
	}

	return memInfo, nil
}

// getDiskInfo gets disk information
func (sic *SystemInfoCollector) getDiskInfo() ([]*DiskInfo, error) {
	cmd := exec.Command("df", "-h")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var diskInfos []*DiskInfo
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || line == "" { // Skip header and empty lines
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// Skip special filesystems
		if strings.HasPrefix(fields[0], "tmpfs") ||
			strings.HasPrefix(fields[0], "udev") ||
			strings.HasPrefix(fields[0], "devpts") {
			continue
		}

		diskInfo := &DiskInfo{
			Device:     fields[0],
			Mountpoint: fields[5],
			Filesystem: "unknown", // df doesn't provide filesystem type
		}

		// Parse sizes (simplified - assumes human readable format)
		if total, err := sic.parseSize(fields[1]); err == nil {
			diskInfo.Total = total
		}
		if used, err := sic.parseSize(fields[2]); err == nil {
			diskInfo.Used = used
		}
		if available, err := sic.parseSize(fields[3]); err == nil {
			diskInfo.Available = available
		}

		// Parse usage percentage
		usageStr := strings.TrimSuffix(fields[4], "%")
		if usage, err := strconv.ParseFloat(usageStr, 64); err == nil {
			diskInfo.Usage = usage
		}

		diskInfos = append(diskInfos, diskInfo)
	}

	return diskInfos, nil
}

// parseSize parses human-readable size strings (e.g., "1.5G", "512M")
func (sic *SystemInfoCollector) parseSize(sizeStr string) (uint64, error) {
	if len(sizeStr) == 0 {
		return 0, fmt.Errorf("empty size string")
	}

	// Get the last character (unit)
	unit := sizeStr[len(sizeStr)-1:]
	valueStr := sizeStr[:len(sizeStr)-1]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, err
	}

	var multiplier uint64 = 1
	switch strings.ToUpper(unit) {
	case "K":
		multiplier = 1024
	case "M":
		multiplier = 1024 * 1024
	case "G":
		multiplier = 1024 * 1024 * 1024
	case "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		// Assume bytes if no unit
		if val, err := strconv.ParseUint(sizeStr, 10, 64); err == nil {
			return val, nil
		}
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}

	return uint64(value * float64(multiplier)), nil
}
