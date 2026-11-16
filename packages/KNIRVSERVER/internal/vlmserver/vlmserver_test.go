package vlmserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

const cpuinfoFixture = `processor	: 0
model name	: Intel(R) Core(TM) i7 CPU 950 @ 3.07GHz
cpu cores	: 4
flags		: fpu sse sse2 sse4_2 popcnt
processor	: 1
cpu cores	: 4
flags		: fpu sse sse2 sse4_2 popcnt
`

const cpuinfoAVX2Fixture = `processor	: 0
cpu cores	: 8
flags		: fpu sse4_2 avx avx2 fma
`

func TestParseCPUInfo(t *testing.T) {
	cores, avx2 := parseCPUInfo(cpuinfoFixture)
	if cores != 4 || avx2 {
		t.Fatalf("i7-950: cores=%d avx2=%v, want 4 false", cores, avx2)
	}
	cores, avx2 = parseCPUInfo(cpuinfoAVX2Fixture)
	if cores != 8 || !avx2 {
		t.Fatalf("avx2 host: cores=%d avx2=%v, want 8 true", cores, avx2)
	}
}

func TestParseMeminfo(t *testing.T) {
	total, avail := parseMeminfo("MemTotal:       47907000 kB\nMemFree:  1000 kB\nMemAvailable:   18096000 kB\n")
	if total != 47907000*1024 || avail != 18096000*1024 {
		t.Fatalf("total=%d avail=%d", total, avail)
	}
}

func TestParseNvidiaSMI(t *testing.T) {
	name, cc, total, free, ok := parseNvidiaSMI("NVIDIA GeForce RTX 3060, 12288, 11074, 8.6\n")
	if !ok || name != "NVIDIA GeForce RTX 3060" || cc != "8.6" || total != 12288*mib || free != 11074*mib {
		t.Fatalf("ok=%v name=%q cc=%q total=%d free=%d", ok, name, cc, total, free)
	}
	name, cc, _, _, ok = parseNvidiaSMI("NVIDIA GeForce RTX 3060, 12288, 11074\n")
	if !ok || name == "" || cc != "" {
		t.Fatalf("three-field output must parse without a compute capability: ok=%v cc=%q", ok, cc)
	}
	if _, _, _, _, ok := parseNvidiaSMI("[Not Supported]"); ok {
		t.Fatal("unsupported output must not parse")
	}
}

// The planner is checked against the numbers measured on the RTX 3060 (measurements.md):
// 22 expert layers on the CPU with the projector on the GPU used 11,043 MiB.
func TestPlanIdleRTX3060KeepsProjectorOnGPUAndLoadsRAM(t *testing.T) {
	h := Host{CPUCores: 4, HasAVX2: false, RAMAvail: 19 << 30, VRAMFree: 11074 * mib, VRAMTotal: 12288 * mib, GPUName: "RTX 3060", DiskFree: 500 << 30}
	p := PlanFor(h, MiniCPMShape, true)
	if p.Mode != "gpu" || p.NGPULayers != 99 {
		t.Fatalf("mode=%s ngl=%d", p.Mode, p.NGPULayers)
	}
	if !p.ProjectorOnGPU {
		t.Fatalf("idle 3060 should keep the projector on the GPU; warnings=%v", p.Warnings)
	}
	if p.NCPUMoE < 20 || p.NCPUMoE > 24 {
		t.Fatalf("n_cpu_moe=%d, expected in the measured 20-24 band", p.NCPUMoE)
	}
	if p.LoadMode != "none" {
		t.Fatalf("19 GiB RAM free should load into RAM, got %s", p.LoadMode)
	}
	if p.Threads != 4 {
		t.Fatalf("threads=%d, want the 4 physical cores", p.Threads)
	}
}

func TestPlanProjectorMovesToCPUWhenItCannotFitOnGPU(t *testing.T) {
	// Below what the whole model plus the projector needs even with every expert on the
	// CPU, but enough for the model without the projector.
	h := Host{CPUCores: 4, RAMAvail: 19 << 30, VRAMFree: 3900 * mib}
	p := PlanFor(h, MiniCPMShape, true)
	if p.ProjectorOnGPU {
		t.Fatalf("expected a CPU projector here; got n_cpu_moe=%d", p.NCPUMoE)
	}
	if len(p.Warnings) == 0 {
		t.Fatal("a CPU projector must come with a warning that vision will be slow")
	}
}

