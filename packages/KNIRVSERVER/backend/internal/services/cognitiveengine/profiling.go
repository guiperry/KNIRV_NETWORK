package cognitiveengine

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"
)

type ProfileType string

const (
	ProfileCPU       ProfileType = "cpu"
	ProfileMemory    ProfileType = "memory"
	ProfileGoroutine ProfileType = "goroutine"
	ProfileBlock     ProfileType = "block"
	ProfileMutex     ProfileType = "mutex"
	ProfileThread    ProfileType = "thread"
	ProfileTrace     ProfileType = "trace"
)

type Profiler struct {
	server       *http.Server
	enabled      bool
	mu           sync.RWMutex
	profileDir   string
	cpuProfile   *os.File
	memProfile   *os.File
	traceFile    *os.File
	isCapturing  bool
	startTime    time.Time
	captureCount int
	maxCaptures  int
	cleanupAge   time.Duration
	handlers     map[ProfileType]http.HandlerFunc
	ctx          context.Context
	cancel       context.CancelFunc
}

type ProfilerConfig struct {
	Enabled      bool
	ListenAddr   string
	ProfileDir   string
	MaxCaptures  int
	CleanupAge   time.Duration
	EnableCPU    bool
	EnableMemory bool
	EnableTrace  bool
}

type ProfileSnapshot struct {
	Type       ProfileType
	Timestamp  time.Time
	FilePath   string
	SizeBytes  int64
	Goroutines int
	HeapMB     int64
	StackMB    int64
}

type ProfileReport struct {
	Snapshots        []ProfileSnapshot
	AnalysisResults  map[string]interface{}
	GoroutineDelta   int
	HeapDeltaMB      int64
	CPUPercent       float64
	AllocationRateMB float64
}

func DefaultProfilerConfig() *ProfilerConfig {
	return &ProfilerConfig{
		Enabled:      false,
		ListenAddr:   "localhost:6060",
		ProfileDir:   "/tmp/cognitive_profiles",
		MaxCaptures:  100,
		CleanupAge:   24 * time.Hour,
		EnableCPU:    true,
		EnableMemory: true,
		EnableTrace:  false,
	}
}

func NewProfiler(cfg *ProfilerConfig) (*Profiler, error) {
	if cfg == nil {
		cfg = DefaultProfilerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := os.MkdirAll(cfg.ProfileDir, 0755); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create profile directory: %w", err)
	}

	p := &Profiler{
		enabled:     cfg.Enabled,
		profileDir:  cfg.ProfileDir,
		maxCaptures: cfg.MaxCaptures,
		cleanupAge:  cfg.CleanupAge,
		handlers:    make(map[ProfileType]http.HandlerFunc),
		ctx:         ctx,
		cancel:      cancel,
	}

	p.setupHandlers()
	return p, nil
}

func (p *Profiler) setupHandlers() {
}

func (p *Profiler) Start(addr string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.enabled {
		return fmt.Errorf("profiler already enabled")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprofServe)
	mux.HandleFunc("/profile/start", p.startCPUProfile)
	mux.HandleFunc("/profile/stop", p.stopCPUProfile)
	mux.HandleFunc("/profile/capture", p.captureAllProfiles)
	mux.HandleFunc("/health", p.healthHandler)

	p.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("Profiler: starting HTTP server on %s", addr)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Profiler: HTTP server error: %v", err)
		}
	}()

	p.enabled = true
	p.startTime = time.Now()
	return nil
}

func pprofServe(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/debug/pprof/cmdline":
		pprofCmdlineHandler(w, r)
	case "/debug/pprof/profile":
		pprofProfileHandler(w, r)
	case "/debug/pprof/symbol":
		pprofSymbolHandler(w, r)
	case "/debug/pprof/trace":
		pprofTraceHandler(w, r)
	default:
		pprofIndexHandler(w, r)
	}
}

func pprofCmdlineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Log the request for debugging
	log.Printf("[pprof] cmdline request from %s", r.RemoteAddr)

	profile := pprof.Lookup("cmdline")
	if profile != nil {
		profile.WriteTo(w, 0)
	} else {
		http.Error(w, "cmdline profile not available", http.StatusNotFound)
	}
}

