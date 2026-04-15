package api

import (
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"backend_server/internal/utils/host"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

type StreamMetricsMessage struct {
	Type      string         `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	CPU       float64        `json:"cpu"`
	Memory    MemoryInfoResp `json:"memory"`
	Uptime    int64          `json:"uptime_seconds"`
	OS        string         `json:"os"`
	Arch      string         `json:"arch"`
}

func NewSystemHandler(collector *host.SystemInfoCollector) *SystemHandler {
	return &SystemHandler{
		collector: collector,
	}
}

func (h *SystemHandler) GetSystemInfo(c *gin.Context) {
	info, err := h.collector.GetCurrentInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	c.JSON(http.StatusOK, SystemMetricsResponse{
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

func (h *SystemHandler) StreamSystemMetrics(c *gin.Context) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade connection"})
		return
	}
	defer ws.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			info, err := h.collector.GetCurrentInfo()
			if err != nil {
				continue
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

			msg := StreamMetricsMessage{
				Type:      "metrics",
				Timestamp: time.Now(),
				CPU:       cpuPercent,
				Memory: MemoryInfoResp{
					Total:      info.MemoryInfo.Total / (1024 * 1024),
					Used:       info.MemoryInfo.Used / (1024 * 1024),
					Available:  info.MemoryInfo.Available / (1024 * 1024),
					Percentage: memPercent,
				},
				Uptime: uptimeSeconds,
				OS:     runtime.GOOS,
				Arch:   runtime.GOARCH,
			}

			if err := ws.WriteJSON(msg); err != nil {
				return
			}
		}
	}
}

func (h *SystemHandler) GetDetailedSystemInfo(c *gin.Context) {
	info, err := h.collector.GetCurrentInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
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