func TestPlanNoCUDAIsCPUOnly(t *testing.T) {
	h := Host{CPUCores: 4, RAMAvail: 40 << 30, VRAMFree: 11074 * mib, GPUName: "RTX 3060"}
	p := PlanFor(h, MiniCPMShape, false)
	if p.Mode != "cpu" || p.NGPULayers != 0 || p.NCPUMoE != MiniCPMShape.Layers {
		t.Fatalf("no CUDA backend must plan CPU-only: %+v", p)
	}
	if len(p.Warnings) == 0 || !strings.Contains(p.Warnings[0], "no CUDA backend") {
		t.Fatalf("the missing CUDA backend must be reported, warnings=%v", p.Warnings)
	}
}

func TestPlanSmallGPUShiftsExpertsToCPU(t *testing.T) {
	big := PlanFor(Host{CPUCores: 4, RAMAvail: 40 << 30, VRAMFree: 11074 * mib}, MiniCPMShape, true)
	small := PlanFor(Host{CPUCores: 4, RAMAvail: 40 << 30, VRAMFree: 4096 * mib}, MiniCPMShape, true)
	if small.NCPUMoE <= big.NCPUMoE {
		t.Fatalf("less VRAM must keep more experts on the CPU: big=%d small=%d", big.NCPUMoE, small.NCPUMoE)
	}
}

func TestPlanLowRAMUsesMmapAndWarns(t *testing.T) {
	p := PlanFor(Host{CPUCores: 4, RAMAvail: 6 << 30, VRAMFree: 11074 * mib}, MiniCPMShape, true)
	if p.LoadMode != "mmap" {
		t.Fatalf("6 GiB RAM cannot hold the CPU-side experts: load mode %s", p.LoadMode)
	}
	if len(p.Warnings) == 0 {
		t.Fatal("mmap fallback must be reported")
	}
}

func TestPlanArgsMatchMeasuredFlags(t *testing.T) {
	p := Plan{NGPULayers: 99, NCPUMoE: 23, ProjectorOnGPU: false, LoadMode: "none", Threads: 4, Ctx: 2048}
	got := strings.Join(p.Args(), " ")
	for _, want := range []string{"-ngl 99", "--n-cpu-moe 23", "-fa on", "-lm none", "-t 4", "-c 2048", "--parallel 1", "--no-mmproj-offload"} {
		if !strings.Contains(got, want) {
			t.Fatalf("args %q missing %q", got, want)
		}
	}
	p.ProjectorOnGPU = true
	if strings.Contains(strings.Join(p.Args(), " "), "--no-mmproj-offload") {
		t.Fatal("a GPU projector must not disable offload")
	}
}

func TestProvisionVerifiesCachesAndSkips(t *testing.T) {
	requireHTTPS = false
	t.Cleanup(func() { requireHTTPS = true })
	payload := []byte("pretend weights")
	var gets atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		w.Write(payload)
	}))
	defer srv.Close()
	dir := t.TempDir()
	files := []ProvisionFile{{Name: "m.gguf", URL: srv.URL, Size: int64(len(payload)), SHA256: sumHex(payload)}}
	if _, err := Provision(context.Background(), dir, files, srv.Client(), nil); err != nil {
		t.Fatalf("first provision: %v", err)
	}
	if _, err := Provision(context.Background(), dir, files, srv.Client(), nil); err != nil {
		t.Fatalf("second provision: %v", err)
	}
	if gets.Load() != 1 {
		t.Fatalf("a verified file must not be downloaded again: GETs=%d", gets.Load())
	}
	if _, err := os.Stat(filepath.Join(dir, "m.gguf.verified")); err != nil {
		t.Fatalf("verification record missing: %v", err)
	}
}

func TestProvisionResumesAndRejectsMismatch(t *testing.T) {
	requireHTTPS = false
	t.Cleanup(func() { requireHTTPS = true })
	payload := []byte("0123456789abcdefghij")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := 0
		if rg := r.Header.Get("Range"); strings.HasPrefix(rg, "bytes=") {
			start, _ = strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(rg, "bytes="), "-"))
			w.WriteHeader(http.StatusPartialContent)
		}
		w.Write(payload[start:])
	}))
	defer srv.Close()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "r.gguf.part"), payload[:7], 0o600)
	good := []ProvisionFile{{Name: "r.gguf", URL: srv.URL, Size: int64(len(payload)), SHA256: sumHex(payload)}}
	if _, err := Provision(context.Background(), dir, good, srv.Client(), nil); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "r.gguf")); !bytes.Equal(got, payload) {
		t.Fatal("resumed file is corrupt")
	}

	bad := []ProvisionFile{{Name: "x.gguf", URL: srv.URL, Size: int64(len(payload)), SHA256: sumHex([]byte("pinned"))}}
	if _, err := Provision(context.Background(), dir, bad, srv.Client(), nil); err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("want sha256 mismatch, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "x.gguf")); !os.IsNotExist(err) {
		t.Fatal("a mismatched file must not stay at its final path")
	}
}

