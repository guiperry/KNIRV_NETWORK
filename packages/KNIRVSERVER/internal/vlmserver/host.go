package vlmserver

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Host is what the planner needs to know about this machine at the moment it plans.
// Every figure is read at runtime; nothing is hard-coded per host.
type Host struct {
	CPUCores int  // physical cores (the "cpu cores" field of /proc/cpuinfo)
	HasAVX2  bool // CPU advertises AVX2
	RAMTotal uint64
	RAMAvail uint64 // MemAvailable: what a new process can take without swapping
	GPUName  string // empty when no NVIDIA GPU answers nvidia-smi
	// GPUComputeCap is the SM version, e.g. "8.6" for Ampere sm_86. Empty when the driver
	// does not report it.
	GPUComputeCap string
	VRAMTotal     uint64
	VRAMFree      uint64
	DiskFree      uint64 // free bytes on the filesystem holding the data directory
}

// ProbeHost reads the host. A missing source leaves its fields zero, which makes the
// planner choose the conservative path for that resource.
func ProbeHost(dataDir string) Host {
	var h Host
	if b, err := readFile("/proc/cpuinfo"); err == nil {
		h.CPUCores, h.HasAVX2 = parseCPUInfo(b)
	}
	if b, err := readFile("/proc/meminfo"); err == nil {
		h.RAMTotal, h.RAMAvail = parseMeminfo(b)
	}
	h.GPUName, h.GPUComputeCap, h.VRAMTotal, h.VRAMFree = probeNvidia()
	h.DiskFree = diskFree(dataDir)
	return h
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

// parseCPUInfo returns the physical core count and whether AVX2 is present. Core count
// is taken from the first "cpu cores" line, which is per socket and is what llama.cpp
// should use for threads (hyperthread siblings do not help its matrix kernels).
func parseCPUInfo(s string) (cores int, avx2 bool) {
	sc := bufio.NewScanner(strings.NewReader(s))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "cpu cores":
			if cores == 0 {
				if n, err := strconv.Atoi(val); err == nil {
					cores = n
				}
			}
		case "flags":
			if !avx2 {
				for _, f := range strings.Fields(val) {
					if f == "avx2" {
						avx2 = true
						break
					}
				}
			}
		}
	}
	return cores, avx2
}

// parseMeminfo returns MemTotal and MemAvailable in bytes (the file reports kB).
func parseMeminfo(s string) (total, avail uint64) {
	for _, line := range strings.Split(s, "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(val)
		if len(fields) == 0 {
			continue
		}
		n, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		switch strings.TrimSpace(key) {
		case "MemTotal":
			total = n * 1024
		case "MemAvailable":
			avail = n * 1024
		}
	}
	return total, avail
}

// parseNvidiaSMI reads one line of
// `nvidia-smi --query-gpu=name,memory.total,memory.free,compute_cap --format=csv,noheader,nounits`
// (memory in MiB). The compute_cap column is optional: older drivers omit it. ok is false
// for anything else.
func parseNvidiaSMI(s string) (name, computeCap string, total, free uint64, ok bool) {
	line := strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
	parts := strings.Split(line, ",")
	if len(parts) != 3 && len(parts) != 4 {
		return "", "", 0, 0, false
	}
	t, err1 := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
	f, err2 := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
	if err1 != nil || err2 != nil {
		return "", "", 0, 0, false
	}
	if len(parts) == 4 {
		computeCap = strings.TrimSpace(parts[3])
		if _, err := strconv.ParseFloat(computeCap, 64); err != nil {
			computeCap = ""
		}
	}
	return strings.TrimSpace(parts[0]), computeCap, t * mib, f * mib, true
}

const mib = 1 << 20

// probeNvidia asks nvidia-smi for the first GPU. It has a short timeout so a wedged
// driver cannot hold up startup.
func probeNvidia() (name, computeCap string, total, free uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=name,memory.total,memory.free,compute_cap", "--format=csv,noheader,nounits").Output()
	if err != nil {
		// Retry without compute_cap for drivers that do not know the field.
		out, err = exec.CommandContext(ctx, "nvidia-smi",
			"--query-gpu=name,memory.total,memory.free", "--format=csv,noheader,nounits").Output()
		if err != nil {
			return "", "", 0, 0
		}
	}
	name, computeCap, total, free, ok := parseNvidiaSMI(string(out))
	if !ok {
		return "", "", 0, 0
	}
	return name, computeCap, total, free
}

// diskFree returns the free bytes on the filesystem that holds path, or 0 if unknown.
func diskFree(path string) uint64 {
	var st syscall.Statfs_t
	dir := path
	for dir != "" && dir != "/" {
		if err := syscall.Statfs(dir, &st); err == nil {
			return st.Bavail * uint64(st.Bsize)
		}
		dir = filepath.Dir(dir)
	}
	if err := syscall.Statfs("/", &st); err == nil {
		return st.Bavail * uint64(st.Bsize)
	}
	return 0
}