func pprofProfileHandler(w http.ResponseWriter, r *http.Request) {
	duration := 30 * time.Second
	if durStr := r.URL.Query().Get("seconds"); durStr != "" {
		if dur, err := time.ParseDuration(durStr + "s"); err == nil && dur > 0 {
			duration = dur
		}
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	profile := pprof.Lookup("profile")
	if profile != nil {
		profile.WriteTo(w, 0)
	} else {
		// Fallback to CPU profiling
		if err := pprof.StartCPUProfile(w); err != nil {
			http.Error(w, "Could not start CPU profile: "+err.Error(), http.StatusInternalServerError)
			return
		}
		time.Sleep(duration)
		pprof.StopCPUProfile()
	}
}

func pprofSymbolHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if r.Method == "POST" {
		// Handle symbol lookup
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		symbols := r.Form["symbol"]
		for _, sym := range symbols {
			if fn := runtime.FuncForPC(uintptr(0)); fn != nil {
				// Simplified symbol lookup
				fmt.Fprintf(w, "%s\n", sym)
			}
		}
	} else {
		// GET request - show usage
		fmt.Fprintf(w, "Symbol lookup endpoint. POST with symbol parameter.\n")
		fmt.Fprintf(w, "Example: curl -d 'symbol=main.main' http://localhost:6060/debug/pprof/symbol\n")
	}
}

func pprofTraceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")

	duration := 1 * time.Second
	if durStr := r.URL.Query().Get("seconds"); durStr != "" {
		if dur, err := time.ParseDuration(durStr + "s"); err == nil && dur > 0 {
			duration = dur
		}
	}

	if err := trace.Start(w); err != nil {
		http.Error(w, "Could not start trace: "+err.Error(), http.StatusInternalServerError)
		return
	}
	time.Sleep(duration)
	trace.Stop()
}

func pprofIndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><head><title>pprof index</title></head><body>")
	fmt.Fprintf(w, "<h1>pprof endpoints</h1>")
	fmt.Fprintf(w, "<p>Request path: %s</p>", r.URL.Path)
	fmt.Fprintf(w, "<ul>")
	fmt.Fprintf(w, "<li><a href='cmdline'>cmdline</a> - show command line</li>")
	fmt.Fprintf(w, "<li><a href='profile?seconds=30'>profile</a> - CPU profile (30 seconds)</li>")
	fmt.Fprintf(w, "<li><a href='symbol'>symbol</a> - symbol lookup</li>")
	fmt.Fprintf(w, "<li><a href='trace?seconds=1'>trace</a> - execution trace (1 second)</li>")
	fmt.Fprintf(w, "<li><a href='goroutine'>goroutine</a> - goroutine profile</li>")
	fmt.Fprintf(w, "<li><a href='heap'>heap</a> - heap profile</li>")
	fmt.Fprintf(w, "<li><a href='threadcreate'>threadcreate</a> - thread creation profile</li>")
	fmt.Fprintf(w, "<li><a href='block'>block</a> - blocking profile</li>")
	fmt.Fprintf(w, "<li><a href='mutex'>mutex</a> - mutex profile</li>")
	fmt.Fprintf(w, "</ul>")
	fmt.Fprintf(w, "<p>Use query parameter 'seconds' to adjust duration for profile and trace endpoints.</p>")
	fmt.Fprintf(w, "</body></html>")
}

func (p *Profiler) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.enabled {
		return nil
	}

	p.cancel()

	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := p.server.Shutdown(ctx); err != nil {
			log.Printf("Profiler: shutdown error: %v", err)
		}
	}

	p.cleanupOldProfiles()

	p.enabled = false
	log.Println("Profiler: stopped")
	return nil
}