// The supervisor's default client must not carry an overall deadline: the first run
// downloads an 18.7 GB weight file. A body that trickles in for longer than the old
// 5 s Timeout still completes.
func TestProvisionThroughDefaultClientToleratesSlowBody(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for a slow body")
	}
	requireHTTPS = false
	t.Cleanup(func() { requireHTTPS = true })
	payload := bytes.Repeat([]byte("w"), 80)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		for i := 0; i < len(payload); i += 16 {
			_, _ = w.Write(payload[i : i+16])
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(1600 * time.Millisecond)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	files := []ProvisionFile{{Name: "slow.gguf", URL: srv.URL, Size: int64(len(payload)), SHA256: sumHex(payload)}}
	s := New(Config{DataDir: dir})
	start := time.Now()
	if _, err := Provision(context.Background(), dir, files, s.cfg.HTTP, nil); err != nil {
		t.Fatalf("default client aborted the slow download after %s: %v", time.Since(start), err)
	}
	if elapsed := time.Since(start); elapsed < 5*time.Second {
		t.Fatalf("download took %s; it must outlive the old 5s overall timeout", elapsed)
	}
	got, err := os.ReadFile(filepath.Join(dir, "slow.gguf"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("slow download produced corrupt data")
	}
}

// The idle watchdog must abort a body that stops making progress, and must leave a
// body alone while it keeps moving. bodyIdleTimeout is shortened for the test.
func TestDownloadIdleWatchdogAbortsStalledBody(t *testing.T) {
	requireHTTPS = false
	t.Cleanup(func() { requireHTTPS = true })
	oldIdle := bodyIdleTimeout
	bodyIdleTimeout = 200 * time.Millisecond
	t.Cleanup(func() { bodyIdleTimeout = oldIdle })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte("partial"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done() // stall mid-body until the client gives up
	}))
	defer srv.Close()

	dir := t.TempDir()
	f := ProvisionFile{Name: "stall.gguf", URL: srv.URL, Size: 1000}
	// Test-side deadline: if the watchdog is missing, the body read would block forever.
	// The context cancels the in-flight request so the test fails instead of hanging.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	err := download(ctx, srv.Client(), f, filepath.Join(dir, "stall.gguf"), func(string, ...any) {})
	elapsed := time.Since(start)
	if ctx.Err() != nil {
		t.Fatalf("stalled body was still downloading at the 5s test deadline; the idle watchdog did not fire")
	}
	if err == nil {
		t.Fatal("a stalled body must fail the download")
	}
	if elapsed > 5*time.Second {
		t.Fatalf("stalled body was only aborted after %s; the idle watchdog should fire near 200ms", elapsed)
	}
}

func TestDownloadIdleWatchdogKeepsActiveSlowBody(t *testing.T) {
	requireHTTPS = false
	t.Cleanup(func() { requireHTTPS = true })
	oldIdle := bodyIdleTimeout
	bodyIdleTimeout = 300 * time.Millisecond
	t.Cleanup(func() { bodyIdleTimeout = oldIdle })

	payload := bytes.Repeat([]byte("a"), 50)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		for i := 0; i < len(payload); i += 10 {
			_, _ = w.Write(payload[i : i+10])
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(120 * time.Millisecond) // each gap is under the idle limit
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "slow.gguf")
	f := ProvisionFile{Name: "slow.gguf", URL: srv.URL, Size: int64(len(payload))}
	start := time.Now()
	if err := download(context.Background(), srv.Client(), f, path, func(string, ...any) {}); err != nil {
		t.Fatalf("an active body must not be aborted by the idle watchdog: %v", err)
	}
	if elapsed := time.Since(start); elapsed <= 300*time.Millisecond {
		t.Fatalf("download finished in %s; it must outlast the idle limit to prove activity resets it", elapsed)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("slow body corrupt or missing: %v", err)
	}
}

