package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"backend_server/internal/utils/host"
	"github.com/gorilla/mux"
)

var (
	uptimeRegex = regexp.MustCompile(`(\d+)\s*(week|day|hour|min|sec)`)
)

type SystemHandler struct {
	collector *host.SystemInfoCollector
	metricsMu sync.RWMutex
	lastCPU   float64
	lastMem   float64
}

type SystemMetricsResponse struct {
	CPU      float64        `json:"cpu"`
	Memory   MemoryInfoResp `json:"memory"`
	Uptime   int64          `json:"uptime_seconds"`
	OS       string         `json:"os"`
	Arch     string         `json:"arch"`
	Hostname string         `json:"hostname"`
}

type MemoryInfoResp struct {
	Total      uint64  `json:"total_mb"`
	Used       uint64  `json:"used_mb"`
	Available  uint64  `json:"available_mb"`
	Percentage float64 `json:"percentage"`
}

func NewSystemHandler(collector *host.SystemInfoCollector) *SystemHandler {
	return &SystemHandler{
		collector: collector,
	}
}

func (h *SystemHandler) GetSystemInfo(w http.ResponseWriter, r *http.Request) {
	info, err := h.collector.GetCurrentInfo()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var memPercent, cpuPercent float64
	if info.MemoryInfo != nil {
		h.metricsMu.Lock()
		h.lastMem = info.MemoryInfo.Usage
		memPercent = info.MemoryInfo.Usage
		h.metricsMu.Unlock()
	}
	if info.CPUInfo != nil {
		h.metricsMu.Lock()
		h.lastCPU = info.CPUInfo.Usage
		cpuPercent = info.CPUInfo.Usage
		h.metricsMu.Unlock()
	}

	uptimeSeconds := parseUptimeToSeconds(info.Uptime)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SystemMetricsResponse{
		CPU: cpuPercent,
		Memory: MemoryInfoResp{
			Total:      info.MemoryInfo.Total / (1024 * 1024),
			Used:       info.MemoryInfo.Used / (1024 * 1024),
			Available:  info.MemoryInfo.Available / (1024 * 1024),
			Percentage: memPercent,
		},
		Uptime:   uptimeSeconds,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Hostname: info.Hostname,
	})
}

func (h *SystemHandler) GetDetailedSystemInfo(w http.ResponseWriter, r *http.Request) {
	info, err := h.collector.GetCurrentInfo()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func parseUptimeToSeconds(uptime string) int64 {
	if uptime == "" {
		return 0
	}

	matches := uptimeRegex.FindAllStringSubmatch(uptime, -1)
	if len(matches) == 0 {
		return 0
	}

	var totalSeconds int64
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		value, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			continue
		}

		unit := strings.ToLower(match[2])
		switch unit {
		case "week", "weeks":
			totalSeconds += value * 7 * 24 * 60 * 60
		case "day", "days":
			totalSeconds += value * 24 * 60 * 60
		case "hour", "hours":
			totalSeconds += value * 60 * 60
		case "min", "mins", "minute", "minutes":
			totalSeconds += value * 60
		case "sec", "secs", "second", "seconds":
			totalSeconds += value
		}
	}

	return totalSeconds
}

func (h *SystemHandler) RegisterRoutes(r *mux.Router) {
	systemRouter := r.PathPrefix("/api/v1/system").Subrouter()
	systemRouter.HandleFunc("/info", h.GetSystemInfo).Methods("GET")
	systemRouter.HandleFunc("/detail", h.GetDetailedSystemInfo).Methods("GET")
}
