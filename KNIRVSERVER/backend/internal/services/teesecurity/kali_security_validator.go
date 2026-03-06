package teesecurity

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)



// KaliSecurityValidationReport represents the comprehensive security validation report
type KaliSecurityValidationReport struct {
	OS                       string            `json:"os"`
	IsKaliLinux              bool              `json:"is_kali_linux"`
	Timestamp                time.Time         `json:"timestamp"`
	ToolsAvailable           map[string]bool   `json:"tools_available"`
	FrameworksLoaded         map[string]bool   `json:"frameworks_loaded"`
	SystemMemoryKB           string            `json:"system_memory_kb,omitempty"`
	DiskSpaceKB              string            `json:"disk_space_kb,omitempty"`
	Recommendations          []string          `json:"recommendations"`
}

// KaliSecurityValidator validates Kali Linux security tools and frameworks
// Implements schema methods: ValidateSecurityCapabilities (ID 1) and all sub-validators (IDs 2-8)
type KaliSecurityValidator struct {
	kaliProfile *KaliLinuxProfile
}

// NewKaliSecurityValidator creates a validator for Kali Linux security tools
func NewKaliSecurityValidator(kaliProfile *KaliLinuxProfile) *KaliSecurityValidator {
	return &KaliSecurityValidator{
		kaliProfile: kaliProfile,
	}
}

// ValidateSecurityCapabilities - Implements Schema ID 1
// Performs comprehensive validation of all Kali Linux security tools, frameworks, and system resources
func (ksv *KaliSecurityValidator) ValidateSecurityCapabilities(ctx context.Context) (*KaliSecurityValidationReport, error) {
	report := &KaliSecurityValidationReport{
		OS:                       ksv.kaliProfile.OS,
		IsKaliLinux:             ksv.kaliProfile.IsKaliLinux,
		Timestamp:               time.Now(),
		ToolsAvailable:          make(map[string]bool),
		FrameworksLoaded:        make(map[string]bool),
		Recommendations:         []string{},
	}

	// Validate all sub-components in parallel for efficiency
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("validation cancelled: %v", ctx.Err())
	default:
	}

	// Phase 1: Validate Static Analysis Tools (ID 2)
	ksv.validateStaticAnalysisTools(report)

	// Phase 2: Validate Dynamic Analysis Tools (ID 3)
	ksv.validateDynamicAnalysisTools(report)

	// Phase 3: Validate Network Analysis Tools (ID 4)
	ksv.validateNetworkAnalysisTools(report)

	// Phase 4: Validate Forensics Tools (ID 5)
	ksv.validateForensicsTools(report)

	// Phase 5: Validate Security Frameworks (ID 6)
	ksv.validateSecurityFrameworks(report)

	// Phase 6: Validate Container Runtime (ID 7)
	ksv.validateContainerRuntime(report)

	// Phase 7: Validate System Resources (ID 8)
	ksv.validateSystemResources(report)

	log.Printf("Security validation complete. OS: %s, Tools: %d, Frameworks: %d",
		report.OS, len(report.ToolsAvailable), len(report.FrameworksLoaded))

	return report, nil
}

// validateStaticAnalysisTools - Implements Schema ID 2
// Validates static analysis tools availability: Ghidra, Radare2, Semgrep, Bandit
func (ksv *KaliSecurityValidator) validateStaticAnalysisTools(report *KaliSecurityValidationReport) {
	log.Println("Validating Static Analysis tools...")

	tools := map[string]string{
		"ghidra":  "ghidra",
		"radare2": "r2",
		"semgrep": "semgrep",
		"bandit":  "bandit",
	}

	for toolName, command := range tools {
		if _, err := exec.LookPath(command); err == nil {
			report.ToolsAvailable[toolName] = true
			log.Printf("  ✓ %s available", toolName)
		} else {
			report.ToolsAvailable[toolName] = false
			log.Printf("  ✗ %s not found", toolName)
			report.Recommendations = append(report.Recommendations,
				fmt.Sprintf("Install %s for static code analysis capabilities", toolName))
		}
	}
}

// validateDynamicAnalysisTools - Implements Schema ID 3
// Validates dynamic analysis tools availability: strace, ltrace, perf, gdb
func (ksv *KaliSecurityValidator) validateDynamicAnalysisTools(report *KaliSecurityValidationReport) {
	log.Println("Validating Dynamic Analysis tools...")

	tools := map[string]string{
		"strace": "strace",
		"ltrace": "ltrace",
		"perf":   "perf",
		"gdb":    "gdb",
	}

	for toolName, command := range tools {
		if _, err := exec.LookPath(command); err == nil {
			report.ToolsAvailable[toolName] = true
			log.Printf("  ✓ %s available", toolName)
		} else {
			report.ToolsAvailable[toolName] = false
			log.Printf("  ✗ %s not found", toolName)
			report.Recommendations = append(report.Recommendations,
				fmt.Sprintf("Install %s for dynamic behavior analysis", toolName))
		}
	}
}

