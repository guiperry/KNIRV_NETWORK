package runtime

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// memAvailable is swapped by tests.
var memAvailable = readMemAvailable

func readMemAvailable() (int64, error) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	for _, l := range strings.Split(string(b), "\n") {
		if f := strings.Fields(l); len(f) >= 2 && f[0] == "MemAvailable:" {
			kb, err := strconv.ParseInt(f[1], 10, 64)
			return kb * 1024, err
		}
	}
	return 0, fmt.Errorf("MemAvailable not found")
}

// PreflightReport lists what is wrong before llama-server is spawned. Errors make a
// start pointless; Warnings explain a likely slow or degraded one.
type PreflightReport struct {
	Errors, Warnings []string
	ModelBytes       int64
	MemAvailBytes    int64
}

// OK reports whether starting is worth attempting.
func (r PreflightReport) OK() bool { return len(r.Errors) == 0 }

// Preflight checks the things that otherwise surface as a silent two-minute timeout:
// an unreadable or truncated model, a missing server binary, memory too small to hold
// the model, and the llama port already taken by something that is not llama-server.
func Preflight(serverPath, model, address string) PreflightReport {
	var r PreflightReport
	if st, err := os.Stat(serverPath); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("llama-server binary %q: %v", serverPath, err))
	} else if st.Mode()&0o111 == 0 {
		r.Errors = append(r.Errors, fmt.Sprintf("llama-server binary %q is not executable", serverPath))
	}

	f, err := os.Open(model)
	if err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("model %q is not readable: %v", model, err))
	} else {
		defer f.Close()
		st, _ := f.Stat()
		r.ModelBytes = st.Size()
		magic := make([]byte, 4)
		if _, err := io.ReadFull(f, magic); err != nil || string(magic) != "GGUF" {
			r.Errors = append(r.Errors, fmt.Sprintf("model %q is not a GGUF file (truncated download?)", model))
		}
	}

	if avail, err := memAvailable(); err == nil {
		r.MemAvailBytes = avail
		if r.ModelBytes > 0 && avail < r.ModelBytes {
			r.Warnings = append(r.Warnings, fmt.Sprintf("only %d MB RAM available for a %d MB model: loading will page and be slow",
				avail>>20, r.ModelBytes>>20))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if code, err := HealthState(ctx, address); err != nil || code == 0 {
		if c, derr := net.DialTimeout("tcp", address, time.Second); derr == nil {
			c.Close()
			r.Errors = append(r.Errors, fmt.Sprintf("%s is already in use by something that is not a healthy llama-server", address))
		}
	}
	return r
}