func TestPinnedFilesAreWellFormed(t *testing.T) {
	for _, f := range MiniCPMFiles {
		if len(f.SHA256) != 64 || f.Size <= 0 || !strings.HasPrefix(f.URL, "https://huggingface.co/openbmb/MiniCPM-V-4.7-35B-A3B-gguf/") {
			t.Fatalf("malformed pin: %+v", f)
		}
	}
}

func TestEnvDoesNotOverrideOperatorChoice(t *testing.T) {
	t.Setenv("KNIRV_VLM_BACKEND", "none")
	s := New(Config{Port: 9999})
	env := strings.Join(s.Env(), "\n")
	if strings.Contains(env, "KNIRV_VLM_BACKEND=") {
		t.Fatal("an explicit KNIRV_VLM_BACKEND must win")
	}
	if !strings.Contains(env, "KNIRV_VLM_MINICPM_ENDPOINT=http://127.0.0.1:9999") {
		t.Fatalf("endpoint not exported: %s", env)
	}
}

func TestResolveBinaryErrorsWithAHint(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("KNIRV_VLM_LLAMA_SERVER", "")
	if _, err := ResolveBinary("", t.TempDir()); err == nil || !strings.Contains(err.Error(), "KNIRV_VLM_LLAMA_SERVER") {
		t.Fatalf("want a hint naming the env var, got %v", err)
	}
}

// TestHelperProcess is not a test: the supervisor test re-executes the test binary as a
// stand-in llama-server. It serves /health on --port until killed.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("VLM_HELPER_SERVER") != "1" {
		return
	}
	port := ""
	for i, a := range os.Args {
		if a == "--port" && i+1 < len(os.Args) {
			port = os.Args[i+1]
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })
	http.Serve(ln, mux)
	os.Exit(0)
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// The supervisor runs the planned command, reports ready, and restarts a server that
// dies. The stand-in server is this test binary re-executed as TestHelperProcess.
func TestSupervisorStartsServesAndRestartsAfterCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns a process")
	}
	requireHTTPS = false
	t.Cleanup(func() { requireHTTPS = true })

	weights := []byte("weights")
	proj := []byte("projector")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "lang.gguf") {
			w.Write(weights)
		} else {
			w.Write(proj)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	// A stand-in "llama-server": the test binary, asked to act as the helper.
	bin := filepath.Join(dir, "llama-server")
	script := fmt.Sprintf("#!/bin/sh\nVLM_HELPER_SERVER=1 exec %q -test.run=TestHelperProcess -- \"$@\"\n", os.Args[0])
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	port := freePort(t)
	files := []ProvisionFile{
		{Name: "lang.gguf", URL: srv.URL + "/lang.gguf", Size: int64(len(weights)), SHA256: sumHex(weights)},
		{Name: "proj.gguf", URL: srv.URL + "/proj.gguf", Size: int64(len(proj)), SHA256: sumHex(proj)},
	}
	s := New(Config{
		DataDir:        dir,
		Files:          files,
		Binary:         bin,
		Port:           port,
		Probe:          func(string) Host { return Host{CPUCores: 4, RAMAvail: 40 << 30} },
		RestartBackoff: []time.Duration{50 * time.Millisecond},
		StableAfter:    time.Hour,
		StartupTimeout: 30 * time.Second,
		StopTimeout:    5 * time.Second,
		HTTP:           srv.Client(),
		Logf:           func(f string, a ...any) { t.Logf("[sup] "+f, a...) },
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })

	waitFor(t, "ready", func() bool { return s.Status()["state"] == "ready" })
	plan, _ := s.Status()["plan"].(map[string]any)
	if plan == nil || plan["mode"] != "cpu" {
		t.Fatalf("with no CUDA backend the plan must be CPU: %v", plan)
	}

	// Kill the server; the supervisor must bring it back.
	s.mu.Lock()
	pid := s.cmd.Process.Pid
	s.mu.Unlock()
	if err := killGroup(pid); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "restart after crash", func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.cmd != nil && s.cmd.Process.Pid != pid
	})
	waitFor(t, "ready again", func() bool { return s.healthy() })
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func sumHex(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// killGroup SIGKILLs the process group led by pid. pid must be positive: a group of 0
// would signal this test process too.
func killGroup(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("refusing to signal process group %d", pid)
	}
	return syscall.Kill(-pid, syscall.SIGKILL)
}