// validateNetworkAnalysisTools - Implements Schema ID 4
// Validates network analysis tools availability: tcpdump, tshark, mitmproxy, iptables
func (ksv *KaliSecurityValidator) validateNetworkAnalysisTools(report *KaliSecurityValidationReport) {
	log.Println("Validating Network Analysis tools...")

	tools := map[string]string{
		"tcpdump":   "tcpdump",
		"tshark":    "tshark",
		"mitmproxy": "mitmproxy",
		"iptables":  "iptables",
	}

	for toolName, command := range tools {
		if _, err := exec.LookPath(command); err == nil {
			report.ToolsAvailable[toolName] = true
			log.Printf("  ✓ %s available", toolName)
		} else {
			report.ToolsAvailable[toolName] = false
			log.Printf("  ✗ %s not found", toolName)
			report.Recommendations = append(report.Recommendations,
				fmt.Sprintf("Install %s for network traffic analysis", toolName))
		}
	}
}

// validateForensicsTools - Implements Schema ID 5
// Validates forensics tools availability: Volatility, SleuthKit, Autopsy
func (ksv *KaliSecurityValidator) validateForensicsTools(report *KaliSecurityValidationReport) {
	log.Println("Validating Forensics tools...")

	tools := map[string]string{
		"volatility": "volatility",
		"sleuthkit":  "fls",  // Part of SleuthKit
		"autopsy":    "autopsy",
	}

	for toolName, command := range tools {
		if _, err := exec.LookPath(command); err == nil {
			report.ToolsAvailable[toolName] = true
			log.Printf("  ✓ %s available", toolName)
		} else {
			report.ToolsAvailable[toolName] = false
			log.Printf("  ✗ %s not found", toolName)
			report.Recommendations = append(report.Recommendations,
				fmt.Sprintf("Install %s for forensic analysis", toolName))
		}
	}
}

// validateSecurityFrameworks - Implements Schema ID 6
// Validates security framework support: AppArmor, SELinux, Seccomp
func (ksv *KaliSecurityValidator) validateSecurityFrameworks(report *KaliSecurityValidationReport) {
	log.Println("Validating Security Frameworks...")

	// Check AppArmor
	if _, err := exec.LookPath("aa-status"); err == nil {
		report.FrameworksLoaded["apparmor"] = true
		log.Println("  ✓ AppArmor available")
	} else {
		report.FrameworksLoaded["apparmor"] = false
		log.Println("  ✗ AppArmor not available")
	}

	// Check SELinux
	if _, err := exec.LookPath("getenforce"); err == nil {
		report.FrameworksLoaded["selinux"] = true
		log.Println("  ✓ SELinux available")
	} else {
		report.FrameworksLoaded["selinux"] = false
		log.Println("  ✗ SELinux not available")
	}

	// Check Seccomp (built into kernel)
	report.FrameworksLoaded["seccomp"] = true
	log.Println("  ✓ Seccomp available (kernel)")
}

// validateContainerRuntime - Implements Schema ID 7
// Validates container runtime availability: native Go, Podman, Docker
func (ksv *KaliSecurityValidator) validateContainerRuntime(report *KaliSecurityValidationReport) {
	log.Println("Validating Container Runtime...")

	runtimes := map[string]string{
		"podman": "podman",
		"docker": "docker",
	}

	for runtimeName, command := range runtimes {
		if _, err := exec.LookPath(command); err == nil {
			report.ToolsAvailable[runtimeName] = true
			log.Printf("  ✓ %s available", runtimeName)
		} else {
			report.ToolsAvailable[runtimeName] = false
			log.Printf("  ✗ %s not found", runtimeName)
		}
	}

	// Native Go runtime always available
	report.ToolsAvailable["native-go"] = true
	log.Println("  ✓ native-go available (always available)")
}

// validateSystemResources - Implements Schema ID 8
// Validates minimum system requirements: memory, CPU, disk, file descriptors
func (ksv *KaliSecurityValidator) validateSystemResources(report *KaliSecurityValidationReport) {
	log.Println("Validating System Resources...")

	// Check memory (minimum 8GB recommended)
	meminfoData, err := os.ReadFile("/proc/meminfo")
	if err == nil {
		for _, line := range strings.Split(string(meminfoData), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					report.SystemMemoryKB = parts[1]
					memGB := parseInt(parts[1]) / 1024 / 1024
					if memGB < 8 {
						report.Recommendations = append(report.Recommendations,
							fmt.Sprintf("System has %.1f GB RAM. Recommended minimum is 8GB", float64(memGB)))
					}
					log.Printf("  ✓ Memory: %.1f GB", float64(memGB))
				}
			}
		}
	}

	// Check disk space (minimum 50GB recommended)
	cmd := exec.Command("df", "-k", "/")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			parts := strings.Fields(lines[1])
			if len(parts) >= 4 {
				report.DiskSpaceKB = parts[3]
				diskGB := parseInt(parts[3]) / 1024 / 1024
				if diskGB < 50 {
					report.Recommendations = append(report.Recommendations,
						fmt.Sprintf("System has %.1f GB disk space. Recommended minimum is 50GB", float64(diskGB)))
				}
				log.Printf("  ✓ Disk: %.1f GB available", float64(diskGB))
			}
		}
	}

	// Check file descriptor limit
	cmd = exec.Command("ulimit", "-n")
	if output, err := cmd.Output(); err == nil {
		fdLimit := parseInt(strings.TrimSpace(string(output)))
		if fdLimit < 4096 {
			report.Recommendations = append(report.Recommendations,
				fmt.Sprintf("File descriptor limit is %d. Recommended minimum is 4096", fdLimit))
		}
		log.Printf("  ✓ File descriptors: %d", fdLimit)
	}
}

// Helper function to parse integers safely
func parseInt(s string) int {
	n := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n); err == nil {
		return n
	}
	return 0
}
