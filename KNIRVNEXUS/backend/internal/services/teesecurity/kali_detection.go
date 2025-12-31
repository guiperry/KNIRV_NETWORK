// File: backend/internal/services/teesecurity/kali_detection.go
package teesecurity

import (
	"os"
	"runtime"
	"strings"
	"log"
	"errors"
	"os/exec"
)

// KaliLinuxProfile represents detected Kali Linux security tools and capabilities
type KaliLinuxProfile struct {
	OS                    string                      // "kali", "ubuntu", "unknown"
	IsKaliLinux          bool
	KernelVersion        string
	ArchitectureSupport  []string                    // ["sgx", "sev-snp", "tdx"]

	// Kali Security Tools - Static Analysis
	StaticAnalysisTools  KaliStaticAnalysisTools

	// Kali Security Tools - Dynamic Analysis
	DynamicAnalysisTools KaliDynamicAnalysisTools

	// Kali Security Tools - Network Inspection
	NetworkAnalysisTools KaliNetworkAnalysisTools

	// Kali Security Tools - Forensics
	ForensicsTools       KaliForensicsTools

	// Security Framework Support
	SecurityFrameworks   KaliSecurityFrameworks

	// Container Runtime
	PreferredRuntime     string                      // "native-go", "podman"
}

// KaliStaticAnalysisTools tracks available static analysis capabilities
type KaliStaticAnalysisTools struct {
	Ghidra       bool // Binary disassembly and reverse engineering
	Radare2      bool // Reverse engineering framework
	Semgrep      bool // Static analysis and pattern matching
	Bandit       bool // Python security linter
}

// KaliDynamicAnalysisTools tracks available dynamic analysis capabilities
type KaliDynamicAnalysisTools struct {
	Strace       bool // System call tracing
	Ltrace       bool // Library call tracing
	Perf         bool // Performance analysis and profiling
	GDB          bool // Debugger for runtime analysis
}

// KaliNetworkAnalysisTools tracks available network inspection capabilities
type KaliNetworkAnalysisTools struct {
	Tcpdump      bool // Packet capture and analysis
	Tshark       bool // Wireshark CLI for packet inspection
	Mitmproxy    bool // MITM proxy for TLS inspection
	Iptables     bool // Network packet filtering
}

// KaliForensicsTools tracks available forensic analysis capabilities
type KaliForensicsTools struct {
	Volatility   bool // Memory forensics framework
	SleuthKit    bool // Filesystem forensics
	Autopsy      bool // Forensic analysis framework
}

// KaliSecurityFrameworks tracks available security frameworks
type KaliSecurityFrameworks struct {
	AppArmor     bool // Mandatory access control framework
	SELinux      bool // Security-Enhanced Linux
	Seccomp      bool // Secure computing mode
}

// DetectKaliEnvironment identifies the running OS and available Kali security tools
func DetectKaliEnvironment() (*KaliLinuxProfile, error) {
	profile := &KaliLinuxProfile{
		OS:                   runtime.GOOS,
		ArchitectureSupport: []string{},
	}

	if runtime.GOOS != "linux" {
		return nil, errors.New("TEE operations require Linux operating system")
	}

	// Read /etc/os-release for distribution info
	osRelease, err := readOSRelease()
	if err != nil {
		log.Printf("Warning: Could not read /etc/os-release: %v", err)
		profile.OS = "unknown"
		profile.PreferredRuntime = "podman"
		return profile, nil
	}

	osReleaseLower := strings.ToLower(osRelease)

	// Determine distribution
	if strings.Contains(osReleaseLower, "kali") {
		profile.OS = "kali"
		profile.IsKaliLinux = true
		profile.PreferredRuntime = "native-go" // Use native Go container runtime for Kali
	} else if strings.Contains(osReleaseLower, "debian") {
		profile.OS = "debian"
		// Check if Kali tools are installed (Debian with Kali tooling)
		if hasKaliTools() {
			log.Println("Detected Debian with Kali security tools - using native-go runtime")
			profile.IsKaliLinux = true // Treat as Kali-equivalent for runtime purposes
			profile.PreferredRuntime = "native-go"
		} else {
			profile.IsKaliLinux = false
			profile.PreferredRuntime = "podman"
		}
	} else if strings.Contains(osReleaseLower, "ubuntu") {
		profile.OS = "ubuntu"
		profile.IsKaliLinux = false
		profile.PreferredRuntime = "podman" // Podman fallback for Ubuntu
	} else {
		profile.OS = "unknown"
		profile.PreferredRuntime = "podman" // Default to Podman for other distributions
	}

	// Detect CPU capabilities for TEE
	profile.ArchitectureSupport = detectTEECapabilities()

	// If Kali Linux (or Debian with Kali tools), detect available security tools
	if profile.IsKaliLinux {
		detectKaliSecurityTools(profile)
	}

	return profile, nil
}

// hasKaliTools checks if essential Kali security tools are installed
func hasKaliTools() bool {
	// Check for core security tools that indicate a Kali-like environment
	essentialTools := []string{"strace", "radare2", "semgrep", "gdb"}
	foundCount := 0

	for _, tool := range essentialTools {
		if _, err := exec.LookPath(tool); err == nil {
			foundCount++
		}
	}

	// If at least 2 essential tools are found, consider it a Kali-tooled environment
	return foundCount >= 2
}