func (p *Profiler) startCPUProfile(w http.ResponseWriter, r *http.Request) {
	if p.cpuProfile != nil {
		http.Error(w, "CPU profiling already started", http.StatusBadRequest)
		return
	}

	profilePath := filepath.Join(p.profileDir, fmt.Sprintf("cpu_%s.prof", time.Now().Format("20060102_150405")))
	f, err := os.Create(profilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create profile file: %v", err), http.StatusInternalServerError)
		return
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		http.Error(w, fmt.Sprintf("Failed to start CPU profile: %v", err), http.StatusInternalServerError)
		return
	}

	p.mu.Lock()
	p.cpuProfile = f
	p.mu.Unlock()

	log.Printf("Profiler: CPU profiling started, writing to %s", profilePath)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("CPU profiling started"))
}

func (p *Profiler) stopCPUProfile(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cpuProfile == nil {
		http.Error(w, "CPU profiling not started", http.StatusBadRequest)
		return
	}

	pprof.StopCPUProfile()

	fileName := p.cpuProfile.Name()
	p.cpuProfile.Close()
	p.cpuProfile = nil

	log.Printf("Profiler: CPU profiling stopped, saved to %s", fileName)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("CPU profiling stopped, saved to %s", fileName)))
}

func (p *Profiler) captureAllProfiles(w http.ResponseWriter, r *http.Request) {
	snapshot, err := p.CaptureProfiles()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to capture profiles: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Profiler: captured %d profiles", len(snapshot.Snapshots))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Captured %d profiles", len(snapshot.Snapshots))))
}

func (p *Profiler) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (p *Profiler) CaptureProfiles() (*ProfileReport, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.captureCount >= p.maxCaptures {
		return nil, fmt.Errorf("max captures reached (%d)", p.maxCaptures)
	}

	report := &ProfileReport{
		Snapshots:       make([]ProfileSnapshot, 0),
		AnalysisResults: make(map[string]interface{}),
	}

	timestamp := time.Now()
	prefix := timestamp.Format("20060102_150405")

	var mBefore, mAfter runtime.MemStats
	runtime.ReadMemStats(&mBefore)

	goroutinesBefore := runtime.NumGoroutine()

	cpuProfilePath := filepath.Join(p.profileDir, fmt.Sprintf("cpu_%s.prof", prefix))
	if err := p.captureCPUProfileLocked(cpuProfilePath); err != nil {
		log.Printf("Profiler: failed to capture CPU profile: %v", err)
	}

	memProfilePath := filepath.Join(p.profileDir, fmt.Sprintf("heap_%s.prof", prefix))
	if err := p.captureHeapProfileLocked(memProfilePath); err != nil {
		log.Printf("Profiler: failed to capture heap profile: %v", err)
	}

	goroutineProfilePath := filepath.Join(p.profileDir, fmt.Sprintf("goroutine_%s.prof", prefix))
	if err := p.captureGoroutineProfileLocked(goroutineProfilePath); err != nil {
		log.Printf("Profiler: failed to capture goroutine profile: %v", err)
	}

	runtime.ReadMemStats(&mAfter)
	goroutinesAfter := runtime.NumGoroutine()

	heapMB := int64(mAfter.HeapAlloc) / (1024 * 1024)
	stackMB := int64(mAfter.StackInuse) / (1024 * 1024)

	report.Snapshots = append(report.Snapshots,
		ProfileSnapshot{
			Type:       ProfileCPU,
			Timestamp:  timestamp,
			FilePath:   cpuProfilePath,
			Goroutines: goroutinesAfter,
			HeapMB:     heapMB,
			StackMB:    stackMB,
		},
		ProfileSnapshot{
			Type:       ProfileMemory,
			Timestamp:  timestamp,
			FilePath:   memProfilePath,
			Goroutines: goroutinesAfter,
			HeapMB:     heapMB,
			StackMB:    stackMB,
		},
		ProfileSnapshot{
			Type:       ProfileGoroutine,
			Timestamp:  timestamp,
			FilePath:   goroutineProfilePath,
			Goroutines: goroutinesAfter,
			HeapMB:     heapMB,
			StackMB:    stackMB,
		},
	)

	report.GoroutineDelta = goroutinesAfter - goroutinesBefore
	report.HeapDeltaMB = int64(mAfter.HeapAlloc-mBefore.HeapAlloc) / (1024 * 1024)
	report.AllocationRateMB = float64(report.HeapDeltaMB)

	for i, snap := range report.Snapshots {
		if info, err := os.Stat(snap.FilePath); err == nil {
			report.Snapshots[i].SizeBytes = info.Size()
		}
	}

	p.captureCount++
	p.cleanupOldProfilesLocked()

	return report, nil
}

