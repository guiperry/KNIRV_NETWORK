package teesecurity

// ContainerConfig holds all configuration for hardened container execution
type ContainerConfig struct {
	// Namespace configuration
	Namespaces NamespaceConfig

	// Cgroup configuration
	Cgroups CgroupConfig

	// Network configuration (Phase 2)
	Network NetworkConfig

	// Security configuration (Phase 3)
	Security SecurityConfig

	// Enable hardening features
	EnableHardening bool

	// Fallback to monitoring if hardening fails
	FallbackToMonitoring bool
}

// NamespaceConfig defines which namespaces to enable
type NamespaceConfig struct {
	EnablePID     bool
	EnableNetwork bool
	EnableMount   bool
	EnableUTS     bool
	EnableIPC     bool
	EnableUser    bool
}

// CgroupConfig defines resource limits
type CgroupConfig struct {
	// CPU limits
	CPU CgroupCPU

	// Memory limits
	Memory CgroupMemory

	// I/O limits
	IO CgroupIO

	// PID limits
	PIDs CgroupPIDs
}

// CgroupCPU defines CPU resource limits
type CgroupCPU struct {
	Quota  int64 // CPU quota (microseconds per period)
	Period int64 // CPU period (microseconds)
	Shares int64 // CPU shares (relative weight)
}

// CgroupMemory defines memory resource limits
type CgroupMemory struct {
	Limit          int64 // Maximum memory (bytes)
	SwapLimit      int64 // Swap limit (bytes, -1 = disabled)
	OOMKillDisable bool
}

// CgroupIO defines I/O resource limits
type CgroupIO struct {
	ReadBPS  int64 // Read bandwidth (bytes/sec)
	WriteBPS int64 // Write bandwidth (bytes/sec)
}

// CgroupPIDs defines process limits
type CgroupPIDs struct {
	Max int64 // Maximum number of PIDs
}

// NetworkConfig defines network namespace configuration
type NetworkConfig struct {
	ContainerInterface string
	ContainerIP        string // CIDR: "10.200.0.2/24"
	GatewayIP          string // "10.200.0.1"
	EnableInternet     bool
	DNSServers         []string
}

// SecurityConfig defines security hardening options
type SecurityConfig struct {
	DropCapabilities bool
	SeccompEnabled   bool
	AppArmorEnabled  bool
	ReadOnlyRoot     bool
	NoNewPrivs       bool
}

// Namespace represents a created namespace
type Namespace struct {
	Type string // "pid", "net", "mnt", "uts", "ipc", "user"
	FD   int    // File descriptor to namespace
}

// CgroupStats holds resource usage statistics
type CgroupStats struct {
	CPUStat     string
	MemoryUsage int64
	MemoryPeak  int64
	PIDsCurrent int
}

// ResourceUsage holds container resource usage information
type ResourceUsage struct {
	MemoryPeak  int64
	MemoryUsage int64
	PIDsUsed    int
}

// VethPair represents a virtual ethernet pair
type VethPair struct {
	HostInterface      string
	ContainerInterface string
}

// GetDefaultContainerConfig returns a secure default configuration
type DefaultContainerConfig func() ContainerConfig

// BindMount represents a bind mount configuration
type BindMount struct {
	Source      string
	Destination string
	ReadOnly    bool
}

// MountConfig defines mount namespace configuration
type MountConfig struct {
	RootFS       string
	MountProc    bool
	MountSys     bool
	MountDev     bool
	ReadOnlyRoot bool
	BindMounts   []BindMount
}

// CapabilityConfig defines capability management
type CapabilityConfig struct {
	Keep     []string
	Drop     []string
	Bounding []string
}

// SeccompConfig defines seccomp filter configuration
type SeccompConfig struct {
	DefaultAction     string
	AllowedSyscalls   []string
	BlockedSyscalls   []string
	UseDefaultProfile bool
}

// MACConfig defines MAC (AppArmor/SELinux) configuration
type MACConfig struct {
	Type         string
	Profile      string
	AutoGenerate bool
	Template     string
}

// DefaultAllowedSyscalls is the default whitelist of allowed syscalls
var DefaultAllowedSyscalls = []string{
	// File operations
	"read", "write", "open", "openat", "close", "stat", "fstat",
	"lstat", "lseek", "access", "getcwd", "readlink",

	// Process management
	"exit", "exit_group", "wait4", "clone", "fork", "vfork",
	"execve", "getpid", "getppid",

	// Memory management
	"mmap", "munmap", "mprotect", "brk", "madvise",

	// Time
	"gettimeofday", "clock_gettime", "nanosleep",

	// Networking (if enabled)
	"socket", "connect", "bind", "listen", "accept", "recv", "send",

	// Signal handling
	"rt_sigaction", "rt_sigprocmask", "rt_sigreturn",
}

// BlockedSyscalls are syscalls that should always be blocked
var BlockedSyscalls = []string{
	"keyctl",            // Kernel keyring
	"add_key",           // Kernel keyring
	"request_key",       // Kernel keyring
	"ptrace",            // Debug other processes
	"process_vm_readv",  // Read other process memory
	"process_vm_writev", // Write other process memory
	"kexec_load",        // Load new kernel
	"reboot",            // Reboot system
	"swapon",            // Swap management
	"swapoff",           // Swap management
	"mount",             // Mount filesystems
	"umount",            // Unmount
	"pivot_root",        // Change root
	"unshare",           // Create new namespaces
	"setns",             // Join existing namespace
}

// DefaultAppArmorProfile is the default AppArmor profile template
const DefaultAppArmorProfile = `
#include <tunables/global>

profile {{PROFILE_NAME}} flags=(attach_disconnected,mediate_deleted) {
  #include <abstractions/base>

  # Deny dangerous capabilities
  deny capability sys_admin,
  deny capability sys_module,
  deny capability sys_rawio,
  deny capability sys_ptrace,
  deny capability sys_boot,
  deny capability mac_admin,
  deny capability mac_override,

  # Allow skill execution
  /skill/** rwix,
  /tmp/skill-*/** rwix,

  # Read-only system files
  /etc/** r,
  /lib/** rm,
  /usr/lib/** rm,

  # Deny everything else
  deny /** w,
  deny /proc/sys/** w,
  deny /sys/** w,
}
`

// SecureCapabilitySet defines the minimal capabilities to keep
var SecureCapabilitySet = []string{
	"CAP_SETUID",
	"CAP_SETGID",
}

// DangerousCapabilities are capabilities that should always be dropped
var DangerousCapabilities = []string{
	"CAP_SYS_ADMIN",
	"CAP_SYS_MODULE",
	"CAP_SYS_RAWIO",
	"CAP_SYS_PTRACE",
	"CAP_SYS_BOOT",
	"CAP_NET_ADMIN",
	"CAP_MAC_ADMIN",
}