// readOSRelease reads /etc/os-release file
func readOSRelease() (string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// detectTEECapabilities checks CPU flags for TEE support
func detectTEECapabilities() []string {
	var capabilities []string

	// Read /proc/cpuinfo for CPU flags
	cpuInfo, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		log.Printf("Warning: Could not read /proc/cpuinfo: %v", err)
		return capabilities
	}

	cpuInfoStr := string(cpuInfo)

	// Check for SGX support
	if strings.Contains(cpuInfoStr, "sgx") {
		capabilities = append(capabilities, "sgx")
	}

	// Check for SEV support (AMD)
	if strings.Contains(cpuInfoStr, "sev") {
		capabilities = append(capabilities, "sev-snp")
	}

	// Check for TDX support (Intel)
	if strings.Contains(cpuInfoStr, "tdx") {
		capabilities = append(capabilities, "tdx")
	}

	return capabilities
}

// detectKaliSecurityTools checks for available Kali Linux security tools
func detectKaliSecurityTools(profile *KaliLinuxProfile) {
	log.Println("Detecting Kali Linux security tools...")

	// Static Analysis Tools
	profile.StaticAnalysisTools.Ghidra = commandExists("ghidra")
	profile.StaticAnalysisTools.Radare2 = commandExists("r2")
	profile.StaticAnalysisTools.Semgrep = commandExists("semgrep")
	profile.StaticAnalysisTools.Bandit = commandExists("bandit")

	// Dynamic Analysis Tools
	profile.DynamicAnalysisTools.Strace = commandExists("strace")
	profile.DynamicAnalysisTools.Ltrace = commandExists("ltrace")
	profile.DynamicAnalysisTools.Perf = commandExists("perf")
	profile.DynamicAnalysisTools.GDB = commandExists("gdb")

	// Network Analysis Tools
	profile.NetworkAnalysisTools.Tcpdump = commandExists("tcpdump")
	profile.NetworkAnalysisTools.Tshark = commandExists("tshark")
	profile.NetworkAnalysisTools.Mitmproxy = commandExists("mitmproxy")
	profile.NetworkAnalysisTools.Iptables = commandExists("iptables")

	// Forensics Tools
	profile.ForensicsTools.Volatility = commandExists("volatility") || commandExists("vol")
	profile.ForensicsTools.SleuthKit = commandExists("fls") || commandExists("istat")
	profile.ForensicsTools.Autopsy = commandExists("autopsy")

	// Security Frameworks
	profile.SecurityFrameworks.AppArmor = securityModuleLoaded("apparmor")
	profile.SecurityFrameworks.SELinux = securityModuleLoaded("selinux")
	profile.SecurityFrameworks.Seccomp = securityModuleLoaded("seccomp")

	logKaliToolsDetected(profile)
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// securityModuleLoaded checks if a security module is available
func securityModuleLoaded(module string) bool {
	// Check /sys/module for loaded modules
	modulePath := "/sys/module/" + module
	if _, err := os.Stat(modulePath); err == nil {
		return true
	}

	// Alternative: check /proc/modules
	modulesData, err := os.ReadFile("/proc/modules")
	if err != nil {
		return false
	}
	return strings.Contains(string(modulesData), module)
}

// logKaliToolsDetected logs available Kali security tools
func logKaliToolsDetected(profile *KaliLinuxProfile) {
	log.Println("=== Kali Linux Security Tools Detected ===")

	log.Println("Static Analysis:")
	log.Printf("  Ghidra: %v", profile.StaticAnalysisTools.Ghidra)
	log.Printf("  Radare2: %v", profile.StaticAnalysisTools.Radare2)
	log.Printf("  Semgrep: %v", profile.StaticAnalysisTools.Semgrep)
	log.Printf("  Bandit: %v", profile.StaticAnalysisTools.Bandit)

	log.Println("Dynamic Analysis:")
	log.Printf("  strace: %v", profile.DynamicAnalysisTools.Strace)
	log.Printf("  ltrace: %v", profile.DynamicAnalysisTools.Ltrace)
	log.Printf("  perf: %v", profile.DynamicAnalysisTools.Perf)
	log.Printf("  gdb: %v", profile.DynamicAnalysisTools.GDB)

	log.Println("Network Analysis:")
	log.Printf("  tcpdump: %v", profile.NetworkAnalysisTools.Tcpdump)
	log.Printf("  tshark: %v", profile.NetworkAnalysisTools.Tshark)
	log.Printf("  mitmproxy: %v", profile.NetworkAnalysisTools.Mitmproxy)
	log.Printf("  iptables: %v", profile.NetworkAnalysisTools.Iptables)

	log.Println("Forensics:")
	log.Printf("  Volatility: %v", profile.ForensicsTools.Volatility)
	log.Printf("  SleuthKit: %v", profile.ForensicsTools.SleuthKit)
	log.Printf("  Autopsy: %v", profile.ForensicsTools.Autopsy)

	log.Println("Security Frameworks:")
	log.Printf("  AppArmor: %v", profile.SecurityFrameworks.AppArmor)
	log.Printf("  SELinux: %v", profile.SecurityFrameworks.SELinux)
	log.Printf("  Seccomp: %v", profile.SecurityFrameworks.Seccomp)
}