func (p *Profiler) captureCPUProfileLocked(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	err = pprof.StartCPUProfile(f)
	if err != nil {
		return err
	}

	time.Sleep(30 * time.Second)
	pprof.StopCPUProfile()

	return nil
}

func (p *Profiler) captureHeapProfileLocked(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	runtime.GC()
	err = pprof.WriteHeapProfile(f)
	f.Close()

	return err
}

func (p *Profiler) captureGoroutineProfileLocked(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return pprof.Lookup("goroutine").WriteTo(f, 0)
}

func (p *Profiler) captureBlockProfileLocked(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	runtime.SetBlockProfileRate(1)
	return pprof.Lookup("block").WriteTo(f, 0)
}

func (p *Profiler) captureMutexProfileLocked(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	runtime.SetMutexProfileFraction(1)
	return pprof.Lookup("mutex").WriteTo(f, 0)
}

func (p *Profiler) StartTrace(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.traceFile != nil {
		return fmt.Errorf("trace already in progress")
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	if err := trace.Start(f); err != nil {
		f.Close()
		return err
	}

	p.traceFile = f
	log.Printf("Profiler: trace started, writing to %s", path)
	return nil
}

func (p *Profiler) StopTrace() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.traceFile == nil {
		return "", fmt.Errorf("no trace in progress")
	}

	trace.Stop()

	fileName := p.traceFile.Name()
	p.traceFile.Close()
	p.traceFile = nil

	log.Printf("Profiler: trace stopped, saved to %s", fileName)
	return fileName, nil
}

func (p *Profiler) GetSystemStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var gcStats debug.GCStats
	debug.ReadGCStats(&gcStats)

	return map[string]interface{}{
		"goroutines":       runtime.NumGoroutine(),
		"gomaxprocs":       runtime.GOMAXPROCS(0),
		"num_cpu":          runtime.NumCPU(),
		"heap_alloc_mb":    m.HeapAlloc / (1024 * 1024),
		"heap_sys_mb":      m.HeapSys / (1024 * 1024),
		"heap_idle_mb":     m.HeapIdle / (1024 * 1024),
		"heap_inuse_mb":    m.HeapInuse / (1024 * 1024),
		"heap_released_mb": m.HeapReleased / (1024 * 1024),
		"heap_objects":     m.HeapObjects,
		"stack_inuse_mb":   m.StackInuse / (1024 * 1024),
		"stack_sys_mb":     m.StackSys / (1024 * 1024),
		"mspan_inuse_mb":   m.MSpanInuse / (1024 * 1024),
		"mspan_sys_mb":     m.MSpanSys / (1024 * 1024),
		"mcache_inuse_mb":  m.MCacheInuse / (1024 * 1024),
		"mcache_sys_mb":    m.MCacheSys / (1024 * 1024),
		"buck_hash_sys_mb": m.BuckHashSys / (1024 * 1024),
		"gc_count":         gcStats.NumGC,
		"last_gc_time":     gcStats.LastGC,
		"next_gc":          m.NextGC,
		"pause_total_ns":   gcStats.PauseTotal,
	}
}

func (p *Profiler) cleanupOldProfiles() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cleanupOldProfilesLocked()
}

func (p *Profiler) cleanupOldProfilesLocked() {
	entries, err := os.ReadDir(p.profileDir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-p.cleanupAge)
	count := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(p.profileDir, entry.Name())); err == nil {
				count++
			}
		}
	}

	if count > 0 {
		log.Printf("Profiler: cleaned up %d old profile files", count)
	}
}

func (p *Profiler) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

func (p *Profiler) GetCaptureCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.captureCount
}

func (p *Profiler) GetUptime() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.startTime.IsZero() {
		return 0
	}
	return time.Since(p.startTime)
}
